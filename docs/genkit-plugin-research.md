# Genkit 插件支持调研报告

## 调研日期

2025年11月24日

## 调研目标

调研 Google Genkit 框架对不同 AI 模型提供商的插件支持情况，特别关注：

1. Google AI (Gemini) 插件
2. Azure OpenAI 插件
3. 阿里云百炼插件

## 一、Genkit 框架概述

### 1.1 Genkit 简介

- **定义**: Google 开发的开源 AI 应用开发框架
- **核心价值**: 提供统一的接口抽象不同 AI 模型服务的访问细节
- **主要特性**:
  - 统一的 API 接口（`generate()` 方法）
  - 支持流式和非流式调用
  - 支持多模态输入（文本、图像、视频、音频）
  - 内置开发工具和调试 UI
  - 支持部署到 Firebase、Cloud Run 或自定义基础设施

### 1.2 Genkit Go SDK

- **版本**: 当前项目使用 v1.1.0（最新版本 v1.2.0）
- **包路径**: `github.com/firebase/genkit/go`
- **核心包**:
  - `genkit`: 主包，提供初始化和生成接口
  - `ai`: AI 相关类型和接口定义
  - `plugins`: 各种模型提供商插件

## 二、Google AI (Gemini) 插件调研

### 2.1 插件信息

- **包名**: `github.com/firebase/genkit/go/plugins/googlegenai`
- **状态**: ✅ 官方支持，已在项目中使用
- **支持模型**:
  - Gemini 2.5 Flash
  - Gemini 2.0 Flash
  - Gemini 1.5 Pro
  - Gemini 1.5 Flash
  - 其他 Gemini 系列模型

### 2.2 当前使用情况

项目中已经集成并使用 Google AI 插件：

```go
import "github.com/firebase/genkit/go/plugins/googlegenai"

// 初始化
c.g = genkit.Init(ctx,
    genkit.WithPlugins(&googlegenai.GoogleAI{
        APIKey: c.config.APIKey,
    }),
    genkit.WithDefaultModel("googleai/"+c.config.Model),
)
```

### 2.3 配置要求

- **必需参数**:
  - `APIKey`: Google AI API 密钥
  - `Model`: 模型名称（如 "gemini-1.5-pro"）
- **可选参数**:
  - `Temperature`: 温度值（0-2）
  - `MaxTokens`: 最大 token 数
  - `TopP`: Top-p 采样参数
  - `TopK`: Top-k 采样参数

### 2.4 功能支持

- ✅ 非流式生成
- ✅ 流式生成
- ✅ Token 使用统计
- ✅ 多轮对话
- ✅ 系统提示词
- ✅ 多模态输入

### 2.5 集成方案

**结论**: 已完成集成，无需额外工作。

## 三、Azure OpenAI 插件调研

### 3.1 官方插件情况

**发现**: Genkit Go SDK **没有**官方的 Azure OpenAI 专用插件。

### 3.2 可用方案

#### 方案 A：使用 OpenAI 插件 + 自定义 BaseURL（推荐）

- **包名**: `github.com/firebase/genkit/go/plugins/compat_oai/openai`
- **状态**: ✅ 官方支持
- **原理**: Azure OpenAI 提供 OpenAI 兼容的 API，可以通过修改 BaseURL 来访问

**实现方式**:

```go
import (
    oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
    "github.com/openai/openai-go"
)

// 构建 Azure OpenAI 的 BaseURL
baseURL := fmt.Sprintf("%s/openai/deployments/%s",
    azureEndpoint,      // 如: https://your-resource.openai.azure.com
    azureDeployment,    // 如: gpt-4
)

// 初始化 OpenAI 插件，指向 Azure
plugin := &oai.OpenAI{
    APIKey:  azureAPIKey,
    BaseURL: baseURL,
}

// 初始化 Genkit
g := genkit.Init(ctx,
    genkit.WithPlugins(plugin),
    genkit.WithDefaultModel("openai/gpt-4"),
)
```

