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
