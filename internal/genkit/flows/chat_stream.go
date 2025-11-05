// Package flows 实现流式对话相关的 Genkit Flow
package flows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
)

// RegisterChatStreamFlows 注册流式对话相关的 Flow
func RegisterChatStreamFlows(g *genkit.Genkit, services *ChatFlowServices) {
	// 注册 chatStreamFlow
	genkit.DefineFlow(
		g,
		"chatStreamFlow",
		func(ctx context.Context, input ChatStreamInput) (ChatStreamOutput, error) {
			return executeChatStreamFlow(ctx, g, input, services)
		},
	)
}

// executeChatStreamFlow 执行流式对话生成 Flow
func executeChatStreamFlow(
	ctx context.Context,
	g *genkit.Genkit,
	input ChatStreamInput,
	services *ChatFlowServices,
) (ChatStreamOutput, error) {
	startTime := time.Now()

	// 1. 参数验证
	if err := validateChatStreamInput(input); err != nil {
		return ChatStreamOutput{}, fmt.Errorf("参数验证失败: %w", err)
	}

	// 2. 权限验证
	if err := validateSessionAccess(ctx, input.SessionID, services); err != nil {
		return ChatStreamOutput{}, fmt.Errorf("权限验证失败: %w", err)
	}

	// 3. 构建上下文（如果未提供）
	var contextResult *ContextBuildOutput
	var err error

	if input.Context == nil {
		services.Logger.InfoContext(ctx, "未提供上下文，自动构建", logger.Fields{
			"sessionId": input.SessionID,
		})

		contextReq := &service.BuildContextRequest{
			SessionID:       input.SessionID,
			UserQuery:       input.UserMessage,
			MaxTokens:       4000,
			Strategy:        "auto",
			IncludeSummary:  true,
			IncludeLongTerm: true,
			ShortTermWindow: 10,
		}

		contextSvcResult, err := services.ContextService.BuildContext(ctx, contextReq)
		if err != nil {
			services.Logger.ErrorContext(ctx, "构建上下文失败", logger.Fields{
				"sessionId": input.SessionID,
				"error":     err.Error(),
			})
			return ChatStreamOutput{}, fmt.Errorf("构建上下文失败: %w", err)
		}

		contextResult = &ContextBuildOutput{
			SessionID:         contextSvcResult.SessionID,
			ShortTermMessages: []MessageContext{},
			TotalTokens:       contextSvcResult.TotalTokens,
			Strategy:          contextSvcResult.Strategy,
			QualityScore:      contextSvcResult.QualityScore,
			BuildTime:         0,
		}
	} else {
		contextResult = input.Context
	}

	// 4. 构建提示词
	prompt := buildPrompt(ChatGenerateInput{
		SessionID:    input.SessionID,
		UserMessage:  input.UserMessage,
		SystemPrompt: input.SystemPrompt,
	}, contextResult)

	services.Logger.InfoContext(ctx, "开始流式生成 AI 响应", logger.Fields{
		"sessionId":                 input.SessionID,
		"contextTokens":             contextResult.TotalTokens,
		"promptLength":              len(prompt),
		"includeTokenStats":         input.IncludeTokenStats,
		"includeIntermediateStates": input.IncludeIntermediateStates,
	})

	// 5. 初始化流式统计
	streamStats := &StreamStats{
		TotalChunks:       0,
		FirstByteTime:     0,
		AverageChunkDelay: 0,
		TotalStreamTime:   0,
	}

	// 6. 创建流式缓冲区
	streamBuffer := newStreamBuffer(input.BufferSize, input.SendInterval)

	// 7. 发送开始块
	startChunk := StreamChunk{
		Type:      "start",
		Timestamp: time.Now().Format(time.RFC3339),
		ChunkID:   0,
		Metadata: map[string]interface{}{
			"sessionId":     input.SessionID,
			"contextTokens": contextResult.TotalTokens,
			"strategy":      contextResult.Strategy,
		},
	}

	if err := sendStreamChunk(ctx, startChunk, services); err != nil {
		services.Logger.WarnContext(ctx, "发送开始块失败", logger.Fields{
			"sessionId": input.SessionID,
			"error":     err.Error(),
		})
	}

	// 8. 调用 Genkit Generate API 进行流式生成
	var fullResponse strings.Builder
	var response *ai.ModelResponse
	chunkID := 1
	firstByteReceived := false
	var firstByteTime time.Time

	// 使用 Genkit 的流式 API
	response, err = genkit.Generate(ctx, g, ai.WithPrompt(prompt))
	if err != nil {
		// 发送错误块
		errorChunk := StreamChunk{
			Type:      "error",
			Timestamp: time.Now().Format(time.RFC3339),
			ChunkID:   chunkID,
			Error: &StreamError{
				Code:        "generation_failed",
				Message:     "AI 生成失败",
				Details:     err.Error(),
				Recoverable: true,
			},
		}

		sendStreamChunk(ctx, errorChunk, services)

		services.Logger.ErrorContext(ctx, "流式生成失败", logger.Fields{
			"sessionId": input.SessionID,
			"error":     err.Error(),
		})

		return ChatStreamOutput{}, fmt.Errorf("流式生成失败: %w", err)
	}

	// 9. 处理响应内容（模拟流式输出）
	// 注意：Genkit Go SDK 可能不直接支持流式输出，这里我们模拟流式行为
	responseText := response.Text()
	fullResponse.WriteString(responseText)

	// 将响应分块发送
	chunks := splitIntoChunks(responseText, input.BufferSize)

	for i, chunk := range chunks {
		if !firstByteReceived {
			firstByteTime = time.Now()
			streamStats.FirstByteTime = firstByteTime.Sub(startTime).Milliseconds()
			firstByteReceived = true
		}

		// 构建内容块
		contentChunk := StreamChunk{
			Type:      "content",
			Content:   chunk,
			Timestamp: time.Now().Format(time.RFC3339),
			ChunkID:   chunkID,
		}

		// 添加中间状态（如果启用）
		if input.IncludeIntermediateStates {
			progress := float64(i+1) / float64(len(chunks))
			contentChunk.State = &IntermediateState{
				CurrentTokens:   estimateTokens(fullResponse.String()[:len(chunk)*(i+1)]),
				EstimatedTotal:  estimateTokens(responseText),
				Progress:        progress,
				ProcessingStage: "generating",
			}
		}

		// 发送内容块
		if err := sendStreamChunk(ctx, contentChunk, services); err != nil {
			services.Logger.WarnContext(ctx, "发送内容块失败", logger.Fields{
				"sessionId": input.SessionID,
				"chunkId":   chunkID,
				"error":     err.Error(),
			})
		}

		streamStats.TotalChunks++
		chunkID++

		// 模拟发送间隔
		if input.SendInterval > 0 && i < len(chunks)-1 {
			time.Sleep(time.Duration(input.SendInterval) * time.Millisecond)
		}
	}

	// 10. 记录 Token 使用情况
	tokenUsage := TokenUsage{
		PromptTokens:     int(response.Usage.InputTokens),
		CompletionTokens: int(response.Usage.OutputTokens),
		TotalTokens:      int(response.Usage.TotalTokens),
	}

	// 11. 发送 Token 统计块（如果启用）
	if input.IncludeTokenStats {
		tokenStatsChunk := StreamChunk{
			Type:       "token_stats",
			TokenStats: &tokenUsage,
			Timestamp:  time.Now().Format(time.RFC3339),
			ChunkID:    chunkID,
		}

		if err := sendStreamChunk(ctx, tokenStatsChunk, services); err != nil {
			services.Logger.WarnContext(ctx, "发送Token统计块失败", logger.Fields{
				"sessionId": input.SessionID,
				"error":     err.Error(),
			})
		}

		chunkID++
	}

	// 12. 计算流式统计
	streamStats.TotalStreamTime = time.Since(startTime).Milliseconds()
	if streamStats.TotalChunks > 0 {
		streamStats.AverageChunkDelay = streamStats.TotalStreamTime / int64(streamStats.TotalChunks)
	}

	// 13. 发送结束块
	endChunk := StreamChunk{
		Type:      "end",
		Timestamp: time.Now().Format(time.RFC3339),
		ChunkID:   chunkID,
		Metadata: map[string]interface{}{
			"totalChunks":    streamStats.TotalChunks,
			"totalTokens":    tokenUsage.TotalTokens,
			"streamTime":     streamStats.TotalStreamTime,
			"firstByteTime":  streamStats.FirstByteTime,
		},
	}

	if err := sendStreamChunk(ctx, endChunk, services); err != nil {
		services.Logger.WarnContext(ctx, "发送结束块失败", logger.Fields{
			"sessionId": input.SessionID,
			"error":     err.Error(),
		})
	}

	services.Logger.InfoContext(ctx, "流式响应生成完成", logger.Fields{
		"sessionId":        input.SessionID,
		"totalChunks":      streamStats.TotalChunks,
		"responseLength":   len(fullResponse.String()),
		"firstByteTime":    streamStats.FirstByteTime,
		"totalStreamTime":  streamStats.TotalStreamTime,
		"promptTokens":     tokenUsage.PromptTokens,
		"completionTokens": tokenUsage.CompletionTokens,
	})

	// 14. 保存消息（如果需要）
	messageID := uuid.New().String()
	if input.SaveMessage {
		go func() {
			saveCtx := context.Background()
			if err := saveMessages(saveCtx, ChatGenerateInput{
				SessionID:   input.SessionID,
				UserMessage: input.UserMessage,
			}, fullResponse.String(), messageID, services); err != nil {
				services.Logger.ErrorContext(saveCtx, "保存消息失败", logger.Fields{
					"sessionId": input.SessionID,
					"messageId": messageID,
					"error":     err.Error(),
				})
			}
		}()
	}

	// 15. 异步生成向量
	if input.SaveMessage {
		go func() {
			vectorCtx := context.Background()
			if err := generateVectorsAsync(vectorCtx, ChatGenerateInput{
				SessionID:   input.SessionID,
				UserMessage: input.UserMessage,
			}, fullResponse.String(), messageID, services); err != nil {
				services.Logger.ErrorContext(vectorCtx, "生成向量失败", logger.Fields{
					"sessionId": input.SessionID,
					"messageId": messageID,
					"error":     err.Error(),
				})
			}
		}()
	}

	// 16. 构建输出
	finishReason := "stop"
	if response.FinishReason != "" {
		finishReason = string(response.FinishReason)
	}

	output := ChatStreamOutput{
		MessageID:      messageID,
		Response:       fullResponse.String(),
		TokenUsage:     tokenUsage,
		FinishReason:   finishReason,
		Model:          getModelName(input.ModelConfig),
		GenerationTime: streamStats.TotalStreamTime,
		ContextInfo: ContextInfo{
			ContextTokens: contextResult.TotalTokens,
			Strategy:      contextResult.Strategy,
			QualityScore:  contextResult.QualityScore,
		},
		StreamStats: *streamStats,
	}

	return output, nil
}

