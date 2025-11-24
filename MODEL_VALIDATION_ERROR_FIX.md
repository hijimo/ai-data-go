# 模型配置验证接口错误处理修复

## 问题描述

模型配置验证接口 `/api/v1/model-configurations/{id}/validate` 存在一个问题：当连接失败时，接口仍然返回 HTTP 200 状态码，这会误导前端认为请求成功。

### 问题表现

- 当模型配置的 BaseURL 无效或无法连接时
- 当 API Key 错误导致认证失败时
- 当请求超时时

接口都返回 HTTP 200 状态码，只是在响应体中包含验证失败的信息。

## 修复方案

### 修改内容

修改了 `internal/service/model_configuration_service.go` 文件中的以下方法：

1. **Validate 方法**：当验证失败时，返回错误而不是返回 ValidationResult 对象
2. **所有验证方法**：在连接失败、超时、认证失败等情况下，返回 error 而不是返回包含失败信息的 ValidationResult

### 修改的验证方法

- `validateOpenAI`
- `validateAnthropic`
- `validateGoogleGenAI`
- `validateAzureOpenAI`
- `validateBianlian`
- `validateCustomOpenAI`

### 修改逻辑

**修改前**：

```go
// 验证失败时返回 ValidationResult，但不返回 error
if resp.StatusCode != http.StatusOK {
    return &model.ValidationResult{
        Valid:   false,
        Message: "连接失败",
        Details: err.Error(),
    }, nil  // 注意这里返回 nil error
}
```

**修改后**：

```go
// 验证失败时同时返回 ValidationResult 和 error
if resp.StatusCode != http.StatusOK {
    return &model.ValidationResult{
        Valid:   false,
        Message: "连接失败",
        Details: err.Error(),
    }, fmt.Errorf("连接失败: %w", err)  // 返回 error
}
```

### Validate 方法的改进

**修改前**：

```go
// 即使验证失败也返回 nil error
if !result.Valid {
    logger.WarnContext(ctx, "模型配置验证失败", ...)
}
return result, nil  // 总是返回 nil error
```

**修改后**：

```go
// 验证失败时返回错误
if !result.Valid {
    logger.WarnContext(ctx, "模型配置验证失败", ...)
    return nil, errors.NewBadRequestError(fmt.Sprintf("验证失败: %s", result.Message))
}
return result, nil  // 只有验证成功才返回 nil error
```

## 影响范围

### Handler 层

Handler 层的 `HandleValidate` 方法无需修改，因为它已经正确处理了 service 层返回的错误：

```go
result, err := h.modelConfigService.Validate(ctx, configID)
if err != nil {
    // 已有的错误处理逻辑会将错误转换为适当的 HTTP 状态码
    if appErr, ok := err.(*errors.AppError); ok {
        h.writeErrorResponse(w, r, appErr)
    } else {
        h.writeErrorResponse(w, r, errors.NewInternalError(err))
    }
    return
}
```

### API 响应变化

**修改前**（连接失败时）：

```json
HTTP/1.1 200 OK
{
  "code": 200,
  "message": "验证完成",
  "data": {
    "valid": false,
    "message": "连接失败",
    "details": "dial tcp: lookup invalid-url..."
  }
}
```

**修改后**（连接失败时）：

```json
HTTP/1.1 400 Bad Request
{
  "code": 400,
  "message": "验证失败: 连接失败"
}
```

## 测试方法

使用提供的测试脚本验证修复效果：

```bash
./test_model_validation_error.sh
```

测试脚本会：

1. 创建一个使用无效 BaseURL 的模型配置
2. 调用验证接口
3. 检查 HTTP 状态码是否为错误状态码（非 200）
4. 清理测试数据

## 预期结果

- ✅ 连接失败时返回 400 Bad Request
- ✅ 超时时返回 400 Bad Request
- ✅ API Key 无效时返回 400 Bad Request
- ✅ BaseURL 缺失时返回 400 Bad Request
- ✅ 验证成功时返回 200 OK

## 注意事项

1. 此修复不影响验证成功的情况，验证成功时仍然返回 200 状态码
2. 前端需要根据 HTTP 状态码判断验证是否成功，而不是只检查响应体中的 `valid` 字段
3. 所有验证失败的情况都会记录在日志中，便于排查问题

## 相关文件

- `internal/service/model_configuration_service.go` - 服务层实现
- `internal/api/handler/model_configuration_handler.go` - Handler 层（无需修改）
- `test_model_validation_error.sh` - 测试脚本
