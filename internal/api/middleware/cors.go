package middleware

import (
	"fmt"
	"net/http"
)

// CORS CORS 中间件配置
type CORS struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORS 返回默认的 CORS 配置
func DefaultCORS() *CORS {
	return &CORS{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24小时
	}
}

// Handler 返回 CORS 中间件处理器
func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 响应头
		origin := r.Header.Get("Origin")
		if origin != "" {
			// 如果允许所有来源，返回请求的 origin（而不是 *）
			if len(c.AllowOrigins) == 1 && c.AllowOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if c.isOriginAllowed(origin) {
				// 检查是否允许该来源
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}
		
		// 设置其他 CORS 头
		if len(c.AllowMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", joinStrings(c.AllowMethods))
		}
		
		if len(c.AllowHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(c.AllowHeaders))
		}
		
		if len(c.ExposeHeaders) > 0 {
			w.Header().Set("Access-Control-Expose-Headers", joinStrings(c.ExposeHeaders))
		}
		
		if c.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		if c.MaxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", c.MaxAge))
		}
		
		// 处理预检请求
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed 检查来源是否被允许
func (c *CORS) isOriginAllowed(origin string) bool {
	for _, allowed := range c.AllowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// joinStrings 连接字符串切片
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}
