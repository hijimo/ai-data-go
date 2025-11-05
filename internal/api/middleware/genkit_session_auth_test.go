package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/model"
	authservice "genkit-ai-service/internal/service/auth"

	"github.com/google/uuid"
)

func TestExtractSessionIDFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "标准会话路径",
			path:     "/api/v1/sessions/123e4567-e89b-12d3-a456-426614174000/messages",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "Genkit 会话路径",
			path:     "/api/v1/genkit/sessions/123e4567-e89b-12d3-a456-426614174000/context",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "没有会话 ID",
			path:     "/api/v1/sessions",
			expected: "",
		},
		{
			name:     "会话列表路径",
			path:     "/api/v1/sessions/list",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSessionIDFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("extractSessionIDFromPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractMemoryIDFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "标准记忆路径",
			path:     "/api/v1/memories/123e4567-e89b-12d3-a456-426614174000",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "Genkit 记忆路径",
			path:     "/api/v1/genkit/memories/123e4567-e89b-12d3-a456-426614174000/details",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "没有记忆 ID",
			path:     "/api/v1/memories",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMemoryIDFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("extractMemoryIDFromPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractSessionIDFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		query    string
		expected string
	}{
		{
			name:     "从路径提取",
			path:     "/api/v1/sessions/123e4567-e89b-12d3-a456-426614174000/messages",
			query:    "",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "从查询参数提取 sessionId",
			path:     "/api/v1/context",
			query:    "sessionId=123e4567-e89b-12d3-a456-426614174000",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "从查询参数提取 session_id",
			path:     "/api/v1/context",
			query:    "session_id=123e4567-e89b-12d3-a456-426614174000",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "路径优先于查询参数",
			path:     "/api/v1/sessions/111e4567-e89b-12d3-a456-426614174000/messages",
			query:    "sessionId=222e4567-e89b-12d3-a456-426614174000",
			expected: "111e4567-e89b-12d3-a456-426614174000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path+"?"+tt.query, nil)
			result := extractSessionIDFromRequest(req)
			if result != tt.expected {
				t.Errorf("extractSessionIDFromRequest() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetUserContext(t *testing.T) {
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	roles := []string{model.RoleTenantAdmin}

	// 创建带有用户上下文的 context
	ctx := context.Background()
	ctx = context.WithValue(ctx, UserIDContextKey, userID)
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, UserRolesKey, roles)

	claims := &model.JWTClaims{
		Subject:  userID,
		TenantID: tenantID,
		Roles:    roles,
	}
	ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)

	// 测试 GetUserContext
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		t.Fatal("GetUserContext() returned nil")
	}

	if userCtx.UserID != userID {
		t.Errorf("UserID = %v, want %v", userCtx.UserID, userID)
	}

	if userCtx.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", userCtx.TenantID, tenantID)
	}

	if !userCtx.IsTenantAdmin() {
		t.Error("IsTenantAdmin() = false, want true")
	}

	if userCtx.IsSystemAdmin() {
		t.Error("IsSystemAdmin() = true, want false")
	}
}

func TestUserContextCanAccessTenant(t *testing.T) {
	tenantID := uuid.New().String()
	otherTenantID := uuid.New().String()

	tests := []struct {
		name            string
		userContext     *UserContext
		targetTenantID  string
		expectedAccess  bool
	}{
		{
			name: "平台管理员可以访问任何租户",
			userContext: &UserContext{
				TenantID: tenantID,
				Roles:    []string{model.RoleSystemAdmin},
			},
			targetTenantID: otherTenantID,
			expectedAccess: true,
		},
		{
			name: "租户管理员可以访问自己的租户",
			userContext: &UserContext{
				TenantID: tenantID,
				Roles:    []string{model.RoleTenantAdmin},
			},
			targetTenantID: tenantID,
			expectedAccess: true,
		},
		{
			name: "租户管理员不能访问其他租户",
			userContext: &UserContext{
				TenantID: tenantID,
				Roles:    []string{model.RoleTenantAdmin},
			},
			targetTenantID: otherTenantID,
			expectedAccess: false,
		},
		{
			name:            "nil 用户上下文不能访问任何租户",
			userContext:     nil,
			targetTenantID:  tenantID,
			expectedAccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.userContext.CanAccessTenant(tt.targetTenantID)
			if result != tt.expectedAccess {
				t.Errorf("CanAccessTenant() = %v, want %v", result, tt.expectedAccess)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "标准路径",
			path:     "/api/v1/sessions/123/messages",
			expected: []string{"api", "v1", "sessions", "123", "messages"},
		},
		{
			name:     "带尾部斜杠的路径",
			path:     "/api/v1/sessions/",
			expected: []string{"api", "v1", "sessions"},
		},
		{
			name:     "空路径",
			path:     "",
			expected: []string{},
		},
		{
			name:     "单个斜杠",
			path:     "/",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.path)
			if len(result) != len(tt.expected) {
				t.Errorf("splitPath() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitPath()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestHasSystemAdminRole(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "有平台管理员角色",
			roles:    []string{model.RoleSystemAdmin},
			expected: true,
		},
		{
			name:     "有多个角色包括平台管理员",
			roles:    []string{model.RoleTenantAdmin, model.RoleSystemAdmin},
			expected: true,
		},
		{
			name:     "只有租户管理员角色",
			roles:    []string{model.RoleTenantAdmin},
			expected: false,
		},
		{
			name:     "没有角色",
			roles:    []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, UserRolesKey, tt.roles)

			result := HasSystemAdminRole(ctx)
			if result != tt.expected {
				t.Errorf("HasSystemAdminRole() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasAdminRole(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "有平台管理员角色",
			roles:    []string{model.RoleSystemAdmin},
			expected: true,
		},
		{
			name:     "有租户管理员角色",
			roles:    []string{model.RoleTenantAdmin},
			expected: true,
		},
		{
			name:     "有两个管理员角色",
			roles:    []string{model.RoleSystemAdmin, model.RoleTenantAdmin},
			expected: true,
		},
		{
			name:     "没有管理员角色",
			roles:    []string{"user"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, UserRolesKey, tt.roles)

			result := HasAdminRole(ctx)
			if result != tt.expected {
				t.Errorf("HasAdminRole() = %v, want %v", result, tt.expected)
			}
		})
	}
}
