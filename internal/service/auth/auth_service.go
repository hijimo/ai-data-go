package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/crypto"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	TenantID    string `json:"tenantId" validate:"required"`       // 租户 ID
	Email       string `json:"email" validate:"required,email"`    // 邮箱
	Password    string `json:"password" validate:"required,min=8"` // 密码（最少 8 位）
	DisplayName string `json:"displayName"`                        // 显示名称
	Phone       string `json:"phone"`                              // 手机号码
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	TenantID string `json:"tenantId"`                        // 租户 ID（可选，可从其他方式识别）
	Email    string `json:"email" validate:"required,email"` // 邮箱
	Password string `json:"password" validate:"required"`    // 密码
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string      `json:"accessToken"`  // 访问令牌
	RefreshToken string      `json:"refreshToken"` // 刷新令牌
	ExpiresIn    int64       `json:"expiresIn"`    // 过期时间（秒）
	TokenType    string      `json:"tokenType"`    // 令牌类型（Bearer）
	User         *model.User `json:"user"`         // 用户信息
}

// AuthService 认证服务接口
// 提供用户注册、登录、Token 刷新、注销和密码修改等功能
type AuthService interface {
	// Register 用户注册
	// 参数：
	//   - ctx: 上下文
	//   - req: 注册请求
	// 返回：
	//   - *model.User: 创建的用户信息
	//   - error: 注册失败时返回错误
	Register(ctx context.Context, req RegisterRequest) (*model.User, error)

	// Login 用户登录
	// 参数：
	//   - ctx: 上下文
	//   - req: 登录请求
	// 返回：
	//   - *LoginResponse: 登录响应（包含 tokens 和用户信息）
	//   - error: 登录失败时返回错误
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)

	// RefreshToken 刷新访问令牌
	// 参数：
	//   - ctx: 上下文
	//   - refreshToken: 刷新令牌字符串
	// 返回：
	//   - *LoginResponse: 新的 tokens 和用户信息
	//   - error: 刷新失败时返回错误
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)

	// Logout 用户注销
	// 参数：
	//   - ctx: 上下文
	//   - accessToken: 访问令牌字符串（用于加入黑名单）
	//   - refreshToken: 刷新令牌字符串
	// 返回：
	//   - error: 注销失败时返回错误
	Logout(ctx context.Context, accessToken, refreshToken string) error

	// ChangePassword 修改密码
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户 ID
	//   - userID: 用户 ID
	//   - oldPassword: 旧密码
	//   - newPassword: 新密码
	// 返回：
	//   - error: 修改失败时返回错误
	ChangePassword(ctx context.Context, tenantID, userID, oldPassword, newPassword string) error

	// UnlockAccount 解锁账户
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户 ID
	//   - userID: 用户 ID
	// 返回：
	//   - error: 解锁失败时返回错误
	UnlockAccount(ctx context.Context, tenantID, userID string) error
}

// authService 认证服务实现
type authService struct {
	userRepo         repository.UserRepository         // 用户数据访问
	tokenService     TokenService                      // Token 管理服务
	auditRepo        repository.AuditRepository        // 审计日志数据访问
	refreshTokenRepo repository.RefreshTokenRepository // 刷新令牌数据访问
	blacklistService TokenBlacklistService             // Token 黑名单服务
	accessTokenTTL   time.Duration                     // Access Token 生命周期
	maxLoginAttempts int                               // 最大登录尝试次数
	lockDuration     time.Duration                     // 账户锁定时长
}

// NewAuthService 创建认证服务实例
// 参数：
//   - userRepo: 用户数据访问接口
//   - tokenService: Token 管理服务接口
//   - auditRepo: 审计日志数据访问接口
//   - refreshTokenRepo: 刷新令牌数据访问接口
//   - blacklistService: Token 黑名单服务接口（可选）
//   - accessTokenTTL: Access Token 生命周期（默认 60 分钟）
//   - maxLoginAttempts: 最大登录尝试次数（默认 5 次）
//   - lockDuration: 账户锁定时长（默认 15 分钟）
//
// 返回：
//   - AuthService: 认证服务实例
func NewAuthService(
	userRepo repository.UserRepository,
	tokenService TokenService,
	auditRepo repository.AuditRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	blacklistService TokenBlacklistService,
	accessTokenTTL time.Duration,
	maxLoginAttempts int,
	lockDuration time.Duration,
) AuthService {
	// 如果未指定 TTL，使用默认值 60 分钟
	if accessTokenTTL == 0 {
		accessTokenTTL = 60 * time.Minute
	}

	// 如果未指定最大登录尝试次数，使用默认值 5 次
	if maxLoginAttempts == 0 {
		maxLoginAttempts = 5
	}

	// 如果未指定锁定时长，使用默认值 15 分钟
	if lockDuration == 0 {
		lockDuration = 15 * time.Minute
	}

	return &authService{
		userRepo:         userRepo,
		tokenService:     tokenService,
		auditRepo:        auditRepo,
		refreshTokenRepo: refreshTokenRepo,
		blacklistService: blacklistService,
		accessTokenTTL:   accessTokenTTL,
		maxLoginAttempts: maxLoginAttempts,
		lockDuration:     lockDuration,
	}
}

