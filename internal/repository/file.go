package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ai-knowledge-platform/internal/model"
)

// FileRepository 文件仓库接口
type FileRepository interface {
	// Create 创建文件记录
	Create(ctx context.Context, file *model.File) error
	// GetByID 根据ID获取文件
	GetByID(ctx context.Context, id uuid.UUID) (*model.File, error)
	// GetBySHA256 根据SHA256获取文件（用于去重）
	GetBySHA256(ctx context.Context, projectID uuid.UUID, sha256 string) (*model.File, error)
	// List 获取项目文件列表
	List(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*model.File, int64, error)
	// Update 更新文件信息
	Update(ctx context.Context, file *model.File) error
	// SoftDelete 软删除文件
	SoftDelete(ctx context.Context, id uuid.UUID) error
	// GetByStatus 根据状态获取文件列表
	GetByStatus(ctx context.Context, status int, limit int) ([]*model.File, error)
}

// fileRepository 文件仓库实现
type fileRepository struct {
	db *gorm.DB
}

// NewFileRepository 创建文件仓库
func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{db: db}
}

// Create 创建文件记录
func (r *fileRepository) Create(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("创建文件记录失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取文件
func (r *fileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		Preload("Project").
		First(&file).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文件不存在")
		}
		return nil, fmt.Errorf("获取文件失败: %w", err)
	}
	
	return &file, nil
}

// GetBySHA256 根据SHA256获取文件（用于去重）
func (r *fileRepository) GetBySHA256(ctx context.Context, projectID uuid.UUID, sha256 string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND sha256 = ? AND is_deleted = ?", projectID, sha256, false).
		First(&file).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 文件不存在，返回nil而不是错误
		}
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	
	return &file, nil
}

// List 获取项目文件列表
func (r *fileRepository) List(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*model.File, int64, error) {
	var files []*model.File
	var total int64
	
	// 获取总数
	err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取文件总数失败: %w", err)
	}
	
	// 获取文件列表
	err = r.db.WithContext(ctx).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&files).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取文件列表失败: %w", err)
	}
	
	return files, total, nil
}

// Update 更新文件信息
func (r *fileRepository) Update(ctx context.Context, file *model.File) error {
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", file.ID, false).
		Updates(file).Error
	if err != nil {
		return fmt.Errorf("更新文件失败: %w", err)
	}
	return nil
}

// SoftDelete 软删除文件
func (r *fileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
	if err != nil {
		return fmt.Errorf("软删除文件失败: %w", err)
	}
	return nil
}

// GetByStatus 根据状态获取文件列表
func (r *fileRepository) GetByStatus(ctx context.Context, status int, limit int) ([]*model.File, error) {
	var files []*model.File
	err := r.db.WithContext(ctx).
		Where("status = ? AND is_deleted = ?", status, false).
		Order("created_at ASC").
		Limit(limit).
		Find(&files).Error
	if err != nil {
		return nil, fmt.Errorf("根据状态获取文件失败: %w", err)
	}
	return files, nil
}