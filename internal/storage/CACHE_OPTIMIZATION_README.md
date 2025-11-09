# 缓存优化实现指南

## 概述

本模块实现了完整的缓存优化策略，包括：

- ✅ 多级缓存（L1本地内存 + L2 Redis）
- ✅ 缓存穿透防护（布隆过滤器 + 空值缓存）
- ✅ 缓存击穿防护（单飞模式）
- ✅ 缓存雪崩防护（随机过期时间）
- ✅ 缓存键管理（统一命名规范）
- ✅ 缓存版本控制（全局失效）
- ✅ 热点数据识别（自动延长TTL）
- ✅ 缓存预热（启动时加载）
- ✅ 缓存统计（命中率监控）

## 架构设计

### 多级缓存架构

```
┌─────────────────────────────────────────────────────────┐
│                    应用层                                │
│                 (Service Layer)                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              CacheOptimizer                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  - 布隆过滤器（防穿透）                           │  │
│  │  - 单飞模式（防击穿）                             │  │
│  │  - 热点追踪（智能TTL）                            │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            MultiLevelCache                               │
│  ┌──────────────────────────────────────────────────┐  │
│  │  L1: 本地内存缓存                                 │  │
│  │  - 快速访问（纳秒级）                             │  │
│  │  - LRU淘汰策略                                    │  │
│  │  - 版本控制                                       │  │
│  └──────────────────────────────────────────────────┘  │
│                     │                                    │
│                     ▼                                    │
│  ┌──────────────────────────────────────────────────┐  │
│  │  L2: Redis 缓存                                   │  │
│  │  - 分布式共享                                     │  │
│  │  - 持久化存储                                     │  │
│  │  - 随机过期（防雪崩）                             │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. MultiLevelCache - 多级缓存

提供L1（本地内存）+ L2（Redis）的两级缓存架构。

**特性**：

- 自动回填：L2命中时自动回填到L1
- 版本控制：支持全局缓存失效
- 随机抖动：防止缓存雪崩
- 空值缓存：防止缓存穿透
- 统计信息：实时监控命中率

**使用示例**：

```go
// 创建多级缓存
config := MultiLevelCacheConfig{
    LocalMaxEntries:   1000,
    LocalDefaultTTL:   5 * time.Minute,
    EnableLocalCache:  true,
    CacheVersion:      "v1",
    NullValueTTL:      1 * time.Minute,
    ExpirationJitter:  30 * time.Second,
}

mlCache := NewMultiLevelCache(redisCache, config, logger)

// 获取缓存
var data MyData
err := mlCache.Get(ctx, "my:key", &data)

// 设置缓存
err = mlCache.Set(ctx, "my:key", data, 10*time.Minute)

// 设置空值（防穿透）
err = mlCache.SetNullValue(ctx, "nonexistent:key")

// 使缓存版本失效
err = mlCache.InvalidateVersion(ctx)

// 获取统计信息
stats := mlCache.GetStats()
fmt.Printf("L1命中率: %s\n", stats["l1_hit_rate"])
fmt.Printf("L2命中率: %s\n", stats["l2_hit_rate"])
```

### 2. CacheOptimizer - 缓存优化器

提供高级缓存优化功能。

**特性**：

- 布隆过滤器：快速判断键是否可能存在
- 单飞模式：防止缓存击穿
- 热点识别：自动延长热点数据TTL
- 缓存预热：批量加载数据

**使用示例**：

```go
// 创建优化器
optimizer := NewCacheOptimizer(mlCache, logger)

// 带保护的获取缓存
var user User
err := optimizer.GetWithProtection(ctx, "user:123", &user, func() (interface{}, error) {
    // 缓存未命中时的加载逻辑
    return userRepo.FindByID(ctx, "123")
})

// 预热缓存
keys := []string{"user:1", "user:2", "user:3"}
err = optimizer.PrewarmCache(ctx, keys, func(key string) (interface{}, error) {
    userID := strings.TrimPrefix(key, "user:")
    return userRepo.FindByID(ctx, userID)
})

// 获取热点数据
hotKeys := optimizer.GetHotKeys()
fmt.Printf("热点数据: %v\n", hotKeys)

// 获取优化统计
stats := optimizer.GetOptimizationStats()
```

### 3. CacheKeyManager - 缓存键管理器

提供统一的缓存键命名规范。

**键格式规范**：

```
{namespace}:{keyType}:{tenantId}:{resourceId}
```

**使用示例**：

```go
// 创建键管理器
keyMgr := NewCacheKeyManager("genkit")

