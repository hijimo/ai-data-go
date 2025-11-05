package errors

import "fmt"

// 错误码常量定义
const (
	// CodeSuccess 成功
	CodeSuccess = 200

	// 客户端错误 4xx
	CodeBadRequest      = 400 // 请求参数错误
	CodeUnauthorized    = 401 // 未授权
	CodeForbidden       = 403 // 禁止访问
	CodeNotFound        = 404 // 资源不存在
	CodeValidationError = 422 // 参数验证失败

	// 服务器错误 5xx
	CodeInternalError      = 500 // 内部错误
	CodeServiceUnavailable = 503 // 服务不可用
	CodeAIServiceError     = 550 // AI 服务错误
	CodeContextCancelled   = 551 // 上下文已取消
	
	// 模型提供商相关错误 560-569
	CodeProviderNotFound = 560 // 提供商不存在
	CodeModelNotFound    = 561 // 模型不存在
	CodeLoadDataError    = 562 // 数据加载错误

	// 会话相关错误 570-579
	CodeSessionNotFound      = 570 // 会话不存在
	CodeSessionAccessDenied  = 571 // 无权访问会话
	CodeSessionAlreadyExists = 572 // 会话已存在

	// 消息相关错误 580-589
	CodeMessageNotFound     = 580 // 消息不存在
	CodeMessageAccessDenied = 581 // 无权访问消息
	CodeMessageSendFailed   = 582 // 消息发送失败

	// 摘要相关错误 590-599
	CodeSummaryGenerationFailed = 590 // 摘要生成失败
	CodeSummaryTriggerFailed    = 591 // 摘要触发检查失败
	CodeSummaryQualityFailed    = 592 // 摘要质量评估失败

	// 上下文管理错误 600-609
	CodeContextBuildFailed    = 600 // 上下文构建失败
	CodeContextOptimizeFailed = 601 // 上下文优化失败
	CodeTokenExceeded         = 602 // Token 超限
	CodeContextQualityLow     = 603 // 上下文质量过低

	// 记忆管理错误 610-619
	CodeMemoryNotFound            = 610 // 记忆不存在
	CodeMemorySearchFailed        = 611 // 记忆检索失败
	CodeMemoryStoreFailed         = 612 // 记忆存储失败
	CodeMemoryCleanupFailed       = 613 // 记忆清理失败
	CodeVectorGenerationFailed    = 614 // 向量生成失败
	CodeVectorSearchFailed        = 615 // 向量检索失败

	// 查询分类错误 620-629
	CodeQueryClassifyFailed = 620 // 查询分类失败

	// Token 管理错误 630-639
	CodeTokenBudgetExceeded   = 630 // Token 预算超限
	CodeTokenOptimizeFailed   = 631 // Token 优化失败
	CodeTokenAnalysisFailed   = 632 // Token 分析失败
	CodeTokenCalculationError = 633 // Token 计算错误

	// 对话生成错误 640-649
	CodeChatGenerationFailed = 640 // 对话生成失败
	CodeChatStreamFailed     = 641 // 流式对话失败
	CodeChatRetryExhausted   = 642 // 重试次数耗尽
	CodeModelConfigInvalid   = 643 // 模型配置无效

	// 批量处理错误 650-659
	CodeBatchProcessingFailed = 650 // 批量处理失败
	CodeBatchPartialFailure   = 651 // 批量部分失败

	// 健康检查错误 660-669
	CodeHealthCheckFailed = 660 // 健康检查失败
	CodeAutoFixFailed     = 661 // 自动修复失败

	// 降级和熔断错误 670-679
	CodeServiceDegraded    = 670 // 服务已降级
	CodeCircuitBreakerOpen = 671 // 熔断器已打开
	CodeFallbackFailed     = 672 // 降级失败
)

