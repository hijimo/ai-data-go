package auth

import (
	"context"
	"errors"
	"fmt"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/crypto"

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

	// List 列出租户（支持按类型过滤）
	List(ctx context.Context, page, pageSize int, tenantType ...string) ([]*model.Tenant, int64, error)

	// ListWithFilter 列出租户（支持过滤条件）
	ListWithFilter(ctx context.Context, page, pageSize int, filter TenantListFilter) ([]*model.Tenant, int64, error)

	// GetByDomain 根据域名获取租户
	GetByDomain(ctx context.Context, domain string) (*model.Tenant, error)

	// CreateWithAdmin 创建租户并自动生成管理员账户
	CreateWithAdmin(ctx context.Context, req CreateTenantWithAdminRequest) (*CreateTenantWithAdminResponse, error)

	// GetByType 根据类型获取租户列表
	GetByType(ctx context.Context, tenantType string) ([]*model.Tenant, error)

	// EnableTenant 启用租户
	EnableTenant(ctx context.Context, id string) error

	// DisableTenant 禁用租户
	DisableTenant(ctx context.Context, id string) error
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	// 租户名称
	Name string `json:"name" validate:"required,min=1,max=255"`
	// 租户域名
	Domain string `json:"domain" validate:"omitempty,max=255"`
	// 租户类型（system 或 tenant）
	Type string `json:"type" validate:"omitempty,oneof=system tenant"`
	// 租户元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// CreateTenantWithAdminRequest 创建租户并自动生成管理员请求
type CreateTenantWithAdminRequest struct {
	// 租户名称
	TenantName string `json:"tenantName" validate:"required,min=1,max=255"`
	// 租户域名
	TenantDomain string `json:"tenantDomain" validate:"required,max=255"`
	// 租户元数据
	TenantMetadata map[string]interface{} `json:"tenantMetadata"`
	// 管理员邮箱（可选，默认为 admin@{domain}）
	AdminEmail string `json:"adminEmail" validate:"omitempty,email"`
	// 管理员显示名称（可选）
	AdminDisplayName string `json:"adminDisplayName" validate:"omitempty,max=255"`
}

// CreateTenantWithAdminResponse 创建租户并自动生成管理员响应
type CreateTenantWithAdminResponse struct {
	// 租户信息
	Tenant *model.Tenant `json:"tenant"`
	// 管理员用户信息
	AdminUser *model.User `json:"adminUser"`
	// 管理员初始密码
	AdminPassword string `json:"adminPassword"`
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

// TenantListFilter 租户列表过滤条件
type TenantListFilter struct {
	// 租户名称（模糊搜索）
	Name string
	// 租户状态（true=启用，false=禁用）
	Status *bool
}

// tenantService 租户服务实现
type tenantService struct {
	tenantRepo repository.TenantRepository
	userRepo   repository.UserRepository
	auditRepo  repository.AuditRepository
}

// NewTenantService 创建租户服务实例
func NewTenantService(tenantRepo repository.TenantRepository, userRepo repository.UserRepository, auditRepo repository.AuditRepository) TenantService {
	return &tenantService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
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

	// 设置租户类型，默认为业务租户
	tenantType := req.Type
	if tenantType == "" {
		tenantType = model.TenantTypeBusiness
	}

	// 如果是平台租户，验证唯一性
	if tenantType == model.TenantTypeSystem {
		systemTenants, err := s.tenantRepo.GetByType(ctx, model.TenantTypeSystem)
		if err == nil && len(systemTenants) > 0 {
			return nil, errors.New("平台租户已存在，不能创建多个")
		}
	}

	// 从 Context 获取创建者信息
	createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)

	tenant := &model.Tenant{
		ID:            uuid.New(),
		Name:          req.Name,
		Domain:        req.Domain,
		Type:          tenantType,
		Status:        true, // 默认启用
		CreatedBy:     createdByUUID,
		CreatedByName: createdByName,
		IsDeleted:     false,
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

	// 验证租户访问权限
	if !s.canAccessTenant(ctx, tenant.ID.String()) {
		return nil, errors.New("权限不足：无法访问其他租户的数据")
	}

	// 移除租户状态验证，允许查询禁用的租户
	// 这样管理员可以查看禁用租户的详细信息

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

	// 验证租户访问权限
	if !s.canAccessTenant(ctx, tenant.ID.String()) {
		return nil, errors.New("权限不足：无法访问其他租户的数据")
	}

	// 验证字段级权限
	if err := s.validateUpdateFields(ctx, req); err != nil {
		return nil, err
	}

	// 防止修改平台租户的类型
	// 注意：UpdateTenantRequest 中没有 Type 字段，所以类型不会被修改
	// 这里只是作为额外的保护措施

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
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 检查是否为平台租户，如果是则拒绝删除
	if tenant.Type == model.TenantTypeSystem {
		return errors.New("不允许删除平台租户")
	}

	// 执行软删除
	if err := s.tenantRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除租户失败: %w", err)
	}

	// 记录审计日志
	auditLog := &model.AuthAudit{
		TenantID: &tenant.ID,
		Event:    model.AuditEventTenantDeleted,
		Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"tenantId":"%s","tenantName":"%s"}`, tenant.ID, tenant.Name))),
	}
	_ = s.auditRepo.Create(ctx, auditLog)

	return nil
}

// List 列出租户（支持按类型过滤）
func (s *tenantService) List(ctx context.Context, page, pageSize int, tenantType ...string) ([]*model.Tenant, int64, error) {
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

	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, 0, errors.New("未找到身份认证信息")
	}

	// 检查用户角色
	isSystemAdmin := hasSystemAdminRole(claims)

	// 如果是租户管理员，只返回当前用户所属的租户
	if !isSystemAdmin {
		// 获取当前用户的租户
		tenant, err := s.tenantRepo.GetByID(ctx, claims.TenantID)
		if err != nil {
			return nil, 0, fmt.Errorf("获取租户失败: %w", err)
		}
		// 返回单个租户作为列表
		return []*model.Tenant{tenant}, 1, nil
	}

	// 平台管理员：返回所有租户或按类型过滤的租户
	// 如果指定了租户类型，使用类型过滤
	if len(tenantType) > 0 && tenantType[0] != "" {
		// 验证租户类型
		if tenantType[0] != model.TenantTypeSystem && tenantType[0] != model.TenantTypeBusiness {
			return nil, 0, fmt.Errorf("无效的租户类型: %s", tenantType[0])
		}

		// 从数据库获取指定类型的列表
		tenants, total, err := s.tenantRepo.ListByType(ctx, tenantType[0], page, pageSize)
		if err != nil {
			return nil, 0, fmt.Errorf("获取租户列表失败: %w", err)
		}
		return tenants, total, nil
	}

	// 从数据库获取所有租户列表
	tenants, total, err := s.tenantRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取租户列表失败: %w", err)
	}

	return tenants, total, nil
}

// ListWithFilter 列出租户（支持过滤条件）
func (s *tenantService) ListWithFilter(ctx context.Context, page, pageSize int, filter TenantListFilter) ([]*model.Tenant, int64, error) {
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

	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, 0, errors.New("未找到身份认证信息")
	}

	// 检查用户角色
	isSystemAdmin := hasSystemAdminRole(claims)

	// 如果是租户管理员，只返回当前用户所属的租户（忽略过滤条件）
	if !isSystemAdmin {
		// 获取当前用户的租户
		tenant, err := s.tenantRepo.GetByID(ctx, claims.TenantID)
		if err != nil {
			return nil, 0, fmt.Errorf("获取租户失败: %w", err)
		}
		// 返回单个租户作为列表
		return []*model.Tenant{tenant}, 1, nil
	}

	// 平台管理员：使用过滤条件查询租户列表
	tenants, total, err := s.tenantRepo.ListWithFilter(ctx, page, pageSize, filter.Name, filter.Status)
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

	// 移除租户状态验证，允许查询禁用的租户
	// 这样管理员可以通过域名查看禁用租户的详细信息

	return tenant, nil
}

// CreateWithAdmin 创建租户并自动生成管理员账户
func (s *tenantService) CreateWithAdmin(ctx context.Context, req CreateTenantWithAdminRequest) (*CreateTenantWithAdminResponse, error) {
	// 1. 验证请求参数
	if req.TenantName == "" {
		return nil, errors.New("租户名称不能为空")
	}
	if req.TenantDomain == "" {
		return nil, errors.New("租户域名不能为空")
	}

	// 2. 检查域名是否已存在
	existing, err := s.tenantRepo.GetByDomain(ctx, req.TenantDomain)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("域名 %s 已被使用", req.TenantDomain)
	}

	// 3. 从 Context 获取创建者信息
	createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)

	// 4. 创建业务租户（type = "tenant"）
	tenant := &model.Tenant{
		ID:            uuid.New(),
		Name:          req.TenantName,
		Domain:        req.TenantDomain,
		Type:          model.TenantTypeBusiness, // 业务租户
		Status:        true,                     // 默认启用
		CreatedBy:     createdByUUID,
		CreatedByName: createdByName,
		IsDeleted:     false,
	}

	// 设置元数据
	if req.TenantMetadata != nil {
		metadata, err := datatypes.NewJSONType(req.TenantMetadata).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("元数据格式错误: %w", err)
		}
		tenant.Metadata = metadata
	}

	// 保存租户到数据库
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("创建租户失败: %w", err)
	}

	// 5. 生成租户管理员邮箱
	adminEmail := req.AdminEmail
	if adminEmail == "" {
		adminEmail = fmt.Sprintf("admin@%s", req.TenantDomain)
	}

	// 6. 生成 16 位随机强密码
	adminPassword, err := crypto.GenerateSecurePassword(16)
	if err != nil {
		// 如果生成失败，尝试回滚租户创建
		_ = s.tenantRepo.Delete(ctx, tenant.ID.String())
		return nil, fmt.Errorf("生成管理员密码失败: %w", err)
	}

	// 7. 对密码进行哈希
	passwordHash, err := crypto.HashPassword(adminPassword)
	if err != nil {
		// 回滚租户创建
		_ = s.tenantRepo.Delete(ctx, tenant.ID.String())
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 8. 创建租户管理员用户
	adminDisplayName := req.AdminDisplayName
	if adminDisplayName == "" {
		adminDisplayName = fmt.Sprintf("%s Admin", req.TenantName)
	}

	adminUser := &model.User{
		ID:            uuid.New(),
		TenantID:      tenant.ID,
		Email:         adminEmail,
		PasswordHash:  passwordHash,
		DisplayName:   adminDisplayName,
		IsActive:      true,
		IsAdmin:       true,
		CreatedBy:     createdByUUID,
		CreatedByName: createdByName,
		IsDeleted:     false,
	}

	// 设置角色为 tenant_admin
	rolesJSON, err := datatypes.NewJSONType([]string{model.RoleTenantAdmin}).MarshalJSON()
	if err != nil {
		// 回滚租户创建
		_ = s.tenantRepo.Delete(ctx, tenant.ID.String())
		return nil, fmt.Errorf("角色数据格式错误: %w", err)
	}
	adminUser.Roles = rolesJSON

	// 保存管理员用户到数据库
	if err := s.userRepo.Create(ctx, adminUser); err != nil {
		// 回滚租户创建
		_ = s.tenantRepo.Delete(ctx, tenant.ID.String())
		return nil, fmt.Errorf("创建管理员用户失败: %w", err)
	}

	// 9. 记录审计日志
	auditLog := &model.AuthAudit{
		TenantID: &tenant.ID,
		Event:    model.AuditEventTenantCreated,
		Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"tenantId":"%s","tenantName":"%s","tenantDomain":"%s"}`, tenant.ID, tenant.Name, tenant.Domain))),
	}
	// 尝试记录审计日志，但不因为日志失败而影响主流程
	_ = s.auditRepo.Create(ctx, auditLog)

	// 10. 返回租户信息和管理员初始密码
	return &CreateTenantWithAdminResponse{
		Tenant:        tenant,
		AdminUser:     adminUser,
		AdminPassword: adminPassword, // 返回明文密码供平台管理员传递给租户管理员
	}, nil
}

