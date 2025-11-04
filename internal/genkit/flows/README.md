# Genkit Flows 实现

本目录包含基于 Google Genkit Go SDK 的 Flow 实现。

## contextBuildFlow

### 功能描述

`contextBuildFlow` 用于构建智能对话上下文，支持三层记忆架构：

- **短期记忆**：最近的对话消息
- **长期记忆**：基于向量相似度检索的历史对话
- **摘要记忆**：压缩的对话历史摘要

### 输入参数

```go
type ContextBuildInput struct {
    SessionID       string // 会话ID（必填，UUID格式）
    UserQuery       string // 用户查询（必填，最大2000字符）
    MaxTokens       int    // 最大Token数（100-32000）
    Strategy        string // 策略：auto/short/full
    IncludeSummary  bool   // 是否包含摘要
    IncludeLongTerm bool   // 是否包含长期记忆
    ShortTermWindow int    // 短期消息窗口大小（1-50）
}
```

### 输出结果

```go
type ContextBuildOutput struct {
    SessionID         string           // 会话ID
    Summary           *SummaryContext  // 摘要上下文
    LongTermMemories  []MemoryContext  // 长期记忆列表
    ShortTermMessages []MessageContext // 短期消息列表
    TotalTokens       int              // 总Token数
    Strategy          string           // 使用的策略
    QualityScore      float64          // 质量评分（0-1）
    BuildTime         int64            // 构建耗时（毫秒）
}
```

### 使用示例

#### 1. 注册 Flow

```go
import (
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/genkit/flows"
    "genkit-ai-service/internal/service"
)

// 创建 Genkit 实例
g := genkit.Init(ctx, nil)

// 创建服务
contextSvc := service.NewContextService(...)

// 创建注册器
registry := genkit.NewRegistry(g, &genkit.Services{
    ContextService: contextSvc,
})

// 注册所有 Flow
registry.RegisterAllFlows(ctx)
```

#### 2. 调用 Flow

```go
// 查找 Flow
flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
    g,
    "contextBuildFlow",
)

// 准备输入
input := flows.ContextBuildInput{
    SessionID:       "550e8400-e29b-41d4-a716-446655440000",
    UserQuery:       "什么是人工智能？",
    MaxTokens:       4000,
    Strategy:        "auto",
    IncludeSummary:  true,
    IncludeLongTerm: true,
    ShortTermWindow: 10,
}

// 执行 Flow
output, err := flow.Run(ctx, input)
if err != nil {
    log.Fatal(err)
}

// 使用输出
fmt.Printf("总Token数: %d\n", output.TotalTokens)
fmt.Printf("质量评分: %.2f\n", output.QualityScore)
fmt.Printf("构建耗时: %dms\n", output.BuildTime)
```

#### 3. HTTP API 调用

```bash
curl -X POST http://localhost:8080/api/v1/context/build \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "userQuery": "什么是人工智能？",
    "maxTokens": 4000,
    "strategy": "auto",
    "includeSummary": true,
    "includeLongTerm": true,
    "shortTermWindow": 10
  }'
```

### 特性

1. **参数验证**：自动验证输入参数的有效性
2. **权限验证**：验证用户是否有权访问指定会话
3. **缓存支持**：服务层自动处理缓存，提高性能
4. **智能优化**：当Token超限时自动优化上下文
5. **质量评分**：计算上下文质量评分，帮助评估上下文质量
6. **性能监控**：记录构建耗时，便于性能分析

### 错误处理

Flow 可能返回以下错误：

- **参数验证失败**：输入参数不符合要求
- **权限验证失败**：用户无权访问指定会话
- **构建上下文失败**：服务层处理失败

### 性能指标

- **P50延迟**：< 200ms（不含AI生成时间）
- **P95延迟**：< 500ms（不含AI生成时间）
- **缓存命中率**：> 60%（相同查询）

### 注意事项

1. 必须先注册 Flow 才能调用
2. 需要有效的 JWT Token 进行认证
3. 会话必须存在且用户有权访问
4. 服务层会自动处理缓存和权限验证
5. 向量检索需要预先生成消息的向量嵌入

## 后续 Flow

## 已实现的 Flow

### queryClassifyFlow

查询分类 Flow，用于分析用户查询的类型和意图。

**输入**：

- `query`：用户查询文本
- `sessionId`：会话ID（可选）
- `recentMessages`：最近的消息（可选）

**输出**：

- `queryType`：查询类型（simple_question、followup_question、complex_query、reference_query、summarization、clarification）
- `intent`：查询意图
- `needsHistory`：是否需要历史上下文
- `needsLongTerm`：是否需要长期记忆
- `recommendedStrategy`：推荐的上下文策略
- `confidence`：分类置信度
- `entities`：关键实体列表

**特性**：

- 基于规则的快速分类
- AI 增强分类（当置信度低于 0.7 时）
- 自动策略推荐
- 实体提取

后续将实现以下 Flow：

- `contextOptimizeFlow`：上下文优化
- `chatGenerateFlow`：对话生成
- `memorySearchFlow`：记忆检索
- `summaryGenerateFlow`：摘要生成
- 等等...

## 测试

运行测试：

```bash
go test -v ./internal/genkit/flows/...
```

运行特定测试：

```bash
go test -v ./internal/genkit/flows/... -run TestContextBuildInput_Validation
```

## 参考文档

- [Genkit Go SDK 文档](https://firebase.google.com/docs/genkit/go/get-started)
- [设计文档](../../../.kiro/specs/genkit-session-management/design.md)
- [需求文档](../../../.kiro/specs/genkit-session-management/requirements.md)
