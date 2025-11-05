package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AuditConfig 审计中间件配置
type AuditConfig struct {
	// AuditRepo 审计日志仓储
	AuditRepo repository.AuditRepository
	// EnabledEvents 启用的事件类型列表（为空则记录所有事件）
	EnabledEvents []string
	// ExcludedPaths 排除的路径列表（不记录审计日志）
	ExcludedPaths []string
}

// AuditMiddleware 审计日志中间件
// 记录所有请求的审计日志
func AuditMiddleware(config AuditConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 检查是否需要排除此路径
			if shouldExcludePath(r.URL.Path, config.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// 记录请求开始时间
			startTime := time.Now()

			// 创建响应包装器以捕获状态码
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// 处理请求
			next.ServeHTTP(rw, r)

			// 记录请求结束时间
			duration := time.Since(startTime)

			// 异步记录审计日志
			go logRequestAudit(r.Context(), r, rw.statusCode, duration, config)
		})
	}
}

// responseWriter 响应包装器，用于捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// shouldExcludePath 检查路径是否应该被排除
func shouldExcludePath(path string, excludedPaths []string) bool {
	for _, excluded := range excludedPaths {
		if path == excluded {
			return true
		}
		// 支持前缀匹配
		if len(excluded) > 0 && excluded[len(excluded)-1] == '*' {
			prefix := excluded[:len(excluded)-1]
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// logRequestAudit 记录请求审计日志
func logRequestAudit(
	ctx context.Context,
	r *http.Request,
	statusCode int,
	duration time.Duration,
	config AuditConfig,
) {
	// 获取 JWT Claims
	claims, _ := GetJWTClaims(ctx)

	// 解析用户 ID 和租户 ID
	var userID, tenantID *uuid.UUID
	if claims != nil {
		if uid, err := uuid.Parse(claims.Subject); err == nil {
			userID = &uid
		}
		if tid, err := uuid.Parse(claims.TenantID); err == nil {
			tenantID = &tid
		}
	}

	// 确定事件类型
	event := determineEventType(r, statusCode)

	// 检查是否需要记录此事件
	if !shouldLogEvent(event, config.EnabledEvents) {
		return
	}

	// 构建元数据
	meta := map[string]interface{}{
		"path":         r.URL.Path,
		"method":       r.Method,
		"status_code":  statusCode,
		"duration_ms":  duration.Milliseconds(),
		"query_params": r.URL.Query(),
	}

	// 添加会话 ID（如果存在）
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		meta["session_id"] = sessionID
	}

	// 添加角色信息
	if claims != nil {
		meta["roles"] = claims.Roles
	}

	metaJSON, _ := json.Marshal(meta)

	// 创建审计日志
	audit := &model.AuthAudit{
		TenantID:  tenantID,
		UserID:    userID,
		Event:     event,
		IP:        getClientIP(r),
		UserAgent: r.UserAgent(),
		Meta:      datatypes.JSON(metaJSON),
	}

	// 记录到数据库
	if config.AuditRepo != nil {
		auditCtx := context.Background()
		if err := config.AuditRepo.Create(auditCtx, audit); err != nil {
			logger.ErrorContext(auditCtx, "记录审计日志失败", logger.Fields{
				"error":     err.Error(),
				"event":     event,
				"user_id":   userID,
				"tenant_id": tenantID,
				"path":      r.URL.Path,
			})
		}
	}

	// 记录到应用日志
	logger.InfoContext(ctx, "请求审计", logger.Fields{
		"event":       event,
		"path":        r.URL.Path,
		"method":      r.Method,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"user_id":     userID,
		"tenant_id":   tenantID,
		"ip":          getClientIP(r),
	})
}

// determineEventType 根据请求和响应确定事件类型
func determineEventType(r *http.Request, statusCode int) string {
	// 根据路径和方法确定事件类型
	path := r.URL.Path
	method := r.Method

	// 会话相关事件
	if contains(path, "/sessions") {
		switch method {
		case http.MethodPost:
			return "session_create"
		case http.MethodGet:
			return "session_read"
		case http.MethodPut, http.MethodPatch:
			return "session_update"
		case http.MethodDelete:
			return "session_delete"
		}
	}

	// 记忆相关事件
	if contains(path, "/memories") {
		switch method {
		case http.MethodPost:
			return "memory_create"
		case http.MethodGet:
			return "memory_read"
		case http.MethodPut, http.MethodPatch:
			return "memory_update"
		case http.MethodDelete:
			return "memory_delete"
		}
	}

	// 摘要相关事件
	if contains(path, "/summaries") {
		switch method {
		case http.MethodPost:
			return "summary_create"
		case http.MethodGet:
			return "summary_read"
		}
	}

	// 上下文相关事件
	if contains(path, "/context") {
		return "context_build"
	}

	// 对话相关事件
	if contains(path, "/chat") || contains(path, "/generate") {
		return "chat_generate"
	}

	// 权限拒绝事件
	if statusCode == http.StatusForbidden {
		return "permission_denied"
	}

	// 认证失败事件
	if statusCode == http.StatusUnauthorized {
		return "authentication_failed"
	}

	// 默认事件
	return "api_request"
}

// shouldLogEvent 检查是否应该记录此事件
func shouldLogEvent(event string, enabledEvents []string) bool {
	// 如果没有配置启用的事件，记录所有事件
	if len(enabledEvents) == 0 {
		return true
	}

	// 检查事件是否在启用列表中
	for _, enabled := range enabledEvents {
		if event == enabled {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

// findSubstring 查找子串
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// LogAuditEvent 手动记录审计事件
// 可以在业务逻辑中调用此函数记录特定事件
func LogAuditEvent(
	ctx context.Context,
	event string,
	meta map[string]interface{},
	auditRepo repository.AuditRepository,
) {
	// 获取 JWT Claims
	claims, _ := GetJWTClaims(ctx)

	// 解析用户 ID 和租户 ID
	var userID, tenantID *uuid.UUID
	if claims != nil {
		if uid, err := uuid.Parse(claims.Subject); err == nil {
			userID = &uid
		}
		if tid, err := uuid.Parse(claims.TenantID); err == nil {
			tenantID = &tid
		}
	}

	// 序列化元数据
	metaJSON, _ := json.Marshal(meta)

	// 创建审计日志
	audit := &model.AuthAudit{
		TenantID:  tenantID,
		UserID:    userID,
		Event:     event,
		IP:        "",
		UserAgent: "",
		Meta:      datatypes.JSON(metaJSON),
	}

	// 异步记录到数据库
	if auditRepo != nil {
		go func() {
			auditCtx := context.Background()
			if err := auditRepo.Create(auditCtx, audit); err != nil {
				logger.ErrorContext(auditCtx, "记录审计日志失败", logger.Fields{
					"error":     err.Error(),
					"event":     event,
					"user_id":   userID,
					"tenant_id": tenantID,
				})
			}
		}()
	}

	// 记录到应用日志
	logger.InfoContext(ctx, "审计事件", logger.Fields{
		"event":     event,
		"user_id":   userID,
		"tenant_id": tenantID,
		"meta":      meta,
	})
}