// GetByType 根据类型获取租户列表
func (s *tenantService) GetByType(ctx context.Context, tenantType string) ([]*model.Tenant, error) {
	// 验证租户类型
	if tenantType != model.TenantTypeSystem && tenantType != model.TenantTypeBusiness {
		return nil, fmt.Errorf("无效的租户类型: %s", tenantType)
	}

	// 从数据库获取
	tenants, err := s.tenantRepo.GetByType(ctx, tenantType)
	if err != nil {
		return nil, fmt.Errorf("获取租户列表失败: %w", err)
	}

	return tenants, nil
}

// EnableTenant 启用租户
func (s *tenantService) EnableTenant(ctx context.Context, id string) error {
	// 验证ID格式
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("无效的租户ID格式")
	}

	// 获取现有租户
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 如果已经是启用状态，直接返回
	if tenant.Status {
		return nil
	}

	// 更新状态为启用
	tenant.Status = true
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("启用租户失败: %w", err)
	}

	// 记录审计日志
	auditLog := &model.AuthAudit{
		TenantID: &tenant.ID,
		Event:    model.AuditEventTenantEnabled,
		Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"tenantId":"%s","tenantName":"%s"}`, tenant.ID, tenant.Name))),
	}
	_ = s.auditRepo.Create(ctx, auditLog)

	return nil
}

// DisableTenant 禁用租户
func (s *tenantService) DisableTenant(ctx context.Context, id string) error {
	// 验证ID格式
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("无效的租户ID格式")
	}

	// 获取现有租户
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 如果已经是禁用状态，直接返回
	if !tenant.Status {
		return nil
	}

	// 更新状态为禁用
	tenant.Status = false
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("禁用租户失败: %w", err)
	}

	// 记录审计日志
	auditLog := &model.AuthAudit{
		TenantID: &tenant.ID,
		Event:    model.AuditEventTenantDisabled,
		Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"tenantId":"%s","tenantName":"%s"}`, tenant.ID, tenant.Name))),
	}
	_ = s.auditRepo.Create(ctx, auditLog)

	return nil
}

// canAccessTenant 验证用户是否有权访问指定租户
// 平台管理员可以访问所有租户，租户管理员只能访问自己的租户
func (s *tenantService) canAccessTenant(ctx context.Context, tenantID string) bool {
	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return false
	}

	// 检查是否为平台管理员
	if hasSystemAdminRole(claims) {
		return true
	}

	// 租户管理员只能访问自己的租户
	return claims.TenantID == tenantID
}

// validateUpdateFields 验证租户更新请求的字段级权限
// 租户管理员只能修改name字段，平台管理员可以修改所有字段
func (s *tenantService) validateUpdateFields(ctx context.Context, req UpdateTenantRequest) error {
	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return errors.New("未找到身份认证信息")
	}

	// 平台管理员可以修改所有字段
	if hasSystemAdminRole(claims) {
		return nil
	}

	// 租户管理员只能修改name字段
	// 检查是否尝试修改其他字段
	if req.Domain != nil {
		return errors.New("权限不足：租户管理员只能修改租户名称")
	}
	if req.Metadata != nil {
		return errors.New("权限不足：租户管理员只能修改租户名称")
	}
	if req.Status != nil {
		return errors.New("权限不足：租户管理员只能修改租户名称")
	}

	return nil
}
