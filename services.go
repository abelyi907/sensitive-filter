package sensitive_filter

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"
)

/**
敏感词检查器

*/

// 字典树节点
type TrieNode struct {
	children    map[rune]*TrieNode
	isEnd       bool
	isHomophone bool // 标记是否为谐音词
}

// SensitiveWordChecker 敏感词检查器
type SensitiveWordChecker struct {
	root          *TrieNode
	homophoneMode bool // 是否启用谐音模式
	deformMode    bool // 是否启用变形词模式
	pinyinArgs    pinyin.Args
}

var SensitiveChecker *SensitiveWordChecker

// 创建新的敏感词检查器
func (swc *SensitiveWordChecker) New() *SensitiveWordChecker {
	temp := &SensitiveWordChecker{
		root: &TrieNode{
			children: make(map[rune]*TrieNode),
			isEnd:    false,
		},
		homophoneMode: true, // 默认启用谐音模式
		deformMode:    true, // 默认启用变形词模式
		pinyinArgs: pinyin.Args{
			Style:     pinyin.Normal, // 不带声调的拼音
			Separator: "",
		},
	}
	return temp
}

// 统一对待处理文件进行处理
func (swc *SensitiveWordChecker) format(text string) string {
	text = strings.ToLower(text)
	text = strings.TrimSpace(strings.ToLower(text))
	return text
}

// EnableHomophoneMode 启用谐音过滤模式
func (swc *SensitiveWordChecker) EnableHomophoneMode() {
	swc.homophoneMode = true
}

// DisableHomophoneMode 禁用谐音过滤模式
func (swc *SensitiveWordChecker) DisableHomophoneMode() {
	swc.homophoneMode = false
}

// EnableDeformMode 启用变形词过滤模式
func (swc *SensitiveWordChecker) EnableDeformMode() {
	swc.deformMode = true
}

// DisableDeformMode 禁用变形词过滤模式
func (swc *SensitiveWordChecker) DisableDeformMode() {
	swc.deformMode = false
}

// removeSeparators 移除文本中的分隔符，并返回清理后的文本和位置映射
func (swc *SensitiveWordChecker) removeSeparators(text string) (string, []int) {
	// 常见的分隔符
	separators := getSeparator()

	runes := []rune(text)
	var cleanedRunes []rune
	positionMap := make([]int, 0)

	for i, char := range runes {
		if !separators[char] {
			cleanedRunes = append(cleanedRunes, char)
			positionMap = append(positionMap, i)
		}
	}

	return string(cleanedRunes), positionMap
}

// containsSeparator 检查文本是否包含分隔符
func containsSeparator(text string) bool {
	separators := getSeparator()

	for _, char := range text {
		if separators[char] {
			return true
		}
	}
	return false
}

// 常见分隔符集合（包级别常量，避免重复创建）
var separatorSet = map[rune]bool{
	'-': true, '_': true, '.': true, '~': true, '@': true,
	'#': true, '$': true, '%': true, '^': true, '&': true,
	'*': true, '(': true, ')': true, '[': true, ']': true,
	'{': true, '}': true, '|': true, '\\': true, '/': true,
	'<': true, '>': true, ',': true, ';': true, ':': true,
	'\'': true, '"': true, '`': true, ' ': true,
}

// getSeparator 返回分隔符集合
func getSeparator() map[rune]bool {
	return separatorSet
}

// isChinese 判断字符是否为中文字符
func isChinese(char rune) bool {
	return char >= '\u4e00' && char <= '\u9fff'
}

// textToPinyin 将文本转换为拼音字符串
func (swc *SensitiveWordChecker) textToPinyin(text string) string {
	runes := []rune(text)
	pinyinParts := pinyin.LazyConvert(text, &swc.pinyinArgs)

	var result strings.Builder
	pyIndex := 0

	for _, char := range runes {
		// 判断是否为中文字符
		if isChinese(char) {
			// 中文字符，使用拼音
			if pyIndex < len(pinyinParts) {
				result.WriteString(pinyinParts[pyIndex])
				pyIndex++
			}
		} else {
			// 非中文字符，保留原字符
			result.WriteRune(char)
		}
	}

	return result.String()
}

