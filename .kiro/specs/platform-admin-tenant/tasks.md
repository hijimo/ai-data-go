# 平台管理员租户系统实施任务

## 任务列表

- [x] 1. 数据库迁移和模型扩展
  - 创建数据库迁移脚本，为 tenants 表添加 type 字段，支持租户类型区分
  - 添加索引和约束确保只能有一个平台租户
  - 更新 Go 数据模型，添加租户类型常量和角色常量
  - _需求: 2.1, 2.2, 2.3, 11.1, 11.2, 11.3_

- [x] 1.1 创建数据库迁移文件
  - 在 `internal/database/migrations/` 目录创建新的迁移文件 `platform_tenant_migration.go`
  - 实现 Up 方法：添加 type 字段、创建索引、添加约束
  - 实现 Down 方法：回滚所有更改
  - _需求: 11.1, 11.2, 11.3_

- [x] 1.2 更新 Tenant 模型
  - 在 `internal/model/auth.go` 中为 Tenant 结构体添加 Type 字段
  - 添加租户类型常量（TenantTypeSystem, TenantTypeBusiness）
  - 添加角色常量（RoleSystemAdmin, RoleTenantAdmin, RoleUser）
  - _需求: 2.1, 2.2, 11.1, 11.5_

- [x] 1.3 更新租户仓储层
  - 在 `internal/repository/tenant_repository.go` 添加 GetByType 方法
  - 更新 Create 方法以支持租户类型验证
  - 添加检查平台租户唯一性的逻辑
  - _需求: 2.2, 2.4_

- [ ]* 1.4 编写迁移测试
  - 测试迁移脚本的 Up 和 Down 方法
  - 验证索引和约束正确创建
  - 测试平台租户唯一性约束
  - _需求: 11.2, 11.3_

- [x] 2. 实现系统初始化服务
  - 创建 BootstrapService 接口和实现，负责系统首次启动时的自举
  - 实现配置加载逻辑，从环境变量读取平台管理员信息
  - 实现平台租户和平台管理员的创建逻辑
  - 集成到应用启动流程，在 main.go 中调用初始化服务
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5, 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 2.1 创建 Bootstrap 配置结构
  - 在 `internal/config/config.go` 中添加 BootstrapConfig 结构
  - 实现从环境变量加载配置的逻辑
  - 添加配置验证（邮箱格式、密码强度等）
  - 设置默认值（邮箱、密码、显示名称）
  - _需求: 10.1, 10.2, 10.3, 10.4_

- [x] 2.2 创建 BootstrapService 接口和实现
  - 在 `internal/service/auth/` 创建 `bootstrap_service.go`
  - 定义 BootstrapService 接口（Initialize, IsInitialized 方法）
  - 实现 bootstrapService 结构体
  - 注入 TenantRepository 和 UserRepository 依赖
  - _需求: 1.1, 1.5_

- [x] 2.3 实现初始化逻辑
  - 实现 IsInitialized 方法：检查是否存在 type='system' 的租户
  - 实现 Initialize 方法：创建平台租户和平台管理员
  - 生成或使用配置的管理员密码
  - 设置管理员角色为 ["system_admin"]，is_admin 为 true
  - 记录初始化日志（包含邮箱和初始密码）
  - _需求: 1.1, 1.2, 1.3, 1.4, 10.5_

- [x] 2.4 集成到应用启动流程
  - 在 `cmd/server/main.go` 中创建 BootstrapService 实例
  - 在数据库连接成功后调用 Initialize 方法
  - 处理初始化错误和日志记录
  - _需求: 1.5_

