# 设计文档

## 概述

本系统实现一个模型提供商和模型信息管理服务，通过在服务启动时从文件系统加载YAML配置文件到内存，并提供RESTful API接口供外部查询。

### 核心功能

1. **启动时数据加载**: 扫描models文件夹，解析所有提供商和模型的YAML配置文件
2. **内存缓存管理**: 将解析后的数据存储在内存中，提供快速查询
3. **RESTful API服务**: 提供5个核心接口，支持提供商和模型信息的查询

### 技术栈

- **语言**: Go 1.25+
- **YAML解析**: gopkg.in/yaml.v3
- **HTTP路由**: 标准库 net/http 或现有路由框架
- **数据结构**: 使用泛型支持类型安全的响应格式

## 架构设计

### 系统架构图

系统采用分层架构，从上到下分为：路由层、处理器层、服务层和数据存储层。

### 分层架构说明

#### 1. 路由层 (Router Layer)

- 负责HTTP请求的路由分发
- 定义API端点和对应的处理器
- 位置: `internal/api/routes/`

#### 2. 处理器层 (Handler Layer)

- 处理HTTP请求和响应
- 参数验证和错误处理
- 调用服务层获取数据
- 位置: `internal/api/handlers/`

#### 3. 服务层 (Service Layer)

- 业务逻辑处理
- 数据查询和过滤
- 与内存存储交互
- 位置: `internal/service/`

#### 4. 数据存储层 (Storage Layer)

- 内存中的数据结构
- 提供线程安全的数据访问
- 位置: `internal/storage/`

## 组件和接口

### 核心组件

#### 1. 数据加载器 (Loader)

**职责**: 在应用启动时加载所有提供商和模型配置

**位置**: `internal/loader/model_loader.go`

**核心方法**:

```go
type ModelLoader interface {
    // LoadAll 加载所有提供商和模型数据
    LoadAll(modelsDir string) error
    
    // LoadProviders 加载所有提供商配置
    LoadProviders(modelsDir string) ([]Provider, error)
    
    // LoadModels 加载指定提供商的所有模型
    LoadModels(providerDir string, providerID string) ([]Model, error)
}
```

**加载流程**:

1. 扫描 `models/` 目录下的所有子文件夹
2. 对每个提供商文件夹:
   - 读取 `provider/[提供商名].yaml`
   - 解析提供商配置，添加 `id` 字段
   - 根据 `models` 字段定义的类型列表加载模型
3. 对每个模型类型:
   - 读取 `models/[类型]/_position.yaml` 获取模型列表
   - 如果 `_position.yaml` 不存在，扫描该类型文件夹下所有 `.yaml` 文件
   - 依次读取每个模型的配置文件
   - 添加 `model_type` 字段
4. 将数据存储到内存存储中

#### 2. 内存存储 (Memory Store)

**职责**: 存储和管理提供商及模型数据

**位置**: `internal/storage/memory_store.go`

**数据结构**:

```go
type MemoryStore struct {
    mu        sync.RWMutex
    providers map[string]*Provider  // key: provider_id
    models    map[string][]Model    // key: provider_id, value: models list
}
```

**核心方法**:

```go
type Store interface {
    // SetProviders 设置提供商列表
    SetProviders(providers []Provider)
    
    // GetProviders 获取所有提供商
    GetProviders() []Provider
    
    // GetProvider 根据ID获取提供商
    GetProvider(providerID string) (*Provider, error)
    
    // SetModels 设置提供商的模型列表
    SetModels(providerID string, models []Model)
    
    // GetModels 获取提供商的所有模型
    GetModels(providerID string) ([]Model, error)
    
    // GetModel 获取指定模型
    GetModel(providerID, modelID string) (*Model, error)
}
```

#### 3. 提供商服务 (Provider Service)

**职责**: 提供商相关的业务逻辑

**位置**: `internal/service/provider_service.go`

**核心方法**:

