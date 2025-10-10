package service

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ai-knowledge-platform/internal/model"
	"ai-knowledge-platform/internal/storage"
)

// MockFileRepository 文件仓库Mock
type MockFileRepository struct {
	mock.Mock
}

func (m *MockFileRepository) Create(ctx context.Context, file *model.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *MockFileRepository) GetBySHA256(ctx context.Context, projectID uuid.UUID, sha256 string) (*model.File, error) {
	args := m.Called(ctx, projectID, sha256)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *MockFileRepository) List(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*model.File, int64, error) {
	args := m.Called(ctx, projectID, limit, offset)
	return args.Get(0).([]*model.File), args.Get(1).(int64), args.Error(2)
}

func (m *MockFileRepository) Update(ctx context.Context, file *model.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockFileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) GetByStatus(ctx context.Context, status int, limit int) ([]*model.File, error) {
	args := m.Called(ctx, status, limit)
	return args.Get(0).([]*model.File), args.Error(1)
}

// createTestFile 创建测试文件
func createTestFile(content string, filename string) (multipart.File, *multipart.FileHeader) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(1024)
	
	file := form.File["file"][0]
	f, _ := file.Open()
	
	return f, file
}

func TestFileService_UploadFile(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockFileRepository, storage.OSSClient)
		request        *UploadFileRequest
		expectedError  string
		expectedResult *UploadFileResponse
	}{
		{
			name: "成功上传新文件",
			setupMocks: func(repo *MockFileRepository, oss storage.OSSClient) {
				// Mock检查重复文件 - 返回nil表示不存在
				repo.On("GetBySHA256", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				// Mock创建文件记录
				repo.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			request: &UploadFileRequest{
				ProjectID:   uuid.New(),
				UploaderID:  uuid.New(),
				Description: "测试文件",
			},
			expectedError: "",
		},
		{
			name: "上传重复文件",
			setupMocks: func(repo *MockFileRepository, oss storage.OSSClient) {
				existingFile := &model.File{
					ID:           uuid.New(),
					Name:         "existing.txt",
					OriginalName: "existing.txt",
					SHA256:       "test-hash",
				}
				// Mock检查重复文件 - 返回现有文件
				repo.On("GetBySHA256", mock.Anything, mock.Anything, mock.Anything).Return(existingFile, nil)
			},
			request: &UploadFileRequest{
				ProjectID:  uuid.New(),
				UploaderID: uuid.New(),
			},
			expectedError: "",
			expectedResult: &UploadFileResponse{
				IsDuplicate: true,
			},
		},
		{
			name: "不支持的文件格式",
			setupMocks: func(repo *MockFileRepository, oss storage.OSSClient) {
				// 不需要设置Mock，因为会在验证阶段失败
			},
			request: &UploadFileRequest{
				ProjectID:  uuid.New(),
				UploaderID: uuid.New(),
			},
			expectedError: "不支持的文件格式",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Mock
			mockRepo := &MockFileRepository{}
			mockOSS := storage.NewMockOSSClient()

			// 设置Mock
			tt.setupMocks(mockRepo, mockOSS)

			// 创建服务
			service := NewFileService(mockRepo, mockOSS)

			// 创建测试文件
			var file multipart.File
			var fileHeader *multipart.FileHeader
			
			if tt.expectedError == "不支持的文件格式" {
				file, fileHeader = createTestFile("test content", "test.xyz")
			} else {
				file, fileHeader = createTestFile("test content", "test.txt")
			}
			
			tt.request.File = file
			tt.request.FileHeader = fileHeader

			// 执行测试
			result, err := service.UploadFile(context.Background(), tt.request)

			// 验证结果
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				
				if tt.expectedResult != nil {
					assert.Equal(t, tt.expectedResult.IsDuplicate, result.IsDuplicate)
				}
			}

			// 验证Mock调用
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestFileService_GetFile(t *testing.T) {
	mockRepo := &MockFileRepository{}
	mockOSS := storage.NewMockOSSClient()
	service := NewFileService(mockRepo, mockOSS)

	fileID := uuid.New()
	expectedFile := &model.File{
		ID:   fileID,
		Name: "test.txt",
	}

	mockRepo.On("GetByID", mock.Anything, fileID).Return(expectedFile, nil)

	result, err := service.GetFile(context.Background(), fileID)

	assert.NoError(t, err)
	assert.Equal(t, expectedFile, result)
	mockRepo.AssertExpectations(t)
}

func TestFileService_ListFiles(t *testing.T) {
	mockRepo := &MockFileRepository{}
	mockOSS := storage.NewMockOSSClient()
	service := NewFileService(mockRepo, mockOSS)

	projectID := uuid.New()
	expectedFiles := []*model.File{
		{ID: uuid.New(), Name: "file1.txt"},
		{ID: uuid.New(), Name: "file2.txt"},
	}
	expectedTotal := int64(2)

	mockRepo.On("List", mock.Anything, projectID, 20, 0).Return(expectedFiles, expectedTotal, nil)

	req := &ListFilesRequest{
		ProjectID: projectID,
		Page:      1,
		PageSize:  20,
	}

	result, err := service.ListFiles(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expectedFiles, result.Files)
	assert.Equal(t, expectedTotal, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)
	assert.Equal(t, 1, result.TotalPages)
	mockRepo.AssertExpectations(t)
}

func TestFileService_DeleteFile(t *testing.T) {
	mockRepo := &MockFileRepository{}
	mockOSS := storage.NewMockOSSClient()
	service := NewFileService(mockRepo, mockOSS)

	fileID := uuid.New()
	file := &model.File{
		ID:      fileID,
		OSSPath: "test/path/file.txt",
	}

	mockRepo.On("GetByID", mock.Anything, fileID).Return(file, nil)
	mockRepo.On("SoftDelete", mock.Anything, fileID).Return(nil)

	err := service.DeleteFile(context.Background(), fileID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_GetFileURL(t *testing.T) {
	mockRepo := &MockFileRepository{}
	mockOSS := storage.NewMockOSSClient()
	service := NewFileService(mockRepo, mockOSS)

	fileID := uuid.New()
	file := &model.File{
		ID:      fileID,
		OSSPath: "test/path/file.txt",
	}
	expectedURL := "http://mock-oss.com/test/path/file.txt"

	mockRepo.On("GetByID", mock.Anything, fileID).Return(file, nil)

	result, err := service.GetFileURL(context.Background(), fileID, time.Hour)

	assert.NoError(t, err)
	assert.Equal(t, expectedURL, result)
	mockRepo.AssertExpectations(t)
}

func TestFileService_ProcessPendingFiles(t *testing.T) {
	mockRepo := &MockFileRepository{}
	mockOSS := storage.NewMockOSSClient()
	service := NewFileService(mockRepo, mockOSS)

	pendingFiles := []*model.File{
		{
			ID:      uuid.New(),
			OSSPath: "existing/file.txt",
			Status:  model.FileStatusUploading,
		},
		{
			ID:      uuid.New(),
			OSSPath: "missing/file.txt",
			Status:  model.FileStatusUploading,
		},
	}

	mockRepo.On("GetByStatus", mock.Anything, model.FileStatusUploading, 100).Return(pendingFiles, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(f *model.File) bool {
		return f.ID == pendingFiles[0].ID && f.Status == model.FileStatusCompleted
	})).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(f *model.File) bool {
		return f.ID == pendingFiles[1].ID && f.Status == model.FileStatusFailed
	})).Return(nil)

	err := service.ProcessPendingFiles(context.Background())

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestValidateFileFormat(t *testing.T) {
	service := &fileService{}

	tests := []struct {
		filename    string
		expectError bool
	}{
		{"test.pdf", false},
		{"test.doc", false},
		{"test.docx", false},
		{"test.txt", false},
		{"test.md", false},
		{"test.html", false},
		{"test.json", false},
		{"test.xyz", true},
		{"test.exe", true},
		{"test", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			err := service.validateFileFormat(tt.filename)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}