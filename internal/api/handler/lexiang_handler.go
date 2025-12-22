package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"genkit-ai-service/internal/logger"
	_ "genkit-ai-service/internal/model" // 用于 Swagger 文档生成
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/lexiang"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"
)

// LexiangHandler 乐享知识库处理器
// 提供乐享知识库的代理访问功能
type LexiangHandler struct {
	client    lexiang.LexiangClient
	logger    logger.Logger
	validator *validator.Validator
}

// NewLexiangHandler 创建乐享处理器实例
func NewLexiangHandler(client lexiang.LexiangClient, log logger.Logger) *LexiangHandler {
	return &LexiangHandler{
		client:    client,
		logger:    log,
		validator: validator.New(),
	}
}

// ============================================================================
// Swagger 请求/响应类型定义
// ============================================================================

// CreateSpaceRequestSwagger 创建知识库请求
// @name CreateSpaceRequest
type CreateSpaceRequestSwagger struct {
	TeamID string `json:"teamId" validate:"required" example:"team_123"`
	Name   string `json:"name" validate:"required,min=1,max=255" example:"我的知识库"`
}

// CreateFolderRequestSwagger 创建文件夹请求
// @name CreateFolderRequest
type CreateFolderRequestSwagger struct {
	ParentID string `json:"parentId" validate:"required" example:"entry_123"`
	Name     string `json:"name" validate:"required,min=1,max=255" example:"新文件夹"`
}

// CreateFileEntryRequestSwagger 创建文件节点请求
// @name CreateFileEntryRequest
type CreateFileEntryRequestSwagger struct {
	ParentID  string `json:"parentId" validate:"required" example:"entry_123"`
	State     string `json:"state" validate:"required" example:"upload_state_xxx"`
	EntryType string `json:"entryType" validate:"required,oneof=file video audio" example:"file"`
	Name      string `json:"name,omitempty" example:"文档.pdf"`
}

// ReuploadFileRequestSwagger 重新上传文件请求
// @name ReuploadFileRequest
type ReuploadFileRequestSwagger struct {
	State string `json:"state" validate:"required" example:"upload_state_xxx"`
}

// UploadSignRequestSwagger 获取上传签名请求
// @name UploadSignRequest
type UploadSignRequestSwagger struct {
	FileName  string `json:"fileName" validate:"required" example:"document.pdf"`
	MediaType string `json:"mediaType" validate:"required,oneof=file video audio" example:"file"`
}

// SpaceItemSwagger 知识库项
type SpaceItemSwagger struct {
	ID          string `json:"id" example:"space_123"`
	Name        string `json:"name" example:"我的知识库"`
	Logo        string `json:"logo,omitempty" example:"https://example.com/logo.png"`
	RootEntryID string `json:"rootEntryId" example:"entry_root_123"`
}

// SpaceResponseSwagger 知识库响应
type SpaceResponseSwagger struct {
	ID                 string `json:"id" example:"space_123"`
	Name               string `json:"name" example:"我的知识库"`
	Logo               string `json:"logo,omitempty"`
	VisibleType        int    `json:"visibleType" example:"0"`
	ManagerInheritType string `json:"managerInheritType" example:"manager"`
	MemberInheritType  string `json:"memberInheritType" example:"viewer"`
	TeamID             string `json:"teamId" example:"team_123"`
	RootEntryID        string `json:"rootEntryId" example:"entry_root_123"`
}

// EntryItemSwagger 知识节点项
type EntryItemSwagger struct {
	ID          string `json:"id" example:"entry_123"`
	Name        string `json:"name" example:"文档.pdf"`
	EntryType   string `json:"entryType" example:"file"`
	HasChildren bool   `json:"hasChildren" example:"false"`
}

// EntryResponseSwagger 知识节点响应
type EntryResponseSwagger struct {
	ID                string `json:"id" example:"entry_123"`
	Name              string `json:"name" example:"文档.pdf"`
	EntryType         string `json:"entryType" example:"file"`
	HasChildren       bool   `json:"hasChildren" example:"false"`
	CreatedAt         string `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt         string `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
	MemberInheritType string `json:"memberInheritType" example:"viewer"`
	DownloadURL       string `json:"downloadUrl,omitempty" example:"https://example.com/download"`
}

// EntryContentResponseSwagger 线上文档内容响应
type EntryContentResponseSwagger struct {
	Name        string `json:"name" example:"在线文档"`
	HTMLContent string `json:"htmlContent" example:"<p>文档内容</p>"`
}

// UploadSignResponseSwagger 上传签名响应
type UploadSignResponseSwagger struct {
	State     string `json:"state" example:"upload_state_xxx"`
	Key       string `json:"key" example:"path/to/file"`
	Bucket    string `json:"bucket" example:"bucket-name"`
	Region    string `json:"region" example:"ap-guangzhou"`
	UploadURL string `json:"uploadUrl" example:"https://bucket.cos.region.myqcloud.com/path/to/file"`
}

