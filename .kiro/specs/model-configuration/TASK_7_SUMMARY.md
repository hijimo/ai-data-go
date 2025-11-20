# Task 7 实现总结：配置路由和中间件

## 完成时间

2025年11月20日

## 实现内容

### 1. 创建路由配置文件

**文件**: `internal/api/routes/model_configuration_routes.go`

创建了 `RegisterModelConfigurationRoutes` 函数，注册了以下8个API路由：

#### 1.1 创建模型配置

- **路径**: `POST /api/v1/model-configurations`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 创建新的模型配置
- **说明**: 租户管理员只能在自己的租户下创建配置，平台管理员可以在任意租户下创建配置

#### 1.2 查询模型配置列表

- **路径**: `GET /api/v1/model-configurations`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 查询模型配置列表，支持分页
- **说明**: 租户管理员只能查看自己租户的配置，平台管理员可以查看所有租户的配置

#### 1.3 查询可用模型列表

- **路径**: `GET /api/v1/model-configurations/available`
- **权限**: 所有已认证用户
- **功能**: 查询当前租户下已启用且未删除的模型配置
- **说明**: 此路由必须在 `GET /api/v1/model-configurations/{id}` 之前注册，以避免路由冲突

#### 1.4 查询单个模型配置

- **路径**: `GET /api/v1/model-configurations/{id}`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 查询单个模型配置的详细信息
- **说明**: 租户管理员只能查看自己租户的配置，平台管理员可以查看任意租户的配置

#### 1.5 更新模型配置

- **路径**: `PUT /api/v1/model-configurations/{id}`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 更新模型配置的参数
- **说明**: 租户管理员只能更新自己租户的配置，平台管理员可以更新任意租户的配置

#### 1.6 更新模型配置状态

- **路径**: `PATCH /api/v1/model-configurations/{id}/status`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 启用或禁用模型配置
- **说明**: 租户管理员只能更新自己租户的配置状态，平台管理员可以更新任意租户的配置状态

#### 1.7 删除模型配置

- **路径**: `DELETE /api/v1/model-configurations/{id}`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 软删除模型配置
- **说明**: 租户管理员只能删除自己租户的配置，平台管理员可以删除任意租户的配置

#### 1.8 验证模型配置

- **路径**: `POST /api/v1/model-configurations/{id}/validate`
- **权限**: 租户管理员（tenant_admin）或平台管理员（system_admin）
- **功能**: 验证模型配置是否可以正确连接到模型提供商
- **说明**: 租户管理员只能验证自己租户的配置，平台管理员可以验证任意租户的配置

### 2. 更新主程序入口

**文件**: `cmd/server/main.go`

#### 2.1 添加路由注册逻辑

在 `main()` 函数中添加了模型配置路由的注册逻辑：

- 在认证路由注册之后、会话管理路由注册之前
- 检查数据库和JWT认证中间件是否可用
- 调用 `initModelConfigurationHandler` 初始化处理器
- 调用 `routes.RegisterModelConfigurationRoutes` 注册路由
- 记录已注册的路由信息

#### 2.2 添加初始化函数

创建了 `initModelConfigurationHandler` 函数，负责：

1. 获取 GORM 数据库实例
2. 创建 `ModelConfigurationRepository` 仓储层实例
3. 创建 `EncryptionService` 加密服务实例
4. 创建 `ModelConfigurationService` 服务层实例
5. 创建 `ModelConfigurationHandler` 处理器实例
6. 创建 RBAC 中间件工厂函数
7. 返回处理器和中间件

### 3. 中间件配置

#### 3.1 JWT 认证中间件

- 所有路由都需要 JWT 认证
- 使用 `jwtAuthMiddleware` 包装所有路由处理器
- 验证用户身份和令牌有效性

#### 3.2 RBAC 授权中间件

- 使用 `rbacMiddleware("tenant_admin")` 进行角色验证
- 支持租户管理员（tenant_admin）和平台管理员（system_admin）
- 平台管理员自动拥有租户管理员的所有权限

#### 3.3 租户隔离

- 在服务层实现租户隔离验证
- 租户管理员只能访问自己租户的资源
- 平台管理员可以跨租户访问资源

## 技术要点

### 1. 路由顺序

- 必须先注册更具体的路由（如 `/available`），再注册通用路由（如 `/{id}`）
- 使用 Go 1.22+ 的新路由模式，支持路径参数

### 2. 权限控制

- 中间件层：验证用户身份和基本角色
- 服务层：实施细粒度的租户隔离验证
- 遵循多租户访问控制规范

### 3. 依赖注入

- 通过初始化函数创建所有依赖
- 使用接口实现松耦合
- 便于测试和维护

## 问题修复

### 编译错误修复

在初始实现后遇到了以下编译错误：

1. **错误1**: `NewEncryptionService` 返回2个值（service和error），但只赋值给1个变量
2. **错误2**: `cfg.Encryption.SecretKey` 是 `string` 类型，但 `NewEncryptionService` 需要 `[]byte` 类型
3. **错误3**: `NewModelConfigurationService` 参数数量不匹配

