# 设计文档

## 概述

本文档描述了 Azure AI Genkit Provider 插件的设计。该插件将使 Genkit Go 框架能够与 Azure OpenAI 服务集成，特别是使用 Azure OpenAI 的 Responses API（/openai/responses 端点）而非传统的 chat/completions 端点。

插件将遵循 Genkit 的插件架构模式，参考现有的 compat_oai 和 googlegenai 插件实现，提供模型定义、文本生成、流式响应和嵌入功能。

## 架构

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Genkit Application                        │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            │ 使用 Plugin API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Azure AI Provider Plugin                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Plugin Interface (Init, Name, DefineModel, etc.)    │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────┴──────────────────────────┐    │
│  │                                                      │    │
│  ▼                                                      ▼    │
│  ┌──────────────────┐                    ┌──────────────────┐
│  │  Model Generator │                    │  Embedder        │
│  │  - 消息转换      │                    │  - 文本向量化    │
│  │  - 工具处理      │                    │  - 批量处理      │
│  │  - 流式响应      │                    │                  │
│  └──────────────────┘                    └──────────────────┘
│           │                                       │           │
└───────────┼───────────────────────────────────────┼───────────┘
            │                                       │
            │ HTTP Requests                         │
            ▼                                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Azure OpenAI Service                            │
│  - /openai/responses (Responses API)                        │
│  - /openai/embeddings (Embeddings API)                      │
└─────────────────────────────────────────────────────────────┘
```

### 插件结构

插件将位于 `internal/genkit/plugins/azure/` 目录，包含以下文件：

- `azure.go`: 插件主文件，实现 Genkit Plugin 接口
- `generate.go`: 模型生成逻辑，包括消息转换和 API 调用
- `embed.go`: 嵌入模型实现
- `types.go`: Azure OpenAI API 的请求和响应类型定义
- `README.md`: 使用文档和示例

## 组件和接口

### 1. AzureAI Plugin 结构

```go
type AzureAI struct {
    mu sync.Mutex
    initted bool
    
    // Azure OpenAI 配置
    APIKey string
    BaseURL string
    APIVersion string
    
    // HTTP 客户端
    httpClient *http.Client
    
    // Provider 标识符
    Provider string
}
```

**职责**:
- 实现 Genkit Plugin 接口
- 管理插件生命周期
- 提供模型和嵌入器定义方法

**接口方法**:
- `Init(ctx context.Context) []api.Action`: 初始化插件
- `Name() string`: 返回插件名称
- `DefineModel(provider, id string, opts ai.ModelOptions) ai.Model`: 定义模型
- `DefineEmbedder(provider, name string, opts *ai.EmbedderOptions) ai.Embedder`: 定义嵌入器

### 2. ModelGenerator 结构

```go
type ModelGenerator struct {
    client *http.Client
    baseURL string
    apiKey string
    apiVersion string
    modelName string
    
    // 请求构建
    messages []Message
    tools []Tool
    config map[string]any
    
    // 错误跟踪
    err error
}
```

**职责**:
- 构建 Azure OpenAI Responses API 请求
- 处理流式和非流式响应
- 转换 Genkit 消息格式到 Azure OpenAI 格式
- 处理工具调用

**方法**:
- `WithMessages(messages []*ai.Message) *ModelGenerator`: 添加消息
- `WithTools(tools []*ai.ToolDefinition) *ModelGenerator`: 添加工具
- `WithConfig(config any) *ModelGenerator`: 添加配置
- `Generate(ctx context.Context, req *ai.ModelRequest, cb func(...) error) (*ai.ModelResponse, error)`: 执行生成

### 3. Azure OpenAI API 类型

```go
// Responses API 请求格式
type ResponsesRequest struct {
    Model string `json:"model"`
    Input []Message `json:"input"`  // 注意：使用 input 而非 messages
    Stream bool `json:"stream,omitempty"`
    Temperature *float64 `json:"temperature,omitempty"`
    MaxTokens *int `json:"max_tokens,omitempty"`
    TopP *float64 `json:"top_p,omitempty"`
    Tools []Tool `json:"tools,omitempty"`
}

type Message struct {
    Role string `json:"role"`
    Content any `json:"content"`  // 可以是 string 或 []ContentPart
}

type ContentPart struct {
    Type string `json:"type"`  // "text" 或 "image_url"
    Text string `json:"text,omitempty"`
    ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
    URL string `json:"url"`
}

