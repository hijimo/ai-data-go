# chatGenerateFlow 实现文档

## 概述

`chatGenerateFlow` 是基于 Google Genkit 实现的对话生成 Flow，负责处理用户消息并生成 AI 响应。该 Flow 集成了上下文管理、提示词构建、AI 生成、消息保存和向量生成等功能。

## 功能特性

### 核心功能

1. **智能上下文构建**
   - 自动调用 `contextBuildFlow` 构建对话上下文
   - 支持手动提供预构建的上下文
   - 整合摘要、长期记忆和短期消息

2. **提示词构建**
   - 系统提示词支持
   - 摘要上下文整合
   - 长期记忆整合
   - 短期消息历史整合
   - 结构化的提示词格式

3. **AI 生成**
   - 集成 Genkit Generate API
   - 支持自定义模型配置
   - 自动重试机制（最多 3 次）
   - Token 使用统计

4. **消息持久化**
   - 异步保存用户消息和 AI 响应
   - 自动关联会话和租户信息
   - 支持可选的消息保存

5. **向量生成**
   - 异步生成消息向量
   - 为用户消息和 AI 响应分别生成向量
   - 为后续的语义检索做准备

6. **权限控制**
   - 多租户隔离验证
   - 会话访问权限检查
   - 平台管理员特权支持

## 输入输出定义

### ChatGenerateInput

```go
type ChatGenerateInput struct {
    SessionID    string               `json:"sessionId" validate:"required,uuid"`
    UserMessage  string               `json:"userMessage" validate:"required,max=4000"`
    Context      *ContextBuildOutput  `json:"context,omitempty"`
    ModelConfig  *ModelConfig         `json:"modelConfig,omitempty"`
    SystemPrompt string               `json:"systemPrompt" validate:"max=1000"`
    SaveMessage  bool                 `json:"saveMessage"`
}
```

**字段说明：**

- `SessionID`: 会话 ID（必填，UUID 格式）
- `UserMessage`: 用户消息内容（必填，最大 4000 字符）
- `Context`: 预构建的上下文（可选，未提供时自动构建）
- `ModelConfig`: 模型配置（可选）
- `SystemPrompt`: 系统提示词（可选，最大 1000 字符）
- `SaveMessage`: 是否保存消息（默认 false）

### ModelConfig

```go
type ModelConfig struct {
    ModelName        string   `json:"modelName" validate:"required"`
    Temperature      float64  `json:"temperature" validate:"min=0,max=2"`
    TopP             float64  `json:"topP" validate:"min=0,max=1"`
    MaxTokens        int      `json:"maxTokens" validate:"min=1,max=4096"`
    StopSequences    []string `json:"stopSequences" validate:"max=4"`
    FrequencyPenalty float64  `json:"frequencyPenalty" validate:"min=-2,max=2"`
    PresencePenalty  float64  `json:"presencePenalty" validate:"min=-2,max=2"`
}
```

**字段说明：**

- `ModelName`: 模型名称（如 "gemini-1.5-flash"）
- `Temperature`: 温度参数（0-2，控制随机性）
- `TopP`: Top-P 采样参数（0-1）
- `MaxTokens`: 最大生成 Token 数（1-4096）
- `StopSequences`: 停止序列（最多 4 个）
- `FrequencyPenalty`: 频率惩罚（-2 到 2）
- `PresencePenalty`: 存在惩罚（-2 到 2）

### ChatGenerateOutput

```go
type ChatGenerateOutput struct {
    MessageID      string      `json:"messageId"`
    Response       string      `json:"response"`
    TokenUsage     TokenUsage  `json:"tokenUsage"`
    FinishReason   string      `json:"finishReason"`
    Model          string      `json:"model"`
    GenerationTime int64       `json:"generationTime"`
    ContextInfo    ContextInfo `json:"contextInfo"`
}
```

**字段说明：**

- `MessageID`: 消息 ID（UUID）
- `Response`: AI 生成的响应内容
- `TokenUsage`: Token 使用统计
- `FinishReason`: 完成原因（如 "stop"）
- `Model`: 使用的模型名称
- `GenerationTime`: 生成耗时（毫秒）
- `ContextInfo`: 上下文信息

## 执行流程

