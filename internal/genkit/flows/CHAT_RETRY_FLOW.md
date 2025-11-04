# chatRetryFlow 实现文档

## 概述

`chatRetryFlow` 是一个智能对话重试 Flow，用于处理 AI 生成失败的情况。它支持三种重试策略（simple、exponential、adaptive），并在所有重试失败后执行回退操作，确保系统的可靠性和用户体验。

## 功能特性

### 1. 三种重试策略

#### Simple（简单重试）

- **描述**: 使用固定间隔重试
- **重试间隔**: 1 秒
- **默认最大重试次数**: 3 次
- **适用场景**: 临时性网络波动、短暂的服务不可用

#### Exponential（指数退避）

- **描述**: 使用指数增长的重试间隔
- **重试间隔**: 1s, 2s, 4s, 8s, 16s（最大 30 秒）
- **默认最大重试次数**: 5 次
- **适用场景**: 速率限制、服务器负载过高

#### Adaptive（自适应）

- **描述**: 根据失败原因动态调整重试策略
- **智能调整**:
  - 速率限制 → 指数退避
  - 超时 → 减少上下文 + 延长间隔
  - 上下文长度超限 → 大幅减少上下文
  - 服务器错误 → 指数退避
- **默认最大重试次数**: 4 次
- **适用场景**: 复杂的生产环境，需要智能应对各种失败情况

### 2. 回退操作

当所有重试失败后，系统会执行以下回退操作（按顺序）：

1. **减少上下文**: 只保留用户消息，移除历史上下文
2. **使用备用模型**: 切换到备用 AI 模型（如果配置）
3. **返回预设响应**: 返回友好的错误提示

### 3. 详细的重试信息

每次重试都会记录：

- 尝试次数
- 失败原因
- 等待时间
- 时间戳

## 输入参数

```go
type ChatRetryInput struct {
    SessionID     string               // 会话ID（必填，UUID格式）
    UserMessage   string               // 用户消息（必填，最大4000字符）
    Context       *ContextBuildOutput  // 上下文（可选，未提供时自动构建）
    ModelConfig   *ModelConfig         // 模型配置（可选）
    SystemPrompt  string               // 系统提示词（可选，最大1000字符）
    SaveMessage   bool                 // 是否保存消息
    RetryStrategy string               // 重试策略（必填：simple/exponential/adaptive）
    MaxRetries    int                  // 最大重试次数（必填，1-10）
}
```

### 参数验证规则

- `sessionId`: 必填，必须是有效的 UUID 格式
- `userMessage`: 必填，长度 1-4000 字符
- `systemPrompt`: 可选，最大 1000 字符
- `retryStrategy`: 必填，只能是 `simple`、`exponential` 或 `adaptive`
- `maxRetries`: 必填，范围 1-10

## 输出结果

```go
type ChatRetryOutput struct {
    MessageID      string      // 消息ID
    Response       string      // AI响应内容
    TokenUsage     TokenUsage  // Token使用统计
    FinishReason   string      // 完成原因
    Model          string      // 使用的模型
    GenerationTime int64       // 生成时间（毫秒）
    ContextInfo    ContextInfo // 上下文信息
    RetryInfo      RetryInfo   // 重试信息
    FallbackUsed   bool        // 是否使用了回退
    FallbackReason string      // 回退原因
}

type RetryInfo struct {
    Strategy       string         // 使用的策略
    TotalAttempts  int            // 总尝试次数
    SuccessAttempt int            // 成功的尝试次数（0表示使用了回退）
    FailedAttempts []RetryAttempt // 失败的尝试列表
    TotalRetryTime int64          // 总重试时间（毫秒）
}

type RetryAttempt struct {
    AttemptNumber int    // 尝试次数
    Error         string // 错误信息
    WaitTime      int64  // 等待时间（毫秒）
    Timestamp     string // 时间戳
}
```

## 使用示例

### 示例 1: 简单重试策略

