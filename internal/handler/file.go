package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ai-knowledge-platform/internal/service"
)

// FileHandler 文件处理器
type FileHandler struct {
	fileService service.FileService
}

// NewFileHandler 创建文件处理器
func NewFileHandler(fileService service.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

// UploadFile 上传文件
// @Summary 上传文件
// @Description 上传文件到指定项目
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Param project_id formData string true "项目ID"
// @Param file formData file true "文件"
// @Param description formData string false "文件描述"
// @Success 200 {object} Response{data=service.UploadFileResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/files/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	// 获取项目ID
	projectIDStr := c.PostForm("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse("项目ID不能为空"))
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的项目ID格式"))
		return
	}

	// 获取上传的文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("获取上传文件失败: "+err.Error()))
		return
	}

	// 打开文件
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("打开文件失败: "+err.Error()))
		return
	}
	defer file.Close()

	// 获取用户ID（从JWT中间件设置）
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse("用户未认证"))
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的用户ID"))
		return
	}

	// 构建请求
	req := &service.UploadFileRequest{
		ProjectID:   projectID,
		File:        file,
		FileHeader:  fileHeader,
		UploaderID:  userID,
		Description: c.PostForm("description"),
	}

	// 调用服务层
	resp, err := h.fileService.UploadFile(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("上传文件失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(resp))
}

// GetFile 获取文件信息
// @Summary 获取文件信息
// @Description 根据文件ID获取文件详细信息
// @Tags 文件管理
// @Produce json
// @Param id path string true "文件ID"
// @Success 200 {object} Response{data=model.File}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /api/v1/files/{id} [get]
func (h *FileHandler) GetFile(c *gin.Context) {
	// 获取文件ID
	fileIDStr := c.Param("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的文件ID格式"))
		return
	}

	// 调用服务层
	file, err := h.fileService.GetFile(c.Request.Context(), fileID)
	if err != nil {
		if err.Error() == "文件不存在" {
			c.JSON(http.StatusNotFound, ErrorResponse("文件不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文件失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(file))
}

// ListFiles 获取文件列表
// @Summary 获取文件列表
// @Description 获取指定项目的文件列表
// @Tags 文件管理
// @Produce json
// @Param project_id query string true "项目ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页大小" default(20)
// @Param status query int false "文件状态过滤"
// @Success 200 {object} Response{data=service.ListFilesResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/files [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	// 获取项目ID
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse("项目ID不能为空"))
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的项目ID格式"))
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 获取状态过滤参数
	var status *int
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status = &s
		}
	}

	// 构建请求
	req := &service.ListFilesRequest{
		ProjectID: projectID,
		Page:      page,
		PageSize:  pageSize,
		Status:    status,
	}

	// 调用服务层
	resp, err := h.fileService.ListFiles(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文件列表失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(resp))
}

// DeleteFile 删除文件
// @Summary 删除文件
// @Description 软删除指定文件
// @Tags 文件管理
// @Produce json
// @Param id path string true "文件ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /api/v1/files/{id} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	// 获取文件ID
	fileIDStr := c.Param("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的文件ID格式"))
		return
	}

	// 调用服务层
	err = h.fileService.DeleteFile(c.Request.Context(), fileID)
	if err != nil {
		if err.Error() == "文件不存在" {
			c.JSON(http.StatusNotFound, ErrorResponse("文件不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse("删除文件失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse("文件删除成功"))
}

// GetFileURL 获取文件访问URL
// @Summary 获取文件访问URL
// @Description 获取文件的临时访问URL
// @Tags 文件管理
// @Produce json
// @Param id path string true "文件ID"
// @Param expiry query int false "过期时间(秒)" default(3600)
// @Success 200 {object} Response{data=map[string]string}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /api/v1/files/{id}/url [get]
func (h *FileHandler) GetFileURL(c *gin.Context) {
	// 获取文件ID
	fileIDStr := c.Param("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的文件ID格式"))
		return
	}

	// 获取过期时间参数
	expirySeconds, _ := strconv.Atoi(c.DefaultQuery("expiry", "3600"))
	expiry := time.Duration(expirySeconds) * time.Second

	// 调用服务层
	url, err := h.fileService.GetFileURL(c.Request.Context(), fileID, expiry)
	if err != nil {
		if err.Error() == "文件不存在" {
			c.JSON(http.StatusNotFound, ErrorResponse("文件不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文件URL失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(map[string]string{
		"url":    url,
		"expiry": strconv.Itoa(expirySeconds),
	}))
}

// RegisterFileRoutes 注册文件相关路由
func RegisterFileRoutes(r *gin.RouterGroup, handler *FileHandler) {
	files := r.Group("/files")
	{
		files.POST("/upload", handler.UploadFile)
		files.GET("", handler.ListFiles)
		files.GET("/:id", handler.GetFile)
		files.DELETE("/:id", handler.DeleteFile)
		files.GET("/:id/url", handler.GetFileURL)
	}
}