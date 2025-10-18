package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// MessageRepository 消息数据访问接口
type MessageRepository interface {
	// Create 创建消息
	Create(ctx context.Context, message *model.ChatMessage) error

	// GetByID 根据ID获取消息
	GetByID(ctx context.Context, messageID string) (*model.ChatMessage, error)

	// GetBySessionID 获取会话的消息列表（支持分页）
	GetBySessionID(ctx context.Context, sessionID string, page, pageSize int) ([]*model.ChatMessage, int, error)

	// GetLatestMessages 获取最新的N条消息
	GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]*model.ChatMessage, error)

	// GetNextSequence 获取下一个序列号
	GetNextSequence(ctx context.Context, sessionID string) (int, error)

	// CountBySessionID 统计会话消息数量
	CountBySessionID(ctx context.Context, sessionID string) (int, error)

	// GetMessagesAfter 获取指定消息之后的所有消息
	GetMessagesAfter(ctx context.Context, sessionID string, afterMessageID string) ([]*model.ChatMessage, error)
}

// messageRepository 消息数据访问实现
type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 创建消息数据访问实例
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{
		db: db,
	}
}

// Create 创建消息
func (r *messageRepository) Create(ctx context.Context, message *model.ChatMessage) error {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return fmt.Errorf("创建消息失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取消息
func (r *messageRepository) GetByID(ctx context.Context, messageID string) (*model.ChatMessage, error) {
	var message model.ChatMessage
	err := r.db.WithContext(ctx).
		Where("id = ?", messageID).
		First(&message).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("消息不存在")
		}
		return nil, fmt.Errorf("查询消息失败: %w", err)
	}

	return &message, nil
}

// GetBySessionID 获取会话的消息列表（支持分页）
func (r *messageRepository) GetBySessionID(ctx context.Context, sessionID string, page, pageSize int) ([]*model.ChatMessage, int, error) {
	var messages []*model.ChatMessage
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&model.ChatMessage{}).
		Where("session_id = ?", sessionID)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计消息总数失败: %w", err)
	}

	// 分页查询，按序列号正序排列
	offset := (page - 1) * pageSize
	err := query.
		Order("sequence ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&messages).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询消息列表失败: %w", err)
	}

	return messages, int(total), nil
}

// GetLatestMessages 获取最新的N条消息
func (r *messageRepository) GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]*model.ChatMessage, error) {
	var messages []*model.ChatMessage

	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("sequence DESC").
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("查询最新消息失败: %w", err)
	}

	// 反转顺序，使其按序列号正序排列
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetNextSequence 获取下一个序列号
func (r *messageRepository) GetNextSequence(ctx context.Context, sessionID string) (int, error) {
	var maxSequence int

	err := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("session_id = ?", sessionID).
		Select("COALESCE(MAX(sequence), 0)").
		Scan(&maxSequence).Error

	if err != nil {
		return 0, fmt.Errorf("获取下一个序列号失败: %w", err)
	}

	return maxSequence + 1, nil
}

// CountBySessionID 统计会话消息数量
func (r *messageRepository) CountBySessionID(ctx context.Context, sessionID string) (int, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("session_id = ?", sessionID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("统计消息数量失败: %w", err)
	}

	return int(count), nil
}

// GetMessagesAfter 获取指定消息之后的所有消息
func (r *messageRepository) GetMessagesAfter(ctx context.Context, sessionID string, afterMessageID string) ([]*model.ChatMessage, error) {
	// 首先获取指定消息的序列号
	var afterMessage model.ChatMessage
	err := r.db.WithContext(ctx).
		Where("id = ? AND session_id = ?", afterMessageID, sessionID).
		First(&afterMessage).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("指定的消息不存在")
		}
		return nil, fmt.Errorf("查询指定消息失败: %w", err)
	}

	// 查询序列号大于指定消息的所有消息
	var messages []*model.ChatMessage
	err = r.db.WithContext(ctx).
		Where("session_id = ? AND sequence > ?", sessionID, afterMessage.Sequence).
		Order("sequence ASC").
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("查询指定消息之后的消息失败: %w", err)
	}

	return messages, nil
}
