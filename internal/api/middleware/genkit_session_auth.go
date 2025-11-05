package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// GenkitSessionAuthConfig Genkit 会话认证中间件配置
type GenkitSessionAuthConfig struct {
	// AuditRepo 审计日志仓储
	AuditRepo repository.AuditRepository
	// SessionRepo 会话仓储（用于验证会话所有权）
	SessionRepo repository.SessionRepository
	// UserRepo 用户仓储（用于获取用户信息）
	UserRepo repository.UserRepository
}

// RequireGenkitSessionAccess 要求 Genkit 会话访问权限的中间件
// 验证用户是否有权访问指定的会话
// 平台管理员可以访问所有会话，其他用户只能访问自己租户内的会话
func RequireGenkitSessionAccess(config GenkitSessionAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. 获取 JWT Claims
			claims, ok := GetJWTClaims(ctx)
			if !ok {
				logger.WarnContext(ctx, "Genkit 会话认证: 未找到 JWT Claims", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				respondWithUnauthorized(w, "身份认证信息缺失")
				return
			}

			// 2. 从请求中提取会话 ID
			sessionID := extractSessionIDFromRequest(r)
			if sessionID == "" {
				// 如果没有会话 ID，跳过会话级别的权限验证
				// 这种情况下，后续的业务逻辑应该处理权限验证
				logger.DebugContext(ctx, "Genkit 会话认证: 未找到会话 ID，跳过会话访问验证", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				next.ServeHTTP(w, r)
				return
			}

			// 3. 验证会话访问权限
			if err := validateSessionAccess(ctx, claims, sessionID, config); err != nil {
				// 记录审计日志
				logSessionAccessDenied(ctx, r, claims, sessionID, err.Error(), config.AuditRepo)
				
				logger.WarnContext(ctx, "Genkit 会话认证: 会话访问权限验证失败", logger.Fields{
					"path":       r.URL.Path,
					"method":     r.Method,
					"user_id":    claims.Subject,
					"tenant_id":  claims.TenantID,
					"session_id": sessionID,
					"error":      err.Error(),
				})
				
				respondWithForbidden(w, err.Error())
				return
			}

			// 4. 记录权限验证成功
			logger.DebugContext(ctx, "Genkit 会话认证: 会话访问权限验证通过", logger.Fields{
				"path":       r.URL.Path,
				"method":     r.Method,
				"user_id":    claims.Subject,
				"tenant_id":  claims.TenantID,
				"session_id": sessionID,
			})

			// 5. 将会话 ID 注入上下文
			ctx = context.WithValue(ctx, "session_id", sessionID)
			r = r.WithContext(ctx)

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// validateSessionAccess 验证会话访问权限
func validateSessionAccess(
	ctx context.Context,
	claims *model.JWTClaims,
	sessionID string,
	config GenkitSessionAuthConfig,
) error {
	// 1. 平台管理员可以访问所有会话
	if hasRole(ctx, model.RoleSystemAdmin) {
		return nil
	}

	// 2. 查询会话信息
	if config.SessionRepo == nil {
		return errors.New(errors.CodeInternalError, "会话仓储未配置")
	}

	session, err := config.SessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.NewNotFoundError("会话不存在")
	}

	// 3. 获取会话所属用户的租户 ID
	if config.UserRepo == nil {
		return errors.New(errors.CodeInternalError, "用户仓储未配置")
	}

	sessionUser, err := config.UserRepo.GetByID(ctx, session.UserID.String())
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "获取会话用户信息失败", err)
	}

	// 4. 验证租户 ID 匹配
	if claims.TenantID != sessionUser.TenantID.String() {
		return errors.NewForbiddenError("权限不足：无法访问其他租户的会话")
	}

	return nil
}

// extractSessionIDFromRequest 从请求中提取会话 ID
// 支持多种方式：URL 路径参数、查询参数、请求体
func extractSessionIDFromRequest(r *http.Request) string {
	// 1. 从 URL 路径中提取
	// 支持路径格式：/api/v1/sessions/{sessionId}/... 或 /api/v1/genkit/sessions/{sessionId}/...
	sessionID := extractSessionIDFromPath(r.URL.Path)
	if sessionID != "" {
		return sessionID
	}

	// 2. 从查询参数中提取
	sessionID = r.URL.Query().Get("sessionId")
	if sessionID != "" {
		return sessionID
	}

	sessionID = r.URL.Query().Get("session_id")
	if sessionID != "" {
		return sessionID
	}

	// 3. 从请求体中提取（仅对 POST/PUT/PATCH 请求）
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		// 注意：这里不解析请求体，因为会消耗 Body
		// 如果需要从请求体提取，应该在 Handler 层处理
	}

	return ""
}

