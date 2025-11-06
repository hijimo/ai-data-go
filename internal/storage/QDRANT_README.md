# Qdrant 向量数据库客户端

## 概述

本模块实现了基于 Qdrant 向量数据库的多租户向量存储和检索功能。采用**单个共享 Collection + Payload 分区**的多租户架构，通过租户ID过滤实现数据隔离。

## 架构设计

### 多租户架构

- **单个共享 Collection**: `conversation_memories`
- **租户隔离**: 通过 Payload 中的 `tenant_id` 字段实现
- **租户索引**: `tenant_id` 字段设置 `is_tenant=true` 索引，优化查询性能
- **自定义分片**: 按租户分布数据，提高查询效率

### Collection 配置

```json
{
  "name": "conversation_memories",
  "vectors": {
    "size": 1536,
    "distance": "Cosine"
  },
  "shard_number": 4,
  "replication_factor": 2,
  "hnsw_config": {
    "m": 16,
    "ef_construction": 100
  }
}
```

### Payload 结构

每个向量点的 payload 包含：

```json
{
  "memory_id": "uuid",
  "tenant_id": "uuid",
  "session_id": "uuid",
  "memory_type": "short_term|long_term|summary",
  "importance": 0.8,
  "created_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

## 使用方法

### 1. 创建客户端

```go
import "genkit-ai-service/internal/storage"

// 创建配置
config := &storage.QdrantConfig{
    Host:   "localhost",
    Port:   6333,
    APIKey: "", // 可选
    UseTLS: false,
}

// 创建客户端
client, err := storage.NewQdrantClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 2. 初始化 Collection

```go
// 在应用启动时调用一次
ctx := context.Background()
if err := client.InitializeCollection(ctx); err != nil {
    log.Fatal(err)
}
```

### 3. 插入向量

```go
import "github.com/google/uuid"

// 准备向量数据
vector := make([]float32, 1536) // text-embedding-ada-002 的向量维度
// ... 填充向量数据

// 创建插入请求
req := &storage.UpsertVectorRequest{
    TenantID:   uuid.MustParse("tenant-uuid"),
    MemoryID:   uuid.New(),
    SessionID:  uuid.MustParse("session-uuid"),
    MemoryType: "long_term",
    Vector:     vector,
    Importance: 0.8,
    Metadata: map[string]interface{}{
        "content": "这是一段对话内容",
        "keywords": []string{"关键词1", "关键词2"},
    },
}

// 插入向量
if err := client.UpsertVector(ctx, req); err != nil {
    log.Printf("插入向量失败: %v", err)
}
```

### 4. 检索向量

```go
// 准备查询向量
queryVector := make([]float32, 1536)
// ... 填充查询向量

// 创建检索请求
searchReq := &storage.SearchVectorRequest{
    TenantID:    uuid.MustParse("tenant-uuid"),
    SessionID:   &sessionID, // 可选，用于会话内搜索
    QueryVector: queryVector,
    TopK:        5,
    MinScore:    0.7,
}

// 执行检索
results, err := client.SearchVectors(ctx, searchReq)
if err != nil {
    log.Printf("检索向量失败: %v", err)
}

// 处理结果
for _, result := range results {
    fmt.Printf("Memory ID: %s, Score: %.2f\n", result.MemoryID, result.Score)
    fmt.Printf("Content: %v\n", result.Payload["content"])
}
```

### 5. 跨会话检索

```go
// 不指定 SessionID，检索租户内所有会话的记忆
searchReq := &storage.SearchVectorRequest{
    TenantID:    uuid.MustParse("tenant-uuid"),
    SessionID:   nil, // 不指定会话ID
    QueryVector: queryVector,
    TopK:        10,
    MinScore:    0.7,
}

results, err := client.SearchVectors(ctx, searchReq)
```

### 6. 按类型过滤

```go
memoryType := "long_term"
searchReq := &storage.SearchVectorRequest{
    TenantID:    uuid.MustParse("tenant-uuid"),
    QueryVector: queryVector,
    TopK:        5,
    MinScore:    0.7,
    MemoryType:  &memoryType, // 只检索长期记忆
}

results, err := client.SearchVectors(ctx, searchReq)
```

### 7. 时间范围过滤

