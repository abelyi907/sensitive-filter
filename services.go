package sensitive_filter

import (
	"bufio"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/mozillazg/go-pinyin"
)

// ============================================================================
// Aho-Corasick Automaton — O(n) single-pass text matching
// ============================================================================

// trieOutput 存储在 AC 节点上的匹配输出
type trieOutput struct {
	word        string // 匹配到的敏感词
	isHomophone bool   // 是否为谐音模式插入的拼音词
	length      int    // rune 长度
}

// TrieNode 字典树节点（兼容旧版 API 测试）
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool   // 当前节点是否为某一敏感词的结尾
	isHomophone bool // 当前节点是否为谐音词结尾

	// Aho-Corasick 字段（惰性构建）
	fail    *TrieNode
	outputs []trieOutput // 包含从 fail 链继承的完整输出列表
	hasOut  bool         // outputs 非空（快速判断）
}

// NewTrieNode 创建新的字典树节点（兼容旧版 API 测试）
func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[rune]*TrieNode)}
}

// ahoCorasick 自动机封装
type ahoCorasick struct {
	root  *TrieNode
	dirty bool // 插入新词后标记需要重建 fail 指针
}

func newAhoCorasick() *ahoCorasick {
	return &ahoCorasick{root: NewTrieNode(), dirty: false}
}

// insert 将单词插入字典树（不重建 fail 链接）
func (ac *ahoCorasick) insert(word string, isHomophone bool) {
	node := ac.root
	for _, r := range word {
		child, ok := node.children[r]
		if !ok {
			child = NewTrieNode()
			node.children[r] = child
		}
		node = child
	}
	node.isEnd = true
	node.isHomophone = isHomophone
	node.outputs = append(node.outputs, trieOutput{
		word:        word,
		isHomophone: isHomophone,
		length:      len([]rune(word)),
	})
	node.hasOut = true
	ac.dirty = true
}

// build 使用 BFS 构建 AC 自动机的 fail 指针并传播输出
func (ac *ahoCorasick) build() {
	if !ac.dirty {
		return
	}

	// BFS 队列
	queue := make([]*TrieNode, 0, 64)

	// 根节点的直接子节点 fail 指向根
	for _, child := range ac.root.children {
		child.fail = ac.root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for char, child := range node.children {
			// 沿 fail 链查找可匹配 char 的节点
			failTarget := node.fail
			for failTarget != nil && failTarget != ac.root {
				if next, ok := failTarget.children[char]; ok {
					child.fail = next
					break
				}
				failTarget = failTarget.fail
			}
			if child.fail == nil {
				// 检查根节点
				if next, ok := ac.root.children[char]; ok {
					child.fail = next
				} else {
					child.fail = ac.root
				}
			}

			// 传播 fail 节点的输出到当前节点
			if child.fail.hasOut {
				child.outputs = append(child.outputs, child.fail.outputs...)
				child.hasOut = true
				// 继承 isEnd / isHomophone 标记（上层方法可按需使用）
				if child.fail.isEnd {
					child.isEnd = child.isEnd || child.fail.isEnd
					child.isHomophone = child.isHomophone || child.fail.isHomophone
				}
			}

			queue = append(queue, child)
		}
	}

	ac.dirty = false
}

// matchResult 匹配结果
type matchResult struct {
	word        string
	start       int // 起始 rune 索引（含）
	end         int // 结束 rune 索引（含）
	isHomophone bool
}

// matchAll 单遍遍历文本，返回所有 AC 匹配结果
func (ac *ahoCorasick) matchAll(text string) []matchResult {
	ac.build()

	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	matches := make([]matchResult, 0, len(runes)/2)
	node := ac.root

	for i, r := range runes {
		// 沿 fail 链寻找可匹配的转移
		for node != ac.root {
			if _, ok := node.children[r]; ok {
				break
			}
			node = node.fail
		}
		if child, ok := node.children[r]; ok {
			node = child
		}

		// 收集当前节点的所有输出
		if node.hasOut {
			for _, out := range node.outputs {
				matches = append(matches, matchResult{
					word:        out.word,
					start:       i - out.length + 1,
					end:         i,
					isHomophone: out.isHomophone,
				})
			}
		}
	}
	return matches
}

// ============================================================================
// 拼音缓存 — 避免同一文本重复转换
// ============================================================================

type pinyinCache struct {
	mu      sync.RWMutex
	simple  map[string]string   // text -> pinyin
	mapped  map[string]pyEntry  // text -> (pinyin, posMap)
}

