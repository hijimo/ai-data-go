# 缓存预热服务使用指南

## 概述

缓存预热服务（CacheWarmer）用于在系统启动时和运行期间自动预热常用数据的缓存，提高系统响应速度和用户体验。

## 功能特性

### 1. 启动时预热

在系统启动时自动预热活跃会话的缓存数据，包括：

- 会话上下文配置
- 最新会话摘要

### 2. 定期预热

在后台定期刷新活跃会话的缓存，确保缓存数据的新鲜度。

### 3. 按需预热

提供API支持按需预热特定数据：

- 会话列表缓存
- Token使用统计缓存

## 使用方法

### 初始化缓存预热器

```go
import (
    "genkit-ai-service/internal/storage"
    "time"
)

// 创建缓存预热配置
config := &storage.CacheWarmerConfig{
    WarmupInterval:    30 * time.Minute, // 定期预热间隔
    ActiveSessionDays: 7,                // 活跃会话的天数阈值
}

// 创建缓存预热器实例
warmer := storage.NewCacheWarmer(
    cacheService,
    contextRepo,
    summaryRepo,
    sessionRepo,
    db,
    logger,
    config,
)
```

### 启动时预热

在应用启动时调用：

```go
func main() {
    // ... 初始化各种服务 ...
    
    // 执行启动时预热
    ctx := context.Background()
    if err := warmer.WarmupOnStartup(ctx); err != nil {
        log.Fatalf("缓存预热失败: %v", err)
    }
    
    // ... 启动HTTP服务器 ...
}
```

### 启动定期预热

在应用启动后启动定期预热：

```go
func main() {
    // ... 初始化各种服务 ...
    
    // 启动定期预热（在后台运行）
    ctx := context.Background()
    warmer.StartPeriodicWarmup(ctx)
    
    // ... 启动HTTP服务器 ...
    
    // 优雅关闭时停止预热
    defer warmer.Stop()
}
```

### 按需预热会话列表

```go
// 预热用户的会话列表
userID := uuid.MustParse("user-uuid")
page := 1
pageSize := 20

if err := warmer.WarmupSessionList(ctx, userID, page, pageSize); err != nil {
    log.Printf("预热会话列表失败: %v", err)
}
```

### 按需预热Token使用统计

```go
// 预热会话的Token使用统计
sessionID := uuid.MustParse("session-uuid")

if err := warmer.WarmupTokenUsage(ctx, sessionID); err != nil {
    log.Printf("预热Token使用统计失败: %v", err)
}
```

## 配置说明

### CacheWarmerConfig

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| WarmupInterval | time.Duration | 30分钟 | 定期预热的时间间隔 |
| ActiveSessionDays | int | 7天 | 活跃会话的天数阈值，最近N天内有更新的会话被视为活跃 |

## 缓存键命名规范

缓存预热器使用以下缓存键命名规范：

- 上下文配置：`context:config:{sessionID}`
- 最新摘要：`summary:latest:{sessionID}`
- 会话列表：`session:list:{userID}:{page}:{pageSize}`
- Token使用统计：`token:usage:{sessionID}`

## 缓存TTL设置

不同类型的缓存使用不同的TTL（生存时间）：

| 缓存类型 | TTL | 说明 |
|---------|-----|------|
| 上下文配置 | 5分钟 | 会话配置变化较少，但需要及时更新 |
| 最新摘要 | 1小时 | 摘要生成频率较低，可以缓存较长时间 |
| 会话列表 | 10分钟 | 会话列表可能频繁变化 |
| Token使用统计 | 5分钟 | Token统计需要相对实时 |

## 性能考虑

### 启动时预热限制

为避免启动时间过长，启动时预热限制为最多100个活跃会话。

### 定期预热策略

定期预热使用相同的限制（100个会话），确保预热操作不会对系统性能造成显著影响。

### 错误处理

- 单个会话预热失败不会影响其他会话的预热
- 预热失败会记录警告日志，但不会中断整体预热流程
- 如果缓存服务未启用，预热操作会自动跳过

## 监控和日志

### 日志级别

- **INFO**: 预热开始、完成、统计信息
- **WARN**: 单个会话预热失败、缓存服务未启用
- **DEBUG**: 详细的预热操作信息
- **ERROR**: 严重错误（如数据库查询失败）

### 监控指标

建议监控以下指标：

- 预热会话数量
- 预热耗时
- 预热成功率
- 缓存命中率

### 日志示例

```
INFO  开始启动时缓存预热
INFO  找到活跃会话 count=45 active_threshold=2024-01-01T00:00:00Z threshold_days=7
DEBUG 预热上下文配置成功 session_id=xxx cache_key=context:config:xxx
DEBUG 预热最新摘要成功 session_id=xxx summary_id=yyy cache_key=summary:latest:xxx
INFO  启动时缓存预热完成 session_count=45 duration_ms=1234
```

## 最佳实践

### 1. 合理设置活跃会话阈值

根据业务特点调整 `ActiveSessionDays`：

- 高频使用场景：3-5天
- 中频使用场景：7天（默认）
- 低频使用场景：14-30天

### 2. 调整预热间隔

根据系统负载和缓存命中率调整 `WarmupInterval`：

- 高负载系统：增加间隔（如1小时）
- 低负载系统：减少间隔（如15分钟）

### 3. 监控缓存命中率

定期检查缓存命中率，如果命中率低于80%，考虑：

- 减少预热间隔
- 增加活跃会话阈值
- 增加缓存TTL

### 4. 优雅关闭

确保在应用关闭时停止定期预热：

```go
// 使用defer确保清理
defer warmer.Stop()

// 或在信号处理中停止
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
warmer.Stop()
```

## 故障排查

### 问题：预热耗时过长

**可能原因**：

- 活跃会话数量过多
- 数据库查询性能问题
- 网络延迟

**解决方案**：

- 减少活跃会话阈值天数
- 优化数据库索引
- 增加预热间隔

### 问题：缓存命中率低

**可能原因**：

- 预热间隔过长
- 缓存TTL过短
- 活跃会话定义不准确

**解决方案**：

- 减少预热间隔
- 增加缓存TTL
- 调整活跃会话阈值

### 问题：预热失败

**可能原因**：

- 缓存服务未启用
- 数据库连接问题
- 权限问题

**解决方案**：

- 检查Redis连接状态
- 检查数据库连接
- 查看错误日志获取详细信息

## 与其他服务的集成

### 与ContextService集成

ContextService在构建上下文时会自动使用缓存：

```go
// ContextService会先尝试从缓存获取
contextConfig, err := contextService.BuildContext(ctx, req)
```

### 与SummaryService集成

SummaryService在查询摘要时会自动使用缓存：

```go
// SummaryService会先尝试从缓存获取
summary, err := summaryService.GetLatestSummary(ctx, sessionID)
```

## 未来改进

计划中的功能增强：

1. **智能预热**：基于访问模式动态调整预热策略
2. **分布式预热**：支持多实例环境下的协调预热
3. **预热优先级**：根据会话重要性设置预热优先级
4. **预热指标**：提供Prometheus指标用于监控
5. **预热API**：提供HTTP API支持手动触发预热

## 参考资料

- [缓存服务文档](./CACHE_SERVICE_README.md)
- [上下文服务文档](../service/CONTEXT_SERVICE_README.md)
- [摘要服务文档](../service/SUMMARY_SERVICE_README.md)
