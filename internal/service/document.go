package service

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"ai-knowledge-platform/internal/model"
	"ai-knowledge-platform/internal/processor"
	"ai-knowledge-platform/internal/repository"
)

// DocumentService 文档处理服务接口
type DocumentService interface {
	// ProcessDocument 处理文档
	ProcessDocument(ctx context.Context, req *ProcessDocumentRequest) (*ProcessDocumentResponse, error)
	// GetDocumentVersions 获取文档版本列表
	GetDocumentVersions(ctx context.Context, fileID uuid.UUID) ([]*model.DocumentVersion, error)
	// GetDocumentVersion 获取文档版本详情
	GetDocumentVersion(ctx context.Context, versionID uuid.UUID) (*DocumentVersionDetail, error)
	// GetChunks 获取文档块列表
	GetChunks(ctx context.Context, versionID uuid.UUID) ([]*model.Chunk, error)
}

// ProcessDocumentRequest 处理文档请求
type ProcessDocumentRequest struct {
	FileID      uuid.UUID                  `json:"file_id" validate:"required"`
	ChunkConfig *processor.ChunkConfig     `json:"chunk_config" validate:"required"`
	Options     *DocumentProcessingOptions `json:"options,omitempty"`
}

// DocumentProcessingOptions 文档处理选项
type DocumentProcessingOptions struct {
	ExtractImages bool `json:"extract_images"`  // 是否提取图片
	ExtractTables bool `json:"extract_tables"`  // 是否提取表格
	ExtractLinks  bool `json:"extract_links"`   // 是否提取链接
	CleanContent  bool `json:"clean_content"`   // 是否清理内容
}

// ProcessDocumentResponse 处理文档响应
type ProcessDocumentResponse struct {
	DocumentVersion *model.DocumentVersion `json:"document_version"`
	Document        *processor.Document    `json:"document"`
	ChunkCount      int                    `json:"chunk_count"`
}

// DocumentVersionDetail 文档版本详情
type DocumentVersionDetail struct {
	Version  *model.DocumentVersion `json:"version"`
	Document *processor.Document    `json:"document"`
	Chunks   []*model.Chunk         `json:"chunks"`
}

// documentService 文档处理服务实现
type documentService struct {
	fileRepo            repository.FileRepository
	documentVersionRepo DocumentVersionRepository
	chunkRepo           ChunkRepository
	processorManager    *processor.ProcessorManager
}

// DocumentVersionRepository 文档版本仓库接口
type DocumentVersionRepository interface {
	Create(ctx context.Context, version *model.DocumentVersion) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.DocumentVersion, error)
	GetByFileID(ctx context.Context, fileID uuid.UUID) ([]*model.DocumentVersion, error)
	Update(ctx context.Context, version *model.DocumentVersion) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// ChunkRepository 文档块仓库接口
type ChunkRepository interface {
	Create(ctx context.Context, chunk *model.Chunk) error
	CreateBatch(ctx context.Context, chunks []*model.Chunk) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Chunk, error)
	GetByDocumentVersionID(ctx context.Context, versionID uuid.UUID) ([]*model.Chunk, error)
	Update(ctx context.Context, chunk *model.Chunk) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// NewDocumentService 创建文档处理服务
func NewDocumentService(
	fileRepo repository.FileRepository,
	documentVersionRepo DocumentVersionRepository,
	chunkRepo ChunkRepository,
) DocumentService {
	return &documentService{
		fileRepo:            fileRepo,
		documentVersionRepo: documentVersionRepo,
		chunkRepo:           chunkRepo,
		processorManager:    processor.NewProcessorManager(),
	}
}

// ProcessDocument 处理文档
func (s *documentService) ProcessDocument(ctx context.Context, req *ProcessDocumentRequest) (*ProcessDocumentResponse, error) {
	// 获取文件信息
	file, err := s.fileRepo.GetByID(ctx, req.FileID)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 创建文档版本记录
	version := &model.DocumentVersion{
		FileID: req.FileID,
		Version: s.getNextVersion(ctx, req.FileID),
		ChunkConfig: map[string]any{
			"strategy":          req.ChunkConfig.Strategy,
			"max_size":          req.ChunkConfig.MaxSize,
			"overlap":           req.ChunkConfig.Overlap,
			"separators":        req.ChunkConfig.Separators,
			"preserve_context":  req.ChunkConfig.PreserveContext,
		},
		Status: model.DocumentVersionStatusProcessing,
	}

	if err := s.documentVersionRepo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("创建文档版本失败: %w", err)
	}

	// 异步处理文档
	go s.processDocumentAsync(context.Background(), file, version, req)

	return &ProcessDocumentResponse{
		DocumentVersion: version,
		Document:        nil, // 异步处理，暂时返回nil
		ChunkCount:      0,
	}, nil
}