```
1. 参数验证
   ├─ 验证 SessionID 格式
   ├─ 验证 UserMessage 长度
   └─ 验证 SystemPrompt 长度

2. 权限验证
   ├─ 获取 JWT 声明
   ├─ 查询会话信息
   └─ 验证租户访问权限

3. 构建上下文
   ├─ 检查是否提供了预构建上下文
   ├─ 未提供则调用 contextBuildFlow
   └─ 获取上下文结果

4. 构建提示词
   ├─ 添加系统提示词
   ├─ 添加摘要上下文
   ├─ 添加长期记忆
   ├─ 添加短期消息历史
   └─ 添加当前用户消息

5. 调用 AI 生成
   ├─ 构建生成选项
   ├─ 调用 Genkit Generate API
   ├─ 失败时自动重试（最多 3 次）
   └─ 提取响应内容和 Token 统计

6. 保存消息（异步）
   ├─ 保存用户消息
   └─ 保存 AI 响应

7. 生成向量（异步）
   ├─ 为用户消息生成向量
   └─ 为 AI 响应生成向量

8. 返回结果
   ├─ 构建输出对象
   └─ 返回给调用方
```

## 使用示例

### 基本用法

```go
// 创建输入
input := flows.ChatGenerateInput{
    SessionID:   "550e8400-e29b-41d4-a716-446655440000",
    UserMessage: "你好，请介绍一下人工智能",
    SaveMessage: true,
}

// 调用 Flow
flow := genkit.LookupFlow[flows.ChatGenerateInput, flows.ChatGenerateOutput](
    g,
    "chatGenerateFlow",
)

output, err := flow.Run(ctx, input)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("AI 响应: %s\n", output.Response)
fmt.Printf("使用 Token: %d\n", output.TokenUsage.TotalTokens)
```

### 使用自定义模型配置

```go
input := flows.ChatGenerateInput{
    SessionID:   "550e8400-e29b-41d4-a716-446655440000",
    UserMessage: "写一首关于春天的诗",
    ModelConfig: &flows.ModelConfig{
        ModelName:   "gemini-1.5-pro",
        Temperature: 0.8,
        MaxTokens:   500,
    },
    SaveMessage: true,
}

output, err := flow.Run(ctx, input)
```

### 使用系统提示词

```go
input := flows.ChatGenerateInput{
    SessionID:    "550e8400-e29b-41d4-a716-446655440000",
    UserMessage:  "解释量子计算",
    SystemPrompt: "你是一个物理学专家，请用通俗易懂的语言解释复杂的物理概念",
    SaveMessage:  true,
}

output, err := flow.Run(ctx, input)
```

### 使用预构建上下文

```go
// 先构建上下文
contextFlow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
    g,
    "contextBuildFlow",
)

contextOutput, err := contextFlow.Run(ctx, flows.ContextBuildInput{
    SessionID:       "550e8400-e29b-41d4-a716-446655440000",
    UserQuery:       "继续讨论",
    MaxTokens:       4000,
    Strategy:        "full",
    IncludeSummary:  true,
    IncludeLongTerm: true,
    ShortTermWindow: 20,
})

// 使用预构建的上下文
input := flows.ChatGenerateInput{
    SessionID:   "550e8400-e29b-41d4-a716-446655440000",
    UserMessage: "继续讨论",
    Context:     &contextOutput,
    SaveMessage: true,
}

output, err := flow.Run(ctx, input)
```

## 错误处理

### 常见错误

1. **参数验证失败**

   ```
   错误: 参数验证失败: sessionId 不能为空
   原因: 未提供 SessionID
   解决: 确保提供有效的 UUID 格式的 SessionID
   ```

2. **权限验证失败**

   ```
   错误: 权限验证失败: 权限不足：无法访问其他租户的会话
   原因: 尝试访问其他租户的会话
   解决: 确保只访问当前租户的会话
   ```

3. **上下文构建失败**

   ```
   错误: 构建上下文失败: 会话不存在
   原因: 指定的会话不存在
   解决: 确保会话 ID 有效且未被删除
   ```

4. **AI 生成失败**

   ```
   错误: AI 生成失败: context deadline exceeded
   原因: AI 服务超时
   解决: 检查网络连接和 AI 服务状态，系统会自动重试
   ```

### 重试机制

Flow 内置了自动重试机制：

