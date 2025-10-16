---
inclusion: always
---

# API接口返回数据格式规范

## 标准响应格式

所有API接口必须严格遵循以下Go语言定义的响应格式：

### 普通接口数据返回格式

```go
// ResponseData 通用响应数据结构
type ResponseData[T any] struct {
 // 响应代码
 Code int `json:"code"`
 // 响应信息
 Message string `json:"message"`
 // 响应数据
 Data *T `json:"data,omitempty"`
}
```

### 列表数据接口返回格式

```go
// PaginationData 分页数据结构
type PaginationData[T any] struct {
 // 数据
 Data T `json:"data"`
 // 当前页码
 PageNo int `json:"pageNo"`
 // 每页大小
 PageSize int `json:"pageSize"`
 // 总记录数
 TotalCount int `json:"totalCount"`
 // 总页数
 TotalPage int `json:"totalPage"`
}

// ResponsePaginationData 分页响应数据结构
type ResponsePaginationData[T any] struct {
 // 响应代码
 Code int `json:"code"`
 // 响应信息
 Message string `json:"message"`
 // 分页数据
 Data PaginationData[T] `json:"data"`
}
```

## 实施要求

### 接口开发规范

- 所有API接口的返回值必须符合上述格式定义
- 普通数据接口使用 `ResponseData[T]` 格式
- 分页列表接口使用 `ResponsePaginationData[T]` 格式
- 不允许使用其他自定义的响应格式

### 代码实现要求

- 在创建接口时，必须导入并使用这些结构体定义
- 处理器函数的返回类型必须明确声明为相应的响应格式
- 响应数据的构造必须严格按照格式要求
- 使用Go 1.18+的泛型特性来确保类型安全

### 字段说明

- `code`: 响应状态码，用于标识请求处理结果
- `message`: 响应消息，提供操作结果的文字描述
- `data`: 实际返回的业务数据
- `pageNo`: 当前页码（从1开始）
- `pageSize`: 每页显示的记录数
- `totalCount`: 符合条件的总记录数
- `totalPage`: 总页数

### 示例用法

```go
// 普通接口示例
userResponse := ResponseData[User]{
 Code:    200,
 Message: "获取用户信息成功",
 Data:    &userInfo,
}

// 分页接口示例
userListResponse := ResponsePaginationData[[]User]{
 Code:    200,
 Message: "获取用户列表成功",
 Data: PaginationData[[]User]{
  Data:       userList,
  PageNo:     1,
  PageSize:   10,
  TotalCount: 100,
  TotalPage:  10,
 },
}
```

## 注意事项

- 在设计和实现任何API接口时，都必须严格遵循此格式
- 不得随意修改或扩展响应格式结构
- 所有相关的结构体定义应该统一管理，避免重复定义
- 确保Go版本支持泛型（Go 1.18+）
