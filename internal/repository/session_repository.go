package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// ErrNotFound 表示资源不存在
var ErrNotFound = errors.New("资源不存在")

// SessionRepository 会话数据访问接口
type SessionRepository interface {
	// Create 创建会话
	Create(ctx context.Context, session *model.ChatSession) error

	// GetByID 根据ID获取会话
	GetByID(ctx context.Context, sessionID string) (*model.ChatSession, error)

	// GetByUserID 获取用户的会话列表（支持分页）
	GetByUserID(ctx context.Context, userID string, page, pageSize int, filters *SessionFilters) ([]*model.ChatSession, int64, error)

	// Update 更新会话
	Update(ctx context.Context, session *model.ChatSession) error

	// UpdateFields 更新指定字段
	UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error

	// SoftDelete 软删除会话
	SoftDelete(ctx context.Context, sessionID string) error

	// Search 搜索会话
	Search(ctx context.Context, userID, keyword string, page, pageSize int) ([]*model.ChatSession, int64, error)

	// IncrementMessageCount 增加消息计数
	IncrementMessageCount(ctx context.Context, sessionID string) error

	// UpdateLastMessage 更新最后一条消息
	UpdateLastMessage(ctx context.Context, sessionID, messageID string) error
}

// SessionFilters 会话过滤条件
type SessionFilters struct {
	IsPinned   *bool
	IsArchived *bool
	ModelName  string
}

// sessionRepository 会话数据访问实现
type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 创建会话数据访问实例
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{
		db: db,
	}
}

// Create 创建会话
func (r *sessionRepository) Create(ctx context.Context, session *model.ChatSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取会话
func (r *sessionRepository) GetByID(ctx context.Context, sessionID string) (*model.ChatSession, error) {
	var session model.ChatSession
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", sessionID, false).
		First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("会话不存在")
		}
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}

	return &session, nil
}

// GetByUserID 获取用户的会话列表（支持分页）
func (r *sessionRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int, filters *SessionFilters) ([]*model.ChatSession, int64, error) {
	var sessions []*model.ChatSession
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&model.ChatSession{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	// 应用过滤条件
	if filters != nil {
		if filters.IsPinned != nil {
			query = query.Where("is_pinned = ?", *filters.IsPinned)
		}
		if filters.IsArchived != nil {
			query = query.Where("is_archived = ?", *filters.IsArchived)
		}
		if filters.ModelName != "" {
			query = query.Where("model_name = ?", filters.ModelName)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计会话总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("is_pinned DESC, updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&sessions).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询会话列表失败: %w", err)
	}

	return sessions, total, nil
}

// Update 更新会话
func (r *sessionRepository) Update(ctx context.Context, session *model.ChatSession) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", session.ID, false).
		Save(session)

	if result.Error != nil {
		return fmt.Errorf("更新会话失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或已删除")
	}

	return nil
}

// UpdateFields 更新指定字段
func (r *sessionRepository) UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("id = ? AND is_deleted = ?", sessionID, false).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("更新会话字段失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或已删除")
	}

	return nil
}

// SoftDelete 软删除会话
func (r *sessionRepository) SoftDelete(ctx context.Context, sessionID string) error {
	result := r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("id = ? AND is_deleted = ?", sessionID, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("软删除会话失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或已删除")
	}

	return nil
}

// Search 搜索会话
func (r *sessionRepository) Search(ctx context.Context, userID, keyword string, page, pageSize int) ([]*model.ChatSession, int64, error) {
	var sessions []*model.ChatSession
	var total int64

	// 构建搜索查询
	query := r.db.WithContext(ctx).Model(&model.ChatSession{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Where("title ILIKE ?", "%"+keyword+"%")

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计搜索结果总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("is_pinned DESC, updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&sessions).Error

	if err != nil {
		return nil, 0, fmt.Errorf("搜索会话失败: %w", err)
	}

	return sessions, total, nil
}

// IncrementMessageCount 增加消息计数
func (r *sessionRepository) IncrementMessageCount(ctx context.Context, sessionID string) error {
	result := r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("id = ? AND is_deleted = ?", sessionID, false).
		UpdateColumn("message_count", gorm.Expr("message_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("增加消息计数失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或已删除")
	}

	return nil
}

// UpdateLastMessage 更新最后一条消息
func (r *sessionRepository) UpdateLastMessage(ctx context.Context, sessionID, messageID string) error {
	result := r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("id = ? AND is_deleted = ?", sessionID, false).
		Update("last_message_id", messageID)

	if result.Error != nil {
		return fmt.Errorf("更新最后一条消息失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或已删除")
	}

	return nil
}
