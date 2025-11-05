// Package flows 实现批量对话处理流程
package flows

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
)

// RegisterBatchConversationFlow 注册批量对话处理Flow
func RegisterBatchConversationFlow(
	g *genkit.Genkit,
	contextSvc service.ContextService,
	chatSvc service.ChatService,
	memorySvc service.MemoryService,
	summarySvc service.SummaryService,
) {
	genkit.DefineFlow(
		g,
		"batchConversationFlow",
		func(ctx context.Context, input BatchConversationInput) (BatchConversationOutput, error) {
			startTime := time.Now()

			logger.InfoContext(ctx, "开始批量对话处理",
				"total_requests", len(input.Requests),
				"max_concurrency", input.MaxConcurrency,
				"timeout", input.Timeout,
				"failure_strategy", input.FailureStrategy,
			)

			// 1. 参数验证
			if err := validateBatchConversationInput(input); err != nil {
				return BatchConversationOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 按优先级排序请求
			sortedRequests := sortRequestsByPriority(input.Requests)

			// 3. 创建超时上下文
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(input.Timeout)*time.Millisecond)
			defer cancel()

			// 4. 并发处理请求
			successResponses, failureResponses, aborted, abortReason := processBatchRequests(
				timeoutCtx,
				sortedRequests,
				input.MaxConcurrency,
				input.FailureStrategy,
				contextSvc,
				chatSvc,
				memorySvc,
				summarySvc,
				input.SaveMemory,
				input.AutoGenerateSummary,
			)

			// 5. 计算统计信息
			totalTime := time.Since(startTime).Milliseconds()
			stats := calculateBatchStats(
				startTime,
				successResponses,
				failureResponses,
				input.MaxConcurrency,
			)

			// 6. 构建输出
			output := BatchConversationOutput{
				TotalRequests:    len(input.Requests),
				SuccessCount:     len(successResponses),
				FailureCount:     len(failureResponses),
				SuccessResponses: successResponses,
				FailureResponses: failureResponses,
				TotalTime:        totalTime,
				AverageTime:      stats.AverageTime,
				MaxTime:          stats.MaxTime,
				MinTime:          stats.MinTime,
				Aborted:          aborted,
				AbortReason:      abortReason,
				ProcessingStats:  stats,
			}

			logger.InfoContext(ctx, "批量对话处理完成",
				"total_requests", output.TotalRequests,
				"success_count", output.SuccessCount,
				"failure_count", output.FailureCount,
				"total_time", totalTime,
				"aborted", aborted,
			)

			return output, nil
		},
	)
}

// validateBatchConversationInput 验证批量对话输入
func validateBatchConversationInput(input BatchConversationInput) error {
	if len(input.Requests) == 0 {
		return fmt.Errorf("请求列表不能为空")
	}

	if len(input.Requests) > 100 {
		return fmt.Errorf("请求数量不能超过100")
	}

	if input.MaxConcurrency < 1 || input.MaxConcurrency > 20 {
		return fmt.Errorf("最大并发数必须在1-20之间")
	}

	if input.Timeout < 1000 || input.Timeout > 300000 {
		return fmt.Errorf("超时时间必须在1000-300000毫秒之间")
	}

	if input.FailureStrategy != "continue" && input.FailureStrategy != "abort" {
		return fmt.Errorf("无效的失败策略: %s", input.FailureStrategy)
	}

	// 验证每个请求
	requestIDs := make(map[string]bool)
	for i, req := range input.Requests {
		if err := validateConversationRequest(req); err != nil {
			return fmt.Errorf("请求[%d]验证失败: %w", i, err)
		}

		// 检查请求ID是否重复
		if requestIDs[req.RequestID] {
			return fmt.Errorf("请求ID重复: %s", req.RequestID)
		}
		requestIDs[req.RequestID] = true
	}

	return nil
}

// validateConversationRequest 验证单个对话请求
func validateConversationRequest(req ConversationRequest) error {
	if req.RequestID == "" {
		return fmt.Errorf("请求ID不能为空")
	}

	if req.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(req.SessionID); err != nil {
		return fmt.Errorf("会话ID格式无效: %w", err)
	}

	if req.UserMessage == "" {
		return fmt.Errorf("用户消息不能为空")
	}

	if len(req.UserMessage) > 4000 {
		return fmt.Errorf("用户消息长度不能超过4000字符")
	}

	if req.MaxTokens < 100 || req.MaxTokens > 32000 {
		return fmt.Errorf("MaxTokens必须在100-32000之间")
	}

	if req.ContextStrategy != "" && req.ContextStrategy != "auto" &&
		req.ContextStrategy != "short" && req.ContextStrategy != "full" {
		return fmt.Errorf("无效的上下文策略: %s", req.ContextStrategy)
	}

	if req.Priority < 0 || req.Priority > 10 {
		return fmt.Errorf("优先级必须在0-10之间")
	}

	return nil
}