// FeedbackItemSwagger 反馈项
type FeedbackItemSwagger struct {
	ID         string `json:"id" example:"feedback_123"`
	Status     string `json:"status" example:"unprocessed"`
	Type       string `json:"type" example:"kb_content_mistake"`
	Content    string `json:"content" example:"内容有误"`
	CreatedAt  string `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	ReviewedAt string `json:"reviewedAt,omitempty"`
	OwnerID    string `json:"ownerId" example:"user_123"`
	EntryID    string `json:"entryId" example:"entry_123"`
}

// ============================================================================
// 知识库管理接口
// ============================================================================

// HandleCreateSpace 创建知识库
// @Summary 创建知识库
// @Description 在指定团队下创建新的知识库
// @Tags Lexiang Knowledge Base
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateSpaceRequestSwagger true "创建知识库请求"
// @Success 201 {object} model.LexiangSpaceDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/spaces [post]
func (h *LexiangHandler) HandleCreateSpace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateSpaceRequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	space, err := h.client.CreateSpace(ctx, req.TeamID, req.Name)
	if err != nil {
		h.logger.Error("创建知识库失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertSpaceResponse(space)
	h.writeJSONResponse(w, http.StatusCreated, response.SuccessWithMessageContext(ctx, "创建知识库成功", resp))
}

// HandleGetSpace 获取知识库详情
// @Summary 获取知识库详情
// @Description 根据知识库ID获取详细信息
// @Tags Lexiang Knowledge Base
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识库ID" example:"space_123"
// @Success 200 {object} model.LexiangSpaceDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 404 {object} model.ErrorResponse "知识库不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/spaces/{id} [get]
func (h *LexiangHandler) HandleGetSpace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	spaceID := h.extractPathParam(r.URL.Path, "spaces")
	if spaceID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("知识库ID不能为空"))
		return
	}

	space, err := h.client.GetSpace(ctx, spaceID)
	if err != nil {
		h.logger.Error("获取知识库详情失败", logger.Fields{"error": err, "spaceId": spaceID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertSpaceResponse(space)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, resp))
}

// HandleListSpaces 获取知识库列表
// @Summary 获取知识库列表
// @Description 获取指定团队下的知识库列表，支持分页
// @Tags Lexiang Knowledge Base
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param teamId query string true "团队ID" example:"team_123"
// @Param limit query int false "每页数量" default(20) example:20
// @Param pageToken query string false "分页游标"
// @Success 200 {object} model.LexiangSpaceListDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/spaces [get]
func (h *LexiangHandler) HandleListSpaces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := r.URL.Query().Get("teamId")
	if teamID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("teamId 参数不能为空"))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	pageToken := r.URL.Query().Get("pageToken")

	spaces, err := h.client.ListSpaces(ctx, teamID, limit, pageToken)
	if err != nil {
		h.logger.Error("获取知识库列表失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertSpaceListResponse(spaces)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, &resp))
}

// HandleUpdateSpace 更新知识库
// @Summary 更新知识库
// @Description 更新知识库名称
// @Tags Lexiang Knowledge Base
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识库ID" example:"space_123"
// @Param request body object{name=string} true "更新请求"
// @Success 200 {object} model.LexiangSpaceDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "知识库不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/spaces/{id} [put]
func (h *LexiangHandler) HandleUpdateSpace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceID := h.extractPathParam(r.URL.Path, "spaces")
	if spaceID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("知识库ID不能为空"))
		return
	}

	var req struct {
		Name string `json:"name" validate:"required,min=1,max=255"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	space, err := h.client.UpdateSpace(ctx, spaceID, req.Name)
	if err != nil {
		h.logger.Error("更新知识库失败", logger.Fields{"error": err, "spaceId": spaceID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertSpaceResponse(space)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithMessageContext(ctx, "更新知识库成功", resp))
}

// HandleDeleteSpace 删除知识库
// @Summary 删除知识库
// @Description 删除指定的知识库
// @Tags Lexiang Knowledge Base
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识库ID" example:"space_123"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "知识库不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/spaces/{id} [delete]
func (h *LexiangHandler) HandleDeleteSpace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceID := h.extractPathParam(r.URL.Path, "spaces")
	if spaceID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("知识库ID不能为空"))
		return
	}

	if err := h.client.DeleteSpace(ctx, spaceID); err != nil {
		h.logger.Error("删除知识库失败", logger.Fields{"error": err, "spaceId": spaceID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithMessageContext[any](ctx, "删除知识库成功", nil))
}

// ============================================================================
// 知识节点管理接口
// ============================================================================

// HandleCreateFolder 创建文件夹
// @Summary 创建文件夹
// @Description 在指定父节点下创建文件夹
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateFolderRequestSwagger true "创建文件夹请求"
// @Success 201 {object} model.LexiangEntryDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries/folder [post]
func (h *LexiangHandler) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateFolderRequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	entry, err := h.client.CreateFolder(ctx, req.ParentID, req.Name)
	if err != nil {
		h.logger.Error("创建文件夹失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertEntryResponse(entry)
	h.writeJSONResponse(w, http.StatusCreated, response.SuccessWithMessageContext(ctx, "创建文件夹成功", resp))
}

// HandleCreateFileEntry 创建文件节点
// @Summary 创建文件节点
// @Description 使用上传的 state 创建文件类型的知识节点
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateFileEntryRequestSwagger true "创建文件节点请求"
// @Success 201 {object} model.LexiangEntryDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries/file [post]
func (h *LexiangHandler) HandleCreateFileEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateFileEntryRequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	entry, err := h.client.CreateFileEntry(ctx, req.ParentID, req.State, lexiang.EntryType(req.EntryType), req.Name)
	if err != nil {
		h.logger.Error("创建文件节点失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertEntryResponse(entry)
	h.writeJSONResponse(w, http.StatusCreated, response.SuccessWithMessageContext(ctx, "创建文件节点成功", resp))
}

// HandleGetEntry 获取知识节点详情
// @Summary 获取知识节点详情
// @Description 根据节点ID获取详细信息
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "节点ID" example:"entry_123"
// @Success 200 {object} model.LexiangEntryDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 404 {object} model.ErrorResponse "节点不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries/{id} [get]
func (h *LexiangHandler) HandleGetEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	entryID := h.extractPathParam(r.URL.Path, "entries")
	if entryID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("节点ID不能为空"))
		return
	}

	entry, err := h.client.GetEntry(ctx, entryID)
	if err != nil {
		h.logger.Error("获取节点详情失败", logger.Fields{"error": err, "entryId": entryID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertEntryResponse(entry)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, resp))
}

// HandleListEntries 获取知识节点列表
// @Summary 获取知识节点列表
// @Description 获取指定知识库或父节点下的知识节点列表
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param spaceId query string true "知识库ID" example:"space_123"
// @Param parentId query string false "父节点ID（空表示根目录）"
// @Param limit query int false "每页数量" default(20)
// @Param pageToken query string false "分页游标"
// @Success 200 {object} model.LexiangEntryListDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries [get]
func (h *LexiangHandler) HandleListEntries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	spaceID := r.URL.Query().Get("spaceId")
	if spaceID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("spaceId 参数不能为空"))
		return
	}

	parentID := r.URL.Query().Get("parentId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	pageToken := r.URL.Query().Get("pageToken")

	entries, err := h.client.ListEntries(ctx, spaceID, parentID, limit, pageToken)
	if err != nil {
		h.logger.Error("获取节点列表失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertEntryListResponse(entries)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, &resp))
}

// HandleDeleteEntry 删除知识节点
// @Summary 删除知识节点
// @Description 删除指定的知识节点
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "节点ID" example:"entry_123"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "节点不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries/{id} [delete]
func (h *LexiangHandler) HandleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryID := h.extractPathParam(r.URL.Path, "entries")
	if entryID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("节点ID不能为空"))
		return
	}

	if err := h.client.DeleteEntry(ctx, entryID); err != nil {
		h.logger.Error("删除节点失败", logger.Fields{"error": err, "entryId": entryID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithMessageContext[any](ctx, "删除节点成功", nil))
}

// HandleReuploadFile 重新上传文件
// @Summary 重新上传文件
// @Description 更新指定节点的文件内容
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "节点ID" example:"entry_123"
// @Param request body ReuploadFileRequestSwagger true "重新上传请求"
// @Success 200 {object} model.AnyDataResponse "上传成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "节点不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries/{id}/reupload [post]
func (h *LexiangHandler) HandleReuploadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryID := h.extractPathParam(r.URL.Path, "entries")
	if entryID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("节点ID不能为空"))
		return
	}

	var req ReuploadFileRequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if err := h.client.ReuploadFile(ctx, entryID, req.State); err != nil {
		h.logger.Error("重新上传文件失败", logger.Fields{"error": err, "entryId": entryID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithMessageContext[any](ctx, "重新上传成功", nil))
}

// HandleGetEntryContent 获取线上文档内容
// @Summary 获取线上文档内容
// @Description 获取线上文档的 HTML 内容（仅支持 entry_type=page 的节点）
// @Tags Lexiang Entries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "节点ID" example:"entry_123"
// @Success 200 {object} model.LexiangEntryContentDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 404 {object} model.ErrorResponse "节点不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/entries/{id}/content [get]
func (h *LexiangHandler) HandleGetEntryContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	entryID := h.extractPathParam(r.URL.Path, "entries")
	if entryID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("节点ID不能为空"))
		return
	}

	content, err := h.client.GetEntryContent(ctx, entryID)
	if err != nil {
		h.logger.Error("获取文档内容失败", logger.Fields{"error": err, "entryId": entryID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := &EntryContentResponseSwagger{
		Name:        content.Name,
		HTMLContent: content.HTMLContent,
	}
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, resp))
}

