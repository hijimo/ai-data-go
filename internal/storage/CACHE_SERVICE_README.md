# 缓存服务使用指南

## 概述

缓存服务提供了一个统一的接口来管理 Redis 缓存，支持 JSON 序列化/反序列化、命名空间隔离、模式匹配删除等功能。

## 功能特性

- ✅ JSON 自动序列化/反序列化
- ✅ 字符串类型缓存支持
- ✅ 命名空间隔离
- ✅ 模式匹配删除
- ✅ 增量操作
- ✅ TTL 管理
- ✅ 批量操作

## 初始化

```go
import (
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/storage"
    "genkit-ai-service/internal/logger"
)

// 创建 Redis 客户端
redisClient, err := database.NewRedisClient(config.Redis, log)
if err != nil {
    // 处理错误
}

// 创建缓存服务
cacheService := storage.NewCacheService(redisClient, log)
```

## 基本用法

### 1. 设置和获取对象（JSON 序列化）

```go
// 定义数据结构
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// 设置缓存（自动序列化为 JSON）
user := User{
    ID:    "123",
    Name:  "张三",
    Email: "zhangsan@example.com",
}
err := cacheService.Set(ctx, "user:123", user, 5*time.Minute)

// 获取缓存（自动反序列化）
var cachedUser User
err = cacheService.Get(ctx, "user:123", &cachedUser)
if err != nil {
    if _, ok := err.(*storage.CacheKeyNotFoundError); ok {
        // 缓存未命中
    } else {
        // 其他错误
    }
}
```

### 2. 设置和获取字符串

```go
// 设置字符串缓存
err := cacheService.SetString(ctx, "session:token", "abc123", 1*time.Hour)

// 获取字符串缓存
token, err := cacheService.GetString(ctx, "session:token")
```

### 3. 删除缓存

```go
// 删除单个键
err := cacheService.Delete(ctx, "user:123")

// 删除多个键
err := cacheService.Delete(ctx, "user:123", "user:456", "user:789")
```

### 4. 模式匹配删除

```go
// 删除所有以 "user:" 开头的键
err := cacheService.DeletePattern(ctx, "user:*")

// 删除所有以 "session:" 开头的键
err := cacheService.DeletePattern(ctx, "session:*")
```

### 5. 检查键是否存在

```go
// 检查单个键
count, err := cacheService.Exists(ctx, "user:123")
if count > 0 {
    // 键存在
}

// 检查多个键
count, err := cacheService.Exists(ctx, "user:123", "user:456")
// count 返回存在的键数量
```

### 6. 增量操作

```go
// 增加计数器
newValue, err := cacheService.Increment(ctx, "page:views", 1)

// 减少计数器
newValue, err := cacheService.Increment(ctx, "inventory:item123", -1)
```

### 7. TTL 管理

```go
// 获取键的剩余生存时间
ttl, err := cacheService.TTL(ctx, "user:123")
if ttl > 0 {
    fmt.Printf("键将在 %v 后过期\n", ttl)
} else if ttl == -1 {
    fmt.Println("键永不过期")
} else if ttl == -2 {
    fmt.Println("键不存在")
}
```

## 命名空间隔离

命名空间功能允许你为不同的功能模块使用独立的键空间，避免键冲突。

### 使用命名空间

```go
// 设置带命名空间的缓存
err := cacheService.SetWithNamespace(ctx, "session", "user123", sessionData, 30*time.Minute)
// 实际键: "session:user123"

// 获取带命名空间的缓存
var data SessionData
err = cacheService.GetWithNamespace(ctx, "session", "user123", &data)

// 删除带命名空间的键
err := cacheService.DeleteWithNamespace(ctx, "session", "user123", "user456")

// 删除整个命名空间
err := cacheService.DeleteNamespace(ctx, "session")
// 删除所有 "session:*" 键
```

### 推荐的命名空间

```go
const (
    NamespaceSession  = "session"   // 会话缓存
    NamespaceContext  = "context"   // 上下文缓存
    NamespaceMemory   = "memory"    // 记忆缓存
    NamespaceSummary  = "summary"   // 摘要缓存
    NamespaceToken    = "token"     // Token 统计
    NamespaceVector   = "vector"    // 向量查询结果
)
```

## 实际应用场景

### 场景 1: 缓存会话上下文

```go
type ContextCache struct {
    SessionID    string
    Summary      string
    Memories     []Memory
    TotalTokens  int
    QualityScore float64
}

// 缓存上下文（5分钟）
func cacheContext(ctx context.Context, sessionID string, contextData ContextCache) error {
    key := fmt.Sprintf("context:%s", sessionID)
    return cacheService.Set(ctx, key, contextData, 5*time.Minute)
}

// 获取缓存的上下文
func getCachedContext(ctx context.Context, sessionID string) (*ContextCache, error) {
    key := fmt.Sprintf("context:%s", sessionID)
    var contextData ContextCache
    err := cacheService.Get(ctx, key, &contextData)
    if err != nil {
        return nil, err
    }
    return &contextData, nil
}
```

### 场景 2: 缓存向量查询结果

