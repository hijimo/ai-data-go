# 上下文管理路由配置

## 概述

本文档描述了上下文管理、记忆管理和摘要管理的 API 路由配置。

## 已实现的路由

### 1. 上下文管理路由 (Context Routes)

文件：`internal/api/routes/context_routes.go`

| 方法 | 路径 | 描述 | 权限要求 |
|------|------|------|----------|
| POST | `/api/v1/contexts/build` | 构建上下文 | JWT + 租户管理员 |
| GET | `/api/v1/contexts/{sessionId}` | 获取上下文配置 | JWT + 租户管理员 |
| PUT | `/api/v1/contexts/{sessionId}` | 更新上下文配置 | JWT + 租户管理员 |

### 2. 记忆管理路由 (Memory Routes)

文件：`internal/api/routes/memory_routes.go`

| 方法 | 路径 | 描述 | 权限要求 |
|------|------|------|----------|
| POST | `/api/v1/memories/search` | 检索记忆 | JWT + 租户管理员 |
| POST | `/api/v1/memories` | 存储记忆 | JWT + 租户管理员 |
| POST | `/api/v1/memories/cleanup` | 清理记忆 | JWT + 租户管理员 |
| GET | `/api/v1/memories/{id}` | 获取记忆详情 | JWT + 租户管理员 |

### 3. 摘要管理路由 (Summary Routes)

文件：`internal/api/routes/summary_routes.go`

| 方法 | 路径 | 描述 | 权限要求 |
|------|------|------|----------|
| POST | `/api/v1/summaries` | 生成摘要 | JWT + 租户管理员 |
| GET | `/api/v1/summaries/{id}` | 获取摘要详情 | JWT + 租户管理员 |
| GET | `/api/v1/summaries/session/{sessionId}` | 获取会话摘要列表 | JWT + 租户管理员 |
| POST | `/api/v1/summaries/check-trigger` | 检查摘要触发条件 | JWT + 租户管理员 |

## 路由注册

所有路由在 `cmd/server/main.go` 中的 `initSessionHandlers` 函数中初始化并注册。

### 注册流程

1. 初始化所需的 Repository 层（包括 MemoryRepository、ContextRepository 等）
2. 初始化所需的 Service 层（ContextService、MemoryService、SummaryService）
3. 创建对应的 Handler 层（ContextHandler、MemoryHandler、SummaryHandler）
4. 调用路由注册函数，应用 JWT 认证和 RBAC 授权中间件

### 中间件链

所有路由都应用了以下中间件（按顺序）：

1. **JWT 认证中间件** (`jwtAuthMiddleware`)
   - 验证 JWT Token 的有效性
   - 提取用户信息和租户信息

2. **RBAC 授权中间件** (`rbacMiddleware("tenant_admin")`)
   - 验证用户角色
   - 确保用户具有租户管理员或平台管理员权限

## 权限控制

根据多租户访问控制规范：

- **平台管理员 (system_admin)**：可以访问所有租户的数据
- **租户管理员 (tenant_admin)**：只能访问自己租户的数据

所有路由都在 Service 层实现了严格的租户隔离验证。

## 测试

测试文件：`internal/api/routes/context_routes_test.go`

运行测试：

```bash
go test -v ./internal/api/routes -run TestRegisterContextRoutes
```

## 依赖项

### 必需的服务

- **ContextService**: 上下文构建和管理服务
- **MemoryService**: 记忆存储和检索服务
- **SummaryService**: 摘要生成和管理服务

### 可选的服务（当前未完全配置）

- **VectorService**: 向量嵌入服务（用于语义检索）
- **QdrantClient**: Qdrant 向量数据库客户端

注意：VectorService 和 QdrantClient 当前传入 nil 值，需要在配置文件中添加相应配置后才能启用完整的向量检索功能。

## 后续工作

1. 在配置文件中添加 Qdrant 和 Vector 服务的配置
2. 实现完整的向量检索功能
3. 添加更多的集成测试
4. 完善 API 文档（Swagger）

## 相关文件

- `internal/api/routes/context_routes.go` - 上下文管理路由
- `internal/api/routes/memory_routes.go` - 记忆管理路由
- `internal/api/routes/summary_routes.go` - 摘要管理路由
- `internal/api/handler/context_handler.go` - 上下文处理器
- `internal/api/handler/memory_handler.go` - 记忆处理器
- `internal/api/handler/summary_handler.go` - 摘要处理器
- `cmd/server/main.go` - 主程序和路由注册
