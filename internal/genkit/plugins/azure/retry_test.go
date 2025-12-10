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
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetryConfig_shouldRetry 测试重试判断逻辑
func TestRetryConfig_shouldRetry(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		name       string
		statusCode int
		attempt    int
		want       bool
	}{
		{
			name:       "429 第一次尝试应该重试",
			statusCode: http.StatusTooManyRequests,
			attempt:    0,
			want:       true,
		},
		{
			name:       "500 第一次尝试应该重试",
			statusCode: http.StatusInternalServerError,
			attempt:    0,
			want:       true,
		},
		{
			name:       "502 第一次尝试应该重试",
			statusCode: http.StatusBadGateway,
			attempt:    0,
			want:       true,
		},
		{
			name:       "503 第一次尝试应该重试",
			statusCode: http.StatusServiceUnavailable,
			attempt:    0,
			want:       true,
		},
		{
			name:       "504 第一次尝试应该重试",
			statusCode: http.StatusGatewayTimeout,
			attempt:    0,
			want:       true,
		},
		{
			name:       "200 不应该重试",
			statusCode: http.StatusOK,
			attempt:    0,
			want:       false,
		},
		{
			name:       "400 不应该重试",
			statusCode: http.StatusBadRequest,
			attempt:    0,
			want:       false,
		},
		{
			name:       "401 不应该重试",
			statusCode: http.StatusUnauthorized,
			attempt:    0,
			want:       false,
		},
		{
			name:       "超过最大重试次数不应该重试",
			statusCode: http.StatusTooManyRequests,
			attempt:    3,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.shouldRetry(tt.statusCode, tt.attempt)
			if got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRetryConfig_calculateBackoff 测试退避时间计算
func TestRetryConfig_calculateBackoff(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{
			name:    "第一次重试",
			attempt: 0,
			want:    1 * time.Second,
		},
		{
			name:    "第二次重试",
			attempt: 1,
			want:    2 * time.Second,
		},
		{
			name:    "第三次重试",
			attempt: 2,
			want:    4 * time.Second,
		},
		{
			name:    "第四次重试",
			attempt: 3,
			want:    8 * time.Second,
		},
		{
			name:    "超过最大退避时间",
			attempt: 10,
			want:    32 * time.Second, // 应该被限制在 MaxBackoff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.calculateBackoff(tt.attempt)
			if got != tt.want {
				t.Errorf("calculateBackoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRetryableHTTPClient_Do_Success 测试成功的请求
func TestRetryableHTTPClient_Do_Success(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), DefaultRetryConfig())

	// 创建请求
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		t.Errorf("状态码 = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestRetryableHTTPClient_Do_RetryOn429 测试 429 错误的重试
func TestRetryableHTTPClient_Do_RetryOn429(t *testing.T) {
	var attemptCount int32

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 3 {
			// 前两次返回 429
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "too many requests"}`))
		} else {
			// 第三次返回成功
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	// 创建重试配置（减少退避时间以加快测试）
	retryConfig := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests: true,
		},
	}

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), retryConfig)

	// 创建请求
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		t.Errorf("状态码 = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// 验证重试次数
	if attemptCount != 3 {
		t.Errorf("尝试次数 = %d, want 3", attemptCount)
	}
}

// TestRetryableHTTPClient_Do_RetryOn500 测试 500 错误的重试
func TestRetryableHTTPClient_Do_RetryOn500(t *testing.T) {
	var attemptCount int32

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 2 {
			// 第一次返回 500
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
		} else {
			// 第二次返回成功
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	// 创建重试配置
	retryConfig := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusInternalServerError: true,
		},
	}

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), retryConfig)

	// 创建请求
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		t.Errorf("状态码 = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// 验证重试次数
	if attemptCount != 2 {
		t.Errorf("尝试次数 = %d, want 2", attemptCount)
	}
}

// TestRetryableHTTPClient_Do_NoRetryOn400 测试 400 错误不重试
func TestRetryableHTTPClient_Do_NoRetryOn400(t *testing.T) {
	var attemptCount int32

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), DefaultRetryConfig())

	// 创建请求
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("状态码 = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// 验证只尝试了一次
	if attemptCount != 1 {
		t.Errorf("尝试次数 = %d, want 1", attemptCount)
	}
}

// TestRetryableHTTPClient_Do_MaxRetriesExceeded 测试超过最大重试次数
func TestRetryableHTTPClient_Do_MaxRetriesExceeded(t *testing.T) {
	var attemptCount int32

	// 创建测试服务器（始终返回 429）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "too many requests"}`))
	}))
	defer server.Close()

	// 创建重试配置
	retryConfig := &RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests: true,
		},
	}

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), retryConfig)

	// 创建请求
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应（应该是最后一次的 429）
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("状态码 = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}

	// 验证尝试次数（初始 + 2 次重试 = 3 次）
	if attemptCount != 3 {
		t.Errorf("尝试次数 = %d, want 3", attemptCount)
	}
}

