# 谐音过滤功能使用指南

## 新增功能说明

本次更新为敏感词过滤库增加了**谐音过滤功能**,可以检测同音字变体,有效防止通过谐音字绕过敏感词过滤。

## 快速开始

### 1. 安装依赖

在使用谐音过滤功能前,需要先安装拼音转换库:

```bash
go mod tidy
```

或者手动添加依赖:

```bash
go get github.com/mozillazg/go-pinyin
```

### 2. 基本使用

```go
package main

import (
    "fmt"
    filter "github.com/abelyi907/sensitive-filter"
)

func main() {
    // 创建检查器
    checker := filter.SensitiveChecker.New()
    
    // 启用谐音过滤模式
    checker.EnableHomophoneMode()
    
    // 添加敏感词
    checker.Insert("敏感")
    checker.Insert("暴力")
    
    // 检测原文
    fmt.Println(checker.Contains("这是敏感内容"))  // true
    
    // 检测谐音词 ("敏敢"与"敏感"拼音相同)
    fmt.Println(checker.Contains("这是敏敢内容"))  // true
    
    // 查找所有敏感词(包括谐音)
    words := checker.FindAll("这里有暴力和睹博行为")
    fmt.Println(words)  // ["暴力", "[谐音]dubo"]
    
    // 替换敏感词和谐音词
    replaced := checker.Replace("不要暴力和睹博", '*')
    fmt.Println(replaced)  // "不要**和**"
}
```

## API 说明

### 启用/禁用谐音模式

```go
// 启用谐音过滤模式
checker.EnableHomophoneMode()

// 禁用谐音过滤模式
checker.DisableHomophoneMode()
```

### 工作原理

1. **插入阶段**: 当启用谐音模式后,调用 `Insert()` 插入敏感词时,系统会:
   - 将原始敏感词插入字典树
   - 同时将敏感词的拼音形式也插入字典树(标记为谐音词)

2. **检测阶段**: 调用 `Contains()`, `FindAll()`, `Replace()` 时,系统会:
   - 首先检测原文本中的敏感词
   - 如果启用谐音模式,同时将文本转换为拼音进行检测
   - 谐音匹配结果会标记 `[谐音]` 前缀

### 示例场景

```go
checker := filter.SensitiveChecker.New()
checker.EnableHomophoneMode()
checker.Insert("武器")

// 以下都会检测到:
checker.Contains("武器")    // true - 原文
checker.Contains("武气")    // true - 谐音 (wu qi)
checker.Contains("五器")    // true - 谐音 (wu qi)
```

## 注意事项

### 优点
- ✅ 有效识别同音字变体,提高过滤准确率
- ✅ 自动处理,无需手动添加所有谐音变体
- ✅ 可灵活启用/禁用,不影响原有功能

### 限制
- ⚠️ 增加内存占用(需要存储拼音形式)
- ⚠️ 可能产生误报(同音但不同义的词)
- ⚠️ 性能略有下降(需要额外进行拼音转换)

### 建议
- 根据实际业务场景选择是否启用谐音模式
- 对于高并发场景,建议评估性能影响
- 定期检查误报情况,必要时调整策略

## 运行测试

```bash
# 运行所有测试
go test -v

# 仅运行谐音过滤测试
go test -v -run TestHomophoneFilter

# 运行性能测试
go test -v -run TestHomophonePerformance
```

## 完整示例

查看 `services_test.go` 文件中的测试用例,了解更多使用场景:

- `TestHomophoneFilter`: 基本谐音过滤功能测试
- `TestMultipleHomophones`: 多个同音字测试
- `TestHomophonePerformance`: 性能测试

## 技术实现

- 使用 `github.com/mozillazg/go-pinyin` 进行中文转拼音
- 在 Trie 节点中添加 `isHomophone` 标记区分原文和谐音
- 检测时分别匹配原文和拼音两种形式
- 保持向后兼容,默认不启用谐音模式
