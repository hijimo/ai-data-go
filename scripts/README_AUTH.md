# 认证系统初始化脚本

本目录包含认证系统的初始化脚本。

## 脚本说明

### auth_migrate.go

数据库迁移脚本，用于创建认证系统所需的数据库表。

**功能**：

- 创建 `tenants` 表（租户表）
- 创建 `users` 表（用户表）
- 创建 `refresh_tokens` 表（刷新令牌表）
- 创建 `auth_audit` 表（认证审计日志表）
- 创建必要的索引和约束

**使用方法**：

```bash
# 方式1：使用 Makefile
make auth-migrate

# 方式2：直接运行
go run scripts/auth_migrate.go
```

### init_auth.go

初始化脚本，用于创建默认租户和管理员用户。

**功能**：

- 创建默认租户（如果不存在）
- 创建管理员用户（如果不存在）
- 显示初始化信息和安全提示

**默认账户信息**：

- 邮箱: `admin@example.com`
- 密码: `Admin@123456`
- 角色: `admin`, `user`

**使用方法**：

```bash
# 方式1：使用 Makefile
make auth-init

# 方式2：直接运行
go run scripts/init_auth.go
```

## 快速开始

### 完整初始化流程

```bash
# 1. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，配置数据库连接等参数

# 2. 执行完整初始化（推荐）
make auth-setup

# 或者分步执行
make auth-migrate  # 先执行迁移
make auth-init     # 再初始化数据
```

### 仅执行迁移

如果只需要创建表结构，不需要初始化数据：

```bash
make auth-migrate
```

### 仅初始化数据

如果表已存在，只需要创建默认租户和管理员：

```bash
make auth-init
```

## 注意事项

1. **首次运行前**：
   - 确保已配置 `.env` 文件
   - 确保数据库已创建并可连接
   - 确保数据库用户有足够的权限

2. **重复运行**：
   - `auth_migrate.go` 可以安全地重复运行（使用 GORM AutoMigrate）
   - `init_auth.go` 会检查是否已存在租户和用户，避免重复创建

3. **安全提示**：
   - 首次登录后请立即修改默认密码
   - 不要在生产环境使用默认密码
   - 建议自定义初始化脚本中的默认值

## 自定义配置

如需自定义默认租户和管理员信息，请编辑 `init_auth.go` 中的常量：

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

## 故障排除

### 问题：数据库连接失败

**解决方案**：

1. 检查 `.env` 文件中的数据库配置
2. 确保数据库服务正在运行
3. 验证数据库用户名和密码
4. 检查网络连接和防火墙设置

### 问题：表已存在错误

**解决方案**：

1. 如果需要重新创建表，先删除现有表
2. 或者跳过迁移，直接运行 `make auth-init`

### 问题：管理员用户已存在

**解决方案**：

- 脚本会自动检测并跳过用户创建
- 如需重置密码，请使用密码修改 API 或直接在数据库中更新

## 相关文档

- [完整使用指南](../docs/AUTH_SETUP.md)
- [需求文档](../.kiro/specs/multi-tenant-auth/requirements.md)
- [设计文档](../.kiro/specs/multi-tenant-auth/design.md)
- [实施计划](../.kiro/specs/multi-tenant-auth/tasks.md)

## 技术支持

如有问题或建议，请查看完整文档或联系开发团队。
