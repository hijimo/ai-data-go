# 模型配置模块设计文档

## 概述

模型配置模块为多租户AI平台提供了灵活的模型提供商连接配置管理能力。该模块允许租户管理员和平台管理员配置、验证、启用/禁用和管理不同的AI模型提供商的连接信息，确保租户数据隔离和安全性。

**重要说明**：

- 本模块的服务名称为 `ModelConfigurationService`，管理动态的模型配置（存储在数据库中）
- 系统中已存在的 `ProviderService` 用于管理静态的模型元数据（从配置文件读取）
- API 路径使用 `/api/v1/model-configurations` 以区分静态的 `/api/v1/providers`

**命名对照表**：

| 概念 | 静态模型元数据 | 动态模型配置 |
|------|--------------|------------|
| 服务名称 | `ProviderService` | `ModelConfigurationService` |
| Handler名称 | `ProviderHandler` | `ModelConfigurationHandler` |
| Repository名称 | N/A（从配置文件读取） | `ModelConfigurationRepository` |
| API路径前缀 | `/api/v1/providers` | `/api/v1/model-configurations` |
| 数据来源 | 配置文件（静态） | 数据库（动态） |
| 用途 | 提供模型元数据信息 | 管理租户的模型连接配置 |

### 设计目标

- 支持多种主流AI模型提供商（OpenAI、Anthropic、Google GenAI、Azure OpenAI等）
- 实现严格的多租户数据隔离
- 提供模型配置的验证机制
- 确保敏感信息（API密钥）的安全存储和传输
- 支持灵活的模型启用/禁用控制
- 实现完整的审计追踪

## 架构

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        API Layer                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  ModelConfiguration Handler                          │   │
│  │  (internal/api/handler)                              │   │
│  │  - HandleCreate                                       │   │
│  │  - HandleList                                         │   │
│  │  - HandleGet                                          │   │
│  │  - HandleUpdate                                       │   │
│  │  - HandleUpdateStatus                                 │   │
│  │  - HandleDelete                                       │   │
│  │  - HandleValidate                                     │   │
│  │  - HandleListAvailable                                │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Middleware Layer                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  - JWT Authentication                                 │   │
│  │  - Role-Based Access Control (RBAC)                  │   │
│  │  - Tenant Context Injection                          │   │
│  │  - Request Logging & Tracing                         │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  ModelConfiguration Service                          │   │
│  │  (internal/service)                                  │   │
│  │  - Create(ctx, req) -> ModelConfiguration            │   │
│  │  - List(ctx, tenantID, page) -> []ModelConfig, total│   │
│  │  - Get(ctx, id) -> ModelConfiguration                │   │
│  │  - Update(ctx, id, req) -> ModelConfiguration        │   │
│  │  - UpdateStatus(ctx, id, enabled) -> error           │   │
│  │  - Delete(ctx, id) -> error                          │   │
│  │  - Validate(ctx, id) -> ValidationResult             │   │
│  │  - ListAvailable(ctx) -> []ModelConfiguration        │   │
│  │                                                        │   │
│  │  Encryption Service                                   │   │
│  │  - EncryptAPIKey(plaintext) -> encrypted             │   │
│  │  - DecryptAPIKey(encrypted) -> plaintext             │   │
│  │  - MaskAPIKey(apiKey) -> masked                      │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  ModelConfiguration Repository                       │   │
│  │  (internal/repository)                               │   │
│  │  - Create(ctx, config) -> ModelConfiguration         │   │
│  │  - FindByID(ctx, id) -> ModelConfiguration           │   │
│  │  - FindByTenant(ctx, tenantID, page) -> []Config     │   │
│  │  - Update(ctx, id, config) -> ModelConfiguration     │   │
│  │  - UpdateStatus(ctx, id, enabled) -> error           │   │
│  │  - SoftDelete(ctx, id, deletedBy) -> error           │   │
│  │  - FindAvailableByTenant(ctx, tenantID) -> []Config  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Database Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  PostgreSQL (GORM)                                    │   │
│  │  - model_configurations table                        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 分层职责

