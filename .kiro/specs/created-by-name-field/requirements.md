# 需求文档

## 简介

本功能旨在增强系统的审计追踪能力，通过在所有数据表中添加 `created_by_name` 字段来冗余记录创建人的显示名称。同时，确保 `created_by` 字段始终从 JWT 令牌中自动获取，而不是由外部请求传入，以提高数据的安全性和一致性。

## 术语表

- **System**: 多租户聊天系统
- **JWT**: JSON Web Token，用于用户身份认证的令牌
- **ORM Model**: 对象关系映射模型，用于定义数据库表结构
- **Migration**: 数据库迁移脚本，用于修改数据库结构
- **created_by**: 创建者用户ID字段（UUID类型）
- **created_by_name**: 创建者显示名称字段（VARCHAR类型）
- **display_name**: 用户的显示名称

## 需求

### 需求 1：数据库字段添加

**用户故事**：作为系统管理员，我希望在所有数据表中记录创建人的显示名称，以便在查询数据时无需关联用户表即可快速获取创建人信息。

#### 验收标准

1. WHEN 系统执行数据库迁移时，THE System SHALL 在所有包含 `created_by` 字段的表中添加 `created_by_name` 字段
2. THE `created_by_name` 字段 SHALL 定义为 VARCHAR(255) 类型
3. THE `created_by_name` 字段 SHALL 允许为空值（NULL）
4. THE System SHALL 为 `created_by_name` 字段添加适当的数据库注释说明其用途

### 需求 2：ORM 模型更新

**用户故事**：作为开发人员，我希望所有 ORM 模型都包含 `created_by_name` 字段定义，以便在代码中正确处理该字段。

#### 验收标准

1. THE System SHALL 在以下模型中添加 `created_by_name` 字段：
   - Tenant（租户模型）
   - User（用户模型）
   - ChatSession（聊天会话模型）
2. THE `created_by_name` 字段 SHALL 使用 GORM 标签定义为 `gorm:"type:varchar(255)"`
3. THE `created_by_name` 字段 SHALL 使用 JSON 标签定义为 `json:"createdByName"`
4. THE `created_by_name` 字段 SHALL 添加示例值标签 `example:"张三"`
5. THE `created_by_name` 字段 SHALL 添加中文注释说明其用途

### 需求 3：创建者信息自动填充

**用户故事**：作为系统开发人员，我希望在创建数据时自动从 JWT 令牌中提取创建者信息，以便确保数据的安全性和一致性。

#### 验收标准

1. WHEN 创建新的租户、用户或会话记录时，THE System SHALL 从 JWT 令牌的 Claims 中提取用户ID
2. WHEN 创建新的租户、用户或会话记录时，THE System SHALL 从 JWT 令牌的 Claims 中提取用户显示名称
3. THE System SHALL 将提取的用户ID设置为 `created_by` 字段的值
4. THE System SHALL 将提取的显示名称设置为 `created_by_name` 字段的值
5. IF 外部请求尝试传入 `created_by` 或 `created_by_name` 字段值，THEN THE System SHALL 忽略这些值并使用 JWT 令牌中的信息

### 需求 4：JWT Claims 扩展

**用户故事**：作为系统架构师，我希望 JWT 令牌中包含用户的显示名称，以便在创建数据时可以直接使用该信息。

#### 验收标准

1. THE JWTClaims 结构体 SHALL 包含 `DisplayName` 字段
2. THE `DisplayName` 字段 SHALL 定义为 string 类型
3. THE `DisplayName` 字段 SHALL 添加 JSON 标签 `json:"displayName"`
4. WHEN 生成 JWT 令牌时，THE System SHALL 将用户的 `display_name` 值设置到 Claims 的 `DisplayName` 字段中
5. WHEN 解析 JWT 令牌时，THE System SHALL 能够正确提取 `DisplayName` 字段的值

### 需求 5：服务层逻辑更新

**用户故事**：作为后端开发人员，我希望在所有创建操作的服务层中自动填充创建者信息，以便保持代码的一致性。

#### 验收标准

1. THE 租户创建服务 SHALL 从 JWT Claims 中获取 `Subject`（用户ID）和 `DisplayName`
2. THE 租户创建服务 SHALL 将获取的用户ID设置为新租户的 `created_by` 字段
3. THE 租户创建服务 SHALL 将获取的显示名称设置为新租户的 `created_by_name` 字段
4. THE 用户创建服务 SHALL 从 JWT Claims 中获取 `Subject`（用户ID）和 `DisplayName`
5. THE 用户创建服务 SHALL 将获取的用户ID设置为新用户的 `created_by` 字段
6. THE 用户创建服务 SHALL 将获取的显示名称设置为新用户的 `created_by_name` 字段
7. THE 会话创建服务 SHALL 从 JWT Claims 中获取 `Subject`（用户ID）和 `DisplayName`
8. THE 会话创建服务 SHALL 将获取的用户ID设置为新会话的 `created_by` 字段
9. THE 会话创建服务 SHALL 将获取的显示名称设置为新会话的 `created_by_name` 字段

### 需求 6：数据库迁移脚本

**用户故事**：作为数据库管理员，我希望有一个安全的迁移脚本来添加新字段，以便在不影响现有数据的情况下升级数据库结构。

#### 验收标准

1. THE System SHALL 创建一个新的数据库迁移文件 `add_created_by_name_migration.go`
2. THE 迁移脚本 SHALL 在事务中执行所有 DDL 操作
3. THE 迁移脚本 SHALL 为 `tenants` 表添加 `created_by_name` 字段
4. THE 迁移脚本 SHALL 为 `users` 表添加 `created_by_name` 字段
5. THE 迁移脚本 SHALL 为 `chat_sessions` 表添加 `created_by_name` 字段
6. THE 迁移脚本 SHALL 提供回滚功能（Down 方法）以删除添加的字段
7. THE 迁移脚本 SHALL 在迁移管理器中注册
8. IF 迁移过程中发生错误，THEN THE System SHALL 回滚所有更改

### 需求 7：向后兼容性

**用户故事**：作为系统维护人员，我希望新字段的添加不会影响现有功能，以便系统能够平滑升级。

#### 验收标准

1. THE `created_by_name` 字段 SHALL 允许为 NULL 值
2. THE 现有数据记录的 `created_by_name` 字段 SHALL 保持为 NULL
3. THE 查询现有数据时 SHALL 不因 `created_by_name` 为 NULL 而报错
4. THE API 响应 SHALL 正确序列化 `created_by_name` 字段（包括 NULL 值）
5. THE 系统 SHALL 在迁移后继续正常运行所有现有功能

### 需求 8：数据一致性验证

**用户故事**：作为质量保证工程师，我希望系统能够验证创建者信息的一致性，以便确保数据的准确性。

#### 验收标准

1. WHEN 创建新记录时，IF JWT Claims 中包含 `DisplayName`，THEN THE System SHALL 使用该值填充 `created_by_name` 字段
2. WHEN 创建新记录时，IF JWT Claims 中不包含 `DisplayName`，THEN THE System SHALL 将 `created_by_name` 字段设置为 NULL
3. THE `created_by` 字段和 `created_by_name` 字段 SHALL 始终来自同一个 JWT 令牌
4. THE System SHALL 不允许外部请求覆盖 `created_by` 或 `created_by_name` 字段的值
