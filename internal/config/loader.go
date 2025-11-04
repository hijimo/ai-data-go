package config

import (
	"fmt"
	"os"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	env string
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		env: GetEnv(),
	}
}

// Load 加载配置
// 优先级: 环境变量 > YAML配置文件 > 默认值
func (l *ConfigLoader) Load() (*Config, error) {
	// 尝试从YAML加载
	config, err := LoadConfigWithEnv(l.env)
	if err != nil {
		// YAML加载失败，尝试从环境变量加载
		fmt.Printf("警告: 无法从YAML加载配置: %v\n", err)
		fmt.Println("尝试从环境变量加载配置...")
		
		config, err = Load()
		if err != nil {
			return nil, fmt.Errorf("配置加载失败: %w", err)
		}
	}

	// 打印配置摘要
	l.printConfigSummary(config)

	return config, nil
}

// LoadWithPath 从指定路径加载配置
func (l *ConfigLoader) LoadWithPath(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	config, err := LoadFromYAML(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	l.printConfigSummary(config)

	return config, nil
}

// printConfigSummary 打印配置摘要
func (l *ConfigLoader) printConfigSummary(config *Config) {
	fmt.Println("========================================")
	fmt.Println("配置加载成功")
	fmt.Println("========================================")
	fmt.Printf("环境: %s\n", l.env)
	fmt.Printf("服务器: %s:%s\n", config.Server.Host, config.Server.Port)
	fmt.Printf("数据库: %s:%s/%s\n", config.Database.Host, config.Database.Port, config.Database.DBName)
	fmt.Printf("Redis: %s:%s (DB:%d)\n", config.Redis.Host, config.Redis.Port, config.Redis.DB)
	fmt.Printf("Genkit模型: %s\n", config.Genkit.Model)
	fmt.Printf("日志级别: %s\n", config.Log.Level)
	fmt.Printf("日志格式: %s\n", config.Log.Format)
	fmt.Println("========================================")
}

// GetEnv 获取当前环境
func (l *ConfigLoader) GetEnv() string {
	return l.env
}

// IsDevelopment 是否为开发环境
func (l *ConfigLoader) IsDevelopment() bool {
	return l.env == "development" || l.env == "dev"
}

// IsProduction 是否为生产环境
func (l *ConfigLoader) IsProduction() bool {
	return l.env == "production" || l.env == "prod"
}

// IsTest 是否为测试环境
func (l *ConfigLoader) IsTest() bool {
	return l.env == "test"
}

// ValidateConfig 验证配置完整性
func ValidateConfig(config *Config) error {
	if err := config.Validate(); err != nil {
		return err
	}

	// 额外的运行时验证
	if config.Database.MaxIdleConns > config.Database.MaxOpenConns {
		return fmt.Errorf("数据库最大空闲连接数(%d)不能大于最大打开连接数(%d)",
			config.Database.MaxIdleConns, config.Database.MaxOpenConns)
	}

	if config.Session.DefaultPageSize > config.Session.MaxPageSize {
		return fmt.Errorf("默认分页大小(%d)不能大于最大分页大小(%d)",
			config.Session.DefaultPageSize, config.Session.MaxPageSize)
	}

	return nil
}

// MustLoad 加载配置，失败则panic
func MustLoad() *Config {
	loader := NewConfigLoader()
	config, err := loader.Load()
	if err != nil {
		panic(fmt.Sprintf("配置加载失败: %v", err))
	}

	if err := ValidateConfig(config); err != nil {
		panic(fmt.Sprintf("配置验证失败: %v", err))
	}

	return config
}

// MustLoadWithPath 从指定路径加载配置，失败则panic
func MustLoadWithPath(configPath string) *Config {
	loader := NewConfigLoader()
	config, err := loader.LoadWithPath(configPath)
	if err != nil {
		panic(fmt.Sprintf("配置加载失败: %v", err))
	}

	if err := ValidateConfig(config); err != nil {
		panic(fmt.Sprintf("配置验证失败: %v", err))
	}

	return config
}
