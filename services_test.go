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
		t.Fatalf("替换结果:" + replacedText)
	}
	if !checker.Contains("我反对使用敏感词2") {
		t.Fatalf("检查是否包含敏感词失败")
	}

	ws := checker.FindAll("我反对使用敏感词1")
	if len(ws) != 1 {
		t.Fatalf("检查是否包含敏感词失败")
	}

}
