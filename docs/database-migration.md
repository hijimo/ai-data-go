# 数据库迁移系统

本文档介绍AI知识管理平台的数据库迁移系统的使用方法。

## 概述

我们使用 [golang-migrate](https://github.com/golang-migrate/migrate) 作为数据库迁移工具，并在此基础上构建了自定义的迁移管理系统。

## 功能特性

- ✅ 支持数据库版本管理
- ✅ 支持迁移回滚
- ✅ 自动初始化种子数据
- ✅ 支持强制版本设置（修复脏状态）
- ✅ 完整的CLI工具
- ✅ 集成到应用启动流程
- ✅ 完善的测试覆盖

## 快速开始

### 1. 初始化数据库

```bash
# 方法1：使用初始化脚本（推荐）
./scripts/init-db.sh

# 方法2：使用Make命令
make db-init
```

### 2. 基本迁移操作

```bash
# 运行所有待执行的迁移
make migrate-up

# 回滚一个迁移版本
make migrate-down

# 查看当前数据库版本
make migrate-version

# 创建新的迁移文件
make migrate-create
# 然后输入迁移文件名，例如：add_user_table
```

### 3. 种子数据管理

```bash
# 初始化种子数据
make db-seed

# 清理种子数据
make db-clean
```

## 高级操作

### 强制设置版本

当迁移处于"脏"状态时，可以强制设置版本：

```bash
make migrate-force
# 然后输入目标版本号
```

### 删除所有表

⚠️ **危险操作**：这将删除所有数据库表

```bash
make migrate-drop
```

### 直接使用CLI工具

```bash
# 构建CLI工具
make build-migrate

# 查看所有可用操作
./bin/migrate -h

# 示例：运行迁移
./bin/migrate -action=up

# 示例：查看版本
./bin/migrate -action=version

# 示例：创建迁移文件
./bin/migrate -action=create -name=add_new_feature
```

## 迁移文件结构

迁移文件位于 `migrations/` 目录下，采用以下命名规范：

```
migrations/
├── 000001_init_schema.up.sql          # 初始化数据库结构
├── 000001_init_schema.down.sql        # 回滚初始化
├── 000002_create_indexes.up.sql       # 创建索引
├── 000002_create_indexes.down.sql     # 删除索引
├── 000003_update_uuid_functions.up.sql # 更新UUID函数和约束
├── 000003_update_uuid_functions.down.sql
├── 000004_add_audit_logs.up.sql       # 添加审计日志表
└── 000004_add_audit_logs.down.sql
```

### 迁移文件编写规范

1. **文件命名**：`{version}_{description}.{up|down}.sql`
2. **版本号**：6位数字，递增
3. **描述**：简短的英文描述，使用下划线分隔
4. **Up文件**：包含正向迁移的SQL语句
5. **Down文件**：包含回滚迁移的SQL语句

### 示例迁移文件

**000005_add_user_preferences.up.sql**

```sql
-- 添加用户偏好设置表
CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    preferences JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 添加索引
CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id) WHERE is_deleted = FALSE;

-- 添加注释
COMMENT ON TABLE user_preferences IS '用户偏好设置表';
```

**000005_add_user_preferences.down.sql**

```sql
-- 删除用户偏好设置表
DROP TABLE IF EXISTS user_preferences;
```

## 数据库表结构

当前系统包含以下17个核心表：

### 项目管理

- `projects` - 项目表
- `project_members` - 项目成员表

### 文档管理

- `files` - 文件表
- `document_versions` - 文档版本表
- `chunks` - 文档块表

### 向量存储

- `vector_indexes` - 向量索引表
- `vector_records` - 向量记录表

### LLM管理

- `llm_providers` - LLM提供商表
- `llm_models` - LLM模型表
- `agents` - Agent表

### 对话系统

- `chat_sessions` - 对话会话表
- `chat_messages` - 对话消息表

### 问答系统

- `questions` - 问题表
- `answers` - 答案表

### 任务系统

- `tasks` - 任务表
- `training_jobs` - 训练任务表

### 审计系统

- `audit_logs` - 审计日志表

## 种子数据

系统会自动初始化以下种子数据：

### LLM提供商

- OpenAI
- Azure OpenAI
- 阿里云千问
- Anthropic Claude
- 百川智能
- 智谱ChatGLM

### LLM模型

每个提供商都包含相应的聊天模型和嵌入模型，例如：

- GPT-4o、GPT-4o Mini、GPT-3.5 Turbo
- Text Embedding 3 Large/Small
- 通义千问系列模型
- Claude 3.5 Sonnet、Claude 3 Haiku
- 等等

## 测试

运行迁移系统的集成测试：

```bash
# 设置环境变量启用集成测试
export RUN_INTEGRATION_TESTS=true

# 运行测试
go test -v ./internal/database/...
```

## 故障排除

### 1. 迁移失败

如果迁移失败，数据库可能处于"脏"状态：

```bash
# 查看当前状态
make migrate-version

# 如果显示 dirty=true，使用force命令修复
make migrate-force
# 输入当前版本号
```

### 2. 连接失败

检查数据库连接配置：

```bash
# 检查环境变量
cat .env

# 测试数据库连接
pg_isready -h localhost -p 5432 -U postgres
```

### 3. 权限问题

确保数据库用户有足够的权限：

```sql
-- 授予创建数据库权限
ALTER USER your_user CREATEDB;

-- 授予所有权限（开发环境）
GRANT ALL PRIVILEGES ON DATABASE your_db TO your_user;
```

## 最佳实践

1. **备份数据**：在生产环境运行迁移前，务必备份数据库
2. **测试迁移**：在开发环境充分测试迁移脚本
3. **原子操作**：每个迁移文件应该是原子的，要么全部成功，要么全部失败
4. **可回滚**：确保每个up迁移都有对应的down迁移
5. **渐进式**：避免在单个迁移中进行过多更改
6. **文档化**：为复杂的迁移添加详细注释

## 配置

迁移系统通过以下环境变量进行配置：

```bash
# 数据库连接
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=ai_knowledge_platform
DB_SSLMODE=disable

# 或者使用完整的数据库URL
DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
```

## 集成到CI/CD

在CI/CD流程中集成迁移：

```yaml
# GitHub Actions 示例
- name: Run Database Migrations
  run: |
    make build-migrate
    ./bin/migrate -action=up
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
```