// textToPinyinWithMapping 将文本转换为拼音，并返回拼音到原文位置的映射
func (swc *SensitiveWordChecker) textToPinyinWithMapping(text string) (string, []int) {
	runes := []rune(text)
	pinyinParts := pinyin.LazyConvert(text, &swc.pinyinArgs)

	// 构建拼音字符串和位置映射
	var pinyinStr strings.Builder
	positionMap := make([]int, 0)
	pyIndex := 0

	for i, char := range runes {
		// 判断是否为中文字符
		if isChinese(char) {
			// 中文字符，使用拼音
			if pyIndex < len(pinyinParts) {
				py := pinyinParts[pyIndex]
				// 记录每个拼音字符对应的原文位置
				for j := 0; j < len(py); j++ {
					positionMap = append(positionMap, i)
				}
				pinyinStr.WriteString(py)
				pyIndex++
			}
		} else {
			// 非中文字符，保留原字符
			positionMap = append(positionMap, i)
			pinyinStr.WriteRune(char)
		}
	}

	return pinyinStr.String(), positionMap
}

// Insert 添加敏感词到字典树
func (swc *SensitiveWordChecker) Insert(word string) {
	node := swc.root
	word = swc.format(word)
	if word == "" {
		return
	}

	for _, char := range word {
		if _, exists := node.children[char]; !exists {
			node.children[char] = &TrieNode{
				children: make(map[rune]*TrieNode),
				isEnd:    false,
			}
		}
		node = node.children[char]
	}
	node.isEnd = true

	// 如果启用了谐音模式，且敏感词只包含中文字符，则同时插入拼音形式
	if swc.homophoneMode && isPureChinese(word) {
		swc.insertPinyin(word)
	}
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

// insertPinyin 插入词的拼音形式（用于谐音匹配）
func (swc *SensitiveWordChecker) insertPinyin(word string) {
	pinyinStr := swc.textToPinyin(word)
	if pinyinStr == "" {
		return
	}

	node := swc.root
	for _, char := range pinyinStr {
		if _, exists := node.children[char]; !exists {
			node.children[char] = &TrieNode{
				children:    make(map[rune]*TrieNode),
				isEnd:       false,
				isHomophone: true,
			}
		}
		node = node.children[char]
	}
	node.isEnd = true
	node.isHomophone = true
}

// 热重载文件
func (swc *SensitiveWordChecker) hotReload(filePath string) {
	for {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			//log.Printf("无法获取文件信息: %v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}
		modTime := fileInfo.ModTime()
		time.Sleep(time.Second * 2)
		newFileInfo, err := os.Stat(filePath)
		if err != nil {
			//log.Printf("无法获取文件信息: %v\n", err)
			continue
		}
		newModTime := newFileInfo.ModTime()
		if modTime != newModTime {
			log.Println("敏感词库已更新，重新加载词库...")
			swcNew := SensitiveChecker.New()
			err = swcNew.LoadFromFileByLine(filePath)
			if err != nil {
				//log.Printf("加载敏感词库变化后重新加载失败: %v\n", err)
				continue
			}

			// 更新当前实例的所有字段，确保指针正确指向新的数据
			swc.root = swcNew.root
		}
	}
}

// 检查文本是否包含敏感词
func (swc *SensitiveWordChecker) Contains(text string) bool {
	text = swc.format(text)

	// 如果启用了变形词模式，先清理分隔符
	checkText := text
	if swc.deformMode {
		checkText, _ = swc.removeSeparators(text)
	}

	// 首先检查原文本
	if len(swc.findMatchesInText(checkText)) > 0 {
		return true
	}

	// 如果应该进行谐音匹配，检查拼音形式
	if !swc.shouldSkipHomophone(text) {
		pinyinText := swc.textToPinyin(checkText)
		if len(swc.findHomophoneMatchesInPinyin(pinyinText)) > 0 {
			return true
		}
	}

	return false
}

// shouldSkipHomophone 判断是否应该跳过谐音匹配
func (swc *SensitiveWordChecker) shouldSkipHomophone(originalText string) bool {
	if !swc.homophoneMode {
		return true
	}
	if !swc.deformMode && containsSeparator(originalText) {
		return true
	}
	return false
}

// findMatchesInText 在文本中查找所有匹配的敏感词（原文匹配）
func (swc *SensitiveWordChecker) findMatchesInText(text string) []string {
	var matches []string
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		node := swc.root
		j := i
		var matchedRunes []rune

		for j < len(runes) {
			char := runes[j]
			if child, exists := node.children[char]; exists {
				node = child
				matchedRunes = append(matchedRunes, char)
				if node.isEnd && !node.isHomophone {
					matches = append(matches, string(matchedRunes))
					break
				}
				j++
			} else {
				break
			}
		}
	}

	return matches
}