**配置参数**:

- `APIKey`: Azure OpenAI API 密钥
- `BaseURL`: Azure OpenAI Endpoint + Deployment
- `APIVersion`: Azure API 版本（如 "2024-02-15-preview"）

**优点**:

- ✅ 使用官方维护的 OpenAI 插件
- ✅ 无需自定义插件代码
- ✅ 自动支持流式和非流式调用
- ✅ 完整的 Token 统计支持

**缺点**:

- ⚠️ 需要正确构建 Azure 特定的 BaseURL
- ⚠️ API 版本需要通过 URL 参数或 Header 传递

#### 方案 B：自定义 Azure OpenAI 插件

如果方案 A 不能满足需求，可以参考 OpenAI 插件实现自定义插件。

**实现要点**:

- 实现 `genkit.Plugin` 接口
- 使用 Azure OpenAI Go SDK
- 处理 Azure 特定的认证和配置
- 转换请求/响应格式

**评估**:

- ❌ 开发工作量大
- ❌ 需要维护自定义代码
- ❌ 不推荐，除非方案 A 完全不可行

### 3.3 集成方案决策

**推荐方案**: 方案 A - 使用 OpenAI 插件 + 自定义 BaseURL

**理由**:

1. Azure OpenAI 完全兼容 OpenAI API
2. 官方 OpenAI 插件已经过充分测试
3. 实现简单，维护成本低
4. 支持所有必需功能

## 四、阿里云百炼插件调研

### 4.1 官方插件情况

**发现**: Genkit Go SDK **没有**阿里云百炼的官方插件。

### 4.2 百炼 API 特性

#### 4.2.1 OpenAI 兼容性

✅ **重要发现**: 阿里云百炼**完全兼容 OpenAI API**！

**官方文档说明**:
> "阿里云百炼兼容OpenAI接口规范，您只需调整 API Key、base_url 和模型名称，即可将原有 OpenAI 代码迁移至阿里云百炼。"

**API 端点**:

- 北京地域: `https://dashscope.aliyuncs.com/compatible-mode/v1`
- 新加坡地域: `https://dashscope-intl.aliyuncs.com/compatible-mode/v1`

**示例代码**（Python）:

```python
from openai import OpenAI

client = OpenAI(
    api_key=os.getenv("DASHSCOPE_API_KEY"),
    base_url="https://dashscope.aliyuncs.com/compatible-mode/v1",
)

completion = client.chat.completions.create(
    model="qwen-plus",
    messages=[{'role': 'user', 'content': '你是谁？'}]
)
```

#### 4.2.2 支持的模型

- **通义千问系列**:
  - qwen-max: 旗舰模型，适合复杂任务
  - qwen-plus: 平衡模型，通用场景
  - qwen-turbo: 高性价比，快速响应
  - qwen-coder: 代码生成专用
- **第三方模型**: DeepSeek, Kimi, GLM 等
- **多模态**: 文本、视觉、图像生成、视频生成、语音等

### 4.3 可用方案

#### 方案 A：使用 OpenAI 插件 + 百炼 BaseURL（强烈推荐）

由于百炼完全兼容 OpenAI API，可以直接使用 Genkit 的 OpenAI 插件。

**实现方式**:

```go
import oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"

// 初始化 OpenAI 插件，指向百炼
plugin := &oai.OpenAI{
    APIKey:  dashscopeAPIKey,
    BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
}

// 初始化 Genkit
g := genkit.Init(ctx,
    genkit.WithPlugins(plugin),
    genkit.WithDefaultModel("openai/qwen-plus"),
)
```

**配置参数**:

- `APIKey`: DashScope API 密钥（环境变量 `DASHSCOPE_API_KEY`）
- `BaseURL`: 百炼兼容模式端点
- `Model`: 模型名称（如 "qwen-plus", "qwen-turbo"）

**优点**:

