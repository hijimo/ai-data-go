package session

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/pkg/errors"

	"gorm.io/gorm"
)

// MessageService 消息业务逻辑接口
type MessageService interface {
	// SendMessage 发送消息（包含AI回复）
	SendMessage(ctx context.Context, req *SendMessageRequest) (*MessageResponse, error)

	// GetMessages 获取消息历史
	GetMessages(ctx context.Context, req *GetMessagesRequest) (*MessageListResponse, error)

	// GetMessageByID 获取单条消息
	GetMessageByID(ctx context.Context, messageID, userID string) (*MessageDetailResponse, error)

	// AbortMessage 中止消息生成
	AbortMessage(ctx context.Context, messageID, userID string) error
}

// messageService 消息服务实现
type messageService struct {
	db                *gorm.DB
	sessionRepo       repository.SessionRepository
	messageRepo       repository.MessageRepository
	aiService         ai.AIService
	logger            logger.Logger
}

// logInfo 安全地记录信息日志
func (s *messageService) logInfo(ctx context.Context, msg string, fields logger.Fields) {
	if s.logger != nil {
		s.logger.InfoContext(ctx, msg, fields)
	}
}

// logWarn 安全地记录警告日志
func (s *messageService) logWarn(ctx context.Context, msg string, fields logger.Fields) {
	if s.logger != nil {
		s.logger.WarnContext(ctx, msg, fields)
	}
}

// logError 安全地记录错误日志
func (s *messageService) logError(ctx context.Context, msg string, fields logger.Fields) {
	if s.logger != nil {
		s.logger.ErrorContext(ctx, msg, fields)
	}
}

