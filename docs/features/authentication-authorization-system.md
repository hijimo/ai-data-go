# 认证与权限系统实现文档

## 概述

本文档描述了AI知识管理平台的认证与权限系统的完整实现，包括JWT认证中间件、RBAC权限控制和KMS密钥管理功能。

## 系统架构

### 1. JWT认证系统

#### 核心组件

- **JWTManager**: JWT令牌管理器，负责令牌的生成、验证和刷新
- **AuthMiddleware**: 认证中间件，验证请求中的JWT令牌
- **AuthHandler**: 认证处理器，提供登录、登出、令牌刷新等接口

#### 主要功能

- JWT令牌生成和验证
- 令牌自动刷新机制
- 多种令牌提取方式（Header、Query、Cookie）
- 用户会话管理

#### 安全特性

- 使用HMAC-SHA256签名算法
- 令牌过期时间控制
- 支持令牌黑名单（预留接口）

### 2. RBAC权限控制系统

#### 角色定义

系统预定义了5种角色，权限层次从高到低：

1. **系统管理员 (system_admin)**
   - 拥有系统所有权限
   - 可以跨项目访问资源

2. **项目所有者 (project_owner)**
   - 项目内拥有所有权限
   - 可以管理项目成员

3. **项目管理员 (project_admin)**
   - 项目管理和配置权限
   - 不能删除项目

4. **项目成员 (project_member)**
   - 项目基本操作权限
   - 可以上传文档、创建Agent等

5. **项目查看者 (project_viewer)**
   - 只读权限
   - 可以查看项目资源但不能修改

#### 权限分类

权限按资源类型分为以下几类：

- 项目权限：project:read, project:write, project:delete, project:manage
- 文档权限：document:read, document:write, document:delete, document:upload
- 向量权限：vector:read, vector:write, vector:delete, vector:search
- LLM权限：llm:read, llm:write, llm:manage, llm:chat
- Agent权限：agent:read, agent:write, agent:delete, agent:manage, agent:chat
- 对话权限：chat:read, chat:write, chat:delete
- 问题答案权限：question:*, answer:*
- 任务权限：task:read, task:write, task:cancel
- 数据集权限：dataset:read, dataset:export
- 训练权限：training:read, training:write, training:cancel
- 系统权限：system:admin, user:manage

#### 权限中间件

- **PermissionMiddleware**: 检查用户是否拥有指定权限
- **AnyPermissionMiddleware**: 检查用户是否拥有任意一个权限
- **SystemAdminMiddleware**: 检查系统管理员权限
- **ProjectOwnerMiddleware**: 检查项目所有者权限
- **ProjectIsolationMiddleware**: 项目数据隔离
- **RequireProjectMiddleware**: 要求项目ID

### 3. KMS密钥管理系统

#### 支持的KMS提供商

1. **阿里云KMS (alibaba_cloud)**
   - 生产环境推荐
   - 支持硬件安全模块(HSM)
   - 高可用性和安全性

2. **本地KMS (local)**
   - 开发环境使用
   - 使用AES-256-GCM加密
   - 基于密钥ID生成密钥

#### 核心功能

- **数据加密/解密**: 支持任意文本数据的加密解密
- **敏感信息管理**: 专门用于管理API密钥、密码等敏感信息
- **数据密钥生成**: 生成用于数据加密的密钥
- **多提供商支持**: 可以同时配置多个KMS提供商
- **健康检查**: 监控KMS服务的可用性

#### 敏感信息类型

- API密钥 (api_key)
- 密码 (password)
- 令牌 (token)
- 私钥 (private_key)
- 证书 (certificate)
- 数据库配置 (database)
- 其他 (other)

## API接口

### 认证接口

```
POST /api/v1/auth/login          # 用户登录
POST /api/v1/auth/refresh        # 刷新令牌
POST /api/v1/auth/logout         # 用户登出
GET  /api/v1/auth/profile        # 获取用户信息
GET  /api/v1/auth/validate       # 验证令牌
```