```go
input := ChatRetryInput{
    SessionID:     "550e8400-e29b-41d4-a716-446655440000",
    UserMessage:   "请帮我总结一下今天的会议内容",
    RetryStrategy: "simple",
    MaxRetries:    3,
    SaveMessage:   true,
}

output, err := flow.Run(ctx, input)
if err != nil {
    log.Printf("对话生成失败: %v", err)
    return
}

log.Printf("响应: %s", output.Response)
log.Printf("重试次数: %d", output.RetryInfo.TotalAttempts)
log.Printf("是否使用回退: %v", output.FallbackUsed)
```

### 示例 2: 指数退避策略

```go
input := ChatRetryInput{
    SessionID:     "550e8400-e29b-41d4-a716-446655440000",
    UserMessage:   "分析这段代码的性能问题",
    RetryStrategy: "exponential",
    MaxRetries:    5,
    SaveMessage:   true,
}

output, err := flow.Run(ctx, input)
if err != nil {
    log.Printf("对话生成失败: %v", err)
    return
}

// 检查重试信息
if output.RetryInfo.TotalAttempts > 1 {
    log.Printf("经过 %d 次重试后成功", output.RetryInfo.TotalAttempts)
    for _, attempt := range output.RetryInfo.FailedAttempts {
        log.Printf("尝试 %d 失败: %s", attempt.AttemptNumber, attempt.Error)
    }
}
```

### 示例 3: 自适应策略

```go
input := ChatRetryInput{
    SessionID:     "550e8400-e29b-41d4-a716-446655440000",
    UserMessage:   "基于历史对话，给出下一步建议",
    RetryStrategy: "adaptive",
    MaxRetries:    4,
    SaveMessage:   true,
    Context:       existingContext, // 提供已构建的上下文
}

output, err := flow.Run(ctx, input)
if err != nil {
    log.Printf("对话生成失败: %v", err)
    return
}

// 检查是否使用了回退
if output.FallbackUsed {
    log.Printf("使用了回退操作: %s", output.FallbackReason)
}

log.Printf("总重试时间: %d ms", output.RetryInfo.TotalRetryTime)
```

## 错误类型分析

Flow 会自动分析错误类型并采取相应的策略：

| 错误类型 | 关键词 | 处理策略 |
|---------|--------|---------|
| rate_limit | "rate limit", "too many requests" | 指数退避 |
| timeout | "timeout", "deadline exceeded" | 减少上下文 + 延长间隔 |
| context_length | "context length", "token limit" | 大幅减少上下文 |
| server_error | "server error", "internal error" | 指数退避 |
| invalid_request | "invalid", "bad request" | 固定间隔 |
| unknown | 其他 | 固定间隔 |

## 性能考虑

### 重试间隔

- **Simple**: 固定 1 秒
- **Exponential**: 1s → 2s → 4s → 8s → 16s（最大 30s）
- **Adaptive**: 根据错误类型动态调整

### 上下文优化

当检测到超时或上下文长度超限时，Flow 会自动优化提示词：

- 保留前 60% 的内容（系统提示词和重要上下文）
- 保留后 40% 的内容（用户消息）
- 中间部分用省略标记替代

### 超时控制

建议设置合理的上下文超时时间：

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