// sortRequestsByPriority 按优先级排序请求（优先级高的在前）
func sortRequestsByPriority(requests []ConversationRequest) []ConversationRequest {
	sorted := make([]ConversationRequest, len(requests))
	copy(sorted, requests)

	sort.Slice(sorted, func(i, j int) bool {
		// 优先级高的在前，如果优先级相同则保持原顺序
		return sorted[i].Priority > sorted[j].Priority
	})

	return sorted
}

// processBatchRequests 并发处理批量请求
func processBatchRequests(
	ctx context.Context,
	requests []ConversationRequest,
	maxConcurrency int,
	failureStrategy string,
	contextSvc service.ContextService,
	chatSvc service.ChatService,
	memorySvc service.MemoryService,
	summarySvc service.SummaryService,
	saveMemory bool,
	autoGenerateSummary bool,
) ([]ConversationResponse, []FailedConversation, bool, string) {
	var (
		successResponses []ConversationResponse
		failureResponses []FailedConversation
		mu               sync.Mutex
		wg               sync.WaitGroup
		aborted          bool
		abortReason      string
	)

	// 创建信号量控制并发数
	semaphore := make(chan struct{}, maxConcurrency)

	// 创建中止信号通道
	abortChan := make(chan struct{})

	for _, req := range requests {
		// 检查是否已中止
		select {
		case <-abortChan:
			// 已中止，跳过剩余请求
			mu.Lock()
			failureResponses = append(failureResponses, FailedConversation{
				RequestID:   req.RequestID,
				SessionID:   req.SessionID,
				UserMessage: req.UserMessage,
				Error:       "批量处理已中止",
				ErrorCode:   "BATCH_ABORTED",
				FailedAt:    time.Now().Format(time.RFC3339),
				Retryable:   true,
				Priority:    req.Priority,
			})
			mu.Unlock()
			continue
		default:
		}

		wg.Add(1)

		go func(request ConversationRequest) {
			defer wg.Done()

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				// 超时或取消
				mu.Lock()
				failureResponses = append(failureResponses, FailedConversation{
					RequestID:   request.RequestID,
					SessionID:   request.SessionID,
					UserMessage: request.UserMessage,
					Error:       "处理超时或已取消",
					ErrorCode:   "TIMEOUT_OR_CANCELLED",
					FailedAt:    time.Now().Format(time.RFC3339),
					Retryable:   true,
					Priority:    request.Priority,
				})
				mu.Unlock()
				return
			}

			// 处理单个请求
			response, err := processSingleRequest(
				ctx,
				request,
				contextSvc,
				chatSvc,
				memorySvc,
				summarySvc,
				saveMemory,
				autoGenerateSummary,
			)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				// 处理失败
				failureResponses = append(failureResponses, FailedConversation{
					RequestID:   request.RequestID,
					SessionID:   request.SessionID,
					UserMessage: request.UserMessage,
					Error:       err.Error(),
					ErrorCode:   "PROCESSING_ERROR",
					FailedAt:    time.Now().Format(time.RFC3339),
					Retryable:   isRetryableError(err),
					Priority:    request.Priority,
				})

				logger.ErrorContext(ctx, "批量请求处理失败",
					"request_id", request.RequestID,
					"session_id", request.SessionID,
					"error", err,
				)

				// 如果失败策略是abort，触发中止
				if failureStrategy == "abort" && !aborted {
					aborted = true
					abortReason = fmt.Sprintf("请求 %s 失败: %s", request.RequestID, err.Error())
					close(abortChan)
					logger.WarnContext(ctx, "触发批量处理中止",
						"request_id", request.RequestID,
						"abort_reason", abortReason,
					)
				}
			} else {
				// 处理成功
				successResponses = append(successResponses, *response)

				logger.InfoContext(ctx, "批量请求处理成功",
					"request_id", request.RequestID,
					"session_id", request.SessionID,
					"message_id", response.MessageID,
					"processing_time", response.ProcessingTime,
				)
			}
		}(req)
	}

	// 等待所有goroutine完成
	wg.Wait()

	return successResponses, failureResponses, aborted, abortReason
}

