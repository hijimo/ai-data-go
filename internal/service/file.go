package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-knowledge-platform/internal/model"
	"ai-knowledge-platform/internal/repository"
	"ai-knowledge-platform/internal/storage"
)

// FileService 文件服务接口
type FileService interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error)
	// GetFile 获取文件信息
	GetFile(ctx context.Context, fileID uuid.UUID) (*model.File, error)
	// ListFiles 获取文件列表
	ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
	// GetFileURL 获取文件访问URL
	GetFileURL(ctx context.Context, fileID uuid.UUID, expiry time.Duration) (string, error)
	// ProcessPendingFiles 处理待处理文件
	ProcessPendingFiles(ctx context.Context) error
}

// UploadFileRequest 上传文件请求
type UploadFileRequest struct {
	ProjectID    uuid.UUID           `json:"project_id" validate:"required"`
	File         multipart.File      `json:"-"`
	FileHeader   *multipart.FileHeader `json:"-"`
	UploaderID   uuid.UUID           `json:"uploader_id" validate:"required"`
	Description  string              `json:"description,omitempty"`
}

// UploadFileResponse 上传文件响应
type UploadFileResponse struct {
	File      *model.File `json:"file"`
	IsDuplicate bool      `json:"is_duplicate"` // 是否为重复文件
}

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	Page      int       `json:"page" validate:"min=1"`
	PageSize  int       `json:"page_size" validate:"min=1,max=100"`
	Status    *int      `json:"status,omitempty"` // 可选的状态过滤
}

// ListFilesResponse 文件列表响应
type ListFilesResponse struct {
	Files      []*model.File `json:"files"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// fileService 文件服务实现
type fileService struct {
	fileRepo  repository.FileRepository
	ossClient storage.OSSClient
}

// NewFileService 创建文件服务
func NewFileService(fileRepo repository.FileRepository, ossClient storage.OSSClient) FileService {
	return &fileService{
		fileRepo:  fileRepo,
		ossClient: ossClient,
	}
}

// UploadFile 上传文件
func (s *fileService) UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error) {
	// 验证文件格式
	if err := s.validateFileFormat(req.FileHeader.Filename); err != nil {
		return nil, err
	}

	// 上传文件到OSS
	uploadResult, err := s.ossClient.UploadFile(ctx, req.File, req.FileHeader.Filename, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	// 检查是否为重复文件
	existingFile, err := s.fileRepo.GetBySHA256(ctx, req.ProjectID, uploadResult.SHA256)
	if err != nil {
		return nil, fmt.Errorf("检查重复文件失败: %w", err)
	}

	if existingFile != nil {
		// 文件已存在，返回现有文件信息
		return &UploadFileResponse{
			File:        existingFile,
			IsDuplicate: true,
		}, nil
	}

	// 创建文件记录
	file := &model.File{
		ProjectID:    req.ProjectID,
		Name:         s.generateFileName(req.FileHeader.Filename),
		OriginalName: req.FileHeader.Filename,
		MimeType:     uploadResult.ContentType,
		Size:         uploadResult.Size,
		SHA256:       uploadResult.SHA256,
		OSSPath:      uploadResult.OSSPath,
		UploaderID:   req.UploaderID,
		Status:       model.FileStatusCompleted, // 上传完成
		Metadata: map[string]any{
			"description": req.Description,
			"upload_time": time.Now(),
		},
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		// 如果数据库创建失败，尝试删除已上传的文件
		_ = s.ossClient.DeleteFile(ctx, uploadResult.OSSPath)
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	return &UploadFileResponse{
		File:        file,
		IsDuplicate: false,
	}, nil
}

// GetFile 获取文件信息
func (s *fileService) GetFile(ctx context.Context, fileID uuid.UUID) (*model.File, error) {
	return s.fileRepo.GetByID(ctx, fileID)
}

// ListFiles 获取文件列表
func (s *fileService) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize
	files, total, err := s.fileRepo.List(ctx, req.ProjectID, req.PageSize, offset)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &ListFilesResponse{
		Files:      files,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// DeleteFile 删除文件
func (s *fileService) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	// 获取文件信息
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	// 软删除数据库记录
	if err := s.fileRepo.SoftDelete(ctx, fileID); err != nil {
		return err
	}

	// 异步删除OSS文件（可以放到队列中处理）
	go func() {
		ctx := context.Background()
		if err := s.ossClient.DeleteFile(ctx, file.OSSPath); err != nil {
			// 记录日志，但不影响主流程
			// TODO: 添加日志记录
		}
	}()

	return nil
}

// GetFileURL 获取文件访问URL
func (s *fileService) GetFileURL(ctx context.Context, fileID uuid.UUID, expiry time.Duration) (string, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	return s.ossClient.GetFileURL(ctx, file.OSSPath, expiry)
}

// ProcessPendingFiles 处理待处理文件
func (s *fileService) ProcessPendingFiles(ctx context.Context) error {
	// 获取状态为上传中的文件
	files, err := s.fileRepo.GetByStatus(ctx, model.FileStatusUploading, 100)
	if err != nil {
		return err
	}

	for _, file := range files {
		// 检查OSS中文件是否存在
		exists, err := s.ossClient.CheckFileExists(ctx, file.OSSPath)
		if err != nil {
			continue
		}

		if exists {
			// 文件存在，更新状态为已完成
			file.Status = model.FileStatusCompleted
		} else {
			// 文件不存在，更新状态为失败
			file.Status = model.FileStatusFailed
		}

		_ = s.fileRepo.Update(ctx, file)
	}

	return nil
}

// validateFileFormat 验证文件格式
func (s *fileService) validateFileFormat(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	
	// 支持的文件格式
	supportedFormats := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".txt":  true,
		".md":   true,
		".html": true,
		".htm":  true,
		".json": true,
		".xml":  true,
		".csv":  true,
	}

	if !supportedFormats[ext] {
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}

	return nil
}

// generateFileName 生成文件名
func (s *fileService) generateFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)
	
	// 如果文件名过长，截取前100个字符
	if len(name) > 100 {
		name = name[:100]
	}
	
	return name + ext
}