// Register 用户注册
func (s *authService) Register(ctx context.Context, req RegisterRequest) (*model.User, error) {
	// 1. 验证邮箱唯一性
	existingUser, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("邮箱已被注册")
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, errors.New("租户ID无效")
	}

	// 2. 哈希密码
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 3. 创建用户记录
	user := &model.User{
		ID:           uuid.New(),
		TenantID:     tenantUUID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Phone:        req.Phone,
		IsActive:     true,
		IsAdmin:      false,
		Roles:        datatypes.JSON([]byte(`["user"]`)), // 默认角色为 user
		Meta:         datatypes.JSON([]byte(`{}`)),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return user, nil
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// 1. 验证租户和邮箱，获取用户
	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		// 记录登录失败审计日志
		s.createAuditLog(ctx, parseUUIDPointer(req.TenantID), nil, "failed_login", map[string]interface{}{
			"email":  req.Email,
			"reason": "user_not_found",
		})
		return nil, errors.New("邮箱或密码错误")
	}

	// 2. 检查账户是否被锁定
	tenantIDStr := user.TenantID.String()
	userIDStr := user.ID.String()

	isLocked, err := s.userRepo.IsAccountLocked(ctx, tenantIDStr, userIDStr)
	if err != nil {
		return nil, fmt.Errorf("检查账户锁定状态失败: %w", err)
	}
	if isLocked {
		s.createAuditLog(ctx, &user.TenantID, &user.ID, "failed_login", map[string]interface{}{
			"email":  req.Email,
			"reason": "account_locked",
		})
		return nil, errors.New("账户已被锁定，请稍后再试")
	}

	// 3. 验证用户状态
	if !user.IsActive {
		s.createAuditLog(ctx, &user.TenantID, &user.ID, "failed_login", map[string]interface{}{
			"email":  req.Email,
			"reason": "user_inactive",
		})
		return nil, errors.New("用户账户未激活")
	}

	// 4. 验证密码
	if err := crypto.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// 增加登录失败次数
		if incrementErr := s.userRepo.IncrementFailedLoginAttempts(ctx, tenantIDStr, userIDStr); incrementErr != nil {
			fmt.Printf("增加登录失败次数失败: %v\n", incrementErr)
		}

		// 重新获取用户信息以获取最新的失败次数
		user, _ = s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
		if user != nil {
			tenantIDStr = user.TenantID.String()
			userIDStr = user.ID.String()
		}

		// 检查是否达到最大失败次数，如果是则锁定账户
		if user != nil && user.FailedLoginAttempts >= s.maxLoginAttempts {
			if lockErr := s.userRepo.LockAccount(ctx, tenantIDStr, userIDStr, s.lockDuration); lockErr != nil {
				fmt.Printf("锁定账户失败: %v\n", lockErr)
			}
			s.createAuditLog(ctx, &user.TenantID, &user.ID, "account_locked", map[string]interface{}{
				"email":                 req.Email,
				"failed_login_attempts": user.FailedLoginAttempts,
				"lock_duration_minutes": s.lockDuration.Minutes(),
			})
			return nil, fmt.Errorf("登录失败次数过多，账户已被锁定 %d 分钟", int(s.lockDuration.Minutes()))
		}

		s.createAuditLog(ctx, &user.TenantID, &user.ID, "failed_login", map[string]interface{}{
			"email":                 req.Email,
			"reason":                "invalid_password",
			"failed_login_attempts": user.FailedLoginAttempts,
		})

		remainingAttempts := s.maxLoginAttempts - user.FailedLoginAttempts
		if remainingAttempts > 0 {
			return nil, fmt.Errorf("邮箱或密码错误，还剩 %d 次尝试机会", remainingAttempts)
		}
		return nil, errors.New("邮箱或密码错误")
	}

	// 5. 密码验证成功，重置登录失败次数
	if err := s.userRepo.ResetFailedLoginAttempts(ctx, tenantIDStr, userIDStr); err != nil {
		// 记录错误但不影响登录流程
		fmt.Printf("重置登录失败次数失败: %v\n", err)
	}

	// 6. 生成 Access Token
	accessToken, err := s.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 7. 生成 Refresh Token
	refreshToken, _, err := s.tokenService.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 8. 更新最后登录时间
	if err := s.userRepo.UpdateLastLogin(ctx, tenantIDStr, userIDStr); err != nil {
		// 记录错误但不影响登录流程
		fmt.Printf("更新最后登录时间失败: %v\n", err)
	}

	// 9. 记录登录成功审计日志
	s.createAuditLog(ctx, &user.TenantID, &user.ID, "login", map[string]interface{}{
		"email": req.Email,
	})

	// 10. 返回登录响应
	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
		User:         user,
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 1. 验证 Refresh Token
	token, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("刷新令牌无效: %w", err)
	}

	// 2. 获取用户信息
	user, err := s.userRepo.GetByID(ctx, token.TenantID.String(), token.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 3. 验证用户状态
	if !user.IsActive {
		return nil, errors.New("用户账户未激活")
	}

	// 4. 生成新的 Access Token
	newAccessToken, err := s.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 5. 生成新的 Refresh Token
	newRefreshToken, newTokenRecord, err := s.tokenService.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 6. 撤销旧的 Refresh Token 并记录 replaced_by
	replacedByID := newTokenRecord.ID.String()
	if err := s.refreshTokenRepo.Revoke(ctx, token.ID.String(), &replacedByID); err != nil {
		// 记录错误但不影响刷新流程
		fmt.Printf("撤销旧令牌失败: %v\n", err)
	}

	// 7. 记录刷新审计日志
	s.createAuditLog(ctx, &user.TenantID, &user.ID, "refresh", map[string]interface{}{
		"old_token_id": token.ID.String(),
		"new_token_id": newTokenRecord.ID.String(),
	})

	// 8. 返回新的 tokens
	return &LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
		User:         user,
	}, nil
}

