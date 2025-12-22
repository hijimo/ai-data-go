package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterLexiangRoutes 注册乐享知识库相关的 API 路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
func RegisterLexiangRoutes(
	mux *http.ServeMux,
	lexiangHandler *handler.LexiangHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 知识库管理路由 ==========

	// POST /api/v1/lexiang/spaces - 创建知识库
	mux.Handle("POST /api/v1/lexiang/spaces",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleCreateSpace))))

	// GET /api/v1/lexiang/spaces - 获取知识库列表
	mux.Handle("GET /api/v1/lexiang/spaces",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleListSpaces))))

	// GET /api/v1/lexiang/spaces/{id} - 获取知识库详情
	mux.Handle("GET /api/v1/lexiang/spaces/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleGetSpace))))

	// PUT /api/v1/lexiang/spaces/{id} - 更新知识库
	mux.Handle("PUT /api/v1/lexiang/spaces/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleUpdateSpace))))

	// DELETE /api/v1/lexiang/spaces/{id} - 删除知识库
	mux.Handle("DELETE /api/v1/lexiang/spaces/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleDeleteSpace))))

	// ========== 知识节点管理路由 ==========

	// POST /api/v1/lexiang/entries/folder - 创建文件夹
	mux.Handle("POST /api/v1/lexiang/entries/folder",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleCreateFolder))))

	// POST /api/v1/lexiang/entries/file - 创建文件节点
	mux.Handle("POST /api/v1/lexiang/entries/file",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleCreateFileEntry))))

	// GET /api/v1/lexiang/entries - 获取知识节点列表
	mux.Handle("GET /api/v1/lexiang/entries",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleListEntries))))

	// GET /api/v1/lexiang/entries/{id} - 获取知识节点详情
	mux.Handle("GET /api/v1/lexiang/entries/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleGetEntry))))

	// GET /api/v1/lexiang/entries/{id}/content - 获取线上文档内容
	mux.Handle("GET /api/v1/lexiang/entries/{id}/content",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleGetEntryContent))))

	// POST /api/v1/lexiang/entries/{id}/reupload - 重新上传文件
	mux.Handle("POST /api/v1/lexiang/entries/{id}/reupload",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleReuploadFile))))

	// DELETE /api/v1/lexiang/entries/{id} - 删除知识节点
	mux.Handle("DELETE /api/v1/lexiang/entries/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleDeleteEntry))))

	// ========== 文件上传路由 ==========

	// POST /api/v1/lexiang/upload/sign - 获取上传签名
	mux.Handle("POST /api/v1/lexiang/upload/sign",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleGetUploadSign))))

	// POST /api/v1/lexiang/upload - 上传文件（完整流程）
	mux.Handle("POST /api/v1/lexiang/upload",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleUploadFile))))

	// ========== 附件下载路由 ==========

	// GET /api/v1/lexiang/files/{id} - 获取附件详情
	mux.Handle("GET /api/v1/lexiang/files/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleGetDocFile))))

	// GET /api/v1/lexiang/files/{id}/download - 下载附件
	mux.Handle("GET /api/v1/lexiang/files/{id}/download",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleDownloadDocFile))))

	// ========== 知识反馈路由 ==========

	// GET /api/v1/lexiang/feedbacks - 获取知识反馈列表
	mux.Handle("GET /api/v1/lexiang/feedbacks",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleListFeedbacks))))

	// ========== AI 问答和搜索路由 ==========

	// POST /api/v1/lexiang/ai/qa - AI问答（非流式）
	mux.Handle("POST /api/v1/lexiang/ai/qa",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleAIQA))))

	// POST /api/v1/lexiang/ai/qa/stream - AI问答（流式）
	mux.Handle("POST /api/v1/lexiang/ai/qa/stream",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleAIQAStream))))

	// POST /api/v1/lexiang/ai/search - AI搜索
	mux.Handle("POST /api/v1/lexiang/ai/search",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(lexiangHandler.HandleAISearch))))
}
