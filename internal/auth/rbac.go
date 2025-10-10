package auth

import (
	"errors"
	"strings"
)

// Permission 权限类型
type Permission string

// 系统权限定义
const (
	// 项目权限
	PermProjectRead   Permission = "project:read"
	PermProjectWrite  Permission = "project:write"
	PermProjectDelete Permission = "project:delete"
	PermProjectManage Permission = "project:manage"

	// 文档权限
	PermDocumentRead   Permission = "document:read"
	PermDocumentWrite  Permission = "document:write"
	PermDocumentDelete Permission = "document:delete"
	PermDocumentUpload Permission = "document:upload"

	// 向量权限
	PermVectorRead   Permission = "vector:read"
	PermVectorWrite  Permission = "vector:write"
	PermVectorDelete Permission = "vector:delete"
	PermVectorSearch Permission = "vector:search"

	// LLM权限
	PermLLMRead   Permission = "llm:read"
	PermLLMWrite  Permission = "llm:write"
	PermLLMManage Permission = "llm:manage"
	PermLLMChat   Permission = "llm:chat"

	// Agent权限
	PermAgentRead   Permission = "agent:read"
	PermAgentWrite  Permission = "agent:write"
	PermAgentDelete Permission = "agent:delete"
	PermAgentManage Permission = "agent:manage"
	PermAgentChat   Permission = "agent:chat"

	// 对话权限
	PermChatRead   Permission = "chat:read"
	PermChatWrite  Permission = "chat:write"
	PermChatDelete Permission = "chat:delete"

	// 问题答案权限
	PermQuestionRead     Permission = "question:read"
	PermQuestionWrite    Permission = "question:write"
	PermQuestionDelete   Permission = "question:delete"
	PermQuestionGenerate Permission = "question:generate"
	PermAnswerRead       Permission = "answer:read"
	PermAnswerWrite      Permission = "answer:write"
	PermAnswerDelete     Permission = "answer:delete"
	PermAnswerGenerate   Permission = "answer:generate"

	// 任务权限
	PermTaskRead   Permission = "task:read"
	PermTaskWrite  Permission = "task:write"
	PermTaskCancel Permission = "task:cancel"

	// 数据集权限
	PermDatasetRead   Permission = "dataset:read"
	PermDatasetExport Permission = "dataset:export"

	// 训练权限
	PermTrainingRead   Permission = "training:read"
	PermTrainingWrite  Permission = "training:write"
	PermTrainingCancel Permission = "training:cancel"

	// 系统管理权限
	PermSystemAdmin Permission = "system:admin"
	PermUserManage  Permission = "user:manage"
)

