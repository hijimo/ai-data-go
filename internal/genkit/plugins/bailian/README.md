# Bailian Plugin for Genkit

阿里云百炼（Bailian）平台的 Genkit 插件实现。

## 功能特性

- 支持阿里云百炼平台的通义千问系列模型
- 兼容 OpenAI API 格式
- 支持流式和非流式响应
- 支持多轮对话
- 支持工具调用（部分模型）
- 支持多模态输入（VL 系列模型）

## 支持的模型

| 模型 ID | 模型名称 | 多轮对话 | 工具调用 | 系统角色 | 多模态 |
|---------|---------|---------|---------|---------|--------|
| qwen-turbo | 通义千问 Turbo | ✓ | ✓ | ✓ | ✗ |
| qwen-plus | 通义千问 Plus | ✓ | ✓ | ✓ | ✗ |
| qwen-max | 通义千问 Max | ✓ | ✓ | ✓ | ✗ |
| qwen3-max | 通义千问 3 Max | ✓ | ✓ | ✓ | ✗ |
| qwen-vl-plus | 通义千问 VL Plus | ✓ | ✗ | ✓ | ✓ |
| qwen-vl-max | 通义千问 VL Max | ✓ | ✗ | ✓ | ✓ |

## 配置方式

Bailian 插件不使用环境变量，所有配置通过代码传入：

```go
// 创建插件时传入配置
plugin := &bailian.Bailian{
    Opts: []option.RequestOption{
        option.WithAPIKey("your-api-key"),
        option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
    },
}
```

**注意**：
- API Key 会自动设置为 `Authorization: Bearer {apiKey}` header
- Base URL 默认为 `https://dashscope.aliyuncs.com/compatible-mode/v1`

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/openai/openai-go/option"
    "genkit-ai-service/internal/genkit/plugins/bailian"
)

func main() {
    ctx := context.Background()
    
    // 创建 Genkit 实例
    g := genkit.New(ctx, nil)
    
    // 创建百炼插件（传入 API Key 和 Base URL）
    bailianPlugin := &bailian.Bailian{
        Opts: []option.RequestOption{
            option.WithAPIKey("your-api-key"),
            option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
        },
    }
    g.RegisterPlugin(bailianPlugin)
    
    // 获取模型
    model := bailianPlugin.Model(g, "qwen-turbo")
    
    // 生成响应
    resp, err := model.Generate(ctx, &ai.ModelRequest{
        Messages: []*ai.Message{
            {
                Role: ai.RoleUser,
                Content: []*ai.Part{
                    ai.NewTextPart("你好，请介绍一下自己"),
                },
            },
        },
    }, nil)
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

### 流式响应

```go
// 创建插件
plugin := &bailian.Bailian{
    Opts: []option.RequestOption{
        option.WithAPIKey("your-api-key"),
        option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
    },
}

// 使用流式回调
err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("讲一个故事"),
            },
        },
    },
}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    // 处理每个流式响应块
    if len(chunk.Content) > 0 {
        fmt.Print(chunk.Content[0].Text)
    }
    return nil
})
```

### 多轮对话

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("我叫张三"),
            },
        },
        {
            Role: ai.RoleModel,
            Content: []*ai.Part{
                ai.NewTextPart("你好张三，很高兴认识你！"),
            },
        },
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("我叫什么名字？"),
            },
        },
    },
}, nil)
```

### 使用系统提示

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleSystem,
            Content: []*ai.Part{
                ai.NewTextPart("你是一个专业的技术顾问，回答要简洁专业"),
            },
        },
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("什么是微服务架构？"),
            },
        },
    },
}, nil)
```

## API 兼容性

百炼插件使用 OpenAI 兼容模式，支持以下 API 端点：

- `POST /chat/completions` - 对话补全
- `GET /models` - 列出可用模型

## 注意事项

1. **API Key 安全**：请妥善保管您的 API Key，不要将其提交到版本控制系统
2. **Base URL**：百炼使用兼容模式的 Base URL，与标准 OpenAI API 不同
3. **模型限制**：不同模型支持的功能不同，请参考上面的支持列表
4. **速率限制**：请遵守阿里云百炼平台的 API 调用限制
5. **多模态模型**：VL 系列模型支持图像输入，但不支持工具调用

## 错误处理

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    // 处理错误
    switch {
    case strings.Contains(err.Error(), "unauthorized"):
        // API Key 无效
    case strings.Contains(err.Error(), "rate limit"):
        // 超过速率限制
    case strings.Contains(err.Error(), "timeout"):
        // 请求超时
    default:
        // 其他错误
    }
}
```

## 相关链接

- [阿里云百炼平台控制台](https://bailian.console.aliyun.com/)
- [百炼模型列表](https://help.aliyun.com/zh/model-studio/getting-started/models)
- [百炼 API 文档](https://help.aliyun.com/zh/model-studio/developer-reference/api-details)
- [OpenAI 兼容模式文档](https://help.aliyun.com/zh/model-studio/developer-reference/compatibility-of-openai-with-dashscope)
- [Genkit 文档](https://firebase.google.com/docs/genkit)

## 许可证

Apache License 2.0