- [ ]* 2.5 编写 Bootstrap 服务测试
  - 测试首次初始化成功创建平台租户和管理员
  - 测试重复初始化不会创建重复数据
  - 测试配置参数正确应用
  - 测试初始化失败时的错误处理
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 3. 扩展租户服务
  - 扩展 TenantService 接口，添加创建租户并自动生成管理员的方法
  - 实现 CreateWithAdmin 方法，创建租户时自动生成租户管理员账户
  - 实现租户类型过滤和查询方法
  - 实现启用/禁用租户的方法
  - 添加平台租户保护逻辑，防止删除平台租户
  - _需求: 2.1, 2.2, 2.3, 2.5, 4.1, 4.2, 4.3, 4.4, 4.5, 5.1, 5.2, 5.3, 5.4, 5.5, 9.1, 9.2, 9.3, 9.4, 9.5_

- [x] 3.1 扩展 TenantService 接口
  - 在 `internal/service/auth/tenant_service.go` 添加新方法定义
  - 添加 CreateWithAdmin 方法签名
  - 添加 GetByType 方法签名
  - 添加 EnableTenant 和 DisableTenant 方法签名
  - _需求: 2.1, 4.1, 5.1, 9.1_

- [x] 3.2 创建请求和响应结构
  - 在 CreateTenantRequest 中添加 Type 字段
  - 创建 CreateTenantWithAdminRequest 结构体
  - 创建 CreateTenantWithAdminResponse 结构体
  - 添加字段验证标签
  - _需求: 5.1, 5.2, 5.3_

- [x] 3.3 实现 CreateWithAdmin 方法
  - 验证请求参数（租户名称、域名等）
  - 创建业务租户（type = "tenant"）
  - 生成租户管理员邮箱（使用请求中的或默认 "admin@{domain}"）
  - 生成 16 位随机强密码
  - 创建租户管理员用户（roles = ["tenant_admin"], is_admin = true）
  - 返回租户信息和管理员初始密码
  - _需求: 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 3.4 实现租户类型相关方法
  - 实现 GetByType 方法：根据类型过滤租户
  - 更新 Create 方法：验证平台租户唯一性
  - 更新 List 方法：支持按类型过滤
  - _需求: 2.1, 2.2, 2.3, 2.4_

- [x] 3.5 实现租户状态管理方法
  - 实现 EnableTenant 方法：设置 status = true
  - 实现 DisableTenant 方法：设置 status = false
  - 更新 Update 方法：支持状态更新
  - _需求: 9.1, 9.2, 9.3, 9.4, 9.5_

- [x] 3.6 添加平台租户保护逻辑
  - 在 Delete 方法中检查租户类型
  - 如果是平台租户（type = "system"），拒绝删除并返回错误
  - 在 Update 方法中防止修改平台租户的类型
  - _需求: 2.5, 4.5_

- [ ]* 3.7 编写租户服务测试
  - 测试 CreateWithAdmin 成功创建租户和管理员
  - 测试不允许创建多个平台租户
  - 测试启用/禁用租户
  - 测试不允许删除平台租户
  - 测试租户类型过滤
  - _需求: 2.2, 2.5, 4.5, 5.1, 5.2, 5.3, 5.4, 5.5, 9.1, 9.2_

- [x] 4. 实现密码生成工具
  - 创建安全密码生成函数，用于自动生成租户管理员密码
  - 实现密码强度验证函数
  - 确保生成的密码包含大小写字母、数字和特殊字符
  - _需求: 5.3_

- [x] 4.1 创建密码生成函数
  - 在 `pkg/crypto/password.go` 添加 GenerateSecurePassword 函数
  - 实现随机密码生成逻辑（至少 16 字符）
  - 确保包含大小写字母、数字和特殊字符
  - 使用加密安全的随机数生成器
  - _需求: 5.3_

- [x] 4.2 实现密码强度验证
  - 添加 ValidatePasswordStrength 函数
  - 检查密码长度（至少 8 字符）
  - 检查是否包含必要的字符类型
  - 返回详细的验证错误信息
  - _需求: 5.3_

- [ ]* 4.3 编写密码工具测试
  - 测试生成的密码符合强度要求
  - 测试密码长度正确
  - 测试密码包含所有必要字符类型
  - 测试密码验证函数
  - _需求: 5.3_