```go
type ProviderService interface {
    // GetAllProviders 获取所有提供商（返回指定字段）
    GetAllProviders() []ProviderListItem
    
    // GetProviderByID 根据ID获取提供商详情
    GetProviderByID(providerID string) (*Provider, error)
    
    // GetProviderModels 获取提供商的所有模型（返回指定字段）
    GetProviderModels(providerID string) ([]ModelListItem, error)
    
    // GetProviderModel 获取提供商的指定模型
    GetProviderModel(providerID, modelID string) (*Model, error)
    
    // GetModelParameterRules 获取模型的参数规则
    GetModelParameterRules(providerID, modelID string) ([]ParameterRule, error)
}
```

#### 4. API处理器 (Handlers)

**职责**: 处理HTTP请求，调用服务层，返回响应

**位置**: `internal/api/handlers/provider_handler.go`

**核心方法**:

```go
type ProviderHandler struct {
    service ProviderService
}

// GetProviders 处理 GET /providers
func (h *ProviderHandler) GetProviders(w http.ResponseWriter, r *http.Request)

// GetProviderByID 处理 GET /providers/{providerId}
func (h *ProviderHandler) GetProviderByID(w http.ResponseWriter, r *http.Request)

// GetProviderModels 处理 GET /providers/{providerId}/models
func (h *ProviderHandler) GetProviderModels(w http.ResponseWriter, r *http.Request)

// GetProviderModel 处理 GET /providers/{providerId}/models/{modelId}
func (h *ProviderHandler) GetProviderModel(w http.ResponseWriter, r *http.Request)

// GetModelParameterRules 处理 GET /providers/{providerId}/models/{modelId}/parameter-rules
func (h *ProviderHandler) GetModelParameterRules(w http.ResponseWriter, r *http.Request)
```

### API路由定义

**位置**: `internal/api/routes/provider_routes.go`

```go
func RegisterProviderRoutes(router *http.ServeMux, handler *ProviderHandler) {
    router.HandleFunc("GET /providers", handler.GetProviders)
    router.HandleFunc("GET /providers/{providerId}", handler.GetProviderByID)
    router.HandleFunc("GET /providers/{providerId}/models", handler.GetProviderModels)
    router.HandleFunc("GET /providers/{providerId}/models/{modelId}", handler.GetProviderModel)
    router.HandleFunc("GET /providers/{providerId}/models/{modelId}/parameter-rules", handler.GetModelParameterRules)
}
```

## 数据模型

### Provider (提供商)

```go
// Provider 提供商完整信息
type Provider struct {
    ID                       string                   `yaml:"-" json:"id"`
    Provider                 string                   `yaml:"provider" json:"provider"`
    Label                    map[string]string        `yaml:"label" json:"label"`
    Background               string                   `yaml:"background" json:"background"`
    IconSmall                map[string]string        `yaml:"icon_small" json:"icon_small"`
    IconLarge                map[string]string        `yaml:"icon_large" json:"icon_large"`
    Help                     ProviderHelp             `yaml:"help" json:"help"`
    ConfigurateMethods       []string                 `yaml:"configurate_methods" json:"configurate_methods"`
    SupportedModelTypes      []string                 `yaml:"supported_model_types" json:"supported_model_types"`
    ProviderCredentialSchema CredentialSchema         `yaml:"provider_credential_schema" json:"provider_credential_schema"`
    ModelCredentialSchema    CredentialSchema         `yaml:"model_credential_schema" json:"model_credential_schema"`
    Models                   map[string]ModelTypeInfo `yaml:"models" json:"models"`
}

// ProviderListItem 提供商列表项（用于列表接口）
type ProviderListItem struct {
    ID                 string            `json:"id"`
    Provider           string            `json:"provider"`
    Label              map[string]string `json:"label"`
    Background         string            `json:"background"`
    IconSmall          map[string]string `json:"icon_small"`
    IconLarge          map[string]string `json:"icon_large"`
    Help               ProviderHelp      `json:"help"`
    ConfigurateMethods []string          `json:"configurate_methods"`
}

type ProviderHelp struct {
    Title map[string]string `yaml:"title" json:"title"`
    URL   map[string]string `yaml:"url" json:"url"`
}

type CredentialSchema struct {
    CredentialFormSchemas []CredentialFormSchema `yaml:"credential_form_schemas" json:"credential_form_schemas"`
}

type CredentialFormSchema struct {
    Variable    string            `yaml:"variable" json:"variable"`
    Label       map[string]string `yaml:"label" json:"label"`
    Type        string            `yaml:"type" json:"type"`
    Required    bool              `yaml:"required" json:"required"`
    Default     string            `yaml:"default,omitempty" json:"default,omitempty"`
    Placeholder map[string]string `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
    Options     []FormOption      `yaml:"options,omitempty" json:"options,omitempty"`
}

