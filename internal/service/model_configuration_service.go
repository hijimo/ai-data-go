package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"

	"github.com/google/uuid"
)

// ModelConfigurationService 模型配置服务接口
type ModelConfigurationService interface {
	// Create 创建模型配置
	Create(ctx context.Context, req model.CreateModelConfigurationRequest) (*model.ModelConfiguration, error)

	// Get 获取模型配置
	Get(ctx context.Context, id uuid.UUID) (*model.ModelConfiguration, error)

	// List 列出模型配置（支持分页和租户过滤）
	List(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]*model.ModelConfiguration, int64, error)

	// Update 更新模型配置
	Update(ctx context.Context, id uuid.UUID, req model.UpdateModelConfigurationRequest) (*model.ModelConfiguration, error)

	// Delete 删除模型配置（软删除）
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateStatus 更新模型配置状态
	UpdateStatus(ctx context.Context, id uuid.UUID, enabled bool) error

	// Validate 验证模型配置是否可以正确连接
	Validate(ctx context.Context, id uuid.UUID) (*model.ValidationResult, error)

	// ListAvailable 获取当前租户下所有可用的模型配置
	ListAvailable(ctx context.Context) ([]*model.ModelConfiguration, error)
}

// modelConfigurationService 模型配置服务实现
type modelConfigurationService struct {
	repo              repository.ModelConfigurationRepository
	encryptionService EncryptionService
}

// NewModelConfigurationService 创建新的模型配置服务实例
func NewModelConfigurationService(
	repo repository.ModelConfigurationRepository,
	encryptionService EncryptionService,
) ModelConfigurationService {
	return &modelConfigurationService{
		repo:              repo,
		encryptionService: encryptionService,
	}
}

// hasRole 检查用户是否具有指定角色（使用 context_service_impl.go 中的实现）
// 注意：此函数已在 context_service_impl.go 中定义，这里不需要重复声明

// Create 创建模型配置
func (s *modelConfigurationService) Create(ctx context.Context, req model.CreateModelConfigurationRequest) (*model.ModelConfiguration, error) {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 验证模型提供商
	if !model.IsValidModelProvider(req.ModelProvider) {
		return nil, errors.NewBadRequestError("无效的模型提供商")
	}

	// 处理租户ID
	var targetTenantID uuid.UUID
	if hasRole(claims, model.RoleTenantAdmin) {
		// 租户管理员只能在自己的租户下创建
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}

		if req.TenantID != nil && *req.TenantID != tenantUUID {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"event":            "permission_denied",
				"reason":           "尝试在其他租户下创建模型配置",
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_tenant_id": req.TenantID.String(),
			})
			return nil, errors.NewForbiddenError("权限不足：只能在当前租户下创建模型配置")
		}
		targetTenantID = tenantUUID
	} else if hasRole(claims, model.RoleSystemAdmin) {
		// 平台管理员必须指定租户ID
		if req.TenantID == nil {
			return nil, errors.NewBadRequestError("平台管理员必须指定租户ID")
		}
		targetTenantID = *req.TenantID
	} else {
		return nil, errors.NewForbiddenError("权限不足：需要管理员权限")
	}

	// 加密API密钥
	encryptedKey, err := s.encryptionService.EncryptAPIKey(req.APIKey)
	if err != nil {
		logger.ErrorContext(ctx, "加密API密钥失败", logger.Fields{
			"error":     err.Error(),
			"user_id":   claims.Subject,
			"tenant_id": targetTenantID.String(),
		})
		return nil, errors.NewInternalError(err)
	}

	// 解析创建者ID
	createdBy, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}

	// 创建模型配置
	config := &model.ModelConfiguration{
		TenantID:      targetTenantID,
		Name:          req.Name,
		Model:         req.Model,
		ModelProvider: req.ModelProvider,
		BaseURL:       req.BaseURL,
		APIKey:        encryptedKey,
		QueryParams:   req.QueryParams,
		IsEnabled:     true,
		CreatedBy:     createdBy,
	}

	// 保存到数据库
	result, err := s.repo.Create(ctx, config)
	if err != nil {
		logger.ErrorContext(ctx, "创建模型配置失败", logger.Fields{
			"error":     err.Error(),
			"user_id":   claims.Subject,
			"tenant_id": targetTenantID.String(),
		})
		return nil, err
	}

	// 记录审计日志
	logger.InfoContext(ctx, "创建模型配置", logger.Fields{
		"event":     "model_config_created",
		"user_id":   claims.Subject,
		"tenant_id": targetTenantID.String(),
		"config_id": result.ID.String(),
		"provider":  req.ModelProvider,
		"model":     req.Model,
	})

	// 脱敏API密钥
	result.APIKey = s.encryptionService.MaskAPIKey(req.APIKey)

	return result, nil
}

