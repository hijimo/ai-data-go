# 设计文档：Genkit 多模型支持

## 设计概述

本设计通过扩展现有的 Genkit 客户端，支持多个 AI 模型提供商。核心思路是从 model_configuration 表中根据租户ID和模型名称动态获取模型配置，保持 Genkit 框架作为统一的抽象层，通过插件机制集成不同的模型提供商，确保上层服务代码无需修改。

## 架构设计

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    API Handler 层                            │
│              (message_handler.go)                           │
│              - 处理 HTTP 请求                                │
│              - 参数验证                                      │
│              - 响应格式化                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                  Service 层                                  │
│           (message_service.go)                              │
│           - 业务逻辑处理                                     │
│           - 数据库操作                                       │
│           - 调用 AI Service                                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                 AI Service 层                                │
│            (genkit_service.go)                              │
│            - 会话管理                                        │
│            - 格式转换                                        │
│            - 调用 Genkit Client                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│              Genkit Client 层 (扩展)                         │
│                (client.go)                                  │
│         ┌──────────────────────────────────┐                │
│         │  配置查询服务                     │                │
│         │  - 根据租户ID+模型名查询配置      │                │
│         │  - 验证配置                       │                │
│         │  - 缓存配置                       │                │
│         └──────────────────────────────────┘                │
│                      ↓                                       │
│         ┌──────────────────────────────────┐                │
│         │  Plugin 初始化器                  │                │
│         │  - Google AI Plugin               │                │
│         │  - Azure OpenAI Plugin            │                │
│         │  - Bailian Plugin                 │                │
│         └──────────────────────────────────┘                │
└─────────────────────────────────────────────────────────────┘
                            ↑
┌─────────────────────────────────────────────────────────────┐
│                  数据库层                                    │
│           (model_configuration 表)                          │
│           - tenant_id (租户ID)                              │
│           - model_name (模型名称)                           │
│           - provider_type (提供商类型)                      │
│           - api_key (API密钥)                               │
│           - configuration (JSON配置)                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┼───────────────────┐
        ↓                   ↓                   ↓
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Google AI    │    │ Azure OpenAI │    │   百炼       │
│   Plugin     │    │   Plugin     │    │  Plugin      │
│ (googlegenai)│    │ (azureopenai)│    │ (custom)     │
└──────────────┘    └──────────────┘    └──────────────┘
        ↓                   ↓                   ↓
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Gemini API   │    │ Azure OpenAI │    │ 百炼 API     │
└──────────────┘    └──────────────┘    └──────────────┘
```

### 模块设计

#### 1. 数据库模型 (model_configuration.go)

**职责**：

- 定义模型配置表结构
- 提供配置查询方法
- 支持配置缓存

**数据结构**：

```go
// ProviderType 提供商类型
type ProviderType string

const (
    ProviderGoogleAI    ProviderType = "google"
    ProviderAzureOpenAI ProviderType = "azure"
    ProviderBailian     ProviderType = "bailian"
)

// ModelConfiguration 模型配置表
type ModelConfiguration struct {
    ID            uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID      uuid.UUID       `gorm:"type:uuid;not null;index:idx_tenant_model" json:"tenantId"`
    ModelName     string          `gorm:"type:varchar(100);not null;index:idx_tenant_model" json:"modelName"`
    ProviderType  ProviderType    `gorm:"type:varchar(50);not null" json:"providerType"`
    APIKey        string          `gorm:"type:text;not null" json:"-"` // 不在JSON中返回
    Configuration datatypes.JSON  `gorm:"type:jsonb" json:"configuration"`
    IsEnabled     bool            `gorm:"default:true" json:"isEnabled"`
    CreatedAt     time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt     time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    IsDeleted     bool            `gorm:"default:false" json:"-"`
}

// ModelConfig 模型配置（从 Configuration JSON 解析）
type ModelConfig struct {
    // Azure 特定配置
    AzureEndpoint   string  `json:"azureEndpoint,omitempty"`
    AzureDeployment string  `json:"azureDeployment,omitempty"`
    AzureAPIVersion string  `json:"azureApiVersion,omitempty"`
    
    // 百炼特定配置
    BailianEndpoint  string `json:"bailianEndpoint,omitempty"`
    BailianWorkspace string `json:"bailianWorkspace,omitempty"`
    
    // 通用配置
    Model              string  `json:"model"`
    DefaultTemperature float64 `json:"defaultTemperature"`
    DefaultMaxTokens   int     `json:"defaultMaxTokens"`
}