#### API层（Handler）

- 接收HTTP请求并解析参数
- 调用服务层方法
- 格式化响应数据
- 处理HTTP状态码

#### 中间件层

- JWT令牌验证
- 角色权限检查（system_admin / tenant_admin）
- 租户上下文注入
- 请求日志和追踪

#### 服务层（Service）

- 实现业务逻辑
- 多租户权限验证
- API密钥加密/解密
- 模型配置验证
- 审计日志记录

#### 仓储层（Repository）

- 数据库CRUD操作
- 查询构建和优化
- 事务管理

## 组件和接口

### 数据模型

#### ModelConfiguration 结构体

```go
// ModelConfiguration 模型配置实体
type ModelConfiguration struct {
    // 主键，UUID类型
    ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    
    // 租户ID，外键关联
    TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_tenant_provider" json:"tenantId"`
    
    // 配置名称
    Name string `gorm:"type:varchar(255);not null" json:"name"`
    
    // 模型标识（如：gpt-4、claude-3-opus等）
    Model string `gorm:"type:varchar(255);not null" json:"model"`
    
    // 模型提供商枚举
    ModelProvider string `gorm:"type:varchar(50);not null;index:idx_tenant_provider" json:"modelProvider"`
    
    // API基础URL（可选，用于自定义端点）
    BaseURL *string `gorm:"type:varchar(500)" json:"baseUrl,omitempty"`
    
    // API密钥（加密存储）
    APIKey string `gorm:"type:text;not null" json:"-"`
    
    // 查询参数（JSON格式，可选）
    QueryParams *string `gorm:"type:jsonb" json:"queryParams,omitempty"`
    
    // 是否启用
    IsEnabled bool `gorm:"default:true;not null" json:"isEnabled"`
    
    // 软删除标记
    IsDeleted bool `gorm:"default:false;not null;index" json:"-"`
    
    // 创建信息
    CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"createdBy"`
    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    
    // 更新信息
    UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updatedBy,omitempty"`
    UpdatedAt *time.Time `json:"updatedAt,omitempty"`
    
    // 删除信息
    DeletedBy *uuid.UUID `gorm:"type:uuid" json:"-"`
    DeletedAt *time.Time `json:"-"`
}
```

#### ModelProvider 枚举

```go
const (
    ModelProviderOpenAI       = "openai"
    ModelProviderAnthropic    = "anthropic"
    ModelProviderGoogleGenAI  = "googlegenai"
    ModelProviderAzureOpenAI  = "azureopenai"
    ModelProviderBianlian     = "bianlian"
    ModelProviderCustomOpenAI = "custom_openai"
)

// ValidModelProviders 有效的模型提供商列表
var ValidModelProviders = []string{
    ModelProviderOpenAI,
    ModelProviderAnthropic,
    ModelProviderGoogleGenAI,
    ModelProviderAzureOpenAI,
    ModelProviderBianlian,
    ModelProviderCustomOpenAI,
}
```

### API接口定义

#### 1. 创建模型配置

**端点**: `POST /api/v1/model-configurations`

**权限**: tenant_admin, system_admin

**请求体**:

```go
type CreateModelConfigurationRequest struct {
    TenantID      *uuid.UUID `json:"tenantId,omitempty"` // 仅system_admin需要
    Name          string     `json:"name" binding:"required"`
    Model         string     `json:"model" binding:"required"`
    ModelProvider string     `json:"modelProvider" binding:"required,oneof=openai anthropic googlegenai azureopenai bianlian custom_openai"`
    BaseURL       *string    `json:"baseUrl,omitempty"`
    APIKey        string     `json:"apiKey" binding:"required"`
    QueryParams   *string    `json:"queryParams,omitempty"`
}
```

**响应**:

```go
type ModelConfigurationResponse struct {
    ID            uuid.UUID  `json:"id"`
    TenantID      uuid.UUID  `json:"tenantId"`
    Name          string     `json:"name"`
    Model         string     `json:"model"`
    ModelProvider string     `json:"modelProvider"`
    BaseURL       *string    `json:"baseUrl,omitempty"`
    APIKey        string     `json:"apiKey"` // 脱敏后的密钥
    QueryParams   *string    `json:"queryParams,omitempty"`
    IsEnabled     bool       `json:"isEnabled"`
    CreatedBy     uuid.UUID  `json:"createdBy"`
    CreatedAt     time.Time  `json:"createdAt"`
    UpdatedBy     *uuid.UUID `json:"updatedBy,omitempty"`
    UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}
