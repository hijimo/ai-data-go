package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"your-project/internal/repository"
	"your-project/internal/service"
)

// MigrationHandler 数据迁移处理器
type MigrationHandler struct {
	migrationService service.MigrationService
}

// NewMigrationHandler 创建数据迁移处理器实例
func NewMigrationHandler(migrationService service.MigrationService) *MigrationHandler {
	return &MigrationHandler{
		migrationService: migrationService,
	}
}

// ExportProjectRequest 导出项目请求
type ExportProjectRequest struct {
	ProjectID string `uri:"project_id" binding:"required,uuid"`
}

// ImportProjectRequest 导入项目请求
type ImportProjectRequest struct {
	ProjectID string                        `uri:"project_id" binding:"required,uuid"`
	Data      *repository.ProjectImportData `json:"data" binding:"required"`
}

// CreateMigrationTaskRequest 创建迁移任务请求
type CreateMigrationTaskRequest struct {
	SourceProjectID string                     `json:"source_project_id" binding:"required,uuid"`
	TargetProjectID string                     `json:"target_project_id" binding:"required,uuid"`
	ImportOptions   *repository.ImportOptions  `json:"import_options" binding:"required"`
}

// GetProjectStatsRequest 获取项目统计请求
type GetProjectStatsRequest struct {
	ProjectID string `uri:"project_id" binding:"required,uuid"`
}

// GetTaskStatusRequest 获取任务状态请求
type GetTaskStatusRequest struct {
	TaskID string `uri:"task_id" binding:"required,uuid"`
}

// CancelTaskRequest 取消任务请求
type CancelTaskRequest struct {
	TaskID string `uri:"task_id" binding:"required,uuid"`
}

// ExportProject 导出项目数据
// @Summary 导出项目数据
// @Description 导出指定项目的所有数据，包括文件、Agent、对话等
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param project_id path string true "项目ID"
// @Success 200 {object} repository.ProjectExportData "导出的项目数据"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 403 {object} ErrorResponse "权限不足"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/projects/{project_id}/export [get]
func (h *MigrationHandler) ExportProject(c *gin.Context) {
	var req ExportProjectRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
			Details: err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_PROJECT_ID",
			Message: "项目ID格式无效",
		})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 导出项目数据
	exportData, err := h.migrationService.ExportProject(c.Request.Context(), projectID, userUUID)
	if err != nil {
		if err.Error() == "用户无权限访问该项目" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "INSUFFICIENT_PERMISSION",
				Message: "权限不足",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "EXPORT_FAILED",
			Message: "导出项目数据失败",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, exportData)
}

