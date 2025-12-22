// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"time"
)

// API 地址常量
const (
	// LexiangBaseURL 乐享 API 基础地址
	LexiangBaseURL = "https://lxapi.lexiangla.com/cgi-bin"
	// LexiangAPIURL 乐享业务 API 地址
	LexiangAPIURL = "https://lxapi.lexiangla.com/cgi-bin/v1"
	// DefaultTimeout 默认 HTTP 超时时间
	DefaultTimeout = 30 * time.Second
	// TokenRefreshBuffer Token 提前刷新时间，避免临界点失效
	TokenRefreshBuffer = 5 * time.Minute
)

// EntryType 知识节点类型
type EntryType string

const (
	// EntryTypeFolder 文件夹
	EntryTypeFolder EntryType = "folder"
	// EntryTypeFile 文件
	EntryTypeFile EntryType = "file"
	// EntryTypeVideo 视频
	EntryTypeVideo EntryType = "video"
	// EntryTypeAudio 音频
	EntryTypeAudio EntryType = "audio"
	// EntryTypeLink 链接
	EntryTypeLink EntryType = "link"
	// EntryTypePage 线上文档
	EntryTypePage EntryType = "page"
	// EntryTypeSmartsheet 智能表格
	EntryTypeSmartsheet EntryType = "smartsheet"
)

// SpaceRole 知识库角色
type SpaceRole string

const (
	// SpaceRoleNone 无角色
	SpaceRoleNone SpaceRole = "none"
	// SpaceRoleViewer 查看者
	SpaceRoleViewer SpaceRole = "viewer"
	// SpaceRoleDownloader 下载者
	SpaceRoleDownloader SpaceRole = "downloader"
	// SpaceRoleEditor 编辑者
	SpaceRoleEditor SpaceRole = "editor"
	// SpaceRoleManager 管理者
	SpaceRoleManager SpaceRole = "manager"
	// SpaceRoleDefault 默认角色（仅用于 member_inherit_type）
	SpaceRoleDefault SpaceRole = "default"
)

// FeedbackStatus 反馈状态
type FeedbackStatus string

const (
	// FeedbackStatusUnprocessed 未处理
	FeedbackStatusUnprocessed FeedbackStatus = "unprocessed"
	// FeedbackStatusProcessing 处理中
	FeedbackStatusProcessing FeedbackStatus = "processing"
	// FeedbackStatusProcessed 已处理
	FeedbackStatusProcessed FeedbackStatus = "processed"
	// FeedbackStatusNotProcess 无需处理
	FeedbackStatusNotProcess FeedbackStatus = "not_process"
)

// FeedbackType 反馈类型
type FeedbackType string

const (
	// FeedbackTypeIncomplete 内容缺失
	FeedbackTypeIncomplete FeedbackType = "kb_content_incomplete"
	// FeedbackTypeMistake 内容有误
	FeedbackTypeMistake FeedbackType = "kb_content_mistake"
	// FeedbackTypeSuggestion 内容建议
	FeedbackTypeSuggestion FeedbackType = "kb_content_suggestion"
	// FeedbackTypeTooOld 内容陈旧
	FeedbackTypeTooOld FeedbackType = "kb_content_too_old"
	// FeedbackTypeOther 其他
	FeedbackTypeOther FeedbackType = "kb_content_other"
)

// Config 乐享客户端配置
type Config struct {
	// AppKey 应用 Key（必填）
	AppKey string
	// AppSecret 应用 Secret（必填）
	AppSecret string
	// StaffID 成员帐号（必填，用于 x-staff-id 请求头）
	StaffID string
	// BaseURL API 基础 URL（可选，默认为官方地址）
	BaseURL string
	// Timeout HTTP 超时时间（可选，默认 30 秒）
	Timeout time.Duration
	// RefreshBuffer Token 提前刷新时间（可选，默认 5 分钟）
	RefreshBuffer time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BaseURL:       LexiangBaseURL,
		Timeout:       DefaultTimeout,
		RefreshBuffer: TokenRefreshBuffer,
	}
}

// ============================================================================
// Token 相关结构体
// ============================================================================

// tokenRequest Token 请求（内部使用）
type tokenRequest struct {
	GrantType string `json:"grant_type"`
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
}

// tokenResponse Token 响应（内部使用）
type tokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

