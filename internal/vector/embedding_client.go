package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPEmbeddingClient HTTP向量化客户端基础实现
type HTTPEmbeddingClient struct {
	config     *EmbeddingConfig
	httpClient *http.Client
	utils      *VectorUtils
}

// NewHTTPEmbeddingClient 创建HTTP向量化客户端
func NewHTTPEmbeddingClient(config *EmbeddingConfig) (*HTTPEmbeddingClient, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	client := &HTTPEmbeddingClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		utils: NewVectorUtils(),
	}
	
	return client, nil
}

// Embed 生成单个文本的向量
func (c *HTTPEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	
	return embeddings[0], nil
}

// EmbedBatch 批量生成文本向量
func (c *HTTPEmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	
	// 根据提供商选择实现
	switch c.config.Provider {
	case EmbeddingProviderOpenAI:
		return c.embedOpenAI(ctx, texts)
	case EmbeddingProviderAzure:
		return c.embedAzure(ctx, texts)
	case EmbeddingProviderQianwen:
		return c.embedQianwen(ctx, texts)
	case EmbeddingProviderBaichuan:
		return c.embedBaichuan(ctx, texts)
	case EmbeddingProviderZhipu:
		return c.embedZhipu(ctx, texts)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.config.Provider)
	}
}

// GetDimension 获取向量维度
func (c *HTTPEmbeddingClient) GetDimension() int {
	return c.config.Dimension
}

// GetModelName 获取模型名称
func (c *HTTPEmbeddingClient) GetModelName() string {
	return c.config.Model
}

// HealthCheck 健康检查
func (c *HTTPEmbeddingClient) HealthCheck(ctx context.Context) error {
	// 使用简单文本测试API连接
	_, err := c.Embed(ctx, "test")
	return err
}

// embedOpenAI OpenAI API实现
func (c *HTTPEmbeddingClient) embedOpenAI(ctx context.Context, texts []string) ([][]float32, error) {
	url := fmt.Sprintf("%s/embeddings", c.config.BaseURL)
	
	requestBody := map[string]interface{}{
		"input": texts,
		"model": c.config.Model,
	}
	
	response, err := c.makeRequest(ctx, "POST", url, requestBody, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.config.APIKey),
		"Content-Type":  "application/json",
	})
	if err != nil {
		return nil, err
	}
	
	// 解析OpenAI响应格式
	var openAIResponse struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(response, &openAIResponse); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}
	
	embeddings := make([][]float32, len(openAIResponse.Data))
	for i, data := range openAIResponse.Data {
		embeddings[i] = data.Embedding
	}
	
	return embeddings, nil
}

// embedAzure Azure OpenAI API实现
func (c *HTTPEmbeddingClient) embedAzure(ctx context.Context, texts []string) ([][]float32, error) {
	// Azure OpenAI的URL格式不同
	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=2023-05-15", 
		c.config.BaseURL, c.config.Model)
	
	requestBody := map[string]interface{}{
		"input": texts,
	}
	
	response, err := c.makeRequest(ctx, "POST", url, requestBody, map[string]string{
		"api-key":      c.config.APIKey,
		"Content-Type": "application/json",
	})
	if err != nil {
		return nil, err
	}
	
	// Azure使用与OpenAI相同的响应格式
	var azureResponse struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(response, &azureResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Azure response: %w", err)
	}
	
	embeddings := make([][]float32, len(azureResponse.Data))
	for i, data := range azureResponse.Data {
		embeddings[i] = data.Embedding
	}
	
	return embeddings, nil
}

// embedQianwen 千问API实现
func (c *HTTPEmbeddingClient) embedQianwen(ctx context.Context, texts []string) ([][]float32, error) {
	url := fmt.Sprintf("%s/services/embeddings/text-embedding/text-embedding", c.config.BaseURL)
	
	requestBody := map[string]interface{}{
		"model": c.config.Model,
		"input": map[string]interface{}{
			"texts": texts,
		},
	}
	
	response, err := c.makeRequest(ctx, "POST", url, requestBody, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.config.APIKey),
		"Content-Type":  "application/json",
	})
	if err != nil {
		return nil, err
	}
	
	// 解析千问响应格式
	var qianwenResponse struct {
		Output struct {
			Embeddings []struct {
				Embedding []float32 `json:"embedding"`
			} `json:"embeddings"`
		} `json:"output"`
	}
	
	if err := json.Unmarshal(response, &qianwenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Qianwen response: %w", err)
	}
	
	embeddings := make([][]float32, len(qianwenResponse.Output.Embeddings))
	for i, data := range qianwenResponse.Output.Embeddings {
		embeddings[i] = data.Embedding
	}
	
	return embeddings, nil
}

// embedBaichuan 百川API实现
func (c *HTTPEmbeddingClient) embedBaichuan(ctx context.Context, texts []string) ([][]float32, error) {
	url := fmt.Sprintf("%s/embeddings", c.config.BaseURL)
	
	requestBody := map[string]interface{}{
		"model": c.config.Model,
		"input": texts,
	}
	
	response, err := c.makeRequest(ctx, "POST", url, requestBody, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.config.APIKey),
		"Content-Type":  "application/json",
	})
	if err != nil {
		return nil, err
	}
	
	// 解析百川响应格式（类似OpenAI）
	var baichuanResponse struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(response, &baichuanResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Baichuan response: %w", err)
	}
	
	embeddings := make([][]float32, len(baichuanResponse.Data))
	for i, data := range baichuanResponse.Data {
		embeddings[i] = data.Embedding
	}
	
	return embeddings, nil
}

// embedZhipu 智谱API实现
func (c *HTTPEmbeddingClient) embedZhipu(ctx context.Context, texts []string) ([][]float32, error) {
	url := fmt.Sprintf("%s/embeddings", c.config.BaseURL)
	
	requestBody := map[string]interface{}{
		"model": c.config.Model,
		"input": texts,
	}
	
	response, err := c.makeRequest(ctx, "POST", url, requestBody, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.config.APIKey),
		"Content-Type":  "application/json",
	})
	if err != nil {
		return nil, err
	}
	
	// 解析智谱响应格式
	var zhipuResponse struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(response, &zhipuResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Zhipu response: %w", err)
	}
	
	embeddings := make([][]float32, len(zhipuResponse.Data))
	for i, data := range zhipuResponse.Data {
		embeddings[i] = data.Embedding
	}
	
	return embeddings, nil
}

// makeRequest 发送HTTP请求
func (c *HTTPEmbeddingClient) makeRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) ([]byte, error) {
	var reqBody io.Reader
	
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}