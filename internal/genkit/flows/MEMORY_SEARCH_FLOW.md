# memorySearchFlow 文档

## 概述

`memorySearchFlow` 是一个基于向量相似度的记忆检索 Flow，用于从会话的长期记忆中检索与查询文本最相关的记忆片段。支持单会话检索和跨会话检索（同租户内），并提供丰富的过滤选项。

## 功能特性

- ✅ 基于向量相似度的语义检索
- ✅ 支持单会话和跨会话检索
- ✅ 多维度过滤（记忆类型、时间范围、重要性）
- ✅ 综合评分（相似度 × 重要性）
- ✅ 异步访问统计更新
- ✅ 多租户数据隔离
- ✅ 完整的权限验证

## 输入参数

### MemorySearchInput

```go
type MemorySearchInput struct {
    SessionID            string   `json:"sessionId"`            // 会话ID（必填）
    Query                string   `json:"query"`                // 查询文本（必填，最大2000字符）
    TopK                 int      `json:"topK"`                 // 返回结果数量（1-20）
    MinSimilarity        float32  `json:"minSimilarity"`        // 最小相似度阈值（0-1）
    TimeRangeDays        int      `json:"timeRangeDays"`        // 时间范围（0-365天，0表示不限制）
    MemoryTypes          []string `json:"memoryTypes"`          // 记忆类型过滤（可选）
    IncludeCrossSessions bool     `json:"includeCrossSessions"` // 是否包含跨会话检索
}
```

### 参数说明

| 参数 | 类型 | 必填 | 说明 | 默认值 | 限制 |
|------|------|------|------|--------|------|
| sessionId | string | 是 | 会话ID | - | 必须是有效的UUID |
| query | string | 是 | 查询文本 | - | 1-2000字符 |
| topK | int | 是 | 返回结果数量 | - | 1-20 |
| minSimilarity | float32 | 是 | 最小相似度 | - | 0-1 |
| timeRangeDays | int | 否 | 时间范围（天） | 0 | 0-365 |
| memoryTypes | []string | 否 | 记忆类型过滤 | [] | short_term, long_term, summary |
| includeCrossSessions | bool | 否 | 跨会话检索 | false | - |

### 记忆类型

- `short_term`: 短期记忆
- `long_term`: 长期记忆
- `summary`: 摘要记忆

## 输出结果

### MemorySearchOutput

```go
type MemorySearchOutput struct {
    Memories          []MemoryResult `json:"memories"`          // 记忆结果列表
    TotalFound        int            `json:"totalFound"`        // 找到的总数
    ReturnedCount     int            `json:"returnedCount"`     // 返回的数量
    SearchTime        int64          `json:"searchTime"`        // 搜索耗时（毫秒）
    AverageSimilarity float32        `json:"averageSimilarity"` // 平均相似度
    SearchStrategy    string         `json:"searchStrategy"`    // 搜索策略
}
```

### MemoryResult

```go
type MemoryResult struct {
    ID           string                 `json:"id"`           // 记忆ID
    SessionID    string                 `json:"sessionId"`    // 会话ID
    MemoryType   string                 `json:"memoryType"`   // 记忆类型
    Content      string                 `json:"content"`      // 内容
    TokenCount   int                    `json:"tokenCount"`   // Token数量
    Similarity   float32                `json:"similarity"`   // 相似度（0-1）
    Importance   float32                `json:"importance"`   // 重要性（0-1）
    Score        float32                `json:"score"`        // 综合得分
    AccessCount  int                    `json:"accessCount"`  // 访问次数
    CreatedAt    string                 `json:"createdAt"`    // 创建时间
    LastAccessAt string                 `json:"lastAccessAt"` // 最后访问时间
    Metadata     map[string]interface{} `json:"metadata"`     // 元数据
}
```

### 搜索策略

- `single_session`: 单会话检索
- `cross_sessions`: 跨会话检索（同租户内）

## 使用示例