// validateChatStreamInput 验证流式对话输入参数
func validateChatStreamInput(input ChatStreamInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("sessionId 不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("sessionId 格式无效: %w", err)
	}

	if input.UserMessage == "" {
		return fmt.Errorf("userMessage 不能为空")
	}

	if len(input.UserMessage) > 4000 {
		return fmt.Errorf("userMessage 长度不能超过 4000 字符")
	}

	if input.SystemPrompt != "" && len(input.SystemPrompt) > 1000 {
		return fmt.Errorf("systemPrompt 长度不能超过 1000 字符")
	}

	// 设置默认值
	if input.BufferSize == 0 {
		input.BufferSize = 10 // 默认每10个字符发送一次
	}

	if input.SendInterval == 0 {
		input.SendInterval = 50 // 默认50毫秒间隔
	}

	if input.BufferSize < 1 || input.BufferSize > 100 {
		return fmt.Errorf("bufferSize 必须在 1-100 之间")
	}

	if input.SendInterval < 10 || input.SendInterval > 1000 {
		return fmt.Errorf("sendInterval 必须在 10-1000 毫秒之间")
	}

	return nil
}

// streamBuffer 流式缓冲区
type streamBuffer struct {
	buffer       strings.Builder
	bufferSize   int
	sendInterval time.Duration
	lastSendTime time.Time
}

