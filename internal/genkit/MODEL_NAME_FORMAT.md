# 模型名称格式配置文档

## 概述

本文档说明 Genkit Client 中各个提供商的模型名称格式配置。模型名称格式遵循 Genkit 框架的命名规范：`{provider}/{model}`。

## 模型名称格式规范

### 1. Google AI (Gemini)

**提供商标识**: `googlegenai`

**模型名称格式**: `googleai/{model}`

**示例**:

- `googleai/gemini-1.5-pro`
- `googleai/gemini-1.5-flash`
- `googleai/gemini-2.0-flash`

**代码实现**:

```go
case "googlegenai":
    plugin := &googlegenai.GoogleAI{
        APIKey: tempConfig.APIKey,
    }
    fullModelName = "googleai/" + genkitConfig.Model
```

### 2. OpenAI

**提供商标识**: `openai`

**模型名称格式**: `openai/{model}`

**示例**:

- `openai/gpt-4`
- `openai/gpt-4-turbo`
- `openai/gpt-3.5-turbo`

**代码实现**:

```go
case "openai":
    plugin := &oai.OpenAI{
        Opts: opts,
    }
    fullModelName = "openai/" + genkitConfig.Model
```

### 3. Azure OpenAI

**提供商标识**: `azureopenai`

**模型名称格式**: `openai/{model}`

**说明**: Azure OpenAI 使用 OpenAI 插件 + 自定义 BaseURL 的方式实现，因此模型名称格式使用 `openai/` 前缀。

**示例**:

- `openai/gpt-4`
- `openai/gpt-4-turbo`
- `openai/gpt-35-turbo`

**代码实现**:

```go
case "azureopenai":
    plugin, err := createAzurePlugin(tempConfig.APIKey, genkitConfig)
    if err != nil {
        return nil, fmt.Errorf("创建 Azure OpenAI 插件失败: %w", err)
    }
    fullModelName = "openai/" + genkitConfig.Model
```

### 4. 阿里云百炼

**提供商标识**: `bianlian`

**模型名称格式**: `openai/{model}`

**说明**: 百炼完全兼容 OpenAI API 规范，使用 OpenAI 插件 + 百炼兼容模式 BaseURL 实现，因此模型名称格式使用 `openai/` 前缀。

**示例**:

- `openai/qwen-plus`
- `openai/qwen-max`
- `openai/qwen-turbo`

**代码实现**:

```go
case "bianlian":
    plugin, err := createBailianPlugin(tempConfig.APIKey, genkitConfig)
    if err != nil {
        return nil, fmt.Errorf("创建百炼插件失败: %w", err)
    }
    fullModelName = "openai/" + genkitConfig.Model
```

### 5. Anthropic (Claude)

**提供商标识**: `anthropic`

**模型名称格式**: `anthropic/{model}`

**示例**:

- `anthropic/claude-3-opus`
- `anthropic/claude-3-sonnet`
- `anthropic/claude-3-haiku`

**代码实现**:

```go
case "anthropic":
    plugin := &anthropic.Anthropic{
        Opts: []option.RequestOption{
            option.WithAPIKey(tempConfig.APIKey),
        },
    }
    fullModelName = "anthropic/" + genkitConfig.Model
```

### 6. 自定义 OpenAI 兼容服务

**提供商标识**: `custom_openai`

**模型名称格式**: `openai/{model}`

**说明**: 用于集成任何 OpenAI 兼容的 API 服务。

**示例**:

- `openai/custom-model-1`
- `openai/custom-model-2`

**代码实现**:

```go
case "custom_openai":
    plugin := &oai.OpenAI{
        Opts: []option.RequestOption{
            option.WithAPIKey(tempConfig.APIKey),
            option.WithBaseURL(*tempConfig.BaseURL),
        },
    }
    fullModelName = "openai/" + genkitConfig.Model
```

## 配置示例

### 数据库配置示例

#### Google AI

```json
{
    "model": "gemini-1.5-pro",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}
```

#### Azure OpenAI

```json
{
    "model": "gpt-4",
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}
```

#### 阿里云百炼

```json
{
    "model": "qwen-plus",
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}
```

## 模型名称格式总结表

| 提供商 | 提供商标识 | 模型名称格式 | 使用的插件 |
|--------|-----------|-------------|-----------|
| Google AI | `googlegenai` | `googleai/{model}` | Google AI 插件 |
| OpenAI | `openai` | `openai/{model}` | OpenAI 插件 |
| Azure OpenAI | `azureopenai` | `openai/{model}` | OpenAI 插件 + Azure BaseURL |
| 阿里云百炼 | `bianlian` | `openai/{model}` | OpenAI 插件 + 百炼 BaseURL |
| Anthropic | `anthropic` | `anthropic/{model}` | Anthropic 插件 |
| 自定义 OpenAI | `custom_openai` | `openai/{model}` | OpenAI 插件 + 自定义 BaseURL |

## 验证方法

### 1. 代码审查

检查 `internal/genkit/client.go` 中的 `initializeProvider()` 函数，确认每个 case 分支都正确设置了 `fullModelName`。

### 2. 单元测试

运行单元测试验证配置：

```bash
go test ./internal/genkit -v
```

### 3. 集成测试

使用实际的 API 密钥进行集成测试，验证模型调用是否成功。

## 注意事项

1. **模型名称格式必须与 Genkit 框架规范一致**
   - 格式：`{provider}/{model}`
   - 提供商前缀必须与插件注册时使用的名称一致

2. **Azure OpenAI 和百炼使用 `openai/` 前缀**
   - 因为它们使用 OpenAI 插件作为底层实现
   - 这是 Genkit 框架的设计要求

3. **配置中的 `model` 字段**
   - 只需要指定模型名称，不需要包含提供商前缀
   - 例如：`"model": "gpt-4"` 而不是 `"model": "openai/gpt-4"`
   - 系统会自动添加正确的前缀

4. **模型名称大小写**
   - 遵循各提供商的官方命名规范
   - 例如：`gemini-1.5-pro`、`gpt-4`、`qwen-plus`

## 相关文件

- `internal/genkit/client.go` - 模型名称格式配置实现
- `internal/genkit/config.go` - 配置结构定义
- `internal/genkit/client_test.go` - 单元测试
- `.kiro/specs/genkit-multi-model-support/design.md` - 设计文档

## 更新历史

- 2024-01-XX: 初始版本，记录所有提供商的模型名称格式
- TASK-3.2: Azure OpenAI 模型名称格式配置完成
- TASK-4.3: 百炼模型名称格式配置完成
