# Repository 层集成测试

本目录包含 Repository 层的集成测试，用于验证数据访问层的功能正确性。

## 测试文件

- `memory_repository_test.go` - 记忆数据访问层测试
- `context_repository_test.go` - 上下文配置数据访问层测试
- `summary_repository_test.go` - 摘要数据访问层测试

## 测试覆盖范围

### MemoryRepository 测试

- ✅ 创建记忆（Create）
- ✅ 根据ID获取记忆（GetByID）
- ✅ 租户隔离验证
- ✅ 向量检索（会话内）
- ✅ 跨会话向量检索
- ✅ 更新访问统计
- ✅ 按策略删除记忆（过期、低质量）
- ✅ 获取过期记忆
- ✅ 批量创建记忆
- ✅ 获取会话的所有记忆
- ✅ 软删除和硬删除

### ContextRepository 测试

- ✅ 创建上下文配置（Create）
- ✅ 根据会话ID获取配置（GetBySessionID）
- ✅ 更新上下文配置（Update）
- ✅ 租户隔离验证
- ✅ 获取最新摘要（GetLatestSummary）
- ✅ 更新Token使用统计（UpdateTokenUsage）
- ✅ 软删除过滤
- ✅ 并发更新测试

### SummaryRepository 测试

- ✅ 创建摘要（Create）
- ✅ 根据ID获取摘要（GetByID）
- ✅ 租户隔离验证
- ✅ 获取会话最新摘要（GetLatestBySessionID）
- ✅ 获取会话摘要列表（ListBySessionID）
- ✅ 更新摘要（Update）
- ✅ 软删除和硬删除
- ✅ 根据类型获取摘要（GetByType）
- ✅ 统计会话摘要数量（CountBySessionID）
- ✅ 关键主题数组测试

## 测试数据库配置

### 前置条件

1. 安装 PostgreSQL 数据库（版本 13+）
2. 创建测试数据库：

```bash
createdb genkit_test
```

3. 配置测试数据库连接（可选）：

```bash
export TEST_DATABASE_URL="host=localhost port=5432 user=postgres password=postgres dbname=genkit_test sslmode=disable"
```

### 默认配置

如果未设置环境变量，测试将使用以下默认配置：

- Host: localhost
- Port: 5432
- User: postgres
- Password: postgres
- Database: genkit_test
- SSL Mode: disable

## 运行测试

### 运行所有 Repository 测试

```bash
go test -v ./internal/repository/...
```

### 运行特定测试文件

```bash
# 测试 MemoryRepository
go test -v ./internal/repository/memory_repository_test.go ./internal/repository/memory_repository.go ./internal/repository/memory_repository_impl.go

# 测试 ContextRepository
go test -v ./internal/repository/context_repository_test.go ./internal/repository/context_repository.go ./internal/repository/context_repository_impl.go

# 测试 SummaryRepository
go test -v ./internal/repository/summary_repository_test.go ./internal/repository/summary_repository.go ./internal/repository/summary_repository_impl.go
```

### 运行特定测试用例

```bash
# 运行特定测试函数
go test -v -run TestMemoryRepository_Create ./internal/repository/

# 运行匹配模式的测试
go test -v -run TestMemoryRepository_.*TenantIsolation ./internal/repository/
```

### 查看测试覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/repository/

# 查看覆盖率详情
go tool cover -html=coverage.out
```

## 测试数据清理

每个测试用例执行后会自动清理测试数据，使用以下 SQL 语句：

```sql
TRUNCATE TABLE conversation_memories CASCADE;
TRUNCATE TABLE conversation_contexts CASCADE;
TRUNCATE TABLE conversation_summaries CASCADE;
```

## 测试最佳实践

### 1. 租户隔离测试

所有涉及租户数据的操作都必须测试租户隔离：

```go
// 创建租户1的数据
tenant1ID := uuid.New()
data1 := createData(tenant1ID)

// 尝试用租户2的ID访问租户1的数据（应该失败）
tenant2ID := uuid.New()
_, err := repo.GetByID(ctx, tenant2ID, data1.ID)
assert.Error(t, err, "应该无法访问其他租户的数据")
```

### 2. 软删除过滤测试

所有查询操作都必须测试软删除过滤：

```go
// 创建数据
data := createData()

// 软删除
repo.SoftDelete(ctx, tenantID, data.ID)

// 验证无法查询到已删除的数据
_, err := repo.GetByID(ctx, tenantID, data.ID)
assert.Error(t, err, "不应该查询到已删除的数据")
```

### 3. 并发测试

对于涉及计数器或统计的操作，应该测试并发安全性：

```go
// 并发更新
done := make(chan bool)
for i := 0; i < 10; i++ {
    go func() {
        err := repo.UpdateCounter(ctx, id)
        assert.NoError(t, err)
        done <- true
    }()
}

// 等待所有goroutine完成
for i := 0; i < 10; i++ {
    <-done
}

// 验证最终结果
result := repo.GetCounter(ctx, id)
assert.Equal(t, 10, result)
```

## 依赖项

测试使用以下依赖库：

- `github.com/stretchr/testify` - 断言和测试工具
- `github.com/google/uuid` - UUID 生成
- `gorm.io/gorm` - ORM 框架
- `gorm.io/driver/postgres` - PostgreSQL 驱动

安装依赖：

```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
go get github.com/google/uuid
go get gorm.io/gorm
go get gorm.io/driver/postgres
```

## 故障排查

### 问题：无法连接到测试数据库

**解决方案**：

1. 确认 PostgreSQL 服务正在运行
2. 检查数据库连接配置是否正确
3. 确认测试数据库已创建

### 问题：表不存在错误

**解决方案**：

测试会自动创建表结构，如果遇到表不存在的错误：

1. 检查模型定义是否正确
2. 确认 AutoMigrate 是否成功执行
3. 手动运行迁移脚本

### 问题：测试数据未清理

**解决方案**：

如果测试数据未被清理，可以手动清理：

```sql
-- 连接到测试数据库
psql -d genkit_test

-- 清理所有表
TRUNCATE TABLE conversation_memories CASCADE;
TRUNCATE TABLE conversation_contexts CASCADE;
TRUNCATE TABLE conversation_summaries CASCADE;
```

## 持续集成

在 CI/CD 流水线中运行测试：

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: genkit_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: go test -v -coverprofile=coverage.out ./internal/repository/
        env:
          TEST_DATABASE_URL: "host=localhost port=5432 user=postgres password=postgres dbname=genkit_test sslmode=disable"
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## 注意事项

1. **测试隔离**：每个测试用例应该独立运行，不依赖其他测试的状态
2. **数据清理**：测试结束后必须清理测试数据，避免影响其他测试
3. **租户隔离**：所有测试都必须验证租户隔离功能
4. **错误处理**：测试应该覆盖正常流程和异常流程
5. **性能考虑**：避免在测试中创建大量数据，影响测试执行速度

## 贡献指南

添加新的测试用例时，请遵循以下规范：

1. 测试函数命名：`Test<Repository>_<Method>_<Scenario>`
2. 使用 `setupTestDB` 和 `cleanupTestDB` 管理测试数据库
3. 使用 `require` 进行前置条件断言
4. 使用 `assert` 进行结果验证
5. 添加清晰的注释说明测试目的
6. 确保测试可以独立运行

## 参考资料

- [GORM 文档](https://gorm.io/docs/)
- [Testify 文档](https://github.com/stretchr/testify)
- [Go 测试最佳实践](https://golang.org/doc/tutorial/add-a-test)
