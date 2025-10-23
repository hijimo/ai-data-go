# 实施任务列表

## 任务概述

本任务列表将需求和设计转化为具体的实施步骤，每个任务都是可独立执行的代码修改。

- [x] 1. 更新数据模型定义
  - 在 Tenant、User、ChatSession 模型中添加 `created_by_name` 字段
  - 添加适当的 GORM 标签和 JSON 标签
  - 添加中文注释说明字段用途
  - _需求: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 2. 扩展 JWT Claims 结构
  - 在 JWTClaims 结构体中添加 `DisplayName` 字段
  - 添加 JSON 标签 `json:"displayName"`
  - 添加字段注释
  - _需求: 4.1, 4.2, 4.3_

- [x] 3. 更新认证服务的令牌生成逻辑
  - 修改 Login 方法，在生成 JWT 时包含用户的 `DisplayName`
  - 修改 RefreshToken 方法，刷新令牌时重新获取最新的 `DisplayName`
  - 确保从数据库查询用户信息时包含 `display_name` 字段
  - _需求: 4.4, 4.5_

- [x] 4. 创建创建者信息提取辅助函数
  - 在 `internal/service/auth/helpers.go` 中创建 `GetCreatorInfoFromContext` 函数
  - 函数从 Context 中提取 JWT Claims
  - 解析 Subject 字段为用户ID（UUID）
  - 提取 DisplayName 字段
  - 返回用户ID指针和显示名称指针
  - 添加错误处理和日志记录
  - _需求: 3.1, 3.2_

- [x] 5. 更新租户服务的创建逻辑
- [x] 5.1 修改 TenantService.Create 方法
  - 使用 `GetCreatorInfoFromContext` 获取创建者信息
  - 设置 `created_by` 和 `created_by_name` 字段
  - 移除对请求参数中 `CreatedBy` 字段的处理
  - _需求: 3.3, 3.4, 3.5, 5.1, 5.2, 5.3_

- [x] 5.2 修改 TenantService.CreateWithAdmin 方法
  - 使用 `GetCreatorInfoFromContext` 获取创建者信息
  - 为租户和管理员用户都设置创建者信息
  - _需求: 3.3, 3.4, 3.5, 5.1, 5.2, 5.3_

- [x] 6. 更新用户服务的创建逻辑
  - 修改 UserService.Create 方法
  - 使用 `GetCreatorInfoFromContext` 获取创建者信息
  - 设置 `created_by` 和 `created_by_name` 字段
  - 移除对请求参数中 `CreatedBy` 字段的处理
  - _需求: 3.3, 3.4, 3.5, 5.4, 5.5, 5.6_

- [x] 7. 更新会话服务的创建逻辑
  - 修改 SessionService.CreateSession 方法
  - 导入 auth 包以使用 `GetCreatorInfoFromContext`
  - 使用辅助函数获取创建者信息
  - 设置 `created_by` 和 `created_by_name` 字段
  - 处理 ChatSession 的 `CreatedBy` 非指针类型
  - _需求: 3.3, 3.4, 3.5, 5.7, 5.8, 5.9_

- [x] 8. 创建数据库迁移脚本
- [x] 8.1 创建迁移文件
  - 在 `internal/db/migrations/` 目录创建 `add_created_by_name_migration.go`
  - 实现 Up 方法：为 tenants、users、chat_sessions 表添加 `created_by_name` 字段
  - 实现 Down 方法：删除添加的字段
  - 使用事务确保原子性
  - 使用 `IF NOT EXISTS` 和 `IF EXISTS` 确保幂等性
  - 添加字段注释
  - _需求: 1.1, 1.2, 1.3, 1.4, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.8_

- [x] 8.2 注册迁移
  - 在迁移管理器中注册新迁移
  - 确保迁移按正确顺序执行
  - _需求: 6.7_

- [x] 9. 更新请求结构体
  - 从 CreateTenantRequest 中移除 `CreatedBy` 字段（如果存在）
  - 从 CreateUserRequest 中移除 `CreatedBy` 字段（如果存在）
  - 确保请求验证不包含这些字段
  - _需求: 3.5_

- [ ]* 10. 编写单元测试
- [ ]* 10.1 测试 JWT Claims 扩展
  - 测试 JWTClaims 结构体包含 DisplayName 字段
  - 测试 JSON 序列化和反序列化
  - _需求: 4.1, 4.2, 4.3_

- [ ]* 10.2 测试辅助函数
  - 测试 GetCreatorInfoFromContext 正常情况
  - 测试 JWT Claims 不存在的情况
  - 测试用户ID解析失败的情况
  - 测试 DisplayName 为空的情况
  - _需求: 3.1, 3.2, 8.1, 8.2, 8.3_

- [ ]* 10.3 测试服务层创建逻辑
  - 测试租户创建时正确设置创建者信息
  - 测试用户创建时正确设置创建者信息
  - 测试会话创建时正确设置创建者信息
  - 测试忽略客户端传入的创建者信息
  - _需求: 3.3, 3.4, 3.5, 5.1-5.9, 8.4_

- [ ]* 10.4 测试数据库迁移
  - 测试迁移 Up 方法成功执行
  - 测试字段正确添加到数据库
  - 测试迁移 Down 方法成功回滚
  - 测试迁移的幂等性
  - _需求: 6.1-6.8_

- [ ]* 11. 编写集成测试
- [ ]* 11.1 端到端测试
  - 测试完整的创建流程（登录 -> 创建资源 -> 验证创建者信息）
  - 测试租户创建的端到端流程
  - 测试用户创建的端到端流程
  - 测试会话创建的端到端流程
  - _需求: 3.1-3.5, 5.1-5.9, 8.3_

- [ ]* 11.2 向后兼容性测试
  - 测试现有数据的 created_by_name 为 NULL
  - 测试 API 响应正确序列化 NULL 值
  - 测试现有功能不受影响
  - _需求: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ]* 12. 性能测试
  - 对比迁移前后的查询性能
  - 测试列表查询不需要 JOIN users 表
  - 测试 JWT 令牌大小变化
  - 验证性能提升符合预期
  - _需求: 无直接对应，但与设计目标相关_

- [ ] 13. 文档更新
  - 更新 API 文档，说明响应中包含 `createdByName` 字段
  - 更新数据模型文档
  - 更新迁移文档
  - 添加使用示例
  - _需求: 无直接对应，但是必要的维护工作_
