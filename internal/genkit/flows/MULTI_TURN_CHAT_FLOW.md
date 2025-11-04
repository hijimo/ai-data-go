# multiTurnChatFlow 实现文档

## 概述

`multiTurnChatFlow` 是一个用于管理多轮对话状态的 Genkit Flow，它负责跟踪对话轮次、评估会话健康度、生成建议操作，并自动调用 `chatGenerateFlow` 生成 AI 响应。

## 功能特性

### 1. 对话轮次跟踪

- 自动跟踪当前对话的轮次数
- 基于会话的消息计数器计算轮次

### 2. 会话状态管理

支持以下会话状态：

- `healthy`: 会话运行良好
- `active`: 会话处于活跃状态
- `needs_summary`: 需要生成摘要（消息数 > 20）
- `token_warning`: Token 使用率过高（> 80%）
- `needs_cleanup`: 上下文质量较低（< 0.6）

### 3. 健康度评估

评估维度：

- **消息数量**: 超过 20 条消息时，健康度 -0.2
- **Token 使用率**: 超过 80% 时，健康度 -0.3
- **上下文质量**: 低于 0.6 时，健康度 -0.2

健康评分范围：0.0 - 1.0

### 4. 智能建议生成

根据会话状态自动生成建议：

- 消息数量过多 → 建议生成摘要
- Token 使用率高 → 建议优化上下文
- 上下文质量低 → 建议重置上下文

### 5. 上下文重置

支持可选的上下文重置功能：

- 清理短期记忆
- 保留摘要
- 重置 Token 计数器

## 输入输出定义

### 输入 (MultiTurnChatInput)

```go
type MultiTurnChatInput struct {
    SessionID    string `json:"sessionId" validate:"required,uuid"`
    UserMessage  string `json:"userMessage" validate:"required,max=4000"`
    ResetContext bool   `json:"resetContext"`
}
```

**字段说明**：

- `sessionId`: 会话 ID（必填，UUID 格式）
- `userMessage`: 用户消息内容（必填，最大 4000 字符）
- `resetContext`: 是否重置上下文（可选，默认 false）

### 输出 (MultiTurnChatOutput)

```go
type MultiTurnChatOutput struct {
    SessionID      string              `json:"sessionId"`
    TurnNumber     int                 `json:"turnNumber"`
    SessionState   string              `json:"sessionState"`
    HealthScore    float64             `json:"healthScore"`
    TokenUsageRate float64             `json:"tokenUsageRate"`
    Suggestions    []string            `json:"suggestions"`
    ContextInfo    MultiTurnContextInfo `json:"contextInfo"`
    Response       string              `json:"response"`
    MessageID      string              `json:"messageId"`
}
```

**字段说明**：

- `sessionId`: 会话 ID
- `turnNumber`: 当前对话轮次
- `sessionState`: 会话状态
- `healthScore`: 健康评分（0-1）
- `tokenUsageRate`: Token 使用率（0-1）
- `suggestions`: 建议操作列表
- `contextInfo`: 上下文信息
- `response`: AI 生成的响应
- `messageId`: 消息 ID

### 上下文信息 (MultiTurnContextInfo)

```go
type MultiTurnContextInfo struct {
    TotalMessages            int     `json:"totalMessages"`
    TotalTokens              int     `json:"totalTokens"`
    MaxTokens                int     `json:"maxTokens"`
    QualityScore             float64 `json:"qualityScore"`
    LastSummaryAt            string  `json:"lastSummaryAt,omitempty"`
    MessagesSinceLastSummary int     `json:"messagesSinceLastSummary"`
}
```

## 执行流程

```
1. 参数验证
   ├─ 验证 sessionId 格式
   ├─ 验证 userMessage 非空
   └─ 验证消息长度限制

2. 权限验证
   ├─ 检查用户认证
   ├─ 验证会话访问权限
   └─ 记录权限验证日志

3. 获取会话信息
   └─ 从数据库查询会话详情

4. 上下文重置（可选）
   ├─ 清理短期记忆
   ├─ 保留摘要
   └─ 重置 Token 计数

5. 会话健康度评估
   ├─ 检查消息数量
   ├─ 计算 Token 使用率
   ├─ 评估上下文质量
   └─ 计算综合健康评分

6. 生成建议操作
   ├─ 根据会话状态生成建议
   └─ 根据健康评分生成额外建议

7. 调用 chatGenerateFlow
   ├─ 构建输入参数
   ├─ 生成 AI 响应
   └─ 保存消息

8. 构建输出
   ├─ 组装上下文信息
   ├─ 计算对话轮次
   └─ 返回完整输出
```

## 使用示例

### 基本用法

```go
// 创建输入
input := flows.MultiTurnChatInput{
    SessionID:   "550e8400-e29b-41d4-a716-446655440000",
    UserMessage: "你好，请介绍一下你自己",
}

// 调用 Flow
output, err := flow.Run(ctx, input)
if err != nil {
    log.Fatal(err)
}

// 处理输出
fmt.Printf("会话状态: %s\n", output.SessionState)
fmt.Printf("健康评分: %.2f\n", output.HealthScore)
fmt.Printf("AI 响应: %s\n", output.Response)
```

