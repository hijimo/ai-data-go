# 认证系统初始化指南

本文档提供多租户用户管理与 JWT 身份认证系统的初始化和使用说明。

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [详细步骤](#详细步骤)
- [配置说明](#配置说明)
- [API 使用示例](#api-使用示例)
- [常见问题](#常见问题)
- [安全建议](#安全建议)

## 系统要求

- Go 1.25 或更高版本
- PostgreSQL 12 或更高版本
- 已配置的数据库连接（参见 `.env` 文件）

## 快速开始

### 1. 配置环境变量

复制 `.env.example` 到 `.env` 并配置必要的参数：

```bash
cp .env.example .env
```

关键配置项：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=genkit_ai_service
DB_SSLMODE=disable

# JWT 配置
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_ISSUER=genkit-ai-service
JWT_AUDIENCE=genkit-api
ACCESS_TOKEN_TTL=60m
REFRESH_TOKEN_TTL=720h  # 30 天

# 密码配置
BCRYPT_COST=12
PASSWORD_MIN_LENGTH=8

# 登录安全
MAX_LOGIN_ATTEMPTS=5
LOGIN_ATTEMPT_WINDOW=15m

# Token 策略
ENABLE_REFRESH_ROTATION=true

# 租户识别
TENANT_IDENTIFY_STRATEGY=header  # header, subdomain, path, cookie
```

### 2. 执行数据库迁移

运行认证系统数据库迁移脚本：

```bash
go run scripts/auth_migrate.go
```

该脚本将创建以下表：

- `tenants` - 租户表
- `users` - 用户表
- `refresh_tokens` - 刷新令牌表
- `auth_audit` - 认证审计日志表

### 3. 初始化默认租户和管理员

运行初始化脚本创建默认租户和管理员用户：

```bash
go run scripts/init_auth.go
```

默认管理员账户信息：

- **邮箱**: `admin@example.com`
- **密码**: `Admin@123456`
- **角色**: `admin`, `user`

⚠️ **重要**: 首次登录后请立即修改默认密码！

### 4. 启动服务

```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 启动。

### 5. 访问 API 文档

打开浏览器访问 Swagger 文档：

```
http://localhost:8080/swagger/index.html
```

## 详细步骤

### 数据库迁移详解

`auth_migrate.go` 脚本执行以下操作：

1. 连接到配置的 PostgreSQL 数据库
2. 创建认证系统所需的所有表
3. 创建必要的索引和约束
4. 验证表结构

如果迁移失败，请检查：

- 数据库连接配置是否正确
- 数据库用户是否有足够的权限
- 数据库是否已存在同名表（可能需要先删除）

### 初始化脚本详解

`init_auth.go` 脚本执行以下操作：

1. 检查是否已存在租户
   - 如果存在，跳过租户创建
   - 如果不存在，创建默认租户

2. 检查是否已存在管理员用户
   - 如果存在，跳过用户创建
   - 如果不存在，创建管理员用户

3. 显示初始化信息和安全提示

### 自定义初始化

如果需要自定义初始租户和管理员信息，可以修改 `scripts/init_auth.go` 中的常量：

```go
const (
    // 默认租户信息
    defaultTenantName   = "你的租户名称"
    defaultTenantDomain = "your-domain"

    // 默认管理员信息
    defaultAdminEmail       = "your-admin@example.com"
    defaultAdminPassword    = "YourSecurePassword123!"
    defaultAdminDisplayName = "你的管理员名称"
)
```

## 配置说明

### JWT 配置

| 配置项 | 说明 | 默认值 | 建议 |
|--------|------|--------|------|
| JWT_SECRET | JWT 签名密钥 | - | 至少 256 位随机字符串 |
| JWT_ISSUER | JWT 签发者 | genkit-ai-service | 保持默认 |
| JWT_AUDIENCE | JWT 受众 | genkit-api | 保持默认 |
| ACCESS_TOKEN_TTL | Access Token 有效期 | 60m | 15m-60m |
| REFRESH_TOKEN_TTL | Refresh Token 有效期 | 720h | 168h-720h |

### 密码配置

| 配置项 | 说明 | 默认值 | 建议 |
|--------|------|--------|------|
| BCRYPT_COST | Bcrypt 成本因子 | 12 | 10-14 |
| PASSWORD_MIN_LENGTH | 密码最小长度 | 8 | 8-16 |

### 安全配置

| 配置项 | 说明 | 默认值 | 建议 |
|--------|------|--------|------|
| MAX_LOGIN_ATTEMPTS | 最大登录尝试次数 | 5 | 3-10 |
| LOGIN_ATTEMPT_WINDOW | 登录尝试时间窗口 | 15m | 10m-30m |
| ENABLE_REFRESH_ROTATION | 启用 Token 轮换 | true | 生产环境必须启用 |

### 租户识别策略

| 策略 | 说明 | 使用场景 |
|------|------|----------|
| header | 从请求头 `X-Tenant-ID` 识别 | API 客户端 |
| subdomain | 从子域名识别 | 多租户 SaaS |
| path | 从 URL 路径识别 | RESTful API |
| cookie | 从 Cookie 识别 | Web 应用 |

## API 使用示例

### 1. 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <租户ID>" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin@123456"
  }'
```

响应示例：

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "550e8400-e29b-41d4-a716-446655440000",
    "expires_in": 3600,
    "token_type": "Bearer",
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "tenant_id": "123e4567-e89b-12d3-a456-426614174001",
      "email": "admin@example.com",
      "display_name": "系统管理员",
      "is_admin": true,
      "roles": ["admin", "user"]
    }
  }
}
```

### 2. 获取当前用户信息

```bash
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Tenant-ID: <租户ID>"
```

### 3. 刷新 Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <租户ID>" \
  -d '{
    "refresh_token": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

### 4. 修改密码

```bash
curl -X POST http://localhost:8080/api/v1/auth/change-password \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Tenant-ID: <租户ID>" \
  -d '{
    "old_password": "Admin@123456",
    "new_password": "NewSecurePassword123!"
  }'
```

### 5. 用户注销

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Tenant-ID: <租户ID>" \
  -d '{
    "refresh_token": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

### 6. 创建新用户（需要管理员权限）

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Tenant-ID: <租户ID>" \
  -d '{
    "email": "user@example.com",
    "password": "UserPassword123!",
    "display_name": "普通用户",
    "roles": ["user"]
  }'
```

## 常见问题

### Q1: 迁移脚本报错 "表已存在"

**A**: 如果需要重新创建表，请先删除现有表：

```sql
DROP TABLE IF EXISTS auth_audit CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
```

然后重新运行迁移脚本。

### Q2: 忘记管理员密码怎么办？

**A**: 可以通过以下方式重置：

1. 直接在数据库中更新密码哈希
2. 删除管理员用户记录，重新运行 `init_auth.go`
3. 使用密码重置功能（如果已实现）

### Q3: 如何创建多个租户？

**A**: 使用租户管理 API：

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{
    "name": "新租户",
    "domain": "new-tenant"
  }'
```

### Q4: Token 过期后如何处理？

**A**: 客户端应该：

1. 检测到 401 错误
2. 使用 Refresh Token 调用刷新接口
3. 获取新的 Access Token
4. 重试原始请求

### Q5: 如何查看审计日志？

**A**: 审计日志存储在 `auth_audit` 表中，可以通过数据库查询或使用审计日志 API（如果已实现）。

### Q6: 生产环境部署注意事项？

**A**:

1. 使用强随机的 JWT_SECRET（至少 256 位）
2. 启用 HTTPS
3. 配置合适的 Token 有效期
4. 启用 Token 轮换
5. 定期清理过期的 Refresh Token
6. 监控登录失败次数
7. 定期审查审计日志

## 安全建议

### 密码安全

1. **强密码策略**
   - 最小长度 8 字符
   - 包含大小写字母、数字和特殊字符
   - 不使用常见密码

2. **密码存储**
   - 使用 bcrypt 哈希（cost=12）
   - 永不存储明文密码
   - 永不在日志中记录密码

3. **密码管理**
   - 定期提醒用户修改密码
   - 密码修改需验证旧密码
   - 密码修改后撤销所有 Token

### Token 安全

1. **Access Token**
   - 短生命周期（15-60 分钟）
   - 存储在内存或 sessionStorage
   - 不存储在 localStorage
   - 通过 Authorization 头传输

2. **Refresh Token**
   - 长生命周期（7-30 天）
   - 使用 HttpOnly Cookie（推荐）
   - 启用 Token 轮换
   - 一次性使用

3. **Token 管理**
   - 定期清理过期 Token
   - 异常情况立即撤销
   - 记录所有 Token 操作

### 租户隔离

1. **数据隔离**
   - 所有查询必须包含 tenant_id
   - 中间件层强制注入租户过滤
   - Repository 层二次验证

2. **权限控制**
   - 基于角色的访问控制（RBAC）
   - 最小权限原则
   - 定期审查权限分配

### 审计与监控

1. **审计日志**
   - 记录所有认证操作
   - 记录失败尝试
   - 记录异常行为

2. **监控告警**
   - 登录失败率
   - Token 刷新失败率
   - 异常 IP 访问
   - 数据库性能

## 相关文档

- [需求文档](../.kiro/specs/multi-tenant-auth/requirements.md)
- [设计文档](../.kiro/specs/multi-tenant-auth/design.md)
- [实施计划](../.kiro/specs/multi-tenant-auth/tasks.md)
- [API 文档](http://localhost:8080/swagger/index.html)

## 技术支持

如有问题或建议，请联系开发团队或提交 Issue。

---

**最后更新**: 2025-10-18
**版本**: 1.0.0
