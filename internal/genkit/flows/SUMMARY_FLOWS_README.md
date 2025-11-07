# 摘要生成 Flow 实现文档

## 概述

本模块实现了基于 Genkit 的摘要生成和触发检查 Flow，用于智能管理会话摘要。

## Flow 列表

### 1. summaryGenerateFlow

**功能**：生成会话摘要

**输入参数**：

- `sessionId` (string, 必需): 会话ID
- `messageIds` ([]string, 可选): 消息ID列表
- `startMessageId` (string, 可选): 起始消息ID
- `endMessageId` (string, 可选): 结束消息ID
- `previousSummary` (string, 可选): 前一个摘要内容（用于增量摘要）
- `summaryType` (string, 必需): 摘要类型（incremental/full）
- `targetLength` (int, 必需): 目标长度（50-1000 Token）

**输出结果**：

- `summaryId` (string): 摘要ID
- `summary` (string): 摘要内容
- `tokenCount` (int): Token数量
- `messageCount` (int): 消息数量
- `startMessageId` (string): 起始消息ID
- `endMessageID` (string): 结束消息ID
- `qualityScore` (float64): 质量评分（0-1）
- `compressionRate` (float64): 压缩率
- `keyTopics` ([]string): 关键主题列表
- `generationTime` (int64): 生成耗时（毫秒）

**特性**：

- 支持增量摘要和完整摘要两种类型
- 自动计算质量评分和压缩率
- 提取关键主题
- 完整的权限验证和租户隔离
- 详细的日志记录

### 2. summaryTriggerCheckFlow

**功能**：检查是否需要生成摘要

**输入参数**：

- `sessionId` (string, 必需): 会话ID
- `checkMode` (string, 必需): 检查模式（auto/force）

**输出结果**：

- `shouldSummarize` (bool): 是否应该生成摘要
- `triggerReason` (string): 触发原因
- `messageIds` ([]string): 建议包含的消息ID列表
- `messageCount` (int): 消息数量
- `estimatedTokenSaving` (int): 估算的Token节省量
- `urgency` (float64): 紧急程度（0-1）
- `recommendedType` (string): 推荐的摘要类型
- `checkTime` (int64): 检查耗时（毫秒）

**特性**：

- 支持自动检查和强制触发两种模式
- 智能评估摘要紧急程度
- 估算Token节省量
- 推荐合适的摘要类型

## 使用示例

### 注册 Flow

```go
import (
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit/flows"
    "genkit-ai-service/internal/service/session"
)

func main() {
    g := genkit.New()
    summarySvc := session.NewSummaryService(...)
    
    // 注册摘要管理Flow
    flows.RegisterSummaryFlows(g, summarySvc)
}
```

### 调用摘要生成 Flow

```go
import (
    "context"
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit/flows"
)

func generateSummary(ctx context.Context, g *genkit.Genkit) {
    flow := genkit.LookupFlow[flows.SummaryGenerateInput, flows.SummaryGenerateOutput](
        g,
        "summaryGenerateFlow",
    )
    
    input := flows.SummaryGenerateInput{
        SessionID:    "session-uuid",
        SummaryType:  "full",
        TargetLength: 200,
    }
    
    output, err := flow.Run(ctx, input)
    if err != nil {
        // 处理错误
        return
    }
    
    // 使用输出结果
    fmt.Printf("摘要ID: %s\n", output.SummaryID)
    fmt.Printf("摘要内容: %s\n", output.Summary)
    fmt.Printf("质量评分: %.2f\n", output.QualityScore)
}
```

### 调用摘要触发检查 Flow

```go
func checkSummaryTrigger(ctx context.Context, g *genkit.Genkit) {
    flow := genkit.LookupFlow[flows.SummaryTriggerCheckInput, flows.SummaryTriggerCheckOutput](
        g,
        "summaryTriggerCheckFlow",
    )
    
    input := flows.SummaryTriggerCheckInput{
        SessionID: "session-uuid",
        CheckMode: "auto",
    }
    
    output, err := flow.Run(ctx, input)
    if err != nil {
        // 处理错误
        return
    }
    
    if output.ShouldSummarize {
        fmt.Printf("需要生成摘要: %s\n", output.TriggerReason)
        fmt.Printf("紧急程度: %.2f\n", output.Urgency)
        fmt.Printf("推荐类型: %s\n", output.RecommendedType)
    }
}
```

## 权限验证

所有 Flow 都实现了完整的权限验证：

1. **JWT 认证**：从上下文中获取 JWT Claims
2. **租户隔离**：验证用户只能访问自己租户的数据
3. **审计日志**：记录所有权限验证失败的尝试

## 错误处理

Flow 实现了完善的错误处理机制：

- 参数验证错误：返回详细的验证失败信息
- 权限验证错误：返回"未认证"或"权限不足"
- 服务层错误：包装并返回具体的错误信息
- 所有错误都会记录到日志中

## 日志记录

Flow 使用结构化日志记录所有关键操作：

- **INFO 级别**：Flow 开始、成功完成
- **WARN 级别**：权限验证失败
- **ERROR 级别**：参数验证失败、服务调用失败

日志字段包括：

- `session_id`: 会话ID
- `summary_type`: 摘要类型
- `target_length`: 目标长度
- `quality_score`: 质量评分
- `compression_rate`: 压缩率
- `generation_time_ms`: 生成耗时

## 测试

模块包含完整的单元测试：

```bash
# 运行所有摘要相关测试
go test -v ./internal/genkit/flows/ -run "TestSummary"

# 运行特定测试
go test -v ./internal/genkit/flows/ -run "TestSummaryGenerateFlow"
go test -v ./internal/genkit/flows/ -run "TestSummaryTriggerCheckFlow"
```

测试覆盖：

- ✅ 正常流程测试
- ✅ 带消息ID列表的测试
- ✅ 无效输入测试
- ✅ 未认证请求测试
- ✅ 强制模式测试
- ✅ 参数验证测试
- ✅ 性能基准测试

## 性能考虑

- Flow 执行时间记录在输出中
- 使用 mock 服务进行单元测试，避免实际服务调用
- 所有 UUID 解析都有错误处理
- 日志使用结构化格式，便于分析

## 依赖关系

- `internal/service/session.SummaryService`: 摘要服务接口
- `internal/service/auth`: JWT 认证和权限验证
- `internal/logger`: 结构化日志记录
- `github.com/firebase/genkit/go/genkit`: Genkit Go SDK
- `github.com/google/uuid`: UUID 处理

## 注意事项

1. **租户隔离**：所有操作都必须验证租户权限
2. **参数验证**：严格验证所有输入参数
3. **错误处理**：所有错误都应该被捕获和记录
4. **日志记录**：使用 logger.Fields 传递结构化字段
5. **测试覆盖**：确保所有边界情况都有测试覆盖

## 相关文档

- [需求文档](../../../.kiro/specs/genkit-session-management/requirements.md)
- [设计文档](../../../.kiro/specs/genkit-session-management/design.md)
- [任务列表](../../../.kiro/specs/genkit-session-management/tasks.md)
