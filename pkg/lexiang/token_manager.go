// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TokenManager Token 管理器接口
// 负责 access_token 的获取、缓存和自动刷新
type TokenManager interface {
	// GetToken 获取有效的 access_token（自动处理刷新）
	// 如果 token 有效且距离过期时间大于 RefreshBuffer，返回缓存的 token
	// 如果 token 无效或临近过期，自动刷新并返回新 token
	GetToken(ctx context.Context) (string, error)

	// InvalidateToken 强制使 token 失效（用于 401 重试场景）
	InvalidateToken()
}

// tokenManagerImpl TokenManager 的实现
type tokenManagerImpl struct {
	// 配置
	appKey        string
	appSecret     string
	baseURL       string
	timeout       time.Duration
	refreshBuffer time.Duration

	// HTTP 客户端
	httpClient *http.Client

	// Token 缓存
	mu        sync.RWMutex
	token     string
	expiresAt time.Time

	// 刷新锁，确保并发刷新时只执行一次
	refreshMu sync.Mutex
}

// NewTokenManager 创建新的 TokenManager 实例
// config 必须包含 AppKey 和 AppSecret
func NewTokenManager(config *Config) (TokenManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config 不能为空")
	}
	if config.AppKey == "" {
		return nil, fmt.Errorf("AppKey 不能为空")
	}
	if config.AppSecret == "" {
		return nil, fmt.Errorf("AppSecret 不能为空")
	}

	// 使用默认值填充未设置的配置
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = LexiangBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	refreshBuffer := config.RefreshBuffer
	if refreshBuffer == 0 {
		refreshBuffer = TokenRefreshBuffer
	}

	return &tokenManagerImpl{
		appKey:        config.AppKey,
		appSecret:     config.AppSecret,
		baseURL:       baseURL,
		timeout:       timeout,
		refreshBuffer: refreshBuffer,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// GetToken 获取有效的 access_token
// 实现逻辑：
// 1. 首先检查缓存的 token 是否有效且未临近过期
// 2. 如果有效，直接返回缓存的 token
// 3. 如果无效或临近过期，调用 refreshToken 刷新
func (tm *tokenManagerImpl) GetToken(ctx context.Context) (string, error) {
	// 快速路径：读锁检查缓存
	tm.mu.RLock()
	if tm.isTokenValid() {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// 慢路径：需要刷新 token
	return tm.refreshToken(ctx)
}

// InvalidateToken 强制使 token 失效
// 用于 401 响应后触发重新获取 token
func (tm *tokenManagerImpl) InvalidateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = ""
	tm.expiresAt = time.Time{}
}

// isTokenValid 检查 token 是否有效
// 有效条件：token 非空且距离过期时间大于 refreshBuffer
// 注意：调用此方法前必须持有读锁或写锁
func (tm *tokenManagerImpl) isTokenValid() bool {
	if tm.token == "" {
		return false
	}
	// 检查是否临近过期（提前 refreshBuffer 时间刷新）
	return time.Now().Add(tm.refreshBuffer).Before(tm.expiresAt)
}

// refreshToken 刷新 token（带双重检查锁）
// 确保多个 goroutine 同时请求时只执行一次刷新操作
func (tm *tokenManagerImpl) refreshToken(ctx context.Context) (string, error) {
	// 获取刷新锁，确保只有一个 goroutine 执行刷新
	tm.refreshMu.Lock()
	defer tm.refreshMu.Unlock()

	// 双重检查：可能其他 goroutine 已经刷新完成
	tm.mu.RLock()
	if tm.isTokenValid() {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// 执行实际的 token 刷新
	token, expiresIn, err := tm.fetchToken(ctx)
	if err != nil {
		return "", err
	}

	// 更新缓存
	tm.mu.Lock()
	tm.token = token
	// 根据 expires_in 计算过期时间
	tm.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	tm.mu.Unlock()

	return token, nil
}

// fetchToken 从乐享 API 获取新的 token
func (tm *tokenManagerImpl) fetchToken(ctx context.Context) (string, int, error) {
	// 构建请求体
	reqBody := tokenRequest{
		GrantType: "client_credentials",
		AppKey:    tm.appKey,
		AppSecret: tm.appSecret,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("序列化 token 请求失败: %w", err)
	}

	// 构建 HTTP 请求
	url := tm.baseURL + "/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("创建 token 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// 发送请求
	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("发送 token 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", 0, handleAPIError(resp)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("读取 token 响应失败: %w", err)
	}

	// 解析响应
	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("解析 token 响应失败: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("token 响应中 access_token 为空")
	}

	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}
