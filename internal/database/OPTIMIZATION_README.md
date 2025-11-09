# 数据库查询优化指南

本文档介绍了 Genkit AI Service 中实现的数据库查询优化功能。

## 功能概述

数据库查询优化模块提供以下功能：

1. **索引验证和管理** - 自动验证必需的索引，检测缺失和未使用的索引
2. **预编译语句** - 使用预编译语句提高查询性能
3. **批量操作** - 支持批量插入、更新和删除操作
4. **查询性能监控** - 记录和分析查询性能，识别慢查询
5. **连接池监控** - 监控数据库连接池的健康状态和利用率

## 快速开始

### 1. 基本配置

```go
import (
    "genkit-ai-service/internal/database"
    "time"
)

// 创建数据库连接
db, err := database.NewPostgresDatabase(&database.PostgresConfig{
    Host:            "localhost",
    Port:            "5432",
    User:            "postgres",
    Password:        "password",
    DBName:          "genkit_db",
    SSLMode:         "disable",
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 5 * time.Minute,
    LogLevel:        "warn",
})

// 连接数据库
ctx := context.Background()
if err := db.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### 2. 应用优化配置

```go
// 使用默认优化配置
config := database.DefaultOptimizationConfig()

// 或自定义配置
config := database.OptimizationConfig{
    EnablePreparedStmt:          true,
    DefaultBatchSize:            100,
    SlowQueryThreshold:          200 * time.Millisecond,
    EnablePerformanceMonitoring: true,
    EnableQueryCache:            false,
    PoolHealthCheckInterval:     30 * time.Second,
    MaxQueryRecords:             1000,
}

// 应用优化
err := database.ApplyOptimizations(db.GetDB(), config)
if err != nil {
    log.Fatal(err)
}
```

### 3. 使用优化管理器

```go
// 创建优化管理器
manager := database.NewOptimizationManager(db.GetDB(), config)

// 初始化（验证索引、启动健康检查等）
if err := manager.Initialize(ctx); err != nil {
    log.Fatal(err)
}
defer manager.Shutdown()

// 获取性能报告
report := manager.GetPerformanceReport()
fmt.Println(report.GetSummary())

