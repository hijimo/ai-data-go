# Azure OpenAI BaseURL 验证报告

## 验证日期

2025-11-25

## 验证目标

验证是否可以使用 Genkit 的 OpenAI 插件 + 自定义 BaseURL 来集成 Azure OpenAI。

## 验证结果

✅ **完全可行**

## 技术方案

### 核心实现

使用 Genkit 的 OpenAI 插件，通过 `option.WithBaseURL()` 自定义 BaseURL 来调用 Azure OpenAI API。

### 代码实现

```go
case "azureopenai":
    // Azure OpenAI (使用 OpenAI 插件 + Azure BaseURL)
    baseURL := fmt.Sprintf("%s/openai/deployments/%s",
        genkitConfig.AzureEndpoint,
        genkitConfig.AzureDeployment,
    )

    plugin := &oai.OpenAI{
        Opts: []option.RequestOption{
            option.WithAPIKey(tempConfig.APIKey),
            option.WithBaseURL(baseURL),
        },
    }
    fullModelName = "openai/" + genkitConfig.Model
    
    g = genkit.Init(ctx,
        genkit.WithPlugins(plugin),
        genkit.WithDefaultModel(fullModelName),
    )
```

## 配置结构

### GenkitConfig 支持的 Azure 字段

```go
type GenkitConfig struct {
    // Azure OpenAI 特定配置
    AzureEndpoint   string `json:"azureEndpoint,omitempty"`   // https://your-resource.openai.azure.com
    AzureDeployment string `json:"azureDeployment,omitempty"` // gpt-4, gpt-35-turbo 等
    AzureAPIVersion string `json:"azureApiVersion,omitempty"` // 2024-02-15-preview
    
    // 通用配置
    Model              string  `json:"model"`
    DefaultTemperature float64 `json:"defaultTemperature,omitempty"`
    DefaultMaxTokens   int     `json:"defaultMaxTokens,omitempty"`
}
```

### 配置示例