// ============================================================================
// 知识库相关结构体
// ============================================================================

// CreateSpaceRequest 创建知识库请求
type CreateSpaceRequest struct {
	Data struct {
		Attributes struct {
			Name               string    `json:"name"`
			Logo               string    `json:"logo,omitempty"`
			VisibleType        int       `json:"visible_type,omitempty"`
			ManagerInheritType SpaceRole `json:"manager_inherit_type,omitempty"`
			MemberInheritType  SpaceRole `json:"member_inherit_type,omitempty"`
		} `json:"attributes"`
		Relationships struct {
			Team struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"team"`
		} `json:"relationships"`
	} `json:"data"`
}

// SpaceResponse 知识库响应
type SpaceResponse struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name               string `json:"name"`
			Logo               string `json:"logo"`
			VisibleType        int    `json:"visible_type"`
			ManagerInheritType string `json:"manager_inherit_type"`
			MemberInheritType  string `json:"member_inherit_type"`
		} `json:"attributes"`
		Relationships struct {
			Team struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"team"`
			RootEntry struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"root_entry"`
		} `json:"relationships"`
	} `json:"data"`
}

// SpaceListResponse 知识库列表响应
type SpaceListResponse struct {
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
			Logo string `json:"logo"`
		} `json:"attributes"`
		Relationships struct {
			RootEntry struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"root_entry"`
		} `json:"relationships"`
	} `json:"data"`
	Meta struct {
		PageToken string `json:"page_token"`
	} `json:"meta"`
}

// ============================================================================
// 知识节点相关结构体
// ============================================================================

// CreateEntryRequest 创建知识节点请求
// 符合官方 API 文档结构：data.attributes + data.relationships
type CreateEntryRequest struct {
	Data struct {
		Attributes struct {
			Name      string    `json:"name,omitempty"`
			EntryType EntryType `json:"entry_type"`
		} `json:"attributes"`
		Relationships struct {
			ParentEntry struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"parent_entry"`
		} `json:"relationships"`
	} `json:"data"`
}

// EntryResponse 知识节点响应
type EntryResponse struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name              string `json:"name"`
			EntryType         string `json:"entry_type"`
			HasChildren       bool   `json:"has_children"`
			CreatedAt         string `json:"created_at"`
			UpdatedAt         string `json:"updated_at"`
			MemberInheritType string `json:"member_inherit_type"`
		} `json:"attributes"`
		Links struct {
			Download string `json:"download"`
		} `json:"links"`
	} `json:"data"`
}

// EntryListResponse 知识节点列表响应
type EntryListResponse struct {
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name        string `json:"name"`
			EntryType   string `json:"entry_type"`
			HasChildren bool   `json:"has_children"`
		} `json:"attributes"`
	} `json:"data"`
	Meta struct {
		PageToken string `json:"page_token"`
	} `json:"meta"`
}

// EntryContentResponse 线上文档内容响应
type EntryContentResponse struct {
	Name        string `json:"name"`
	HTMLContent string `json:"html_content"`
}

// ============================================================================
// 文件上传相关结构体
// ============================================================================

// UploadSignRequest 获取上传签名请求
type UploadSignRequest struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"` // file, video, audio
}

// UploadSignResponse 获取上传签名响应
// 根据腾讯乐享 API 文档：https://lexiang.tencent.com/wiki/api/12004.html
// 响应结构：options 在顶层包含 Bucket/Region，object 包含 key/state/auth/headers
type UploadSignResponse struct {
	// Options COS 存储桶配置（顶层字段）
	Options struct {
		Bucket string `json:"Bucket"`
		Region string `json:"Region"`
	} `json:"options"`
	// Object 文件对象信息（包含 state）
	Object struct {
		Key   string `json:"key"`
		State string `json:"state"`
		Auth  struct {
			Authorization     string `json:"Authorization"`
			XCosSecurityToken string `json:"XCosSecurityToken"`
		} `json:"auth"`
		Headers struct {
			ContentType        string `json:"Content-Type"`
			ContentDisposition string `json:"Content-Disposition"`
		} `json:"headers"`
	} `json:"object"`
}

// ============================================================================
// 附件下载相关结构体
// ============================================================================

// DocFileResponse 附件详情响应
type DocFileResponse struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
		Links struct {
			Download string `json:"download"`
		} `json:"links"`
	} `json:"data"`
}

