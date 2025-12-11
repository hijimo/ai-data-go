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

//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"genkit-ai-service/internal/genkit/plugins/azure"

	"github.com/firebase/genkit/go/ai"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取配置
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	baseURL := os.Getenv("AZURE_OPENAI_BASE_URL")

	if apiKey == "" || baseURL == "" {
		log.Fatal("请设置 AZURE_OPENAI_API_KEY 和 AZURE_OPENAI_BASE_URL 环境变量")
	}

	// 示例 1: 单个工具调用
	example1_SingleToolCall(ctx, apiKey, baseURL)

	// 示例 2: 多个工具调用
	example2_MultipleToolCalls(ctx, apiKey, baseURL)

	// 示例 3: 工具调用链（多轮对话）
	example3_ToolCallChain(ctx, apiKey, baseURL)

	// 示例 4: 强制工具调用
	example4_ForceToolCall(ctx, apiKey, baseURL)
}

// 示例 1: 单个工具调用
func example1_SingleToolCall(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 1: 单个工具调用 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 定义天气查询工具
	tools := []*ai.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的当前天气信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{
						"type":        "string",
						"description": "城市名称，例如：北京、上海",
					},
					"unit": map[string]any{
						"type":        "string",
						"enum":        []string{"celsius", "fahrenheit"},
						"description": "温度单位",
					},
				},
				"required": []string{"city"},
			},
		},
	}

	// 第一轮：模型决定调用工具
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("北京今天天气怎么样？"),
				},
			},
		},
		Tools: tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	// 检查是否有工具调用
	var toolCall *ai.ToolRequest
	for _, part := range resp.Message.Content {
		if part.IsToolRequest() {
			toolCall = part.ToolRequest
			break
		}
	}

	if toolCall == nil {
		fmt.Println("模型没有调用工具")
		return
	}

	fmt.Printf("工具调用: %s\n", toolCall.Name)
	fmt.Printf("参数: %v\n", toolCall.Input)

	// 模拟执行工具
	weatherResult := executeGetWeather(toolCall.Input)
	fmt.Printf("工具结果: %s\n", weatherResult)

	// 第二轮：将工具结果返回给模型
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("北京今天天气怎么样？"),
			},
		},
		resp.Message, // 包含工具调用的助手消息
		{
			Role: ai.RoleTool,
			Content: []*ai.Part{
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolCall.Name,
					Ref:    toolCall.Ref,
					Output: weatherResult,
				}),
			},
		},
	}

	finalResp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
		Tools:    tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("最终响应: %s\n\n", finalResp.Message.Content[0].Text)
}

// 示例 2: 多个工具调用
func example2_MultipleToolCalls(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 2: 多个工具调用 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 定义多个工具
	tools := []*ai.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的天气信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string"},
				},
				"required": []string{"city"},
			},
		},
		{
			Name:        "get_time",
			Description: "获取指定城市的当前时间",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string"},
				},
				"required": []string{"city"},
			},
		},
		{
			Name:        "calculate",
			Description: "执行数学计算",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{
						"type":        "string",
						"description": "数学表达式，例如：2+2",
					},
				},
				"required": []string{"expression"},
			},
		},
	}

	// 询问需要多个工具的问题
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("请告诉我北京的天气和当前时间。"),
				},
			},
		},
		Tools: tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	// 收集所有工具调用
	var toolCalls []*ai.ToolRequest
	for _, part := range resp.Message.Content {
		if part.IsToolRequest() {
			toolCalls = append(toolCalls, part.ToolRequest)
		}
	}

	fmt.Printf("模型调用了 %d 个工具:\n", len(toolCalls))
	for i, tc := range toolCalls {
		fmt.Printf("%d. %s(%v)\n", i+1, tc.Name, tc.Input)
	}

	// 执行所有工具并构建响应消息
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("请告诉我北京的天气和当前时间。"),
			},
		},
		resp.Message,
	}

	// 为每个工具调用添加响应
	for _, tc := range toolCalls {
		var result any
		switch tc.Name {
		case "get_weather":
			result = executeGetWeather(tc.Input)
		case "get_time":
			result = executeGetTime(tc.Input)
		}

		messages = append(messages, &ai.Message{
			Role: ai.RoleTool,
			Content: []*ai.Part{
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   tc.Name,
					Ref:    tc.Ref,
					Output: result,
				}),
			},
		})
	}

	// 获取最终响应
	finalResp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
		Tools:    tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("\n最终响应: %s\n\n", finalResp.Message.Content[0].Text)
}

