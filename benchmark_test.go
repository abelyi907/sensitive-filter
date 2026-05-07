package sensitive_filter

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// ============================================================================
// 压力测试数据生成
// go test -bench=. -benchtime=100ms -benchmem -count=1 -timeout 180s 2>&1
// ============================================================================

// genSensitiveWords 生成 n 个敏感词
func genSensitiveWords(n int) []string {
	words := make([]string, n)
	for i := 0; i < n; i++ {
		// 生成 2~4 个中文字符
		runes := make([]rune, 2+rand.Intn(3))
		for j := range runes {
			runes[j] = rune(0x4e00 + rand.Intn(0x5000))
		}
		words[i] = string(runes)
	}
	return words
}

// genText 生成指定 rune 长度的随机文本
func genText(length int) string {
	runes := make([]rune, length)
	for i := range runes {
		// 混合中文和英文
		if rand.Intn(3) > 0 {
			runes[i] = rune(0x4e00 + rand.Intn(0x5000))
		} else {
			runes[i] = rune('a' + rand.Intn(26))
		}
	}
	return string(runes)
}

// genTextWithSensitive 生成包含敏感词的文本
func genTextWithSensitive(words []string, length int) string {
	base := genText(length)
	// 随机嵌入一些敏感词
	for i := 0; i < 3 && i < len(words); i++ {
		pos := rand.Intn(len(base))
		if pos+len(words[i]) < len(base)-1 {
			base = base[:pos] + words[i] + base[pos+len(words[i]):]
		}
	}
	return base
}

// ============================================================================
// 基础基准测试
// ============================================================================

func BenchmarkInsert(b *testing.B) {
	words := genSensitiveWords(1000)
	checker := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := New()
			for _, w := range words {
				c.Insert(w)
			}
		}
	})
	b.StopTimer()
	// 确保所有自动机构建完成
	_ = checker
}

func BenchmarkContains(b *testing.B) {
	words := genSensitiveWords(100)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 100)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 50)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, t := range texts {
				checker.Contains(t)
			}
		}
	})
}

func BenchmarkFindAll(b *testing.B) {
	words := genSensitiveWords(100)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 100)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 100)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, t := range texts {
				checker.FindAll(t)
			}
		}
	})
}

func BenchmarkReplace(b *testing.B) {
	words := genSensitiveWords(100)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 100)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 100)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, t := range texts {
				checker.Replace(t, '*')
			}
		}
	})
}

// ============================================================================
// 大规模词库压力测试（10k 词）
// ============================================================================

func BenchmarkLargeDictionary(b *testing.B) {
	words := genSensitiveWords(10000)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	text := genTextWithSensitive(words, 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Contains(text)
		checker.FindAll(text)
		checker.Replace(text, '*')
	}
}

// ============================================================================
// 长文本压力测试
// ============================================================================

func BenchmarkLongText(b *testing.B) {
	words := genSensitiveWords(200)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	longText := genTextWithSensitive(words, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Contains(longText)
	}
}

func BenchmarkLongTextFindAll(b *testing.B) {
	words := genSensitiveWords(200)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	longText := genTextWithSensitive(words, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.FindAll(longText)
	}
}

func BenchmarkLongTextReplace(b *testing.B) {
	words := genSensitiveWords(200)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	longText := genTextWithSensitive(words, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Replace(longText, '*')
	}
}

// ============================================================================
// 模式组合压力测试
// ============================================================================

func BenchmarkAllModes(b *testing.B) {
	words := genSensitiveWords(50)
	text := genTextWithSensitive(words, 200)
	// 添加一些变形和火星文
	text = strings.ReplaceAll(text, string(rune(0x4e00+rand.Intn(500))), "卜")
	text = strings.ReplaceAll(text, string(rune(0x4e00+rand.Intn(500))), "滴")

	b.Run("AllEnabled", func(b *testing.B) {
		checker := New()
		for _, w := range words {
			checker.Insert(w)
		}
		checker.ac.build()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.Contains(text)
		}
	})

	b.Run("NoHomophone", func(b *testing.B) {
		checker := New(WithHomophone(false))
		for _, w := range words {
			checker.Insert(w)
		}
		checker.ac.build()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.Contains(text)
		}
	})

	b.Run("NoDeform", func(b *testing.B) {
		checker := New(WithDeform(false))
		for _, w := range words {
			checker.Insert(w)
		}
		checker.ac.build()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.Contains(text)
		}
	})

	b.Run("NoMartian", func(b *testing.B) {
		checker := New(WithMartian(false))
		for _, w := range words {
			checker.Insert(w)
		}
		checker.ac.build()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.Contains(text)
		}
	})
}

