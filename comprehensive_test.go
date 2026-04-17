package sensitive_filter

import (
	"fmt"
	"testing"
)

// TestAllFeaturesCombined 综合测试所有功能（谐音+变形+火星文）
func TestAllFeaturesCombined(t *testing.T) {
	checker := SensitiveChecker.New()

	// 确保所有模式都启用
	checker.EnableHomophoneMode()
	checker.EnableDeformMode()
	checker.EnableMartianMode()

	// 插入敏感词
	checker.Insert("不要")
	checker.Insert("暴力")
	checker.Insert("赌博")
	checker.Insert("敏感")

	fmt.Println("\n=== 综合测试开始 ===")

	// 测试1: 纯原文
	t.Log("测试1: 纯原文")
	if !checker.Contains("不要暴力") {
		t.Error("纯原文检测失败")
	}

	// 测试2: 火星文
	t.Log("测试2: 火星文")
	if !checker.Contains("卜要暴力") {
		t.Error("火星文检测失败")
	}

	// 测试3: 变形词
	t.Log("测试3: 变形词")
	if !checker.Contains("暴-力") {
		t.Error("变形词检测失败")
	}

	// 测试4: 谐音词
	t.Log("测试4: 谐音词")
	if !checker.Contains("dubo") {
		t.Error("谐音词检测失败")
	}

	// 测试5: 火星文 + 变形词
	t.Log("测试5: 火星文 + 变形词")
	if !checker.Contains("卜-要") {
		t.Error("火星文+变形词检测失败")
	}

	// 测试6: 火星文 + 谐音
	t.Log("测试6: 火星文 + 谐音")
	if !checker.Contains("卜要dubo") {
		t.Error("火星文+谐音检测失败")
	}

	// 测试7: 变形词 + 谐音
	t.Log("测试7: 变形词 + 谐音")
	if !checker.Contains("睹-博") {
		t.Error("变形词+谐音检测失败")
	}

	// 测试8: 火星文 + 变形词 + 谐音
	t.Log("测试8: 火星文 + 变形词 + 谐音")
	if !checker.Contains("卜-dubo") {
		t.Error("火星文+变形词+谐音检测失败")
	}

	// 测试9: 复杂混合场景
	t.Log("测试9: 复杂混合场景")
	text := "卜要暴-力和睹_博，这是敏敢内容"
	if !checker.Contains(text) {
		t.Error("复杂混合场景检测失败")
	}

	words := checker.FindAll(text)
	t.Logf("找到的词: %v", words)
	if len(words) == 0 {
		t.Error("复杂混合场景查找失败")
	}

	replaced := checker.Replace(text, '*')
	t.Logf("替换结果: %s", replaced)

	t.Log("=== 综合测试完成 ===\n")
}

// TestMartianEdgeCases 测试火星文的边界情况
func TestMartianEdgeCases(t *testing.T) {
	checker := SensitiveChecker.New()
	checker.EnableMartianMode()

	checker.Insert("喜欢")
	checker.Insert("什么")
	checker.Insert("知道")

	// 测试1: 空字符串
	result := normalizeMartian("")
	if result != "" {
		t.Errorf("空字符串处理失败，期望 '', 实际 '%s'", result)
	}

	// 测试2: 无火星文
	result = normalizeMartian("正常文字")
	if result != "正常文字" {
		t.Errorf("无火星文处理失败，期望 '正常文字', 实际 '%s'", result)
	}

	// 测试3: 连续火星文
	result = normalizeMartian("稀饭神马")
	expected := "喜欢什么"
	if result != expected {
		t.Errorf("连续火星文处理失败，期望 '%s', 实际 '%s'", expected, result)
	}

	// 测试4: 火星文与普通文字混合
	result = normalizeMartian("我稀饭吃苹果")
	expected = "我喜欢吃苹果"
	if result != expected {
		t.Errorf("混合文本处理失败，期望 '%s', 实际 '%s'", expected, result)
	}

	// 测试5: 重叠匹配（优先匹配长词）
	AddMartianMapping("酱紫", "这样子")
	AddMartianMapping("紫", "子")
	result = normalizeMartian("酱紫")
	expected = "这样子" // 应该匹配"酱紫"而不是"酱"+"紫"
	if result != expected {
		t.Errorf("重叠匹配失败，期望 '%s', 实际 '%s'", expected, result)
	}
	RemoveMartianMapping("紫") // 清理测试数据
}

// TestModeCombinations 测试不同模式组合
func TestModeCombinations(t *testing.T) {
	testCases := []struct {
		name          string
		homophone     bool
		deform        bool
		martian       bool
		testText      string
		shouldContain bool
	}{
		{"全禁用", false, false, false, "卜要暴-力", false},
		{"仅火星文", false, false, true, "卜要", true},
		{"仅变形词", false, true, false, "暴-力", true},
		{"仅谐音", true, false, false, "dubo", true},
		{"火星文+变形", false, true, true, "卜-要", true},
		{"火星文+谐音", true, false, true, "卜要dubo", true},
		{"变形+谐音", true, true, false, "睹-博", true},
		{"全启用", true, true, true, "卜-dubo", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := SensitiveChecker.New()

			if tc.homophone {
				checker.EnableHomophoneMode()
			} else {
				checker.DisableHomophoneMode()
			}

			if tc.deform {
				checker.EnableDeformMode()
			} else {
				checker.DisableDeformMode()
			}

			if tc.martian {
				checker.EnableMartianMode()
			} else {
				checker.DisableMartianMode()
			}

			checker.Insert("不要")
			checker.Insert("暴力")
			checker.Insert("赌博")

			result := checker.Contains(tc.testText)
			if result != tc.shouldContain {
				t.Errorf("模式 '%s': 文本 '%s' 期望 %v, 实际 %v",
					tc.name, tc.testText, tc.shouldContain, result)
			}
		})
	}
}
