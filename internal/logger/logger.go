package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	// DebugLevel 调试级别
	DebugLevel Level = iota
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 从字符串解析日志级别
func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Format 日志格式
type Format string

const (
	// JSONFormat JSON格式
	JSONFormat Format = "json"
	// TextFormat 文本格式
	TextFormat Format = "text"
)

// contextKey 上下文键类型
type contextKey string

const (
	// SessionIDKey 会话ID键
	SessionIDKey contextKey = "sessionId"
	// RequestIDKey 请求ID键
	RequestIDKey contextKey = "requestId"
	// UserIDKey 用户ID键
	UserIDKey contextKey = "userId"
)

// Fields 日志字段
type Fields map[string]interface{}

// Logger 日志记录器接口
type Logger interface {
	// Debug 记录调试级别日志
	Debug(msg string, fields ...Fields)
	// Info 记录信息级别日志
	Info(msg string, fields ...Fields)
	// Warn 记录警告级别日志
	Warn(msg string, fields ...Fields)
	// Error 记录错误级别日志
	Error(msg string, fields ...Fields)
	
	// DebugContext 使用上下文记录调试级别日志
	DebugContext(ctx context.Context, msg string, fields ...Fields)
	// InfoContext 使用上下文记录信息级别日志
	InfoContext(ctx context.Context, msg string, fields ...Fields)
	// WarnContext 使用上下文记录警告级别日志
	WarnContext(ctx context.Context, msg string, fields ...Fields)
	// ErrorContext 使用上下文记录错误级别日志
	ErrorContext(ctx context.Context, msg string, fields ...Fields)
	
	// WithFields 创建带有预设字段的日志记录器
	WithFields(fields Fields) Logger
	// WithContext 创建带有上下文的日志记录器
	WithContext(ctx context.Context) Logger
	
	// SetLevel 设置日志级别
	SetLevel(level Level)
	// SetFormat 设置日志格式
	SetFormat(format Format)
	// SetOutput 设置输出目标
	SetOutput(w io.Writer)
}

// logger 日志记录器实现
type logger struct {
	level      Level
	format     Format
	output     io.Writer
	fields     Fields
	mu         sync.RWMutex
	callerSkip int
}

// logEntry 日志条目
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

var (
	// defaultLogger 默认日志记录器
	defaultLogger Logger
	// once 确保只初始化一次
	once sync.Once
)

// New 创建新的日志记录器
func New(level Level, format Format, output io.Writer) Logger {
	return &logger{
		level:      level,
		format:     format,
		output:     output,
		fields:     make(Fields),
		callerSkip: 2,
	}
}

// Init 初始化默认日志记录器
func Init(level string, format string) {
	once.Do(func() {
		l := ParseLevel(level)
		f := JSONFormat
		if strings.ToLower(format) == "text" {
			f = TextFormat
		}
		defaultLogger = New(l, f, os.Stdout)
	})
}

// Default 获取默认日志记录器
func Default() Logger {
	if defaultLogger == nil {
		Init("info", "json")
	}
	return defaultLogger
}

// Debug 记录调试级别日志
func Debug(msg string, fields ...Fields) {
	Default().Debug(msg, fields...)
}

// Info 记录信息级别日志
func Info(msg string, fields ...Fields) {
	Default().Info(msg, fields...)
}

// Warn 记录警告级别日志
func Warn(msg string, fields ...Fields) {
	Default().Warn(msg, fields...)
}

// Error 记录错误级别日志
func Error(msg string, fields ...Fields) {
	Default().Error(msg, fields...)
}

// DebugContext 使用上下文记录调试级别日志
func DebugContext(ctx context.Context, msg string, fields ...Fields) {
	Default().DebugContext(ctx, msg, fields...)
}

// InfoContext 使用上下文记录信息级别日志
func InfoContext(ctx context.Context, msg string, fields ...Fields) {
	Default().InfoContext(ctx, msg, fields...)
}

// WarnContext 使用上下文记录警告级别日志
func WarnContext(ctx context.Context, msg string, fields ...Fields) {
	Default().WarnContext(ctx, msg, fields...)
}

// ErrorContext 使用上下文记录错误级别日志
func ErrorContext(ctx context.Context, msg string, fields ...Fields) {
	Default().ErrorContext(ctx, msg, fields...)
}

// WithFields 创建带有预设字段的日志记录器
func WithFields(fields Fields) Logger {
	return Default().WithFields(fields)
}

// WithContext 创建带有上下文的日志记录器
func WithContext(ctx context.Context) Logger {
	return Default().WithContext(ctx)
}

// Debug 记录调试级别日志
func (l *logger) Debug(msg string, fields ...Fields) {
	l.log(DebugLevel, msg, fields...)
}

// Info 记录信息级别日志
func (l *logger) Info(msg string, fields ...Fields) {
	l.log(InfoLevel, msg, fields...)
}

// Warn 记录警告级别日志
func (l *logger) Warn(msg string, fields ...Fields) {
	l.log(WarnLevel, msg, fields...)
}

