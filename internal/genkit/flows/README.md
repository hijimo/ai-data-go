# Genkit Flows

本目录包含基于 Google Genkit 的 Flow 实现，用于会话管理模块的各种功能。

## 已实现的 Flow

### 1. contextBuildFlow - 上下文构建 Flow

智能构建对话上下文，包括短期记忆、长期记忆和摘要。

#### 功能特性

- **参数验证**：严格的输入参数验证
- **权限验证**：租户级别的访问控制
- **智能上下文构建**：
  - 短期记忆：最近 N 条消息
  - 长期记忆：基于向量相似度的历史记忆检索
  - 摘要记忆：压缩的对话历史
- **Token 优化**：自动裁剪以满足 Token 预算
- **质量评分**：评估构建的上下文质量
- **监控指标**：记录执行时间和状态
- **结构化日志**：详细的执行日志

#### 输入参数

```go
type ContextBuildInput struct {
    SessionID       string  // 会话ID（必填，UUID格式）
    UserQuery       string  // 用户查询（必填，最大2000字符）
    MaxTokens       int     // 最大Token数量（100-32000）
    Strategy        string  // 上下文策略：auto/short/full
    IncludeSummary  bool    // 是否包含摘要
    IncludeLongTerm bool    // 是否包含长期记忆
    ShortTermWindow int     // 短期记忆窗口大小（1-50）
}
```

#### 输出结果

```go
type ContextBuildOutput struct {
    SessionID         string           // 会话ID
    Summary           *SummaryContext  // 摘要上下文
    LongTermMemories  []MemoryContext  // 长期记忆列表
    ShortTermMessages []MessageContext // 短期消息列表
    TotalTokens       int              // 总Token数量
    Strategy          string           // 使用的策略
    QualityScore      float64          // 上下文质量评分（0-1）
    BuildTime         int64            // 构建耗时（毫秒）
}
```

#### 使用示例

```go
// 1. 注册 Flow
RegisterContextFlows(genkitInstance, contextService)

// 2. 查找 Flow
flow := genkit.LookupFlow[ContextBuildInput, ContextBuildOutput](
    genkitInstance,
    "contextBuildFlow",
)

// 3. 调用 Flow
input := ContextBuildInput{
    SessionID:       "session-uuid",
    UserQuery:       "用户的问题",
    MaxTokens:       4000,
    Strategy:        "auto",
    IncludeSummary:  true,
    IncludeLongTerm: true,
    ShortTermWindow: 10,
}

output, err := flow.Run(ctx, input)
if err != nil {
    // 处理错误
}

// 4. 使用输出结果
fmt.Printf("总Token数: %d\n", output.TotalTokens)
fmt.Printf("质量评分: %.2f\n", output.QualityScore)
```

## 文件结构

```
internal/genkit/flows/
├── README.md              # 本文件
├── types.go               # 共享类型定义
├── context_flows.go       # 上下文相关 Flow 实现
└── context_flows_test.go  # 单元测试
```

## 测试

运行测试：

```bash
cd internal/genkit/flows
go test -v
```

## 监控指标

Flow 执行会自动记录以下监控指标：

- **执行次数**：按 Flow 名称和状态（success/error）统计
- **执行时间**：记录每次执行的耗时
- **租户级别指标**：按租户ID统计使用情况

## 日志记录

Flow 使用结构化日志记录，包含以下信息：

- 执行开始/结束
- 参数验证结果
- 权限验证结果
- 服务调用结果
- 性能指标

日志格式：

```json
{
  "timestamp": "2025-11-06T14:05:11Z",
  "level": "INFO",
  "message": "开始执行上下文构建Flow",
  "fields": {
    "session_id": "uuid",
    "user_query": "查询内容",
    "max_tokens": 4000,
    "strategy": "auto"
  }
}
```

## 错误处理

Flow 会返回以下类型的错误：

- **参数验证错误**：400 Bad Request
- **权限验证错误**：401 Unauthorized / 403 Forbidden
- **服务层错误**：500 Internal Server Error

所有错误都遵循统一的错误格式，包含错误码、消息和详细信息。

## 性能要求

根据设计文档，contextBuildFlow 应满足以下性能要求：

- **P50 延迟**：< 200ms
- **P95 延迟**：< 500ms
- **并发支持**：> 100 QPS

## 安全考虑

- **租户隔离**：严格的租户级别访问控制
- **参数验证**：防止注入攻击
- **审计日志**：记录所有权限验证失败的尝试

## 下一步

待实现的 Flow：

- [ ] 13. 记忆管理 Flow（memorySearchFlow, memoryStoreFlow, memoryCleanupFlow）
- [ ] 14. 摘要生成 Flow（summaryGenerateFlow, summaryTriggerCheckFlow）
- [ ] 其他业务 Flow...

## 参考文档

- [设计文档](../../.kiro/specs/genkit-session-management/design.md)
- [需求文档](../../.kiro/specs/genkit-session-management/requirements.md)
- [任务列表](../../.kiro/specs/genkit-session-management/tasks.md)
