# 并发性能测试总结

## 概述

本文档总结了 Genkit 多模型支持的并发性能测试实现。

## 实现的测试

### 1. TestConcurrentCallsPerformance

主测试函数，包含 5 个子测试场景。

#### 子测试 1: 低并发测试（10个并发）

**目的**: 验证系统在低负载下的基本性能

**配置**:

- 并发数: 10
- 提供商: Google AI (Gemini)
- 请求内容: "并发测试消息 X：请简短回答什么是AI？"

**验证点**:

- 所有请求成功（成功率 = 100%）
- 平均延迟 < 10秒

**输出指标**:

- 总请求数、成功数、失败数
- 成功率、总耗时、吞吐量
- 延迟统计（平均、最小、最大、中位数、标准差）

#### 子测试 2: 中等并发测试（50个并发）

**目的**: 验证系统在中等负载下的性能

**配置**:

- 并发数: 50
- 提供商: Google AI (Gemini)

**验证点**:

- 成功率 ≥ 95%
- 平均延迟 < 15秒

#### 子测试 3: 高并发测试（100个并发）

**目的**: 验证系统能够处理至少100个并发请求（满足需求）

**配置**:

- 并发数: 100
- 提供商: Google AI (Gemini)

**验证点**:

- 成功率 ≥ 90%
- 平均延迟 < 20秒
- 输出确认信息："✓ 系统能够处理 100 个并发请求"

#### 子测试 4: 并发流式调用测试（20个并发）

**目的**: 验证流式调用的并发性能

**配置**:

- 并发数: 20
- 提供商: Google AI (Gemini)
- 调用方式: GenerateStream

**验证点**:

- 所有请求成功（成功率 = 100%）
- 平均 TTFB < 5秒

**输出指标**:

- TTFB 统计（平均、最小、最大、中位数）
- 总时间统计
- 数据块数量、总长度

#### 子测试 5: 混合并发测试（多提供商）

**目的**: 验证多提供商并发调用的性能

**配置**:

- 并发数: 50
- 提供商: Google AI + Azure OpenAI（轮流使用）
- 需要配置 Azure OpenAI 环境变量

**验证点**:

- 成功率 ≥ 90%
- 输出确认信息："✓ 系统能够处理多提供商的并发请求"

## 核心函数

### runConcurrentCalls

运行并发调用测试的核心函数。

**功能**:

1. 启动指定数量的 goroutine
2. 每个 goroutine 独立调用 Generate
3. 使用 channel 收集结果
4. 计算统计数据

**返回**: ConcurrentTestResults（包含成功率、延迟统计、错误详情）

### runConcurrentStreamCalls

运行并发流式调用测试。

**功能**:

1. 启动并发流式调用
2. 测量每个请求的 TTFB
3. 统计数据块数量和总长度
4. 计算 TTFB 和总时间统计

**返回**: StreamTestResults（包含 TTFB 统计、总时间统计）

### runMixedConcurrentCalls

运行混合提供商并发测试。

**功能**:

1. 轮流使用不同的提供商
2. 测试提供商切换的并发性能
3. 验证多提供商场景下的稳定性

**返回**: ConcurrentTestResults

### 分析函数

- `analyzeResults`: 分析并发测试结果
- `analyzeStreamResults`: 分析流式并发测试结果
- `analyzeMixedResults`: 分析混合并发测试结果

## 数据结构

### ConcurrentCallResult

单个并发调用的结果。

```go
type ConcurrentCallResult struct {
    Index    int           // 请求索引
    Latency  time.Duration // 延迟
    Success  bool          // 是否成功
    Error    error         // 错误信息
    Response string        // 响应内容
}
```

### ConcurrentTestResults

并发测试结果汇总。

```go
type ConcurrentTestResults struct {
    TotalCount   int           // 总请求数
    SuccessCount int           // 成功数
    ErrorCount   int           // 失败数
    TotalTime    time.Duration // 总耗时
    Stats        LatencyStats  // 延迟统计
    Errors       []error       // 错误列表
}
```

### StreamCallResult

单个流式调用的结果。

```go
type StreamCallResult struct {
    Index       int           // 请求索引
    TTFB        time.Duration // 首字节时间
    TotalTime   time.Duration // 总时间
    ChunkCount  int           // 数据块数量
    Success     bool          // 是否成功
    Error       error         // 错误信息
    TotalLength int           // 总长度
}
```

### StreamTestResults

流式测试结果汇总。

```go
type StreamTestResults struct {
    TotalCount   int          // 总请求数
    SuccessCount int          // 成功数
    ErrorCount   int          // 失败数
    Stats        LatencyStats // TTFB 统计
    TotalStats   LatencyStats // 总时间统计
    Errors       []error      // 错误列表
}
```

## 测试流程

### 1. 环境准备

```
设置测试环境
  ↓
连接测试数据库
  ↓
创建 Repository 和 Client
  ↓
检查环境变量
```

