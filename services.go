package sensitive_filter

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mozillazg/go-pinyin"
)

// TrieNode 字典树节点
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	// isHomophone 标记是否为谐音词（仅在 isEnd=true 时有意义）
	isHomophone bool
}

// NewTrieNode 创建新的字典树节点
func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[rune]*TrieNode)}
}

// matchResult 敏感词匹配结果
type matchResult struct {
	word  string
	start int
	end   int
}

// SensitiveWordChecker 敏感词检查器
type SensitiveWordChecker struct {
	root          *TrieNode
	homophoneMode bool // 谐音模式
	deformMode    bool // 变形词模式
	martianMode   bool // 火星文模式
}

// SensitiveChecker 默认全局实例
var SensitiveChecker *SensitiveWordChecker

// 拼音转换参数（不可变常量）
var pinyinArgs = pinyin.Args{
	Style:     pinyin.Normal,
	Separator: "",
}

// New 创建新的敏感词检查器，默认启用所有模式
func (swc *SensitiveWordChecker) New() *SensitiveWordChecker {
	return &SensitiveWordChecker{
		root:          NewTrieNode(),
		homophoneMode: true,
		deformMode:    true,
		martianMode:   true,
	}
}

// format 统一归一化处理：小写、trim、火星文转换
func (swc *SensitiveWordChecker) format(text string) string {
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)
	if swc.martianMode {
		text = normalizeMartian(text)
	}
	return text
}

// EnableHomophoneMode 启用谐音过滤模式
func (swc *SensitiveWordChecker) EnableHomophoneMode() { swc.homophoneMode = true }

// DisableHomophoneMode 禁用谐音过滤模式
func (swc *SensitiveWordChecker) DisableHomophoneMode() { swc.homophoneMode = false }

// EnableDeformMode 启用变形词过滤模式
func (swc *SensitiveWordChecker) EnableDeformMode() { swc.deformMode = true }

// DisableDeformMode 禁用变形词过滤模式
func (swc *SensitiveWordChecker) DisableDeformMode() { swc.deformMode = false }

// EnableMartianMode 启用火星文过滤模式
func (swc *SensitiveWordChecker) EnableMartianMode() { swc.martianMode = true }

// DisableMartianMode 禁用火星文过滤模式
func (swc *SensitiveWordChecker) DisableMartianMode() { swc.martianMode = false }

// AddMartianMapping 添加自定义火星文映射
func AddMartianMapping(martian, standard string) {
	martianMap[martian] = standard
}

// RemoveMartianMapping 移除火星文映射
func RemoveMartianMapping(martian string) {
	delete(martianMap, martian)
}

// GetMartianMappings 获取所有火星文映射副本
func GetMartianMappings() map[string]string {
	c := make(map[string]string, len(martianMap))
	for k, v := range martianMap {
		c[k] = v
	}
	return c
}

// removeSeparators 移除文本中的分隔符，返回清理后的文本和位置映射
func removeSeparators(text string) (string, []int) {
	runes := []rune(text)
	cleanedRunes := make([]rune, 0, len(runes))
	positionMap := make([]int, 0, len(runes))

	for i, char := range runes {
		if !separatorSet[char] {
			cleanedRunes = append(cleanedRunes, char)
			positionMap = append(positionMap, i)
		}
	}
	return string(cleanedRunes), positionMap
}

// containsSeparator 检查文本是否包含分隔符
func containsSeparator(text string) bool {
	for _, char := range text {
		if separatorSet[char] {
			return true
		}
	}
	return false
}