```go
// 缓存向量查询结果（30分钟）
func cacheVectorSearchResult(ctx context.Context, queryHash string, results []Memory) error {
    key := fmt.Sprintf("vector:search:%s", queryHash)
    return cacheService.Set(ctx, key, results, 30*time.Minute)
}

// 获取缓存的查询结果
func getCachedVectorSearchResult(ctx context.Context, queryHash string) ([]Memory, error) {
    key := fmt.Sprintf("vector:search:%s", queryHash)
    var results []Memory
    err := cacheService.Get(ctx, key, &results)
    if err != nil {
        return nil, err
    }
    return results, nil
}
```

### 场景 3: Token 使用统计

```go
// 增加 Token 使用量
func incrementTokenUsage(ctx context.Context, tenantID string, tokens int64) (int64, error) {
    key := fmt.Sprintf("token:usage:daily:%s:%s", 
        tenantID, 
        time.Now().Format("2006-01-02"))
    return cacheService.Increment(ctx, key, tokens)
}

// 获取今日 Token 使用量
func getTodayTokenUsage(ctx context.Context, tenantID string) (int64, error) {
    key := fmt.Sprintf("token:usage:daily:%s:%s", 
        tenantID, 
        time.Now().Format("2006-01-02"))
    
    var usage int64
    err := cacheService.Get(ctx, key, &usage)
    if err != nil {
        if _, ok := err.(*storage.CacheKeyNotFoundError); ok {
            return 0, nil // 今日还没有使用
        }
        return 0, err
    }
    return usage, nil
}
```

### 场景 4: 缓存预热

```go
// 预热活跃会话的缓存
func warmupActiveSessions(ctx context.Context, sessionIDs []string) error {
    for _, sessionID := range sessionIDs {
        // 构建上下文
        contextData, err := buildContext(ctx, sessionID)
        if err != nil {
            log.Warn("预热会话失败", "session_id", sessionID, "error", err)
            continue
        }
        
        // 缓存上下文
        key := fmt.Sprintf("context:%s", sessionID)
        if err := cacheService.Set(ctx, key, contextData, 5*time.Minute); err != nil {
            log.Warn("缓存上下文失败", "session_id", sessionID, "error", err)
        }
    }
    return nil
}
```

## 错误处理

```go
// 检查缓存键不存在错误
value, err := cacheService.GetString(ctx, "key")
if err != nil {
    if _, ok := err.(*storage.CacheKeyNotFoundError); ok {
        // 缓存未命中，从数据库加载
        value = loadFromDatabase(ctx, "key")
        // 设置缓存
        cacheService.SetString(ctx, "key", value, 10*time.Minute)
    } else {
        // 其他错误
        return err
    }
}
```

## 性能优化建议

### 1. 合理设置过期时间

```go
// 短期数据（5分钟）
cacheService.Set(ctx, "context:session123", data, 5*time.Minute)

// 中期数据（30分钟）
cacheService.Set(ctx, "vector:search:hash", results, 30*time.Minute)

// 长期数据（1小时）
cacheService.Set(ctx, "summary:session123", summary, 1*time.Hour)
```

### 2. 使用命名空间组织键

```go
// 好的做法：使用命名空间
cacheService.SetWithNamespace(ctx, "session", "user123", data, ttl)

// 或者使用明确的前缀
key := fmt.Sprintf("session:%s", userID)
cacheService.Set(ctx, key, data, ttl)
```

### 3. 批量操作

```go
// 批量删除
keys := []string{"user:1", "user:2", "user:3"}
cacheService.Delete(ctx, keys...)

// 批量检查存在性
count, _ := cacheService.Exists(ctx, keys...)
```

### 4. 缓存穿透防护

```go
// 缓存空值防止缓存穿透
value, err := cacheService.Get(ctx, key, &data)
if err != nil {
    if _, ok := err.(*storage.CacheKeyNotFoundError); ok {
        // 从数据库查询
        dbValue, err := db.Query(ctx, id)
        if err != nil {
            if errors.Is(err, sql.ErrNoRows) {
                // 缓存空值（短时间）
                cacheService.Set(ctx, key, nil, 1*time.Minute)
            }
            return err
        }
        // 缓存正常值
        cacheService.Set(ctx, key, dbValue, 10*time.Minute)
        return dbValue, nil
    }
    return nil, err
}
```

## 监控和调试

### 检查缓存服务状态

```go
if !cacheService.IsEnabled() {
    log.Warn("缓存服务未启用，将直接访问数据库")
    // 降级逻辑
}
```

### 日志记录

缓存服务会自动记录以下日志：

- **Debug**: 缓存命中、缓存设置
- **Info**: 批量删除操作
- **Error**: 缓存操作失败

## 注意事项

1. **序列化限制**: `Set` 方法会将对象序列化为 JSON，确保你的数据结构可以被 JSON 序列化
2. **大对象缓存**: 避免缓存过大的对象（建议 < 1MB），可能影响 Redis 性能
3. **键命名规范**: 使用有意义的键名和命名空间，便于管理和调试
4. **过期时间**: 根据数据更新频率合理设置过期时间
5. **错误处理**: 始终检查缓存操作的错误，特别是 `CacheKeyNotFoundError`
6. **Redis 禁用**: 如果 Redis 被禁用，所有缓存操作将返回错误，需要实现降级逻辑

## 相关需求

本实现满足以下需求：

- **需求 1.3**: 上下文构建中的缓存优化
- **需求 5.1**: 缓存策略和性能优化
