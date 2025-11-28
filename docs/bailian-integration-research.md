# 阿里云百炼集成调研报告

## 调研日期

2025-11-27

## 调研目标

调研阿里云百炼 API，确定与 Genkit 框架的集成方案。

## 核心发现

### ✅ 百炼支持 OpenAI 兼容接口

阿里云百炼**完全兼容 OpenAI API 规范**，这是一个重大发现！

**关键信息**：

- 百炼提供 OpenAI 兼容的 API 接口
- 只需调整 API Key、base_url 和模型名称，即可将 OpenAI 代码迁移至百炼
- 支持流式和非流式调用
- 支持多模态（文本、图像、视频）

## API 接口详情

### 1. 接口地址

#### 北京地域（中国大陆）

- **Base URL**: `https://dashscope.aliyuncs.com/compatible-mode/v1`
- **Chat Completions**: `POST https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions`

#### 新加坡地域（国际）

- **Base URL**: `https://dashscope-intl.aliyuncs.com/compatible-mode/v1`
- **Chat Completions**: `POST https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions`

#### 金融云

- **Base URL**: `https://dashscope-finance.aliyuncs.com/compatible-mode/v1`
- **Chat Completions**: `POST https://dashscope-finance.aliyuncs.com/compatible-mode/v1/chat/completions`

### 2. 认证方式

- **认证方式**: Bearer Token
- **API Key 获取**: 通过阿里云百炼控制台的密钥管理页面创建
- **环境变量**: `DASHSCOPE_API_KEY`

### 3. 支持的模型

#### 通义千问系列

- **qwen-max**: 通义千问系列效果最好的模型，适合处理复杂、多步骤任务
- **qwen-plus**: 在效果、速度和成本上表现均衡，适用于通用场景
- **qwen-turbo**: 高性价比、低延迟，适合需要快速响应的简单任务
- **qwen-coder**: 擅长工具调用和环境交互，专用于代码生成与理解

#### 多模态模型

- **qwen-vl-plus**: 视觉理解模型
- **qwen-vl-max**: 高级视觉理解模型
- **qwen-vl-max-latest**: 最新视觉理解模型

#### 其他第三方模型

- DeepSeek
- Kimi
- GLM

### 4. API 调用示例

#### Go 语言示例（非流式）

```go
package main

import (
    "context"
    "os"
    
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

func main() {
    client := openai.NewClient(
        option.WithAPIKey(os.Getenv("DASHSCOPE_API_KEY")),
        option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
    )
    
    chatCompletion, err := client.Chat.Completions.New(
        context.TODO(), 
        openai.ChatCompletionNewParams{
            Messages: []openai.ChatCompletionMessageParamUnion{
                openai.UserMessage("你是谁"),
            },
            Model: "qwen-plus",
        },
    )
    
    if err != nil {
        panic(err.Error())
    }
    
    println(chatCompletion.Choices[0].Message.Content)
}
```

#### 流式调用

百炼完全支持 OpenAI 的流式调用规范：

- 设置 `stream: true` 参数
- 支持 `stream_options: {include_usage: true}` 获取 Token 使用统计
- 返回 SSE (Server-Sent Events) 格式的流式响应

## 集成方案决策

### ✅ 推荐方案：使用 OpenAI 插件 + 自定义 BaseURL

**方案描述**：
使用 Genkit 的 OpenAI 插件，通过配置自定义 BaseURL 来调用百炼 API。

**技术实现**：

```go
import (
    "github.com/firebase/genkit/go/plugins/openai"
    "github.com/openai/openai-go/option"
)

func createBailianPlugin(apiKey string, config *ModelConfig) genkit.Plugin {
    return &openai.OpenAI{
        Opts: []option.RequestOption{
            option.WithAPIKey(apiKey),
            option.WithBaseURL(config.BailianEndpoint), // https://dashscope.aliyuncs.com/compatible-mode/v1
        },
    }
}
```

**配置示例**：

```json
{
    "model": "qwen-plus",
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}
```

**模型名称格式**：

- 在 Genkit 中注册时使用：`openai/qwen-plus`
- 实际调用时使用：`qwen-plus`

### 方案优势

1. **无需自定义插件**：直接使用 Genkit 官方的 OpenAI 插件
2. **完全兼容**：百炼 API 完全兼容 OpenAI 规范
3. **维护成本低**：无需维护自定义插件代码
4. **实现简单**：只需配置 BaseURL 和 API Key
5. **功能完整**：支持流式、非流式、多模态等所有功能

### 方案劣势

1. **依赖 OpenAI SDK**：需要确保 Genkit 的 OpenAI 插件支持自定义 BaseURL
2. **模型名称映射**：需要正确映射百炼的模型名称

## 配置字段定义

### 必需字段

