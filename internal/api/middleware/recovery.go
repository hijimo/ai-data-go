package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// Recovery panic 恢复中间件
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := string(debug.Stack())
				
				// 记录 panic 错误
				logger.ErrorContext(r.Context(), "Panic recovered", logger.Fields{
					"error": fmt.Sprintf("%v", err),
					"stack": stack,
					"path":  r.URL.Path,
				})
				
				// 返回内部错误响应
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				
				resp := response.Error[any](errors.CodeInternalError, errors.MsgInternalError)
				
				// 序列化响应
				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}