output, err := flow.Run(ctx, input)
```

## 监控和日志

Flow 会记录详细的日志信息：

```
[INFO] 开始对话重试流程 sessionId=xxx retryStrategy=adaptive maxRetries=4
[INFO] 执行自适应重试 attempt=1 maxRetries=4
[WARN] 重试失败，分析错误类型 attempt=1 error=timeout errorType=timeout
[INFO] 检测到超时，尝试减少上下文 originalTokens=3500
[INFO] 重试成功 attempt=2
[INFO] 对话重试流程完成 totalAttempts=2 successAttempt=2 fallbackUsed=false
```

## 最佳实践

### 1. 选择合适的重试策略

- **开发/测试环境**: 使用 `simple` 策略，快速失败
- **生产环境（低负载）**: 使用 `exponential` 策略
- **生产环境（高负载）**: 使用 `adaptive` 策略

### 2. 设置合理的最大重试次数

- **Simple**: 3 次（总耗时约 3 秒）
- **Exponential**: 5 次（总耗时约 31 秒）
- **Adaptive**: 4 次（根据错误类型动态调整）

### 3. 监控重试指标

定期检查以下指标：

- 重试成功率
- 平均重试次数
- 回退使用频率
- 各错误类型的分布

### 4. 优化上下文大小

如果频繁遇到超时或上下文长度超限：

- 减少短期消息窗口
- 优化摘要生成
- 清理低质量的长期记忆

## 与其他 Flow 的集成

### 与 chatGenerateFlow 的关系

`chatRetryFlow` 是 `chatGenerateFlow` 的增强版本：

- 包含所有 `chatGenerateFlow` 的功能
- 增加了智能重试机制
- 增加了回退操作
- 提供更详细的执行信息

### 与 multiTurnChatFlow 的集成

可以在 `multiTurnChatFlow` 中使用 `chatRetryFlow` 替代 `chatGenerateFlow`：

```go
// 在 multiTurnChatFlow 中使用 chatRetryFlow
chatRetryInput := ChatRetryInput{
    SessionID:     input.SessionID,
    UserMessage:   input.UserMessage,
    RetryStrategy: "adaptive",
    MaxRetries:    4,
    SaveMessage:   true,
}

chatOutput, err := executeChatRetryFlow(ctx, g, chatRetryInput, services)
```

## 故障排查

### 问题 1: 所有重试都失败

**可能原因**:

- AI 服务完全不可用
- 网络连接问题
- 配置错误（API 密钥无效）

**解决方案**:

1. 检查 AI 服务状态
2. 验证网络连接
3. 确认 API 密钥配置正确
4. 查看详细的错误日志

### 问题 2: 频繁触发回退操作

**可能原因**:

- 上下文过大
- 请求频率过高
- 模型配置不当

**解决方案**:

1. 减少上下文大小
2. 实施速率限制
3. 调整模型参数
4. 使用更强大的模型

### 问题 3: 重试时间过长

**可能原因**:

- 使用了 exponential 策略且重试次数过多
- 网络延迟高

**解决方案**:

1. 减少最大重试次数
2. 使用 simple 或 adaptive 策略
3. 设置合理的超时时间

## 需求映射

本实现满足以下需求：

- **需求 1**: Flow 定义和注册 ✅
  - 使用 `genkit.DefineFlow` 定义 Flow
  - 提供类型安全的输入输出
  - 支持参数验证

- **需求 8**: 对话重试和回退 Flow ✅
  - 支持三种重试策略（simple、exponential、adaptive）
  - 根据失败原因选择重试策略
  - 实现回退操作（减少上下文、使用备用模型、返回预设响应）
  - 记录所有重试尝试和回退操作

- **需求 32**: 降级策略 ✅
  - AI 服务不可用时执行回退操作
  - 提供预设响应作为最后的降级方案
  - 记录所有降级操作

## 测试覆盖

实现包含以下测试：

1. **单元测试**
   - 输入参数验证
   - 错误类型分析
   - 提示词优化
   - 重试信息结构

2. **性能测试**
   - 错误分析性能
   - 提示词优化性能

3. **集成测试**
   - 不同重试策略的行为
   - 回退操作的触发
   - 上下文优化逻辑

## 版本历史

- **v1.0.0** (2024-01-01): 初始实现
  - 支持三种重试策略
  - 实现回退操作
  - 提供详细的重试信息

## 相关文档

- [chatGenerateFlow 文档](./CHAT_GENERATE_FLOW.md)
- [multiTurnChatFlow 文档](./MULTI_TURN_CHAT_FLOW.md)
- [需求文档](../../../.kiro/specs/genkit-session-management/requirements.md)
- [设计文档](../../../.kiro/specs/genkit-session-management/design.md)
