# Genkit 插件使用示例

## 概述

本文档提供了如何使用 Genkit Client 的插件动态创建功能的示例。

## 基本用法

### 1. 创建客户端

```go
import (
    "context"
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/repository"
)

// 创建带有模型配置仓储的客户端
client := genkit.NewClientWithRepo(modelConfigRepo)
```

### 2. 使用 getOrInitGenkit 获取实例

```go
ctx := context.Background()
tenantID := "tenant-uuid-123"
modelName := "gpt-4"

// 自动从数据库查询配置并初始化对应的插件
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    log.Fatalf("初始化失败: %v", err)
}

// 使用 Genkit 实例进行生成
result, err := genkit.Generate(ctx, g, ai.WithPrompt("Hello, world!"))
if err != nil {
    log.Fatalf("生成失败: %v", err)
}

fmt.Println(result.Text())
```

## 提供商配置示例

### Google AI (Gemini)

**数据库配置**：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    query_params
) VALUES (
    'tenant-uuid-123',
    'Gemini Pro',
    'gemini-1.5-pro',
    'googlegenai',
    'your-google-api-key',
    '{"defaultTemperature": 0.7, "defaultMaxTokens": 2048}'
);
```

**使用**：

```go
g, config, err := client.getOrInitGenkit(ctx, "tenant-uuid-123", "gemini-1.5-pro")
// 自动使用 Google AI 插件
```

### OpenAI

**数据库配置**：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    query_params
) VALUES (
    'tenant-uuid-123',
    'GPT-4',
    'gpt-4',
    'openai',
    'your-openai-api-key',
    '{"defaultTemperature": 0.7, "defaultMaxTokens": 2048}'
);
```

**使用**：

```go
g, config, err := client.getOrInitGenkit(ctx, "tenant-uuid-123", "gpt-4")
// 自动使用 OpenAI 插件
```

### Azure OpenAI

**数据库配置**：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    query_params
) VALUES (
    'tenant-uuid-123',
    'Azure GPT-4',
    'gpt-4',
    'azureopenai',
    'your-azure-api-key',
    '{
        "azureEndpoint": "https://your-resource.openai.azure.com",
        "azureDeployment": "gpt-4-deployment",
        "azureApiVersion": "2024-02-15-preview",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }'
);
```

**使用**：

```go
g, config, err := client.getOrInitGenkit(ctx, "tenant-uuid-123", "gpt-4")
// 自动使用 OpenAI 插件 + Azure BaseURL
```

### 阿里云百炼

**数据库配置**：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    query_params
) VALUES (
    'tenant-uuid-123',
    '通义千问',
    'qwen-plus',
    'bianlian',
    'your-dashscope-api-key',
    '{
        "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "bailianWorkspace": "default",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }'
);
```

**使用**：

```go
g, config, err := client.getOrInitGenkit(ctx, "tenant-uuid-123", "qwen-plus")
// 自动使用 OpenAI 插件 + 百炼兼容模式 BaseURL
```

### Anthropic (Claude)

**数据库配置**：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    query_params
) VALUES (
    'tenant-uuid-123',
    'Claude 3 Opus',
    'claude-3-opus-20240229',
    'anthropic',
    'your-anthropic-api-key',
    '{"defaultTemperature": 0.7, "defaultMaxTokens": 2048}'
);
```

**使用**：

```go
g, config, err := client.getOrInitGenkit(ctx, "tenant-uuid-123", "claude-3-opus-20240229")
// 自动使用 Anthropic 插件
```

### 自定义 OpenAI 兼容服务

**数据库配置**：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    base_url,
    query_params
) VALUES (
    'tenant-uuid-123',
    '自定义模型',
    'custom-model',
    'custom_openai',
    'your-custom-api-key',
    'https://custom-openai-service.com/v1',
    '{"defaultTemperature": 0.7, "defaultMaxTokens": 2048}'
);
```

**使用**：