// Get 获取模型配置
func (s *modelConfigurationService) Get(ctx context.Context, id uuid.UUID) (*model.ModelConfiguration, error) {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 查询配置
	config, err := s.repo.FindByID(ctx, id)
	if err != nil {
		// 对于租户管理员，即使资源不存在也返回403
		if !hasRole(claims, model.RoleSystemAdmin) {
			return nil, errors.NewForbiddenError("权限不足")
		}
		return nil, err
	}

	// 租户管理员只能访问自己租户的配置
	if hasRole(claims, model.RoleTenantAdmin) {
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}

		if config.TenantID != tenantUUID {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"event":            "permission_denied",
				"reason":           "尝试访问其他租户的模型配置",
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_config_id": id.String(),
				"target_tenant_id": config.TenantID.String(),
			})
			return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的模型配置")
		}
	}

	// 脱敏API密钥
	config.APIKey = s.encryptionService.MaskAPIKey(config.APIKey)

	return config, nil
}

// List 列出模型配置（支持分页和租户过滤）
func (s *modelConfigurationService) List(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]*model.ModelConfiguration, int64, error) {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, 0, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	var filterTenantID *uuid.UUID

	// 租户管理员只能查看自己租户的配置
	if hasRole(claims, model.RoleTenantAdmin) {
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return nil, 0, errors.NewInternalError(err)
		}
		filterTenantID = &tenantUUID
	} else if hasRole(claims, model.RoleSystemAdmin) {
		// 平台管理员可以查看所有或指定租户的配置
		filterTenantID = tenantID
	} else {
		return nil, 0, errors.NewForbiddenError("权限不足：需要管理员权限")
	}

	// 查询配置列表
	configs, total, err := s.repo.FindByTenant(ctx, filterTenantID, pageNo, pageSize)
	if err != nil {
		logger.ErrorContext(ctx, "查询模型配置列表失败", logger.Fields{
			"error":   err.Error(),
			"user_id": claims.Subject,
		})
		return nil, 0, err
	}

	// 脱敏所有配置的API密钥
	for i := range configs {
		configs[i].APIKey = s.encryptionService.MaskAPIKey(configs[i].APIKey)
	}

	return configs, total, nil
}