type FormOption struct {
    Label map[string]string `yaml:"label" json:"label"`
    Value string            `yaml:"value" json:"value"`
}

type ModelTypeInfo struct {
    Position   string   `yaml:"position,omitempty" json:"position,omitempty"`
    Predefined []string `yaml:"predefined" json:"predefined"`
}
```

### Model (模型)

```go
// Model 模型完整信息
type Model struct {
    Model            string                 `yaml:"model" json:"model"`
    Label            map[string]string      `yaml:"label" json:"label"`
    ModelType        string                 `yaml:"model_type" json:"model_type"`
    Features         []string               `yaml:"features,omitempty" json:"features,omitempty"`
    ModelProperties  ModelProperties        `yaml:"model_properties" json:"model_properties"`
    ParameterRules   []ParameterRule        `yaml:"parameter_rules,omitempty" json:"parameter_rules,omitempty"`
    Pricing          Pricing                `yaml:"pricing,omitempty" json:"pricing,omitempty"`
    Deprecated       bool                   `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
}

// ModelListItem 模型列表项（用于列表接口）
type ModelListItem struct {
    Model           string            `json:"model"`
    Label           map[string]string `json:"label"`
    ModelType       string            `json:"model_type"`
    Features        []string          `json:"features,omitempty"`
    ModelProperties ModelProperties   `json:"model_properties"`
    ParameterRules  []ParameterRule   `json:"parameter_rules,omitempty"`
    Pricing         Pricing           `json:"pricing,omitempty"`
}

type ModelProperties struct {
    Mode        string `yaml:"mode" json:"mode"`
    ContextSize int    `yaml:"context_size" json:"context_size"`
}

type ParameterRule struct {
    Name        string            `yaml:"name" json:"name"`
    UseTemplate string            `yaml:"use_template,omitempty" json:"use_template,omitempty"`
    Label       map[string]string `yaml:"label,omitempty" json:"label,omitempty"`
    Type        string            `yaml:"type" json:"type"`
    Required    bool              `yaml:"required,omitempty" json:"required,omitempty"`
    Default     interface{}       `yaml:"default,omitempty" json:"default,omitempty"`
    Min         interface{}       `yaml:"min,omitempty" json:"min,omitempty"`
    Max         interface{}       `yaml:"max,omitempty" json:"max,omitempty"`
    Help        map[string]string `yaml:"help,omitempty" json:"help,omitempty"`
    Options     []string          `yaml:"options,omitempty" json:"options,omitempty"`
}

type Pricing struct {
    Input    string `yaml:"input" json:"input"`
    Output   string `yaml:"output" json:"output"`
    Unit     string `yaml:"unit" json:"unit"`
    Currency string `yaml:"currency" json:"currency"`
}
```

### 响应格式

系统使用已定义的泛型响应格式：

```go
// ResponseData 通用响应数据结构
type ResponseData[T any] struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    *T     `json:"data,omitempty"`
}
```

## 错误处理

### 错误类型

系统定义以下错误类型：

```go
// 错误码定义
const (
    CodeSuccess           = 200
    CodeBadRequest        = 400
    CodeNotFound          = 404
    CodeInternalError     = 500
)

// 错误消息
const (
    MsgSuccess              = "success"
    MsgProviderNotFound     = "provider not found"
    MsgModelNotFound        = "model not found"
    MsgInvalidParameter     = "invalid parameter"
    MsgInternalError        = "internal server error"
)
```

### 错误处理策略

#### 1. 数据加载阶段错误

- **文件不存在**: 记录警告日志，跳过该文件，继续处理其他文件
- **YAML解析错误**: 记录错误日志（包含文件路径和错误详情），跳过该文件
- **必需字段缺失**: 记录错误日志，跳过该配置
- **目录访问错误**: 记录错误日志，返回加载失败

**日志示例**:

```go
logger.Warn("position file not found, will scan directory", 
    "provider", providerID, 
    "type", modelType, 
    "path", positionPath)

logger.Error("failed to parse provider yaml", 
    "provider", providerID, 
    "path", yamlPath, 
    "error", err)
```

#### 2. API请求阶段错误

- **提供商不存在**: 返回 404 状态码和 `MsgProviderNotFound` 消息
- **模型不存在**: 返回 404 状态码和 `MsgModelNotFound` 消息
- **参数无效**: 返回 400 状态码和 `MsgInvalidParameter` 消息
- **内部错误**: 返回 500 状态码和 `MsgInternalError` 消息，记录详细错误堆栈

**响应示例**:

```json
{
    "code": 404,
    "message": "provider not found",
    "data": null
}
```

### 日志记录

使用结构化日志记录关键操作：

```go
// 启动加载
logger.Info("starting to load model providers", "dir", modelsDir)

// 加载完成
logger.Info("model providers loaded successfully", 
    "providers", len(providers), 
    "total_models", totalModels)

// 错误记录
logger.Error("failed to load provider", 
    "provider", providerID, 
    "error", err, 
    "stack", string(debug.Stack()))
```

## 测试策略

### 单元测试

#### 1. 数据加载器测试

**测试文件**: `internal/loader/model_loader_test.go`

**测试用例**:

- 测试加载单个提供商配置
- 测试加载提供商时文件不存在的情况
- 测试加载提供商时YAML格式错误的情况
- 测试加载模型时使用 `_position.yaml` 的情况
- 测试加载模型时不存在 `_position.yaml` 的情况
- 测试加载模型时YAML格式错误的情况
- 测试完整的加载流程

**测试数据**: 在 `testdata/` 目录下创建测试用的YAML文件

#### 2. 内存存储测试

**测试文件**: `internal/storage/memory_store_test.go`

**测试用例**:

- 测试设置和获取提供商列表
- 测试根据ID获取提供商
- 测试获取不存在的提供商
- 测试设置和获取模型列表
- 测试根据ID获取模型
- 测试获取不存在的模型
- 测试并发读写安全性

#### 3. 服务层测试

**测试文件**: `internal/service/provider_service_test.go`

**测试用例**:

- 测试获取所有提供商（验证返回字段）
- 测试根据ID获取提供商详情
- 测试获取提供商的模型列表（验证返回字段）
- 测试获取指定模型详情
- 测试获取模型参数规则
- 测试各种错误情况

**Mock**: 使用 mock 的 Store 接口

#### 4. 处理器层测试

**测试文件**: `internal/api/handlers/provider_handler_test.go`

**测试用例**:

- 测试每个API端点的正常响应
- 测试每个API端点的错误响应
- 测试响应格式是否符合 `ResponseData[T]` 规范
- 测试HTTP状态码是否正确

**Mock**: 使用 mock 的 Service 接口

### 集成测试

**测试文件**: `test/integration/provider_api_test.go`

**测试用例**:

- 测试完整的数据加载流程
- 测试所有API端点的端到端调用
- 测试使用真实的YAML文件（从 `models/` 目录）
- 测试并发请求处理

### 测试覆盖率目标

- 核心业务逻辑: 80%+
- 数据加载器: 75%+
- 服务层: 80%+
- 处理器层: 70%+

## 性能考虑

### 内存使用

#### 数据量估算

假设系统有：

- 10个提供商
- 每个提供商平均100个模型
- 每个提供商配置约5KB
- 每个模型配置约2KB

**总内存占用估算**:

- 提供商数据: 10 × 5KB = 50KB
- 模型数据: 10 × 100 × 2KB = 2MB
- 总计: 约 2.05MB

这个内存占用量对于现代服务器来说非常小，完全可以接受。

#### 内存优化策略

1. **延迟加载**: 当前设计采用启动时全量加载，适合数据量不大的场景
2. **数据压缩**: 如果未来数据量增长，可以考虑对不常用字段进行压缩存储
3. **分级缓存**: 可以将热点数据和冷数据分开存储

### 并发性能

#### 读写锁设计

```go
type MemoryStore struct {
    mu        sync.RWMutex  // 使用读写锁
    providers map[string]*Provider
    models    map[string][]Model
}

// 读操作使用 RLock
func (s *MemoryStore) GetProvider(id string) (*Provider, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // ...
}

// 写操作使用 Lock
func (s *MemoryStore) SetProviders(providers []Provider) {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ...
}
```

#### 性能特点

- **读操作**: 支持高并发，多个goroutine可以同时读取
- **写操作**: 仅在启动时执行一次，不影响运行时性能
- **查询复杂度**: O(1) - 使用map进行索引

### API响应时间

**预期响应时间**:

- 获取提供商列表: < 5ms
- 获取单个提供商: < 1ms
- 获取模型列表: < 10ms
- 获取单个模型: < 1ms
- 获取参数规则: < 1ms

### 扩展性考虑

#### 水平扩展

当前设计支持无状态部署，可以轻松进行水平扩展：

- 每个实例独立加载数据到内存
- 无需共享状态或分布式缓存
- 可以通过负载均衡器分发请求

#### 数据更新策略

当前版本采用启动时加载，如果需要支持动态更新：

**方案1: 定期重新加载**

```go
// 每隔一段时间重新加载数据
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        loader.LoadAll(modelsDir)
    }
}()
```

**方案2: 文件监听**

```go
// 使用 fsnotify 监听文件变化
watcher, _ := fsnotify.NewWatcher()
watcher.Add(modelsDir)
go func() {
    for event := range watcher.Events {
        if event.Op&fsnotify.Write == fsnotify.Write {
            loader.LoadAll(modelsDir)
        }
    }
}()
```

**方案3: API触发重载**

```go
// 提供管理接口手动触发重载
router.HandleFunc("POST /admin/reload", handler.ReloadData)
```

当前实现不包含动态更新功能，如有需要可在后续迭代中添加。

## 实现细节

### 启动流程

```go
func main() {
    // 1. 初始化日志
    logger := logger.NewLogger()
    
    // 2. 创建内存存储
    store := storage.NewMemoryStore()
    
    // 3. 创建加载器
    loader := loader.NewModelLoader(store, logger)
    
    // 4. 加载数据
    if err := loader.LoadAll("./models"); err != nil {
        logger.Fatal("failed to load model data", "error", err)
    }
    
    // 5. 创建服务层
    providerService := service.NewProviderService(store)
    
    // 6. 创建处理器
    providerHandler := handlers.NewProviderHandler(providerService, logger)
    
    // 7. 注册路由
    router := http.NewServeMux()
    routes.RegisterProviderRoutes(router, providerHandler)
    
    // 8. 启动服务器
    server := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }
    
    logger.Info("server starting", "addr", server.Addr)
    if err := server.ListenAndServe(); err != nil {
        logger.Fatal("server failed", "error", err)
    }
}
```

### 数据加载详细流程

#### 1. 加载提供商

```go
func (l *ModelLoader) LoadProviders(modelsDir string) ([]Provider, error) {
    var providers []Provider
    
    // 读取 models 目录
    entries, err := os.ReadDir(modelsDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read models directory: %w", err)
    }
    
    // 遍历每个提供商文件夹
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        
        providerID := entry.Name()
        
        // 跳过特殊文件夹
        if strings.HasPrefix(providerID, ".") || strings.HasPrefix(providerID, "_") {
            continue
        }
        
        // 读取 provider yaml
        yamlPath := filepath.Join(modelsDir, providerID, "provider", providerID+".yaml")
        data, err := os.ReadFile(yamlPath)
        if err != nil {
            l.logger.Error("failed to read provider yaml", 
                "provider", providerID, 
                "path", yamlPath, 
                "error", err)
            continue
        }
        
        // 解析 YAML
        var provider Provider
        if err := yaml.Unmarshal(data, &provider); err != nil {
            l.logger.Error("failed to parse provider yaml", 
                "provider", providerID, 
                "path", yamlPath, 
                "error", err)
            continue
        }
        
        // 设置 ID
        provider.ID = providerID
        
        providers = append(providers, provider)
        
        l.logger.Info("loaded provider", "provider", providerID)
    }
    
    return providers, nil
}
```

#### 2. 加载模型

```go
func (l *ModelLoader) LoadModels(providerDir string, providerID string, modelTypes map[string]ModelTypeInfo) ([]Model, error) {
    var allModels []Model
    
    // 遍历每个模型类型
    for modelType, typeInfo := range modelTypes {
        modelsDir := filepath.Join(providerDir, "models", modelType)
        
        // 检查目录是否存在
        if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
            l.logger.Warn("model type directory not found", 
                "provider", providerID, 
                "type", modelType, 
                "path", modelsDir)
            continue
        }
        
        var modelNames []string
        
        // 尝试读取 _position.yaml
        if typeInfo.Position != "" {
            positionPath := filepath.Join(providerDir, typeInfo.Position)
            data, err := os.ReadFile(positionPath)
            if err == nil {
                if err := yaml.Unmarshal(data, &modelNames); err != nil {
                    l.logger.Error("failed to parse position yaml", 
                        "provider", providerID, 
                        "type", modelType, 
                        "error", err)
                }
            } else {
                l.logger.Warn("position file not found, will scan directory", 
                    "provider", providerID, 
                    "type", modelType)
            }
        }
        
        // 如果没有 position 文件，扫描目录
        if len(modelNames) == 0 {
            entries, err := os.ReadDir(modelsDir)
            if err != nil {
                l.logger.Error("failed to read models directory", 
                    "provider", providerID, 
                    "type", modelType, 
                    "error", err)
                continue
            }
            
            for _, entry := range entries {
                if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
                    continue
                }
                // 排除特殊文件
                if strings.HasPrefix(entry.Name(), "_") {
                    continue
                }
                modelName := strings.TrimSuffix(entry.Name(), ".yaml")
                modelNames = append(modelNames, modelName)
            }
        }
        
        // 加载每个模型
        for _, modelName := range modelNames {
            modelPath := filepath.Join(modelsDir, modelName+".yaml")
            data, err := os.ReadFile(modelPath)
            if err != nil {
                l.logger.Error("failed to read model yaml", 
                    "provider", providerID, 
                    "type", modelType, 
                    "model", modelName, 
                    "error", err)
                continue
            }
            
            var model Model
            if err := yaml.Unmarshal(data, &model); err != nil {
                l.logger.Error("failed to parse model yaml", 
                    "provider", providerID, 
                    "type", modelType, 
                    "model", modelName, 
                    "error", err)
                continue
            }
            
            // 设置 model_type
            model.ModelType = modelType
            
            allModels = append(allModels, model)
        }
        
        l.logger.Info("loaded models", 
            "provider", providerID, 
            "type", modelType, 
            "count", len(modelNames))
    }
    
    return allModels, nil
}
```

### API实现示例

#### 获取提供商列表

```go
func (h *ProviderHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
    // 调用服务层
    providers := h.service.GetAllProviders()
    
    // 构建响应
    resp := response.Success(&providers)
    
    // 返回JSON
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}
```

#### 获取提供商详情

```go
func (h *ProviderHandler) GetProviderByID(w http.ResponseWriter, r *http.Request) {
    // 获取路径参数
    providerID := r.PathValue("providerId")
    
    // 调用服务层
    provider, err := h.service.GetProviderByID(providerID)
    if err != nil {
        // 处理错误
        var resp model.ResponseData[Provider]
        if errors.Is(err, ErrProviderNotFound) {
            resp = response.Error[Provider](errors.CodeNotFound, errors.MsgProviderNotFound)
            w.WriteHeader(http.StatusNotFound)
        } else {
            resp = response.Error[Provider](errors.CodeInternalError, errors.MsgInternalError)
            w.WriteHeader(http.StatusInternalServerError)
            h.logger.Error("failed to get provider", "provider", providerID, "error", err)
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
        return
    }
    
    // 构建成功响应
    resp := response.Success(provider)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}
```

#### 获取模型参数规则

```go
func (h *ProviderHandler) GetModelParameterRules(w http.ResponseWriter, r *http.Request) {
    // 获取路径参数
    providerID := r.PathValue("providerId")
    modelID := r.PathValue("modelId")
    
    // 调用服务层
    rules, err := h.service.GetModelParameterRules(providerID, modelID)
    if err != nil {
        var resp model.ResponseData[[]ParameterRule]
        if errors.Is(err, ErrProviderNotFound) {
            resp = response.Error[[]ParameterRule](errors.CodeNotFound, errors.MsgProviderNotFound)
            w.WriteHeader(http.StatusNotFound)
        } else if errors.Is(err, ErrModelNotFound) {
            resp = response.Error[[]ParameterRule](errors.CodeNotFound, errors.MsgModelNotFound)
            w.WriteHeader(http.StatusNotFound)
        } else {
            resp = response.Error[[]ParameterRule](errors.CodeInternalError, errors.MsgInternalError)
            w.WriteHeader(http.StatusInternalServerError)
            h.logger.Error("failed to get parameter rules", 
                "provider", providerID, 
                "model", modelID, 
                "error", err)
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
        return
    }
    
    // 构建成功响应
    resp := response.Success(&rules)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}
```

## 配置管理

### 环境变量

```bash
# 服务配置
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# 数据目录
MODELS_DIR=./models

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=json
```

### 配置文件

**位置**: `config/config.yaml`

```yaml
server:
  port: 8080
  host: 0.0.0.0
  read_timeout: 30s
  write_timeout: 30s

models:
  dir: ./models
  
logging:
  level: info
  format: json
  output: stdout
```

### 配置加载

```go
type Config struct {
    Server ServerConfig `yaml:"server"`
    Models ModelsConfig `yaml:"models"`
    Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
    Port         int           `yaml:"port"`
    Host         string        `yaml:"host"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
}

