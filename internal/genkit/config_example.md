# GenkitConfig 使用示例

## 概述

`GenkitConfig` 结构体用于解析存储在 `model_configurations` 表中的配置信息。它支持多个 AI 模型提供商的特定配置。

## 支持的提供商

### 1. Google AI (Gemini)

```json
{
  "model": "gemini-1.5-pro",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

### 2. Azure OpenAI

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

### 3. 阿里云百炼

```json
{
  "model": "qwen-turbo",
  "bailianEndpoint": "https://dashscope.aliyuncs.com",
  "bailianWorkspace": "default",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

## 使用方法

### 解析配置

```go
import "genkit-ai-service/internal/genkit"

// 从 JSON 字符串解析配置
configJSON := `{
    "model": "gpt-4",
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}`

config, err := genkit.ParseGenkitConfig(configJSON)
if err != nil {
    log.Fatalf("解析配置失败: %v", err)
}
```

### 验证配置

```go
// 验证 Azure OpenAI 配置
err := config.Validate("azureopenai")
if err != nil {
    log.Fatalf("配置验证失败: %v", err)
}

// 验证百炼配置
err = config.Validate("bianlian")
if err != nil {
    log.Fatalf("配置验证失败: %v", err)
}

// 验证 Google AI 配置
err = config.Validate("googlegenai")
if err != nil {
    log.Fatalf("配置验证失败: %v", err)
}
```

### 序列化配置

```go
// 将配置转换为 JSON 字符串
jsonStr, err := config.ToJSON()
if err != nil {
    log.Fatalf("序列化配置失败: %v", err)
}

fmt.Println(jsonStr)
```

## 配置字段说明

### 通用字段

- `model` (必需): 模型名称，如 "gemini-1.5-pro", "gpt-4", "qwen-turbo"
- `defaultTemperature` (可选): 默认温度值，控制输出的随机性 (0-2)
- `defaultMaxTokens` (可选): 默认最大 token 数

### Azure OpenAI 特定字段

- `azureEndpoint` (必需): Azure OpenAI 资源端点
- `azureDeployment` (必需): 部署名称
- `azureApiVersion` (必需): API 版本，如 "2024-02-15-preview"

### 百炼特定字段

- `bailianEndpoint` (必需): 百炼 API 端点
- `bailianWorkspace` (必需): 工作空间名称

## 错误处理

### 常见错误

1. **配置 JSON 为空**

   ```
   错误: 配置JSON不能为空
   ```

2. **JSON 格式无效**

   ```
   错误: 解析配置JSON失败: invalid character...
   ```

3. **模型名称为空**

   ```
   错误: 模型名称不能为空
   ```

4. **Azure OpenAI 缺少必需字段**

   ```
   错误: Azure OpenAI 配置缺少必需字段: azureEndpoint
   错误: Azure OpenAI 配置缺少必需字段: azureDeployment
   错误: Azure OpenAI 配置缺少必需字段: azureApiVersion
   ```

5. **百炼缺少必需字段**

   ```
   错误: 百炼配置缺少必需字段: bailianEndpoint
   错误: 百炼配置缺少必需字段: bailianWorkspace
   ```

6. **不支持的提供商类型**

   ```
   错误: 不支持的提供商类型: unknown
   ```

## 完整示例

```go
package main

import (
    "fmt"
    "log"
    "genkit-ai-service/internal/genkit"
)

func main() {
    // 示例 1: Google AI 配置
    googleConfig := `{
        "model": "gemini-1.5-pro",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }`
    
    config, err := genkit.ParseGenkitConfig(googleConfig)
    if err != nil {
        log.Fatalf("解析配置失败: %v", err)
    }
    
    if err := config.Validate("googlegenai"); err != nil {
        log.Fatalf("配置验证失败: %v", err)
    }
    
    fmt.Println("Google AI 配置验证成功")
    
    // 示例 2: Azure OpenAI 配置
    azureConfig := `{
        "model": "gpt-4",
        "azureEndpoint": "https://your-resource.openai.azure.com",
        "azureDeployment": "gpt-4",
        "azureApiVersion": "2024-02-15-preview",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }`
    
    config, err = genkit.ParseGenkitConfig(azureConfig)
    if err != nil {
        log.Fatalf("解析配置失败: %v", err)
    }
    
    if err := config.Validate("azureopenai"); err != nil {
        log.Fatalf("配置验证失败: %v", err)
    }
    
    fmt.Println("Azure OpenAI 配置验证成功")
    
    // 示例 3: 百炼配置
    bailianConfig := `{
        "model": "qwen-turbo",
        "bailianEndpoint": "https://dashscope.aliyuncs.com",
        "bailianWorkspace": "default",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }`
    
    config, err = genkit.ParseGenkitConfig(bailianConfig)
    if err != nil {
        log.Fatalf("解析配置失败: %v", err)
    }
    
    if err := config.Validate("bianlian"); err != nil {
        log.Fatalf("配置验证失败: %v", err)
    }
    
    fmt.Println("百炼配置验证成功")
}
```

## 与 ModelConfiguration 集成

在实际使用中，`GenkitConfig` 通常从 `model_configurations` 表的 `query_params` 字段解析：

```go
import (
    "context"
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/repository"
)

func getModelConfig(ctx context.Context, repo repository.ModelConfigurationRepository, tenantID, modelName string) (*genkit.GenkitConfig, error) {
    // 从数据库获取模型配置
    modelConfig, err := repo.GetByTenantAndModel(ctx, tenantID, modelName)
    if err != nil {
        return nil, err
    }
    
    // 解析配置 JSON
    config, err := genkit.ParseGenkitConfig(*modelConfig.QueryParams)
    if err != nil {
        return nil, err
    }
    
    // 验证配置
    if err := config.Validate(modelConfig.ModelProvider); err != nil {
        return nil, err
    }
    
    return config, nil
}
```