// Update 更新模型配置
func (s *modelConfigurationService) Update(ctx context.Context, id uuid.UUID, req model.UpdateModelConfigurationRequest) (*model.ModelConfiguration, error) {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 先查询配置
	config, err := s.repo.FindByID(ctx, id)
	if err != nil {
		// 对于租户管理员，即使资源不存在也返回403
		if !hasRole(claims, model.RoleSystemAdmin) {
			return nil, errors.NewForbiddenError("权限不足")
		}
		return nil, err
	}

	// 租户管理员只能更新自己租户的配置
	if hasRole(claims, model.RoleTenantAdmin) {
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}

		if config.TenantID != tenantUUID {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"event":            "permission_denied",
				"reason":           "尝试更新其他租户的模型配置",
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_config_id": id.String(),
				"target_tenant_id": config.TenantID.String(),
			})
			return nil, errors.NewForbiddenError("权限不足：无法更新其他租户的模型配置")
		}
	}

	// 更新字段
	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.Model != nil {
		config.Model = *req.Model
	}
	if req.BaseURL != nil {
		config.BaseURL = req.BaseURL
	}
	if req.APIKey != nil {
		// 加密新的API密钥
		encryptedKey, err := s.encryptionService.EncryptAPIKey(*req.APIKey)
		if err != nil {
			logger.ErrorContext(ctx, "加密API密钥失败", logger.Fields{
				"error":     err.Error(),
				"user_id":   claims.Subject,
				"config_id": id.String(),
			})
			return nil, errors.NewInternalError(err)
		}
		config.APIKey = encryptedKey
	}
	if req.QueryParams != nil {
		config.QueryParams = req.QueryParams
	}

	// 更新时间和更新者
	now := time.Now()
	updatedBy, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	config.UpdatedBy = &updatedBy
	config.UpdatedAt = &now

	// 保存更新
	result, err := s.repo.Update(ctx, id, config)
	if err != nil {
		logger.ErrorContext(ctx, "更新模型配置失败", logger.Fields{
			"error":     err.Error(),
			"user_id":   claims.Subject,
			"config_id": id.String(),
		})
		return nil, err
	}

	// 记录审计日志
	logger.InfoContext(ctx, "更新模型配置", logger.Fields{
		"event":     "model_config_updated",
		"user_id":   claims.Subject,
		"tenant_id": config.TenantID.String(),
		"config_id": id.String(),
	})

	// 脱敏API密钥
	result.APIKey = s.encryptionService.MaskAPIKey(result.APIKey)

	return result, nil
}

// Delete 删除模型配置（软删除）
func (s *modelConfigurationService) Delete(ctx context.Context, id uuid.UUID) error {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 先查询配置
	config, err := s.repo.FindByID(ctx, id)
	if err != nil {
		// 对于租户管理员，即使资源不存在也返回403
		if !hasRole(claims, model.RoleSystemAdmin) {
			return errors.NewForbiddenError("权限不足")
		}
		return err
	}

	// 租户管理员只能删除自己租户的配置
	if hasRole(claims, model.RoleTenantAdmin) {
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return errors.NewInternalError(err)
		}

		if config.TenantID != tenantUUID {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"event":            "permission_denied",
				"reason":           "尝试删除其他租户的模型配置",
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_config_id": id.String(),
				"target_tenant_id": config.TenantID.String(),
			})
			return errors.NewForbiddenError("权限不足：无法删除其他租户的模型配置")
		}
	}

	// 解析删除者ID
	deletedBy, err := uuid.Parse(claims.Subject)
	if err != nil {
		return errors.NewInternalError(err)
	}

	// 执行软删除
	if err := s.repo.SoftDelete(ctx, id, deletedBy); err != nil {
		logger.ErrorContext(ctx, "删除模型配置失败", logger.Fields{
			"error":     err.Error(),
			"user_id":   claims.Subject,
			"config_id": id.String(),
		})
		return err
	}

	// 记录审计日志
	logger.InfoContext(ctx, "删除模型配置", logger.Fields{
		"event":     "model_config_deleted",
		"user_id":   claims.Subject,
		"tenant_id": config.TenantID.String(),
		"config_id": id.String(),
	})

	return nil
}

