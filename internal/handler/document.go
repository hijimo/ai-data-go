package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ai-knowledge-platform/internal/processor"
	"ai-knowledge-platform/internal/service"
)

// DocumentHandler 文档处理器
type DocumentHandler struct {
	documentService service.DocumentService
}

// NewDocumentHandler 创建文档处理器
func NewDocumentHandler(documentService service.DocumentService) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
	}
}

// ProcessDocument 处理文档
// @Summary 处理文档
// @Description 解析文档并进行分块处理
// @Tags 文档处理
// @Accept json
// @Produce json
// @Param request body ProcessDocumentRequest true "处理文档请求"
// @Success 200 {object} Response{data=service.ProcessDocumentResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/documents/process [post]
func (h *DocumentHandler) ProcessDocument(c *gin.Context) {
	var req ProcessDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("请求参数错误: "+err.Error()))
		return
	}

	// 构建服务请求
	serviceReq := &service.ProcessDocumentRequest{
		FileID: req.FileID,
		ChunkConfig: &processor.ChunkConfig{
			Strategy:        processor.ChunkStrategy(req.ChunkConfig.Strategy),
			MaxSize:         req.ChunkConfig.MaxSize,
			Overlap:         req.ChunkConfig.Overlap,
			Separators:      req.ChunkConfig.Separators,
			PreserveContext: req.ChunkConfig.PreserveContext,
		},
		Options: &service.DocumentProcessingOptions{
			ExtractImages: req.Options.ExtractImages,
			ExtractTables: req.Options.ExtractTables,
			ExtractLinks:  req.Options.ExtractLinks,
			CleanContent:  req.Options.CleanContent,
		},
	}

	// 调用服务层
	resp, err := h.documentService.ProcessDocument(c.Request.Context(), serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("处理文档失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(resp))
}

// GetDocumentVersions 获取文档版本列表
// @Summary 获取文档版本列表
// @Description 获取指定文件的所有文档版本
// @Tags 文档处理
// @Produce json
// @Param file_id path string true "文件ID"
// @Success 200 {object} Response{data=[]model.DocumentVersion}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/documents/{file_id}/versions [get]
func (h *DocumentHandler) GetDocumentVersions(c *gin.Context) {
	// 获取文件ID
	fileIDStr := c.Param("file_id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的文件ID格式"))
		return
	}

	// 调用服务层
	versions, err := h.documentService.GetDocumentVersions(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文档版本失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(versions))
}

