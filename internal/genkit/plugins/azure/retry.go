// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int

	// InitialBackoff 初始退避时间
	InitialBackoff time.Duration

	// MaxBackoff 最大退避时间
	MaxBackoff time.Duration

	// BackoffMultiplier 退避时间倍数
	BackoffMultiplier float64

	// RetryableStatusCodes 可重试的 HTTP 状态码
	RetryableStatusCodes map[int]bool
}

// DefaultRetryConfig 返回默认的重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        32 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests:     true, // 429
			http.StatusInternalServerError: true, // 500
			http.StatusBadGateway:          true, // 502
			http.StatusServiceUnavailable:  true, // 503
			http.StatusGatewayTimeout:      true, // 504
		},
	}
}

// shouldRetry 判断是否应该重试
func (c *RetryConfig) shouldRetry(statusCode int, attempt int) bool {
	// 检查是否超过最大重试次数
	if attempt >= c.MaxRetries {
		return false
	}

	// 检查状态码是否可重试
	return c.RetryableStatusCodes[statusCode]
}

// calculateBackoff 计算退避时间（指数退避）
func (c *RetryConfig) calculateBackoff(attempt int) time.Duration {
	// 计算指数退避时间
	backoff := float64(c.InitialBackoff) * math.Pow(c.BackoffMultiplier, float64(attempt))

	// 限制最大退避时间
	if backoff > float64(c.MaxBackoff) {
		backoff = float64(c.MaxBackoff)
	}

	return time.Duration(backoff)
}

// RetryableHTTPClient 支持重试的 HTTP 客户端包装器
type RetryableHTTPClient struct {
	client      *http.Client
	retryConfig *RetryConfig
}

// NewRetryableHTTPClient 创建一个支持重试的 HTTP 客户端
func NewRetryableHTTPClient(client *http.Client, retryConfig *RetryConfig) *RetryableHTTPClient {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	return &RetryableHTTPClient{
		client:      client,
		retryConfig: retryConfig,
	}
}

// Do 执行 HTTP 请求，支持自动重试
func (c *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}

		// 执行请求
		resp, lastErr = c.client.Do(req)

		// 如果请求成功
		if lastErr == nil {
			// 检查状态码
			if !c.retryConfig.shouldRetry(resp.StatusCode, attempt) {
				// 不需要重试，返回响应
				return resp, nil
			}

			// 需要重试，关闭响应体
			resp.Body.Close()

			// 如果是最后一次尝试，返回响应
			if attempt >= c.retryConfig.MaxRetries {
				return resp, nil
			}

			// 计算退避时间
			backoff := c.retryConfig.calculateBackoff(attempt)

			// 等待退避时间
			select {
			case <-time.After(backoff):
				// 继续重试
				continue
			case <-req.Context().Done():
				// 上下文已取消
				return nil, req.Context().Err()
			}
		}

		// 如果是网络错误且不是最后一次尝试
		if attempt < c.retryConfig.MaxRetries {
			// 计算退避时间
			backoff := c.retryConfig.calculateBackoff(attempt)

			// 等待退避时间
			select {
			case <-time.After(backoff):
				// 继续重试
				continue
			case <-req.Context().Done():
				// 上下文已取消
				return nil, req.Context().Err()
			}
		}
	}

	// 所有重试都失败了
	if lastErr != nil {
		return nil, fmt.Errorf("请求失败，已重试 %d 次: %w", c.retryConfig.MaxRetries, lastErr)
	}

	// 返回最后一次响应（状态码不可重试）
	return resp, nil
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	// RequestTimeout 单个请求的超时时间
	RequestTimeout time.Duration

	// StreamTimeout 流式请求的超时时间
	StreamTimeout time.Duration

	// DialTimeout 连接超时时间
	DialTimeout time.Duration

	// TLSHandshakeTimeout TLS 握手超时时间
	TLSHandshakeTimeout time.Duration

	// IdleConnTimeout 空闲连接超时时间
	IdleConnTimeout time.Duration
}

// DefaultTimeoutConfig 返回默认的超时配置
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		RequestTimeout:      30 * time.Second,
		StreamTimeout:       60 * time.Second,
		DialTimeout:         10 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}
}

// NewHTTPClientWithTimeout 创建一个配置了超时的 HTTP 客户端
func NewHTTPClientWithTimeout(timeoutConfig *TimeoutConfig) *http.Client {
	if timeoutConfig == nil {
		timeoutConfig = DefaultTimeoutConfig()
	}

	return &http.Client{
		Timeout: timeoutConfig.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     timeoutConfig.IdleConnTimeout,
			TLSHandshakeTimeout: timeoutConfig.TLSHandshakeTimeout,
			DisableKeepAlives:   false,
		},
	}
}

// WithTimeout 为请求添加超时上下文
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}
