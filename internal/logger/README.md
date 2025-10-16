# Logger 日志管理模块

## 概述

Logger 模块提供了一个功能完整的结构化日志系统，支持多种日志级别、JSON/文本格式输出以及上下文信息注入。

## 特性

- ✅ 支持多种日志级别（DEBUG、INFO、WARN、ERROR）
- ✅ 支持 JSON 和文本两种输出格式
- ✅ 支持结构化字段
- ✅ 支持上下文信息自动注入（sessionId、requestId、userId）
- ✅ 线程安全
- ✅ 支持字段链式调用
- ✅ 调试模式下自动记录调用者信息

## 快速开始

### 初始化

```go
import "genkit-ai-service/internal/logger"

// 初始化默认日志记录器
logger.Init("info", "json")
```

### 基本使用

```go
// 记录不同级别的日志
logger.Debug("调试信息")
logger.Info("一般信息")
logger.Warn("警告信息")
logger.Error("错误信息")
```

### 带字段的日志

```go
logger.Info("用户登录", logger.Fields{
    "userId":   "user-123",
    "username": "john_doe",
    "ip":       "192.168.1.1",
})
```

### 使用上下文

```go
// 创建带有会话ID的上下文
ctx := context.WithValue(context.Background(), logger.SessionIDKey, "session-abc123")
ctx = context.WithValue(ctx, logger.RequestIDKey, "request-xyz789")

// 使用上下文记录日志，会自动包含 sessionId 和 requestId
logger.InfoContext(ctx, "处理AI对话请求")
```

### 字段链式调用

```go
// 创建带有预设字段的日志记录器
serviceLogger := logger.WithFields(logger.Fields{
    "service": "ai-service",
    "version": "1.0.0",
})

// 使用预设字段的日志记录器
serviceLogger.Info("服务初始化完成")
serviceLogger.Info("开始处理请求", logger.Fields{
    "requestId": "req-123",
})
```

## 日志级别

| 级别  | 说明                     | 使用场景                   |
| ----- | ------------------------ | -------------------------- |
| DEBUG | 详细的调试信息           | 开发和调试阶段             |
| INFO  | 一般信息                 | 正常的业务流程记录         |
| WARN  | 警告信息                 | 潜在问题，但不影响正常运行 |
| ERROR | 错误信息                 | 错误情况，需要关注         |

## 日志格式

### JSON 格式

```json
{
  "timestamp": "2025-10-15T10:30:00Z",
  "level": "INFO",
  "message": "AI对话完成",
  "fields": {
    "sessionId": "session-123",
    "model": "gemini-2.5-flash",
    "duration": "1.5s",
    "totalTokens": 60
  }
}
```

### 文本格式

```
2025-10-15T10:30:00Z [INFO] AI对话完成 sessionId=session-123 model=gemini-2.5-flash duration=1.5s totalTokens=60
```

## 上下文键

模块提供了以下预定义的上下文键：

- `logger.SessionIDKey` - 会话ID
- `logger.RequestIDKey` - 请求ID
- `logger.UserIDKey` - 用户ID

## API 参考

### 全局函数

```go
// 初始化默认日志记录器
func Init(level string, format string)

// 获取默认日志记录器
func Default() Logger

// 全局日志记录函数
func Debug(msg string, fields ...Fields)
func Info(msg string, fields ...Fields)
func Warn(msg string, fields ...Fields)
func Error(msg string, fields ...Fields)

// 全局上下文日志记录函数
func DebugContext(ctx context.Context, msg string, fields ...Fields)
func InfoContext(ctx context.Context, msg string, fields ...Fields)
func WarnContext(ctx context.Context, msg string, fields ...Fields)
func ErrorContext(ctx context.Context, msg string, fields ...Fields)

// 创建带有字段的日志记录器
func WithFields(fields Fields) Logger

// 创建带有上下文的日志记录器
func WithContext(ctx context.Context) Logger
```

### Logger 接口

```go
type Logger interface {
    Debug(msg string, fields ...Fields)
    Info(msg string, fields ...Fields)
    Warn(msg string, fields ...Fields)
    Error(msg string, fields ...Fields)
    
    DebugContext(ctx context.Context, msg string, fields ...Fields)
    InfoContext(ctx context.Context, msg string, fields ...Fields)
    WarnContext(ctx context.Context, msg string, fields ...Fields)
    ErrorContext(ctx context.Context, msg string, fields ...Fields)
    
    WithFields(fields Fields) Logger
    WithContext(ctx context.Context) Logger
    
    SetLevel(level Level)
    SetFormat(format Format)
    SetOutput(w io.Writer)
}
```

## 最佳实践

### 1. 在应用启动时初始化

```go
func main() {
    // 从配置加载日志设置
    cfg, _ := config.Load()
    logger.Init(cfg.Log.Level, cfg.Log.Format)
    
    logger.Info("应用启动")
}
```

### 2. 在服务中使用上下文日志

```go
func (s *AIService) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    logger.InfoContext(ctx, "开始处理对话请求", logger.Fields{
        "model": s.model,
    })
    
    // 处理逻辑...
    
    logger.InfoContext(ctx, "对话请求处理完成", logger.Fields{
        "duration": time.Since(start).String(),
    })
    
    return response, nil
}
```

### 3. 为服务创建专用日志记录器

```go
type AIService struct {
    logger logger.Logger
}

func NewAIService() *AIService {
    return &AIService{
        logger: logger.WithFields(logger.Fields{
            "service": "ai-service",
        }),
    }
}

func (s *AIService) Process() {
    s.logger.Info("处理中...")
}
```

### 4. 错误日志记录

```go
if err != nil {
    logger.Error("AI服务调用失败", logger.Fields{
        "error":     err.Error(),
        "sessionId": sessionID,
        "model":     model,
    })
    return err
}
```

## 配置

通过环境变量配置日志：

```bash
# 日志级别：debug, info, warn, error
LOG_LEVEL=info

# 日志格式：json, text
LOG_FORMAT=json
```

## 性能考虑

- 日志记录器使用读写锁，支持并发安全
- 低于当前日志级别的日志会被快速过滤，不会产生额外开销
- JSON 序列化使用标准库，性能稳定
- 调用者信息仅在 DEBUG 级别记录，避免性能影响

## 测试

运行测试：

```bash
go test ./internal/logger -v
```

查看示例：

```bash
go test ./internal/logger -run Example
```