- ✅ 零开发成本，直接使用现有插件
- ✅ 完全兼容 OpenAI API
- ✅ 自动支持流式和非流式调用
- ✅ 支持中文优化模型
- ✅ 官方维护，稳定可靠

**缺点**:

- ⚠️ 模型名称需要使用百炼的命名（如 "qwen-plus"）
- ⚠️ 某些百炼特有功能可能不支持（如果有的话）

#### 方案 B：自定义百炼插件

如果需要使用百炼的原生 API（非兼容模式）或特有功能。

**实现要点**:

- 实现 `genkit.Plugin` 接口
- 使用百炼原生 SDK 或 HTTP 客户端
- 处理百炼特定的请求/响应格式
- 实现流式和非流式调用

**评估**:

- ❌ 开发工作量大
- ❌ 需要维护自定义代码
- ❌ 不推荐，因为方案 A 已经足够

### 4.4 集成方案决策

**推荐方案**: 方案 A - 使用 OpenAI 插件 + 百炼兼容模式 BaseURL

**理由**:

1. 百炼官方提供 OpenAI 兼容接口
2. 零开发成本，直接复用现有插件
3. 支持所有必需功能（流式、非流式、Token 统计）
4. 官方维护，稳定可靠
5. 支持中文优化的通义千问模型

## 五、社区插件生态

### 5.1 JavaScript/TypeScript 生态

Genkit 的 JS/TS 版本有丰富的社区插件：

- Azure OpenAI 插件（社区维护）
- Anthropic 插件（社区维护）
- Cohere 插件（社区维护）
- Mistral 插件（社区维护）
- Groq 插件（社区维护）

### 5.2 Go 生态

Go SDK 的社区插件相对较少，但官方插件已经覆盖主要需求：

- ✅ Google AI (Gemini)
- ✅ OpenAI
- ✅ Anthropic
- ✅ Vertex AI
- ✅ Ollama（本地模型）

## 六、集成方案总结

### 6.1 最终方案

| 提供商 | 集成方案 | 插件 | 开发工作量 |
|--------|---------|------|-----------|
| Google AI (Gemini) | 已完成 | `googlegenai` | 无 |
| Azure OpenAI | OpenAI 插件 + Azure BaseURL | `compat_oai/openai` | 低 |
| 阿里云百炼 | OpenAI 插件 + 百炼 BaseURL | `compat_oai/openai` | 低 |

### 6.2 统一架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                  Genkit Client 层                            │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  配置查询服务                                         │  │
│  │  - 根据租户ID+模型名查询 model_configurations        │  │
│  │  - 解析 provider_type 和 configuration              │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Plugin 初始化器                                      │  │
│  │                                                        │  │
│  │  switch (provider_type) {                            │  │
│  │    case "google":                                     │  │
│  │      plugin = &googlegenai.GoogleAI{...}             │  │
│  │                                                        │  │
│  │    case "azure":                                      │  │
│  │      plugin = &openai.OpenAI{                        │  │
│  │        BaseURL: azureEndpoint + deployment           │  │
│  │      }                                                 │  │
│  │                                                        │  │
│  │    case "bailian":                                    │  │
│  │      plugin = &openai.OpenAI{                        │  │
│  │        BaseURL: "dashscope.aliyuncs.com/..."        │  │
│  │      }                                                 │  │
│  │  }                                                     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 6.3 配置示例

#### Google AI 配置

