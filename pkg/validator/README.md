# 参数验证器

本包提供了统一的参数验证功能，基于 `go-playground/validator/v10` 库实现。

## 功能特性

- ✅ 结构体标签验证
- ✅ 自定义验证规则支持
- ✅ 中文错误消息
- ✅ 格式化的错误输出
- ✅ 支持常见验证规则（必填、范围、格式等）

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "genkit-ai-service/pkg/validator"
)

type ChatRequest struct {
    Message     string   `json:"message" validate:"required"`
    Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
    MaxTokens   *int     `json:"maxTokens,omitempty" validate:"omitempty,gt=0"`
}

func main() {
    // 创建请求
    temp := 0.7
    req := ChatRequest{
        Message:     "你好",
        Temperature: &temp,
    }
    
    // 验证请求
    if errors := validator.ValidateStruct(req); errors != nil {
        for _, err := range errors {
            fmt.Printf("字段 %s 验证失败: %s\n", err.Field, err.Message)
        }
        return
    }
    
    fmt.Println("验证通过")
}
```

### 使用自定义验证器实例

```go
// 创建验证器实例
v := validator.New()

// 验证结构体
err := v.Validate(data)
if err != nil {
    // 格式化错误
    errors := v.FormatErrors(err)
    // 处理错误...
}
```

### 使用默认验证器

```go
// 直接使用包级别函数
err := validator.Validate(data)
errors := validator.ValidateStruct(data)
```

## 支持的验证标签

### 必填字段

```go
type Request struct {
    Name string `validate:"required"`
}
```

### 数值范围

```go
type Options struct {
    Temperature float64 `validate:"gte=0,lte=2"`      // 大于等于0，小于等于2
    MaxTokens   int     `validate:"gt=0,lte=10000"`   // 大于0，小于等于10000
    Age         int     `validate:"min=0,max=150"`    // 最小0，最大150
}
```

### 字符串长度

```go
type User struct {
    Username string `validate:"min=3,max=20"`  // 长度3-20个字符
    Password string `validate:"min=8"`         // 最小8个字符
}
```

### 格式验证

```go
type Contact struct {
    Email string `validate:"email"`           // 邮箱格式
    URL   string `validate:"url"`             // URL格式
}
```

### 可选字段

```go
type Request struct {
    // 如果提供了值，则必须在0-2范围内
    Temperature *float64 `validate:"omitempty,gte=0,lte=2"`
}
```

### 枚举值

```go
type Config struct {
    Status string `validate:"oneof=active inactive pending"`
}
```

## 错误处理

### ValidationError 结构

```go
type ValidationError struct {
    Field   string `json:"field"`    // 字段名
    Message string `json:"message"`  // 错误消息
}
```

### 错误消息示例

- `message 是必填字段`
- `temperature 必须大于或等于 0`
- `temperature 必须小于或等于 2`
- `maxTokens 必须大于 0`
- `email 必须是有效的邮箱地址`

## 在 API 处理器中使用

```go
func ChatHandler(w http.ResponseWriter, r *http.Request) {
    var req ChatRequest
    
    // 解析请求体
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        // 处理解析错误...
        return
    }
    
    // 验证请求参数
    if errors := validator.ValidateStruct(req); errors != nil {
        // 返回验证错误
        response := ResponseData[[]validator.ValidationError]{
            Code:    422,
            Message: "参数验证失败",
            Data:    &errors,
        }
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // 处理业务逻辑...
}
```

## 自定义验证规则

如果需要添加自定义验证规则，可以在 `registerCustomValidations` 函数中注册：

```go
func registerCustomValidations(v *validator.Validate) {
    // 注册自定义验证规则
    v.RegisterValidation("custom_rule", func(fl validator.FieldLevel) bool {
        // 实现验证逻辑
        return true
    })
}
```

## 测试

运行测试：

```bash
go test ./pkg/validator/... -v
```

运行示例：

```bash
go test ./pkg/validator/... -run Example
```

## 相关需求

- 需求 4.1: 参数验证
- 需求 4.4: 参数范围限制

## 依赖

- `github.com/go-playground/validator/v10`: 验证库