- `apiKey`: 百炼 API Key
- `bailianEndpoint`: 百炼 API 端点（根据地域选择）
- `model`: 模型名称（如 qwen-plus, qwen-max, qwen-turbo）

### 可选字段

- `defaultTemperature`: 默认温度参数（0-2）
- `defaultMaxTokens`: 默认最大 Token 数
- `region`: 地域选择（beijing, singapore, finance）

## 数据库配置示例

```sql
-- 百炼配置示例
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) VALUES
('tenant-uuid-1', 'qwen-plus', 'bailian', 'sk-xxx', '{
    "model": "qwen-plus",
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "region": "beijing",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}');

-- 不同地域的配置
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) VALUES
('tenant-uuid-2', 'qwen-max', 'bailian', 'sk-xxx', '{
    "model": "qwen-max",
    "bailianEndpoint": "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
    "region": "singapore",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 4096
}');
```

## 实现要点

### 1. 插件创建

```go
func (c *client) createBailianPlugin(apiKey string, modelConfig *ModelConfig) (genkit.Plugin, error) {
    // 验证配置
    if modelConfig.BailianEndpoint == "" {
        return nil, fmt.Errorf("百炼 Endpoint 不能为空")
    }
    
    // 创建 OpenAI 插件，使用百炼的 BaseURL
    plugin := &openai.OpenAI{
        Opts: []option.RequestOption{
            option.WithAPIKey(apiKey),
            option.WithBaseURL(modelConfig.BailianEndpoint),
        },
    }
    
    return plugin, nil
}
```

### 2. 模型名称处理

```go
// 在 getOrInitGenkit 中
switch config.ProviderType {
case ProviderBailian:
    plugin, err := c.createBailianPlugin(config.APIKey, &modelConfig)
    if err != nil {
        return nil, nil, fmt.Errorf("创建百炼插件失败: %w", err)
    }
    
    // 使用 openai 前缀，因为使用的是 OpenAI 插件
    fullModelName = "openai/" + modelConfig.Model
}
```

### 3. 错误处理

百炼 API 返回的错误格式与 OpenAI 兼容，可以直接使用 OpenAI SDK 的错误处理机制。

### 4. 流式响应

百炼的流式响应完全兼容 OpenAI 的 SSE 格式，无需特殊处理。

## 测试计划

### 1. 非流式调用测试

- 测试基本的文本生成
- 测试中文处理能力
- 测试参数传递（temperature, maxTokens）
- 测试 Token 统计
- 测试错误处理

### 2. 流式调用测试

- 测试流式响应接收
- 测试中文流式输出
- 测试流式响应完整性
- 测试流式中断处理
- 验证 SSE 格式转换

### 3. 多模型测试

- 测试 qwen-plus
- 测试 qwen-max
- 测试 qwen-turbo
- 测试模型切换

### 4. 地域测试

- 测试北京地域
- 测试新加坡地域（如果需要）

## 性能考虑

### 延迟

- 北京地域：适合中国大陆用户，延迟较低
- 新加坡地域：适合国际用户
- 建议根据用户地理位置选择合适的地域

### 并发

- 百炼支持高并发调用
- 建议实现连接池复用
- 建议实现请求限流

### 成本

- 按 Token 计费
- qwen-turbo 性价比最高
- qwen-plus 平衡性能和成本
- qwen-max 效果最好但成本较高

## 安全考虑

### API Key 管理

- API Key 存储在数据库中（加密）
- 日志中脱敏处理
- 不在错误信息中暴露

### 数据隐私

- 阿里云严格保护数据隐私
- 不会将用户数据用于模型训练
- 传输数据经过加密

## 参考文档

- [阿里云百炼官方文档](https://help.aliyun.com/zh/model-studio/)
- [首次调用通义千问 API](https://help.aliyun.com/zh/model-studio/first-api-call-to-qwen)
- [通义千问 API 参考](https://help.aliyun.com/zh/model-studio/qwen-api-reference)
- [模型列表](https://help.aliyun.com/zh/model-studio/models)
- [流式输出](https://help.aliyun.com/zh/model-studio/stream)

## 结论

阿里云百炼完全支持 OpenAI 兼容接口，这使得集成变得非常简单。我们可以直接使用 Genkit 的 OpenAI 插件，通过配置自定义 BaseURL 来调用百炼 API，无需开发自定义插件。

**推荐实施方案**：

1. 使用 Genkit OpenAI 插件
2. 配置百炼的 BaseURL
3. 使用百炼的模型名称
4. 保持与 OpenAI 相同的调用方式

这种方案实现简单、维护成本低、功能完整，是最优的集成方案。

## 下一步行动

1. ✅ 完成 API 文档调研
2. ⏭️ 验证 Genkit OpenAI 插件是否支持自定义 BaseURL
3. ⏭️ 实现百炼插件集成
4. ⏭️ 编写集成测试
5. ⏭️ 编写使用文档
