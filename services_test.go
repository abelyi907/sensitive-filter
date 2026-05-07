package sensitive_filter

import (
	"os"
	"testing"
	"time"
)

// 初始化内容.测试完成后，如何还原始敏感词列表
func Init() func() {
	fileName := "words.txt"
	content := []string{"敏感词1", "敏感词2", "敏感词3"}
	_ = ClearFileContent(fileName, func() {
		writeFile(fileName, content)
	})
	return func() {
		_ = ClearFileContent(fileName, func() {
			writeFile(fileName, content)
		})
	}
}
func ClearFileContent(filePath string, f func()) error {
	// 以写入模式打开文件，如果文件不存在则创建，如果存在则清空
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	f()
	return nil
}

func writeFile(filename string, lines []string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for _, line := range lines {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			panic(err)
		}
	}
}

func implodeStr(arr []string) string {
	if len(arr) == 0 {
		return ""
	}
	result := ""
	for i := 0; i < len(arr); i++ {
		if i > 0 {
			result += ","
		}
		result += arr[i]
	}
	return result
}

// 追加一行到文件
func appendLine(filename, text string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + text)
	return err
}

func TestReload(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.LoadFromTextFile("words.txt")
	if checker.Contains("测试非词") {
		t.Fatal("报错，不应该检测到敏感词")
	}

	_ = appendLine("words.txt", "测试非词")
	time.Sleep(time.Second * 4)
	//向words.txt文件中添加敏感词"测试敏感词"，然后等待2秒钟，测试是否能够检测到新添加的敏感词

	if !checker.Contains("测试非词") {
		t.Fatal("重新加载敏感词失败")
	}

}

// 测试谐音过滤功能
func TestHomophoneFilter(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()

	// 启用谐音模式
	//checker.EnableHomophoneMode()

	// 插入一些敏感词
	checker.Insert("敏感")
	checker.Insert("飞鸣")
	checker.Insert("读宽")

	// 测试1: 检测原文本中的敏感词
	if !checker.Contains("这是敏感内容") {
		t.Fatal("检测敏感词失败")
	}

	// 测试2: 检测谐音词（"敏敢"与"敏感"拼音相同）
	if !checker.Contains("这是敏敢内容") {
		t.Fatal("检查谐音敏感词失败")
	}

	if !checker.Contains("这是dukuan内容") {
		t.Fatal("检查谐音敏感词失败!")
	}

	// 测试3: 查找所有敏感词（包括谐音）
	words := checker.FindAll("这里有飞鸣和睹博行为")
	if len(words) == 0 {
		t.Fatal("查找敏感词或谐音词失败")
	}

	// 测试4: 替换敏感词和谐音词
	replaced := checker.Replace("不要飞鸣和飞明,对feiming", '*')

	if replaced != "不要**和**,对*******" {
		t.Fatal("替换谐音敏感词失败:" + replaced)
	}

	replaced = checker.Replace("不要飞-ming和飞-明", '*')
	t.Log("==>", replaced)
	if replaced != "不要*-****和*-*" {
		t.Fatal("替换变形敏感词失败")
	}

	// 测试6: 检测变形词
	if !checker.Contains("飞-ming") {
		t.Fatal("检测变形词失败")
	}

	if !checker.Contains("飞_鸣") {
		t.Fatal("检测变形词失败")
	}

	// 测试7: 查找变形词
	words = checker.FindAll("这里有飞-ming和飞~明行为")
	if len(words) == 0 {
		t.Fatal("查找变形词失败")
	}

	// 测试8: 禁用变形词模式后，不应该检测到变形词
	checker.DisableDeformMode()
	if checker.Contains("飞-ming") {
		t.Fatal("禁用变形词模式后还是检查到变形词了")
	}

	// 但仍然应该检测到原文
	if !checker.Contains("飞鸣") {
		t.Fatal("应该检测到原始敏感词")
	}

	// 重新启用变形词模式
	checker.EnableDeformMode()

	// 测试5: 禁用谐音模式后，不应该检测到谐音词
	checker.DisableHomophoneMode()
	if checker.Contains("这是敏敢内容") {
		t.Fatal("禁用谐音模式后还是检查到谐音词了")
	}

	// 但仍然应该检测到原文
	if !checker.Contains("这是敏感内容") {
		t.Fatal("应该检测到原始敏感词")
	}
}

