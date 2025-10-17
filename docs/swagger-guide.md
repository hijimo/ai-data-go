# Swagger API 文档使用指南

## 概述

本项目已集成 [swaggo/swag](https://github.com/swaggo/swag) 来自动生成 OpenAPI 3.0 规范的 API 文档。

## 快速开始

### 1. 启动服务器

```bash
./bin/server
```

### 2. 访问 Swagger UI

在浏览器中打开：

```
http://localhost:8080/swagger/index.html
```

### 3. 访问 OpenAPI JSON

```
http://localhost:8080/swagger/doc.json
```

## 功能特性

### ✅ 已实现的功能

- **自动生成 API 文档**：通过代码注释自动生成 OpenAPI 规范
- **Swagger UI 集成**：提供交互式 API 文档界面
- **支持泛型**：正确处理 Go 1.18+ 的泛型类型
- **标准响应格式**：统一的 `ResponseData[T]` 响应结构
- **错误响应**：标准化的错误响应格式
- **多语言支持**：API 文档支持中文描述

### 📋 已文档化的接口

1. **GET /api/v1/providers** - 获取所有提供商列表
2. **GET /api/v1/providers/{providerId}** - 获取提供商详情
3. **GET /api/v1/providers/{providerId}/models** - 获取提供商的模型列表
4. **GET /api/v1/providers/{providerId}/models/{modelId}** - 获取模型详情
5. **GET /api/v1/providers/{providerId}/models/{modelId}/parameter-rules** - 获取模型参数规则

## 开发指南

### 添加新的 API 接口文档

在 Handler 函数上方添加 Swagger 注释：

```go
// GetProviders 处理 GET /providers 请求
// @Summary 获取所有提供商列表
// @Description 获取系统中所有可用的模型提供商列表
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseData[[]model.Provider] "成功返回提供商列表"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /providers [get]
func (h *ProviderHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
    // 实现代码...
}
```

### 常用注释标签

- `@Summary` - 接口简短描述
- `@Description` - 接口详细描述
- `@Tags` - 接口分组标签
- `@Accept` - 接受的内容类型
- `@Produce` - 返回的内容类型
- `@Param` - 参数定义
- `@Success` - 成功响应
- `@Failure` - 错误响应
- `@Router` - 路由路径和方法

### 参数类型

```go
// 路径参数
// @Param providerId path string true "提供商ID" example(gemini)

// 查询参数
// @Param page query int false "页码" default(1)

// 请求体
// @Param request body model.CreateRequest true "创建请求"
```

### 重新生成文档

修改代码注释后，需要重新生成 Swagger 文档：

```bash
# 使用 swag 命令生成文档
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# 或者使用完整路径
~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

### 响应结构示例

#### 成功响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "gemini",
    "provider": "gemini",
    "label": {
      "en_US": "Google Gemini",
      "zh_Hans": "谷歌 Gemini"
    }
  }
}
```

#### 错误响应

```json
{
  "code": 404,
  "message": "提供商不存在"
}
```

#### 分页响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [...],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 100,
    "totalPage": 10
  }
}
```

## 配置说明

### main.go 配置

在 `cmd/server/main.go` 中的全局配置：

```go
// @title Genkit AI Service API
// @version 1.0.0
// @description AI 模型提供商管理服务 API 文档

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @tag.name providers
// @tag.description 模型提供商管理接口
```

### 修改配置

如果需要修改 API 文档的基本信息（如标题、版本、主机地址等），请编辑 `cmd/server/main.go` 文件中的注释，然后重新生成文档。

## 依赖包

项目使用了以下 Swagger 相关的包：

```go
import (
    _ "genkit-ai-service/docs" // Swagger 文档
    httpSwagger "github.com/swaggo/http-swagger"
)
```

在 `go.mod` 中：

```
github.com/swaggo/swag v1.16.6
github.com/swaggo/http-swagger v1.3.4
github.com/swaggo/files v1.0.1
```

## 注意事项

### 泛型支持

- ✅ 支持：`ResponseData[T]`、`ResponseData[[]T]`
- ❌ 不支持：`ResponseData[interface{}]`（使用 `ErrorResponse` 代替）

### interface{} 字段

对于 `interface{}` 类型的字段（如 `ParameterRule.Default`），不要添加 `example` 标签，因为 Swagger 无法确定具体类型。

### 文档更新

每次修改 API 接口或注释后，都需要重新运行 `swag init` 命令来更新文档。建议将此命令添加到构建脚本中。

## 故障排除

### 问题：swag 命令找不到

**解决方案**：

```bash
# 安装 swag 工具
go install github.com/swaggo/swag/cmd/swag@latest

# 使用完整路径
~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

### 问题：文档没有更新

**解决方案**：

1. 删除 `docs` 目录下的生成文件
2. 重新运行 `swag init` 命令
3. 重新编译并启动服务器

### 问题：泛型类型解析错误

**解决方案**：

- 确保使用具体类型而不是 `interface{}`
- 对于错误响应，使用 `model.ErrorResponse` 而不是 `model.ResponseData[interface{}]`

## 最佳实践

1. **保持注释更新**：修改接口时同步更新 Swagger 注释
2. **使用示例值**：为参数和响应添加 `example` 标签
3. **详细描述**：提供清晰的接口描述和参数说明
4. **错误处理**：为所有可能的错误情况添加 `@Failure` 注释
5. **分组管理**：使用 `@Tags` 对接口进行合理分组
6. **版本控制**：在 `@version` 中记录 API 版本变更

## 参考资源

- [Swaggo 官方文档](https://github.com/swaggo/swag)
- [OpenAPI 3.0 规范](https://swagger.io/specification/)
- [Swagger UI 文档](https://swagger.io/tools/swagger-ui/)
