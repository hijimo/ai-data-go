# 模型配置模块需求文档

## 简介

本文档定义了多租户系统中模型配置管理功能的需求。该功能允许租户管理员和平台管理员配置和管理不同的AI模型提供商的连接配置，包括模型的连接参数、验证、启用/禁用和删除操作。

**注意**：本模块管理的是动态的模型配置（存储在数据库中），与系统中已存在的静态 Provider 服务（从配置文件读取模型元数据）是不同的概念。

## 术语表

- **System**: 多租户AI平台系统
- **ModelConfiguration**: 模型配置实体，包含模型提供商的连接信息
- **ModelProvider**: 模型提供商枚举，包括OpenAI、Anthropic、GoogleGenAI、AzureOpenAI、Bianlian和CustomOpenAI
- **TenantAdmin**: 租户管理员角色
- **SystemAdmin**: 平台管理员角色
- **ValidationEndpoint**: 模型配置验证接口
- **LogicalDeletion**: 逻辑删除，通过标记字段而非物理删除数据

## 需求

### 需求1：模型配置创建

**用户故事**: 作为租户管理员，我希望能够创建新的模型配置，以便我的租户可以使用不同的AI模型提供商。

#### 验收标准

1. 当租户管理员提交创建模型配置请求时，THE System SHALL 验证所有必填字段（name、model、modelProvider、apiKey）是否已提供
2. 当创建模型配置时，THE System SHALL 将modelProvider限制为预定义的枚举值（openai、anthropic、googlegenai、azureopenai、bianlian、custom_openai）
3. 当租户管理员创建模型配置时，THE System SHALL 自动将配置关联到当前租户ID
4. 当模型配置创建成功时，THE System SHALL 返回完整的模型配置信息，包括自动生成的UUID主键
5. 当平台管理员创建模型配置时，THE System SHALL 要求明确指定租户ID

### 需求2：模型配置查询

**用户故事**: 作为租户管理员，我希望能够查看我租户下的所有模型配置，以便管理和选择合适的模型。

#### 验收标准

1. 当租户管理员请求模型配置列表时，THE System SHALL 仅返回当前租户下的模型配置
2. 当平台管理员请求模型配置列表时，THE System SHALL 支持通过tenantId参数过滤特定租户的配置
3. 当查询模型配置列表时，THE System SHALL 支持分页参数（pageNo、pageSize）
4. 当查询模型配置列表时，THE System SHALL 排除已逻辑删除的记录（is_deleted=true）
5. 当请求单个模型配置详情时，THE System SHALL 验证该配置是否属于当前租户（租户管理员）或允许跨租户访问（平台管理员）

### 需求3：模型配置更新

**用户故事**: 作为租户管理员，我希望能够更新模型配置的参数，以便调整模型的连接设置。

#### 验收标准

1. 当租户管理员提交更新请求时，THE System SHALL 验证目标模型配置是否属于当前租户
2. 当更新模型配置时，THE System SHALL 允许修改name、model、baseUrl、apiKey和queryParams字段
3. 当更新模型配置时，THE System SHALL 禁止修改modelProvider字段
4. 当更新模型配置时，THE System SHALL 自动更新updated_at时间戳
5. 当平台管理员更新模型配置时，THE System SHALL 允许跨租户更新操作

### 需求4：模型配置验证

**用户故事**: 作为租户管理员，我希望能够验证模型配置是否可以正确连接，以便确保配置的有效性。

#### 验收标准

1. 当用户请求验证模型配置时，THE System SHALL 使用配置的参数尝试连接到模型提供商
2. 当验证成功时，THE System SHALL 返回成功状态和连接响应信息
3. 当验证失败时，THE System SHALL 返回失败状态和详细的错误信息
4. 当租户管理员验证模型配置时，THE System SHALL 确保该配置属于当前租户
5. 当验证请求超过30秒时，THE System SHALL 返回超时错误

### 需求5：模型配置启用/禁用

**用户故事**: 作为租户管理员，我希望能够启用或禁用模型配置，以便控制哪些模型可以被使用。

#### 验收标准

1. 当租户管理员请求更改模型配置状态时，THE System SHALL 验证该配置是否属于当前租户
2. 当更新状态为enabled时，THE System SHALL 将is_enabled字段设置为true
3. 当更新状态为disabled时，THE System SHALL 将is_enabled字段设置为false
4. 当模型配置被禁用时，THE System SHALL 在后续的模型列表查询中标记该配置为不可用
5. 当平台管理员更改状态时，THE System SHALL 允许跨租户操作

### 需求6：模型配置删除

**用户故事**: 作为租户管理员，我希望能够删除不再使用的模型配置，以便保持配置列表的整洁。

#### 验收标准

1. 当租户管理员请求删除模型配置时，THE System SHALL 验证该配置是否属于当前租户
2. 当删除模型配置时，THE System SHALL 执行逻辑删除，将is_deleted字段设置为true
3. 当删除模型配置时，THE System SHALL 保留deleted_at时间戳
4. 当删除模型配置时，THE System SHALL 保留deleted_by用户ID
5. 当查询模型配置列表时，THE System SHALL 自动过滤已删除的记录

### 需求7：模型选择列表

**用户故事**: 作为租户用户，我希望能够获取可用的模型列表，以便选择合适的模型处理我的工作。

#### 验收标准

1. 当用户请求可用模型列表时，THE System SHALL 仅返回当前租户下已启用的模型配置（is_enabled=true）
2. 当返回模型列表时，THE System SHALL 排除已逻辑删除的配置（is_deleted=true）
3. 当返回模型列表时，THE System SHALL 包含模型的基本信息（id、name、model、modelProvider）
4. 当返回模型列表时，THE System SHALL 排除敏感信息（apiKey、queryParams）
5. 当租户下没有可用模型时，THE System SHALL 返回空列表和成功状态码

### 需求8：数据安全和隔离

**用户故事**: 作为系统管理员，我希望确保模型配置数据在多租户环境中是安全和隔离的，以便保护各租户的敏感信息。

#### 验收标准

1. 当存储apiKey时，THE System SHALL 对其进行加密存储
2. 当返回模型配置信息时，THE System SHALL 对apiKey进行脱敏处理（仅显示前4位和后4位）
3. 当租户管理员访问模型配置时，THE System SHALL 确保无法访问其他租户的配置
4. 当记录操作日志时，THE System SHALL 排除apiKey等敏感信息
5. 当权限验证失败时，THE System SHALL 记录审计日志并返回403错误

### 需求9：审计和追踪

**用户故事**: 作为平台管理员，我希望能够追踪模型配置的创建、修改和删除操作，以便进行审计和问题排查。

#### 验收标准

1. 当创建模型配置时，THE System SHALL 记录created_by用户ID和created_at时间戳
2. 当更新模型配置时，THE System SHALL 记录updated_by用户ID和updated_at时间戳
3. 当删除模型配置时，THE System SHALL 记录deleted_by用户ID和deleted_at时间戳
4. 当执行敏感操作时，THE System SHALL 记录审计日志，包含操作类型、用户ID、租户ID和操作时间
5. 当权限验证失败时，THE System SHALL 记录失败尝试的审计日志
