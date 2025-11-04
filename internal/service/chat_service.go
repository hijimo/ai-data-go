package service

import (
	"context"
)

// ChatService 对话服务接口
type ChatService interface {
	// GenerateResponse 生成AI响应
	GenerateResponse(ctx context.Context, req GenerateResponseRequest) (*GenerateResponseResult, error)
}

// GenerateResponseRequest 生成响应请求
type GenerateResponseRequest struct {
	SessionID    string
	UserMessage  string
	Context      *ContextResult
	ModelConfig  *ModelConfig
	SystemPrompt string
	SaveMessage  bool
}

// GenerateResponseResult 生成响应结果
type GenerateResponseResult struct {
	MessageID      string
	Response       string
	TokenUsage     TokenUsageResult
	FinishReason   string
	Model          string
	GenerationTime int64
	ContextInfo    ContextInfoResult
}

// ModelConfig 模型配置
type ModelConfig struct {
	ModelName        string
	Temperature      float64
	TopP             float64
	MaxTokens        int
	StopSequences    []string
	FrequencyPenalty float64
	PresencePenalty  float64
}

// TokenUsageResult Token使用结果
type TokenUsageResult struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ContextInfoResult 上下文信息结果
type ContextInfoResult struct {
	ContextTokens int
	Strategy      string
	QualityScore  float64
}

// chatServiceImpl 对话服务实现（占位符）
type chatServiceImpl struct {
	// TODO: 添加依赖
}

// NewChatService 创建对话服务实例
func NewChatService() ChatService {
	return &chatServiceImpl{}
}

// GenerateResponse 生成AI响应（占位符实现）
func (s *chatServiceImpl) GenerateResponse(ctx context.Context, req GenerateResponseRequest) (*GenerateResponseResult, error) {
	// TODO: 实现实际的AI响应生成逻辑
	return &GenerateResponseResult{
		MessageID:    "placeholder-message-id",
		Response:     "这是一个占位符响应",
		TokenUsage:   TokenUsageResult{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		FinishReason: "stop",
		Model:        "gemini-1.5-flash",
		GenerationTime: 1000,
		ContextInfo: ContextInfoResult{
			ContextTokens: 100,
			Strategy:      "auto",
			QualityScore:  0.85,
		},
	}, nil
}
