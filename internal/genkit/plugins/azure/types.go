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
	"encoding/json"
	"fmt"
)

// DefaultAPIVersion 是 Azure OpenAI API 的默认版本
const DefaultAPIVersion = "2025-04-01-preview"

// ResponsesRequest 表示 Azure OpenAI Responses API 的请求格式
// 注意：使用 input 字段而非 messages 字段（符合 Responses API 规范）
type ResponsesRequest struct {
	// Model 要使用的模型名称
	Model string `json:"model"`

	// Input 消息数组（注意：使用 input 而非 messages）
	Input []Message `json:"input"`

	// Stream 是否使用流式响应
	Stream bool `json:"stream,omitempty"`

	// Temperature 采样温度（0-2）
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens 生成的最大 token 数
	MaxTokens *int `json:"max_tokens,omitempty"`

	// TopP 核采样参数
	TopP *float64 `json:"top_p,omitempty"`

	// Tools 可用的工具列表
	Tools []Tool `json:"tools,omitempty"`

	// ToolChoice 工具选择策略
	ToolChoice any `json:"tool_choice,omitempty"`

	// FrequencyPenalty 频率惩罚（-2.0 到 2.0）
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty 存在惩罚（-2.0 到 2.0）
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// Stop 停止序列
	Stop []string `json:"stop,omitempty"`

	// User 用户标识符
	User string `json:"user,omitempty"`
}

