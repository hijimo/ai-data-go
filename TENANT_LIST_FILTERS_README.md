# 租户列表过滤功能说明

## 功能概述

为租户列表接口添加了两个过滤条件：

1. **租户名称模糊搜索**：支持按租户名称进行模糊匹配
2. **租户状态过滤**：支持按启用/禁用状态进行过滤

## API 接口

### 端点

```
GET /api/v1/tenants
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| pageNo | int | 否 | 页码（从1开始，默认1） | 1 |
| pageSize | int | 否 | 每页大小（1-100，默认20） | 20 |
| name | string | 否 | 租户名称模糊搜索 | "示例" |
| status | boolean | 否 | 租户状态过滤（true=启用，false=禁用） | true |

### 请求示例

#### 1. 基本列表查询（无过滤）

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 2. 按名称模糊搜索

```bash
# 搜索名称包含"平台"的租户
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10&name=平台" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 3. 按状态过滤

```bash
# 查询所有启用的租户
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10&status=true" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 查询所有禁用的租户
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10&status=false" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 4. 组合过滤

```bash
# 搜索名称包含"测试"且状态为启用的租户
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10&name=测试&status=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 响应示例

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "平台租户",
        "domain": "platform.local",
        "type": "system",
        "status": true,
        "metadata": {},
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-01T00:00:00Z"
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 1,
    "totalPage": 1
  }
}
```

## 权限说明

### 平台管理员（system_admin）

- 可以查看所有租户列表
- 可以使用所有过滤条件
- 返回符合过滤条件的所有租户

### 租户管理员（tenant_admin）

- 只能查看自己所属的租户信息
- **所有过滤参数会被忽略**
- 始终只返回当前用户所属的租户（单条记录）

## 实现细节

### 1. Handler 层（tenant_handler.go）

- 解析查询参数 `name` 和 `status`
- 验证 `status` 参数格式（必须是 "true" 或 "false"）
- 构建 `TenantListFilter` 对象传递给服务层

### 2. Service 层（tenant_service.go）

- 新增 `ListWithFilter` 方法
- 新增 `TenantListFilter` 结构体
- 实现角色权限验证：
  - 租户管理员：忽略过滤条件，只返回自己的租户
  - 平台管理员：应用过滤条件查询

### 3. Repository 层（tenant_repository.go）

- 新增 `ListWithFilter` 方法
- 使用 GORM 动态构建查询条件
- 租户名称使用 `ILIKE` 进行大小写不敏感的模糊匹配
- 租户状态使用精确匹配

### 查询逻辑

```go
// 基础条件：未删除的租户
query := db.Where("is_deleted = ?", false)

// 名称模糊搜索（大小写不敏感）
if name != "" {
    query = query.Where("name ILIKE ?", "%"+name+"%")
}

// 状态过滤
if status != nil {
    query = query.Where("status = ?", *status)
}
```

## 测试

### 运行测试脚本

```bash
./test_tenant_list_filters.sh
```

测试脚本会执行以下测试：

1. 基本列表查询（无过滤条件）
2. 租户名称模糊搜索
3. 租户状态过滤
4. 组合过滤（名称 + 状态）
5. 无效参数处理
6. 租户管理员权限验证

### 手动测试步骤

#### 准备工作

1. 启动服务：`make run` 或 `go run cmd/server/main.go`
2. 使用平台管理员账户登录获取 token

#### 测试用例

**测试1：基本列表查询**

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

预期：返回所有租户的分页列表

**测试2：名称模糊搜索**

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?name=平台" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

预期：返回名称包含"平台"的租户

**测试3：状态过滤**

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?status=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

预期：返回所有启用的租户

**测试4：组合过滤**

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?name=测试&status=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

预期：返回名称包含"测试"且状态为启用的租户

**测试5：无效参数**

```bash
curl -X GET "http://localhost:8080/api/v1/tenants?status=invalid" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

预期：返回 400 错误，提示"无效的状态参数"

## 注意事项

1. **大小写不敏感**：租户名称搜索使用 `ILIKE`，支持大小写不敏感的模糊匹配
2. **参数验证**：`status` 参数必须是 "true" 或 "false"，其他值会返回 400 错误
3. **权限隔离**：租户管理员的过滤参数会被忽略，确保租户数据隔离
4. **性能考虑**：建议在 `tenants` 表的 `name` 字段上创建索引以提升搜索性能

## 数据库索引建议

为了提升查询性能，建议添加以下索引：

```sql
-- 租户名称索引（支持模糊搜索）
CREATE INDEX idx_tenants_name ON tenants USING gin(name gin_trgm_ops);

-- 租户状态索引
CREATE INDEX idx_tenants_status ON tenants(status) WHERE is_deleted = false;

-- 组合索引（状态 + 创建时间）
CREATE INDEX idx_tenants_status_created ON tenants(status, created_at DESC) WHERE is_deleted = false;
```

注意：使用 `gin_trgm_ops` 需要启用 PostgreSQL 的 `pg_trgm` 扩展：

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

## 相关文件

- `internal/api/handler/tenant_handler.go` - Handler 层实现
- `internal/service/auth/tenant_service.go` - Service 层实现
- `internal/repository/tenant_repository.go` - Repository 层实现
- `test_tenant_list_filters.sh` - 测试脚本

## 更新日志

### 2024-01-XX

- ✅ 添加租户名称模糊搜索功能
- ✅ 添加租户状态过滤功能
- ✅ 支持组合过滤条件
- ✅ 添加参数验证
- ✅ 更新 Swagger 文档
- ✅ 创建测试脚本
