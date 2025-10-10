package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleManager_RegisterRole(t *testing.T) {
	rm := NewRoleManager()
	
	customRole := Role{
		Name:        "custom_role",
		DisplayName: "自定义角色",
		Description: "测试用自定义角色",
		Permissions: []Permission{PermProjectRead, PermDocumentRead},
	}
	
	rm.RegisterRole(customRole)
	
	role, err := rm.GetRole("custom_role")
	assert.NoError(t, err)
	assert.Equal(t, "custom_role", role.Name)
	assert.Equal(t, "自定义角色", role.DisplayName)
	assert.Len(t, role.Permissions, 2)
}

func TestRoleManager_GetRole(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试获取存在的角色
	role, err := rm.GetRole("system_admin")
	assert.NoError(t, err)
	assert.Equal(t, "system_admin", role.Name)
	assert.Equal(t, "系统管理员", role.DisplayName)
	
	// 测试获取不存在的角色
	_, err = rm.GetRole("non_existent_role")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "角色不存在")
}

func TestRoleManager_HasPermission(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试系统管理员权限
	assert.True(t, rm.HasPermission("system_admin", PermSystemAdmin))
	assert.True(t, rm.HasPermission("system_admin", PermProjectRead))
	assert.True(t, rm.HasPermission("system_admin", PermDocumentWrite))
	
	// 测试项目查看者权限
	assert.True(t, rm.HasPermission("project_viewer", PermProjectRead))
	assert.True(t, rm.HasPermission("project_viewer", PermDocumentRead))
	assert.False(t, rm.HasPermission("project_viewer", PermDocumentWrite))
	assert.False(t, rm.HasPermission("project_viewer", PermProjectDelete))
	
	// 测试不存在的角色
	assert.False(t, rm.HasPermission("non_existent_role", PermProjectRead))
}

func TestRoleManager_HasAnyPermission(t *testing.T) {
	rm := NewRoleManager()
	
	permissions := []Permission{PermProjectWrite, PermDocumentWrite}
	
	// 项目成员应该拥有文档写权限
	assert.True(t, rm.HasAnyPermission("project_member", permissions))
	
	// 项目查看者不应该拥有任何写权限
	assert.False(t, rm.HasAnyPermission("project_viewer", permissions))
	
	// 系统管理员应该拥有所有权限
	assert.True(t, rm.HasAnyPermission("system_admin", permissions))
}

func TestRoleManager_HasAllPermissions(t *testing.T) {
	rm := NewRoleManager()
	
	permissions := []Permission{PermProjectRead, PermDocumentRead}
	
	// 项目成员应该拥有这两个权限
	assert.True(t, rm.HasAllPermissions("project_member", permissions))
	
	// 项目查看者也应该拥有这两个权限
	assert.True(t, rm.HasAllPermissions("project_viewer", permissions))
	
	// 测试包含写权限的情况
	writePermissions := []Permission{PermProjectRead, PermDocumentWrite}
	assert.True(t, rm.HasAllPermissions("project_member", writePermissions))
	assert.False(t, rm.HasAllPermissions("project_viewer", writePermissions))
}

func TestRoleManager_CheckUserPermissions(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试单个角色
	userRoles := []string{"project_member"}
	requiredPermissions := []Permission{PermProjectRead, PermDocumentRead}
	
	assert.True(t, rm.CheckUserPermissions(userRoles, requiredPermissions))
	
	// 测试权限不足
	requiredPermissions = []Permission{PermProjectDelete}
	assert.False(t, rm.CheckUserPermissions(userRoles, requiredPermissions))
	
	// 测试多个角色
	userRoles = []string{"project_viewer", "project_member"}
	requiredPermissions = []Permission{PermDocumentWrite}
	assert.True(t, rm.CheckUserPermissions(userRoles, requiredPermissions))
	
	// 测试空权限要求
	assert.True(t, rm.CheckUserPermissions(userRoles, []Permission{}))
}

func TestRoleManager_CheckUserAnyPermission(t *testing.T) {
	rm := NewRoleManager()
	
	userRoles := []string{"project_viewer"}
	permissions := []Permission{PermDocumentWrite, PermDocumentRead}
	
	// 项目查看者拥有读权限，应该返回true
	assert.True(t, rm.CheckUserAnyPermission(userRoles, permissions))
	
	// 测试都没有的权限
	permissions = []Permission{PermDocumentWrite, PermDocumentDelete}
	assert.False(t, rm.CheckUserAnyPermission(userRoles, permissions))
	
	// 测试空权限列表
	assert.True(t, rm.CheckUserAnyPermission(userRoles, []Permission{}))
}