// ImportProject 导入项目数据
// @Summary 导入项目数据
// @Description 将数据导入到指定项目中
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param project_id path string true "目标项目ID"
// @Param request body ImportProjectRequest true "导入数据"
// @Success 200 {object} model.Task "导入任务信息"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 403 {object} ErrorResponse "权限不足"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/projects/{project_id}/import [post]
func (h *MigrationHandler) ImportProject(c *gin.Context) {
	var req ImportProjectRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
			Details: err.Error(),
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST_BODY",
			Message: "请求体格式无效",
			Details: err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_PROJECT_ID",
			Message: "项目ID格式无效",
		})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 导入项目数据
	task, err := h.migrationService.ImportProject(c.Request.Context(), projectID, req.Data, userUUID)
	if err != nil {
		if err.Error() == "用户无权限修改该项目" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "INSUFFICIENT_PERMISSION",
				Message: "权限不足",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "IMPORT_FAILED",
			Message: "导入项目数据失败",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetProjectStats 获取项目数据统计
// @Summary 获取项目数据统计
// @Description 获取项目的数据统计信息，包括文件数量、大小等
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param project_id path string true "项目ID"
// @Success 200 {object} repository.ProjectDataStats "项目数据统计"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 403 {object} ErrorResponse "权限不足"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/projects/{project_id}/stats [get]
func (h *MigrationHandler) GetProjectStats(c *gin.Context) {
	var req GetProjectStatsRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
			Details: err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_PROJECT_ID",
			Message: "项目ID格式无效",
		})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 获取项目数据统计
	stats, err := h.migrationService.GetProjectStats(c.Request.Context(), projectID, userUUID)
	if err != nil {
		if err.Error() == "用户无权限访问该项目" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "INSUFFICIENT_PERMISSION",
				Message: "权限不足",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "GET_STATS_FAILED",
			Message: "获取项目统计失败",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateMigrationTask 创建数据迁移任务
// @Summary 创建数据迁移任务
// @Description 创建从源项目到目标项目的数据迁移任务
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param request body CreateMigrationTaskRequest true "迁移任务参数"
// @Success 200 {object} model.Task "迁移任务信息"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 403 {object} ErrorResponse "权限不足"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/migration/tasks [post]
func (h *MigrationHandler) CreateMigrationTask(c *gin.Context) {
	var req CreateMigrationTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
			Details: err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	sourceProjectID, err := uuid.Parse(req.SourceProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_SOURCE_PROJECT_ID",
			Message: "源项目ID格式无效",
		})
		return
	}

	targetProjectID, err := uuid.Parse(req.TargetProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_TARGET_PROJECT_ID",
			Message: "目标项目ID格式无效",
		})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 创建迁移任务
	task, err := h.migrationService.CreateMigrationTask(c.Request.Context(), sourceProjectID, targetProjectID, req.ImportOptions, userUUID)
	if err != nil {
		if err.Error() == "用户无权限访问源项目" || err.Error() == "用户无权限修改目标项目" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "INSUFFICIENT_PERMISSION",
				Message: "权限不足",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "CREATE_TASK_FAILED",
			Message: "创建迁移任务失败",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetMigrationTaskStatus 获取迁移任务状态
// @Summary 获取迁移任务状态
// @Description 获取指定迁移任务的状态和进度
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} model.Task "任务状态信息"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 403 {object} ErrorResponse "权限不足"
// @Failure 404 {object} ErrorResponse "任务不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/migration/tasks/{task_id} [get]
func (h *MigrationHandler) GetMigrationTaskStatus(c *gin.Context) {
	var req GetTaskStatusRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
			Details: err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_TASK_ID",
			Message: "任务ID格式无效",
		})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 获取任务状态
	task, err := h.migrationService.GetMigrationTaskStatus(c.Request.Context(), taskID, userUUID)
	if err != nil {
		if err.Error() == "用户无权限访问该任务" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "INSUFFICIENT_PERMISSION",
				Message: "权限不足",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "GET_TASK_STATUS_FAILED",
			Message: "获取任务状态失败",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// CancelMigrationTask 取消迁移任务
// @Summary 取消迁移任务
// @Description 取消正在进行的迁移任务
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} SuccessResponse "取消成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 403 {object} ErrorResponse "权限不足"
// @Failure 404 {object} ErrorResponse "任务不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/migration/tasks/{task_id}/cancel [post]
func (h *MigrationHandler) CancelMigrationTask(c *gin.Context) {
	var req CancelTaskRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效",
			Details: err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_TASK_ID",
			Message: "任务ID格式无效",
		})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 取消任务
	err = h.migrationService.CancelMigrationTask(c.Request.Context(), taskID, userUUID)
	if err != nil {
		if err.Error() == "用户无权限取消该任务" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "INSUFFICIENT_PERMISSION",
				Message: "权限不足",
			})
			return
		}
		if err.Error() == "只能取消正在处理的任务" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "INVALID_TASK_STATUS",
				Message: "只能取消正在处理的任务",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "CANCEL_TASK_FAILED",
			Message: "取消任务失败",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "任务已成功取消",
	})
}

// ListMigrationTasks 获取迁移任务列表
// @Summary 获取迁移任务列表
// @Description 获取用户的迁移任务列表，支持分页
// @Tags 数据迁移
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param status query int false "任务状态过滤"
// @Success 200 {object} PaginatedResponse "任务列表"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/migration/tasks [get]
func (h *MigrationHandler) ListMigrationTasks(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "用户未认证",
		})
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "用户ID格式无效",
		})
		return
	}

	// 这里可以实现获取任务列表的逻辑
	// 由于篇幅限制，这里只返回一个示例响应
	c.JSON(http.StatusOK, PaginatedResponse{
		Data:       []interface{}{},
		Total:      0,
		Page:       page,
		Limit:      limit,
		TotalPages: 0,
	})
}