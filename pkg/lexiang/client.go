// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// LexiangClient 乐享 API 客户端接口
// 封装所有与乐享 API 的交互，自动处理 token 管理和请求重试
type LexiangClient interface {
	// HTTP 方法
	// Do 发送 HTTP 请求，自动添加 Authorization 和 Content-Type 头
	// 当收到 401 响应时，自动刷新 token 并重试一次
	Do(ctx context.Context, method, path string, body any) (*http.Response, error)

	// DoWithHeader 发送 HTTP 请求，支持自定义请求头
	// headers 中的自定义头会被添加到请求中（如 x-staff-id）
	DoWithHeader(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error)

	// Get 发送 GET 请求
	Get(ctx context.Context, path string) (*http.Response, error)

	// Post 发送 POST 请求
	Post(ctx context.Context, path string, body any) (*http.Response, error)

	// Put 发送 PUT 请求
	Put(ctx context.Context, path string, body any) (*http.Response, error)

	// Delete 发送 DELETE 请求
	Delete(ctx context.Context, path string) (*http.Response, error)

	// GetTokenManager 获取 TokenManager 实例（用于测试）
	GetTokenManager() TokenManager

	// 知识库管理方法
	// CreateSpace 创建知识库
	// teamID: 知识库所属团队 ID
	// name: 知识库名称
	CreateSpace(ctx context.Context, teamID, name string) (*SpaceResponse, error)

	// UpdateSpace 更新知识库
	// spaceID: 知识库 ID
	// name: 新的知识库名称
	UpdateSpace(ctx context.Context, spaceID, name string) (*SpaceResponse, error)

	// DeleteSpace 删除知识库
	// spaceID: 知识库 ID
	DeleteSpace(ctx context.Context, spaceID string) error

	// ListSpaces 获取知识库列表
	// teamID: 团队 ID
	// limit: 拉取条数（0 表示使用默认值）
	// pageToken: 分页游标（空字符串表示第一页）
	ListSpaces(ctx context.Context, teamID string, limit int, pageToken string) (*SpaceListResponse, error)

	// GetSpace 获取知识库详情
	// spaceID: 知识库 ID
	GetSpace(ctx context.Context, spaceID string) (*SpaceResponse, error)

	// 知识节点管理方法
	// CreateFolder 创建文件夹
	// parentID: 父节点 ID（可使用知识库的 root_entry ID）
	// name: 文件夹名称
	CreateFolder(ctx context.Context, parentID, name string) (*EntryResponse, error)

	// CreateFileEntry 创建文件知识节点
	// parentID: 父节点 ID
	// state: 文件上传临时标识（从上传签名接口获取）
	// entryType: 节点类型（file/video/audio）
	// name: 节点标题，缺省时使用上传的文件名
	CreateFileEntry(ctx context.Context, parentID, state string, entryType EntryType, name string) (*EntryResponse, error)

	// ReuploadFile 重新上传文件
	// entryID: 知识节点 ID
	// state: 新文件的上传临时标识
	ReuploadFile(ctx context.Context, entryID, state string) error

	// DeleteEntry 删除知识节点
	// entryID: 知识节点 ID
	DeleteEntry(ctx context.Context, entryID string) error

	// ListEntries 获取知识节点列表
	// spaceID: 知识库 ID
	// parentID: 父节点 ID，空字符串表示查询根目录
	// limit: 拉取条数（0 表示使用默认值）
	// pageToken: 分页游标（空字符串表示第一页）
	ListEntries(ctx context.Context, spaceID, parentID string, limit int, pageToken string) (*EntryListResponse, error)

	// GetEntry 获取知识节点详情
	// entryID: 知识节点 ID
	GetEntry(ctx context.Context, entryID string) (*EntryResponse, error)

	// GetEntryContent 获取线上文档内容
	// entryID: 知识节点 ID（仅支持 entry_type=page 的节点）
	GetEntryContent(ctx context.Context, entryID string) (*EntryContentResponse, error)

	// 文件上传方法
	// GetUploadSign 获取上传签名
	// fileName: 文件名称（需带扩展名）
	// mediaType: 媒体类型（file/video/audio）
	GetUploadSign(ctx context.Context, fileName, mediaType string) (*UploadSignResponse, error)

	// UploadFileToCOS 上传文件到腾讯云 COS
	// sign: 上传签名响应（从 GetUploadSign 获取）
	// fileData: 文件二进制数据
	UploadFileToCOS(ctx context.Context, sign *UploadSignResponse, fileData []byte) error

	// UploadFile 完整的文件上传流程
	// fileName: 文件名称（需带扩展名）
	// mediaType: 媒体类型（file/video/audio）
	// fileData: 文件二进制数据
	// 返回 state 用于后续创建知识节点
	UploadFile(ctx context.Context, fileName, mediaType string, fileData []byte) (string, error)

	// 附件下载方法
	// GetDocFile 获取附件详情
	// fileID: 附件 ID
	// 返回附件详情，包含名称和下载链接
	GetDocFile(ctx context.Context, fileID string) (*DocFileResponse, error)

	// DownloadDocFile 下载附件
	// fileID: 附件 ID
	// 返回文件数据、文件名和错误
	DownloadDocFile(ctx context.Context, fileID string) ([]byte, string, error)

	// 知识反馈方法
	// ListFeedbacks 获取知识反馈列表
	// spaceID: 知识库 ID
	// limit: 拉取条数（0 表示使用默认值）
	// pageToken: 分页游标（空字符串表示第一页）
	// 返回反馈列表，包含关联的用户和节点信息（included 字段）
	ListFeedbacks(ctx context.Context, spaceID string, limit int, pageToken string) (*FeedbackListResponse, error)
}

