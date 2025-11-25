# Genkit 实例缓存机制实现总结

## 实现概述

本次实现完成了 Genkit Client 的实例缓存机制，确保每个租户和模型组合只创建一个 Genkit 实例，并通过读写锁保证并发安全。

## 核心功能

### 1. 缓存机制

- **缓存键格式**: `{tenantID}_{modelName}`
- **缓存存储**: `map[string]*genkit.Genkit`
- **并发保护**: `sync.RWMutex` 读写锁

### 2. 双重检查锁定模式

实现了标准的双重检查锁定（Double-Checked Locking）模式，确保：

1. **第一次检查（读锁）**: 快速检查缓存是否存在
2. **获取写锁**: 如果缓存未命中，获取写锁准备初始化
3. **第二次检查（写锁）**: 再次检查缓存，防止多个 goroutine 同时初始化
4. **初始化并缓存**: 只有在确认缓存不存在时才初始化新实例

### 3. 缓存管理方法

#### `getOrInitGenkit(ctx, tenantID, modelName)`

- 获取或初始化 Genkit 实例
- 自动从数据库查询配置
- 验证配置有效性
- 缓存实例以提高性能

#### `ClearCache(tenantID, modelName)`

- 清理指定租户和模型的缓存
- 用于配置更新后强制重新初始化

#### `ClearAllCache()`

- 清理所有缓存实例
- 用于系统维护或重置

#### `GetCacheSize()`

- 获取当前缓存的实例数量
- 用于监控和调试

#### `Close()`

- 关闭客户端并清理所有缓存
- 确保资源正确释放

## 并发安全性

### 读写锁使用策略

1. **读操作（查询缓存）**: 使用 `RLock()` / `RUnlock()`
   - 允许多个 goroutine 同时读取
   - 不阻塞其他读操作

2. **写操作（更新缓存）**: 使用 `Lock()` / `Unlock()`
   - 独占访问，阻塞所有其他操作
   - 确保缓存更新的原子性

### 并发测试验证

通过 `TestConcurrentGetOrInitGenkit` 测试验证：

- 10 个 goroutine 并发调用同一个租户和模型
- 验证只创建了一个 Genkit 实例
- 验证所有 goroutine 获得的是同一个实例

## 性能优化

### 1. 懒加载

- 只在首次使用时初始化 Genkit 实例
- 避免启动时加载所有可能的配置

### 2. 实例复用

- 同一租户和模型的多次调用共享同一个实例
- 减少初始化开销和内存占用

### 3. 读写分离

- 使用读写锁而非互斥锁
- 允许多个读操作并发执行
- 只在写操作时独占访问

## 代码示例

### 基本使用

```go
// 创建客户端（注入配置仓储）
client := NewClientWithRepo(configRepo)

// 获取或初始化实例（自动缓存）
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    return err
}

// 使用实例进行 AI 调用
// ...
```

### 缓存管理

```go
// 清理特定缓存（配置更新后）
client.ClearCache(tenantID, modelName)

// 清理所有缓存（系统维护）
client.ClearAllCache()

// 获取缓存统计
size := client.GetCacheSize()
log.Printf("当前缓存实例数: %d", size)

// 关闭客户端（清理所有资源）
client.Close()
```

## 测试覆盖

### 单元测试

1. ✅ `TestGetOrInitGenkit_Success` - 成功初始化
2. ✅ `TestGetOrInitGenkit_CachedInstance` - 缓存命中
3. ✅ `TestGetOrInitGenkit_ModelDisabled` - 模型禁用
4. ✅ `TestGetOrInitGenkit_InvalidTenantID` - 无效租户ID
5. ✅ `TestGetOrInitGenkit_ConfigNotFound` - 配置不存在
6. ✅ `TestGetOrInitGenkit_InvalidConfig` - 无效配置
7. ✅ `TestGetOrInitGenkit_UnsupportedProvider` - 不支持的提供商
8. ✅ `TestGetOrInitGenkit_NoRepository` - 未注入仓储
9. ✅ `TestGetOrInitGenkit_InvalidJSON` - 无效JSON配置

### 缓存管理测试

10. ✅ `TestClearCache` - 清理指定缓存
11. ✅ `TestClearAllCache` - 清理所有缓存
12. ✅ `TestGetCacheSize` - 获取缓存大小
13. ✅ `TestClose` - 关闭客户端

### 并发测试

14. ✅ `TestConcurrentGetOrInitGenkit` - 并发访问安全性

## 实现文件

- `internal/genkit/client.go` - 核心实现
- `internal/genkit/client_dynamic_test.go` - 单元测试

## 验收标准完成情况

- ✅ 实现 Genkit 实例缓存机制（key: tenantID_modelName）
- ✅ 添加并发安全的读写锁
- ✅ 使用双重检查锁定模式
- ✅ 提供缓存管理方法
- ✅ 完整的单元测试覆盖
- ✅ 并发安全性验证

## 后续工作

本次实现完成了缓存机制的核心功能。后续任务将继续完成：

1. 实现配置解析逻辑（部分已完成）
2. 实现插件动态创建逻辑（Google AI 已完成，Azure 和百炼待实现）
3. 扩展 Generate 方法支持租户和模型参数

## 注意事项

1. **配置更新**: 当模型配置更新时，需要调用 `ClearCache()` 清理缓存
2. **内存管理**: 缓存会持续占用内存，如果租户和模型组合很多，需要考虑缓存淘汰策略
3. **错误处理**: 初始化失败的实例不会被缓存，下次调用会重新尝试
4. **线程安全**: 所有缓存操作都是线程安全的，可以在并发环境中安全使用

## 性能指标

- **缓存命中**: O(1) 时间复杂度
- **并发读取**: 支持多个 goroutine 同时读取缓存
- **初始化开销**: 只在首次使用时发生
- **内存占用**: 每个租户-模型组合一个实例

## 总结

本次实现成功完成了 Genkit 实例缓存机制，通过双重检查锁定模式确保了并发安全性，并提供了完善的缓存管理功能。所有测试均已通过，代码质量良好，可以安全地用于生产环境。
