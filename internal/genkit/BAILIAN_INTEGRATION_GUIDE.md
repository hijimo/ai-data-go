# 阿里云百炼集成指南

## 概述

本指南介绍如何在 Genkit AI Service 中使用阿里云百炼（DashScope）模型。

## 百炼 API 特点

- **OpenAI 兼容**: 百炼提供完全兼容 OpenAI API 的接口
- **多地域支持**: 支持北京、新加坡、金融云等多个地域
- **丰富的模型**: 支持 Qwen 系列模型（qwen-plus, qwen-max, qwen-turbo 等）

## 配置方式

### 1. 数据库配置

在 `model_configurations` 表中添加百炼模型配置：

```sql
INSERT INTO model_configurations (
    tenant_id, 
    model_name, 
    provider_type, 
    api_key, 
    configuration
) VALUES (
    'your-tenant-uuid',
    'qwen-plus',
    'bianlian',
    'your-bailian-api-key',
    '{
        "model": "qwen-plus",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }'
);
```

### 2. 配置字段说明

#### 必需字段

- `tenant_id`: 租户 UUID
- `model_name`: 模型名称（用于标识，可自定义）
- `provider_type`: 必须为 `"bianlian"`
- `api_key`: 百炼 API 密钥
- `configuration.model`: 实际的百炼模型名称

#### 可选字段

- `configuration.bailianEndpoint`: 自定义 API 端点（默认使用北京地域）
- `configuration.defaultTemperature`: 默认温度参数
- `configuration.defaultMaxTokens`: 默认最大 token 数

## 支持的端点

### 北京地域（默认）

```
https://dashscope.aliyuncs.com/compatible-mode/v1
```

### 新加坡地域

```
https://dashscope-intl.aliyuncs.com/compatible-mode/v1
```

### 金融云

```
https://dashscope-finance.aliyuncs.com/compatible-mode/v1
```

## 配置示例

### 示例 1: 使用默认端点（北京）

```sql
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) 
VALUES (
    '738dbb1f-83e6-4bf5-935c-f0498236440d',
    'qwen-plus',
    'bianlian',
    'sk-xxx',
    '{
        "model": "qwen-plus",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }'
);
```

### 示例 2: 使用新加坡地域

```sql
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) 
VALUES (
    '738dbb1f-83e6-4bf5-935c-f0498236440d',
    'qwen-max-intl',
    'bianlian',
    'sk-xxx',
    '{
        "model": "qwen-max",
        "bailianEndpoint": "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }'
);
```

### 示例 3: 使用金融云

```sql
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) 
VALUES (
    '738dbb1f-83e6-4bf5-935c-f0498236440d',
    'qwen-turbo-finance',
    'bianlian',
    'sk-xxx',
    '{
        "model": "qwen-turbo",
        "bailianEndpoint": "https://dashscope-finance.aliyuncs.com/compatible-mode/v1",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
    }'
);
```

## 使用方式

### Go 代码调用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/repository"
)

func main() {
    ctx := context.Background()
    
    // 创建配置仓储（假设已初始化数据库连接）
    configRepo := repository.NewModelConfigurationRepository(db)
    
    // 创建 Genkit 客户端
    client := genkit.NewClientWithRepo(configRepo)
    
    // 调用百炼模型（非流式）
    result, err := client.Generate(
        ctx,
        "738dbb1f-83e6-4bf5-935c-f0498236440d", // 租户ID
        "qwen-plus",                             // 模型名称
        "你好，请用中文介绍一下阿里云百炼",        // 提示词
        nil,                                     // 选项
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("回复:", result.Text)
    fmt.Printf("Token 使用: %d (输入: %d, 输出: %d)\n",
        result.Usage.TotalTokens,
        result.Usage.PromptTokens,
        result.Usage.CompletionTokens,
    )
}
```

### 流式调用

```go
// 调用百炼模型（流式）
streamChan, err := client.GenerateStream(
    ctx,
    "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "qwen-plus",
    "请写一首关于人工智能的诗",
    nil,
)
if err != nil {
    log.Fatal(err)
}

// 处理流式响应
for chunk := range streamChan {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    
    if chunk.Done {
        // 流式结束
        fmt.Printf("\n\nToken 使用: %d\n", chunk.Usage.TotalTokens)
        break
    }
    
    // 输出内容块
    fmt.Print(chunk.Content)
}
```

### HTTP API 调用

```bash
# 非流式调用
curl -X POST http://localhost:8080/api/v1/chat/sessions/{session_id}/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "message": "你好，请介绍一下自己",
    "options": {
      "modelName": "qwen-plus"
    }
  }'