```

#### 2. 查询模型配置列表

**端点**: `GET /api/v1/model-configurations`

**权限**: tenant_admin, system_admin

**查询参数**:

- `tenantId` (可选，仅system_admin): 过滤特定租户
- `pageNo` (默认: 1): 页码
- `pageSize` (默认: 10): 每页大小

**响应**: 使用 `ResponsePaginationData[[]ModelConfigurationResponse]` 格式

#### 3. 查询单个模型配置

**端点**: `GET /api/v1/model-configurations/{id}`

**权限**: tenant_admin, system_admin

**响应**: 使用 `ResponseData[ModelConfigurationResponse]` 格式

#### 4. 更新模型配置

**端点**: `PUT /api/v1/model-configurations/{id}`

**权限**: tenant_admin, system_admin

**请求体**:

```go
type UpdateModelConfigurationRequest struct {
    Name        *string `json:"name,omitempty"`
    Model       *string `json:"model,omitempty"`
    BaseURL     *string `json:"baseUrl,omitempty"`
    APIKey      *string `json:"apiKey,omitempty"`
    QueryParams *string `json:"queryParams,omitempty"`
}
```

**响应**: 使用 `ResponseData[ModelConfigurationResponse]` 格式

#### 5. 更新模型配置状态

**端点**: `PATCH /api/v1/model-configurations/{id}/status`

**权限**: tenant_admin, system_admin

**请求体**:

```go
type UpdateStatusRequest struct {
    Status string `json:"status" binding:"required,oneof=enabled disabled"`
}
```

**响应**: 使用 `ResponseData[ModelConfigurationResponse]` 格式

#### 6. 删除模型配置

**端点**: `DELETE /api/v1/model-configurations/{id}`

**权限**: tenant_admin, system_admin

**响应**: 使用 `ResponseData[interface{}]` 格式（data为null）

#### 7. 验证模型配置

**端点**: `POST /api/v1/model-configurations/{id}/validate`

**权限**: tenant_admin, system_admin

**响应**:

```go
type ValidationResult struct {
    Valid   bool   `json:"valid"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

使用 `ResponseData[ValidationResult]` 格式

#### 8. 查询可用模型列表

**端点**: `GET /api/v1/model-configurations/available`

**权限**: 所有已认证用户

**响应**:

```go
type AvailableModelConfigurationResponse struct {
    ID            uuid.UUID `json:"id"`
    Name          string    `json:"name"`
    Model         string    `json:"model"`
    ModelProvider string    `json:"modelProvider"`
}
```

使用 `ResponseData[[]AvailableModelConfigurationResponse]` 格式

## 核心业务逻辑

### 权限控制逻辑

#### 租户管理员（tenant_admin）

```go
// 创建模型配置
func (s *ModelConfigurationService) Create(ctx context.Context, req CreateModelConfigurationRequest) (*ModelConfiguration, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能在自己的租户下创建
    if hasRole(claims, model.RoleTenantAdmin) {
        if req.TenantID != nil && *req.TenantID != claims.TenantID {
            return nil, errors.NewForbiddenError("权限不足：只能在当前租户下创建模型配置")
        }
        req.TenantID = &claims.TenantID
    }
    
    // 平台管理员必须指定租户ID
    if hasRole(claims, model.RoleSystemAdmin) {
        if req.TenantID == nil {
            return nil, errors.NewBadRequestError("平台管理员必须指定租户ID")
        }
    }
    
    // 验证模型提供商
    if !isValidProvider(req.ModelProvider) {
        return nil, errors.NewBadRequestError("无效的模型提供商")
    }
    
    // 加密API密钥
    encryptedKey, err := s.encryptionService.EncryptAPIKey(req.APIKey)
    if err != nil {
        return nil, errors.NewInternalError("加密API密钥失败")
    }
    
    config := &ModelConfiguration{
        TenantID:      *req.TenantID,
        Name:          req.Name,
        Model:         req.Model,
        ModelProvider: req.ModelProvider,
        BaseURL:       req.BaseURL,
        APIKey:        encryptedKey,
        QueryParams:   req.QueryParams,
        IsEnabled:     true,
        CreatedBy:     claims.Subject,
    }
    
    // 记录审计日志
    logger.InfoContext(ctx, "创建模型配置",
        "event", "provider_created",
        "user_id", claims.Subject,
        "tenant_id", *req.TenantID,
        "provider", req.ModelProvider,
    )
    
    return s.repo.Create(ctx, config)
}

// 查询模型配置
func (s *ModelConfigurationService) Get(ctx context.Context, id uuid.UUID) (*ModelConfiguration, error) {
    config, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能访问自己租户的配置
    if hasRole(claims, model.RoleTenantAdmin) {
        if config.TenantID != claims.TenantID {
            logger.WarnContext(ctx, "权限验证失败",
                "event", "permission_denied",
                "reason", "尝试访问其他租户的模型配置",
                "user_id", claims.Subject,
                "user_tenant_id", claims.TenantID,
                "target_config_id", id,
                "target_tenant_id", config.TenantID,
            )
            return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的模型配置")
        }
    }
    
    // 脱敏API密钥
    config.APIKey = s.encryptionService.MaskAPIKey(config.APIKey)
    
    return config, nil
}

// 更新模型配置
func (s *ModelConfigurationService) Update(ctx context.Context, id uuid.UUID, req UpdateModelConfigurationRequest) (*ModelConfiguration, error) {
    // 先查询配置
    config, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能更新自己租户的配置
    if hasRole(claims, model.RoleTenantAdmin) {
        if config.TenantID != claims.TenantID {
            logger.WarnContext(ctx, "权限验证失败",
                "event", "permission_denied",
                "reason", "尝试更新其他租户的模型配置",
                "user_id", claims.Subject,
                "user_tenant_id", claims.TenantID,
                "target_config_id", id,
                "target_tenant_id", config.TenantID,
            )
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
        encryptedKey, err := s.encryptionService.EncryptAPIKey(*req.APIKey)
        if err != nil {
            return nil, errors.NewInternalError("加密API密钥失败")
        }
        config.APIKey = encryptedKey
    }
    if req.QueryParams != nil {
        config.QueryParams = req.QueryParams
    }
    
    now := time.Now()
    config.UpdatedBy = &claims.Subject
    config.UpdatedAt = &now
    
    // 记录审计日志
    logger.InfoContext(ctx, "更新模型配置",
        "event", "provider_updated",
        "user_id", claims.Subject,
        "tenant_id", config.TenantID,
        "config_id", id,
    )
    
    return s.repo.Update(ctx, id, config)
}

// 列表查询
func (s *ModelConfigurationService) List(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]ModelConfiguration, int64, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能查看自己租户的配置
    if hasRole(claims, model.RoleTenantAdmin) {
        currentTenantID := claims.TenantID
        configs, total, err := s.repo.FindByTenant(ctx, &currentTenantID, pageNo, pageSize)
        if err != nil {
            return nil, 0, err
        }
        
        // 脱敏所有配置的API密钥
        for i := range configs {
            configs[i].APIKey = s.encryptionService.MaskAPIKey(configs[i].APIKey)
        }
        
        return configs, total, nil
    }
    
    // 平台管理员可以查看所有或指定租户的配置
    if hasRole(claims, model.RoleSystemAdmin) {
        configs, total, err := s.repo.FindByTenant(ctx, tenantID, pageNo, pageSize)
        if err != nil {
            return nil, 0, err
        }
        
        // 脱敏所有配置的API密钥
        for i := range configs {
            configs[i].APIKey = s.encryptionService.MaskAPIKey(configs[i].APIKey)
        }
        
        return configs, total, nil
    }
    
    return nil, 0, errors.NewForbiddenError("权限不足")
}
```

### API密钥加密逻辑

```go
// EncryptionService API密钥加密服务
type EncryptionService interface {
    EncryptAPIKey(plaintext string) (string, error)
    DecryptAPIKey(encrypted string) (string, error)
    MaskAPIKey(apiKey string) string
}

type encryptionServiceImpl struct {
    secretKey []byte
}

// EncryptAPIKey 使用AES-256-GCM加密API密钥
func (s *encryptionServiceImpl) EncryptAPIKey(plaintext string) (string, error) {
    block, err := aes.NewCipher(s.secretKey)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey 解密API密钥
func (s *encryptionServiceImpl) DecryptAPIKey(encrypted string) (string, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(s.secretKey)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", errors.New("密文太短")
    }
    
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}

// MaskAPIKey 脱敏API密钥（显示前4位和后4位）
func (s *encryptionServiceImpl) MaskAPIKey(apiKey string) string {
    if len(apiKey) <= 8 {
        return "****"
    }
    return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}
```

### 模型配置验证逻辑

```go
// Validate 验证模型配置是否可以正确连接
func (s *ModelConfigurationService) Validate(ctx context.Context, id uuid.UUID) (*ValidationResult, error) {
    // 查询配置
    config, err := s.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 解密API密钥
    apiKey, err := s.encryptionService.DecryptAPIKey(config.APIKey)
    if err != nil {
        return &ValidationResult{
            Valid:   false,
            Message: "解密API密钥失败",
        }, nil
    }
    
    // 设置30秒超时
    validateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // 根据不同的提供商进行验证
    switch config.ModelProvider {
    case ModelProviderOpenAI:
        return s.validateOpenAI(validateCtx, config, apiKey)
    case ModelProviderAnthropic:
        return s.validateAnthropic(validateCtx, config, apiKey)
    case ModelProviderGoogleGenAI:
        return s.validateGoogleGenAI(validateCtx, config, apiKey)
    case ModelProviderAzureOpenAI:
        return s.validateAzureOpenAI(validateCtx, config, apiKey)
    case ModelProviderBianlian:
        return s.validateBianlian(validateCtx, config, apiKey)
    case ModelProviderCustomOpenAI:
        return s.validateCustomOpenAI(validateCtx, config, apiKey)
    default:
        return &ValidationResult{
            Valid:   false,
            Message: "不支持的模型提供商",
        }, nil
    }
}

// validateOpenAI 验证OpenAI配置
func (s *ModelConfigurationService) validateOpenAI(ctx context.Context, config *ModelConfiguration, apiKey string) (*ValidationResult, error) {
    client := &http.Client{Timeout: 30 * time.Second}
    
    baseURL := "https://api.openai.com/v1"
    if config.BaseURL != nil {
        baseURL = *config.BaseURL
    }
    
    req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
    if err != nil {
        return &ValidationResult{
            Valid:   false,
            Message: "创建请求失败",
            Details: err.Error(),
        }, nil
    }
    
    req.Header.Set("Authorization", "Bearer "+apiKey)
    
    resp, err := client.Do(req)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return &ValidationResult{
                Valid:   false,
                Message: "验证超时",
                Details: "连接超过30秒",
            }, nil
        }
        return &ValidationResult{
            Valid:   false,
            Message: "连接失败",
            Details: err.Error(),
        }, nil
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusOK {
        return &ValidationResult{
            Valid:   true,
            Message: "验证成功",
        }, nil
    }
    
    body, _ := io.ReadAll(resp.Body)
    return &ValidationResult{
        Valid:   false,
        Message: "验证失败",
        Details: fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body)),
    }, nil
}
```

### 可用模型列表逻辑

```go
// ListAvailable 获取当前租户下所有可用的模型配置
func (s *ModelConfigurationService) ListAvailable(ctx context.Context) ([]ModelConfiguration, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 查询当前租户下已启用且未删除的配置
    configs, err := s.repo.FindAvailableByTenant(ctx, claims.TenantID)
    if err != nil {
        return nil, err
    }
    
    // 移除敏感信息
    result := make([]ModelConfiguration, len(configs))
    for i, config := range configs {
        result[i] = ModelConfiguration{
            ID:            config.ID,
            Name:          config.Name,
            Model:         config.Model,
            ModelProvider: config.ModelProvider,
        }
    }
    
    return result, nil
}
```

## 数据库设计

### 表结构

```sql
CREATE TABLE model_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    model_provider VARCHAR(50) NOT NULL,
    base_url VARCHAR(500),
    api_key TEXT NOT NULL,
    query_params JSONB,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMP,
    deleted_by UUID REFERENCES users(id),
    deleted_at TIMESTAMP,
    
    CONSTRAINT chk_model_provider CHECK (
        model_provider IN ('openai', 'anthropic', 'googlegenai', 'azureopenai', 'bianlian', 'custom_openai')
    )
);

