# Sensitive Filter - 敏感词过滤服务

一个基于双字典树（Trie）算法实现的高效敏感词过滤 Go 语言库，支持敏感词检测、查找、替换以及文件热重载功能。

## 功能特性

- ✅ **高效检测**：基于字典树（Trie）数据结构，实现 O(n) 时间复杂度的敏感词匹配
- ✅ **多种操作**：支持敏感词检测、查找、替换等多种操作
- ✅ **文件加载**：支持从文本文件加载敏感词库，每行一个敏感词
- ✅ **热重载**：自动监控敏感词库文件变化，实时重新加载，无需重启服务
- ✅ **大小写不敏感**：自动将文本转换为小写进行匹配
- ✅ **中文支持**：完美支持中文、英文及其他 Unicode 字符
- ✅ **谐音过滤**：支持基于拼音的谐音词检测，有效识别同音字变体

## 安装

```bash
go get github.com/abelyi907/sensitive-filter
```

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
	filter "github.com/abelyi907/sensitive-filter"
)

func main() {
   
	checker := filter.SensitiveChecker.New()  // 创建敏感词检查器
	checker.LoadFromTextFile("./words.txt")   // 从文件加载敏感词库
    
    

    // 测试文本
    testText := "这是一个包含敏感词1和敏感词2内容的文本，涉及信息。"

    // 1. 替换敏感词
    replacedText := checker.Replace(testText, '*')
    fmt.Printf("替换后的文本：%s\n", replacedText)

    // 2. 检测是否包含敏感词
    hasSensitive := checker.Contains(testText)
    fmt.Printf("文本是否包含敏感词：%v\n", hasSensitive)

    // 3. 找出所有敏感词
    foundWords := checker.FindAll(testText)
    fmt.Printf("找到的敏感词：%v\n", foundWords)
}
```

### 谐音过滤功能

启用谐音过滤模式后，系统会自动检测同音字变体：

```go
package main

import (
    "fmt"
	filter "github.com/abelyi907/sensitive-filter"
)

func main() {
    checker := filter.SensitiveChecker.New()
    
    // 启用谐音过滤模式
    checker.EnableHomophoneMode()
    
    // 添加敏感词
    checker.Insert("敏感")
    checker.Insert("暴力")
    
    // 检测原始敏感词
    fmt.Println(checker.Contains("这是敏感内容"))  // true
    
    // 检测谐音词（"敏敢"与"敏感"拼音相同）
    fmt.Println(checker.Contains("这是敏敢内容"))  // true
    
    // 查找所有敏感词（包括谐音）
    words := checker.FindAll("这里有暴力和睹博行为")
    fmt.Println(words)  // ["暴力", "dubo"]
    
    // 替换敏感词和谐音词
    replaced := checker.Replace("不要暴力和睹博", '*')
    fmt.Println(replaced)  // "不要**和**"
    
    // 禁用谐音模式
    checker.DisableHomophoneMode()
    fmt.Println(checker.Contains("这是敏敢内容"))  // false
}
```

### API 说明

#### 1. 创建检查器实例

```go
checker :=  filter.SensitiveChecker.New() 
```

#### 2. 加载敏感词库

从文件加载（支持热重载）：

```go
checker.LoadFromTextFile("path/to/sensitive-words.txt")
```

手动加载（不支持热重载）：

```go
err := checker.LoadFromFileByLine("path/to/sensitive-words.txt")
```

#### 3. 添加单个敏感词

```go
checker.Insert("敏感词")
```

#### 4. 启用/禁用谐音过滤模式

```go
// 启用谐音模式（自动检测同音字）
checker.EnableHomophoneMode()

// 禁用谐音模式
checker.DisableHomophoneMode()
```

#### 5. 检测文本是否包含敏感词

```go
contains := checker.Contains("这段文本包含敏感词")
// 返回：true 或 false
```

#### 6. 查找文本中的所有敏感词

```go
words := checker.FindAll("这段文本包含多个敏感词和违禁词")
// 返回：["敏感词", "违禁词"]
```

#### 7. 替换敏感词

```go
replaced := checker.Replace("这段文本包含敏感词", '*')
// 返回："这段文本包含***"
```

## 核心特性详解

### 🔥 文件热重载

`LoadFromTextFile` 方法会启动一个后台 goroutine，持续监控敏感词库文件的变化。当文件被修改时，会自动重新加载词库，无需重启应用程序。

```go
// 启动后自动监控文件变化
checker.LoadFromTextFile("sensitive-words.txt")