# 流式调用
curl -X POST http://localhost:8080/api/v1/chat/sessions/{session_id}/messages/stream \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "message": "请写一首关于人工智能的诗",
    "options": {
      "modelName": "qwen-plus"
    }
  }'
```

## 支持的模型

百炼支持多种 Qwen 系列模型：

- `qwen-turbo`: 快速响应，适合对话场景
- `qwen-plus`: 平衡性能和质量
- `qwen-max`: 最强性能，适合复杂任务
- `qwen-max-longcontext`: 支持长文本上下文
- 其他模型请参考[百炼文档](https://help.aliyun.com/zh/model-studio/)

## 常见问题

### Q1: 如何获取百炼 API 密钥？

访问[阿里云百炼控制台](https://dashscope.console.aliyun.com/)，在 API-KEY 管理页面创建新的 API 密钥。

### Q2: 如何选择合适的地域？

- **北京地域**: 国内用户，默认选择
- **新加坡地域**: 海外用户或需要国际访问
- **金融云**: 金融行业客户，满足合规要求

### Q3: 百炼支持哪些功能？

由于百炼完全兼容 OpenAI API，支持：

- ✅ 文本生成
- ✅ 流式输出
- ✅ 多轮对话
- ✅ 系统提示词
- ✅ 温度和 token 控制

### Q4: 如何处理中文？

百炼原生支持中文，无需特殊处理。Qwen 系列模型在中文理解和生成方面表现优秀。

### Q5: 如何切换模型？

只需在配置中修改 `model` 字段，或创建新的配置记录使用不同的 `model_name`。

## 错误处理

### 常见错误

1. **API 密钥无效**

   ```
   错误: 获取模型实例失败: 创建百炼插件失败: invalid API key
   ```

   解决: 检查 API 密钥是否正确

2. **模型不存在**

   ```
   错误: 模型已禁用: qwen-xxx
   ```

   解决: 检查模型名称是否正确，或在数据库中启用该模型

3. **配置不存在**

   ```
   错误: 获取模型配置失败: record not found
   ```

   解决: 确保在 `model_configurations` 表中添加了对应的配置

## 性能优化

### 1. 实例缓存

Genkit Client 会自动缓存已初始化的实例，相同租户和模型的后续调用会复用实例，提高性能。

### 2. 并发控制

客户端使用读写锁保护实例缓存，支持高并发访问。

### 3. 连接复用

底层 HTTP 连接会自动复用，减少连接开销。

## 监控和日志

### 日志示例

```
INFO  获取或初始化 Genkit 实例 tenant_id=738dbb1f-83e6-4bf5-935c-f0498236440d model_name=qwen-plus
INFO  初始化提供商 provider=bianlian model=qwen-plus
INFO  生成内容成功 model=qwen-plus tokens=150 duration=1.2s
```

### 监控指标

- 调用次数
- 响应时间
- Token 使用量
- 错误率

## 参考资料

- [阿里云百炼官方文档](https://help.aliyun.com/zh/model-studio/)
- [百炼 API 参考](https://help.aliyun.com/zh/model-studio/developer-reference/api-reference)
- [Qwen 模型介绍](https://help.aliyun.com/zh/model-studio/getting-started/models)
- [OpenAI API 兼容说明](https://help.aliyun.com/zh/model-studio/developer-reference/compatibility-of-openai-with-dashscope)

## 更新日志

- **2025-11-28**: 完成百炼插件集成到 Genkit Client
- **2025-11-27**: 实现百炼自定义插件
- **2025-11-26**: 完成百炼集成调研
