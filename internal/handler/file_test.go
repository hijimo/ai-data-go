package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ai-knowledge-platform/internal/model"
	"ai-knowledge-platform/internal/service"
)

// MockFileService 文件服务Mock
type MockFileService struct {
	mock.Mock
}

func (m *MockFileService) UploadFile(ctx context.Context, req *service.UploadFileRequest) (*service.UploadFileResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*service.UploadFileResponse), args.Error(1)
}

func (m *MockFileService) GetFile(ctx context.Context, fileID uuid.UUID) (*model.File, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *MockFileService) ListFiles(ctx context.Context, req *service.ListFilesRequest) (*service.ListFilesResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*service.ListFilesResponse), args.Error(1)
}

func (m *MockFileService) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	args := m.Called(ctx, fileID)
	return args.Error(0)
}

func (m *MockFileService) GetFileURL(ctx context.Context, fileID uuid.UUID, expiry time.Duration) (string, error) {
	args := m.Called(ctx, fileID, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockFileService) ProcessPendingFiles(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 添加用户认证中间件Mock
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	
	return router
}

func createMultipartRequest(filename, content string, extraFields map[string]string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	part.Write([]byte(content))

	// 添加其他字段
	for key, value := range extraFields {
		writer.WriteField(key, value)
	}

	writer.Close()

	req, err := http.NewRequest("POST", "/api/v1/files/upload", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

func TestFileHandler_UploadFile(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockFileService)
		filename       string
		content        string
		extraFields    map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "成功上传文件",
			setupMock: func(mockService *MockFileService) {
				response := &service.UploadFileResponse{
					File: &model.File{
						ID:           uuid.New(),
						Name:         "test.txt",
						OriginalName: "test.txt",
						Size:         12,
					},
					IsDuplicate: false,
				}
				mockService.On("UploadFile", mock.Anything, mock.Anything).Return(response, nil)
			},
			filename: "test.txt",
			content:  "test content",
			extraFields: map[string]string{
				"project_id":  uuid.New().String(),
				"description": "测试文件",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "缺少项目ID",
			setupMock: func(mockService *MockFileService) {
				// 不需要设置Mock，因为会在验证阶段失败
			},
			filename: "test.txt",
			content:  "test content",
			extraFields: map[string]string{
				"description": "测试文件",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "项目ID不能为空",
		},
		{
			name: "无效的项目ID格式",
			setupMock: func(mockService *MockFileService) {
				// 不需要设置Mock
			},
			filename: "test.txt",
			content:  "test content",
			extraFields: map[string]string{
				"project_id": "invalid-uuid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的项目ID格式",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Mock服务
			mockService := &MockFileService{}
			tt.setupMock(mockService)

			// 创建处理器
			handler := NewFileHandler(mockService)

			// 设置路由
			router := setupTestRouter()
			RegisterFileRoutes(router.Group("/api/v1"), handler)

			// 创建请求
			req, err := createMultipartRequest(tt.filename, tt.content, tt.extraFields)
			assert.NoError(t, err)

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["message"], tt.expectedError)
			}

			// 验证Mock调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestFileHandler_GetFile(t *testing.T) {
	mockService := &MockFileService{}
	handler := NewFileHandler(mockService)

	fileID := uuid.New()
	expectedFile := &model.File{
		ID:   fileID,
		Name: "test.txt",
	}

	mockService.On("GetFile", mock.Anything, fileID).Return(expectedFile, nil)

	router := setupTestRouter()
	RegisterFileRoutes(router.Group("/api/v1"), handler)

	req, _ := http.NewRequest("GET", "/api/v1/files/"+fileID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	mockService.AssertExpectations(t)
}

func TestFileHandler_ListFiles(t *testing.T) {
	mockService := &MockFileService{}
	handler := NewFileHandler(mockService)

	projectID := uuid.New()
	expectedResponse := &service.ListFilesResponse{
		Files: []*model.File{
			{ID: uuid.New(), Name: "file1.txt"},
			{ID: uuid.New(), Name: "file2.txt"},
		},
		Total:      2,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	mockService.On("ListFiles", mock.Anything, mock.MatchedBy(func(req *service.ListFilesRequest) bool {
		return req.ProjectID == projectID && req.Page == 1 && req.PageSize == 20
	})).Return(expectedResponse, nil)

	router := setupTestRouter()
	RegisterFileRoutes(router.Group("/api/v1"), handler)

	req, _ := http.NewRequest("GET", "/api/v1/files?project_id="+projectID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	mockService.AssertExpectations(t)
}

func TestFileHandler_DeleteFile(t *testing.T) {
	mockService := &MockFileService{}
	handler := NewFileHandler(mockService)

	fileID := uuid.New()
	mockService.On("DeleteFile", mock.Anything, fileID).Return(nil)

	router := setupTestRouter()
	RegisterFileRoutes(router.Group("/api/v1"), handler)

	req, _ := http.NewRequest("DELETE", "/api/v1/files/"+fileID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	mockService.AssertExpectations(t)
}

func TestFileHandler_GetFileURL(t *testing.T) {
	mockService := &MockFileService{}
	handler := NewFileHandler(mockService)

	fileID := uuid.New()
	expectedURL := "http://example.com/file.txt"

	mockService.On("GetFileURL", mock.Anything, fileID, mock.Anything).Return(expectedURL, nil)

	router := setupTestRouter()
	RegisterFileRoutes(router.Group("/api/v1"), handler)

	req, _ := http.NewRequest("GET", "/api/v1/files/"+fileID.String()+"/url", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, expectedURL, data["url"])

	mockService.AssertExpectations(t)
}