-- 索引
CREATE INDEX idx_model_configs_tenant_provider ON model_configurations(tenant_id, model_provider);
CREATE INDEX idx_model_configs_deleted ON model_configurations(is_deleted);
CREATE INDEX idx_model_configs_enabled ON model_configurations(is_enabled) WHERE is_deleted = false;
```

### 数据迁移

```go
// ModelConfigurationMigration 模型配置表迁移
type ModelConfigurationMigration struct {
    db *gorm.DB
}

func (m *ModelConfigurationMigration) Up() error {
    return m.db.Exec(`
        CREATE TABLE IF NOT EXISTS model_configurations (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            tenant_id UUID NOT NULL REFERENCES tenants(id),
            name VARCHAR(255) NOT NULL,
            model VARCHAR(255) NOT NULL,
            model_provider VARCHAR(50) NOT NULL,
            base_url VARCHAR(500),
            api_key TEXT NOT NULL,
            query_params JSONB,
            is_enabled BOOLEAN NOT NULL DEFAULT true,
            is_deleted BOOLEAN NOT NULL DEFAULT false,
            created_by UUID NOT NULL REFERENCES users(id),
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_by UUID REFERENCES users(id),
            updated_at TIMESTAMP,
            deleted_by UUID REFERENCES users(id),
            deleted_at TIMESTAMP,
            
            CONSTRAINT chk_model_provider CHECK (
                model_provider IN ('openai', 'anthropic', 'googlegenai', 'azureopenai', 'bianlian', 'custom_openai')
            )
        );
        
        CREATE INDEX IF NOT EXISTS idx_model_configs_tenant_provider 
            ON model_configurations(tenant_id, model_provider);
        CREATE INDEX IF NOT EXISTS idx_model_configs_deleted 
            ON model_configurations(is_deleted);
        CREATE INDEX IF NOT EXISTS idx_model_configs_enabled 
            ON model_configurations(is_enabled) WHERE is_deleted = false;
    `).Error
}

