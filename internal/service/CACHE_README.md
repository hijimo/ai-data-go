# 缓存服务使用指南

## 概述

缓存服务提供了一套完整的 Redis 缓存管理功能，包括基本的缓存操作、缓存键管理和缓存预热机制。

## 核心组件

### 1. CacheService

缓存服务接口，提供基本的缓存操作。

#### 主要方法

- `Get(ctx, key, dest)` - 获取缓存
- `Set(ctx, key, value, ttl)` - 设置缓存
- `Delete(ctx, keys...)` - 删除缓存
- `DeletePattern(ctx, pattern)` - 按模式删除缓存
- `Exists(ctx, key)` - 检查缓存是否存在
- `Increment(ctx, key, delta)` - 增加计数器
- `Expire(ctx, key, ttl)` - 设置过期时间
- `TTL(ctx, key)` - 获取剩余生存时间
- `HashQuery(query)` - 对查询字符串进行哈希

#### 使用示例

```go
// 创建缓存服务
cache := service.NewCacheService(redisClient, "genkit")

// 设置缓存
type UserData struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

user := UserData{
    Name:  "张三",
    Email: "zhangsan@example.com",
}

err := cache.Set(ctx, "user:123", user, 5*time.Minute)

// 获取缓存
var cachedUser UserData
err = cache.Get(ctx, "user:123", &cachedUser)

// 删除缓存
err = cache.Delete(ctx, "user:123")

// 按模式删除
err = cache.DeletePattern(ctx, "user:*")
```

### 2. CacheKeys

缓存键管理器，提供标准化的缓存键生成和 TTL 管理。

#### 支持的缓存类型

| 缓存类型 | 方法 | TTL | 说明 |
|---------|------|-----|------|
| 上下文 | `ContextKey(sessionID, queryHash)` | 5分钟 | 会话上下文缓存 |
| 向量检索 | `VectorSearchKey(sessionID, queryHash)` | 30分钟 | 向量检索结果 |
| 摘要 | `SummaryKey(sessionID)` | 1小时 | 会话摘要 |
| 会话列表 | `SessionListKey(userID)` | 10分钟 | 用户会话列表 |
| Token使用 | `TokenUsageKey(sessionID)` | 5分钟 | Token使用统计 |
| 配额 | `QuotaKey(tenantID, type)` | 5分钟 | 租户配额 |
| 上下文配置 | `ContextConfigKey(sessionID)` | 10分钟 | 上下文配置 |
| 消息 | `MessageKey(messageID)` | 30分钟 | 消息缓存 |
| 记忆 | `MemoryKey(memoryID)` | 1小时 | 记忆缓存 |
| AI响应 | `AIResponseKey(sessionID, queryHash)` | 1小时 | AI响应（降级用） |
| 速率限制 | `RateLimitKey(tenantID, endpoint)` | 1分钟 | 速率限制 |
| 熔断器 | `CircuitBreakerKey(service)` | 5分钟 | 熔断器状态 |

#### 使用示例

```go
keys := service.NewCacheKeys()

// 构建上下文缓存键
sessionID := "session-123"
queryHash := cache.HashQuery("用户查询内容")
contextKey := keys.ContextKey(sessionID, queryHash)
contextTTL := keys.ContextTTL()

// 设置缓存
err := cache.Set(ctx, contextKey, contextData, contextTTL)

// 构建摘要缓存键
summaryKey := keys.SummaryKey(sessionID)
summaryTTL := keys.SummaryTTL()

// 设置摘要缓存
err = cache.Set(ctx, summaryKey, summaryData, summaryTTL)
```

### 3. CacheWarmer

缓存预热器，提供缓存预热和失效管理功能。

#### 主要方法

- `WarmupOnStartup(ctx)` - 启动时预热
- `StartPeriodicWarmup(ctx, interval)` - 启动定期预热
- `InvalidateSession(ctx, sessionID)` - 使会话缓存失效
- `InvalidateUser(ctx, userID)` - 使用户缓存失效
- `InvalidateTenant(ctx, tenantID)` - 使租户缓存失效
- `RefreshSessionContext(ctx, sessionID)` - 刷新会话上下文缓存
- `GetCacheStats(ctx)` - 获取缓存统计信息

#### 使用示例

```go
// 创建缓存预热器
warmer := service.NewCacheWarmer(cache, keys, logger)

// 启动时预热
err := warmer.WarmupOnStartup(ctx)

// 启动定期预热（在 goroutine 中运行）
go warmer.StartPeriodicWarmup(ctx, 30*time.Minute)

// 使会话缓存失效
err = warmer.InvalidateSession(ctx, "session-123")

// 使用户缓存失效
err = warmer.InvalidateUser(ctx, "user-456")

// 刷新会话上下文缓存
err = warmer.RefreshSessionContext(ctx, "session-123")
```

## 缓存策略

### 缓存键命名规范

所有缓存键都遵循以下格式：

```
{namespace}:{type}:{id}:{subtype}
```

例如：

- `genkit:context:session-123:abc123` - 上下文缓存
- `genkit:summary:session-123:latest` - 摘要缓存
- `genkit:quota:tenant-456:daily` - 配额缓存

### TTL 策略

不同类型的缓存有不同的 TTL：

- **短期缓存**（1-5分钟）：频繁变化的数据，如 Token 使用统计、速率限制
- **中期缓存**（10-30分钟）：相对稳定的数据，如会话列表、向量检索结果
- **长期缓存**（1小时）：较少变化的数据，如摘要、记忆、AI 响应