// 错误消息常量
const (
	MsgSuccess             = "成功"
	MsgBadRequest          = "请求参数错误"
	MsgUnauthorized        = "未授权"
	MsgForbidden           = "禁止访问"
	MsgNotFound            = "资源不存在"
	MsgValidationError     = "参数验证失败"
	MsgInternalError       = "内部错误"
	MsgServiceUnavailable  = "服务不可用"
	MsgAIServiceError      = "AI 服务错误"
	MsgContextCancelled    = "请求已取消"
	MsgProviderNotFound    = "提供商不存在"
	MsgModelNotFound       = "模型不存在"
	MsgLoadDataError       = "数据加载失败"
	MsgSessionNotFound             = "会话不存在"
	MsgSessionAccessDenied         = "无权访问会话"
	MsgMessageNotFound             = "消息不存在"
	MsgMessageAccessDenied         = "无权访问消息"
	MsgMessageSendFailed           = "消息发送失败"
	MsgSummaryGenerationFailed     = "摘要生成失败"
	MsgSummaryTriggerFailed        = "摘要触发检查失败"
	MsgSummaryQualityFailed        = "摘要质量评估失败"
	MsgContextBuildFailed          = "上下文构建失败"
	MsgContextOptimizeFailed       = "上下文优化失败"
	MsgTokenExceeded               = "Token 超限"
	MsgContextQualityLow           = "上下文质量过低"
	MsgMemoryNotFound              = "记忆不存在"
	MsgMemorySearchFailed          = "记忆检索失败"
	MsgMemoryStoreFailed           = "记忆存储失败"
	MsgMemoryCleanupFailed         = "记忆清理失败"
	MsgVectorGenerationFailed      = "向量生成失败"
	MsgVectorSearchFailed          = "向量检索失败"
	MsgQueryClassifyFailed         = "查询分类失败"
	MsgTokenBudgetExceeded         = "Token 预算超限"
	MsgTokenOptimizeFailed         = "Token 优化失败"
	MsgTokenAnalysisFailed         = "Token 分析失败"
	MsgTokenCalculationError       = "Token 计算错误"
	MsgChatGenerationFailed        = "对话生成失败"
	MsgChatStreamFailed            = "流式对话失败"
	MsgChatRetryExhausted          = "重试次数耗尽"
	MsgModelConfigInvalid          = "模型配置无效"
	MsgBatchProcessingFailed       = "批量处理失败"
	MsgBatchPartialFailure         = "批量部分失败"
	MsgHealthCheckFailed           = "健康检查失败"
	MsgAutoFixFailed               = "自动修复失败"
	MsgServiceDegraded             = "服务已降级"
	MsgCircuitBreakerOpen          = "熔断器已打开"
	MsgFallbackFailed              = "降级失败"
)

// AppError 自定义应用错误类型
type AppError struct {
	Code    int    // 错误码
	Message string // 错误消息
	Err     error  // 原始错误
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误，支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装现有错误
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义的错误构造函数

// NewBadRequestError 创建请求参数错误
func NewBadRequestError(message string) *AppError {
	if message == "" {
		message = MsgBadRequest
	}
	return New(CodeBadRequest, message)
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = MsgUnauthorized
	}
	return New(CodeUnauthorized, message)
}

// NewForbiddenError 创建禁止访问错误
func NewForbiddenError(message string) *AppError {
	if message == "" {
		message = MsgForbidden
	}
	return New(CodeForbidden, message)
}

// NewValidationError 创建参数验证错误
func NewValidationError(message string) *AppError {
	if message == "" {
		message = MsgValidationError
	}
	return New(CodeValidationError, message)
}

// NewNotFoundError 创建资源不存在错误
func NewNotFoundError(message string) *AppError {
	if message == "" {
		message = MsgNotFound
	}
	return New(CodeNotFound, message)
}

// NewInternalError 创建内部错误
func NewInternalError(err error) *AppError {
	return Wrap(CodeInternalError, MsgInternalError, err)
}

// NewAIServiceError 创建 AI 服务错误
func NewAIServiceError(err error) *AppError {
	return Wrap(CodeAIServiceError, MsgAIServiceError, err)
}

// NewContextCancelledError 创建上下文取消错误
func NewContextCancelledError() *AppError {
	return New(CodeContextCancelled, MsgContextCancelled)
}

// NewServiceUnavailableError 创建服务不可用错误
func NewServiceUnavailableError(message string) *AppError {
	if message == "" {
		message = MsgServiceUnavailable
	}
	return New(CodeServiceUnavailable, message)
}

// NewProviderNotFoundError 创建提供商不存在错误
func NewProviderNotFoundError(providerID string) *AppError {
	message := MsgProviderNotFound
	if providerID != "" {
		message = fmt.Sprintf("提供商 '%s' 不存在", providerID)
	}
	return New(CodeProviderNotFound, message)
}

