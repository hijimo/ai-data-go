package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Level
	}{
		{"debug", "debug", DebugLevel},
		{"info", "info", InfoLevel},
		{"warn", "warn", WarnLevel},
		{"warning", "warning", WarnLevel},
		{"error", "error", ErrorLevel},
		{"uppercase", "INFO", InfoLevel},
		{"invalid", "invalid", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Level.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestLoggerBasicLogging(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	log.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("log output should contain message, got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("log output should contain level, got: %s", output)
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := New(WarnLevel, JSONFormat, &buf)

	// 这些不应该被记录
	log.Debug("debug message")
	log.Info("info message")

	// 这些应该被记录
	log.Warn("warn message")
	log.Error("error message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("debug message should not be logged at WARN level")
	}
	if strings.Contains(output, "info message") {
		t.Error("info message should not be logged at WARN level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should be logged at WARN level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("error message should be logged at WARN level")
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	log.Info("test message", Fields{
		"key1": "value1",
		"key2": 123,
	})

	output := buf.String()
	if !strings.Contains(output, "key1") || !strings.Contains(output, "value1") {
		t.Errorf("log output should contain fields, got: %s", output)
	}
}

func TestLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	ctx := context.WithValue(context.Background(), SessionIDKey, "session-123")
	ctx = context.WithValue(ctx, RequestIDKey, "request-456")

	log.InfoContext(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "session-123") {
		t.Errorf("log output should contain sessionId, got: %s", output)
	}
	if !strings.Contains(output, "request-456") {
		t.Errorf("log output should contain requestId, got: %s", output)
	}
}

func TestLoggerWithFieldsChaining(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	logWithFields := log.WithFields(Fields{
		"service": "test-service",
	})

	logWithFields.Info("test message", Fields{
		"action": "test-action",
	})

	output := buf.String()
	if !strings.Contains(output, "service") || !strings.Contains(output, "test-service") {
		t.Errorf("log output should contain preset fields, got: %s", output)
	}
	if !strings.Contains(output, "action") || !strings.Contains(output, "test-action") {
		t.Errorf("log output should contain additional fields, got: %s", output)
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	log.Info("test message", Fields{
		"key": "value",
	})

	output := buf.String()
	var entry logEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("log output should be valid JSON, got error: %v, output: %s", err, output)
	}

	if entry.Message != "test message" {
		t.Errorf("entry.Message = %s, want 'test message'", entry.Message)
	}
	if entry.Level != "INFO" {
		t.Errorf("entry.Level = %s, want 'INFO'", entry.Level)
	}
}

func TestLoggerTextFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, TextFormat, &buf)

	log.Info("test message", Fields{
		"key": "value",
	})

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("text format should contain [INFO], got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("text format should contain message, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("text format should contain fields, got: %s", output)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	log.Debug("should not appear")
	
	log.SetLevel(DebugLevel)
	log.Debug("should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("debug message should not be logged before level change")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("debug message should be logged after level change")
	}
}

func TestDefaultLogger(t *testing.T) {
	// 重置默认日志记录器
	defaultLogger = nil
	once = sync.Once{}

	Init("info", "json")
	log := Default()

	if log == nil {
		t.Error("Default() should return a logger")
	}
}

func TestGlobalFunctions(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger = New(InfoLevel, JSONFormat, &buf)

	Info("global info message")

	output := buf.String()
	if !strings.Contains(output, "global info message") {
		t.Errorf("global Info() should log message, got: %s", output)
	}
}

func TestContextFunctions(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger = New(InfoLevel, JSONFormat, &buf)

	ctx := context.WithValue(context.Background(), SessionIDKey, "test-session")
	InfoContext(ctx, "context message")

	output := buf.String()
	if !strings.Contains(output, "context message") {
		t.Errorf("InfoContext() should log message, got: %s", output)
	}
	if !strings.Contains(output, "test-session") {
		t.Errorf("InfoContext() should include context fields, got: %s", output)
	}
}

func TestLoggerWithTraceID(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	// 导入 middleware 包以使用 SetTraceID
	ctx := context.Background()
	// 使用 middleware.SetTraceID 设置 TraceID
	// 注意：这里需要导入 middleware 包
	// 为了测试，我们直接使用 context.WithValue
	ctx = context.WithValue(ctx, contextKey("traceId"), "trace-1704067200-a3f9k2")

	log.InfoContext(ctx, "test message with trace")

	output := buf.String()
	if !strings.Contains(output, "trace-1704067200-a3f9k2") {
		t.Errorf("log output should contain traceId, got: %s", output)
	}
	if !strings.Contains(output, "traceId") {
		t.Errorf("log output should contain traceId field name, got: %s", output)
	}
}

func TestLoggerWithoutTraceID(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	// Context 中没有 TraceID
	ctx := context.Background()

	log.InfoContext(ctx, "test message without trace")

	output := buf.String()
	// 应该正常记录日志，只是没有 traceId 字段
	if !strings.Contains(output, "test message without trace") {
		t.Errorf("log output should contain message, got: %s", output)
	}
	// 验证日志是有效的 JSON
	var entry logEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("log output should be valid JSON even without traceId, got error: %v", err)
	}
}

// TestExtractContextFields 测试从 Context 提取字段
func TestExtractContextFields(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected map[string]interface{}
	}{
		{
			name: "提取 TraceID",
			ctx:  context.WithValue(context.Background(), TraceIDKey, "trace-123"),
			expected: map[string]interface{}{
				"traceId": "trace-123",
			},
		},
		{
			name: "提取多个字段",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, TraceIDKey, "trace-456")
				ctx = context.WithValue(ctx, SessionIDKey, "session-789")
				ctx = context.WithValue(ctx, RequestIDKey, "request-abc")
				ctx = context.WithValue(ctx, UserIDKey, "user-def")
				return ctx
			}(),
			expected: map[string]interface{}{
				"traceId":   "trace-456",
				"sessionId": "session-789",
				"requestId": "request-abc",
				"userId":    "user-def",
			},
		},
		{
			name:     "空 Context",
			ctx:      context.Background(),
			expected: map[string]interface{}{},
		},
		{
			name: "TraceID 为空字符串时不添加",
			ctx:  context.WithValue(context.Background(), TraceIDKey, ""),
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := extractContextFields(tt.ctx)

			if len(fields) != len(tt.expected) {
				t.Errorf("期望字段数量 %d，实际 %d", len(tt.expected), len(fields))
			}

			for key, expectedValue := range tt.expected {
				actualValue, ok := fields[key]
				if !ok {
					t.Errorf("缺少字段 %s", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("字段 %s: 期望 %v，实际 %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestLoggerTraceIDInJSON 测试 TraceID 在 JSON 日志中的格式
func TestLoggerTraceIDInJSON(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	ctx := context.WithValue(context.Background(), TraceIDKey, "trace-json-test")
	log.InfoContext(ctx, "test message")

	output := buf.String()
	var entry logEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("无法解析 JSON 日志: %v", err)
	}

	// 验证 TraceID 在 fields 中
	traceID, ok := entry.Fields["traceId"]
	if !ok {
		t.Error("fields 中缺少 traceId")
	}
	if traceID != "trace-json-test" {
		t.Errorf("期望 traceId 为 trace-json-test，实际为 %v", traceID)
	}
}

// TestLoggerTraceIDInText 测试 TraceID 在文本日志中的格式
func TestLoggerTraceIDInText(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, TextFormat, &buf)

	ctx := context.WithValue(context.Background(), TraceIDKey, "trace-text-test")
	log.InfoContext(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "traceId=trace-text-test") {
		t.Errorf("文本日志应包含 traceId=trace-text-test，实际输出: %s", output)
	}
}

// TestLoggerMultipleContextFields 测试多个上下文字段同时存在
func TestLoggerMultipleContextFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)

	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "trace-multi-001")
	ctx = context.WithValue(ctx, SessionIDKey, "session-multi-001")
	ctx = context.WithValue(ctx, RequestIDKey, "request-multi-001")

	log.InfoContext(ctx, "test with multiple fields")

	output := buf.String()
	var entry logEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("无法解析 JSON 日志: %v", err)
	}

	// 验证所有字段都存在
	expectedFields := map[string]string{
		"traceId":   "trace-multi-001",
		"sessionId": "session-multi-001",
		"requestId": "request-multi-001",
	}

	for key, expectedValue := range expectedFields {
		actualValue, ok := entry.Fields[key]
		if !ok {
			t.Errorf("fields 中缺少 %s", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("字段 %s: 期望 %s，实际 %v", key, expectedValue, actualValue)
		}
	}
}

// BenchmarkExtractContextFields 性能测试：提取上下文字段
func BenchmarkExtractContextFields(b *testing.B) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "trace-bench-001")
	ctx = context.WithValue(ctx, SessionIDKey, "session-bench-001")
	ctx = context.WithValue(ctx, RequestIDKey, "request-bench-001")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractContextFields(ctx)
	}
}

// BenchmarkLoggerWithTraceID 性能测试：带 TraceID 的日志记录
func BenchmarkLoggerWithTraceID(b *testing.B) {
	var buf bytes.Buffer
	log := New(InfoLevel, JSONFormat, &buf)
	ctx := context.WithValue(context.Background(), TraceIDKey, "trace-bench-002")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.InfoContext(ctx, "benchmark message")
	}
}
