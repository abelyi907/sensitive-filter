package sensitive_filter

import (
	"testing"
)

// 测试服务
func TestService(t *testing.T) {
	checker := SensitiveChecker.New()

	// 从文件加载敏感词库
	checker.LoadFromTextFile("./words.txt")

	// 测试文本
	testText := "这是一个包含敏感词1和敏感词3内容的文本，涉及信息。"

	// 1. 替换敏感词
	replacedText := checker.Replace(testText, '*')
	if replacedText != "这是一个包含****和****内容的文本，涉及信息。" {
		t.Fatalf("替换敏感词失败")
	}
	if !checker.Contains("我反对使用敏感词2") {
		t.Fatalf("检查是否包含敏感词失败")
	}

	ws := checker.FindAll("我反对使用敏感词1")
	if len(ws) != 1 {
		t.Fatalf("检查是否包含敏感词失败")
	}

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

	// 测试3: 查找所有敏感词（包括谐音）
	words := checker.FindAll("这里有暴力和睹博行为")
	if len(words) == 0 {
		t.Fatal("查找敏感词或谐音词失败")
	}
	t.Logf("找到的敏感词: %v", words)

	// 测试4: 替换敏感词和谐音词
	replaced := checker.Replace("不要暴力和睹博,对baoli", '*')
	t.Log("==>", replaced)
	if replaced != "不要**和**,对*****" {
		t.Fatal("替换谐音敏感词失败")
	}

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

// 测试性能：确保谐音模式不会显著影响性能
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