### 示例1：基本的单会话检索

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "query": "如何使用Python处理JSON数据？",
  "topK": 5,
  "minSimilarity": 0.7
}
```

**响应示例：**

```json
{
  "memories": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "sessionId": "550e8400-e29b-41d4-a716-446655440000",
      "memoryType": "long_term",
      "content": "Python中可以使用json模块来处理JSON数据...",
      "tokenCount": 50,
      "similarity": 0.92,
      "importance": 0.85,
      "score": 0.782,
      "accessCount": 3,
      "createdAt": "2025-11-01T10:00:00Z",
      "lastAccessAt": "2025-11-01T12:30:00Z",
      "metadata": {
        "source": "user_message",
        "keywords": ["python", "json", "数据处理"]
      }
    }
  ],
  "totalFound": 1,
  "returnedCount": 1,
  "searchTime": 85,
  "averageSimilarity": 0.92,
  "searchStrategy": "single_session"
}
```

### 示例2：带过滤条件的检索

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "query": "机器学习算法",
  "topK": 10,
  "minSimilarity": 0.6,
  "timeRangeDays": 30,
  "memoryTypes": ["long_term"]
}
```

### 示例3：跨会话检索

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "query": "数据库优化技巧",
  "topK": 10,
  "minSimilarity": 0.7,
  "includeCrossSessions": true
}
```

## 工作流程

### 1. 参数验证

- 验证会话ID格式
- 验证查询文本长度
- 验证TopK范围
- 验证相似度阈值
- 验证记忆类型

### 2. 权限验证

- 从上下文获取JWT声明
- 验证用户已认证
- 提取租户ID

### 3. 生成查询向量

- 调用向量服务生成查询文本的向量
- 处理向量生成失败的情况

### 4. 执行向量检索

**单会话检索：**

- 如果有过滤条件，使用 `SearchByVectorWithFilters`
- 否则使用 `SearchByVector`

**跨会话检索：**

- 使用 `SearchByVectorCrossSessions`
- 限制在同一租户内

### 5. 计算相似度和得分

- 计算余弦相似度
- 计算综合得分 = 相似度 × 重要性
- 格式化时间字段

### 6. 异步更新访问统计

- 在后台goroutine中批量更新访问次数
- 更新最后访问时间
- 记录更新结果

### 7. 构建输出

- 计算平均相似度
- 记录搜索耗时
- 返回结果

## 性能指标

### 目标性能

- **P50延迟**: < 100ms
- **P95延迟**: < 300ms
- **并发支持**: > 100 QPS

### 性能优化

1. **向量索引优化**
   - 使用IVFFlat索引
   - 定期重建索引

2. **查询优化**
   - 使用参数化查询
   - 限制返回结果数量
   - 应用早期过滤

3. **异步处理**
   - 访问统计异步更新
   - 避免阻塞主流程

## 错误处理

### 常见错误

| 错误代码 | 错误信息 | 原因 | 解决方案 |
|---------|---------|------|---------|
| 400 | 参数验证失败 | 输入参数不符合要求 | 检查参数格式和范围 |
| 401 | 未认证 | JWT Token无效或缺失 | 提供有效的认证Token |
| 403 | 权限不足 | 尝试访问其他租户的数据 | 确保访问自己租户的数据 |
| 500 | 生成查询向量失败 | 向量服务异常 | 检查向量服务状态 |
| 500 | 向量检索失败 | 数据库查询异常 | 检查数据库连接和索引 |

### 错误示例

```json
{
  "code": 400,
  "message": "参数验证失败: TopK不能超过20",
  "details": "TopK=25 超过了最大限制20"
}
```

## 监控指标

### 关键指标

1. **执行次数**
   - 指标名: `genkit_flow_executions_total{flow_name="memorySearchFlow"}`
   - 标签: status (success/error)

2. **执行时间**
   - 指标名: `genkit_flow_duration_seconds{flow_name="memorySearchFlow"}`
   - 类型: Histogram

3. **向量检索性能**
   - 平均检索时间
   - P50/P95/P99延迟

4. **缓存命中率**
   - 向量缓存命中率
   - 查询结果缓存命中率

### 告警规则

```yaml
- alert: MemorySearchFlowHighLatency
  expr: histogram_quantile(0.95, genkit_flow_duration_seconds{flow_name="memorySearchFlow"}) > 0.3
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "记忆搜索Flow延迟过高"
    description: "P95延迟超过300ms"