// ============================================================================
// 文件上传接口
// ============================================================================

// HandleGetUploadSign 获取上传签名
// @Summary 获取上传签名
// @Description 获取文件上传到腾讯云 COS 的签名信息
// @Tags Lexiang Upload
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UploadSignRequestSwagger true "获取上传签名请求"
// @Success 200 {object} model.LexiangUploadSignDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/upload/sign [post]
func (h *LexiangHandler) HandleGetUploadSign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req UploadSignRequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	sign, err := h.client.GetUploadSign(ctx, req.FileName, req.MediaType)
	if err != nil {
		h.logger.Error("获取上传签名失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertUploadSignResponse(sign)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, resp))
}

// HandleUploadFile 上传文件（完整流程）
// @Summary 上传文件
// @Description 完整的文件上传流程：获取签名 -> 上传到 COS -> 返回 state
// @Tags Lexiang Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "要上传的文件"
// @Param mediaType formData string true "媒体类型" Enums(file, video, audio)
// @Success 200 {object} model.LexiangUploadStateDataResponse "上传成功，返回 state 用于创建知识节点"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/upload [post]
func (h *LexiangHandler) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 解析 multipart form，最大 100MB
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("解析表单失败: "+err.Error()))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("获取上传文件失败: "+err.Error()))
		return
	}
	defer file.Close()

	mediaType := r.FormValue("mediaType")
	if mediaType == "" {
		mediaType = "file"
	}

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("读取文件内容失败: "+err.Error()))
		return
	}

	// 调用完整上传流程
	state, err := h.client.UploadFile(ctx, header.Filename, mediaType, fileData)
	if err != nil {
		h.logger.Error("上传文件失败", logger.Fields{"error": err.Error(), "fileName": header.Filename})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := map[string]string{"state": state}
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithMessageContext(ctx, "上传文件成功", &resp))
}

