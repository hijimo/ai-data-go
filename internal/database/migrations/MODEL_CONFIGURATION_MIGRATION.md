# 模型配置模块数据库迁移

## 概述

本迁移为多租户AI平台添加了模型配置管理功能，允许租户管理员和平台管理员配置和管理不同的AI模型提供商。

## 迁移内容

### 创建的表

#### model_configurations 表

存储租户的AI模型提供商配置信息。

**字段说明**：

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | UUID | PRIMARY KEY | 模型配置唯一标识符 |
| tenant_id | UUID | NOT NULL, FK | 所属租户ID |
| name | VARCHAR(255) | NOT NULL | 配置名称 |
| model | VARCHAR(255) | NOT NULL | 模型标识（如：gpt-4、claude-3-opus等） |
| model_provider | VARCHAR(50) | NOT NULL | 模型提供商枚举值 |
| base_url | VARCHAR(500) | NULL | API基础URL（可选） |
| api_key | TEXT | NOT NULL | API密钥（加密存储） |
| query_params | JSONB | NULL | 查询参数（JSON格式） |
| is_enabled | BOOLEAN | NOT NULL, DEFAULT true | 是否启用 |
| is_deleted | BOOLEAN | NOT NULL, DEFAULT false | 软删除标记 |
| created_by | UUID | NOT NULL, FK | 创建者用户ID |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 创建时间 |
| updated_by | UUID | NULL, FK | 更新者用户ID |
| updated_at | TIMESTAMP | NULL | 更新时间 |
| deleted_by | UUID | NULL, FK | 删除者用户ID |
| deleted_at | TIMESTAMP | NULL | 删除时间 |

**支持的模型提供商**：

- `openai` - OpenAI
- `anthropic` - Anthropic (Claude)
- `googlegenai` - Google GenAI (Gemini)
- `azureopenai` - Azure OpenAI
- `bianlian` - Bianlian
- `custom_openai` - 自定义OpenAI兼容端点

### 索引

1. **idx_model_configs_tenant_provider** - 复合索引（tenant_id, model_provider），用于按租户和提供商查询
2. **idx_model_configs_deleted** - 单列索引（is_deleted），用于过滤已删除记录
3. **idx_model_configs_enabled** - 部分索引（is_enabled WHERE is_deleted = false），用于查询可用配置
4. **idx_model_configs_tenant_id** - 单列索引（tenant_id WHERE is_deleted = false），用于租户列表查询
5. **idx_model_configs_created_at** - 单列索引（created_at DESC），用于按创建时间排序

### 外键约束

1. **fk_model_configurations_tenant** - 关联到 tenants 表，级联删除
2. **fk_model_configurations_created_by** - 关联到 users 表（创建者）
3. **fk_model_configurations_updated_by** - 关联到 users 表（更新者）
4. **fk_model_configurations_deleted_by** - 关联到 users 表（删除者）

### 检查约束

**chk_model_provider** - 确保 model_provider 字段只能是预定义的枚举值之一

## 执行迁移

### 自动执行（推荐）

迁移会在应用启动时自动执行：

```bash
go run cmd/server/main.go
```

### 手动执行

使用迁移脚本：

```bash
go run scripts/migrate.go
```

### 验证迁移

检查表是否创建成功：

```sql
-- 检查表是否存在
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name = 'model_configurations';

-- 检查表结构
\d model_configurations

-- 检查索引
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'model_configurations';

-- 检查约束
SELECT conname, contype, pg_get_constraintdef(oid) 
FROM pg_constraint 
WHERE conrelid = 'model_configurations'::regclass;
```

## 回滚迁移

如果需要回滚此迁移：

```go
migration := migrations.NewModelConfigurationMigration(db)
err := migration.Down()
```

**警告**：回滚将删除 model_configurations 表及其所有数据！

## 数据安全

### API密钥加密

- API密钥在存储前必须使用 AES-256-GCM 算法加密
- 加密密钥通过环境变量 `ENCRYPTION_SECRET_KEY` 配置
- 返回给客户端时进行脱敏处理（仅显示前4位和后4位）

### 多租户隔离

- 所有查询都必须包含租户ID过滤
- 租户管理员只能访问自己租户的配置
- 平台管理员可以访问所有租户的配置

### 软删除

- 删除操作不会物理删除数据
- 通过 is_deleted 字段标记删除状态
- 所有查询自动过滤已删除的记录

## 相关文件

- **模型定义**: `internal/model/provider.go`
- **迁移文件**: `internal/database/migrations/model_configuration_migration.go`
- **需求文档**: `.kiro/specs/model-configuration/requirements.md`
- **设计文档**: `.kiro/specs/model-configuration/design.md`
- **任务列表**: `.kiro/specs/model-configuration/tasks.md`

## 注意事项

1. **依赖关系**：此迁移依赖于 tenants 和 users 表，必须在初始迁移之后执行
2. **UUID支持**：需要 PostgreSQL 13+ 或启用 pgcrypto 扩展
3. **加密配置**：部署前必须配置 ENCRYPTION_SECRET_KEY 环境变量
4. **索引优化**：部分索引用于优化常见查询，减少索引大小

## 性能考虑

- 使用复合索引优化按租户和提供商的查询
- 使用部分索引减少索引大小，提高查询性能
- JSONB 类型用于灵活存储查询参数
- 外键约束确保数据完整性

## 后续步骤

完成迁移后，需要实现以下组件：

1. EncryptionService - API密钥加密服务
2. ProviderRepository - 数据访问层
3. ProviderService - 业务逻辑层
4. ProviderHandler - API处理器
5. 路由配置和中间件

详见任务列表：`.kiro/specs/model-configuration/tasks.md`