// 获取索引报告
indexReport, err := manager.GetIndexReport(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(indexReport.GetSummary())

// 生成优化建议
recommendations, err := manager.GenerateRecommendations(ctx)
if err != nil {
    log.Fatal(err)
}
for _, rec := range recommendations.Recommendations {
    fmt.Println("建议:", rec)
}
```

## 核心功能详解

### 1. 索引验证

索引验证器会检查所有必需的索引是否存在，并提供创建缺失索引的功能。

```go
// 获取索引验证器
validator := db.GetIndexValidator()

// 验证索引
missingIndexes, err := validator.ValidateIndexes(ctx)
if err != nil {
    log.Fatal(err)
}

// 打印缺失的索引
for _, idx := range missingIndexes {
    fmt.Printf("缺失索引 - 表: %s, 列: %v, 原因: %s\n", 
        idx.TableName, idx.Columns, idx.Reason)
}

// 创建缺失的索引
if len(missingIndexes) > 0 {
    err := validator.CreateMissingIndexes(ctx, missingIndexes)
    if err != nil {
        log.Fatal(err)
    }
}

// 获取未使用的索引
unusedIndexes, err := validator.GetUnusedIndexes(ctx)
if err != nil {
    log.Fatal(err)
}

// 生成完整的索引报告
report, err := validator.GenerateIndexReport(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(report.GetSummary())
```

### 2. 查询优化器

查询优化器提供批量操作和预编译语句支持。

```go
// 获取查询优化器
optimizer := db.GetQueryOptimizer()

// 批量插入
users := []User{
    {Name: "Alice", Email: "alice@example.com"},
    {Name: "Bob", Email: "bob@example.com"},
    // ... 更多记录
}
err := optimizer.BatchInsert(ctx, users, 100) // 每批100条
if err != nil {
    log.Fatal(err)
}

// 批量更新
updates := map[string]interface{}{
    "status": "active",
    "updated_at": time.Now(),
}
conditions := map[string]interface{}{
    "tenant_id": tenantID,
    "is_deleted": false,
}
err = optimizer.BatchUpdate(ctx, &User{}, updates, conditions)
if err != nil {
    log.Fatal(err)
}

// 批量删除（软删除）
err = optimizer.BatchDelete(ctx, &User{}, conditions)
if err != nil {
    log.Fatal(err)
}
```

### 3. 性能监控

性能监控器记录所有查询的执行情况，帮助识别性能问题。

```go
// 获取性能监控器
perfMonitor := db.GetPerformanceMonitor()

// 获取所有查询
queries := perfMonitor.GetAllQueries()
for _, q := range queries {
    fmt.Printf("SQL: %s, 耗时: %v, 慢查询: %v\n", 
        q.SQL, q.Duration, q.IsSlow)
}

// 获取慢查询
slowQueries := perfMonitor.GetSlowQueries()
fmt.Printf("发现 %d 个慢查询\n", len(slowQueries))

// 获取查询统计
stats := perfMonitor.GetQueryStats()
fmt.Printf("总查询数: %d, 慢查询: %d (%.2f%%), 平均耗时: %v\n",
    stats.TotalQueries, stats.SlowQueries, 
    stats.SlowQueryRate, stats.AvgDuration)

// 清空查询记录
perfMonitor.Clear()
```

### 4. 连接池监控

连接池监控器提供连接池的健康状态和统计信息。

```go
// 获取连接池监控器
poolMonitor := db.GetConnectionPoolMonitor()

// 获取连接池统计
stats, err := poolMonitor.GetPoolStats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("最大连接数: %d\n", stats.MaxOpenConnections)
fmt.Printf("当前打开连接: %d\n", stats.OpenConnections)
fmt.Printf("使用中: %d\n", stats.InUse)
fmt.Printf("空闲: %d\n", stats.Idle)
fmt.Printf("利用率: %.2f%%\n", stats.GetUtilization())

// 检查健康状态
if !stats.IsHealthy() {
    log.Println("警告: 连接池不健康")
}
```

## 性能优化最佳实践

### 1. 索引优化

- **定期验证索引**: 使用索引验证器定期检查缺失的索引
- **删除未使用的索引**: 未使用的索引会增加写操作的开销
- **复合索引顺序**: 将选择性高的列放在前面
- **覆盖索引**: 对于频繁查询的列，考虑创建覆盖索引

```go
// 定期验证索引（例如在启动时或定时任务中）
missingIndexes, _ := validator.ValidateIndexes(ctx)
if len(missingIndexes) > 0 {
    log.Printf("发现 %d 个缺失的索引", len(missingIndexes))
    // 可以选择自动创建或发送告警
}
```

### 2. 批量操作

- **使用批量插入**: 比单条插入快10-100倍
- **合理的批次大小**: 通常100-1000条为宜
- **事务处理**: 批量操作应该在事务中进行

```go
// 推荐：批量插入
optimizer.BatchInsert(ctx, records, 100)

// 不推荐：循环单条插入
for _, record := range records {
    db.Create(&record)
}
```

### 3. 预编译语句

- **启用预编译语句**: 对于重复执行的查询，预编译语句可以提高性能
- **参数化查询**: 使用参数化查询防止 SQL 注入

```go
// 启用预编译语句
config := database.OptimizationConfig{
    EnablePreparedStmt: true,
}
```

### 4. 连接池配置

- **合理设置连接数**: 根据并发量和数据库性能调整
- **监控连接池**: 定期检查连接池利用率
- **连接生命周期**: 设置合理的连接最大生命周期

```go
// 推荐的连接池配置
config := &database.PostgresConfig{
    MaxOpenConns:    25,  // 最大打开连接数
    MaxIdleConns:    5,   // 最大空闲连接数
    ConnMaxLifetime: 5 * time.Minute, // 连接最大生命周期
}
```

### 5. 查询优化

- **避免 SELECT ***: 只查询需要的列
- **使用 LIMIT**: 限制返回的行数
- **添加 WHERE 条件**: 使用索引列进行过滤
- **避免 N+1 查询**: 使用 JOIN 或预加载

```go
// 推荐：只查询需要的列
db.Select("id", "name", "email").Find(&users)

// 不推荐：查询所有列
db.Find(&users)
```

## 监控和告警

### 1. 慢查询监控

```go
// 设置慢查询阈值
config := database.OptimizationConfig{
    SlowQueryThreshold: 200 * time.Millisecond,
}

// 定期检查慢查询
slowQueries := perfMonitor.GetSlowQueries()
if len(slowQueries) > 10 {
    // 发送告警
    log.Printf("警告: 发现 %d 个慢查询", len(slowQueries))
}
```

### 2. 连接池监控

```go
// 定期检查连接池健康状态
stats, _ := poolMonitor.GetPoolStats()
if stats.GetUtilization() > 80 {
    log.Println("警告: 连接池利用率过高")
}

if !stats.IsHealthy() {
    log.Println("警告: 连接池不健康")
}
```

### 3. 性能报告

```go
// 生成性能报告
report := manager.GetPerformanceReport()
if report.HasIssues() {
    log.Println("警告: 发现性能问题")
    log.Println(report.GetSummary())
}
```

## 故障排查

### 问题1: 慢查询过多

**症状**: 慢查询率超过10%

**解决方案**:

1. 检查是否缺少索引
2. 优化查询语句
3. 考虑添加缓存
4. 检查数据量是否过大

```go
// 获取慢查询详情
slowQueries := perfMonitor.GetSlowQueries()
for _, q := range slowQueries {
    fmt.Printf("慢查询: %s, 耗时: %v\n", q.SQL, q.Duration)
}

// 检查缺失的索引
missingIndexes, _ := validator.ValidateIndexes(ctx)
```

### 问题2: 连接池耗尽

**症状**: 连接池利用率持续100%，等待时间长

**解决方案**:

1. 增加最大连接数
2. 优化查询性能
3. 检查是否有连接泄漏
4. 减少连接持有时间

```go
// 检查连接池状态
stats, _ := poolMonitor.GetPoolStats()
if stats.GetUtilization() > 90 {
    // 增加最大连接数
    // 或优化查询性能
}
```

### 问题3: 数据库错误率高

**症状**: 数据库错误频繁发生

**解决方案**:

1. 检查数据库连接是否稳定
2. 检查查询语句是否正确
3. 检查数据库资源是否充足
4. 查看数据库日志

```go
// 获取查询统计
stats := perfMonitor.GetQueryStats()
if stats.ErrorQueries > 0 {
    errorRate := float64(stats.ErrorQueries) / float64(stats.TotalQueries) * 100
    fmt.Printf("错误率: %.2f%%\n", errorRate)
}
```

## 环境变量配置

数据库优化相关的环境变量：

```bash
# 数据库连接配置
DB_MAX_OPEN_CONNS=25          # 最大打开连接数
DB_MAX_IDLE_CONNS=5           # 最大空闲连接数
DB_CONN_MAX_LIFETIME=5m       # 连接最大生命周期

# 性能监控配置
DB_SLOW_QUERY_THRESHOLD=200ms # 慢查询阈值
DB_ENABLE_PERF_MONITORING=true # 启用性能监控
DB_MAX_QUERY_RECORDS=1000     # 最大查询记录数

# 优化配置
DB_ENABLE_PREPARED_STMT=true  # 启用预编译语句
DB_DEFAULT_BATCH_SIZE=100     # 默认批次大小
DB_ENABLE_QUERY_CACHE=false   # 启用查询缓存
```

## 总结

数据库查询优化是提高系统性能的关键。通过合理使用索引、批量操作、预编译语句和连接池配置，可以显著提升数据库性能。同时，持续监控和优化是保持系统高性能的重要手段。

## 相关文档

- [数据库迁移指南](./migrations/README.md)
- [GORM 使用指南](./README.md)
- [监控指标说明](../monitoring/README.md)