// Role 角色定义
type Role struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"display_name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// 预定义角色
var (
	// 系统管理员 - 拥有所有权限
	RoleSystemAdmin = Role{
		Name:        "system_admin",
		DisplayName: "系统管理员",
		Description: "拥有系统所有权限",
		Permissions: []Permission{
			PermSystemAdmin, PermUserManage,
			PermProjectRead, PermProjectWrite, PermProjectDelete, PermProjectManage,
			PermDocumentRead, PermDocumentWrite, PermDocumentDelete, PermDocumentUpload,
			PermVectorRead, PermVectorWrite, PermVectorDelete, PermVectorSearch,
			PermLLMRead, PermLLMWrite, PermLLMManage, PermLLMChat,
			PermAgentRead, PermAgentWrite, PermAgentDelete, PermAgentManage, PermAgentChat,
			PermChatRead, PermChatWrite, PermChatDelete,
			PermQuestionRead, PermQuestionWrite, PermQuestionDelete, PermQuestionGenerate,
			PermAnswerRead, PermAnswerWrite, PermAnswerDelete, PermAnswerGenerate,
			PermTaskRead, PermTaskWrite, PermTaskCancel,
			PermDatasetRead, PermDatasetExport,
			PermTrainingRead, PermTrainingWrite, PermTrainingCancel,
		},
	}

	// 项目所有者 - 项目内所有权限
	RoleProjectOwner = Role{
		Name:        "project_owner",
		DisplayName: "项目所有者",
		Description: "项目内拥有所有权限",
		Permissions: []Permission{
			PermProjectRead, PermProjectWrite, PermProjectDelete, PermProjectManage,
			PermDocumentRead, PermDocumentWrite, PermDocumentDelete, PermDocumentUpload,
			PermVectorRead, PermVectorWrite, PermVectorDelete, PermVectorSearch,
			PermLLMRead, PermLLMWrite, PermLLMManage, PermLLMChat,
			PermAgentRead, PermAgentWrite, PermAgentDelete, PermAgentManage, PermAgentChat,
			PermChatRead, PermChatWrite, PermChatDelete,
			PermQuestionRead, PermQuestionWrite, PermQuestionDelete, PermQuestionGenerate,
			PermAnswerRead, PermAnswerWrite, PermAnswerDelete, PermAnswerGenerate,
			PermTaskRead, PermTaskWrite, PermTaskCancel,
			PermDatasetRead, PermDatasetExport,
			PermTrainingRead, PermTrainingWrite, PermTrainingCancel,
		},
	}

	// 项目管理员 - 项目管理权限
	RoleProjectAdmin = Role{
		Name:        "project_admin",
		DisplayName: "项目管理员",
		Description: "项目管理和配置权限",
		Permissions: []Permission{
			PermProjectRead, PermProjectWrite, PermProjectManage,
			PermDocumentRead, PermDocumentWrite, PermDocumentDelete, PermDocumentUpload,
			PermVectorRead, PermVectorWrite, PermVectorDelete, PermVectorSearch,
			PermLLMRead, PermLLMWrite, PermLLMChat,
			PermAgentRead, PermAgentWrite, PermAgentDelete, PermAgentManage, PermAgentChat,
			PermChatRead, PermChatWrite, PermChatDelete,
			PermQuestionRead, PermQuestionWrite, PermQuestionDelete, PermQuestionGenerate,
			PermAnswerRead, PermAnswerWrite, PermAnswerDelete, PermAnswerGenerate,
			PermTaskRead, PermTaskWrite, PermTaskCancel,
			PermDatasetRead, PermDatasetExport,
			PermTrainingRead, PermTrainingWrite, PermTrainingCancel,
		},
	}

	// 项目成员 - 基本操作权限
	RoleProjectMember = Role{
		Name:        "project_member",
		DisplayName: "项目成员",
		Description: "项目基本操作权限",
		Permissions: []Permission{
			PermProjectRead,
			PermDocumentRead, PermDocumentWrite, PermDocumentUpload,
			PermVectorRead, PermVectorSearch,
			PermLLMRead, PermLLMChat,
			PermAgentRead, PermAgentWrite, PermAgentChat,
			PermChatRead, PermChatWrite,
			PermQuestionRead, PermQuestionWrite, PermQuestionGenerate,
			PermAnswerRead, PermAnswerWrite, PermAnswerGenerate,
			PermTaskRead,
			PermDatasetRead,
			PermTrainingRead,
		},
	}

	// 项目查看者 - 只读权限
	RoleProjectViewer = Role{
		Name:        "project_viewer",
		DisplayName: "项目查看者",
		Description: "项目只读权限",
		Permissions: []Permission{
			PermProjectRead,
			PermDocumentRead,
			PermVectorRead, PermVectorSearch,
			PermLLMRead, PermLLMChat,
			PermAgentRead, PermAgentChat,
			PermChatRead,
			PermQuestionRead,
			PermAnswerRead,
			PermTaskRead,
			PermDatasetRead,
			PermTrainingRead,
		},
	}
)

// RoleManager 角色管理器
type RoleManager struct {
	roles map[string]Role
}