// TestRetryableHTTPClient_Do_ContextCancellation 测试上下文取消
func TestRetryableHTTPClient_Do_ContextCancellation(t *testing.T) {
	// 创建测试服务器（始终返回 429）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "too many requests"}`))
	}))
	defer server.Close()

	// 创建重试配置
	retryConfig := &RetryConfig{
		MaxRetries:        10,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests: true,
		},
	}

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), retryConfig)

	// 创建可取消的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	_, err = client.Do(req)

	// 验证错误（应该是上下文取消错误）
	if err == nil {
		t.Error("期望返回错误，但没有错误")
	}
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("错误类型 = %v, want context.DeadlineExceeded or context.Canceled", err)
	}
}

// TestNewHTTPClientWithTimeout 测试创建配置了超时的 HTTP 客户端
func TestNewHTTPClientWithTimeout(t *testing.T) {
	config := &TimeoutConfig{
		RequestTimeout:      10 * time.Second,
		StreamTimeout:       20 * time.Second,
		DialTimeout:         5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		IdleConnTimeout:     30 * time.Second,
	}

	client := NewHTTPClientWithTimeout(config)

	if client == nil {
		t.Fatal("客户端不应该为 nil")
	}

	if client.Timeout != config.RequestTimeout {
		t.Errorf("客户端超时 = %v, want %v", client.Timeout, config.RequestTimeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("传输层应该是 *http.Transport 类型")
	}

	if transport.IdleConnTimeout != config.IdleConnTimeout {
		t.Errorf("空闲连接超时 = %v, want %v", transport.IdleConnTimeout, config.IdleConnTimeout)
	}

	if transport.TLSHandshakeTimeout != config.TLSHandshakeTimeout {
		t.Errorf("TLS 握手超时 = %v, want %v", transport.TLSHandshakeTimeout, config.TLSHandshakeTimeout)
	}
}

// TestWithTimeout 测试添加超时上下文
func TestWithTimeout(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	newCtx, cancel := WithTimeout(ctx, timeout)
	defer cancel()

	deadline, ok := newCtx.Deadline()
	if !ok {
		t.Error("上下文应该有截止时间")
	}

	// 验证截止时间大约在 5 秒后
	expectedDeadline := time.Now().Add(timeout)
	diff := deadline.Sub(expectedDeadline)
	if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
		t.Errorf("截止时间差异 = %v, 应该接近 0", diff)
	}
}

// BenchmarkRetryableHTTPClient_Do 基准测试重试客户端
func BenchmarkRetryableHTTPClient_Do(b *testing.B) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), DefaultRetryConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("请求失败: %v", err)
		}
		resp.Body.Close()
	}
}

// TestRetryableHTTPClient_Do_ExponentialBackoff 测试指数退避
func TestRetryableHTTPClient_Do_ExponentialBackoff(t *testing.T) {
	var attemptCount int32
	var attemptTimes []time.Time

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptTimes = append(attemptTimes, time.Now())
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 4 {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// 创建重试配置
	retryConfig := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests: true,
		},
	}

	// 创建重试客户端
	client := NewRetryableHTTPClient(server.Client(), retryConfig)

	// 创建请求
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证尝试次数
	if attemptCount != 4 {
		t.Errorf("尝试次数 = %d, want 4", attemptCount)
	}

	// 验证退避时间（允许一些误差）
	if len(attemptTimes) >= 2 {
		firstBackoff := attemptTimes[1].Sub(attemptTimes[0])
		if firstBackoff < 90*time.Millisecond || firstBackoff > 150*time.Millisecond {
			t.Errorf("第一次退避时间 = %v, 期望约 100ms", firstBackoff)
		}
	}

	if len(attemptTimes) >= 3 {
		secondBackoff := attemptTimes[2].Sub(attemptTimes[1])
		if secondBackoff < 180*time.Millisecond || secondBackoff > 250*time.Millisecond {
			t.Errorf("第二次退避时间 = %v, 期望约 200ms", secondBackoff)
		}
	}
}

// Example_retryableHTTPClient 示例：使用重试客户端
func Example_retryableHTTPClient() {
	// 创建重试配置
	retryConfig := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        32 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests:     true,
			http.StatusInternalServerError: true,
		},
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建重试客户端
	client := NewRetryableHTTPClient(httpClient, retryConfig)

	// 创建请求
	req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)

	// 执行请求（自动重试）
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("状态码: %d\n", resp.StatusCode)
}
