# Genkit 会话管理模块集成测试指南

## 概述

本文档描述了 Genkit 会话管理模块的集成测试策略、测试环境设置和测试执行方法。

## 测试层次

### 1. 单元测试

**位置**: `internal/genkit/flows/*_test.go`, `internal/service/*_test.go`

**目的**: 测试单个函数和方法的逻辑正确性

**特点**:

- 使用 mock 对象隔离依赖
- 快速执行
- 覆盖边界条件和错误处理

**运行方式**:

```bash
# 运行所有单元测试
go test ./internal/genkit/flows/... -v

# 运行特定包的单元测试
go test ./internal/service/... -v

# 查看测试覆盖率
go test ./internal/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 2. 集成测试

**位置**:

- `internal/genkit/flows/integration_test.go`
- `internal/service/integration_test.go`

**目的**: 测试多个组件之间的交互

**特点**:

- 使用真实的数据库（SQLite内存模式）
- 测试完整的业务流程
- 验证多租户隔离
- 测试并发访问

**运行方式**:

```bash
# 运行所有集成测试
go test ./internal/genkit/flows/integration_test.go -v
go test ./internal/service/integration_test.go -v

# 跳过集成测试（快速测试）
go test ./... -short
```

### 3. 端到端测试

**位置**: `cmd/server/e2e_test.go`

**目的**: 测试完整的HTTP API流程

**特点**:

- 模拟真实的HTTP请求
- 测试API路由和中间件
- 验证JWT认证
- 测试多租户隔离
- 性能测试

**运行方式**:

```bash
# 运行端到端测试
go test ./cmd/server/e2e_test.go -v

# 运行特定测试
go test ./cmd/server/e2e_test.go -v -run TestE2E_ContextBuildFlow
```

## 测试环境设置

### 数据库

集成测试和端到端测试使用 SQLite 内存数据库，无需额外配置。

```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
```

### 测试数据

每个测试都会自动创建：

- 测试租户
- 测试用户
- 测试会话
- JWT令牌（端到端测试）

### 清理

测试结束后自动清理所有资源。

## 测试场景

### 1. Flow 集成测试

#### 上下文构建 Flow

**测试场景**:

- 构建基本上下文
- 构建包含长期记忆的上下文
- 构建包含摘要的上下文
- Token超限自动优化

**验证点**:

- 上下文包含正确的消息数量
- Token计算准确
- 质量评分在合理范围内
- 性能满足要求（P50 < 200ms）

#### 记忆检索 Flow

**测试场景**:

- 单会话记忆检索
- 跨会话记忆检索
- 按时间范围过滤
- 按记忆类型过滤

**验证点**:

- 检索结果相关性
- 相似度计算正确
- 访问统计更新
- 租户隔离

#### 摘要生成 Flow

**测试场景**:

- 生成完整摘要
- 生成增量摘要
- 摘要质量评估
- 摘要触发条件检查

**验证点**:

- 摘要内容完整
- 压缩率合理
- 关键主题提取
- 质量评分达标

### 2. 服务层集成测试

#### 上下文服务

**测试场景**:

- 完整的上下文构建流程
- 上下文优化（三种策略）
- 上下文配置管理
- Token超限处理

**验证点**:

- 服务层逻辑正确
- 仓储层调用正确
- 缓存使用正确
- 错误处理完善

#### 记忆服务

**测试场景**:

- 存储和检索记忆
- 记忆清理（四种策略）
- 访问统计更新
- 批量操作

**验证点**:

- 向量生成和存储
- 检索结果排序
- 清理策略执行
- 软删除和硬删除

#### 摘要服务

**测试场景**:

- 摘要生成
- 摘要触发检查
- 摘要质量评估
- 摘要链管理

**验证点**:

- AI服务调用
- 摘要后处理
- 质量评分计算
- 数据库存储

### 3. 多租户隔离测试

**测试场景**:

- 租户1无法访问租户2的数据
- 租户2可以访问自己的数据
- 平台管理员可以访问所有租户
- 跨租户操作被拒绝

**验证点**:

- 权限验证正确
- 数据过滤正确
- 错误响应正确（403 Forbidden）
- 审计日志记录

### 4. 端到端测试

#### API测试

**测试场景**:

- 上下文构建API
- 记忆检索API
- 摘要生成API
- 未授权访问

**验证点**:

- HTTP状态码正确
- 响应格式正确
- JWT认证工作
- 错误处理完善

#### 完整流程测试

**测试场景**:

1. 创建消息
2. 构建上下文
3. 添加更多消息
4. 再次构建上下文
5. 存储记忆
6. 检索记忆

**验证点**:

- 流程顺利执行
- 数据一致性
- 状态正确更新

#### 性能测试

**测试场景**:

- API响应时间
- 并发请求处理
- 资源使用

**验证点**:

- 响应时间 < 1秒
- 并发请求成功率 > 90%
- 无内存泄漏

## 运行所有测试

### 快速测试（仅单元测试）

```bash
go test ./... -short -v
```

### 完整测试（包括集成测试）

```bash
go test ./... -v
```

### 测试覆盖率

```bash
# 生成覆盖率报告
go test ./... -cover -coverprofile=coverage.out

