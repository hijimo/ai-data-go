package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"genkit-ai-service/internal/logger"
)

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader 捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write 写入响应体
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logger 请求日志中间件
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 生成请求ID
		requestID := uuid.New().String()
		
		// 将请求ID注入到上下文中
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
		r = r.WithContext(ctx)
		
		// 设置响应头
		w.Header().Set("X-Request-ID", requestID)
		
		// 包装 ResponseWriter
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        false,
		}
		
		// 记录开始时间
		start := time.Now()
		
		// 记录请求开始
		logger.InfoContext(ctx, "HTTP request started", logger.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"query":      r.URL.RawQuery,
			"remoteAddr": r.RemoteAddr,
			"userAgent":  r.UserAgent(),
		})
		
		// 调用下一个处理器
		next.ServeHTTP(rw, r)
		
		// 计算耗时
		duration := time.Since(start)
		
		// 记录请求完成
		logger.InfoContext(ctx, "HTTP request completed", logger.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"statusCode": rw.statusCode,
			"duration":   duration.String(),
			"durationMs": duration.Milliseconds(),
		})
	})
}
