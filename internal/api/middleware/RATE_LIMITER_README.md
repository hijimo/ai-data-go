# 速率限制中间件

## 概述

速率限制中间件使用令牌桶算法实现了基于 IP 和租户的请求速率限制，用于防止 API 滥用和保护系统资源。

## 特性

- **令牌桶算法**：平滑的速率限制，允许突发流量
- **双重限制**：支持基于 IP 和租户的独立限制
- **内存存储**：使用内存存储限制状态，性能高效
- **自动清理**：定期清理过期的限制记录
- **灵活配置**：支持自定义容量和补充率
- **重置功能**：支持手动重置限制（用于测试或管理）

## 令牌桶算法

令牌桶算法的工作原理：

1. **桶容量**：桶可以容纳的最大令牌数
2. **补充率**：每秒向桶中添加的令牌数
3. **请求处理**：
   - 每个请求消耗一个令牌
   - 如果桶中有令牌，请求通过，令牌减一
   - 如果桶中没有令牌，请求被拒绝
4. **令牌补充**：按照补充率定期向桶中添加令牌，直到达到容量上限

### 优势

- **平滑限流**：允许短时间的突发流量
- **公平性**：长期来看，所有客户端都受到相同的限制
- **灵活性**：可以通过调整容量和补充率来适应不同的场景

## 使用方法

### 基本使用

```go
import (
    "genkit-ai-service/internal/api/middleware"
    "genkit-ai-service/internal/logger"
)

// 创建日志记录器
log := logger.New()

// 使用默认配置创建速率限制中间件
rateLimiter := middleware.NewRateLimiterMiddleware(nil, log)

// 在路由中应用速率限制
router.Use(rateLimiter.RateLimit())
```

### 自定义配置

```go
// 创建自定义配置
config := &middleware.RateLimiterConfig{
    // IP 限制：每秒 10 个请求，桶容量 20
    IPCapacity:   20,
    IPRefillRate: 10,
    
    // 租户限制：每秒 50 个请求，桶容量 100
    TenantCapacity:   100,
    TenantRefillRate: 50,
    
    // 启用两种限制
    EnableIPLimit:     true,
    EnableTenantLimit: true,
}

rateLimiter := middleware.NewRateLimiterMiddleware(config, log)
router.Use(rateLimiter.RateLimit())
```

### 仅 IP 限制

```go
config := &middleware.RateLimiterConfig{
    IPCapacity:        10,
    IPRefillRate:      5,
    EnableIPLimit:     true,
    EnableTenantLimit: false,
}

rateLimiter := middleware.NewRateLimiterMiddleware(config, log)
router.Use(rateLimiter.RateLimitByIP())
```

### 仅租户限制

```go
config := &middleware.RateLimiterConfig{
    TenantCapacity:    50,
    TenantRefillRate:  25,
    EnableIPLimit:     false,
    EnableTenantLimit: true,
}

rateLimiter := middleware.NewRateLimiterMiddleware(config, log)
router.Use(rateLimiter.RateLimitByTenant())
```

### 针对特定路由的限制

```go
// 为不同的路由组应用不同的限制
publicAPI := router.Group("/api/v1/public")
{
    // 公开 API 使用更严格的限制
    strictConfig := &middleware.RateLimiterConfig{
        IPCapacity:   5,
        IPRefillRate: 2,
    }
    strictLimiter := middleware.NewRateLimiterMiddleware(strictConfig, log)
    publicAPI.Use(strictLimiter.RateLimitByIP())
    
    publicAPI.GET("/data", handler.GetPublicData)
}

authenticatedAPI := router.Group("/api/v1/private")
{
    // 认证 API 使用较宽松的限制
    normalConfig := &middleware.RateLimiterConfig{
        IPCapacity:   20,
        IPRefillRate: 10,
    }
    normalLimiter := middleware.NewRateLimiterMiddleware(normalConfig, log)
    authenticatedAPI.Use(middleware.JWTAuth())
    authenticatedAPI.Use(normalLimiter.RateLimit())
    
    authenticatedAPI.GET("/data", handler.GetPrivateData)
}
```

## 配置参数

### IPCapacity

- **类型**：int
- **说明**：IP 令牌桶的容量（最大令牌数）
- **默认值**：20
- **建议值**：
  - 公开 API：5-10
  - 认证 API：20-50
  - 内部 API：50-100

### IPRefillRate

- **类型**：int
- **说明**：IP 令牌桶每秒补充的令牌数
- **默认值**：10
- **建议值**：
  - 公开 API：2-5
  - 认证 API：10-20
  - 内部 API：20-50

### TenantCapacity

- **类型**：int
- **说明**：租户令牌桶的容量（最大令牌数）
- **默认值**：100
- **建议值**：
  - 小型租户：50-100
  - 中型租户：100-200
  - 大型租户：200-500

### TenantRefillRate

- **类型**：int
- **说明**：租户令牌桶每秒补充的令牌数
- **默认值**：50
- **建议值**：
  - 小型租户：25-50
  - 中型租户：50-100
  - 大型租户：100-200

### EnableIPLimit

- **类型**：bool
- **说明**：是否启用基于 IP 的速率限制
- **默认值**：true

### EnableTenantLimit

- **类型**：bool
- **说明**：是否启用基于租户的速率限制
- **默认值**：true

## 响应格式

当请求被速率限制时，返回以下响应：

### HTTP 状态码

```
429 Too Many Requests
```