// TableName 指定表名
func (ModelConfiguration) TableName() string {
    return "model_configurations"
}
```

#### 2. 配置仓储模块 (model_configuration_repository.go)

**职责**：

- 查询模型配置
- 支持租户隔离
- 提供配置缓存

**核心方法**：

```go
type ModelConfigurationRepository interface {
    // GetByTenantAndModel 根据租户ID和模型名称获取配置
    GetByTenantAndModel(ctx context.Context, tenantID uuid.UUID, modelName string) (*ModelConfiguration, error)
    
    // ListByTenant 获取租户的所有模型配置
    ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ModelConfiguration, error)
    
    // Create 创建模型配置
    Create(ctx context.Context, config *ModelConfiguration) error
    
    // Update 更新模型配置
    Update(ctx context.Context, config *ModelConfiguration) error
    
    // Delete 删除模型配置（软删除）
    Delete(ctx context.Context, id uuid.UUID) error
}
```

#### 3. 客户端模块 (client.go)

**职责**：

- 管理 Genkit 实例
- 根据配置初始化插件
- 提供统一的生成接口

**核心方法**：

```go
type Client interface {
    // Generate 生成内容（根据租户ID和模型名称）
    Generate(ctx context.Context, tenantID uuid.UUID, modelName string, prompt string, options *GenerateOptions) (*GenerateResult, error)
    
    // GenerateStream 流式生成（根据租户ID和模型名称）
    GenerateStream(ctx context.Context, tenantID uuid.UUID, modelName string, prompt string, options *GenerateOptions) (<-chan StreamChunk, error)
    
    // Close 关闭客户端
    Close() error
}
```

**实现细节**：

```go
type client struct {
    configRepo ModelConfigurationRepository
    instances  map[string]*genkit.Genkit  // 缓存 Genkit 实例，key: tenantID_modelName
    mu         sync.RWMutex
}

// Generate 生成内容
func (c *client) Generate(ctx context.Context, tenantID uuid.UUID, modelName string, prompt string, options *GenerateOptions) (*GenerateResult, error) {
    // 获取或初始化 Genkit 实例
    g, modelConfig, err := c.getOrInitGenkit(ctx, tenantID, modelName)
    if err != nil {
        return nil, err
    }
    
    // 调用 Genkit 生成
    return c.generate(ctx, g, modelConfig, prompt, options)
}

// getOrInitGenkit 获取或初始化 Genkit 实例
func (c *client) getOrInitGenkit(ctx context.Context, tenantID uuid.UUID, modelName string) (*genkit.Genkit, *ModelConfiguration, error) {
    cacheKey := fmt.Sprintf("%s_%s", tenantID.String(), modelName)
    
    // 尝试从缓存获取
    c.mu.RLock()
    g, exists := c.instances[cacheKey]
    c.mu.RUnlock()
    
    if exists {
        // 从数据库获取配置（用于参数）
        config, err := c.configRepo.GetByTenantAndModel(ctx, tenantID, modelName)
        if err != nil {
            return nil, nil, fmt.Errorf("获取模型配置失败: %w", err)
        }
        return g, config, nil
    }
    
    // 从数据库查询配置
    config, err := c.configRepo.GetByTenantAndModel(ctx, tenantID, modelName)
    if err != nil {
        return nil, nil, fmt.Errorf("获取模型配置失败: %w", err)
    }
    
    if !config.IsEnabled {
        return nil, nil, fmt.Errorf("模型已禁用: %s", modelName)
    }
    
    // 解析配置
    var modelConfig ModelConfig
    if err := json.Unmarshal(config.Configuration, &modelConfig); err != nil {
        return nil, nil, fmt.Errorf("解析模型配置失败: %w", err)
    }
    
    // 根据类型创建插件
    var plugin genkit.Plugin
    var fullModelName string
    
    switch config.ProviderType {
    case ProviderGoogleAI:
        plugin = &googlegenai.GoogleAI{
            APIKey: config.APIKey,
        }
        fullModelName = "googleai/" + modelConfig.Model
        
    case ProviderAzureOpenAI:
        plugin = createAzurePlugin(config.APIKey, &modelConfig)
        fullModelName = "azureopenai/" + modelConfig.Model
        
    case ProviderBailian:
        plugin = createBailianPlugin(config.APIKey, &modelConfig)
        fullModelName = "bailian/" + modelConfig.Model
        
    default:
        return nil, nil, fmt.Errorf("不支持的提供商类型: %s", config.ProviderType)
    }
    
    // 初始化 Genkit 实例
    g = genkit.Init(ctx,
        genkit.WithPlugins(plugin),
        genkit.WithDefaultModel(fullModelName),
    )
    
    // 缓存实例
    c.mu.Lock()
    c.instances[cacheKey] = g
    c.mu.Unlock()
    
    return g, config, nil
}
```

#### 3. Azure OpenAI 插件模块

**方案 A：使用官方插件（如果存在）**

```go
import "github.com/firebase/genkit/go/plugins/azureopenai"

