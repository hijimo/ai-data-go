# 健康检查服务

## 概述

健康检查服务提供系统健康状态监控功能，用于检查服务及其依赖项的运行状态。

## 功能特性

- ✅ 检查 PostgreSQL 数据库连接状态和响应时间
- ✅ 检查 Redis 缓存连接状态和响应时间
- ✅ 检查 AI 服务（Genkit）可用性和响应时间
- ✅ 检查系统资源使用情况（CPU、内存、Goroutines）
- ✅ 收集服务版本信息和运行时间
- ✅ 返回详细的健康状态（healthy、degraded、unhealthy）
- ✅ 提供各项检查的响应时间和详细信息

## 使用方法

### 创建健康检查服务

```go
import (
    "genkit-ai-service/internal/service/health"
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/database"
    "github.com/redis/go-redis/v9"
)

// 初始化依赖
genkitClient := genkit.NewClient()
db := database.NewPostgresDatabase(dbConfig)
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// 创建健康检查服务
// Redis 是可选的，如果不需要可以传 nil
healthService := health.NewService(genkitClient, db, redisClient, "1.0.0")
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
fmt.Printf("时间戳: %s\n", status.Timestamp)

// 打印各项检查结果
for name, check := range status.Checks {
    fmt.Printf("%s: %s (%dms) - %s\n", 
        name, check.Status, check.ResponseTime, check.Message)
}

// 打印系统信息
if status.SystemInfo != nil {
    fmt.Printf("CPU核心数: %d\n", status.SystemInfo.CPUCores)
    fmt.Printf("Goroutines: %d\n", status.SystemInfo.Goroutines)
    fmt.Printf("内存使用率: %.2f%%\n", status.SystemInfo.MemoryUsage)
}
```

## 健康状态响应

### HealthStatus 结构

```go
type HealthStatus struct {
    Status       string                 // 整体状态：healthy, degraded, unhealthy
    Version      string                 // 服务版本
    Uptime       string                 // 运行时间（格式：1h30m45s）
    Timestamp    string                 // 检查时间戳（RFC3339格式）
    Checks       map[string]CheckResult // 各项检查结果
    SystemInfo   *SystemInfo            // 系统资源信息
}

type CheckResult struct {
    Status       string                 // 状态：healthy, degraded, unhealthy
    Message      string                 // 状态消息
    ResponseTime int64                  // 响应时间（毫秒）
    Details      map[string]interface{} // 详细信息
}

type SystemInfo struct {
    CPUCores    int     // CPU核心数
    Goroutines  int     // Goroutine数量
    MemoryAlloc uint64  // 已分配内存（字节）
    MemoryTotal uint64  // 总内存（字节）
    MemoryUsage float64 // 内存使用率（百分比）
}
```

### 检查项状态值

- `healthy`: 服务正常运行
- `degraded`: 服务可用但性能下降（如响应缓慢）
- `unhealthy`: 服务不可用或连接失败

### 整体状态判断规则

1. 如果数据库（核心服务）不健康 → `unhealthy`
2. 如果任何服务不健康 → `unhealthy`
3. 如果有服务降级 → `degraded`
4. 所有服务正常 → `healthy`

### 检查项说明

- **database**: PostgreSQL数据库连接检查
  - 响应时间 > 1000ms → degraded
  - 连接失败 → unhealthy
  
- **redis**: Redis缓存连接检查（可选）
  - 未配置 → degraded（可选服务）
  - 响应时间 > 500ms → degraded
  - 连接失败 → degraded
  
- **ai_service**: AI服务可用性检查
  - 响应时间 > 5000ms → degraded
  - 超时或连接失败 → unhealthy
  
- **system_resources**: 系统资源检查
  - 内存使用率 > 85% → degraded
  - Goroutine数量 > 10000 → degraded

## API 接口

健康检查接口通过 HTTP 端点暴露：

### 端点

```
GET /health
```