// NewRoleManager 创建角色管理器
func NewRoleManager() *RoleManager {
	rm := &RoleManager{
		roles: make(map[string]Role),
	}

	// 注册预定义角色
	rm.RegisterRole(RoleSystemAdmin)
	rm.RegisterRole(RoleProjectOwner)
	rm.RegisterRole(RoleProjectAdmin)
	rm.RegisterRole(RoleProjectMember)
	rm.RegisterRole(RoleProjectViewer)

	return rm
}

// RegisterRole 注册角色
func (rm *RoleManager) RegisterRole(role Role) {
	rm.roles[role.Name] = role
}

// GetRole 获取角色
func (rm *RoleManager) GetRole(roleName string) (Role, error) {
	role, exists := rm.roles[roleName]
	if !exists {
		return Role{}, errors.New("角色不存在: " + roleName)
	}
	return role, nil
}

// GetAllRoles 获取所有角色
func (rm *RoleManager) GetAllRoles() []Role {
	roles := make([]Role, 0, len(rm.roles))
	for _, role := range rm.roles {
		roles = append(roles, role)
	}
	return roles
}

// HasPermission 检查角色是否拥有指定权限
func (rm *RoleManager) HasPermission(roleName string, permission Permission) bool {
	role, err := rm.GetRole(roleName)
	if err != nil {
		return false
	}

	for _, perm := range role.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission 检查角色是否拥有任意一个权限
func (rm *RoleManager) HasAnyPermission(roleName string, permissions []Permission) bool {
	for _, permission := range permissions {
		if rm.HasPermission(roleName, permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions 检查角色是否拥有所有权限
func (rm *RoleManager) HasAllPermissions(roleName string, permissions []Permission) bool {
	for _, permission := range permissions {
		if !rm.HasPermission(roleName, permission) {
			return false
		}
	}
	return true
}

// CheckUserPermissions 检查用户权限
func (rm *RoleManager) CheckUserPermissions(userRoles []string, requiredPermissions []Permission) bool {
	// 如果没有要求权限，则允许访问
	if len(requiredPermissions) == 0 {
		return true
	}

	// 检查用户的任意角色是否拥有所需权限
	for _, roleName := range userRoles {
		if rm.HasAllPermissions(roleName, requiredPermissions) {
			return true
		}
	}

	return false
}

// CheckUserAnyPermission 检查用户是否拥有任意一个权限
func (rm *RoleManager) CheckUserAnyPermission(userRoles []string, permissions []Permission) bool {
	// 如果没有要求权限，则允许访问
	if len(permissions) == 0 {
		return true
	}

	// 检查用户的任意角色是否拥有任意一个权限
	for _, roleName := range userRoles {
		if rm.HasAnyPermission(roleName, permissions) {
			return true
		}
	}

	return false
}

// GetUserPermissions 获取用户的所有权限
func (rm *RoleManager) GetUserPermissions(userRoles []string) []Permission {
	permissionSet := make(map[Permission]bool)

	// 收集所有角色的权限
	for _, roleName := range userRoles {
		role, err := rm.GetRole(roleName)
		if err != nil {
			continue
		}

		for _, permission := range role.Permissions {
			permissionSet[permission] = true
		}
	}

	// 转换为切片
	permissions := make([]Permission, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions
}

// IsSystemAdmin 检查用户是否为系统管理员
func (rm *RoleManager) IsSystemAdmin(userRoles []string) bool {
	for _, roleName := range userRoles {
		if roleName == RoleSystemAdmin.Name {
			return true
		}
	}
	return false
}

// IsProjectOwner 检查用户是否为项目所有者
func (rm *RoleManager) IsProjectOwner(userRoles []string) bool {
	for _, roleName := range userRoles {
		if roleName == RoleProjectOwner.Name {
			return true
		}
	}
	return false
}

// ParsePermission 解析权限字符串
func ParsePermission(permStr string) Permission {
	return Permission(strings.ToLower(permStr))
}

// ParsePermissions 解析权限字符串列表
func ParsePermissions(permStrs []string) []Permission {
	permissions := make([]Permission, len(permStrs))
	for i, permStr := range permStrs {
		permissions[i] = ParsePermission(permStr)
	}
	return permissions
}