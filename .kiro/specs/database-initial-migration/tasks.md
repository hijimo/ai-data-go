# 实施计划

- [x] 1. 清理现有迁移历史
  - 备份并归档现有的分散迁移文件
  - 清理数据库中的旧迁移记录
  - _需求: 1.1, 1.2, 1.3_

- [x] 1.1 备份现有迁移文件
  - 将 `auth_migration.go` 和 `session_migration.go` 移动到 `internal/database/migrations/backup/` 目录
  - 创建 `backup/README.md` 说明备份原因和日期
  - _需求: 1.1, 1.2_

- [x] 1.2 清理迁移记录表
  - 检查数据库中是否存在旧的迁移记录表（如 `schema_migrations`）
  - 如果存在，编写脚本清理这些记录表
  - 确保不影响现有业务数据
  - _需求: 1.3_

- [x] 2. 创建初始迁移文件
  - 在 `internal/database/migrations/` 目录下创建 `initial_migration.go` 文件
  - 实现 `InitialMigration` 结构体和构造函数
  - 实现 `Name()` 方法返回迁移名称
  - _需求: 2.1_

- [x] 2.1 实现 Up 方法 - 启用 UUID 扩展
  - 编写启用 PostgreSQL UUID 扩展的 SQL（pgcrypto 或 uuid-ossp）
  - 添加错误处理逻辑
  - _需求: 2.5_

- [x] 2.2 实现 Up 方法 - 创建 tenants 表
  - 编写创建 tenants 表的 SQL，包含所有字段定义
  - 添加主键约束和唯一索引
  - 添加表和列的中文注释
  - _需求: 2.2, 2.6, 2.8_

- [x] 2.3 实现 Up 方法 - 创建 users 表
  - 编写创建 users 表的 SQL，包含所有字段定义
  - 添加主键约束、外键约束和复合唯一索引
  - 添加表和列的中文注释
  - _需求: 2.2, 2.7, 2.8_

- [x] 2.4 实现 Up 方法 - 创建 refresh_tokens 表
  - 编写创建 refresh_tokens 表的 SQL
  - 添加主键、外键约束和索引
  - 添加表和列的中文注释
  - _需求: 2.2, 2.6, 2.7, 2.8_

- [x] 2.5 实现 Up 方法 - 创建 email_verification_tokens 表
  - 编写创建 email_verification_tokens 表的 SQL
  - 添加主键、外键约束和索引
  - 添加表和列的中文注释
  - _需求: 2.2, 2.6, 2.7, 2.8_

- [x] 2.6 实现 Up 方法 - 创建 auth_audit 表
  - 编写创建 auth_audit 表的 SQL
  - 添加主键和索引（不需要外键约束）
  - 添加表和列的中文注释
  - _需求: 2.2, 2.6, 2.8_

- [x] 2.7 实现 Up 方法 - 创建 chat_sessions 表
  - 编写创建 chat_sessions 表的 SQL
  - 添加主键、外键约束和索引
  - 添加表和列的中文注释
  - _需求: 2.3, 2.6, 2.7, 2.8_

- [x] 2.8 实现 Up 方法 - 创建 chat_messages 表
  - 编写创建 chat_messages 表的 SQL
  - 添加主键、外键约束和索引
  - 添加表和列的中文注释
  - _需求: 2.3, 2.6, 2.7, 2.8_

- [x] 2.9 实现 Up 方法 - 创建 chat_summaries 表
  - 编写创建 chat_summaries 表的 SQL
  - 添加主键、外键约束和索引
  - 添加表和列的中文注释
  - _需求: 2.3, 2.6, 2.7, 2.8_

- [x] 2.10 实现 Down 方法
  - 编写按逆序删除所有表的 SQL
  - 使用 CASCADE 确保依赖关系正确处理
  - 添加错误处理逻辑
  - _需求: 2.4_

- [x] 3. 更新 ORM 模型定义
  - 确保所有模型文件中的字段类型和 GORM 标签与数据库结构一致
  - _需求: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_

- [x] 3.1 更新 Tenant 模型
  - 在 `internal/model/auth.go` 中更新 Tenant 结构体
  - 确保主键类型为 UUID，使用 `gen_random_uuid()` 默认值
  - 添加所有必要的 GORM 标签和 JSON 标签
  - _需求: 4.1_

- [x] 3.2 更新 User 模型
  - 在 `internal/model/auth.go` 中更新 User 结构体
  - 确保主键和外键类型为 UUID
  - 添加正确的外键关系和约束标签
  - 确保复合唯一索引标签正确
  - _需求: 4.1, 4.3, 4.6_

- [x] 3.3 更新 RefreshToken 模型
  - 在 `internal/model/auth.go` 中更新 RefreshToken 结构体
  - 确保主键和外键类型为 UUID
  - 添加正确的外键关系和约束标签
  - _需求: 4.1, 4.3_

- [x] 3.4 更新 EmailVerificationToken 模型
  - 在 `internal/model/auth.go` 中更新 EmailVerificationToken 结构体
  - 确保主键和外键类型为 UUID
  - 添加正确的外键关系和约束标签
  - _需求: 4.1, 4.3_

- [x] 3.5 更新 AuthAudit 模型
  - 在 `internal/model/auth.go` 中更新 AuthAudit 结构体
  - 确保主键类型为 UUID
  - 确保 JSONB 字段使用 `datatypes.JSON` 类型
  - _需求: 4.1, 4.4_

- [x] 3.6 更新 ChatSession 模型
  - 在 `internal/model/session.go` 中更新 ChatSession 结构体
  - 确保主键和外键类型为 UUID
  - 添加正确的外键关系和约束标签
  - _需求: 4.2, 4.3_

