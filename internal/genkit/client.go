package genkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	anthropic "github.com/firebase/genkit/go/plugins/compat_oai/anthropic"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/google/uuid"
	"github.com/openai/openai-go/option"
)

// Client Genkit 客户端接口
type Client interface {
	// Initialize 初始化客户端
	Initialize(ctx context.Context, config *Config) error

	// InitializeModel 初始化并设置模型
	InitializeModel(ctx context.Context) error

	// Generate 生成内容（根据租户ID和模型名称）
	Generate(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (*GenerateResult, error)

	// GenerateStream 流式生成内容（根据租户ID和模型名称）
	GenerateStream(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (<-chan StreamChunk, error)

	// GetGenkit 获取底层的Genkit实例（用于注册Flows）
	GetGenkit() *genkit.Genkit

	// Close 关闭客户端
	Close() error
}

// client Genkit 客户端实现
type client struct {
	config     *Config
	g          *genkit.Genkit
	configRepo repository.ModelConfigurationRepository // 模型配置仓储
	instances  map[string]*genkit.Genkit                // Genkit 实例缓存，key: tenantID_modelName
	mu         sync.RWMutex                             // 读写锁，保护 instances
}

// NewClient 创建新的 Genkit 客户端
func NewClient() Client {
	return &client{
		instances: make(map[string]*genkit.Genkit),
	}
}

// NewClientWithRepo 创建新的 Genkit 客户端（注入 ModelConfigurationRepository）
func NewClientWithRepo(configRepo repository.ModelConfigurationRepository) Client {
	return &client{
		configRepo: configRepo,
		instances:  make(map[string]*genkit.Genkit),
	}
}

// Initialize 初始化客户端
func (c *client) Initialize(ctx context.Context, config *Config) error {
	if config == nil {
		logger.ErrorContext(ctx, "初始化失败：配置不能为空", logger.Fields{})
		return fmt.Errorf("配置不能为空")
	}

	if config.APIKey == "" {
		logger.ErrorContext(ctx, "初始化失败：API 密钥不能为空", logger.Fields{})
		return fmt.Errorf("API 密钥不能为空")
	}

	if config.Model == "" {
		logger.ErrorContext(ctx, "初始化失败：模型名称不能为空", logger.Fields{})
		return fmt.Errorf("模型名称不能为空")
	}

	c.config = config

	return nil
}

// InitializeModel 初始化并设置模型
func (c *client) InitializeModel(ctx context.Context) error {
	if c.config == nil {
		logger.ErrorContext(ctx, "初始化模型失败：客户端未初始化", logger.Fields{})
		return fmt.Errorf("客户端未初始化，请先调用 Initialize")
	}

	// 初始化 Genkit，配置 Google AI 插件和默认模型
	c.g = genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: c.config.APIKey,
		}),
		genkit.WithDefaultModel("googleai/"+c.config.Model),
	)

	return nil
}