// findMatchesWithPositions 在文本中查找所有匹配的敏感词及其位置（原文匹配）
func (swc *SensitiveWordChecker) findMatchesWithPositions(text string) []struct {
	word  string
	start int
	end   int
} {
	var matches []struct {
		word  string
		start int
		end   int
	}
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		node := swc.root
		j := i
		var matchedRunes []rune

		for j < len(runes) {
			char := runes[j]
			if child, exists := node.children[char]; exists {
				node = child
				matchedRunes = append(matchedRunes, char)
				if node.isEnd && !node.isHomophone {
					matches = append(matches, struct {
						word  string
						start int
						end   int
					}{string(matchedRunes), i, j})
					break
				}
				j++
			} else {
				break
			}
		}
	}

	return matches
}

// findHomophoneMatchesInPinyin 在拼音文本中查找谐音匹配
func (swc *SensitiveWordChecker) findHomophoneMatchesInPinyin(pinyinText string) []string {
	var matches []string
	pinyinRunes := []rune(pinyinText)

	for i := 0; i < len(pinyinRunes); i++ {
		node := swc.root
		j := i
		var matchedRunes []rune

		for j < len(pinyinRunes) {
			char := pinyinRunes[j]
			if child, exists := node.children[char]; exists {
				node = child
				matchedRunes = append(matchedRunes, char)
				if node.isEnd && node.isHomophone {
					matches = append(matches, string(matchedRunes))
					break
				}
				j++
			} else {
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

// 找到文本中所有的敏感词
func (swc *SensitiveWordChecker) FindAll(text string) []string {
	var foundWords []string
	text = swc.format(text)

	// 如果启用了变形词模式，先清理分隔符
	checkText := text
	if swc.deformMode {
		checkText, _ = swc.removeSeparators(text)
	}

	// 记录已经找到的匹配位置，避免重复
	matchedPositions := make(map[int]bool)

	// 查找原文本中的敏感词
	matches := swc.findMatchesWithPositions(checkText)
	for _, match := range matches {
		foundWords = append(foundWords, match.word)
		markMatchedPositions(matchedPositions, match.start, match.end)
	}

	// 如果应该进行谐音匹配，查找拼音形式的匹配
	if !swc.shouldSkipHomophone(text) {
		pinyinText := swc.textToPinyin(checkText)
		pinyinRunes := []rune(pinyinText)

		for i := 0; i < len(pinyinRunes); i++ {
			// 如果这个位置已经在原文匹配中被覆盖，跳过
			if matchedPositions[i] {
				continue
			}

			node := swc.root
			j := i
			var matchedRunes []rune

			for j < len(pinyinRunes) {
				char := pinyinRunes[j]
				if child, exists := node.children[char]; exists {
					node = child
					matchedRunes = append(matchedRunes, char)
					if node.isEnd && node.isHomophone {
						// 检查是否已经有原文匹配覆盖了这个范围
						if !hasPositionOverlap(matchedPositions, i, j) {
							foundWords = append(foundWords, string(matchedRunes))
							markMatchedPositions(matchedPositions, i, j)
						}
						break
					}
					j++
				} else {
					break
				}
			}
		}
	}

	return foundWords
}

// 从文件加载敏感词列表，每行一个敏感词
func (swc *SensitiveWordChecker) LoadFromFileByLine(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {

		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			swc.Insert(line)
		}
	}

	return scanner.Err()
}

// Replace 敏感词替换，将敏感词替换为指定字符
func (swc *SensitiveWordChecker) Replace(text string, replacement rune) string {
	text = swc.format(text)
	runes := []rune(text)
	textLen := len(runes)

	// 记录需要替换的位置
	replacePositions := make(map[int]bool)

	// 如果启用了变形词模式，先清理分隔符并获取位置映射
	checkText := text
	var deformPositionMap []int
	if swc.deformMode {
		checkText, deformPositionMap = swc.removeSeparators(text)
	}

	checkRunes := []rune(checkText)
	checkLen := len(checkRunes)

	// 首先查找清理后文本中的敏感词（包括中文和拼音）
	for i := 0; i < checkLen; i++ {
		node := swc.root
		j := i
		matchStart := i

		for j < checkLen {
			char := checkRunes[j]
			if child, exists := node.children[char]; exists {
				node = child
				// 匹配到敏感词（无论是原文还是拼音形式）
				if node.isEnd {
					// 记录需要替换的位置
					for k := matchStart; k <= j; k++ {
						if swc.deformMode && len(deformPositionMap) > 0 {
							// 通过位置映射找到原文中的位置
							if k < len(deformPositionMap) {
								replacePositions[deformPositionMap[k]] = true
							}
						} else {
							replacePositions[k] = true
						}
					}
					break
				}
				j++
			} else {
				break
			}
		}
	}

	// 如果启用了谐音模式，同时将中文转换为拼音后查找匹配
	// 但如果禁用了变形词模式且原文包含分隔符，则跳过谐音检查
	if swc.homophoneMode {
		skipHomophone := false
		if !swc.deformMode {
			if containsSeparator(text) {
				skipHomophone = true
			}
		}

		if !skipHomophone {
			pinyinText, positionMap := swc.textToPinyinWithMapping(checkText)
			pinyinRunes := []rune(pinyinText)
			pinyinLen := len(pinyinRunes)

			for i := 0; i < pinyinLen; i++ {
				node := swc.root
				j := i
				matchStart := i
				var matchedLen int

				for j < pinyinLen {
					char := pinyinRunes[j]
					if child, exists := node.children[char]; exists {
						node = child
						matchedLen++
						if node.isEnd && node.isHomophone {
							// 找到谐音匹配，通过位置映射找到原文中对应的字符
							for k := matchStart; k < matchStart+matchedLen && k < len(positionMap); k++ {
								cleanedPos := positionMap[k]
								var originalPos int
								if swc.deformMode && len(deformPositionMap) > 0 {
									// 通过变形词位置映射找到原文位置
									if cleanedPos < len(deformPositionMap) {
										originalPos = deformPositionMap[cleanedPos]
									} else {
										continue
									}
								} else {
									originalPos = cleanedPos
								}
								if originalPos < textLen {
									replacePositions[originalPos] = true
								}
							}
							break
						}
						j++
					} else {
						break
					}
				}
			}
		}
	}

	// 执行替换
	for i := 0; i < textLen; i++ {
		if replacePositions[i] {
			runes[i] = replacement
		}
	}

	return string(runes)
}

// LoadFromTextFile 从文本文件中加载敏感词库,监察库中的内容有变化时，重新加载
func (swc *SensitiveWordChecker) LoadFromTextFile(filepath string) {
	err := swc.LoadFromFileByLine(filepath)
	if err != nil {
		log.Printf("加载敏感词库失败: %v\n", err)
		return
	}
	// 监控词库变化，若词库有变化，则重新加载
	go swc.hotReload(filepath)
}
