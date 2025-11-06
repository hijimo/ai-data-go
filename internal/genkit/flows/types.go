package flows

// ContextBuildInput 上下文构建Flow的输入参数
type ContextBuildInput struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid"`
	// 用户查询
	UserQuery string `json:"userQuery" validate:"required,max=2000"`
	// 最大Token数量
	MaxTokens int `json:"maxTokens" validate:"min=100,max=32000"`
	// 上下文策略：auto（自动）、short（仅短期）、full（完整）
	Strategy string `json:"strategy" validate:"oneof=auto short full"`
	// 是否包含摘要
	IncludeSummary bool `json:"includeSummary"`
	// 是否包含长期记忆
	IncludeLongTerm bool `json:"includeLongTerm"`
	// 短期记忆窗口大小（最近N条消息）
	ShortTermWindow int `json:"shortTermWindow" validate:"min=1,max=50"`
}

// ContextBuildOutput 上下文构建Flow的输出结果
type ContextBuildOutput struct {
	// 会话ID
	SessionID string `json:"sessionId"`
	// 摘要上下文
	Summary *SummaryContext `json:"summary,omitempty"`
	// 长期记忆列表
	LongTermMemories []MemoryContext `json:"longTermMemories,omitempty"`
	// 短期消息列表
	ShortTermMessages []MessageContext `json:"shortTermMessages"`
	// 总Token数量
	TotalTokens int `json:"totalTokens"`
	// 使用的策略
	Strategy string `json:"strategy"`
	// 上下文质量评分（0-1）
	QualityScore float64 `json:"qualityScore"`
	// 构建耗时（毫秒）
	BuildTime int64 `json:"buildTime"`
}

// SummaryContext 摘要上下文
type SummaryContext struct {
	// 摘要内容
	Content string `json:"content"`
	// Token数量
	TokenCount int `json:"tokenCount"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 覆盖范围描述
	Coverage string `json:"coverage"`
}

// MemoryContext 记忆上下文
type MemoryContext struct {
	// 记忆ID
	ID string `json:"id"`
	// 记忆内容
	Content string `json:"content"`
	// Token数量
	TokenCount int `json:"tokenCount"`
	// 重要性评分（0-1）
	Importance float32 `json:"importance"`
	// 相似度评分（0-1）
	Similarity float32 `json:"similarity"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
}

// MessageContext 消息上下文
type MessageContext struct {
	// 消息ID
	ID string `json:"id"`
	// 角色：user（用户）、assistant（助手）、system（系统）
	Role string `json:"role"`
	// 消息内容
	Content string `json:"content"`
	// Token数量
	TokenCount int `json:"tokenCount"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
}