// newStreamBuffer 创建新的流式缓冲区
func newStreamBuffer(bufferSize int, sendIntervalMs int) *streamBuffer {
	return &streamBuffer{
		bufferSize:   bufferSize,
		sendInterval: time.Duration(sendIntervalMs) * time.Millisecond,
		lastSendTime: time.Now(),
	}
}

// add 添加内容到缓冲区
func (sb *streamBuffer) add(content string) {
	sb.buffer.WriteString(content)
}

// shouldFlush 判断是否应该刷新缓冲区
func (sb *streamBuffer) shouldFlush() bool {
	// 检查缓冲区大小
	if sb.buffer.Len() >= sb.bufferSize {
		return true
	}

	// 检查时间间隔
	if time.Since(sb.lastSendTime) >= sb.sendInterval {
		return true
	}

	return false
}

// flush 刷新缓冲区并返回内容
func (sb *streamBuffer) flush() string {
	content := sb.buffer.String()
	sb.buffer.Reset()
	sb.lastSendTime = time.Now()
	return content
}

// sendStreamChunk 发送流式块
func sendStreamChunk(ctx context.Context, chunk StreamChunk, services *ChatFlowServices) error {
	// 这里应该通过 WebSocket 或 SSE 发送块
	// 由于 Genkit 的限制，我们只记录日志
	services.Logger.InfoContext(ctx, "发送流式块", logger.Fields{
		"type":    chunk.Type,
		"chunkId": chunk.ChunkID,
	})

	// TODO: 实际实现应该通过 WebSocket 或 SSE 发送
	// 例如：websocket.WriteJSON(chunk) 或 sse.Send(chunk)

	return nil
}

// splitIntoChunks 将文本分割成块
func splitIntoChunks(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 10
	}

	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}

	return chunks
}

// estimateTokens 估算文本的 Token 数量
func estimateTokens(text string) int {
	// 简单估算：中文约 1.5 字符/token，英文约 4 字符/token
	// 这里使用平均值：约 2.5 字符/token
	return len([]rune(text)) * 10 / 25
}
