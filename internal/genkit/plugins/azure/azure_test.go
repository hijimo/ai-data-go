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

package azure

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

// TestAzureAI_Init 测试 AzureAI 插件的初始化
func TestAzureAI_Init(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *AzureAI
		wantPanic bool
		panicMsg  string
	}{
		{
			name: "成功初始化 - 提供所有必需参数",
			plugin: &AzureAI{
				APIKey:   "test-api-key",
				BaseURL:  "https://test.openai.azure.com",
				Provider: "azure",
			},
			wantPanic: false,
		},
		{
			name: "成功初始化 - 使用默认 API 版本",
			plugin: &AzureAI{
				APIKey:  "test-api-key",
				BaseURL: "https://test.openai.azure.com",
			},
			wantPanic: false,
		},
		{
			name: "成功初始化 - 使用自定义 API 版本",
			plugin: &AzureAI{
				APIKey:     "test-api-key",
				BaseURL:    "https://test.openai.azure.com",
				APIVersion: "2024-12-01-preview",
			},
			wantPanic: false,
		},
		{
			name: "失败 - 缺少 API Key",
			plugin: &AzureAI{
				BaseURL: "https://test.openai.azure.com",
			},
			wantPanic: true,
			panicMsg:  "azure: APIKey is required",
		},
		{
			name: "失败 - 缺少 Base URL",
			plugin: &AzureAI{
				APIKey: "test-api-key",
			},
			wantPanic: true,
			panicMsg:  "azure: BaseURL is required",
		},
		{
			name: "失败 - 重复初始化",
			plugin: &AzureAI{
				APIKey:  "test-api-key",
				BaseURL: "https://test.openai.azure.com",
				initted: true, // 已经初始化
			},
			wantPanic: true,
			panicMsg:  "azure.Init already called",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Init() 应该 panic，但没有")
					} else if msg, ok := r.(string); ok && msg != tt.panicMsg {
						t.Errorf("Init() panic 消息 = %v, 期望 %v", msg, tt.panicMsg)
					}
				}()
			}

			ctx := context.Background()
			actions := tt.plugin.Init(ctx)

			if !tt.wantPanic {
				// 验证初始化成功
				if !tt.plugin.initted {
					t.Error("Init() 后 initted 应该为 true")
				}

				// 验证 HTTP 客户端已创建
				if tt.plugin.httpClient == nil {
					t.Error("Init() 后 httpClient 应该不为 nil")
				}

				// 验证默认 API 版本
				if tt.plugin.APIVersion == "" {
					t.Error("Init() 后 APIVersion 应该不为空")
				}
				if tt.plugin.APIVersion != DefaultAPIVersion && tt.plugin.APIVersion != "2024-12-01-preview" {
					t.Errorf("Init() APIVersion = %v, 期望 %v 或自定义版本", tt.plugin.APIVersion, DefaultAPIVersion)
				}

				// 验证默认 Provider
				if tt.plugin.Provider == "" {
					t.Error("Init() 后 Provider 应该不为空")
				}

				// 验证返回的 actions
				if actions == nil {
					t.Error("Init() 应该返回非 nil 的 actions 切片")
				}
			}
		})
	}
}

// TestAzureAI_Name 测试 Name() 方法
func TestAzureAI_Name(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{
			name:     "默认 provider",
			provider: "",
			want:     "azure",
		},
		{
			name:     "自定义 provider",
			provider: "my-azure",
			want:     "my-azure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &AzureAI{
				APIKey:   "test-api-key",
				BaseURL:  "https://test.openai.azure.com",
				Provider: tt.provider,
			}

			ctx := context.Background()
			plugin.Init(ctx)

			if got := plugin.Name(); got != tt.want {
				t.Errorf("Name() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

// TestAzureAI_ThreadSafety 测试线程安全的初始化检查
func TestAzureAI_ThreadSafety(t *testing.T) {
	plugin := &AzureAI{
		APIKey:  "test-api-key",
		BaseURL: "https://test.openai.azure.com",
	}

	ctx := context.Background()

	// 第一次初始化应该成功
	plugin.Init(ctx)

	// 第二次初始化应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("第二次调用 Init() 应该 panic")
		}
	}()

	plugin.Init(ctx)
}

// TestAzureAI_DefineModel_NotInitialized 测试未初始化时调用 DefineModel
func TestAzureAI_DefineModel_NotInitialized(t *testing.T) {
	plugin := &AzureAI{
		APIKey:  "test-api-key",
		BaseURL: "https://test.openai.azure.com",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("未初始化时调用 DefineModel() 应该 panic")
		} else if msg, ok := r.(string); ok && msg != "AzureAI.Init not called" {
			t.Errorf("DefineModel() panic 消息 = %v, 期望 'AzureAI.Init not called'", msg)
		}
	}()

	plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{Supports: &BasicText})
}

// TestAzureAI_DefineEmbedder_NotInitialized 测试未初始化时调用 DefineEmbedder
func TestAzureAI_DefineEmbedder_NotInitialized(t *testing.T) {
	plugin := &AzureAI{
		APIKey:  "test-api-key",
		BaseURL: "https://test.openai.azure.com",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("未初始化时调用 DefineEmbedder() 应该 panic")
		} else if msg, ok := r.(string); ok && msg != "AzureAI.Init not called" {
			t.Errorf("DefineEmbedder() panic 消息 = %v, 期望 'AzureAI.Init not called'", msg)
		}
	}()

	plugin.DefineEmbedder("azure", "text-embedding-ada-002", nil)
}

// TestAzureAI_HTTPClientConfiguration 测试 HTTP 客户端配置
func TestAzureAI_HTTPClientConfiguration(t *testing.T) {
	plugin := &AzureAI{
		APIKey:  "test-api-key",
		BaseURL: "https://test.openai.azure.com",
	}

	ctx := context.Background()
	plugin.Init(ctx)

	// 验证 HTTP 客户端配置
	if plugin.httpClient == nil {
		t.Fatal("httpClient 不应该为 nil")
	}

	// 验证超时设置
	if plugin.httpClient.Timeout == 0 {
		t.Error("httpClient.Timeout 应该被设置")
	}

	// 验证 Transport 配置
	if plugin.httpClient.Transport == nil {
		t.Error("httpClient.Transport 不应该为 nil")
	}
}