- [x] 5. 扩展 JWT Token 和认证服务
  - 扩展 JWTClaims 结构，添加 Roles 字段
  - 更新 Token 生成逻辑，在 Token 中包含用户角色信息
  - 更新登录流程，验证租户和用户状态
  - 实现禁用用户时撤销 Token 的逻辑
  - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 6.5, 8.1, 8.2, 8.3, 8.4, 8.5, 9.2, 9.3_

- [x] 5.1 扩展 JWTClaims 结构
  - 在 `internal/model/jwt.go` 中为 JWTClaims 添加 Roles 字段
  - 更新 Claims 的 JSON 标签
  - 确保向后兼容现有 Token
  - _需求: 3.1, 8.2, 8.3_

- [x] 5.2 更新 Token 生成逻辑
  - 在 `internal/service/auth/token_service.go` 更新 GenerateAccessToken 方法
  - 从用户模型读取 Roles 字段
  - 将角色信息包含在 JWT Token 中
  - 确保 Token 包含所有必要的用户信息
  - _需求: 8.2, 8.3_

- [x] 5.3 更新登录验证逻辑
  - 在 `internal/service/auth/auth_service.go` 更新 Login 方法
  - 添加租户状态检查（tenant.status）
  - 添加用户激活状态检查（user.is_active）
  - 如果租户被禁用，拒绝登录并返回错误
  - 如果用户被禁用，拒绝登录并返回错误
  - _需求: 8.1, 9.2, 9.3_

- [x] 5.4 实现用户禁用时撤销 Token
  - 在用户服务的 Update 方法中检测 is_active 状态变化
  - 当用户被禁用时，调用 TokenBlacklistService 撤销所有 Refresh Token
  - 确保用户无法使用现有 Token 访问系统
  - _需求: 6.5, 9.3_

- [ ]* 5.5 编写 Token 和认证测试
  - 测试 Token 包含正确的角色信息
  - 测试租户被禁用时无法登录
  - 测试用户被禁用时无法登录
  - 测试禁用用户时 Token 被撤销
  - _需求: 8.1, 8.2, 8.3, 9.2, 9.3, 6.5_

- [x] 6. 实现 RBAC 权限验证中间件
  - 创建基于角色的访问控制中间件
  - 实现 RequireSystemAdmin 中间件，验证平台管理员权限
  - 实现 RequireTenantAdmin 中间件，验证租户管理员权限
  - 实现 RequireTenantAccess 中间件，验证租户访问权限
  - 实现权限不足时的错误处理
  - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 6.1 创建 RBAC 中间件文件
  - 在 `internal/api/middleware/` 创建 `rbac.go` 文件
  - 定义中间件函数签名
  - 创建辅助函数用于从 Token 提取角色信息
  - _需求: 7.1, 7.2, 7.3_

- [x] 6.2 实现 RequireSystemAdmin 中间件
  - 从 JWT Token 中获取用户角色
  - 检查角色是否包含 "system_admin"
  - 如果不包含，返回 403 Forbidden 错误
  - 如果包含，继续处理请求
  - _需求: 7.1, 7.4_

- [x] 6.3 实现 RequireTenantAdmin 中间件
  - 从 JWT Token 中获取用户角色
  - 检查角色是否包含 "system_admin" 或 "tenant_admin"
  - 如果都不包含，返回 403 Forbidden 错误
  - 如果包含，继续处理请求
  - _需求: 7.2, 7.4_

- [x] 6.4 实现 RequireTenantAccess 中间件
  - 从 JWT Token 中获取用户的 tenant_id 和角色
  - 从请求路径或参数中获取目标 tenant_id
  - 如果用户是 system_admin，允许访问
  - 如果用户的 tenant_id 与目标 tenant_id 匹配，允许访问
  - 否则返回 403 Forbidden 错误
  - _需求: 7.3, 7.4_

