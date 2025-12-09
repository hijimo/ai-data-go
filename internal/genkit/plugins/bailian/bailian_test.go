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

package bailian

import (
	"context"
	"os"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBailianPlugin_Name(t *testing.T) {
	plugin := &Bailian{}
	assert.Equal(t, "bailian", plugin.Name())
}

func TestBailianPlugin_Init(t *testing.T) {
	ctx := context.Background()

	// 创建插件并传入配置
	plugin := &Bailian{
		Opts: []option.RequestOption{
			option.WithAPIKey("test-api-key"),
			option.WithBaseURL("https://test.example.com/v1"),
		},
	}

	actions := plugin.Init(ctx)

	// 验证返回的 actions 包含所有支持的模型
	assert.NotEmpty(t, actions)
	assert.GreaterOrEqual(t, len(actions), len(supportedModels))
}

func TestBailianPlugin_DefineModel(t *testing.T) {
	ctx := context.Background()

	// 创建插件并传入配置
	plugin := &Bailian{
		Opts: []option.RequestOption{
			option.WithAPIKey("test-api-key"),
			option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		},
	}

	plugin.Init(ctx)

	// 测试定义模型
	opts := ai.ModelOptions{
		Label: "Test Model",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      true,
			SystemRole: true,
			Media:      false,
		},
	}

	model := plugin.DefineModel("qwen-turbo", opts)
	assert.NotNil(t, model)
}

func TestBailianPlugin_Model(t *testing.T) {
	ctx := context.Background()

	// 创建插件并传入配置
	plugin := &Bailian{
		Opts: []option.RequestOption{
			option.WithAPIKey("test-api-key"),
			option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		},
	}

	plugin.Init(ctx)

	// 先定义模型
	opts := supportedModels["qwen-turbo"]
	model := plugin.DefineModel("qwen-turbo", opts)
	assert.NotNil(t, model)
}

func TestSupportedModels(t *testing.T) {
	// 验证所有支持的模型都有正确的配置
	for modelID, opts := range supportedModels {
		t.Run(modelID, func(t *testing.T) {
			assert.NotEmpty(t, opts.Label, "模型标签不能为空")
			assert.NotNil(t, opts.Supports, "模型支持配置不能为空")
			assert.NotEmpty(t, opts.Versions, "模型版本列表不能为空")

			// 验证基本能力
			assert.True(t, opts.Supports.Multiturn, "所有模型都应支持多轮对话")
			assert.True(t, opts.Supports.SystemRole, "所有模型都应支持系统角色")
		})
	}
}

func TestBailianPlugin_Integration(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	apiKey := os.Getenv("BAILIAN_API_KEY")
	if apiKey == "" {
		t.Skip("跳过集成测试：未设置 BAILIAN_API_KEY 环境变量")
	}

	ctx := context.Background()

	// 创建插件并传入配置
	plugin := &Bailian{
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		},
	}

	// 初始化插件
	actions := plugin.Init(ctx)
	require.NotEmpty(t, actions)

	// 注意：实际的集成测试需要完整的 Genkit 环境
	// 这里只测试插件的初始化和模型定义
	opts := supportedModels["qwen-turbo"]
	model := plugin.DefineModel("qwen-turbo", opts)
	require.NotNil(t, model)
}

func TestBailianPlugin_StreamingIntegration(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	apiKey := os.Getenv("BAILIAN_API_KEY")
	if apiKey == "" {
		t.Skip("跳过集成测试：未设置 BAILIAN_API_KEY 环境变量")
	}

	ctx := context.Background()

	// 创建插件并传入配置
	plugin := &Bailian{
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		},
	}

	// 初始化插件
	plugin.Init(ctx)

	// 注意：实际的流式测试需要完整的 Genkit 环境和真实的 API 调用
	// 这里只测试插件的初始化
	opts := supportedModels["qwen-turbo"]
	model := plugin.DefineModel("qwen-turbo", opts)
	require.NotNil(t, model)
}
