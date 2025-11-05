package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// YAMLConfig YAML配置文件结构
type YAMLConfig struct {
	Server     YAMLServerConfig     `yaml:"server"`
	Genkit     YAMLGenkitConfig     `yaml:"genkit"`
	Database   YAMLDatabaseConfig   `yaml:"database"`
	Redis      YAMLRedisConfig      `yaml:"redis"`
	Log        YAMLLogConfig        `yaml:"log"`
	Session    YAMLSessionConfig    `yaml:"session"`
	Models     YAMLModelsConfig     `yaml:"models"`
	Auth       YAMLAuthConfig       `yaml:"auth"`
	Bootstrap  YAMLBootstrapConfig  `yaml:"bootstrap"`
	Monitoring YAMLMonitoringConfig `yaml:"monitoring"`
	Cache      YAMLCacheConfig      `yaml:"cache"`
	Vector     YAMLVectorConfig     `yaml:"vector"`
}

// YAMLServerConfig 服务器配置
type YAMLServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
	Mode string `yaml:"mode"` // debug, release
}

// YAMLGenkitConfig Genkit配置
type YAMLGenkitConfig struct {
	Provider           string  `yaml:"provider"`
	APIKey             string  `yaml:"api_key"`
	Model              string  `yaml:"model"`
	DefaultTemperature float64 `yaml:"default_temperature"`
	DefaultMaxTokens   int     `yaml:"default_max_tokens"`
	LogLevel           string  `yaml:"log_level"`
	Timeout            string  `yaml:"timeout"`
}