// Error 记录错误级别日志
func (l *logger) Error(msg string, fields ...Fields) {
	l.log(ErrorLevel, msg, fields...)
}

// DebugContext 使用上下文记录调试级别日志
func (l *logger) DebugContext(ctx context.Context, msg string, fields ...Fields) {
	l.logContext(ctx, DebugLevel, msg, fields...)
}

// InfoContext 使用上下文记录信息级别日志
func (l *logger) InfoContext(ctx context.Context, msg string, fields ...Fields) {
	l.logContext(ctx, InfoLevel, msg, fields...)
}

// WarnContext 使用上下文记录警告级别日志
func (l *logger) WarnContext(ctx context.Context, msg string, fields ...Fields) {
	l.logContext(ctx, WarnLevel, msg, fields...)
}

// ErrorContext 使用上下文记录错误级别日志
func (l *logger) ErrorContext(ctx context.Context, msg string, fields ...Fields) {
	l.logContext(ctx, ErrorLevel, msg, fields...)
}

// WithFields 创建带有预设字段的日志记录器
func (l *logger) WithFields(fields Fields) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	newFields := make(Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &logger{
		level:      l.level,
		format:     l.format,
		output:     l.output,
		fields:     newFields,
		callerSkip: l.callerSkip,
	}
}

// WithContext 创建带有上下文的日志记录器
func (l *logger) WithContext(ctx context.Context) Logger {
	fields := extractContextFields(ctx)
	return l.WithFields(fields)
}

// SetLevel 设置日志级别
func (l *logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat 设置日志格式
func (l *logger) SetFormat(format Format) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = format
}

// SetOutput 设置输出目标
func (l *logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// log 记录日志
func (l *logger) log(level Level, msg string, fields ...Fields) {
	l.mu.RLock()
	if level < l.level {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()
	
	entry := l.buildEntry(level, msg, fields...)
	l.write(entry)
}

// logContext 使用上下文记录日志
func (l *logger) logContext(ctx context.Context, level Level, msg string, fields ...Fields) {
	l.mu.RLock()
	if level < l.level {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()
	
	// 从上下文提取字段
	ctxFields := extractContextFields(ctx)
	allFields := []Fields{ctxFields}
	allFields = append(allFields, fields...)
	
	entry := l.buildEntry(level, msg, allFields...)
	l.write(entry)
}

// buildEntry 构建日志条目
func (l *logger) buildEntry(level Level, msg string, fields ...Fields) *logEntry {
	entry := &logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}
	
	// 添加预设字段
	l.mu.RLock()
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	l.mu.RUnlock()
	
	// 添加传入的字段
	for _, f := range fields {
		for k, v := range f {
			entry.Fields[k] = v
		}
	}
	
	// 添加调用者信息（仅在 DEBUG 级别）
	if level == DebugLevel {
		if caller := getCaller(l.callerSkip); caller != "" {
			entry.Caller = caller
		}
	}
	
	return entry
}

// write 写入日志
func (l *logger) write(entry *logEntry) {
	l.mu.RLock()
	format := l.format
	output := l.output
	l.mu.RUnlock()
	
	var line string
	if format == JSONFormat {
		line = l.formatJSON(entry)
	} else {
		line = l.formatText(entry)
	}
	
	fmt.Fprintln(output, line)
}

// formatJSON 格式化为JSON
func (l *logger) formatJSON(entry *logEntry) string {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal log entry: %v"}`, err)
	}
	return string(data)
}

// formatText 格式化为文本
func (l *logger) formatText(entry *logEntry) string {
	var sb strings.Builder
	
	sb.WriteString(entry.Timestamp)
	sb.WriteString(" [")
	sb.WriteString(entry.Level)
	sb.WriteString("] ")
	sb.WriteString(entry.Message)
	
	if len(entry.Fields) > 0 {
		sb.WriteString(" ")
		for k, v := range entry.Fields {
			sb.WriteString(fmt.Sprintf("%s=%v ", k, v))
		}
	}
	
	if entry.Caller != "" {
		sb.WriteString(" caller=")
		sb.WriteString(entry.Caller)
	}
	
	return strings.TrimSpace(sb.String())
}

// extractContextFields 从上下文提取字段
func extractContextFields(ctx context.Context) Fields {
	fields := make(Fields)
	
	if sessionID := ctx.Value(SessionIDKey); sessionID != nil {
		fields["sessionId"] = sessionID
	}
	
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields["requestId"] = requestID
	}
	
	if userID := ctx.Value(UserIDKey); userID != nil {
		fields["userId"] = userID
	}
	
	return fields
}

// getCaller 获取调用者信息
func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 2)
	if !ok {
		return ""
	}
	
	// 只保留文件名，不包含完整路径
	idx := strings.LastIndex(file, "/")
	if idx >= 0 {
		file = file[idx+1:]
	}
	
	return fmt.Sprintf("%s:%d", file, line)
}
