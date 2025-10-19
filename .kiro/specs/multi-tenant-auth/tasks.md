# 多租户用户管理与 JWT 身份认证实施计划

本文档定义了实现多租户用户管理与 JWT 身份认证系统的具体任务列表。每个任务都是可执行的代码实现步骤，按照依赖关系组织。

## 任务列表

- [x] 1. 创建数据模型和数据库迁移
  - 在 `internal/model` 目录创建认证相关的数据模型
  - 定义 Tenant、User、RefreshToken、AuthAudit 结构体
  - 添加 GORM 标签和 JSON 标签
  - 在 `internal/database/migrations` 创建认证表迁移脚本
  - 实现 SQL 表创建逻辑，包含完整的中文注释
  - _需求: 1.1, 2.1, 2.2, 11.1, 11.2, 11.3, 11.4, 11.5_

- [x] 2. 实现 Repository 层
- [x] 2.1 实现 TenantRepository
  - 在 `internal/repository` 创建 `tenant_repository.go`
  - 实现 Create、GetByID、GetByDomain、Update、Delete、List 方法
  - 实现租户隔离的查询逻辑
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 2.2 实现 UserRepository
  - 在 `internal/repository` 创建 `user_repository.go`
  - 实现 Create、GetByID、GetByEmail、Update、Delete、List、UpdateLastLogin 方法
  - 确保所有查询都包含 tenant_id 过滤
  - _需求: 2.1, 2.2, 2.3, 2.4_

- [x] 2.3 实现 RefreshTokenRepository
  - 在 `internal/repository` 创建 `refresh_token_repository.go`
  - 实现 Create、GetByTokenHash、Revoke、RevokeAllByUser、DeleteExpired 方法
  - 实现 token 轮换逻辑
  - _需求: 4.1, 4.2, 4.3, 4.5_

- [x] 2.4 实现 AuditRepository
  - 在 `internal/repository` 创建 `audit_repository.go`
  - 实现 Create 和 List 方法
  - 支持多条件过滤查询
  - _需求: 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 3. 实现密码安全工具
  - 在 `pkg` 目录创建 `crypto` 包
  - 实现 HashPassword 函数（使用 bcrypt，cost=12）
  - 实现 VerifyPassword 函数
  - 实现密码强度验证函数
  - _需求: 9.1, 9.2, 9.3, 9.4, 9.5_

- [x] 4. 实现 TokenService
- [x] 4.1 创建 JWT Claims 结构和 TokenService 接口
  - 在 `internal/model` 创建 `jwt.go`，定义 JWTClaims 结构
  - 在 `internal/service/auth` 创建 `token_service.go`
  - 定义 TokenService 接口
  - _需求: 12.1, 12.2, 12.3, 12.4, 12.5_

- [x] 4.2 实现 Access Token 生成和验证
  - 实现 GenerateAccessToken 方法
  - 实现 ValidateAccessToken 方法
  - 使用 golang-jwt/jwt v5 库
  - 设置 60 分钟过期时间
  - _需求: 3.2, 3.3, 8.2, 8.3_

- [x] 4.3 实现 Refresh Token 生成和验证
  - 实现 GenerateRefreshToken 方法（生成 UUID）
  - 实现 ValidateRefreshToken 方法
  - 实现 HashToken 方法（SHA256）
  - 实现 RevokeRefreshToken 方法
  - _需求: 3.3, 4.1, 4.2, 4.3, 4.4_

- [x] 5. 实现 AuthService
- [x] 5.1 创建 AuthService 接口和实现
  - 在 `internal/service/auth` 创建 `auth_service.go`
  - 定义 AuthService 接口
  - 创建 authService 结构体，注入依赖（UserRepository、TokenService、AuditService）
  - _需求: 2.1, 3.1, 4.1, 5.1_

- [x] 5.2 实现用户注册功能
  - 实现 Register 方法
  - 验证邮箱唯一性
  - 哈希密码
  - 创建用户记录
  - _需求: 2.1, 2.2, 9.1_

- [x] 5.3 实现用户登录功能
  - 实现 Login 方法
  - 验证租户、邮箱和密码
  - 生成 Access Token 和 Refresh Token
  - 更新最后登录时间
  - 记录登录审计日志
  - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 10.1, 10.2_

- [x] 5.4 实现 Token 刷新功能
  - 实现 RefreshToken 方法
  - 验证 Refresh Token
  - 生成新的 Access Token 和 Refresh Token
  - 撤销旧的 Refresh Token 并记录 replaced_by
  - 记录刷新审计日志
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5, 10.3_

- [x] 5.5 实现用户注销功能
  - 实现 Logout 方法
  - 撤销 Refresh Token
  - 记录注销审计日志
  - _需求: 5.1, 5.2, 5.3, 10.4_

- [x] 5.6 实现密码修改功能
  - 实现 ChangePassword 方法
  - 验证旧密码
  - 哈希新密码
  - 更新用户记录
  - 撤销所有 Refresh Token（强制重新登录）
  - _需求: 9.5_

- [x] 6. 实现 TenantService 和 UserService
- [x] 6.1 实现 TenantService
  - 在 `internal/service/auth` 创建 `tenant_service.go`
  - 实现 Create、Get、Update、Delete、List、GetByDomain 方法
  - 添加租户状态验证
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 6.2 实现 UserService
  - 在 `internal/service/auth` 创建 `user_service.go`
  - 实现 Create、Get、Update、Delete、List 方法
  - 确保所有操作都包含租户隔离
  - _需求: 2.1, 2.2, 2.3, 2.4_