func createAzurePlugin(config *Config) genkit.Plugin {
    return &azureopenai.AzureOpenAI{
        APIKey:     config.APIKey,
        Endpoint:   config.AzureEndpoint,
        Deployment: config.AzureDeployment,
        APIVersion: config.AzureAPIVersion,
    }
}
```

**方案 B：使用 OpenAI 插件 + 自定义 BaseURL**

```go
import "github.com/firebase/genkit/go/plugins/openai"

func createAzurePlugin(config *Config) genkit.Plugin {
    baseURL := fmt.Sprintf("%s/openai/deployments/%s",
        config.AzureEndpoint,
        config.AzureDeployment,
    )
    
    return &openai.OpenAI{
        APIKey:  config.APIKey,
        BaseURL: baseURL,
    }
}
```

**方案 C：自定义插件（如果以上方案都不可行）**

```go
// internal/genkit/plugins/azure/azure.go
package azure

import (
    "context"
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
)

type AzureOpenAIPlugin struct {
    client     *azopenai.Client
    deployment string
}

func (p *AzureOpenAIPlugin) Init(ctx context.Context, g *genkit.Genkit) error {
    // 注册模型
    ai.DefineModel(g, "azureopenai", p.deployment, 
        &ai.ModelCapabilities{
            Multiturn:  true,
            SystemRole: true,
            Media:      false,
        },
        p.generate,
    )
    return nil
}

func (p *AzureOpenAIPlugin) generate(
    ctx context.Context,
    req *ai.ModelRequest,
    cb ai.ModelStreamingCallback,
) (*ai.ModelResponse, error) {
    // 实现 Azure OpenAI 调用逻辑
    // ...
}
```

#### 4. 百炼插件模块

**实现策略**：

1. **检查百炼是否支持 OpenAI 兼容接口**
   - 如果支持，使用 OpenAI 插件
   - 如果不支持，自定义插件

2. **自定义百炼插件**

```go
// internal/genkit/plugins/bailian/bailian.go
package bailian

import (
    "context"
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
)

type BailianPlugin struct {
    APIKey    string
    Endpoint  string
    Workspace string
    client    *bailianClient  // 百炼 SDK 客户端
}

func (p *BailianPlugin) Init(ctx context.Context, g *genkit.Genkit) error {
    // 初始化百炼客户端
    p.client = newBailianClient(p.APIKey, p.Endpoint)
    
    // 注册模型
    ai.DefineModel(g, "bailian", "qwen-turbo",
        &ai.ModelCapabilities{
            Multiturn:  true,
            SystemRole: true,
            Media:      false,
        },
        p.generate,
    )
    
    return nil
}

func (p *BailianPlugin) generate(
    ctx context.Context,
    req *ai.ModelRequest,
    cb ai.ModelStreamingCallback,
) (*ai.ModelResponse, error) {
    // 转换请求格式
    bailianReq := p.convertRequest(req)
    
    // 如果有流式回调
    if cb != nil {
        return p.generateStream(ctx, bailianReq, cb)
    }
    
    // 非流式调用
    resp, err := p.client.Generate(ctx, bailianReq)
    if err != nil {
        return nil, err
    }
    
    // 转换响应格式
    return p.convertResponse(resp), nil
}

func (p *BailianPlugin) generateStream(
    ctx context.Context,
    req *bailianRequest,
    cb ai.ModelStreamingCallback,
) (*ai.ModelResponse, error) {
    stream, err := p.client.GenerateStream(ctx, req)
    if err != nil {
        return nil, err
    }
    
    var fullText string
    for chunk := range stream {
        fullText += chunk.Text
        
        // 调用流式回调
        if err := cb(ctx, &ai.ModelResponseChunk{
            Content: []*ai.Part{ai.NewTextPart(chunk.Text)},
        }); err != nil {
            return nil, err
        }
    }
    
    return &ai.ModelResponse{
        Message: &ai.Message{
            Content: []*ai.Part{ai.NewTextPart(fullText)},
        },
    }, nil
}
```

#### 5. 配置服务模块

```go
// internal/service/model_configuration_service.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
)

