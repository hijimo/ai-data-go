package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/internal/storage"
	"genkit-ai-service/pkg/errors"
)

// TestProviderHandlerValidation 测试处理器层的输入验证
func TestProviderHandlerValidation(t *testing.T) {
	// 创建测试依赖
	store := storage.NewMemoryStore()
	log := logger.Default()
	providerService := service.NewProviderService(store)
	handler := NewProviderHandler(providerService, log)

	tests := []struct {
		name           string
		method         string
		path           string
		pathParams     map[string]string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "有效的提供商ID",
			method:         "GET",
			path:           "/providers/{providerId}",
			pathParams:     map[string]string{"providerId": "tongyi"},
			expectedStatus: http.StatusNotFound, // 因为没有加载数据，所以返回404
			expectedCode:   errors.CodeProviderNotFound,
		},
		{
			name:           "无效的提供商ID - 包含路径遍历",
			method:         "GET",
			path:           "/providers/{providerId}",
			pathParams:     map[string]string{"providerId": "../etc"},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   errors.CodeValidationError,
		},
		{
			name:           "无效的提供商ID - 包含斜杠",
			method:         "GET",
			path:           "/providers/{providerId}",
			pathParams:     map[string]string{"providerId": "provider/test"},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   errors.CodeValidationError,
		},
		{
			name:           "无效的提供商ID - 空字符串",
			method:         "GET",
			path:           "/providers/{providerId}",
			pathParams:     map[string]string{"providerId": ""},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   errors.CodeValidationError,
		},
		{
			name:           "有效的模型ID",
			method:         "GET",
			path:           "/providers/{providerId}/models/{modelId}",
			pathParams:     map[string]string{"providerId": "tongyi", "modelId": "qwen-max"},
			expectedStatus: http.StatusNotFound, // 因为没有加载数据，所以返回404
			expectedCode:   errors.CodeProviderNotFound,
		},
		{
			name:           "无效的模型ID - 包含路径遍历",
			method:         "GET",
			path:           "/providers/{providerId}/models/{modelId}",
			pathParams:     map[string]string{"providerId": "tongyi", "modelId": "../model"},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   errors.CodeValidationError,
		},
		{
			name:           "无效的模型ID - 包含斜杠",
			method:         "GET",
			path:           "/providers/{providerId}/models/{modelId}",
			pathParams:     map[string]string{"providerId": "tongyi", "modelId": "model/test"},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   errors.CodeValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建请求
			req := httptest.NewRequest(tt.method, tt.path, nil)
			
			// 设置路径参数（模拟 Go 1.22+ 的路由）
			req.SetPathValue("providerId", tt.pathParams["providerId"])
			if modelID, ok := tt.pathParams["modelId"]; ok {
				req.SetPathValue("modelId", modelID)
			}

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 调用相应的处理器
			if _, ok := tt.pathParams["modelId"]; ok {
				handler.GetProviderModel(w, req)
			} else {
				handler.GetProviderByID(w, req)
			}

			// 检查状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
			}

			// 解析响应
			var resp model.ResponseData[interface{}]
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			// 检查错误码
			if resp.Code != tt.expectedCode {
				t.Errorf("期望错误码 %d, 得到 %d", tt.expectedCode, resp.Code)
			}

			t.Logf("响应: code=%d, message=%s", resp.Code, resp.Message)
		})
	}
}

// TestProviderHandlerValidationWithData 测试有数据时的验证
func TestProviderHandlerValidationWithData(t *testing.T) {
	// 创建测试依赖
	store := storage.NewMemoryStore()
	log := logger.Default()
	
	// 添加测试数据
	testProvider := model.Provider{
		ID:       "test-provider",
		Provider: "test",
		Label:    map[string]string{"en": "Test Provider"},
	}
	store.SetProviders([]model.Provider{testProvider})
	
	testModel := model.Model{
		Model:     "test-model",
		Label:     map[string]string{"en": "Test Model"},
		ModelType: "llm",
	}
	store.SetModels("test-provider", []model.Model{testModel})
	
	providerService := service.NewProviderService(store)
	handler := NewProviderHandler(providerService, log)

	tests := []struct {
		name           string
		providerId     string
		modelId        string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "有效的提供商ID - 存在的提供商",
			providerId:     "test-provider",
			expectedStatus: http.StatusOK,
			expectedCode:   errors.CodeSuccess,
		},
		{
			name:           "有效的提供商ID - 不存在的提供商",
			providerId:     "non-existent",
			expectedStatus: http.StatusNotFound,
			expectedCode:   errors.CodeProviderNotFound,
		},
		{
			name:           "无效的提供商ID",
			providerId:     "../etc",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   errors.CodeValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建请求
			req := httptest.NewRequest("GET", "/providers/"+tt.providerId, nil)
			req.SetPathValue("providerId", tt.providerId)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 调用处理器
			handler.GetProviderByID(w, req)

			// 检查状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
			}

			// 解析响应
			var resp model.ResponseData[interface{}]
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			// 检查错误码
			if resp.Code != tt.expectedCode {
				t.Errorf("期望错误码 %d, 得到 %d", tt.expectedCode, resp.Code)
			}

			t.Logf("响应: code=%d, message=%s", resp.Code, resp.Message)
		})
	}
}
