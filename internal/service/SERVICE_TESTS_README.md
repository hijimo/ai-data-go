# Service 层集成测试

## 概述

本目录包含 Service 层的集成测试，用于验证业务逻辑、权限控制和错误处理。

## 已创建的测试文件

### 1. context_service_test.go

测试上下文服务（ContextService）的核心功能：

**测试覆盖：**

- ✅ BuildContext - 构建会话上下文
  - 成功构建上下文
  - 包含长期记忆的上下文构建
  - 未授权访问（跨租户）
  - 会话不存在
  
- ✅ OptimizeContext - 优化上下文Token使用
  - 激进优化策略
  - 平衡优化策略
  
- ✅ GetContextConfig - 获取上下文配置
  - 成功获取配置
  
- ✅ UpdateContextConfig - 更新上下文配置
  - 成功更新配置
  - 无效参数验证
  
- ✅ 权限验证
  - 平台管理员可以访问所有会话
  - 租户管理员只能访问自己租户的会话

**Mock 对象：**

- MockSessionRepository
- MockMessageRepository
- MockMemoryRepository
- MockContextRepository
- MockSummaryRepository
- MockUserRepository
- MockVectorService
- MockTokenManager

### 2. memory_service_test.go

测试记忆管理服务（MemoryService）的核心功能：

**测试覆盖：**

- ✅ SearchMemories - 检索记忆
  - 成功检索记忆
  - 跨会话检索
  - 无效参数验证
  
- ✅ StoreMemory - 存储记忆
  - 成功存储记忆
  - 向量生成失败重试
  - 无效参数验证
  
- ✅ CleanupMemories - 清理记忆
  - 软删除成功
  - 硬删除（同时删除向量）
  - 无效策略验证
  
- ✅ UpdateMemoryAccess - 更新记忆访问统计
  - 成功更新
  - 未授权租户访问
  
- ✅ 权限验证
  - 平台管理员可以访问所有租户
  - 租户管理员只能访问自己租户

**Mock 对象：**

- MockMemoryRepository
- MockSessionRepository
- MockUserRepository
- MockVectorService
- MockQdrantClient
- MockTokenManager

### 3. summary_service_integration_test.go

测试摘要服务（SummaryService）的核心功能：

**测试覆盖：**

- ✅ GenerateSummary - 生成摘要
  - 成功生成完整摘要
  - 生成增量摘要
  
- ✅ CheckSummaryTrigger - 检查摘要触发条件
  - 消息数量达到阈值
  - Token使用量超限
  - 不应触发摘要
  
- ✅ EvaluateSummaryQuality - 评估摘要质量
  - 高质量摘要
  - 低质量摘要
  
- ✅ GetSummary - 获取摘要详情
  - 成功获取
  
- ✅ ListSummaries - 获取摘要列表
  - 成功获取列表
  
- ✅ 权限验证
  - 未授权访问
  - 平台管理员可以访问所有租户

**Mock 对象：**

- MockSummaryRepository
- MockMessageRepository
- MockContextRepository
- MockSessionRepository
- MockGenkitClient
- MockTokenManager

## 测试原则

### 1. 使用 Mock 对象

所有测试都使用 Mock 对象来隔离依赖，确保测试的独立性和可重复性。

### 2. 权限验证测试

每个服务都包含权限验证测试，确保：

- 平台管理员（system_admin）可以访问所有租户的数据
- 租户管理员（tenant_admin）只能访问自己租户的数据
- 跨租户访问被正确拒绝

### 3. 错误处理测试

测试各种错误场景：

- 参数验证失败
- 资源不存在
- 权限不足
- 外部服务失败

### 4. 业务逻辑测试

验证核心业务逻辑：

- 上下文构建和优化
- 记忆检索和存储
- 摘要生成和质量评估

## 运行测试

```bash
# 运行所有 Service 层测试
go test ./internal/service/... -v

# 运行特定服务的测试
go test ./internal/service -run TestContextService -v
go test ./internal/service -run TestMemoryService -v
go test ./internal/service/session -run TestSummaryService -v

# 运行特定测试用例
go test ./internal/service -run TestContextService_BuildContext_Success -v
```

## 测试覆盖率

```bash
# 生成测试覆盖率报告
go test ./internal/service/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 注意事项

### Mock 对象的维护

当接口定义发生变化时，需要同步更新 Mock 对象的实现。确保 Mock 对象实现了接口的所有方法。

### 测试数据的准备

测试使用辅助函数创建测试数据：

- `createTestContext()` - 创建带JWT声明的测试上下文
- `createTestSession()` - 创建测试会话
- `createTestUser()` - 创建测试用户
- `createTestMessages()` - 创建测试消息列表
- `createTestMemories()` - 创建测试记忆列表

### 异步操作的测试

某些服务方法包含异步操作（如更新访问统计），测试时需要注意：

- 使用 `time.Sleep()` 等待异步操作完成
- 或者使用 channel 进行同步
- 或者将异步操作改为同步以便测试

## 未来改进

1. **增加更多边界条件测试**
   - 大数据量测试
   - 并发访问测试
   - 性能测试

2. **集成真实数据库测试**
   - 使用 testcontainers 启动真实数据库
   - 测试复杂的数据库操作

3. **端到端测试**
   - 测试完整的业务流程
   - 测试多个服务之间的协作

4. **测试覆盖率提升**
   - 目标：达到 80% 以上的代码覆盖率
   - 补充缺失的测试用例

## 相关文档

- [Repository 层测试文档](../repository/README_TESTS.md)
- [多租户访问控制规范](../../.kiro/steering/multi-tenant-access-control.md)
- [API响应格式规范](../../.kiro/steering/api-response-format.md)