```json
{
  "provider_type": "google",
  "api_key": "your-google-api-key",
  "configuration": {
    "model": "gemini-1.5-pro",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

#### Azure OpenAI 配置

```json
{
  "provider_type": "azure",
  "api_key": "your-azure-api-key",
  "configuration": {
    "model": "gpt-4",
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

#### 阿里云百炼配置

```json
{
  "provider_type": "bailian",
  "api_key": "your-dashscope-api-key",
  "configuration": {
    "model": "qwen-plus",
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "bailianRegion": "beijing",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

## 七、技术风险评估

### 7.1 Azure OpenAI 风险

- **风险**: BaseURL 构建可能需要特殊处理
- **缓解**: 参考 Azure OpenAI 官方文档，进行充分测试
- **影响**: 低 - OpenAI 插件已经支持自定义 BaseURL

### 7.2 百炼风险

- **风险**: 兼容模式可能不支持某些特有功能
- **缓解**: 使用标准 OpenAI API 功能，避免依赖特有功能
- **影响**: 低 - 基本功能完全兼容

### 7.3 版本兼容性风险

- **风险**: Genkit SDK 版本更新可能导致 API 变化
- **缓解**: 锁定依赖版本，定期测试升级
- **影响**: 低 - Genkit 已经相对稳定

## 八、开发建议

### 8.1 实施顺序

1. **阶段 1**: 扩展配置结构，支持多提供商
2. **阶段 2**: 实现 Azure OpenAI 集成（使用 OpenAI 插件）
3. **阶段 3**: 实现百炼集成（使用 OpenAI 插件）
4. **阶段 4**: 完整测试和优化

### 8.2 测试策略

- **单元测试**: 测试配置解析和插件初始化
- **集成测试**: 测试实际 API 调用（需要真实 API 密钥）
- **端到端测试**: 测试完整的请求流程

### 8.3 文档要求

- 配置示例文档
- API 使用指南
- 故障排查指南
- 迁移指南

## 九、结论

### 9.1 核心发现

1. ✅ Google AI (Gemini) 已完成集成，无需额外工作
2. ✅ Azure OpenAI 可以通过 OpenAI 插件 + 自定义 BaseURL 实现
3. ✅ 阿里云百炼完全兼容 OpenAI API，可以直接使用 OpenAI 插件
4. ✅ 所有三个提供商都可以通过 Genkit 统一接口访问
5. ✅ 无需开发自定义插件，开发工作量大幅降低

### 9.2 技术可行性

**评估**: ✅ 完全可行

**理由**:

- Genkit 提供了灵活的插件机制
- OpenAI 插件支持自定义 BaseURL
- Azure 和百炼都提供 OpenAI 兼容接口
- 所有必需功能（流式、非流式、Token 统计）都得到支持

### 9.3 预期工作量

- **原计划**: 15-20 天（包括自定义插件开发）
- **实际预期**: 8-10 天（使用现有插件）
- **节省**: 约 50% 的开发时间

### 9.4 下一步行动

1. ✅ 完成调研文档（本文档）
2. ⏭️ 扩展 Genkit 配置结构
3. ⏭️ 实现 Azure OpenAI 集成
4. ⏭️ 实现百炼集成
5. ⏭️ 编写测试用例
6. ⏭️ 编写使用文档

## 十、参考资料

### 10.1 官方文档

- [Genkit 官方文档](https://firebase.google.com/docs/genkit)
- [Genkit Go SDK](https://pkg.go.dev/github.com/firebase/genkit/go)
- [Azure OpenAI 文档](https://learn.microsoft.com/azure/ai-services/openai/)
- [阿里云百炼文档](https://help.aliyun.com/zh/model-studio/)

### 10.2 插件文档

- [Google AI 插件](https://pkg.go.dev/github.com/firebase/genkit/go/plugins/googlegenai)
- [OpenAI 插件](https://pkg.go.dev/github.com/firebase/genkit/go/plugins/compat_oai/openai)
- [Anthropic 插件](https://pkg.go.dev/github.com/firebase/genkit/go/plugins/compat_oai/anthropic)

### 10.3 API 参考

- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [Azure OpenAI API 文档](https://learn.microsoft.com/azure/ai-services/openai/reference)
- [百炼 OpenAI 兼容接口](https://help.aliyun.com/zh/model-studio/first-api-call-to-qwen)

---

**调研完成日期**: 2025年11月24日  
**调研人员**: AI Assistant  
**审核状态**: 待审核
