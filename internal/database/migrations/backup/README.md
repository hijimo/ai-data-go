# 迁移文件备份目录

## 备份说明

本目录包含已归档的旧迁移文件，这些文件在数据库迁移整合过程中被移至此处作为备份。

## 备份日期

- **备份时间**: 2025-01-20
- **备份原因**: 整合分散的迁移文件为统一的初始迁移基线

## 备份文件列表

### 1. auth_migration.go

**原始位置**: `internal/database/migrations/auth_migration.go`

**功能说明**:

- 认证相关表的迁移脚本
- 包含以下表的创建逻辑：
  - `tenants` - 租户表
  - `users` - 用户表
  - `refresh_tokens` - 刷新令牌表
  - `auth_audit` - 认证审计日志表
  - `email_verification_tokens` - 邮箱验证令牌表

**特性**:

- 支持 PostgreSQL UUID 类型
- 包含完整的索引和外键约束
- 添加了中文表注释和列注释
- 实现了 Up 和 Down 方法

### 2. session_migration.go

**原始位置**: `internal/database/migrations/session_migration.go`

**功能说明**:

- 会话管理相关表的迁移脚本
- 包含以下表的创建逻辑：
  - `chat_sessions` - 聊天会话表
  - `chat_messages` - 聊天消息表
  - `chat_summaries` - 聊天摘要表

**特性**:

- 支持 PostgreSQL UUID 类型
- 包含复合索引优化查询性能
- 实现了 Up 和 Down 方法

## 为什么备份？

在数据库迁移整合项目中，我们将分散的迁移文件（`auth_migration.go` 和 `session_migration.go`）整合为一个统一的初始迁移文件（`initial_migration.go`）。

**整合的优势**:

1. **统一管理**: 所有表结构在一个文件中定义，便于维护
2. **清晰的基线**: 建立明确的数据库结构版本基线
3. **简化部署**: 新环境只需执行一个初始迁移即可
4. **版本控制**: 后续的数据库变更可以基于这个基线进行增量迁移

**为什么保留备份**:

1. **历史参考**: 保留原始实现作为参考
2. **回滚能力**: 如果需要，可以参考原始实现
3. **文档价值**: 记录了迁移整合前的状态
4. **安全考虑**: 避免直接删除可能还有用的代码

## 使用说明

### 查看备份文件

这些文件仅作为备份和参考，不应在生产代码中使用。

```bash
# 查看认证迁移备份
cat internal/database/migrations/backup/auth_migration.go

# 查看会话迁移备份
cat internal/database/migrations/backup/session_migration.go
```

### 恢复备份（不推荐）

如果确实需要恢复旧的迁移文件：

```bash
# 备份当前的初始迁移
cp internal/database/migrations/initial_migration.go \
   internal/database/migrations/initial_migration.go.bak

# 恢复旧的迁移文件
cp internal/database/migrations/backup/auth_migration.go \
   internal/database/migrations/auth_migration.go
cp internal/database/migrations/backup/session_migration.go \
   internal/database/migrations/session_migration.go
```

**警告**: 恢复旧的迁移文件可能导致与新的初始迁移冲突，请谨慎操作。

## 新的迁移结构

整合后的迁移结构：

```
internal/database/migrations/
├── backup/                      # 备份目录
│   ├── README.md               # 本文件
│   ├── auth_migration.go       # 认证迁移备份
│   └── session_migration.go    # 会话迁移备份
├── initial_migration.go        # 统一的初始迁移（新）
├── migration_manager.go        # 迁移管理器
└── README.md                   # 迁移目录说明
```

## 迁移整合对比

### 整合前（旧结构）

```go
// 需要分别执行两个迁移
manager.Register(NewAuthMigration(db))
manager.Register(NewSessionMigration(db))
```

### 整合后（新结构）

```go
// 只需执行一个初始迁移
manager.Register(NewInitialMigration(db))
```

## 相关文档

- [初始迁移设计文档](../../../../.kiro/specs/database-initial-migration/design.md)
- [迁移任务列表](../../../../.kiro/specs/database-initial-migration/tasks.md)
- [数据库迁移指南](../../../../docs/database-migration-guide.md)

## 清理说明

### 何时可以删除备份？

建议在以下情况下才考虑删除备份文件：

1. 新的初始迁移已在生产环境稳定运行至少 3 个月
2. 所有团队成员都熟悉新的迁移结构
3. 已有完整的文档记录迁移整合过程
4. 确认不再需要参考旧的实现

### 如何删除备份

```bash
# 删除整个备份目录
rm -rf internal/database/migrations/backup/

# 或者只删除备份文件，保留 README
rm internal/database/migrations/backup/auth_migration.go
rm internal/database/migrations/backup/session_migration.go
```

## 注意事项

1. **不要修改备份文件**: 这些文件仅作为历史记录，不应修改
2. **不要在代码中引用**: 新代码应该使用 `initial_migration.go`
3. **定期审查**: 定期审查是否还需要保留这些备份
4. **版本控制**: 这些备份文件应该保留在版本控制系统中

## 更新日志

### 2025-01-20

- 创建备份目录
- 移动 `auth_migration.go` 到备份目录
- 移动 `session_migration.go` 到备份目录
- 创建本 README 文档

## 联系方式

如有关于迁移整合的问题，请参考：

- 项目文档: `docs/database-migration-guide.md`
- 设计文档: `.kiro/specs/database-initial-migration/design.md`
- 或联系项目维护者