// NewModelNotFoundError 创建模型不存在错误
func NewModelNotFoundError(modelID string) *AppError {
	message := MsgModelNotFound
	if modelID != "" {
		message = fmt.Sprintf("模型 '%s' 不存在", modelID)
	}
	return New(CodeModelNotFound, message)
}

// NewLoadDataError 创建数据加载错误
func NewLoadDataError(err error) *AppError {
	return Wrap(CodeLoadDataError, MsgLoadDataError, err)
}

// NewSessionNotFoundError 创建会话不存在错误
func NewSessionNotFoundError(sessionID string) *AppError {
	message := MsgSessionNotFound
	if sessionID != "" {
		message = fmt.Sprintf("会话 '%s' 不存在", sessionID)
	}
	return New(CodeSessionNotFound, message)
}

// NewSessionAccessDeniedError 创建会话访问拒绝错误
func NewSessionAccessDeniedError() *AppError {
	return New(CodeSessionAccessDenied, MsgSessionAccessDenied)
}

// NewMessageNotFoundError 创建消息不存在错误
func NewMessageNotFoundError(messageID string) *AppError {
	message := MsgMessageNotFound
	if messageID != "" {
		message = fmt.Sprintf("消息 '%s' 不存在", messageID)
	}
	return New(CodeMessageNotFound, message)
}

// NewMessageAccessDeniedError 创建消息访问拒绝错误
func NewMessageAccessDeniedError() *AppError {
	return New(CodeMessageAccessDenied, MsgMessageAccessDenied)
}

// NewMessageSendFailedError 创建消息发送失败错误
func NewMessageSendFailedError(err error) *AppError {
	return Wrap(CodeMessageSendFailed, MsgMessageSendFailed, err)
}

// NewSummaryGenerationFailedError 创建摘要生成失败错误
func NewSummaryGenerationFailedError(err error) *AppError {
	return Wrap(CodeSummaryGenerationFailed, MsgSummaryGenerationFailed, err)
}

// NewSummaryTriggerFailedError 创建摘要触发检查失败错误
func NewSummaryTriggerFailedError(err error) *AppError {
	return Wrap(CodeSummaryTriggerFailed, MsgSummaryTriggerFailed, err)
}

// NewSummaryQualityFailedError 创建摘要质量评估失败错误
func NewSummaryQualityFailedError(err error) *AppError {
	return Wrap(CodeSummaryQualityFailed, MsgSummaryQualityFailed, err)
}

// NewContextBuildFailedError 创建上下文构建失败错误
func NewContextBuildFailedError(err error) *AppError {
	return Wrap(CodeContextBuildFailed, MsgContextBuildFailed, err)
}

// NewContextOptimizeFailedError 创建上下文优化失败错误
func NewContextOptimizeFailedError(err error) *AppError {
	return Wrap(CodeContextOptimizeFailed, MsgContextOptimizeFailed, err)
}

// NewTokenExceededError 创建 Token 超限错误
func NewTokenExceededError(current, limit int) *AppError {
	message := fmt.Sprintf("Token 超限: 当前 %d, 限制 %d", current, limit)
	return New(CodeTokenExceeded, message)
}

// NewContextQualityLowError 创建上下文质量过低错误
func NewContextQualityLowError(score float64) *AppError {
	message := fmt.Sprintf("上下文质量过低: %.2f", score)
	return New(CodeContextQualityLow, message)
}

// NewMemoryNotFoundError 创建记忆不存在错误
func NewMemoryNotFoundError(memoryID string) *AppError {
	message := MsgMemoryNotFound
	if memoryID != "" {
		message = fmt.Sprintf("记忆 '%s' 不存在", memoryID)
	}
	return New(CodeMemoryNotFound, message)
}

// NewMemorySearchFailedError 创建记忆检索失败错误
func NewMemorySearchFailedError(err error) *AppError {
	return Wrap(CodeMemorySearchFailed, MsgMemorySearchFailed, err)
}

// NewMemoryStoreFailedError 创建记忆存储失败错误
func NewMemoryStoreFailedError(err error) *AppError {
	return Wrap(CodeMemoryStoreFailed, MsgMemoryStoreFailed, err)
}

// NewMemoryCleanupFailedError 创建记忆清理失败错误
func NewMemoryCleanupFailedError(err error) *AppError {
	return Wrap(CodeMemoryCleanupFailed, MsgMemoryCleanupFailed, err)
}