// Message 表示对话消息
type Message struct {
	// Role 消息角色：system, user, assistant, tool
	Role string `json:"role"`

	// Content 消息内容，可以是字符串或 ContentPart 数组
	Content any `json:"content,omitempty"`

	// Name 消息发送者的名称（可选）
	Name string `json:"name,omitempty"`

	// ToolCalls 工具调用列表（仅用于 assistant 角色）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID 工具调用 ID（仅用于 tool 角色）
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ContentPart 表示消息内容的一部分
type ContentPart struct {
	// Type 内容类型：text 或 image_url
	Type string `json:"type"`

	// Text 文本内容（当 type 为 text 时）
	Text string `json:"text,omitempty"`

	// ImageURL 图像 URL（当 type 为 image_url 时）
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL 表示图像 URL
type ImageURL struct {
	// URL 图像的 URL（支持 http/https URL 或 base64 编码）
	URL string `json:"url"`

	// Detail 图像细节级别：low, high, auto
	Detail string `json:"detail,omitempty"`
}

// Tool 表示可用的工具
type Tool struct {
	// Type 工具类型，目前仅支持 "function"
	Type string `json:"type"`

	// Function 函数定义
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 表示函数定义
type FunctionDefinition struct {
	// Name 函数名称
	Name string `json:"name"`

	// Description 函数描述
	Description string `json:"description,omitempty"`

	// Parameters 函数参数的 JSON Schema
	Parameters map[string]any `json:"parameters,omitempty"`

	// Strict 是否使用严格模式
	Strict bool `json:"strict,omitempty"`
}

// ToolCall 表示工具调用
type ToolCall struct {
	// ID 工具调用的唯一标识符
	ID string `json:"id"`

	// Type 工具类型，目前仅支持 "function"
	Type string `json:"type"`

	// Function 函数调用信息
	Function FunctionCall `json:"function"`
}

// FunctionCall 表示函数调用
type FunctionCall struct {
	// Name 函数名称
	Name string `json:"name"`

	// Arguments 函数参数（JSON 字符串）
	Arguments string `json:"arguments"`
}

// ResponsesResponse 表示 Azure OpenAI Responses API 的响应格式
type ResponsesResponse struct {
	// ID 响应的唯一标识符
	ID string `json:"id"`

	// Object 对象类型，通常为 "chat.completion"
	Object string `json:"object"`

	// Created 创建时间戳
	Created int64 `json:"created"`

	// Model 使用的模型名称
	Model string `json:"model"`

	// Choices 响应选项列表
	Choices []Choice `json:"choices"`

	// Usage token 使用统计
	Usage Usage `json:"usage"`

	// SystemFingerprint 系统指纹
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// Choice 表示响应选项
type Choice struct {
	// Index 选项索引
	Index int `json:"index"`

	// Message 响应消息
	Message Message `json:"message"`

	// FinishReason 完成原因：stop, length, content_filter, tool_calls
	FinishReason string `json:"finish_reason"`

	// Logprobs 对数概率信息
	Logprobs any `json:"logprobs,omitempty"`
}

// Usage 表示 token 使用统计
type Usage struct {
	// PromptTokens 提示 token 数
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens 完成 token 数
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens 总 token 数
	TotalTokens int `json:"total_tokens"`

	// PromptTokensDetails 提示 token 详细信息
	PromptTokensDetails *TokenDetails `json:"prompt_tokens_details,omitempty"`

	// CompletionTokensDetails 完成 token 详细信息
	CompletionTokensDetails *TokenDetails `json:"completion_tokens_details,omitempty"`
}

// TokenDetails 表示 token 详细信息
type TokenDetails struct {
	// CachedTokens 缓存的 token 数
	CachedTokens int `json:"cached_tokens,omitempty"`

	// ReasoningTokens 推理 token 数
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`

	// AudioTokens 音频 token 数
	AudioTokens int `json:"audio_tokens,omitempty"`

	// AcceptedPredictionTokens 接受的预测 token 数
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`

	// RejectedPredictionTokens 拒绝的预测 token 数
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// StreamChunk 表示流式响应的数据块（传统 OpenAI 格式，已废弃）
type StreamChunk struct {
	// ID 响应的唯一标识符
	ID string `json:"id"`

	// Object 对象类型，通常为 "chat.completion.chunk"
	Object string `json:"object"`

	// Created 创建时间戳
	Created int64 `json:"created"`

	// Model 使用的模型名称
	Model string `json:"model"`

	// Choices 响应选项列表
	Choices []StreamChoice `json:"choices"`

	// SystemFingerprint 系统指纹
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// StreamChoice 表示流式响应的选项
type StreamChoice struct {
	// Index 选项索引
	Index int `json:"index"`

	// Delta 增量消息
	Delta DeltaMessage `json:"delta"`

	// FinishReason 完成原因
	FinishReason string `json:"finish_reason,omitempty"`
}

// DeltaMessage 表示流式响应的增量消息
type DeltaMessage struct {
	// Role 消息角色
	Role string `json:"role,omitempty"`

	// Content 消息内容
	Content string `json:"content,omitempty"`

	// ToolCalls 工具调用列表
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ResponsesStreamEvent 表示 Azure Responses API 的 SSE 事件
// 支持的事件类型：
// - response.created
// - response.output_item.added
// - response.content_part.added
// - response.content_part.delta
// - response.content_part.done
// - response.output_item.done
// - response.done
type ResponsesStreamEvent struct {
	// Type 事件类型
	Type string `json:"type"`

	// SequenceNumber 序列号
	SequenceNumber int `json:"sequence_number,omitempty"`

	// ItemID 项目 ID
	ItemID string `json:"item_id,omitempty"`

	// OutputIndex 输出索引
	OutputIndex int `json:"output_index,omitempty"`

	// ContentIndex 内容索引
	ContentIndex int `json:"content_index,omitempty"`

	// Part 内容部分
	Part *ResponsePart `json:"part,omitempty"`

	// Delta 增量内容（可能是字符串或结构体）
	Delta json.RawMessage `json:"delta,omitempty"`

	// Response 完整响应（仅在 response.done 事件中）
	Response *ResponsesStreamResponse `json:"response,omitempty"`
}

// ResponsePart 表示响应内容部分
type ResponsePart struct {
	// Type 内容类型：output_text, function_call 等
	Type string `json:"type"`

	// Text 文本内容（当 type 为 output_text 时）
	Text string `json:"text,omitempty"`

	// Annotations 注释列表
	Annotations []any `json:"annotations,omitempty"`

	// Logprobs 对数概率
	Logprobs []any `json:"logprobs,omitempty"`

	// FunctionCall 函数调用（当 type 为 function_call 时）
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// ResponseDelta 表示增量内容
type ResponseDelta struct {
	// Text 文本增量
	Text string `json:"text,omitempty"`

	// Arguments 参数增量（用于函数调用）
	Arguments string `json:"arguments,omitempty"`
}

// ResponsesStreamResponse 表示完整的流式响应
type ResponsesStreamResponse struct {
	// ID 响应 ID
	ID string `json:"id"`

	// Object 对象类型
	Object string `json:"object"`

	// Created 创建时间戳
	Created int64 `json:"created_at"`

	// Model 模型名称
	Model string `json:"model"`

	// Status 响应状态：in_progress, completed, failed, incomplete
	Status string `json:"status"`

	// Output 输出项列表
	Output []ResponseOutputItem `json:"output,omitempty"`

	// Usage token 使用统计
	Usage *Usage `json:"usage,omitempty"`

	// Error 错误信息（当 status 为 failed 时）
	Error *ErrorDetail `json:"error,omitempty"`

	// IncompleteDetails 不完整详情（当 status 为 incomplete 时）
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
}

// IncompleteDetails 表示响应不完整的详情
type IncompleteDetails struct {
	// Reason 不完整的原因：max_tokens, content_filter 等
	Reason string `json:"reason"`
}

// ResponseOutputItem 表示响应输出项
type ResponseOutputItem struct {
	// ID 项目 ID
	ID string `json:"id"`

	// Type 项目类型：message, function_call 等
	Type string `json:"type"`

	// Role 角色（当 type 为 message 时）
	Role string `json:"role,omitempty"`

	// Content 内容列表
	Content []ResponseContentItem `json:"content,omitempty"`
}

// ResponseContentItem 表示响应内容项
type ResponseContentItem struct {
	// Type 内容类型：output_text, function_call 等
	Type string `json:"type"`

	// Text 文本内容
	Text string `json:"text,omitempty"`

	// Annotations 注释列表
	Annotations []any `json:"annotations,omitempty"`

	// Logprobs 对数概率
	Logprobs []any `json:"logprobs,omitempty"`

	// FunctionCall 函数调用
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// EmbeddingRequest 表示嵌入请求
type EmbeddingRequest struct {
	// Model 要使用的嵌入模型名称
	Model string `json:"model"`

	// Input 要嵌入的文本或文本数组
	Input any `json:"input"`

	// EncodingFormat 编码格式：float 或 base64
	EncodingFormat string `json:"encoding_format,omitempty"`

	// Dimensions 嵌入向量的维度（可选）
	Dimensions int `json:"dimensions,omitempty"`

	// User 用户标识符
	User string `json:"user,omitempty"`
}

// EmbeddingResponse 表示嵌入响应
type EmbeddingResponse struct {
	// Object 对象类型，通常为 "list"
	Object string `json:"object"`

	// Data 嵌入数据列表
	Data []EmbeddingData `json:"data"`

	// Model 使用的模型名称
	Model string `json:"model"`

	// Usage token 使用统计
	Usage EmbeddingUsage `json:"usage"`
}

// EmbeddingData 表示单个嵌入数据
type EmbeddingData struct {
	// Object 对象类型，通常为 "embedding"
	Object string `json:"object"`

	// Index 嵌入索引
	Index int `json:"index"`

	// Embedding 嵌入向量
	Embedding []float64 `json:"embedding"`
}

// EmbeddingUsage 表示嵌入的 token 使用统计
type EmbeddingUsage struct {
	// PromptTokens 提示 token 数
	PromptTokens int `json:"prompt_tokens"`

	// TotalTokens 总 token 数
	TotalTokens int `json:"total_tokens"`
}

// ErrorResponse 表示 API 错误响应
type ErrorResponse struct {
	// Error 错误信息
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 表示错误详情
type ErrorDetail struct {
	// Message 错误消息
	Message string `json:"message"`

	// Type 错误类型
	Type string `json:"type"`

	// Code 错误代码
	Code string `json:"code,omitempty"`

	// Param 相关参数
	Param string `json:"param,omitempty"`
}

// AzureAIError 表示 Azure AI 插件的错误
type AzureAIError struct {
	// Type 错误类型：config, request, network, api, parse
	Type string

	// Code HTTP 状态码或错误代码
	Code string

	// Message 错误消息
	Message string

	// Details 错误详情
	Details any

	// Err 原始错误
	Err error
}

// Error 实现 error 接口
func (e *AzureAIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Type, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AzureAIError) Unwrap() error {
	return e.Err
}

// NewConfigError 创建配置错误
func NewConfigError(message string, details any) *AzureAIError {
	return &AzureAIError{
		Type:    "config",
		Code:    "invalid_config",
		Message: message,
		Details: details,
	}
}

// NewRequestError 创建请求错误
func NewRequestError(message string, err error) *AzureAIError {
	return &AzureAIError{
		Type:    "request",
		Code:    "invalid_request",
		Message: message,
		Err:     err,
	}
}

// NewNetworkError 创建网络错误
func NewNetworkError(message string, err error) *AzureAIError {
	return &AzureAIError{
		Type:    "network",
		Code:    "network_error",
		Message: message,
		Err:     err,
	}
}

// NewAPIError 创建 API 错误
func NewAPIError(code, message string, details any) *AzureAIError {
	return &AzureAIError{
		Type:    "api",
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewParseError 创建解析错误
func NewParseError(message string, err error) *AzureAIError {
	return &AzureAIError{
		Type:    "parse",
		Code:    "parse_error",
		Message: message,
		Err:     err,
	}
}