type ModelsConfig struct {
    Dir string `yaml:"dir"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
    Output string `yaml:"output"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## 部署考虑

### Docker部署

**Dockerfile**:

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:latest

WORKDIR /app

# 复制二进制文件
COPY --from=builder /server .

# 复制模型配置文件
COPY models ./models

# 复制配置文件
COPY config ./config

EXPOSE 8080

CMD ["./server"]
```

**docker-compose.yml**:

```yaml
version: '3.8'

services:
  model-provider-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - MODELS_DIR=/app/models
      - LOG_LEVEL=info
    volumes:
      - ./models:/app/models:ro
    restart: unless-stopped
```

### 健康检查

添加健康检查端点：

```go
// 健康检查处理器
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Unix(),
        "providers_count": h.store.GetProvidersCount(),
        "models_count": h.store.GetModelsCount(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(status)
}

// 注册路由
router.HandleFunc("GET /health", healthHandler.HealthCheck)
```

### 监控指标

建议添加以下监控指标：

1. **请求指标**
   - 请求总数
   - 请求延迟（P50, P95, P99）
   - 错误率

2. **系统指标**
   - 内存使用量
   - CPU使用率
   - Goroutine数量

3. **业务指标**
   - 提供商数量
   - 模型总数
   - 各API端点调用次数

可以使用 Prometheus + Grafana 进行监控。

## 安全考虑

### 输入验证

1. **路径参数验证**
   - 验证 `providerId` 和 `modelId` 格式
   - 防止路径遍历攻击
   - 限制参数长度

```go
func validateProviderID(id string) error {
    if len(id) == 0 || len(id) > 100 {
        return errors.New("invalid provider id length")
    }
    
    // 只允许字母、数字、下划线和连字符
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, id)
    if !matched {
        return errors.New("invalid provider id format")
    }
    
    return nil
}
```

2. **防止目录遍历**
   - 使用 `filepath.Clean()` 清理路径
   - 验证路径在允许的目录内

```go
func safePath(base, path string) (string, error) {
    fullPath := filepath.Join(base, path)
    cleanPath := filepath.Clean(fullPath)
    
    if !strings.HasPrefix(cleanPath, base) {
        return "", errors.New("invalid path")
    }
    
    return cleanPath, nil
}
```

### CORS配置

如果需要支持跨域请求：

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### 速率限制

建议添加速率限制防止滥用：

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiter *rate.Limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !rl.limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 日志脱敏

确保日志中不包含敏感信息：

```go
func sanitizeForLog(data interface{}) interface{} {
    // 移除或脱敏敏感字段
    // 例如：API密钥、密码等
    return data
}
```

## 未来扩展

### 可能的功能增强

#### 1. 缓存优化

- 添加LRU缓存减少内存占用
- 支持缓存预热
- 实现缓存失效策略

#### 2. 数据版本管理

- 支持多版本模型配置
- 提供版本切换API
- 记录配置变更历史

#### 3. 搜索和过滤

```go
// 按特性搜索模型
GET /providers/{providerId}/models?features=multi-tool-call

