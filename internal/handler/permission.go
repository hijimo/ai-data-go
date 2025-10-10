package handler

import (
	"net/http"

	"ai-knowledge-platform/internal/auth"
	"ai-knowledge-platform/internal/middleware"
	"github.com/gin-gonic/gin"
)

// PermissionHandler 权限处理器
type PermissionHandler struct {
	roleManager *auth.RoleManager
}

// NewPermissionHandler 创建权限处理器
func NewPermissionHandler(roleManager *auth.RoleManager) *PermissionHandler {
	return &PermissionHandler{
		roleManager: roleManager,
	}
}

// RoleResponse 角色响应结构
type RoleResponse struct {
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name"`
	Description string             `json:"description"`
	Permissions []auth.Permission  `json:"permissions"`
}

// UserPermissionResponse 用户权限响应结构
type UserPermissionResponse struct {
	UserID      string            `json:"user_id"`
	ProjectID   string            `json:"project_id,omitempty"`
	Roles       []string          `json:"roles"`
	Permissions []auth.Permission `json:"permissions"`
}

// ListRoles 获取所有角色列表
// @Summary 获取所有角色列表
// @Description 获取系统中定义的所有角色及其权限
// @Tags 权限管理
// @Security BearerAuth
// @Produce json
// @Success 200 {array} RoleResponse "角色列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Router /api/v1/permissions/roles [get]
func (h *PermissionHandler) ListRoles(c *gin.Context) {
	roles := h.roleManager.GetAllRoles()
	
	response := make([]RoleResponse, len(roles))
	for i, role := range roles {
		response[i] = RoleResponse{
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			Permissions: role.Permissions,
		}
	}
	
	c.JSON(http.StatusOK, response)
}

// GetRole 获取指定角色详情
// @Summary 获取角色详情
// @Description 获取指定角色的详细信息和权限列表
// @Tags 权限管理
// @Security BearerAuth
// @Produce json
// @Param role_name path string true "角色名称"
// @Success 200 {object} RoleResponse "角色详情"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Failure 404 {object} map[string]interface{} "角色不存在"
// @Router /api/v1/permissions/roles/{role_name} [get]
func (h *PermissionHandler) GetRole(c *gin.Context) {
	roleName := c.Param("role_name")
	
	role, err := h.roleManager.GetRole(roleName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "角色不存在",
			"message": err.Error(),
		})
		return
	}
	
	response := RoleResponse{
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		Permissions: role.Permissions,
	}
	
	c.JSON(http.StatusOK, response)
}

// GetUserPermissions 获取当前用户权限
// @Summary 获取当前用户权限
// @Description 获取当前登录用户的角色和权限信息
// @Tags 权限管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} UserPermissionResponse "用户权限信息"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /api/v1/permissions/user [get]
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	projectID := middleware.GetProjectID(c)
	userRoles := middleware.GetUserRoles(c)
	
	permissions := h.roleManager.GetUserPermissions(userRoles)
	
	response := UserPermissionResponse{
		UserID:      userID,
		ProjectID:   projectID,
		Roles:       userRoles,
		Permissions: permissions,
	}
	
	c.JSON(http.StatusOK, response)
}

// CheckPermission 检查用户权限
// @Summary 检查用户权限
// @Description 检查当前用户是否拥有指定权限
// @Tags 权限管理
// @Security BearerAuth
// @Produce json
// @Param permission query string true "权限名称"
// @Success 200 {object} map[string]interface{} "权限检查结果"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /api/v1/permissions/check [get]
func (h *PermissionHandler) CheckPermission(c *gin.Context) {
	permissionStr := c.Query("permission")
	if permissionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "参数错误",
			"message": "缺少权限参数",
		})
		return
	}
	
	permission := auth.ParsePermission(permissionStr)
	userRoles := middleware.GetUserRoles(c)
	
	hasPermission := h.roleManager.CheckUserPermissions(userRoles, []auth.Permission{permission})
	
	c.JSON(http.StatusOK, gin.H{
		"permission":     permission,
		"has_permission": hasPermission,
		"user_roles":     userRoles,
	})
}

