package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	// UserIDKey 用户ID上下文键
	UserIDKey ContextKey = "userID"
	
	// UserIDHeader 用户ID请求头名称
	UserIDHeader = "X-User-ID"
)

// UserContext 用户上下文中间件
// 从请求头中提取 UserID 并存入上下文
func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中获取 UserID
		userID := r.Header.Get(UserIDHeader)
		
		// 如果没有提供 UserID，返回未授权错误
		if userID == "" {
			logger.WarnContext(r.Context(), "未提供用户ID", logger.Fields{
				"path":   r.URL.Path,
				"method": r.Method,
			})
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			
			resp := response.Error[any](errors.CodeUnauthorized, "未提供用户身份信息")
			
			if data, err := json.Marshal(resp); err == nil {
				w.Write(data)
			}
			return
		}
		
		// 将 UserID 存入上下文
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		r = r.WithContext(ctx)
		
		// 记录用户信息
		logger.DebugContext(ctx, "用户上下文已设置", logger.Fields{
			"userID": userID,
			"path":   r.URL.Path,
		})
		
		// 调用下一个处理器
		next.ServeHTTP(w, r)
	})
}

// GetUserID 从上下文中获取 UserID
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// MustGetUserID 从上下文中获取 UserID，如果不存在则 panic
// 注意：仅在确保已经过 UserContext 中间件处理后使用
func MustGetUserID(ctx context.Context) string {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("用户ID未在上下文中设置")
	}
	return userID
}