- [x] 3.7 更新 ChatMessage 模型
  - 在 `internal/model/session.go` 中更新 ChatMessage 结构体
  - 确保主键和外键类型为 UUID
  - 确保 JSONB 字段使用 `datatypes.JSON` 类型
  - 添加正确的外键关系和约束标签
  - _需求: 4.2, 4.3, 4.4_

- [x] 3.8 更新 ChatSummary 模型
  - 在 `internal/model/session.go` 中更新 ChatSummary 结构体
  - 确保主键和外键类型为 UUID
  - 添加正确的外键关系和约束标签
  - _需求: 4.2, 4.3_

- [x] 4. 更新迁移管理器
  - 修改 `internal/database/migrations/migration_manager.go` 以支持初始迁移
  - _需求: 3.1, 3.2, 3.3, 3.4_

- [x] 4.1 添加 RegisterInitialMigration 方法
  - 在 MigrationManager 中实现注册初始迁移的方法
  - 确保初始迁移在所有迁移之前执行
  - _需求: 3.1, 3.2_

- [x] 4.2 创建 RunInitialMigration 函数
  - 实现独立的函数用于单独执行初始迁移
  - 添加详细的错误处理和日志记录
  - _需求: 3.3, 3.4_

- [x] 4.3 更新 RunAllMigrations 方法
  - 确保 RunAllMigrations 首先执行初始迁移
  - 添加迁移顺序验证逻辑
  - _需求: 3.2_

- [x] 5. 创建迁移脚本
  - 在 `scripts/` 目录下创建便捷的迁移执行脚本
  - _需求: 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 5.1 创建 init_migration.go 脚本
  - 编写脚本连接数据库并执行初始迁移
  - 添加命令行参数支持（如数据库连接字符串）
  - 添加详细的日志输出
  - _需求: 5.1, 5.3_

- [x] 5.2 创建 reset_db.go 脚本
  - 编写脚本执行 Down 方法清空数据库
  - 然后执行 Up 方法重建表结构
  - 添加确认提示防止误操作
  - _需求: 5.2, 5.4, 5.5_

- [x] 6. 更新数据库配置
  - 确保配置文件包含所有必要的数据库设置
  - _需求: 6.1, 6.2, 6.3_

- [x] 6.1 更新 .env.example 文件
  - 添加 DATABASE_URL 配置示例
  - 添加 PostgreSQL 连接参数说明
  - 添加 UUID 扩展相关说明
  - _需求: 6.1, 6.2, 6.3_

- [x] 6.2 验证配置加载逻辑
  - 检查 `internal/config/config.go` 是否正确加载数据库配置
  - 确保所有必要的配置项都有默认值或验证
  - _需求: 6.1, 6.2_

- [x] 7. 集成到应用启动流程
  - 在应用启动时自动执行迁移
  - _需求: 9.1, 9.2, 9.3, 9.4_

- [x] 7.1 更新 main.go 集成迁移调用
  - 在 `cmd/server/main.go` 中添加迁移执行逻辑
  - 在数据库连接后、服务启动前执行迁移
  - _需求: 9.1_

- [x] 7.2 添加迁移检查逻辑
  - 实现检查是否需要执行迁移的逻辑
  - 添加迁移状态记录机制
  - _需求: 9.2_

- [x] 7.3 实现迁移失败处理
  - 当迁移失败时阻止应用启动
  - 输出清晰的错误信息和解决建议
  - _需求: 9.3_

- [x] 7.4 验证 Repository 层兼容性
  - 检查所有 Repository 层代码是否与更新后的模型兼容
  - 修复任何类型不匹配或字段缺失的问题
  - _需求: 9.4_

- [ ]* 8. 编写测试
  - 为初始迁移和相关功能编写测试
  - _需求: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6_

- [ ]* 8.1 编写 InitialMigration 单元测试
  - 在 `internal/database/migrations/` 目录下创建 `initial_migration_test.go`
  - 测试 Up 方法成功创建所有表
  - 测试 Down 方法成功删除所有表
  - 测试表结构、索引和约束的正确性
  - _需求: 7.1_

- [ ]* 8.2 编写 MigrationManager 单元测试
  - 测试迁移注册功能
  - 测试迁移执行顺序
  - 测试错误处理逻辑
  - _需求: 7.1_

- [ ]* 8.3 编写集成测试 - 完整迁移流程
  - 在空数据库上执行完整迁移
  - 验证所有表、索引和约束都已正确创建
  - _需求: 7.2, 7.3, 7.4_

- [ ]* 8.4 编写集成测试 - 回滚测试
  - 测试 Up 后执行 Down 的完整流程
  - 验证所有表都已删除
  - 测试再次执行 Up 的可重复性
  - _需求: 7.5, 7.6_

- [ ]* 8.5 编写集成测试 - 数据完整性
  - 测试外键约束是否生效
  - 测试级联删除功能
  - 测试唯一约束
  - _需求: 7.2, 7.3, 7.4_

- [ ] 9. 更新文档
  - 更新项目文档以反映新的迁移机制
  - _需求: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 9.1 更新数据库迁移指南
  - 更新 `docs/database-migration-guide.md` 文档
  - 添加初始迁移的详细说明
  - 添加数据库结构版本信息
  - _需求: 8.1, 8.2, 8.3_

- [ ] 9.2 添加迁移脚本使用示例
  - 在文档中添加 init_migration.go 的使用示例
  - 添加 reset_db.go 的使用示例和注意事项
  - _需求: 8.4_

- [ ] 9.3 添加常见问题和解决方案
  - 整理迁移过程中可能遇到的问题
  - 提供详细的解决方案和最佳实践
  - _需求: 8.5_