// ============================================================================
// 知识反馈相关结构体
// ============================================================================

// FeedbackListResponse 反馈列表响应
type FeedbackListResponse struct {
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Status     string `json:"status"`
			Type       string `json:"type"`
			Content    string `json:"content"`
			CreatedAt  string `json:"created_at"`
			ReviewedAt string `json:"reviewed_at"`
		} `json:"attributes"`
		Relationships struct {
			Owner struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"owner"`
			Entry struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"entry"`
		} `json:"relationships"`
	} `json:"data"`
	Included []struct {
		Type       string         `json:"type"`
		ID         string         `json:"id"`
		Attributes map[string]any `json:"attributes"`
	} `json:"included"`
	Meta struct {
		PageToken string `json:"page_token"`
	} `json:"meta"`
}

// ============================================================================
// AI 问答相关结构体
// ============================================================================

// QAMode AI问答模式
type QAMode string

const (
	// QAModeNormal 使用默认模型进行快速回答
	QAModeNormal QAMode = "normal"
	// QAModeNormalDSV3 使用deepseek-v3-0324模型进行快速回答
	QAModeNormalDSV3 QAMode = "normal-ds-v3"
	// QAModeNormalDSV31 使用deepseek-v3.1模型进行快速回答
	QAModeNormalDSV31 QAMode = "normal-ds-v3.1"
	// QAModeReasoning 使用deepseek-r1-0528模型进行深度思考
	QAModeReasoning QAMode = "reasoning"
	// QAModeReasoningDSV31 使用deepseek-v3.1-terminus-think模型进行深度思考
	QAModeReasoningDSV31 QAMode = "reasoning-ds-v3.1"
	// QAModeResearch 使用deepseek-r1-0528模型进行专业研究
	QAModeResearch QAMode = "research"
	// QAModeResearchDSV31 使用deepseek-v3.1-terminus-think模型进行专业研究
	QAModeResearchDSV31 QAMode = "research-ds-v3.1"
)

// TargetType 知识范围类型
type TargetType string

const (
	// TargetTypeSpace 知识库
	TargetTypeSpace TargetType = "space"
	// TargetTypeTeam 团队空间
	TargetTypeTeam TargetType = "team"
	// TargetTypeTeamCode 团队空间code
	TargetTypeTeamCode TargetType = "team_code"
	// TargetTypeKBEntry 知识节点
	TargetTypeKBEntry TargetType = "kb_entry"
)

// Target 知识范围目标
type Target struct {
	// Type 范围类型
	Type TargetType `json:"type"`
	// ID 对应范围类型的实体 ID
	ID string `json:"id"`
}

// AIQARequest AI问答请求
type AIQARequest struct {
	// Query 用户最新输入问题，最长 1024 字符（必填）
	Query string `json:"query"`
	// Stream 是否流式输出
	Stream bool `json:"stream,omitempty"`
	// AnonymousStaffID 匿名用户 ID，当 x-staff-id=system-bot 时有效，长度范围：[16, 32]
	AnonymousStaffID string `json:"anonymous_staff_id,omitempty"`
	// SkipFAQ 是否跳过已关联的FAQ
	SkipFAQ bool `json:"skip_faq,omitempty"`
	// NewSession 是否重置并开启全新话题
	NewSession bool `json:"new_session,omitempty"`
	// SessionID 会话ID，用于区分不同会话，长度：40
	SessionID string `json:"session_id,omitempty"`
	// QAMode 思考方式
	QAMode QAMode `json:"qa_mode,omitempty"`
	// MaxChars 回答内容字数限制
	MaxChars int `json:"max_chars,omitempty"`
	// Language 系统提示词的语言（zh-CN 或 en）
	Language string `json:"language,omitempty"`
	// Targets 知识范围，最多可传入20个范围对象
	Targets []Target `json:"targets,omitempty"`
}

// ReferenceChunk 参考内容段落
type ReferenceChunk struct {
	// Content 匹配的实体内容
	Content string `json:"content"`
	// TargetID 匹配的实体ID
	TargetID string `json:"target_id"`
	// TargetType 匹配的实体类型
	TargetType string `json:"target_type"`
	// Title 匹配的实体标题
	Title string `json:"title"`
	// URL 匹配的实体页面访问路径
	URL string `json:"url"`
}