// CheckMultiplePermissions 批量检查用户权限
// @Summary 批量检查用户权限
// @Description 批量检查当前用户是否拥有多个权限
// @Tags 权限管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param permissions body []string true "权限列表"
// @Success 200 {object} map[string]interface{} "权限检查结果"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /api/v1/permissions/check-multiple [post]
func (h *PermissionHandler) CheckMultiplePermissions(c *gin.Context) {
	var permissionStrs []string
	if err := c.ShouldBindJSON(&permissionStrs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}
	
	permissions := auth.ParsePermissions(permissionStrs)
	userRoles := middleware.GetUserRoles(c)
	
	results := make(map[string]bool)
	for _, permission := range permissions {
		results[string(permission)] = h.roleManager.CheckUserPermissions(userRoles, []auth.Permission{permission})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"permissions": results,
		"user_roles":  userRoles,
	})
}

// GetPermissionsByResource 按资源获取权限列表
// @Summary 按资源获取权限列表
// @Description 获取指定资源类型的所有权限
// @Tags 权限管理
// @Security BearerAuth
// @Produce json
// @Param resource query string true "资源类型" Enums(project,document,vector,llm,agent,chat,question,answer,task,dataset,training,system)
// @Success 200 {object} map[string]interface{} "权限列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /api/v1/permissions/by-resource [get]
func (h *PermissionHandler) GetPermissionsByResource(c *gin.Context) {
	resource := c.Query("resource")
	if resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "参数错误",
			"message": "缺少资源类型参数",
		})
		return
	}
	
	// 定义资源权限映射
	resourcePermissions := map[string][]auth.Permission{
		"project": {
			auth.PermProjectRead, auth.PermProjectWrite, 
			auth.PermProjectDelete, auth.PermProjectManage,
		},
		"document": {
			auth.PermDocumentRead, auth.PermDocumentWrite,
			auth.PermDocumentDelete, auth.PermDocumentUpload,
		},
		"vector": {
			auth.PermVectorRead, auth.PermVectorWrite,
			auth.PermVectorDelete, auth.PermVectorSearch,
		},
		"llm": {
			auth.PermLLMRead, auth.PermLLMWrite,
			auth.PermLLMManage, auth.PermLLMChat,
		},
		"agent": {
			auth.PermAgentRead, auth.PermAgentWrite,
			auth.PermAgentDelete, auth.PermAgentManage, auth.PermAgentChat,
		},
		"chat": {
			auth.PermChatRead, auth.PermChatWrite, auth.PermChatDelete,
		},
		"question": {
			auth.PermQuestionRead, auth.PermQuestionWrite,
			auth.PermQuestionDelete, auth.PermQuestionGenerate,
		},
		"answer": {
			auth.PermAnswerRead, auth.PermAnswerWrite,
			auth.PermAnswerDelete, auth.PermAnswerGenerate,
		},
		"task": {
			auth.PermTaskRead, auth.PermTaskWrite, auth.PermTaskCancel,
		},
		"dataset": {
			auth.PermDatasetRead, auth.PermDatasetExport,
		},
		"training": {
			auth.PermTrainingRead, auth.PermTrainingWrite, auth.PermTrainingCancel,
		},
		"system": {
			auth.PermSystemAdmin, auth.PermUserManage,
		},
	}
	
	permissions, exists := resourcePermissions[resource]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "参数错误",
			"message": "不支持的资源类型",
		})
		return
	}
	
	// 检查用户对这些权限的拥有情况
	userRoles := middleware.GetUserRoles(c)
	permissionStatus := make(map[string]bool)
	for _, permission := range permissions {
		permissionStatus[string(permission)] = h.roleManager.CheckUserPermissions(userRoles, []auth.Permission{permission})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"resource":    resource,
		"permissions": permissionStatus,
		"user_roles":  userRoles,
	})
}