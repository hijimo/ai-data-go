package auth

import (
	"context"
	"errors"
	"fmt"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TenantService 租户服务接口
type TenantService interface {
	// Create 创建租户
	Create(ctx context.Context, req CreateTenantRequest) (*model.Tenant, error)

	// Get 获取租户
	Get(ctx context.Context, id string) (*model.Tenant, error)

	// Update 更新租户
	Update(ctx context.Context, id string, req UpdateTenantRequest) (*model.Tenant, error)

	// Delete 删除租户
	Delete(ctx context.Context, id string) error

	// List 列出租户
	List(ctx context.Context, page, pageSize int) ([]*model.Tenant, int64, error)

	// GetByDomain 根据域名获取租户
	GetByDomain(ctx context.Context, domain string) (*model.Tenant, error)
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	// 租户名称
	Name string `json:"name" validate:"required,min=1,max=255"`
	// 租户域名
	Domain string `json:"domain" validate:"omitempty,max=255"`
	// 租户元数据
	Metadata map[string]interface{} `json:"metadata"`
	// 创建者用户ID
	CreatedBy *string `json:"createdBy"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	// 租户名称
	Name *string `json:"name" validate:"omitempty,min=1,max=255"`
	// 租户域名
	Domain *string `json:"domain" validate:"omitempty,max=255"`
	// 租户元数据
	Metadata map[string]interface{} `json:"metadata"`
	// 租户状态
	Status *bool `json:"status"`
}

// tenantService 租户服务实现
type tenantService struct {
	tenantRepo repository.TenantRepository
}

// NewTenantService 创建租户服务实例
func NewTenantService(tenantRepo repository.TenantRepository) TenantService {
	return &tenantService{
		tenantRepo: tenantRepo,
	}
}

// Create 创建租户
func (s *tenantService) Create(ctx context.Context, req CreateTenantRequest) (*model.Tenant, error) {
	// 验证请求
	if req.Name == "" {
		return nil, errors.New("租户名称不能为空")
	}

	// 如果提供了域名，检查域名是否已存在
	if req.Domain != "" {
		existing, err := s.tenantRepo.GetByDomain(ctx, req.Domain)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("域名 %s 已被使用", req.Domain)
		}
	}

	// 创建租户对象
	tenant := &model.Tenant{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Domain:    req.Domain,
		Status:    true, // 默认启用
		CreatedBy: req.CreatedBy,
		IsDeleted: false,
	}

	// 设置元数据
	if req.Metadata != nil {
		metadata, err := datatypes.NewJSONType(req.Metadata).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("元数据格式错误: %w", err)
		}
		tenant.Metadata = metadata
	}

	// 保存到数据库
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("创建租户失败: %w", err)
	}

	return tenant, nil
}

// Get 获取租户
func (s *tenantService) Get(ctx context.Context, id string) (*model.Tenant, error) {
	// 验证ID格式
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("无效的租户ID格式")
	}

	// 从数据库获取
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取租户失败: %w", err)
	}

	// 验证租户状态
	if !tenant.Status {
		return nil, errors.New("租户已被禁用")
	}

	return tenant, nil
}

// Update 更新租户
func (s *tenantService) Update(ctx context.Context, id string, req UpdateTenantRequest) (*model.Tenant, error) {
	// 验证ID格式
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("无效的租户ID格式")
	}

	// 获取现有租户
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("租户不存在: %w", err)
	}

	// 更新字段
	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.New("租户名称不能为空")
		}
		tenant.Name = *req.Name
	}

	if req.Domain != nil {
		// 如果域名发生变化，检查新域名是否已被使用
		if *req.Domain != tenant.Domain && *req.Domain != "" {
			existing, err := s.tenantRepo.GetByDomain(ctx, *req.Domain)
			if err == nil && existing != nil && existing.ID != tenant.ID {
				return nil, fmt.Errorf("域名 %s 已被使用", *req.Domain)
			}
		}
		tenant.Domain = *req.Domain
	}

	if req.Status != nil {
		tenant.Status = *req.Status
	}

	if req.Metadata != nil {
		metadata, err := datatypes.NewJSONType(req.Metadata).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("元数据格式错误: %w", err)
		}
		tenant.Metadata = metadata
	}

	// 保存更新
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("更新租户失败: %w", err)
	}

	return tenant, nil
}

// Delete 删除租户
func (s *tenantService) Delete(ctx context.Context, id string) error {
	// 验证ID格式
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("无效的租户ID格式")
	}

	// 验证租户是否存在
	_, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 执行软删除
	if err := s.tenantRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除租户失败: %w", err)
	}

	return nil
}

// List 列出租户
func (s *tenantService) List(ctx context.Context, page, pageSize int) ([]*model.Tenant, int64, error) {
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

	// 从数据库获取列表
	tenants, total, err := s.tenantRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取租户列表失败: %w", err)
	}

	return tenants, total, nil
}

// GetByDomain 根据域名获取租户
func (s *tenantService) GetByDomain(ctx context.Context, domain string) (*model.Tenant, error) {
	if domain == "" {
		return nil, errors.New("域名不能为空")
	}

	// 从数据库获取
	tenant, err := s.tenantRepo.GetByDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("获取租户失败: %w", err)
	}

	// 验证租户状态
	if !tenant.Status {
		return nil, errors.New("租户已被禁用")
	}

	return tenant, nil
}