// 按类型过滤
GET /providers/{providerId}/models?type=llm

// 按价格排序
GET /providers/{providerId}/models?sort=price&order=asc
```

#### 4. 批量查询

```go
// 批量获取多个提供商
POST /providers/batch
{
    "provider_ids": ["tongyi", "gemini"]
}

// 批量获取多个模型
POST /providers/{providerId}/models/batch
{
    "model_ids": ["qwen-max", "qwen-plus"]
}
```

#### 5. 统计和分析

```go
// 获取统计信息
GET /stats
{
    "total_providers": 10,
    "total_models": 1000,
    "models_by_type": {
        "llm": 800,
        "tts": 100,
        "text_embedding": 100
    }
}
```

#### 6. 配置验证

- 添加YAML schema验证
- 提供配置检查工具
- 自动检测配置错误

#### 7. 国际化支持

- 根据请求头返回对应语言
- 支持多语言标签和描述

```go
// 根据 Accept-Language 返回对应语言
GET /providers
Accept-Language: zh-CN
```

### 技术债务和改进

1. **错误处理**: 使用更细粒度的错误类型
2. **日志**: 添加请求追踪ID
3. **文档**: 生成OpenAPI/Swagger文档
4. **测试**: 增加性能测试和压力测试
5. **监控**: 集成分布式追踪系统

## 总结

### 设计决策

1. **内存存储**: 选择内存存储而非数据库，因为：
   - 数据量小（约2MB）
   - 读多写少（启动时写入一次）
   - 查询性能要求高
   - 无需持久化（数据来源于文件系统）

2. **分层架构**: 采用清晰的分层设计，便于：
   - 代码组织和维护
   - 单元测试
   - 未来扩展

3. **泛型响应**: 使用Go泛型实现类型安全的响应格式，提供：
   - 统一的API响应结构
   - 编译时类型检查
   - 更好的代码复用

4. **错误处理**: 采用明确的错误处理策略：
   - 加载阶段：记录日志，继续处理
   - 运行阶段：返回标准错误响应

### 关键技术点

- **YAML解析**: 使用 `gopkg.in/yaml.v3` 解析配置文件
- **并发安全**: 使用 `sync.RWMutex` 保证并发读写安全
- **路径处理**: 使用 `filepath` 包处理跨平台路径
- **HTTP路由**: 使用Go 1.22+的新路由模式
- **结构化日志**: 使用键值对格式的日志记录

### 项目结构

```
.
├── cmd/
│   └── server/
│       └── main.go                 # 应用入口
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   └── provider_handler.go    # API处理器
│   │   └── routes/
│   │       └── provider_routes.go     # 路由定义
│   ├── config/
│   │   └── config.go               # 配置管理
│   ├── loader/
│   │   └── model_loader.go         # 数据加载器
│   ├── model/
│   │   ├── provider.go             # 提供商数据模型
│   │   └── model.go                # 模型数据模型
│   ├── service/
│   │   └── provider_service.go     # 业务逻辑层
│   └── storage/
│       └── memory_store.go         # 内存存储
├── pkg/
│   ├── errors/
│   │   └── errors.go               # 错误定义
│   └── response/
│       └── response.go             # 响应格式
├── models/                         # 模型配置文件
│   ├── tongyi/
│   └── gemini/
├── config/
│   └── config.yaml                 # 配置文件
├── go.mod
└── go.sum
```

### 开发优先级

1. **第一阶段**: 核心功能
   - 数据模型定义
   - 数据加载器实现
   - 内存存储实现
   - 基础API实现

2. **第二阶段**: 完善功能
   - 错误处理优化
   - 日志记录完善
   - 单元测试编写

3. **第三阶段**: 生产就绪
   - 集成测试
   - 性能优化
   - 文档完善
   - 部署配置