type pyEntry struct {
	pinyin  string
	posMap  []int
}

func newPinyinCache() *pinyinCache {
	return &pinyinCache{
		simple: make(map[string]string, 64),
		mapped: make(map[string]pyEntry, 64),
	}
}

func (c *pinyinCache) get(text string) (string, bool) {
	c.mu.RLock()
	s, ok := c.simple[text]
	c.mu.RUnlock()
	return s, ok
}

func (c *pinyinCache) set(text, pinyin string) {
	c.mu.Lock()
	c.simple[text] = pinyin
	c.mu.Unlock()
}

func (c *pinyinCache) getMapped(text string) (string, []int, bool) {
	c.mu.RLock()
	e, ok := c.mapped[text]
	c.mu.RUnlock()
	if ok {
		return e.pinyin, e.posMap, true
	}
	return "", nil, false
}

func (c *pinyinCache) setMapped(text, pinyin string, posMap []int) {
	c.mu.Lock()
	c.mapped[text] = pyEntry{pinyin: pinyin, posMap: posMap}
	c.mu.Unlock()
}

// ============================================================================
// 拼音转换函数
// ============================================================================

var pinyinArgs = pinyin.Args{
	Style:     pinyin.Normal,
	Separator: "",
}

// isChinese 判断字符是否为中文字符
func isChinese(char rune) bool {
	return char >= '\u4e00' && char <= '\u9fff'
}

// isPureChinese 检查字符串是否只包含中文字符
func isPureChinese(s string) bool {
	for _, r := range s {
		if !isChinese(r) {
			return false
		}
	}
	return true
}