// UpdateStatus 更新模型配置状态
func (s *modelConfigurationService) UpdateStatus(ctx context.Context, id uuid.UUID, enabled bool) error {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 先查询配置
	config, err := s.repo.FindByID(ctx, id)
	if err != nil {
		// 对于租户管理员，即使资源不存在也返回403
		if !hasRole(claims, model.RoleSystemAdmin) {
			return errors.NewForbiddenError("权限不足")
		}
		return err
	}

	// 租户管理员只能更新自己租户的配置状态
	if hasRole(claims, model.RoleTenantAdmin) {
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return errors.NewInternalError(err)
		}

		if config.TenantID != tenantUUID {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"event":            "permission_denied",
				"reason":           "尝试更新其他租户的模型配置状态",
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_config_id": id.String(),
				"target_tenant_id": config.TenantID.String(),
			})
			return errors.NewForbiddenError("权限不足：无法更新其他租户的模型配置状态")
		}
	}

	// 更新状态
	if err := s.repo.UpdateStatus(ctx, id, enabled); err != nil {
		logger.ErrorContext(ctx, "更新模型配置状态失败", logger.Fields{
			"error":     err.Error(),
			"user_id":   claims.Subject,
			"config_id": id.String(),
		})
		return err
	}

	// 记录审计日志
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	logger.InfoContext(ctx, "更新模型配置状态", logger.Fields{
		"event":     "model_config_status_updated",
		"user_id":   claims.Subject,
		"tenant_id": config.TenantID.String(),
		"config_id": id.String(),
		"status":    status,
	})

	return nil
}

// ListAvailable 获取当前租户下所有可用的模型配置
func (s *modelConfigurationService) ListAvailable(ctx context.Context) ([]*model.ModelConfiguration, error) {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 解析租户ID
	tenantUUID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}

	// 查询当前租户下已启用且未删除的配置
	configs, err := s.repo.FindAvailableByTenant(ctx, tenantUUID)
	if err != nil {
		logger.ErrorContext(ctx, "查询可用模型配置失败", logger.Fields{
			"error":     err.Error(),
			"user_id":   claims.Subject,
			"tenant_id": claims.TenantID,
		})
		return nil, err
	}

	// 移除敏感信息（不需要脱敏，直接清空）
	result := make([]*model.ModelConfiguration, len(configs))
	for i, config := range configs {
		result[i] = &model.ModelConfiguration{
			ID:            config.ID,
			Name:          config.Name,
			Model:         config.Model,
			ModelProvider: config.ModelProvider,
		}
	}

	return result, nil
}

// Validate 验证模型配置是否可以正确连接
func (s *modelConfigurationService) Validate(ctx context.Context, id uuid.UUID) (*model.ValidationResult, error) {
	// 获取JWT Claims
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok {
		return nil, errors.NewUnauthorizedError("未找到身份认证信息")
	}

	// 查询配置
	config, err := s.repo.FindByID(ctx, id)
	if err != nil {
		// 对于租户管理员，即使资源不存在也返回403
		if !hasRole(claims, model.RoleSystemAdmin) {
			return nil, errors.NewForbiddenError("权限不足")
		}
		return nil, err
	}

	// 租户管理员只能验证自己租户的配置
	if hasRole(claims, model.RoleTenantAdmin) {
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}

		if config.TenantID != tenantUUID {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"event":            "permission_denied",
				"reason":           "尝试验证其他租户的模型配置",
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_config_id": id.String(),
				"target_tenant_id": config.TenantID.String(),
			})
			return nil, errors.NewForbiddenError("权限不足：无法验证其他租户的模型配置")
		}
	}

	// 解密API密钥
	apiKey, err := s.encryptionService.DecryptAPIKey(config.APIKey)
	if err != nil {
		logger.ErrorContext(ctx, "解密API密钥失败", logger.Fields{
			"error":     err.Error(),
			"config_id": id.String(),
		})
		return &model.ValidationResult{
			Valid:   false,
			Message: "解密API密钥失败",
			Details: err.Error(),
		}, nil
	}

	// 设置30秒超时
	validateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 根据不同的提供商进行验证
	var result *model.ValidationResult
	switch config.ModelProvider {
	case model.ModelProviderOpenAI:
		result = s.validateOpenAI(validateCtx, config, apiKey)
	case model.ModelProviderAnthropic:
		result = s.validateAnthropic(validateCtx, config, apiKey)
	case model.ModelProviderGoogleGenAI:
		result = s.validateGoogleGenAI(validateCtx, config, apiKey)
	case model.ModelProviderAzureOpenAI:
		result = s.validateAzureOpenAI(validateCtx, config, apiKey)
	case model.ModelProviderBianlian:
		result = s.validateBianlian(validateCtx, config, apiKey)
	case model.ModelProviderCustomOpenAI:
		result = s.validateCustomOpenAI(validateCtx, config, apiKey)
	default:
		return nil, errors.NewBadRequestError("不支持的模型提供商")
	}

	// 记录验证结果
	if result.Valid {
		logger.InfoContext(ctx, "模型配置验证成功", logger.Fields{
			"event":     "validation_success",
			"config_id": id.String(),
			"provider":  config.ModelProvider,
		})
	} else {
		logger.WarnContext(ctx, "模型配置验证失败", logger.Fields{
			"event":     "validation_failed",
			"config_id": id.String(),
			"provider":  config.ModelProvider,
			"message":   result.Message,
			"details":   result.Details,
		})
	}

	// 无论验证成功还是失败，都返回结果对象，不返回错误
	// Handler层会根据result.Valid字段判断验证是否成功
	return result, nil
}