// ============================================================================
// 附件下载接口
// ============================================================================

// HandleGetDocFile 获取附件详情
// @Summary 获取附件详情
// @Description 获取附件的详情信息，包含下载链接
// @Tags Lexiang Download
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "附件ID" example:"file_123"
// @Success 200 {object} model.LexiangDocFileDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 404 {object} model.ErrorResponse "附件不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/files/{id} [get]
func (h *LexiangHandler) HandleGetDocFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fileID := h.extractPathParam(r.URL.Path, "files")
	if fileID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("附件ID不能为空"))
		return
	}

	docFile, err := h.client.GetDocFile(ctx, fileID)
	if err != nil {
		h.logger.Error("获取附件详情失败", logger.Fields{"error": err, "fileId": fileID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := map[string]string{
		"id":          docFile.Data.ID,
		"name":        docFile.Data.Attributes.Name,
		"downloadUrl": docFile.Data.Links.Download,
	}
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, &resp))
}

// HandleDownloadDocFile 下载附件
// @Summary 下载附件
// @Description 下载附件内容
// @Tags Lexiang Download
// @Accept json
// @Produce octet-stream
// @Security BearerAuth
// @Param id path string true "附件ID" example:"file_123"
// @Success 200 {file} binary "文件内容"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 404 {object} model.ErrorResponse "附件不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/files/{id}/download [get]
func (h *LexiangHandler) HandleDownloadDocFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fileID := h.extractPathParam(r.URL.Path, "files")
	if fileID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("附件ID不能为空"))
		return
	}

	fileData, fileName, err := h.client.DownloadDocFile(ctx, fileID)
	if err != nil {
		h.logger.Error("下载附件失败", logger.Fields{"error": err, "fileId": fileID})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(fileData)))
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)
}

// ============================================================================
// 知识反馈接口
// ============================================================================

