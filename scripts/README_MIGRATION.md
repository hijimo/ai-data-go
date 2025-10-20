# 数据库迁移脚本使用指南

本目录包含用于数据库迁移管理的便捷脚本。

## 脚本列表

### 1. init_migration.go - 初始迁移脚本

用于在新环境中执行初始数据库迁移，创建所有必要的表结构。

#### 功能

- 连接到 PostgreSQL 数据库
- 执行初始迁移，创建所有表
- 支持命令行参数覆盖环境变量
- 提供详细的日志输出

#### 使用方法

```bash
# 使用环境变量配置（推荐）
go run scripts/init_migration.go

# 使用命令行参数覆盖配置
go run scripts/init_migration.go -host localhost -port 5432 -user postgres -dbname mydb

# 显示详细日志
go run scripts/init_migration.go -verbose

# 显示帮助信息
go run scripts/init_migration.go -help
```

#### 命令行参数

- `-host` - 数据库主机地址（覆盖 DB_HOST）
- `-port` - 数据库端口（覆盖 DB_PORT）
- `-user` - 数据库用户名（覆盖 DB_USER）
- `-password` - 数据库密码（覆盖 DB_PASSWORD）
- `-dbname` - 数据库名称（覆盖 DB_NAME）
- `-sslmode` - SSL 模式（覆盖 DB_SSLMODE）
- `-verbose` - 显示详细日志
- `-help` - 显示帮助信息

#### 创建的表

执行成功后，将创建以下表：

- `tenants` - 租户表
- `users` - 用户表
- `refresh_tokens` - 刷新令牌表
- `email_verification_tokens` - 邮箱验证令牌表
- `auth_audit` - 认证审计表
- `chat_sessions` - 聊天会话表
- `chat_messages` - 聊天消息表
- `chat_summaries` - 聊天摘要表

### 2. reset_db.go - 数据库重置脚本

用于在开发环境中重置数据库，删除所有表并重新创建。

⚠️ **警告**: 此工具会删除所有数据，仅用于开发环境！

#### 功能

- 删除数据库中的所有表（Down 方法）
- 重新创建所有表结构（Up 方法）
- 提供确认提示防止误操作
- 支持强制模式跳过确认

#### 使用方法

```bash
# 交互式重置（需要确认）
go run scripts/reset_db.go

# 强制重置（跳过确认，危险！）
go run scripts/reset_db.go -force

# 使用命令行参数覆盖配置
go run scripts/reset_db.go -host localhost -port 5432 -user postgres -dbname mydb

# 显示详细日志
go run scripts/reset_db.go -verbose

# 显示帮助信息
go run scripts/reset_db.go -help
```

#### 命令行参数

- `-host` - 数据库主机地址（覆盖 DB_HOST）
- `-port` - 数据库端口（覆盖 DB_PORT）
- `-user` - 数据库用户名（覆盖 DB_USER）
- `-password` - 数据库密码（覆盖 DB_PASSWORD）
- `-dbname` - 数据库名称（覆盖 DB_NAME）
- `-sslmode` - SSL 模式（覆盖 DB_SSLMODE）
- `-force` - 跳过确认提示，直接执行重置
- `-verbose` - 显示详细日志
- `-help` - 显示帮助信息

#### 安全提示

- ⚠️ 此工具仅用于开发环境
- ⚠️ 不要在生产环境中使用
- ⚠️ 操作前请确保已备份重要数据
- ⚠️ 使用 `-force` 参数时请格外小心

#### 确认流程

在交互模式下，脚本会要求：

1. 输入数据库名称以确认操作
2. 输入 "yes" 或 "y" 最终确认

只有两步确认都通过后，才会执行重置操作。

## 环境变量配置

所有脚本都支持通过环境变量配置数据库连接：

```bash
# 数据库配置
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=genkit_ai_service
export DB_SSLMODE=disable

# 连接池配置（可选）
export DB_MAX_OPEN_CONNS=25
export DB_MAX_IDLE_CONNS=5
export DB_CONN_MAX_LIFETIME=5m

# 日志级别（可选）
export DB_LOG_LEVEL=warn
```

或者使用 `.env` 文件：

```bash
# 复制示例配置文件
cp .env.example .env

# 编辑配置文件
vim .env
```

## 使用场景

### 场景 1: 新环境初始化

在全新的环境中首次设置数据库：

```bash
# 1. 配置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=genkit_ai_service

# 2. 执行初始迁移
go run scripts/init_migration.go
```

### 场景 2: 开发环境重置

在开发过程中需要清空数据库并重新开始：

```bash
# 交互式重置（推荐）
go run scripts/reset_db.go

# 或者强制重置（快速但危险）
go run scripts/reset_db.go -force
```

### 场景 3: CI/CD 环境

在持续集成环境中自动化数据库设置：

```bash
# 使用命令行参数，避免依赖环境变量
go run scripts/init_migration.go \
  -host $CI_DB_HOST \
  -port $CI_DB_PORT \
  -user $CI_DB_USER \
  -password $CI_DB_PASSWORD \
  -dbname $CI_DB_NAME \
  -sslmode disable
```

### 场景 4: 测试环境准备

在运行集成测试前准备干净的数据库：

```bash
# 重置测试数据库
go run scripts/reset_db.go \
  -host localhost \
  -port 5432 \
  -user test_user \
  -dbname test_db \
  -force
```

## 故障排查

### 连接失败

如果遇到数据库连接失败：

1. 检查数据库服务是否运行
2. 验证连接参数是否正确
3. 检查防火墙设置
4. 确认用户权限

```bash
# 测试数据库连接
psql -h localhost -p 5432 -U postgres -d genkit_ai_service
```

### 权限不足

如果遇到权限错误：

```sql
-- 授予必要的权限
GRANT ALL PRIVILEGES ON DATABASE genkit_ai_service TO your_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO your_user;
```

### UUID 扩展错误

如果遇到 UUID 相关错误：

```sql
-- 手动启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

## 最佳实践

1. **使用环境变量**: 将敏感信息（如密码）存储在环境变量或 `.env` 文件中
2. **备份数据**: 在执行重置操作前，始终备份重要数据
3. **测试环境**: 先在测试环境中验证脚本，再应用到其他环境
4. **版本控制**: 不要将 `.env` 文件提交到版本控制系统
5. **日志记录**: 使用 `-verbose` 参数获取详细日志，便于调试

## 相关文档

- [数据库迁移指南](../docs/database-migration-guide.md)
- [初始迁移设计文档](../.kiro/specs/database-initial-migration/design.md)
- [需求文档](../.kiro/specs/database-initial-migration/requirements.md)

## 支持

如有问题或建议，请查看项目文档或联系开发团队。