// processDocumentAsync 异步处理文档
func (s *documentService) processDocumentAsync(ctx context.Context, file *model.File, version *model.DocumentVersion, req *ProcessDocumentRequest) {
	defer func() {
		if r := recover(); r != nil {
			// 处理panic，更新状态为失败
			version.Status = model.DocumentVersionStatusFailed
			s.documentVersionRepo.Update(ctx, version)
		}
	}()

	// 这里应该从OSS下载文件内容
	// 为了演示，我们创建一个模拟的文件内容读取器
	reader, err := s.getFileReader(ctx, file)
	if err != nil {
		version.Status = model.DocumentVersionStatusFailed
		s.documentVersionRepo.Update(ctx, version)
		return
	}
	defer reader.Close()

	// 解析文档
	metadata := &processor.FileMetadata{
		Filename:    file.OriginalName,
		ContentType: file.MimeType,
		Size:        file.Size,
		SHA256:      file.SHA256,
	}

	doc, err := s.processorManager.ProcessDocument(ctx, reader, metadata)
	if err != nil {
		version.Status = model.DocumentVersionStatusFailed
		s.documentVersionRepo.Update(ctx, version)
		return
	}

	// 应用处理选项
	if req.Options != nil {
		doc = s.applyProcessingOptions(doc, req.Options)
	}

	// 分块处理
	chunks, err := s.chunkDocument(doc, req.ChunkConfig)
	if err != nil {
		version.Status = model.DocumentVersionStatusFailed
		s.documentVersionRepo.Update(ctx, version)
		return
	}

	// 保存文档块
	modelChunks := make([]*model.Chunk, len(chunks))
	for i, chunk := range chunks {
		modelChunks[i] = &model.Chunk{
			DocumentVersionID: version.ID,
			Sequence:          i + 1,
			Content:           chunk.Content,
			Metadata: map[string]any{
				"start_offset": chunk.StartOffset,
				"end_offset":   chunk.EndOffset,
				"token_count":  chunk.TokenCount,
				"type":         chunk.Type,
			},
			EmbeddingStatus: model.EmbeddingStatusPending,
		}
	}

	if err := s.chunkRepo.CreateBatch(ctx, modelChunks); err != nil {
		version.Status = model.DocumentVersionStatusFailed
		s.documentVersionRepo.Update(ctx, version)
		return
	}

	// 更新版本状态
	version.Status = model.DocumentVersionStatusCompleted
	version.ChunkCount = len(chunks)
	s.documentVersionRepo.Update(ctx, version)
}

// getFileReader 获取文件读取器（模拟实现）
func (s *documentService) getFileReader(ctx context.Context, file *model.File) (io.ReadCloser, error) {
	// 在实际实现中，这里应该从OSS下载文件
	// 这里返回一个模拟的读取器
	content := fmt.Sprintf("这是文件 %s 的模拟内容。\n\n这是第二段内容。", file.OriginalName)
	return io.NopCloser(strings.NewReader(content)), nil
}

// applyProcessingOptions 应用处理选项
func (s *documentService) applyProcessingOptions(doc *processor.Document, options *DocumentProcessingOptions) *processor.Document {
	if !options.ExtractImages {
		doc.Images = []processor.ImageInfo{}
	}
	if !options.ExtractTables {
		doc.Tables = []processor.TableInfo{}
	}
	if !options.ExtractLinks {
		doc.Links = []processor.LinkInfo{}
	}
	if options.CleanContent {
		doc.Content = s.cleanContent(doc.Content)
	}
	return doc
}

// cleanContent 清理内容
func (s *documentService) cleanContent(content string) string {
	// 移除多余的空白字符
	content = strings.TrimSpace(content)
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	return content
}

// getNextVersion 获取下一个版本号
func (s *documentService) getNextVersion(ctx context.Context, fileID uuid.UUID) int {
	versions, err := s.documentVersionRepo.GetByFileID(ctx, fileID)
	if err != nil || len(versions) == 0 {
		return 1
	}
	
	maxVersion := 0
	for _, v := range versions {
		if v.Version > maxVersion {
			maxVersion = v.Version
		}
	}
	
	return maxVersion + 1
}

// GetDocumentVersions 获取文档版本列表
func (s *documentService) GetDocumentVersions(ctx context.Context, fileID uuid.UUID) ([]*model.DocumentVersion, error) {
	return s.documentVersionRepo.GetByFileID(ctx, fileID)
}

// GetDocumentVersion 获取文档版本详情
func (s *documentService) GetDocumentVersion(ctx context.Context, versionID uuid.UUID) (*DocumentVersionDetail, error) {
	version, err := s.documentVersionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}

	chunks, err := s.chunkRepo.GetByDocumentVersionID(ctx, versionID)
	if err != nil {
		return nil, err
	}

	// 重构文档内容（从块中）
	doc := s.reconstructDocument(version, chunks)

	return &DocumentVersionDetail{
		Version:  version,
		Document: doc,
		Chunks:   chunks,
	}, nil
}

// reconstructDocument 从块重构文档
func (s *documentService) reconstructDocument(version *model.DocumentVersion, chunks []*model.Chunk) *processor.Document {
	var contentBuilder strings.Builder
	
	for _, chunk := range chunks {
		contentBuilder.WriteString(chunk.Content)
		contentBuilder.WriteString("\n")
	}

	return &processor.Document{
		Title:     fmt.Sprintf("文档版本 %d", version.Version),
		Content:   contentBuilder.String(),
		Metadata:  map[string]interface{}{},
		Structure: &processor.DocumentStructure{},
		Images:    []processor.ImageInfo{},
		Tables:    []processor.TableInfo{},
		Links:     []processor.LinkInfo{},
		Language:  "",
		WordCount: len(strings.Fields(contentBuilder.String())),
		PageCount: 1,
	}
}

// GetChunks 获取文档块列表
func (s *documentService) GetChunks(ctx context.Context, versionID uuid.UUID) ([]*model.Chunk, error) {
	return s.chunkRepo.GetByDocumentVersionID(ctx, versionID)
}

// chunkDocument 分块文档
func (s *documentService) chunkDocument(doc *processor.Document, config *processor.ChunkConfig) ([]processor.Chunk, error) {
	chunkerManager := processor.NewChunkerManager()
	return chunkerManager.ChunkDocument(doc, config)
}