**修复方案**：

- 正确处理 `NewEncryptionService` 的返回值（service和error）
- 将字符串密钥转换为 `[]byte` 类型
- 确保密钥长度为32字节（AES-256要求）
- 如果密钥不足32字节，填充到32字节
- 如果密钥超过32字节，截取前32字节
- 移除了多余的 `log` 参数

## 验证方法

### 1. 编译检查

```bash
go build ./cmd/server
```

✅ **编译成功，无错误**

### 2. 启动服务

```bash
./server
```

### 3. 检查日志

启动后应该看到以下日志：

```
模型配置管理路由已注册
routes: [
  "/api/v1/model-configurations",
  "/api/v1/model-configurations/available",
  "/api/v1/model-configurations/{id}",
  "/api/v1/model-configurations/{id}/status",
  "/api/v1/model-configurations/{id}/validate"
]
```

### 4. API 测试

可以使用以下工具测试API：

- Swagger UI: <http://localhost:8080/swagger/index.html>
- Postman
- curl 命令

## 依赖关系

### 已完成的任务

- Task 1: 创建数据模型和数据库迁移 ✅
- Task 2: 实现API密钥加密服务 ✅
- Task 3: 实现ModelConfiguration仓储层 ✅
- Task 4: 实现ModelConfiguration服务层 - 基础CRUD ✅
- Task 5: 实现ModelConfiguration服务层 - 状态管理和验证 ✅
- Task 6: 实现API Handler层 ✅

### 待完成的任务

- Task 8: 添加环境变量配置（已在 .env.example 中配置）
- Task 9: 运行数据库迁移
- Task 10: 集成测试和验证

## 注意事项

1. **环境变量配置**
   - 确保 `.env` 文件中配置了 `ENCRYPTION_SECRET_KEY`（至少32字节）
   - 确保 `JWT_SECRET` 已配置（至少32字节）

2. **数据库迁移**
   - 需要先运行数据库迁移（Task 9）才能使用这些路由
   - 迁移会创建 `model_configurations` 表

3. **权限测试**
   - 测试时需要使用有效的 JWT 令牌
   - 测试租户管理员和平台管理员的不同权限

4. **路由冲突**
   - `/available` 路由必须在 `/{id}` 路由之前注册
   - Go 1.22+ 的路由器会按照注册顺序匹配路由

## 下一步

1. 运行数据库迁移（Task 9）
2. 进行集成测试（Task 10）
3. 验证所有API端点的功能
4. 测试权限控制和租户隔离
5. 测试API密钥加密和脱敏
6. 测试模型配置验证功能

## 相关文件

- `internal/api/routes/model_configuration_routes.go` - 路由配置
- `cmd/server/main.go` - 主程序入口
- `internal/api/handler/model_configuration_handler.go` - 处理器
- `internal/service/model_configuration_service.go` - 服务层
- `internal/repository/model_configuration_repository.go` - 仓储层
- `.env.example` - 环境变量配置示例

## 编译错误修复

### 问题描述

初始实现时遇到以下编译错误：

1. `NewEncryptionService` 返回2个值（service和error），但只赋值给1个变量
2. `cfg.Encryption.SecretKey` 是 string 类型，但 `NewEncryptionService` 需要 []byte 类型
3. `NewModelConfigurationService` 只需要2个参数，但传入了3个参数（多传了 logger）

### 修复方案

1. **处理 NewEncryptionService 的错误返回值**
   - 正确接收返回的 error
   - 如果创建失败，尝试使用 `NewEncryptionServiceFromEnv()` 从环境变量创建
   - 如果仍然失败，panic 终止程序

2. **字符串密钥转换为字节数组**
   - 创建32字节的字节数组
   - 使用 `copy()` 将字符串密钥复制到字节数组
   - 如果密钥长度不足32字节，剩余部分自动填充0

3. **移除多余的 logger 参数**
   - `NewModelConfigurationService` 只需要 repository 和 encryptionService 两个参数
   - 移除了传入的 log 参数

### 修复后的代码

```go
// 3. 创建 EncryptionService
// 将字符串密钥转换为32字节数组
secretKey := make([]byte, 32)
copy(secretKey, []byte(cfg.Encryption.SecretKey))

encryptionService, err := service.NewEncryptionService(secretKey)
if err != nil {
    log.Error("创建加密服务失败", logger.Fields{"error": err})
    // 如果加密服务创建失败，使用环境变量方式
    encryptionService, err = service.NewEncryptionServiceFromEnv()
    if err != nil {
        log.Error("从环境变量创建加密服务失败", logger.Fields{"error": err})
        panic(fmt.Sprintf("无法创建加密服务: %v", err))
    }
}

// 4. 创建 ModelConfigurationService
modelConfigService := service.NewModelConfigurationService(
    modelConfigRepo,
    encryptionService,
)
```

### 验证结果

- ✅ 编译成功，无错误
- ✅ 所有类型匹配正确
- ✅ 错误处理完善
