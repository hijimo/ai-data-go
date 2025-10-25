# 用户列表搜索功能说明

## 功能概述

为用户列表接口添加了搜索功能，支持对 `displayName`（显示名称）、`phone`（手机号）、`email`（邮箱）三个字段进行模糊搜索。

## 实现层级

### 1. 仓储层（Repository）

**文件**: `internal/repository/user_repository.go`

**修改内容**:

- `List()` 方法：添加 `search` 参数，支持租户内用户搜索
- `ListAll()` 方法：添加 `search` 参数，支持跨租户用户搜索（平台管理员）

**搜索实现**:

```go
// 添加搜索条件（模糊匹配 displayName、phone、email）
if search != "" {
    searchPattern := "%" + search + "%"
    query = query.Where(
        "display_name ILIKE ? OR phone ILIKE ? OR email ILIKE ?",
        searchPattern, searchPattern, searchPattern,
    )
}
```

**特性**:

- 使用 `ILIKE` 实现不区分大小写的模糊匹配
- 三个字段使用 `OR` 逻辑连接，匹配任一字段即可
- 支持部分匹配（如搜索 "138" 可以匹配 "13800138000"）

### 2. 服务层（Service）

**文件**: `internal/service/auth/user_service.go`

**修改内容**:

- `List()` 方法：添加 `search` 参数，并传递给仓储层

**权限控制**:

- 租户管理员：只能搜索自己租户下的用户
- 平台管理员：可以搜索所有租户或指定租户的用户

### 3. 处理器层（Handler）

**文件**: `internal/api/handler/user_handler.go`

**修改内容**:

- `HandleList()` 方法：从查询参数中获取 `search` 参数

**API 参数**:

- `pageNo`: 页码（从1开始，默认1）
- `pageSize`: 每页大小（1-100，默认20）
- `tenantId`: 租户ID（可选，仅平台管理员可用）
- `search`: 搜索关键词（可选，支持模糊匹配）

## API 使用示例

### 1. 按显示名称搜索

```bash
curl -X GET "http://localhost:8080/api/v1/users?search=张三" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 2. 按手机号搜索

```bash
curl -X GET "http://localhost:8080/api/v1/users?search=13800138000" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. 按邮箱搜索

```bash
curl -X GET "http://localhost:8080/api/v1/users?search=user@example.com" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 4. 部分匹配搜索

```bash
# 搜索所有包含 "test" 的用户（匹配邮箱、显示名称或手机号）
curl -X GET "http://localhost:8080/api/v1/users?search=test" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 5. 结合租户ID和搜索（平台管理员）

```bash
curl -X GET "http://localhost:8080/api/v1/users?tenantId=550e8400-e29b-41d4-a716-446655440000&search=张三" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 6. 结合分页和搜索

```bash
curl -X GET "http://localhost:8080/api/v1/users?search=test&pageNo=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "tenantId": "550e8400-e29b-41d4-a716-446655440000",
        "email": "zhangsan@test.com",
        "displayName": "张三",
        "phone": "13800138001",
        "isActive": true,
        "isAdmin": false,
        "roles": ["user"],
        "createdAt": "2025-01-20T10:00:00Z",
        "updatedAt": "2025-01-20T10:00:00Z"
      }
    ],
    "pageNo": 1,
    "pageSize": 20,
    "totalCount": 1,
    "totalPage": 1
  }
}
```

## 搜索特性

### 1. 不区分大小写

- 搜索 "zhang" 可以匹配 "Zhang"、"ZHANG"、"zhang"

### 2. 模糊匹配

- 搜索 "138" 可以匹配 "13800138000"
- 搜索 "test" 可以匹配 "<test@example.com>"

### 3. 多字段搜索

- 同一个关键词会在 displayName、phone、email 三个字段中搜索
- 只要任一字段匹配即返回该用户

### 4. 空搜索

- 如果 `search` 参数为空或不提供，返回所有用户（受分页限制）

## 权限控制

### 租户管理员

- 只能搜索自己租户下的用户
- `tenantId` 参数会被忽略
- 搜索范围自动限制在当前租户

### 平台管理员

- 可以搜索所有租户的用户
- 可以通过 `tenantId` 参数限制搜索范围
- 不提供 `tenantId` 时搜索所有租户

## 测试脚本

运行测试脚本验证搜索功能：

```bash
./test_user_list_search.sh
```

测试脚本会执行以下测试：

1. 平台管理员登录
2. 创建测试用户（张三、李四、王五）
3. 按 displayName 搜索
4. 按 phone 搜索
5. 按 email 搜索
6. 模糊搜索
7. 部分手机号搜索
8. 结合租户ID和搜索
9. 空搜索
10. 清理测试数据

## 性能考虑

### 数据库索引建议

为了提高搜索性能，建议在数据库中添加以下索引：

```sql
-- 为 displayName 添加索引（支持模糊搜索）
CREATE INDEX idx_users_display_name ON users USING gin (display_name gin_trgm_ops);

-- 为 phone 添加索引（支持模糊搜索）
CREATE INDEX idx_users_phone ON users USING gin (phone gin_trgm_ops);

-- 为 email 添加索引（已存在唯一索引，无需额外添加）
```

**注意**: 使用 `gin_trgm_ops` 需要启用 PostgreSQL 的 `pg_trgm` 扩展：

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

### 搜索优化建议

1. **限制搜索关键词长度**: 建议在前端限制搜索关键词最少2-3个字符
2. **使用分页**: 始终使用分页参数，避免一次性返回大量数据
3. **缓存热门搜索**: 对于频繁搜索的关键词，可以考虑使用缓存

## 注意事项

1. **搜索不区分大小写**: 使用 `ILIKE` 实现，适用于 PostgreSQL
2. **特殊字符**: 搜索关键词中的特殊字符（如 `%`、`_`）会被当作普通字符处理
3. **性能影响**: 模糊搜索可能影响性能，建议添加数据库索引
4. **租户隔离**: 搜索始终遵循租户隔离原则，租户管理员无法搜索其他租户的用户

## 后续优化建议

1. **全文搜索**: 如果需要更强大的搜索功能，可以考虑使用 PostgreSQL 的全文搜索功能
2. **搜索高亮**: 在返回结果中高亮显示匹配的关键词
3. **搜索历史**: 记录用户的搜索历史，提供搜索建议
4. **高级搜索**: 支持多条件组合搜索（如同时指定显示名称和手机号）
