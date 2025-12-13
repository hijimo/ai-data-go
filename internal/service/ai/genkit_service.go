package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	authservice "genkit-ai-service/internal/service/auth"
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
//
//	client: Genkit 客户端
//	contextManager: 上下文管理器
//	log: 日志记录器
//
// 返回:
//
//	AIService: AI 服务实例
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

	if req.MessageID != "" {
		// 使用现有会话（通过消息ID）
		sessionID = req.MessageID
		existingCtx, exists := s.contextManager.GetSession(sessionID)
		if !exists {
			s.logger.WarnContext(ctx, "会话不存在，创建新会话", logger.Fields{
				"requestedMessageId": req.MessageID,
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

	// 从上下文中获取租户ID
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		s.logger.ErrorContext(ctx, "无法从上下文获取JWT Claims", logger.Fields{
			"sessionId": sessionID,
		})
		return nil, errors.NewUnauthorizedError("身份认证信息缺失")
	}

	tenantID := claims.TenantID
	if tenantID == "" {
		s.logger.ErrorContext(ctx, "JWT Claims 中缺少租户ID", logger.Fields{
			"sessionId": sessionID,
			"userId":    claims.Subject,
		})
		return nil, errors.NewUnauthorizedError("租户信息缺失")
	}

	// 记录租户ID
	s.logger.DebugContext(ctx, "从上下文获取租户ID", logger.Fields{
		"sessionId": sessionID,
		"tenantId":  tenantID,
		"userId":    claims.Subject,
	})

	// 从请求中获取模型名称
	// 如果请求中指定了模型名称，使用它；否则使用默认模型
	modelName := "gemini-pro" // 默认模型
	if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
		modelName = *req.Options.ModelName
		s.logger.DebugContext(ctx, "使用请求中指定的模型", logger.Fields{
			"sessionId": sessionID,
			"modelName": modelName,
		})
	} else {
		s.logger.DebugContext(ctx, "使用默认模型", logger.Fields{
			"sessionId": sessionID,
			"modelName": modelName,
		})
	}

	// 构建生成选项
	options := s.buildGenerateOptions(req.Options)

	// 调用 Genkit 生成响应
	result, err := s.client.Generate(sessionCtx, tenantID, modelName, req.Message, options)
	if err != nil {
		// 检查是否是上下文取消错误
		if sessionCtx.Err() == context.Canceled {
			s.logger.WarnContext(ctx, "对话请求被取消", logger.Fields{
				"sessionId": sessionID,
				"tenantId":  tenantID,
				"modelName": modelName,
				"error":     err.Error(),
			})
			return nil, errors.NewContextCancelledError()
		}

		s.logger.ErrorContext(ctx, "AI 生成失败", logger.Fields{
			"sessionId": sessionID,
			"tenantId":  tenantID,
			"modelName": modelName,
			"message":   req.Message,
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

	// 记录成功日志（包含详细的 Token 使用统计）
	duration := time.Since(startTime)
	logFields := logger.Fields{
		"sessionId": sessionID,
		"tenantId":  tenantID,
		"modelName": modelName,
		"model":     result.Model,
		"duration":  duration.String(),
	}

	// 添加 Token 使用统计
	if response.Usage != nil {
		logFields["promptTokens"] = response.Usage.PromptTokens
		logFields["completionTokens"] = response.Usage.CompletionTokens
		logFields["totalTokens"] = response.Usage.TotalTokens
	}

	s.logger.InfoContext(sessionCtx, "对话请求处理完成", logFields)

	return response, nil
}

// ChatStream 流式对话（腾讯云格式）
func (s *genkitService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan *model.TencentCloudStreamMessage, error) {
	startTime := time.Now()

	// 创建或获取会话
	var sessionID string
	var sessionCtx context.Context

	if req.MessageID != "" {
		// 使用现有会话（通过消息ID）
		sessionID = req.MessageID
		existingCtx, exists := s.contextManager.GetSession(sessionID)
		if !exists {
			s.logger.WarnContext(ctx, "会话不存在，创建新会话", logger.Fields{
				"requestedMessageId": req.MessageID,
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
	s.logger.InfoContext(sessionCtx, "开始处理流式对话请求", logger.Fields{
		"sessionId": sessionID,
		"message":   req.Message,
	})

	// 从上下文中获取租户ID
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		s.logger.ErrorContext(ctx, "无法从上下文获取JWT Claims", logger.Fields{
			"sessionId": sessionID,
		})
		return nil, errors.NewUnauthorizedError("身份认证信息缺失")
	}

	tenantID := claims.TenantID
	if tenantID == "" {
		s.logger.ErrorContext(ctx, "JWT Claims 中缺少租户ID", logger.Fields{
			"sessionId": sessionID,
			"userId":    claims.Subject,
		})
		return nil, errors.NewUnauthorizedError("租户信息缺失")
	}

	// 记录租户ID
	s.logger.DebugContext(ctx, "从上下文获取租户ID", logger.Fields{
		"sessionId": sessionID,
		"tenantId":  tenantID,
		"userId":    claims.Subject,
	})

	// 从请求中获取模型名称
	// 如果请求中指定了模型名称，使用它；否则使用默认模型
	modelName := "gemini-pro" // 默认模型
	if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
		modelName = *req.Options.ModelName
		s.logger.DebugContext(ctx, "使用请求中指定的模型", logger.Fields{
			"sessionId": sessionID,
			"modelName": modelName,
		})
	} else {
		s.logger.DebugContext(ctx, "使用默认模型", logger.Fields{
			"sessionId": sessionID,
			"modelName": modelName,
		})
	}

	// 构建生成选项
	options := s.buildGenerateOptions(req.Options)

	// 调用 Genkit 流式生成
	genkitStream, err := s.client.GenerateStream(sessionCtx, tenantID, modelName, req.Message, options)
	if err != nil {
		s.logger.ErrorContext(ctx, "启动流式生成失败", logger.Fields{
			"sessionId": sessionID,
			"tenantId":  tenantID,
			"modelName": modelName,
			"message":   req.Message,
			"error":     err.Error(),
		})
		return nil, errors.NewAIServiceError(err)
	}

	// 创建输出通道
	outputChan := make(chan *model.TencentCloudStreamMessage, 10)

	// 在 goroutine 中转换流式数据为腾讯云格式
	go func() {
		defer close(outputChan)

		var fullContent string
		var lastModel string
		var lastUsage *model.Usage

		for chunk := range genkitStream {
			// 检查是否有错误
			if chunk.Error != nil {
				// 检查是否是上下文取消错误
				if sessionCtx.Err() == context.Canceled {
					s.logger.WarnContext(ctx, "流式对话请求被取消", logger.Fields{
						"sessionId":  sessionID,
						"tenantId":   tenantID,
						"modelName":  modelName,
						"chunkCount": len(fullContent),
					})
				} else {
					s.logger.ErrorContext(ctx, "流式生成出错", logger.Fields{
						"sessionId":  sessionID,
						"tenantId":   tenantID,
						"modelName":  modelName,
						"chunkCount": len(fullContent),
						"error":      chunk.Error.Error(),
						"errorType":  fmt.Sprintf("%T", chunk.Error),
					})
				}

				// 解析错误信息，提取错误代码和消息
				errorMsg := chunk.Error.Error()
				errorCode := "AI_SERVICE_ERROR"

				// 尝试识别常见错误类型
				if sessionCtx.Err() == context.Canceled {
					errorCode = "REQUEST_CANCELLED"
					errorMsg = "请求已被取消"
				} else if contains(errorMsg, "overloaded") || contains(errorMsg, "503") {
					errorCode = "SERVICE_OVERLOADED"
				} else if contains(errorMsg, "timeout") || contains(errorMsg, "deadline") {
					errorCode = "REQUEST_TIMEOUT"
				} else if contains(errorMsg, "rate limit") || contains(errorMsg, "429") {
					errorCode = "RATE_LIMIT_EXCEEDED"
				} else if contains(errorMsg, "401") || contains(errorMsg, "unauthorized") {
					errorCode = "UNAUTHORIZED"
				}

				// 发送错误消息（腾讯云格式 - 使用 tool_call_error stage）
				outputChan <- &model.TencentCloudStreamMessage{
					CompletionID: sessionID,
					SessionID:    sessionID,
					Processes: model.ProcessInfo{
						Stage:   model.StreamStageOutput,
						Message: chunk.Error.Error(),
						Detail: &model.ToolCallDetail{
							ToolName: "ai_generation",
							ToolID:   sessionID,
							Error: &model.ToolCallError{
								Code:    errorCode,
								Message: errorMsg,
							},
						},
					},
					FinishReason: "error",
					IsStop:       true,
				}
				return
			}

			// 如果是完成块
			if chunk.Done {
				// 保存模型和使用信息
				if chunk.Model != "" {
					lastModel = chunk.Model
				}
				if chunk.Usage != nil {
					lastUsage = &model.Usage{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
					}
				}

				// 发送finish事件（腾讯云格式）
				outputChan <- &model.TencentCloudStreamMessage{
					CompletionID: sessionID,
					SessionID:    sessionID,
					Processes: model.ProcessInfo{
						Stage:   model.StreamStageOutput,
						Message: "",
					},
					Content:      fullContent,
					FinishReason: "stop",
					IsStop:       true,
					AnswerSource: "ai-model",
					Usage:        lastUsage,
				}

				// 记录完成日志（包含详细的 Token 使用统计）
				duration := time.Since(startTime)
				logFields := logger.Fields{
					"sessionId": sessionID,
					"tenantId":  tenantID,
					"modelName": modelName,
					"model":     lastModel,
					"duration":  duration.String(),
				}

				// 添加 Token 使用统计
				if lastUsage != nil {
					logFields["promptTokens"] = lastUsage.PromptTokens
					logFields["completionTokens"] = lastUsage.CompletionTokens
					logFields["totalTokens"] = lastUsage.TotalTokens
				}

				s.logger.InfoContext(sessionCtx, "流式对话请求处理完成", logFields)
				break
			}

			// 跳过空内容
			if chunk.Content == "" {
				continue
			}

			// 累积内容
			fullContent += chunk.Content

			// 发送增量内容（腾讯云格式）
			outputChan <- &model.TencentCloudStreamMessage{
				CompletionID: sessionID,
				SessionID:    sessionID,
				Processes: model.ProcessInfo{
					Stage:        model.StreamStageOutput,
					Message:      "",
					DeltaContent: chunk.Content, // 同时填充 processes.delta_content 字段
				},
				DeltaContent: chunk.Content,
				IsStop:       false,
			}
		}
	}()

	return outputChan, nil
}

// AbortChat 中止对话
func (s *genkitService) AbortChat(ctx context.Context, messageID string) error {
	s.logger.InfoContext(ctx, "尝试中止对话", logger.Fields{
		"messageId": messageID,
	})

	// 检查会话是否存在
	sessionCtx, exists := s.contextManager.GetSession(messageID)
	if !exists {
		// 会话不存在，视为幂等操作，直接返回成功
		s.logger.InfoContext(ctx, "会话不存在，无需中止", logger.Fields{
			"messageId": messageID,
		})
		return nil
	}

	// 检查会话是否已经完成
	if sessionCtx.Err() != nil {
		// 会话已完成或已取消，视为幂等操作，直接返回成功
		s.logger.InfoContext(ctx, "会话已完成或已取消，无需中止", logger.Fields{
			"messageId": messageID,
		})
		return nil
	}

	// 取消会话
	err := s.contextManager.CancelSession(messageID)
	if err != nil {
		s.logger.ErrorContext(ctx, "取消会话失败", logger.Fields{
			"messageId": messageID,
			"error":     err.Error(),
		})
		return errors.NewInternalError(err)
	}

	s.logger.InfoContext(ctx, "会话已成功中止", logger.Fields{
		"messageId": messageID,
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

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