// 构建各类缓存键
contextKey := keyMgr.BuildContextKey(tenantID, sessionID)
// 结果: genkit:context:tenant123:session456

memoryKey := keyMgr.BuildMemoryKey(tenantID, memoryID)
// 结果: genkit:memory:tenant123:memory789

summaryKey := keyMgr.BuildSummaryKey(tenantID, sessionID)
// 结果: genkit:summary:tenant123:session456

vectorKey := keyMgr.BuildVectorSearchKey(tenantID, "查询文本", 10)
// 结果: genkit:vector:tenant123:a1b2c3d4

tokenKey := keyMgr.BuildTokenUsageKey(tenantID, "2024-01-15")
// 结果: genkit:token:tenant123:2024-01-15

// 构建模式匹配键
pattern := keyMgr.BuildPatternForTenant(tenantID)
// 结果: genkit:*:tenant123:*

pattern = keyMgr.BuildPatternForTenantAndType(tenantID, KeyTypeContext)
// 结果: genkit:context:tenant123:*

// 解析缓存键
keyType, tenantID, parts := keyMgr.ParseKey("genkit:context:tenant123:session456")
// keyType: "context"
// tenantID: "tenant123"
// parts: ["context", "tenant123", "session456"]
```

## 防护机制详解

### 1. 缓存穿透防护

**问题**：大量请求查询不存在的数据，导致请求直达数据库。

**解决方案**：

- 布隆过滤器：快速判断键是否可能存在
- 空值缓存：缓存不存在的键（短TTL）

```go
// 自动防护
err := optimizer.GetWithProtection(ctx, key, &dest, func() (interface{}, error) {
    data, err := db.Query(id)
    if err == sql.ErrNoRows {
        return nil, nil  // 返回nil表示数据不存在
    }
    return data, err
})
```

### 2. 缓存击穿防护

**问题**：热点数据过期瞬间，大量请求同时访问数据库。

**解决方案**：

- 单飞模式：同一时刻只有一个请求加载数据
- 其他请求等待第一个请求完成

```go
// 自动使用单飞模式
err := optimizer.GetWithProtection(ctx, key, &dest, loader)
// 多个并发请求会自动合并为一个数据库查询
```

### 3. 缓存雪崩防护

**问题**：大量缓存同时过期，导致数据库压力激增。

**解决方案**：

- 随机过期时间：在基础TTL上添加随机抖动

```go
// 自动添加随机抖动
mlCache.Set(ctx, key, value, 10*time.Minute)
// 实际TTL: 10分钟 + [0, 30秒]的随机值
```

### 4. 热点数据优化

**问题**：部分数据访问频率极高，频繁过期重建。

**解决方案**：

- 自动识别热点数据（5分钟内访问≥10次）
- 自动延长热点数据TTL

```go
// 自动识别和优化
err := optimizer.GetWithProtection(ctx, key, &dest, loader)
// 热点数据自动使用30分钟TTL，普通数据10分钟
```

## 实际应用场景

### 场景1：会话上下文缓存

```go
type ContextService struct {
    optimizer *CacheOptimizer
    keyMgr    *CacheKeyManager
    repo      ContextRepository
}

