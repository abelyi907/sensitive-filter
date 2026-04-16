# 更新日志

## [Unreleased] - 2026-04-16

### Added - 新增功能

#### 谐音过滤功能
- ✅ 添加基于拼音的谐音词检测功能
- ✅ 新增 `EnableHomophoneMode()` 方法启用谐音模式
- ✅ 新增 `DisableHomophoneMode()` 方法禁用谐音模式
- ✅ 增强 `Contains()` 方法支持谐音检测
- ✅ 增强 `FindAll()` 方法支持查找谐音词
- ✅ 增强 `Replace()` 方法支持替换谐音词
- ✅ 在 TrieNode 中添加 `isHomophone` 标记区分原文和谐音
- ✅ 添加 `textToPinyin()` 内部方法用于中文转拼音
- ✅ 添加 `insertPinyin()` 内部方法用于插入拼音形式

#### 依赖管理
- ✅ 添加 `github.com/mozillazg/go-pinyin` 依赖用于中文转拼音
- ✅ 更新 `go.mod` 文件

#### 文档
- ✅ 更新 README.md 添加谐音过滤功能说明和使用示例
- ✅ 创建 HOMOPHONE_USAGE.md 详细使用指南
- ✅ 添加 API 文档说明

#### 测试
- ✅ 添加 `TestHomophoneFilter` 测试基本谐音过滤功能
- ✅ 添加 `TestMultipleHomophones` 测试多个同音字场景
- ✅ 添加 `TestHomophonePerformance` 性能测试

### Changed - 变更

- 扩展 `TrieNode` 结构,添加 `isHomophone` 字段
- 扩展 `SensitiveWordChecker` 结构,添加 `homophoneMode` 和 `pinyinArgs` 字段
- 修改 `Insert()` 方法,在谐音模式下自动插入拼音形式
- 优化 `Contains()`, `FindAll()`, `Replace()` 方法支持双重检测(原文+拼音)

### Technical Details - 技术细节

**工作原理:**
1. 启用谐音模式后,插入敏感词时同时存储原文和拼音两种形式
2. 检测时将文本也转换为拼音进行匹配
3. 谐音匹配结果会标记 `[谐音]` 前缀以便区分
4. 默认不启用谐音模式,保持向后兼容

**性能影响:**
- 内存占用增加约 50-100%(存储拼音形式)
- 检测时间略有增加(需要额外的拼音转换)
- 建议在需要防谐音绕过时启用

### Backward Compatibility - 向后兼容

- ✅ 完全向后兼容,默认不启用谐音模式
- ✅ 现有代码无需修改即可正常工作
- ✅ 可选择性启用谐音功能

### Usage Example - 使用示例

```go
checker := filter.SensitiveChecker.New()
checker.EnableHomophoneMode()
checker.Insert("敏感")

// 以下都会检测到
checker.Contains("敏感内容")  // true - 原文
checker.Contains("敏敢内容")  // true - 谐音 (min gan)
```

### Notes - 注意事项

- 使用前需要运行 `go mod tidy` 安装依赖
- 谐音模式可能产生误报,建议根据实际场景评估
- 高并发场景下建议评估性能影响