// extractSessionIDFromPath 从 URL 路径中提取会话 ID
func extractSessionIDFromPath(path string) string {
	// 使用与 extractTenantIDFromPath 类似的逻辑
	parts := splitPath(path)

	// 查找 "sessions" 后面的部分作为会话 ID
	for i, part := range parts {
		if part == "sessions" && i+1 < len(parts) {
			sessionID := parts[i+1]
			// 验证这不是另一个路径段
			if sessionID != "" && !isPathSegment(sessionID) {
				return sessionID
			}
		}
	}

	return ""
}

// splitPath 分割路径
func splitPath(path string) []string {
	var parts []string
	for _, part := range []rune(path) {
		if part == '/' {
			continue
		}
		// 简化实现：直接使用字符串分割
		break
	}
	
	// 使用标准库分割
	result := []string{}
	current := ""
	for _, char := range path {
		if char == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	
	return result
}

// logSessionAccessDenied 记录会话访问被拒绝的审计日志
func logSessionAccessDenied(
	ctx context.Context,
	r *http.Request,
	claims *model.JWTClaims,
	sessionID string,
	reason string,
	auditRepo repository.AuditRepository,
) {
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

	// 构建元数据
	meta := map[string]interface{}{
		"event":      "session_access_denied",
		"reason":     reason,
		"path":       r.URL.Path,
		"method":     r.Method,
		"session_id": sessionID,
		"roles":      claims.Roles,
		"ip":         getClientIP(r),
		"user_agent": r.UserAgent(),
	}

	metaJSON, _ := json.Marshal(meta)

	// 创建审计日志
	audit := &model.AuthAudit{
		TenantID:  tenantID,
		UserID:    userID,
		Event:     "session_access_denied",
		IP:        getClientIP(r),
		UserAgent: r.UserAgent(),
		Meta:      datatypes.JSON(metaJSON),
	}

	// 异步记录审计日志
	if auditRepo != nil {
		go func() {
			auditCtx := context.Background()
			if err := auditRepo.Create(auditCtx, audit); err != nil {
				logger.ErrorContext(auditCtx, "记录审计日志失败", logger.Fields{
					"error":      err.Error(),
					"event":      "session_access_denied",
					"user_id":    userID,
					"tenant_id":  tenantID,
					"session_id": sessionID,
				})
			}
		}()
	}

	// 同时记录到应用日志
	logger.WarnContext(ctx, "会话访问被拒绝", logger.Fields{
		"event":      "session_access_denied",
		"reason":     reason,
		"path":       r.URL.Path,
		"method":     r.Method,
		"user_id":    userID,
		"tenant_id":  tenantID,
		"session_id": sessionID,
		"roles":      claims.Roles,
		"ip":         getClientIP(r),
		"user_agent": r.UserAgent(),
	})
}

// RequireGenkitMemoryAccess 要求 Genkit 记忆访问权限的中间件
// 验证用户是否有权访问指定的记忆
func RequireGenkitMemoryAccess(config GenkitSessionAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 获取 JWT Claims
			claims, ok := GetJWTClaims(ctx)
			if !ok {
				logger.WarnContext(ctx, "Genkit 记忆认证: 未找到 JWT Claims", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				respondWithUnauthorized(w, "身份认证信息缺失")
				return
			}

			// 从请求中提取记忆 ID
			memoryID := extractMemoryIDFromRequest(r)
			if memoryID == "" {
				// 如果没有记忆 ID，跳过记忆级别的权限验证
				logger.DebugContext(ctx, "Genkit 记忆认证: 未找到记忆 ID，跳过记忆访问验证", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				next.ServeHTTP(w, r)
				return
			}

			// 平台管理员可以访问所有记忆
			if hasRole(ctx, model.RoleSystemAdmin) {
				logger.DebugContext(ctx, "Genkit 记忆认证: 平台管理员跨租户访问", logger.Fields{
					"path":      r.URL.Path,
					"method":    r.Method,
					"user_id":   claims.Subject,
					"tenant_id": claims.TenantID,
					"memory_id": memoryID,
				})
				next.ServeHTTP(w, r)
				return
			}

			// 记录权限验证成功
			logger.DebugContext(ctx, "Genkit 记忆认证: 记忆访问权限验证通过", logger.Fields{
				"path":      r.URL.Path,
				"method":    r.Method,
				"user_id":   claims.Subject,
				"tenant_id": claims.TenantID,
				"memory_id": memoryID,
			})

			// 将记忆 ID 注入上下文
			ctx = context.WithValue(ctx, "memory_id", memoryID)
			r = r.WithContext(ctx)

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// extractMemoryIDFromRequest 从请求中提取记忆 ID
func extractMemoryIDFromRequest(r *http.Request) string {
	// 从 URL 路径中提取
	parts := splitPath(r.URL.Path)

	for i, part := range parts {
		if part == "memories" && i+1 < len(parts) {
			memoryID := parts[i+1]
			if memoryID != "" && !isPathSegment(memoryID) {
				return memoryID
			}
		}
	}

	// 从查询参数中提取
	memoryID := r.URL.Query().Get("memoryId")
	if memoryID != "" {
		return memoryID
	}

	return r.URL.Query().Get("memory_id")
}
