# 认证系统快速参考

## 快速开始

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 完整初始化（推荐）
make auth-setup

# 3. 启动服务
make run
```

## 默认管理员账户

- **邮箱**: `admin@example.com`
- **密码**: `Admin@123456`
- **角色**: `admin`, `user`

⚠️ **首次登录后请立即修改密码！**

## 常用命令

### 初始化命令

```bash
make auth-migrate    # 执行数据库迁移
make auth-init       # 初始化租户和管理员
make auth-setup      # 完整初始化（迁移+初始化）
```

### 服务命令

```bash
make build          # 编译项目
make run            # 运行服务器
make dev            # 开发模式（生成文档并运行）
make swagger        # 生成 Swagger 文档
```

## API 端点

### 认证相关

```
POST   /api/v1/auth/register          # 用户注册
POST   /api/v1/auth/login             # 用户登录
POST   /api/v1/auth/refresh           # 刷新 Token
POST   /api/v1/auth/logout            # 用户注销
POST   /api/v1/auth/change-password   # 修改密码
GET    /api/v1/auth/me                # 获取当前用户信息
```

### 租户管理（需要管理员权限）

```
POST   /api/v1/tenants                # 创建租户
GET    /api/v1/tenants                # 列出租户
GET    /api/v1/tenants/:id            # 获取租户详情
PUT    /api/v1/tenants/:id            # 更新租户
DELETE /api/v1/tenants/:id            # 删除租户
```

### 用户管理（需要租户管理员权限）

```
POST   /api/v1/users                  # 创建用户
GET    /api/v1/users                  # 列出用户
GET    /api/v1/users/:id              # 获取用户详情
PUT    /api/v1/users/:id              # 更新用户
DELETE /api/v1/users/:id              # 删除用户
```

## 登录示例

```bash
# 获取租户ID（从初始化输出或数据库查询）
TENANT_ID="your-tenant-id"

# 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin@123456"
  }'

# 保存返回的 access_token
ACCESS_TOKEN="your-access-token"

# 使用 Token 访问受保护的 API
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## 环境变量配置

### 必需配置

```bash
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=genkit_ai_service

# JWT 密钥（生产环境必须修改）
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

### 可选配置

```bash
# Token 有效期
ACCESS_TOKEN_TTL=60m        # Access Token 有效期（默认 60 分钟）
REFRESH_TOKEN_TTL=720h      # Refresh Token 有效期（默认 30 天）

# 密码策略
BCRYPT_COST=12              # Bcrypt 成本因子（默认 12）
PASSWORD_MIN_LENGTH=8       # 密码最小长度（默认 8）

# 登录安全
MAX_LOGIN_ATTEMPTS=5        # 最大登录尝试次数（默认 5）
LOGIN_ATTEMPT_WINDOW=15m    # 登录尝试时间窗口（默认 15 分钟）

# Token 策略
ENABLE_REFRESH_ROTATION=true  # 启用 Token 轮换（默认 true）

# 租户识别
TENANT_IDENTIFY_STRATEGY=header  # 租户识别策略（默认 header）
```

## 数据库表

- `tenants` - 租户表
- `users` - 用户表
- `refresh_tokens` - 刷新令牌表
- `auth_audit` - 认证审计日志表

## 故障排除

### 数据库连接失败

```bash
# 检查数据库是否运行
psql -h localhost -U postgres -d genkit_ai_service

# 检查环境变量
cat .env | grep DB_
```

### 表已存在

```sql
-- 删除现有表（谨慎操作！）
DROP TABLE IF EXISTS auth_audit CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- 重新运行迁移
make auth-migrate
```

### 忘记管理员密码

```bash
# 方式1：删除用户记录，重新初始化
# 在数据库中执行：
# DELETE FROM users WHERE email = 'admin@example.com';

# 方式2：重新运行初始化（会跳过已存在的用户）
make auth-init
```

## 安全检查清单

- [ ] 已修改默认管理员密码
- [ ] 已配置强随机的 JWT_SECRET
- [ ] 已启用 HTTPS（生产环境）
- [ ] 已配置合适的 Token 有效期
- [ ] 已启用 Token 轮换
- [ ] 已设置登录失败限制
- [ ] 已配置数据库备份
- [ ] 已启用审计日志

## 相关文档

- **完整指南**: [docs/AUTH_SETUP.md](AUTH_SETUP.md)
- **脚本说明**: [scripts/README_AUTH.md](../scripts/README_AUTH.md)
- **需求文档**: [.kiro/specs/multi-tenant-auth/requirements.md](../.kiro/specs/multi-tenant-auth/requirements.md)
- **设计文档**: [.kiro/specs/multi-tenant-auth/design.md](../.kiro/specs/multi-tenant-auth/design.md)
- **API 文档**: <http://localhost:8080/swagger/index.html>

## 技术支持

如有问题，请查看完整文档或联系开发团队。

---

**版本**: 1.0.0  
**最后更新**: 2025-10-18