// NewMessageService 创建消息服务实例
func NewMessageService(
	db *gorm.DB,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	aiService ai.AIService,
	log logger.Logger,
) MessageService {
	return &messageService{
		db:          db,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		aiService:   aiService,
		logger:      log,
	}
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	SessionID string              `json:"sessionId" validate:"required,uuid"`
	Message   string              `json:"message" validate:"required"`
	UserID    string              `json:"userId" validate:"required,uuid"`
	Options   *model.ChatOptions  `json:"options,omitempty"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	MessageID   string       `json:"messageId"`
	SessionID   string       `json:"sessionId"`
	UserMessage *Message     `json:"userMessage"`
	AIMessage   *Message     `json:"aiMessage"`
	Model       string       `json:"model"`
	Usage       *model.Usage `json:"usage,omitempty"`
}

// Message 消息
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Sequence  int       `json:"sequence"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetMessagesRequest 获取消息历史请求
type GetMessagesRequest struct {
	SessionID string `json:"sessionId" validate:"required,uuid"`
	UserID    string `json:"userId" validate:"required,uuid"`
	PageNo    int    `json:"pageNo" validate:"required,min=1"`
	PageSize  int    `json:"pageSize" validate:"required,min=1,max=100"`
}

// MessageListResponse 消息列表响应
type MessageListResponse struct {
	Messages   []*MessageDetailResponse `json:"messages"`
	PageNo     int                      `json:"pageNo"`
	PageSize   int                      `json:"pageSize"`
	TotalCount int                      `json:"totalCount"`
	TotalPage  int                      `json:"totalPage"`
}

// MessageDetailResponse 消息详情响应
type MessageDetailResponse struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"sessionId"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Tokens    int                    `json:"tokens"`
	Sequence  int                    `json:"sequence"`
	CreatedAt time.Time              `json:"createdAt"`
	ToolCalls map[string]interface{} `json:"toolCalls,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// SendMessage 发送消息
func (s *messageService) SendMessage(ctx context.Context, req *SendMessageRequest) (*MessageResponse, error) {
	s.logInfo(ctx, "开始发送消息", logger.Fields{
		"sessionId": req.SessionID,
		"userId":    req.UserID,
	})

	// 1. 验证会话存在且属于用户
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewSessionNotFoundError(req.SessionID)
		}
		s.logError(ctx, "获取会话失败", logger.Fields{
			"sessionId": req.SessionID,
			"error":     err.Error(),
		})
		return nil, errors.NewInternalError(err)
	}

	// 验证会话所有权
	if session.UserID != req.UserID {
		s.logWarn(ctx, "用户尝试访问其他用户的会话", logger.Fields{
			"sessionId":    req.SessionID,
			"userId":       req.UserID,
			"sessionOwner": session.UserID,
		})
		return nil, errors.NewSessionAccessDeniedError()
	}

	// 2. 开始数据库事务
	var userMessage *model.ChatMessage
	var aiMessage *model.ChatMessage
	var aiResponse *model.ChatResponse

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 2.1 获取下一个序列号
		nextSeq, err := s.messageRepo.GetNextSequence(ctx, req.SessionID)
		if err != nil {
			return fmt.Errorf("获取消息序列号失败: %w", err)
		}

		// 2.2 保存用户消息
		userMessage = &model.ChatMessage{
			SessionID: req.SessionID,
			Role:      "user",
			Content:   req.Message,
			Sequence:  nextSeq,
			CreatedAt: time.Now(),
		}

		if err := s.messageRepo.Create(ctx, userMessage); err != nil {
			return fmt.Errorf("保存用户消息失败: %w", err)
		}

		s.logInfo(ctx, "用户消息已保存", logger.Fields{
			"messageId": userMessage.ID,
			"sequence":  userMessage.Sequence,
		})

		// 2.3 调用 AI 服务生成回复
		chatReq := &model.ChatRequest{
			Message:   req.Message,
			MessageID: req.SessionID, // 使用会话ID作为消息ID传递给AI服务
			Options:   req.Options,
		}

		aiResponse, err = s.aiService.Chat(ctx, chatReq)
		if err != nil {
			// AI 服务失败，保存错误信息到用户消息
			userMessage.Error = err.Error()
			if updateErr := s.messageRepo.Create(ctx, userMessage); updateErr != nil {
				s.logError(ctx, "更新消息错误信息失败", logger.Fields{
					"messageId": userMessage.ID,
					"error":     updateErr.Error(),
				})
			}
			return errors.NewMessageSendFailedError(err)
		}

		// 2.4 保存 AI 回复消息
		aiMessage = &model.ChatMessage{
			SessionID: req.SessionID,
			Role:      "assistant",
			Content:   aiResponse.Message,
			Sequence:  nextSeq + 1,
			CreatedAt: time.Now(),
		}

		// 如果有 token 使用信息，保存到消息中
		if aiResponse.Usage != nil {
			aiMessage.Tokens = aiResponse.Usage.CompletionTokens
		}

		if err := s.messageRepo.Create(ctx, aiMessage); err != nil {
			return fmt.Errorf("保存AI消息失败: %w", err)
		}

		s.logInfo(ctx, "AI消息已保存", logger.Fields{
			"messageId": aiMessage.ID,
			"sequence":  aiMessage.Sequence,
			"tokens":    aiMessage.Tokens,
		})

		// 2.5 更新会话信息
		if err := s.sessionRepo.UpdateLastMessage(ctx, req.SessionID, aiMessage.ID); err != nil {
			return fmt.Errorf("更新会话最后消息失败: %w", err)
		}

		if err := s.sessionRepo.IncrementMessageCount(ctx, req.SessionID); err != nil {
			return fmt.Errorf("更新会话消息计数失败: %w", err)
		}
		if err := s.sessionRepo.IncrementMessageCount(ctx, req.SessionID); err != nil {
			return fmt.Errorf("更新会话消息计数失败: %w", err)
		}

		return nil
	})

	if err != nil {
		s.logError(ctx, "发送消息事务失败", logger.Fields{
			"sessionId": req.SessionID,
			"error":     err.Error(),
		})
		// 如果是已知的应用错误，直接返回
		if appErr, ok := err.(*errors.AppError); ok {
			return nil, appErr
		}
		return nil, errors.NewInternalError(err)
	}

	// 3. 构建响应
	response := &MessageResponse{
		MessageID: aiMessage.ID,
		SessionID: req.SessionID,
		UserMessage: &Message{
			ID:        userMessage.ID,
			Role:      userMessage.Role,
			Content:   userMessage.Content,
			Sequence:  userMessage.Sequence,
			CreatedAt: userMessage.CreatedAt,
		},
		AIMessage: &Message{
			ID:        aiMessage.ID,
			Role:      aiMessage.Role,
			Content:   aiMessage.Content,
			Sequence:  aiMessage.Sequence,
			CreatedAt: aiMessage.CreatedAt,
		},
		Model: aiResponse.Model,
		Usage: aiResponse.Usage,
	}

	s.logInfo(ctx, "消息发送成功", logger.Fields{
		"sessionId": req.SessionID,
		"userMsgId": userMessage.ID,
		"aiMsgId":   aiMessage.ID,
		"model":     aiResponse.Model,
	})

	return response, nil
}

// GetMessages 获取消息历史
func (s *messageService) GetMessages(ctx context.Context, req *GetMessagesRequest) (*MessageListResponse, error) {
	s.logInfo(ctx, "获取消息历史", logger.Fields{
		"sessionId": req.SessionID,
		"userId":    req.UserID,
		"pageNo":    req.PageNo,
		"pageSize":  req.PageSize,
	})

	// 1. 验证会话存在且属于用户
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewSessionNotFoundError(req.SessionID)
		}
		s.logError(ctx, "获取会话失败", logger.Fields{
			"sessionId": req.SessionID,
			"error":     err.Error(),
		})
		return nil, errors.NewInternalError(err)
	}

	// 验证会话所有权
	if session.UserID != req.UserID {
		s.logWarn(ctx, "用户尝试访问其他用户的会话消息", logger.Fields{
			"sessionId":    req.SessionID,
			"userId":       req.UserID,
			"sessionOwner": session.UserID,
		})
		return nil, errors.NewSessionAccessDeniedError()
	}

	// 2. 查询消息列表
	messages, totalCount, err := s.messageRepo.GetBySessionID(ctx, req.SessionID, req.PageNo, req.PageSize)
	if err != nil {
		s.logError(ctx, "查询消息列表失败", logger.Fields{
			"sessionId": req.SessionID,
			"error":     err.Error(),
		})
		return nil, errors.NewInternalError(err)
	}

	// 3. 转换为响应格式
	messageDetails := make([]*MessageDetailResponse, 0, len(messages))
	for _, msg := range messages {
		detail := &MessageDetailResponse{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      msg.Role,
			Content:   msg.Content,
			Tokens:    msg.Tokens,
			Sequence:  msg.Sequence,
			CreatedAt: msg.CreatedAt,
			Error:     msg.Error,
		}

		// 处理 ToolCalls 和 Meta（如果有）
		if msg.ToolCalls != nil {
			detail.ToolCalls = make(map[string]interface{})
			// 这里需要根据实际的 ToolCalls 结构进行转换
		}
		if msg.Meta != nil {
			detail.Meta = make(map[string]interface{})
			// 这里需要根据实际的 Meta 结构进行转换
		}

		messageDetails = append(messageDetails, detail)
	}

	// 4. 计算总页数
	totalPage := totalCount / req.PageSize
	if totalCount%req.PageSize > 0 {
		totalPage++
	}

	response := &MessageListResponse{
		Messages:   messageDetails,
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		TotalCount: totalCount,
		TotalPage:  totalPage,
	}

	s.logInfo(ctx, "消息历史查询成功", logger.Fields{
		"sessionId":  req.SessionID,
		"totalCount": totalCount,
		"pageNo":     req.PageNo,
	})

	return response, nil
}

// GetMessageByID 获取单条消息
func (s *messageService) GetMessageByID(ctx context.Context, messageID, userID string) (*MessageDetailResponse, error) {
	s.logInfo(ctx, "获取消息详情", logger.Fields{
		"messageId": messageID,
		"userId":    userID,
	})

	// 1. 查询消息
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewMessageNotFoundError(messageID)
		}
		s.logError(ctx, "查询消息失败", logger.Fields{
			"messageId": messageID,
			"error":     err.Error(),
		})
		return nil, errors.NewInternalError(err)
	}

	// 2. 验证消息所属会话的所有权
	session, err := s.sessionRepo.GetByID(ctx, message.SessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewSessionNotFoundError(message.SessionID)
		}
		s.logError(ctx, "获取会话失败", logger.Fields{
			"sessionId": message.SessionID,
			"error":     err.Error(),
		})
		return nil, errors.NewInternalError(err)
	}

	if session.UserID != userID {
		s.logWarn(ctx, "用户尝试访问其他用户的消息", logger.Fields{
			"messageId":    messageID,
			"userId":       userID,
			"sessionOwner": session.UserID,
		})
		return nil, errors.NewMessageAccessDeniedError()
	}

	// 3. 构建响应
	response := &MessageDetailResponse{
		ID:        message.ID,
		SessionID: message.SessionID,
		Role:      message.Role,
		Content:   message.Content,
		Tokens:    message.Tokens,
		Sequence:  message.Sequence,
		CreatedAt: message.CreatedAt,
		Error:     message.Error,
	}

	// 处理 ToolCalls 和 Meta（如果有）
	if message.ToolCalls != nil {
		response.ToolCalls = make(map[string]interface{})
		// 这里需要根据实际的 ToolCalls 结构进行转换
	}
	if message.Meta != nil {
		response.Meta = make(map[string]interface{})
		// 这里需要根据实际的 Meta 结构进行转换
	}

	s.logInfo(ctx, "消息详情查询成功", logger.Fields{
		"messageId": messageID,
	})

	return response, nil
}

// AbortMessage 中止消息生成
func (s *messageService) AbortMessage(ctx context.Context, messageID, userID string) error {
	s.logInfo(ctx, "中止消息生成", logger.Fields{
		"messageId": messageID,
		"userId":    userID,
	})

	// 1. 查询消息
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 消息不存在，视为幂等操作，直接返回成功
			s.logInfo(ctx, "消息不存在，无需中止", logger.Fields{
				"messageId": messageID,
			})
			return nil
		}
		s.logError(ctx, "查询消息失败", logger.Fields{
			"messageId": messageID,
			"error":     err.Error(),
		})
		return errors.NewInternalError(err)
	}

	// 2. 验证消息所属会话的所有权
	session, err := s.sessionRepo.GetByID(ctx, message.SessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewSessionNotFoundError(message.SessionID)
		}
		s.logError(ctx, "获取会话失败", logger.Fields{
			"sessionId": message.SessionID,
			"error":     err.Error(),
		})
		return errors.NewInternalError(err)
	}

	if session.UserID != userID {
		s.logWarn(ctx, "用户尝试中止其他用户的消息", logger.Fields{
			"messageId":    messageID,
			"userId":       userID,
			"sessionOwner": session.UserID,
		})
		return errors.NewMessageAccessDeniedError()
	}

	// 3. 调用 AI 服务中止会话
	// 注意：这里使用 SessionID 而不是 MessageID，因为 AI 服务是基于会话的
	if err := s.aiService.AbortChat(ctx, message.SessionID); err != nil {
		s.logError(ctx, "中止AI会话失败", logger.Fields{
			"sessionId": message.SessionID,
			"messageId": messageID,
			"error":     err.Error(),
		})
		return errors.NewInternalError(err)
	}

	s.logInfo(ctx, "消息生成已中止", logger.Fields{
		"messageId": messageID,
		"sessionId": message.SessionID,
	})

	return nil
}
