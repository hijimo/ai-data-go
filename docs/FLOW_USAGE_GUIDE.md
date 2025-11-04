# Genkit Flow 使用指南

## 概述

本文档介绍如何使用 Google Genkit Go SDK 中定义的各种 Flow。Flow 是 Genkit 的核心概念，代表一个可执行的工作流程，具有类型安全的输入和输出。

## 目录

- [Flow 基础](#flow-基础)
- [上下文管理 Flow](#上下文管理-flow)
- [对话生成 Flow](#对话生成-flow)
- [记忆管理 Flow](#记忆管理-flow)
- [摘要管理 Flow](#摘要管理-flow)
- [Token 管理 Flow](#token-管理-flow)
- [复合 Flow](#复合-flow)
- [最佳实践](#最佳实践)

## Flow 基础

### 什么是 Flow？

Flow 是 Genkit 中的可执行工作流程，具有以下特点：

1. **类型安全**：输入和输出都有明确的类型定义
2. **可组合**：Flow 可以调用其他 Flow
3. **可观测**：自动记录执行日志和指标
4. **可测试**：易于编写单元测试和集成测试

### Flow 的基本结构

```go
// 定义输入类型
type MyFlowInput struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

// 定义输出类型
type MyFlowOutput struct {
    Result string `json:"result"`
    Status string `json:"status"`
}

// 定义 Flow
func RegisterMyFlow(g *genkit.Genkit, svc MyService) {
    genkit.DefineFlow(
        g,
        "myFlow",
        func(ctx context.Context, input MyFlowInput) (MyFlowOutput, error) {
            // Flow 逻辑
            result, err := svc.DoSomething(ctx, input.Field1)
            if err != nil {
                return MyFlowOutput{}, err
            }
            
            return MyFlowOutput{
                Result: result,
                Status: "success",
            }, nil
        },
    )
}
```

### 调用 Flow

```go
// 查找 Flow
flow := genkit.LookupFlow[MyFlowInput, MyFlowOutput](g, "myFlow")

// 执行 Flow
output, err := flow.Run(ctx, MyFlowInput{
    Field1: "value1",
    Field2: 42,
})

if err != nil {
    log.Printf("Flow 执行失败: %v", err)
    return
}

log.Printf("Flow 结果: %s", output.Result)
```

## 上下文管理 Flow

### 1. contextBuildFlow - 构建会话上下文

**用途**：构建包含短期记忆、长期记忆和摘要的完整会话上下文。

**输入**：

```go
type ContextBuildInput struct {
    SessionID       string `json:"sessionId"`       // 会话 ID
    UserQuery       string `json:"userQuery"`       // 用户查询（可选）
    MaxTokens       int    `json:"maxTokens"`       // 最大 Token 数
    Strategy        string `json:"strategy"`        // 策略：auto/quality/speed
    IncludeSummary  bool   `json:"includeSummary"`  // 是否包含摘要
    IncludeLongTerm bool   `json:"includeLongTerm"` // 是否包含长期记忆
    ShortTermWindow int    `json:"shortTermWindow"` // 短期记忆窗口大小
}
```

**输出**：

```go
type ContextBuildOutput struct {
    SessionID         string                   `json:"sessionId"`
    Summary           *ConversationSummary     `json:"summary,omitempty"`
    LongTermMemories  []*ConversationMemory    `json:"longTermMemories"`
    ShortTermMessages []*ConversationMessage   `json:"shortTermMessages"`
    TotalTokens       int                      `json:"totalTokens"`
    Strategy          string                   `json:"strategy"`
    QualityScore      float64                  `json:"qualityScore"`
    BuildTime         int64                    `json:"buildTime"`
}
```

**使用示例**：

```go
// 查找 Flow
flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
    g,
    "contextBuildFlow",
)

// 执行 Flow
output, err := flow.Run(ctx, flows.ContextBuildInput{
    SessionID:       "session-uuid",
    UserQuery:       "用户的当前查询",
    MaxTokens:       4000,
    Strategy:        "auto",
    IncludeSummary:  true,
    IncludeLongTerm: true,
    ShortTermWindow: 10,
})

if err != nil {
    log.Printf("构建上下文失败: %v", err)
    return
}

log.Printf("上下文构建完成，总 Token: %d, 质量评分: %.2f",
    output.TotalTokens, output.QualityScore)
```

**应用场景**：

- 在生成 AI 回复前构建完整上下文
- 评估当前会话的上下文质量
- 为用户展示当前上下文的组成

### 2. queryClassifyFlow - 查询分类

**用途**：分析用户查询并推荐最佳上下文策略。

**输入**：

```go
type QueryClassifyInput struct {
    SessionID string `json:"sessionId"` // 会话 ID
    UserQuery string `json:"userQuery"` // 用户查询
}
```

**输出**：

```go
type QueryClassifyOutput struct {
    QueryType           string  `json:"queryType"`           // 查询类型
    NeedsHistory        bool    `json:"needsHistory"`        // 是否需要历史
    NeedsMemory         bool    `json:"needsMemory"`         // 是否需要记忆
    Complexity          float64 `json:"complexity"`          // 复杂度
    RecommendedStrategy string  `json:"recommendedStrategy"` // 推荐策略
    Confidence          float64 `json:"confidence"`          // 置信度
    Reasoning           string  `json:"reasoning"`           // 推理过程
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.QueryClassifyInput, flows.QueryClassifyOutput](
    g,
    "queryClassifyFlow",
)

output, err := flow.Run(ctx, flows.QueryClassifyInput{
    SessionID: "session-uuid",
    UserQuery: "上次我们讨论的技术方案进展如何？",
})

if err != nil {
    log.Printf("查询分类失败: %v", err)
    return
}

// 根据分类结果选择策略
if output.NeedsHistory {
    log.Println("需要加载历史上下文")
}

log.Printf("推荐策略: %s (置信度: %.2f)", 
    output.RecommendedStrategy, output.Confidence)
```

**应用场景**：

- 智能选择上下文策略
- 优化 Token 使用
- 提高响应速度

### 3. contextOptimizeFlow - 上下文优化

**用途**：优化现有上下文以减少 Token 使用。

**输入**：

```go
type ContextOptimizeInput struct {
    SessionID      string         `json:"sessionId"`      // 会话 ID
    CurrentContext ContextResult  `json:"currentContext"` // 当前上下文
    TargetTokens   int            `json:"targetTokens"`   // 目标 Token 数
    Strategy       string         `json:"strategy"`       // 优化策略
}
```

**输出**：

```go
type ContextOptimizeOutput struct {
    OptimizedContext ContextResult `json:"optimizedContext"` // 优化后的上下文
    OriginalTokens   int           `json:"originalTokens"`   // 原始 Token 数
    OptimizedTokens  int           `json:"optimizedTokens"`  // 优化后 Token 数
    TokensSaved      int           `json:"tokensSaved"`      // 节省的 Token 数
    QualityLoss      float64       `json:"qualityLoss"`      // 质量损失
    Strategy         string        `json:"strategy"`         // 使用的策略
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.ContextOptimizeInput, flows.ContextOptimizeOutput](
    g,
    "contextOptimizeFlow",
)

output, err := flow.Run(ctx, flows.ContextOptimizeInput{
    SessionID:      "session-uuid",
    CurrentContext: currentContext,
    TargetTokens:   3000,
    Strategy:       "balanced", // aggressive/balanced/conservative
})

if err != nil {
    log.Printf("上下文优化失败: %v", err)
    return
}

log.Printf("优化完成，节省 %d Token，质量损失: %.2f%%",
    output.TokensSaved, output.QualityLoss*100)
```

**优化策略**：

- **aggressive**：激进优化，最大化 Token 节省
- **balanced**：平衡优化，兼顾质量和效率
- **conservative**：保守优化，最小化质量损失

## 对话生成 Flow

### 4. chatGenerateFlow - 生成对话回复

**用途**：生成 AI 对话回复（非流式）。

**输入**：

```go
type ChatGenerateInput struct {
    SessionID     string              `json:"sessionId"`     // 会话 ID
    UserMessage   string              `json:"userMessage"`   // 用户消息
    ContextConfig ContextBuildInput   `json:"contextConfig"` // 上下文配置
    GenerateConfig GenerateConfig     `json:"generateConfig"` // 生成配置
}

type GenerateConfig struct {
    Temperature     float64 `json:"temperature"`     // 温度
    MaxOutputTokens int     `json:"maxOutputTokens"` // 最大输出 Token
    TopP            float64 `json:"topP"`            // Top-P
    TopK            int     `json:"topK"`            // Top-K
}
```

**输出**：

```go
type ChatGenerateOutput struct {
    MessageID   string      `json:"messageId"`   // 消息 ID
    Content     string      `json:"content"`     // 回复内容
    TokenStats  TokenStats  `json:"tokenStats"`  // Token 统计
    ContextUsed ContextInfo `json:"contextUsed"` // 使用的上下文
    GeneratedAt string      `json:"generatedAt"` // 生成时间
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.ChatGenerateInput, flows.ChatGenerateOutput](
    g,
    "chatGenerateFlow",
)

output, err := flow.Run(ctx, flows.ChatGenerateInput{
    SessionID:   "session-uuid",
    UserMessage: "请解释一下量子计算的基本原理",
    ContextConfig: flows.ContextBuildInput{
        MaxTokens:       4000,
        Strategy:        "auto",
        IncludeSummary:  true,
        IncludeLongTerm: true,
        ShortTermWindow: 10,
    },
    GenerateConfig: flows.GenerateConfig{
        Temperature:     0.7,
        MaxOutputTokens: 1000,
        TopP:            0.9,
        TopK:            40,
    },
})

if err != nil {
    log.Printf("生成回复失败: %v", err)
    return
}

log.Printf("生成的回复: %s", output.Content)
log.Printf("Token 使用: 输入=%d, 输出=%d, 总计=%d",
    output.TokenStats.InputTokens,
    output.TokenStats.OutputTokens,
    output.TokenStats.TotalTokens)
```

### 5. chatStreamFlow - 流式对话生成

**用途**：生成 AI 对话回复（流式）。

**输入**：与 `chatGenerateFlow` 相同

**输出**：

```go
type ChatStreamOutput struct {
    StreamChannel chan StreamChunk `json:"-"` // 流式输出通道
}

type StreamChunk struct {
    Type       string     `json:"type"`       // 块类型
    Content    string     `json:"content"`    // 内容
    TokenStats TokenStats `json:"tokenStats"` // Token 统计
    Error      error      `json:"error"`      // 错误
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.ChatStreamInput, flows.ChatStreamOutput](
    g,
    "chatStreamFlow",
)

output, err := flow.Run(ctx, flows.ChatStreamInput{
    SessionID:   "session-uuid",
    UserMessage: "请详细解释深度学习",
    // ... 其他配置
})

if err != nil {
    log.Printf("启动流式生成失败: %v", err)
    return
}

// 处理流式输出
for chunk := range output.StreamChannel {
    switch chunk.Type {
    case "start":
        log.Println("开始生成")
    case "content":
        fmt.Print(chunk.Content) // 逐步显示内容
    case "token_stats":
        log.Printf("Token 统计: %+v", chunk.TokenStats)
    case "end":
        log.Println("\n生成完成")
    case "error":
        log.Printf("生成错误: %v", chunk.Error)
    }
}
```

### 6. multiTurnChatFlow - 多轮对话管理

**用途**：管理多轮对话并评估会话健康度。

**输入**：

```go
type MultiTurnChatInput struct {
    SessionID    string `json:"sessionId"`    // 会话 ID
    CheckHealth  bool   `json:"checkHealth"`  // 是否检查健康度
    AutoOptimize bool   `json:"autoOptimize"` // 是否自动优化
}
```

**输出**：

```go
type MultiTurnChatOutput struct {
    SessionID     string        `json:"sessionId"`     // 会话 ID
    TurnCount     int           `json:"turnCount"`     // 轮次数
    HealthScore   float64       `json:"healthScore"`   // 健康评分
    Issues        []string      `json:"issues"`        // 问题列表
    Suggestions   []string      `json:"suggestions"`   // 建议列表
    ContextStatus ContextStatus `json:"contextStatus"` // 上下文状态
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.MultiTurnChatInput, flows.MultiTurnChatOutput](
    g,
    "multiTurnChatFlow",
)

output, err := flow.Run(ctx, flows.MultiTurnChatInput{
    SessionID:    "session-uuid",
    CheckHealth:  true,
    AutoOptimize: true,
})

if err != nil {
    log.Printf("多轮对话检查失败: %v", err)
    return
}

if output.HealthScore < 0.7 {
    log.Printf("会话健康度较低: %.2f", output.HealthScore)
    log.Printf("问题: %v", output.Issues)
    log.Printf("建议: %v", output.Suggestions)
}
```

### 7. chatRetryFlow - 对话重试

**用途**：重试失败的对话生成。

**输入**：

```go
type ChatRetryInput struct {
    SessionID  string `json:"sessionId"`  // 会话 ID
    MessageID  string `json:"messageId"`  // 失败的消息 ID
    Strategy   string `json:"strategy"`   // 重试策略
    MaxRetries int    `json:"maxRetries"` // 最大重试次数
}
```

**输出**：

```go
type ChatRetryOutput struct {
    MessageID  string `json:"messageId"`  // 新消息 ID
    Content    string `json:"content"`    // 回复内容
    RetryCount int    `json:"retryCount"` // 重试次数
    Strategy   string `json:"strategy"`   // 使用的策略
    Success    bool   `json:"success"`    // 是否成功
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.ChatRetryInput, flows.ChatRetryOutput](
    g,
    "chatRetryFlow",
)

output, err := flow.Run(ctx, flows.ChatRetryInput{
    SessionID:  "session-uuid",
    MessageID:  "failed-message-uuid",
    Strategy:   "exponential", // simple/exponential/adaptive
    MaxRetries: 3,
})

if err != nil {
    log.Printf("重试失败: %v", err)
    return
}

if output.Success {
    log.Printf("重试成功，重试了 %d 次", output.RetryCount)
} else {
    log.Println("重试失败，已达到最大重试次数")
}
```

## 记忆管理 Flow

### 8. memorySearchFlow - 搜索记忆

**用途**：基于向量相似度搜索长期记忆。

**输入**：

```go
type MemorySearchInput struct {
    SessionID     string  `json:"sessionId"`     // 会话 ID
    Query         string  `json:"query"`         // 搜索查询
    TopK          int     `json:"topK"`          // 返回结果数量
    MinSimilarity float32 `json:"minSimilarity"` // 最小相似度
    CrossSession  bool    `json:"crossSession"`  // 是否跨会话搜索
}
```

**输出**：

```go
type MemorySearchOutput struct {
    Memories   []*ConversationMemory `json:"memories"`   // 搜索结果
    TotalCount int                   `json:"totalCount"` // 总数
    SearchTime int64                 `json:"searchTime"` // 搜索耗时（毫秒）
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.MemorySearchInput, flows.MemorySearchOutput](
    g,
    "memorySearchFlow",
)

output, err := flow.Run(ctx, flows.MemorySearchInput{
    SessionID:     "session-uuid",
    Query:         "关于数据库优化的讨论",
    TopK:          5,
    MinSimilarity: 0.7,
    CrossSession:  false,
})

if err != nil {
    log.Printf("搜索记忆失败: %v", err)
    return
}

log.Printf("找到 %d 条相关记忆，耗时 %d ms", 
    output.TotalCount, output.SearchTime)

for _, memory := range output.Memories {
    log.Printf("记忆 [相似度: %.2f, 重要性: %.2f]: %s",
        memory.Similarity, memory.Importance, memory.Content)
}
```

**应用场景**：

- 检索历史讨论内容
- 查找相关知识点
- 构建上下文时获取长期记忆

### 9. memoryStoreFlow - 存储记忆

**用途**：创建新的长期记忆。

**输入**：

```go
type MemoryStoreInput struct {
    SessionID string                 `json:"sessionId"` // 会话 ID
    Content   string                 `json:"content"`   // 记忆内容
    Metadata  map[string]interface{} `json:"metadata"`  // 元数据
    ExpiresAt *time.Time             `json:"expiresAt"` // 过期时间
}
```

**输出**：

```go
type MemoryStoreOutput struct {
    ID         string                 `json:"id"`         // 记忆 ID
    SessionID  string                 `json:"sessionId"`  // 会话 ID
    Content    string                 `json:"content"`    // 内容
    Importance float64                `json:"importance"` // 重要性评分
    Metadata   map[string]interface{} `json:"metadata"`   // 元数据
    CreatedAt  string                 `json:"createdAt"`  // 创建时间
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.MemoryStoreInput, flows.MemoryStoreOutput](
    g,
    "memoryStoreFlow",
)

output, err := flow.Run(ctx, flows.MemoryStoreInput{
    SessionID: "session-uuid",
    Content:   "用户提到他们的系统使用 PostgreSQL 数据库，版本 14.5",
    Metadata: map[string]interface{}{
        "source": "user_mention",
        "tags":   []string{"技术栈", "数据库"},
    },
    ExpiresAt: nil, // 永不过期
})

if err != nil {
    log.Printf("存储记忆失败: %v", err)
    return
}

log.Printf("记忆已存储，ID: %s, 重要性: %.2f", 
    output.ID, output.Importance)
```

**应用场景**：

- 保存用户提到的重要信息
- 存储关键决策和结论
- 记录技术细节和配置

### 10. memoryCleanupFlow - 清理记忆

**用途**：批量清理过期或低质量的记忆。

**输入**：

```go
type MemoryCleanupInput struct {
    TenantID  string `json:"tenantId"`  // 租户 ID
    Strategy  string `json:"strategy"`  // 清理策略
    Mode      string `json:"mode"`      // 删除模式
    BatchSize int    `json:"batchSize"` // 批量大小
    DryRun    bool   `json:"dryRun"`    // 预览模式
}
```

**输出**：

```go
type MemoryCleanupOutput struct {
    DeletedCount        int    `json:"deletedCount"`        // 删除数量
    Strategy            string `json:"strategy"`            // 使用的策略
    Mode                string `json:"mode"`                // 删除模式
    EstimatedSpaceSaved string `json:"estimatedSpaceSaved"` // 预计节省空间
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.MemoryCleanupInput, flows.MemoryCleanupOutput](
    g,
    "memoryCleanupFlow",
)

// 先预览
previewOutput, err := flow.Run(ctx, flows.MemoryCleanupInput{
    TenantID:  "tenant-uuid",
    Strategy:  "expired", // expired/low_quality/unused/all
    Mode:      "soft",
    BatchSize: 100,
    DryRun:    true, // 预览模式
})

if err != nil {
    log.Printf("预览清理失败: %v", err)
    return
}

log.Printf("预览：将删除 %d 条记忆", previewOutput.DeletedCount)

// 确认后执行
output, err := flow.Run(ctx, flows.MemoryCleanupInput{
    TenantID:  "tenant-uuid",
    Strategy:  "expired",
    Mode:      "soft",
    BatchSize: 100,
    DryRun:    false,
})

if err != nil {
    log.Printf("清理失败: %v", err)
    return
}

log.Printf("已删除 %d 条记忆，节省空间: %s", 
    output.DeletedCount, output.EstimatedSpaceSaved)
```

**清理策略**：

- **expired**：清理已过期的记忆
- **low_quality**：清理低质量记忆（重要性低且访问少）
- **unused**：清理长期未访问的记忆（90天）
- **all**：清理所有符合条件的记忆

## 摘要管理 Flow

### 11. summaryGenerateFlow - 生成摘要

**用途**：为会话生成摘要。

**输入**：

```go
type SummaryGenerateInput struct {
    SessionID    string       `json:"sessionId"`    // 会话 ID
    MessageRange MessageRange `json:"messageRange"` // 消息范围
    Style        string       `json:"style"`        // 摘要风格
}

type MessageRange struct {
    Start int `json:"start"` // 起始位置
    End   int `json:"end"`   // 结束位置
}
```

**输出**：

```go
type SummaryGenerateOutput struct {
    ID           string   `json:"id"`           // 摘要 ID
    SessionID    string   `json:"sessionId"`    // 会话 ID
    Content      string   `json:"content"`      // 摘要内容
    KeyTopics    []string `json:"keyTopics"`    // 关键主题
    MessageCount int      `json:"messageCount"` // 消息数量
    QualityScore float64  `json:"qualityScore"` // 质量评分
    CreatedAt    string   `json:"createdAt"`    // 创建时间
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.SummaryGenerateInput, flows.SummaryGenerateOutput](
    g,
    "summaryGenerateFlow",
)

output, err := flow.Run(ctx, flows.SummaryGenerateInput{
    SessionID: "session-uuid",
    MessageRange: flows.MessageRange{
        Start: 0,
        End:   50,
    },
    Style: "concise", // concise/detailed/bullet_points
})

if err != nil {
    log.Printf("生成摘要失败: %v", err)
    return
}

log.Printf("摘要已生成，ID: %s", output.ID)
log.Printf("关键主题: %v", output.KeyTopics)
log.Printf("质量评分: %.2f", output.QualityScore)
log.Printf("摘要内容:\n%s", output.Content)
```

**摘要风格**：

- **concise**：简洁摘要，突出要点
- **detailed**：详细摘要，包含更多细节
- **bullet_points**：要点列表，结构化展示

### 12. summaryTriggerFlow - 检查摘要触发条件

**用途**：检查是否应该生成摘要。

**输入**：

```go
type SummaryTriggerInput struct {
    SessionID string `json:"sessionId"` // 会话 ID
}
```

**输出**：

```go
type SummaryTriggerOutput struct {
    ShouldGenerate    bool              `json:"shouldGenerate"`    // 是否应生成
    Score             float64           `json:"score"`             // 综合得分
    Reasons           []string          `json:"reasons"`           // 触发原因
    EstimatedBenefit  EstimatedBenefit  `json:"estimatedBenefit"`  // 预计收益
}

type EstimatedBenefit struct {
    TokenSavings        int     `json:"tokenSavings"`        // Token 节省
    QualityImprovement  float64 `json:"qualityImprovement"`  // 质量提升
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.SummaryTriggerInput, flows.SummaryTriggerOutput](
    g,
    "summaryTriggerFlow",
)

output, err := flow.Run(ctx, flows.SummaryTriggerInput{
    SessionID: "session-uuid",
})

if err != nil {
    log.Printf("检查触发条件失败: %v", err)
    return
}

if output.ShouldGenerate {
    log.Printf("建议生成摘要，得分: %.2f", output.Score)
    log.Printf("原因: %v", output.Reasons)
    log.Printf("预计节省 %d Token", output.EstimatedBenefit.TokenSavings)
    
    // 生成摘要
    // ...
}
```

**触发条件**：

1. 消息数量达到阈值（默认 50 条）
2. Token 使用率较高（> 80%）
3. 距离上次摘要时间较长（> 1 小时）
4. 会话复杂度较高
5. 用户明确请求

### 13. summaryQualityFlow - 评估摘要质量

**用途**：评估现有摘要的质量。

**输入**：

```go
type SummaryQualityInput struct {
    SummaryID string `json:"summaryId"` // 摘要 ID
}
```

**输出**：

```go
type SummaryQualityOutput struct {
    SummaryID    string              `json:"summaryId"`    // 摘要 ID
    OverallScore float64             `json:"overallScore"` // 总体评分
    Dimensions   QualityDimensions   `json:"dimensions"`   // 各维度评分
    Issues       []string            `json:"issues"`       // 问题列表
    Suggestions  []string            `json:"suggestions"`  // 改进建议
}

type QualityDimensions struct {
    Completeness float64 `json:"completeness"` // 完整性
    Accuracy     float64 `json:"accuracy"`     // 准确性
    Conciseness  float64 `json:"conciseness"`  // 简洁性
    Relevance    float64 `json:"relevance"`    // 相关性
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.SummaryQualityInput, flows.SummaryQualityOutput](
    g,
    "summaryQualityFlow",
)

output, err := flow.Run(ctx, flows.SummaryQualityInput{
    SummaryID: "summary-uuid",
})

if err != nil {
    log.Printf("质量评估失败: %v", err)
    return
}

log.Printf("摘要质量评分: %.2f", output.OverallScore)
log.Printf("完整性: %.2f, 准确性: %.2f, 简洁性: %.2f, 相关性: %.2f",
    output.Dimensions.Completeness,
    output.Dimensions.Accuracy,
    output.Dimensions.Conciseness,
    output.Dimensions.Relevance)

if len(output.Issues) > 0 {
    log.Printf("发现问题: %v", output.Issues)
    log.Printf("改进建议: %v", output.Suggestions)
}
```

## Token 管理 Flow

### 14. tokenBudgetFlow - Token 预算管理

**用途**：管理和监控 Token 使用预算。

**输入**：

```go
type TokenBudgetInput struct {
    SessionID string `json:"sessionId"` // 会话 ID
    Budget    int    `json:"budget"`    // 预算
}
```

**输出**：

```go
type TokenBudgetOutput struct {
    SessionID        string      `json:"sessionId"`        // 会话 ID
    Budget           int         `json:"budget"`           // 预算
    Used             int         `json:"used"`             // 已使用
    Remaining        int         `json:"remaining"`        // 剩余
    UtilizationRate  float64     `json:"utilizationRate"`  // 使用率
    Status           string      `json:"status"`           // 状态
    Suggestions      []string    `json:"suggestions"`      // 建议
    Prediction       Prediction  `json:"prediction"`       // 预测
}

type Prediction struct {
    EstimatedTurnsRemaining int    `json:"estimatedTurnsRemaining"` // 预计剩余轮次
    EstimatedTimeRemaining  string `json:"estimatedTimeRemaining"`  // 预计剩余时间
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.TokenBudgetInput, flows.TokenBudgetOutput](
    g,
    "tokenBudgetFlow",
)

output, err := flow.Run(ctx, flows.TokenBudgetInput{
    SessionID: "session-uuid",
    Budget:    10000,
})

if err != nil {
    log.Printf("预算检查失败: %v", err)
    return
}

log.Printf("Token 使用情况: %d/%d (%.1f%%)",
    output.Used, output.Budget, output.UtilizationRate*100)

switch output.Status {
case "critical":
    log.Println("警告：Token 即将耗尽！")
    log.Printf("建议: %v", output.Suggestions)
case "warning":
    log.Println("注意：Token 使用率较高")
case "healthy":
    log.Println("Token 使用正常")
}

log.Printf("预计还可进行 %d 轮对话", 
    output.Prediction.EstimatedTurnsRemaining)
```

### 15. tokenOptimizeFlow - Token 优化

**用途**：优化 Token 使用。

**输入**：

```go
type TokenOptimizeInput struct {
    SessionID       string  `json:"sessionId"`       // 会话 ID
    Content         string  `json:"content"`         // 要优化的内容
    TargetReduction float64 `json:"targetReduction"` // 目标减少比例
    Strategy        string  `json:"strategy"`        // 优化策略
}
```

**输出**：

```go
type TokenOptimizeOutput struct {
    OriginalTokens   int     `json:"originalTokens"`   // 原始 Token 数
    OptimizedTokens  int     `json:"optimizedTokens"`  // 优化后 Token 数
    TokensSaved      int     `json:"tokensSaved"`      // 节省的 Token 数
    ReductionRate    float64 `json:"reductionRate"`    // 减少比例
    OptimizedContent string  `json:"optimizedContent"` // 优化后的内容
    QualityScore     float64 `json:"qualityScore"`     // 质量评分
    Strategy         string  `json:"strategy"`         // 使用的策略
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.TokenOptimizeInput, flows.TokenOptimizeOutput](
    g,
    "tokenOptimizeFlow",
)

output, err := flow.Run(ctx, flows.TokenOptimizeInput{
    SessionID:       "session-uuid",
    Content:         longContent,
    TargetReduction: 0.3, // 减少 30%
    Strategy:        "smart", // compress/summarize/truncate/smart
})

if err != nil {
    log.Printf("Token 优化失败: %v", err)
    return
}

log.Printf("优化完成：%d -> %d Token (节省 %d, %.1f%%)",
    output.OriginalTokens,
    output.OptimizedTokens,
    output.TokensSaved,
    output.ReductionRate*100)
log.Printf("质量评分: %.2f", output.QualityScore)
```

**优化策略**：

- **compress**：压缩内容，去除冗余
- **summarize**：生成摘要
- **truncate**：截断内容
- **smart**：智能选择最佳策略

### 16. tokenAnalysisFlow - Token 使用分析

**用途**：分析 Token 使用情况。

**输入**：

```go
type TokenAnalysisInput struct {
    TenantID  string    `json:"tenantId"`  // 租户 ID
    TimeRange TimeRange `json:"timeRange"` // 时间范围
    Dimension string    `json:"dimension"` // 分析维度
}

type TimeRange struct {
    Start string `json:"start"` // 开始时间
    End   string `json:"end"`   // 结束时间
}
```

**输出**：

```go
type TokenAnalysisOutput struct {
    Dimension   string                 `json:"dimension"`   // 分析维度
    TotalTokens int                    `json:"totalTokens"` // 总 Token 数
    Breakdown   map[string]int         `json:"breakdown"`   // 分类统计
    TopSessions []SessionTokenUsage    `json:"topSessions"` // Top 会话
    Trends      TrendAnalysis          `json:"trends"`      // 趋势分析
    Suggestions []string               `json:"suggestions"` // 优化建议
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.TokenAnalysisInput, flows.TokenAnalysisOutput](
    g,
    "tokenAnalysisFlow",
)

output, err := flow.Run(ctx, flows.TokenAnalysisInput{
    TenantID: "tenant-uuid",
    TimeRange: flows.TimeRange{
        Start: "2024-01-01T00:00:00Z",
        End:   "2024-01-31T23:59:59Z",
    },
    Dimension: "usage", // usage/trend/cost/efficiency
})

if err != nil {
    log.Printf("Token 分析失败: %v", err)
    return
}

log.Printf("总 Token 使用: %d", output.TotalTokens)
log.Printf("输入: %d, 输出: %d", 
    output.Breakdown["input"], output.Breakdown["output"])
log.Printf("日均使用: %d", output.Trends.DailyAverage)
log.Printf("优化建议: %v", output.Suggestions)
```

## 复合 Flow

### 17. completeConversationFlow - 完整对话流程

**用途**：编排完整的对话生成流程，包括上下文构建、生成、记忆存储等。

**输入**：

```go
type CompleteConversationInput struct {
    SessionID      string            `json:"sessionId"`      // 会话 ID
    UserMessage    string            `json:"userMessage"`    // 用户消息
    ContextConfig  ContextBuildInput `json:"contextConfig"`  // 上下文配置
    GenerateConfig GenerateConfig    `json:"generateConfig"` // 生成配置
    StoreMemory    bool              `json:"storeMemory"`    // 是否存储记忆
    CheckSummary   bool              `json:"checkSummary"`   // 是否检查摘要
}
```

**输出**：

```go
type CompleteConversationOutput struct {
    MessageID      string            `json:"messageId"`      // 消息 ID
    Content        string            `json:"content"`        // 回复内容
    TokenStats     TokenStats        `json:"tokenStats"`     // Token 统计
    Steps          []StepResult      `json:"steps"`          // 步骤结果
    TotalTime      int64             `json:"totalTime"`      // 总耗时
    MemoryStored   bool              `json:"memoryStored"`   // 是否存储了记忆
    SummaryCreated bool              `json:"summaryCreated"` // 是否创建了摘要
}

type StepResult struct {
    Name     string `json:"name"`     // 步骤名称
    Success  bool   `json:"success"`  // 是否成功
    Duration int64  `json:"duration"` // 耗时（毫秒）
    Error    string `json:"error"`    // 错误信息
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.CompleteConversationInput, flows.CompleteConversationOutput](
    g,
    "completeConversationFlow",
)

output, err := flow.Run(ctx, flows.CompleteConversationInput{
    SessionID:   "session-uuid",
    UserMessage: "请帮我分析一下这个问题",
    ContextConfig: flows.ContextBuildInput{
        MaxTokens:       4000,
        Strategy:        "auto",
        IncludeSummary:  true,
        IncludeLongTerm: true,
        ShortTermWindow: 10,
    },
    GenerateConfig: flows.GenerateConfig{
        Temperature:     0.7,
        MaxOutputTokens: 1000,
    },
    StoreMemory:  true,
    CheckSummary: true,
})

if err != nil {
    log.Printf("完整对话流程失败: %v", err)
    return
}

log.Printf("对话完成，总耗时: %d ms", output.TotalTime)
log.Printf("回复: %s", output.Content)

// 查看各步骤耗时
for _, step := range output.Steps {
    if step.Success {
        log.Printf("✓ %s: %d ms", step.Name, step.Duration)
    } else {
        log.Printf("✗ %s: %s", step.Name, step.Error)
    }
}

if output.SummaryCreated {
    log.Println("已自动生成摘要")
}
```

**执行步骤**：

1. 查询分类（可选）
2. 构建上下文
3. 生成回复
4. 保存消息
5. 存储记忆（可选）
6. 检查摘要触发（可选）
7. 生成摘要（如果需要）

### 18. batchConversationFlow - 批量对话处理

**用途**：批量处理多个对话请求。

**输入**：

```go
type BatchConversationInput struct {
    Conversations   []ConversationRequest `json:"conversations"`   // 对话请求列表
    Concurrency     int                   `json:"concurrency"`     // 并发数
    FailureStrategy string                `json:"failureStrategy"` // 失败策略
}

type ConversationRequest struct {
    SessionID   string `json:"sessionId"`   // 会话 ID
    UserMessage string `json:"userMessage"` // 用户消息
}
```

**输出**：

```go
type BatchConversationOutput struct {
    Total          int                      `json:"total"`          // 总数
    Successful     int                      `json:"successful"`     // 成功数
    Failed         int                      `json:"failed"`         // 失败数
    Results        []ConversationResult     `json:"results"`        // 结果列表
    ProcessingTime int64                    `json:"processingTime"` // 处理耗时
}

type ConversationResult struct {
    SessionID string `json:"sessionId"` // 会话 ID
    Success   bool   `json:"success"`   // 是否成功
    MessageID string `json:"messageId"` // 消息 ID
    Content   string `json:"content"`   // 回复内容
    Error     string `json:"error"`     // 错误信息
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.BatchConversationInput, flows.BatchConversationOutput](
    g,
    "batchConversationFlow",
)

output, err := flow.Run(ctx, flows.BatchConversationInput{
    Conversations: []flows.ConversationRequest{
        {SessionID: "session-1", UserMessage: "消息1"},
        {SessionID: "session-2", UserMessage: "消息2"},
        {SessionID: "session-3", UserMessage: "消息3"},
    },
    Concurrency:     3,
    FailureStrategy: "continue", // continue/abort
})

if err != nil {
    log.Printf("批量处理失败: %v", err)
    return
}

log.Printf("批量处理完成: 成功 %d/%d, 耗时 %d ms",
    output.Successful, output.Total, output.ProcessingTime)

// 处理结果
for _, result := range output.Results {
    if result.Success {
        log.Printf("✓ 会话 %s: %s", result.SessionID, result.Content)
    } else {
        log.Printf("✗ 会话 %s: %s", result.SessionID, result.Error)
    }
}
```

### 19. sessionHealthCheckFlow - 会话健康检查

**用途**：全面检查会话健康状态。

**输入**：

```go
type SessionHealthCheckInput struct {
    SessionID string   `json:"sessionId"` // 会话 ID
    Checks    []string `json:"checks"`    // 检查项
    AutoFix   bool     `json:"autoFix"`   // 是否自动修复
}
```

**输出**：

```go
type SessionHealthCheckOutput struct {
    SessionID      string                  `json:"sessionId"`      // 会话 ID
    OverallHealth  float64                 `json:"overallHealth"`  // 总体健康度
    Status         string                  `json:"status"`         // 状态
    Checks         map[string]CheckResult  `json:"checks"`         // 检查结果
    AutoFixApplied bool                    `json:"autoFixApplied"` // 是否应用了自动修复
    FixedIssues    []string                `json:"fixedIssues"`    // 已修复的问题
    Recommendations []string               `json:"recommendations"` // 建议
}

type CheckResult struct {
    Status string   `json:"status"` // 状态：healthy/warning/critical
    Score  float64  `json:"score"`  // 评分
    Issues []string `json:"issues"` // 问题列表
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.SessionHealthCheckInput, flows.SessionHealthCheckOutput](
    g,
    "sessionHealthCheckFlow",
)

output, err := flow.Run(ctx, flows.SessionHealthCheckInput{
    SessionID: "session-uuid",
    Checks:    []string{"context", "token", "memory", "summary", "performance"},
    AutoFix:   true,
})

if err != nil {
    log.Printf("健康检查失败: %v", err)
    return
}

log.Printf("会话健康度: %.2f (%s)", output.OverallHealth, output.Status)

// 查看各项检查结果
for checkName, result := range output.Checks {
    log.Printf("%s: %s (%.2f)", checkName, result.Status, result.Score)
    if len(result.Issues) > 0 {
        log.Printf("  问题: %v", result.Issues)
    }
}

if output.AutoFixApplied {
    log.Printf("已自动修复: %v", output.FixedIssues)
}

if len(output.Recommendations) > 0 {
    log.Printf("建议: %v", output.Recommendations)
}
```

**检查项**：

- **context**：上下文质量和完整性
- **token**：Token 使用情况
- **memory**：记忆存储和检索
- **summary**：摘要质量和时效性
- **performance**：响应时间和性能

## 最佳实践

### 1. Flow 组合

合理组合 Flow 以实现复杂功能：

```go
// 智能对话流程
func SmartConversation(ctx context.Context, sessionID, userMessage string) error {
    // 1. 先分类查询
    classifyFlow := genkit.LookupFlow[flows.QueryClassifyInput, flows.QueryClassifyOutput](
        g, "queryClassifyFlow",
    )
    
    classifyResult, err := classifyFlow.Run(ctx, flows.QueryClassifyInput{
        SessionID: sessionID,
        UserQuery: userMessage,
    })
    if err != nil {
        return err
    }
    
    // 2. 根据分类结果构建上下文
    contextFlow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
        g, "contextBuildFlow",
    )
    
    contextResult, err := contextFlow.Run(ctx, flows.ContextBuildInput{
        SessionID:       sessionID,
        UserQuery:       userMessage,
        Strategy:        classifyResult.RecommendedStrategy,
        IncludeLongTerm: classifyResult.NeedsMemory,
        IncludeSummary:  classifyResult.NeedsHistory,
    })
    if err != nil {
        return err
    }
    
    // 3. 生成回复
    chatFlow := genkit.LookupFlow[flows.ChatGenerateInput, flows.ChatGenerateOutput](
        g, "chatGenerateFlow",
    )
    
    _, err = chatFlow.Run(ctx, flows.ChatGenerateInput{
        SessionID:   sessionID,
        UserMessage: userMessage,
        // ... 使用上面的上下文结果
    })
    
    return err
}
```

### 2. 错误处理

正确处理 Flow 执行错误：

```go
output, err := flow.Run(ctx, input)
if err != nil {
    // 检查错误类型
    switch {
    case errors.Is(err, ErrSessionNotFound):
        log.Println("会话不存在")
        // 创建新会话
    case errors.Is(err, ErrTokenExceeded):
        log.Println("Token 超限")
        // 生成摘要或优化上下文
    case errors.Is(err, ErrAIServiceTimeout):
        log.Println("AI 服务超时")
        // 重试或降级
    default:
        log.Printf("未知错误: %v", err)
    }
    return
}
```

### 3. 上下文传递

在 Flow 之间传递上下文信息：

```go
// 在上下文中设置会话信息
ctx = context.WithValue(ctx, "session_id", sessionID)
ctx = context.WithValue(ctx, "user_id", userID)
ctx = context.WithValue(ctx, "tenant_id", tenantID)

// Flow 内部可以获取这些信息
sessionID := ctx.Value("session_id").(string)
```

### 4. 性能优化

优化 Flow 执行性能：

```go
// 并行执行独立的 Flow
var wg sync.WaitGroup
var contextResult flows.ContextBuildOutput
var summaryResult flows.SummaryTriggerOutput

wg.Add(2)

// 并行构建上下文
go func() {
    defer wg.Done()
    contextResult, _ = contextFlow.Run(ctx, contextInput)
}()

// 并行检查摘要触发
go func() {
    defer wg.Done()
    summaryResult, _ = summaryTriggerFlow.Run(ctx, summaryInput)
}()

wg.Wait()

// 使用结果
if summaryResult.ShouldGenerate {
    // 生成摘要
}
```

### 5. 监控和日志

为 Flow 添加监控和日志：

```go
// 记录 Flow 执行
startTime := time.Now()
output, err := flow.Run(ctx, input)
duration := time.Since(startTime)

// 记录指标
metrics.RecordFlowDuration("myFlow", duration)
metrics.RecordFlowExecution("myFlow", err == nil)

// 记录日志
logger.InfoContext(ctx, "Flow 执行完成",
    "flow", "myFlow",
    "duration_ms", duration.Milliseconds(),
    "success", err == nil,
)
```

### 6. 测试 Flow

编写 Flow 的单元测试：

```go
func TestContextBuildFlow(t *testing.T) {
    // 准备测试环境
    ctx := context.Background()
    g := setupTestGenkit(t)
    
    // 注册 Flow
    flows.RegisterContextFlows(g, mockContextService)
    
    // 查找 Flow
    flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
        g, "contextBuildFlow",
    )
    
    // 执行测试
    output, err := flow.Run(ctx, flows.ContextBuildInput{
        SessionID: "test-session",
        MaxTokens: 4000,
        Strategy:  "auto",
    })
    
    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, output)
    assert.Equal(t, "test-session", output.SessionID)
    assert.Greater(t, output.TotalTokens, 0)
}
```

## 常见问题

### Q1: Flow 执行超时怎么办？

A: 可以设置上下文超时：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := flow.Run(ctx, input)
if err == context.DeadlineExceeded {
    log.Println("Flow 执行超时")
}
```

### Q2: 如何在 Flow 中访问数据库？

A: 通过服务层访问：

```go
func RegisterMyFlow(g *genkit.Genkit, svc MyService) {
    genkit.DefineFlow(g, "myFlow", func(ctx context.Context, input MyInput) (MyOutput, error) {
        // 通过服务层访问数据库
        data, err := svc.GetData(ctx, input.ID)
        if err != nil {
            return MyOutput{}, err
        }
        
        return MyOutput{Data: data}, nil
    })
}
```

### Q3: Flow 之间如何共享数据？

A: 使用上下文或返回值：

```go
// 方法1：通过上下文
ctx = context.WithValue(ctx, "shared_data", data)

// 方法2：通过返回值
output1, _ := flow1.Run(ctx, input1)
output2, _ := flow2.Run(ctx, flows.Input2{
    Data: output1.Result,
})
```

### Q4: 如何处理 Flow 中的并发？

A: 使用 goroutine 和 channel：

```go
genkit.DefineFlow(g, "parallelFlow", func(ctx context.Context, input Input) (Output, error) {
    results := make(chan Result, len(input.Items))
    
    for _, item := range input.Items {
        go func(item Item) {
            result := processItem(item)
            results <- result
        }(item)
    }
    
    // 收集结果
    var allResults []Result
    for i := 0; i < len(input.Items); i++ {
        allResults = append(allResults, <-results)
    }
    
    return Output{Results: allResults}, nil
})
```

## 总结

Genkit Flow 提供了强大的工作流编排能力，通过合理使用和组合各种 Flow，可以构建复杂的 AI 应用。关键要点：

1. **类型安全**：充分利用 Go 的类型系统
2. **可组合**：将复杂逻辑拆分为多个 Flow
3. **错误处理**：正确处理各种错误情况
4. **性能优化**：合理使用并发和缓存
5. **可观测性**：添加日志和监控
6. **测试**：编写完善的单元测试和集成测试

---

**文档版本**: v1.0.0  
**最后更新**: 2024-01-01  
**维护者**: 开发团队