### 响应头

```
Retry-After: 60
X-RateLimit-Limit: 20
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699999999
```

### 响应体

```json
{
  "code": 429,
  "message": "请求过于频繁，请稍后再试",
  "traceId": "abc123..."
}
```

## 管理功能

### 重置 IP 限制

```go
// 重置特定 IP 的限制（用于测试或管理）
rateLimiter.ResetIPLimit("192.168.1.1")
```

### 重置租户限制

```go
// 重置特定租户的限制（用于测试或管理）
rateLimiter.ResetTenantLimit("tenant-uuid")
```

## 监控和日志

速率限制中间件会记录以下事件：

### IP 限制触发

```
级别: WARN
消息: IP速率限制触发
字段:
  - ip: 客户端 IP 地址
  - path: 请求路径
  - method: HTTP 方法
```

### 租户限制触发

```
级别: WARN
消息: 租户速率限制触发
字段:
  - tenantId: 租户 ID
  - path: 请求路径
  - method: HTTP 方法
```

## 性能考虑

### 内存使用

- 每个唯一的 IP 或租户会创建一个令牌桶实例
- 每个令牌桶占用约 100 字节内存
- 自动清理机制会定期删除 10 分钟未使用的令牌桶

### 并发性能

- 使用读写锁保护共享状态
- 令牌桶操作的时间复杂度为 O(1)
- 支持高并发场景

### 清理机制

- 每 5 分钟执行一次清理
- 删除超过 10 分钟未使用的令牌桶
- 清理过程不会阻塞请求处理

## 最佳实践

### 1. 分层限制

为不同类型的 API 应用不同的限制：

```go
// 公开 API - 严格限制
publicLimiter := middleware.NewRateLimiterMiddleware(&middleware.RateLimiterConfig{
    IPCapacity:   5,
    IPRefillRate: 2,
}, log)

// 认证 API - 正常限制
authLimiter := middleware.NewRateLimiterMiddleware(&middleware.RateLimiterConfig{
    IPCapacity:   20,
    IPRefillRate: 10,
}, log)

// 内部 API - 宽松限制
internalLimiter := middleware.NewRateLimiterMiddleware(&middleware.RateLimiterConfig{
    IPCapacity:   100,
    IPRefillRate: 50,
}, log)
```

### 2. 组合使用

同时使用 IP 和租户限制以提供多层保护：

```go
config := &middleware.RateLimiterConfig{
    IPCapacity:        20,
    IPRefillRate:      10,
    TenantCapacity:    100,
    TenantRefillRate:  50,
    EnableIPLimit:     true,
    EnableTenantLimit: true,
}
```

### 3. 监控和告警

监控速率限制触发的频率，及时发现异常流量：

```go
// 在日志中记录速率限制事件
// 使用监控系统（如 Prometheus）收集指标
// 设置告警规则，当限制触发频率过高时发送通知
```

### 4. 动态调整

根据系统负载和业务需求动态调整限制参数：

```go
// 在高峰期降低限制
// 在低峰期提高限制
// 为 VIP 租户提供更高的限制
```

## 测试

### 单元测试

```go
func TestRateLimiter(t *testing.T) {
    limiter := middleware.NewInMemoryRateLimiter(3, 1)
    
    // 前 3 个请求应该通过
    assert.True(t, limiter.Allow("key1"))
    assert.True(t, limiter.Allow("key1"))
    assert.True(t, limiter.Allow("key1"))
    
    // 第 4 个请求应该被拒绝
    assert.False(t, limiter.Allow("key1"))
}
```

### 集成测试

```go
func TestRateLimiterMiddleware(t *testing.T) {
    router := gin.New()
    router.Use(rateLimiter.RateLimit())
    router.GET("/test", handler)
    
    // 发送多个请求测试限制
    for i := 0; i < 5; i++ {
        req := httptest.NewRequest("GET", "/test", nil)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        if i < 3 {
            assert.Equal(t, http.StatusOK, w.Code)
        } else {
            assert.Equal(t, http.StatusTooManyRequests, w.Code)
        }
    }
}
```

## 故障排查

### 问题：所有请求都被限制

**可能原因**：

- 容量设置过小
- 补充率设置过低
- 多个实例共享相同的 IP

**解决方案**：

- 增加容量和补充率
- 检查负载均衡器配置
- 使用 X-Forwarded-For 头获取真实 IP

### 问题：限制不生效

**可能原因**：

- 中间件未正确应用
- 配置中禁用了限制
- 租户 ID 未正确设置到上下文

**解决方案**：

- 检查中间件顺序
- 验证配置参数
- 确保 JWT 中间件在速率限制之前

### 问题：内存使用过高

**可能原因**：

- 大量唯一 IP 或租户
- 清理机制未正常工作

**解决方案**：

- 检查清理协程是否运行
- 考虑使用 Redis 等外部存储
- 调整清理间隔和过期时间

## 未来改进

- [ ] 支持 Redis 作为存储后端（分布式部署）
- [ ] 支持滑动窗口算法
- [ ] 支持动态配置更新
- [ ] 支持白名单和黑名单
- [ ] 集成 Prometheus 指标
- [ ] 支持基于用户的限制
- [ ] 支持基于 API 端点的限制
- [ ] 支持限制策略的热更新

## 参考资料

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [Rate Limiting Strategies](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)
- [OWASP API Security - Rate Limiting](https://owasp.org/www-project-api-security/)