// processSingleRequest 处理单个对话请求
func processSingleRequest(
	ctx context.Context,
	request ConversationRequest,
	contextSvc service.ContextService,
	chatSvc service.ChatService,
	memorySvc service.MemoryService,
	summarySvc service.SummaryService,
	saveMemory bool,
	autoGenerateSummary bool,
) (*ConversationResponse, error) {
	startTime := time.Now()

	// 1. 构建上下文
	contextResult, err := contextSvc.BuildContext(ctx, &service.BuildContextRequest{
		SessionID:       request.SessionID,
		UserQuery:       request.UserMessage,
		MaxTokens:       request.MaxTokens,
		Strategy:        request.ContextStrategy,
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("构建上下文失败: %w", err)
	}

	// 2. 生成AI响应
	chatResult, err := chatSvc.GenerateResponse(ctx, service.GenerateResponseRequest{
		SessionID:    request.SessionID,
		UserMessage:  request.UserMessage,
		Context:      contextResult,
		ModelConfig:  convertModelConfig(request.ModelConfig),
		SystemPrompt: request.SystemPrompt,
		SaveMessage:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("生成AI响应失败: %w", err)
	}

	// 3. 异步存储记忆（如果启用）
	if saveMemory {
		go func() {
			asyncCtx := context.Background()
			_, err := memorySvc.StoreMemory(asyncCtx, service.StoreMemoryRequest{
				SessionID:  request.SessionID,
				MessageIDs: []string{chatResult.MessageID},
				MemoryType: "long_term",
				Importance: 0.5,
			})
			if err != nil {
				logger.ErrorContext(asyncCtx, "异步存储记忆失败",
					"request_id", request.RequestID,
					"session_id", request.SessionID,
					"message_id", chatResult.MessageID,
					"error", err,
				)
			}
		}()
	}

	// 4. 异步检查并生成摘要（如果启用）
	if autoGenerateSummary {
		go func() {
			asyncCtx := context.Background()
			triggerResult, err := summarySvc.CheckSummaryTrigger(asyncCtx, request.SessionID)
			if err != nil {
				logger.ErrorContext(asyncCtx, "检查摘要触发失败",
					"request_id", request.RequestID,
					"session_id", request.SessionID,
					"error", err,
				)
				return
			}

			if triggerResult.ShouldSummarize {
				_, err := summarySvc.GenerateSummary(asyncCtx, service.GenerateSummaryRequest{
					SessionID:    request.SessionID,
					SummaryType:  triggerResult.RecommendedType,
					TargetLength: 500,
				})
				if err != nil {
					logger.ErrorContext(asyncCtx, "异步生成摘要失败",
						"request_id", request.RequestID,
						"session_id", request.SessionID,
						"error", err,
					)
				}
			}
		}()
	}

	// 5. 构建响应
	processingTime := time.Since(startTime).Milliseconds()

	response := &ConversationResponse{
		RequestID:      request.RequestID,
		SessionID:      request.SessionID,
		MessageID:      chatResult.MessageID,
		Response:       chatResult.Response,
		TokenUsage:     chatResult.TokenUsage,
		FinishReason:   chatResult.FinishReason,
		Model:          chatResult.Model,
		ProcessingTime: processingTime,
		ContextInfo:    chatResult.ContextInfo,
		Priority:       request.Priority,
		CompletedAt:    time.Now().Format(time.RFC3339),
	}

	return response, nil
}

// calculateBatchStats 计算批量处理统计信息
func calculateBatchStats(
	startTime time.Time,
	successResponses []ConversationResponse,
	failureResponses []FailedConversation,
	maxConcurrency int,
) BatchProcessingStats {
	totalRequests := len(successResponses) + len(failureResponses)
	totalTime := time.Since(startTime).Milliseconds()

	// 计算Token统计
	totalTokens := 0
	for _, resp := range successResponses {
		totalTokens += resp.TokenUsage.TotalTokens
	}

	averageTokens := 0
	if len(successResponses) > 0 {
		averageTokens = totalTokens / len(successResponses)
	}

	// 计算时间统计
	var minTime, maxTime, totalProcessingTime int64
	if len(successResponses) > 0 {
		minTime = successResponses[0].ProcessingTime
		maxTime = successResponses[0].ProcessingTime

		for _, resp := range successResponses {
			totalProcessingTime += resp.ProcessingTime
			if resp.ProcessingTime < minTime {
				minTime = resp.ProcessingTime
			}
			if resp.ProcessingTime > maxTime {
				maxTime = resp.ProcessingTime
			}
		}
	}

	averageTime := int64(0)
	if len(successResponses) > 0 {
		averageTime = totalProcessingTime / int64(len(successResponses))
	}

	// 计算超时数量
	timeoutCount := 0
	for _, failure := range failureResponses {
		if failure.ErrorCode == "TIMEOUT_OR_CANCELLED" {
			timeoutCount++
		}
	}

	// 计算成功率
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(len(successResponses)) / float64(totalRequests)
	}

	// 计算吞吐量（请求/秒）
	throughput := 0.0
	if totalTime > 0 {
		throughput = float64(totalRequests) / (float64(totalTime) / 1000.0)
	}

	// 实际使用的并发数（估算）
	concurrencyUsed := maxConcurrency
	if totalRequests < maxConcurrency {
		concurrencyUsed = totalRequests
	}

	return BatchProcessingStats{
		StartTime:               startTime.Format(time.RFC3339),
		EndTime:                 time.Now().Format(time.RFC3339),
		TotalTokensUsed:         totalTokens,
		AverageTokensPerRequest: averageTokens,
		ConcurrencyUsed:         concurrencyUsed,
		TimeoutCount:            timeoutCount,
		SuccessRate:             successRate,
		ThroughputPerSecond:     throughput,
	}
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 可重试的错误类型
	retryableErrors := []string{
		"timeout",
		"超时",
		"connection",
		"连接",
		"temporary",
		"暂时",
		"unavailable",
		"不可用",
		"rate limit",
		"限流",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	// 简单的字符串包含检查
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