- 最大重试次数：3 次
- 重试间隔：递增（1秒、2秒、3秒）
- 重试条件：AI 生成失败时自动重试
- 日志记录：每次重试都会记录日志

## 性能考虑

### 异步操作

以下操作是异步执行的，不会阻塞主流程：

1. **消息保存**：在后台 goroutine 中保存用户消息和 AI 响应
2. **向量生成**：在后台 goroutine 中生成消息向量

### 性能指标

- **P50 延迟**：< 3 秒（不含 AI 生成时间）
- **P95 延迟**：< 5 秒（不含 AI 生成时间）
- **重试开销**：每次重试增加 1-3 秒延迟

### 优化建议

1. **预构建上下文**：对于需要精确控制上下文的场景，预先构建上下文可以减少延迟
2. **调整模型配置**：使用较小的 `MaxTokens` 可以加快生成速度
3. **禁用消息保存**：对于不需要持久化的场景，设置 `SaveMessage: false`

## 安全考虑

### 多租户隔离

- 所有会话访问都经过租户权限验证
- 平台管理员可以访问所有租户的会话
- 租户管理员只能访问自己租户的会话
- 权限验证失败会记录审计日志

### 输入验证

- SessionID 必须是有效的 UUID 格式
- UserMessage 长度限制为 4000 字符
- SystemPrompt 长度限制为 1000 字符
- 所有输入都经过严格验证

### 日志记录

- 记录所有 Flow 执行的关键步骤
- 记录权限验证失败的尝试
- 记录 AI 生成失败和重试信息
- 不记录敏感信息（如完整的用户消息内容）

## 监控和可观测性

### 日志字段

```go
logger.InfoContext(ctx, "开始生成 AI 响应", logger.Fields{
    "sessionId":     input.SessionID,
    "contextTokens": contextResult.TotalTokens,
    "promptLength":  len(prompt),
})
```

### 关键指标

- `sessionId`: 会话 ID
- `contextTokens`: 上下文 Token 数
- `promptLength`: 提示词长度
- `responseLength`: 响应长度
- `promptTokens`: 提示 Token 数
- `completionTokens`: 完成 Token 数
- `totalTokens`: 总 Token 数
- `retryCount`: 重试次数
- `generationTime`: 生成耗时

## 依赖服务

### 必需服务

1. **ContextService**: 上下文管理服务
2. **MessageRepo**: 消息仓储
3. **SessionRepo**: 会话仓储
4. **VectorService**: 向量服务
5. **Logger**: 日志服务

### 可选服务

- **CacheService**: 缓存服务（用于优化性能）
- **MonitoringService**: 监控服务（用于指标收集）

## 测试

### 单元测试

```bash
go test -v ./internal/genkit/flows -run TestValidateChatGenerateInput
go test -v ./internal/genkit/flows -run TestBuildPrompt
go test -v ./internal/genkit/flows -run TestGetModelName
go test -v ./internal/genkit/flows -run TestHasRole
```

### 集成测试

```bash
go test -v ./internal/genkit/flows -run TestChatGenerateFlowIntegration
```

## 未来改进

### 计划功能

1. **流式响应支持**：实现 `chatStreamFlow` 支持流式生成
2. **上下文缓存**：缓存频繁使用的上下文以提高性能
3. **智能重试策略**：根据错误类型选择不同的重试策略
4. **Token 预算管理**：集成 Token 预算检查，防止超额使用
5. **多模型支持**：支持在同一会话中切换不同的 AI 模型

### 性能优化

1. **并行处理**：并行执行上下文构建和向量生成
2. **批量操作**：批量保存消息和生成向量
3. **连接池**：优化数据库连接池配置
4. **缓存策略**：实现多级缓存策略

## 相关文档

- [contextBuildFlow 文档](./CONTEXT_BUILD_FLOW.md)
- [queryClassifyFlow 文档](./QUERY_CLASSIFY_FLOW.md)
- [contextOptimizeFlow 文档](./CONTEXT_OPTIMIZE_FLOW.md)
- [Genkit Flow 设计文档](./README.md)

## 变更日志

### v1.0.0 (2025-01-XX)

- 初始实现
- 支持基本的对话生成功能
- 集成上下文管理
- 实现消息保存和向量生成
- 添加权限验证和多租户隔离
- 实现自动重试机制
