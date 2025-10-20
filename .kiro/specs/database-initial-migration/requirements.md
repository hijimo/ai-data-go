# 需求文档

## 简介

本项目需要将现有的数据库结构整合为一个统一的初始迁移，并确保所有ORM模型定义与数据库结构完全一致。当前系统存在分散的迁移文件（认证迁移和会话迁移），需要将它们合并为一个初始迁移基线，以便于后续的版本管理和数据库维护。

## 术语表

- **System**: 指代数据库迁移管理系统
- **InitialMigration**: 初始迁移，包含所有现有表结构的基线迁移
- **MigrationManager**: 迁移管理器，负责注册和执行数据库迁移
- **ORM模型**: 使用GORM定义的Go语言数据模型
- **PostgreSQL**: 项目使用的关系型数据库
- **UUID扩展**: PostgreSQL的gen_random_uuid()函数所需的扩展

## 需求

### 需求 1：清理现有迁移历史

**用户故事：** 作为数据库管理员，我希望能够清理现有的分散迁移历史，以便建立统一的迁移基线

#### 验收标准

1. 当执行迁移清理时，System应该提供备份现有迁移文件的功能
2. 当清理迁移历史时，System应该保留原有的auth_migration.go和session_migration.go作为参考
3. 当清理完成时，System应该确保数据库中不存在旧的迁移记录表

### 需求 2：创建统一初始迁移

**用户故事：** 作为开发者，我希望有一个统一的初始迁移文件，以便在新环境中快速建立完整的数据库结构

#### 验收标准

1. System应该创建initial_migration.go文件在internal/database/migrations目录下
2. InitialMigration应该包含所有认证相关表的创建逻辑（tenants, users, refresh_tokens, auth_audit, email_verification_tokens）
3. InitialMigration应该包含所有会话管理表的创建逻辑（chat_sessions, chat_messages, chat_summaries）
4. 当执行InitialMigration的Up方法时，System应该按正确的依赖顺序创建表（先基础表，后关联表）
5. 当数据库为PostgreSQL时，System应该启用UUID扩展（uuid-ossp或pgcrypto）
6. InitialMigration应该包含所有必要的索引创建逻辑
7. InitialMigration应该包含所有外键约束的定义
8. 当数据库为PostgreSQL时，InitialMigration应该添加表和列的中文注释

### 需求 3：更新迁移管理器

**用户故事：** 作为系统架构师，我希望迁移管理器能够正确注册和执行初始迁移，以便确保迁移顺序的正确性

#### 验收标准

1. System应该在MigrationManager中添加RegisterInitialMigration方法
2. 当调用RunAllMigrations时，System应该首先执行InitialMigration
3. System应该提供RunInitialMigration函数用于单独执行初始迁移
4. 当迁移失败时，System应该返回包含迁移名称的详细错误信息

### 需求 4：验证ORM模型定义

**用户故事：** 作为开发者，我希望所有ORM模型定义与数据库结构完全一致，以便避免运行时错误

#### 验收标准

1. System应该确保internal/model/auth.go中所有模型的主键类型为UUID
2. System应该确保internal/model/session.go中所有模型的主键类型为UUID
3. 当模型定义外键关系时，System应该使用正确的GORM标签（foreignKey, constraint）
4. System应该确保所有JSONB字段使用datatypes.JSON类型
5. System应该确保所有时间字段使用time.Time类型
6. System应该确保所有索引标签与迁移文件中的索引定义一致

### 需求 5：创建迁移脚本

**用户故事：** 作为运维人员，我希望有便捷的脚本来执行初始迁移和数据库重置，以便简化部署流程

#### 验收标准

1. System应该创建scripts/init_migration.go脚本用于执行初始迁移
2. System应该创建scripts/reset_db.go脚本用于开发环境的数据库重置
3. 当执行init_migration.go时，System应该连接数据库并执行InitialMigration
4. 当执行reset_db.go时，System应该先执行Down方法清空所有表，再执行Up方法重建表结构
5. 当脚本执行失败时，System应该输出清晰的错误信息并返回非零退出码

### 需求 6：更新数据库配置

**用户故事：** 作为配置管理员，我希望数据库配置文件包含所有必要的设置，以便确保迁移正常运行

#### 验收标准

1. System应该在.env.example中包含DATABASE_URL配置示例
2. System应该在.env.example中包含PostgreSQL连接参数说明
3. 当数据库为PostgreSQL时，System应该确保支持UUID扩展的配置

### 需求 7：测试迁移功能

**用户故事：** 作为质量保证工程师，我希望能够在干净的数据库上测试迁移，以便验证迁移的正确性

#### 验收标准

1. 当在空数据库上执行InitialMigration时，System应该成功创建所有表
2. 当迁移完成后，System应该验证所有表都已创建
3. 当迁移完成后，System应该验证所有索引都已创建
4. 当迁移完成后，System应该验证所有外键约束都已创建
5. 当执行Down方法时，System应该成功删除所有表
6. 当执行Down后再执行Up时，System应该能够重新创建所有表

### 需求 8：更新文档

**用户故事：** 作为新加入的开发者，我希望有清晰的文档说明如何使用初始迁移，以便快速上手

#### 验收标准

1. System应该更新docs/database-migration-guide.md文档
2. 文档应该包含初始迁移的使用说明
3. 文档应该包含数据库结构版本信息
4. 文档应该包含迁移脚本的使用示例
5. 文档应该包含常见问题和解决方案

### 需求 9：调整代码集成

**用户故事：** 作为应用开发者，我希望应用启动时自动执行初始迁移，以便简化部署流程

#### 验收标准

1. System应该在cmd/server/main.go中集成初始迁移调用
2. 当应用启动时，System应该检查是否需要执行迁移
3. 当迁移失败时，System应该阻止应用启动并输出错误信息
4. System应该确保所有Repository层代码与更新后的模型定义兼容