// YAMLDatabaseConfig 数据库配置
type YAMLDatabaseConfig struct {
	URL             string `yaml:"url"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Database        string `yaml:"database"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	SSLMode         string `yaml:"ssl_mode"`
	MaxConnections  int    `yaml:"max_connections"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
	LogLevel        string `yaml:"log_level"`
}

// YAMLRedisConfig Redis配置
type YAMLRedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
	Enabled  bool   `yaml:"enabled"`
}

// YAMLLogConfig 日志配置
type YAMLLogConfig struct {
	Level         string `yaml:"level"`
	Format        string `yaml:"format"`
	EnableFile    bool   `yaml:"enable_file"`
	LogDir        string `yaml:"log_dir"`
	EnableConsole bool   `yaml:"enable_console"`
}

// YAMLSessionConfig 会话配置
type YAMLSessionConfig struct {
	Timeout          string `yaml:"timeout"`
	CleanupInterval  string `yaml:"cleanup_interval"`
	SummaryThreshold int    `yaml:"summary_threshold"`
	DefaultPageSize  int    `yaml:"default_page_size"`
	MaxPageSize      int    `yaml:"max_page_size"`
	MaxTitleLength   int    `yaml:"max_title_length"`
}

// YAMLModelsConfig 模型配置
type YAMLModelsConfig struct {
	Dir string `yaml:"dir"`
}

// YAMLAuthConfig 认证配置
type YAMLAuthConfig struct {
	JWTSecret              string `yaml:"jwt_secret"`
	JWTIssuer              string `yaml:"jwt_issuer"`
	JWTAudience            string `yaml:"jwt_audience"`
	AccessTokenTTL         string `yaml:"access_token_ttl"`
	RefreshTokenTTL        string `yaml:"refresh_token_ttl"`
	BcryptCost             int    `yaml:"bcrypt_cost"`
	MaxLoginAttempts       int    `yaml:"max_login_attempts"`
	LoginAttemptWindow     string `yaml:"login_attempt_window"`
	PasswordMinLength      int    `yaml:"password_min_length"`
	EnableRefreshRotation  bool   `yaml:"enable_refresh_rotation"`
	TenantIdentifyStrategy string `yaml:"tenant_identify_strategy"`
	TokenCleanupInterval   string `yaml:"token_cleanup_interval"`
	EnableTokenBlacklist   bool   `yaml:"enable_token_blacklist"`
}

// YAMLBootstrapConfig 系统初始化配置
type YAMLBootstrapConfig struct {
	AdminEmail       string `yaml:"admin_email"`
	AdminPassword    string `yaml:"admin_password"`
	AdminDisplayName string `yaml:"admin_display_name"`
	TenantName       string `yaml:"tenant_name"`
	TenantDomain     string `yaml:"tenant_domain"`
}

// YAMLMonitoringConfig 监控配置
type YAMLMonitoringConfig struct {
	PrometheusPort  int    `yaml:"prometheus_port"`
	JaegerEndpoint  string `yaml:"jaeger_endpoint"`
	EnableTracing   bool   `yaml:"enable_tracing"`
	EnableMetrics   bool   `yaml:"enable_metrics"`
	MetricsPath     string `yaml:"metrics_path"`
	TracingSampling float64 `yaml:"tracing_sampling"`
}

// YAMLCacheConfig 缓存配置
type YAMLCacheConfig struct {
	Namespace           string            `yaml:"namespace"`
	DefaultTTL          string            `yaml:"default_ttl"`
	EnableWarmup        bool              `yaml:"enable_warmup"`
	WarmupInterval      string            `yaml:"warmup_interval"`
	ContextTTL          string            `yaml:"context_ttl"`
	VectorSearchTTL     string            `yaml:"vector_search_ttl"`
	SummaryTTL          string            `yaml:"summary_ttl"`
	SessionListTTL      string            `yaml:"session_list_ttl"`
	TokenUsageTTL       string            `yaml:"token_usage_ttl"`
	CustomTTL           map[string]string `yaml:"custom_ttl"`
}

// YAMLVectorConfig 向量服务配置
type YAMLVectorConfig struct {
	Provider       string `yaml:"provider"` // google, openai
	EmbeddingModel string `yaml:"embedding_model"`
	Dimension      int    `yaml:"dimension"`
	BatchSize      int    `yaml:"batch_size"`
	Timeout        string `yaml:"timeout"`
}

// LoadFromYAML 从YAML文件加载配置
func LoadFromYAML(configPath string) (*Config, error) {
	// 读取YAML文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 替换环境变量
	dataStr := replaceEnvVars(string(data))

	// 解析YAML
	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal([]byte(dataStr), &yamlConfig); err != nil {
		return nil, fmt.Errorf("解析YAML配置失败: %w", err)
	}

	// 转换为Config结构
	config, err := convertYAMLToConfig(&yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("转换配置失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// replaceEnvVars 替换配置中的环境变量
// 支持格式: ${VAR_NAME} 或 ${VAR_NAME:default_value}
func replaceEnvVars(content string) string {
	// 匹配 ${VAR_NAME} 或 ${VAR_NAME:default}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取变量名和默认值
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		
		varName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}
		
		// 获取环境变量
		value := os.Getenv(varName)
		if value == "" {
			return defaultValue
		}
		
		return value
	})
}

// convertYAMLToConfig 将YAML配置转换为Config结构
func convertYAMLToConfig(yamlConfig *YAMLConfig) (*Config, error) {
	config := &Config{}

	// 转换服务器配置
	config.Server = ServerConfig{
		Port: yamlConfig.Server.Port,
		Host: yamlConfig.Server.Host,
	}

	// 转换Genkit配置
	config.Genkit = GenkitConfig{
		APIKey:             yamlConfig.Genkit.APIKey,
		Model:              yamlConfig.Genkit.Model,
		DefaultTemperature: yamlConfig.Genkit.DefaultTemperature,
		DefaultMaxTokens:   yamlConfig.Genkit.DefaultMaxTokens,
	}

	// 转换数据库配置
	connMaxLifetime, err := parseDuration(yamlConfig.Database.ConnMaxLifetime, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("解析数据库连接最大生命周期失败: %w", err)
	}

	config.Database = DatabaseConfig{
		URL:             yamlConfig.Database.URL,
		Host:            yamlConfig.Database.Host,
		Port:            fmt.Sprintf("%d", yamlConfig.Database.Port),
		User:            yamlConfig.Database.User,
		Password:        yamlConfig.Database.Password,
		DBName:          yamlConfig.Database.Database,
		SSLMode:         yamlConfig.Database.SSLMode,
		MaxOpenConns:    yamlConfig.Database.MaxConnections,
		MaxIdleConns:    yamlConfig.Database.MaxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
		LogLevel:        yamlConfig.Database.LogLevel,
	}

	// 转换Redis配置
	config.Redis = RedisConfig{
		Host:     yamlConfig.Redis.Host,
		Port:     fmt.Sprintf("%d", yamlConfig.Redis.Port),
		Password: yamlConfig.Redis.Password,
		DB:       yamlConfig.Redis.Database,
		Enabled:  yamlConfig.Redis.Enabled,
	}

	// 转换日志配置
	config.Log = LogConfig{
		Level:         yamlConfig.Log.Level,
		Format:        yamlConfig.Log.Format,
		EnableFile:    yamlConfig.Log.EnableFile,
		LogDir:        yamlConfig.Log.LogDir,
		EnableConsole: yamlConfig.Log.EnableConsole,
	}

	// 转换会话配置
	timeout, err := parseDuration(yamlConfig.Session.Timeout, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("解析会话超时时间失败: %w", err)
	}

	cleanupInterval, err := parseDuration(yamlConfig.Session.CleanupInterval, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("解析会话清理间隔失败: %w", err)
	}

	config.Session = SessionConfig{
		Timeout:          timeout,
		CleanupInterval:  cleanupInterval,
		SummaryThreshold: yamlConfig.Session.SummaryThreshold,
		DefaultPageSize:  yamlConfig.Session.DefaultPageSize,
		MaxPageSize:      yamlConfig.Session.MaxPageSize,
		MaxTitleLength:   yamlConfig.Session.MaxTitleLength,
	}

	// 转换模型配置
	config.Models = ModelsConfig{
		Dir: yamlConfig.Models.Dir,
	}

	// 转换认证配置
	accessTokenTTL, err := parseDuration(yamlConfig.Auth.AccessTokenTTL, 60*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("解析Access Token TTL失败: %w", err)
	}

	refreshTokenTTL, err := parseDuration(yamlConfig.Auth.RefreshTokenTTL, 30*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("解析Refresh Token TTL失败: %w", err)
	}

	loginAttemptWindow, err := parseDuration(yamlConfig.Auth.LoginAttemptWindow, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("解析登录尝试时间窗口失败: %w", err)
	}

	tokenCleanupInterval, err := parseDuration(yamlConfig.Auth.TokenCleanupInterval, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("解析Token清理间隔失败: %w", err)
	}

	config.Auth = AuthConfig{
		JWTSecret:              yamlConfig.Auth.JWTSecret,
		JWTIssuer:              yamlConfig.Auth.JWTIssuer,
		JWTAudience:            yamlConfig.Auth.JWTAudience,
		AccessTokenTTL:         accessTokenTTL,
		RefreshTokenTTL:        refreshTokenTTL,
		BcryptCost:             yamlConfig.Auth.BcryptCost,
		MaxLoginAttempts:       yamlConfig.Auth.MaxLoginAttempts,
		LoginAttemptWindow:     loginAttemptWindow,
		PasswordMinLength:      yamlConfig.Auth.PasswordMinLength,
		EnableRefreshRotation:  yamlConfig.Auth.EnableRefreshRotation,
		TenantIdentifyStrategy: yamlConfig.Auth.TenantIdentifyStrategy,
		TokenCleanupInterval:   tokenCleanupInterval,
		EnableTokenBlacklist:   yamlConfig.Auth.EnableTokenBlacklist,
	}

	// 转换系统初始化配置
	config.Bootstrap = BootstrapConfig{
		AdminEmail:       yamlConfig.Bootstrap.AdminEmail,
		AdminPassword:    yamlConfig.Bootstrap.AdminPassword,
		AdminDisplayName: yamlConfig.Bootstrap.AdminDisplayName,
		TenantName:       yamlConfig.Bootstrap.TenantName,
		TenantDomain:     yamlConfig.Bootstrap.TenantDomain,
	}

	return config, nil
}

// parseDuration 解析时间间隔字符串
func parseDuration(durationStr string, defaultValue time.Duration) (time.Duration, error) {
	if durationStr == "" {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return defaultValue, err
	}

	return duration, nil
}

// LoadConfigWithEnv 根据环境加载配置
// 优先级: 环境变量 > YAML配置文件 > 默认值
func LoadConfigWithEnv(env string) (*Config, error) {
	// 确定配置文件路径
	configPath := getConfigPath(env)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); err == nil {
		// 配置文件存在，从YAML加载
		return LoadFromYAML(configPath)
	}

	// 配置文件不存在，从环境变量加载
	return Load()
}

// getConfigPath 获取配置文件路径
func getConfigPath(env string) string {
	// 优先使用环境变量指定的配置文件
	if configPath := os.Getenv("CONFIG_FILE"); configPath != "" {
		return configPath
	}

	// 根据环境确定配置文件
	configDir := getEnv("CONFIG_DIR", "config")
	
	switch strings.ToLower(env) {
	case "production", "prod":
		return fmt.Sprintf("%s/prod.yaml", configDir)
	case "development", "dev":
		return fmt.Sprintf("%s/dev.yaml", configDir)
	case "test":
		return fmt.Sprintf("%s/test.yaml", configDir)
	default:
		return fmt.Sprintf("%s/config.yaml", configDir)
	}
}

// GetEnv 获取当前环境
func GetEnv() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development"
	}
	return env
}