- [x] 7. 实现中间件
- [x] 7.1 实现 TenantIdentifier 中间件
  - 在 `internal/api/middleware` 创建 `tenant.go`
  - 实现从请求头提取租户 ID 的逻辑
  - 实现从子域名提取租户的逻辑（可选）
  - 将租户 ID 注入请求上下文
  - 处理租户不存在或禁用的情况
  - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 7.2 实现 JWTAuth 中间件
  - 在 `internal/api/middleware` 创建 `jwt_auth.go`
  - 从 Authorization 头提取 Bearer Token
  - 验证 JWT 签名和过期时间
  - 提取 Claims 并注入上下文
  - 处理 Token 无效、过期等错误
  - _需求: 8.1, 8.2, 8.3, 8.4, 8.5_

- [x] 7.3 实现 RBACAuthorizer 中间件
  - 在 `internal/api/middleware` 创建 `rbac.go`
  - 从上下文提取用户角色
  - 验证是否具有所需角色
  - 返回权限不足错误
  - _需求: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 8. 实现 Handler 层
- [x] 8.1 实现 AuthHandler
  - 在 `internal/api/handler` 创建 `auth_handler.go`
  - 实现 HandleRegister 方法
  - 实现 HandleLogin 方法
  - 实现 HandleRefresh 方法
  - 实现 HandleLogout 方法
  - 实现 HandleChangePassword 方法
  - 实现 HandleMe 方法（获取当前用户信息）
  - 添加请求验证和错误处理
  - 使用项目标准的响应格式（ResponseData）
  - _需求: 2.1, 3.1, 4.1, 5.1, 9.5_

- [x] 8.2 实现 TenantHandler
  - 在 `internal/api/handler` 创建 `tenant_handler.go`
  - 实现 HandleCreate、HandleGet、HandleUpdate、HandleDelete、HandleList 方法
  - 添加管理员权限验证
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 8.3 实现 UserHandler
  - 在 `internal/api/handler` 创建 `user_handler.go`
  - 实现 HandleCreate、HandleGet、HandleUpdate、HandleDelete、HandleList 方法
  - 添加租户管理员权限验证
  - _需求: 2.1, 2.2, 2.3, 2.4_

- [x] 9. 配置路由和依赖注入
- [x] 9.1 更新配置结构
  - 在 `internal/config/config.go` 添加 AuthConfig 结构
  - 添加 JWT、密码、登录安全等配置项
  - 实现配置验证逻辑
  - 更新 .env.example 文件
  - _需求: 3.2, 4.3, 9.1_

- [x] 9.2 注册认证路由
  - 在 `internal/api/router.go` 或创建新的路由文件
  - 注册认证相关路由（/api/v1/auth/*）
  - 注册租户管理路由（/api/v1/tenants/*）
  - 注册用户管理路由（/api/v1/users/*）
  - 应用中间件链（TenantIdentifier -> JWTAuth -> RBACAuthorizer）
  - _需求: 6.1, 8.1_

- [x] 9.3 实现依赖注入
  - 在 `cmd/server/main.go` 初始化认证相关服务
  - 创建 Repository 实例
  - 创建 Service 实例
  - 创建 Handler 实例
  - 配置中间件
  - _需求: 所有需求_

- [x] 10. 添加 Swagger 文档
  - 为所有认证 API 添加 Swagger 注释
  - 定义请求和响应模型
  - 添加认证说明（Bearer Token）
  - 更新 Swagger 文档
  - _需求: 所有 API 相关需求_

- [x] 11. 实现数据库清理任务
  - 创建定时任务清理过期的 Refresh Token
  - 实现审计日志归档逻辑（可选）
  - 在应用启动时启动清理任务
  - _需求: 4.4_

- [x] 12. 创建初始化脚本
  - 创建数据库迁移脚本
  - 创建初始租户和管理员用户的脚本
  - 添加使用说明文档
  - _需求: 1.1, 2.1_

## 可选任务（增强功能）

以下任务为可选的增强功能，可在核心功能完成后实施：

- [x] 13. 实现登录失败限制
  - 记录登录失败次数
  - 实现账户临时锁定机制
  - 添加解锁功能
  - _需求: 3.5_

- [x] 14. 实现邮箱验证功能
  - 生成验证令牌
  - 发送验证邮件
  - 实现验证端点
  - _需求: 2.2_

- [x] 15. 添加审计日志查询 API
  - 实现审计日志查询 Handler
  - 支持多条件过滤
  - 支持分页
  - _需求: 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 16. 实现 Token 黑名单机制
  - 使用 Redis 存储已撤销的 Access Token
  - 在 JWT 验证中检查黑名单
  - 实现黑名单清理逻辑
  - _需求: 5.4_

- [x] 17. 添加性能监控
  - 添加认证相关的 metrics
  - 实现慢查询日志
  - 添加告警规则
  - _需求: 所有需求_

## 实施说明

### 执行顺序

任务按照依赖关系组织，建议按照编号顺序执行。每个任务完成后应：

1. 运行相关测试确保功能正常
2. 更新文档
3. 提交代码

### 测试要求

- 每个 Repository 和 Service 方法都应有对应的单元测试
- 关键流程（登录、刷新、注销）应有集成测试
- 中间件应有独立的测试用例

### 代码规范

- 遵循项目现有的代码风格
- 所有公开函数和方法添加中文注释
- 错误处理使用项目统一的错误格式
- API 响应使用项目标准的 ResponseData 格式

### 安全注意事项

- 永远不要在日志中记录密码或 Token
- 所有数据库查询必须包含租户隔离
- 验证所有用户输入
- 使用参数化查询防止 SQL 注入