func (s *ContextService) GetContext(ctx context.Context, tenantID, sessionID string) (*Context, error) {
    key := s.keyMgr.BuildContextKey(tenantID, sessionID)
    
    var context Context
    err := s.optimizer.GetWithProtection(ctx, key, &context, func() (interface{}, error) {
        return s.repo.GetBySessionID(ctx, sessionID)
    })
    
    return &context, err
}
```

### 场景2：向量搜索结果缓存

```go
func (s *MemoryService) SearchMemories(ctx context.Context, tenantID, query string, limit int) ([]Memory, error) {
    key := s.keyMgr.BuildVectorSearchKey(tenantID, query, limit)
    
    var memories []Memory
    err := s.optimizer.GetWithProtection(ctx, key, &memories, func() (interface{}, error) {
        // 生成向量并搜索
        vector, err := s.vectorService.GenerateEmbedding(ctx, query)
        if err != nil {
            return nil, err
        }
        return s.qdrant.SearchVectors(ctx, vector, limit, tenantID)
    })
    
    return memories, err
}
```

### 场景3：Token使用量统计

```go
func (s *TokenService) GetDailyUsage(ctx context.Context, tenantID string) (int64, error) {
    date := time.Now().Format("2006-01-02")
    key := s.keyMgr.BuildTokenUsageKey(tenantID, date)
    
    var usage int64
    err := s.optimizer.GetWithProtection(ctx, key, &usage, func() (interface{}, error) {
        return s.repo.GetDailyUsage(ctx, tenantID, date)
    })
    
    return usage, err
}
```

### 场景4：租户数据批量失效

```go
func (s *TenantService) InvalidateTenantCache(ctx context.Context, tenantID string) error {
    // 删除租户所有缓存
    pattern := s.keyMgr.BuildPatternForTenant(tenantID)
    return s.mlCache.DeletePattern(ctx, pattern)
}
```

### 场景5：缓存预热

```go
func (s *CacheWarmer) WarmupActiveSessions(ctx context.Context) error {
    // 获取活跃会话列表
    sessions, err := s.sessionRepo.GetActiveSessions(ctx, 100)
    if err != nil {
        return err
    }
    
    // 构建缓存键
    keys := make([]string, len(sessions))
    for i, session := range sessions {
        keys[i] = s.keyMgr.BuildContextKey(session.TenantID, session.ID)
    }
    
    // 预热缓存
    return s.optimizer.PrewarmCache(ctx, keys, func(key string) (interface{}, error) {
        _, tenantID, parts := s.keyMgr.ParseKey(key)
        sessionID := parts[2]
        return s.contextService.BuildContext(ctx, tenantID, sessionID)
    })
}
```

## 监控和调优

### 获取缓存统计

```go
// 多级缓存统计
cacheStats := mlCache.GetStats()
fmt.Printf("L1命中率: %s\n", cacheStats["l1_hit_rate"])
fmt.Printf("L2命中率: %s\n", cacheStats["l2_hit_rate"])
fmt.Printf("本地缓存大小: %d/%d\n", cacheStats["local_size"], cacheStats["local_max"])

// 优化器统计
optStats := optimizer.GetOptimizationStats()
fmt.Printf("热点数据数量: %d\n", optStats["hot_key_count"])
fmt.Printf("布隆过滤器大小: %d\n", optStats["bloom_size"])
fmt.Printf("热点键列表: %v\n", optStats["hot_keys"])
```

### 性能调优建议

1. **本地缓存大小**：
   - 根据内存大小调整 `LocalMaxEntries`
   - 建议：1000-10000条

2. **本地缓存TTL**：
   - 根据数据更新频率调整 `LocalDefaultTTL`
   - 建议：1-5分钟

3. **空值缓存TTL**：
   - 防止缓存穿透的关键
   - 建议：30秒-2分钟

4. **过期抖动范围**：
   - 防止缓存雪崩
   - 建议：TTL的5-10%

5. **热点数据阈值**：
   - 当前：5分钟内访问≥10次
   - 可根据业务调整

## 配置示例

```go
// 生产环境配置
config := MultiLevelCacheConfig{
    LocalMaxEntries:   5000,              // 本地缓存5000条
    LocalDefaultTTL:   3 * time.Minute,   // 本地缓存3分钟
    EnableLocalCache:  true,              // 启用本地缓存
    CacheVersion:      "v1",              // 缓存版本
    NullValueTTL:      1 * time.Minute,   // 空值缓存1分钟
    ExpirationJitter:  30 * time.Second,  // 30秒随机抖动
}

// 开发环境配置
devConfig := MultiLevelCacheConfig{
    LocalMaxEntries:   100,
    LocalDefaultTTL:   1 * time.Minute,
    EnableLocalCache:  true,
    CacheVersion:      "dev",
    NullValueTTL:      10 * time.Second,
    ExpirationJitter:  5 * time.Second,
}
```

## 注意事项

1. **内存管理**：
   - 本地缓存会占用应用内存
   - 合理设置 `LocalMaxEntries`
   - 监控内存使用情况

2. **数据一致性**：
   - 多级缓存可能导致短暂不一致
   - 更新数据时记得清除缓存
   - 使用版本控制进行全局失效

3. **热点数据**：
   - 自动识别可能不够精确
   - 可手动标记关键数据
   - 定期检查热点列表

4. **缓存穿透**：
   - 布隆过滤器有误判率
   - 空值缓存是最后防线
   - 监控数据库查询量

5. **缓存预热**：
   - 启动时预热可能较慢
   - 可异步执行
   - 失败不应影响启动

## 相关需求

本实现满足以下需求：

- **需求 5.1**: 缓存策略和性能优化
  - 多级缓存架构
  - 缓存穿透/击穿/雪崩防护
  - 热点数据优化
  - 缓存预热

## 下一步

- [ ] 集成 Prometheus 监控指标
- [ ] 实现缓存预热定时任务
- [ ] 添加缓存管理API
- [ ] 实现更精确的LRU淘汰算法
- [ ] 支持分布式布隆过滤器
