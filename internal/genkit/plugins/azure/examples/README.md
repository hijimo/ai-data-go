# Azure AI Genkit Provider - 示例代码

本目录包含 Azure AI Genkit Provider 的完整示例代码，展示了插件的各种功能和使用场景。

## 目录

- [环境准备](#环境准备)
- [示例列表](#示例列表)
- [运行示例](#运行示例)
- [示例说明](#示例说明)

## 环境准备

在运行示例之前，需要设置以下环境变量：

```bash
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_BASE_URL="https://your-resource.openai.azure.com"
```

你可以从 Azure Portal 获取这些信息：

1. 登录 [Azure Portal](https://portal.azure.com)
2. 导航到你的 Azure OpenAI 资源
3. 在"Keys and Endpoint"部分找到 API Key 和 Endpoint

## 示例列表

### 1. basic_usage.go - 基本使用

展示插件的基本功能：

- ✅ 简单文本生成
- ✅ 多轮对话
- ✅ 使用系统提示
- ✅ 配置生成参数
- ✅ 多模态输入（文本 + 图像）

**适合人群**：初次使用插件的开发者

### 2. streaming_example.go - 流式响应

展示如何使用流式响应：

- ✅ 基本流式响应
- ✅ 收集流式内容
- ✅ 流式响应中处理工具调用
- ✅ 流式响应中的错误处理

**适合人群**：需要实时响应的应用场景

### 3. tool_calling_example.go - 工具调用

展示如何使用工具调用（Function Calling）：

- ✅ 单个工具调用
- ✅ 多个工具调用
- ✅ 工具调用链（多轮对话）
- ✅ 强制工具调用

**适合人群**：需要集成外部功能的应用

### 4. embedder_example.go - 文本嵌入

展示如何使用文本嵌入功能：

- ✅ 单个文档嵌入
- ✅ 批量文档嵌入
- ✅ 多部分文档嵌入

**适合人群**：需要向量化文本的应用（如搜索、推荐）

### 5. error_handling_example.go - 错误处理

展示如何处理各种错误：

- ✅ 配置错误处理
- ✅ 网络错误处理
- ✅ API 错误处理
- ✅ 错误类型判断
- ✅ 错误恢复和重试

**适合人群**：需要健壮错误处理的生产环境

### 6. retry_example.go - 重试和超时

展示如何配置重试和超时：

- ✅ 使用默认配置
- ✅ 自定义重试配置
- ✅ 自定义超时配置
- ✅ 使用上下文超时
- ✅ 禁用重试

**适合人群**：需要精细控制请求行为的应用

## 运行示例

### 方式 1：直接运行

```bash
# 运行基本使用示例
go run internal/genkit/plugins/azure/examples/basic_usage.go

# 运行流式响应示例
go run internal/genkit/plugins/azure/examples/streaming_example.go

# 运行工具调用示例
go run internal/genkit/plugins/azure/examples/tool_calling_example.go

# 运行嵌入示例
go run internal/genkit/plugins/azure/examples/embedder_example.go

# 运行错误处理示例
go run internal/genkit/plugins/azure/examples/error_handling_example.go

# 运行重试示例
go run internal/genkit/plugins/azure/examples/retry_example.go
```

### 方式 2：使用 Makefile（如果项目有）

```bash
make run-azure-examples
```

### 方式 3：编译后运行

```bash
# 编译
go build -o basic_usage internal/genkit/plugins/azure/examples/basic_usage.go

# 运行
./basic_usage
```

## 示例说明

### basic_usage.go

这是最基础的示例，展示了插件的核心功能。

**示例 1：最简单的使用方式**
```go
plugin := &azure.AzureAI{
    APIKey:  apiKey,
    BaseURL: baseURL,
}
plugin.Init(ctx)

model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
    Label: "GPT-4",
})

resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("你好！"),
            },
        },
    },
}, nil)
```

**示例 2：多轮对话**

展示如何维护对话历史，实现多轮对话。

**示例 3：使用系统提示**

展示如何使用系统提示来定义 AI 的行为和角色。

**示例 4：配置生成参数**

展示如何配置 temperature、max_tokens、top_p 等参数。

**示例 5：多模态输入**

展示如何同时发送文本和图像给支持多模态的模型。

### streaming_example.go

展示流式响应的各种用法。

**示例 1：基本流式响应**

实时打印每个数据块，提供即时反馈。

**示例 2：收集流式内容**

在流式接收的同时收集完整内容。

**示例 3：流式响应中处理工具调用**

展示如何在流式响应中检测和处理工具调用。

**示例 4：流式响应中的错误处理**

展示如何在回调函数中处理错误和中断流式响应。

### tool_calling_example.go

展示工具调用的完整流程。

**示例 1：单个工具调用**

完整的工具调用流程：
1. 定义工具
2. 模型决定调用工具
3. 执行工具
4. 返回结果给模型
5. 获取最终响应

**示例 2：多个工具调用**

展示模型如何同时调用多个工具。

**示例 3：工具调用链**

展示复杂的多轮对话场景，如预订航班的完整流程。

**示例 4：强制工具调用**

展示如何使用 `tool_choice` 参数强制模型调用特定工具。

### embedder_example.go

展示文本嵌入的各种场景。

**示例 1：单个文档嵌入**

最基本的嵌入用法。

**示例 2：批量文档嵌入**

一次请求处理多个文档，提高效率。

**示例 3：多部分文档嵌入**

展示如何处理包含多个部分的文档。

### error_handling_example.go

展示如何处理各种错误情况。

**示例 1：配置错误处理**

展示缺少必需配置时的错误处理。

**示例 2：网络错误处理**

展示网络连接问题的错误处理。

**示例 3：API 错误处理**

展示 API 返回错误时的处理方式。

**示例 4：错误类型判断**

展示如何根据错误类型采取不同的处理策略。

**示例 5：错误恢复和重试**

展示如何实现自定义的重试逻辑。

### retry_example.go

展示重试和超时的各种配置。

**示例 1：使用默认配置**

展示插件的默认重试和超时行为。

**示例 2：自定义重试配置**

展示如何调整重试次数、退避时间等参数。

**示例 3：自定义超时配置**

展示如何调整各种超时参数。

**示例 4：使用上下文超时**

展示如何使用 Go 的 context 来控制超时。

**示例 5：禁用重试**

展示如何完全禁用重试机制。

## 最佳实践

### 1. 错误处理

始终检查错误并根据错误类型采取适当的行动：

```go
if err != nil {
    var azErr *azure.AzureAIError
    if errors.As(err, &azErr) {
        switch azErr.Type {
        case azure.ErrorTypeConfig:
            // 配置错误 - 修复配置
        case azure.ErrorTypeNetwork:
            // 网络错误 - 重试
        case azure.ErrorTypeAPI:
            // API 错误 - 根据状态码决定
        }
    }
}
```

### 2. 资源复用

复用插件和模型实例，避免重复创建：

```go
// ✅ 好的做法：复用实例
var (
    plugin *azure.AzureAI
    model  ai.Model
)

func init() {
    plugin = &azure.AzureAI{...}
    plugin.Init(context.Background())
    model = plugin.DefineModel(...)
}

// ❌ 不好的做法：每次都创建新实例
func generateText() {
    plugin := &azure.AzureAI{...}  // 不要这样做
    model := plugin.DefineModel(...)
}
```

### 3. 超时控制

为所有请求设置合理的超时：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := model.Generate(ctx, req, nil)
```

### 4. 流式响应

对于长文本生成，使用流式响应提升用户体验：

```go
resp, err := model.Generate(ctx, req, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    // 实时处理数据块
    return nil
})
```

### 5. 批量处理

对于嵌入任务，使用批量处理提高效率：

```go
// 一次处理多个文档
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: documents, // 10-100 个文档
})
```

## 故障排除

### 问题 1：环境变量未设置

**错误信息**：
```
请设置 AZURE_OPENAI_API_KEY 和 AZURE_OPENAI_BASE_URL 环境变量
```

**解决方案**：
```bash
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_BASE_URL="https://your-resource.openai.azure.com"
```

### 问题 2：API Key 无效

**错误信息**：
```
API 错误 (状态码 401): Unauthorized
```

**解决方案**：
- 检查 API Key 是否正确
- 确认 API Key 是否已激活
- 检查 API Key 是否有权限访问指定的模型

### 问题 3：模型不存在

**错误信息**：
```
API 错误 (状态码 404): Model not found
```

**解决方案**：
- 确认模型名称是否正确
- 检查模型是否已在 Azure OpenAI 中部署
- 使用 Azure Portal 查看可用的模型部署

### 问题 4：速率限制

**错误信息**：
```
API 错误 (状态码 429): Too Many Requests
```

**解决方案**：
- 插件会自动重试，等待一段时间
- 如果频繁遇到，考虑增加重试间隔或减少请求频率
- 联系 Azure 支持增加配额

### 问题 5：网络连接问题

**错误信息**：
```
网络错误: dial tcp: lookup your-resource.openai.azure.com: no such host
```

**解决方案**：
- 检查网络连接
- 确认 BaseURL 是否正确
- 检查防火墙设置
- 尝试使用代理

## 进一步学习

- [主 README](../README.md) - 完整的插件文档
- [错误处理文档](../ERROR_HANDLING.md) - 详细的错误处理指南
- [重试和超时配置](../RETRY_AND_TIMEOUT.md) - 重试和超时的详细说明
- [嵌入器使用指南](../EMBEDDER_README.md) - 嵌入功能的详细说明

## 反馈和贡献

如果你发现示例中的问题或有改进建议，欢迎：

1. 提交 Issue
2. 创建 Pull Request
3. 在讨论区分享你的使用经验

## 许可证

Apache License 2.0
