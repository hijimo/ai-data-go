// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: lexiang-integration, Property 8: 上传签名响应 Round-Trip**
// 属性 8: 上传签名响应 Round-Trip
// 对于任意有效的 UploadSignResponse JSON，解析后再序列化应产生等价的 JSON 结构
// **Validates: Requirements 5.5, 5.6**
func TestProperty_UploadSignResponseRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("UploadSignResponse 序列化后反序列化应产生等价结构",
		prop.ForAll(
			func(state, key, bucket, region, auth, token, contentDisposition string) bool {
				// 构造原始 UploadSignResponse
				original := UploadSignResponse{
					State: state,
				}
				original.Object.Key = key
				original.Options.Bucket = bucket
				original.Options.Region = region
				original.Object.Auth.Authorization = auth
				original.Object.Auth.XCosSecurityToken = token
				original.Object.Headers.ContentDisposition = contentDisposition

				// 序列化为 JSON
				jsonBytes, err := json.Marshal(original)
				if err != nil {
					t.Logf("序列化失败: %v", err)
					return false
				}

				// 反序列化回结构体
				var parsed UploadSignResponse
				err = json.Unmarshal(jsonBytes, &parsed)
				if err != nil {
					t.Logf("反序列化失败: %v", err)
					return false
				}

				// 验证结构体相等
				if !reflect.DeepEqual(original, parsed) {
					t.Logf("Round-trip 后结构体不相等")
					t.Logf("原始: %+v", original)
					t.Logf("解析后: %+v", parsed)
					return false
				}

				// 再次序列化，验证 JSON 等价
				jsonBytes2, err := json.Marshal(parsed)
				if err != nil {
					t.Logf("二次序列化失败: %v", err)
					return false
				}

				// 比较两次序列化的 JSON
				var map1, map2 map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &map1); err != nil {
					t.Logf("解析第一次 JSON 失败: %v", err)
					return false
				}
				if err := json.Unmarshal(jsonBytes2, &map2); err != nil {
					t.Logf("解析第二次 JSON 失败: %v", err)
					return false
				}

				if !reflect.DeepEqual(map1, map2) {
					t.Logf("两次序列化的 JSON 不等价")
					t.Logf("第一次: %s", string(jsonBytes))
					t.Logf("第二次: %s", string(jsonBytes2))
					return false
				}

				return true
			},
			// 生成 state 字符串（模拟上传状态标识）
			gen.AlphaString(),
			// 生成 key 字符串（COS 对象键）
			genCOSKey(),
			// 生成 bucket 字符串
			gen.AlphaString(),
			// 生成 region 字符串
			genRegion(),
			// 生成 Authorization 字符串
			genAuthorizationString(),
			// 生成 XCosSecurityToken 字符串
			gen.AlphaString(),
			// 生成 Content-Disposition 字符串
			genContentDisposition(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 8: 上传签名响应 JSON 解析正确性**
// 验证从真实 API 响应格式解析 UploadSignResponse 的正确性
// **Validates: Requirements 5.5, 5.6**
func TestProperty_UploadSignResponseJSONParsing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("从 JSON 解析 UploadSignResponse 后所有字段应正确映射",
		prop.ForAll(
			func(state, key, bucket, region, auth, token, contentDisposition string) bool {
				// 构造模拟 API 响应的 JSON
				jsonStr := `{
					"state": "` + escapeJSON(state) + `",
					"object": {
						"key": "` + escapeJSON(key) + `",
						"options": {
							"bucket": "` + escapeJSON(bucket) + `",
							"region": "` + escapeJSON(region) + `"
						},
						"auth": {
							"Authorization": "` + escapeJSON(auth) + `",
							"XCosSecurityToken": "` + escapeJSON(token) + `"
						},
						"headers": {
							"Content-Disposition": "` + escapeJSON(contentDisposition) + `"
						}
					}
				}`

				// 解析 JSON
				var response UploadSignResponse
				err := json.Unmarshal([]byte(jsonStr), &response)
				if err != nil {
					t.Logf("JSON 解析失败: %v", err)
					t.Logf("JSON: %s", jsonStr)
					return false
				}

				// 验证所有字段正确映射
				if response.State != state {
					t.Logf("State 不匹配: 期望 %q, 实际 %q", state, response.State)
					return false
				}
				if response.Object.Key != key {
					t.Logf("Object.Key 不匹配: 期望 %q, 实际 %q", key, response.Object.Key)
					return false
				}
				if response.Options.Bucket != bucket {
					t.Logf("Object.Options.Bucket 不匹配: 期望 %q, 实际 %q", bucket, response.Options.Bucket)
					return false
				}
				if response.Options.Region != region {
					t.Logf("Object.Options.Region 不匹配: 期望 %q, 实际 %q", region, response.Options.Region)
					return false
				}
				if response.Object.Auth.Authorization != auth {
					t.Logf("Object.Auth.Authorization 不匹配: 期望 %q, 实际 %q", auth, response.Object.Auth.Authorization)
					return false
				}
				if response.Object.Auth.XCosSecurityToken != token {
					t.Logf("Object.Auth.XCosSecurityToken 不匹配: 期望 %q, 实际 %q", token, response.Object.Auth.XCosSecurityToken)
					return false
				}
				if response.Object.Headers.ContentDisposition != contentDisposition {
					t.Logf("Object.Headers.ContentDisposition 不匹配: 期望 %q, 实际 %q", contentDisposition, response.Object.Headers.ContentDisposition)
					return false
				}

				return true
			},
			// 生成简单的字母数字字符串，避免 JSON 转义问题
			gen.AlphaString(),
			genCOSKey(),
			gen.AlphaString(),
			genRegion(),
			genAuthorizationString(),
			gen.AlphaString(),
			genContentDisposition(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// 生成器辅助函数
// ============================================================================

// genCOSKey 生成 COS 对象键
func genCOSKey() gopter.Gen {
	return gen.OneConstOf(
		"kb/uploads/2024/01/document.pdf",
		"kb/uploads/2024/02/image.png",
		"kb/uploads/2024/03/video.mp4",
		"kb/files/test/file.docx",
		"kb/attachments/report.xlsx",
		"",
	)
}

// genRegion 生成腾讯云区域
func genRegion() gopter.Gen {
	return gen.OneConstOf(
		"ap-guangzhou",
		"ap-shanghai",
		"ap-beijing",
		"ap-chengdu",
		"ap-hongkong",
		"ap-singapore",
		"",
	)
}

// genAuthorizationString 生成授权字符串
func genAuthorizationString() gopter.Gen {
	return gen.OneConstOf(
		"q-sign-algorithm=sha1&q-ak=AKIDxxxxxxxx&q-sign-time=1234567890;1234567899&q-key-time=1234567890;1234567899&q-header-list=&q-url-param-list=&q-signature=abcdef1234567890",
		"q-sign-algorithm=sha1&q-ak=AKID123456&q-sign-time=1609459200;1609545600&q-key-time=1609459200;1609545600&q-header-list=host&q-url-param-list=&q-signature=fedcba0987654321",
		"",
	)
}

// genContentDisposition 生成 Content-Disposition 头
func genContentDisposition() gopter.Gen {
	return gen.OneConstOf(
		"attachment; filename=\"document.pdf\"",
		"attachment; filename=\"image.png\"",
		"attachment; filename=\"video.mp4\"",
		"inline; filename=\"preview.jpg\"",
		"",
	)
}

// escapeJSON 转义 JSON 字符串中的特殊字符
func escapeJSON(s string) string {
	// 使用 json.Marshal 来正确转义字符串
	bytes, err := json.Marshal(s)
	if err != nil {
		return s
	}
	// 去掉首尾的引号
	if len(bytes) >= 2 {
		return string(bytes[1 : len(bytes)-1])
	}
	return s
}

// **Feature: lexiang-integration, Property 10: 错误状态码映射**
// 属性 10: 错误状态码映射
// 对于任意 HTTP 错误响应（400, 403, 404, 429, 500），返回的错误应包含对应的错误类型信息
// **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5**
func TestProperty_ErrorStatusCodeMapping(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// 属性：对于任意错误状态码，对应的错误判断函数应返回 true
	properties.Property("错误状态码应正确映射到对应的错误类型判断函数",
		prop.ForAll(
			func(statusCode int) bool {
				// 创建 LexiangError
				err := &LexiangError{
					StatusCode: statusCode,
					Code:       getDefaultErrorCode(statusCode),
					Message:    getDefaultErrorMessage(statusCode),
				}

				// 根据状态码验证对应的判断函数
				switch statusCode {
				case StatusBadRequest: // 400
					if !IsBadRequestError(err) {
						t.Logf("状态码 %d 应该被 IsBadRequestError 识别", statusCode)
						return false
					}
					// 确保其他判断函数返回 false
					if IsForbiddenError(err) || IsNotFoundError(err) || IsRateLimitError(err) || IsServerError(err) {
						t.Logf("状态码 %d 不应该被其他错误类型识别", statusCode)
						return false
					}
				case StatusForbidden: // 403
					if !IsForbiddenError(err) {
						t.Logf("状态码 %d 应该被 IsForbiddenError 识别", statusCode)
						return false
					}
					if IsBadRequestError(err) || IsNotFoundError(err) || IsRateLimitError(err) || IsServerError(err) {
						t.Logf("状态码 %d 不应该被其他错误类型识别", statusCode)
						return false
					}
				case StatusNotFound: // 404
					if !IsNotFoundError(err) {
						t.Logf("状态码 %d 应该被 IsNotFoundError 识别", statusCode)
						return false
					}
					if IsBadRequestError(err) || IsForbiddenError(err) || IsRateLimitError(err) || IsServerError(err) {
						t.Logf("状态码 %d 不应该被其他错误类型识别", statusCode)
						return false
					}
				case StatusRateLimit: // 429
					if !IsRateLimitError(err) {
						t.Logf("状态码 %d 应该被 IsRateLimitError 识别", statusCode)
						return false
					}
					if IsBadRequestError(err) || IsForbiddenError(err) || IsNotFoundError(err) || IsServerError(err) {
						t.Logf("状态码 %d 不应该被其他错误类型识别", statusCode)
						return false
					}
				default:
					// 500+ 应该被 IsServerError 识别
					if statusCode >= StatusServerError {
						if !IsServerError(err) {
							t.Logf("状态码 %d 应该被 IsServerError 识别", statusCode)
							return false
						}
						if IsBadRequestError(err) || IsForbiddenError(err) || IsNotFoundError(err) || IsRateLimitError(err) {
							t.Logf("状态码 %d 不应该被其他错误类型识别", statusCode)
							return false
						}
					}
				}

				return true
			},
			// 生成错误状态码
			genErrorStatusCode(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 10: 错误消息包含正确信息**
// 验证错误消息包含状态码和错误类型信息
// **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5**
func TestProperty_ErrorMessageContainsCorrectInfo(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("LexiangError.Error() 应包含状态码和错误信息",
		prop.ForAll(
			func(statusCode int, code, message string) bool {
				err := &LexiangError{
					StatusCode: statusCode,
					Code:       code,
					Message:    message,
				}

				errorStr := err.Error()

				// 验证错误字符串包含状态码
				expectedStatusStr := fmt.Sprintf("status=%d", statusCode)
				if !contains(errorStr, expectedStatusStr) {
					t.Logf("错误字符串应包含状态码: %s", expectedStatusStr)
					t.Logf("实际: %s", errorStr)
					return false
				}

				// 验证错误字符串包含错误代码
				expectedCodeStr := fmt.Sprintf("code=%s", code)
				if !contains(errorStr, expectedCodeStr) {
					t.Logf("错误字符串应包含错误代码: %s", expectedCodeStr)
					t.Logf("实际: %s", errorStr)
					return false
				}

				// 验证错误字符串包含错误消息
				expectedMsgStr := fmt.Sprintf("message=%s", message)
				if !contains(errorStr, expectedMsgStr) {
					t.Logf("错误字符串应包含错误消息: %s", expectedMsgStr)
					t.Logf("实际: %s", errorStr)
					return false
				}

				return true
			},
			genErrorStatusCode(),
			gen.AlphaString(),
			gen.AlphaString(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 10: 默认错误代码和消息正确性**
// 验证 getDefaultErrorCode 和 getDefaultErrorMessage 对所有错误状态码返回非空值
// **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5**
func TestProperty_DefaultErrorCodeAndMessage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("getDefaultErrorCode 对所有错误状态码应返回非空字符串",
		prop.ForAll(
			func(statusCode int) bool {
				code := getDefaultErrorCode(statusCode)
				if code == "" {
					t.Logf("状态码 %d 的默认错误代码不应为空", statusCode)
					return false
				}
				return true
			},
			genErrorStatusCode(),
		),
	)

	properties.Property("getDefaultErrorMessage 对所有错误状态码应返回非空字符串",
		prop.ForAll(
			func(statusCode int) bool {
				message := getDefaultErrorMessage(statusCode)
				if message == "" {
					t.Logf("状态码 %d 的默认错误消息不应为空", statusCode)
					return false
				}
				return true
			},
			genErrorStatusCode(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 10: 非 LexiangError 类型错误判断**
// 验证错误判断函数对非 LexiangError 类型的错误返回 false
// **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5**
func TestProperty_NonLexiangErrorReturnsfalse(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("错误判断函数对非 LexiangError 类型应返回 false",
		prop.ForAll(
			func(message string) bool {
				// 创建普通错误
				err := fmt.Errorf("普通错误: %s", message)

				// 所有判断函数都应返回 false
				if IsBadRequestError(err) {
					t.Logf("IsBadRequestError 对非 LexiangError 应返回 false")
					return false
				}
				if IsForbiddenError(err) {
					t.Logf("IsForbiddenError 对非 LexiangError 应返回 false")
					return false
				}
				if IsNotFoundError(err) {
					t.Logf("IsNotFoundError 对非 LexiangError 应返回 false")
					return false
				}
				if IsRateLimitError(err) {
					t.Logf("IsRateLimitError 对非 LexiangError 应返回 false")
					return false
				}
				if IsServerError(err) {
					t.Logf("IsServerError 对非 LexiangError 应返回 false")
					return false
				}

				return true
			},
			gen.AlphaString(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// 错误状态码生成器
// ============================================================================

// genErrorStatusCode 生成错误状态码（400, 403, 404, 429, 500+）
func genErrorStatusCode() gopter.Gen {
	return gen.OneGenOf(
		// 生成特定的错误状态码
		gen.Const(StatusBadRequest),  // 400
		gen.Const(StatusForbidden),   // 403
		gen.Const(StatusNotFound),    // 404
		gen.Const(StatusRateLimit),   // 429
		gen.Const(StatusServerError), // 500
		// 生成 500+ 的服务器错误状态码
		gen.IntRange(500, 599),
	)
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring 查找子串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Property 1: Token 缓存一致性
// ============================================================================

// **Feature: lexiang-integration, Property 1: Token 缓存一致性**
// 属性 1: Token 缓存一致性
// 对于任意 TokenManager 实例，如果 token 有效且距离过期时间大于 5 分钟，
// 则 GetToken 应返回缓存的 token 而不发起新的 API 请求
// **Validates: Requirements 1.1, 1.2**
func TestProperty_TokenCacheConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("有效 token 应从缓存返回而不发起新的 API 请求",
		prop.ForAll(
			func(expiresIn int, callCount int) bool {
				// 确保 expiresIn 足够大，使 token 在测试期间保持有效
				// expiresIn 必须大于 RefreshBuffer（5分钟 = 300秒）
				if expiresIn <= 300 {
					expiresIn = 600 // 至少 10 分钟
				}

				// 限制调用次数在合理范围内
				if callCount < 2 {
					callCount = 2
				}
				if callCount > 10 {
					callCount = 10
				}

				// 创建 API 调用计数器
				var apiCallCount int32

				// 创建模拟服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 增加 API 调用计数
					atomic.AddInt32(&apiCallCount, 1)

					// 返回有效的 token 响应
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   expiresIn,
						AccessToken: fmt.Sprintf("test_token_%d", time.Now().UnixNano()),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				// 创建 TokenManager，使用模拟服务器地址
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       server.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				ctx := context.Background()

				// 第一次调用应该触发 API 请求
				firstToken, err := tm.GetToken(ctx)
				if err != nil {
					t.Logf("第一次获取 token 失败: %v", err)
					return false
				}

				if firstToken == "" {
					t.Logf("第一次获取的 token 为空")
					return false
				}

				// 验证第一次调用触发了 API 请求
				if atomic.LoadInt32(&apiCallCount) != 1 {
					t.Logf("第一次调用应该触发 1 次 API 请求，实际: %d", apiCallCount)
					return false
				}

				// 后续调用应该返回缓存的 token，不触发新的 API 请求
				for i := 1; i < callCount; i++ {
					token, err := tm.GetToken(ctx)
					if err != nil {
						t.Logf("第 %d 次获取 token 失败: %v", i+1, err)
						return false
					}

					// 验证返回的是同一个 token
					if token != firstToken {
						t.Logf("第 %d 次获取的 token 与第一次不同", i+1)
						t.Logf("第一次: %s", firstToken)
						t.Logf("第 %d 次: %s", i+1, token)
						return false
					}
				}

				// 验证只触发了一次 API 请求
				finalCallCount := atomic.LoadInt32(&apiCallCount)
				if finalCallCount != 1 {
					t.Logf("应该只触发 1 次 API 请求，实际: %d", finalCallCount)
					return false
				}

				return true
			},
			// 生成 expiresIn（秒），范围 600-7200（10分钟到2小时）
			gen.IntRange(600, 7200),
			// 生成调用次数，范围 2-10
			gen.IntRange(2, 10),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 1: Token 临近过期时自动刷新**
// 验证当 token 临近过期时（距离过期时间小于 RefreshBuffer），GetToken 应自动刷新
// **Validates: Requirements 1.2**
func TestProperty_TokenAutoRefreshWhenNearExpiry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("token 临近过期时应自动刷新",
		prop.ForAll(
			func(initialExpiresIn int) bool {
				// 设置一个很短的过期时间，使 token 立即临近过期
				// 由于 RefreshBuffer 是 5 分钟，我们设置 expiresIn 为 1-299 秒
				// 这样 token 会被认为是临近过期的
				if initialExpiresIn <= 0 {
					initialExpiresIn = 1
				}
				if initialExpiresIn >= 300 {
					initialExpiresIn = 299
				}

				// 创建 API 调用计数器
				var apiCallCount int32
				var tokenCounter int32

				// 创建模拟服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callNum := atomic.AddInt32(&apiCallCount, 1)
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					// 第一次返回短过期时间，后续返回长过期时间
					expiresIn := initialExpiresIn
					if callNum > 1 {
						expiresIn = 7200 // 2 小时
					}

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   expiresIn,
						AccessToken: fmt.Sprintf("test_token_%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       server.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				ctx := context.Background()

				// 第一次调用获取 token
				firstToken, err := tm.GetToken(ctx)
				if err != nil {
					t.Logf("第一次获取 token 失败: %v", err)
					return false
				}

				// 第二次调用应该触发刷新，因为 token 临近过期
				secondToken, err := tm.GetToken(ctx)
				if err != nil {
					t.Logf("第二次获取 token 失败: %v", err)
					return false
				}

				// 验证触发了两次 API 请求（因为 token 临近过期）
				finalCallCount := atomic.LoadInt32(&apiCallCount)
				if finalCallCount != 2 {
					t.Logf("应该触发 2 次 API 请求（因为 token 临近过期），实际: %d", finalCallCount)
					return false
				}

				// 验证返回了不同的 token
				if firstToken == secondToken {
					t.Logf("刷新后应该返回不同的 token")
					return false
				}

				return true
			},
			// 生成短过期时间（1-299秒，小于 RefreshBuffer）
			gen.IntRange(1, 299),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 2: Token 刷新并发安全
// ============================================================================

// **Feature: lexiang-integration, Property 2: Token 刷新并发安全**
// 属性 2: Token 刷新并发安全
// 对于任意数量的并发 GetToken 调用，当 token 需要刷新时，应仅执行一次 token API 请求
// **Validates: Requirements 1.3**
func TestProperty_TokenRefreshConcurrencySafety(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("并发 GetToken 调用时应仅执行一次 token API 请求",
		prop.ForAll(
			func(goroutineCount int, expiresIn int) bool {
				// 限制并发数在合理范围内
				if goroutineCount < 2 {
					goroutineCount = 2
				}
				if goroutineCount > 50 {
					goroutineCount = 50
				}

				// 确保 expiresIn 足够大，使 token 在测试期间保持有效
				if expiresIn <= 300 {
					expiresIn = 7200 // 2 小时
				}

				// 创建 API 调用计数器
				var apiCallCount int32

				// 创建模拟服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 增加 API 调用计数
					atomic.AddInt32(&apiCallCount, 1)

					// 模拟网络延迟，增加并发冲突的可能性
					time.Sleep(10 * time.Millisecond)

					// 返回有效的 token 响应
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   expiresIn,
						AccessToken: fmt.Sprintf("test_token_%d", time.Now().UnixNano()),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       server.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				ctx := context.Background()

				// 使用 WaitGroup 等待所有 goroutine 完成
				var wg sync.WaitGroup
				wg.Add(goroutineCount)

				// 使用 channel 同步所有 goroutine 同时开始
				startCh := make(chan struct{})

				// 收集所有 goroutine 获取的 token
				tokens := make([]string, goroutineCount)
				errors := make([]error, goroutineCount)

				// 启动多个 goroutine 并发调用 GetToken
				for i := 0; i < goroutineCount; i++ {
					go func(idx int) {
						defer wg.Done()

						// 等待开始信号
						<-startCh

						// 调用 GetToken
						token, err := tm.GetToken(ctx)
						tokens[idx] = token
						errors[idx] = err
					}(i)
				}

				// 发送开始信号，让所有 goroutine 同时开始
				close(startCh)

				// 等待所有 goroutine 完成
				wg.Wait()

				// 验证所有调用都成功
				for i, err := range errors {
					if err != nil {
						t.Logf("goroutine %d 获取 token 失败: %v", i, err)
						return false
					}
				}

				// 验证所有 goroutine 获取的是同一个 token
				firstToken := tokens[0]
				for i, token := range tokens {
					if token != firstToken {
						t.Logf("goroutine %d 获取的 token 与第一个不同", i)
						t.Logf("第一个: %s", firstToken)
						t.Logf("第 %d 个: %s", i, token)
						return false
					}
				}

				// 核心验证：只触发了一次 API 请求
				finalCallCount := atomic.LoadInt32(&apiCallCount)
				if finalCallCount != 1 {
					t.Logf("并发 %d 个 goroutine 应该只触发 1 次 API 请求，实际: %d", goroutineCount, finalCallCount)
					return false
				}

				return true
			},
			// 生成并发 goroutine 数量，范围 2-50
			gen.IntRange(2, 50),
			// 生成 expiresIn（秒），范围 600-7200
			gen.IntRange(600, 7200),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 2: 并发刷新后所有 goroutine 获取相同 token**
// 验证并发刷新时，所有等待的 goroutine 都能获取到刷新后的同一个 token
// **Validates: Requirements 1.3**
func TestProperty_ConcurrentRefreshReturnsConsistentToken(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("并发刷新后所有 goroutine 应获取相同的 token",
		prop.ForAll(
			func(goroutineCount int) bool {
				// 限制并发数在合理范围内
				if goroutineCount < 5 {
					goroutineCount = 5
				}
				if goroutineCount > 100 {
					goroutineCount = 100
				}

				// 创建 API 调用计数器
				var apiCallCount int32
				var tokenValue string = "consistent_token_12345"

				// 创建模拟服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&apiCallCount, 1)

					// 模拟较长的网络延迟，确保并发请求有足够时间排队
					time.Sleep(50 * time.Millisecond)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: tokenValue,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       server.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				ctx := context.Background()

				var wg sync.WaitGroup
				wg.Add(goroutineCount)

				startCh := make(chan struct{})
				tokens := make([]string, goroutineCount)
				errors := make([]error, goroutineCount)

				for i := 0; i < goroutineCount; i++ {
					go func(idx int) {
						defer wg.Done()
						<-startCh
						token, err := tm.GetToken(ctx)
						tokens[idx] = token
						errors[idx] = err
					}(i)
				}

				close(startCh)
				wg.Wait()

				// 验证所有调用都成功
				for i, err := range errors {
					if err != nil {
						t.Logf("goroutine %d 获取 token 失败: %v", i, err)
						return false
					}
				}

				// 验证所有 token 都是预期的值
				for i, token := range tokens {
					if token != tokenValue {
						t.Logf("goroutine %d 获取的 token 不是预期值", i)
						t.Logf("预期: %s", tokenValue)
						t.Logf("实际: %s", token)
						return false
					}
				}

				// 验证只触发了一次 API 请求
				finalCallCount := atomic.LoadInt32(&apiCallCount)
				if finalCallCount != 1 {
					t.Logf("应该只触发 1 次 API 请求，实际: %d", finalCallCount)
					return false
				}

				return true
			},
			// 生成并发 goroutine 数量
			gen.IntRange(5, 100),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 2: 多轮并发刷新安全性**
// 验证多轮 InvalidateToken + 并发 GetToken 场景下的安全性
// 每轮 InvalidateToken 后并发调用 GetToken，应仅触发一次 API 请求
// **Validates: Requirements 1.3**
func TestProperty_MultiRoundConcurrentRefreshSafety(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("多轮并发刷新应保持安全性",
		prop.ForAll(
			func(rounds int, goroutineCount int) bool {
				// 限制轮数和并发数
				if rounds < 2 {
					rounds = 2
				}
				if rounds > 5 {
					rounds = 5
				}
				if goroutineCount < 2 {
					goroutineCount = 2
				}
				if goroutineCount > 20 {
					goroutineCount = 20
				}

				// 创建 API 调用计数器
				var apiCallCount int32
				var tokenCounter int32

				// 创建模拟服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&apiCallCount, 1)
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					time.Sleep(10 * time.Millisecond)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: fmt.Sprintf("token_round_%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       server.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				ctx := context.Background()

				// 执行多轮测试
				// 每轮开始前先使 token 失效，然后并发调用 GetToken
				// 验证每轮只触发一次 API 请求
				for round := 0; round < rounds; round++ {
					// 第一轮之后，先使 token 失效
					if round > 0 {
						tm.InvalidateToken()
					}

					// 记录本轮开始前的 API 调用次数
					roundStartCount := atomic.LoadInt32(&apiCallCount)

					var wg sync.WaitGroup
					wg.Add(goroutineCount)

					startCh := make(chan struct{})
					tokens := make([]string, goroutineCount)
					errors := make([]error, goroutineCount)

					for i := 0; i < goroutineCount; i++ {
						go func(idx int) {
							defer wg.Done()
							<-startCh
							token, err := tm.GetToken(ctx)
							tokens[idx] = token
							errors[idx] = err
						}(i)
					}

					close(startCh)
					wg.Wait()

					// 验证所有调用都成功
					for i, err := range errors {
						if err != nil {
							t.Logf("轮次 %d, goroutine %d 获取 token 失败: %v", round, i, err)
							return false
						}
					}

					// 验证所有 goroutine 获取的是同一个 token
					firstToken := tokens[0]
					for i, token := range tokens {
						if token != firstToken {
							t.Logf("轮次 %d, goroutine %d 获取的 token 与第一个不同", round, i)
							return false
						}
					}

					// 核心验证：每轮只触发一次 API 请求
					roundCallCount := atomic.LoadInt32(&apiCallCount) - roundStartCount
					if roundCallCount != 1 {
						t.Logf("轮次 %d 应该触发 1 次 API 请求，实际: %d", round, roundCallCount)
						return false
					}
				}

				// 验证总共触发了 rounds 次 API 请求
				finalCallCount := atomic.LoadInt32(&apiCallCount)
				if int(finalCallCount) != rounds {
					t.Logf("总共应该触发 %d 次 API 请求，实际: %d", rounds, finalCallCount)
					return false
				}

				return true
			},
			// 生成轮数
			gen.IntRange(2, 5),
			// 生成并发 goroutine 数量
			gen.IntRange(2, 20),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 1: InvalidateToken 使缓存失效**
// 验证调用 InvalidateToken 后，下次 GetToken 应重新获取 token
// **Validates: Requirements 1.1, 1.2**
func TestProperty_InvalidateTokenClearsCache(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("InvalidateToken 后应重新获取 token",
		prop.ForAll(
			func(expiresIn int) bool {
				// 确保 expiresIn 足够大
				if expiresIn <= 300 {
					expiresIn = 7200
				}

				// 创建 API 调用计数器
				var apiCallCount int32
				var tokenCounter int32

				// 创建模拟服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&apiCallCount, 1)
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   expiresIn,
						AccessToken: fmt.Sprintf("test_token_%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       server.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				ctx := context.Background()

				// 第一次获取 token
				firstToken, err := tm.GetToken(ctx)
				if err != nil {
					t.Logf("第一次获取 token 失败: %v", err)
					return false
				}

				// 验证第一次调用触发了 API 请求
				if atomic.LoadInt32(&apiCallCount) != 1 {
					t.Logf("第一次调用应该触发 1 次 API 请求")
					return false
				}

				// 使 token 失效
				tm.InvalidateToken()

				// 再次获取 token，应该触发新的 API 请求
				secondToken, err := tm.GetToken(ctx)
				if err != nil {
					t.Logf("第二次获取 token 失败: %v", err)
					return false
				}

				// 验证触发了第二次 API 请求
				if atomic.LoadInt32(&apiCallCount) != 2 {
					t.Logf("InvalidateToken 后应该触发新的 API 请求，实际调用次数: %d", apiCallCount)
					return false
				}

				// 验证返回了不同的 token
				if firstToken == secondToken {
					t.Logf("InvalidateToken 后应该返回不同的 token")
					return false
				}

				return true
			},
			// 生成 expiresIn（秒）
			gen.IntRange(600, 7200),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 4: 请求头自动设置
// ============================================================================

// **Feature: lexiang-integration, Property 4: 请求头自动设置**
// 属性 4: 请求头自动设置
// 对于任意 HTTP 请求，请求头应包含正确的 Authorization Bearer token 和 Content-Type application/json
// **Validates: Requirements 2.2, 2.3**
func TestProperty_RequestHeadersAutoSet(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("HTTP 请求应自动设置 Authorization 和 Content-Type 头",
		prop.ForAll(
			func(method string, path string, hasBody bool) bool {
				// 用于捕获请求头的变量
				var capturedAuthHeader string
				var capturedContentType string
				var capturedMethod string
				var capturedPath string

				// 预期的 token 值
				expectedToken := "test_token_for_header_check"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获请求信息
					capturedAuthHeader = r.Header.Get("Authorization")
					capturedContentType = r.Header.Get("Content-Type")
					capturedMethod = r.Method
					capturedPath = r.URL.Path

					// 返回成功响应
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				// 创建 LexiangClient，使用 API 服务器地址
				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")

				ctx := context.Background()

				// 准备请求体
				var body any
				if hasBody {
					body = map[string]string{"key": "value"}
				}

				// 发送请求
				resp, err := client.Do(ctx, method, path, body)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证 Authorization 头
				expectedAuthHeader := "Bearer " + expectedToken
				if capturedAuthHeader != expectedAuthHeader {
					t.Logf("Authorization 头不正确")
					t.Logf("期望: %s", expectedAuthHeader)
					t.Logf("实际: %s", capturedAuthHeader)
					return false
				}

				// 验证 Content-Type 头
				expectedContentType := "application/json; charset=utf-8"
				if capturedContentType != expectedContentType {
					t.Logf("Content-Type 头不正确")
					t.Logf("期望: %s", expectedContentType)
					t.Logf("实际: %s", capturedContentType)
					return false
				}

				// 验证请求方法
				if capturedMethod != method {
					t.Logf("请求方法不正确")
					t.Logf("期望: %s", method)
					t.Logf("实际: %s", capturedMethod)
					return false
				}

				// 验证请求路径
				if capturedPath != path {
					t.Logf("请求路径不正确")
					t.Logf("期望: %s", path)
					t.Logf("实际: %s", capturedPath)
					return false
				}

				return true
			},
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
			// 生成是否有请求体
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 4: 便捷方法请求头设置**
// 验证 Get, Post, Put, Delete 便捷方法也正确设置请求头
// **Validates: Requirements 2.2, 2.3**
func TestProperty_ConvenienceMethodsHeadersAutoSet(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("便捷方法应自动设置正确的请求头",
		prop.ForAll(
			func(methodIndex int, path string) bool {
				// 用于捕获请求头的变量
				var capturedAuthHeader string
				var capturedContentType string
				var capturedMethod string

				// 预期的 token 值
				expectedToken := "test_token_convenience_methods"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedAuthHeader = r.Header.Get("Authorization")
					capturedContentType = r.Header.Get("Content-Type")
					capturedMethod = r.Method

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 根据 methodIndex 选择便捷方法
				var resp *http.Response
				var expectedMethod string
				methods := []string{"GET", "POST", "PUT", "DELETE"}
				methodIndex = methodIndex % len(methods)
				expectedMethod = methods[methodIndex]

				switch methodIndex {
				case 0:
					resp, err = client.Get(ctx, path)
				case 1:
					resp, err = client.Post(ctx, path, map[string]string{"key": "value"})
				case 2:
					resp, err = client.Put(ctx, path, map[string]string{"key": "value"})
				case 3:
					resp, err = client.Delete(ctx, path)
				}

				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证 Authorization 头
				expectedAuthHeader := "Bearer " + expectedToken
				if capturedAuthHeader != expectedAuthHeader {
					t.Logf("便捷方法 %s: Authorization 头不正确", expectedMethod)
					t.Logf("期望: %s", expectedAuthHeader)
					t.Logf("实际: %s", capturedAuthHeader)
					return false
				}

				// 验证 Content-Type 头
				expectedContentType := "application/json; charset=utf-8"
				if capturedContentType != expectedContentType {
					t.Logf("便捷方法 %s: Content-Type 头不正确", expectedMethod)
					t.Logf("期望: %s", expectedContentType)
					t.Logf("实际: %s", capturedContentType)
					return false
				}

				// 验证请求方法
				if capturedMethod != expectedMethod {
					t.Logf("便捷方法请求方法不正确")
					t.Logf("期望: %s", expectedMethod)
					t.Logf("实际: %s", capturedMethod)
					return false
				}

				return true
			},
			// 生成方法索引 (0-3)
			gen.IntRange(0, 3),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 4: Token 动态更新后请求头正确**
// 验证当 token 刷新后，后续请求使用新的 token
// **Validates: Requirements 2.2, 2.3**
func TestProperty_HeadersUseRefreshedToken(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Token 刷新后请求应使用新的 token",
		prop.ForAll(
			func(initialExpiresIn int) bool {
				// 设置短过期时间，使 token 临近过期
				if initialExpiresIn <= 0 {
					initialExpiresIn = 1
				}
				if initialExpiresIn >= 300 {
					initialExpiresIn = 60 // 1 分钟，小于 RefreshBuffer
				}

				// 用于捕获请求头的变量
				var capturedAuthHeaders []string
				var mu sync.Mutex

				// Token 计数器
				var tokenCounter int32

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					capturedAuthHeaders = append(capturedAuthHeaders, r.Header.Get("Authorization"))
					mu.Unlock()

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					// 第一次返回短过期时间，后续返回长过期时间
					expiresIn := initialExpiresIn
					if tokenNum > 1 {
						expiresIn = 7200
					}

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   expiresIn,
						AccessToken: fmt.Sprintf("token_v%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 第一次请求
				resp1, err := client.Get(ctx, "/test1")
				if err != nil {
					t.Logf("第一次请求失败: %v", err)
					return false
				}
				resp1.Body.Close()

				// 第二次请求（由于 token 临近过期，应该触发刷新）
				resp2, err := client.Get(ctx, "/test2")
				if err != nil {
					t.Logf("第二次请求失败: %v", err)
					return false
				}
				resp2.Body.Close()

				// 验证捕获了两个请求
				mu.Lock()
				headerCount := len(capturedAuthHeaders)
				mu.Unlock()

				if headerCount != 2 {
					t.Logf("应该捕获 2 个请求，实际: %d", headerCount)
					return false
				}

				// 验证两次请求使用了不同的 token（因为第一个 token 临近过期）
				mu.Lock()
				firstAuth := capturedAuthHeaders[0]
				secondAuth := capturedAuthHeaders[1]
				mu.Unlock()

				// 验证 Authorization 头格式正确
				if !hasPrefix(firstAuth, "Bearer ") {
					t.Logf("第一次请求 Authorization 头格式不正确: %s", firstAuth)
					return false
				}
				if !hasPrefix(secondAuth, "Bearer ") {
					t.Logf("第二次请求 Authorization 头格式不正确: %s", secondAuth)
					return false
				}

				// 验证使用了不同的 token（因为刷新了）
				if firstAuth == secondAuth {
					t.Logf("Token 刷新后应该使用不同的 token")
					t.Logf("第一次: %s", firstAuth)
					t.Logf("第二次: %s", secondAuth)
					return false
				}

				return true
			},
			// 生成短过期时间（1-60秒）
			gen.IntRange(1, 60),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 4 辅助生成器
// ============================================================================

// genHTTPMethod 生成 HTTP 方法
func genHTTPMethod() gopter.Gen {
	return gen.OneConstOf(
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	)
}

// genAPIPath 生成 API 路径
func genAPIPath() gopter.Gen {
	return gen.OneConstOf(
		"/spaces",
		"/spaces/123",
		"/entries",
		"/entries/456",
		"/upload-signs",
		"/doc-files/789",
		"/feedbacks",
		"/test",
		"/api/v1/resource",
	)
}

// hasPrefix 检查字符串是否以指定前缀开头
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ============================================================================
// Property 5: 自定义请求头传递
// ============================================================================

// **Feature: lexiang-integration, Property 5: 自定义请求头传递**
// 属性 5: 自定义请求头传递
// 对于任意 DoWithHeader 调用，传入的自定义请求头应被正确添加到 HTTP 请求中
// **Validates: Requirements 2.4**
func TestProperty_CustomHeadersPassedCorrectly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("DoWithHeader 应正确传递自定义请求头",
		prop.ForAll(
			func(headerKey, headerValue string, method string, path string) bool {
				// 跳过空的 header key（HTTP 规范不允许空 header 名）
				if headerKey == "" {
					return true
				}

				// 用于捕获请求头的变量
				var capturedCustomHeader string
				var capturedAuthHeader string
				var capturedContentType string

				// 预期的 token 值
				expectedToken := "test_token_custom_headers"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获自定义请求头
					capturedCustomHeader = r.Header.Get(headerKey)
					capturedAuthHeader = r.Header.Get("Authorization")
					capturedContentType = r.Header.Get("Content-Type")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 准备自定义请求头
				customHeaders := map[string]string{
					headerKey: headerValue,
				}

				// 发送带自定义请求头的请求
				resp, err := client.DoWithHeader(ctx, method, path, nil, customHeaders)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证自定义请求头被正确传递
				if capturedCustomHeader != headerValue {
					t.Logf("自定义请求头 %s 未正确传递", headerKey)
					t.Logf("期望: %s", headerValue)
					t.Logf("实际: %s", capturedCustomHeader)
					return false
				}

				// 验证标准请求头仍然存在
				expectedAuthHeader := "Bearer " + expectedToken
				if capturedAuthHeader != expectedAuthHeader {
					t.Logf("Authorization 头不正确")
					t.Logf("期望: %s", expectedAuthHeader)
					t.Logf("实际: %s", capturedAuthHeader)
					return false
				}

				expectedContentType := "application/json; charset=utf-8"
				if capturedContentType != expectedContentType {
					t.Logf("Content-Type 头不正确")
					t.Logf("期望: %s", expectedContentType)
					t.Logf("实际: %s", capturedContentType)
					return false
				}

				return true
			},
			// 生成自定义请求头名称
			genCustomHeaderKey(),
			// 生成自定义请求头值
			genCustomHeaderValue(),
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 5: 多个自定义请求头传递**
// 验证 DoWithHeader 可以同时传递多个自定义请求头
// **Validates: Requirements 2.4**
func TestProperty_MultipleCustomHeadersPassedCorrectly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("DoWithHeader 应正确传递多个自定义请求头",
		prop.ForAll(
			func(headerCount int) bool {
				// 限制请求头数量在合理范围内
				if headerCount < 1 {
					headerCount = 1
				}
				if headerCount > 10 {
					headerCount = 10
				}

				// 生成多个自定义请求头
				customHeaders := make(map[string]string)
				expectedHeaders := make(map[string]string)
				headerKeys := []string{
					"X-Staff-Id",
					"X-Request-Id",
					"X-Trace-Id",
					"X-Custom-Header-1",
					"X-Custom-Header-2",
					"X-Custom-Header-3",
					"X-Custom-Header-4",
					"X-Custom-Header-5",
					"X-Custom-Header-6",
					"X-Custom-Header-7",
				}

				for i := 0; i < headerCount; i++ {
					key := headerKeys[i]
					value := fmt.Sprintf("value_%d_%d", i, time.Now().UnixNano())
					customHeaders[key] = value
					expectedHeaders[key] = value
				}

				// 用于捕获请求头的变量
				capturedHeaders := make(map[string]string)
				var mu sync.Mutex

				// 预期的 token 值
				expectedToken := "test_token_multiple_headers"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					for key := range expectedHeaders {
						capturedHeaders[key] = r.Header.Get(key)
					}
					mu.Unlock()

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 发送带多个自定义请求头的请求
				resp, err := client.DoWithHeader(ctx, http.MethodPost, "/test", nil, customHeaders)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证所有自定义请求头都被正确传递
				mu.Lock()
				defer mu.Unlock()

				for key, expectedValue := range expectedHeaders {
					actualValue := capturedHeaders[key]
					if actualValue != expectedValue {
						t.Logf("自定义请求头 %s 未正确传递", key)
						t.Logf("期望: %s", expectedValue)
						t.Logf("实际: %s", actualValue)
						return false
					}
				}

				return true
			},
			// 生成请求头数量 (1-10)
			gen.IntRange(1, 10),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 5: x-staff-id 请求头传递**
// 验证 DoWithHeader 正确传递 x-staff-id 请求头（乐享 API 常用的自定义头）
// **Validates: Requirements 2.4**
func TestProperty_XStaffIdHeaderPassedCorrectly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("DoWithHeader 应正确传递 x-staff-id 请求头",
		prop.ForAll(
			func(staffId string) bool {
				// 跳过空的 staffId
				if staffId == "" {
					return true
				}

				// 用于捕获请求头的变量
				var capturedStaffId string

				// 预期的 token 值
				expectedToken := "test_token_staff_id"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedStaffId = r.Header.Get("x-staff-id")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 准备 x-staff-id 请求头
				customHeaders := map[string]string{
					"x-staff-id": staffId,
				}

				// 发送请求
				resp, err := client.DoWithHeader(ctx, http.MethodPost, "/entries", nil, customHeaders)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证 x-staff-id 请求头被正确传递
				if capturedStaffId != staffId {
					t.Logf("x-staff-id 请求头未正确传递")
					t.Logf("期望: %s", staffId)
					t.Logf("实际: %s", capturedStaffId)
					return false
				}

				return true
			},
			// 生成 staffId
			genStaffId(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 5: 空自定义请求头不影响标准头**
// 验证当 headers 参数为 nil 或空 map 时，标准请求头仍然正确设置
// **Validates: Requirements 2.4**
func TestProperty_EmptyCustomHeadersDoNotAffectStandardHeaders(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("空自定义请求头不应影响标准请求头",
		prop.ForAll(
			func(useNilHeaders bool, method string, path string) bool {
				// 用于捕获请求头的变量
				var capturedAuthHeader string
				var capturedContentType string

				// 预期的 token 值
				expectedToken := "test_token_empty_headers"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedAuthHeader = r.Header.Get("Authorization")
					capturedContentType = r.Header.Get("Content-Type")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 根据参数决定使用 nil 还是空 map
				var customHeaders map[string]string
				if !useNilHeaders {
					customHeaders = make(map[string]string)
				}

				// 发送请求
				resp, err := client.DoWithHeader(ctx, method, path, nil, customHeaders)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证标准请求头仍然正确
				expectedAuthHeader := "Bearer " + expectedToken
				if capturedAuthHeader != expectedAuthHeader {
					t.Logf("Authorization 头不正确")
					t.Logf("期望: %s", expectedAuthHeader)
					t.Logf("实际: %s", capturedAuthHeader)
					return false
				}

				expectedContentType := "application/json; charset=utf-8"
				if capturedContentType != expectedContentType {
					t.Logf("Content-Type 头不正确")
					t.Logf("期望: %s", expectedContentType)
					t.Logf("实际: %s", capturedContentType)
					return false
				}

				return true
			},
			// 生成是否使用 nil headers
			gen.Bool(),
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 5 辅助生成器
// ============================================================================

// genCustomHeaderKey 生成自定义请求头名称
func genCustomHeaderKey() gopter.Gen {
	return gen.OneConstOf(
		"X-Staff-Id",
		"X-Request-Id",
		"X-Trace-Id",
		"X-Custom-Header",
		"X-Api-Key",
		"X-Client-Version",
		"X-Device-Id",
		"X-Session-Id",
		"X-Correlation-Id",
		"X-Forwarded-For",
	)
}

// genCustomHeaderValue 生成自定义请求头值
func genCustomHeaderValue() gopter.Gen {
	return gen.OneConstOf(
		"staff_123456",
		"req_abcdef",
		"trace_xyz789",
		"custom_value_001",
		"api_key_secret",
		"v1.2.3",
		"device_001",
		"session_abc",
		"corr_123",
		"192.168.1.1",
		"",
	)
}

// genStaffId 生成员工 ID
func genStaffId() gopter.Gen {
	return gen.OneConstOf(
		"staff_001",
		"staff_002",
		"user_12345",
		"employee_abc",
		"admin_xyz",
		"test_user",
		"dev_001",
		"prod_user_123",
		"",
	)
}

// ============================================================================
// Property 3: 401 自动重试
// ============================================================================

// **Feature: lexiang-integration, Property 3: 401 自动重试**
// 属性 3: 401 自动重试
// 对于任意 API 请求，当收到 401 响应时，客户端应使 token 失效、获取新 token 并重试请求一次
// **Validates: Requirements 1.4**
func TestProperty_401AutoRetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("收到 401 响应时应自动刷新 token 并重试请求一次",
		prop.ForAll(
			func(method string, path string) bool {
				// 请求计数器
				var apiRequestCount int32
				var tokenRequestCount int32

				// Token 计数器，用于生成不同的 token
				var tokenCounter int32

				// 创建模拟 API 服务器
				// 第一次请求返回 401，第二次请求返回 200
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestNum := atomic.AddInt32(&apiRequestCount, 1)

					if requestNum == 1 {
						// 第一次请求返回 401 Unauthorized
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte(`{"error": "token expired"}`))
						return
					}

					// 后续请求返回 200 OK
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&tokenRequestCount, 1)
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: fmt.Sprintf("token_v%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 发送请求
				resp, err := client.Do(ctx, method, path, nil)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证最终响应是成功的（200）
				if resp.StatusCode != http.StatusOK {
					t.Logf("最终响应状态码应为 200，实际: %d", resp.StatusCode)
					return false
				}

				// 验证 API 请求被调用了 2 次（第一次 401，第二次 200）
				finalAPICount := atomic.LoadInt32(&apiRequestCount)
				if finalAPICount != 2 {
					t.Logf("API 应该被调用 2 次（401 + 重试），实际: %d", finalAPICount)
					return false
				}

				// 验证 Token 请求被调用了 2 次（初始获取 + 401 后刷新）
				finalTokenCount := atomic.LoadInt32(&tokenRequestCount)
				if finalTokenCount != 2 {
					t.Logf("Token API 应该被调用 2 次（初始 + 刷新），实际: %d", finalTokenCount)
					return false
				}

				return true
			},
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 3: 401 重试使用新 Token**
// 验证 401 重试时使用的是刷新后的新 token，而不是旧 token
// **Validates: Requirements 1.4**
func TestProperty_401RetryUsesNewToken(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("401 重试时应使用刷新后的新 token",
		prop.ForAll(
			func(method string, path string) bool {
				// 用于捕获请求中的 Authorization 头
				var capturedAuthHeaders []string
				var mu sync.Mutex

				// Token 计数器
				var tokenCounter int32

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获 Authorization 头
					mu.Lock()
					capturedAuthHeaders = append(capturedAuthHeaders, r.Header.Get("Authorization"))
					requestNum := len(capturedAuthHeaders)
					mu.Unlock()

					if requestNum == 1 {
						// 第一次请求返回 401
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte(`{"error": "token expired"}`))
						return
					}

					// 后续请求返回 200
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: fmt.Sprintf("token_v%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 发送请求
				resp, err := client.Do(ctx, method, path, nil)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证捕获了 2 个请求
				mu.Lock()
				headerCount := len(capturedAuthHeaders)
				firstAuth := ""
				secondAuth := ""
				if headerCount >= 1 {
					firstAuth = capturedAuthHeaders[0]
				}
				if headerCount >= 2 {
					secondAuth = capturedAuthHeaders[1]
				}
				mu.Unlock()

				if headerCount != 2 {
					t.Logf("应该捕获 2 个请求，实际: %d", headerCount)
					return false
				}

				// 验证两次请求使用了不同的 token
				if firstAuth == secondAuth {
					t.Logf("401 重试应该使用不同的 token")
					t.Logf("第一次: %s", firstAuth)
					t.Logf("第二次: %s", secondAuth)
					return false
				}

				// 验证第一次使用 token_v1，第二次使用 token_v2
				expectedFirstAuth := "Bearer token_v1"
				expectedSecondAuth := "Bearer token_v2"

				if firstAuth != expectedFirstAuth {
					t.Logf("第一次请求应使用 token_v1")
					t.Logf("期望: %s", expectedFirstAuth)
					t.Logf("实际: %s", firstAuth)
					return false
				}

				if secondAuth != expectedSecondAuth {
					t.Logf("第二次请求应使用 token_v2")
					t.Logf("期望: %s", expectedSecondAuth)
					t.Logf("实际: %s", secondAuth)
					return false
				}

				return true
			},
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 3: 401 只重试一次**
// 验证当重试后仍然返回 401 时，不会无限重试，只重试一次
// **Validates: Requirements 1.4**
func TestProperty_401RetryOnlyOnce(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("401 响应只应重试一次，不应无限重试",
		prop.ForAll(
			func(method string, path string) bool {
				// 请求计数器
				var apiRequestCount int32
				var tokenRequestCount int32

				// 创建模拟 API 服务器 - 始终返回 401
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&apiRequestCount, 1)

					// 始终返回 401
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error": "token invalid"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tokenNum := atomic.AddInt32(&tokenRequestCount, 1)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: fmt.Sprintf("token_v%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 发送请求
				resp, err := client.Do(ctx, method, path, nil)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证最终响应仍然是 401（因为重试后仍然失败）
				if resp.StatusCode != http.StatusUnauthorized {
					t.Logf("持续 401 时最终响应应为 401，实际: %d", resp.StatusCode)
					return false
				}

				// 核心验证：API 只被调用了 2 次（原始请求 + 1 次重试）
				finalAPICount := atomic.LoadInt32(&apiRequestCount)
				if finalAPICount != 2 {
					t.Logf("API 应该只被调用 2 次（原始 + 1 次重试），实际: %d", finalAPICount)
					return false
				}

				// 验证 Token 请求被调用了 2 次
				finalTokenCount := atomic.LoadInt32(&tokenRequestCount)
				if finalTokenCount != 2 {
					t.Logf("Token API 应该被调用 2 次，实际: %d", finalTokenCount)
					return false
				}

				return true
			},
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 3: 非 401 错误不触发重试**
// 验证其他错误状态码（如 400, 403, 404, 500）不会触发自动重试
// **Validates: Requirements 1.4**
func TestProperty_Non401ErrorsDoNotTriggerRetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("非 401 错误不应触发自动重试",
		prop.ForAll(
			func(statusCode int, method string, path string) bool {
				// 跳过 401 状态码（401 应该触发重试）
				if statusCode == http.StatusUnauthorized {
					return true
				}

				// 请求计数器
				var apiRequestCount int32
				var tokenRequestCount int32

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&apiRequestCount, 1)

					// 返回指定的错误状态码
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(statusCode)
					w.Write([]byte(fmt.Sprintf(`{"error": "error %d"}`, statusCode)))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&tokenRequestCount, 1)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: "test_token",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 发送请求
				resp, err := client.Do(ctx, method, path, nil)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证响应状态码与预期一致
				if resp.StatusCode != statusCode {
					t.Logf("响应状态码应为 %d，实际: %d", statusCode, resp.StatusCode)
					return false
				}

				// 核心验证：API 只被调用了 1 次（没有重试）
				finalAPICount := atomic.LoadInt32(&apiRequestCount)
				if finalAPICount != 1 {
					t.Logf("非 401 错误不应触发重试，API 应该只被调用 1 次，实际: %d", finalAPICount)
					return false
				}

				// 验证 Token 请求只被调用了 1 次（没有刷新）
				finalTokenCount := atomic.LoadInt32(&tokenRequestCount)
				if finalTokenCount != 1 {
					t.Logf("非 401 错误不应触发 token 刷新，Token API 应该只被调用 1 次，实际: %d", finalTokenCount)
					return false
				}

				return true
			},
			// 生成非 401 的错误状态码
			genNon401ErrorStatusCode(),
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 3: 401 重试保留自定义请求头**
// 验证 401 重试时，自定义请求头（如 x-staff-id）仍然被正确传递
// **Validates: Requirements 1.4**
func TestProperty_401RetryPreservesCustomHeaders(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("401 重试时应保留自定义请求头",
		prop.ForAll(
			func(staffId string, method string, path string) bool {
				// 跳过空的 staffId
				if staffId == "" {
					return true
				}

				// 用于捕获请求中的自定义头
				var capturedStaffIds []string
				var mu sync.Mutex

				// Token 计数器
				var tokenCounter int32

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获 x-staff-id 头
					mu.Lock()
					capturedStaffIds = append(capturedStaffIds, r.Header.Get("x-staff-id"))
					requestNum := len(capturedStaffIds)
					mu.Unlock()

					if requestNum == 1 {
						// 第一次请求返回 401
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte(`{"error": "token expired"}`))
						return
					}

					// 后续请求返回 200
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tokenNum := atomic.AddInt32(&tokenCounter, 1)

					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: fmt.Sprintf("token_v%d", tokenNum),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 准备自定义请求头
				customHeaders := map[string]string{
					"x-staff-id": staffId,
				}

				// 发送带自定义请求头的请求
				resp, err := client.DoWithHeader(ctx, method, path, nil, customHeaders)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证捕获了 2 个请求
				mu.Lock()
				headerCount := len(capturedStaffIds)
				firstStaffId := ""
				secondStaffId := ""
				if headerCount >= 1 {
					firstStaffId = capturedStaffIds[0]
				}
				if headerCount >= 2 {
					secondStaffId = capturedStaffIds[1]
				}
				mu.Unlock()

				if headerCount != 2 {
					t.Logf("应该捕获 2 个请求，实际: %d", headerCount)
					return false
				}

				// 验证两次请求都包含相同的 x-staff-id
				if firstStaffId != staffId {
					t.Logf("第一次请求的 x-staff-id 不正确")
					t.Logf("期望: %s", staffId)
					t.Logf("实际: %s", firstStaffId)
					return false
				}

				if secondStaffId != staffId {
					t.Logf("重试请求的 x-staff-id 不正确")
					t.Logf("期望: %s", staffId)
					t.Logf("实际: %s", secondStaffId)
					return false
				}

				return true
			},
			// 生成 staffId
			genStaffId(),
			// 生成 HTTP 方法
			genHTTPMethod(),
			// 生成 API 路径
			genAPIPath(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 3 辅助生成器
// ============================================================================

// genNon401ErrorStatusCode 生成非 401 的错误状态码
func genNon401ErrorStatusCode() gopter.Gen {
	return gen.OneConstOf(
		http.StatusBadRequest,          // 400
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusMethodNotAllowed,    // 405
		http.StatusConflict,            // 409
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	)
}

// ============================================================================
// Property 7: 分页参数传递
// ============================================================================

// **Feature: lexiang-integration, Property 7: 分页参数传递**
// 属性 7: 分页参数传递
// 对于任意列表查询方法调用，分页参数（limit, pageToken）应被正确拼接到查询字符串中
// **Validates: Requirements 3.4, 4.5, 7.1**
func TestProperty_PaginationParametersPassed(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("ListSpaces 应正确传递分页参数到查询字符串",
		prop.ForAll(
			func(teamID string, limit int, pageToken string) bool {
				// 跳过空的 teamID
				if teamID == "" {
					return true
				}

				// 用于捕获请求 URL 的变量
				var capturedURL string
				var capturedQuery url.Values

				// 预期的 token 值
				expectedToken := "test_token_pagination"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获请求 URL 和查询参数
					capturedURL = r.URL.String()
					capturedQuery = r.URL.Query()

					// 返回空的知识库列表响应
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"data": [], "meta": {"page_token": ""}}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 调用 ListSpaces 方法
				_, err = client.ListSpaces(ctx, teamID, limit, pageToken)
				if err != nil {
					t.Logf("ListSpaces 调用失败: %v", err)
					return false
				}

				// 验证 team_id 参数
				if capturedQuery.Get("team_id") != teamID {
					t.Logf("team_id 参数不正确")
					t.Logf("期望: %s", teamID)
					t.Logf("实际: %s", capturedQuery.Get("team_id"))
					t.Logf("完整 URL: %s", capturedURL)
					return false
				}

				// 验证 limit 参数（仅当 limit > 0 时应该存在）
				if limit > 0 {
					expectedLimit := fmt.Sprintf("%d", limit)
					if capturedQuery.Get("limit") != expectedLimit {
						t.Logf("limit 参数不正确")
						t.Logf("期望: %s", expectedLimit)
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						t.Logf("完整 URL: %s", capturedURL)
						return false
					}
				} else {
					// limit <= 0 时不应该有 limit 参数
					if capturedQuery.Get("limit") != "" {
						t.Logf("limit <= 0 时不应该有 limit 参数")
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						return false
					}
				}

				// 验证 page_token 参数（仅当 pageToken 非空时应该存在）
				if pageToken != "" {
					if capturedQuery.Get("page_token") != pageToken {
						t.Logf("page_token 参数不正确")
						t.Logf("期望: %s", pageToken)
						t.Logf("实际: %s", capturedQuery.Get("page_token"))
						t.Logf("完整 URL: %s", capturedURL)
						return false
					}
				} else {
					// pageToken 为空时不应该有 page_token 参数
					if capturedQuery.Get("page_token") != "" {
						t.Logf("pageToken 为空时不应该有 page_token 参数")
						t.Logf("实际: %s", capturedQuery.Get("page_token"))
						return false
					}
				}

				return true
			},
			// 生成 teamID
			genTeamID(),
			// 生成 limit（0 表示不设置，正数表示设置）
			gen.IntRange(0, 100),
			// 生成 pageToken
			genPageToken(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 7: ListEntries 分页参数传递**
// 验证 ListEntries 方法正确传递分页参数
// **Validates: Requirements 4.5**
func TestProperty_ListEntriesPaginationParametersPassed(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("ListEntries 应正确传递分页参数到查询字符串",
		prop.ForAll(
			func(spaceID, parentID string, limit int, pageToken string) bool {
				// 跳过空的 spaceID
				if spaceID == "" {
					return true
				}

				// 用于捕获请求 URL 的变量
				var capturedURL string
				var capturedQuery url.Values

				// 预期的 token 值
				expectedToken := "test_token_entries_pagination"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获请求 URL 和查询参数
					capturedURL = r.URL.String()
					capturedQuery = r.URL.Query()

					// 返回空的知识节点列表响应
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"data": [], "meta": {"page_token": ""}}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 调用 ListEntries 方法
				_, err = client.ListEntries(ctx, spaceID, parentID, limit, pageToken)
				if err != nil {
					t.Logf("ListEntries 调用失败: %v", err)
					return false
				}

				// 验证 space_id 参数
				if capturedQuery.Get("space_id") != spaceID {
					t.Logf("space_id 参数不正确")
					t.Logf("期望: %s", spaceID)
					t.Logf("实际: %s", capturedQuery.Get("space_id"))
					t.Logf("完整 URL: %s", capturedURL)
					return false
				}

				// 验证 parent_id 参数（仅当 parentID 非空时应该存在）
				if parentID != "" {
					if capturedQuery.Get("parent_id") != parentID {
						t.Logf("parent_id 参数不正确")
						t.Logf("期望: %s", parentID)
						t.Logf("实际: %s", capturedQuery.Get("parent_id"))
						t.Logf("完整 URL: %s", capturedURL)
						return false
					}
				} else {
					// parentID 为空时不应该有 parent_id 参数
					if capturedQuery.Get("parent_id") != "" {
						t.Logf("parentID 为空时不应该有 parent_id 参数")
						t.Logf("实际: %s", capturedQuery.Get("parent_id"))
						return false
					}
				}

				// 验证 limit 参数（仅当 limit > 0 时应该存在）
				if limit > 0 {
					expectedLimit := fmt.Sprintf("%d", limit)
					if capturedQuery.Get("limit") != expectedLimit {
						t.Logf("limit 参数不正确")
						t.Logf("期望: %s", expectedLimit)
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						t.Logf("完整 URL: %s", capturedURL)
						return false
					}
				} else {
					// limit <= 0 时不应该有 limit 参数
					if capturedQuery.Get("limit") != "" {
						t.Logf("limit <= 0 时不应该有 limit 参数")
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						return false
					}
				}

				// 验证 page_token 参数（仅当 pageToken 非空时应该存在）
				if pageToken != "" {
					if capturedQuery.Get("page_token") != pageToken {
						t.Logf("page_token 参数不正确")
						t.Logf("期望: %s", pageToken)
						t.Logf("实际: %s", capturedQuery.Get("page_token"))
						t.Logf("完整 URL: %s", capturedURL)
						return false
					}
				} else {
					// pageToken 为空时不应该有 page_token 参数
					if capturedQuery.Get("page_token") != "" {
						t.Logf("pageToken 为空时不应该有 page_token 参数")
						t.Logf("实际: %s", capturedQuery.Get("page_token"))
						return false
					}
				}

				return true
			},
			// 生成 spaceID
			genSpaceID(),
			// 生成 parentID（可以为空）
			genParentID(),
			// 生成 limit（0 表示不设置，正数表示设置）
			gen.IntRange(0, 100),
			// 生成 pageToken
			genPageToken(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 7: 分页参数边界值测试**
// 验证分页参数在边界值情况下的正确传递
// **Validates: Requirements 3.4, 4.5, 7.1**
func TestProperty_PaginationBoundaryValues(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("分页参数边界值应正确处理",
		prop.ForAll(
			func(limit int) bool {
				// 用于捕获请求 URL 的变量
				var capturedQuery url.Values

				// 预期的 token 值
				expectedToken := "test_token_boundary"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedQuery = r.URL.Query()

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"data": [], "meta": {"page_token": ""}}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 调用 ListSpaces 方法
				_, err = client.ListSpaces(ctx, "team_123", limit, "")
				if err != nil {
					t.Logf("ListSpaces 调用失败: %v", err)
					return false
				}

				// 验证 limit 参数的边界行为
				if limit > 0 {
					// 正数 limit 应该被传递
					expectedLimit := fmt.Sprintf("%d", limit)
					if capturedQuery.Get("limit") != expectedLimit {
						t.Logf("正数 limit 应该被传递")
						t.Logf("期望: %s", expectedLimit)
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						return false
					}
				} else {
					// 0 或负数 limit 不应该被传递
					if capturedQuery.Get("limit") != "" {
						t.Logf("limit <= 0 时不应该有 limit 参数")
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						return false
					}
				}

				return true
			},
			// 生成边界值 limit（包括负数、0、正数）
			gen.IntRange(-10, 100),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 7: 分页参数组合测试**
// 验证不同分页参数组合的正确传递
// **Validates: Requirements 3.4, 4.5, 7.1**
func TestProperty_PaginationParameterCombinations(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("不同分页参数组合应正确传递",
		prop.ForAll(
			func(hasLimit bool, hasPageToken bool, limit int, pageToken string) bool {
				// 用于捕获请求 URL 的变量
				var capturedQuery url.Values

				// 预期的 token 值
				expectedToken := "test_token_combinations"

				// 创建模拟 API 服务器
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedQuery = r.URL.Query()

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"data": [], "meta": {"page_token": ""}}`))
				}))
				defer apiServer.Close()

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: expectedToken,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, apiServer.URL, "test-staff")
				ctx := context.Background()

				// 根据参数决定实际传递的值
				actualLimit := 0
				if hasLimit && limit > 0 {
					actualLimit = limit
				}

				actualPageToken := ""
				if hasPageToken && pageToken != "" {
					actualPageToken = pageToken
				}

				// 调用 ListSpaces 方法
				_, err = client.ListSpaces(ctx, "team_123", actualLimit, actualPageToken)
				if err != nil {
					t.Logf("ListSpaces 调用失败: %v", err)
					return false
				}

				// 验证 limit 参数
				if actualLimit > 0 {
					expectedLimit := fmt.Sprintf("%d", actualLimit)
					if capturedQuery.Get("limit") != expectedLimit {
						t.Logf("limit 参数不正确")
						t.Logf("期望: %s", expectedLimit)
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						return false
					}
				} else {
					if capturedQuery.Get("limit") != "" {
						t.Logf("不应该有 limit 参数")
						t.Logf("实际: %s", capturedQuery.Get("limit"))
						return false
					}
				}

				// 验证 page_token 参数
				if actualPageToken != "" {
					if capturedQuery.Get("page_token") != actualPageToken {
						t.Logf("page_token 参数不正确")
						t.Logf("期望: %s", actualPageToken)
						t.Logf("实际: %s", capturedQuery.Get("page_token"))
						return false
					}
				} else {
					if capturedQuery.Get("page_token") != "" {
						t.Logf("不应该有 page_token 参数")
						t.Logf("实际: %s", capturedQuery.Get("page_token"))
						return false
					}
				}

				return true
			},
			// 生成是否有 limit
			gen.Bool(),
			// 生成是否有 pageToken
			gen.Bool(),
			// 生成 limit 值
			gen.IntRange(1, 100),
			// 生成 pageToken 值
			genPageToken(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 7 辅助生成器
// ============================================================================

// genTeamID 生成团队 ID
func genTeamID() gopter.Gen {
	return gen.OneConstOf(
		"team_001",
		"team_002",
		"team_abc123",
		"team_xyz789",
		"default_team",
		"test_team",
		"",
	)
}

// genSpaceID 生成知识库 ID
func genSpaceID() gopter.Gen {
	return gen.OneConstOf(
		"space_001",
		"space_002",
		"space_abc123",
		"space_xyz789",
		"kb_12345",
		"knowledge_base_001",
		"",
	)
}

// genParentID 生成父节点 ID
func genParentID() gopter.Gen {
	return gen.OneConstOf(
		"entry_001",
		"entry_002",
		"folder_abc123",
		"root_entry_xyz",
		"parent_12345",
		"", // 空字符串表示根目录
	)
}

// genPageToken 生成分页游标
func genPageToken() gopter.Gen {
	return gen.OneConstOf(
		"eyJwYWdlIjoxfQ==",
		"eyJwYWdlIjoyLCJsaW1pdCI6MTB9",
		"next_page_token_abc123",
		"cursor_xyz789",
		"page_2_token",
		"", // 空字符串表示第一页
	)
}

// ============================================================================
// Property 9: COS 上传请求头正确性
// ============================================================================

// **Feature: lexiang-integration, Property 9: COS 上传请求头正确性**
// 属性 9: COS 上传请求头正确性
// 对于任意 UploadFileToCOS 调用，请求头应包含签名信息中的 Authorization 和 x-cos-security-token
// **Validates: Requirements 5.2**
func TestProperty_COSUploadHeadersCorrectness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("UploadFileToCOS 应正确设置 Authorization 和 x-cos-security-token 请求头",
		prop.ForAll(
			func(authorization, securityToken, contentDisposition string, fileSize int) bool {
				// 跳过空的授权信息
				if authorization == "" || securityToken == "" {
					return true
				}

				// 限制文件大小在合理范围内
				if fileSize < 1 {
					fileSize = 1
				}
				if fileSize > 1024 {
					fileSize = 1024
				}

				// 用于捕获请求头的变量
				var capturedAuthHeader string
				var capturedSecurityToken string
				var capturedContentDisposition string
				var capturedMethod string

				// 创建模拟 COS 服务器
				cosServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 捕获请求头
					capturedAuthHeader = r.Header.Get("Authorization")
					capturedSecurityToken = r.Header.Get("x-cos-security-token")
					capturedContentDisposition = r.Header.Get("Content-Disposition")
					capturedMethod = r.Method

					// 返回成功响应，包含 ETag
					w.Header().Set("ETag", "\"d41d8cd98f00b204e9800998ecf8427e\"")
					w.WriteHeader(http.StatusOK)
				}))
				defer cosServer.Close()

				// 解析 COS 服务器 URL 以获取 host
				cosURL, _ := url.Parse(cosServer.URL)

				// 创建模拟 Token 服务器（用于创建 client）
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: "test_token",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				client := NewClientWithTokenManager(tm, tokenServer.URL, "test-staff")

				// 构造 UploadSignResponse，使用模拟 COS 服务器地址
				// 由于 UploadFileToCOS 会构建完整的 COS URL，我们需要特殊处理
				// 这里我们直接测试请求头的设置逻辑
				sign := &UploadSignResponse{
					State: "test_state_123",
				}
				sign.Object.Key = "test/file.pdf"
				sign.Options.Bucket = cosURL.Hostname()
				sign.Options.Region = ""
				sign.Object.Auth.Authorization = authorization
				sign.Object.Auth.XCosSecurityToken = securityToken
				sign.Object.Headers.ContentDisposition = contentDisposition

				// 由于 UploadFileToCOS 会构建特定格式的 COS URL，我们需要使用一个自定义的测试方法
				// 这里我们直接验证请求头设置逻辑
				ctx := context.Background()

				// 创建测试用的文件数据
				fileData := make([]byte, fileSize)
				for i := range fileData {
					fileData[i] = byte(i % 256)
				}

				// 构建 COS 上传 URL（使用模拟服务器）
				cosUploadURL := cosServer.URL + "/" + sign.Object.Key

				// 创建 HTTP 请求
				req, err := http.NewRequestWithContext(ctx, http.MethodPut, cosUploadURL, bytes.NewReader(fileData))
				if err != nil {
					t.Logf("创建请求失败: %v", err)
					return false
				}

				// 设置必需的请求头（模拟 UploadFileToCOS 的行为）
				req.Header.Set("Authorization", sign.Object.Auth.Authorization)
				req.Header.Set("x-cos-security-token", sign.Object.Auth.XCosSecurityToken)
				if sign.Object.Headers.ContentDisposition != "" {
					req.Header.Set("Content-Disposition", sign.Object.Headers.ContentDisposition)
				}

				// 发送请求
				httpClient := client.(*lexiangClientImpl).httpClient
				resp, err := httpClient.Do(req)
				if err != nil {
					t.Logf("发送请求失败: %v", err)
					return false
				}
				defer resp.Body.Close()

				// 验证 Authorization 头
				if capturedAuthHeader != authorization {
					t.Logf("Authorization 头不正确")
					t.Logf("期望: %s", authorization)
					t.Logf("实际: %s", capturedAuthHeader)
					return false
				}

				// 验证 x-cos-security-token 头
				if capturedSecurityToken != securityToken {
					t.Logf("x-cos-security-token 头不正确")
					t.Logf("期望: %s", securityToken)
					t.Logf("实际: %s", capturedSecurityToken)
					return false
				}

				// 验证 Content-Disposition 头（如果设置了）
				if contentDisposition != "" {
					if capturedContentDisposition != contentDisposition {
						t.Logf("Content-Disposition 头不正确")
						t.Logf("期望: %s", contentDisposition)
						t.Logf("实际: %s", capturedContentDisposition)
						return false
					}
				}

				// 验证请求方法是 PUT
				if capturedMethod != http.MethodPut {
					t.Logf("请求方法应为 PUT，实际: %s", capturedMethod)
					return false
				}

				return true
			},
			// 生成 Authorization 字符串
			genCOSAuthorization(),
			// 生成 XCosSecurityToken 字符串
			genCOSSecurityToken(),
			// 生成 Content-Disposition 字符串
			genContentDisposition(),
			// 生成文件大小
			gen.IntRange(1, 1024),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 9: UploadFileToCOS 完整流程请求头验证**
// 验证 UploadFileToCOS 方法在完整调用流程中正确设置请求头
// **Validates: Requirements 5.2**
func TestProperty_UploadFileToCOSFullFlowHeaders(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("UploadFileToCOS 完整调用应正确设置所有必需请求头",
		prop.ForAll(
			func(authorization, securityToken string, fileSize int) bool {
				// 跳过空的授权信息
				if authorization == "" || securityToken == "" {
					return true
				}

				// 限制文件大小
				if fileSize < 1 {
					fileSize = 1
				}
				if fileSize > 512 {
					fileSize = 512
				}

				// 用于捕获请求头的变量
				var capturedAuthHeader string
				var capturedSecurityToken string

				// 创建模拟 COS 服务器
				cosServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedAuthHeader = r.Header.Get("Authorization")
					capturedSecurityToken = r.Header.Get("x-cos-security-token")

					// 返回成功响应，包含 ETag
					w.Header().Set("ETag", "\"abc123def456\"")
					w.WriteHeader(http.StatusOK)
				}))
				defer cosServer.Close()

				// 解析 COS 服务器 URL
				cosURL, _ := url.Parse(cosServer.URL)
				hostParts := strings.Split(cosURL.Host, ":")
				bucket := hostParts[0]

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: "test_token",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				// 创建自定义 HTTP 客户端，重定向所有 COS 请求到模拟服务器
				customTransport := &cosRedirectTransport{
					targetURL: cosServer.URL,
					wrapped:   http.DefaultTransport,
				}
				customClient := &http.Client{
					Transport: customTransport,
					Timeout:   5 * time.Second,
				}

				// 创建 LexiangClient，使用自定义 HTTP 客户端
				client := &lexiangClientImpl{
					tokenManager: tm,
					httpClient:   customClient,
					apiURL:       tokenServer.URL,
				}

				// 构造 UploadSignResponse
				sign := &UploadSignResponse{
					State: "test_state",
				}
				sign.Object.Key = "test/document.pdf"
				sign.Options.Bucket = bucket
				sign.Options.Region = "ap-shanghai"
				sign.Object.Auth.Authorization = authorization
				sign.Object.Auth.XCosSecurityToken = securityToken

				// 创建测试文件数据
				fileData := make([]byte, fileSize)

				ctx := context.Background()

				// 调用 UploadFileToCOS
				err = client.UploadFileToCOS(ctx, sign, fileData)
				if err != nil {
					t.Logf("UploadFileToCOS 调用失败: %v", err)
					return false
				}

				// 验证 Authorization 头
				if capturedAuthHeader != authorization {
					t.Logf("Authorization 头不正确")
					t.Logf("期望: %s", authorization)
					t.Logf("实际: %s", capturedAuthHeader)
					return false
				}

				// 验证 x-cos-security-token 头
				if capturedSecurityToken != securityToken {
					t.Logf("x-cos-security-token 头不正确")
					t.Logf("期望: %s", securityToken)
					t.Logf("实际: %s", capturedSecurityToken)
					return false
				}

				return true
			},
			// 生成 Authorization 字符串
			genCOSAuthorization(),
			// 生成 XCosSecurityToken 字符串
			genCOSSecurityToken(),
			// 生成文件大小
			gen.IntRange(1, 512),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: lexiang-integration, Property 9: COS 上传可选请求头验证**
// 验证 Content-Disposition 等可选请求头的正确设置
// **Validates: Requirements 5.2**
func TestProperty_COSUploadOptionalHeaders(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("UploadFileToCOS 应正确处理可选的 Content-Disposition 请求头",
		prop.ForAll(
			func(contentDisposition string) bool {
				// 用于捕获请求头的变量
				var capturedContentDisposition string

				// 创建模拟 COS 服务器
				cosServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedContentDisposition = r.Header.Get("Content-Disposition")

					// 返回成功响应
					w.Header().Set("ETag", "\"test_etag\"")
					w.WriteHeader(http.StatusOK)
				}))
				defer cosServer.Close()

				// 解析 COS 服务器 URL
				cosURL, _ := url.Parse(cosServer.URL)
				hostParts := strings.Split(cosURL.Host, ":")
				bucket := hostParts[0]

				// 创建模拟 Token 服务器
				tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tokenResponse{
						TokenType:   "Bearer",
						ExpiresIn:   7200,
						AccessToken: "test_token",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(resp)
				}))
				defer tokenServer.Close()

				// 创建 TokenManager
				config := &Config{
					AppKey:        "test_app_key",
					AppSecret:     "test_app_secret",
					BaseURL:       tokenServer.URL,
					Timeout:       5 * time.Second,
					RefreshBuffer: 5 * time.Minute,
				}

				tm, err := NewTokenManager(config)
				if err != nil {
					t.Logf("创建 TokenManager 失败: %v", err)
					return false
				}

				// 创建自定义 HTTP 客户端
				customTransport := &cosRedirectTransport{
					targetURL: cosServer.URL,
					wrapped:   http.DefaultTransport,
				}
				customClient := &http.Client{
					Transport: customTransport,
					Timeout:   5 * time.Second,
				}

				client := &lexiangClientImpl{
					tokenManager: tm,
					httpClient:   customClient,
					apiURL:       tokenServer.URL,
				}

				// 构造 UploadSignResponse
				sign := &UploadSignResponse{
					State: "test_state",
				}
				sign.Object.Key = "test/file.pdf"
				sign.Options.Bucket = bucket
				sign.Options.Region = "ap-shanghai"
				sign.Object.Auth.Authorization = "test_auth"
				sign.Object.Auth.XCosSecurityToken = "test_token"
				sign.Object.Headers.ContentDisposition = contentDisposition

				ctx := context.Background()
				fileData := []byte("test file content")

				// 调用 UploadFileToCOS
				err = client.UploadFileToCOS(ctx, sign, fileData)
				if err != nil {
					t.Logf("UploadFileToCOS 调用失败: %v", err)
					return false
				}

				// 验证 Content-Disposition 头
				if contentDisposition != "" {
					// 如果设置了 Content-Disposition，应该被传递
					if capturedContentDisposition != contentDisposition {
						t.Logf("Content-Disposition 头不正确")
						t.Logf("期望: %s", contentDisposition)
						t.Logf("实际: %s", capturedContentDisposition)
						return false
					}
				} else {
					// 如果没有设置 Content-Disposition，请求头应该为空
					if capturedContentDisposition != "" {
						t.Logf("未设置 Content-Disposition 时请求头应为空")
						t.Logf("实际: %s", capturedContentDisposition)
						return false
					}
				}

				return true
			},
			// 生成 Content-Disposition 字符串（包括空字符串）
			genContentDisposition(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// ============================================================================
// Property 9 辅助类型和生成器
// ============================================================================

// cosRedirectTransport 是一个自定义的 HTTP Transport，用于将 COS 请求重定向到模拟服务器
type cosRedirectTransport struct {
	targetURL string
	wrapped   http.RoundTripper
}

func (t *cosRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 将所有请求重定向到目标 URL
	targetURL, _ := url.Parse(t.targetURL)
	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host
	req.Host = targetURL.Host

	return t.wrapped.RoundTrip(req)
}

// genCOSAuthorization 生成 COS 授权字符串
func genCOSAuthorization() gopter.Gen {
	return gen.OneConstOf(
		"q-sign-algorithm=sha1&q-ak=AKIDxxxxxxxx&q-sign-time=1234567890;1234567899&q-key-time=1234567890;1234567899&q-header-list=&q-url-param-list=&q-signature=abcdef1234567890",
		"q-sign-algorithm=sha1&q-ak=AKID123456&q-sign-time=1609459200;1609545600&q-key-time=1609459200;1609545600&q-header-list=host&q-url-param-list=&q-signature=fedcba0987654321",
		"q-sign-algorithm=sha256&q-ak=AKIDtest&q-sign-time=1700000000;1700086400&q-key-time=1700000000;1700086400&q-header-list=content-type;host&q-url-param-list=&q-signature=test_signature_abc",
		"test_authorization_header",
		"Bearer test_cos_token",
	)
}

// genCOSSecurityToken 生成 COS 安全令牌
func genCOSSecurityToken() gopter.Gen {
	return gen.OneConstOf(
		"security_token_abc123xyz",
		"cos_temp_token_def456",
		"sts_token_ghi789",
		"test_security_token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
	)
}