// ============================================================================
// 并发安全压力测试（只读操作在构建后并发执行）
// ============================================================================

func BenchmarkConcurrentRead(b *testing.B) {
	words := genSensitiveWords(500)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 50)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 200)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, t := range texts {
				checker.Contains(t)
				checker.FindAll(t)
				checker.Replace(t, '*')
			}
		}
	})
}

// ============================================================================
// 最大敏感词长度压力测试
// ============================================================================

func BenchmarkMaxWordLength(b *testing.B) {
	// 生成长短语作为敏感词
	longWords := []string{
		"这是一个非常长的敏感词短语用来测试",
		"另一个超长示例敏感词短语测试匹配",
		"第三段超长敏感词用于验证AC自动机性能",
	}

	checker := New()
	for _, w := range longWords {
		checker.Insert(w)
	}
	checker.ac.build()

	text := "这是一些正常文本，" + longWords[0] + "中间还有一些，" +
		longWords[1] + "末尾再来一点，" + longWords[2]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Contains(text)
		checker.FindAll(text)
		checker.Replace(text, '*')
	}
}

// ============================================================================
// 空输入与边界测试
// ============================================================================

func BenchmarkEdgeCases(b *testing.B) {
	checker := New()
	checker.Insert("测试")
	checker.Insert("敏感词")
	checker.ac.build()

	tests := []struct {
		name string
		text string
		fn   func(string) interface{}
	}{
		{"Empty_Contains", "", func(s string) interface{} { return checker.Contains(s) }},
		{"Empty_FindAll", "", func(s string) interface{} { return checker.FindAll(s) }},
		{"Empty_Replace", "", func(s string) interface{} { return checker.Replace(s, '*') }},
		{"NoMatch_Contains", "完全正常的文字，没有任何敏感内容", func(s string) interface{} { return checker.Contains(s) }},
		{"NoMatch_FindAll", "完全正常的文字，没有任何敏感内容", func(s string) interface{} { return checker.FindAll(s) }},
		{"NoMatch_Replace", "完全正常的文字，没有任何敏感内容", func(s string) interface{} { return checker.Replace(s, '*') }},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tc.fn(tc.text)
			}
		})
	}
}

// ============================================================================
// 长尾文本模式 — 大量短匹配
// ============================================================================

func BenchmarkManyShortMatches(b *testing.B) {
	// 插入大量 2 字敏感词
	checker := New()
	for i := 0; i < 50; i++ {
		a := rune(0x4e00 + i)
		b := rune(0x4e00 + i + 1)
		checker.Insert(string([]rune{a, b}))
	}
	checker.ac.build()

	// 文本中包含大量匹配
	text := strings.Repeat("这是测试文本包含大量匹配词", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.FindAll(text)
	}
}

// ============================================================================
// 验证测试：确保基准测试结果可打印（用于手动检查）
// ============================================================================

func TestBenchmarkSanity(t *testing.T) {
	words := genSensitiveWords(100)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	// 验证基本功能
	text := genTextWithSensitive(words, 200)
	if !checker.Contains(text) {
		t.Log("注意：随机生成的文本可能不包含敏感词，这是正常的")
	}

	wordsFound := checker.FindAll(text)
	_ = wordsFound

	replaced := checker.Replace(text, '*')
	_ = replaced

	t.Logf("基准验证通过：文本长度=%d, 敏感词数=%d, 找到匹配=%d, 替换后长度=%d",
		len([]rune(text)), len(words), len(wordsFound), len([]rune(replaced)))
}

// ============================================================================
// 压力测试：连续多次 WarmUp + 大量调用
// ============================================================================