// HandleListFeedbacks 获取知识反馈列表
// @Summary 获取知识反馈列表
// @Description 获取指定知识库的用户反馈列表
// @Tags Lexiang Feedback
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param spaceId query string true "知识库ID" example:"space_123"
// @Param limit query int false "每页数量" default(20)
// @Param pageToken query string false "分页游标"
// @Success 200 {object} model.LexiangFeedbackListDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/feedbacks [get]
func (h *LexiangHandler) HandleListFeedbacks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	spaceID := r.URL.Query().Get("spaceId")
	if spaceID == "" {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("spaceId 参数不能为空"))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	pageToken := r.URL.Query().Get("pageToken")

	feedbacks, err := h.client.ListFeedbacks(ctx, spaceID, limit, pageToken)
	if err != nil {
		h.logger.Error("获取反馈列表失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	resp := h.convertFeedbackListResponse(feedbacks)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, &resp))
}

// ============================================================================
// 辅助方法
// ============================================================================

// extractPathParam 从URL路径中提取参数
// 路径格式: /api/v1/lexiang/{resource}/{id}
func (h *LexiangHandler) extractPathParam(path, resource string) string {
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			// 检查下一个部分是否是子路由
			nextPart := parts[i+1]
			if nextPart != "folder" && nextPart != "file" && nextPart != "sign" && nextPart != "download" && nextPart != "content" && nextPart != "reupload" {
				return nextPart
			}
		}
	}
	return ""
}

// writeJSONResponse 写入 JSON 响应
func (h *LexiangHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *LexiangHandler) writeErrorResponse(w http.ResponseWriter, ctx context.Context, appErr *errors.AppError) {
	resp := response.ErrorWithContext[any](ctx, appErr.Code, appErr.Message)
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeNotFound:
		statusCode = http.StatusNotFound
	case errors.CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.CodeForbidden:
		statusCode = http.StatusForbidden
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	}
	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *LexiangHandler) writeValidationErrorResponse(w http.ResponseWriter, ctx context.Context, validationErrors []validator.ValidationError) {
	errorData := map[string]interface{}{"errors": validationErrors}
	resp := response.ErrorWithDataContext(ctx, errors.CodeValidationError, errors.MsgValidationError, &errorData)
	h.writeJSONResponse(w, http.StatusUnprocessableEntity, resp)
}

// convertLexiangError 转换乐享错误为应用错误
func (h *LexiangHandler) convertLexiangError(err error) *errors.AppError {
	if lexErr, ok := err.(*lexiang.LexiangError); ok {
		switch lexErr.StatusCode {
		case 400:
			return errors.NewBadRequestError(lexErr.Message)
		case 401:
			return errors.NewUnauthorizedError(lexErr.Message)
		case 403:
			return errors.NewForbiddenError(lexErr.Message)
		case 404:
			return errors.NewNotFoundError(lexErr.Message)
		case 429:
			return errors.NewServiceUnavailableError("请求频率超限，请稍后重试")
		default:
			return errors.NewInternalError(err)
		}
	}
	return errors.NewInternalError(err)
}

// convertSpaceResponse 转换知识库响应
func (h *LexiangHandler) convertSpaceResponse(space *lexiang.SpaceResponse) *SpaceResponseSwagger {
	return &SpaceResponseSwagger{
		ID:                 space.Data.ID,
		Name:               space.Data.Attributes.Name,
		Logo:               space.Data.Attributes.Logo,
		VisibleType:        space.Data.Attributes.VisibleType,
		ManagerInheritType: space.Data.Attributes.ManagerInheritType,
		MemberInheritType:  space.Data.Attributes.MemberInheritType,
		TeamID:             space.Data.Relationships.Team.Data.ID,
		RootEntryID:        space.Data.Relationships.RootEntry.Data.ID,
	}
}

// convertSpaceListResponse 转换知识库列表响应
func (h *LexiangHandler) convertSpaceListResponse(spaces *lexiang.SpaceListResponse) []SpaceItemSwagger {
	result := make([]SpaceItemSwagger, len(spaces.Data))
	for i, s := range spaces.Data {
		result[i] = SpaceItemSwagger{
			ID:          s.ID,
			Name:        s.Attributes.Name,
			Logo:        s.Attributes.Logo,
			RootEntryID: s.Relationships.RootEntry.Data.ID,
		}
	}
	return result
}

// convertEntryResponse 转换知识节点响应
func (h *LexiangHandler) convertEntryResponse(entry *lexiang.EntryResponse) *EntryResponseSwagger {
	return &EntryResponseSwagger{
		ID:                entry.Data.ID,
		Name:              entry.Data.Attributes.Name,
		EntryType:         entry.Data.Attributes.EntryType,
		HasChildren:       entry.Data.Attributes.HasChildren,
		CreatedAt:         entry.Data.Attributes.CreatedAt,
		UpdatedAt:         entry.Data.Attributes.UpdatedAt,
		MemberInheritType: entry.Data.Attributes.MemberInheritType,
		DownloadURL:       entry.Data.Links.Download,
	}
}

// convertEntryListResponse 转换知识节点列表响应
func (h *LexiangHandler) convertEntryListResponse(entries *lexiang.EntryListResponse) []EntryItemSwagger {
	result := make([]EntryItemSwagger, len(entries.Data))
	for i, e := range entries.Data {
		result[i] = EntryItemSwagger{
			ID:          e.ID,
			Name:        e.Attributes.Name,
			EntryType:   e.Attributes.EntryType,
			HasChildren: e.Attributes.HasChildren,
		}
	}
	return result
}

