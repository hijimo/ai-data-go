// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import "time"

// API 地址常量
const (
	// LexiangBaseURL 乐享 API 基础地址
	LexiangBaseURL = "https://lxapi.lexiangla.com/cgi-bin"
	// LexiangAPIURL 乐享业务 API 地址
	LexiangAPIURL = "https://lxapi.lexiangla.com/cgi-bin/v1/kb"
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
type CreateEntryRequest struct {
	Name          string    `json:"name,omitempty"`
	State         string    `json:"state,omitempty"`
	EntryType     EntryType `json:"entry_type"`
	Relationships struct {
		ParentEntry struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"parent_entry"`
	} `json:"relationships"`
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
type UploadSignResponse struct {
	State  string `json:"state"`
	Object struct {
		Key     string `json:"key"`
		Options struct {
			Bucket string `json:"bucket"`
			Region string `json:"region"`
		} `json:"options"`
		Auth struct {
			Authorization     string `json:"Authorization"`
			XCosSecurityToken string `json:"XCosSecurityToken"`
		} `json:"auth"`
		Headers struct {
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