type ModelConfigurationService struct {
    repo repository.ModelConfigurationRepository
}

func NewModelConfigurationService(repo repository.ModelConfigurationRepository) *ModelConfigurationService {
    return &ModelConfigurationService{
        repo: repo,
    }
}

// GetModelConfig 获取模型配置（带权限验证）
func (s *ModelConfigurationService) GetModelConfig(ctx context.Context, tenantID uuid.UUID, modelName string) (*model.ModelConfiguration, error) {
    // 从上下文获取当前用户信息
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能查询自己租户的配置
    if !hasRole(claims, model.RoleSystemAdmin) && claims.TenantID != tenantID.String() {
        return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的模型配置")
    }
    
    // 查询配置
    config, err := s.repo.GetByTenantAndModel(ctx, tenantID, modelName)
    if err != nil {
        return nil, fmt.Errorf("查询模型配置失败: %w", err)
    }
    
    if !config.IsEnabled {
        return nil, errors.NewBadRequestError("模型已禁用")
    }
    
    return config, nil
}

// CreateModelConfig 创建模型配置
func (s *ModelConfigurationService) CreateModelConfig(ctx context.Context, req *CreateModelConfigRequest) (*model.ModelConfiguration, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能为自己的租户创建配置
    if !hasRole(claims, model.RoleSystemAdmin) {
        if req.TenantID != claims.TenantID {
            return nil, errors.NewForbiddenError("权限不足：只能为当前租户创建模型配置")
        }
    }
    
    // 验证配置
    if err := s.validateConfig(req); err != nil {
        return nil, err
    }
    
    // 创建配置
    config := &model.ModelConfiguration{
        TenantID:      uuid.MustParse(req.TenantID),
        ModelName:     req.ModelName,
        ProviderType:  model.ProviderType(req.ProviderType),
        APIKey:        req.APIKey,
        Configuration: req.Configuration,
        IsEnabled:     true,
    }
    
    if err := s.repo.Create(ctx, config); err != nil {
        return nil, fmt.Errorf("创建模型配置失败: %w", err)
    }
    
    return config, nil
}
```

## 数据流设计

### 非流式调用流程

```
用户请求
  ↓
Handler: 解析请求参数（包括可选的 provider 参数）
  ↓
Service: 验证会话权限
  ↓
Service: 保存用户消息
  ↓
AI Service: 获取或创建会话上下文
  ↓
AI Service: 调用 Genkit Client.Generate()
  ↓
Genkit Client: 选择提供商（请求指定 or 默认）
  ↓
Genkit Client: 获取对应的 Genkit 实例
  ↓
Genkit Client: 调用 genkit.Generate()
  ↓
Plugin: 调用对应的 AI API
  ↓
Plugin: 返回响应
  ↓
Genkit Client: 解析响应
  ↓
AI Service: 转换为统一格式
  ↓
Service: 保存 AI 消息
  ↓
Service: 更新会话信息
  ↓
Handler: 返回响应给用户
```

### 流式调用流程

```
用户请求
  ↓
Handler: 解析请求参数
  ↓
Handler: 设置 SSE 响应头
  ↓
Service: 验证会话权限
  ↓
Service: 保存用户消息
  ↓
Service: 创建 AI 消息记录（空内容）
  ↓
AI Service: 调用 Genkit Client.GenerateStream()
  ↓
Genkit Client: 选择提供商
  ↓
Genkit Client: 调用 genkit.Generate() with streaming callback
  ↓
Plugin: 开始流式调用 AI API
  ↓
[循环] Plugin: 接收响应块
  ↓
[循环] Plugin: 调用 streaming callback
  ↓
[循环] Genkit Client: 发送到 StreamChunk channel
  ↓
[循环] AI Service: 转换为腾讯云格式
  ↓
[循环] AI Service: 累积完整内容
  ↓
[循环] Service: 发送到输出 channel
  ↓
[循环] Handler: 写入 SSE 响应
  ↓
[循环] Handler: Flush 缓冲区
  ↓
Plugin: 流式结束
  ↓
Service: 更新 AI 消息内容
  ↓
Service: 更新会话信息
  ↓
