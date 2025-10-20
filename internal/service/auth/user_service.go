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

// UserService 用户服务接口
type UserService interface {
	// Create 创建用户
	Create(ctx context.Context, req CreateUserRequest) (*model.User, error)

	// Get 获取用户
	Get(ctx context.Context, tenantID, userID string) (*model.User, error)

	// Update 更新用户
	Update(ctx context.Context, tenantID, userID string, req UpdateUserRequest) (*model.User, error)

	// Delete 删除用户
	Delete(ctx context.Context, tenantID, userID string) error

	// List 列出用户
	List(ctx context.Context, tenantID string, page, pageSize int) ([]*model.User, int64, error)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	// 所属租户ID
	TenantID string `json:"tenantId" validate:"required"`
	// 用户邮箱
	Email string `json:"email" validate:"required,email"`
	// 密码
	Password string `json:"password" validate:"required,min=8"`
	// 显示名称
	DisplayName string `json:"displayName"`
	// 手机号码
	Phone string `json:"phone"`
	// 是否为管理员
	IsAdmin bool `json:"isAdmin"`
	// 用户角色
	Roles []string `json:"roles"`
	// 用户元数据
	Meta map[string]interface{} `json:"meta"`
	// 创建者用户ID
	CreatedBy *string `json:"createdBy"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	// 用户邮箱
	Email *string `json:"email" validate:"omitempty,email"`
	// 显示名称
	DisplayName *string `json:"displayName"`
	// 手机号码
	Phone *string `json:"phone"`
	// 账户是否激活
	IsActive *bool `json:"isActive"`
	// 是否为管理员
	IsAdmin *bool `json:"isAdmin"`
	// 用户角色
	Roles []string `json:"roles"`
	// 用户元数据
	Meta map[string]interface{} `json:"meta"`
}

// userService 用户服务实现
type userService struct {
	userRepo   repository.UserRepository
	tenantRepo repository.TenantRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, tenantRepo repository.TenantRepository) UserService {
	return &userService{
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
	}
}

// Create 创建用户
func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
	// 验证租户ID格式
	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, errors.New("无效的租户ID格式")
	}

	// 验证租户是否存在且启用
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("租户不存在: %w", err)
	}
	if !tenant.Status {
		return nil, errors.New("租户已被禁用")
	}

	// 验证邮箱格式
	if req.Email == "" {
		return nil, errors.New("邮箱不能为空")
	}

	// 检查邮箱在该租户下是否已存在
	existingUser, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("邮箱 %s 在该租户下已被使用", req.Email)
	}

	// 验证密码强度
	if err := crypto.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("密码验证失败: %w", err)
	}

	// 哈希密码
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	var createdByUUID *uuid.UUID
	if req.CreatedBy != nil {
		parsed := parseUUIDPointer(*req.CreatedBy)
		if parsed == nil {
			return nil, errors.New("创建者用户ID格式无效")
		}
		createdByUUID = parsed
	}

	// 创建用户对象
	user := &model.User{
		ID:           uuid.New(),
		TenantID:     tenantUUID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Phone:        req.Phone,
		IsActive:     true, // 默认激活
		IsAdmin:      req.IsAdmin,
		CreatedBy:    createdByUUID,
		IsDeleted:    false,
	}

	// 设置角色
	if req.Roles != nil && len(req.Roles) > 0 {
		roles, err := datatypes.NewJSONType(req.Roles).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("角色格式错误: %w", err)
		}
		user.Roles = roles
	} else {
		// 默认角色为 user
		roles, _ := datatypes.NewJSONType([]string{"user"}).MarshalJSON()
		user.Roles = roles
	}

	// 设置元数据
	if req.Meta != nil {
		meta, err := datatypes.NewJSONType(req.Meta).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("元数据格式错误: %w", err)
		}
		user.Meta = meta
	}

	// 保存到数据库
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return user, nil
}

// Get 获取用户
func (s *userService) Get(ctx context.Context, tenantID, userID string) (*model.User, error) {
	// 验证租户ID格式
	if _, err := uuid.Parse(tenantID); err != nil {
		return nil, errors.New("无效的租户ID格式")
	}

	// 验证用户ID格式
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.New("无效的用户ID格式")
	}

	// 从数据库获取（自动包含租户隔离）
	user, err := s.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return user, nil
}

// Update 更新用户
func (s *userService) Update(ctx context.Context, tenantID, userID string, req UpdateUserRequest) (*model.User, error) {
	// 验证租户ID格式
	if _, err := uuid.Parse(tenantID); err != nil {
		return nil, errors.New("无效的租户ID格式")
	}

	// 验证用户ID格式
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.New("无效的用户ID格式")
	}

	// 获取现有用户（自动包含租户隔离）
	user, err := s.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 更新字段
	if req.Email != nil {
		if *req.Email == "" {
			return nil, errors.New("邮箱不能为空")
		}
		// 如果邮箱发生变化，检查新邮箱是否已被使用
		if *req.Email != user.Email {
			existingUser, err := s.userRepo.GetByEmail(ctx, tenantID, *req.Email)
			if err == nil && existingUser != nil && existingUser.ID != user.ID {
				return nil, fmt.Errorf("邮箱 %s 已被使用", *req.Email)
			}
		}
		user.Email = *req.Email
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}

	if req.Phone != nil {
		user.Phone = *req.Phone
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if req.Roles != nil {
		roles, err := datatypes.NewJSONType(req.Roles).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("角色格式错误: %w", err)
		}
		user.Roles = roles
	}

	if req.Meta != nil {
		meta, err := datatypes.NewJSONType(req.Meta).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("元数据格式错误: %w", err)
		}
		user.Meta = meta
	}

	// 保存更新（自动包含租户隔离）
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return user, nil
}

// Delete 删除用户
func (s *userService) Delete(ctx context.Context, tenantID, userID string) error {
	// 验证租户ID格式
	if _, err := uuid.Parse(tenantID); err != nil {
		return errors.New("无效的租户ID格式")
	}

	// 验证用户ID格式
	if _, err := uuid.Parse(userID); err != nil {
		return errors.New("无效的用户ID格式")
	}

	// 验证用户是否存在（自动包含租户隔离）
	_, err := s.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 执行软删除（自动包含租户隔离）
	if err := s.userRepo.Delete(ctx, tenantID, userID); err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	return nil
}

// List 列出用户
func (s *userService) List(ctx context.Context, tenantID string, page, pageSize int) ([]*model.User, int64, error) {
	// 验证租户ID格式
	if _, err := uuid.Parse(tenantID); err != nil {
		return nil, 0, errors.New("无效的租户ID格式")
	}

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

	// 从数据库获取列表（自动包含租户隔离）
	users, total, err := s.userRepo.List(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取用户列表失败: %w", err)
	}

	return users, total, nil
}
