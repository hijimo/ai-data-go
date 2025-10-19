package repository

import (
	"context"
	"errors"
	"time"

	"genkit-ai-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditFilter 审计日志过滤条件
type AuditFilter struct {
	TenantID  *uuid.UUID // 租户 ID
	UserID    *uuid.UUID // 用户 ID
	Event     string     // 事件类型
	StartTime *time.Time // 开始时间
	EndTime   *time.Time // 结束时间
}

// AuditRepository 审计日志数据访问接口
type AuditRepository interface {
	// Create 创建审计日志
	Create(ctx context.Context, audit *model.AuthAudit) error

	// List 列出审计日志（支持过滤和分页）
	List(ctx context.Context, filter AuditFilter, page, pageSize int) ([]*model.AuthAudit, int64, error)
}

// auditRepository 审计日志数据访问实现
type auditRepository struct {
	db *gorm.DB
}

// NewAuditRepository 创建审计日志数据访问实例
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{
		db: db,
	}
}

// Create 创建审计日志
func (r *auditRepository) Create(ctx context.Context, audit *model.AuthAudit) error {
	if audit == nil {
		return errors.New("audit cannot be nil")
	}

	// 验证必需字段
	if audit.Event == "" {
		return errors.New("event is required")
	}

	return r.db.WithContext(ctx).Create(audit).Error
}

// List 列出审计日志（支持多条件过滤和分页）
func (r *auditRepository) List(ctx context.Context, filter AuditFilter, page, pageSize int) ([]*model.AuthAudit, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var audits []*model.AuthAudit
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建查询
	query := r.db.WithContext(ctx).Model(&model.AuthAudit{})

	// 应用过滤条件
	query = r.applyFilters(query, filter)

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	if err := query.
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&audits).Error; err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

// applyFilters 应用过滤条件
func (r *auditRepository) applyFilters(query *gorm.DB, filter AuditFilter) *gorm.DB {
	// 按租户 ID 过滤
	if filter.TenantID != nil && *filter.TenantID != uuid.Nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}

	// 按用户 ID 过滤
	if filter.UserID != nil && *filter.UserID != uuid.Nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	// 按事件类型过滤
	if filter.Event != "" {
		query = query.Where("event = ?", filter.Event)
	}

	// 按开始时间过滤
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}

	// 按结束时间过滤
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	return query
}