```go
g, config, err := client.getOrInitGenkit(ctx, "tenant-uuid-123", "custom-model")
// 自动使用 OpenAI 插件 + 自定义 BaseURL
```

## 缓存机制

客户端会自动缓存 Genkit 实例，缓存键为 `{tenantID}_{modelName}`：

```go
// 第一次调用：从数据库查询配置并初始化
g1, _, err := client.getOrInitGenkit(ctx, "tenant-123", "gpt-4")

// 第二次调用：直接从缓存获取，无需重新初始化
g2, _, err := client.getOrInitGenkit(ctx, "tenant-123", "gpt-4")

// g1 和 g2 是同一个实例
```

### 清理缓存

```go
// 清理特定租户和模型的缓存
client.ClearCache("tenant-123", "gpt-4")

// 清理所有缓存
client.ClearAllCache()

// 获取当前缓存大小
size := client.GetCacheSize()
```

## 错误处理

```go
g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "获取模型配置失败"):
        // 配置不存在或数据库错误
        log.Printf("配置错误: %v", err)
        
    case strings.Contains(err.Error(), "模型已禁用"):
        // 模型被禁用
        log.Printf("模型不可用: %v", err)
        
    case strings.Contains(err.Error(), "配置验证失败"):
        // 配置不完整或无效
        log.Printf("配置无效: %v", err)
        
    case strings.Contains(err.Error(), "不支持的提供商类型"):
        // 提供商类型不支持
        log.Printf("提供商不支持: %v", err)
        
    default:
        log.Printf("未知错误: %v", err)
    }
    return
}
```

## 并发安全

客户端使用读写锁保护缓存，支持并发访问：

```go
// 多个 goroutine 可以安全地并发调用
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        g, _, err := client.getOrInitGenkit(ctx, "tenant-123", "gpt-4")
        if err != nil {
            log.Printf("错误: %v", err)
            return
        }
        // 使用 g 进行生成...
    }()
}
wg.Wait()
```

## 最佳实践

### 1. 复用客户端实例

```go
// ✅ 推荐：在应用启动时创建一个客户端实例，全局复用
var genkitClient genkit.Client

func init() {
    genkitClient = genkit.NewClientWithRepo(modelConfigRepo)
}

// ❌ 不推荐：每次都创建新的客户端实例
func handleRequest() {
    client := genkit.NewClientWithRepo(modelConfigRepo) // 浪费资源
    // ...
}
```

### 2. 配置更新后清理缓存

```go
// 更新模型配置后，清理对应的缓存
func updateModelConfig(tenantID, modelName string, newConfig *Config) error {
    // 更新数据库配置
    if err := repo.Update(ctx, tenantID, modelName, newConfig); err != nil {
        return err
    }
    
    // 清理缓存，强制下次使用时重新初始化
    genkitClient.ClearCache(tenantID, modelName)
    
    return nil
}
```

### 3. 使用上下文控制超时

```go
// 设置超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("初始化超时")
    }
    return err
}
```

## 监控和日志

建议在生产环境中添加监控和日志：

```go
import "genkit-ai-service/internal/logger"

// 记录初始化
logger.InfoContext(ctx, "初始化 Genkit 实例", 
    "tenant_id", tenantID,
    "model_name", modelName,
    "provider", config.ModelProvider,
)

// 记录缓存命中
if cached {
    logger.DebugContext(ctx, "使用缓存的 Genkit 实例",
        "tenant_id", tenantID,
        "model_name", modelName,
    )
}

// 记录错误
if err != nil {
    logger.ErrorContext(ctx, "初始化 Genkit 实例失败",
        "tenant_id", tenantID,
        "model_name", modelName,
        "error", err.Error(),
    )
}
```

## 参考资料

- [Genkit 官方文档](https://firebase.google.com/docs/genkit)
- [插件研究文档](../../docs/genkit-plugin-research.md)
- [实现总结](./PLUGIN_DYNAMIC_CREATION_SUMMARY.md)
