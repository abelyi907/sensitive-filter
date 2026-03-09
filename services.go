package sensitive_filter

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"
	"unicode"
)

/**
敏感词检查器
使用:
  // 创建敏感词检查器
  checker := sensitive_filter.New()
  // 从文件加载敏感词库
  checker.LoadFromTextFile("D:/abel/mygit/sensitive-check/document/敏感词库.txt")

  // 测试文本
  testText := "这是一个包含色情和暴力内容的文本，涉及信息。"

  replacedText := checker.Replace(testText, '*')
  fmt.Printf("替换后的文本: %s\n", replacedText)

    hasSensitive := checker.Contains(testText)
  fmt.Printf("文本是否包含敏感词: %v\n", hasSensitive)

  foundWords := checker.FindAll(testText)
  fmt.Printf("找到的敏感词: %v\n", foundWords)

*/

// 字典树节点
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
}

// 敏感词检查器
type SensitiveWordChecker struct {
	root *TrieNode
}

var SensitiveChecker *SensitiveWordChecker

// 创建新的敏感词检查器
func (swc *SensitiveWordChecker) New() *SensitiveWordChecker {
	temp := &SensitiveWordChecker{
		root: &TrieNode{
			children: make(map[rune]*TrieNode),
			isEnd:    false,
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

// 添加敏感词到字典树
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

	return false
}

// 找到文本中所有的敏感词
func (swc *SensitiveWordChecker) FindAll(text string) []string {
	var foundWords []string
	text = swc.format(text)
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
				if node.isEnd {
					foundWords = append(foundWords, string(matchedRunes))
					break
				}
				j++
			} else {
				break
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

// 敏感词替换，将敏感词替换为指定字符
func (swc *SensitiveWordChecker) Replace(text string, replacement rune) string {
	runes := []rune(text)
	textLen := len(runes)
	i := 0

	for i < textLen {
		node := swc.root
		j := i
		matchStart := i
		var matchedRunes []rune

		for j < textLen {
			char := unicode.ToLower(runes[j])
			if child, exists := node.children[char]; exists {
				node = child
				matchedRunes = append(matchedRunes, runes[j])
				if node.isEnd {
					// 找到敏感词，进行替换
					for k := matchStart; k < j+1; k++ {
						runes[k] = replacement
					}
					i = j + 1
					break
				}
				j++
			} else {
				i++
				break
			}
		}

		if j >= textLen {
			i++
		}
	}

	return string(runes)
}

// 从文本文件中加载敏感词库,监察库中的内容有变化时，重新加载
func (swc *SensitiveWordChecker) LoadFromTextFile(filepath string) {
	err := swc.LoadFromFileByLine(filepath)
	if err != nil {
		log.Printf("加载敏感词库失败: %v\n", err)
		return
	}
	// 监控词库变化，若词库有变化，则重新加载
	go swc.hotReload(filepath)
}