// textToPinyin 将文本转换为拼音字符串（使用缓存）
func textToPinyin(text string) string {
	if text == "" {
		return text
	}
	// 如果没有中文，无需转换
	hasChinese := false
	for _, r := range text {
		if isChinese(r) {
			hasChinese = true
			break
		}
	}
	if !hasChinese {
		return text
	}

	pinyinParts := pinyin.LazyConvert(text, &pinyinArgs)
	if len(pinyinParts) == 0 {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))
	pyIndex := 0
	for _, char := range text {
		if isChinese(char) && pyIndex < len(pinyinParts) {
			result.WriteString(pinyinParts[pyIndex])
			pyIndex++
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// textToPinyinWithMapping 将文本转换为拼音，并返回拼音到原文位置的映射
func textToPinyinWithMapping(text string) (string, []int) {
	runes := []rune(text)
	pinyinParts := pinyin.LazyConvert(text, &pinyinArgs)

	var buf strings.Builder
	positionMap := make([]int, 0, len(runes))
	pyIndex := 0

	for i, char := range runes {
		if isChinese(char) && pyIndex < len(pinyinParts) {
			py := pinyinParts[pyIndex]
			for j := 0; j < len(py); j++ {
				positionMap = append(positionMap, i)
			}
			buf.WriteString(py)
			pyIndex++
		} else {
			positionMap = append(positionMap, i)
			buf.WriteRune(char)
		}
	}
	return buf.String(), positionMap
}

// ============================================================================
// 常量：分隔符集、火星文映射表
// ============================================================================

// separatorSet 常见分隔符集合（使用 rune 数组 + 函数检查，降低 map GC 压力）
var separatorTable [65536]bool

func init() {
	seps := []rune{'-', '_', '.', '~', '@', '#', '$', '%', '^', '&',
		'*', '(', ')', '[', ']', '{', '}', '|', '\\', '/',
		'<', '>', ',', ';', ':', '\'', '"', '`', ' '}
	for _, s := range seps {
		if int(s) < len(separatorTable) {
			separatorTable[s] = true
		}
	}
}

func isSeparator(r rune) bool {
	return int(r) < len(separatorTable) && separatorTable[r]
}

func containsSeparator(text string) bool {
	for _, r := range text {
		if isSeparator(r) {
			return true
		}
	}
	return false
}

// removeSeparators 移除文本中的分隔符，返回清理后的文本和位置映射
func removeSeparators(text string) (string, []int) {
	runes := []rune(text)
	cleaned := make([]rune, 0, len(runes))
	posMap := make([]int, 0, len(runes))

	for i, r := range runes {
		if !isSeparator(r) {
			cleaned = append(cleaned, r)
			posMap = append(posMap, i)
		}
	}
	return string(cleaned), posMap
}

// martianMap 火星文映射表
var martianMap = map[string]string{
	"卜": "不", "8": "不",
	"滴": "的", "德": "的",
	"乐": "了", "啦": "了",
	"莪": "我", "偶": "我",
	"伱": "你", "祢": "你",
	"ta": "他", "TA": "他",

	"稀饭": "喜欢", "神马": "什么", "虾米": "什么",
	"酱紫": "这样子",
	"表": "不要", "造": "知道", "晓得": "知道",
	"灰常": "非常",
	"粉": "很", "素": "是", "系": "是",
	"木有": "没有", "米有": "没有",
	"肿么": "怎么", "为神马": "为什么",
	"童鞋": "同学",
	"淫": "人", "银": "人",
	"盆友": "朋友",
	"粑粑": "爸爸", "麻咪": "妈妈",
	"北鼻": "baby", "达令": "darling",
	"3Q": "谢谢", "thx": "谢谢", "3ks": "谢谢",
	"88": "拜拜", "白白": "拜拜",
	"安": "晚安", "早安": "早上好", "午安": "中午好",
}

// normalizeMartian 将火星文转换为标准文字（贪心最长词匹配）
func normalizeMartian(text string) string {
	if text == "" {
		return text
	}

	var result strings.Builder
	runes := []rune(text)
	result.Grow(utf8.RuneCountInString(text))

	for i, runeLen := 0, len(runes); i < runeLen; {
		matched := false

		// 优先匹配最长词（从4字到2字）
		for length := 4; length >= 2; length-- {
			if end := i + length; end <= runeLen {
				if standard, exists := martianMap[string(runes[i:end])]; exists {
					result.WriteString(standard)
					i = end
					matched = true
					break
				}
			}
		}

		if !matched {
			if standard, exists := martianMap[string(runes[i])]; exists {
				result.WriteString(standard)
			} else {
				result.WriteRune(runes[i])
			}
			i++
		}
	}
	return result.String()
}

// ============================================================================
// 火星文映射管理（线程安全的读写锁）
// ============================================================================

var (
	martianMu      sync.RWMutex
)

// AddMartianMapping 添加自定义火星文映射
func AddMartianMapping(martian, standard string) {
	martianMu.Lock()
	martianMap[martian] = standard
	martianMu.Unlock()
}

// RemoveMartianMapping 移除火星文映射
func RemoveMartianMapping(martian string) {
	martianMu.Lock()
	delete(martianMap, martian)
	martianMu.Unlock()
}

// GetMartianMappings 获取所有火星文映射副本
func GetMartianMappings() map[string]string {
	martianMu.RLock()
	defer martianMu.RUnlock()
	c := make(map[string]string, len(martianMap))
	for k, v := range martianMap {
		c[k] = v
	}
	return c
}

// ============================================================================
// SensitiveWordChecker — 主检查器（Builder 模式）
// ============================================================================

// Option 函数式配置选项
type Option func(*SensitiveWordChecker)

// WithHomophone 设置谐音模式
func WithHomophone(enabled bool) Option {
	return func(swc *SensitiveWordChecker) { swc.homophoneMode = enabled }
}

// WithDeform 设置变形词模式
func WithDeform(enabled bool) Option {
	return func(swc *SensitiveWordChecker) { swc.deformMode = enabled }
}

// WithMartian 设置火星文模式
func WithMartian(enabled bool) Option {
	return func(swc *SensitiveWordChecker) { swc.martianMode = enabled }
}

// SensitiveWordChecker 敏感词检查器
type SensitiveWordChecker struct {
	ac            *ahoCorasick
	pinyinCache   *pinyinCache
	homophoneMode bool
	deformMode    bool
	martianMode   bool
}

// SensitiveChecker 默认全局实例
var SensitiveChecker *SensitiveWordChecker

func init() {
	SensitiveChecker = New()
}

// New 创建新的敏感词检查器，默认启用所有模式
func New(opts ...Option) *SensitiveWordChecker {
	swc := &SensitiveWordChecker{
		ac:            newAhoCorasick(),
		pinyinCache:   newPinyinCache(),
		homophoneMode: true,
		deformMode:    true,
		martianMode:   true,
	}
	for _, opt := range opts {
		opt(swc)
	}
	return swc
}

// New 实例方法（兼容旧版 API：SensitiveChecker.New()）
func (swc *SensitiveWordChecker) New() *SensitiveWordChecker {
	return New(
		WithHomophone(swc.homophoneMode),
		WithDeform(swc.deformMode),
		WithMartian(swc.martianMode),
	)
}

// ============================================================================
// 模式控制
// ============================================================================

func (swc *SensitiveWordChecker) EnableHomophoneMode()  { swc.homophoneMode = true }
func (swc *SensitiveWordChecker) DisableHomophoneMode() { swc.homophoneMode = false }
func (swc *SensitiveWordChecker) EnableDeformMode()     { swc.deformMode = true }
func (swc *SensitiveWordChecker) DisableDeformMode()    { swc.deformMode = false }
func (swc *SensitiveWordChecker) EnableMartianMode()    { swc.martianMode = true }
func (swc *SensitiveWordChecker) DisableMartianMode()   { swc.martianMode = false }

// ============================================================================
// 内部预处理
// ============================================================================

// format 统一归一化处理：小写、trim、火星文转换
func (swc *SensitiveWordChecker) format(text string) string {
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)
	if swc.martianMode {
		text = normalizeMartian(text)
	}
	return text
}

// canHomophoneMatch 判断是否允许谐音匹配
func (swc *SensitiveWordChecker) canHomophoneMatch(text string) bool {
	if !swc.homophoneMode {
		return false
	}
	if !swc.deformMode && containsSeparator(text) {
		return false
	}
	return true
}

// cachedPinyin 使用缓存的拼音转换
func (swc *SensitiveWordChecker) cachedPinyin(text string) string {
	if s, ok := swc.pinyinCache.get(text); ok {
		return s
	}
	s := textToPinyin(text)
	swc.pinyinCache.set(text, s)
	return s
}

// cachedPinyinWithMapping 使用缓存的拼音+位置映射转换
func (swc *SensitiveWordChecker) cachedPinyinWithMapping(text string) (string, []int) {
	if s, m, ok := swc.pinyinCache.getMapped(text); ok {
		return s, m
	}
	s, m := textToPinyinWithMapping(text)
	swc.pinyinCache.setMapped(text, s, m)
	return s, m
}

// ============================================================================
// 核心 API
// ============================================================================

// Insert 添加敏感词到字典树
func (swc *SensitiveWordChecker) Insert(word string) {
	word = swc.format(word)
	if word == "" {
		return
	}

	swc.ac.insert(word, false)

	// 纯中文词同时插入拼音形式用于谐音匹配
	if swc.homophoneMode && isPureChinese(word) {
		pinyinStr := textToPinyin(word)
		if pinyinStr != "" {
			swc.ac.insert(pinyinStr, true)
		}
	}
}

// Contains 检查文本是否包含敏感词
func (swc *SensitiveWordChecker) Contains(text string) bool {
	text = swc.format(text)

	checkText := text
	if swc.deformMode {
		checkText, _ = removeSeparators(text)
	}

	// 原文匹配（使用 AC 单遍扫描，不区分 isHomophone — 拼音路径与原文字路径不重叠）
	matches := swc.ac.matchAll(checkText)
	for _, m := range matches {
		if !m.isHomophone {
			return true
		}
	}

	// 谐音匹配
	if swc.canHomophoneMatch(text) {
		pinyinText := swc.cachedPinyin(checkText)
		pinyinMatches := swc.ac.matchAll(pinyinText)
		for _, m := range pinyinMatches {
			if m.isHomophone {
				return true
			}
		}
	}
	return false
}

// FindAll 查找文本中所有敏感词
func (swc *SensitiveWordChecker) FindAll(text string) []string {
	text = swc.format(text)

	checkText := text
	if swc.deformMode {
		checkText, _ = removeSeparators(text)
	}

	matchedPositions := make(map[int]bool)
	found := make([]string, 0)

	// 原文匹配：只匹配非谐音模式
	matches := swc.ac.matchAll(checkText)
	// 过滤：每个 start 位置只取最短匹配
	seenStart := make(map[int]bool)
	for _, m := range matches {
		if m.isHomophone || seenStart[m.start] {
			continue
		}
		seenStart[m.start] = true
		found = append(found, m.word)
		markMatchedPositions(matchedPositions, m.start, m.end)
	}

	// 谐音匹配
	if swc.canHomophoneMatch(text) {
		pinyinText := swc.cachedPinyin(checkText)
		pinyinRunes := []rune(pinyinText)

		for i, pyLen := 0, len(pinyinRunes); i < pyLen; i++ {
			if matchedPositions[i] {
				continue
			}
			node := swc.ac.root
			var matchedRunes []rune
			for j := i; j < pyLen; j++ {
				child, exists := node.children[pinyinRunes[j]]
				if !exists {
					break
				}
				node = child
				matchedRunes = append(matchedRunes, pinyinRunes[j])
				if node.isEnd && node.isHomophone && !hasPositionOverlap(matchedPositions, i, j) {
					found = append(found, string(matchedRunes))
					markMatchedPositions(matchedPositions, i, j)
					break
				}
			}
		}
	}
	return found
}

// Replace 替换文本中的敏感词为指定字符
func (swc *SensitiveWordChecker) Replace(text string, replacement rune) string {
	text = swc.format(text)
	runes := []rune(text)
	textLen := len(runes)

	// 变形词模式：清理分隔符并获取位置映射
	checkText := text
	var deformPosMap []int
	if swc.deformMode {
		checkText, deformPosMap = removeSeparators(text)
	}

	replacePos := make(map[int]bool, textLen/2)
	// 第一遍：原文匹配（含谐音 — 不区分 isHomophone）
	matches := swc.ac.matchAll(checkText)
	for _, m := range matches {
		for k := m.start; k <= m.end; k++ {
			pos := k
			if swc.deformMode && k < len(deformPosMap) {
				pos = deformPosMap[k]
			}
			if pos < textLen {
				replacePos[pos] = true
			}
		}
	}

	// 第二遍：谐音匹配
	if swc.canHomophoneMatch(text) {
		pinyinText, positionMap := swc.cachedPinyinWithMapping(checkText)
		pinyinRunes := []rune(pinyinText)

		for i, pyLen := 0, len(pinyinRunes); i < pyLen; i++ {
			node := swc.ac.root
			matchedLen := 0
			for j := i; j < pyLen; j++ {
				child, exists := node.children[pinyinRunes[j]]
				if !exists {
					break
				}
				node = child
				matchedLen++
				if node.isEnd && node.isHomophone {
					for k := i; k < i+matchedLen && k < len(positionMap); k++ {
						cleanedPos := positionMap[k]
						pos := cleanedPos
						if swc.deformMode && cleanedPos < len(deformPosMap) {
							pos = deformPosMap[cleanedPos]
						}
						if pos < textLen {
							replacePos[pos] = true
						}
					}
					break
				}
			}
		}
	}

	// 执行替换
	for i := 0; i < textLen; i++ {
		if replacePos[i] {
			runes[i] = replacement
		}
	}
	return string(runes)
}

// ============================================================================
// 辅助函数（位置标记）
// ============================================================================

func markMatchedPositions(positions map[int]bool, start, end int) {
	for k := start; k <= end; k++ {
		positions[k] = true
	}
}

func hasPositionOverlap(positions map[int]bool, start, end int) bool {
	for k := start; k <= end; k++ {
		if positions[k] {
			return true
		}
	}
	return false
}

// ============================================================================
// 文件加载与热重载
// ============================================================================

// LoadFromFileByLine 从文件加载敏感词列表，每行一个
func (swc *SensitiveWordChecker) LoadFromFileByLine(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			swc.Insert(line)
		}
	}
	// 显式构建 fail 链接
	swc.ac.build()
	return scanner.Err()
}

