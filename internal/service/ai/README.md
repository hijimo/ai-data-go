# AI 服务模块

## 概述

AI 服务模块提供核心的 AI 对话能力和会话管理功能。该模块封装了与 Genkit AI 的交互逻辑，并提供会话上下文管理、超时控制和自动清理等功能。

## 组件

### ContextManager - 上下文管理器

上下文管理器负责管理 AI 对话会话的生命周期，包括会话创建、获取、取消和自动清理。

#### 主要功能

1. **会话创建**：生成唯一的会话ID，创建可取消的上下文
2. **会话获取**：根据会话ID获取对应的上下文
3. **会话取消**：主动取消正在进行的会话
4. **自动清理**：定期清理超时或已取消的会话
5. **并发安全**：支持多个goroutine并发访问

#### 使用示例

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/yourusername/genkit-ai-service/internal/service/ai"
)

func main() {
    // 创建上下文管理器
    // 参数：会话超时时间(30分钟)，自动清理间隔(5分钟)
    cm := ai.NewContextManager(30*time.Minute, 5*time.Minute)
    
    // 启动自动清理
    cm.Start()
    defer cm.Stop()
    
    // 创建新会话
    ctx := context.Background()
    sessionID, sessionCtx, cancel := cm.CreateSession(ctx)
    defer cancel()
    
    fmt.Printf("会话ID: %s\n", sessionID)
    
    // 使用会话上下文进行操作
    select {
    case <-sessionCtx.Done():
        fmt.Println("会话已取消")
    case <-time.After(1 * time.Second):
        fmt.Println("操作完成")
    }
    
    // 获取会话
    retrievedCtx, exists := cm.GetSession(sessionID)
    if exists {
        fmt.Println("会话存在")
    }
    
    // 取消会话
    err := cm.CancelSession(sessionID)
    if err != nil {
        fmt.Printf("取消失败: %v\n", err)
    }
}
```

#### 接口定义

```go
type ContextManager interface {
    // CreateSession 创建新会话
    // 返回：会话ID、会话上下文、取消函数
    CreateSession(ctx context.Context) (string, context.Context, context.CancelFunc)
    
    // GetSession 获取会话上下文
    // 参数：会话ID
    // 返回：会话上下文、是否存在
    GetSession(sessionID string) (context.Context, bool)
    
    // CancelSession 取消会话
    // 参数：会话ID
    // 返回：错误信息（如果会话不存在）
    CancelSession(sessionID string) error
    
    // CleanupSession 清理会话
    // 参数：会话ID
    CleanupSession(sessionID string)
    
    // Start 启动自动清理
    Start()
    
    // Stop 停止自动清理并清理所有会话
    Stop()
}
```

#### 配置参数

- **timeout**：会话超时时间，超过此时间未访问的会话将被自动清理
- **cleanupInterval**：自动清理检查间隔，定期扫描并清理过期会话

#### 会话生命周期

1. **创建**：调用 `CreateSession` 创建新会话，生成唯一ID
2. **使用**：通过会话ID获取上下文，进行AI对话操作
3. **更新**：每次调用 `GetSession` 会更新最后访问时间
4. **取消**：
   - 主动取消：调用 `CancelSession` 或返回的 `cancel` 函数
   - 自动取消：超时或上下文错误时自动清理
5. **清理**：从管理器中移除会话记录

#### 并发安全

上下文管理器使用读写锁（`sync.RWMutex`）保护内部状态，支持多个goroutine并发访问：

- 读操作（`GetSession`）使用读锁，允许并发读取
- 写操作（`CreateSession`、`CancelSession`、`CleanupSession`）使用写锁，确保数据一致性

#### 自动清理机制

启动 `Start()` 后，管理器会定期执行清理任务：

1. 检查所有会话的最后访问时间
2. 清理超过超时时间的会话
3. 清理上下文已取消或出错的会话
4. 调用 `Stop()` 时停止清理并清理所有剩余会话

#### 注意事项

1. **资源管理**：始终调用返回的 `cancel` 函数或 `CancelSession` 来释放资源
2. **启动清理**：在生产环境中应调用 `Start()` 启动自动清理
3. **优雅关闭**：应用退出前调用 `Stop()` 确保所有会话被正确清理
4. **会话ID唯一性**：使用 UUID 确保会话ID全局唯一

## 测试

运行测试：

```bash
# 运行所有测试
go test ./internal/service/ai/...

# 运行测试并显示覆盖率
go test -cover ./internal/service/ai/...

# 运行测试并生成详细输出
go test -v ./internal/service/ai/...

