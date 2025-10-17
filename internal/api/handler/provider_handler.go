package handler

import (
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"
)

// ProviderHandler 提供商处理器
type ProviderHandler struct {
	providerService service.ProviderService
	logger          logger.Logger
}

// NewProviderHandler 创建新的提供商处理器
func NewProviderHandler(providerService service.ProviderService, logger logger.Logger) *ProviderHandler {
	return &ProviderHandler{
		providerService: providerService,
		logger:          logger,
	}
}

// GetProviders 处理 GET /providers 请求
// @Summary 获取所有提供商列表
// @Description 获取系统中所有可用的模型提供商列表
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseData[[]model.Provider] "成功返回提供商列表"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /providers [get]
func (h *ProviderHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
	// 记录请求日志
	h.logger.Info("收到获取提供商列表请求", map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	})

	// 调用服务层获取所有提供商
	providers := h.providerService.GetAllProviders()

	// 构建成功响应
	resp := response.Success(&providers)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 记录响应日志
	h.logger.Info("获取提供商列表成功", map[string]interface{}{
		"count": len(providers),
	})
}

// GetProviderByID 处理 GET /providers/{providerId} 请求
// @Summary 获取提供商详情
// @Description 根据提供商ID获取详细信息
// @Tags providers
// @Accept json
// @Produce json
// @Param providerId path string true "提供商ID" example(gemini)
// @Success 200 {object} model.ResponseData[model.Provider] "成功返回提供商详情"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 404 {object} model.ErrorResponse "提供商不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /providers/{providerId} [get]
func (h *ProviderHandler) GetProviderByID(w http.ResponseWriter, r *http.Request) {
	// 获取路径参数
	providerID := r.PathValue("providerId")

	// 验证提供商ID
	if err := validator.ValidateProviderID(providerID); err != nil {
		h.handleValidationError(w, err, "提供商ID验证失败")
		return
	}

	// 记录请求日志
	h.logger.Info("收到获取提供商详情请求", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"providerId": providerID,
	})

	// 调用服务层获取提供商
	provider, err := h.providerService.GetProviderByID(providerID)
	if err != nil {
		// 处理错误
		h.handleError(w, err, "获取提供商详情失败", map[string]interface{}{
			"providerId": providerID,
		})
		return
	}

	// 构建成功响应
	resp := response.Success(provider)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 记录响应日志
	h.logger.Info("获取提供商详情成功", map[string]interface{}{
		"providerId": providerID,
	})
}

// GetProviderModels 处理 GET /providers/{providerId}/models 请求
// @Summary 获取提供商的模型列表
// @Description 获取指定提供商的所有可用模型列表
// @Tags providers
// @Accept json
// @Produce json
// @Param providerId path string true "提供商ID" example(gemini)
// @Success 200 {object} model.ResponseData[[]model.Model] "成功返回模型列表"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 404 {object} model.ErrorResponse "提供商不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /providers/{providerId}/models [get]
func (h *ProviderHandler) GetProviderModels(w http.ResponseWriter, r *http.Request) {
	// 获取路径参数
	providerID := r.PathValue("providerId")

	// 验证提供商ID
	if err := validator.ValidateProviderID(providerID); err != nil {
		h.handleValidationError(w, err, "提供商ID验证失败")
		return
	}

	// 记录请求日志
	h.logger.Info("收到获取提供商模型列表请求", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"providerId": providerID,
	})

	// 调用服务层获取模型列表
	models, err := h.providerService.GetProviderModels(providerID)
	if err != nil {
		// 处理错误
		h.handleError(w, err, "获取提供商模型列表失败", map[string]interface{}{
			"providerId": providerID,
		})
		return
	}

	// 构建成功响应
	resp := response.Success(&models)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 记录响应日志
	h.logger.Info("获取提供商模型列表成功", map[string]interface{}{
		"providerId": providerID,
		"count":      len(models),
	})
}

