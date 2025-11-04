# 性能测试快速开始指南

## 快速运行

### 一键运行所有性能测试

```bash
./scripts/run_performance_tests.sh
```

这将：

- 运行所有性能基准测试
- 生成详细报告
- 生成性能摘要
- 检查性能指标是否达标

## 运行特定测试

### 1. 上下文构建性能

```bash
# 基础上下文构建
go test -bench=BenchmarkContextService_BuildContext$ -benchmem ./internal/service

# 包含长期记忆
go test -bench=BenchmarkContextService_BuildContext_WithLongTerm -benchmem ./internal/service

# 包含摘要
go test -bench=BenchmarkContextService_BuildContext_WithSummary -benchmem ./internal/service

# 并发测试
go test -bench=BenchmarkContextService_BuildContext_Parallel -benchmem ./internal/service
```

### 2. 向量检索性能

```bash
# Top-5检索
go test -bench=BenchmarkMemoryRepository_SearchByVector$ -benchmem ./internal/repository

# Top-10检索
go test -bench=BenchmarkMemoryRepository_SearchByVector_TopK10 -benchmem ./internal/repository

# Top-50检索
go test -bench=BenchmarkMemoryRepository_SearchByVector_TopK50 -benchmem ./internal/repository

# 跨会话检索
go test -bench=BenchmarkMemoryRepository_SearchByVectorCrossSessions -benchmem ./internal/repository
```

### 3. 缓存性能

```bash
# 小数据缓存
go test -bench=BenchmarkCacheService_Set_Small -benchmem ./internal/service

# 中等数据缓存
go test -bench=BenchmarkCacheService_Set_Medium -benchmem ./internal/service

# 大数据缓存
go test -bench=BenchmarkCacheService_Set_Large -benchmem ./internal/service

# 并发缓存
go test -bench=BenchmarkCacheService_Parallel -benchmem ./internal/service
```

### 4. 记忆服务性能

```bash
# 存储记忆
go test -bench=BenchmarkMemoryService_StoreMemory -benchmem ./internal/service

# 搜索记忆
go test -bench=BenchmarkMemoryService_SearchMemories$ -benchmem ./internal/service

# 跨会话搜索
go test -bench=BenchmarkMemoryService_SearchMemories_CrossSessions -benchmem ./internal/service

# 清理记忆
go test -bench=BenchmarkMemoryService_CleanupMemories -benchmem ./internal/service
```

## 性能分析

### CPU性能分析

```bash
# 生成CPU性能分析文件
go test -bench=BenchmarkContextService_BuildContext \
  -cpuprofile=cpu.prof ./internal/service

# 查看CPU性能分析（Web界面）
go tool pprof -http=:8080 cpu.prof

# 查看CPU性能分析（命令行）
go tool pprof cpu.prof
```

### 内存性能分析

```bash
# 生成内存性能分析文件
go test -bench=BenchmarkContextService_BuildContext \
  -memprofile=mem.prof ./internal/service

# 查看内存性能分析（Web界面）
go tool pprof -http=:8080 mem.prof

# 查看内存性能分析（命令行）
go tool pprof mem.prof
```

### 阻塞分析

```bash
# 生成阻塞分析文件
go test -bench=BenchmarkContextService_BuildContext_Parallel \
  -blockprofile=block.prof ./internal/service

# 查看阻塞分析
go tool pprof -http=:8080 block.prof
```

### Trace追踪

```bash
# 生成trace文件
go test -bench=BenchmarkContextService_BuildContext \
  -trace=trace.out ./internal/service

# 查看trace
go tool trace trace.out
```

## 性能对比

### 对比不同版本性能

```bash
# 运行基准测试并保存结果（修改前）
go test -bench=. -benchmem ./internal/service > old.txt

# 修改代码...

# 再次运行基准测试（修改后）
go test -bench=. -benchmem ./internal/service > new.txt

# 安装benchcmp工具
go install golang.org/x/tools/cmd/benchcmp@latest

# 对比结果
benchcmp old.txt new.txt
```

### 对比不同策略性能

```bash
# 对比不同上下文构建策略
go test -bench="BenchmarkContextService_Strategy" -benchmem ./internal/service

# 对比不同优化策略
go test -bench="BenchmarkOptimizeContext_Strategy" -benchmem ./internal/service
```