### 权限管理接口

```
GET  /api/v1/permissions/roles                    # 获取角色列表
GET  /api/v1/permissions/roles/{role_name}        # 获取角色详情
GET  /api/v1/permissions/user                     # 获取用户权限
GET  /api/v1/permissions/check                    # 检查权限
POST /api/v1/permissions/check-multiple           # 批量检查权限
GET  /api/v1/permissions/by-resource              # 按资源获取权限
```

### KMS管理接口

```
POST /api/v1/kms/encrypt                # 加密数据
POST /api/v1/kms/decrypt                # 解密数据
POST /api/v1/kms/secrets/encrypt        # 加密敏感信息
POST /api/v1/kms/secrets/decrypt        # 解密敏感信息
GET  /api/v1/kms/providers              # 列出KMS提供商
GET  /api/v1/kms/health                 # KMS健康检查
POST /api/v1/kms/data-key               # 生成数据密钥
```

## 配置说明

### JWT配置

```go
type AuthConfig struct {
    JWTSecret              string        // JWT签名密钥
    JWTExpiration          time.Duration // JWT过期时间
    RefreshTokenSecret     string        // 刷新令牌密钥
    RefreshTokenExpiration time.Duration // 刷新令牌过期时间
}
```

### KMS配置

```go
type KMSConfig struct {
    DefaultProvider string                    // 默认KMS提供商
    Providers       map[string]*kms.KMSConfig // KMS提供商配置
}
```

### 环境变量

```bash
# JWT配置
JWT_SECRET=your-jwt-secret-key
REFRESH_TOKEN_SECRET=your-refresh-token-secret

# 本地KMS配置
KMS_DEFAULT_PROVIDER=local
KMS_LOCAL_KEY_ID=local-development-key-id

# 阿里云KMS配置
ALIBABA_CLOUD_ACCESS_KEY_ID=your-access-key-id
ALIBABA_CLOUD_ACCESS_KEY_SECRET=your-access-key-secret
ALIBABA_CLOUD_KMS_KEY_ID=your-kms-key-id
ALIBABA_CLOUD_REGION=cn-hangzhou
ALIBABA_CLOUD_KMS_ENDPOINT=kms.cn-hangzhou.aliyuncs.com
```

## 安全最佳实践

### 1. JWT安全

- 使用强随机密钥
- 设置合理的过期时间
- 实施令牌黑名单机制
- 使用HTTPS传输

### 2. 权限控制

- 最小权限原则
- 定期审查用户权限
- 项目级数据隔离
- 敏感操作审计日志

### 3. 密钥管理

- 使用硬件安全模块
- 定期轮换密钥
- 加密存储敏感配置
- 监控KMS服务状态

## 测试覆盖

系统包含完整的单元测试，覆盖以下方面：

- JWT令牌生成和验证
- 权限检查逻辑
- KMS加密解密功能
- 中间件功能测试
- 错误处理测试

## 部署注意事项

### 生产环境

1. 使用阿里云KMS或其他云厂商KMS服务
2. 配置强随机JWT密钥
3. 启用HTTPS
4. 配置适当的令牌过期时间
5. 实施监控和告警

### 开发环境

1. 可以使用本地KMS进行开发
2. 使用较短的令牌过期时间便于测试
3. 启用详细的日志记录

## 扩展性

系统设计具有良好的扩展性：

- 支持添加新的KMS提供商
- 可以扩展权限类型和角色
- 支持自定义认证方式
- 预留了审计日志接口

## 总结

本认证与权限系统提供了完整的安全解决方案，包括：

- 基于JWT的无状态认证
- 灵活的RBAC权限控制
- 企业级KMS密钥管理
- 完善的API接口
- 全面的测试覆盖

系统设计遵循安全最佳实践，具有良好的可扩展性和可维护性，能够满足企业级应用的安全需求。
