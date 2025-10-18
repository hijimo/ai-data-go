package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

func TestUserContext(t *testing.T) {
	tests := []struct {
		name           string
		userIDHeader   string
		wantStatusCode int
		wantUserID     string
		wantError      bool
	}{
		{
			name:           "有效的用户ID",
			userIDHeader:   "user-123",
			wantStatusCode: http.StatusOK,
			wantUserID:     "user-123",
			wantError:      false,
		},
		{
			name:           "缺少用户ID",
			userIDHeader:   "",
			wantStatusCode: http.StatusUnauthorized,
			wantUserID:     "",
			wantError:      true,
		},
		{
			name:           "UUID格式的用户ID",
			userIDHeader:   "550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantUserID:     "550e8400-e29b-41d4-a716-446655440000",
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试处理器
			var capturedUserID string
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 尝试从上下文获取 UserID
				if userID, ok := GetUserID(r.Context()); ok {
					capturedUserID = userID
				}
				w.WriteHeader(http.StatusOK)
			})

			// 应用中间件
			handler := UserContext(testHandler)

			// 创建测试请求
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.userIDHeader != "" {
				req.Header.Set(UserIDHeader, tt.userIDHeader)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 执行请求
			handler.ServeHTTP(rr, req)

			// 验证状态码
			if rr.Code != tt.wantStatusCode {
				t.Errorf("状态码 = %v, 期望 %v", rr.Code, tt.wantStatusCode)
			}

			// 验证用户ID
			if !tt.wantError && capturedUserID != tt.wantUserID {
				t.Errorf("用户ID = %v, 期望 %v", capturedUserID, tt.wantUserID)
			}

			// 验证错误响应
			if tt.wantError {
				var resp model.ResponseData[any]
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("解析响应失败: %v", err)
				}

				if resp.Code != errors.CodeUnauthorized {
					t.Errorf("错误码 = %v, 期望 %v", resp.Code, errors.CodeUnauthorized)
				}

				if resp.Message == "" {
					t.Error("错误消息不应为空")
				}
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func() context.Context
		wantUserID string
		wantOK     bool
	}{
		{
			name: "上下文包含用户ID",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, "user-123")
			},
			wantUserID: "user-123",
			wantOK:     true,
		},
		{
			name: "上下文不包含用户ID",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantUserID: "",
			wantOK:     false,
		},
		{
			name: "上下文包含错误类型的值",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, 123)
			},
			wantUserID: "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			userID, ok := GetUserID(ctx)

			if ok != tt.wantOK {
				t.Errorf("ok = %v, 期望 %v", ok, tt.wantOK)
			}

			if userID != tt.wantUserID {
				t.Errorf("userID = %v, 期望 %v", userID, tt.wantUserID)
			}
		})
	}
}

func TestMustGetUserID(t *testing.T) {
	t.Run("上下文包含用户ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, "user-123")
		userID := MustGetUserID(ctx)

		if userID != "user-123" {
			t.Errorf("userID = %v, 期望 user-123", userID)
		}
	})

	t.Run("上下文不包含用户ID应该panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("期望 panic，但没有发生")
			}
		}()

		ctx := context.Background()
		MustGetUserID(ctx)
	})
}

func TestUserContextIntegration(t *testing.T) {
	// 测试完整的中间件链
	t.Run("中间件链集成测试", func(t *testing.T) {
		var capturedUserID string

		// 创建处理器链
		handler := UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUserID = MustGetUserID(r.Context())
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		// 创建请求
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(UserIDHeader, "integration-user-456")

		// 执行请求
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// 验证
		if rr.Code != http.StatusOK {
			t.Errorf("状态码 = %v, 期望 %v", rr.Code, http.StatusOK)
		}

		if capturedUserID != "integration-user-456" {
			t.Errorf("用户ID = %v, 期望 integration-user-456", capturedUserID)
		}

		if rr.Body.String() != "success" {
			t.Errorf("响应体 = %v, 期望 success", rr.Body.String())
		}
	})
}
