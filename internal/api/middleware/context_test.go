package middleware

import (
	"context"
	"regexp"
	"sync"
	"testing"
)

// TestSetTraceID 测试 SetTraceID 函数
func TestSetTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "trace-1704067200-a3f9k2"
	
	// 注入 TraceID
	ctx = SetTraceID(ctx, traceID)
	
	// 验证 TraceID 已注入
	if value := ctx.Value(TraceIDKey); value == nil {
		t.Error("TraceID 未成功注入到 Context")
	} else if value.(string) != traceID {
		t.Errorf("期望 TraceID 为 %s，实际为 %s", traceID, value.(string))
	}
}

// TestGetTraceID 测试 GetTraceID 函数
func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		traceID  string
		expected string
	}{
		{
			name:     "正常获取 TraceID",
			ctx:      SetTraceID(context.Background(), "trace-1704067200-a3f9k2b8c1d4"),
			expected: "trace-1704067200-a3f9k2b8c1d4",
		},
		{
			name:     "Context 中无 TraceID",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "Context 为 nil",
			ctx:      nil,
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTraceID(tt.ctx)
			if result != tt.expected {
				t.Errorf("期望 %s，实际 %s", tt.expected, result)
			}
		})
	}
}

// TestGenerateTraceID 测试 GenerateTraceID 函数
func TestGenerateTraceID(t *testing.T) {
	// 测试 TraceID 格式
	traceID := GenerateTraceID()
	
	// 验证格式：trace-{timestamp}-{nanoHex}{random}
	// 示例：trace-1704067200-a3f9k2b8c1d4 (10位时间戳 + 6位纳秒十六进制 + 6位随机)
	pattern := `^trace-\d{10}-[0-9a-f]{12}$`
	matched, err := regexp.MatchString(pattern, traceID)
	if err != nil {
		t.Fatalf("正则表达式错误: %v", err)
	}
	if !matched {
		t.Errorf("TraceID 格式不正确: %s，期望格式: trace-{timestamp}-{nanoHex}{random}", traceID)
	}
	
	// 验证长度
	expectedLen := len("trace-") + 10 + len("-") + 12 // trace-{10位时间戳}-{12位随机}
	if len(traceID) != expectedLen {
		t.Errorf("TraceID 长度不正确: %d，期望 %d", len(traceID), expectedLen)
	}
}

// TestGenerateTraceIDUniqueness 测试 TraceID 唯一性
func TestGenerateTraceIDUniqueness(t *testing.T) {
	const count = 10000
	traceIDs := make(map[string]bool, count)
	mu := sync.Mutex{}
	
	// 并发生成 TraceID
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			traceID := GenerateTraceID()
			
			mu.Lock()
			if traceIDs[traceID] {
				t.Errorf("发现重复的 TraceID: %s", traceID)
			}
			traceIDs[traceID] = true
			mu.Unlock()
		}()
	}
	
	wg.Wait()
	
	// 验证生成了预期数量的唯一 TraceID
	if len(traceIDs) != count {
		t.Errorf("期望生成 %d 个唯一 TraceID，实际生成 %d 个", count, len(traceIDs))
	}
}

// TestGenerateRandomString 测试 generateRandomString 函数
func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"长度为6", 6},
		{"长度为8", 8},
		{"长度为10", 10},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.length)
			
			// 验证长度
			if len(result) != tt.length {
				t.Errorf("期望长度 %d，实际长度 %d", tt.length, len(result))
			}
			
			// 验证字符集（十六进制）
			pattern := `^[0-9a-f]+$`
			matched, err := regexp.MatchString(pattern, result)
			if err != nil {
				t.Fatalf("正则表达式错误: %v", err)
			}
			if !matched {
				t.Errorf("随机字符串包含非十六进制字符: %s", result)
			}
		})
	}
}

// BenchmarkGenerateTraceID 性能测试：TraceID 生成
func BenchmarkGenerateTraceID(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		GenerateTraceID()
	}
}

// BenchmarkSetTraceID 性能测试：TraceID 注入
func BenchmarkSetTraceID(b *testing.B) {
	ctx := context.Background()
	traceID := "trace-1704067200-a3f9k2b8c1d4"
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetTraceID(ctx, traceID)
	}
}

// BenchmarkGetTraceID 性能测试：TraceID 提取
func BenchmarkGetTraceID(b *testing.B) {
	ctx := SetTraceID(context.Background(), "trace-1704067200-a3f9k2b8c1d4")
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTraceID(ctx)
	}
}

// BenchmarkGenerateTraceIDParallel 并发性能测试：TraceID 生成
func BenchmarkGenerateTraceIDParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GenerateTraceID()
		}
	})
}