## 常用参数

### benchtime

控制每个基准测试运行的时间：

```bash
# 运行3秒
go test -bench=. -benchtime=3s ./internal/service

# 运行1000次
go test -bench=. -benchtime=1000x ./internal/service
```

### benchmem

显示内存分配统计：

```bash
go test -bench=. -benchmem ./internal/service
```

### cpu

设置使用的CPU核心数：

```bash
# 使用1个核心
go test -bench=. -cpu=1 ./internal/service

# 使用1,2,4个核心分别测试
go test -bench=. -cpu=1,2,4 ./internal/service
```

### count

重复运行测试：

```bash
# 运行5次取平均值
go test -bench=. -count=5 ./internal/service
```

## 性能指标解读

### 基准测试输出示例

```
BenchmarkContextService_BuildContext-8    1000    1234567 ns/op    12345 B/op    123 allocs/op
```

解读：

- `BenchmarkContextService_BuildContext-8`: 测试名称-CPU核心数
- `1000`: 运行次数
- `1234567 ns/op`: 每次操作耗时（纳秒）
- `12345 B/op`: 每次操作分配的内存（字节）
- `123 allocs/op`: 每次操作的内存分配次数

### 性能目标

| 操作 | 目标延迟 | 目标内存 |
|------|----------|----------|
| 上下文构建 | < 10ms | < 100KB |
| 向量检索 | < 50ms | < 50KB |
| 缓存读取 | < 1ms | < 10KB |
| 缓存写入 | < 2ms | < 20KB |

## 常见问题

### Q: 如何只运行特定的测试？

使用 `-bench` 参数指定测试名称的正则表达式：

```bash
# 只运行BuildContext相关测试
go test -bench=BuildContext ./internal/service

# 只运行并发测试
go test -bench=Parallel ./internal/service
```

### Q: 如何增加测试运行时间以获得更准确的结果？

使用 `-benchtime` 参数：

```bash
go test -bench=. -benchtime=10s ./internal/service
```

### Q: 如何查看详细的性能分析？

使用 pprof 工具：

```bash
go test -bench=. -cpuprofile=cpu.prof ./internal/service
go tool pprof -http=:8080 cpu.prof
```

### Q: 如何对比两次测试的性能差异？

使用 benchcmp 工具：

```bash
go test -bench=. ./internal/service > old.txt
# 修改代码
go test -bench=. ./internal/service > new.txt
benchcmp old.txt new.txt
```

### Q: 测试结果波动很大怎么办？

1. 增加运行时间：`-benchtime=10s`
2. 多次运行取平均：`-count=10`
3. 关闭其他程序减少干扰
4. 使用固定的CPU核心数：`-cpu=4`

## 性能优化流程

1. **运行基准测试**

   ```bash
   go test -bench=. -benchmem ./internal/service > baseline.txt
   ```

2. **识别性能瓶颈**

   ```bash
   go test -bench=. -cpuprofile=cpu.prof ./internal/service
   go tool pprof -http=:8080 cpu.prof
   ```

3. **优化代码**
   - 根据性能分析结果优化
   - 减少内存分配
   - 优化算法复杂度

4. **验证优化效果**

   ```bash
   go test -bench=. -benchmem ./internal/service > optimized.txt
   benchcmp baseline.txt optimized.txt
   ```

5. **重复直到达到目标**

## 自动化性能测试

### CI/CD集成

在CI/CD流程中添加性能测试：

```yaml
# .github/workflows/performance.yml
name: Performance Tests

on: [push, pull_request]

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run Performance Tests
        run: ./scripts/run_performance_tests.sh
      - name: Upload Results
        uses: actions/upload-artifact@v2
        with:
          name: performance-results
          path: test-results/performance/
```

### 定期性能测试

使用cron定期运行性能测试：

```bash
# 每天凌晨2点运行性能测试
0 2 * * * cd /path/to/project && ./scripts/run_performance_tests.sh
```

## 更多资源

- [完整性能测试文档](./PERFORMANCE_TESTING.md)
- [任务完成总结](./TASK_41_SUMMARY.md)
- [Go性能测试官方文档](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [pprof使用指南](https://github.com/google/pprof/blob/master/doc/README.md)
