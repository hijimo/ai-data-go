package model

// StreamStage 流式输出阶段
type StreamStage string

const (
	// 工具调用相关阶段
	StreamStageToolCallStart    StreamStage = "tool_call_start"    // 开始调用工具
	StreamStageToolCallProgress StreamStage = "tool_call_progress" // 工具调用进行中
	StreamStageToolCallComplete StreamStage = "tool_call_complete" // 工具调用完成
	StreamStageToolCallError    StreamStage = "tool_call_error"    // 工具调用失败

	// 搜索/检索相关阶段
	StreamStageInternalSearching         StreamStage = "internal_searching"          // 正在搜索知识库
	StreamStageFinishedInternalSearching StreamStage = "finished_internal_searching" // 搜索完成
	StreamStageResourceRetrievalStart    StreamStage = "resource_retrieval_start"    // 开始检索资源
	StreamStageResourceRetrievalComplete StreamStage = "resource_retrieval_complete" // 资源检索完成

	// 生成相关阶段
	StreamStageThinking StreamStage = "thinking" // 正在思考
	StreamStageOutput   StreamStage = ""         // 正式输出（空字符串）
)

// StreamEventType 流式事件类型
type StreamEventType string

const (
	StreamEventData   StreamEventType = "data"   // 数据事件（默认）
	StreamEventFinish StreamEventType = "finish" // 结束事件
)

// TencentCloudStreamMessage 腾讯云流式消息（完整格式）
type TencentCloudStreamMessage struct {
	// 本次对话的唯一标识
	CompletionID string `json:"completion_id"`
	// 会话ID（可能为空，在finish事件中返回）
	SessionID string `json:"session_id,omitempty"`
	// 处理过程信息
	Processes ProcessInfo `json:"processes"`
	// 增量输出内容（正式输出阶段使用）
	DeltaContent string `json:"delta_content"`
	// 完整内容（仅在finish事件中提供）
	Content string `json:"content"`
	// 结束原因
	FinishReason string `json:"finish_reason"`
	// 是否停止
	IsStop bool `json:"is_stop"`
	// 答案来源
	AnswerSource string `json:"answer_source,omitempty"`
	// 附加内容
	AdditionalContent *AdditionalContent `json:"additional_content,omitempty"`
}

// ProcessInfo 处理过程信息
type ProcessInfo struct {
	// 当前阶段
	Stage StreamStage `json:"stage"`
	// 阶段描述信息
	Message string `json:"message"`
	// 增量内容（思考阶段使用）
	DeltaContent string `json:"delta_content"`
	// 完整内容
	Content string `json:"content"`
	// 详细信息（根据不同阶段类型不同）
	Detail interface{} `json:"detail"`
}

// ToolCallDetail 工具调用详情
type ToolCallDetail struct {
	// 工具名称
	ToolName string `json:"tool_name"`
	// 工具ID
	ToolID string `json:"tool_id"`
	// 进度（0-100，仅在progress阶段使用）
	Progress int `json:"progress,omitempty"`
	// 结果（仅在complete阶段使用）
	Result *ToolCallResult `json:"result,omitempty"`
	// 错误（仅在error阶段使用）
	Error *ToolCallError `json:"error,omitempty"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	// 状态：success 或 error
	Status string `json:"status"`
	// 数据
	Data interface{} `json:"data"`
}

// ToolCallError 工具调用错误
type ToolCallError struct {
	// 错误代码
	Code string `json:"code"`
	// 错误消息
	Message string `json:"message"`
}

// SearchDetail 搜索详情
type SearchDetail struct {
	// 空间名称
	SpaceName string `json:"space_name,omitempty"`
	// 空间数量
	SpaceCount int `json:"space_count,omitempty"`
	// 文档数量
	DocCount int `json:"doc_count,omitempty"`
}

// ResourceRetrievalDetail 资源检索详情
type ResourceRetrievalDetail struct {
	// 查询关键词
	Query string `json:"query"`
	// 资源类型
	ResourceType string `json:"resource_type"`
	// 资源数量
	ResourceCount int `json:"resource_count,omitempty"`
	// 资源列表
	Resources []interface{} `json:"resources,omitempty"`
}

// AdditionalContent 附加内容
type AdditionalContent struct {
	// 上下文限制引用块数量
	ContextLimitReferenceChunksTopN int `json:"context_limit_reference_chunks_top_n,omitempty"`
	// 引用块列表（搜索完成阶段）
	ReferenceChunks []ReferenceChunk `json:"reference_chunks,omitempty"`
	// 生成的问题
	GeneratedQuestion string `json:"generated_question,omitempty"`
	// 引用文档列表（finish阶段）
	ReferenceDocs []ReferenceDoc `json:"reference_docs,omitempty"`
	// 场景
	Scenario string `json:"scenario,omitempty"`
}

// ReferenceChunk 引用块
type ReferenceChunk struct {
	// 块ID
	BlockID string `json:"block_id"`
	// 内容
	Content string `json:"content"`
	// 带MLLM处理的内容
	ContentWithMLLM string `json:"content_with_mllm"`
	// 文件信息
	FileInfo FileInfo `json:"file_info"`
	// 文件页码
	FilePages interface{} `json:"file_pages"`
	// 文件类型
	FileType string `json:"file_type"`
	// 会议信息
	MeetingInfo MeetingInfo `json:"meeting_info"`
	// 所有者
	Owner Owner `json:"owner"`
	// 空间信息
	SpaceInfo SpaceInfo `json:"space_info"`
	// 目标ID
	TargetID string `json:"target_id"`
	// 目标类型
	TargetType string `json:"target_type"`
	// 标题
	Title string `json:"title"`
	// 更新时间
	UpdatedAt int64 `json:"updated_at"`
	// URL
	URL string `json:"url"`
}

// ReferenceDoc 引用文档
type ReferenceDoc struct {
	// 块ID
	BlockID string `json:"block_id"`
	// 文件类型
	FileType string `json:"file_type"`
	// 目标ID
	TargetID string `json:"target_id"`
	// 目标类型
	TargetType string `json:"target_type"`
	// 标题
	Title string `json:"title"`
	// URL
	URL string `json:"url"`
}

// FileInfo 文件信息
type FileInfo struct {
	// 目标ID
	TargetID string `json:"target_id"`
	// 目标类型
	TargetType string `json:"target_type"`
}

// MeetingInfo 会议信息
type MeetingInfo struct {
	// 封面URL
	CoverURL string `json:"cover_url"`
	// 开始时间
	StartTime int64 `json:"start_time"`
}

// Owner 所有者
type Owner struct {
	// 头像
	Avatar string `json:"avatar"`
	// 显示名称
	DisplayName string `json:"display_name"`
}

// SpaceInfo 空间信息
type SpaceInfo struct {
	// ID
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
}