// lexiangClientImpl LexiangClient 的实现
type lexiangClientImpl struct {
	// 配置
	apiURL  string
	staffID string
	timeout int

	// HTTP 客户端
	httpClient *http.Client

	// Token 管理器
	tokenManager TokenManager
}

// NewClient 创建新的 LexiangClient 实例
// config 必须包含 AppKey、AppSecret 和 StaffID
func NewClient(config *Config) (LexiangClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config 不能为空")
	}

	if config.StaffID == "" {
		return nil, fmt.Errorf("StaffID 不能为空")
	}

	// 创建 TokenManager
	tm, err := NewTokenManager(config)
	if err != nil {
		return nil, fmt.Errorf("创建 TokenManager 失败: %w", err)
	}

	// 确定 API URL
	apiURL := LexiangAPIURL
	if config.BaseURL != "" && config.BaseURL != LexiangBaseURL {
		// 如果自定义了 BaseURL，则 API URL 也需要调整
		apiURL = config.BaseURL + "/v1"
	}

	// 确定超时时间
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &lexiangClientImpl{
		apiURL:  apiURL,
		staffID: config.StaffID,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		tokenManager: tm,
	}, nil
}

// NewClientFromEnv 从环境变量创建 LexiangClient 实例
// 读取环境变量：LEXIANG_APP_KEY, LEXIANG_APP_SECRET, LEXIANG_STAFF_ID
func NewClientFromEnv() (LexiangClient, error) {
	appKey := os.Getenv("LEXIANG_APP_KEY")
	appSecret := os.Getenv("LEXIANG_APP_SECRET")
	staffID := os.Getenv("LEXIANG_STAFF_ID")

	if appKey == "" {
		return nil, fmt.Errorf("环境变量 LEXIANG_APP_KEY 未设置")
	}
	if appSecret == "" {
		return nil, fmt.Errorf("环境变量 LEXIANG_APP_SECRET 未设置")
	}
	if staffID == "" {
		return nil, fmt.Errorf("环境变量 LEXIANG_STAFF_ID 未设置")
	}

	config := &Config{
		AppKey:    appKey,
		AppSecret: appSecret,
		StaffID:   staffID,
	}

	return NewClient(config)
}

// NewClientWithTokenManager 使用自定义 TokenManager 创建 LexiangClient
// 主要用于测试场景
func NewClientWithTokenManager(tm TokenManager, apiURL string, staffID string) LexiangClient {
	if apiURL == "" {
		apiURL = LexiangAPIURL
	}
	if staffID == "" {
		staffID = "test-staff-id"
	}
	return &lexiangClientImpl{
		apiURL:  apiURL,
		staffID: staffID,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		tokenManager: tm,
	}
}

// GetTokenManager 获取 TokenManager 实例
func (c *lexiangClientImpl) GetTokenManager() TokenManager {
	return c.tokenManager
}

// Do 发送 HTTP 请求
// 自动添加 Authorization Bearer token 和 Content-Type 头
// 当收到 401 响应时，自动使 token 失效并重试一次
func (c *lexiangClientImpl) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	return c.DoWithHeader(ctx, method, path, body, nil)
}

// DoWithHeader 发送 HTTP 请求，支持自定义请求头
// 实现逻辑：
// 1. 获取 token
// 2. 构建请求并设置标准头和自定义头
// 3. 发送请求
// 4. 如果收到 401，使 token 失效并重试一次
func (c *lexiangClientImpl) DoWithHeader(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	// 第一次尝试
	resp, err := c.doRequest(ctx, method, path, body, headers)
	if err != nil {
		return nil, err
	}

	// 检查是否需要重试（401 响应）
	if resp.StatusCode == http.StatusUnauthorized {
		// 关闭第一次响应的 body
		resp.Body.Close()

		// 使 token 失效
		c.tokenManager.InvalidateToken()

		// 重试一次
		resp, err = c.doRequest(ctx, method, path, body, headers)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// doRequest 执行实际的 HTTP 请求
func (c *lexiangClientImpl) doRequest(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	// 获取 token
	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 token 失败: %w", err)
	}

	// 构建请求 URL
	url := c.apiURL + path

	// 序列化请求体
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置标准请求头
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	// 只有写操作（非 GET）才需要 x-staff-id 请求头
	if method != http.MethodGet {
		req.Header.Set("x-staff-id", c.staffID)
	}

	// 设置自定义请求头（可覆盖默认值）
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}

	return resp, nil
}

// Get 发送 GET 请求
func (c *lexiangClientImpl) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, path, nil)
}

// Post 发送 POST 请求
func (c *lexiangClientImpl) Post(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, path, body)
}

// Put 发送 PUT 请求
func (c *lexiangClientImpl) Put(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.Do(ctx, http.MethodPut, path, body)
}

// Delete 发送 DELETE 请求
func (c *lexiangClientImpl) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, path, nil)
}
