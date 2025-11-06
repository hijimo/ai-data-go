# Qdrant 配置更新说明

## 更新内容

已更新 Qdrant 客户端以支持 Qdrant Cloud 的配置方式，同时保持对自托管 Qdrant 的兼容性。

## 配置变更

### 新的 QdrantConfig 结构

```go
type QdrantConfig struct {
    // 方式1：使用完整的 Endpoint URL（推荐用于 Qdrant Cloud）
    Endpoint string // 完整的 Qdrant 端点 URL
    
    // 方式2：使用 Host + Port（用于自托管）
    Host   string // Qdrant 服务器地址
    Port   int    // Qdrant 服务器端口
    UseTLS bool   // 是否使用 TLS（仅用于方式2）
    
    // 通用配置
    APIKey    string // API Key / Access Token（必需）
    ClusterID string // 集群ID（可选，用于日志记录）
}
```

### 环境变量配置

#### Qdrant Cloud（当前使用）

您的 `.env` 文件中已配置：

```bash
QDRANT_ENDPOINT=https://37612f1c-dafd-48ab-afe7-7852d81a0868.us-west-2-0.aws.cloud.qdrant.io
QDRANT_ACCESS_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
QDRANT_CLUSTER_ID=37612f1c-dafd-48ab-afe7-7852d81a0868
```

#### 自托管 Qdrant（备选）

如果使用自托管 Qdrant，可以配置：

```bash
QDRANT_HOST=localhost
QDRANT_PORT=6333
QDRANT_API_KEY=your-api-key
QDRANT_USE_TLS=false
```

## 使用方法

### 方式1：使用辅助函数（推荐）

```go
import "genkit-ai-service/internal/storage"

// 从环境变量自动加载配置
config, err := storage.LoadQdrantConfigFromEnv()
if err != nil {
    log.Fatal("加载 Qdrant 配置失败:", err)
}

// 创建客户端
client, err := storage.NewQdrantClient(config)
if err != nil {
    log.Fatal("创建 Qdrant 客户端失败:", err)
}
defer client.Close()

// 初始化 Collection
ctx := context.Background()
if err := client.InitializeCollection(ctx); err != nil {
    log.Fatal("初始化 Collection 失败:", err)
}
```

### 方式2：手动配置

#### Qdrant Cloud

```go
config := &storage.QdrantConfig{
    Endpoint:  os.Getenv("QDRANT_ENDPOINT"),
    APIKey:    os.Getenv("QDRANT_ACCESS_KEY"),
    ClusterID: os.Getenv("QDRANT_CLUSTER_ID"),
}

client, err := storage.NewQdrantClient(config)
```

#### 自托管

```go
config := &storage.QdrantConfig{
    Host:   "localhost",
    Port:   6333,
    APIKey: "your-api-key",
    UseTLS: false,
}

client, err := storage.NewQdrantClient(config)
```

## 配置优先级

客户端会按以下优先级选择配置：

1. **Endpoint**（如果设置）→ 使用 Qdrant Cloud 模式
2. **Host + Port**（如果 Endpoint 未设置）→ 使用自托管模式
3. 如果两者都未设置 → 返回错误

## API Key 处理

- **必需**：所有配置都必须提供 `APIKey`
- **Qdrant Cloud**：使用 `QDRANT_ACCESS_KEY` 环境变量（JWT token）
- **自托管**：使用 `QDRANT_API_KEY` 环境变量（如果启用了认证）

## HTTP 请求头

客户端会自动在所有请求中添加以下头部：

```http
Content-Type: application/json
api-key: <your-api-key>
```

## 测试验证

所有测试已更新并通过：

```bash
$ go test ./internal/storage/... -run TestNewQdrantClient -v
=== RUN   TestNewQdrantClient
=== RUN   TestNewQdrantClient/nil_config
=== RUN   TestNewQdrantClient/missing_API_key
=== RUN   TestNewQdrantClient/missing_endpoint_and_host
=== RUN   TestNewQdrantClient/valid_config_with_endpoint_(Qdrant_Cloud)
=== RUN   TestNewQdrantClient/valid_config_with_host_(self-hosted)
=== RUN   TestNewQdrantClient/valid_config_with_default_port
--- PASS: TestNewQdrantClient (0.00s)
PASS
```

## 迁移指南

### 从旧配置迁移

如果您之前使用的是：

```go
// 旧配置
config := &storage.QdrantConfig{
    Host: "localhost",
    Port: 6333,
}
```

需要更新为：

```go
// 新配置（必须提供 APIKey）
config := &storage.QdrantConfig{
    Host:   "localhost",
    Port:   6333,
    APIKey: "your-api-key",
}
```

### 使用 Qdrant Cloud

如果要使用 Qdrant Cloud（推荐）：

```go
// 新配置
config := &storage.QdrantConfig{
    Endpoint:  "https://xxx.cloud.qdrant.io",
    APIKey:    "your-access-token",
    ClusterID: "xxx", // 可选
}
```

## 相关文件

- `internal/storage/qdrant_client.go` - 接口定义和配置结构
- `internal/storage/qdrant_client_impl.go` - 客户端实现
- `internal/storage/config_example.go` - 配置加载辅助函数
- `internal/storage/QDRANT_README.md` - 完整使用文档
- `.env` - 环境变量配置

## 注意事项

1. **API Key 安全**：不要将 API Key 提交到版本控制系统
2. **环境变量**：确保在生产环境中正确设置环境变量
3. **连接测试**：建议在应用启动时测试 Qdrant 连接
4. **错误处理**：正确处理配置加载和客户端创建的错误

## 下一步

配置已更新完成，可以：

1. ✅ 使用 Qdrant Cloud 配置
2. ✅ 初始化 Collection
3. ✅ 存储和检索向量
4. ⏭️ 集成到记忆服务中（任务8）

## 更新日志

### 2024-01-XX

- ✅ 添加 Qdrant Cloud 支持（Endpoint 配置）
- ✅ 保持自托管 Qdrant 兼容性
- ✅ 强制要求 API Key
- ✅ 添加 ClusterID 字段（可选）
- ✅ 更新测试用例
- ✅ 添加配置加载辅助函数
- ✅ 更新文档
