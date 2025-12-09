// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bailian_test

import (
	"context"
	"fmt"
	"os"

	"genkit-ai-service/internal/genkit/plugins/bailian"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Example_basic 演示基本的模型调用
func Example_basic() {
	// 设置 API Key
	os.Setenv("BAILIAN_API_KEY", "your-api-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	// 初始化百炼插件
	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	// 获取模型
	model := plugin.Model(g, "qwen-turbo")

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
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content[0].Text)
}

// Example_streaming 演示流式响应
func Example_streaming() {
	os.Setenv("BAILIAN_API_KEY", "your-api-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	model := plugin.Model(g, "qwen-turbo")

	// 使用流式回调
	_, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("讲一个简短的故事"),
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

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println() // 换行
}

// Example_multiTurn 演示多轮对话
func Example_multiTurn() {
	os.Setenv("BAILIAN_API_KEY", "your-api-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	model := plugin.Model(g, "qwen-turbo")

	// 多轮对话
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

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content[0].Text)
	// Output: 你叫张三。
}

// Example_systemPrompt 演示使用系统提示
func Example_systemPrompt() {
	os.Setenv("BAILIAN_API_KEY", "your-api-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	model := plugin.Model(g, "qwen-turbo")

	// 使用系统提示
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleSystem,
				Content: []*ai.Part{
					ai.NewTextPart("你是一个专业的技术顾问，回答要简洁专业，不超过50字"),
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

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content[0].Text)
}

// Example_withConfig 演示使用配置参数
func Example_withConfig() {
	os.Setenv("BAILIAN_API_KEY", "your-api-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	model := plugin.Model(g, "qwen-turbo")

	// 使用配置参数
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("写一首关于春天的诗"),
				},
			},
		},
		Config: map[string]any{
			"temperature": 0.8, // 提高创造性
			"maxTokens":   200, // 限制输出长度
			"topP":        0.9, // 核采样参数
		},
	}, nil)

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content[0].Text)
}

// Example_multimodal 演示多模态模型使用
func Example_multimodal() {
	os.Setenv("BAILIAN_API_KEY", "your-api-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	// 使用支持多模态的模型
	model := plugin.Model(g, "qwen-vl-plus")

	// 发送图像和文本
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewMediaPart("image/jpeg", "https://example.com/image.jpg"),
					ai.NewTextPart("这张图片里有什么？"),
				},
			},
		},
	}, nil)

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content[0].Text)
}

// Example_errorHandling 演示错误处理
func Example_errorHandling() {
	os.Setenv("BAILIAN_API_KEY", "invalid-key")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	model := plugin.Model(g, "qwen-turbo")

	_, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("你好"),
				},
			},
		},
	}, nil)

	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		// 根据错误类型进行处理
		// - 检查是否是认证错误
		// - 检查是否是速率限制
		// - 检查是否是网络错误
		return
	}
}

// Example_customBaseURL 演示使用自定义 Base URL
func Example_customBaseURL() {
	// 设置自定义 Base URL（例如使用代理）
	os.Setenv("BAILIAN_API_KEY", "your-api-key")
	os.Setenv("BAILIAN_BASE_URL", "https://your-proxy.example.com/v1")

	ctx := context.Background()
	g := genkit.New(ctx, nil)

	plugin := &bailian.Bailian{}
	g.RegisterPlugin(plugin)

	model := plugin.Model(g, "qwen-turbo")

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("你好"),
				},
			},
		},
	}, nil)

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content[0].Text)
}
