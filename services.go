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
		homophoneMode: false,
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

	// 如果启用了谐音模式，同时插入拼音形式
	if swc.homophoneMode {
		swc.insertPinyin(word)
	}
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
	fs := swc.FindAll(text)
	if len(fs) == 0 {
		return text
	}
	//循环敏感词
	for _, f := range fs {
		text = strings.ReplaceAll(text, f, strings.Repeat(string(replacement), len(f)))
	}
	return text
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
