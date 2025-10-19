# 设计文档

## 概述

本设计文档描述了如何统一移除项目中所有 `@Router` 注解的 `/api/v1` 前缀。当前项目中存在两种路由格式：

1. 带 `/api/v1` 前缀的路由（如 `auth_handler.go`、`audit_handler.go`、`monitoring_handler.go`）
2. 不带 `/api/v1` 前缀的路由（如 `chat.go`、`session_handler.go`、`provider_handler.go`）

同时，实际的路由注册在 `router.go` 中使用了 `/api/v1` 前缀。本次修改将统一移除 Swagger 注解中的 `/api/v1` 前缀，使注解与实际路由注册保持一致。

## 架构

### 当前架构

```
Handler 文件
├── @Router 注解（Swagger 文档）
│   ├── 部分带 /api/v1 前缀
│   └── 部分不带 /api/v1 前缀
└── 处理器函数

router.go
└── 路由注册（实际 HTTP 路由）
    └── 统一使用 /api/v1 前缀
```

### 目标架构

```
Handler 文件
├── @Router 注解（Swagger 文档）
│   └── 统一不带 /api/v1 前缀
└── 处理器函数

router.go
└── 路由注册（实际 HTTP 路由）
    └── 保持 /api/v1 前缀不变
```

## 组件和接口

### 受影响的文件

根据代码扫描，以下文件包含需要修改的 `@Router` 注解：

#### 1. 认证相关 Handler

- `internal/api/handler/auth_handler.go`
  - `/api/v1/auth/register` → `/auth/register`
  - `/api/v1/auth/login` → `/auth/login`
  - `/api/v1/auth/refresh` → `/auth/refresh`
  - `/api/v1/auth/logout` → `/auth/logout`
  - `/api/v1/auth/change-password` → `/auth/change-password`
  - `/api/v1/auth/unlock-account` → `/auth/unlock-account`
  - `/api/v1/auth/me` → `/auth/me`
  - `/api/v1/auth/verify-email` → `/auth/verify-email`
  - `/api/v1/auth/resend-verification` → `/auth/resend-verification`

#### 2. 审计日志 Handler

- `internal/api/handler/audit_handler.go`
  - `/api/v1/audit/auth` → `/audit/auth`

#### 3. 监控 Handler

- `internal/api/handler/monitoring_handler.go`
  - `/api/v1/monitoring/metrics` → `/monitoring/metrics`
  - 其他监控相关路由

#### 4. 其他 Handler（已经是正确格式，无需修改）

- `internal/api/handler/chat.go` - `/chat`
- `internal/api/handler/chat_stream.go` - `/chat/stream`
- `internal/api/handler/abort.go` - `/chat/abort`
- `internal/api/handler/health.go` - `/health`
- `internal/api/handler/session_handler.go` - `/chat/sessions/*`
- `internal/api/handler/provider_handler.go` - `/providers/*`
- `internal/api/handler/message_handler.go` - 消息相关路由

### 修改模式

对于每个包含 `/api/v1` 前缀的 `@Router` 注解：

**修改前：**

```go
// @Router /api/v1/auth/register [post]
```

**修改后：**

```go
// @Router /auth/register [post]
```

### 不受影响的部分

1. **路由注册代码**：`router.go` 中的实际路由注册保持不变，继续使用 `/api/v1` 前缀
2. **HTTP 方法标识符**：`[get]`、`[post]`、`[patch]`、`[delete]` 等保持不变
3. **其他 Swagger 注解**：`@Summary`、`@Description`、`@Param`、`@Success`、`@Failure` 等保持不变
4. **处理器函数实现**：函数逻辑不需要任何修改

## 数据模型

本次修改不涉及数据模型变更。

## 错误处理

### 潜在问题

1. **Swagger 文档生成失败**
   - 原因：注解格式错误
   - 解决：修改后运行 `swag init` 验证文档生成

2. **路由路径不匹配**
   - 原因：注解路径与实际注册路径不一致
   - 解决：确保 `router.go` 中的路由注册使用完整的 `/api/v1` 前缀

3. **编译错误**
   - 原因：注释格式错误
   - 解决：确保注释语法正确

### 验证步骤

1. 修改完成后运行 `go build` 确保编译通过
2. 运行 `swag init` 重新生成 Swagger 文档
3. 检查生成的 `docs/swagger.json` 和 `docs/swagger.yaml` 文件
4. 启动服务并访问 Swagger UI 验证文档正确性

## 测试策略

### 1. 静态验证

- 使用 `grep` 或 `rg` 搜索所有 `@Router` 注解
- 确认没有遗漏的 `/api/v1` 前缀

### 2. 编译验证

- 运行 `go build ./...` 确保所有包编译通过
- 运行 `go vet ./...` 检查代码问题

### 3. Swagger 文档验证

- 运行 `swag init` 重新生成文档
- 检查生成的文档中路由路径格式
- 访问 Swagger UI 确认文档可正常显示

### 4. 功能验证

- 启动服务
- 使用 Swagger UI 或 curl 测试几个关键 API
- 确认 API 仍然可以正常访问（因为实际路由注册未改变）

## 实施计划

### 阶段 1：准备工作

1. 备份当前代码
2. 扫描所有包含 `@Router /api/v1` 的文件
3. 创建修改清单

### 阶段 2：执行修改

1. 修改 `auth_handler.go` 中的所有 `@Router` 注解
2. 修改 `audit_handler.go` 中的所有 `@Router` 注解
3. 修改 `monitoring_handler.go` 中的所有 `@Router` 注解
4. 修改其他可能包含 `/api/v1` 前缀的 handler 文件

### 阶段 3：验证

1. 运行编译验证
2. 重新生成 Swagger 文档
3. 启动服务并测试
4. 检查 Swagger UI

### 阶段 4：文档更新

1. 更新相关文档（如果有引用路由路径的文档）
2. 提交代码变更

## 回滚计划

如果修改后出现问题：

1. 使用 Git 回滚到修改前的版本
2. 或者手动恢复 `/api/v1` 前缀

## 注意事项

1. **只修改 Swagger 注解**：不要修改实际的路由注册代码
2. **保持一致性**：确保所有 `@Router` 注解都不包含 `/api/v1` 前缀
3. **HTTP 方法不变**：只修改路径部分，不修改 HTTP 方法标识符
4. **文档同步**：修改后需要重新生成 Swagger 文档
5. **测试覆盖**：修改后需要验证 API 功能正常

## 预期结果

修改完成后：

- 所有 `@Router` 注解统一不包含 `/api/v1` 前缀
- Swagger 文档正确生成并显示路由信息
- API 功能正常，可以正常访问
- 代码风格统一，易于维护