- [x] 6.5 实现错误处理
  - 创建统一的权限错误响应格式
  - 处理未认证情况（返回 401）
  - 处理权限不足情况（返回 403）
  - 记录权限验证失败的审计日志
  - _需求: 7.4, 7.5_

- [ ]* 6.6 编写 RBAC 中间件测试
  - 测试 RequireSystemAdmin 正确验证平台管理员权限
  - 测试 RequireTenantAdmin 正确验证租户管理员权限
  - 测试 RequireTenantAccess 正确验证租户访问权限
  - 测试权限不足时返回 403
  - 测试未认证时返回 401
  - _需求: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 7. 实现平台管理 API
  - 创建平台管理 API Handler，处理租户管理相关请求
  - 实现创建租户（带管理员）接口
  - 实现列出所有租户接口
  - 实现启用/禁用租户接口
  - 实现删除租户接口
  - 配置路由并应用 RequireSystemAdmin 中间件
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5, 9.1, 9.2, 9.4, 9.5_

- [x] 7.1 创建平台管理 Handler
  - 在 `internal/api/handler/` 创建 `platform_handler.go`
  - 定义 PlatformHandler 结构体
  - 注入 TenantService 依赖
  - 创建 NewPlatformHandler 构造函数
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 7.2 实现创建租户接口
  - 实现 CreateTenant 方法（POST /api/v1/platform/tenants）
  - 解析和验证请求体（CreateTenantWithAdminRequest）
  - 调用 TenantService.CreateWithAdmin 方法
  - 返回租户信息和管理员初始密码
  - 使用标准响应格式（ResponseData）
  - _需求: 4.1, 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 7.3 实现列出租户接口
  - 实现 ListTenants 方法（GET /api/v1/platform/tenants）
  - 支持分页参数（page, pageSize）
  - 支持租户类型过滤（type 参数）
  - 返回分页数据（ResponsePaginationData）
  - _需求: 4.2, 2.4_

- [x] 7.4 实现启用/禁用租户接口
  - 实现 UpdateTenantStatus 方法（PATCH /api/v1/platform/tenants/:id/status）
  - 解析请求体（status 字段）
  - 调用 TenantService.EnableTenant 或 DisableTenant
  - 返回更新后的租户信息
  - _需求: 4.3, 4.4, 9.1, 9.4_

- [x] 7.5 实现删除租户接口
  - 实现 DeleteTenant 方法（DELETE /api/v1/platform/tenants/:id）
  - 验证租户 ID
  - 调用 TenantService.Delete 方法
  - 处理平台租户保护错误
  - 返回成功响应
  - _需求: 4.5_

- [x] 7.6 配置平台管理路由
  - 在 `internal/api/routes/` 创建或更新路由配置
  - 注册平台管理 API 路由
  - 应用 RequireSystemAdmin 中间件到所有平台管理路由
  - 应用 JWT 认证中间件
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5_

- [ ]* 7.7 编写平台管理 API 测试
  - 测试创建租户成功返回租户和管理员信息
  - 测试列出租户支持分页和过滤
  - 测试启用/禁用租户
  - 测试删除租户
  - 测试非平台管理员无法访问这些接口
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 8. 实现租户管理 API
  - 创建租户管理 API Handler，处理租户内用户管理
  - 实现创建用户接口
  - 实现列出租户用户接口
  - 实现启用/禁用用户接口
  - 实现删除用户接口
  - 配置路由并应用 RequireTenantAdmin 和 RequireTenantAccess 中间件
  - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 8.1 扩展用户 Handler
  - 在 `internal/api/handler/user_handler.go` 中添加新方法
  - 或创建新的 `tenant_user_handler.go` 文件
  - 确保 Handler 支持租户隔离
  - _需求: 6.1, 6.2, 6.3, 6.4_