Handler: 关闭 SSE 连接
```

## 接口设计

### 数据库表结构

```sql
-- 模型配置表
CREATE TABLE model_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    api_key TEXT NOT NULL,
    configuration JSONB,
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false,
    
    -- 联合唯一索引：同一租户下模型名称唯一
    CONSTRAINT uk_tenant_model UNIQUE (tenant_id, model_name, is_deleted),
    
    -- 索引
    INDEX idx_tenant_model (tenant_id, model_name),
    INDEX idx_tenant_enabled (tenant_id, is_enabled),
    
    -- 外键
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 配置示例数据
-- Google AI (Gemini)
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) VALUES
('tenant-uuid-1', 'gemini-pro', 'google', 'your-google-api-key', '{
    "model": "gemini-1.5-pro",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}');

-- Azure OpenAI
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) VALUES
('tenant-uuid-1', 'gpt-4', 'azure', 'your-azure-api-key', '{
    "model": "gpt-4",
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}');

-- 阿里云百炼
INSERT INTO model_configurations (tenant_id, model_name, provider_type, api_key, configuration) VALUES
('tenant-uuid-1', 'qwen-turbo', 'bailian', 'your-bailian-api-key', '{
    "model": "qwen-turbo",
    "bailianEndpoint": "https://dashscope.aliyuncs.com",
    "bailianWorkspace": "default",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}');