// getOrInitGenkit 获取或初始化 Genkit 实例
// 根据租户ID和模型名称从数据库查询配置，并初始化对应的 Genkit 实例
// 实例会被缓存以提高性能
// 使用双重检查锁定模式确保并发安全
func (c *client) getOrInitGenkit(ctx context.Context, tenantID, modelName string) (*genkit.Genkit, *GenkitConfig, error) {
	if c.configRepo == nil {
		logger.ErrorContext(ctx, "获取 Genkit 实例失败：配置仓储未初始化", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
		})
		return nil, nil, fmt.Errorf("模型配置仓储未初始化")
	}

	// 生成缓存键
	cacheKey := fmt.Sprintf("%s_%s", tenantID, modelName)
	
	// 记录提供商选择开始
	logger.DebugContext(ctx, "开始获取或初始化 Genkit 实例", logger.Fields{
		"tenantId":  tenantID,
		"modelName": modelName,
		"cacheKey":  cacheKey,
	})

	// 第一次检查：尝试从缓存获取实例（使用读锁）
	c.mu.RLock()
	g, exists := c.instances[cacheKey]
	c.mu.RUnlock()

	if exists {
		logger.DebugContext(ctx, "从缓存获取 Genkit 实例", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
			"cacheHit":  true,
		})
		
		// 从数据库获取配置（用于返回配置信息）
		tenantUUID, err := parseUUID(tenantID)
		if err != nil {
			logger.ErrorContext(ctx, "解析租户ID失败", logger.Fields{
				"tenantId": tenantID,
				"error":    err.Error(),
			})
			return nil, nil, fmt.Errorf("无效的租户ID: %w", err)
		}

		modelConfig, err := c.configRepo.GetByTenantAndModel(ctx, tenantUUID, modelName)
		if err != nil {
			logger.ErrorContext(ctx, "获取模型配置失败", logger.Fields{
				"tenantId":  tenantID,
				"modelName": modelName,
				"error":     err.Error(),
			})
			return nil, nil, fmt.Errorf("获取模型配置失败: %w", err)
		}

		// 解析配置
		genkitConfig, err := c.parseModelConfiguration(ctx, modelConfig)
		if err != nil {
			logger.ErrorContext(ctx, "解析模型配置失败", logger.Fields{
				"tenantId":  tenantID,
				"modelName": modelName,
				"error":     err.Error(),
			})
			return nil, nil, fmt.Errorf("解析模型配置失败: %w", err)
		}

		return g, genkitConfig, nil
	}

	// 缓存未命中，需要初始化新实例
	logger.InfoContext(ctx, "缓存未命中，准备初始化新的 Genkit 实例", logger.Fields{
		"tenantId":  tenantID,
		"modelName": modelName,
		"cacheHit":  false,
	})
	
	// 使用写锁保护初始化过程
	c.mu.Lock()
	defer c.mu.Unlock()

	// 第二次检查：在获取写锁后再次检查缓存
	// 防止多个 goroutine 同时初始化同一个实例
	if g, exists := c.instances[cacheKey]; exists {
		logger.DebugContext(ctx, "双重检查：实例已被其他 goroutine 初始化", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
		})
		
		// 另一个 goroutine 已经初始化了实例
		// 从数据库获取配置
		tenantUUID, err := parseUUID(tenantID)
		if err != nil {
			logger.ErrorContext(ctx, "解析租户ID失败", logger.Fields{
				"tenantId": tenantID,
				"error":    err.Error(),
			})
			return nil, nil, fmt.Errorf("无效的租户ID: %w", err)
		}

		modelConfig, err := c.configRepo.GetByTenantAndModel(ctx, tenantUUID, modelName)
		if err != nil {
			logger.ErrorContext(ctx, "获取模型配置失败", logger.Fields{
				"tenantId":  tenantID,
				"modelName": modelName,
				"error":     err.Error(),
			})
			return nil, nil, fmt.Errorf("获取模型配置失败: %w", err)
		}

		// 解析配置
		genkitConfig, err := c.parseModelConfiguration(ctx, modelConfig)
		if err != nil {
			logger.ErrorContext(ctx, "解析模型配置失败", logger.Fields{
				"tenantId":  tenantID,
				"modelName": modelName,
				"error":     err.Error(),
			})
			return nil, nil, fmt.Errorf("解析模型配置失败: %w", err)
		}

		return g, genkitConfig, nil
	}

	// 从数据库查询配置
	logger.DebugContext(ctx, "从数据库查询模型配置", logger.Fields{
		"tenantId":  tenantID,
		"modelName": modelName,
	})
	
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "解析租户ID失败", logger.Fields{
			"tenantId": tenantID,
			"error":    err.Error(),
		})
		return nil, nil, fmt.Errorf("无效的租户ID: %w", err)
	}

	modelConfig, err := c.configRepo.GetByTenantAndModel(ctx, tenantUUID, modelName)
	if err != nil {
		logger.ErrorContext(ctx, "获取模型配置失败", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
			"error":     err.Error(),
		})
		return nil, nil, fmt.Errorf("获取模型配置失败: %w", err)
	}

	// 检查模型是否启用
	if !modelConfig.IsEnabled {
		logger.WarnContext(ctx, "模型已禁用", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
		})
		return nil, nil, fmt.Errorf("模型已禁用: %s", modelName)
	}

	// 解析配置
	genkitConfig, err := c.parseModelConfiguration(ctx, modelConfig)
	if err != nil {
		logger.ErrorContext(ctx, "解析模型配置失败", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
			"error":     err.Error(),
		})
		return nil, nil, fmt.Errorf("解析模型配置失败: %w", err)
	}

	// 验证配置
	if err := genkitConfig.Validate(modelConfig.ModelProvider); err != nil {
		logger.ErrorContext(ctx, "配置验证失败", logger.Fields{
			"tenantId":       tenantID,
			"modelName":      modelName,
			"modelProvider":  modelConfig.ModelProvider,
			"error":          err.Error(),
		})
		return nil, nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 记录提供商选择（不记录敏感信息）
	logger.InfoContext(ctx, "选择模型提供商", logger.Fields{
		"tenantId":      tenantID,
		"modelName":     modelName,
		"provider":      modelConfig.ModelProvider,
		"model":         genkitConfig.Model,
	})

	// 根据提供商类型创建插件并初始化 Genkit 实例
	g, err = c.initializeProvider(ctx, modelConfig, genkitConfig)
	if err != nil {
		logger.ErrorContext(ctx, "初始化提供商失败", logger.Fields{
			"tenantId":      tenantID,
			"modelName":     modelName,
			"provider":      modelConfig.ModelProvider,
			"error":         err.Error(),
		})
		return nil, nil, fmt.Errorf("初始化提供商失败: %w", err)
	}

	// 缓存实例（已经持有写锁，无需再次加锁）
	c.instances[cacheKey] = g
	
	logger.InfoContext(ctx, "成功初始化并缓存 Genkit 实例", logger.Fields{
		"tenantId":      tenantID,
		"modelName":     modelName,
		"provider":      modelConfig.ModelProvider,
		"cacheKey":      cacheKey,
	})

	return g, genkitConfig, nil
}