func (m *ModelConfigurationMigration) Down() error {
    return m.db.Exec(`
        DROP TABLE IF EXISTS model_configurations;
    `).Error
}
```

## 错误处理

### 错误类型定义

```go
var (
    // ErrProviderNotFound 模型配置不存在
    ErrProviderNotFound = errors.NewNotFoundError("模型配置不存在")
    
    // ErrInvalidProvider 无效的模型提供商
    ErrInvalidProvider = errors.NewBadRequestError("无效的模型提供商")
    
    // ErrProviderAccessDenied 无权访问该模型配置
    ErrProviderAccessDenied = errors.NewForbiddenError("权限不足：无法访问该模型配置")
    
    // ErrProviderValidationFailed 模型配置验证失败
    ErrProviderValidationFailed = errors.NewBadRequestError("模型配置验证失败")
    
    // ErrEncryptionFailed API密钥加密失败
    ErrEncryptionFailed = errors.NewInternalError("API密钥加密失败")
    
    // ErrDecryptionFailed API密钥解密失败
    ErrDecryptionFailed = errors.NewInternalError("API密钥解密失败")
)
```

### 错误处理策略

1. **权限错误**: 返回403 Forbidden，记录审计日志
2. **资源不存在**: 返回404 Not Found
3. **验证错误**: 返回400 Bad Request，包含详细错误信息
4. **加密/解密错误**: 返回500 Internal Server Error，记录错误日志
5. **超时错误**: 返回408 Request Timeout

### 错误响应格式

```go
// 使用标准响应格式
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    *struct{} `json:"data,omitempty"`
}

