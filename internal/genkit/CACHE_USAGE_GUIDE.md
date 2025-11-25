# Genkit 实例缓存使用指南

## 概述

Genkit Client 实现了智能的实例缓存机制，确保每个租户和模型组合只创建一个 Genkit 实例，提高性能并减少资源消耗。

## 核心概念

### 缓存键

缓存键由租户ID和模型名称组成：

```
cacheKey = "{tenantID}_{modelName}"
```

例如：

- `738dbb1f-83e6-4bf5-935c-f0498236440d_gemini-pro`
- `550e8400-e29b-41d4-a716-446655440000_gpt-4`

### 缓存生命周期

1. **首次调用**: 从数据库查询配置，初始化 Genkit 实例，缓存实例
2. **后续调用**: 直接从缓存获取实例，无需重新初始化
3. **配置更新**: 需要手动清理缓存，强制重新初始化
4. **客户端关闭**: 自动清理所有缓存

## 基本使用

### 1. 创建客户端

```go
import (
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/repository"
)

// 创建配置仓储
configRepo := repository.NewModelConfigurationRepository(db)

// 创建 Genkit 客户端（注入仓储）
client := genkit.NewClientWithRepo(configRepo)
```

### 2. 获取或初始化实例

```go
// 自动从缓存获取或初始化新实例
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    log.Printf("初始化失败: %v", err)
    return err
}

// 使用实例进行 AI 调用
// g 是 *genkit.Genkit 实例
// config 是 *GenkitConfig 配置信息
```

### 3. 使用实例

```go
// 非流式调用
result, err := client.Generate(ctx, prompt, options)

// 流式调用
streamChan, err := client.GenerateStream(ctx, prompt, options)
```

## 缓存管理

### 清理指定缓存

当模型配置更新后，需要清理对应的缓存：

```go
// 清理特定租户和模型的缓存
client.ClearCache(tenantID, modelName)

// 下次调用会重新从数据库读取配置并初始化
```

### 清理所有缓存

系统维护或重启时清理所有缓存：

```go
// 清理所有缓存实例
client.ClearAllCache()
```

### 获取缓存统计

监控缓存使用情况：

```go
// 获取当前缓存的实例数量
size := client.GetCacheSize()
log.Printf("当前缓存实例数: %d", size)
```

### 关闭客户端

应用关闭时清理资源：

```go
// 关闭客户端并清理所有缓存
defer client.Close()
```

## 并发使用

缓存机制是完全线程安全的，可以在并发环境中安全使用：

```go
// 多个 goroutine 可以同时调用
for i := 0; i < 10; i++ {
    go func() {
        g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
        if err != nil {
            log.Printf("错误: %v", err)
            return
        }
        
        // 使用实例...
    }()
}
```

**保证**：

- 同一租户和模型只会初始化一次
- 所有 goroutine 获得的是同一个实例
- 不会出现竞态条件

## 配置更新场景

### 场景 1: 更新模型配置

```go
// 1. 更新数据库中的配置
err := configService.UpdateModelConfig(ctx, configID, updateReq)
if err != nil {
    return err
}

// 2. 清理缓存，强制重新初始化
client.ClearCache(tenantID, modelName)

// 3. 下次调用会使用新配置
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
```

### 场景 2: 禁用模型

```go
// 1. 禁用模型
err := configService.DisableModel(ctx, configID)
if err != nil {
    return err
}

// 2. 清理缓存
client.ClearCache(tenantID, modelName)

// 3. 下次调用会返回"模型已禁用"错误
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
// err: "模型已禁用: gemini-pro"
```

### 场景 3: 删除模型配置

```go
// 1. 删除配置
err := configService.DeleteModelConfig(ctx, configID)
if err != nil {
    return err
}

// 2. 清理缓存
client.ClearCache(tenantID, modelName)

// 3. 下次调用会返回"配置不存在"错误
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
// err: "获取模型配置失败: record not found"
```

## 性能优化建议

### 1. 预热缓存

在应用启动时预热常用的模型：

```go
func warmupCache(client *genkit.Client, configs []ModelConfig) {
    for _, config := range configs {
        go func(c ModelConfig) {
            _, _, err := client.getOrInitGenkit(
                context.Background(),
                c.TenantID,
                c.ModelName,
            )
            if err != nil {
                log.Printf("预热失败: %v", err)
            }
        }(config)
    }
}
```

### 2. 监控缓存命中率