- [x] 8.2 实现创建用户接口
  - 实现 CreateUser 方法（POST /api/v1/tenants/:tenantId/users）
  - 解析和验证请求体（CreateUserRequest）
  - 验证请求者有权限在该租户创建用户
  - 调用 UserService.Create 方法
  - 返回创建的用户信息
  - _需求: 6.1_

- [x] 8.3 实现列出用户接口
  - 实现 ListUsers 方法（GET /api/v1/tenants/:tenantId/users）
  - 支持分页参数
  - 验证请求者有权限查看该租户用户
  - 调用 UserService.List 方法
  - 返回分页用户列表
  - _需求: 6.2_

- [x] 8.4 实现启用/禁用用户接口
  - 实现 UpdateUserStatus 方法（PATCH /api/v1/tenants/:tenantId/users/:userId/status）
  - 解析请求体（isActive 字段）
  - 验证请求者有权限修改该用户
  - 调用 UserService.Update 方法
  - 如果禁用用户，触发 Token 撤销
  - 返回更新后的用户信息
  - _需求: 6.4, 6.5_

- [x] 8.5 实现删除用户接口
  - 实现 DeleteUser 方法（DELETE /api/v1/tenants/:tenantId/users/:userId）
  - 验证请求者有权限删除该用户
  - 调用 UserService.Delete 方法
  - 返回成功响应
  - _需求: 6.5_

- [x] 8.6 配置租户管理路由
  - 在路由配置中注册租户管理 API
  - 应用 RequireTenantAdmin 中间件
  - 应用 RequireTenantAccess 中间件
  - 应用 JWT 认证中间件
  - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ]* 8.7 编写租户管理 API 测试
  - 测试租户管理员可以创建用户
  - 测试租户管理员只能查看本租户用户
  - 测试租户管理员可以启用/禁用用户
  - 测试租户管理员无法访问其他租户数据
  - 测试平台管理员可以访问所有租户数据
  - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 9. 扩展审计日志功能
  - 扩展审计日志服务，记录平台管理和租户管理操作
  - 在租户创建、删除、启用、禁用时记录审计日志
  - 在用户创建、删除、启用、禁用时记录审计日志
  - 确保审计日志包含操作者、操作类型、目标对象和时间戳
  - _需求: 12.1, 12.2, 12.3, 12.4, 12.5_

- [x] 9.1 扩展审计日志事件类型
  - 在审计日志服务中添加新的事件类型常量
  - 添加 tenant_created, tenant_deleted, tenant_enabled, tenant_disabled
  - 添加 user_created, user_deleted, user_enabled, user_disabled
  - _需求: 12.1, 12.2, 12.3, 12.4, 12.5_

- [x] 9.2 在租户服务中集成审计日志
  - 在 CreateWithAdmin 方法中记录 tenant_created 事件
  - 在 Delete 方法中记录 tenant_deleted 事件
  - 在 EnableTenant 方法中记录 tenant_enabled 事件
  - 在 DisableTenant 方法中记录 tenant_disabled 事件
  - 包含操作者 ID、租户 ID 和相关元数据
  - _需求: 12.1, 12.2, 12.3_

- [x] 9.3 在用户服务中集成审计日志
  - 在 Create 方法中记录 user_created 事件
  - 在 Delete 方法中记录 user_deleted 事件
  - 在 Update 方法中检测状态变化并记录相应事件
  - 包含操作者 ID、用户 ID、租户 ID 和相关元数据
  - _需求: 12.4, 12.5_

- [ ]* 9.4 编写审计日志测试
  - 测试租户操作正确记录审计日志
  - 测试用户操作正确记录审计日志
  - 验证日志包含必要的信息
  - 测试日志查询功能
  - _需求: 12.1, 12.2, 12.3, 12.4, 12.5_

- [x] 10. 更新 Swagger 文档
  - 为新的 API 接口添加 Swagger 注释
  - 更新请求和响应模型的文档
  - 添加权限要求说明
  - 生成更新后的 Swagger 文档
  - _需求: 所有 API 相关需求_

