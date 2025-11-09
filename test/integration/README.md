# API 端到端集成测试

本目录包含 Genkit 会话管理模块的 API 端到端集成测试。

## 测试文件

- `context_api_test.go` - 上下文管理 API 测试
- `memory_api_test.go` - 记忆管理 API 测试
- `summary_api_test.go` - 摘要管理 API 测试

## 测试覆盖范围

### 功能测试

- ✅ 完整的 API 调用流程
- ✅ 请求参数验证
- ✅ 响应格式验证
- ✅ 业务逻辑验证

### 安全测试

- ✅ 认证和授权验证
- ✅ 租户隔离验证
- ✅ 跨租户访问控制
- ✅ 平台管理员权限验证

### 错误处理测试

- ✅ 参数验证错误
- ✅ 资源不存在错误
- ✅ 权限不足错误
- ✅ 未认证错误

## 前置条件

### 1. 数据库配置

确保测试数据库已配置并可访问：

```bash
# 设置环境变量
export DATABASE_URL="postgresql://user:password@localhost:5432/test_db?sslmode=disable"
```

### 2. 依赖安装

```bash
# 安装测试依赖
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
```

### 3. 数据库迁移

运行数据库迁移以创建必要的表结构：

```bash
go run scripts/migrate.go
```

## 运行测试

### 运行所有集成测试

```bash
cd test/integration
go test -v ./...
```

### 运行特定测试文件

```bash
# 上下文 API 测试
go test -v context_api_test.go

# 记忆 API 测试
go test -v memory_api_test.go

# 摘要 API 测试
go test -v summary_api_test.go
```

### 运行特定测试用例

```bash
# 运行特定测试函数
go test -v -run TestBuildContext_Success

# 运行匹配模式的测试
go test -v -run TestBuildContext
```

### 生成测试覆盖率报告

```bash
# 生成覆盖率报告
go test -v -coverprofile=coverage.out ./...

# 查看覆盖率
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

## 测试结构

每个测试文件都遵循相同的结构：

```go
// 1. 测试套件结构
type TestXXXAPI struct {
    db           *gorm.DB
    router       http.Handler
    log          logger.Logger
    systemAdmin  *testUser
    tenantAdmin1 *testUser
    tenantAdmin2 *testUser
    tenant1      *model.Tenant
    tenant2      *model.Tenant
    // ... 其他测试数据
}

// 2. 设置和清理函数
func setupXXXAPITest(t *testing.T) *TestXXXAPI
func teardownXXXAPITest(t *testing.T, test *TestXXXAPI)

// 3. 测试用例
func TestXXX_Success(t *testing.T)
func TestXXX_Unauthorized(t *testing.T)
func TestXXX_CrossTenantAccess(t *testing.T)
func TestXXX_ValidationError(t *testing.T)