// 示例
{
    "code": 403,
    "message": "权限不足：无法访问其他租户的模型配置",
    "data": null
}
```

## 测试策略

### 单元测试

#### Service层测试

```go
func TestModelConfigurationService_Create(t *testing.T) {
    tests := []struct {
        name    string
        role    string
        req     CreateModelConfigurationRequest
        wantErr bool
        errType error
    }{
        {
            name: "租户管理员创建配置成功",
            role: model.RoleTenantAdmin,
            req: CreateModelConfigurationRequest{
                Name:          "OpenAI GPT-4",
                Model:         "gpt-4",
                ModelProvider: ModelProviderOpenAI,
                APIKey:        "sk-test123",
            },
            wantErr: false,
        },
        {
            name: "租户管理员尝试在其他租户创建配置失败",
            role: model.RoleTenantAdmin,
            req: CreateModelConfigurationRequest{
                TenantID:      &otherTenantID,
                Name:          "OpenAI GPT-4",
                Model:         "gpt-4",
                ModelProvider: ModelProviderOpenAI,
                APIKey:        "sk-test123",
            },
            wantErr: true,
            errType: ErrProviderAccessDenied,
        },
        {
            name: "平台管理员创建配置成功",
            role: model.RoleSystemAdmin,
            req: CreateModelConfigurationRequest{
                TenantID:      &someTenantID,
                Name:          "OpenAI GPT-4",
                Model:         "gpt-4",
                ModelProvider: ModelProviderOpenAI,
                APIKey:        "sk-test123",
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试实现
        })
    }
}
```

#### Repository层测试

```go
func TestModelConfigurationRepository_FindByTenant(t *testing.T) {
    // 准备测试数据
    // 执行查询
    // 验证结果
}

