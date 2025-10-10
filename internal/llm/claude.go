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

// ClaudeProvider Claude提供商实现
type ClaudeProvider struct {
	config     *ClaudeConfig
	httpClient *http.Client
}

// ClaudeConfig Claude配置
type ClaudeConfig struct {
	BaseProviderConfig
	Version string `json:"version,omitempty"`
}

// NewClaudeProvider 创建Claude提供商
func NewClaudeProvider(config *ClaudeConfig) *ClaudeProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	if config.Version == "" {
		config.Version = "2023-06-01"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &ClaudeProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetProviderType 获取提供商类型
func (p *ClaudeProvider) GetProviderType() ProviderType {
	return ProviderClaude
}

// GetProviderName 获取提供商名称
func (p *ClaudeProvider) GetProviderName() string {
	if p.config.Name != "" {
		return p.config.Name
	}
	return "Claude"
}

// GenerateText 生成文本
func (p *ClaudeProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// 构建Claude格式的请求体
	requestBody := map[string]interface{}{
		"model":      req.Model,
		"messages":   req.Messages,
		"max_tokens": 4096, // Claude要求必须设置max_tokens
		"stream":     false,
	}

	// 添加可选参数
	if req.MaxTokens != nil {
		requestBody["max_tokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		requestBody["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		requestBody["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		requestBody["stop_sequences"] = req.Stop
	}

	// 发送请求
	resp, err := p.makeRequest(ctx, "POST", "/v1/messages", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析Claude响应格式
	var claudeResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, WrapLLMError(ErrCodeInvalidResponse, "解析Claude响应失败", ProviderClaude, err)
	}

	// 提取文本内容
	var content string
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	// 转换为标准格式
	response := &GenerateResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   claudeResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: claudeResp.StopReason,
			},
		},
		Usage: Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}

	return response, nil
}

// GenerateStream 流式生成文本
func (p *ClaudeProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamResponse, error) {
	// 构建Claude格式的请求体
	requestBody := map[string]interface{}{
		"model":      req.Model,
		"messages":   req.Messages,
		"max_tokens": 4096,
		"stream":     true,
	}

	// 添加可选参数
	if req.MaxTokens != nil {
		requestBody["max_tokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		requestBody["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		requestBody["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		requestBody["stop_sequences"] = req.Stop
	}

	// 发送请求
	resp, err := p.makeRequest(ctx, "POST", "/v1/messages", requestBody)
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
			
			// 跳过空行和注释行
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// 处理SSE格式
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				
				// 检查结束标记
				if data == "[DONE]" {
					responseChan <- &StreamResponse{Done: true}
					return
				}

				// 解析Claude流式响应
				var claudeEvent struct {
					Type  string `json:"type"`
					Index int    `json:"index"`
					Delta struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"delta"`
					Message struct {
						ID      string `json:"id"`
						Type    string `json:"type"`
						Role    string `json:"role"`
						Content []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"content"`
						Model        string `json:"model"`
						StopReason   string `json:"stop_reason"`
						StopSequence string `json:"stop_sequence"`
						Usage        struct {
							InputTokens  int `json:"input_tokens"`
							OutputTokens int `json:"output_tokens"`
						} `json:"usage"`
					} `json:"message"`
				}

				if err := json.Unmarshal([]byte(data), &claudeEvent); err != nil {
					responseChan <- &StreamResponse{
						Done: true,
						Error: &StreamError{
							Code:    ErrCodeInvalidResponse,
							Message: "解析Claude流式响应失败: " + err.Error(),
						},
					}
					return
				}

				// 根据事件类型处理
				switch claudeEvent.Type {
				case "content_block_delta":
					if claudeEvent.Delta.Type == "text_delta" {
						streamResp := &StreamResponse{
							ID:      "claude-stream",
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   req.Model,
							Choices: []StreamChoice{
								{
									Index: claudeEvent.Index,
									Delta: MessageDelta{
										Content: claudeEvent.Delta.Text,
									},
								},
							},
						}
						responseChan <- streamResp
					}
				case "message_stop":
					finishReason := claudeEvent.Message.StopReason
					streamResp := &StreamResponse{
						ID:      claudeEvent.Message.ID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   claudeEvent.Message.Model,
						Choices: []StreamChoice{
							{
								Index:        0,
								FinishReason: &finishReason,
							},
						},
						Usage: &Usage{
							PromptTokens:     claudeEvent.Message.Usage.InputTokens,
							CompletionTokens: claudeEvent.Message.Usage.OutputTokens,
							TotalTokens:      claudeEvent.Message.Usage.InputTokens + claudeEvent.Message.Usage.OutputTokens,
						},
					}
					responseChan <- streamResp
				}
			}
		}

		if err := scanner.Err(); err != nil {
			responseChan <- &StreamResponse{
				Done: true,
				Error: &StreamError{
					Code:    ErrCodeStreamClosed,
					Message: "读取Claude流式响应失败: " + err.Error(),
				},
			}
		}
	}()

	return responseChan, nil
}