// GetProviderModel 处理 GET /providers/{providerId}/models/{modelId} 请求
// @Summary 获取模型详情
// @Description 获取指定提供商的指定模型的详细信息
// @Tags providers
// @Accept json
// @Produce json
// @Param providerId path string true "提供商ID" example(gemini)
// @Param modelId path string true "模型ID" example(gemini-1.5-flash)
// @Success 200 {object} model.ResponseData[model.Model] "成功返回模型详情"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 404 {object} model.ErrorResponse "提供商或模型不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /providers/{providerId}/models/{modelId} [get]
func (h *ProviderHandler) GetProviderModel(w http.ResponseWriter, r *http.Request) {
	// 获取路径参数
	providerID := r.PathValue("providerId")
	modelID := r.PathValue("modelId")

	// 验证提供商ID
	if err := validator.ValidateProviderID(providerID); err != nil {
		h.handleValidationError(w, err, "提供商ID验证失败")
		return
	}

	// 验证模型ID
	if err := validator.ValidateModelID(modelID); err != nil {
		h.handleValidationError(w, err, "模型ID验证失败")
		return
	}

	// 记录请求日志
	h.logger.Info("收到获取模型详情请求", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"providerId": providerID,
		"modelId":    modelID,
	})

	// 调用服务层获取模型
	m, err := h.providerService.GetProviderModel(providerID, modelID)
	if err != nil {
		// 处理错误
		h.handleError(w, err, "获取模型详情失败", map[string]interface{}{
			"providerId": providerID,
			"modelId":    modelID,
		})
		return
	}

	// 构建成功响应
	resp := response.Success(m)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 记录响应日志
	h.logger.Info("获取模型详情成功", map[string]interface{}{
		"providerId": providerID,
		"modelId":    modelID,
	})
}

// GetModelParameterRules 处理 GET /providers/{providerId}/models/{modelId}/parameter-rules 请求
// @Summary 获取模型参数规则
// @Description 获取指定模型的所有参数配置规则
// @Tags providers
// @Accept json
// @Produce json
// @Param providerId path string true "提供商ID" example(gemini)
// @Param modelId path string true "模型ID" example(gemini-1.5-flash)
// @Success 200 {object} model.ResponseData[[]model.ParameterRule] "成功返回参数规则列表"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 404 {object} model.ErrorResponse "提供商或模型不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /providers/{providerId}/models/{modelId}/parameter-rules [get]
func (h *ProviderHandler) GetModelParameterRules(w http.ResponseWriter, r *http.Request) {
	// 获取路径参数
	providerID := r.PathValue("providerId")
	modelID := r.PathValue("modelId")

	// 验证提供商ID
	if err := validator.ValidateProviderID(providerID); err != nil {
		h.handleValidationError(w, err, "提供商ID验证失败")
		return
	}

	// 验证模型ID
	if err := validator.ValidateModelID(modelID); err != nil {
		h.handleValidationError(w, err, "模型ID验证失败")
		return
	}

	// 记录请求日志
	h.logger.Info("收到获取模型参数规则请求", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"providerId": providerID,
		"modelId":    modelID,
	})

	// 调用服务层获取参数规则
	rules, err := h.providerService.GetModelParameterRules(providerID, modelID)
	if err != nil {
		// 处理错误
		h.handleError(w, err, "获取模型参数规则失败", map[string]interface{}{
			"providerId": providerID,
			"modelId":    modelID,
		})
		return
	}

	// 构建成功响应
	resp := response.Success(&rules)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 记录响应日志
	h.logger.Info("获取模型参数规则成功", map[string]interface{}{
		"providerId": providerID,
		"modelId":    modelID,
		"count":      len(rules),
	})
}

// handleValidationError 处理验证错误
func (h *ProviderHandler) handleValidationError(w http.ResponseWriter, err error, logMessage string) {
	// 记录验证错误日志
	h.logger.Warn(logMessage, map[string]interface{}{
		"error": err.Error(),
	})

	// 构建错误响应
	resp := response.Error[interface{}](errors.CodeValidationError, err.Error())

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码验证错误响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// handleError 统一处理错误响应
func (h *ProviderHandler) handleError(w http.ResponseWriter, err error, logMessage string, logFields map[string]interface{}) {
	// 判断错误类型并设置相应的HTTP状态码
	var httpStatus int
	var resp model.ResponseData[interface{}]

	// 尝试转换为 AppError
	if appErr, ok := err.(*errors.AppError); ok {
		// 根据错误码设置HTTP状态码
		switch appErr.Code {
		case errors.CodeProviderNotFound, errors.CodeModelNotFound:
			httpStatus = http.StatusNotFound
		case errors.CodeBadRequest, errors.CodeValidationError:
			httpStatus = http.StatusBadRequest
		default:
			httpStatus = http.StatusInternalServerError
		}

		// 构建错误响应
		resp = response.Error[interface{}](appErr.Code, appErr.Message)

		// 记录错误日志
		if logFields == nil {
			logFields = make(map[string]interface{})
		}
		logFields["error"] = appErr.Error()
		logFields["errorCode"] = appErr.Code
		h.logger.Error(logMessage, logFields)
	} else {
		// 未知错误，返回内部错误
		httpStatus = http.StatusInternalServerError
		resp = response.Error[interface{}](errors.CodeInternalError, errors.MsgInternalError)

		// 记录错误日志
		if logFields == nil {
			logFields = make(map[string]interface{})
		}
		logFields["error"] = err.Error()
		h.logger.Error(logMessage, logFields)
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码错误响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}
}