// convertUploadSignResponse 转换上传签名响应
func (h *LexiangHandler) convertUploadSignResponse(sign *lexiang.UploadSignResponse) *UploadSignResponseSwagger {
	uploadURL := "https://" + sign.Options.Bucket + ".cos." + sign.Options.Region + ".myqcloud.com/" + sign.Object.Key
	return &UploadSignResponseSwagger{
		State:     sign.Object.State,
		Key:       sign.Object.Key,
		Bucket:    sign.Options.Bucket,
		Region:    sign.Options.Region,
		UploadURL: uploadURL,
	}
}

// convertFeedbackListResponse 转换反馈列表响应
func (h *LexiangHandler) convertFeedbackListResponse(feedbacks *lexiang.FeedbackListResponse) []FeedbackItemSwagger {
	result := make([]FeedbackItemSwagger, len(feedbacks.Data))
	for i, f := range feedbacks.Data {
		result[i] = FeedbackItemSwagger{
			ID:         f.ID,
			Status:     f.Attributes.Status,
			Type:       f.Attributes.Type,
			Content:    f.Attributes.Content,
			CreatedAt:  f.Attributes.CreatedAt,
			ReviewedAt: f.Attributes.ReviewedAt,
			OwnerID:    f.Relationships.Owner.Data.ID,
			EntryID:    f.Relationships.Entry.Data.ID,
		}
	}
	return result
}

// ============================================================================
// AI 问答和搜索接口
// ============================================================================

// AIQARequestSwagger AI问答请求
// @name AIQARequest
type AIQARequestSwagger struct {
	Query            string            `json:"query" validate:"required,max=1024" example:"如何使用知识库？"`
	Stream           bool              `json:"stream,omitempty" example:"false"`
	AnonymousStaffID string            `json:"anonymousStaffId,omitempty" example:"anonymous_user_123456"`
	SkipFAQ          bool              `json:"skipFaq,omitempty" example:"false"`
	NewSession       bool              `json:"newSession,omitempty" example:"true"`
	SessionID        string            `json:"sessionId,omitempty" example:"session_abc123"`
	QAMode           string            `json:"qaMode,omitempty" example:"normal" enums:"normal,normal-ds-v3,normal-ds-v3.1,reasoning,reasoning-ds-v3.1,research,research-ds-v3.1"`
	MaxChars         int               `json:"maxChars,omitempty" example:"2000"`
	Language         string            `json:"language,omitempty" example:"zh-CN" enums:"zh-CN,en"`
	Targets          []AITargetSwagger `json:"targets,omitempty"`
}

// AITargetSwagger AI知识范围目标
type AITargetSwagger struct {
	Type string `json:"type" example:"space" enums:"space,team,team_code,kb_entry"`
	ID   string `json:"id" example:"space_123"`
}

// AISearchRequestSwagger AI搜索请求
// @name AISearchRequest
type AISearchRequestSwagger struct {
	Query   string            `json:"query" validate:"required,max=1024" example:"知识库使用指南"`
	TopN    int               `json:"topN" validate:"required,min=1,max=50" example:"10"`
	Targets []AITargetSwagger `json:"targets,omitempty"`
}

// ReferenceChunkSwagger 参考内容段落
type ReferenceChunkSwagger struct {
	Content    string `json:"content" example:"这是匹配的内容段落..."`
	TargetID   string `json:"targetId" example:"entry_123"`
	TargetType string `json:"targetType" example:"kb_entry"`
	Title      string `json:"title" example:"知识库使用指南"`
	URL        string `json:"url" example:"https://lexiang.tencent.com/kb/entry/123"`
}

// ReferenceDocSwagger 参考文档来源
type ReferenceDocSwagger struct {
	Title string `json:"title" example:"知识库使用指南"`
	URL   string `json:"url" example:"https://lexiang.tencent.com/kb/entry/123"`
}

// AdditionalContentSwagger 附加内容信息
type AdditionalContentSwagger struct {
	GeneratedQuestion string                  `json:"generatedQuestion,omitempty" example:"如何使用乐享知识库？"`
	ReferenceChunks   []ReferenceChunkSwagger `json:"referenceChunks,omitempty"`
	ReferenceDocs     []ReferenceDocSwagger   `json:"referenceDocs,omitempty"`
}

