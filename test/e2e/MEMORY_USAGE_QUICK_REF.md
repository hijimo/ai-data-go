# 内存使用测试快速参考

## 概述

内存使用测试用于衡量 Genkit 多模型支持在不同负载下的内存使用情况，确保系统没有内存泄漏并且内存使用在合理范围内。

## 测试场景

### 1. 基准内存使用

- **目的**: 测量单次调用的内存开销
- **验证**: 单次调用内存增长 < 10MB

### 2. 连续调用内存使用

- **目的**: 测试 50 次连续调用的内存增长趋势
- **验证**: 连续调用内存增长 < 20MB，无明显内存泄漏

### 3. 并发调用内存使用

- **目的**: 测试 50 个并发调用的内存使用
- **验证**: 并发调用后堆内存增长 < 50MB

### 4. 多提供商内存使用

- **目的**: 测试多个提供商（Google AI + Azure OpenAI）的内存开销
- **验证**: 多提供商内存增长 < 30MB

### 5. 长时间运行内存稳定性

- **目的**: 测试 100 次调用的内存稳定性
- **验证**:
  - 内存增长率 < 5MB/100次调用
  - 总内存增长 < 30MB

## 环境要求

### 必需环境变量

```bash
# Google AI API 密钥（必需）
export GOOGLE_API_KEY="your-google-api-key"

# 数据库连接（可选，默认使用测试数据库）
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/genkit_test?sslmode=disable"
```

### 可选环境变量（用于多提供商测试）

```bash
# Azure OpenAI 配置
export AZURE_OPENAI_API_KEY="your-azure-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="gpt-4"
```

## 运行测试

### 方式 1: 使用测试脚本（推荐）

```bash
# 设置环境变量
export GOOGLE_API_KEY="your-google-api-key"

# 运行测试脚本
./test/test_memory_usage.sh
```

### 方式 2: 直接运行 Go 测试

```bash
# 运行所有内存使用测试
go test -v -run TestMemoryUsage ./test/e2e/

# 运行特定子测试
go test -v -run TestMemoryUsage/基准内存使用 ./test/e2e/
go test -v -run TestMemoryUsage/连续调用内存使用 ./test/e2e/
go test -v -run TestMemoryUsage/并发调用内存使用 ./test/e2e/
go test -v -run TestMemoryUsage/多提供商内存使用 ./test/e2e/
go test -v -run TestMemoryUsage/长时间运行内存稳定性 ./test/e2e/
```

### 方式 3: 跳过长时间测试

```bash
# 使用 -short 标志跳过性能测试
go test -v -short ./test/e2e/
```

## 测试输出示例

```
========== 阶段 1: 设置测试环境 ==========
✓ 测试环境设置完成
========== 阶段 2: 创建测试配置 ==========
✓ 测试配置创建完成
=== RUN   TestMemoryUsage/基准内存使用
========== 基准内存使用测试 ==========
基准内存状态:
  分配的堆内存: 12.34 MB
  系统内存: 45.67 MB
  堆对象数: 123456
  GC 次数: 5

执行单次调用...

单次调用后内存状态:
  分配的堆内存: 15.23 MB
  系统内存: 46.12 MB
  堆对象数: 125678
  GC 次数: 6

✓ 单次调用内存增长: 2.89 MB
--- PASS: TestMemoryUsage/基准内存使用 (3.45s)
```

## 内存指标说明

### 关键指标

1. **分配的堆内存 (Alloc)**
   - 当前分配给堆对象的字节数
   - 这是最重要的指标，反映实际使用的内存

2. **系统内存 (Sys)**
   - 从操作系统获取的总内存
   - 包括堆、栈、其他内部数据结构

3. **堆对象数 (HeapObjects)**
   - 当前分配的堆对象数量
   - 可以帮助识别对象泄漏

4. **GC 次数 (NumGC)**
   - 垃圾回收执行的次数
   - 频繁的 GC 可能表示内存压力

### 内存增长阈值

| 测试场景 | 阈值 | 说明 |
|---------|------|------|
| 单次调用 | < 10MB | 单次 API 调用的内存开销 |
| 连续调用 (50次) | < 20MB | 检测内存泄漏 |
| 并发调用 (50个) | < 50MB | 并发场景的内存开销 |
| 多提供商 | < 30MB | 多个提供商实例的开销 |
| 长时间运行 (100次) | < 30MB | 长期稳定性 |
| 增长率 | < 5MB/100次 | 内存泄漏检测 |

## 故障排查

### 问题 1: 内存增长超过阈值

**可能原因**:

- 缓存未正确清理
- Goroutine 泄漏
- 连接未关闭

**排查步骤**:

1. 检查 Goroutine 数量是否持续增长
2. 检查是否有未关闭的连接
3. 使用 `pprof` 进行详细的内存分析

```bash
# 启用内存分析
go test -v -run TestMemoryUsage -memprofile=mem.prof ./test/e2e/

# 分析内存使用
go tool pprof mem.prof
```

### 问题 2: GC 次数过多

**可能原因**:

- 创建了大量临时对象
- 内存分配过于频繁

**解决方案**:

- 使用对象池减少分配
- 复用缓冲区
- 优化数据结构

### 问题 3: 测试超时

**可能原因**:

- API 响应慢
- 网络问题
- 并发数过高

**解决方案**:

- 增加测试超时时间
- 减少并发数
- 检查网络连接

## 性能优化建议

### 1. 减少内存分配

- 使用 `sync.Pool` 复用对象
- 预分配切片容量
- 避免不必要的字符串拼接

### 2. 优化缓存策略

- 设置合理的缓存大小限制
- 实现 LRU 淘汰策略
- 定期清理过期缓存

### 3. 控制并发

- 使用 worker pool 限制并发数
- 实现请求队列
- 添加背压机制

### 4. 监控和告警

- 设置内存使用告警
- 监控 GC 频率和暂停时间
- 跟踪 Goroutine 数量

## 相关文档

- [性能测试快速参考](./PERFORMANCE_TEST_QUICK_REF.md)
- [并发性能测试](./CONCURRENT_PERFORMANCE_QUICK_REF.md)
- [错误场景测试](./ERROR_SCENARIOS_QUICK_REF.md)

## 注意事项

1. **测试环境**: 确保在稳定的环境中运行测试，避免其他进程干扰
2. **多次运行**: 内存测试结果可能有波动，建议多次运行取平均值
3. **GC 影响**: 测试中会强制执行 GC，这可能影响实际性能
4. **API 限制**: 注意 API 调用频率限制，避免被限流
5. **清理资源**: 测试结束后会自动清理测试数据，但建议检查数据库

## 测试维护

### 更新阈值

如果系统优化后内存使用降低，可以相应降低阈值：

```go
// 在 test/e2e/performance_test.go 中更新
assert.Less(t, memIncrease, 10.0, "单次调用内存增长应该小于 10MB")
```

### 添加新测试场景

参考现有测试结构添加新的内存测试场景：

```go
t.Run("新测试场景", func(t *testing.T) {
    // 1. 记录初始内存
    runtime.GC()
    var startMemStats runtime.MemStats
    runtime.ReadMemStats(&startMemStats)
    
    // 2. 执行测试操作
    // ...
    
    // 3. 记录结束内存
    runtime.GC()
    var endMemStats runtime.MemStats
    runtime.ReadMemStats(&endMemStats)
    
    // 4. 验证内存增长
    memIncrease := float64(endMemStats.Alloc-startMemStats.Alloc) / 1024 / 1024
    assert.Less(t, memIncrease, threshold, "内存增长应该小于阈值")
})
```