// parseUUID 解析 UUID 字符串
func parseUUID(uuidStr string) (uuid.UUID, error) {
	return uuid.Parse(uuidStr)
}

// parseModelConfiguration 解析模型配置
// 从 ModelConfiguration 中提取并解析 GenkitConfig
func (c *client) parseModelConfiguration(ctx context.Context, modelConfig interface{}) (*GenkitConfig, error) {
	// 使用类型断言获取实际的配置对象
	type ModelConfig interface {
		GetModel() string
		GetQueryParams() *string
	}

	// 创建基础配置
	genkitConfig := &GenkitConfig{}

	// 使用反射或类型断言来获取字段
	// 这里我们假设传入的是 repository 返回的模型配置对象
	// 需要根据实际的 repository 接口来调整
	
	// 临时方案：使用 JSON 序列化/反序列化来转换
	configJSON, err := json.Marshal(modelConfig)
	if err != nil {
		logger.ErrorContext(ctx, "序列化模型配置失败", logger.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("序列化模型配置失败: %w", err)
	}

	var tempConfig struct {
		Model       string  `json:"model"`
		QueryParams *string `json:"queryParams"`
	}

	if err := json.Unmarshal(configJSON, &tempConfig); err != nil {
		logger.ErrorContext(ctx, "反序列化模型配置失败", logger.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("反序列化模型配置失败: %w", err)
	}

	// 设置基本字段
	genkitConfig.Model = tempConfig.Model

	// 解析 QueryParams（如果存在）
	if tempConfig.QueryParams != nil && *tempConfig.QueryParams != "" {
		if err := json.Unmarshal([]byte(*tempConfig.QueryParams), genkitConfig); err != nil {
			logger.ErrorContext(ctx, "解析 QueryParams 失败", logger.Fields{
				"queryParams": *tempConfig.QueryParams,
				"error":       err.Error(),
			})
			return nil, fmt.Errorf("解析 QueryParams 失败: %w", err)
		}
		
		// 确保 Model 字段不被 QueryParams 覆盖（除非 QueryParams 中明确指定）
		if genkitConfig.Model == "" {
			genkitConfig.Model = tempConfig.Model
		}
	}

	return genkitConfig, nil
}

// createAzurePlugin 创建 Azure OpenAI 插件
// 使用 OpenAI 插件 + 自定义 BaseURL 的方式集成 Azure OpenAI
// BaseURL 格式: https://{endpoint}/openai/deployments/{deployment}
func createAzurePlugin(ctx context.Context, apiKey string, genkitConfig *GenkitConfig) (*oai.OpenAI, error) {
	// 验证必需的配置字段
	if genkitConfig.AzureEndpoint == "" {
		logger.ErrorContext(ctx, "Azure OpenAI 配置缺少必需字段", logger.Fields{
			"missingField": "azureEndpoint",
		})
		return nil, fmt.Errorf("Azure OpenAI 配置缺少必需字段: azureEndpoint")
	}
	if genkitConfig.AzureDeployment == "" {
		logger.ErrorContext(ctx, "Azure OpenAI 配置缺少必需字段", logger.Fields{
			"missingField": "azureDeployment",
		})
		return nil, fmt.Errorf("Azure OpenAI 配置缺少必需字段: azureDeployment")
	}

	// 构建 Azure OpenAI 的 BaseURL
	// 格式: https://{endpoint}/openai/deployments/{deployment}
	baseURL := fmt.Sprintf("%s/openai/deployments/%s",
		genkitConfig.AzureEndpoint,
		genkitConfig.AzureDeployment,
	)

	// 创建 OpenAI 插件，配置 Azure 特定的 BaseURL
	plugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		},
	}

	return plugin, nil
}