### 带上下文重置

```go
input := flows.MultiTurnChatInput{
    SessionID:    "550e8400-e29b-41d4-a716-446655440000",
    UserMessage:  "让我们重新开始讨论",
    ResetContext: true,
}

output, err := flow.Run(ctx, input)
```

### 检查建议

```go
output, err := flow.Run(ctx, input)
if err != nil {
    log.Fatal(err)
}

// 检查是否需要采取行动
if output.SessionState == "needs_summary" {
    fmt.Println("建议生成摘要:")
    for _, suggestion := range output.Suggestions {
        fmt.Printf("- %s\n", suggestion)
    }
}

if output.TokenUsageRate > 0.8 {
    fmt.Println("警告: Token 使用率过高")
}
```

## 会话状态说明

### healthy

- **条件**: 消息数 ≤ 20，Token 使用率 ≤ 80%，上下文质量 ≥ 0.6
- **健康评分**: 1.0
- **建议**: 无需特殊操作

### active

- **条件**: 会话有消息记录
- **健康评分**: 根据其他因素计算
- **建议**: 继续对话

### needs_summary

- **条件**: 消息数 > 20
- **健康评分**: 0.8
- **建议**: 生成对话摘要以优化上下文

### token_warning

- **条件**: Token 使用率 > 80%
- **健康评分**: 0.7
- **建议**: 优化上下文或生成摘要

### needs_cleanup

- **条件**: 上下文质量 < 0.6
- **健康评分**: 0.8
- **建议**: 重置上下文但保留摘要

## 性能指标

### 目标性能

- **执行时间**: < 5 秒（不含 AI 生成时间）
- **健康评估**: < 100 毫秒
- **建议生成**: < 50 毫秒

### 监控指标

- Flow 执行次数
- 平均执行时间
- 健康评分分布
- 会话状态分布
- Token 使用率分布

## 错误处理

### 常见错误

1. **参数验证失败**
   - 错误: `sessionId 不能为空`
   - 解决: 提供有效的 UUID 格式的 sessionId

2. **权限验证失败**
   - 错误: `权限不足：无法访问其他用户的会话`
   - 解决: 确保用户有权访问该会话

3. **会话不存在**
   - 错误: `获取会话信息失败`
   - 解决: 检查 sessionId 是否正确

4. **AI 生成失败**
   - 错误: `生成对话响应失败`
   - 解决: 检查 AI 服务状态，查看重试日志

## 日志记录

### 关键日志点

```go
// 开始执行
services.Logger.InfoContext(ctx, "开始多轮对话管理", logger.Fields{
    "sessionId":    input.SessionID,
    "messageCount": session.MessageCount,
    "resetContext": input.ResetContext,
})

// 健康度评估完成
services.Logger.InfoContext(ctx, "会话健康度评估完成", logger.Fields{
    "sessionId":    input.SessionID,
    "sessionState": sessionState,
    "healthScore":  healthScore,
})

// 执行完成
services.Logger.InfoContext(ctx, "多轮对话管理完成", logger.Fields{
    "sessionId":     input.SessionID,
    "turnNumber":    output.TurnNumber,
    "sessionState":  output.SessionState,
    "healthScore":   output.HealthScore,
    "executionTime": executionTime,
})
```

## 未来优化

### 待实现功能

1. **上下文重置逻辑**
   - 实现实际的上下文清理
   - 保留摘要和重要记忆
   - 重置 Token 计数器

2. **从数据库获取实际数据**
   - 从 `conversation_contexts` 表获取 Token 使用情况
   - 从 `conversation_contexts` 表获取上下文质量评分
   - 从 `conversation_summaries` 表获取最后摘要时间

3. **更精确的健康度评估**
   - 考虑对话时间跨度
   - 考虑用户满意度
   - 考虑响应质量

4. **智能建议优化**
   - 基于历史数据的个性化建议
   - 预测性建议（提前预警）
   - 可执行的建议（一键操作）

## 相关 Flow

- `chatGenerateFlow`: 生成 AI 响应
- `contextBuildFlow`: 构建对话上下文
- `summaryGenerateFlow`: 生成对话摘要
- `contextOptimizeFlow`: 优化上下文

## 测试

### 单元测试

```bash
go test -v ./internal/genkit/flows -run TestMultiTurnChatFlow
go test -v ./internal/genkit/flows -run TestMultiTurnChatOutput
go test -v ./internal/genkit/flows -run TestSessionHealthEvaluation
```

### 测试覆盖率

```bash
go test -cover ./internal/genkit/flows
```

## 参考文档

- [需求文档](../../../.kiro/specs/genkit-session-management/requirements.md) - 需求 7
- [设计文档](../../../.kiro/specs/genkit-session-management/design.md)
- [任务列表](../../../.kiro/specs/genkit-session-management/tasks.md) - 任务 13