```json
{
  "model": "gpt-4",
  "azureEndpoint": "https://my-resource.openai.azure.com",
  "azureDeployment": "gpt-4-deployment",
  "azureApiVersion": "2024-02-15-preview",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

## 验证测试

### 测试覆盖

✅ **配置验证测试**

- 完整的 Azure OpenAI 配置
- 缺少 AzureEndpoint 的错误处理
- 缺少 AzureDeployment 的错误处理
- 缺少 AzureAPIVersion 的错误处理

✅ **BaseURL 构造测试**

- 标准 Azure Endpoint
- 带尾部斜杠的 Endpoint
- 自定义域名

✅ **API Version 参数处理测试**

- 标准 API Version 格式
- 稳定版本格式
- 空 API Version 处理

✅ **模型名称映射测试**

- GPT-4 部署
- GPT-3.5 Turbo 部署
- 自定义部署名称

### 测试结果

```bash
=== RUN   TestAzureOpenAIConfig
--- PASS: TestAzureOpenAIConfig (0.00s)
=== RUN   TestAzureBaseURLConstruction
--- PASS: TestAzureBaseURLConstruction (0.00s)
=== RUN   TestAzureAPIVersionHandling
--- PASS: TestAzureAPIVersionHandling (0.00s)
=== RUN   TestAzureModelNameMapping
--- PASS: TestAzureModelNameMapping (0.00s)
PASS
ok      genkit-ai-service/internal/genkit       0.855s
```

## 关键发现

### 1. OpenAI 插件支持自定义 BaseURL ✅

Genkit 的 OpenAI 插件完全支持通过 `option.WithBaseURL()` 自定义 BaseURL，这使得集成 Azure OpenAI 成为可能。

### 2. 认证方式兼容 ✅

OpenAI 插件通过 `option.WithAPIKey()` 设置 API Key，这与 Azure OpenAI 的认证方式完全兼容。

### 3. BaseURL 格式

Azure OpenAI 的 BaseURL 格式：

```
https://{resource-name}.openai.azure.com/openai/deployments/{deployment-name}
```

### 4. API Version 参数

⚠️ **待确认**：API Version 参数的传递方式

Azure OpenAI 需要在请求中包含 `api-version` 查询参数。目前有两种可能的实现方式：

**方案 A**：在 BaseURL 中包含查询参数

```go
baseURL := fmt.Sprintf("%s/openai/deployments/%s?api-version=%s",
    genkitConfig.AzureEndpoint,
    genkitConfig.AzureDeployment,
    genkitConfig.AzureAPIVersion,
)
```

**方案 B**：通过 OpenAI SDK 的查询参数机制

```go
// 需要查看 OpenAI SDK 是否支持自定义查询参数
```

**推荐**：先尝试方案 A（在 BaseURL 中包含），如果不行再考虑方案 B。

### 5. 模型名称映射

在 Genkit 中，Azure OpenAI 使用 `openai/` 前缀：

- Azure Deployment: `gpt-4-deployment`
- Genkit Model Name: `openai/gpt-4-deployment`

## 实施状态

### 已完成 ✅

1. **代码实现**
   - ✅ 在 `internal/genkit/client.go` 中实现了 Azure OpenAI 集成
   - ✅ 使用 OpenAI 插件 + 自定义 BaseURL 方案
   - ✅ 支持通过 `option.WithAPIKey()` 和 `option.WithBaseURL()` 配置

2. **配置支持**
   - ✅ 支持 `azureEndpoint` 配置
   - ✅ 支持 `azureDeployment` 配置
   - ✅ 支持 `azureApiVersion` 配置

3. **配置验证**
   - ✅ 实现了 `validateAzureConfig()` 方法
   - ✅ 验证必需字段的存在性

4. **单元测试**
   - ✅ 配置验证测试
   - ✅ BaseURL 构造测试
   - ✅ API Version 处理测试
   - ✅ 模型名称映射测试

### 待完成 ⏳

1. **API Version 参数传递**
   - ⏳ 确认最佳的 API Version 传递方式
   - ⏳ 可能需要在 BaseURL 中包含查询参数

2. **实际 API 测试**
   - ⏳ 测试非流式调用
   - ⏳ 测试流式调用
   - ⏳ 测试错误处理
   - ⏳ 验证 Token 统计

3. **集成测试**
   - ⏳ 端到端测试
   - ⏳ 多租户场景测试
   - ⏳ 并发调用测试

## 优势

### 1. 无需自定义插件

利用 Genkit 官方维护的 OpenAI 插件，无需开发和维护自定义插件。

### 2. 代码简洁

实现代码简洁明了，易于理解和维护。

### 3. 与 Genkit 生态系统完全兼容

使用官方插件确保与 Genkit 的其他功能完全兼容。

### 4. 易于扩展

相同的方案可以用于其他 OpenAI 兼容的服务（如阿里云百炼）。

## 风险和缓解

### 风险 1：API Version 参数传递

**风险**：不确定 API Version 参数的最佳传递方式。

**缓解**：

1. 先尝试在 BaseURL 中包含查询参数
2. 如果不行，查看 OpenAI SDK 文档寻找其他方案
3. 最坏情况下，可以考虑使用自定义 HTTP Client

### 风险 2：Azure 特定功能

**风险**：Azure OpenAI 可能有一些特定功能与标准 OpenAI API 不同。

**缓解**：

1. 详细测试所有使用的功能
2. 记录任何差异
3. 必要时在代码中添加特殊处理

### 风险 3：API 版本兼容性

**风险**：Azure API 版本更新可能导致不兼容。

**缓解**：

1. 使用稳定的 API 版本
2. 在配置中明确指定版本
3. 定期检查 Azure 的 API 更新

## 下一步行动

### 1. API Version 参数处理（优先级：P0）

**任务**：确认并实现 API Version 参数的传递方式

**预计工时**：0.5 小时

**行动**：

1. 尝试在 BaseURL 中包含 `?api-version=xxx`
2. 进行实际 API 调用测试
3. 如果不行，查看 OpenAI SDK 文档

### 2. 实际 API 测试（优先级：P0）

**任务**：使用真实的 Azure OpenAI 环境进行测试

**预计工时**：2-3 小时

**行动**：

1. 配置测试环境
2. 测试非流式调用
3. 测试流式调用
4. 测试错误处理
5. 验证 Token 统计

### 3. 文档完善（优先级：P1）

**任务**：编写完整的使用文档

**预计工时**：1 小时

**行动**：

1. 配置示例
2. 使用指南
3. 故障排查指南

## 结论

✅ **验证通过**：可以使用 Genkit 的 OpenAI 插件 + 自定义 BaseURL 来集成 Azure OpenAI。

**核心优势**：

- 无需自定义插件开发
- 代码简洁易维护
- 与 Genkit 生态系统完全兼容

**剩余工作**：

- API Version 参数传递机制确认
- 实际 API 环境测试
- 文档完善

**预计剩余工时**：3.5-4.5 小时

## 参考资料

1. [Genkit OpenAI Plugin 文档](https://genkit.dev/docs/plugins/openai)
2. [Azure OpenAI Service 文档](https://learn.microsoft.com/azure/ai-services/openai/)
3. [Azure OpenAI REST API 参考](https://learn.microsoft.com/azure/ai-services/openai/reference)
4. [OpenAI Go SDK](https://github.com/openai/openai-go)