- alert: MemorySearchFlowHighErrorRate
  expr: rate(genkit_flow_executions_total{flow_name="memorySearchFlow",status="error"}[5m]) > 0.1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "记忆搜索Flow错误率过高"
    description: "错误率超过10%"
```

## 安全考虑

### 多租户隔离

1. **数据隔离**
   - 所有查询自动包含租户ID过滤
   - 跨会话检索限制在同一租户内
   - 使用参数化查询防止SQL注入

2. **权限验证**
   - 验证JWT Token有效性
   - 检查用户所属租户
   - 记录跨租户访问尝试

3. **审计日志**
   - 记录所有检索请求
   - 记录权限验证失败
   - 包含用户ID、租户ID、会话ID

### 输入验证

- 严格的参数类型检查
- 长度和范围限制
- UUID格式验证
- 防止注入攻击

## 最佳实践

### 1. 选择合适的TopK

- 一般场景: TopK = 5
- 需要更多上下文: TopK = 10
- 避免过大的TopK影响性能

### 2. 设置合理的相似度阈值

- 高精度场景: minSimilarity = 0.8
- 平衡场景: minSimilarity = 0.7
- 召回优先: minSimilarity = 0.6

### 3. 使用过滤条件

- 优先使用记忆类型过滤
- 合理设置时间范围
- 避免过度过滤导致结果为空

### 4. 跨会话检索的使用

- 仅在需要时启用
- 注意性能影响
- 考虑租户数据量

## 集成示例

### Go代码调用

```go
import (
    "context"
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit/flows"
)

func searchMemories(ctx context.Context, g *genkit.Genkit) error {
    // 查找Flow
    flow := genkit.LookupFlow[flows.MemorySearchInput, flows.MemorySearchOutput](
        g,
        "memorySearchFlow",
    )
    
    // 准备输入
    input := flows.MemorySearchInput{
        SessionID:     "550e8400-e29b-41d4-a716-446655440000",
        Query:         "如何优化数据库查询？",
        TopK:          5,
        MinSimilarity: 0.7,
    }
    
    // 执行Flow
    output, err := flow.Run(ctx, input)
    if err != nil {
        return err
    }
    
    // 处理结果
    for _, memory := range output.Memories {
        fmt.Printf("记忆: %s (相似度: %.2f)\n", memory.Content, memory.Similarity)
    }
    
    return nil
}
```

### HTTP API调用

```bash
curl -X POST http://localhost:8080/api/v1/memory/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "query": "如何优化数据库查询？",
    "topK": 5,
    "minSimilarity": 0.7
  }'
```

## 相关Flow

- `contextBuildFlow`: 构建对话上下文（使用记忆搜索结果）
- `memoryStoreFlow`: 存储新的记忆
- `memoryCleanupFlow`: 清理过期记忆

## 更新日志

### v1.0.0 (2025-11-01)

- ✅ 初始实现
- ✅ 支持单会话和跨会话检索
- ✅ 实现多维度过滤
- ✅ 添加异步访问统计更新
- ✅ 完整的单元测试覆盖

## 参考资料

- [需求文档 - 需求12](../../.kiro/specs/genkit-session-management/requirements.md#需求-12长期记忆检索-flow)
- [设计文档 - memorySearchFlow](../../.kiro/specs/genkit-session-management/design.md#3-memorysearchflow)
- [pgvector文档](https://github.com/pgvector/pgvector)
- [余弦相似度](https://en.wikipedia.org/wiki/Cosine_similarity)
