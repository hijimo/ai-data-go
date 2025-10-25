package auth

import (
	"context"
	"fmt"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/crypto"
	"genkit-ai-service/pkg/errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UserService 用户服务接口
type UserService interface {
	// Create 创建用户
	Create(ctx context.Context, req CreateUserRequest) (*model.User, error)

	// Get 获取用户
	Get(ctx context.Context, userID string) (*model.User, error)

	// Update 更新用户
	Update(ctx context.Context, userID string, req UpdateUserRequest) (*model.User, error)

	// Delete 删除用户
	Delete(ctx context.Context, userID string) error

	// List 列出用户（tenantID和search可选，仅平台管理员可用）
	List(ctx context.Context, page, pageSize int, search string, tenantID ...string) ([]*model.User, int64, error)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	// 所属租户ID（可选，租户管理员自动使用当前租户ID）
	TenantID string `json:"tenantId" validate:"omitempty"`
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
	userRepo         repository.UserRepository
	tenantRepo       repository.TenantRepository
	refreshTokenRepo repository.RefreshTokenRepository
	auditRepo        repository.AuditRepository
}

// NewUserService 创建用户服务实例
func NewUserService(
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	auditRepo repository.AuditRepository,
) UserService {
	return &userService{
		userRepo:         userRepo,
		tenantRepo:       tenantRepo,
		refreshTokenRepo: refreshTokenRepo,
		auditRepo:        auditRepo,
	}
}

// Create 创建用户
func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 检查是否为平台管理员
	isSystemAdmin := hasSystemAdminRole(claims)

	// 处理租户ID
	targetTenantID := req.TenantID
	if !isSystemAdmin {
		// 租户管理员：如果未提供tenantId，使用当前租户ID
		if targetTenantID == "" {
			targetTenantID = claims.TenantID
		} else if targetTenantID != claims.TenantID {
			// 如果提供了tenantId但与当前租户不匹配，返回权限错误
			return nil, errors.NewForbiddenError("权限不足：只能在当前租户下创建用户")
		}
	} else {
		// 平台管理员：必须提供tenantId
		if targetTenantID == "" {
			return nil, errors.NewBadRequestError("平台管理员必须指定租户ID")
		}
	}

	// 验证租户ID格式
	tenantUUID, err := uuid.Parse(targetTenantID)
	if err != nil {
		return nil, errors.NewBadRequestError("无效的租户ID格式")
	}

	// 验证租户是否存在且启用
	tenant, err := s.tenantRepo.GetByID(ctx, targetTenantID)
	if err != nil {
		return nil, errors.NewNotFoundError("租户不存在")
	}
	if !tenant.Status {
		return nil, errors.NewBadRequestError("租户已被禁用")
	}

	// 验证邮箱格式
	if req.Email == "" {
		return nil, errors.NewBadRequestError("邮箱不能为空")
	}

	// 检查邮箱在该租户下是否已存在
	existingUser, err := s.userRepo.GetByEmail(ctx, targetTenantID, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.NewBadRequestError(fmt.Sprintf("邮箱 %s 在该租户下已被使用", req.Email))
	}

	// 验证密码强度
	if err := crypto.ValidatePassword(req.Password); err != nil {
		return nil, errors.NewBadRequestError(fmt.Sprintf("密码验证失败: %v", err))
	}

	// 哈希密码
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("密码加密失败: %w", err))
	}

	// 从 Context 获取创建者信息（从 JWT 令牌中提取）
	createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)

	// 创建用户对象
	user := &model.User{
		ID:            uuid.New(),
		TenantID:      tenantUUID,
		Email:         req.Email,
		PasswordHash:  passwordHash,
		DisplayName:   req.DisplayName,
		Phone:         req.Phone,
		IsActive:      true, // 默认激活
		IsAdmin:       req.IsAdmin,
		CreatedBy:     createdByUUID,
		CreatedByName: createdByName,
		IsDeleted:     false,
	}

	// 设置角色
	if req.Roles != nil && len(req.Roles) > 0 {
		roles, err := datatypes.NewJSONType(req.Roles).MarshalJSON()
		if err != nil {
			return nil, errors.NewBadRequestError(fmt.Sprintf("角色格式错误: %v", err))
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
			return nil, errors.NewBadRequestError(fmt.Sprintf("元数据格式错误: %v", err))
		}
		user.Meta = meta
	}

	// 保存到数据库
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("创建用户失败: %w", err))
	}

	// 记录审计日志
	auditLog := &model.AuthAudit{
		TenantID: &user.TenantID,
		UserID:   &user.ID,
		Event:    model.AuditEventUserCreated,
		Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"userId":"%s","email":"%s","tenantId":"%s"}`, user.ID, user.Email, user.TenantID))),
	}
	_ = s.auditRepo.Create(ctx, auditLog)

	return user, nil
}

// Get 获取用户
func (s *userService) Get(ctx context.Context, userID string) (*model.User, error) {
	// 验证用户ID格式
	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.NewBadRequestError("无效的用户ID格式")
	}

	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 检查是否为平台管理员
	isSystemAdmin := hasSystemAdminRole(claims)

	var user *model.User
	if isSystemAdmin {
		// 平台管理员：可以查询任意租户的用户
		user, err = s.userRepo.GetByIDOnly(ctx, userID)
		if err != nil {
			return nil, errors.NewNotFoundError("用户不存在")
		}
	} else {
		// 租户管理员：只能查询自己租户的用户
		user, err = s.userRepo.GetByID(ctx, claims.TenantID, userID)
		if err != nil {
			return nil, errors.NewNotFoundError("用户不存在")
		}
	}

	// 验证用户访问权限
	if !s.canAccessUser(ctx, user.TenantID.String()) {
		return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的用户")
	}

	return user, nil
}

// Update 更新用户
func (s *userService) Update(ctx context.Context, userID string, req UpdateUserRequest) (*model.User, error) {
	// 验证用户ID格式
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.NewBadRequestError("无效的用户ID格式")
	}

	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 检查是否为平台管理员
	isSystemAdmin := hasSystemAdminRole(claims)

	// 获取现有用户
	var user *model.User
	var err error
	if isSystemAdmin {
		// 平台管理员：可以更新任意租户的用户
		user, err = s.userRepo.GetByIDOnly(ctx, userID)
	} else {
		// 租户管理员：只能更新自己租户的用户
		user, err = s.userRepo.GetByID(ctx, claims.TenantID, userID)
	}
	if err != nil {
		return nil, errors.NewNotFoundError("用户不存在")
	}

	// 验证用户访问权限
	if !s.canAccessUser(ctx, user.TenantID.String()) {
		return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的用户")
	}

	// 记录原始的激活状态，用于检测状态变化
	originalIsActive := user.IsActive

	// 更新字段
	if req.Email != nil {
		if *req.Email == "" {
			return nil, errors.NewBadRequestError("邮箱不能为空")
		}
		// 如果邮箱发生变化，检查新邮箱是否已被使用
		if *req.Email != user.Email {
			existingUser, err := s.userRepo.GetByEmail(ctx, user.TenantID.String(), *req.Email)
			if err == nil && existingUser != nil && existingUser.ID != user.ID {
				return nil, errors.NewBadRequestError(fmt.Sprintf("邮箱 %s 已被使用", *req.Email))
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
			return nil, errors.NewBadRequestError(fmt.Sprintf("角色格式错误: %v", err))
		}
		user.Roles = roles
	}

	if req.Meta != nil {
		meta, err := datatypes.NewJSONType(req.Meta).MarshalJSON()
		if err != nil {
			return nil, errors.NewBadRequestError(fmt.Sprintf("元数据格式错误: %v", err))
		}
		user.Meta = meta
	}

	// 保存更新
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("更新用户失败: %w", err))
	}

	// 检测用户激活状态变化：如果用户被禁用，撤销所有 Refresh Token
	if originalIsActive && !user.IsActive {
		// 用户从激活变为禁用，撤销所有 Token
		if err := s.refreshTokenRepo.RevokeAllByUser(ctx, user.TenantID.String(), userID); err != nil {
			// 记录错误但不影响更新流程
			fmt.Printf("撤销用户 Token 失败: %v\n", err)
		}

		// 记录用户禁用审计日志
		auditLog := &model.AuthAudit{
			TenantID: &user.TenantID,
			UserID:   &user.ID,
			Event:    model.AuditEventUserDisabled,
			Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"userId":"%s","email":"%s","tenantId":"%s"}`, user.ID, user.Email, user.TenantID))),
		}
		_ = s.auditRepo.Create(ctx, auditLog)
	} else if !originalIsActive && user.IsActive {
		// 用户从禁用变为激活
		// 记录用户启用审计日志
		auditLog := &model.AuthAudit{
			TenantID: &user.TenantID,
			UserID:   &user.ID,
			Event:    model.AuditEventUserEnabled,
			Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"userId":"%s","email":"%s","tenantId":"%s"}`, user.ID, user.Email, user.TenantID))),
		}
		_ = s.auditRepo.Create(ctx, auditLog)
	}

	return user, nil
}

// Delete 删除用户
func (s *userService) Delete(ctx context.Context, userID string) error {
	// 验证用户ID格式
	if _, err := uuid.Parse(userID); err != nil {
		return errors.NewBadRequestError("无效的用户ID格式")
	}

	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 检查是否为平台管理员
	isSystemAdmin := hasSystemAdminRole(claims)

	// 获取用户信息
	var user *model.User
	var err error
	if isSystemAdmin {
		// 平台管理员：可以删除任意租户的用户
		user, err = s.userRepo.GetByIDOnly(ctx, userID)
	} else {
		// 租户管理员：只能删除自己租户的用户
		user, err = s.userRepo.GetByID(ctx, claims.TenantID, userID)
	}
	if err != nil {
		return errors.NewNotFoundError("用户不存在")
	}

	// 验证用户访问权限
	if !s.canAccessUser(ctx, user.TenantID.String()) {
		return errors.NewForbiddenError("权限不足：无法访问其他租户的用户")
	}

	// 执行软删除
	if err := s.userRepo.Delete(ctx, user.TenantID.String(), userID); err != nil {
		return errors.NewInternalError(fmt.Errorf("删除用户失败: %w", err))
	}

	// 记录审计日志
	auditLog := &model.AuthAudit{
		TenantID: &user.TenantID,
		UserID:   &user.ID,
		Event:    model.AuditEventUserDeleted,
		Meta:     datatypes.JSON([]byte(fmt.Sprintf(`{"userId":"%s","email":"%s","tenantId":"%s"}`, user.ID, user.Email, user.TenantID))),
	}
	_ = s.auditRepo.Create(ctx, auditLog)

	return nil
}

// List 列出用户（tenantID和search可选，仅平台管理员可用）
func (s *userService) List(ctx context.Context, page, pageSize int, search string, tenantID ...string) ([]*model.User, int64, error) {
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
		return nil, 0, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 检查是否为平台管理员
	isSystemAdmin := hasSystemAdminRole(claims)

	// 如果是租户管理员，只返回当前租户的用户
	if !isSystemAdmin {
		users, total, err := s.userRepo.List(ctx, claims.TenantID, page, pageSize, search)
		if err != nil {
			return nil, 0, errors.NewInternalError(fmt.Errorf("获取用户列表失败: %w", err))
		}
		return users, total, nil
	}

	// 平台管理员：根据是否提供tenantID参数返回不同的列表
	if len(tenantID) > 0 && tenantID[0] != "" {
		// 提供了tenantID，返回指定租户的用户列表
		targetTenantID := tenantID[0]
		// 验证租户ID格式
		if _, err := uuid.Parse(targetTenantID); err != nil {
			return nil, 0, errors.NewBadRequestError("无效的租户ID格式")
		}
		users, total, err := s.userRepo.List(ctx, targetTenantID, page, pageSize, search)
		if err != nil {
			return nil, 0, errors.NewInternalError(fmt.Errorf("获取用户列表失败: %w", err))
		}
		return users, total, nil
	}

	// 未提供tenantID，返回所有租户的用户列表
	users, total, err := s.userRepo.ListAll(ctx, page, pageSize, search)
	if err != nil {
		return nil, 0, errors.NewInternalError(fmt.Errorf("获取用户列表失败: %w", err))
	}

	return users, total, nil
}

// canAccessUser 验证用户是否有权访问指定租户的用户
// 平台管理员可以访问所有租户的用户，租户管理员只能访问自己租户的用户
func (s *userService) canAccessUser(ctx context.Context, userTenantID string) bool {
	// 获取JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		return false
	}

	// 检查是否为平台管理员
	if hasSystemAdminRole(claims) {
		return true
	}

	// 租户管理员只能访问自己租户的用户
	return claims.TenantID == userTenantID
}
