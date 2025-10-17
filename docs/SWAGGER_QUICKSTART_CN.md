# Swagger 快速开始指南

## 🚀 5 分钟快速上手

### 第 1 步：生成 Swagger 文档

```bash
make swagger
```

或者手动执行：

```bash
~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

### 第 2 步：启动服务器

```bash
make run
```

或者：

```bash
./bin/server
```

### 第 3 步：访问 Swagger UI

在浏览器中打开：

```
http://localhost:8080/swagger/index.html
```

## 🎯 主要功能

### 1. 查看 API 文档

Swagger UI 提供了所有 API 接口的详细文档，包括：

- 📝 接口描述
- 📥 请求参数
- 📤 响应格式
- 🔍 数据模型

### 2. 在线测试 API

点击任意接口，然后点击 "Try it out" 按钮：

1. 填写必要的参数
2. 点击 "Execute" 执行请求
3. 查看实际的响应结果

### 3. 查看数据模型

在页面底部的 "Schemas" 部分可以查看所有数据结构的定义。

## 📋 可用的 API 接口

### 提供商管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/providers` | 获取所有提供商列表 |
| GET | `/api/v1/providers/{providerId}` | 获取提供商详情 |
| GET | `/api/v1/providers/{providerId}/models` | 获取提供商的模型列表 |
| GET | `/api/v1/providers/{providerId}/models/{modelId}` | 获取模型详情 |
| GET | `/api/v1/providers/{providerId}/models/{modelId}/parameter-rules` | 获取模型参数规则 |

## 💡 使用示例

### 示例 1：获取所有提供商

1. 在 Swagger UI 中找到 `GET /api/v1/providers` 接口
2. 点击 "Try it out"
3. 点击 "Execute"
4. 查看返回的提供商列表

### 示例 2：获取 Gemini 提供商详情

1. 找到 `GET /api/v1/providers/{providerId}` 接口
2. 点击 "Try it out"
3. 在 `providerId` 参数中输入 `gemini`
4. 点击 "Execute"
5. 查看 Gemini 提供商的详细信息

### 示例 3：获取 Gemini 的模型列表

1. 找到 `GET /api/v1/providers/{providerId}/models` 接口
2. 点击 "Try it out"
3. 在 `providerId` 参数中输入 `gemini`
4. 点击 "Execute"
5. 查看 Gemini 提供的所有模型

## 🔧 开发者指南

### 添加新接口的文档

在 Handler 函数上方添加注释：

```go
// @Summary 接口简短描述
// @Description 接口详细描述
// @Tags 标签名称
// @Accept json
// @Produce json
// @Param paramName path string true "参数描述" example(示例值)
// @Success 200 {object} ResponseType "成功描述"
// @Failure 400 {object} ErrorResponse "错误描述"
// @Router /path [method]
func (h *Handler) YourHandler(w http.ResponseWriter, r *http.Request) {
    // 实现代码
}
```

### 重新生成文档

修改注释后，运行：

```bash
make swagger
```

然后重启服务器即可看到更新后的文档。

## 📚 更多资源

- [完整的 Swagger 使用指南](./swagger-guide.md)
- [Swaggo 官方文档](https://github.com/swaggo/swag)
- [OpenAPI 规范](https://swagger.io/specification/)

## ❓ 常见问题

### Q: 如何修改 API 文档的标题和描述？

A: 编辑 `cmd/server/main.go` 文件中的注释，然后重新生成文档。

### Q: 文档没有更新怎么办？

A: 确保运行了 `make swagger` 命令，并重启了服务器。

### Q: 如何在生产环境中禁用 Swagger UI？

A: 可以通过环境变量或配置文件控制是否注册 Swagger 路由。

## 🎉 完成

现在你已经掌握了 Swagger 的基本使用方法。开始探索和测试你的 API 吧！