// separatorSet 常见分隔符集合
var separatorSet = map[rune]bool{
	'-': true, '_': true, '.': true, '~': true, '@': true,
	'#': true, '$': true, '%': true, '^': true, '&': true,
	'*': true, '(': true, ')': true, '[': true, ']': true,
	'{': true, '}': true, '|': true, '\\': true, '/': true,
	'<': true, '>': true, ',': true, ';': true, ':': true,
	'\'': true, '"': true, '`': true, ' ': true,
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

// normalizeMartian 将火星文转换为标准文字（贪心最长词匹配，O(n)）
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
			end := i + length
			if end <= runeLen {
				if standard, exists := martianMap[string(runes[i:end])]; exists {
					result.WriteString(standard)
					i = end
					matched = true
					break
				}
			}
		}

		if !matched {
			// 单字匹配
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

// isChinese 判断字符是否为中文字符
func isChinese(char rune) bool {
	return char >= '\u4e00' && char <= '\u9fff'
}

// textToPinyin 将文本转换为拼音字符串（非中文原样保留）
func textToPinyin(text string) string {
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

// Insert 添加敏感词到字典树
func (swc *SensitiveWordChecker) Insert(word string) {
	word = swc.format(word)
	if word == "" {
		return
	}

	insertToTrie(swc.root, word, false)

	// 纯中文词同时插入拼音形式用于谐音匹配
	if swc.homophoneMode && isPureChinese(word) {
		pinyinStr := textToPinyin(word)
		if pinyinStr != "" {
			insertToTrie(swc.root, pinyinStr, true)
		}
	}
}

// insertToTrie 将单词插入字典树
func insertToTrie(root *TrieNode, word string, isHomophone bool) {
	node := root
	for _, char := range word {
		child, exists := node.children[char]
		if !exists {
			child = NewTrieNode()
			node.children[char] = child
		}
		node = child
	}
	node.isEnd = true
	node.isHomophone = isHomophone
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

// hotReload 热重载敏感词库（后台 goroutine，每 2 秒检测文件变更）
func (swc *SensitiveWordChecker) hotReload(filePath string) {
	for {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		modTime := fileInfo.ModTime()
		time.Sleep(2 * time.Second)
		newInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		if !modTime.Equal(newInfo.ModTime()) {
			log.Println("敏感词库已更新，重新加载词库...")
			swcNew := SensitiveChecker.New()
			if err := swcNew.LoadFromFileByLine(filePath); err != nil {
				continue
			}
			swc.root = swcNew.root
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

	// 原文匹配
	if hasMatchInText(swc.root, checkText, false) {
		return true
	}

	// 谐音匹配
	if swc.canHomophoneMatch(text) {
		pinyinText := textToPinyin(checkText)
		if hasMatchInText(swc.root, pinyinText, true) {
			return true
		}
	}
	return false
}

// hasMatchInText 在文本中查找敏感词匹配
// matchHomophoneOnly: true=只匹配谐音词, false=匹配全部
func hasMatchInText(root *TrieNode, text string, matchHomophoneOnly bool) bool {
	runes := []rune(text)
	for i, runeLen := 0, len(runes); i < runeLen; i++ {
		node := root
		for j := i; j < runeLen; j++ {
			child, exists := node.children[runes[j]]
			if !exists {
				break
			}
			node = child
			if node.isEnd && (!matchHomophoneOnly || node.isHomophone) {
				return true
			}
		}
	}
	return false
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

// findMatchesWithPositions 查找所有原文匹配的敏感词及位置
func findMatchesWithPositions(root *TrieNode, text string) []matchResult {
	runes := []rune(text)
	matches := make([]matchResult, 0)

	for i, runeLen := 0, len(runes); i < runeLen; i++ {
		node := root
		var matchedRunes []rune
		for j := i; j < runeLen; j++ {
			child, exists := node.children[runes[j]]
			if !exists {
				break
			}
			node = child
			matchedRunes = append(matchedRunes, runes[j])
			if node.isEnd && !node.isHomophone {
				matches = append(matches, matchResult{string(matchedRunes), i, j})
				break
			}
		}
	}
	return matches
}

// markMatchedPositions 标记已匹配的位置范围
func markMatchedPositions(positions map[int]bool, start, end int) {
	for k := start; k <= end; k++ {
		positions[k] = true
	}
}

// hasPositionOverlap 检查位置范围是否与已标记位置重叠
func hasPositionOverlap(positions map[int]bool, start, end int) bool {
	for k := start; k <= end; k++ {
		if positions[k] {
			return true
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

	// 原文匹配
	for _, m := range findMatchesWithPositions(swc.root, checkText) {
		found = append(found, m.word)
		markMatchedPositions(matchedPositions, m.start, m.end)
	}

	// 谐音匹配
	if swc.canHomophoneMatch(text) {
		pinyinText := textToPinyin(checkText)
		pinyinRunes := []rune(pinyinText)

		for i, pyLen := 0, len(pinyinRunes); i < pyLen; i++ {
			if matchedPositions[i] {
				continue
			}
			node := swc.root
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
	return scanner.Err()
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
	checkRunes := []rune(checkText)

	// 第一遍：原文匹配（含谐音）
	for i, checkLen := 0, len(checkRunes); i < checkLen; i++ {
		node := swc.root
		for j := i; j < checkLen; j++ {
			child, exists := node.children[checkRunes[j]]
			if !exists {
				break
			}
			node = child
			if node.isEnd {
				for k := i; k <= j; k++ {
					pos := k
					if swc.deformMode && k < len(deformPosMap) {
						pos = deformPosMap[k]
					}
					if pos < textLen {
						replacePos[pos] = true
					}
				}
				break
			}
		}
	}

	// 第二遍：谐音匹配
	if swc.canHomophoneMatch(text) {
		pinyinText, positionMap := textToPinyinWithMapping(checkText)
		pinyinRunes := []rune(pinyinText)

		for i, pyLen := 0, len(pinyinRunes); i < pyLen; i++ {
			node := swc.root
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

// LoadFromTextFile 从文件加载敏感词库，并启动热重载监控
func (swc *SensitiveWordChecker) LoadFromTextFile(filepath string) {
	if err := swc.LoadFromFileByLine(filepath); err != nil {
		log.Printf("加载敏感词库失败: %v\n", err)
		return
	}
	go swc.hotReload(filepath)
}