// validateOpenAI 验证OpenAI配置
func (s *modelConfigurationService) validateOpenAI(ctx context.Context, config *model.ModelConfiguration, apiKey string) *model.ValidationResult {
	client := &http.Client{Timeout: 30 * time.Second}

	baseURL := "https://api.openai.com/v1"
	if config.BaseURL != nil && *config.BaseURL != "" {
		baseURL = *config.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &model.ValidationResult{
				Valid:   false,
				Message: "验证超时",
				Details: "连接超过30秒",
			}
		}
		return &model.ValidationResult{
			Valid:   false,
			Message: "连接失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &model.ValidationResult{
			Valid:   true,
			Message: "验证成功",
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return &model.ValidationResult{
		Valid:   false,
		Message: "验证失败",
		Details: fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body)),
	}
}

// validateAnthropic 验证Anthropic配置
func (s *modelConfigurationService) validateAnthropic(ctx context.Context, config *model.ModelConfiguration, apiKey string) *model.ValidationResult {
	client := &http.Client{Timeout: 30 * time.Second}

	baseURL := "https://api.anthropic.com/v1"
	if config.BaseURL != nil && *config.BaseURL != "" {
		baseURL = *config.BaseURL
	}

	// Anthropic使用messages端点进行验证
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", nil)
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &model.ValidationResult{
				Valid:   false,
				Message: "验证超时",
				Details: "连接超过30秒",
			}
		}
		return &model.ValidationResult{
			Valid:   false,
			Message: "连接失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	// Anthropic API在没有body的情况下会返回400，但如果API key有效，会返回特定的错误信息
	// 401表示认证失败，其他错误码表示API key有效但请求格式不对
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return &model.ValidationResult{
			Valid:   false,
			Message: "API密钥无效",
			Details: fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body)),
		}
	}

	// 其他状态码表示API key有效
	return &model.ValidationResult{
		Valid:   true,
		Message: "验证成功",
	}
}

// validateGoogleGenAI 验证Google GenAI配置
func (s *modelConfigurationService) validateGoogleGenAI(ctx context.Context, config *model.ModelConfiguration, apiKey string) *model.ValidationResult {
	client := &http.Client{Timeout: 30 * time.Second}

	baseURL := "https://generativelanguage.googleapis.com/v1beta"
	if config.BaseURL != nil && *config.BaseURL != "" {
		baseURL = *config.BaseURL
	}

	// 使用models.list端点验证
	url := fmt.Sprintf("%s/models?key=%s", baseURL, apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &model.ValidationResult{
				Valid:   false,
				Message: "验证超时",
				Details: "连接超过30秒",
			}
		}
		return &model.ValidationResult{
			Valid:   false,
			Message: "连接失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &model.ValidationResult{
			Valid:   true,
			Message: "验证成功",
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return &model.ValidationResult{
		Valid:   false,
		Message: "验证失败",
		Details: fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body)),
	}
}