// 当你修改并保存 sensitive-words.txt 文件后
// 系统会自动检测到变化并重新加载词库
// 日志输出：敏感词库已更新，重新加载词库...
```

### 🎯 谐音过滤

启用谐音过滤模式后，系统会将中文转换为拼音进行匹配，从而检测同音字变体。

**工作原理：**
1. 插入敏感词时，同时将其拼音形式存入字典树
2. 检测文本时，同时将原文和拼音形式进行匹配
3. 谐音匹配结果会标记为 `[谐音]` 前缀

**使用示例：**
```go
checker.EnableHomophoneMode()
checker.Insert("敏感")

// 以下都会检测到
checker.Contains("敏感内容")  // true - 原文匹配
checker.Contains("敏敢内容")  // true - 谐音匹配 (min gan)
checker.Contains("民感内容")  // true - 谐音匹配 (min gan)
```

**注意事项：**
- 谐音模式会增加内存占用（存储拼音形式）
- 可能会产生误报（同音但不同义的词）
- 建议根据实际场景选择是否启用

### 📝 文件格式要求

敏感词库文件应为纯文本格式，每行一个敏感词：

```
敏感词1
敏感词2
敏感词3
```

### ⚙️ 文本预处理

所有输入的文本和敏感词都会经过以下处理：

- 转换为小写（不区分大小写）
- 去除首尾空白字符

这确保了匹配的准确性和一致性。

## 性能特点

- **时间复杂度**：O(n)，其中 n 为文本长度
- **空间复杂度**：O(m)，其中 m 为所有敏感词的总字符数
- **并发安全**：当前版本未做并发保护，如在并发场景使用，请自行添加锁机制

## 项目结构

```
sensitive-filter/
├── services.go          # 核心实现代码
└── README.md            # 项目说明文档
```

## 使用场景

- 📱 社交媒体内容审核
- 💬 即时通讯消息过滤
- 📰 新闻评论管理
- 🎮 游戏聊天系统
- 📚 用户生成内容（UGC）平台
- 🔍 任何需要文本过滤的场景

## 注意事项

1. **并发使用**：如果在高并发场景下使用，建议在外部添加读写锁保护
2. **内存占用**：敏感词库较大时，会占用较多内存，请根据实际场景评估
3. **文件路径**：使用 Windows 路径时，注意使用正斜杠 `/` 或双反斜杠 `\\`
4. **热重载延迟**：文件监控有 2 秒的检测间隔，修改文件后最多 2 秒内生效
5. **谐音模式**：启用谐音模式会增加内存占用和可能的误报，建议根据实际需求选择

## 示例代码

完整示例：

```go
package main

import (
    "fmt"
    "log"
   filter "github.com/abelyi907/sensitive-filter"
)

func main() {
    // 创建敏感词检查器
    checker := filter.SensitiveChecker.New()
    
    // 加载敏感词库（带热重载）
    err := checker.LoadFromTextFile("./config/sensitive-words.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    // 手动添加一些敏感词
    checker.Insert("广告")
    checker.Insert("推销")
    
    // 待检测文本
    texts := []string{
        "你好，这是一个测试",
        "这里包含广告和推销内容",
        "Normal text without issues",
    }
    
    // 批量处理
    for _, text := range texts {
        fmt.Printf("\n原文本：%s\n", text)
        
        // 检测
        if checker.Contains(text) {
            fmt.Println("⚠️  包含敏感词")
            
            // 查找所有敏感词
            words := checker.FindAll(text)
            fmt.Printf("敏感词列表：%v\n", words)
            
            // 替换
            cleaned := checker.Replace(text, '*')
            fmt.Printf("处理后：%s\n", cleaned)
        } else {
            fmt.Println("✅ 内容安全")
        }
    }
}
```

## 许可证

[MIT License](LICENSE)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题或建议，请通过以下方式联系：
- Email: yybjroam@qq.com
- GitHub Issues: [提交 Issue](https://github.com/your-username/sensitive-filter/issues)