// ReferenceDoc 参考文档来源
type ReferenceDoc struct {
	// Title 匹配的实体标题
	Title string `json:"title"`
	// URL 匹配的实体的页面访问路径
	URL string `json:"url"`
}

// AdditionalContent 附加内容信息
type AdditionalContent struct {
	// GeneratedQuestion 基于会话历史对用户query的语义改写结果
	GeneratedQuestion string `json:"generated_question,omitempty"`
	// ReferenceChunks 回答参考的内容段落列表
	ReferenceChunks []ReferenceChunk `json:"reference_chunks,omitempty"`
	// ReferenceDocs 回答参考的实体来源列表
	ReferenceDocs []ReferenceDoc `json:"reference_docs,omitempty"`
}

// AIQAResponseData AI问答响应数据
type AIQAResponseData struct {
	// Content 返回的完整答案内容
	Content string `json:"content"`
	// AnswerSource 答案来源：internal(全站知识)，internal-team(指定团队空间)，internal-space(指定知识库)
	AnswerSource string `json:"answer_source"`
	// ReasoningContent 深度思考过程（当qa_mode等于reasoning或research时返回）
	ReasoningContent string `json:"reasoning_content,omitempty"`
	// SessionID 会话ID
	SessionID string `json:"session_id"`
	// AdditionalContent 附加内容信息
	AdditionalContent *AdditionalContent `json:"additional_content,omitempty"`
}

// AIQAResponse AI问答响应（非流式）
type AIQAResponse struct {
	// Code 状态码：0表示成功，非0表示失败
	Code int `json:"code"`
	// Message 状态信息
	Message string `json:"message"`
	// RequestID 请求的唯一标识ID
	RequestID string `json:"request_id"`
	// Data 响应数据主体
	Data *AIQAResponseData `json:"data,omitempty"`
}

// AIQAStreamEvent AI问答流式响应事件
type AIQAStreamEvent struct {
	// CompletionID 任务id
	CompletionID string `json:"completion_id,omitempty"`
	// SessionID 会话id
	SessionID string `json:"session_id,omitempty"`
	// Processes 处理过程信息
	Processes []struct {
		Message string `json:"message"`
	} `json:"processes,omitempty"`
	// DeltaContent 内容块
	DeltaContent string `json:"delta_content,omitempty"`
	// Content 完整答案内容
	Content string `json:"content,omitempty"`
	// FinishReason 回答结束原因
	FinishReason string `json:"finish_reason,omitempty"`
	// IsStop 回答完成状态
	IsStop bool `json:"is_stop,omitempty"`
	// AnswerSource 答案来源
	AnswerSource string `json:"answer_source,omitempty"`
	// AdditionalContent 附加内容信息
	AdditionalContent *AdditionalContent `json:"additional_content,omitempty"`
}

// ============================================================================
// AI 搜索相关结构体
// ============================================================================

// AISearchRequest AI搜索请求
type AISearchRequest struct {
	// Query 搜索关键词，最长 1024 字符（必填）
	Query string `json:"query"`
	// TopN 指定返回 rerank 后排名前n的文档，最大 50（必填）
	TopN int `json:"top_n"`
	// Targets 知识范围，最多可传入20个对象
	Targets []Target `json:"targets,omitempty"`
}

// AISearchResultItem AI搜索结果项
type AISearchResultItem struct {
	// Title 文档标题
	Title string `json:"title"`
	// Content 根据query找到关联性最高的片段内容，markdown格式
	Content string `json:"content"`
	// URL 文档链接
	URL string `json:"url"`
}

// AISearchResponseData AI搜索响应数据
type AISearchResponseData struct {
	// List 搜索结果数组
	List []AISearchResultItem `json:"list"`
}

// AISearchResponse AI搜索响应
type AISearchResponse struct {
	// Code 状态码：0表示成功，非0表示失败
	Code int `json:"code"`
	// Message 状态信息
	Message string `json:"message"`
	// RequestID 请求的唯一标识ID
	RequestID string `json:"request_id"`
	// Data 响应数据
	Data *AISearchResponseData `json:"data,omitempty"`
}
