package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QianwenProvider 千问提供商实现
type QianwenProvider struct {
	config     *QianwenConfig
	httpClient *http.Client
}

// QianwenConfig 千问配置
type QianwenConfig struct {
	BaseProviderConfig
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// NewQianwenProvider 创建千问提供商
func NewQianwenProvider(config *QianwenConfig) *QianwenProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://dashscope.aliyuncs.com/api/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &QianwenProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetProviderType 获取提供商类型
func (p *QianwenProvider) GetProviderType() ProviderType {
	return ProviderQianwen
}

// GetProviderName 获取提供商名称
func (p *QianwenProvider) GetProviderName() string {
	if p.config.Name != "" {
		return p.config.Name
	}
	return "通义千问"
}

// GenerateText 生成文本
func (p *QianwenProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// 构建千问格式的请求体
	requestBody := map[string]interface{}{
		"model": req.Model,
		"input": map[string]interface{}{
			"messages": req.Messages,
		},
		"parameters": make(map[string]interface{}),
	}

	// 添加参数
	parameters := requestBody["parameters"].(map[string]interface{})
	if req.Temperature != nil {
		parameters["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		parameters["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		parameters["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		parameters["stop"] = req.Stop
	}

	// 发送请求
	resp, err := p.makeRequest(ctx, "POST", "/services/aigc/text-generation/generation", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析千问响应格式
	var qianwenResp struct {
		Output struct {
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
		RequestID string `json:"request_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&qianwenResp); err != nil {
		return nil, WrapLLMError(ErrCodeInvalidResponse, "解析千问响应失败", ProviderQianwen, err)
	}

	// 转换为标准格式
	response := &GenerateResponse{
		ID:      qianwenResp.RequestID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: qianwenResp.Output.Text,
				},
				FinishReason: qianwenResp.Output.FinishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     qianwenResp.Usage.InputTokens,
			CompletionTokens: qianwenResp.Usage.OutputTokens,
			TotalTokens:      qianwenResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// GenerateStream 流式生成文本
func (p *QianwenProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamResponse, error) {
	// 构建千问格式的请求体
	requestBody := map[string]interface{}{
		"model": req.Model,
		"input": map[string]interface{}{
			"messages": req.Messages,
		},
		"parameters": map[string]interface{}{
			"incremental_output": true,
		},
	}

	// 添加参数
	parameters := requestBody["parameters"].(map[string]interface{})
	if req.Temperature != nil {
		parameters["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		parameters["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		parameters["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		parameters["stop"] = req.Stop
	}

	// 发送请求
	resp, err := p.makeRequest(ctx, "POST", "/services/aigc/text-generation/generation", requestBody)
	if err != nil {
		return nil, err
	}

	// 创建响应通道
	responseChan := make(chan *StreamResponse, 10)

	// 启动goroutine处理流式响应
	go func() {
		defer close(responseChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			
			// 跳过空行
			if line == "" {
				continue
			}

			// 处理千问的SSE格式
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data)
				
				// 检查结束标记
				if data == "[DONE]" {
					responseChan <- &StreamResponse{Done: true}
					return
				}

				// 解析千问响应
				var qianwenResp struct {
					Output struct {
						Text         string `json:"text"`
						FinishReason string `json:"finish_reason"`
					} `json:"output"`
					Usage struct {
						InputTokens  int `json:"input_tokens"`
						OutputTokens int `json:"output_tokens"`
						TotalTokens  int `json:"total_tokens"`
					} `json:"usage"`
					RequestID string `json:"request_id"`
				}

				if err := json.Unmarshal([]byte(data), &qianwenResp); err != nil {
					responseChan <- &StreamResponse{
						Done: true,
						Error: &StreamError{
							Code:    ErrCodeInvalidResponse,
							Message: "解析千问流式响应失败: " + err.Error(),
						},
					}
					return
				}

				// 转换为标准格式
				streamResp := &StreamResponse{
					ID:      qianwenResp.RequestID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   req.Model,
					Choices: []StreamChoice{
						{
							Index: 0,
							Delta: MessageDelta{
								Content: qianwenResp.Output.Text,
							},
						},
					},
				}

				if qianwenResp.Output.FinishReason != "" {
					finishReason := qianwenResp.Output.FinishReason
					streamResp.Choices[0].FinishReason = &finishReason
					streamResp.Usage = &Usage{
						PromptTokens:     qianwenResp.Usage.InputTokens,
						CompletionTokens: qianwenResp.Usage.OutputTokens,
						TotalTokens:      qianwenResp.Usage.TotalTokens,
					}
				}

				responseChan <- streamResp
			}
		}

		if err := scanner.Err(); err != nil {
			responseChan <- &StreamResponse{
				Done: true,
				Error: &StreamError{
					Code:    ErrCodeStreamClosed,
					Message: "读取千问流式响应失败: " + err.Error(),
				},
			}
		}
	}()

	return responseChan, nil
}

// ListModels 获取模型列表
func (p *QianwenProvider) ListModels(ctx context.Context) ([]Model, error) {
	// 千问的预定义模型列表
	models := []Model{
		{
			ID:          "qwen-turbo",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "alibaba",
			DisplayName: "通义千问-Turbo",
			Description: "快速响应的对话模型，适合日常对话",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "function_calling"},
			Limits: ModelLimits{
				MaxTokens:       8192,
				MaxInputTokens:  8192,
				MaxOutputTokens: 1500,
				ContextWindow:   8192,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.0008,
				OutputPrice: 0.002,
				Currency:    "CNY",
			},
		},
		{
			ID:          "qwen-plus",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "alibaba",
			DisplayName: "通义千问-Plus",
			Description: "平衡性能和成本的对话模型",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "function_calling"},
			Limits: ModelLimits{
				MaxTokens:       32768,
				MaxInputTokens:  32768,
				MaxOutputTokens: 2000,
				ContextWindow:   32768,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.004,
				OutputPrice: 0.012,
				Currency:    "CNY",
			},
		},
		{
			ID:          "qwen-max",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "alibaba",
			DisplayName: "通义千问-Max",
			Description: "最强大的千问模型，适合复杂任务",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "function_calling", "multimodal"},
			Limits: ModelLimits{
				MaxTokens:       8192,
				MaxInputTokens:  8192,
				MaxOutputTokens: 2000,
				ContextWindow:   8192,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.02,
				OutputPrice: 0.06,
				Currency:    "CNY",
			},
		},
		{
			ID:          "qwen-max-longcontext",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "alibaba",
			DisplayName: "通义千问-Max-长文本",
			Description: "支持长文本的千问模型",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "long_context"},
			Limits: ModelLimits{
				MaxTokens:       30000,
				MaxInputTokens:  30000,
				MaxOutputTokens: 2000,
				ContextWindow:   30000,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.02,
				OutputPrice: 0.06,
				Currency:    "CNY",
			},
		},
	}

	return models, nil
}

// HealthCheck 健康检查
func (p *QianwenProvider) HealthCheck(ctx context.Context) error {
	// 使用简单的请求测试连接
	testReq := &GenerateRequest{
		Model: "qwen-turbo",
		Messages: []Message{
			{Role: "user", Content: "你好"},
		},
		MaxTokens: func() *int { i := 1; return &i }(),
	}

	_, err := p.GenerateText(ctx, testReq)
	if err != nil {
		return WrapLLMError(ErrCodeServiceUnavailable, "千问服务不可用", ProviderQianwen, err)
	}

	return nil
}

// makeRequest 发送HTTP请求
func (p *QianwenProvider) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, WrapLLMError(ErrCodeInvalidRequest, "序列化请求体失败", ProviderQianwen, err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := p.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, WrapLLMError(ErrCodeInvalidRequest, "创建请求失败", ProviderQianwen, err)
	}

	// 设置千问特有的请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("X-DashScope-SSE", "enable") // 启用SSE

	if p.config.WorkspaceID != "" {
		req.Header.Set("X-DashScope-WorkspaceId", p.config.WorkspaceID)
	}

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, WrapLLMError(ErrCodeAPICallFailed, "千问API调用失败", ProviderQianwen, err)
	}

	// 检查响应状态码
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		
		var errorResp struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		}
		
		json.NewDecoder(resp.Body).Decode(&errorResp)
		
		code := p.mapErrorCode(resp.StatusCode, errorResp.Code)
		message := errorResp.Message
		if message == "" {
			message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		
		return nil, NewLLMErrorWithDetails(code, message, ProviderQianwen, map[string]interface{}{
			"request_id": errorResp.RequestID,
			"error_code": errorResp.Code,
		})
	}

	return resp, nil
}

// mapErrorCode 映射错误码
func (p *QianwenProvider) mapErrorCode(statusCode int, errorCode string) string {
	switch statusCode {
	case 401:
		return ErrCodeUnauthorized
	case 429:
		return ErrCodeRateLimitExceeded
	case 503:
		return ErrCodeServiceUnavailable
	default:
		// 根据千问的错误码进行映射
		switch errorCode {
		case "InvalidApiKey":
			return ErrCodeUnauthorized
		case "Throttling.RateQuota":
			return ErrCodeRateLimitExceeded
		case "Throttling.AllocationQuota":
			return ErrCodeQuotaExceeded
		case "InternalError.Timeout":
			return ErrCodeTimeout
		default:
			return ErrCodeAPICallFailed
		}
	}
}