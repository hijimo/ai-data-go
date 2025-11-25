# getOrInitGenkit() 方法使用指南

## 概述

`getOrInitGenkit()` 方法是 Genkit Client 的核心方法，用于根据租户ID和模型名称动态获取或初始化 Genkit 实例。

## 方法签名

```go
func (c *client) getOrInitGenkit(ctx context.Context, tenantID, modelName string) (*genkit.Genkit, *GenkitConfig, error)
```

## 参数说明

- `ctx`: 上下文对象
- `tenantID`: 租户ID（UUID字符串格式）
- `modelName`: 模型配置名称（对应 `model_configurations.name` 字段）

## 返回值

- `*genkit.Genkit`: Genkit 实例
- `*GenkitConfig`: 解析后的配置对象
- `error`: 错误信息

## 工作流程

1. **缓存查找**：首先检查是否已有缓存的实例
2. **配置查询**：从数据库查询租户的模型配置
3. **配置验证**：验证配置的完整性和有效性
4. **插件初始化**：根据提供商类型创建对应的插件
5. **实例缓存**：将初始化的实例存入缓存

## 使用示例

### 基本使用

```go
// 创建客户端（注入 ModelConfigurationRepository）
client := genkit.NewClientWithRepo(configRepo)

// 获取或初始化 Genkit 实例
g, config, err := client.getOrInitGenkit(ctx, tenantID, "gemini-pro")
if err != nil {
    log.Fatalf("初始化失败: %v", err)
}

// 使用 Genkit 实例进行生成
resp, err := genkit.Generate(ctx, g, ai.WithPrompt("你好"))
```

### 在 Generate 方法中使用

```go
func (c *client) Generate(ctx context.Context, tenantID, modelName, prompt string) (*GenerateResult, error) {
    // 获取或初始化 Genkit 实例
    g, config, err := c.getOrInitGenkit(ctx, tenantID, modelName)
    if err != nil {
        return nil, fmt.Errorf("获取 Genkit 实例失败: %w", err)
    }
    
    // 使用实例进行生成
    resp, err := genkit.Generate(ctx, g, ai.WithPrompt(prompt))
    if err != nil {
        return nil, fmt.Errorf("生成内容失败: %w", err)
    }
    
    // 构建结果
    return &GenerateResult{
        Text:  resp.Text(),
        Model: config.Model,
        Usage: extractUsage(resp.Usage),
    }, nil
}
```

## 错误处理

### 常见错误

1. **无效的租户ID**

   ```
   错误: 无效的租户ID: invalid UUID length: 11
   原因: 租户ID格式不正确
   解决: 确保传入有效的UUID字符串
   ```

2. **模型配置不存在**

   ```
   错误: 获取模型配置失败: 模型配置不存在
   原因: 数据库中没有对应的配置记录
   解决: 检查租户ID和模型名称是否正确
   ```

3. **模型已禁用**

   ```
   错误: 模型已禁用: gemini-pro
   原因: 配置的 is_enabled 字段为 false
   解决: 启用模型配置或使用其他模型
   ```

4. **配置验证失败**

   ```
   错误: 配置验证失败: Azure OpenAI 配置缺少必需字段: azureEndpoint
   原因: 配置不完整
   解决: 补充缺失的配置字段
   ```

5. **不支持的提供商**

   ```
   错误: 不支持的提供商类型: unknown-provider
   原因: 提供商类型不在支持列表中
   解决: 使用支持的提供商类型（googlegenai, azureopenai, bianlian）
   ```

## 性能优化

### 缓存机制

- 实例按 `tenantID_modelName` 缓存
- 首次访问时初始化，后续访问直接从缓存获取
- 使用读写锁保证并发安全

### 并发访问

```go
// 多个 goroutine 可以安全地并发调用
for i := 0; i < 10; i++ {
    go func() {
        g, config, err := client.getOrInitGenkit(ctx, tenantID, modelName)
        // 使用 g 和 config
    }()
}
```

## 配置要求

### 数据库配置

在 `model_configurations` 表中需要有对应的记录：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key, 
    query_params,
    is_enabled
) VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'gemini-pro',
    'gemini-1.5-pro',
    'googlegenai',
    'your-api-key',
    '{"model":"gemini-1.5-pro","defaultTemperature":0.7,"defaultMaxTokens":2048}',
    true
);
```

### QueryParams 格式

```json
{
  "model": "gemini-1.5-pro",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

对于 Azure OpenAI：

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

对于百炼：

```json
{
  "model": "qwen-turbo",
  "bailianEndpoint": "https://dashscope.aliyuncs.com",
  "bailianWorkspace": "default",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

## 注意事项

1. **仓储注入**：必须使用 `NewClientWithRepo()` 创建客户端，否则会返回"模型配置仓储未初始化"错误

2. **租户隔离**：不同租户的相同模型名称会创建独立的实例

3. **配置更新**：如果数据库中的配置更新，需要清除缓存或重启应用才能生效

4. **提供商支持**：当前只实现了 `googlegenai` 提供商，其他提供商会返回"暂未实现"错误

## 后续扩展

在后续任务中将实现：

- Azure OpenAI 插件集成（TASK-3.x）
- 百炼插件集成（TASK-4.x）
- 配置热更新机制
- 实例生命周期管理