// AIQAResponseSwagger AI问答响应数据
type AIQAResponseSwagger struct {
	Content           string                    `json:"content" example:"知识库是用于存储和管理文档的工具..."`
	AnswerSource      string                    `json:"answerSource" example:"internal"`
	ReasoningContent  string                    `json:"reasoningContent,omitempty" example:"让我思考一下这个问题..."`
	SessionID         string                    `json:"sessionId" example:"session_abc123"`
	AdditionalContent *AdditionalContentSwagger `json:"additionalContent,omitempty"`
}

// AISearchResultItemSwagger AI搜索结果项
type AISearchResultItemSwagger struct {
	Title   string `json:"title" example:"知识库使用指南"`
	Content string `json:"content" example:"## 如何使用知识库\n\n知识库是..."`
	URL     string `json:"url" example:"https://lexiang.tencent.com/kb/entry/123"`
}

// AISearchResponseSwagger AI搜索响应数据
type AISearchResponseSwagger struct {
	List []AISearchResultItemSwagger `json:"list"`
}

// HandleAIQA AI问答（非流式）
// @Summary AI问答
// @Description 向乐享AI助手提问，获取基于知识库内容的回答
// @Tags Lexiang AI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AIQARequestSwagger true "AI问答请求"
// @Success 200 {object} model.LexiangAIQADataResponse "问答成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/ai/qa [post]
func (h *LexiangHandler) HandleAIQA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AIQARequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	// 转换请求
	qaReq := h.convertAIQARequest(&req)

	// 调用 AI 问答
	qaResp, err := h.client.AIQA(ctx, qaReq)
	if err != nil {
		h.logger.Error("AI问答失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	// 转换响应
	resp := h.convertAIQAResponse(qaResp)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, resp))
}

// HandleAIQAStream AI问答（流式）
// @Summary AI问答（流式）
// @Description 向乐享AI助手提问，以 Server-Sent Events 流式返回回答
// @Tags Lexiang AI
// @Accept json
// @Produce text/event-stream
// @Security BearerAuth
// @Param request body AIQARequestSwagger true "AI问答请求"
// @Success 200 {string} string "SSE 流式响应"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/ai/qa/stream [post]
func (h *LexiangHandler) HandleAIQAStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AIQARequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	// 转换请求
	qaReq := h.convertAIQARequest(&req)

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 获取 Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeErrorResponse(w, ctx, errors.NewInternalError(nil))
		return
	}

	// 调用流式 AI 问答
	eventChan, errChan := h.client.AIQAStream(ctx, qaReq)

	// 处理流式响应
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// 通道关闭，发送结束标记
				w.Write([]byte("data: [DONE]\n\n"))
				flusher.Flush()
				return
			}

			// 转换并发送事件
			eventData := h.convertAIQAStreamEvent(event)
			jsonData, err := json.Marshal(eventData)
			if err != nil {
				h.logger.Error("序列化流式事件失败", logger.Fields{"error": err})
				continue
			}

			w.Write([]byte("data: "))
			w.Write(jsonData)
			w.Write([]byte("\n\n"))
			flusher.Flush()

		case err, ok := <-errChan:
			if ok && err != nil {
				h.logger.Error("AI问答流式响应错误", logger.Fields{"error": err})
				// 发送错误事件
				errorEvent := map[string]string{"error": err.Error()}
				jsonData, _ := json.Marshal(errorEvent)
				w.Write([]byte("data: "))
				w.Write(jsonData)
				w.Write([]byte("\n\n"))
				flusher.Flush()
			}
			return

		case <-ctx.Done():
			return
		}
	}
}

// HandleAISearch AI搜索
// @Summary AI搜索
// @Description 使用AI在知识库中搜索相关文档
// @Tags Lexiang AI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AISearchRequestSwagger true "AI搜索请求"
// @Success 200 {object} model.LexiangAISearchDataResponse "搜索成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /lexiang/ai/search [post]
func (h *LexiangHandler) HandleAISearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AISearchRequestSwagger
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, ctx, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.writeValidationErrorResponse(w, ctx, validationErrors)
		return
	}

	// 转换请求
	searchReq := h.convertAISearchRequest(&req)

	// 调用 AI 搜索
	searchResp, err := h.client.AISearch(ctx, searchReq)
	if err != nil {
		h.logger.Error("AI搜索失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, ctx, h.convertLexiangError(err))
		return
	}

	// 转换响应
	resp := h.convertAISearchResponse(searchResp)
	h.writeJSONResponse(w, http.StatusOK, response.SuccessWithContext(ctx, resp))
}

// convertAIQARequest 转换 AI 问答请求
func (h *LexiangHandler) convertAIQARequest(req *AIQARequestSwagger) *lexiang.AIQARequest {
	qaReq := &lexiang.AIQARequest{
		Query:            req.Query,
		Stream:           req.Stream,
		AnonymousStaffID: req.AnonymousStaffID,
		SkipFAQ:          req.SkipFAQ,
		NewSession:       req.NewSession,
		SessionID:        req.SessionID,
		QAMode:           lexiang.QAMode(req.QAMode),
		MaxChars:         req.MaxChars,
		Language:         req.Language,
	}

	if len(req.Targets) > 0 {
		qaReq.Targets = make([]lexiang.Target, len(req.Targets))
		for i, t := range req.Targets {
			qaReq.Targets[i] = lexiang.Target{
				Type: lexiang.TargetType(t.Type),
				ID:   t.ID,
			}
		}
	}

	return qaReq
}

