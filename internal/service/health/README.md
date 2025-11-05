# 健康检查服务

## 概述

健康检查服务提供系统健康状态监控功能，用于检查服务及其依赖项的运行状态。

## 功能特性

- 检查 Genkit AI 服务连接状态
- 检查 PostgreSQL 数据库连接状态
- 收集服务版本信息
- 计算服务运行时间
- 返回整体健康状态

## 使用方法

### 创建健康检查服务

```go
import (
    "genkit-ai-service/internal/service/health"
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/database"
)

// 初始化依赖
genkitClient := genkit.NewClient()
db := database.NewPostgresDatabase(dbConfig)

// 创建健康检查服务
healthService := health.NewService(genkitClient, db, "1.0.0")
```

### 执行健康检查

```go
ctx := context.Background()
status, err := healthService.Check(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("状态: %s\n", status.Status)
fmt.Printf("版本: %s\n", status.Version)
fmt.Printf("运行时间: %s\n", status.Uptime)
fmt.Printf("依赖状态: %+v\n", status.Dependencies)
```

## 健康状态响应

### HealthStatus 结构

```go
type HealthStatus struct {
    Status       string            // 整体状态：healthy, unhealthy
    Version      string            // 服务版本
    Uptime       string            // 运行时间（格式：1h30m45s）
    Dependencies map[string]string // 依赖服务状态
}
```

### 依赖状态值

- `connected`: 依赖服务连接正常
- `disconnected`: 依赖服务连接失败
- `not_configured`: 依赖服务未配置

### 整体状态判断

- `healthy`: 所有依赖服务状态为 `connected`
- `unhealthy`: 至少有一个依赖服务状态不是 `connected`

## API 接口

健康检查接口通过 HTTP 端点暴露：

### 端点

```
GET /health
```

### 成功响应（200 OK）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime": "2h30m15s",
    "dependencies": {
      "genkit": "connected",
      "database": "connected"
    }
  }
}
```

### 不健康响应（503 Service Unavailable）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "status": "unhealthy",
    "version": "1.0.0",
    "uptime": "2h30m15s",
    "dependencies": {
      "genkit": "disconnected",
      "database": "connected"
    }
  }
}
```

### 错误响应（500 Internal Server Error）

```json
{
  "code": 500,
  "message": "健康检查失败",
  "data": null
}
```

## 监控集成

健康检查接口可以与以下监控系统集成：

- **Kubernetes**: 用作 liveness 和 readiness 探针
- **负载均衡器**: 用于后端健康检查
- **监控系统**: Prometheus、Datadog 等
- **服务网格**: Istio、Linkerd 等

### Kubernetes 配置示例

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## 超时配置

健康检查对每个依赖项使用 5 秒超时，避免长时间等待：

- Genkit 连接检查：5 秒超时
- 数据库连接检查：5 秒超时

## 测试

运行健康检查服务测试：

```bash
go test ./internal/service/health/... -v
```

运行健康检查处理器测试：

```bash
go test ./internal/api/handler/... -run TestHealth -v
```

## 注意事项

1. **性能影响**: 健康检查会执行实际的连接测试，频繁调用可能影响性能
2. **超时设置**: 确保超时时间合理，避免阻塞过久
3. **依赖可选**: 如果某些依赖不是必需的，可以传入 nil
4. **版本管理**: 建议通过构建时注入版本号
