package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterProviderRoutes 注册提供商相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
func RegisterProviderRoutes(mux *http.ServeMux, handler *handler.ProviderHandler) {
	// GET /api/v1/providers - 获取所有提供商列表
	mux.HandleFunc("GET /api/v1/providers", handler.GetProviders)

	// GET /api/v1/providers/{providerId} - 根据ID获取提供商详情
	mux.HandleFunc("GET /api/v1/providers/{providerId}", handler.GetProviderByID)

	// GET /api/v1/providers/{providerId}/models - 获取提供商的所有模型列表
	mux.HandleFunc("GET /api/v1/providers/{providerId}/models", handler.GetProviderModels)

	// GET /api/v1/providers/{providerId}/models/{modelId} - 获取提供商的指定模型详情
	mux.HandleFunc("GET /api/v1/providers/{providerId}/models/{modelId}", handler.GetProviderModel)

	// GET /api/v1/providers/{providerId}/models/{modelId}/parameter-rules - 获取模型的参数规则
	mux.HandleFunc("GET /api/v1/providers/{providerId}/models/{modelId}/parameter-rules", handler.GetModelParameterRules)
}