### 健康响应（200 OK）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime": "2h30m15s",
    "timestamp": "2025-11-02T10:30:00Z",
    "checks": {
      "database": {
        "status": "healthy",
        "message": "数据库连接正常",
        "responseTime": 15,
        "details": {
          "type": "PostgreSQL"
        }
      },
      "redis": {
        "status": "healthy",
        "message": "Redis连接正常",
        "responseTime": 5
      },
      "ai_service": {
        "status": "healthy",
        "message": "AI服务正常",
        "responseTime": 1250,
        "details": {
          "provider": "Google Genkit"
        }
      },
      "system_resources": {
        "status": "healthy",
        "message": "系统资源正常",
        "details": {
          "cpuCores": 8,
          "goroutines": 42,
          "memoryAlloc": 52428800,
          "memoryTotal": 104857600,
          "memoryUsage": "50.00%"
        }
      }
    },
    "systemInfo": {
      "cpuCores": 8,
      "goroutines": 42,
      "memoryAlloc": 52428800,
      "memoryTotal": 104857600,
      "memoryUsage": 50.0
    }
  }
}
```

### 降级响应（200 OK）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "status": "degraded",
    "version": "1.0.0",
    "uptime": "2h30m15s",
    "timestamp": "2025-11-02T10:30:00Z",
    "checks": {
      "database": {
        "status": "healthy",
        "message": "数据库连接正常",
        "responseTime": 15
      },
      "redis": {
        "status": "degraded",
        "message": "Redis未配置（可选服务）"
      },
      "ai_service": {
        "status": "degraded",
        "message": "AI服务响应缓慢",
        "responseTime": 6500
      },
      "system_resources": {
        "status": "healthy",
        "message": "系统资源正常"
      }
    },
    "systemInfo": {
      "cpuCores": 8,
      "goroutines": 42,
      "memoryAlloc": 52428800,
      "memoryTotal": 104857600,
      "memoryUsage": 50.0
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
    "timestamp": "2025-11-02T10:30:00Z",
    "checks": {
      "database": {
        "status": "unhealthy",
        "message": "数据库连接失败: connection refused",
        "responseTime": 5000
      },
      "redis": {
        "status": "degraded",
        "message": "Redis连接失败: connection refused",
        "responseTime": 3000
      },
      "ai_service": {
        "status": "unhealthy",
        "message": "AI服务响应超时",
        "responseTime": 10000
      },
      "system_resources": {
        "status": "healthy",
        "message": "系统资源正常"
      }
    },
    "systemInfo": {
      "cpuCores": 8,
      "goroutines": 42,
      "memoryAlloc": 52428800,
      "memoryTotal": 104857600,
      "memoryUsage": 50.0
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

健康检查对每个依赖项使用不同的超时时间，避免长时间等待：

- 数据库连接检查：5秒超时
- Redis连接检查：3秒超时
- AI服务检查：10秒超时（AI生成需要更长时间）
- 系统资源检查：无超时（本地操作）

## 性能阈值

各项检查的性能阈值：

| 检查项 | 正常阈值 | 降级阈值 | 说明 |
|--------|---------|---------|------|
| 数据库响应时间 | < 1000ms | ≥ 1000ms | 超过1秒视为响应缓慢 |
| Redis响应时间 | < 500ms | ≥ 500ms | 超过500ms视为响应缓慢 |
| AI服务响应时间 | < 5000ms | ≥ 5000ms | 超过5秒视为响应缓慢 |
| 内存使用率 | < 85% | ≥ 85% | 超过85%视为使用率较高 |
| Goroutine数量 | < 10000 | ≥ 10000 | 超过10000视为数量过多 |

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
   - 建议设置合理的检查间隔（如10-30秒）
   - 避免在高负载时频繁调用

2. **超时设置**: 确保超时时间合理，避免阻塞过久
   - 数据库：5秒
   - Redis：3秒
   - AI服务：10秒

3. **依赖可选**:
   - Redis是可选服务，未配置时会返回degraded状态
   - 数据库和AI服务是必需的，失败会导致unhealthy状态

4. **版本管理**: 建议通过构建时注入版本号

   ```bash
   go build -ldflags "-X main.Version=1.0.0" ./cmd/server
   ```

5. **系统资源监控**:
   - 内存使用率基于Go运行时统计
   - Goroutine数量反映并发处理能力
   - CPU核心数用于评估系统容量

6. **降级策略**:
   - Redis失败不会导致整体unhealthy（可选服务）
   - 数据库失败会立即导致unhealthy（核心服务）
   - AI服务失败会导致unhealthy（核心功能）

7. **监控建议**:
   - 设置告警规则监控unhealthy状态
   - 关注degraded状态的持续时间
   - 监控响应时间趋势
   - 定期检查系统资源使用情况
