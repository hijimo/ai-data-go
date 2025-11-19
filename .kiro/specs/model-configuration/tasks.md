# 模型配置模块实现计划

## 任务列表

- [x] 1. 创建数据模型和数据库迁移
  - 在 `internal/model/provider.go` 中定义 ModelConfiguration 结构体
  - 定义 ModelProvider 枚举常量和验证函数
  - 在 `internal/database/migrations/` 中创建数据库迁移文件
  - 创建 model_configurations 表，包含所有必需字段和索引
  - _需求: 1.1, 1.4, 2.4, 6.2, 8.1, 9.1, 9.2, 9.3_

- [x] 2. 实现API密钥加密服务
  - 在 `internal/service/` 中创建 encryption_service.go
  - 实现 EncryptionService 接口（EncryptAPIKey, DecryptAPIKey, MaskAPIKey）
  - 使用 AES-256-GCM 算法加密API密钥
  - 实现密钥脱敏逻辑（显示前4位和后4位）
  - 从环境变量读取加密密钥配置
  - _需求: 8.1, 8.2, 8.4_

- [x] 3. 实现Provider仓储层
  - 在 `internal/repository/` 中创建 provider_repository.go
  - 实现 ProviderRepository 接口
  - 实现 Create 方法（创建模型配置）
  - 实现 FindByID 方法（按ID查询）
  - 实现 FindByTenant 方法（按租户查询，支持分页）
  - 实现 Update 方法（更新模型配置）
  - 实现 UpdateStatus 方法（更新启用/禁用状态）
  - 实现 SoftDelete 方法（逻辑删除）
  - 实现 FindAvailableByTenant 方法（查询可用模型列表）
  - 所有查询自动过滤 is_deleted=true 的记录
  - _需求: 2.4, 5.4, 6.2, 6.5, 7.2_

- [ ] 4. 实现Provider服务层 - 基础CRUD
  - 在 `internal/service/` 中创建 provider_service.go
  - 实现 ProviderService 接口
  - 实现 Create 方法，包含租户权限验证和API密钥加密
  - 实现 Get 方法，包含租户权限验证和密钥脱敏
  - 实现 List 方法，根据角色自动过滤租户数据
  - 实现 Update 方法，包含租户权限验证和字段更新
  - 实现 Delete 方法，包含租户权限验证和软删除
  - 所有方法记录审计日志
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.5, 3.1, 3.2, 3.4, 6.1, 6.3, 6.4, 8.3, 8.5, 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 5. 实现Provider服务层 - 状态管理和验证
  - 实现 UpdateStatus 方法（启用/禁用模型配置）
  - 实现 Validate 方法（验证模型配置连接）
  - 实现 validateOpenAI 方法（验证OpenAI配置）
  - 实现 validateAnthropic 方法（验证Anthropic配置）
  - 实现 validateGoogleGenAI 方法（验证Google GenAI配置）
  - 实现 validateAzureOpenAI 方法（验证Azure OpenAI配置）
  - 实现 validateBianlian 方法（验证Bianlian配置）
  - 实现 validateCustomOpenAI 方法（验证自定义OpenAI配置）
  - 实现 ListAvailable 方法（查询可用模型列表）
  - 验证请求设置30秒超时
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5, 5.1, 5.2, 5.3, 5.4, 5.5, 7.1, 7.3, 7.4, 7.5_

- [ ] 6. 实现API Handler层
  - 在 `internal/api/handler/` 中创建或更新 provider_handler.go
  - 实现 HandleCreate（创建模型配置）
  - 实现 HandleList（查询模型配置列表）
  - 实现 HandleGet（查询单个模型配置）
  - 实现 HandleUpdate（更新模型配置）
  - 实现 HandleUpdateStatus（更新状态）
  - 实现 HandleDelete（删除模型配置）
  - 实现 HandleValidate（验证模型配置）
  - 实现 HandleListAvailable（查询可用模型列表）
  - 所有响应使用标准格式（ResponseData 或 ResponsePaginationData）
  - 正确处理错误并返回适当的HTTP状态码
  - _需求: 所有需求的API层实现_

- [ ] 7. 配置路由和中间件
  - 在 `internal/api/routes/` 中配置Provider相关路由
  - POST /api/v1/providers - 创建（RequireTenantAdmin）
  - GET /api/v1/providers - 列表（RequireTenantAdmin）
  - GET /api/v1/providers/:id - 详情（RequireTenantAdmin）
  - PUT /api/v1/providers/:id - 更新（RequireTenantAdmin）
  - PATCH /api/v1/providers/:id/status - 状态（RequireTenantAdmin）
  - DELETE /api/v1/providers/:id - 删除（RequireTenantAdmin）
  - POST /api/v1/providers/:id/validate - 验证（RequireTenantAdmin）
  - GET /api/v1/providers/available - 可用列表（JWT认证）
  - 确保所有路由都有适当的认证和授权中间件
  - _需求: 所有需求的路由配置_

- [ ] 8. 添加环境变量配置
  - 在 `.env.example` 中添加 ENCRYPTION_SECRET_KEY 配置项
  - 在 `.env.example` 中添加 PROVIDER_VALIDATION_TIMEOUT 配置项
  - 在 `internal/config/config.go` 中添加相关配置结构
  - 实现配置加载和验证逻辑
  - _需求: 8.1_

- [ ] 9. 运行数据库迁移
  - 执行迁移脚本创建 model_configurations 表
  - 验证表结构和索引是否正确创建
  - 验证外键约束是否正确设置
  - _需求: 1.4, 2.4, 6.2_

- [ ] 10. 集成测试和验证
  - 测试平台管理员创建模型配置（指定租户ID）
  - 测试租户管理员创建模型配置（自动使用当前租户）
  - 测试租户管理员尝试访问其他租户配置（应返回403）
  - 测试平台管理员跨租户访问（应成功）
  - 测试模型配置验证功能（各种提供商）
  - 测试启用/禁用功能
  - 测试软删除功能
  - 测试可用模型列表（仅返回已启用且未删除的配置）
  - 测试API密钥脱敏（返回时仅显示部分内容）
  - 验证审计日志是否正确记录
  - _需求: 所有需求的集成测试_