func TestModelConfigurationRepository_SoftDelete(t *testing.T) {
    // 测试软删除逻辑
}
```

### 集成测试

```go
func TestModelConfigurationAPI_Integration(t *testing.T) {
    // 设置测试环境
    // 测试完整的API流程
    // 验证数据库状态
}
```

### 测试覆盖目标

- 单元测试覆盖率: ≥ 80%
- 集成测试覆盖所有API端点
- 权限测试覆盖所有角色组合

## 安全考虑

### API密钥安全

1. **加密存储**: 使用AES-256-GCM加密算法
2. **传输安全**: 仅通过HTTPS传输
3. **脱敏显示**: 返回时仅显示前4位和后4位
4. **日志排除**: 审计日志中不记录完整密钥

### 多租户隔离

1. **服务层验证**: 所有操作都在服务层验证租户权限
2. **数据库隔离**: 查询时强制添加租户ID过滤
3. **审计日志**: 记录所有跨租户访问尝试

### 输入验证

1. **模型提供商**: 限制为预定义枚举值
2. **URL验证**: 验证baseURL格式
3. **JSON验证**: 验证queryParams的JSON格式
4. **长度限制**: 限制字符串字段长度

## 性能优化

### 数据库优化

1. **索引策略**:
   - 复合索引: (tenant_id, model_provider)
   - 单列索引: is_deleted, is_enabled
   - 部分索引: is_enabled WHERE is_deleted = false

2. **查询优化**:
   - 使用预编译语句
   - 避免N+1查询
   - 合理使用分页

### 缓存策略

```go
// 可用模型列表缓存（可选）
type CachedModelConfigurationService struct {
    service ModelConfigurationService
    cache   cache.Cache
}