```go
timeRange := &storage.TimeRange{
    Start: time.Now().AddDate(0, 0, -7), // 最近7天
    End:   time.Now(),
}

searchReq := &storage.SearchVectorRequest{
    TenantID:    uuid.MustParse("tenant-uuid"),
    QueryVector: queryVector,
    TopK:        5,
    MinScore:    0.7,
    TimeRange:   timeRange,
}

results, err := client.SearchVectors(ctx, searchReq)
```

### 8. 删除向量

```go
// 删除单个向量
tenantID := uuid.MustParse("tenant-uuid")
memoryID := uuid.MustParse("memory-uuid")

if err := client.DeleteVector(ctx, tenantID, memoryID); err != nil {
    log.Printf("删除向量失败: %v", err)
}
```

### 9. 批量删除

```go
// 按条件批量删除
filter := map[string]interface{}{
    "memory_type": "short_term",
    "session_id":  "session-uuid",
}

if err := client.DeleteByFilter(ctx, tenantID, filter); err != nil {
    log.Printf("批量删除失败: %v", err)
}
```

### 10. 更新 Payload

```go
// 更新向量的 payload
payload := map[string]interface{}{
    "importance": 0.9,
    "access_count": 10,
}

if err := client.UpdatePayload(ctx, tenantID, memoryID, payload); err != nil {
    log.Printf("更新 payload 失败: %v", err)
}
```

## 安全特性

### 租户隔离

- 所有检索操作**强制**包含 `tenant_id` 过滤条件
- 租户管理员只能访问自己租户的数据
- 平台管理员可以访问所有租户的数据（需在上层服务实现权限控制）

### 数据验证

- 插入时验证必填字段（tenant_id, memory_id, session_id）
- 验证向量维度（必须为1536）
- 防止修改租户ID（UpdatePayload 时自动删除 tenant_id 字段）

## 性能优化

### 索引配置

- `tenant_id`: 设置 `is_tenant=true`，优化租户级别查询
- `session_id`: keyword 索引，优化会话内查询
- `memory_type`: keyword 索引，优化类型过滤
- `created_at`: datetime 索引，优化时间范围查询

### HNSW 参数

- `m`: 16（每个节点的连接数）
- `ef_construction`: 100（构建索引时的搜索深度）
- 这些参数在准确性和性能之间取得平衡

### 分片策略

- 默认 4 个分片，可根据租户数量调整
- 副本因子为 2，提供高可用性

## 配置说明

### 环境变量

```bash
# Qdrant 服务器地址
QDRANT_HOST=localhost

# Qdrant 服务器端口
QDRANT_PORT=6333

# API Key（可选）
QDRANT_API_KEY=your-api-key

# 是否使用 TLS
QDRANT_USE_TLS=false
```

### 配置文件示例

```yaml
qdrant:
  host: localhost
  port: 6333
  api_key: ""
  use_tls: false
```

## 错误处理

### 常见错误

1. **租户ID为空**: 所有操作都必须提供有效的租户ID
2. **向量维度错误**: 向量维度必须为1536
3. **连接失败**: 检查 Qdrant 服务是否运行
4. **权限不足**: 检查 API Key 是否正确

### 错误示例

```go
// 错误：租户ID为空
err := client.UpsertVector(ctx, &storage.UpsertVectorRequest{
    TenantID: uuid.Nil, // 错误！
    // ...
})
// 返回: "租户ID不能为空"

// 错误：向量维度不正确
err := client.UpsertVector(ctx, &storage.UpsertVectorRequest{
    Vector: make([]float32, 100), // 错误！应该是1536
    // ...
})
// 返回: "向量维度必须为 1536，当前为 100"
```

## 测试

### 运行单元测试

```bash
go test -v ./internal/storage
```

### 运行特定测试

```bash
go test -v ./internal/storage -run TestNewQdrantClient
```

## 注意事项

1. **初始化**: 在应用启动时调用 `InitializeCollection` 一次即可
2. **租户隔离**: 始终在检索时提供租户ID，系统会自动添加租户过滤条件
3. **向量维度**: 确保向量维度为1536（text-embedding-ada-002）
4. **性能**: 对于大规模数据，考虑增加分片数量
5. **高可用**: 生产环境建议设置 `replication_factor >= 2`

## 依赖

- Go 1.21+
- Qdrant 1.11+（支持 `is_tenant` 索引）

## 相关文档

- [Qdrant 官方文档](https://qdrant.tech/documentation/)
- [Qdrant 多租户支持](https://qdrant.tech/documentation/guides/multiple-partitions/)
- [HNSW 算法](https://arxiv.org/abs/1603.09320)
