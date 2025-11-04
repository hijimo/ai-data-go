// Package flows 定义 Genkit Flow 的输入输出类型
package flows

// ContextBuildInput 上下文构建输入
type ContextBuildInput struct {
	SessionID       string `json:"sessionId" validate:"required,uuid"`
	UserQuery       string `json:"userQuery" validate:"required,max=2000"`
	MaxTokens       int    `json:"maxTokens" validate:"min=100,max=32000"`
	Strategy        string `json:"strategy" validate:"oneof=auto short full"`
	IncludeSummary  bool   `json:"includeSummary"`
	IncludeLongTerm bool   `json:"includeLongTerm"`
	ShortTermWindow int    `json:"shortTermWindow" validate:"min=1,max=50"`
}

// ContextBuildOutput 上下文构建输出
type ContextBuildOutput struct {
	SessionID         string           `json:"sessionId"`
	Summary           *SummaryContext  `json:"summary,omitempty"`
	LongTermMemories  []MemoryContext  `json:"longTermMemories,omitempty"`
	ShortTermMessages []MessageContext `json:"shortTermMessages"`
	TotalTokens       int              `json:"totalTokens"`
	Strategy          string           `json:"strategy"`
	QualityScore      float64          `json:"qualityScore"`
	BuildTime         int64            `json:"buildTime"`
}

// SummaryContext 摘要上下文
type SummaryContext struct {
	Content    string `json:"content"`
	TokenCount int    `json:"tokenCount"`
	CreatedAt  string `json:"createdAt"`
	Coverage   string `json:"coverage"`
}

// MemoryContext 记忆上下文
type MemoryContext struct {
	ID         string  `json:"id"`
	Content    string  `json:"content"`
	TokenCount int     `json:"tokenCount"`
	Importance float32 `json:"importance"`
	Similarity float32 `json:"similarity"`
	CreatedAt  string  `json:"createdAt"`
}

// MessageContext 消息上下文
type MessageContext struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	TokenCount int    `json:"tokenCount"`
	CreatedAt  string `json:"createdAt"`
}

// QueryClassifyInput 查询分类输入
type QueryClassifyInput struct {
	Query          string   `json:"query" validate:"required,max=2000"`
	SessionID      string   `json:"sessionId" validate:"omitempty,uuid"`
	RecentMessages []string `json:"recentMessages" validate:"max=5"`
}

// QueryClassifyOutput 查询分类输出
type QueryClassifyOutput struct {
	QueryType           string   `json:"queryType"`
	Intent              string   `json:"intent"`
	NeedsHistory        bool     `json:"needsHistory"`
	NeedsLongTerm       bool     `json:"needsLongTerm"`
	RecommendedStrategy string   `json:"recommendedStrategy"`
	Confidence          float64  `json:"confidence"`
	Entities            []string `json:"entities"`
}

// QueryClassifyInput 查询分类输入
type QueryClassifyInput struct {
	Query     string `json:"query" validate:"required,max=2000"`
	SessionID string `json:"sessionId" validate:"omitempty,uuid"`
}

// QueryClassifyOutput 查询分类输出
type QueryClassifyOutput struct {
	QueryType           string   `json:"queryType"`
	NeedsHistory        bool     `json:"needsHistory"`
	KeyEntities         []string `json:"keyEntities"`
	RecommendedStrategy string   `json:"recommendedStrategy"`
	Confidence          float64  `json:"confidence"`
	Reasoning           string   `json:"reasoning"`
}

// ContextOptimizeInput 上下文优化输入
type ContextOptimizeInput struct {
	Context         *ContextBuildOutput `json:"context" validate:"required"`
	TargetTokens    int                 `json:"targetTokens" validate:"required,min=100,max=32000"`
	Strategy        string              `json:"strategy" validate:"required,oneof=aggressive balanced conservative"`
	PreserveSummary bool                `json:"preserveSummary"`
}

// ContextOptimizeOutput 上下文优化输出
type ContextOptimizeOutput struct {
	SessionID         string           `json:"sessionId"`
	Summary           *SummaryContext  `json:"summary,omitempty"`
	LongTermMemories  []MemoryContext  `json:"longTermMemories,omitempty"`
	ShortTermMessages []MessageContext `json:"shortTermMessages"`
	TotalTokens       int              `json:"totalTokens"`
	Strategy          string           `json:"strategy"`
	QualityScore      float64          `json:"qualityScore"`
	QualityLoss       float64          `json:"qualityLoss"`
	OptimizationTime  int64            `json:"optimizationTime"`
	Operations        []string         `json:"operations"`
}

