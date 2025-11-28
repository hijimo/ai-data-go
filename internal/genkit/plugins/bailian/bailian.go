package bailian

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/core/api"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

// BailianPlugin 阿里云百炼插件
// 百炼完全兼容 OpenAI API 规范，因此我们使用 OpenAI 插件作为底层实现
// 通过配置自定义 BaseURL 来调用百炼 API
type BailianPlugin struct {
	// APIKey 百炼 API 密钥
	APIKey string
	
	// Endpoint 百炼 API 端点
	// 默认: https://dashscope.aliyuncs.com/compatible-mode/v1
	// 新加坡地域: https://dashscope-intl.aliyuncs.com/compatible-mode/v1
	// 金融云: https://dashscope-finance.aliyuncs.com/compatible-mode/v1
	Endpoint string
	
	// Model 模型名称（如 qwen-plus, qwen-max, qwen-turbo）
	Model string
	
	// Region 地域（beijing, singapore, finance）
	// 可选，用于自动选择合适的 Endpoint
	Region string
	
	// oaiPlugin 底层的 OpenAI 插件实例
	oaiPlugin *oai.OpenAI
}

// Config 百炼插件配置
type Config struct {
	// APIKey 百炼 API 密钥（必需）
	APIKey string
	
	// Endpoint 百炼 API 端点（可选）
	// 如果未指定，将根据 Region 自动选择
	Endpoint string
	
	// Model 模型名称（必需）
	Model string
	
	// Region 地域（可选）
	// 可选值: beijing, singapore, finance
	// 默认: beijing
	Region string
}

// DefaultEndpoints 默认的百炼 API 端点
var DefaultEndpoints = map[string]string{
	"beijing":   "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"singapore": "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
	"finance":   "https://dashscope-finance.aliyuncs.com/compatible-mode/v1",
}

// NewBailianPlugin 创建新的百炼插件实例
func NewBailianPlugin(config *Config) (*BailianPlugin, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	
	if config.APIKey == "" {
		return nil, fmt.Errorf("API 密钥不能为空")
	}
	
	if config.Model == "" {
		return nil, fmt.Errorf("模型名称不能为空")
	}
	
	// 确定 Endpoint
	endpoint := config.Endpoint
	if endpoint == "" {
		// 根据 Region 选择默认 Endpoint
		region := config.Region
		if region == "" {
			region = "beijing" // 默认使用北京地域
		}
		
		var ok bool
		endpoint, ok = DefaultEndpoints[region]
		if !ok {
			return nil, fmt.Errorf("不支持的地域: %s", region)
		}
	}
	
	// 创建底层的 OpenAI 插件
	oaiPlugin := &oai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey(config.APIKey),
			option.WithBaseURL(endpoint),
		},
	}
	
	return &BailianPlugin{
		APIKey:    config.APIKey,
		Endpoint:  endpoint,
		Model:     config.Model,
		Region:    config.Region,
		oaiPlugin: oaiPlugin,
	}, nil
}

// Init 初始化插件
// 实现 genkit.Plugin 接口
func (p *BailianPlugin) Init(ctx context.Context) []api.Action {
	if p.oaiPlugin == nil {
		// 如果 OpenAI 插件未初始化，返回空的 Action 列表
		return []api.Action{}
	}
	
	// 委托给底层的 OpenAI 插件进行初始化
	// OpenAI 插件会注册模型和相关功能
	return p.oaiPlugin.Init(ctx)
}

// GetModel 获取模型名称
func (p *BailianPlugin) GetModel() string {
	return p.Model
}

// GetEndpoint 获取 API 端点
func (p *BailianPlugin) GetEndpoint() string {
	return p.Endpoint
}

// GetRegion 获取地域
func (p *BailianPlugin) GetRegion() string {
	if p.Region == "" {
		return "beijing"
	}
	return p.Region
}

// Validate 验证插件配置
func (p *BailianPlugin) Validate() error {
	if p.APIKey == "" {
		return fmt.Errorf("API 密钥不能为空")
	}
	
	if p.Endpoint == "" {
		return fmt.Errorf("API 端点不能为空")
	}
	
	if p.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	
	if p.oaiPlugin == nil {
		return fmt.Errorf("OpenAI 插件未初始化")
	}
	
	return nil
}

// 注意：generate 和 generateStream 方法不需要显式实现
// 实际的生成逻辑由底层的 OpenAI 插件处理
// 百炼 API 完全兼容 OpenAI 规范，因此无需额外转换
