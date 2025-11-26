package examples

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

// AzureOpenAIExample 演示如何使用 OpenAI 插件 + 自定义 BaseURL 集成 Azure OpenAI
func AzureOpenAIExample() {
	ctx := context.Background()

	// Azure OpenAI 配置
	azureEndpoint := "https://my-resource.openai.azure.com"
	azureDeployment := "gpt-4-deployment"
	azureAPIKey := "your-azure-api-key"
	azureAPIVersion := "2024-02-15-preview"

	// 构造 BaseURL
	// 方案 A：在 BaseURL 中包含 API Version 查询参数
	baseURL := fmt.Sprintf("%s/openai/deployments/%s?api-version=%s",
		azureEndpoint,
		azureDeployment,
		azureAPIVersion,
	)

	// 创建 OpenAI 插件，配置 Azure 的 BaseURL
	plugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey(azureAPIKey),
			option.WithBaseURL(baseURL),
		},
	}

	// 初始化 Genkit
	g := genkit.Init(ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel("openai/gpt-4"), // 使用 openai/ 前缀
	)

	// 非流式调用示例
	resp, err := genkit.Generate(ctx, g,
		ai.WithPrompt("你好，请介绍一下自己"),
	)
	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		return
	}

	fmt.Printf("响应: %s\n", resp.Text())
	if resp.Usage != nil {
		fmt.Printf("Token 使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.InputTokens,
			resp.Usage.OutputTokens,
			resp.Usage.TotalTokens,
		)
	}

	// 流式调用示例
	fmt.Println("\n流式调用示例:")
	_, err = genkit.Generate(ctx, g,
		ai.WithPrompt("请写一首关于春天的诗"),
		ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
			fmt.Print(chunk.Text())
			return nil
		}),
	)
	if err != nil {
		fmt.Printf("\n流式生成失败: %v\n", err)
		return
	}
	fmt.Println()
}

// AzureOpenAIExampleWithoutAPIVersion 演示不包含 API Version 的方案
// 注意：这个方案可能不工作，需要实际测试
func AzureOpenAIExampleWithoutAPIVersion() {
	ctx := context.Background()

	// Azure OpenAI 配置
	azureEndpoint := "https://my-resource.openai.azure.com"
	azureDeployment := "gpt-4-deployment"
	azureAPIKey := "your-azure-api-key"

	// 构造 BaseURL（不包含 API Version）
	baseURL := fmt.Sprintf("%s/openai/deployments/%s",
		azureEndpoint,
		azureDeployment,
	)

	// 创建 OpenAI 插件
	plugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey(azureAPIKey),
			option.WithBaseURL(baseURL),
			// 可能需要通过其他方式添加 api-version 参数
		},
	}

	// 初始化 Genkit
	g := genkit.Init(ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel("openai/gpt-4"),
	)

	// 测试调用
	resp, err := genkit.Generate(ctx, g,
		ai.WithPrompt("Hello, Azure OpenAI!"),
	)
	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		return
	}

	fmt.Printf("响应: %s\n", resp.Text())
}

// MultiProviderExample 演示如何在同一个应用中使用多个提供商
func MultiProviderExample() {
	ctx := context.Background()

	// Azure OpenAI 配置
	azureBaseURL := "https://my-resource.openai.azure.com/openai/deployments/gpt-4?api-version=2024-02-15-preview"
	azurePlugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey("azure-api-key"),
			option.WithBaseURL(azureBaseURL),
		},
	}

	// 标准 OpenAI 配置
	openaiPlugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey("openai-api-key"),
			// 不设置 BaseURL，使用默认的 OpenAI endpoint
		},
	}

	// 初始化两个独立的 Genkit 实例
	azureGenkit := genkit.Init(ctx,
		genkit.WithPlugins(azurePlugin),
		genkit.WithDefaultModel("openai/gpt-4"),
	)

	openaiGenkit := genkit.Init(ctx,
		genkit.WithPlugins(openaiPlugin),
		genkit.WithDefaultModel("openai/gpt-4"),
	)

	// 使用 Azure OpenAI
	fmt.Println("使用 Azure OpenAI:")
	azureResp, err := genkit.Generate(ctx, azureGenkit,
		ai.WithPrompt("你好"),
	)
	if err != nil {
		fmt.Printf("Azure 调用失败: %v\n", err)
	} else {
		fmt.Printf("Azure 响应: %s\n", azureResp.Text())
	}

	// 使用标准 OpenAI
	fmt.Println("\n使用标准 OpenAI:")
	openaiResp, err := genkit.Generate(ctx, openaiGenkit,
		ai.WithPrompt("Hello"),
	)
	if err != nil {
		fmt.Printf("OpenAI 调用失败: %v\n", err)
	} else {
		fmt.Printf("OpenAI 响应: %s\n", openaiResp.Text())
	}
}