```

### API 请求扩展

```json
// POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好，请介绍一下自己",
  "options": {
    "modelName": "gpt-4",  // 可选，指定使用的模型名称（系统会根据当前租户ID和模型名称查询配置）
    "temperature": 0.8,
    "maxTokens": 1000
  }
}
```

### 管理接口

```json
// POST /api/v1/model-configurations - 创建模型配置
{
  "tenantId": "tenant-uuid-1",  // 平台管理员必须指定，租户管理员自动使用当前租户
  "modelName": "gpt-4",
  "providerType": "azure",
  "apiKey": "your-api-key",
  "configuration": {
    "model": "gpt-4",
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}

// GET /api/v1/model-configurations - 查询租户的模型配置列表
// GET /api/v1/model-configurations/{modelName} - 查询特定模型配置
// PUT /api/v1/model-configurations/{id} - 更新模型配置
// DELETE /api/v1/model-configurations/{id} - 删除模型配置
```

### 环境变量

```bash
# Google AI
GOOGLE_API_KEY=your_google_api_key

# Azure OpenAI
AZURE_OPENAI_KEY=your_azure_key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com

# 阿里云百炼
BAILIAN_API_KEY=your_bailian_key
BAILIAN_ENDPOINT=https://dashscope.aliyuncs.com
```

## 错误处理设计

### 错误类型

```go
// 配置错误
ErrInvalidConfig        = "配置无效"
ErrProviderNotFound     = "提供商不存在"
ErrMissingAPIKey        = "缺少 API 密钥"

// 初始化错误
ErrPluginInitFailed     = "插件初始化失败"
ErrProviderInitFailed   = "提供商初始化失败"

// 运行时错误
ErrProviderUnavailable  = "提供商不可用"
ErrAPICallFailed        = "API 调用失败"
ErrStreamingFailed      = "流式调用失败"
```

### 错误处理策略

1. **配置错误**：启动时验证，失败则拒绝启动
2. **初始化错误**：记录错误日志，标记提供商不可用
3. **运行时错误**：返回友好错误信息，记录详细日志
4. **API 错误**：解析错误响应，返回具体错误原因

### 错误日志格式

```go
logger.ErrorContext(ctx, "提供商初始化失败", logger.Fields{
    "provider": providerName,
    "type":     providerConfig.Provider,
    "error":    err.Error(),
    "traceId":  traceID,
})
```

## 性能优化设计

### 1. 懒加载提供商

只在首次使用时初始化提供商，避免启动时加载所有提供商。

```go
func (c *client) getOrInitProvider(ctx context.Context, name string) (*genkit.Genkit, error) {
    c.mu.RLock()
    g, exists := c.instances[name]
    c.mu.RUnlock()
    
    if exists {
        return g, nil
    }
    
    // 初始化提供商
    if err := c.InitializeProvider(ctx, name); err != nil {
        return nil, err
    }
    
    return c.instances[name], nil
}
```

### 2. 连接池复用

每个提供商维护一个 Genkit 实例，复用底层连接。

### 3. 并发控制

使用读写锁保护提供商实例映射，支持并发读取。

```go
type client struct {
    instances map[string]*genkit.Genkit
    mu        sync.RWMutex
}
```

## 安全设计

### 1. API 密钥管理

- 配置文件中使用环境变量引用
- 日志中脱敏处理
- 不在错误信息中暴露密钥

```go
func maskAPIKey(key string) string {
    if len(key) <= 8 {
        return "****"
    }
    return key[:4] + "****" + key[len(key)-4:]
}
```

### 2. 输入验证

- 验证提供商名称
- 验证模型参数范围
- 防止注入攻击

### 3. 访问控制

- 遵循多租户访问控制规范
- 验证用户权限
- 记录审计日志

## 测试设计

### 单元测试

```go
// client_test.go
func TestInitializeProvider(t *testing.T) {
    tests := []struct {
        name        string
        config      *Config
        wantErr     bool
        errContains string
    }{
        {
            name: "Google AI 初始化成功",
            config: &Config{
                Provider: ProviderGoogleAI,
                APIKey:   "test-key",
                Model:    "gemini-pro",
            },
            wantErr: false,
        },
        {
            name: "缺少 API 密钥",
            config: &Config{
                Provider: ProviderGoogleAI,
                Model:    "gemini-pro",
            },
            wantErr:     true,
            errContains: "API 密钥不能为空",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### 集成测试

```go
// integration_test.go
func TestAzureOpenAIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    // 加载配置
    config := loadTestConfig()
    
    // 初始化客户端
    client := NewClient()
    err := client.Initialize(context.Background(), config)
    require.NoError(t, err)
    
    // 测试生成
    result, err := client.Generate(context.Background(), "Hello", &GenerateOptions{
        Provider: "azure-gpt4",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, result.Text)
}
```

## 部署设计

### 配置文件位置

```
config/
  ├── ai.yaml              # AI 配置
  ├── ai.dev.yaml          # 开发环境配置
  ├── ai.prod.yaml         # 生产环境配置
  └── ai.example.yaml      # 配置示例
```

### 环境变量

```bash
# .env
AI_CONFIG_PATH=config/ai.yaml
GOOGLE_API_KEY=xxx
AZURE_OPENAI_KEY=xxx
BAILIAN_API_KEY=xxx
```

### 启动流程

1. 加载配置文件
2. 验证配置有效性
3. 初始化默认提供商
4. 启动 HTTP 服务
5. 懒加载其他提供商

## 监控设计

### 指标收集

```go
// 记录每次调用
metrics.RecordAICall(ctx, metrics.AICallMetrics{
    Provider:     providerName,
    Model:        modelName,
    Duration:     duration,
    TokensUsed:   usage.TotalTokens,
    Success:      err == nil,
    ErrorType:    getErrorType(err),
})
```

### 日志记录

```go
// 调用开始
logger.InfoContext(ctx, "开始 AI 调用", logger.Fields{
    "provider": providerName,
    "model":    modelName,
    "prompt":   truncate(prompt, 100),
})

// 调用结束
logger.InfoContext(ctx, "AI 调用完成", logger.Fields{
    "provider": providerName,
    "model":    modelName,
    "duration": duration.String(),
    "tokens":   usage.TotalTokens,
})
```

## 迁移计划

### 阶段 1：准备（1 天）

- 创建配置文件结构
- 准备测试环境
- 准备 API 密钥

### 阶段 2：核心实现（3 天）

- 扩展配置结构
- 重构 Genkit Client
- 实现多提供商支持

### 阶段 3：Azure OpenAI 集成（2 天）

- 集成 Azure OpenAI 插件
- 测试非流式调用
- 测试流式调用

### 阶段 4：百炼集成（3 天）

- 实现百炼插件
- 测试非流式调用
- 测试流式调用

### 阶段 5：测试和优化（2 天）

- 单元测试
- 集成测试
- 性能测试
- 错误处理测试

### 阶段 6：文档和部署（1 天）

- 编写使用文档
- 编写迁移指南
- 部署到测试环境

## 风险和缓解

### 风险 1：Genkit 不支持某些插件

**缓解**：自定义插件实现，参考 Genkit 插件开发文档

### 风险 2：不同提供商的响应格式差异大

**缓解**：在插件层统一转换为 Genkit 标准格式

### 风险 3：性能下降

**缓解**：使用懒加载、连接池、并发优化

### 风险 4：配置复杂度增加

**缓解**：提供配置示例、验证工具、清晰的错误信息

## 参考资料

- [Google Genkit 文档](https://firebase.google.com/docs/genkit)
- [Genkit Go SDK](https://github.com/firebase/genkit/tree/main/go)
- [Azure OpenAI Go SDK](https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/ai/azopenai)
- [阿里云百炼文档](https://help.aliyun.com/zh/model-studio/)