// 测试谐音模式下的多个同音字
func TestMultipleHomophones(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.EnableHomophoneMode()

	// 插入敏感词
	checker.Insert("武器")

	// 测试不同的同音字组合
	testCases := []struct {
		text     string
		expected bool
	}{
		{"这是武器", true},   // 原文
		{"这是武气", true},   // 谐音
		{"这是五器", true},   // 谐音
		{"这是无害的", false}, // 不相关
	}

	for _, tc := range testCases {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}
}

// 测试变形词过滤功能
func TestDefaultFilter(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()

	// 插入敏感词
	checker.Insert("飞鸣")
	checker.Insert("读宽")
	checker.Insert("武器")
	checker.Insert("张三四")

	// 测试1: 检测各种分隔符的变形词
	testCases := []struct {
		text     string
		expected bool
	}{
		{"飞鸣", true},           // 正常词
		{"飞-鸣", true},          // 横线分隔
		{"飞_鸣", true},          // 下划线分隔
		{"飞.鸣", true},          // 点号分隔
		{"飞~鸣", true},          // 波浪线分隔
		{"飞~ming", true},       // 波浪线分隔
		{"fei~鸣", true},        // 波浪线分隔
		{"fei~ming", true},     // 波浪线分隔
		{"武-器", true},          // 另一个词
		{"五-器", true},          // 另一个词
		{"zhang-san-si", true}, // 另一个词
		{"zhang-三-si", true},   // 另一个词
		{"无害的", false},         // 不相关
	}

	for _, tc := range testCases {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}

	// 测试2: 替换变形词（保留分隔符）
	test2Cases := []struct {
		text       string
		expectText string
	}{
		{"飞鸣", "**"},             // 正常词
		{"飞-鸣", "*-*"},           // 横线分隔
		{"飞_鸣", "*_*"},           // 下划线分隔
		{"飞.鸣", "*.*"},           // 点号分隔
		{"飞~ming", "*~****"},     // 波浪线分隔
		{"fei~鸣", "***~*"},       // 波浪线分隔
		{"fei~ming", "***~****"}, // 波浪线分隔
		{"武-器", "*-*"},           // 另一个词
		{"五-器", "*-*"},           // 另一个词
		{"张-san-si", "*-***-**"}, // 另一个词
		{"张-三-si", "*-*-**"},     // 另一个词
		{"无害的", "无害的"},           // 不相关
	}

	for _, tc := range test2Cases {
		result := checker.Replace(tc.text, '*')
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

	// 测试3: 查找变形词
	test3Cases := []struct {
		text       string
		expectText string
	}{
		{"飞鸣", "飞鸣"},  // 正常词
		{"飞-鸣", "飞鸣"}, // 横线分隔
		{"飞_鸣", "飞鸣"}, // 下划线分隔
		{"飞.鸣", "飞鸣"}, // 点号分隔

		{"fei~鸣", "feiming"},           // 波浪线分隔
		{"fei~ming", "feiming"},        // 波浪线分隔
		{"武-器", "武器"},                  // 另一个词
		{"五-器", "wuqi"},                // 另一个词
		{"张-san-si", "zhangsansi"},     // 另一个词
		{"zhang-san-si", "zhangsansi"}, // 另一个词
		{"张sansi", "zhangsansi"},       // 另一个词
		{"zhang三si", "zhangsansi"},     // 另一个词
		{"无害的", ""},                    // 不相关
	}

	for _, tc := range test3Cases {
		result := implodeStr(checker.FindAll(tc.text))
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

}
func TestDisableDeformModeFilter(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.DisableDeformMode()
	//checker.DisableHomophoneMode()
	// 插入敏感词
	checker.Insert("飞鸣")
	checker.Insert("读宽")
	checker.Insert("武器")

	// 测试1: 检测各种分隔符的变形词
	testCases := []struct {
		text     string
		expected bool
	}{
		{"飞鸣", true},        // 正常词
		{"飞-鸣", false},      // 横线分隔
		{"飞_鸣", false},      // 下划线分隔
		{"飞.鸣", false},      // 点号分隔
		{"飞~鸣", false},      // 波浪线分隔
		{"飞~ming", false},   // 波浪线分隔
		{"fei~鸣", false},    // 波浪线分隔
		{"fei~ming", false}, // 波浪线分隔
		{"武-器", false},      // 另一个词
		{"五-器", false},      // 另一个词
		{"五器", true},        // 另一个词
		{"无害的", false},      // 不相关
	}

	for _, tc := range testCases {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}

	// 测试2: 替换变形词（保留分隔符）
	test2Cases := []struct {
		text       string
		expectText string
	}{
		{"飞鸣", "**"},             // 正常词
		{"飞-鸣", "飞-鸣"},           // 横线分隔
		{"飞_鸣", "飞_鸣"},           // 下划线分隔
		{"飞.鸣", "飞.鸣"},           // 点号分隔
		{"飞~ming", "飞~ming"},     // 波浪线分隔
		{"fei~鸣", "fei~鸣"},       // 波浪线分隔
		{"fei~ming", "fei~ming"}, // 波浪线分隔
		{"武-器", "武-器"},           // 另一个词
		{"五-器", "五-器"},           // 另一个词
		{"无害的", "无害的"},           // 不相关
	}

	for _, tc := range test2Cases {
		result := checker.Replace(tc.text, '*')
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

	// 测试3: 查找变形词
	test3Cases := []struct {
		text       string
		expectText string
	}{
		{"飞鸣", "飞鸣"}, // 正常词
		{"飞-鸣", ""},  // 横线分隔
		{"飞_鸣", ""},  // 下划线分隔
		{"飞.鸣", ""},  // 点号分隔

		{"fei~ming", ""}, // 波浪线分隔
		{"fei~ming", ""}, // 波浪线分隔
		{"武-器", ""},      // 另一个词
		{"五-器", ""},      // 另一个词
		{"无害的", ""},      // 不相关
	}

	for _, tc := range test3Cases {
		result := implodeStr(checker.FindAll(tc.text))
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

}

func TestDisableHomophoneModeFilter(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.DisableDeformMode()
	checker.DisableHomophoneMode()
	// 插入敏感词
	checker.Insert("飞鸣")
	checker.Insert("读宽")
	checker.Insert("武器")

	// 测试1: 检测各种分隔符的变形词
	testCases := []struct {
		text     string
		expected bool
	}{
		{"飞鸣", true},        // 正常词
		{"飞-ming", false},   // 横线分隔
		{"飞~ming", false},   // 波浪线分隔
		{"fei~ming", false}, // 波浪线分隔
		{"fei~ming", false}, // 波浪线分隔
		{"武-器", false},      // 另一个词
		{"五-器", false},      // 另一个词
		{"五器", false},       // 另一个词
		{"无害的", false},      // 不相关
	}

	for _, tc := range testCases {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}

	// 测试2: 替换变形词（保留分隔符）
	test2Cases := []struct {
		text       string
		expectText string
	}{
		{"飞鸣", "**"},             // 正常词
		{"飞-鸣", "飞-鸣"},           // 横线分隔
		{"飞_鸣", "飞_鸣"},           // 下划线分隔
		{"飞.鸣", "飞.鸣"},           // 点号分隔
		{"飞~ming", "飞~ming"},     // 波浪线分隔
		{"fei~鸣", "fei~鸣"},       // 波浪线分隔
		{"fei~ming", "fei~ming"}, // 波浪线分隔
		{"武-器", "武-器"},           // 另一个词
		{"五-器", "五-器"},           // 另一个词
		{"无害的", "无害的"},           // 不相关
	}

	for _, tc := range test2Cases {
		result := checker.Replace(tc.text, '*')
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

	// 测试3: 查找变形词
	test3Cases := []struct {
		text       string
		expectText string
	}{
		{"飞鸣", "飞鸣"}, // 正常词
		{"飞-鸣", ""},  // 横线分隔
		{"飞_鸣", ""},  // 下划线分隔
		{"飞.鸣", ""},  // 点号分隔

		{"fei~鸣", ""},    // 波浪线分隔
		{"fei~ming", ""}, // 波浪线分隔
		{"武-器", ""},      // 另一个词
		{"五-器", ""},      // 另一个词
		{"无害的", ""},      // 不相关
	}

	for _, tc := range test3Cases {
		result := implodeStr(checker.FindAll(tc.text))
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

}

func TestHomophonePerformance(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()

	// 插入大量敏感词
	for i := 0; i < 100; i++ {
		checker.Insert("敏感词" + string(rune(i)))
	}

	testText := "这是一段测试文本，包含一些敏感词和普通内容"

	// 执行多次检测
	for i := 0; i < 1000; i++ {
		checker.Contains(testText)
	}
}

// 测试火星文过滤功能
func TestMartianFilter(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()

	// 插入敏感词
	checker.Insert("不要")
	checker.Insert("飞鸣")
	checker.Insert("喜欢")
	checker.Insert("知道")
	checker.Insert("什么")
	checker.Insert("没有")

	// 测试1: 检测单字火星文
	t.Log("=== 测试单字火星文 ===")
	testCases1 := []struct {
		text     string
		expected bool
	}{
		{"卜要", true},    // "卜"->"不"，"不要"是敏感词
		{"滴飞鸣", true},   // "滴"->"的"，"的飞鸣"中包含"飞鸣"
		{"莪喜欢", true},   // "莪"->"我"，"我喜欢"中包含"喜欢"
		{"造了", true},    // "造"->"知道"，"知道了"中包含"知道"
		{"神马", true},    // "神马"->"什么"，"什么"是敏感词
		{"木有", true},    // "木有"->"没有"，"没有"是敏感词
		{"正常文字", false}, // 不包含任何敏感词
	}

	for _, tc := range testCases1 {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}

	// 测试2: 检测词语火星文
	t.Log("=== 测试词语火星文 ===")
	testCases2 := []struct {
		text     string
		expected bool
	}{
		{"稀饭你", true},  // "喜欢" -> "稀饭"，"喜欢"是敏感词
		{"酱紫啊", false}, // "这样子" -> "酱紫"，但"这样子"不是敏感词
		{"灰常好", false}, // "非常" 不在敏感词中
		{"童鞋们", false}, // "同学" 不在敏感词中
	}

	for _, tc := range testCases2 {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}

	// 测试3: 查找火星文
	t.Log("=== 测试查找火星文 ===")
	words := checker.FindAll("卜要飞鸣，莪稀饭神马")
	if len(words) == 0 {
		t.Error("查找火星文失败")
	}
	t.Logf("找到的词: %v", words)

	// 测试4: 替换火星文
	t.Log("=== 测试替换火星文 ===")
	replaced := checker.Replace("卜要飞鸣，莪稀饭神马", '*')
	t.Logf("替换结果: %s", replaced)
	if replaced == "" {
		t.Error("替换火星文失败")
	}

	// 测试5: 禁用火星文模式
	t.Log("=== 测试禁用火星文模式 ===")
	checker.DisableMartianMode()
	if checker.Contains("卜要") {
		t.Error("禁用火星文模式后仍检测到火星文")
	}

	// 重新启用
	checker.EnableMartianMode()
	if !checker.Contains("卜要") {
		t.Error("启用火星文模式后应该检测到火星文")
	}

	// 测试6: 火星文+变形词组合
	t.Log("=== 测试火星文+变形词组合 ===")
	checker.Insert("飞鸣")
	if !checker.Contains("飞-明") {
		t.Error("检测谐音+变形词失败")
	}

	// 测试7: 火星文+谐音组合
	t.Log("=== 测试火星文+谐音组合 ===")
	if !checker.Contains("卜要feiming") {
		t.Error("检测火星文+谐音失败")
	}
}

// 测试火星文标准化函数
func TestNormamingzeMartian(t *testing.T) {
	defer Init()()
	testCases := []struct {
		input    string
		expected string
	}{
		{"卜要", "不要"},
		{"滴", "的"},
		{"莪稀饭神马", "我喜欢什么"},
		{"酱紫", "这样子"},
		{"肿么木有", "怎么没有"},
		{"正常文字", "正常文字"},
		{"8要", "不要"},
		{"3Q", "谢谢"},
		{"88", "拜拜"},
	}

	for _, tc := range testCases {
		result := normalizeMartian(tc.input)
		if result != tc.expected {
			t.Errorf("输入 '%s' 期望 '%s', 实际 '%s'", tc.input, tc.expected, result)
		}
	}
}

// 测试火星文性能
func TestMartianPerformance(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()

	// 插入敏感词
	for i := 0; i < 100; i++ {
		checker.Insert("敏感词" + string(rune('A'+i%26)))
	}

	testText := "这是一段测试文本，包含卜要、滴、莪等火星文"

	// 执行多次检测
	for i := 0; i < 1000; i++ {
		checker.Contains(testText)
	}
}

// 测试自定义火星文映射
func TestCustomMartianMapping(t *testing.T) {
	// 添加自定义映射
	AddMartianMapping("偶滴", "我的")
	AddMartianMapping("表酱紫", "不要这样子")

	// 验证映射已添加
	mappings := GetMartianMappings()
	if mappings["偶滴"] != "我的" {
		t.Error("自定义火星文映射添加失败")
	}

	// 测试标准化
	result := normalizeMartian("偶滴神马")
	if result != "我的什么" {
		t.Errorf("期望 '我的什么', 实际 '%s'", result)
	}

	result = normalizeMartian("表酱紫")
	if result != "不要这样子" {
		t.Errorf("期望 '不要这样子', 实际 '%s'", result)
	}

	// 移除映射
	RemoveMartianMapping("偶滴")
	mappings = GetMartianMappings()
	if _, exists := mappings["偶滴"]; exists {
		t.Error("移除火星文映射失败")
	}
}

// TestNewTrieNode 验证字典树节点构造
func TestNewTrieNode(t *testing.T) {
	defer Init()()
	node := NewTrieNode()
	if node == nil {
		t.Fatal("NewTrieNode() 返回 nil")
	}
	if node.children == nil {
		t.Fatal("新节点的 children 不应为 nil")
	}
	if len(node.children) != 0 {
		t.Fatal("新节点的 children 应为空")
	}
	if node.isEnd {
		t.Fatal("新节点的 isEnd 应初始化为 false")
	}
	if node.isHomophone {
		t.Fatal("新节点的 isHomophone 应初始化为 false")
	}
}

// TestDefaultModes 验证默认模式配置
func TestDefaultModes(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	if !checker.homophoneMode {
		t.Error("homophoneMode 应默认启用")
	}
	if !checker.deformMode {
		t.Error("deformMode 应默认启用")
	}
	if !checker.martianMode {
		t.Error("martianMode 应默认启用")
	}
}

// TestContainsEdgeCases Contains 边界情况测试
func TestContainsEdgeCases(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.Insert("敏感")

	// 空字符串
	if checker.Contains("") {
		t.Error("空字符串不应包含敏感词")
	}

	// 纯分隔符
	if checker.Contains("-~_.") {
		t.Error("纯分隔符不应包含敏感词")
	}

	// 单字符不匹配
	if checker.Contains("敏") {
		t.Error("单字符不应匹配完整敏感词")
	}

	// 完全不相关文本
	if checker.Contains("完全正常的文字") {
		t.Error("不相关文本不应包含敏感词")
	}

	// 超长文本（分隔符的情况）
	if checker.Contains("") {
		t.Error("空字符串不应包含敏感词")
	}
}

// TestFindAllEdgeCases FindAll 边界情况测试
func TestFindAllEdgeCases(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.Insert("敏感")
	checker.Insert("敏感词")
	checker.Insert("赌")

	// 空文本
	words := checker.FindAll("")
	if len(words) != 0 {
		t.Error("空文本的 FindAll 应返回空切片")
	}

	// 无匹配
	words = checker.FindAll("正常文字")
	if len(words) != 0 {
		t.Error("无匹配时应返回空切片")
	}

	// 嵌套敏感词：插入"赌"和"赌博"，检测"赌博"时应同时找到两个
	checker2 := SensitiveChecker.New()
	checker2.Insert("赌")
	checker2.Insert("赌博")
	words = checker2.FindAll("赌博")
	if len(words) != 1 {
		t.Errorf("'赌博' 应找到 1 个敏感词, 实际找到 %d: %v", len(words), words)
	}
	// FindAll 返回"赌博"（较长的匹配优先）

	// 多个不同敏感词（禁用谐音模式，只匹配原文）
	checker3 := SensitiveChecker.New()
	checker3.DisableHomophoneMode()
	checker3.Insert("飞鸣")
	checker3.Insert("读宽")
	checker3.Insert("武器")
	words = checker3.FindAll("飞鸣读宽武器")
	if len(words) != 3 {
		t.Errorf("应找到 3 个敏感词, 实际找到 %d: %v", len(words), words)
	}
	// 验证具体匹配的词
	for _, w := range words {
		if w != "飞鸣" && w != "读宽" && w != "武器" {
			t.Errorf("意外的匹配词: '%s'", w)
		}
	}
}

// TestReplaceEdgeCases Replace 边界情况测试
func TestReplaceEdgeCases(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.Insert("敏感")

	// 空文本
	result := checker.Replace("", '*')
	if result != "" {
		t.Errorf("空文本替换应返回空, 实际 '%s'", result)
	}

	// 无匹配文本，应保持原样
	result = checker.Replace("正常文字", '*')
	if result != "正常文字" {
		t.Errorf("无匹配时应返回原文, 实际 '%s'", result)
	}

	// 单字符替换
	result = checker.Replace("敏", '*')
	if result != "敏" {
		t.Errorf("单字符不应被替换, 实际 '%s'", result)
	}

	// 替换字符与原字符相同
	result = checker.Replace("敏感", '敏')
	if result != "敏敏" {
		// 替换字符为 '*'
		t.Errorf("期望 '敏敏', 实际 '%s'", result)
	}

	// 替换为Unicode字符
	result = checker.Replace("敏感", '□')
	if result != "□□" {
		t.Errorf("期望 '□□', 实际 '%s'", result)
	}
}

// TestCaseInsensitivity 大小写不敏感测试
func TestCaseInsensitivity(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.EnableHomophoneMode()
	checker.Insert("读宽")

	// 英文大小写
	tests := []struct {
		text     string
		expected bool
	}{
		{"dukuan", true},
		{"DUKUAN", true},
		{"DuKuan", true},
		{"dUkUAn", true},
	}

	for _, tc := range tests {
		result := checker.Contains(tc.text)
		if result != tc.expected {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expected, result)
		}
	}
}

// TestNestedSensitiveWords 嵌套/重叠敏感词测试
func TestNestedSensitiveWords(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.Insert("读")
	checker.Insert("读宽")
	checker.Insert("读宽网站")

	// Contains 应检测到最短匹配即可
	if !checker.Contains("读宽网站") {
		t.Error("'读宽网站' 应包含敏感词")
	}

	if !checker.Contains("读宽") {
		t.Error("'读宽' 应包含敏感词")
	}

	if !checker.Contains("读") {
		t.Error("'读' 应包含敏感词")
	}

	// 替换应替换整个最长匹配
	replaced := checker.Replace("读宽网站", '*')
	if replaced == "" {
		t.Error("替换结果不应为空")
	}

	// 变形模式下的嵌套
	if !checker.Contains("读~宽") {
		t.Error("'读~宽' 应包含敏感词")
	}
}

// TestLoadFromFileByLine 从文件加载敏感词测试
func TestLoadFromFileByLine(t *testing.T) {
	defer Init()()
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "sensitive-*.txt")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入测试数据
	content := "测试敏感词1\n测试敏感词2\n测试敏感词3\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	checker := SensitiveChecker.New()
	err = checker.LoadFromFileByLine(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadFromFileByLine 失败: %v", err)
	}

	// 验证加载的敏感词
	if !checker.Contains("测试敏感词1") {
		t.Error("从文件加载的 '测试敏感词1' 未检测到")
	}
	if !checker.Contains("测试敏感词2") {
		t.Error("从文件加载的 '测试敏感词2' 未检测到")
	}
	if !checker.Contains("测试敏感词3") {
		t.Error("从文件加载的 '测试敏感词3' 未检测到")
	}

	// 验证未加载的词不应该被检测到
	if checker.Contains("不存在的词") {
		t.Error("不应该检测到未加载的词")
	}

	// 测试加载不存在的文件
	err = checker.LoadFromFileByLine("不存在的文件.txt")
	if err == nil {
		t.Error("加载不存在的文件应返回错误")
	}
}

// TestInsertEdgeCases Insert 边界情况
func TestInsertEdgeCases(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()

	// 插入空字符串（不应 panic）
	checker.Insert("")
	checker.Insert("   ")

	// 插入后，不应检测到任何内容
	if checker.Contains("") {
		t.Error("未插入任何词，空字符串不应包含敏感词")
	}

	// 插入带空格的词，format 会 trim
	checker.Insert("  新敏感词  ")
	if !checker.Contains("新敏感词") {
		t.Error("插入带空格的词后应能检测到 trim 后的词")
	}

	// 插入纯英文词
	checker.Insert("testword")
	if !checker.Contains("testword") {
		t.Error("纯英文词插入后应能检测到")
	}

	// 插入数字
	checker.Insert("12345")
	if !checker.Contains("12345") {
		t.Error("数字词插入后应能检测到")
	}
}

// TestReplaceWithMultipleMatches 多匹配替换测试
func TestReplaceWithMultipleMatches(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.Insert("飞鸣")
	checker.Insert("读宽")

	result := checker.Replace("飞鸣和读宽", '*')
	if result != "**和**" {
		t.Errorf("期望 '**和**', 实际 '%s'", result)
	}

	// 变形模式下的多匹配替换
	result = checker.Replace("飞-鸣和读_宽", '*')
	if result != "*-*和*_*" {
		t.Errorf("期望 '*-*和*_*', 实际 '%s'", result)
	}
}

// TestHomophoneWithDeformDisabled 谐音模式下禁用变形模式
func TestHomophoneWithDeformDisabled(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.DisableDeformMode()
	checker.EnableHomophoneMode()
	checker.Insert("读宽")

	// 纯原文应该匹配
	if !checker.Contains("读宽") {
		t.Error("禁用变形模式后应匹配原文")
	}

	// 纯拼音应该匹配
	if !checker.Contains("dukuan") {
		t.Error("禁用变形模式后应匹配拼音")
	}

	// 含分隔符的变形拼音不应匹配
	if checker.Contains("du-kuan") {
		t.Error("禁用变形模式后不应匹配含分隔符的拼音")
	}

	// 含分隔符的中文不应匹配
	if checker.Contains("读-宽") {
		t.Error("禁用变形模式后不应匹配含分隔符的中文")
	}
}

// TestFormatBehavior 测试 format 函数行为（归一化）
func TestFormatBehavior(t *testing.T) {
	defer Init()()
	checker := SensitiveChecker.New()
	checker.DisableMartianMode()
	checker.Insert("TestWord")

	// format 会将输入转为小写
	if !checker.Contains("TESTWORD") {
		t.Error("Contains 应进行大小写归一化")
	}

	// format 会 trim 空格
	if !checker.Contains("  TestWord  ") {
		t.Error("Contains 应 trim 空格")
	}
}
