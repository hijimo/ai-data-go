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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

const (
	provider = "bailian"
	// 默认的百炼 API 端点（兼容模式）
	defaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
)

// 支持的模型列表
// 参考阿里云百炼平台文档：
// - 模型列表：https://help.aliyun.com/zh/model-studio/getting-started/models
// - API 文档：https://help.aliyun.com/zh/model-studio/developer-reference/api-details
// - OpenAI 兼容模式：https://help.aliyun.com/zh/model-studio/developer-reference/compatibility-of-openai-with-dashscope
var supportedModels = map[string]ai.ModelOptions{
	"qwen-turbo": {
		Label: "通义千问 Turbo",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      true,
			SystemRole: true,
			Media:      false,
		},
		Versions: []string{"qwen-turbo"},
	},
	"qwen-plus": {
		Label: "通义千问 Plus",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      true,
			SystemRole: true,
			Media:      false,
		},
		Versions: []string{"qwen-plus"},
	},
	"qwen-max": {
		Label: "通义千问 Max",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      true,
			SystemRole: true,
			Media:      false,
		},
		Versions: []string{"qwen-max"},
	},
	"qwen3-max": {
		Label: "通义千问 3 Max",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      true,
			SystemRole: true,
			Media:      false,
		},
		Versions: []string{"qwen3-max"},
	},
	"qwen-vl-plus": {
		Label: "通义千问 VL Plus",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      false,
			SystemRole: true,
			Media:      true,
		},
		Versions: []string{"qwen-vl-plus"},
	},
	"qwen-vl-max": {
		Label: "通义千问 VL Max",
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			Tools:      false,
			SystemRole: true,
			Media:      true,
		},
		Versions: []string{"qwen-vl-max"},
	},
}

// Bailian 百炼插件结构
// 注意：API Key 和 Base URL 必须通过 Opts 字段传入，不从环境变量读取
type Bailian struct {
	// Opts 请求选项，必须包含：
	// - option.WithAPIKey(apiKey): API 密钥（会自动设置为 Bearer token）
	// - option.WithBaseURL(baseURL): API 端点 URL
	Opts             []option.RequestOption
	openAICompatible compat_oai.OpenAICompatible
	// EnableDebugLog 是否启用调试日志（打印 HTTP 请求详情）
	EnableDebugLog bool
}

// Name 实现 genkit.Plugin 接口
func (b *Bailian) Name() string {
	return provider
}

// Init 初始化插件
// 注意：调用此方法前，必须先设置 b.Opts，包含 API Key 和 Base URL
func (b *Bailian) Init(ctx context.Context) []api.Action {
	// 如果启用调试日志，添加 HTTP 请求拦截中间件
	if b.EnableDebugLog {
		b.Opts = append(b.Opts, option.WithMiddleware(b.loggingMiddleware))
	}

	// 初始化 OpenAICompatible
	// OpenAI SDK 会自动将 API Key 设置为 "Authorization: Bearer {apiKey}" header
	b.openAICompatible.Opts = b.Opts
	compatActions := b.openAICompatible.Init(ctx)

	var actions []api.Action
	actions = append(actions, compatActions...)

	// 定义默认模型
	for model, opts := range supportedModels {
		actions = append(actions, b.DefineModel(model, opts).(api.Action))
	}

	return actions
}

// loggingMiddleware HTTP 请求日志中间件
// 在发送请求前打印完整的 HTTP 请求信息（包括 headers 和 body）
func (b *Bailian) loggingMiddleware(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	// 读取请求体
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		// 恢复请求体，以便后续使用
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// 构建 curl 命令
	curlCmd := buildCurlCommand(req, bodyBytes)

	// 打印请求详情
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("百炼 API 请求详情")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("方法: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	fmt.Println("\n请求头:")
	for key, values := range req.Header {
		for _, value := range values {
			// 脱敏 Authorization header
			if key == "Authorization" && strings.HasPrefix(value, "Bearer ") {
				token := value[7:]
				if len(token) > 20 {
					value = "Bearer " + token[:10] + "****" + token[len(token)-6:]
				} else {
					value = "Bearer ****"
				}
			}
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if len(bodyBytes) > 0 {
		fmt.Println("\n请求体:")
		// 格式化 JSON（如果是 JSON）
		if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
			fmt.Println(formatJSON(string(bodyBytes)))
		} else {
			fmt.Println(string(bodyBytes))
		}
	}

	fmt.Println("\n等效的 curl 命令:")
	fmt.Println(curlCmd)
	fmt.Println(strings.Repeat("=", 80) + "\n")

	// 调用下一个中间件或实际的请求
	return next(req)
}

// buildCurlCommand 构建等效的 curl 命令
func buildCurlCommand(req *http.Request, bodyBytes []byte) string {
	var cmd strings.Builder

	cmd.WriteString(fmt.Sprintf("curl -X %s '%s'", req.Method, req.URL.String()))

	// 添加 headers
	for key, values := range req.Header {
		for _, value := range values {
			// 脱敏 Authorization header
			if key == "Authorization" && strings.HasPrefix(value, "Bearer ") {
				token := value[7:]
				if len(token) > 20 {
					value = "Bearer " + token[:10] + "****" + token[len(token)-6:]
				} else {
					value = "Bearer ****"
				}
			}
			cmd.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", key, value))
		}
	}

	// 添加 body
	if len(bodyBytes) > 0 {
		// 转义单引号
		body := strings.ReplaceAll(string(bodyBytes), "'", "'\\''")
		cmd.WriteString(fmt.Sprintf(" \\\n  -d '%s'", body))
	}

	return cmd.String()
}

// formatJSON 简单的 JSON 格式化（用于打印）
func formatJSON(jsonStr string) string {
	// 这里使用简单的缩进，实际项目中可以使用 json.Indent
	var result strings.Builder
	indent := 0
	inString := false
	escape := false

	for i, char := range jsonStr {
		if escape {
			result.WriteRune(char)
			escape = false
			continue
		}

		switch char {
		case '\\':
			result.WriteRune(char)
			escape = true
		case '"':
			result.WriteRune(char)
			inString = !inString
		case '{', '[':
			result.WriteRune(char)
			if !inString {
				indent++
				if i+1 < len(jsonStr) && jsonStr[i+1] != '}' && jsonStr[i+1] != ']' {
					result.WriteString("\n" + strings.Repeat("  ", indent))
				}
			}
		case '}', ']':
			if !inString {
				indent--
				if i > 0 && jsonStr[i-1] != '{' && jsonStr[i-1] != '[' {
					result.WriteString("\n" + strings.Repeat("  ", indent))
				}
			}
			result.WriteRune(char)
		case ',':
			result.WriteRune(char)
			if !inString {
				result.WriteString("\n" + strings.Repeat("  ", indent))
			}
		case ':':
			result.WriteRune(char)
			if !inString {
				result.WriteString(" ")
			}
		default:
			result.WriteRune(char)
		}
	}

	return result.String()
}

// Model 获取指定名称的模型
func (b *Bailian) Model(g *genkit.Genkit, id string) ai.Model {
	return b.openAICompatible.Model(g, api.NewName(provider, id))
}

// DefineModel 定义模型
func (b *Bailian) DefineModel(id string, opts ai.ModelOptions) ai.Model {
	return b.openAICompatible.DefineModel(provider, id, opts)
}