// GetDocumentVersion 获取文档版本详情
// @Summary 获取文档版本详情
// @Description 获取指定版本的详细信息，包括文档内容和分块信息
// @Tags 文档处理
// @Produce json
// @Param version_id path string true "版本ID"
// @Success 200 {object} Response{data=service.DocumentVersionDetail}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/documents/versions/{version_id} [get]
func (h *DocumentHandler) GetDocumentVersion(c *gin.Context) {
	// 获取版本ID
	versionIDStr := c.Param("version_id")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的版本ID格式"))
		return
	}

	// 调用服务层
	detail, err := h.documentService.GetDocumentVersion(c.Request.Context(), versionID)
	if err != nil {
		if err.Error() == "文档版本不存在" {
			c.JSON(http.StatusNotFound, ErrorResponse("文档版本不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文档版本详情失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(detail))
}

// GetChunks 获取文档块列表
// @Summary 获取文档块列表
// @Description 获取指定版本的所有文档块
// @Tags 文档处理
// @Produce json
// @Param version_id path string true "版本ID"
// @Success 200 {object} Response{data=[]model.Chunk}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/documents/versions/{version_id}/chunks [get]
func (h *DocumentHandler) GetChunks(c *gin.Context) {
	// 获取版本ID
	versionIDStr := c.Param("version_id")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的版本ID格式"))
		return
	}

	// 调用服务层
	chunks, err := h.documentService.GetChunks(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文档块失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(chunks))
}

// PreviewChunks 预览分块结果
// @Summary 预览分块结果
// @Description 预览指定配置下的分块结果，不保存到数据库
// @Tags 文档处理
// @Accept json
// @Produce json
// @Param request body PreviewChunksRequest true "预览分块请求"
// @Success 200 {object} Response{data=PreviewChunksResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/documents/preview-chunks [post]
func (h *DocumentHandler) PreviewChunks(c *gin.Context) {
	var req PreviewChunksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("请求参数错误: "+err.Error()))
		return
	}

	// 创建临时文档
	doc := &processor.Document{
		Content: req.Content,
		Title:   "预览文档",
	}

	// 构建分块配置
	config := &processor.ChunkConfig{
		Strategy:        processor.ChunkStrategy(req.ChunkConfig.Strategy),
		MaxSize:         req.ChunkConfig.MaxSize,
		Overlap:         req.ChunkConfig.Overlap,
		Separators:      req.ChunkConfig.Separators,
		PreserveContext: req.ChunkConfig.PreserveContext,
	}

	// 执行分块
	chunkerManager := processor.NewChunkerManager()
	chunks, err := chunkerManager.ChunkDocument(doc, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("分块预览失败: "+err.Error()))
		return
	}

	// 构建响应
	var chunkPreviews []ChunkPreview
	totalTokens := 0
	totalSize := 0

	for i, chunk := range chunks {
		preview := ChunkPreview{
			Index:      i + 1,
			Content:    h.truncateContent(chunk.Content, 200), // 限制预览内容长度
			TokenCount: chunk.TokenCount,
			Size:       len(chunk.Content),
		}
		chunkPreviews = append(chunkPreviews, preview)
		totalTokens += chunk.TokenCount
		totalSize += len(chunk.Content)
	}

	resp := PreviewChunksResponse{
		ChunkCount:  len(chunks),
		Chunks:      chunkPreviews,
		TotalTokens: totalTokens,
		TotalSize:   totalSize,
	}

	c.JSON(http.StatusOK, SuccessResponse(resp))
}

// truncateContent 截断内容用于预览
func (h *DocumentHandler) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// 请求和响应结构体

// ProcessDocumentRequest 处理文档请求
type ProcessDocumentRequest struct {
	FileID      uuid.UUID                  `json:"file_id" binding:"required"`
	ChunkConfig ChunkConfigRequest         `json:"chunk_config" binding:"required"`
	Options     DocumentProcessingOptions  `json:"options"`
}

// ChunkConfigRequest 分块配置请求
type ChunkConfigRequest struct {
	Strategy        string   `json:"strategy" binding:"required"`
	MaxSize         int      `json:"max_size" binding:"required,min=100,max=10000"`
	Overlap         int      `json:"overlap" binding:"min=0"`
	Separators      []string `json:"separators"`
	PreserveContext bool     `json:"preserve_context"`
}

// DocumentProcessingOptions 文档处理选项
type DocumentProcessingOptions struct {
	ExtractImages bool `json:"extract_images"`
	ExtractTables bool `json:"extract_tables"`
	ExtractLinks  bool `json:"extract_links"`
	CleanContent  bool `json:"clean_content"`
}

// PreviewChunksRequest 预览分块请求
type PreviewChunksRequest struct {
	Content     string             `json:"content" binding:"required"`
	ChunkConfig ChunkConfigRequest `json:"chunk_config" binding:"required"`
}

// PreviewChunksResponse 预览分块响应
type PreviewChunksResponse struct {
	ChunkCount  int            `json:"chunk_count"`
	Chunks      []ChunkPreview `json:"chunks"`
	TotalTokens int            `json:"total_tokens"`
	TotalSize   int            `json:"total_size"`
}

// ChunkPreview 分块预览
type ChunkPreview struct {
	Index      int    `json:"index"`
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`
	Size       int    `json:"size"`
}

// RegisterDocumentRoutes 注册文档处理相关路由
func RegisterDocumentRoutes(r *gin.RouterGroup, handler *DocumentHandler) {
	documents := r.Group("/documents")
	{
		documents.POST("/process", handler.ProcessDocument)
		documents.POST("/preview-chunks", handler.PreviewChunks)
		documents.GET("/:file_id/versions", handler.GetDocumentVersions)
		documents.GET("/versions/:version_id", handler.GetDocumentVersion)
		documents.GET("/versions/:version_id/chunks", handler.GetChunks)
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}