func (s *CachedModelConfigurationService) ListAvailable(ctx context.Context) ([]ModelConfiguration, error) {
    claims := middleware.GetJWTClaims(ctx)
    cacheKey := fmt.Sprintf("available_providers:%s", claims.TenantID)
    
    // 尝试从缓存获取
    var configs []ModelConfiguration
    if err := s.cache.Get(cacheKey, &configs); err == nil {
        return configs, nil
    }
    
    // 缓存未命中，查询数据库
    configs, err := s.service.ListAvailable(ctx)
    if err != nil {
        return nil, err
    }
    
    // 缓存结果（5分钟）
    s.cache.Set(cacheKey, configs, 5*time.Minute)
    
    return configs, nil
}
```

## 监控和日志

### 关键指标

1. **API性能指标**:
   - 请求响应时间
   - 请求成功率
   - 并发请求数

2. **业务指标**:
   - 模型配置创建数量
   - 验证成功/失败率
   - 各提供商使用分布

3. **安全指标**:
   - 权限验证失败次数
   - 跨租户访问尝试次数

### 日志记录

```go
// 操作日志
logger.InfoContext(ctx, "模型配置操作",
    "event", "provider_created",
    "user_id", userID,
    "tenant_id", tenantID,
    "provider", modelProvider,
    "config_id", configID,
)

// 权限失败日志
logger.WarnContext(ctx, "权限验证失败",
    "event", "permission_denied",
    "reason", "尝试访问其他租户的模型配置",
    "user_id", userID,
    "user_tenant_id", userTenantID,
    "target_config_id", configID,
    "target_tenant_id", targetTenantID,
)

// 验证失败日志
logger.ErrorContext(ctx, "模型配置验证失败",
    "event", "validation_failed",
    "config_id", configID,
    "provider", modelProvider,
    "error", err.Error(),
)
```

## 部署考虑

### 环境变量

```bash
# API密钥加密密钥（32字节）
ENCRYPTION_SECRET_KEY=your-32-byte-secret-key-here

# 数据库连接
DATABASE_URL=postgresql://user:pass@localhost:5432/dbname

# 验证超时时间（秒）
PROVIDER_VALIDATION_TIMEOUT=30
```

### 配置管理

```go
type ModelConfigurationConfig struct {
    EncryptionKey        string        `env:"ENCRYPTION_SECRET_KEY,required"`
    ValidationTimeout    time.Duration `env:"PROVIDER_VALIDATION_TIMEOUT" envDefault:"30s"`
    CacheEnabled         bool          `env:"PROVIDER_CACHE_ENABLED" envDefault:"true"`
    CacheTTL             time.Duration `env:"PROVIDER_CACHE_TTL" envDefault:"5m"`
}
```

## 未来扩展

### 可能的增强功能

1. **模型配置模板**: 预定义常用模型配置模板
2. **批量导入/导出**: 支持批量管理模型配置
3. **配置版本控制**: 跟踪配置变更历史
4. **使用统计**: 记录各模型的使用频率和成本
5. **自动故障转移**: 主模型不可用时自动切换到备用模型
6. **配额管理**: 限制每个租户的模型配置数量
7. **健康检查**: 定期自动验证模型配置的可用性

### 技术债务

1. 考虑使用专门的密钥管理服务（如AWS KMS、HashiCorp Vault）
2. 实现更细粒度的权限控制（如只读权限）
3. 添加配置变更通知机制
4. 实现配置的审批流程（可选）
