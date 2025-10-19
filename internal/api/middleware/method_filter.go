package middleware

import (
	"net/http"
)

// MethodFilter 创建一个 HTTP 方法过滤中间件
// 只允许指定的 HTTP 方法通过，其他方法返回 405 Method Not Allowed
func MethodFilter(allowedMethod string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowedMethod {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MethodFilterMultiple 创建一个支持多个 HTTP 方法的过滤中间件
func MethodFilterMultiple(allowedMethods []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := false
		for _, method := range allowedMethods {
			if r.Method == method {
				allowed = true
				break
			}
		}
		
		if !allowed {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
