# 快速开始 - 谐音过滤功能

## 1. 安装依赖

```bash
go mod tidy
```

这会自动下载 `github.com/mozillazg/go-pinyin` 依赖。

## 2. 基本使用

### 启用谐音模式

```go
package main

import (
    "fmt"
    filter "github.com/abelyi907/sensitive-filter"
)

func main() {
    // 创建检查器
    checker := filter.SensitiveChecker.New()
    
    // 启用谐音过滤
    checker.EnableHomophoneMode()
    
    // 添加敏感词
    checker.Insert("敏感")
    checker.Insert("暴力")
    
    // 检测原文和谐音
    fmt.Println(checker.Contains("敏感内容"))  // true
    fmt.Println(checker.Contains("敏敢内容"))  // true (谐音)
}
```

### 禁用谐音模式

```go
checker.DisableHomophoneMode()
// 此时只检测原文,不检测谐音
```

## 3. 完整示例

```go
package main

import (
    "fmt"
    filter "github.com/abelyi907/sensitive-filter"
)

func main() {
    checker := filter.SensitiveChecker.New()
    checker.EnableHomophoneMode()
    
    // 从文件加载(支持热重载)
    checker.LoadFromTextFile("./words.txt")
    
    // 添加额外敏感词
    checker.Insert("武器")
    checker.Insert("赌博")
    
    text := "这是武气和睹博的内容"
    
    // 检测
    if checker.Contains(text) {
        fmt.Println("包含敏感词!")
        
        // 查找所有敏感词
        words := checker.FindAll(text)
        fmt.Printf("敏感词: %v\n", words)
        
        // 替换
        cleaned := checker.Replace(text, '*')
        fmt.Printf("处理后: %s\n", cleaned)
    }
}
```

## 4. 运行测试

```bash
# 运行所有测试
go test -v

# 仅测试谐音功能
go test -v -run TestHomophone
```

## 5. 常见场景

### 场景1: 社交媒体内容审核

```go
checker := filter.SensitiveChecker.New()
checker.EnableHomophoneMode()
checker.Insert("暴力")
checker.Insert("色情")

// 用户发布内容
content := "这个视频太暴力和色晴了"
if checker.Contains(content) {
    // 拦截或标记
    fmt.Println("内容违规")
}
```

### 场景2: 游戏聊天过滤

```go
checker := filter.SensitiveChecker.New()
checker.EnableHomophoneMode()
checker.LoadFromTextFile("./game-banned-words.txt")

message := "玩家说了一些武器和睹博相关内容"
cleaned := checker.Replace(message, '*')
// 发送清理后的消息
```

### 场景3: 评论系统

```go
checker := filter.SensitiveChecker.New()
checker.EnableHomophoneMode()

comment := "这条评论包含敏感词1和谐音词"
words := checker.FindAll(comment)

if len(words) > 0 {
    fmt.Printf("发现 %d 个敏感词: %v\n", len(words), words)
    // 需要审核
}
```

## 6. 性能提示

- **不需要谐音检测时**: 不要启用谐音模式,保持默认即可
- **高并发场景**: 建议评估性能影响,必要时添加缓存
- **内存优化**: 定期清理不用的检查器实例

## 7. 常见问题

**Q: 谐音模式会影响性能吗?**  
A: 会有一定影响,因为需要额外的拼音转换。但对于大多数应用场景,影响可以接受。

**Q: 会产生误报吗?**  
A: 可能。同音但不同义的词可能被误判。建议根据实际场景调整敏感词库。

**Q: 如何减少误报?**  
A: 
1. 谨慎选择需要启用谐音模式的场景
2. 定期审查检测结果
3. 结合上下文判断

**Q: 支持多语言吗?**  
A: 主要针对中文设计。英文和其他语言也能工作,但谐音检测效果可能不如中文明显。

## 8. 下一步

- 查看 [HOMOPHONE_USAGE.md](HOMOPHONE_USAGE.md) 了解更多细节
- 查看 [README.md](README.md) 了解完整 API
- 运行测试用例学习更多使用场景