func TestRoleManager_GetUserPermissions(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试单个角色
	userRoles := []string{"project_viewer"}
	permissions := rm.GetUserPermissions(userRoles)
	
	assert.Contains(t, permissions, PermProjectRead)
	assert.Contains(t, permissions, PermDocumentRead)
	assert.NotContains(t, permissions, PermDocumentWrite)
	
	// 测试多个角色（权限应该合并）
	userRoles = []string{"project_viewer", "project_member"}
	permissions = rm.GetUserPermissions(userRoles)
	
	assert.Contains(t, permissions, PermProjectRead)
	assert.Contains(t, permissions, PermDocumentRead)
	assert.Contains(t, permissions, PermDocumentWrite)
	
	// 测试空角色列表
	permissions = rm.GetUserPermissions([]string{})
	assert.Empty(t, permissions)
}

func TestRoleManager_IsSystemAdmin(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试系统管理员
	assert.True(t, rm.IsSystemAdmin([]string{"system_admin"}))
	assert.True(t, rm.IsSystemAdmin([]string{"project_member", "system_admin"}))
	
	// 测试非系统管理员
	assert.False(t, rm.IsSystemAdmin([]string{"project_member"}))
	assert.False(t, rm.IsSystemAdmin([]string{"project_owner"}))
	assert.False(t, rm.IsSystemAdmin([]string{}))
}

func TestRoleManager_IsProjectOwner(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试项目所有者
	assert.True(t, rm.IsProjectOwner([]string{"project_owner"}))
	assert.True(t, rm.IsProjectOwner([]string{"project_member", "project_owner"}))
	
	// 测试非项目所有者
	assert.False(t, rm.IsProjectOwner([]string{"project_member"}))
	assert.False(t, rm.IsProjectOwner([]string{"system_admin"}))
	assert.False(t, rm.IsProjectOwner([]string{}))
}

func TestParsePermission(t *testing.T) {
	// 测试权限解析
	perm := ParsePermission("PROJECT:READ")
	assert.Equal(t, Permission("project:read"), perm)
	
	perm = ParsePermission("document:write")
	assert.Equal(t, Permission("document:write"), perm)
}

func TestParsePermissions(t *testing.T) {
	// 测试权限列表解析
	permStrs := []string{"PROJECT:READ", "DOCUMENT:WRITE", "vector:search"}
	permissions := ParsePermissions(permStrs)
	
	expected := []Permission{
		Permission("project:read"),
		Permission("document:write"),
		Permission("vector:search"),
	}
	
	assert.Equal(t, expected, permissions)
}

func TestPredefinedRoles(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试所有预定义角色都已注册
	expectedRoles := []string{
		"system_admin",
		"project_owner", 
		"project_admin",
		"project_member",
		"project_viewer",
	}
	
	for _, roleName := range expectedRoles {
		role, err := rm.GetRole(roleName)
		assert.NoError(t, err, "角色 %s 应该存在", roleName)
		assert.NotEmpty(t, role.DisplayName, "角色 %s 应该有显示名称", roleName)
		assert.NotEmpty(t, role.Description, "角色 %s 应该有描述", roleName)
		assert.NotEmpty(t, role.Permissions, "角色 %s 应该有权限", roleName)
	}
}

func TestRoleHierarchy(t *testing.T) {
	rm := NewRoleManager()
	
	// 测试角色权限层次
	// 系统管理员应该拥有最多权限
	systemAdminPerms := rm.GetUserPermissions([]string{"system_admin"})
	projectOwnerPerms := rm.GetUserPermissions([]string{"project_owner"})
	projectMemberPerms := rm.GetUserPermissions([]string{"project_member"})
	projectViewerPerms := rm.GetUserPermissions([]string{"project_viewer"})
	
	// 系统管理员权限应该最多
	assert.True(t, len(systemAdminPerms) >= len(projectOwnerPerms))
	assert.True(t, len(projectOwnerPerms) >= len(projectMemberPerms))
	assert.True(t, len(projectMemberPerms) >= len(projectViewerPerms))
	
	// 项目查看者应该只有读权限
	for _, perm := range projectViewerPerms {
		permStr := string(perm)
		assert.True(t, 
			containsAny(permStr, []string{":read", ":search", ":chat"}),
			"项目查看者权限 %s 应该是只读类型", permStr)
	}
}

// 辅助函数：检查字符串是否包含任意一个子字符串
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}