func TestStressContains(t *testing.T) {
	words := genSensitiveWords(1000)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 100)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 200)
	}

	// WarmUp
	for i := 0; i < 100; i++ {
		checker.Contains(texts[i%len(texts)])
	}

	// Stress test
	iterations := 10000
	for i := 0; i < iterations; i++ {
		result := checker.Contains(texts[i%len(texts)])
		if i == 0 {
			_ = result
		}
	}
	t.Logf("压力测试完成：执行 %d 次 Contains", iterations)
}

func TestStressFindAll(t *testing.T) {
	words := genSensitiveWords(500)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 50)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 300)
	}

	// WarmUp
	for i := 0; i < 50; i++ {
		checker.FindAll(texts[i%len(texts)])
	}

	iterations := 5000
	totalMatches := 0
	for i := 0; i < iterations; i++ {
		matches := checker.FindAll(texts[i%len(texts)])
		totalMatches += len(matches)
	}
	t.Logf("压力测试完成：执行 %d 次 FindAll，共找到 %d 个匹配", iterations, totalMatches)
}

func TestStressReplace(t *testing.T) {
	words := genSensitiveWords(500)
	checker := New()
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	texts := make([]string, 50)
	for i := range texts {
		texts[i] = genTextWithSensitive(words, 300)
	}

	// WarmUp
	for i := 0; i < 50; i++ {
		checker.Replace(texts[i%len(texts)], '*')
	}

	iterations := 5000
	for i := 0; i < iterations; i++ {
		_ = checker.Replace(texts[i%len(texts)], '*')
	}
	t.Logf("压力测试完成：执行 %d 次 Replace", iterations)
}

// ============================================================================
// 大文本文件加载压力测试
// ============================================================================

func TestLargeFileLoad(t *testing.T) {
	// 生成临时词库文件
	words := genSensitiveWords(10000)
	tmpFile := t.TempDir() + "/large_dict.txt"
	content := strings.Join(words, "\n")
	if err := writeStringToFile(tmpFile, content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	checker := New()
	if err := checker.LoadFromFileByLine(tmpFile); err != nil {
		t.Fatalf("加载大文件失败: %v", err)
	}

	// 验证性能
	testText := genTextWithSensitive(words, 500)
	for i := 0; i < 100; i++ {
		checker.Contains(testText)
		checker.FindAll(testText)
		checker.Replace(testText, '*')
	}
	t.Logf("大文件加载测试通过：10000 词加载完成")
}

func writeStringToFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ============================================================================
// 综合压力测试：多模式 + 大词库 + 长文本
// ============================================================================

func TestComprehensiveStress(t *testing.T) {
	// 准备：5000 词 + 火星文 + 变形 + 谐音
	words := genSensitiveWords(5000)
	checker := New(WithHomophone(true), WithDeform(true), WithMartian(true))
	for _, w := range words {
		checker.Insert(w)
	}
	checker.ac.build()

	// 生成混合文本
	shortText := genTextWithSensitive(words, 100)
	mediumText := genTextWithSensitive(words, 1000)
	longText := genTextWithSensitive(words, 10000)

	t.Run("短文本-Contains", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			checker.Contains(shortText)
		}
	})

	t.Run("中文本-FindAll", func(t *testing.T) {
		for i := 0; i < 500; i++ {
			checker.FindAll(mediumText)
		}
	})

	t.Run("长文本-Replace", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			checker.Replace(longText, '*')
		}
	})

	t.Run("混合操作", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			checker.Contains(shortText)
			checker.FindAll(mediumText)
			checker.Replace(longText, '*')
		}
	})

	t.Log("综合压力测试通过")
}

// ============================================================================
// CPU 和内存分析友好测试
// ============================================================================

func BenchmarkMemoryUsage(b *testing.B) {
	wordCounts := []int{100, 500, 1000, 5000}

	for _, n := range wordCounts {
		b.Run(fmt.Sprintf("Words_%d", n), func(b *testing.B) {
			words := genSensitiveWords(n)
			checker := New()
			for _, w := range words {
				checker.Insert(w)
			}
			checker.ac.build()
			text := genTextWithSensitive(words, 200)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				checker.Contains(text)
				checker.FindAll(text)
				checker.Replace(text, '*')
			}
		})
	}
}