// validateAzureOpenAI 验证Azure OpenAI配置
func (s *modelConfigurationService) validateAzureOpenAI(ctx context.Context, config *model.ModelConfiguration, apiKey string) *model.ValidationResult {
	if config.BaseURL == nil || *config.BaseURL == "" {
		return &model.ValidationResult{
			Valid:   false,
			Message: "Azure OpenAI需要配置BaseURL",
			Details: "BaseURL格式: https://{resource-name}.openai.azure.com",
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 构建URL: {baseurl}/openai/deployments/{modelname}?{queryParams}
	url := fmt.Sprintf("%sopenai/deployments/%s/responses", *config.BaseURL, config.Model)
	
	// 附加queryParams
	if config.QueryParams != nil && *config.QueryParams != "" {
		// QueryParams 是 JSON 字符串，需要解析
		var params map[string]string
		if err := json.Unmarshal([]byte(*config.QueryParams), &params); err == nil && len(params) > 0 {
			url += "?"
			first := true
			for key, value := range params {
				if !first {
					url += "&"
				}
				url += fmt.Sprintf("%s=%s", key, value)
				first = false
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	req.Header.Set("api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &model.ValidationResult{
				Valid:   false,
				Message: "验证超时",
				Details: "连接超过30秒",
			}
		}
		return &model.ValidationResult{
			Valid:   false,
			Message: "连接失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &model.ValidationResult{
			Valid:   true,
			Message: "验证成功",
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return &model.ValidationResult{
		Valid:   false,
		Message: "验证失败",
		Details: fmt.Sprintf("状态码: %d, 响应: %s, url: %s", resp.StatusCode, string(body), url),
	}
}

// validateBianlian 验证Bianlian配置
func (s *modelConfigurationService) validateBianlian(ctx context.Context, config *model.ModelConfiguration, apiKey string) *model.ValidationResult {
	if config.BaseURL == nil || *config.BaseURL == "" {
		return &model.ValidationResult{
			Valid:   false,
			Message: "Bianlian需要配置BaseURL",
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 构建请求体
	requestBody := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "最多只允许输出10个字",
			},
			{
				"role":    "user",
				"content": "你是谁？",
			},
		},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "构建请求体失败",
			Details: err.Error(),
		}
	}

	// 使用chat/completions端点验证
	url := fmt.Sprintf("%s/chat/completions", *config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &model.ValidationResult{
				Valid:   false,
				Message: "验证超时",
				Details: "连接超过30秒",
			}
		}
		return &model.ValidationResult{
			Valid:   false,
			Message: "连接失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &model.ValidationResult{
			Valid:   true,
			Message: "验证成功",
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return &model.ValidationResult{
		Valid:   false,
		Message: "验证失败",
		Details: fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body)),
	}
}

// validateCustomOpenAI 验证自定义OpenAI配置
func (s *modelConfigurationService) validateCustomOpenAI(ctx context.Context, config *model.ModelConfiguration, apiKey string) *model.ValidationResult {
	if config.BaseURL == nil || *config.BaseURL == "" {
		return &model.ValidationResult{
			Valid:   false,
			Message: "自定义OpenAI需要配置BaseURL",
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 使用models端点验证
	url := fmt.Sprintf("%s/v1/models", *config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &model.ValidationResult{
			Valid:   false,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &model.ValidationResult{
				Valid:   false,
				Message: "验证超时",
				Details: "连接超过30秒",
			}
		}
		return &model.ValidationResult{
			Valid:   false,
			Message: "连接失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &model.ValidationResult{
			Valid:   true,
			Message: "验证成功",
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return &model.ValidationResult{
		Valid:   false,
		Message: "验证失败",
		Details: fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body)),
	}
}