# 查看HTML报告
go tool cover -html=coverage.out

# 查看覆盖率统计
go tool cover -func=coverage.out
```

### 性能测试

```bash
# 运行性能基准测试
go test ./... -bench=. -benchmem

# 运行特定的性能测试
go test ./internal/service/... -bench=BenchmarkContextBuild -benchmem
```

## 测试最佳实践

### 1. 测试隔离

- 每个测试使用独立的数据库实例
- 测试之间不共享状态
- 使用 `t.Cleanup()` 确保资源清理

### 2. 测试数据

- 使用有意义的测试数据
- 避免硬编码ID
- 使用UUID生成唯一标识

### 3. 断言

- 使用 `testify/assert` 进行断言
- 提供清晰的错误消息
- 验证关键字段

### 4. Mock使用

- 单元测试使用mock隔离依赖
- 集成测试使用真实组件
- Mock行为应该符合实际

### 5. 错误处理

- 测试正常流程
- 测试错误流程
- 测试边界条件

## 持续集成

### GitHub Actions 配置示例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Run tests
        run: go test ./... -v -cover
      
      - name: Generate coverage report
        run: |
          go test ./... -coverprofile=coverage.out
          go tool cover -html=coverage.out -o coverage.html
      
      - name: Upload coverage
        uses: actions/upload-artifact@v2
        with:
          name: coverage
          path: coverage.html
```

## 故障排查

### 测试失败

1. 检查错误消息
2. 查看测试日志
3. 验证测试数据
4. 检查环境配置

### 性能问题

1. 使用性能分析工具
2. 检查数据库查询
3. 优化算法
4. 增加缓存

### 并发问题

1. 使用竞态检测：`go test -race`
2. 检查共享状态
3. 使用适当的同步机制

## 测试指标

### 目标

- 单元测试覆盖率 > 80%
- 集成测试覆盖核心流程
- 端到端测试覆盖主要API
- 所有测试通过率 100%

### 性能指标

- 上下文构建 P50 < 200ms
- 上下文构建 P95 < 500ms
- 记忆检索 P50 < 100ms
- 记忆检索 P95 < 300ms
- API响应时间 < 1秒

## 总结

本测试套件提供了全面的测试覆盖，包括：

1. **单元测试**: 验证单个组件的正确性
2. **集成测试**: 验证组件之间的交互
3. **端到端测试**: 验证完整的API流程
4. **多租户隔离测试**: 确保数据安全
5. **性能测试**: 验证性能要求

通过运行这些测试，可以确保系统的质量、可靠性和安全性。
