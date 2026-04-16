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

// 敏感词检查器
type SensitiveWordChecker struct {
	root          *TrieNode
	homophoneMode bool // 是否启用谐音模式
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

// textToPinyin 将文本转换为拼音字符串
func (swc *SensitiveWordChecker) textToPinyin(text string) string {
	py := pinyin.LazyConvert(text, &swc.pinyinArgs)
	return strings.Join(py, "")
}

// textToPinyinWithMapping 将文本转换为拼音，并返回拼音到原文位置的映射
func (swc *SensitiveWordChecker) textToPinyinWithMapping(text string) (string, []int) {
	runes := []rune(text)
	pinyinParts := pinyin.LazyConvert(text, &swc.pinyinArgs)

	// 构建拼音字符串和位置映射
	var pinyinStr strings.Builder
	positionMap := make([]int, 0)

	for i, py := range pinyinParts {
		if i < len(runes) {
			// 如果拼音为空（非中文字符），保留原字符
			if py == "" {
				py = string(runes[i])
			}
			// 记录每个拼音字符对应的原文位置
			for j := 0; j < len(py); j++ {
				positionMap = append(positionMap, i)
			}
		}
		pinyinStr.WriteString(py)
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
		if r < '\u4e00' || r > '\u9fff' {
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
	runes := []rune(text)

	// 首先检查原文本
	for i := 0; i < len(runes); i++ {
		node := swc.root
		j := i

		for j < len(runes) {
			char := runes[j]
			if child, exists := node.children[char]; exists {
				node = child
				if node.isEnd {
					return true
				}
				j++
			} else {
				break
			}
		}
	}

	// 如果启用了谐音模式，同时检查拼音形式
	if swc.homophoneMode {
		pinyinText := swc.textToPinyin(text)
		pinyinRunes := []rune(pinyinText)

		for i := 0; i < len(pinyinRunes); i++ {
			node := swc.root
			j := i

			for j < len(pinyinRunes) {
				char := pinyinRunes[j]
				if child, exists := node.children[char]; exists {
					node = child
					if node.isEnd && node.isHomophone {
						return true
					}
					j++
				} else {
					break
				}
			}
		}
	}

	return false
}

// 找到文本中所有的敏感词
func (swc *SensitiveWordChecker) FindAll(text string) []string {
	var foundWords []string
	text = swc.format(text)
	runes := []rune(text)

	// 查找原文本中的敏感词
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
					foundWords = append(foundWords, string(matchedRunes))
					break
				}
				j++
			} else {
				break
			}
		}
	}

	// 如果启用了谐音模式，同时查找拼音形式的匹配
	if swc.homophoneMode {
		pinyinText := swc.textToPinyin(text)
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
						// 记录检测到谐音匹配
						foundWords = append(foundWords, string(matchedRunes))
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

	// 首先查找原文本中的敏感词
	for i := 0; i < textLen; i++ {
		node := swc.root
		j := i
		matchStart := i

		for j < textLen {
			char := runes[j]
			if child, exists := node.children[char]; exists {
				node = child
				if node.isEnd && !node.isHomophone {
					// 找到敏感词，记录需要替换的位置
					for k := matchStart; k <= j; k++ {
						replacePositions[k] = true
					}
					break
				}
				j++
			} else {
				break
			}
		}
	}

	// 如果启用了谐音模式，同时查找拼音形式的匹配
	if swc.homophoneMode {
		pinyinText, positionMap := swc.textToPinyinWithMapping(text)
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
							originalPos := positionMap[k]
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