// ChatGenerateInput 对话生成输入
type ChatGenerateInput struct {
	SessionID    string               `json:"sessionId" validate:"required,uuid"`
	UserMessage  string               `json:"userMessage" validate:"required,max=4000"`
	Context      *ContextBuildOutput  `json:"context,omitempty"`
	ModelConfig  *ModelConfig         `json:"modelConfig,omitempty"`
	SystemPrompt string               `json:"systemPrompt" validate:"max=1000"`
	SaveMessage  bool                 `json:"saveMessage"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	ModelName        string   `json:"modelName" validate:"required"`
	Temperature      float64  `json:"temperature" validate:"min=0,max=2"`
	TopP             float64  `json:"topP" validate:"min=0,max=1"`
	MaxTokens        int      `json:"maxTokens" validate:"min=1,max=4096"`
	StopSequences    []string `json:"stopSequences" validate:"max=4"`
	FrequencyPenalty float64  `json:"frequencyPenalty" validate:"min=-2,max=2"`
	PresencePenalty  float64  `json:"presencePenalty" validate:"min=-2,max=2"`
}

// ChatGenerateOutput 对话生成输出
type ChatGenerateOutput struct {
	MessageID      string      `json:"messageId"`
	Response       string      `json:"response"`
	TokenUsage     TokenUsage  `json:"tokenUsage"`
	FinishReason   string      `json:"finishReason"`
	Model          string      `json:"model"`
	GenerationTime int64       `json:"generationTime"`
	ContextInfo    ContextInfo `json:"contextInfo"`
}

// TokenUsage Token使用统计
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// ContextInfo 上下文信息
type ContextInfo struct {
	ContextTokens int     `json:"contextTokens"`
	Strategy      string  `json:"strategy"`
	QualityScore  float64 `json:"qualityScore"`
}

// MultiTurnChatInput 多轮对话管理输入
type MultiTurnChatInput struct {
	SessionID    string `json:"sessionId" validate:"required,uuid"`
	UserMessage  string `json:"userMessage" validate:"required,max=4000"`
	ResetContext bool   `json:"resetContext"`
}

// MultiTurnChatOutput 多轮对话管理输出
type MultiTurnChatOutput struct {
	SessionID      string              `json:"sessionId"`
	TurnNumber     int                 `json:"turnNumber"`
	SessionState   string              `json:"sessionState"`
	HealthScore    float64             `json:"healthScore"`
	TokenUsageRate float64             `json:"tokenUsageRate"`
	Suggestions    []string            `json:"suggestions"`
	ContextInfo    MultiTurnContextInfo `json:"contextInfo"`
	Response       string              `json:"response"`
	MessageID      string              `json:"messageId"`
}

// MultiTurnContextInfo 多轮对话上下文信息
type MultiTurnContextInfo struct {
	TotalMessages   int     `json:"totalMessages"`
	TotalTokens     int     `json:"totalTokens"`
	MaxTokens       int     `json:"maxTokens"`
	QualityScore    float64 `json:"qualityScore"`
	LastSummaryAt   string  `json:"lastSummaryAt,omitempty"`
	MessagesSinceLastSummary int `json:"messagesSinceLastSummary"`
}

// ChatRetryInput 对话重试输入
type ChatRetryInput struct {
	SessionID    string               `json:"sessionId" validate:"required,uuid"`
	UserMessage  string               `json:"userMessage" validate:"required,max=4000"`
	Context      *ContextBuildOutput  `json:"context,omitempty"`
	ModelConfig  *ModelConfig         `json:"modelConfig,omitempty"`
	SystemPrompt string               `json:"systemPrompt" validate:"max=1000"`
	SaveMessage  bool                 `json:"saveMessage"`
	RetryStrategy string              `json:"retryStrategy" validate:"required,oneof=simple exponential adaptive"`
	MaxRetries   int                  `json:"maxRetries" validate:"min=1,max=10"`
}

// ChatRetryOutput 对话重试输出
type ChatRetryOutput struct {
	MessageID      string         `json:"messageId"`
	Response       string         `json:"response"`
	TokenUsage     TokenUsage     `json:"tokenUsage"`
	FinishReason   string         `json:"finishReason"`
	Model          string         `json:"model"`
	GenerationTime int64          `json:"generationTime"`
	ContextInfo    ContextInfo    `json:"contextInfo"`
	RetryInfo      RetryInfo      `json:"retryInfo"`
	FallbackUsed   bool           `json:"fallbackUsed"`
	FallbackReason string         `json:"fallbackReason,omitempty"`
}

// RetryInfo 重试信息
type RetryInfo struct {
	Strategy      string        `json:"strategy"`
	TotalAttempts int           `json:"totalAttempts"`
	SuccessAttempt int          `json:"successAttempt"`
	FailedAttempts []RetryAttempt `json:"failedAttempts,omitempty"`
	TotalRetryTime int64         `json:"totalRetryTime"`
}

// RetryAttempt 重试尝试记录
type RetryAttempt struct {
	AttemptNumber int    `json:"attemptNumber"`
	Error         string `json:"error"`
	WaitTime      int64  `json:"waitTime"`
	Timestamp     string `json:"timestamp"`
}

// FallbackOperation 回退操作
type FallbackOperation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}

// MemoryStoreInput 记忆存储输入
type MemoryStoreInput struct {
	SessionID      string                 `json:"sessionId" validate:"required,uuid"`      // 会话ID
	MessageIDs     []string               `json:"messageIds" validate:"dive,uuid"`         // 消息ID列表
	MemoryType     string                 `json:"memoryType" validate:"required,oneof=short_term long_term summary"` // 记忆类型
	Content        string                 `json:"content" validate:"max=10000"`            // 内容（可选，如果提供则不从消息提取）
	Importance     *float32               `json:"importance" validate:"omitempty,min=0,max=1"` // 重要性（可选，自动评估）
	ExpirationDays int                    `json:"expirationDays" validate:"min=0,max=365"` // 过期天数（0表示不过期）
	Metadata       map[string]interface{} `json:"metadata"`                                // 元数据
}

// MemoryStoreOutput 记忆存储输出
type MemoryStoreOutput struct {
	MemoryID       string                 `json:"memoryId"`       // 记忆ID
	SessionID      string                 `json:"sessionId"`      // 会话ID
	MemoryType     string                 `json:"memoryType"`     // 记忆类型
	Content        string                 `json:"content"`        // 内容
	TokenCount     int                    `json:"tokenCount"`     // Token数量
	Importance     float32                `json:"importance"`     // 重要性
	KeyEntities    []string               `json:"keyEntities"`    // 关键实体
	Keywords       []string               `json:"keywords"`       // 关键词
	ExpiresAt      string                 `json:"expiresAt"`      // 过期时间
	Metadata       map[string]interface{} `json:"metadata"`       // 元数据
	VectorGenerated bool                  `json:"vectorGenerated"` // 是否生成向量
	StorageTime    int64                  `json:"storageTime"`    // 存储耗时（毫秒）
}

// MemoryCleanupInput 记忆清理输入
type MemoryCleanupInput struct {
	SessionID  string `json:"sessionId" validate:"omitempty,uuid"`                    // 会话ID（可选，为空则清理租户下所有会话）
	Strategy   string `json:"strategy" validate:"required,oneof=expired low_quality unused all"` // 清理策略
	Mode       string `json:"mode" validate:"required,oneof=soft hard"`              // 清理模式：soft（软删除）、hard（硬删除）
	BatchSize  int    `json:"batchSize" validate:"min=1,max=1000"`                   // 批量处理大小
	Execute    bool   `json:"execute"`                                               // 是否执行删除（false为预览模式）
}

// MemoryCleanupOutput 记忆清理输出
type MemoryCleanupOutput struct {
	SessionID      string          `json:"sessionId,omitempty"`      // 会话ID
	Strategy       string          `json:"strategy"`                 // 清理策略
	Mode           string          `json:"mode"`                     // 清理模式
	CleanedCount   int             `json:"cleanedCount"`             // 清理数量
	FreedSpace     int64           `json:"freedSpace"`               // 释放空间（字节）
	FreedTokens    int             `json:"freedTokens"`              // 释放Token数量
	Details        []CleanupDetail `json:"details"`                  // 清理详情
	PreviewMode    bool            `json:"previewMode"`              // 是否为预览模式
	CleanupTime    int64           `json:"cleanupTime"`              // 清理耗时（毫秒）
	TotalProcessed int             `json:"totalProcessed"`           // 总处理数量
}

// CleanupDetail 清理详情
type CleanupDetail struct {
	MemoryID   string  `json:"memoryId"`   // 记忆ID
	SessionID  string  `json:"sessionId"`  // 会话ID
	MemoryType string  `json:"memoryType"` // 记忆类型
	Reason     string  `json:"reason"`     // 清理原因
	Size       int64   `json:"size"`       // 大小（字节）
	TokenCount int     `json:"tokenCount"` // Token数量
	Importance float32 `json:"importance"` // 重要性
	CreatedAt  string  `json:"createdAt"`  // 创建时间
	LastAccess string  `json:"lastAccess"` // 最后访问时间
}
// ========== 摘要相关类型 ==========

// SummaryGenerateInput 摘要生成输入
type SummaryGenerateInput struct {
	SessionID       string   `json:"sessionId"`       // 会话ID
	MessageIDs      []string `json:"messageIds"`      // 消息ID列表（可选）
	StartMessageID  string   `json:"startMessageID"`  // 起始消息ID（可选）
	EndMessageID    string   `json:"endMessageID"`    // 结束消息ID（可选）
	PreviousSummary string   `json:"previousSummary"` // 之前的摘要（增量摘要时使用）
	SummaryType     string   `json:"summaryType"`     // 摘要类型：incremental（增量）、full（完整）
	TargetLength    int      `json:"targetLength"`    // 目标长度（Token数）
}

// SummaryGenerateOutput 摘要生成输出
type SummaryGenerateOutput struct {
	SummaryID       string   `json:"summaryId"`       // 摘要ID
	Summary         string   `json:"summary"`         // 摘要内容
	TokenCount      int      `json:"tokenCount"`      // Token数量
	MessageCount    int      `json:"messageCount"`    // 消息数量
	StartMessageID  string   `json:"startMessageID"`  // 起始消息ID
	EndMessageID    string   `json:"endMessageID"`    // 结束消息ID
	QualityScore    float64  `json:"qualityScore"`    // 质量评分（0-1）
	CompressionRate float64  `json:"compressionRate"` // 压缩率（节省的Token比例）
	KeyTopics       []string `json:"keyTopics"`       // 关键主题列表
	GenerationTime  int64    `json:"generationTime"`  // 生成耗时（毫秒）
}

// SummaryTriggerInput 摘要触发检查输入
type SummaryTriggerInput struct {
	SessionID string `json:"sessionId" validate:"required,uuid"` // 会话ID
	CheckMode string `json:"checkMode" validate:"oneof=auto force"` // 检查模式：auto（自动）、force（强制）
}

// SummaryTriggerOutput 摘要触发检查输出
type SummaryTriggerOutput struct {
	ShouldSummarize       bool     `json:"shouldSummarize"`       // 是否应该生成摘要
	TriggerReason         string   `json:"triggerReason"`         // 触发原因
	TriggerConditions     []string `json:"triggerConditions"`     // 满足的触发条件列表
	MessagesSinceLastSummary int   `json:"messagesSinceLastSummary"` // 自上次摘要后的消息数
	CurrentTokenCount     int      `json:"currentTokenCount"`     // 当前Token数量
	MaxTokens             int      `json:"maxTokens"`             // 最大Token限制
	TokenUsageRate        float64  `json:"tokenUsageRate"`        // Token使用率（0-1）
	ContextQualityScore   float64  `json:"contextQualityScore"`   // 上下文质量评分（0-1）
	TimeSinceLastSummary  int64    `json:"timeSinceLastSummary"`  // 距离上次摘要的时间（秒）
	EstimatedTokenSaving  int      `json:"estimatedTokenSaving"`  // 预计节省的Token数量
	Urgency               float64  `json:"urgency"`               // 紧急程度（0-1）
	RecommendedType       string   `json:"recommendedType"`       // 推荐的摘要类型：incremental或full
	TriggerScore          float64  `json:"triggerScore"`          // 综合触发得分（0-1）
	CheckTime             int64    `json:"checkTime"`             // 检查耗时（毫秒）
}

// SummaryQualityInput 摘要质量评估输入
type SummaryQualityInput struct {
	SummaryID        string   `json:"summaryId" validate:"omitempty,uuid"`        // 摘要ID（可选，如果提供则从数据库加载）
	Summary          string   `json:"summary" validate:"required_without=SummaryID,max=10000"` // 摘要内容
	OriginalMessages []string `json:"originalMessages" validate:"required,min=1"` // 原始消息列表
	Dimensions       []string `json:"dimensions" validate:"dive,oneof=completeness conciseness coherence accuracy"` // 评估维度（可选，默认全部）
}

// SummaryQualityOutput 摘要质量评估输出
type SummaryQualityOutput struct {
	SummaryID        string                 `json:"summaryId,omitempty"`        // 摘要ID
	OverallScore     float64                `json:"overallScore"`               // 总体质量评分（0-1）
	DimensionScores  map[string]float64     `json:"dimensionScores"`            // 各维度评分
	Passed           bool                   `json:"passed"`                     // 是否通过质量检查（>= 0.7）
	Issues           []QualityIssue         `json:"issues"`                     // 质量问题列表
	Suggestions      []string               `json:"suggestions"`                // 改进建议
	KeyInfoCoverage  float64                `json:"keyInfoCoverage"`            // 关键信息覆盖率（0-1）
	RedundancyScore  float64                `json:"redundancyScore"`            // 冗余度评分（0-1，越低越好）
	EvaluationTime   int64                  `json:"evaluationTime"`             // 评估耗时（毫秒）
}

// QualityIssue 质量问题
type QualityIssue struct {
	Dimension   string  `json:"dimension"`   // 维度：completeness、conciseness、coherence、accuracy
	Severity    string  `json:"severity"`    // 严重程度：low、medium、high
	Description string  `json:"description"` // 问题描述
	Score       float64 `json:"score"`       // 该维度的评分（0-1）
	Impact      string  `json:"impact"`      // 影响说明
}

// ========== 流式响应相关类型 ==========

// ChatStreamInput 流式对话输入
type ChatStreamInput struct {
	SessionID                string               `json:"sessionId" validate:"required,uuid"`     // 会话ID
	UserMessage              string               `json:"userMessage" validate:"required,max=4000"` // 用户消息
	Context                  *ContextBuildOutput  `json:"context,omitempty"`                      // 上下文（可选）
	ModelConfig              *ModelConfig         `json:"modelConfig,omitempty"`                  // 模型配置
	SystemPrompt             string               `json:"systemPrompt" validate:"max=1000"`       // 系统提示词
	SaveMessage              bool                 `json:"saveMessage"`                            // 是否保存消息
	IncludeTokenStats        bool                 `json:"includeTokenStats"`                      // 是否包含Token统计
	IncludeIntermediateStates bool                `json:"includeIntermediateStates"`              // 是否包含中间状态
	BufferSize               int                  `json:"bufferSize" validate:"min=1,max=100"`    // 缓冲区大小（字符数）
	SendInterval             int                  `json:"sendInterval" validate:"min=10,max=1000"` // 发送间隔（毫秒）
}

// ChatStreamOutput 流式对话输出
type ChatStreamOutput struct {
	MessageID      string      `json:"messageId"`      // 消息ID
	Response       string      `json:"response"`       // 完整响应
	TokenUsage     TokenUsage  `json:"tokenUsage"`     // Token使用统计
	FinishReason   string      `json:"finishReason"`   // 完成原因
	Model          string      `json:"model"`          // 模型名称
	GenerationTime int64       `json:"generationTime"` // 生成耗时（毫秒）
	ContextInfo    ContextInfo `json:"contextInfo"`    // 上下文信息
	StreamStats    StreamStats `json:"streamStats"`    // 流式统计
}

// StreamStats 流式统计信息
type StreamStats struct {
	TotalChunks       int   `json:"totalChunks"`       // 总块数
	FirstByteTime     int64 `json:"firstByteTime"`     // 首字节时间（毫秒）
	AverageChunkDelay int64 `json:"averageChunkDelay"` // 平均块延迟（毫秒）
	TotalStreamTime   int64 `json:"totalStreamTime"`   // 总流式时间（毫秒）
}

// StreamChunk 流式块
type StreamChunk struct {
	Type      string                 `json:"type"`                // 块类型：start、content、token_stats、end、error
	Content   string                 `json:"content,omitempty"`   // 内容（content类型）
	TokenStats *TokenUsage           `json:"tokenStats,omitempty"` // Token统计（token_stats类型）
	State     *IntermediateState     `json:"state,omitempty"`     // 中间状态（content类型，如果启用）
	Error     *StreamError           `json:"error,omitempty"`     // 错误信息（error类型）
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // 元数据
	Timestamp string                 `json:"timestamp"`           // 时间戳
	ChunkID   int                    `json:"chunkId"`             // 块ID（序号）
}

// IntermediateState 中间状态
type IntermediateState struct {
	CurrentTokens     int     `json:"currentTokens"`     // 当前Token数
	EstimatedTotal    int     `json:"estimatedTotal"`    // 预计总Token数
	Progress          float64 `json:"progress"`          // 进度（0-1）
	ProcessingStage   string  `json:"processingStage"`   // 处理阶段
}

// StreamError 流式错误
type StreamError struct {
	Code    string `json:"code"`    // 错误代码
	Message string `json:"message"` // 错误消息
	Details string `json:"details"` // 错误详情
	Recoverable bool `json:"recoverable"` // 是否可恢复
}

// ========== 完整对话流程相关类型 ==========

// CompleteConversationInput 完整对话流程输入
type CompleteConversationInput struct {
	SessionID            string       `json:"sessionId" validate:"required,uuid"`     // 会话ID
	UserMessage          string       `json:"userMessage" validate:"required,max=4000"` // 用户消息
	ModelConfig          *ModelConfig `json:"modelConfig,omitempty"`                  // 模型配置
	SystemPrompt         string       `json:"systemPrompt" validate:"max=1000"`       // 系统提示词
	EnableQueryClassify  bool         `json:"enableQueryClassify"`                    // 是否启用查询分类
	AutoOptimizeContext  bool         `json:"autoOptimizeContext"`                    // 是否自动优化上下文
	EnableStreaming      bool         `json:"enableStreaming"`                        // 是否启用流式响应
	SaveMemory           bool         `json:"saveMemory"`                             // 是否保存记忆
	AutoGenerateSummary  bool         `json:"autoGenerateSummary"`                    // 是否自动生成摘要
	MaxTokens            int          `json:"maxTokens" validate:"min=100,max=32000"` // 最大Token数
	ContextStrategy      string       `json:"contextStrategy" validate:"oneof=auto short full"` // 上下文策略
}

// CompleteConversationOutput 完整对话流程输出
type CompleteConversationOutput struct {
	MessageID        string                  `json:"messageId"`        // 消息ID
	Response         string                  `json:"response"`         // AI响应
	TokenUsage       TokenUsage              `json:"tokenUsage"`       // Token使用统计
	FinishReason     string                  `json:"finishReason"`     // 完成原因
	Model            string                  `json:"model"`            // 模型名称
	ExecutedSteps    []ExecutedStep          `json:"executedSteps"`    // 已执行的步骤
	TotalTime        int64                   `json:"totalTime"`        // 总耗时（毫秒）
	ContextInfo      ContextInfo             `json:"contextInfo"`      // 上下文信息
	QueryClassification *QueryClassifyOutput `json:"queryClassification,omitempty"` // 查询分类结果
	MemoryStored     bool                    `json:"memoryStored"`     // 是否已存储记忆
	SummaryGenerated bool                    `json:"summaryGenerated"` // 是否已生成摘要
	Warnings         []string                `json:"warnings,omitempty"` // 警告信息
}

// ExecutedStep 已执行的步骤
type ExecutedStep struct {
	StepName    string `json:"stepName"`    // 步骤名称
	Status      string `json:"status"`      // 状态：success、failed、skipped
	Duration    int64  `json:"duration"`    // 耗时（毫秒）
	Error       string `json:"error,omitempty"` // 错误信息（如果失败）
	IsOptional  bool   `json:"isOptional"`  // 是否为可选步骤
	Description string `json:"description"` // 步骤描述
}

// ========== 批量对话处理相关类型 ==========

// BatchConversationInput 批量对话处理输入
type BatchConversationInput struct {
	Requests         []ConversationRequest `json:"requests" validate:"required,min=1,max=100,dive"` // 对话请求列表
	MaxConcurrency   int                   `json:"maxConcurrency" validate:"min=1,max=20"`          // 最大并发数
	Timeout          int                   `json:"timeout" validate:"min=1000,max=300000"`          // 超时时间（毫秒）
	FailureStrategy  string                `json:"failureStrategy" validate:"required,oneof=continue abort"` // 失败策略
	EnableStreaming  bool                  `json:"enableStreaming"`                                 // 是否启用流式响应
	SaveMemory       bool                  `json:"saveMemory"`                                      // 是否保存记忆
	AutoGenerateSummary bool               `json:"autoGenerateSummary"`                             // 是否自动生成摘要
}

// ConversationRequest 单个对话请求
type ConversationRequest struct {
	RequestID       string       `json:"requestId" validate:"required"`                  // 请求ID（用于标识）
	SessionID       string       `json:"sessionId" validate:"required,uuid"`             // 会话ID
	UserMessage     string       `json:"userMessage" validate:"required,max=4000"`       // 用户消息
	ModelConfig     *ModelConfig `json:"modelConfig,omitempty"`                          // 模型配置
	SystemPrompt    string       `json:"systemPrompt" validate:"max=1000"`               // 系统提示词
	MaxTokens       int          `json:"maxTokens" validate:"min=100,max=32000"`         // 最大Token数
	ContextStrategy string       `json:"contextStrategy" validate:"oneof=auto short full"` // 上下文策略
	Priority        int          `json:"priority" validate:"min=0,max=10"`               // 优先级（0-10，数字越大优先级越高）
}

// BatchConversationOutput 批量对话处理输出
type BatchConversationOutput struct {
	TotalRequests    int                      `json:"totalRequests"`    // 总请求数
	SuccessCount     int                      `json:"successCount"`     // 成功数量
	FailureCount     int                      `json:"failureCount"`     // 失败数量
	SuccessResponses []ConversationResponse   `json:"successResponses"` // 成功的响应列表
	FailureResponses []FailedConversation     `json:"failureResponses"` // 失败的请求列表
	TotalTime        int64                    `json:"totalTime"`        // 总耗时（毫秒）
	AverageTime      int64                    `json:"averageTime"`      // 平均耗时（毫秒）
	MaxTime          int64                    `json:"maxTime"`          // 最大耗时（毫秒）
	MinTime          int64                    `json:"minTime"`          // 最小耗时（毫秒）
	Aborted          bool                     `json:"aborted"`          // 是否因失败策略而中止
	AbortReason      string                   `json:"abortReason,omitempty"` // 中止原因
	ProcessingStats  BatchProcessingStats     `json:"processingStats"`  // 处理统计
}

// ConversationResponse 对话响应
type ConversationResponse struct {
	RequestID      string      `json:"requestId"`      // 请求ID
	SessionID      string      `json:"sessionId"`      // 会话ID
	MessageID      string      `json:"messageId"`      // 消息ID
	Response       string      `json:"response"`       // AI响应
	TokenUsage     TokenUsage  `json:"tokenUsage"`     // Token使用统计
	FinishReason   string      `json:"finishReason"`   // 完成原因
	Model          string      `json:"model"`          // 模型名称
	ProcessingTime int64       `json:"processingTime"` // 处理耗时（毫秒）
	ContextInfo    ContextInfo `json:"contextInfo"`    // 上下文信息
	Priority       int         `json:"priority"`       // 优先级
	CompletedAt    string      `json:"completedAt"`    // 完成时间
}

// FailedConversation 失败的对话请求
type FailedConversation struct {
	RequestID   string `json:"requestId"`   // 请求ID
	SessionID   string `json:"sessionId"`   // 会话ID
	UserMessage string `json:"userMessage"` // 用户消息
	Error       string `json:"error"`       // 错误信息
	ErrorCode   string `json:"errorCode"`   // 错误代码
	FailedAt    string `json:"failedAt"`    // 失败时间
	Retryable   bool   `json:"retryable"`   // 是否可重试
	Priority    int    `json:"priority"`    // 优先级
}

// BatchProcessingStats 批量处理统计
type BatchProcessingStats struct {
	StartTime         string  `json:"startTime"`         // 开始时间
	EndTime           string  `json:"endTime"`           // 结束时间
	TotalTokensUsed   int     `json:"totalTokensUsed"`   // 总Token使用量
	AverageTokensPerRequest int `json:"averageTokensPerRequest"` // 平均每请求Token数
	ConcurrencyUsed   int     `json:"concurrencyUsed"`   // 实际使用的并发数
	TimeoutCount      int     `json:"timeoutCount"`      // 超时数量
	SuccessRate       float64 `json:"successRate"`       // 成功率（0-1）
	ThroughputPerSecond float64 `json:"throughputPerSecond"` // 吞吐量（请求/秒）
}

// ========== 会话健康检查相关类型 ==========

// SessionHealthCheckInput 会话健康检查输入
type SessionHealthCheckInput struct {
	SessionID   string   `json:"sessionId" validate:"required,uuid"`                    // 会话ID
	CheckItems  []string `json:"checkItems" validate:"dive,oneof=context token memory summary performance"` // 检查项（可选，默认全部）
	AutoFix     bool     `json:"autoFix"`                                               // 是否自动修复
	DetailLevel string   `json:"detailLevel" validate:"oneof=basic detailed full"`      // 详细级别
}

// SessionHealthCheckOutput 会话健康检查输出
type SessionHealthCheckOutput struct {
	SessionID       string                 `json:"sessionId"`       // 会话ID
	OverallHealth   string                 `json:"overallHealth"`   // 整体健康状态：healthy、warning、critical
	OverallScore    float64                `json:"overallScore"`    // 整体健康评分（0-1）
	CheckResults    []HealthCheckResult    `json:"checkResults"`    // 检查结果列表
	Issues          []HealthIssue          `json:"issues"`          // 健康问题列表
	Recommendations []string               `json:"recommendations"` // 建议列表
	FixOperations   []FixOperation         `json:"fixOperations"`   // 修复操作列表
	CheckTime       int64                  `json:"checkTime"`       // 检查耗时（毫秒）
	LastCheckAt     string                 `json:"lastCheckAt"`     // 最后检查时间
	NextCheckSuggested string              `json:"nextCheckSuggested"` // 建议下次检查时间
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	CheckItem   string                 `json:"checkItem"`   // 检查项：context、token、memory、summary、performance
	Status      string                 `json:"status"`      // 状态：healthy、warning、critical
	Score       float64                `json:"score"`       // 评分（0-1）
	Message     string                 `json:"message"`     // 消息
	Details     map[string]interface{} `json:"details"`     // 详细信息
	Issues      []string               `json:"issues"`      // 问题列表
	CheckTime   int64                  `json:"checkTime"`   // 检查耗时（毫秒）
}

// HealthIssue 健康问题
type HealthIssue struct {
	CheckItem   string  `json:"checkItem"`   // 检查项
	Severity    string  `json:"severity"`    // 严重程度：low、medium、high、critical
	Type        string  `json:"type"`        // 问题类型
	Description string  `json:"description"` // 问题描述
	Impact      string  `json:"impact"`      // 影响说明
	Suggestion  string  `json:"suggestion"`  // 修复建议
	AutoFixable bool    `json:"autoFixable"` // 是否可自动修复
	Priority    int     `json:"priority"`    // 优先级（1-10）
}

// FixOperation 修复操作
type FixOperation struct {
	OperationType string                 `json:"operationType"` // 操作类型
	CheckItem     string                 `json:"checkItem"`     // 相关检查项
	Description   string                 `json:"description"`   // 操作描述
	Status        string                 `json:"status"`        // 状态：pending、success、failed、skipped
	Result        string                 `json:"result"`        // 结果说明
	Error         string                 `json:"error,omitempty"` // 错误信息（如果失败）
	ExecutionTime int64                  `json:"executionTime"` // 执行耗时（毫秒）
	Details       map[string]interface{} `json:"details"`       // 详细信息
}
