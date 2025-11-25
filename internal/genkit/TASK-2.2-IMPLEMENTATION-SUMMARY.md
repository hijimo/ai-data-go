# TASK-2.2 实现总结：getOrInitGenkit() 方法

## 任务概述

实现 `getOrInitGenkit()` 方法，根据租户ID和模型名称动态获取或初始化 Genkit 实例。

## 实现内容

### 1. 核心方法实现

在 `internal/genkit/client.go` 中实现了 `getOrInitGenkit()` 方法，包含以下功能：

#### 方法签名

```go
func (c *client) getOrInitGenkit(ctx context.Context, tenantID, modelName string) (*genkit.Genkit, *GenkitConfig, error)
```

#### 核心功能

1. **缓存机制**
   - 使用 `tenantID_modelName` 作为缓存键
   - 使用读写锁（`sync.RWMutex`）保证并发安全
   - 首次访问时初始化实例，后续访问直接从缓存获取

2. **配置查询**
   - 从 `ModelConfigurationRepository` 查询租户的模型配置
   - 验证租户ID的有效性
   - 检查模型是否启用

3. **配置解析**
   - 从 `QueryParams` JSON 字段解析 `GenkitConfig`
   - 设置模型名称等基本字段
   - 根据提供商类型验证配置完整性

4. **插件初始化**
   - 根据 `ModelProvider` 字段选择对应的插件
   - 当前支持 `googlegenai` 提供商
   - 为 `azureopenai` 和 `bianlian` 预留扩展点

5. **实例缓存**
   - 初始化成功后将实例存入缓存
   - 使用写锁保护缓存写入操作

### 2. 辅助函数

实现了 `parseUUID()` 辅助函数用于解析 UUID 字符串：

```go
func parseUUID(uuidStr string) (uuid.UUID, error)
```

### 3. Bug 修复

修复了 `ModelConfigurationRepository.GetByTenantAndModel()` 方法中的字段名错误：

- 将 `model_name` 改为 `name`，与 `ModelConfiguration` 结构体字段保持一致

### 4. 单元测试

在 `internal/genkit/client_dynamic_test.go` 中实现了全面的单元测试：

#### 测试用例

1. **TestGetOrInitGenkit_Success** - 测试成功获取或初始化实例
2. **TestGetOrInitGenkit_CachedInstance** - 测试从缓存获取实例
3. **TestGetOrInitGenkit_ModelDisabled** - 测试模型已禁用的情况
4. **TestGetOrInitGenkit_InvalidTenantID** - 测试无效的租户ID
5. **TestGetOrInitGenkit_ConfigNotFound** - 测试配置不存在的情况
6. **TestGetOrInitGenkit_InvalidConfig** - 测试无效配置的情况
7. **TestGetOrInitGenkit_UnsupportedProvider** - 测试不支持的提供商
8. **TestGetOrInitGenkit_NoRepository** - 测试未注入仓储的情况
9. **TestGetOrInitGenkit_InvalidJSON** - 测试无效的 JSON 配置
10. **TestParseUUID** - 测试 UUID 解析函数

#### Mock 实现

实现了 `MockModelConfigurationRepository` 用于单元测试，模拟了所有仓储接口方法。

## 测试结果

所有测试用例均通过：

```
=== RUN   TestGetOrInitGenkit_Success
--- PASS: TestGetOrInitGenkit_Success (0.00s)
=== RUN   TestGetOrInitGenkit_CachedInstance
--- PASS: TestGetOrInitGenkit_CachedInstance (0.00s)
=== RUN   TestGetOrInitGenkit_ModelDisabled
--- PASS: TestGetOrInitGenkit_ModelDisabled (0.00s)
=== RUN   TestGetOrInitGenkit_InvalidTenantID
--- PASS: TestGetOrInitGenkit_InvalidTenantID (0.00s)
=== RUN   TestGetOrInitGenkit_ConfigNotFound
--- PASS: TestGetOrInitGenkit_ConfigNotFound (0.00s)
=== RUN   TestGetOrInitGenkit_InvalidConfig
--- PASS: TestGetOrInitGenkit_InvalidConfig (0.00s)
=== RUN   TestGetOrInitGenkit_UnsupportedProvider
--- PASS: TestGetOrInitGenkit_UnsupportedProvider (0.00s)
=== RUN   TestGetOrInitGenkit_NoRepository
--- PASS: TestGetOrInitGenkit_NoRepository (0.00s)
=== RUN   TestGetOrInitGenkit_InvalidJSON
--- PASS: TestGetOrInitGenkit_InvalidJSON (0.00s)
PASS
ok      genkit-ai-service/internal/genkit       0.820s
```

## 实现特点

### 1. 并发安全

- 使用 `sync.RWMutex` 保护实例缓存
- 读操作使用读锁，写操作使用写锁
- 避免了竞态条件

### 2. 错误处理

- 详细的错误信息，便于调试
- 区分不同类型的错误（配置错误、验证错误、初始化错误等）
- 使用 `fmt.Errorf` 包装错误，保留错误链

### 3. 性能优化

- 实例缓存避免重复初始化
- 懒加载机制，只在需要时初始化
- 读写锁支持并发读取

### 4. 可扩展性

- 为 Azure OpenAI 和百炼预留了扩展点
- 插件化设计，易于添加新的提供商
- 配置验证逻辑独立，便于维护

## 依赖关系

### 新增依赖

- `encoding/json` - 用于解析配置 JSON
- `github.com/google/uuid` - 用于 UUID 处理

### 依赖的接口

- `repository.ModelConfigurationRepository` - 模型配置仓储接口
- `GenkitConfig` - 配置结构体（在 config.go 中定义）

## 后续任务

根据任务列表，接下来需要：

1. **TASK-2.3**: 扩展 Generate 和 GenerateStream 方法，支持租户ID和模型名称参数
2. **TASK-3.x**: 实现 Azure OpenAI 插件集成
3. **TASK-4.x**: 实现百炼插件集成

## 注意事项

1. **配置字段映射**
   - `ModelConfiguration.Name` 对应模型配置的名称（用于查询）
   - `ModelConfiguration.Model` 对应实际的模型标识（如 gemini-1.5-pro）
   - `GenkitConfig.Model` 从 `ModelConfiguration.Model` 复制

2. **提供商类型**
   - 当前只实现了 `googlegenai` 提供商
   - 其他提供商返回"暂未实现"错误

3. **缓存键格式**
   - 格式：`{tenantID}_{modelName}`
   - 确保租户间的模型实例隔离

## 文件清单

### 修改的文件

1. `internal/genkit/client.go`
   - 添加 `getOrInitGenkit()` 方法
   - 添加 `parseUUID()` 辅助函数
   - 添加必要的 import

2. `internal/repository/model_configuration_repository.go`
   - 修复 `GetByTenantAndModel()` 方法中的字段名

### 新增的文件

1. `internal/genkit/client_dynamic_test.go`
   - 完整的单元测试套件
   - Mock 仓储实现

2. `internal/genkit/TASK-2.2-IMPLEMENTATION-SUMMARY.md`
   - 本实现总结文档

## 验收标准检查

- [x] 实现 `getOrInitGenkit()` 方法（根据租户ID和模型名称）
- [x] 实现 Genkit 实例缓存机制（key: tenantID_modelName）
- [x] 添加并发安全的读写锁
- [x] 实现配置解析逻辑
- [x] 实现插件动态创建逻辑
- [x] 保持向后兼容性
- [x] 编写单元测试

所有验收标准均已满足！