### 缓存失效策略

1. **主动失效**：数据更新时主动删除相关缓存
2. **被动失效**：依赖 TTL 自动过期
3. **模式失效**：使用 `DeletePattern` 批量删除相关缓存

## 最佳实践

### 1. 使用 CacheKeys 管理缓存键

```go
// ✅ 推荐：使用 CacheKeys
keys := service.NewCacheKeys()
key := keys.ContextKey(sessionID, queryHash)
ttl := keys.ContextTTL()

// ❌ 不推荐：硬编码缓存键
key := fmt.Sprintf("context:%s:%s", sessionID, queryHash)
ttl := 5 * time.Minute
```

### 2. 处理缓存未命中

```go
// 尝试从缓存获取
var data MyData
err := cache.Get(ctx, key, &data)
if err == service.ErrCacheNotFound {
    // 缓存未命中，从数据库加载
    data, err = loadFromDatabase(ctx)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    _ = cache.Set(ctx, key, data, ttl)
}
```

### 3. 异步缓存更新

```go
// 同步返回数据
result := processData(data)

// 异步更新缓存
go func() {
    ctx := context.Background()
    _ = cache.Set(ctx, key, result, ttl)
}()

return result
```

### 4. 缓存预热

```go
// 在应用启动时预热关键数据
func (s *MyService) Initialize(ctx context.Context) error {
    // 预热活跃会话
    sessions, _ := s.getActiveSessions(ctx)
    for _, session := range sessions {
        key := s.keys.ContextConfigKey(session.ID)
        config, _ := s.loadContextConfig(ctx, session.ID)
        _ = s.cache.Set(ctx, key, config, s.keys.ContextConfigTTL())
    }
    
    return nil
}
```

### 5. 批量删除缓存

```go
// 使用模式删除会话相关的所有缓存
pattern := keys.SessionPattern(sessionID)
err := cache.DeletePattern(ctx, pattern)
```

### 6. Redis 未启用时的处理

缓存服务在 Redis 未启用时会优雅降级：

```go
// 创建缓存服务（Redis 客户端可以为 nil）
cache := service.NewCacheService(nil, "genkit")

// Get 返回 ErrCacheNotFound
err := cache.Get(ctx, key, &data)
// err == service.ErrCacheNotFound

// Set 和 Delete 静默失败（不返回错误）
_ = cache.Set(ctx, key, data, ttl)
_ = cache.Delete(ctx, key)
```

## 监控和调试

### 检查缓存是否存在

```go
exists, err := cache.Exists(ctx, key)
if err != nil {
    log.Printf("检查缓存失败: %v", err)
}
if exists {
    log.Printf("缓存存在: %s", key)
}
```

### 检查缓存 TTL

```go
ttl, err := cache.TTL(ctx, key)
if err != nil {
    log.Printf("获取 TTL 失败: %v", err)
}
log.Printf("缓存剩余时间: %v", ttl)
```

### 获取缓存统计

```go
stats, err := warmer.GetCacheStats(ctx)
if err != nil {
    log.Printf("获取统计失败: %v", err)
}
log.Printf("缓存统计: %+v", stats)
```

## 性能优化

### 1. 批量操作

```go
// 批量删除
keys := []string{"key1", "key2", "key3"}
err := cache.Delete(ctx, keys...)
```

### 2. 使用哈希减少键长度

```go
// 对长查询进行哈希
query := "这是一个很长的用户查询..."
queryHash := cache.HashQuery(query)
key := keys.ContextKey(sessionID, queryHash)
```

### 3. 合理设置 TTL

- 频繁变化的数据使用较短的 TTL
- 稳定的数据使用较长的 TTL
- 避免设置过长的 TTL 导致内存浪费

### 4. 避免缓存穿透

```go
// 缓存空结果
if data == nil {
    // 缓存一个特殊值表示数据不存在
    _ = cache.Set(ctx, key, "NULL", 1*time.Minute)
    return nil
}
```

## 故障处理

### Redis 连接失败

缓存服务会优雅降级，不会影响主要业务逻辑：

```go
// 即使 Redis 不可用，应用仍然可以正常运行
// 只是没有缓存加速
err := cache.Set(ctx, key, data, ttl)
// err 可能不为 nil，但不会导致应用崩溃
```

### 缓存数据损坏

```go
var data MyData
err := cache.Get(ctx, key, &data)
if err != nil {
    // 缓存获取失败或数据损坏
    // 从数据库重新加载
    data, err = loadFromDatabase(ctx)
    if err != nil {
        return nil, err
    }
    
    // 更新缓存
    _ = cache.Set(ctx, key, data, ttl)
}
```

## 测试

### 单元测试

使用 `miniredis` 进行单元测试：

```go
import (
    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

func TestMyService(t *testing.T) {
    // 创建测试用的 Redis
    mr, _ := miniredis.Run()
    defer mr.Close()
    
    client := redis.NewClient(&redis.Options{
        Addr: mr.Addr(),
    })
    
    cache := service.NewCacheService(client, "test")
    
    // 测试代码...
}
```

### 集成测试

使用真实的 Redis 进行集成测试：

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    cache := service.NewCacheService(client, "test")
    
    // 测试代码...
}
```

## 相关文档

- [Redis 配置](../../internal/config/config.go)
- [Redis 客户端](../../internal/database/redis.go)
- [需求文档](../../.kiro/specs/genkit-session-management/requirements.md) - 需求 30
- [设计文档](../../.kiro/specs/genkit-session-management/design.md) - 缓存设计章节