// NewVectorGenerationFailedError 创建向量生成失败错误
func NewVectorGenerationFailedError(err error) *AppError {
	return Wrap(CodeVectorGenerationFailed, MsgVectorGenerationFailed, err)
}

// NewVectorSearchFailedError 创建向量检索失败错误
func NewVectorSearchFailedError(err error) *AppError {
	return Wrap(CodeVectorSearchFailed, MsgVectorSearchFailed, err)
}

// NewQueryClassifyFailedError 创建查询分类失败错误
func NewQueryClassifyFailedError(err error) *AppError {
	return Wrap(CodeQueryClassifyFailed, MsgQueryClassifyFailed, err)
}

// NewTokenBudgetExceededError 创建 Token 预算超限错误
func NewTokenBudgetExceededError(used, budget int) *AppError {
	message := fmt.Sprintf("Token 预算超限: 已使用 %d, 预算 %d", used, budget)
	return New(CodeTokenBudgetExceeded, message)
}

// NewTokenOptimizeFailedError 创建 Token 优化失败错误
func NewTokenOptimizeFailedError(err error) *AppError {
	return Wrap(CodeTokenOptimizeFailed, MsgTokenOptimizeFailed, err)
}

// NewTokenAnalysisFailedError 创建 Token 分析失败错误
func NewTokenAnalysisFailedError(err error) *AppError {
	return Wrap(CodeTokenAnalysisFailed, MsgTokenAnalysisFailed, err)
}

// NewTokenCalculationError 创建 Token 计算错误
func NewTokenCalculationError(err error) *AppError {
	return Wrap(CodeTokenCalculationError, MsgTokenCalculationError, err)
}

// NewChatGenerationFailedError 创建对话生成失败错误
func NewChatGenerationFailedError(err error) *AppError {
	return Wrap(CodeChatGenerationFailed, MsgChatGenerationFailed, err)
}

// NewChatStreamFailedError 创建流式对话失败错误
func NewChatStreamFailedError(err error) *AppError {
	return Wrap(CodeChatStreamFailed, MsgChatStreamFailed, err)
}

// NewChatRetryExhaustedError 创建重试次数耗尽错误
func NewChatRetryExhaustedError(attempts int) *AppError {
	message := fmt.Sprintf("重试次数耗尽: 已尝试 %d 次", attempts)
	return New(CodeChatRetryExhausted, message)
}

// NewModelConfigInvalidError 创建模型配置无效错误
func NewModelConfigInvalidError(reason string) *AppError {
	message := MsgModelConfigInvalid
	if reason != "" {
		message = fmt.Sprintf("模型配置无效: %s", reason)
	}
	return New(CodeModelConfigInvalid, message)
}

// NewBatchProcessingFailedError 创建批量处理失败错误
func NewBatchProcessingFailedError(err error) *AppError {
	return Wrap(CodeBatchProcessingFailed, MsgBatchProcessingFailed, err)
}

// NewBatchPartialFailureError 创建批量部分失败错误
func NewBatchPartialFailureError(succeeded, failed int) *AppError {
	message := fmt.Sprintf("批量部分失败: 成功 %d, 失败 %d", succeeded, failed)
	return New(CodeBatchPartialFailure, message)
}

// NewHealthCheckFailedError 创建健康检查失败错误
func NewHealthCheckFailedError(err error) *AppError {
	return Wrap(CodeHealthCheckFailed, MsgHealthCheckFailed, err)
}

// NewAutoFixFailedError 创建自动修复失败错误
func NewAutoFixFailedError(err error) *AppError {
	return Wrap(CodeAutoFixFailed, MsgAutoFixFailed, err)
}

// NewServiceDegradedError 创建服务已降级错误
func NewServiceDegradedError(service string) *AppError {
	message := MsgServiceDegraded
	if service != "" {
		message = fmt.Sprintf("服务 '%s' 已降级", service)
	}
	return New(CodeServiceDegraded, message)
}

// NewCircuitBreakerOpenError 创建熔断器已打开错误
func NewCircuitBreakerOpenError(service string) *AppError {
	message := MsgCircuitBreakerOpen
	if service != "" {
		message = fmt.Sprintf("服务 '%s' 熔断器已打开", service)
	}
	return New(CodeCircuitBreakerOpen, message)
}

// NewFallbackFailedError 创建降级失败错误
func NewFallbackFailedError(err error) *AppError {
	return Wrap(CodeFallbackFailed, MsgFallbackFailed, err)
}