// ListModels 获取模型列表
func (p *ClaudeProvider) ListModels(ctx context.Context) ([]Model, error) {
	// Claude的预定义模型列表
	models := []Model{
		{
			ID:          "claude-3-5-sonnet-20241022",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "anthropic",
			DisplayName: "Claude 3.5 Sonnet",
			Description: "最新的Claude 3.5 Sonnet模型，平衡性能和速度",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "vision", "function_calling"},
			Limits: ModelLimits{
				MaxTokens:       8192,
				MaxInputTokens:  200000,
				MaxOutputTokens: 8192,
				ContextWindow:   200000,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.003,
				OutputPrice: 0.015,
				Currency:    "USD",
			},
		},
		{
			ID:          "claude-3-opus-20240229",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "anthropic",
			DisplayName: "Claude 3 Opus",
			Description: "最强大的Claude模型，适合复杂任务",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "vision", "function_calling"},
			Limits: ModelLimits{
				MaxTokens:       4096,
				MaxInputTokens:  200000,
				MaxOutputTokens: 4096,
				ContextWindow:   200000,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.015,
				OutputPrice: 0.075,
				Currency:    "USD",
			},
		},
		{
			ID:          "claude-3-haiku-20240307",
			Object:      "model",
			Created:     time.Now().Unix(),
			OwnedBy:     "anthropic",
			DisplayName: "Claude 3 Haiku",
			Description: "快速且经济的Claude模型",
			ModelType:   ModelTypeChat,
			Capabilities: []string{"text", "vision"},
			Limits: ModelLimits{
				MaxTokens:       4096,
				MaxInputTokens:  200000,
				MaxOutputTokens: 4096,
				ContextWindow:   200000,
			},
			Pricing: &ModelPricing{
				InputPrice:  0.00025,
				OutputPrice: 0.00125,
				Currency:    "USD",
			},
		},
	}

	return models, nil
}

// HealthCheck 健康检查
func (p *ClaudeProvider) HealthCheck(ctx context.Context) error {
	// 使用简单的请求测试连接
	testReq := &GenerateRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: func() *int { i := 1; return &i }(),
	}

	_, err := p.GenerateText(ctx, testReq)
	if err != nil {
		return WrapLLMError(ErrCodeServiceUnavailable, "Claude服务不可用", ProviderClaude, err)
	}

	return nil
}

// makeRequest 发送HTTP请求
func (p *ClaudeProvider) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, WrapLLMError(ErrCodeInvalidRequest, "序列化请求体失败", ProviderClaude, err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := p.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, WrapLLMError(ErrCodeInvalidRequest, "创建请求失败", ProviderClaude, err)
	}

	// 设置Claude特有的请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", p.config.Version)

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, WrapLLMError(ErrCodeAPICallFailed, "Claude API调用失败", ProviderClaude, err)
	}

	// 检查响应状态码
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		
		var errorResp struct {
			Type  string `json:"type"`
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		
		json.NewDecoder(resp.Body).Decode(&errorResp)
		
		code := p.mapErrorCode(resp.StatusCode, errorResp.Error.Type)
		message := errorResp.Error.Message
		if message == "" {
			message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		
		return nil, NewLLMError(code, message, ProviderClaude)
	}

	return resp, nil
}

// mapErrorCode 映射错误码
func (p *ClaudeProvider) mapErrorCode(statusCode int, errorType string) string {
	switch statusCode {
	case 401:
		return ErrCodeUnauthorized
	case 429:
		return ErrCodeRateLimitExceeded
	case 503:
		return ErrCodeServiceUnavailable
	default:
		// 根据Claude的错误类型进行映射
		switch errorType {
		case "authentication_error":
			return ErrCodeUnauthorized
		case "permission_error":
			return ErrCodeUnauthorized
		case "rate_limit_error":
			return ErrCodeRateLimitExceeded
		case "api_error":
			return ErrCodeServiceUnavailable
		case "overloaded_error":
			return ErrCodeServiceUnavailable
		default:
			return ErrCodeAPICallFailed
		}
	}
}