// createBailianPlugin 创建阿里云百炼插件
// 百炼完全兼容 OpenAI API 规范，使用自定义的 BailianPlugin 封装
// 支持根据地域自动选择合适的 API 端点
func createBailianPlugin(ctx context.Context, apiKey string, genkitConfig *GenkitConfig) (*oai.OpenAI, error) {
	// 导入百炼插件包
	// 注意：这里我们直接使用 OpenAI 插件，因为百炼完全兼容 OpenAI API
	// BailianPlugin 主要用于配置验证和端点选择
	
	// 确定 Endpoint
	endpoint := genkitConfig.BailianEndpoint
	if endpoint == "" {
		// 使用默认的北京地域端点
		endpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		logger.InfoContext(ctx, "使用默认百炼端点", logger.Fields{
			"endpoint": endpoint,
		})
	}
	
	// 创建 OpenAI 插件，配置百炼特定的 BaseURL
	plugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL(endpoint),
		},
	}
	
	return plugin, nil
}

// initializeProvider 根据提供商类型初始化 Genkit 实例
// 支持的提供商：
// - googlegenai: Google AI (Gemini)
// - openai: OpenAI
// - azureopenai: Azure OpenAI (使用 OpenAI 插件 + Azure BaseURL)
// - bianlian: 阿里云百炼 (使用 OpenAI 插件 + 百炼兼容模式 BaseURL)
// - anthropic: Anthropic (Claude)
// - custom_openai: 自定义 OpenAI 兼容服务
func (c *client) initializeProvider(ctx context.Context, modelConfig interface{}, genkitConfig *GenkitConfig) (*genkit.Genkit, error) {
	// 提取模型配置的基本信息
	configJSON, err := json.Marshal(modelConfig)
	if err != nil {
		logger.ErrorContext(ctx, "序列化模型配置失败", logger.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("序列化模型配置失败: %w", err)
	}

	var tempConfig struct {
		ModelProvider string  `json:"modelProvider"`
		APIKey        string  `json:"apiKey"`
		BaseURL       *string `json:"baseUrl"`
	}

	if err := json.Unmarshal(configJSON, &tempConfig); err != nil {
		logger.ErrorContext(ctx, "反序列化模型配置失败", logger.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("反序列化模型配置失败: %w", err)
	}

	logger.InfoContext(ctx, "开始初始化提供商", logger.Fields{
		"provider": tempConfig.ModelProvider,
		"model":    genkitConfig.Model,
	})

	var fullModelName string
	var g *genkit.Genkit

	switch tempConfig.ModelProvider {
	case "googlegenai":
		// Google AI (Gemini) 插件
		logger.InfoContext(ctx, "初始化 Google AI 提供商", logger.Fields{
			"provider": "googlegenai",
			"model":    genkitConfig.Model,
		})
		
		plugin := &googlegenai.GoogleAI{
			APIKey: tempConfig.APIKey,
		}
		fullModelName = "googleai/" + genkitConfig.Model
		
		// 初始化 Genkit 实例
		g = genkit.Init(ctx,
			genkit.WithPlugins(plugin),
			genkit.WithDefaultModel(fullModelName),
		)
		
		logger.InfoContext(ctx, "Google AI 提供商初始化成功", logger.Fields{
			"provider":      "googlegenai",
			"fullModelName": fullModelName,
		})

	case "openai":
		// OpenAI 插件
		logger.InfoContext(ctx, "初始化 OpenAI 提供商", logger.Fields{
			"provider": "openai",
			"model":    genkitConfig.Model,
			"hasCustomBaseURL": tempConfig.BaseURL != nil && *tempConfig.BaseURL != "",
		})
		
		opts := []option.RequestOption{
			option.WithAPIKey(tempConfig.APIKey),
		}
		
		// 如果配置了自定义 BaseURL，添加到选项中
		if tempConfig.BaseURL != nil && *tempConfig.BaseURL != "" {
			opts = append(opts, option.WithBaseURL(*tempConfig.BaseURL))
			logger.DebugContext(ctx, "使用自定义 BaseURL", logger.Fields{
				"baseURL": *tempConfig.BaseURL,
			})
		}
		
		plugin := &oai.OpenAI{
			Opts: opts,
		}
		fullModelName = "openai/" + genkitConfig.Model
		
		// 初始化 Genkit 实例
		g = genkit.Init(ctx,
			genkit.WithPlugins(plugin),
			genkit.WithDefaultModel(fullModelName),
		)
		
		logger.InfoContext(ctx, "OpenAI 提供商初始化成功", logger.Fields{
			"provider":      "openai",
			"fullModelName": fullModelName,
		})

	case "azureopenai":
		// Azure OpenAI (使用 OpenAI 插件 + Azure BaseURL)
		logger.InfoContext(ctx, "初始化 Azure OpenAI 提供商", logger.Fields{
			"provider":        "azureopenai",
			"model":           genkitConfig.Model,
			"azureEndpoint":   genkitConfig.AzureEndpoint,
			"azureDeployment": genkitConfig.AzureDeployment,
		})
		
		plugin, err := createAzurePlugin(ctx, tempConfig.APIKey, genkitConfig)
		if err != nil {
			logger.ErrorContext(ctx, "创建 Azure OpenAI 插件失败", logger.Fields{
				"provider": "azureopenai",
				"error":    err.Error(),
			})
			return nil, fmt.Errorf("创建 Azure OpenAI 插件失败: %w", err)
		}
		
		fullModelName = "openai/" + genkitConfig.Model
		
		// 初始化 Genkit 实例
		g = genkit.Init(ctx,
			genkit.WithPlugins(plugin),
			genkit.WithDefaultModel(fullModelName),
		)
		
		logger.InfoContext(ctx, "Azure OpenAI 提供商初始化成功", logger.Fields{
			"provider":      "azureopenai",
			"fullModelName": fullModelName,
		})

	case "bianlian":
		// 阿里云百炼 (使用 OpenAI 插件 + 百炼兼容模式 BaseURL)
		// 百炼提供 OpenAI 兼容接口
		logger.InfoContext(ctx, "初始化阿里云百炼提供商", logger.Fields{
			"provider":        "bianlian",
			"model":           genkitConfig.Model,
			"bailianEndpoint": genkitConfig.BailianEndpoint,
		})
		
		plugin, err := createBailianPlugin(ctx, tempConfig.APIKey, genkitConfig)
		if err != nil {
			logger.ErrorContext(ctx, "创建百炼插件失败", logger.Fields{
				"provider": "bianlian",
				"error":    err.Error(),
			})
			return nil, fmt.Errorf("创建百炼插件失败: %w", err)
		}
		
		fullModelName = "openai/" + genkitConfig.Model
		
		// 初始化 Genkit 实例
		g = genkit.Init(ctx,
			genkit.WithPlugins(plugin),
			genkit.WithDefaultModel(fullModelName),
		)
		
		logger.InfoContext(ctx, "阿里云百炼提供商初始化成功", logger.Fields{
			"provider":      "bianlian",
			"fullModelName": fullModelName,
		})

	case "anthropic":
		// Anthropic (Claude) 插件
		// Anthropic 插件使用环境变量 ANTHROPIC_API_KEY 或通过 Opts 设置
		logger.InfoContext(ctx, "初始化 Anthropic 提供商", logger.Fields{
			"provider": "anthropic",
			"model":    genkitConfig.Model,
		})
		
		plugin := &anthropic.Anthropic{
			Opts: []option.RequestOption{
				option.WithAPIKey(tempConfig.APIKey),
			},
		}
		fullModelName = "anthropic/" + genkitConfig.Model
		
		// 初始化 Genkit 实例
		g = genkit.Init(ctx,
			genkit.WithPlugins(plugin),
			genkit.WithDefaultModel(fullModelName),
		)
		
		logger.InfoContext(ctx, "Anthropic 提供商初始化成功", logger.Fields{
			"provider":      "anthropic",
			"fullModelName": fullModelName,
		})

	case "custom_openai":
		// 自定义 OpenAI 兼容服务
		// 必须提供 BaseURL
		logger.InfoContext(ctx, "初始化自定义 OpenAI 提供商", logger.Fields{
			"provider": "custom_openai",
			"model":    genkitConfig.Model,
		})
		
		if tempConfig.BaseURL == nil || *tempConfig.BaseURL == "" {
			logger.ErrorContext(ctx, "自定义 OpenAI 提供商缺少 BaseURL", logger.Fields{
				"provider": "custom_openai",
			})
			return nil, fmt.Errorf("自定义 OpenAI 提供商必须指定 baseUrl")
		}

		logger.DebugContext(ctx, "使用自定义 BaseURL", logger.Fields{
			"baseURL": *tempConfig.BaseURL,
		})

		plugin := &oai.OpenAI{
			Opts: []option.RequestOption{
				option.WithAPIKey(tempConfig.APIKey),
				option.WithBaseURL(*tempConfig.BaseURL),
			},
		}
		fullModelName = "openai/" + genkitConfig.Model
		
		// 初始化 Genkit 实例
		g = genkit.Init(ctx,
			genkit.WithPlugins(plugin),
			genkit.WithDefaultModel(fullModelName),
		)
		
		logger.InfoContext(ctx, "自定义 OpenAI 提供商初始化成功", logger.Fields{
			"provider":      "custom_openai",
			"fullModelName": fullModelName,
		})

	default:
		logger.ErrorContext(ctx, "不支持的提供商类型", logger.Fields{
			"provider": tempConfig.ModelProvider,
		})
		return nil, fmt.Errorf("不支持的提供商类型: %s", tempConfig.ModelProvider)
	}

	return g, nil
}

// Generate 生成内容（根据租户ID和模型名称）
func (c *client) Generate(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (*GenerateResult, error) {
	startTime := time.Now()
	
	// 参数验证
	if tenantID == "" {
		logger.ErrorContext(ctx, "租户ID不能为空", logger.Fields{})
		return nil, fmt.Errorf("租户ID不能为空")
	}

	if modelName == "" {
		logger.ErrorContext(ctx, "模型名称不能为空", logger.Fields{
			"tenantId": tenantID,
		})
		return nil, fmt.Errorf("模型名称不能为空")
	}

	if prompt == "" {
		logger.ErrorContext(ctx, "提示词不能为空", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
		})
		return nil, fmt.Errorf("提示词不能为空")
	}

	// TraceID 会自动从 context 中提取并记录到日志中
	logger.InfoContext(ctx, "开始生成内容", logger.Fields{
		"tenantId":   tenantID,
		"modelName":  modelName,
		"promptLen":  len(prompt),
	})

	// 获取或初始化 Genkit 实例
	g, genkitConfig, err := c.getOrInitGenkit(ctx, tenantID, modelName)
	if err != nil {
		// 错误处理：配置不存在或模型禁用
		logger.ErrorContext(ctx, "获取模型实例失败", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("获取模型实例失败: %w", err)
	}

	// 调用 Genkit 生成
	// 注意：当前简化实现，暂不支持自定义 temperature、maxTokens 等参数
	// 这些参数可以通过 genkit.WithDefaultModel 在初始化时设置
	resp, err := genkit.Generate(ctx, g, ai.WithPrompt(prompt))
	if err != nil {
		duration := time.Since(startTime)
		logger.ErrorContext(ctx, "生成内容失败", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
			"model":     genkitConfig.Model,
			"duration":  duration.String(),
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("生成内容失败: %w", err)
	}

	// 构建结果
	result := &GenerateResult{
		Text:  resp.Text(),
		Model: genkitConfig.Model,
	}

	// 提取 token 使用情况
	if resp.Usage != nil {
		result.Usage = &Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
	}

	duration := time.Since(startTime)
	// TraceID 会自动从 context 中提取并记录到日志中
	logger.InfoContext(ctx, "生成内容成功", logger.Fields{
		"tenantId":         tenantID,
		"modelName":        modelName,
		"model":            genkitConfig.Model,
		"duration":         duration.String(),
		"durationMs":       duration.Milliseconds(),
		"promptTokens":     result.Usage.PromptTokens,
		"completionTokens": result.Usage.CompletionTokens,
		"totalTokens":      result.Usage.TotalTokens,
		"responseLen":      len(result.Text),
	})

	return result, nil
}

// GenerateStream 流式生成内容（根据租户ID和模型名称）
func (c *client) GenerateStream(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (<-chan StreamChunk, error) {
	startTime := time.Now()
	
	// 参数验证
	if tenantID == "" {
		logger.ErrorContext(ctx, "租户ID不能为空", logger.Fields{})
		return nil, fmt.Errorf("租户ID不能为空")
	}

	if modelName == "" {
		logger.ErrorContext(ctx, "模型名称不能为空", logger.Fields{
			"tenantId": tenantID,
		})
		return nil, fmt.Errorf("模型名称不能为空")
	}

	if prompt == "" {
		logger.ErrorContext(ctx, "提示词不能为空", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
		})
		return nil, fmt.Errorf("提示词不能为空")
	}

	// TraceID 会自动从 context 中提取并记录到日志中
	logger.InfoContext(ctx, "开始流式生成内容", logger.Fields{
		"tenantId":   tenantID,
		"modelName":  modelName,
		"promptLen":  len(prompt),
	})

	// 获取或初始化 Genkit 实例
	g, genkitConfig, err := c.getOrInitGenkit(ctx, tenantID, modelName)
	if err != nil {
		// 错误处理：配置不存在或模型禁用
		logger.ErrorContext(ctx, "获取模型实例失败", logger.Fields{
			"tenantId":  tenantID,
			"modelName": modelName,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("获取模型实例失败: %w", err)
	}

	logger.InfoContext(ctx, "开始流式调用", logger.Fields{
		"tenantId":  tenantID,
		"modelName": modelName,
		"model":     genkitConfig.Model,
	})

	// 创建流式响应通道
	streamChan := make(chan StreamChunk, 10)

	// 在 goroutine 中处理流式响应
	go func() {
		defer close(streamChan)

		var chunkCount int
		var totalContent string
		firstChunkTime := time.Time{}

		// 调用 Genkit 流式生成，使用 WithStreaming 回调处理每个 chunk
		resp, err := genkit.Generate(ctx, g,
			ai.WithPrompt(prompt),
			ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
				// 记录首字节时间
				if chunkCount == 0 {
					firstChunkTime = time.Now()
					ttfb := firstChunkTime.Sub(startTime)
					logger.InfoContext(ctx, "收到首个响应块", logger.Fields{
						"tenantId":  tenantID,
						"modelName": modelName,
						"model":     genkitConfig.Model,
						"ttfb":      ttfb.String(),
					})
				}
				
				chunkCount++
				totalContent += chunk.Text()
				
				// 发送流式数据块
				select {
				case streamChan <- StreamChunk{
					Content: chunk.Text(),
					Done:    false,
				}:
				case <-ctx.Done():
					logger.WarnContext(ctx, "流式生成被取消", logger.Fields{
						"tenantId":   tenantID,
						"modelName":  modelName,
						"chunkCount": chunkCount,
					})
					return ctx.Err()
				}
				return nil
			}),
		)

		if err != nil {
			duration := time.Since(startTime)
			logger.ErrorContext(ctx, "流式生成失败", logger.Fields{
				"tenantId":   tenantID,
				"modelName":  modelName,
				"model":      genkitConfig.Model,
				"duration":   duration.String(),
				"chunkCount": chunkCount,
				"error":      err.Error(),
			})
			
			// 发送错误
			streamChan <- StreamChunk{
				Content: "",
				Done:    true,
				Error:   err,
			}
			return
		}

		// 发送完成标记，包含最终的使用统计
		var usage *Usage
		if resp.Usage != nil {
			usage = &Usage{
				PromptTokens:     int(resp.Usage.InputTokens),
				CompletionTokens: int(resp.Usage.OutputTokens),
				TotalTokens:      int(resp.Usage.TotalTokens),
			}
		}

		streamChan <- StreamChunk{
			Content: "",
			Done:    true,
			Model:   genkitConfig.Model,
			Usage:   usage,
		}

		duration := time.Since(startTime)
		var ttfb time.Duration
		if !firstChunkTime.IsZero() {
			ttfb = firstChunkTime.Sub(startTime)
		}
		
		// TraceID 会自动从 context 中提取并记录到日志中
		logger.InfoContext(ctx, "流式生成完成", logger.Fields{
			"tenantId":         tenantID,
			"modelName":        modelName,
			"model":            genkitConfig.Model,
			"duration":         duration.String(),
			"durationMs":       duration.Milliseconds(),
			"ttfb":             ttfb.String(),
			"ttfbMs":           ttfb.Milliseconds(),
			"chunkCount":       chunkCount,
			"totalContentLen":  len(totalContent),
			"promptTokens":     usage.PromptTokens,
			"completionTokens": usage.CompletionTokens,
			"totalTokens":      usage.TotalTokens,
		})
	}()

	return streamChan, nil
}

// GetGenkit 获取底层的Genkit实例
func (c *client) GetGenkit() *genkit.Genkit {
	return c.g
}

// Close 关闭客户端
func (c *client) Close() error {
	// 清理所有缓存的实例
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// 清空缓存
	c.instances = make(map[string]*genkit.Genkit)
	
	return nil
}

// ClearCache 清理指定租户和模型的缓存实例
// 用于配置更新后强制重新初始化
func (c *client) ClearCache(tenantID, modelName string) {
	cacheKey := fmt.Sprintf("%s_%s", tenantID, modelName)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.instances, cacheKey)
}

// ClearAllCache 清理所有缓存实例
func (c *client) ClearAllCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.instances = make(map[string]*genkit.Genkit)
}

// GetCacheSize 获取当前缓存的实例数量
func (c *client) GetCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.instances)
}