// hotReload 热重载敏感词库（后台 goroutine，每 2 秒检测文件变更）
// lastModTime 传入加载时的文件时间戳，避免首次检测的时序竞态
// 每次循环先执行 stat 检查再 sleep，确保首次检查立即进行
func (swc *SensitiveWordChecker) hotReload(filePath string, lastModTime time.Time) {
	for {
		newInfo, err := os.Stat(filePath)
		if err == nil && !newInfo.ModTime().Equal(lastModTime) {
			log.Println("敏感词库已更新，重新加载词库...")
			swcNew := New(
				WithHomophone(swc.homophoneMode),
				WithDeform(swc.deformMode),
				WithMartian(swc.martianMode),
			)
			if err := swcNew.LoadFromFileByLine(filePath); err != nil {
				continue
			}
			// 原子替换自动机
			swc.ac = swcNew.ac
			swc.pinyinCache = swcNew.pinyinCache
			lastModTime = newInfo.ModTime()
		}
		time.Sleep(2 * time.Second)
	}
}

// LoadFromTextFile 从文件加载敏感词库，并启动热重载监控
func (swc *SensitiveWordChecker) LoadFromTextFile(filepath string) {
	if err := swc.LoadFromFileByLine(filepath); err != nil {
		log.Printf("加载敏感词库失败: %v\n", err)
		return
	}
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		log.Printf("获取文件状态失败: %v\n", err)
		return
	}
	go swc.hotReload(filepath, fileInfo.ModTime())
}
