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

// OpenAIProvider OpenAI提供商实现
type OpenAIProvider struct {
	config     *OpenAIConfig
	httpClient *http.Client
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	BaseProviderConfig
	Organization string `json:"organization,omitempty"`
	Project      string `json:"project,omitempty"`
}

// NewOpenAIProvider 创建OpenAI提供商
func NewOpenAIProvider(config *OpenAIConfig) *OpenAIProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &OpenAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetProviderType 获取提供商类型
func (p *OpenAIProvider) GetProviderType() ProviderType {
	return ProviderOpenAI
}

// GetProviderName 获取提供商名称
func (p *OpenAIProvider) GetProviderName() string {
	if p.config.Name != "" {
		return p.config.Name
	}
	return "OpenAI"
}

// GenerateText 生成文本
func (p *OpenAIProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// 构建请求体
	requestBody := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}

	// 添加可选参数
	if req.Temperature != nil {
		requestBody["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		requestBody["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		requestBody["top_p"] = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		requestBody["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		requestBody["presence_penalty"] = *req.PresencePenalty
	}
	if len(req.Stop) > 0 {
		requestBody["stop"] = req.Stop
	}
	if req.User != "" {
		requestBody["user"] = req.User
	}

	// 发送请求
	resp, err := p.makeRequest(ctx, "POST", "/chat/completions", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	var response GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, WrapLLMError(ErrCodeInvalidResponse, "解析响应失败", ProviderOpenAI, err)
	}

	return &response, nil
}

// GenerateStream 流式生成文本
func (p *OpenAIProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamResponse, error) {
	// 构建请求体
	requestBody := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	// 添加可选参数
	if req.Temperature != nil {
		requestBody["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		requestBody["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		requestBody["top_p"] = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		requestBody["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		requestBody["presence_penalty"] = *req.PresencePenalty
	}
	if len(req.Stop) > 0 {
		requestBody["stop"] = req.Stop
	}
	if req.User != "" {
		requestBody["user"] = req.User
	}

	// 发送请求
	resp, err := p.makeRequest(ctx, "POST", "/chat/completions", requestBody)
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

				// 解析JSON数据
				var streamResp StreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					responseChan <- &StreamResponse{
						Done: true,
						Error: &StreamError{
							Code:    ErrCodeInvalidResponse,
							Message: "解析流式响应失败: " + err.Error(),
						},
					}
					return
				}

				responseChan <- &streamResp
			}
		}

		if err := scanner.Err(); err != nil {
			responseChan <- &StreamResponse{
				Done: true,
				Error: &StreamError{
					Code:    ErrCodeStreamClosed,
					Message: "读取流式响应失败: " + err.Error(),
				},
			}
		}
	}()

	return responseChan, nil
}

// ListModels 获取模型列表
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]Model, error) {
	resp, err := p.makeRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, WrapLLMError(ErrCodeInvalidResponse, "解析模型列表失败", ProviderOpenAI, err)
	}

	models := make([]Model, 0, len(response.Data))
	for _, m := range response.Data {
		model := Model{
			ID:          m.ID,
			Object:      m.Object,
			Created:     m.Created,
			OwnedBy:     m.OwnedBy,
			DisplayName: p.getModelDisplayName(m.ID),
			Description: p.getModelDescription(m.ID),
			ModelType:   p.getModelType(m.ID),
			Capabilities: p.getModelCapabilities(m.ID),
			Limits:      p.getModelLimits(m.ID),
			Pricing:     p.getModelPricing(m.ID),
		}
		models = append(models, model)
	}

	return models, nil
}

// HealthCheck 健康检查
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	resp, err := p.makeRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewLLMError(ErrCodeServiceUnavailable, "OpenAI服务不可用", ProviderOpenAI)
	}

	return nil
}

// makeRequest 发送HTTP请求
func (p *OpenAIProvider) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, WrapLLMError(ErrCodeInvalidRequest, "序列化请求体失败", ProviderOpenAI, err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := p.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, WrapLLMError(ErrCodeInvalidRequest, "创建请求失败", ProviderOpenAI, err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	
	if p.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", p.config.Organization)
	}
	if p.config.Project != "" {
		req.Header.Set("OpenAI-Project", p.config.Project)
	}

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, WrapLLMError(ErrCodeAPICallFailed, "API调用失败", ProviderOpenAI, err)
	}

	// 检查响应状态码
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		
		json.NewDecoder(resp.Body).Decode(&errorResp)
		
		code := p.mapErrorCode(resp.StatusCode, errorResp.Error.Type)
		message := errorResp.Error.Message
		if message == "" {
			message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		
		return nil, NewLLMError(code, message, ProviderOpenAI)
	}

	return resp, nil
}

// mapErrorCode 映射错误码
func (p *OpenAIProvider) mapErrorCode(statusCode int, errorType string) string {
	switch statusCode {
	case 401:
		return ErrCodeUnauthorized
	case 429:
		return ErrCodeRateLimitExceeded
	case 503:
		return ErrCodeServiceUnavailable
	default:
		return ErrCodeAPICallFailed
	}
}

