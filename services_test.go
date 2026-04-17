package sensitive_filter

import (
	"testing"
)

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

// 测试谐音过滤功能
func TestHomophoneFilter(t *testing.T) {
	checker := SensitiveChecker.New()

	// 启用谐音模式
	//checker.EnableHomophoneMode()

	// 插入一些敏感词
	checker.Insert("敏感")
	checker.Insert("暴力")
	checker.Insert("赌博")

	// 测试1: 检测原文本中的敏感词
	if !checker.Contains("这是敏感内容") {
		t.Fatal("检测敏感词失败")
	}

	// 测试2: 检测谐音词（"敏敢"与"敏感"拼音相同）
	if !checker.Contains("这是敏敢内容") {
		t.Fatal("检查谐音敏感词失败")
	}

	if !checker.Contains("这是dubo内容") {
		t.Fatal("检查谐音敏感词失败!")
	}

	// 测试3: 查找所有敏感词（包括谐音）
	words := checker.FindAll("这里有暴力和睹博行为")
	if len(words) == 0 {
		t.Fatal("查找敏感词或谐音词失败")
	}

	// 测试4: 替换敏感词和谐音词
	replaced := checker.Replace("不要暴力和睹博,对baoli", '*')

	if replaced != "不要**和**,对*****" {
		t.Fatal("替换谐音敏感词失败")
	}

	replaced = checker.Replace("不要暴-力和睹-博", '*')
	t.Log("==>", replaced)
	if replaced != "不要*-*和*-*" {
		t.Fatal("替换变形敏感词失败")
	}

	// 测试6: 检测变形词
	if !checker.Contains("暴-力") {
		t.Fatal("检测变形词失败")
	}

	if !checker.Contains("赌_博") {
		t.Fatal("检测变形词失败")
	}

	// 测试7: 查找变形词
	words = checker.FindAll("这里有暴-力和赌~博行为")
	if len(words) == 0 {
		t.Fatal("查找变形词失败")
	}

	// 测试8: 禁用变形词模式后，不应该检测到变形词
	checker.DisableDeformMode()
	if checker.Contains("暴-力") {
		t.Fatal("禁用变形词模式后还是检查到变形词了")
	}

	// 但仍然应该检测到原文
	if !checker.Contains("暴力") {
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
	checker := SensitiveChecker.New()

	// 插入敏感词
	checker.Insert("暴力")
	checker.Insert("赌博")
	checker.Insert("武器")

	// 测试1: 检测各种分隔符的变形词
	testCases := []struct {
		text     string
		expected bool
	}{
		{"暴力", true},     // 正常词
		{"暴-力", true},    // 横线分隔
		{"暴_力", true},    // 下划线分隔
		{"暴.力", true},    // 点号分隔
		{"暴~力", true},    // 波浪线分隔
		{"暴~li", true},   // 波浪线分隔
		{"bao~力", true},  // 波浪线分隔
		{"bao~li", true}, // 波浪线分隔
		{"武-器", true},    // 另一个词
		{"五-器", true},    // 另一个词
		{"无害的", false},   // 不相关
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
		{"暴力", "**"},         // 正常词
		{"暴-力", "*-*"},       // 横线分隔
		{"暴_力", "*_*"},       // 下划线分隔
		{"暴.力", "*.*"},       // 点号分隔
		{"暴~li", "*~**"},     // 波浪线分隔
		{"bao~力", "***~*"},   // 波浪线分隔
		{"bao~li", "***~**"}, // 波浪线分隔
		{"武-器", "*-*"},       // 另一个词
		{"五-器", "*-*"},       // 另一个词
		{"无害的", "无害的"},       // 不相关
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
		{"暴力", "暴力"},  // 正常词
		{"暴-力", "暴力"}, // 横线分隔
		{"暴_力", "暴力"}, // 下划线分隔
		{"暴.力", "暴力"}, // 点号分隔

		{"bao~力", "baoli"},  // 波浪线分隔
		{"bao~li", "baoli"}, // 波浪线分隔
		{"武-器", "武器"},       // 另一个词
		{"五-器", "wuqi"},     // 另一个词
		{"无害的", ""},         // 不相关
	}

	for _, tc := range test3Cases {
		result := implodeStr(checker.FindAll(tc.text))
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

}
func TestDisableDeformModeFilter(t *testing.T) {
	checker := SensitiveChecker.New()
	checker.DisableDeformMode()
	//checker.DisableHomophoneMode()
	// 插入敏感词
	checker.Insert("暴力")
	checker.Insert("赌博")
	checker.Insert("武器")

	// 测试1: 检测各种分隔符的变形词
	testCases := []struct {
		text     string
		expected bool
	}{
		{"暴力", true},      // 正常词
		{"暴-力", false},    // 横线分隔
		{"暴_力", false},    // 下划线分隔
		{"暴.力", false},    // 点号分隔
		{"暴~力", false},    // 波浪线分隔
		{"暴~li", false},   // 波浪线分隔
		{"bao~力", false},  // 波浪线分隔
		{"bao~li", false}, // 波浪线分隔
		{"武-器", false},    // 另一个词
		{"五-器", false},    // 另一个词
		{"五器", true},      // 另一个词
		{"无害的", false},    // 不相关
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
		{"暴力", "**"},         // 正常词
		{"暴-力", "暴-力"},       // 横线分隔
		{"暴_力", "暴_力"},       // 下划线分隔
		{"暴.力", "暴.力"},       // 点号分隔
		{"暴~li", "暴~li"},     // 波浪线分隔
		{"bao~力", "bao~力"},   // 波浪线分隔
		{"bao~li", "bao~li"}, // 波浪线分隔
		{"武-器", "武-器"},       // 另一个词
		{"五-器", "五-器"},       // 另一个词
		{"无害的", "无害的"},       // 不相关
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
		{"暴力", "暴力"}, // 正常词
		{"暴-力", ""},  // 横线分隔
		{"暴_力", ""},  // 下划线分隔
		{"暴.力", ""},  // 点号分隔

		{"bao~力", ""},  // 波浪线分隔
		{"bao~li", ""}, // 波浪线分隔
		{"武-器", ""},    // 另一个词
		{"五-器", ""},    // 另一个词
		{"无害的", ""},    // 不相关
	}

	for _, tc := range test3Cases {
		result := implodeStr(checker.FindAll(tc.text))
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

}

func TestDisableHomophoneModeFilter(t *testing.T) {
	checker := SensitiveChecker.New()
	checker.DisableDeformMode()
	checker.DisableHomophoneMode()
	// 插入敏感词
	checker.Insert("暴力")
	checker.Insert("赌博")
	checker.Insert("武器")

	// 测试1: 检测各种分隔符的变形词
	testCases := []struct {
		text     string
		expected bool
	}{
		{"暴力", true},      // 正常词
		{"暴-力", false},    // 横线分隔
		{"暴~li", false},   // 波浪线分隔
		{"bao~力", false},  // 波浪线分隔
		{"bao~li", false}, // 波浪线分隔
		{"武-器", false},    // 另一个词
		{"五-器", false},    // 另一个词
		{"五器", false},     // 另一个词
		{"无害的", false},    // 不相关
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
		{"暴力", "**"},         // 正常词
		{"暴-力", "暴-力"},       // 横线分隔
		{"暴_力", "暴_力"},       // 下划线分隔
		{"暴.力", "暴.力"},       // 点号分隔
		{"暴~li", "暴~li"},     // 波浪线分隔
		{"bao~力", "bao~力"},   // 波浪线分隔
		{"bao~li", "bao~li"}, // 波浪线分隔
		{"武-器", "武-器"},       // 另一个词
		{"五-器", "五-器"},       // 另一个词
		{"无害的", "无害的"},       // 不相关
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
		{"暴力", "暴力"}, // 正常词
		{"暴-力", ""},  // 横线分隔
		{"暴_力", ""},  // 下划线分隔
		{"暴.力", ""},  // 点号分隔

		{"bao~力", ""},  // 波浪线分隔
		{"bao~li", ""}, // 波浪线分隔
		{"武-器", ""},    // 另一个词
		{"五-器", ""},    // 另一个词
		{"无害的", ""},    // 不相关
	}

	for _, tc := range test3Cases {
		result := implodeStr(checker.FindAll(tc.text))
		if result != tc.expectText {
			t.Errorf("文本 '%s' 期望 %v, 实际 %v", tc.text, tc.expectText, result)
		}
	}

}

func TestHomophonePerformance(t *testing.T) {
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