### 2. 配置创建

```
创建测试租户
  ↓
创建模型配置
  ↓
插入数据库
  ↓
预热调用（首次初始化）
```

### 3. 并发测试

```
启动 N 个 goroutine
  ↓
每个 goroutine 独立调用 API
  ↓
通过 channel 收集结果
  ↓
等待所有 goroutine 完成
  ↓
计算统计数据
  ↓
验证结果
```

### 4. 结果分析

```
计算成功率
  ↓
计算吞吐量
  ↓
计算延迟统计
  ↓
分析错误类型
  ↓
输出详细报告
```

### 5. 清理

```
删除测试配置
  ↓
关闭数据库连接
```

## 性能指标

### 基本指标

- **成功率**: 成功请求数 / 总请求数 × 100%
- **吞吐量**: 成功请求数 / 总耗时（请求/秒）
- **总耗时**: 从开始到所有请求完成的时间

### 延迟指标

- **平均延迟**: 所有成功请求的平均响应时间
- **最小延迟**: 最快的请求响应时间
- **最大延迟**: 最慢的请求响应时间
- **中位数延迟**: 50% 的请求响应时间
- **标准差**: 延迟的离散程度

### 流式调用指标

- **TTFB**: Time To First Byte，首字节时间
- **总时间**: 从发起请求到接收完所有数据的时间
- **数据块数量**: 流式响应的数据块总数
- **总长度**: 所有数据块的总字符数

## 运行方式

### 使用测试脚本

```bash
# 设置环境变量
export GOOGLE_API_KEY=your_api_key

# 运行测试
./test/test_concurrent_performance.sh
```

### 直接运行 Go 测试

```bash
# 运行所有子测试
go test -v -timeout 30m ./test/e2e -run TestConcurrentCallsPerformance

# 运行特定子测试
go test -v -timeout 30m ./test/e2e -run TestConcurrentCallsPerformance/低并发测试
go test -v -timeout 30m ./test/e2e -run TestConcurrentCallsPerformance/高并发测试
```

## 预期结果

### 低并发（10个并发）

```
✓ 并发测试结果分析:
  总请求数: 10
  成功数: 10
  失败数: 0
  成功率: 100.00%
  总耗时: ~15s
  吞吐量: ~0.66 请求/秒

  延迟统计:
    平均延迟: ~1.5s
    最小延迟: ~1.2s
    最大延迟: ~1.9s
```

### 高并发（100个并发）

```
✓ 并发测试结果分析:
  总请求数: 100
  成功数: 95+
  失败数: <5
  成功率: ≥95%
  总耗时: ~45s
  吞吐量: ~2.1 请求/秒

  延迟统计:
    平均延迟: ~4.5s
    最小延迟: ~2.3s
    最大延迟: ~9.0s

✓ 系统能够处理 100 个并发请求
```

## 故障排查

### 问题 1: 数据库连接失败

**错误信息**: `failed to connect to database`

**解决方案**:

1. 检查数据库是否运行
2. 验证数据库连接参数
3. 确保测试数据库存在

### 问题 2: API 密钥错误

**错误信息**: `authentication failed` 或 `invalid API key`

**解决方案**:

1. 检查环境变量是否正确设置
2. 验证 API 密钥是否有效
3. 检查 API 密钥配额

### 问题 3: 高失败率

**症状**: 成功率低于预期

**可能原因**:

- API 配额不足
- 网络不稳定
- 提供商限流

**解决方案**:

1. 检查 API 配额
2. 降低并发数重试
3. 检查网络连接

### 问题 4: 测试超时

**症状**: 测试运行超过 30 分钟

**解决方案**:

1. 增加超时时间：`-timeout 60m`
2. 降低并发数
3. 检查是否有死锁

## 性能优化建议

### 1. 连接池优化

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
}
```

### 2. 并发控制

```go
// 使用信号量限制并发数
semaphore := make(chan struct{}, maxConcurrency)
```

### 3. 超时控制

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 4. 资源管理

- 及时关闭连接
- 使用 defer 确保资源释放
- 监控内存使用

## 相关文档

- [并发性能测试快速参考](./CONCURRENT_PERFORMANCE_QUICK_REF.md)
- [性能测试快速参考](./PERFORMANCE_TEST_QUICK_REF.md)
- [任务完成报告](../../TASK_6.2_CONCURRENT_PERFORMANCE_COMPLETION.md)

## 总结

并发性能测试实现了完整的测试套件，包括：

1. ✅ 5 个测试场景（低、中、高并发 + 流式 + 混合）
2. ✅ 全面的性能指标（成功率、延迟、吞吐量、TTFB）
3. ✅ 详细的结果分析
4. ✅ 便捷的测试工具
5. ✅ 满足需求：验证系统能够处理至少 100 个并发请求

测试实现遵循 Go 测试最佳实践，提供清晰的结果分析和详细的文档。