// getModelDisplayName 获取模型显示名称
func (p *OpenAIProvider) getModelDisplayName(modelID string) string {
	displayNames := map[string]string{
		"gpt-4":                    "GPT-4",
		"gpt-4-turbo":             "GPT-4 Turbo",
		"gpt-4-turbo-preview":     "GPT-4 Turbo Preview",
		"gpt-4o":                  "GPT-4o",
		"gpt-4o-mini":             "GPT-4o Mini",
		"gpt-3.5-turbo":           "GPT-3.5 Turbo",
		"gpt-3.5-turbo-16k":       "GPT-3.5 Turbo 16K",
		"text-embedding-3-large":  "Text Embedding 3 Large",
		"text-embedding-3-small":  "Text Embedding 3 Small",
		"text-embedding-ada-002":  "Text Embedding Ada 002",
	}
	
	if displayName, exists := displayNames[modelID]; exists {
		return displayName
	}
	return modelID
}

// getModelDescription 获取模型描述
func (p *OpenAIProvider) getModelDescription(modelID string) string {
	descriptions := map[string]string{
		"gpt-4":                   "最强大的GPT-4模型，适合复杂任务",
		"gpt-4-turbo":            "更快的GPT-4模型，性价比更高",
		"gpt-4o":                 "最新的多模态GPT-4模型",
		"gpt-4o-mini":            "轻量级的GPT-4o模型",
		"gpt-3.5-turbo":          "快速且经济的对话模型",
		"text-embedding-3-large": "高质量的文本嵌入模型",
		"text-embedding-3-small": "轻量级的文本嵌入模型",
	}
	
	if description, exists := descriptions[modelID]; exists {
		return description
	}
	return ""
}

// getModelType 获取模型类型
func (p *OpenAIProvider) getModelType(modelID string) ModelType {
	if strings.Contains(modelID, "embedding") {
		return ModelTypeEmbedding
	}
	if strings.HasPrefix(modelID, "gpt-") {
		return ModelTypeChat
	}
	return ModelTypeCompletion
}

// getModelCapabilities 获取模型能力
func (p *OpenAIProvider) getModelCapabilities(modelID string) []string {
	capabilities := map[string][]string{
		"gpt-4o": {"text", "image", "function_calling"},
		"gpt-4":  {"text", "function_calling"},
		"gpt-4-turbo": {"text", "function_calling", "json_mode"},
		"gpt-3.5-turbo": {"text", "function_calling"},
		"text-embedding-3-large": {"embedding"},
		"text-embedding-3-small": {"embedding"},
	}
	
	if caps, exists := capabilities[modelID]; exists {
		return caps
	}
	return []string{"text"}
}

// getModelLimits 获取模型限制
func (p *OpenAIProvider) getModelLimits(modelID string) ModelLimits {
	limits := map[string]ModelLimits{
		"gpt-4": {
			MaxTokens:       8192,
			MaxInputTokens:  8192,
			MaxOutputTokens: 4096,
			ContextWindow:   8192,
		},
		"gpt-4-turbo": {
			MaxTokens:       128000,
			MaxInputTokens:  128000,
			MaxOutputTokens: 4096,
			ContextWindow:   128000,
		},
		"gpt-4o": {
			MaxTokens:       128000,
			MaxInputTokens:  128000,
			MaxOutputTokens: 4096,
			ContextWindow:   128000,
		},
		"gpt-4o-mini": {
			MaxTokens:       128000,
			MaxInputTokens:  128000,
			MaxOutputTokens: 16384,
			ContextWindow:   128000,
		},
		"gpt-3.5-turbo": {
			MaxTokens:       4096,
			MaxInputTokens:  4096,
			MaxOutputTokens: 4096,
			ContextWindow:   4096,
		},
	}
	
	if limit, exists := limits[modelID]; exists {
		return limit
	}
	return ModelLimits{
		MaxTokens:     4096,
		ContextWindow: 4096,
	}
}

// getModelPricing 获取模型定价
func (p *OpenAIProvider) getModelPricing(modelID string) *ModelPricing {
	pricing := map[string]ModelPricing{
		"gpt-4": {
			InputPrice:  0.03,
			OutputPrice: 0.06,
			Currency:    "USD",
		},
		"gpt-4-turbo": {
			InputPrice:  0.01,
			OutputPrice: 0.03,
			Currency:    "USD",
		},
		"gpt-4o": {
			InputPrice:  0.005,
			OutputPrice: 0.015,
			Currency:    "USD",
		},
		"gpt-4o-mini": {
			InputPrice:  0.00015,
			OutputPrice: 0.0006,
			Currency:    "USD",
		},
		"gpt-3.5-turbo": {
			InputPrice:  0.0015,
			OutputPrice: 0.002,
			Currency:    "USD",
		},
	}
	
	if price, exists := pricing[modelID]; exists {
		return &price
	}
	return nil
}