type Tool struct {
    Type string `json:"type"`  // "function"
    Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
    Name string `json:"name"`
    Description string `json:"description,omitempty"`
    Parameters map[string]any `json:"parameters,omitempty"`
}

// Responses API 响应格式
type ResponsesResponse struct {
    ID string `json:"id"`
    Object string `json:"object"`
    Created int64 `json:"created"`
    Model string `json:"model"`
    Choices []Choice `json:"choices"`
    Usage Usage `json:"usage"`
    SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

type Choice struct {
    Index int `json:"index"`
    Message Message `json:"message"`
    FinishReason string `json:"finish_reason"`
}

type Usage struct {
    PromptTokens int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens int `json:"total_tokens"`
}
```

## 数据模型

### 消息转换映射

| Genkit Message Role | Azure OpenAI Role | 说明 |
|---------------------|-------------------|------|
| ai.RoleSystem | "system" | 系统提示 |
| ai.RoleUser | "user" | 用户消息 |
| ai.RoleModel | "assistant" | 助手消息 |
| ai.RoleTool | "tool" | 工具响应 |

### 内容类型映射

| Genkit Part Type | Azure OpenAI Content | 说明 |
|------------------|----------------------|------|
| Text | string 或 {type: "text", text: "..."} | 纯文本 |
| Media (image) | {type: "image_url", image_url: {url: "..."}} | 图像 URL |
| ToolRequest | message.tool_calls | 工具调用请求 |
| ToolResponse | role: "tool", content: "..." | 工具执行结果 |

### FinishReason 映射

| Azure OpenAI | Genkit | 说明 |
|--------------|--------|------|
| "stop" | ai.FinishReasonStop | 正常完成 |
| "length" | ai.FinishReasonLength | 达到最大长度 |
| "content_filter" | ai.FinishReasonBlocked | 内容过滤 |
| "tool_calls" | ai.FinishReasonStop | 工具调用完成 |


## 正确性属性

*属性是一个特征或行为，应该在系统的所有有效执行中保持为真——本质上是关于系统应该做什么的正式陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### 属性 1: API 版本参数正确性
*对于任何* 指定了 API 版本的请求，构建的 URL 应该包含正确的 api-version 查询参数
**验证需求: 1.4**

### 属性 2: 默认 API 版本
*对于任何* 未指定 API 版本的请求，构建的 URL 应该包含默认的 api-version=2025-04-01-preview
**验证需求: 1.5**

### 属性 3: Responses API 端点使用
*对于任何* 模型生成请求，构建的 URL 应该包含 /openai/responses 路径而非 /chat/completions
**验证需求: 2.2, 9.1**

### 属性 4: 多模态内容转换
*对于任何* 包含文本和图像的消息，转换后的 Azure OpenAI 格式应该包含对应的 text 和 image_url 内容部分
**验证需求: 2.3**

### 属性 5: 工具定义转换
*对于任何* Genkit 工具定义，转换后的 Azure OpenAI 格式应该包含 type="function" 和正确的 function 字段
**验证需求: 2.4, 6.1**

### 属性 6: 系统消息位置
*对于任何* 包含系统消息的消息列表，转换后的 input 数组中系统消息应该位于第一个位置
**验证需求: 2.5, 5.2**

### 属性 7: 流式模式触发
*对于任何* 包含回调函数的请求，构建的请求体中 stream 字段应该为 true
**验证需求: 3.1**

### 属性 8: SSE 数据解析
*对于任何* 符合 SSE 格式的数据流，解析后应该能够提取出所有 data: 行的 JSON 内容
**验证需求: 3.2**

### 属性 9: 流式回调调用
*对于任何* 流式响应的数据块，如果包含内容，回调函数应该被调用一次
**验证需求: 3.3**

### 属性 10: 流式响应聚合
*对于任何* 流式响应的多个数据块，最终响应应该包含所有数据块的聚合内容
**验证需求: 3.4**

### 属性 11: 嵌入端点使用
*对于任何* 嵌入请求，构建的 URL 应该包含 /openai/embeddings 路径
**验证需求: 4.2**

### 属性 12: 批量嵌入处理
*对于任何* 包含多个文本的嵌入请求，所有文本应该被包含在单个 API 请求的 input 数组中
**验证需求: 4.3**

### 属性 13: 向量数据提取
*对于任何* 嵌入响应，提取的向量数组长度应该等于请求中的文本数量
**验证需求: 4.4**

### 属性 14: 用户消息内容转换
*对于任何* 包含文本和媒体的用户消息，转换后应该包含对应数量的 content parts
**验证需求: 5.1**

### 属性 15: 助手消息工具调用
*对于任何* 包含工具调用的助手消息，转换后应该包含 tool_calls 数组
**验证需求: 5.3**

### 属性 16: 工具响应 ID 保留
*对于任何* 工具响应，转换后的 tool_call_id 应该与原始工具调用的 ID 相同
**验证需求: 5.4, 6.3**

### 属性 17: 图像格式支持
*对于任何* URL 或 base64 格式的图像，转换后都应该生成有效的 image_url 内容部分
**验证需求: 5.5**

### 属性 18: 工具调用解析
*对于任何* 包含工具调用的 API 响应，解析后应该提取出工具名称、参数和 ID
**验证需求: 6.2**

### 属性 19: 工具调用响应关联
*对于任何* 工具调用和对应的工具响应，它们应该通过相同的 ID 关联
**验证需求: 6.4**

### 属性 20: Token 使用统计提取
*对于任何* 包含 usage 字段的响应，应该正确提取 prompt_tokens、completion_tokens 和 total_tokens
**验证需求: 7.1**

### 属性 21: FinishReason 映射
*对于任何* Azure OpenAI 的 finish_reason 值，应该映射到对应的 Genkit FinishReason 枚举值
**验证需求: 7.2**

### 属性 22: 系统指纹存储
*对于任何* 包含 system_fingerprint 的响应，该值应该被存储在响应的 Custom 字段中
**验证需求: 7.3**

### 属性 23: 模型信息记录
*对于任何* 响应，实际使用的模型名称应该被记录在响应元数据中
**验证需求: 7.4**

### 属性 24: Temperature 参数传递
*对于任何* 包含 temperature 配置的请求，该参数应该出现在 API 请求体中
**验证需求: 8.1**

### 属性 25: MaxTokens 参数传递
*对于任何* 包含 max_tokens 配置的请求，该参数应该出现在 API 请求体中
**验证需求: 8.2**

### 属性 26: TopP 参数传递
*对于任何* 包含 top_p 配置的请求，该参数应该出现在 API 请求体中
**验证需求: 8.3**

### 属性 27: Map 配置支持
*对于任何* map[string]any 格式的配置，应该能够被正确解析并应用到请求中
**验证需求: 8.4**

### 属性 28: Input 字段使用
*对于任何* 请求，序列化的 JSON 应该包含 input 字段而非 messages 字段
**验证需求: 9.2**

### 属性 29: 认证头设置
*对于任何* HTTP 请求，应该包含 api-key 认证头
**验证需求: 9.3**

### 属性 30: 响应格式解析
*对于任何* 符合 Azure OpenAI 响应格式的 JSON，应该能够被正确解析为 Genkit ModelResponse
**验证需求: 9.4**

## 错误处理

### 错误类型

1. **配置错误**
   - 缺少必需的 API Key
   - 缺少必需的 Base URL
   - 无效的 API 版本格式

2. **请求构建错误**
   - 无效的消息格式
   - 无效的工具定义
   - 无效的配置参数

3. **网络错误**
   - 连接超时
   - DNS 解析失败
   - TLS 握手失败

4. **API 错误**
   - 401 Unauthorized: API Key 无效
   - 429 Too Many Requests: 速率限制
   - 500 Internal Server Error: Azure 服务错误
   - 400 Bad Request: 请求格式错误

5. **响应解析错误**
   - 无效的 JSON 格式
   - 缺少必需字段
   - 类型不匹配

### 错误处理策略

```go
// 错误包装
type AzureAIError struct {
    Type string  // "config", "request", "network", "api", "parse"
    Code string  // HTTP 状态码或错误代码
    Message string
    Details any
    Err error
}

func (e *AzureAIError) Error() string {
    return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

func (e *AzureAIError) Unwrap() error {
    return e.Err
}
```

### 错误恢复

- **重试策略**: 对于 429 和 5xx 错误，实现指数退避重试
- **降级策略**: 在流式响应失败时，回退到非流式模式
- **超时控制**: 为所有 HTTP 请求设置合理的超时时间（默认 30 秒）

## 测试策略

### 单元测试

单元测试将验证各个组件的功能：

1. **消息转换测试**
   - 测试各种角色的消息转换
   - 测试多模态内容转换
   - 测试工具调用和响应转换

2. **请求构建测试**
   - 测试 URL 构建（端点、查询参数）
   - 测试请求体序列化
   - 测试认证头设置

3. **响应解析测试**
   - 测试成功响应解析
   - 测试错误响应解析
   - 测试流式响应解析

4. **配置处理测试**
   - 测试各种配置参数的应用
   - 测试默认值处理
   - 测试配置验证

### 属性测试

属性测试将使用 Go 的 testing/quick 包或第三方库（如 gopter）来验证通用属性：

- **测试库**: 使用 `github.com/leanovate/gopter` 进行属性测试
- **测试配置**: 每个属性测试至少运行 100 次迭代
- **标签格式**: 每个属性测试必须包含注释 `**Feature: azure-ai-genkit-provider, Property {number}: {property_text}**`
- **单一属性**: 每个正确性属性由单个属性测试实现

属性测试示例：

```go
// **Feature: azure-ai-genkit-provider, Property 3: Responses API 端点使用**
func TestProperty_ResponsesAPIEndpoint(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("所有请求都使用 /openai/responses 端点", 
        prop.ForAll(
            func(modelName string, messages []string) bool {
                // 生成随机请求
                req := buildRequest(modelName, messages)
                url := req.URL.Path
                
                // 验证端点
                return strings.Contains(url, "/openai/responses")
            },
            gen.AnyString(),
            gen.SliceOf(gen.AnyString()),
        ),
    )
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### 集成测试

集成测试将验证与实际 Azure OpenAI 服务的交互：

1. **基本生成测试**
   - 测试简单的文本生成
   - 测试多轮对话
   - 测试系统提示

2. **流式响应测试**
   - 测试流式文本生成
   - 测试流式工具调用

3. **工具调用测试**
   - 测试单个工具调用
   - 测试多个工具调用
   - 测试工具调用链

4. **嵌入测试**
   - 测试单个文本嵌入
   - 测试批量文本嵌入

5. **错误处理测试**
   - 测试无效 API Key
   - 测试无效请求格式
   - 测试网络超时

### 测试数据生成

使用生成器创建测试数据：

```go
// 消息生成器
func genMessage() gopter.Gen {
    return gen.OneGenOf(
        genTextMessage(),
        genImageMessage(),
        genToolCallMessage(),
        genToolResponseMessage(),
    )
}

// 工具定义生成器
func genToolDefinition() gopter.Gen {
    return gopter.CombineGens(
        gen.Identifier(),  // name
        gen.AnyString(),   // description
        genJSONSchema(),   // parameters
    ).Map(func(values []interface{}) *ai.ToolDefinition {
        return &ai.ToolDefinition{
            Name:        values[0].(string),
            Description: values[1].(string),
            InputSchema: values[2].(map[string]any),
        }
    })
}
```

## 实现细节

### HTTP 客户端配置

```go
func newHTTPClient() *http.Client {
    return &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
        },
    }
}
```

### 流式响应处理

```go
func (g *ModelGenerator) parseSSE(reader *bufio.Reader, callback func(*ai.ModelResponseChunk) error) error {
    for {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            if err == io.EOF {
                break
            }
            return err
        }
        
        // 解析 SSE 格式: "data: {...}\n"
        if bytes.HasPrefix(line, []byte("data: ")) {
            data := bytes.TrimPrefix(line, []byte("data: "))
            data = bytes.TrimSpace(data)
            
            // 检查结束标记
            if bytes.Equal(data, []byte("[DONE]")) {
                break
            }
            
            // 解析 JSON
            var chunk ResponseChunk
            if err := json.Unmarshal(data, &chunk); err != nil {
                return err
            }
            
            // 转换并调用回调
            modelChunk := convertToModelChunk(&chunk)
            if err := callback(modelChunk); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### 消息转换实现

```go
func convertMessages(messages []*ai.Message) ([]Message, error) {
    var result []Message
    
    for _, msg := range messages {
        switch msg.Role {
        case ai.RoleSystem:
            result = append(result, Message{
                Role:    "system",
                Content: extractText(msg.Content),
            })
        case ai.RoleUser:
            content := convertUserContent(msg.Content)
            result = append(result, Message{
                Role:    "user",
                Content: content,
            })
        case ai.RoleModel:
            azMsg, err := convertAssistantMessage(msg)
            if err != nil {
                return nil, err
            }
            result = append(result, azMsg)
        case ai.RoleTool:
            toolMsgs := convertToolResponses(msg.Content)
            result = append(result, toolMsgs...)
        }
    }
    
    return result, nil
}

func convertUserContent(parts []*ai.Part) any {
    // 如果只有文本，返回字符串
    if len(parts) == 1 && parts[0].IsText() {
        return parts[0].Text
    }
    
    // 否则返回 content parts 数组
    var contentParts []ContentPart
    for _, part := range parts {
        if part.IsText() {
            contentParts = append(contentParts, ContentPart{
                Type: "text",
                Text: part.Text,
            })
        } else if part.IsMedia() {
            contentParts = append(contentParts, ContentPart{
                Type: "image_url",
                ImageURL: &ImageURL{URL: part.Text},
            })
        }
    }
    return contentParts
}
```

### 工具转换实现

```go
func convertTools(tools []*ai.ToolDefinition) []Tool {
    var result []Tool
    for _, tool := range tools {
        result = append(result, Tool{
            Type: "function",
            Function: FunctionDefinition{
                Name:        tool.Name,
                Description: tool.Description,
                Parameters:  tool.InputSchema,
            },
        })
    }
    return result
}

func extractToolCalls(msg *ai.Message) []ToolCall {
    var toolCalls []ToolCall
    for _, part := range msg.Content {
        if part.IsToolRequest() {
            toolCalls = append(toolCalls, ToolCall{
                ID:   part.ToolRequest.Ref,
                Type: "function",
                Function: FunctionCall{
                    Name:      part.ToolRequest.Name,
                    Arguments: marshalJSON(part.ToolRequest.Input),
                },
            })
        }
    }
    return toolCalls
}
```

## 部署和使用

### 安装

```bash
go get github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure"
)

func main() {
    ctx := context.Background()
    
    // 创建 Genkit 实例
    g := genkit.New(ctx, nil)
    
    // 初始化 Azure AI Provider
    azurePlugin := &azure.AzureAI{
        APIKey:     "your-api-key",
        BaseURL:    "https://your-resource.openai.azure.com",
        APIVersion: "2025-04-01-preview",
        Provider:   "azure",
    }
    
    // 注册插件
    g.RegisterPlugin(azurePlugin)
    
    // 定义模型
    model := azurePlugin.DefineModel("azure", "gpt-4", azure.ModelOptions{
        Label: "GPT-4",
        Supports: &azure.Multimodal,
    })
    
    // 使用模型
    resp, err := model.Generate(ctx, &genkit.GenerateRequest{
        Messages: []*genkit.Message{
            {Role: "user", Content: []*genkit.Part{genkit.NewTextPart("Hello!")}},
        },
    }, nil)
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

### 流式使用

```go
resp, err := model.Generate(ctx, req, func(ctx context.Context, chunk *genkit.ModelResponseChunk) error {
    for _, part := range chunk.Content {
        if part.IsText() {
            fmt.Print(part.Text)
        }
    }
    return nil
})
```

### 工具调用

```go
tools := []*genkit.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "Get current weather",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{"type": "string"},
            },
            "required": []string{"location"},
        },
    },
}

resp, err := model.Generate(ctx, &genkit.GenerateRequest{
    Messages: messages,
    Tools:    tools,
}, nil)
```

## 性能考虑

### 连接池

- 使用 HTTP 连接池减少连接建立开销
- 配置合理的空闲连接数和超时时间

### 请求批处理

- 嵌入请求支持批量处理，减少 API 调用次数
- 建议批量大小：10-100 个文本

### 缓存策略

- 考虑缓存嵌入结果（在应用层实现）
- 考虑缓存模型响应（对于相同的输入）

### 超时设置

- 默认请求超时：30 秒
- 流式请求超时：60 秒
- 可通过配置调整

## 安全考虑

### API Key 管理

- 不要在代码中硬编码 API Key
- 使用环境变量或密钥管理服务
- 定期轮换 API Key

### 数据隐私

- 注意 Azure OpenAI 的数据处理政策
- 不要发送敏感个人信息
- 考虑使用 Azure 的私有端点

### 错误信息

- 不要在错误信息中暴露 API Key
- 不要在日志中记录完整的请求/响应（可能包含敏感信息）
- 使用结构化日志，便于过滤敏感字段

## 维护和扩展

### 版本兼容性

- 支持多个 Azure OpenAI API 版本
- 在新版本发布时及时更新
- 保持向后兼容性

### 监控和日志

- 记录 API 调用次数和延迟
- 记录错误率和错误类型
- 使用结构化日志便于分析

### 扩展点

- 支持自定义 HTTP 客户端
- 支持自定义重试策略
- 支持自定义错误处理
- 支持插件中间件
