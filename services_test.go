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
	testText := "这是一个包含色情和暴力内容的文本，涉及信息。"

	// 1. 替换敏感词
	replacedText := checker.Replace(testText, '*')
	if replacedText != "这是一个包含**和**内容的文本，涉及信息。" {
		t.Fatalf("测试结果:fails")
	}

}