// 4. 辅助函数
func (test *TestXXXAPI) makeRequest(...)
func cleanupTestData(...)
func createTestXXX(...)
```

## 测试场景

### 1. 上下文管理 API (context_api_test.go)

#### 构建上下文 (POST /api/v1/contexts/build)

- ✅ 成功构建上下文
- ✅ 未认证访问被拒绝
- ✅ 跨租户访问被拒绝
- ✅ 平台管理员可访问所有租户
- ✅ 参数验证错误处理

#### 获取上下文配置 (GET /api/v1/contexts/:sessionId)

- ✅ 成功获取配置
- ✅ 资源不存在返回404
- ✅ 跨租户访问被拒绝

#### 更新上下文配置 (PUT /api/v1/contexts/:sessionId)

- ✅ 成功更新配置
- ✅ 跨租户更新被拒绝
- ✅ 参数验证错误处理

### 2. 记忆管理 API (memory_api_test.go)

#### 检索记忆 (POST /api/v1/memories/search)

- ✅ 成功检索记忆
- ✅ 未认证访问被拒绝
- ✅ 跨租户访问被拒绝
- ✅ 平台管理员可访问所有租户
- ✅ 参数验证错误处理

#### 存储记忆 (POST /api/v1/memories)

- ✅ 成功存储记忆
- ✅ 跨租户存储被拒绝
- ✅ 参数验证错误处理

#### 清理记忆 (POST /api/v1/memories/cleanup)

- ✅ 成功清理记忆（预览模式）
- ✅ 参数验证错误处理
- ✅ 支持多种清理策略

#### 获取记忆详情 (GET /api/v1/memories/:id)

- ⚠️ 功能未完全实现（返回503）

### 3. 摘要管理 API (summary_api_test.go)

#### 生成摘要 (POST /api/v1/summaries)

- ✅ 成功生成摘要
- ✅ 未认证访问被拒绝
- ✅ 跨租户访问被拒绝
- ✅ 平台管理员可访问所有租户
- ✅ 参数验证错误处理

#### 获取摘要详情 (GET /api/v1/summaries/:id)

- ✅ 成功获取摘要
- ✅ 资源不存在返回404
- ✅ 跨租户访问被拒绝

#### 获取摘要列表 (GET /api/v1/sessions/:sessionId/summaries)

- ✅ 成功获取摘要列表
- ✅ 跨租户访问被拒绝

#### 检查触发条件 (POST /api/v1/sessions/:sessionId/summaries/check-trigger)

- ✅ 成功检查触发条件
- ✅ 跨租户访问被拒绝

## 租户隔离测试

所有测试都验证了以下租户隔离规则：

1. **租户管理员**：只能访问自己租户的资源
2. **平台管理员**：可以访问所有租户的资源
3. **跨租户访问**：租户管理员尝试访问其他租户资源时返回 403 Forbidden
4. **未认证访问**：未提供认证令牌时返回 401 Unauthorized

## 测试数据管理

### 测试数据创建

每个测试套件在 `setup` 函数中创建以下测试数据：

- 2个测试租户（tenant1, tenant2）
- 3个测试用户（systemAdmin, tenantAdmin1, tenantAdmin2）
- 2个测试会话（session1, session2）
- 相关的业务数据（上下文配置、记忆、摘要等）

### 测试数据清理

每个测试套件在 `teardown` 函数中清理所有测试数据，确保测试之间的隔离。

## 注意事项

### 1. 测试隔离

- 每个测试用例都应该是独立的
- 使用 `setup` 和 `teardown` 函数确保测试环境的一致性
- 避免测试之间的数据污染

### 2. 数据库事务

- 测试使用真实的数据库连接
- 确保测试数据库与生产数据库分离
- 建议使用专门的测试数据库

### 3. 性能考虑

- 集成测试比单元测试慢
- 可以使用 `-short` 标志跳过集成测试：

  ```bash
  go test -short ./...
  ```

### 4. 并发测试

- 默认情况下，Go 测试是并发运行的
- 如果测试之间有依赖，使用 `-p 1` 标志串行运行：

  ```bash
  go test -p 1 -v ./...
  ```

## 持续集成

在 CI/CD 流水线中运行测试：

```yaml
# .github/workflows/test.yml 示例
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: test_db
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Run migrations
        run: go run scripts/migrate.go
        env:
          DATABASE_URL: postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable
      
      - name: Run integration tests
        run: go test -v ./test/integration/...
        env:
          DATABASE_URL: postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable
```

## 故障排查

### 测试失败

1. **数据库连接失败**
   - 检查 `DATABASE_URL` 环境变量
   - 确保数据库服务正在运行
   - 验证数据库凭据

2. **表不存在**
   - 运行数据库迁移
   - 检查迁移脚本是否正确执行

3. **权限错误**
   - 检查数据库用户权限
   - 确保测试用户有创建/删除表的权限

4. **测试超时**
   - 增加测试超时时间：`go test -timeout 30m`
   - 检查数据库性能
   - 优化测试数据量

### 调试技巧

```bash
# 启用详细日志
go test -v -run TestBuildContext_Success

# 查看测试输出
go test -v ./... 2>&1 | tee test.log

# 使用 delve 调试器
dlv test -- -test.run TestBuildContext_Success
```

## 扩展测试

### 添加新的测试用例

1. 在相应的测试文件中添加新的测试函数
2. 遵循现有的命名约定：`TestXXX_Scenario`
3. 使用 `setup` 和 `teardown` 函数管理测试数据
4. 验证所有相关的业务逻辑和安全规则

### 添加新的测试文件

1. 创建新的测试文件：`xxx_api_test.go`
2. 定义测试套件结构
3. 实现 `setup` 和 `teardown` 函数
4. 编写测试用例
5. 更新本 README 文档

## 参考资料

- [Go Testing 文档](https://golang.org/pkg/testing/)
- [Testify 断言库](https://github.com/stretchr/testify)
- [GORM 文档](https://gorm.io/docs/)
- [HTTP 测试最佳实践](https://golang.org/pkg/net/http/httptest/)