# 运行示例测试
go test -v -run Example ./internal/service/ai/...
```

### AIService - AI 服务接口

AI 服务接口定义了核心的对话功能，提供统一的 API 供上层调用。

#### 接口定义

```go
type AIService interface {
    // Chat 发起对话
    Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error)
    
    // ChatStream 流式对话（预留接口）
    ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error)
    
    // AbortChat 中止对话
    AbortChat(ctx context.Context, sessionID string) error
}
```

### GenkitService - Genkit AI 服务实现

基于 Firebase Genkit 的 AI 服务实现，提供完整的对话功能。

#### 主要功能

1. **对话处理**：处理用户消息，调用 Genkit 生成 AI 响应
2. **会话管理**：自动创建和管理对话会话
3. **参数配置**：支持温度、最大 token 数等高级参数
4. **错误处理**：统一的错误处理和日志记录
5. **上下文取消**：支持中止正在进行的对话

#### 使用示例

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/logger"
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/service/ai"
)

func main() {
    // 初始化 Genkit 客户端
    genkitClient := genkit.NewClient()
    err := genkitClient.Initialize(context.Background(), &genkit.Config{
        APIKey: "your-api-key",
        Model: "gemini-2.5-flash",
        DefaultTemperature: 0.7,
        DefaultMaxTokens: 2000,
    })
    if err != nil {
        panic(err)
    }
    
    // 创建上下文管理器
    contextManager := ai.NewContextManager(30*time.Minute, 5*time.Minute)
    contextManager.Start()
    defer contextManager.Stop()
    
    // 创建日志记录器
    log := logger.Default()
    
    // 创建 AI 服务
    aiService := ai.NewGenkitService(genkitClient, contextManager, log)
    
    // 发起对话
    req := &model.ChatRequest{
        Message: "你好，请介绍一下 Firebase",
    }
    
    resp, err := aiService.Chat(context.Background(), req)
    if err != nil {
        fmt.Printf("对话失败: %v\n", err)
        return
    }
    
    fmt.Printf("会话ID: %s\n", resp.SessionID)
    fmt.Printf("AI 响应: %s\n", resp.Message)
    fmt.Printf("使用的模型: %s\n", resp.Model)
    if resp.Usage != nil {
        fmt.Printf("Token 使用: %d (提示词: %d, 生成: %d)\n",
            resp.Usage.TotalTokens,
            resp.Usage.PromptTokens,
            resp.Usage.CompletionTokens)
    }
}
```

#### 高级参数配置

```go
// 使用自定义参数
temp := 0.8
maxTokens := 1000
topP := 0.9
topK := 40

req := &model.ChatRequest{
    Message: "写一首关于春天的诗",
    Options: &model.ChatOptions{
        Temperature: &temp,
        MaxTokens: &maxTokens,
        TopP: &topP,
        TopK: &topK,
    },
}

resp, err := aiService.Chat(ctx, req)
```

#### 继续现有会话

```go
// 第一次对话
req1 := &model.ChatRequest{
    Message: "我想了解 Go 语言",
}

resp1, err := aiService.Chat(ctx, req1)
if err != nil {
    // 处理错误
}

sessionID := resp1.SessionID

// 继续对话（使用相同的会话ID）
req2 := &model.ChatRequest{
    Message: "它有哪些优势？",
    SessionID: sessionID,
}

resp2, err := aiService.Chat(ctx, req2)
```

#### 中止对话

```go
// 在另一个 goroutine 中中止对话
go func() {
    time.Sleep(1 * time.Second)
    err := aiService.AbortChat(context.Background(), sessionID)
    if err != nil {
        fmt.Printf("中止失败: %v\n", err)
    }
}()

// 发起长时间对话
resp, err := aiService.Chat(ctx, req)
if err != nil {
    // 可能是上下文取消错误
}
```

#### 错误处理

服务层使用统一的错误类型：

```go
import "genkit-ai-service/pkg/errors"

resp, err := aiService.Chat(ctx, req)
if err != nil {
    switch e := err.(type) {
    case *errors.AppError:
        switch e.Code {
        case errors.CodeContextCancelled:
            fmt.Println("对话被取消")
        case errors.CodeAIServiceError:
            fmt.Println("AI 服务错误")
        case errors.CodeNotFound:
            fmt.Println("会话不存在")
        default:
            fmt.Printf("其他错误: %v\n", e)
        }
    default:
        fmt.Printf("未知错误: %v\n", err)
    }
}
```

#### 日志记录

服务层集成了结构化日志记录：

- 记录每次对话的开始和完成
- 记录会话 ID、模型、耗时、token 使用情况
- 记录错误和警告信息
- 支持上下文字段（如 sessionId、requestId）

日志示例：

```json
{
  "timestamp": "2025-10-15T10:30:00Z",
  "level": "INFO",
  "message": "开始处理对话请求",
  "fields": {
    "sessionId": "abc123",
    "message": "你好"
  }
}

{
  "timestamp": "2025-10-15T10:30:01Z",
  "level": "INFO",
  "message": "对话请求处理完成",
  "fields": {
    "sessionId": "abc123",
    "model": "gemini-2.5-flash",
    "duration": "1.2s",
    "tokens": {
      "promptTokens": 10,
      "completionTokens": 50,
      "totalTokens": 60
    }
  }
}
```

## 未来扩展

该模块为未来的高级功能预留了扩展空间：

1. **流式响应**：实现 `ChatStream` 接口，支持流式返回AI生成内容
2. **多模型支持**：支持不同的AI模型和提供商
3. **对话历史**：持久化对话历史到数据库
4. **RAG 集成**：集成检索增强生成功能
5. **多轮对话优化**：改进会话上下文管理，支持更长的对话历史

## 依赖

- `github.com/google/uuid`：用于生成唯一的会话ID
- Go 标准库：`context`、`sync`、`time`