- [x] 10.1 为平台管理 API 添加 Swagger 注释
  - 在 PlatformHandler 的方法中添加 Swagger 注释
  - 描述请求参数、请求体、响应格式
  - 标注需要的权限（system_admin）
  - 添加示例请求和响应
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 10.2 为租户管理 API 添加 Swagger 注释
  - 在用户管理 Handler 的方法中添加 Swagger 注释
  - 描述请求参数、请求体、响应格式
  - 标注需要的权限（tenant_admin 或 system_admin）
  - 添加示例请求和响应
  - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 10.3 更新数据模型文档
  - 为 Tenant 模型的 Type 字段添加文档
  - 为 User 模型的 Roles 字段添加文档
  - 添加角色常量的说明
  - 更新 JWT Token 结构的文档
  - _需求: 2.1, 3.1, 8.2_

- [x] 10.4 生成 Swagger 文档
  - 运行 swag init 命令生成文档
  - 验证文档生成正确
  - 测试 Swagger UI 显示正常
  - _需求: 所有 API 相关需求_

- [ ] 11. 集成测试和端到端测试
  - 编写系统初始化流程的集成测试
  - 编写租户生命周期的集成测试
  - 编写权限验证的集成测试
  - 编写端到端场景测试
  - _需求: 所有需求_

- [ ] 11.1 编写系统初始化集成测试
  - 测试应用启动时自动创建平台租户和管理员
  - 测试可以使用平台管理员登录
  - 测试平台管理员拥有正确的权限
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5_

- [ ] 11.2 编写租户生命周期集成测试
  - 测试平台管理员创建业务租户
  - 测试租户管理员自动创建
  - 测试使用租户管理员登录
  - 测试租户管理员创建普通用户
  - 测试禁用租户后用户无法登录
  - 测试启用租户后用户可以登录
  - 测试删除租户
  - _需求: 4.1, 5.1, 5.2, 5.3, 5.4, 5.5, 6.1, 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 11.3 编写权限验证集成测试
  - 测试创建两个业务租户
  - 测试租户 A 的管理员无法访问租户 B 的数据
  - 测试平台管理员可以访问所有租户的数据
  - 测试普通用户只能访问自己的数据
  - _需求: 6.5, 7.1, 7.2, 7.3, 7.4_

- [ ] 11.4 编写端到端场景测试
  - 测试新系统部署场景
  - 测试租户管理场景
  - 测试权限边界场景
  - _需求: 所有需求_

- [ ] 12. 文档和部署准备
  - 编写部署文档，说明环境变量配置和初始化流程
  - 编写运维手册，说明监控指标和告警规则
  - 更新 README 文档
  - 准备迁移脚本和回滚方案
  - _需求: 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 12.1 编写部署文档
  - 创建 `docs/PLATFORM_ADMIN_DEPLOYMENT.md`
  - 说明环境变量配置
  - 说明数据库迁移步骤
  - 说明首次部署流程
  - 说明升级部署流程
  - _需求: 10.1, 10.2, 10.3, 10.4_

- [ ] 12.2 编写运维手册
  - 创建 `docs/PLATFORM_ADMIN_OPERATIONS.md`
  - 说明监控指标和告警规则
  - 说明日志管理
  - 说明备份和恢复流程
  - 说明常见问题排查
  - _需求: 10.5_

- [ ] 12.3 更新 README 文档
  - 在主 README 中添加平台管理员租户系统的说明
  - 添加快速开始指南
  - 添加 API 文档链接
  - _需求: 所有需求_

- [ ] 12.4 准备迁移和回滚方案
  - 验证迁移脚本的 Up 和 Down 方法
  - 准备数据备份脚本
  - 准备回滚步骤文档
  - 测试回滚流程
  - _需求: 11.1, 11.2, 11.3_