```go
type CacheMetrics struct {
    Hits   int64
    Misses int64
}

func (m *CacheMetrics) HitRate() float64 {
    total := m.Hits + m.Misses
    if total == 0 {
        return 0
    }
    return float64(m.Hits) / float64(total)
}
```

### 3. 定期清理不活跃缓存

如果租户和模型组合很多，可以实现 LRU 缓存淘汰策略：

```go
// 注意：当前实现不支持自动淘汰
// 如果需要，可以扩展实现 LRU 缓存
```

## 错误处理

### 常见错误

1. **模型配置仓储未初始化**

```go
err: "模型配置仓储未初始化"
// 解决：使用 NewClientWithRepo() 创建客户端
```

2. **无效的租户ID**

```go
err: "无效的租户ID: invalid UUID length: 10"
// 解决：确保传入有效的 UUID 格式
```

3. **模型已禁用**

```go
err: "模型已禁用: gemini-pro"
// 解决：启用模型或使用其他模型
```

4. **配置不存在**

```go
err: "获取模型配置失败: record not found"
// 解决：创建模型配置或检查租户ID和模型名称
```

5. **配置验证失败**

```go
err: "配置验证失败: Azure OpenAI 配置缺少 endpoint"
// 解决：补全必需的配置字段
```

## 最佳实践

### 1. 使用依赖注入

```go
type AIService struct {
    genkitClient genkit.Client
}

func NewAIService(genkitClient genkit.Client) *AIService {
    return &AIService{
        genkitClient: genkitClient,
    }
}
```

### 2. 统一错误处理

```go
func (s *AIService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    g, config, err := s.genkitClient.getOrInitGenkit(
        ctx,
        req.TenantID,
        req.ModelName,
    )
    if err != nil {
        // 记录日志
        log.Printf("初始化 Genkit 失败: %v", err)
        
        // 返回友好的错误信息
        return nil, fmt.Errorf("AI 服务暂时不可用，请稍后重试")
    }
    
    // 使用实例...
}
```

### 3. 添加监控指标

```go
func (s *AIService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    start := time.Now()
    
    g, config, err := s.genkitClient.getOrInitGenkit(
        ctx,
        req.TenantID,
        req.ModelName,
    )
    
    // 记录初始化耗时
    initDuration := time.Since(start)
    metrics.RecordGenkitInit(req.TenantID, req.ModelName, initDuration, err == nil)
    
    if err != nil {
        return nil, err
    }
    
    // 继续处理...
}
```

### 4. 优雅关闭

```go
func main() {
    // 创建客户端
    client := genkit.NewClientWithRepo(configRepo)
    
    // 注册关闭处理
    defer func() {
        log.Println("关闭 Genkit 客户端...")
        if err := client.Close(); err != nil {
            log.Printf("关闭失败: %v", err)
        }
    }()
    
    // 应用逻辑...
}
```

## 调试技巧

### 1. 查看缓存状态

```go
// 获取缓存大小
size := client.GetCacheSize()
log.Printf("当前缓存实例数: %d", size)

// 如果缓存为空，检查是否正确初始化
if size == 0 {
    log.Println("警告：缓存为空，可能未正确初始化")
}
```

### 2. 验证缓存命中

```go
// 第一次调用
start := time.Now()
g1, _, err := client.getOrInitGenkit(ctx, tenantID, modelName)
duration1 := time.Since(start)
log.Printf("第一次调用耗时: %v", duration1)

// 第二次调用（应该从缓存获取，更快）
start = time.Now()
g2, _, err := client.getOrInitGenkit(ctx, tenantID, modelName)
duration2 := time.Since(start)
log.Printf("第二次调用耗时: %v", duration2)

// 验证是同一个实例
if g1 == g2 {
    log.Println("✓ 缓存命中")
} else {
    log.Println("✗ 缓存未命中")
}
```

### 3. 测试并发安全性

```go
func TestConcurrency() {
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            g, _, err := client.getOrInitGenkit(ctx, tenantID, modelName)
            if err != nil {
                errors <- err
                return
            }
            
            // 使用实例...
        }()
    }
    
    wg.Wait()
    close(errors)
    
    // 检查错误
    for err := range errors {
        log.Printf("并发错误: %v", err)
    }
}
```

## 总结

Genkit 实例缓存机制提供了：

- ✅ 自动缓存管理
- ✅ 并发安全保证
- ✅ 性能优化
- ✅ 灵活的缓存控制
- ✅ 完整的错误处理

通过正确使用缓存机制，可以显著提高系统性能并减少资源消耗。
