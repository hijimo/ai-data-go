package auth

import (
	"context"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/crypto"

	"gorm.io/datatypes"
)

// BootstrapResult 初始化结果
type BootstrapResult struct {
	Initialized   bool   // 是否执行了初始化（false 表示系统已初始化，跳过）
	AdminEmail    string // 管理员邮箱
	AdminPassword string // 管理员初始密码（仅在新创建时返回）
	TenantID      string // 平台租户 ID
}

// BootstrapService 系统初始化服务接口
type BootstrapService interface {
	// Initialize 初始化平台租户和平台管理员
	// 如果系统已初始化，则跳过
	// 返回初始化结果，包含管理员邮箱和密码信息
	Initialize(ctx context.Context) (*BootstrapResult, error)

	// IsInitialized 检查系统是否已初始化
	// 通过检查是否存在 type='system' 的租户来判断
	IsInitialized(ctx context.Context) (bool, error)
}

// bootstrapService 系统初始化服务实现
type bootstrapService struct {
	tenantRepo repository.TenantRepository
	userRepo   repository.UserRepository
	config     *config.BootstrapConfig
}

// NewBootstrapService 创建系统初始化服务实例
func NewBootstrapService(
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
	config *config.BootstrapConfig,
) BootstrapService {
	return &bootstrapService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		config:     config,
	}
}

// IsInitialized 检查系统是否已初始化
func (s *bootstrapService) IsInitialized(ctx context.Context) (bool, error) {
	// 检查是否存在 type='system' 的租户
	tenants, err := s.tenantRepo.GetByType(ctx, "system")
	if err != nil {
		return false, err
	}

	return len(tenants) > 0, nil
}

// Initialize 初始化平台租户和平台管理员
func (s *bootstrapService) Initialize(ctx context.Context) (*BootstrapResult, error) {
	// 1. 检查系统是否已初始化
	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return nil, err
	}

	if initialized {
		// 系统已初始化，跳过
		return &BootstrapResult{
			Initialized: false,
		}, nil
	}

	// 2. 创建平台租户
	tenant, err := s.createPlatformTenant(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 创建平台管理员
	adminPassword, err := s.createPlatformAdmin(ctx, tenant)
	if err != nil {
		return nil, err
	}

	// 4. 返回初始化结果
	return &BootstrapResult{
		Initialized:   true,
		AdminEmail:    s.config.AdminEmail,
		AdminPassword: adminPassword,
		TenantID:      tenant.ID.String(),
	}, nil
}

// createPlatformTenant 创建平台租户
func (s *bootstrapService) createPlatformTenant(ctx context.Context) (*model.Tenant, error) {
	tenant := &model.Tenant{
		Name:   s.config.TenantName,
		Domain: s.config.TenantDomain,
		Type:   model.TenantTypeSystem,
		Status: true,
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// createPlatformAdmin 创建平台管理员
func (s *bootstrapService) createPlatformAdmin(ctx context.Context, tenant *model.Tenant) (string, error) {
	// 1. 确定管理员密码
	password := s.config.AdminPassword
	if password == "" {
		// 如果未配置密码，生成随机密码
		var err error
		password, err = crypto.GenerateSecurePassword(16)
		if err != nil {
			return "", err
		}
	}

	// 2. 哈希密码
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return "", err
	}

	// 3. 创建管理员用户
	admin := &model.User{
		TenantID:     tenant.ID,
		Email:        s.config.AdminEmail,
		PasswordHash: passwordHash,
		DisplayName:  s.config.AdminDisplayName,
		IsActive:     true,
		IsAdmin:      true,
		Roles:        datatypes.JSON([]byte(`["system_admin"]`)),
	}

	if err := s.userRepo.Create(ctx, admin); err != nil {
		return "", err
	}

	// 返回明文密码（仅用于初始化日志）
	return password, nil
}