// Logout 用户注销
func (s *authService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// 1. 验证 Refresh Token
	token, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("刷新令牌无效: %w", err)
	}

	// 2. 将 Access Token 加入黑名单（如果启用）
	if s.blacklistService != nil && accessToken != "" {
		// 验证 access token 以获取过期时间
		claims, err := s.tokenService.ValidateAccessToken(accessToken)
		if err == nil && claims != nil && claims.ExpiresAt != nil {
			expiresAt := claims.ExpiresAt.Time
			if err := s.blacklistService.AddToBlacklist(ctx, accessToken, expiresAt); err != nil {
				// 记录错误但不阻止注销流程
				// 因为即使黑名单添加失败，refresh token 仍会被撤销
				fmt.Printf("将 access token 加入黑名单失败: %v\n", err)
			}
		}
	}

	// 3. 撤销 Refresh Token
	if err := s.refreshTokenRepo.Revoke(ctx, token.ID.String(), nil); err != nil {
		return fmt.Errorf("撤销令牌失败: %w", err)
	}

	// 4. 记录注销审计日志
	s.createAuditLog(ctx, &token.TenantID, &token.UserID, "logout", map[string]interface{}{
		"token_id": token.ID.String(),
	})

	return nil
}

// ChangePassword 修改密码
func (s *authService) ChangePassword(ctx context.Context, tenantID, userID, oldPassword, newPassword string) error {
	// 1. 获取用户信息
	user, err := s.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 2. 验证旧密码
	if err := crypto.VerifyPassword(user.PasswordHash, oldPassword); err != nil {
		return errors.New("旧密码错误")
	}

	// 3. 哈希新密码
	newPasswordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	// 4. 更新用户密码
	user.PasswordHash = newPasswordHash
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 5. 撤销所有 Refresh Token（强制重新登录）
	if err := s.refreshTokenRepo.RevokeAllByUser(ctx, tenantID, userID); err != nil {
		// 记录错误但不影响密码修改流程
		fmt.Printf("撤销所有令牌失败: %v\n", err)
	}

	// 6. 记录密码修改审计日志
	tenantUUID := parseUUIDPointer(tenantID)
	userUUID := parseUUIDPointer(userID)
	s.createAuditLog(ctx, tenantUUID, userUUID, "change_password", map[string]interface{}{
		"success": true,
	})

	return nil
}

// UnlockAccount 解锁账户
func (s *authService) UnlockAccount(ctx context.Context, tenantID, userID string) error {
	// 1. 验证用户是否存在
	user, err := s.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 2. 解锁账户
	if err := s.userRepo.UnlockAccount(ctx, tenantID, userID); err != nil {
		return fmt.Errorf("解锁账户失败: %w", err)
	}

	// 3. 记录解锁审计日志
	tenantUUID := parseUUIDPointer(tenantID)
	userUUID := parseUUIDPointer(userID)
	s.createAuditLog(ctx, tenantUUID, userUUID, "account_unlocked", map[string]interface{}{
		"email": user.Email,
	})

	return nil
}

// createAuditLog 创建审计日志（内部辅助方法）
func (s *authService) createAuditLog(ctx context.Context, tenantID, userID *uuid.UUID, event string, meta map[string]interface{}) {
	// 从上下文中提取 IP 和 User-Agent（如果可用）
	ip := ""
	userAgent := ""

	// 注意：实际实现中应该从 HTTP 请求上下文中提取这些信息
	// 这里只是占位符实现

	// 将 meta 转换为 JSON
	var metaJSON datatypes.JSON
	if meta != nil {
		if data, err := json.Marshal(meta); err == nil {
			metaJSON = datatypes.JSON(data)
		}
	}

	audit := &model.AuthAudit{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Event:     event,
		IP:        ip,
		UserAgent: userAgent,
		Meta:      metaJSON,
	}

	// 异步记录审计日志，不阻塞主流程
	go func() {
		if err := s.auditRepo.Create(context.Background(), audit); err != nil {
			fmt.Printf("记录审计日志失败: %v\n", err)
		}
	}()
}

func parseUUIDPointer(id string) *uuid.UUID {
	if id == "" {
		return nil
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil
	}
	return &parsed
}
