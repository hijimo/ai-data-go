# 记忆管理 Flow 使用指南

## 概述

记忆管理Flow提供了三个核心功能：

1. **memorySearchFlow** - 基于向量相似度检索记忆
2. **memoryStoreFlow** - 存储新的记忆
3. **memoryCleanupFlow** - 清理过期或低质量的记忆

## Flow 定义

### 1. memorySearchFlow - 记忆检索

**功能**：基于向量相似度检索相关的历史对话记忆

**输入参数** (`MemorySearchInput`):

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "query": "用户查询文本",
  "topK": 5,
  "minSimilarity": 0.7,
  "timeRangeDays": 30,
  "memoryTypes": ["long_term"],
  "includeCrossSessions": false
}
```

**字段说明**：

- `sessionId` (必须): 会话ID
- `query` (必须): 查询文本，用于生成向量进行相似度搜索
- `topK` (可选): 返回结果数量，默认5，范围1-20
- `minSimilarity` (可选): 最小相似度阈值，默认0.7，范围0-1
- `timeRangeDays` (可选): 时间范围（天数），0表示不限制
- `memoryTypes` (可选): 记忆类型过滤，可选值：short_term, long_term, summary
- `includeCrossSessions` (可选): 是否包含跨会话检索，默认false

**输出结果** (`MemorySearchOutput`):

```json
{
  "memories": [
    {
      "id": "memory-uuid",
      "sessionId": "session-uuid",
      "memoryType": "long_term",
      "content": "记忆内容",
      "tokenCount": 150,
      "similarity": 0.85,
      "importance": 0.8,
      "score": 0.68,
      "accessCount": 5,
      "createdAt": "2024-01-01T12:00:00Z",
      "lastAccessAt": "2024-01-02T12:00:00Z",
      "metadata": {}
    }
  ],
  "totalFound": 5,
  "returnedCount": 5,
  "searchTime": 120,
  "averageSimilarity": 0.82,
  "searchStrategy": "session"
}
```

**使用示例**：

```go
// 在Handler中调用
flow := genkit.LookupFlow[flows.MemorySearchInput, flows.MemorySearchOutput](
    genkitClient,
    "memorySearchFlow",
)

input := flows.MemorySearchInput{
    SessionID:     sessionID,
    Query:         userQuery,
    TopK:          5,
    MinSimilarity: 0.7,
}

output, err := flow.Run(ctx, input)
if err != nil {
    // 处理错误
}

// 使用检索结果
for _, memory := range output.Memories {
    fmt.Printf("找到相关记忆: %s (相似度: %.2f)\n", memory.Content, memory.Similarity)
}
```

### 2. memoryStoreFlow - 记忆存储

**功能**：将对话消息转换为长期记忆并存储

**输入参数** (`MemoryStoreInput`):

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "messageIds": ["msg-uuid-1", "msg-uuid-2"],
  "memoryType": "long_term",
  "content": "要存储的记忆内容",
  "importance": 0.8,
  "expirationDays": 90,
  "metadata": {
    "source": "user_message",
    "tags": ["important"]
  }
}
```

**字段说明**：

- `sessionId` (必须): 会话ID
- `messageIds` (可选): 关联的消息ID列表
- `memoryType` (必须): 记忆类型，可选值：short_term, long_term, summary
- `content` (必须): 记忆内容，最大4000字符
- `importance` (可选): 重要性评分，默认0.5，范围0-1
- `expirationDays` (可选): 过期天数，0表示不过期
- `metadata` (可选): 元数据，用于存储额外信息

**输出结果** (`MemoryStoreOutput`):

```json
{
  "memoryId": "memory-uuid",
  "sessionId": "session-uuid",
  "memoryType": "long_term",
  "tokenCount": 150,
  "importance": 0.8,
  "expiresAt": "2024-04-01T12:00:00Z",
  "vectorStatus": "generated",
  "storeTime": 250
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.MemoryStoreInput, flows.MemoryStoreOutput](
    genkitClient,
    "memoryStoreFlow",
)

input := flows.MemoryStoreInput{
    SessionID:      sessionID,
    MessageIDs:     []string{messageID1, messageID2},
    MemoryType:     "long_term",
    Content:        "重要的对话内容",
    Importance:     0.8,
    ExpirationDays: 90,
}

output, err := flow.Run(ctx, input)
if err != nil {
    // 处理错误
}

fmt.Printf("记忆已存储，ID: %s\n", output.MemoryID)
```

### 3. memoryCleanupFlow - 记忆清理

**功能**：根据策略清理过期或低质量的记忆

