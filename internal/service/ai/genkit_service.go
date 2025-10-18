package ai

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

// genkitService 基于 Genkit 的 AI 服务实现
type genkitService struct {
	client         genkit.Client
	contextManager ContextManager
	logger         logger.Logger
}

// NewGenkitService 创建新的 Genkit AI 服务
// 参数:
//   client: Genkit 客户端
//   contextManager: 上下文管理器
//   log: 日志记录器
// 返回:
//   AIService: AI 服务实例
func NewGenkitService(client genkit.Client, contextManager ContextManager, log logger.Logger) AIService {
	return &genkitService{
		client:         client,
		contextManager: contextManager,
		logger:         log,
	}
}

// Chat 发起对话
func (s *genkitService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	startTime := time.Now()

	// 创建或获取会话
	var sessionID string
	var sessionCtx context.Context

	if req.SessionID != "" {
		// 使用现有会话
		sessionID = req.SessionID
		existingCtx, exists := s.contextManager.GetSession(sessionID)
		if !exists {
			s.logger.WarnContext(ctx, "会话不存在，创建新会话", logger.Fields{
				"requestedSessionId": req.SessionID,
			})
			sessionID, sessionCtx, _ = s.contextManager.CreateSession(ctx)
		} else {
			sessionCtx = existingCtx
		}
	} else {
		// 创建新会话
		sessionID, sessionCtx, _ = s.contextManager.CreateSession(ctx)
	}

	// 记录请求日志
	s.logger.InfoContext(sessionCtx, "开始处理对话请求", logger.Fields{
		"sessionId": sessionID,
		"message":   req.Message,
	})

	// 构建生成选项
	options := s.buildGenerateOptions(req.Options)

	// 调用 Genkit 生成响应
	result, err := s.client.Generate(sessionCtx, req.Message, options)
	if err != nil {
		// 检查是否是上下文取消错误
		if sessionCtx.Err() == context.Canceled {
			s.logger.WarnContext(ctx, "对话请求被取消", logger.Fields{
				"sessionId": sessionID,
				"error":     err.Error(),
			})
			return nil, errors.NewContextCancelledError()
		}

		s.logger.ErrorContext(ctx, "AI 生成失败", logger.Fields{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return nil, errors.NewAIServiceError(err)
	}

	// 构建响应
	response := &model.ChatResponse{
		SessionID: sessionID,
		Message:   result.Text,
		Model:     result.Model,
	}

	// 添加 token 使用情况
	if result.Usage != nil {
		response.Usage = &model.Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
	}

	// 记录成功日志
	duration := time.Since(startTime)
	s.logger.InfoContext(sessionCtx, "对话请求处理完成", logger.Fields{
		"sessionId": sessionID,
		"model":     result.Model,
		"duration":  duration.String(),
		"tokens":    response.Usage,
	})

	return response, nil
}

// ChatStream 流式对话（预留接口）
func (s *genkitService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error) {
	// TODO: 实现流式对话功能
	// 当前版本暂不实现，返回未实现错误
	return nil, fmt.Errorf("流式对话功能暂未实现")
}

// AbortChat 中止对话
func (s *genkitService) AbortChat(ctx context.Context, sessionID string) error {
	s.logger.InfoContext(ctx, "尝试中止对话", logger.Fields{
		"sessionId": sessionID,
	})

	// 检查会话是否存在
	sessionCtx, exists := s.contextManager.GetSession(sessionID)
	if !exists {
		// 会话不存在，视为幂等操作，直接返回成功
		s.logger.InfoContext(ctx, "会话不存在，无需中止", logger.Fields{
			"sessionId": sessionID,
		})
		return nil
	}

	// 检查会话是否已经完成
	if sessionCtx.Err() != nil {
		// 会话已完成或已取消，视为幂等操作，直接返回成功
		s.logger.InfoContext(ctx, "会话已完成或已取消，无需中止", logger.Fields{
			"sessionId": sessionID,
		})
		return nil
	}

	// 取消会话
	err := s.contextManager.CancelSession(sessionID)
	if err != nil {
		s.logger.ErrorContext(ctx, "取消会话失败", logger.Fields{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return errors.NewInternalError(err)
	}

	s.logger.InfoContext(ctx, "会话已成功中止", logger.Fields{
		"sessionId": sessionID,
	})

	return nil
}

// buildGenerateOptions 构建生成选项
func (s *genkitService) buildGenerateOptions(options *model.ChatOptions) *genkit.GenerateOptions {
	if options == nil {
		return nil
	}

	return &genkit.GenerateOptions{
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
		TopP:        options.TopP,
		TopK:        options.TopK,
	}
}