// convertAIQAResponse 转换 AI 问答响应
func (h *LexiangHandler) convertAIQAResponse(resp *lexiang.AIQAResponse) *AIQAResponseSwagger {
	if resp.Data == nil {
		return &AIQAResponseSwagger{}
	}

	result := &AIQAResponseSwagger{
		Content:          resp.Data.Content,
		AnswerSource:     resp.Data.AnswerSource,
		ReasoningContent: resp.Data.ReasoningContent,
		SessionID:        resp.Data.SessionID,
	}

	if resp.Data.AdditionalContent != nil {
		result.AdditionalContent = h.convertAdditionalContent(resp.Data.AdditionalContent)
	}

	return result
}

// convertAdditionalContent 转换附加内容
func (h *LexiangHandler) convertAdditionalContent(ac *lexiang.AdditionalContent) *AdditionalContentSwagger {
	result := &AdditionalContentSwagger{
		GeneratedQuestion: ac.GeneratedQuestion,
	}

	if len(ac.ReferenceChunks) > 0 {
		result.ReferenceChunks = make([]ReferenceChunkSwagger, len(ac.ReferenceChunks))
		for i, chunk := range ac.ReferenceChunks {
			result.ReferenceChunks[i] = ReferenceChunkSwagger{
				Content:    chunk.Content,
				TargetID:   chunk.TargetID,
				TargetType: chunk.TargetType,
				Title:      chunk.Title,
				URL:        chunk.URL,
			}
		}
	}

	if len(ac.ReferenceDocs) > 0 {
		result.ReferenceDocs = make([]ReferenceDocSwagger, len(ac.ReferenceDocs))
		for i, doc := range ac.ReferenceDocs {
			result.ReferenceDocs[i] = ReferenceDocSwagger{
				Title: doc.Title,
				URL:   doc.URL,
			}
		}
	}

	return result
}

// convertAIQAStreamEvent 转换流式事件
func (h *LexiangHandler) convertAIQAStreamEvent(event *lexiang.AIQAStreamEvent) map[string]interface{} {
	result := make(map[string]interface{})

	if event.CompletionID != "" {
		result["completionId"] = event.CompletionID
	}
	if event.SessionID != "" {
		result["sessionId"] = event.SessionID
	}
	if event.DeltaContent != "" {
		result["deltaContent"] = event.DeltaContent
	}
	if event.Content != "" {
		result["content"] = event.Content
	}
	if event.FinishReason != "" {
		result["finishReason"] = event.FinishReason
	}
	if event.IsStop {
		result["isStop"] = event.IsStop
	}
	if event.AnswerSource != "" {
		result["answerSource"] = event.AnswerSource
	}
	if len(event.Processes) > 0 {
		processes := make([]map[string]string, len(event.Processes))
		for i, p := range event.Processes {
			processes[i] = map[string]string{"message": p.Message}
		}
		result["processes"] = processes
	}
	if event.AdditionalContent != nil {
		result["additionalContent"] = h.convertAdditionalContent(event.AdditionalContent)
	}

	return result
}

// convertAISearchRequest 转换 AI 搜索请求
func (h *LexiangHandler) convertAISearchRequest(req *AISearchRequestSwagger) *lexiang.AISearchRequest {
	searchReq := &lexiang.AISearchRequest{
		Query: req.Query,
		TopN:  req.TopN,
	}

	if len(req.Targets) > 0 {
		searchReq.Targets = make([]lexiang.Target, len(req.Targets))
		for i, t := range req.Targets {
			searchReq.Targets[i] = lexiang.Target{
				Type: lexiang.TargetType(t.Type),
				ID:   t.ID,
			}
		}
	}

	return searchReq
}

// convertAISearchResponse 转换 AI 搜索响应
func (h *LexiangHandler) convertAISearchResponse(resp *lexiang.AISearchResponse) *AISearchResponseSwagger {
	if resp.Data == nil {
		return &AISearchResponseSwagger{List: []AISearchResultItemSwagger{}}
	}

	result := &AISearchResponseSwagger{
		List: make([]AISearchResultItemSwagger, len(resp.Data.List)),
	}

	for i, item := range resp.Data.List {
		result.List[i] = AISearchResultItemSwagger{
			Title:   item.Title,
			Content: item.Content,
			URL:     item.URL,
		}
	}

	return result
}