**输入参数** (`MemoryCleanupInput`):

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "strategy": "expired",
  "mode": "soft",
  "batchSize": 100,
  "execute": false
}
```

**字段说明**：

- `sessionId` (可选): 会话ID，为空则清理租户所有记忆
- `strategy` (必须): 清理策略
  - `expired`: 清理已过期的记忆
  - `low_quality`: 清理低质量记忆（重要性<0.3且访问次数<2）
  - `unused`: 清理未使用记忆（90天未访问）
  - `all`: 清理所有记忆
- `mode` (必须): 清理模式
  - `soft`: 软删除（标记is_deleted=true）
  - `hard`: 硬删除（物理删除）
- `batchSize` (可选): 批量处理大小，默认100，范围10-1000
- `execute` (可选): 是否执行删除，false时仅预览

**输出结果** (`MemoryCleanupOutput`):

```json
{
  "cleanedCount": 15,
  "freedSpace": 102400,
  "details": [
    {
      "memoryId": "memory-uuid",
      "reason": "已过期",
      "size": 6826,
      "createdAt": "2023-01-01T12:00:00Z",
      "lastAccess": "2023-01-02T12:00:00Z"
    }
  ],
  "preview": false,
  "cleanupTime": 180
}
```

**使用示例**：

```go
flow := genkit.LookupFlow[flows.MemoryCleanupInput, flows.MemoryCleanupOutput](
    genkitClient,
    "memoryCleanupFlow",
)

// 预览模式：查看将要清理的记忆
previewInput := flows.MemoryCleanupInput{
    SessionID: sessionID,
    Strategy:  "expired",
    Mode:      "soft",
    BatchSize: 100,
    Execute:   false, // 预览模式
}

previewOutput, err := flow.Run(ctx, previewInput)
if err != nil {
    // 处理错误
}

fmt.Printf("预览：将清理 %d 条记忆，释放 %d 字节\n", 
    previewOutput.CleanedCount, previewOutput.FreedSpace)

// 确认后执行清理
if userConfirmed {
    executeInput := previewInput
    executeInput.Execute = true
    
    output, err := flow.Run(ctx, executeInput)
    if err != nil {
        // 处理错误
    }
    
    fmt.Printf("已清理 %d 条记忆\n", output.CleanedCount)
}
```

## 注册Flow

在应用启动时注册所有记忆管理Flow：

```go
import (
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit/flows"
    "genkit-ai-service/internal/service"
)

func initializeGenkit(memorySvc service.MemoryService) *genkit.Genkit {
    g := genkit.Init(context.Background(), nil)
    
    // 注册记忆管理Flow
    flows.RegisterMemoryFlows(g, memorySvc)
    
    return g
}
```

## 错误处理

所有Flow都会返回详细的错误信息：

```go
output, err := flow.Run(ctx, input)
if err != nil {
    // 错误类型判断
    switch {
    case strings.Contains(err.Error(), "参数验证失败"):
        // 处理参数错误
    case strings.Contains(err.Error(), "权限不足"):
        // 处理权限错误
    case strings.Contains(err.Error(), "记忆检索失败"):
        // 处理检索错误
    default:
        // 处理其他错误
    }
}
```

## 性能考虑

1. **记忆检索**：
   - 平均响应时间：100-300ms（P95）
   - 建议TopK不超过20
   - 使用合适的相似度阈值（0.7-0.8）

2. **记忆存储**：
   - 平均响应时间：200-500ms
   - 向量生成是异步的
   - 批量存储时注意批次大小

3. **记忆清理**：
   - 建议使用预览模式先查看
   - 批量处理大小建议100-500
   - 定期清理可以优化性能

## 监控指标

Flow执行会自动记录以下指标：

- 执行次数
- 执行时间
- 成功/失败率
- 检索结果数量
- 清理数量

可以通过日志查看详细信息：

```
INFO: 记忆检索完成 session_id=xxx found_count=5 duration_ms=120
INFO: 记忆存储完成 memory_id=xxx token_count=150 duration_ms=250
INFO: 记忆清理完成 cleaned_count=15 freed_space=102400 duration_ms=180
```

## 最佳实践

1. **记忆检索**：
   - 根据查询类型调整TopK和相似度阈值
   - 使用时间范围过滤减少检索范围
   - 跨会话检索仅在必要时使用

2. **记忆存储**：
   - 合理设置重要性评分
   - 为重要记忆设置较长的过期时间
   - 使用元数据存储额外的上下文信息

3. **记忆清理**：
   - 定期执行清理任务（建议每周）
   - 先使用预览模式确认
   - 根据业务需求选择合适的清理策略

## 相关需求

本实现满足以下需求：

- 需求 2.1: 长期记忆存储
- 需求 2.2: 向量相似度检索
- 需求 2.3: 记忆访问统计
- 需求 3.1: 记忆清理策略
- 需求 3.2: 软删除和硬删除

## 下一步

- 实现摘要生成Flow（任务14）
- 实现API Handler层（任务15-17）
- 配置路由（任务18-20）
