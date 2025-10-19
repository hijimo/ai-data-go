package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// TenantContextKey 租户上下文键类型
type TenantContextKey string

const (
	// TenantIDKey 租户ID上下文键
	TenantIDKey TenantContextKey = "tenant_id"

	// TenantIDHeader 租户ID请求头名称
	TenantIDHeader = "X-Tenant-ID"

	// TenantDomainHeader 租户域名请求头名称
	TenantDomainHeader = "X-Tenant-Domain"
)

// TenantIdentifierConfig 租户识别中间件配置
type TenantIdentifierConfig struct {
	// Strategy 租户识别策略：header, subdomain, path, cookie
	Strategy string
	// TenantRepo 租户仓库，用于验证租户
	TenantRepo repository.TenantRepository
	// BaseDomain 基础域名，用于子域名识别（如 api.example.com）
	BaseDomain string
}

// TenantIdentifier 租户识别中间件
// 从请求中识别租户并注入到上下文中
func TenantIdentifier(config TenantIdentifierConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenantID string
			var err error

			// 根据策略识别租户
			switch config.Strategy {
			case "header":
				tenantID, err = identifyFromHeader(r)
			case "subdomain":
				tenantID, err = identifyFromSubdomain(r, config.BaseDomain, config.TenantRepo)
			case "path":
				tenantID, err = identifyFromPath(r)
			case "cookie":
				tenantID, err = identifyFromCookie(r)
			default:
				// 默认使用请求头识别
				tenantID, err = identifyFromHeader(r)
			}

			// 如果识别失败，尝试其他方式
			if err != nil || tenantID == "" {
				// 尝试从请求头识别（作为后备方案）
				if config.Strategy != "header" {
					tenantID, err = identifyFromHeader(r)
				}
			}

			// 如果仍然无法识别租户
			if err != nil || tenantID == "" {
				logger.WarnContext(r.Context(), "无法识别租户", logger.Fields{
					"path":     r.URL.Path,
					"method":   r.Method,
					"strategy": config.Strategy,
					"error":    err,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				resp := response.Error[any](errors.CodeBadRequest, "无法识别租户，请提供有效的租户标识")

				if data, jsonErr := json.Marshal(resp); jsonErr == nil {
					w.Write(data)
				}
				return
			}

			// 验证租户是否存在且启用
			if config.TenantRepo != nil {
				tenant, err := config.TenantRepo.GetByID(r.Context(), tenantID)
				if err != nil {
					logger.WarnContext(r.Context(), "租户不存在", logger.Fields{
						"tenant_id": tenantID,
						"path":      r.URL.Path,
						"error":     err,
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)

					resp := response.Error[any](errors.CodeNotFound, "租户不存在")

					if data, jsonErr := json.Marshal(resp); jsonErr == nil {
						w.Write(data)
					}
					return
				}

				// 检查租户是否启用
				if !tenant.Status {
					logger.WarnContext(r.Context(), "租户已禁用", logger.Fields{
						"tenant_id": tenantID,
						"path":      r.URL.Path,
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)

					resp := response.Error[any](errors.CodeForbidden, "租户已禁用")

					if data, jsonErr := json.Marshal(resp); jsonErr == nil {
						w.Write(data)
					}
					return
				}
			}

			// 将租户 ID 注入上下文
			ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
			r = r.WithContext(ctx)

			// 记录租户信息
			logger.DebugContext(ctx, "租户上下文已设置", logger.Fields{
				"tenant_id": tenantID,
				"path":      r.URL.Path,
				"strategy":  config.Strategy,
			})

			// 调用下一个处理器
			next.ServeHTTP(w, r)
		})
	}
}

// identifyFromHeader 从请求头识别租户
func identifyFromHeader(r *http.Request) (string, error) {
	// 优先从 X-Tenant-ID 头获取
	if tidStr := r.Header.Get(TenantIDHeader); tidStr != "" {
		return tidStr, nil
	}

	// 尝试从 X-Tenant-Domain 头获取域名，然后查询租户
	// 注意：这需要数据库查询，在这里只返回空，由调用者处理
	if domain := r.Header.Get(TenantDomainHeader); domain != "" {
		// 这里不直接查询数据库，返回错误让调用者处理
		return "", nil
	}

	return "", nil
}

// identifyFromSubdomain 从子域名识别租户
func identifyFromSubdomain(r *http.Request, baseDomain string, tenantRepo repository.TenantRepository) (string, error) {
	host := r.Host

	// 移除端口号
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// 如果没有配置基础域名，无法识别
	if baseDomain == "" {
		return "", nil
	}

	// 检查是否是子域名
	if !strings.HasSuffix(host, "."+baseDomain) {
		return "", nil
	}

	// 提取子域名
	subdomain := strings.TrimSuffix(host, "."+baseDomain)

	// 如果子域名为空或包含多个点，无法识别
	if subdomain == "" || strings.Contains(subdomain, ".") {
		return "", nil
	}

	// 通过域名查询租户
	if tenantRepo != nil {
		tenant, err := tenantRepo.GetByDomain(r.Context(), subdomain)
		if err != nil {
			return "", err
		}
		return tenant.ID, nil
	}

	return "", nil
}

// identifyFromPath 从 URL 路径识别租户
func identifyFromPath(r *http.Request) (string, error) {
	// 期望路径格式：/api/v1/tenants/{tenant_id}/...
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	// 查找 tenants 段
	for i, part := range parts {
		if part == "tenants" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}

	return "", nil
}

// identifyFromCookie 从 Cookie 识别租户
func identifyFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("tenant_id")
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

// GetTenantID 从上下文中获取租户 ID
func GetTenantID(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(TenantIDKey).(string)
	return tenantID, ok
}

// MustGetTenantID 从上下文中获取租户 ID，如果不存在则 panic
// 注意：仅在确保已经过 TenantIdentifier 中间件处理后使用
func MustGetTenantID(ctx context.Context) string {
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		panic("租户ID未在上下文中设置")
	}
	return tenantID
}
