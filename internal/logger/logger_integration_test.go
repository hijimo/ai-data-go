package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestTraceIDIntegration 测试 TraceID 从 Context 到日志的完整流程
func TestTraceIDIntegration(t *testing.T) {
	t.Run("TraceID 正确提取并记录到日志", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(InfoLevel, JSONFormat, &buf)

		// 模拟 middleware 设置 TraceID 到 Context
		ctx := context.Background()
		traceID := "trace-1704067200-a3f9k2"
		ctx = context.WithValue(ctx, TraceIDKey, traceID)

		// 记录日志
		log.InfoContext(ctx, "测试消息")

		// 解析日志输出
		output := buf.String()
		var entry logEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("日志输出应该是有效的 JSON: %v, 输出: %s", err, output)
		}

		// 验证 TraceID 在日志字段中
		if entry.Fields["traceId"] != traceID {
			t.Errorf("期望日志包含 traceId=%s, 得到 %v", traceID, entry.Fields["traceId"])
		}

		// 验证日志消息
		if entry.Message != "测试消息" {
			t.Errorf("期望消息为 '测试消息', 得到 '%s'", entry.Message)
		}

		// 验证日志级别
		if entry.Level != "INFO" {
			t.Errorf("期望级别为 'INFO', 得到 '%s'", entry.Level)
		}
	})

	t.Run("多个 Context 字段同时存在", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(InfoLevel, JSONFormat, &buf)

		// 设置多个 Context 字段
		ctx := context.Background()
		ctx = context.WithValue(ctx, TraceIDKey, "trace-123")
		ctx = context.WithValue(ctx, SessionIDKey, "session-456")
		ctx = context.WithValue(ctx, RequestIDKey, "request-789")
		ctx = context.WithValue(ctx, UserIDKey, "user-abc")

		// 记录日志
		log.InfoContext(ctx, "多字段测试")

		// 解析日志输出
		output := buf.String()
		var entry logEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("日志输出应该是有效的 JSON: %v", err)
		}

		// 验证所有字段都存在
		expectedFields := map[string]string{
			"traceId":   "trace-123",
			"sessionId": "session-456",
			"requestId": "request-789",
			"userId":    "user-abc",
		}

		for key, expectedValue := range expectedFields {
			if entry.Fields[key] != expectedValue {
				t.Errorf("期望字段 %s=%s, 得到 %v", key, expectedValue, entry.Fields[key])
			}
		}
	})

	t.Run("TraceID 为空字符串时不记录", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(InfoLevel, JSONFormat, &buf)

		// 设置空的 TraceID
		ctx := context.Background()
		ctx = context.WithValue(ctx, TraceIDKey, "")

		// 记录日志
		log.InfoContext(ctx, "空 TraceID 测试")

		// 解析日志输出
		output := buf.String()
		var entry logEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("日志输出应该是有效的 JSON: %v", err)
		}

		// 验证 traceId 字段不存在
		if _, exists := entry.Fields["traceId"]; exists {
			t.Error("空 TraceID 不应该出现在日志字段中")
		}
	})

	t.Run("文本格式日志包含 TraceID", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(InfoLevel, TextFormat, &buf)

		// 设置 TraceID
		ctx := context.Background()
		traceID := "trace-1704067200-xyz123"
		ctx = context.WithValue(ctx, TraceIDKey, traceID)

		// 记录日志
		log.InfoContext(ctx, "文本格式测试")

		// 验证文本输出包含 TraceID
		output := buf.String()
		if !strings.Contains(output, "traceId="+traceID) {
			t.Errorf("文本格式日志应该包含 traceId=%s, 输出: %s", traceID, output)
		}
	})

	t.Run("WithContext 方法正确提取 TraceID", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(InfoLevel, JSONFormat, &buf)

		// 设置 TraceID
		ctx := context.Background()
		traceID := "trace-withcontext-test"
		ctx = context.WithValue(ctx, TraceIDKey, traceID)

		// 使用 WithContext 创建带有上下文字段的 logger
		logWithCtx := log.WithContext(ctx)

		// 记录日志（不需要传递 Context）
		logWithCtx.Info("WithContext 测试")

		// 解析日志输出
		output := buf.String()
		var entry logEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("日志输出应该是有效的 JSON: %v", err)
		}

		// 验证 TraceID 在日志字段中
		if entry.Fields["traceId"] != traceID {
			t.Errorf("期望日志包含 traceId=%s, 得到 %v", traceID, entry.Fields["traceId"])
		}
	})
}
