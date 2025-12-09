# Bailian 插件集成说明

本文档说明如何在 Genkit Client 中集成和使用 Bailian 插件。

## 集成方式

Bailian 插件已经集成到 `internal/genkit/client.go` 中，通过 `initializeProvider` 函数自动处理。

### 1. 导入插件

```go
import (
    "genkit-ai-service/internal/genkit/plugins/bailian"
)
```

### 2. 创建插件实例

在 `client.go` 中的 `createBailianPlugin` 函数负责创建 Bailian 插件实例：

```go
func createBailianPlugin(ctx context.Context, apiKey string, baseURL *string) (*bailian.Bailian, error) {
    // 构建 RequestOption 列表
    opts := []option.RequestOption{
        option.WithAPIKey(apiKey),
    }

    // 设置 Base URL
    bailianBaseURL := "https://dashscope.aliyuncs.com/compatible-mode/v1"
    if baseURL != nil && *baseURL != "" {
        bailianBaseURL = *baseURL
    }
    opts = append(opts, option.WithBaseURL(bailianBaseURL))

    // 创建 Bailian 插件实例
    plugin := &bailian.Bailian{
        Opts: opts,
    }

    return plugin, nil
}
```

### 3. 初始化提供商

在 `initializeProvider` 函数中处理 "bianlian" 提供商类型：

```go
case "bianlian":
    logger.InfoContext(ctx, "初始化阿里云百炼提供商", logger.Fields{
        "provider": "bianlian",
        "model":    genkitConfig.Model,
        "baseURL":  tempConfig.BaseURL,
    })

    plugin, err := createBailianPlugin(ctx, tempConfig.APIKey, tempConfig.BaseURL)
    if err != nil {
        return nil, fmt.Errorf("创建百炼插件失败: %w", err)
    }

    // 百炼插件使用 "bailian" 作为提供商前缀
    fullModelName = "bailian/" + genkitConfig.Model

    // 初始化 Genkit 实例
    g = genkit.Init(ctx,
        genkit.WithPlugins(plugin),
        genkit.WithDefaultModel(fullModelName),
    )
```

## 配置说明

### 数据库配置

在 `model_configurations` 表中创建百炼配置：

```sql
INSERT INTO model_configurations (
    tenant_id,
    name,
    model,
    model_provider,
    api_key,
    base_url,
    is_enabled
) VALUES (
    'tenant-uuid',
    'qwen-turbo',
    'qwen-turbo',
    'bianlian',
    'sk-your-api-key',
    'https://dashscope.aliyuncs.com/compatible-mode/v1',
    true
);
```

### API 请求示例

```bash
curl -X POST http://localhost:8080/api/v1/chat/generate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-jwt-token" \
  -d '{
    "tenantId": "tenant-uuid",
    "modelName": "qwen-turbo",
    "prompt": "你好，请介绍一下自己"
  }'
```

## 模型名称映射

| 数据库中的 model | Genkit 中的完整模型名 | 说明 |
|-----------------|---------------------|------|
| qwen-turbo | bailian/qwen-turbo | 通义千问 Turbo |
| qwen-plus | bailian/qwen-plus | 通义千问 Plus |
| qwen-max | bailian/qwen-max | 通义千问 Max |
| qwen3-max | bailian/qwen3-max | 通义千问 3 Max |
| qwen-vl-plus | bailian/qwen-vl-plus | 通义千问 VL Plus（多模态）|
| qwen-vl-max | bailian/qwen-vl-max | 通义千问 VL Max（多模态）|

## 配置传递方式

插件不使用环境变量，所有配置通过代码传入。在多租户场景下，配置从数据库读取：

```go
// 从数据库获取配置后，创建插件
plugin := &bailian.Bailian{
    Opts: []option.RequestOption{
        option.WithAPIKey(apiKey),           // 从数据库读取
        option.WithBaseURL(baseURL),         // 从数据库读取
    },
}
```

**重要说明**：
- API Key 会自动设置为 `Authorization: Bearer {apiKey}` header
- 这符合百炼 API 的认证要求

## 缓存机制

Client 会缓存每个租户+模型的 Genkit 实例，提高性能：

- 缓存键格式：`{tenantID}_{modelName}`
- 首次调用时初始化并缓存
- 后续调用直接使用缓存的实例
- 可通过 `ClearCache` 方法清理缓存

## 日志记录

插件会记录以下关键事件：

1. **插件创建**：记录 Base URL 和配置信息
2. **提供商初始化**：记录完整的模型名称
3. **生成请求**：记录租户ID、模型名称、耗时等
4. **错误处理**：记录详细的错误信息

示例日志：

```json
{
  "level": "info",
  "msg": "创建百炼插件",
  "baseURL": "https://dashscope.aliyuncs.com/compatible-mode/v1"
}

{
  "level": "info",
  "msg": "阿里云百炼提供商初始化成功",
  "provider": "bianlian",
  "fullModelName": "bailian/qwen-turbo"
}
```

## 错误处理

常见错误及处理方式：

### 1. API Key 无效

```
错误: 创建百炼插件失败: unauthorized
解决: 检查 model_configurations 表中的 api_key 是否正确
```

### 2. Base URL 错误

```
错误: 连接失败: dial tcp: lookup invalid-url
解决: 检查 base_url 字段是否正确配置
```

### 3. 模型不存在

```
错误: 模型不支持: invalid-model
解决: 确认使用的是支持的模型名称（qwen-turbo, qwen-plus, qwen-max 等）
```

## 性能优化建议

1. **使用缓存**：避免频繁创建新的 Genkit 实例
2. **连接池**：HTTP 客户端会自动管理连接池
3. **超时设置**：建议设置合理的请求超时时间
4. **并发控制**：根据 API 限制控制并发请求数

## 测试

运行集成测试：

```bash
# 设置环境变量
export BAILIAN_API_KEY="your-api-key"

# 运行测试
go test -v ./internal/genkit/plugins/bailian/...
```

## 相关文件

- `internal/genkit/plugins/bailian/bailian.go` - 插件实现
- `internal/genkit/plugins/bailian/README.md` - 插件使用文档
- `internal/genkit/client.go` - Client 集成代码
- `internal/service/model_configuration_service.go` - 配置验证逻辑

## 更新历史

- 2024-12-09: 创建新的 Bailian 插件，替换旧的实现
- 使用标准的 OpenAI 兼容模式
- 简化配置，只需要 API Key 和 Base URL