// 示例 3: 工具调用链（多轮对话）
func example3_ToolCallChain(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 3: 工具调用链（多轮对话）===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	tools := []*ai.ToolDefinition{
		{
			Name:        "search_flights",
			Description: "搜索航班信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{"type": "string"},
					"to":   map[string]any{"type": "string"},
					"date": map[string]any{"type": "string"},
				},
				"required": []string{"from", "to", "date"},
			},
		},
		{
			Name:        "book_flight",
			Description: "预订航班",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"flight_id": map[string]any{"type": "string"},
				},
				"required": []string{"flight_id"},
			},
		},
	}

	// 初始消息
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("我想预订明天从北京到上海的航班。"),
			},
		},
	}

	// 第一轮：搜索航班
	fmt.Println("第一轮：搜索航班")
	resp1, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
		Tools:    tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	// 处理工具调用
	for _, part := range resp1.Message.Content {
		if part.IsToolRequest() {
			fmt.Printf("工具调用: %s(%v)\n", part.ToolRequest.Name, part.ToolRequest.Input)

			// 模拟搜索航班
			flightResult := map[string]any{
				"flights": []map[string]any{
					{"id": "CA1234", "time": "08:00", "price": 800},
					{"id": "MU5678", "time": "10:00", "price": 750},
				},
			}

			messages = append(messages, resp1.Message)
			messages = append(messages, &ai.Message{
				Role: ai.RoleTool,
				Content: []*ai.Part{
					ai.NewToolResponsePart(&ai.ToolResponse{
						Name:   part.ToolRequest.Name,
						Ref:    part.ToolRequest.Ref,
						Output: flightResult,
					}),
				},
			})
		}
	}

	// 第二轮：模型展示航班选项
	fmt.Println("\n第二轮：展示航班选项")
	resp2, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
		Tools:    tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("模型响应: %s\n", resp2.Message.Content[0].Text)

	// 用户选择航班
	messages = append(messages, resp2.Message)
	messages = append(messages, &ai.Message{
		Role: ai.RoleUser,
		Content: []*ai.Part{
			ai.NewTextPart("我选择 MU5678 航班。"),
		},
	})

	// 第三轮：预订航班
	fmt.Println("\n第三轮：预订航班")
	resp3, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
		Tools:    tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	// 处理预订工具调用
	for _, part := range resp3.Message.Content {
		if part.IsToolRequest() {
			fmt.Printf("工具调用: %s(%v)\n", part.ToolRequest.Name, part.ToolRequest.Input)

			bookingResult := map[string]any{
				"status":         "confirmed",
				"booking_id":     "BK123456",
				"flight_id":      "MU5678",
				"passenger":      "张三",
				"total_price":    750,
				"payment_status": "paid",
			}

			messages = append(messages, resp3.Message)
			messages = append(messages, &ai.Message{
				Role: ai.RoleTool,
				Content: []*ai.Part{
					ai.NewToolResponsePart(&ai.ToolResponse{
						Name:   part.ToolRequest.Name,
						Ref:    part.ToolRequest.Ref,
						Output: bookingResult,
					}),
				},
			})
		}
	}

	// 第四轮：确认预订
	fmt.Println("\n第四轮：确认预订")
	resp4, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
		Tools:    tools,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("最终响应: %s\n\n", resp4.Message.Content[0].Text)
}

// 示例 4: 强制工具调用
func example4_ForceToolCall(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 4: 强制工具调用 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	tools := []*ai.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取天气信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string"},
				},
				"required": []string{"city"},
			},
		},
	}

	// 使用 tool_choice 强制调用特定工具
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("你好"),
				},
			},
		},
		Tools: tools,
		Config: map[string]any{
			"tool_choice": map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": "get_weather",
				},
			},
		},
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	// 检查工具调用
	for _, part := range resp.Message.Content {
		if part.IsToolRequest() {
			fmt.Printf("强制调用工具: %s\n", part.ToolRequest.Name)
			fmt.Printf("参数: %v\n", part.ToolRequest.Input)
		}
	}

	fmt.Println()
}

// 模拟工具执行函数

func executeGetWeather(input any) any {
	inputMap, _ := input.(map[string]any)
	city := inputMap["city"].(string)

	return map[string]any{
		"city":        city,
		"temperature": 22,
		"condition":   "晴朗",
		"humidity":    60,
		"wind_speed":  "5 km/h",
	}
}

func executeGetTime(input any) any {
	inputMap, _ := input.(map[string]any)
	city := inputMap["city"].(string)

	return map[string]any{
		"city":     city,
		"time":     "14:30:00",
		"timezone": "Asia/Shanghai",
		"date":     "2025-01-15",
	}
}
