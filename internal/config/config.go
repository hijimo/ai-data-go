package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config 应用配置结构
type Config struct {
	Server     ServerConfig
	Genkit     GenkitConfig
	Database   DatabaseConfig
	Log        LogConfig
	Session    SessionConfig
	Models     ModelsConfig
	Auth       AuthConfig
	Redis      RedisConfig
	Bootstrap  BootstrapConfig
	Encryption EncryptionConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string // 服务器端口
	Host string // 服务器主机地址
}

// GenkitConfig Genkit AI 配置
type GenkitConfig struct {
	APIKey             string  // API密钥
	Model              string  // 默认模型
	DefaultTemperature float64 // 默认温度参数
	DefaultMaxTokens   int     // 默认最大token数
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	URL             string        // 数据库连接 URL（优先使用，格式：postgres://user:pass@host:port/dbname?sslmode=disable）
	Host            string        // 数据库主机
	Port            string        // 数据库端口
	User            string        // 数据库用户名
	Password        string        // 数据库密码
	DBName          string        // 数据库名称
	SSLMode         string        // SSL模式
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
	LogLevel        string        // GORM 日志级别 (silent, error, warn, info)
}

// LogConfig 日志配置
type LogConfig struct {
	Level         string // 日志级别 (debug, info, warn, error)
	Format        string // 日志格式 (json, text)
	EnableFile    bool   // 是否启用文件日志
	LogDir        string // 日志文件目录
	EnableConsole bool   // 是否同时输出到控制台
}

// SessionConfig 会话配置
type SessionConfig struct {
	Timeout          time.Duration // 会话超时时间
	CleanupInterval  time.Duration // 会话清理间隔
	SummaryThreshold int           // 摘要生成阈值（消息数量）
	DefaultPageSize  int           // 默认分页大小
	MaxPageSize      int           // 最大分页大小
	MaxTitleLength   int           // 会话标题最大长度
}

// ModelsConfig 模型配置
type ModelsConfig struct {
	Dir string // 模型配置文件目录
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret              string        // JWT 签名密钥
	JWTIssuer              string        // JWT 签发者
	JWTAudience            string        // JWT 受众
	AccessTokenTTL         time.Duration // Access Token 生命周期
	RefreshTokenTTL        time.Duration // Refresh Token 生命周期
	BcryptCost             int           // Bcrypt cost factor
	MaxLoginAttempts       int           // 最大登录尝试次数
	LoginAttemptWindow     time.Duration // 登录尝试时间窗口
	PasswordMinLength      int           // 密码最小长度
	EnableRefreshRotation  bool          // 是否启用 Refresh Token 轮换
	TenantIdentifyStrategy string        // 租户识别策略：header, subdomain, path, cookie
	TokenCleanupInterval   time.Duration // Token 清理间隔
	EnableTokenBlacklist   bool          // 是否启用 Token 黑名单
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string // Redis 主机
	Port     string // Redis 端口
	Password string // Redis 密码
	DB       int    // Redis 数据库编号
	Enabled  bool   // 是否启用 Redis
}

// BootstrapConfig 系统初始化配置
type BootstrapConfig struct {
	AdminEmail       string // 平台管理员邮箱
	AdminPassword    string // 平台管理员初始密码（留空则自动生成）
	AdminDisplayName string // 平台管理员显示名称
	TenantName       string // 平台租户名称
	TenantDomain     string // 平台租户域名
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	SecretKey              string        // API密钥加密密钥（32字节）
	ProviderValidationTimeout time.Duration // 模型配置验证超时时间
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	// 尝试加载 .env 文件（如果存在）
	_ = godotenv.Load()

	config := &Config{}

	// 加载服务器配置
	config.Server = ServerConfig{
		Port: getEnv("SERVER_PORT", "8080"),
		Host: getEnv("SERVER_HOST", "0.0.0.0"),
	}

	// 加载 Genkit 配置
	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		// 尝试从其他可能的环境变量名获取
		apiKey = os.Getenv("AI_API_KEY")
	}
	
	config.Genkit = GenkitConfig{
		APIKey:             apiKey,
		Model:              getEnv("GENKIT_MODEL", "gemini-2.5-flash"),
		DefaultTemperature: getEnvFloat("GENKIT_DEFAULT_TEMPERATURE", 0.7),
		DefaultMaxTokens:   getEnvInt("GENKIT_DEFAULT_MAX_TOKENS", 2000),
	}

	// 加载数据库配置
	// 优先使用 DATABASE_URL，如果未设置则使用独立的配置项
	config.Database = DatabaseConfig{
		URL:             os.Getenv("DATABASE_URL"),
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        os.Getenv("DB_PASSWORD"),
		DBName:          getEnv("DB_NAME", "genkit_ai_service"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		LogLevel:        getEnv("DB_LOG_LEVEL", "warn"),
	}

	// 加载日志配置
	config.Log = LogConfig{
		Level:         getEnv("LOG_LEVEL", "info"),
		Format:        getEnv("LOG_FORMAT", "json"),
		EnableFile:    getEnvBool("LOG_ENABLE_FILE", true),
		LogDir:        getEnv("LOG_DIR", "logs"),
		EnableConsole: getEnvBool("LOG_ENABLE_CONSOLE", true),
	}

	// 加载会话配置
	config.Session = SessionConfig{
		Timeout:          getEnvDuration("SESSION_TIMEOUT", 30*time.Minute),
		CleanupInterval:  getEnvDuration("SESSION_CLEANUP_INTERVAL", 5*time.Minute),
		SummaryThreshold: getEnvInt("SESSION_SUMMARY_THRESHOLD", 50),
		DefaultPageSize:  getEnvInt("SESSION_DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:      getEnvInt("SESSION_MAX_PAGE_SIZE", 100),
		MaxTitleLength:   getEnvInt("SESSION_MAX_TITLE_LENGTH", 255),
	}

	// 加载模型配置
	config.Models = ModelsConfig{
		Dir: getEnv("MODELS_DIR", "./models"),
	}

	// 加载认证配置
	config.Auth = AuthConfig{
		JWTSecret:              getEnv("JWT_SECRET", ""),
		JWTIssuer:              getEnv("JWT_ISSUER", "genkit-ai-service"),
		JWTAudience:            getEnv("JWT_AUDIENCE", "genkit-api"),
		AccessTokenTTL:         getEnvDuration("ACCESS_TOKEN_TTL", 60*time.Minute),
		RefreshTokenTTL:        getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		BcryptCost:             getEnvInt("BCRYPT_COST", 12),
		MaxLoginAttempts:       getEnvInt("MAX_LOGIN_ATTEMPTS", 5),
		LoginAttemptWindow:     getEnvDuration("LOGIN_ATTEMPT_WINDOW", 15*time.Minute),
		PasswordMinLength:      getEnvInt("PASSWORD_MIN_LENGTH", 8),
		EnableRefreshRotation:  getEnvBool("ENABLE_REFRESH_ROTATION", true),
		TenantIdentifyStrategy: getEnv("TENANT_IDENTIFY_STRATEGY", "header"),
		TokenCleanupInterval:   getEnvDuration("TOKEN_CLEANUP_INTERVAL", 1*time.Hour),
		EnableTokenBlacklist:   getEnvBool("ENABLE_TOKEN_BLACKLIST", true),
	}

	// 加载 Redis 配置
	config.Redis = RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
		Enabled:  getEnvBool("REDIS_ENABLED", true),
	}

	// 加载系统初始化配置
	config.Bootstrap = BootstrapConfig{
		AdminEmail:       getEnv("PLATFORM_ADMIN_EMAIL", "admin@system.local"),
		AdminPassword:    getEnv("PLATFORM_ADMIN_PASSWORD", ""), // 留空则自动生成
		AdminDisplayName: getEnv("PLATFORM_ADMIN_NAME", "Platform Admin"),
		TenantName:       getEnv("PLATFORM_TENANT_NAME", "Platform"),
		TenantDomain:     getEnv("PLATFORM_TENANT_DOMAIN", "system.local"),
	}

	// 加载加密配置
	config.Encryption = EncryptionConfig{
		SecretKey:              getEnv("ENCRYPTION_SECRET_KEY", ""),
		ProviderValidationTimeout: getEnvDuration("PROVIDER_VALIDATION_TIMEOUT", 30*time.Second),
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}
	
	port, err := strconv.Atoi(c.Server.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("服务器端口必须是1-65535之间的有效数字")
	}

	// 验证 Genkit 配置
	if c.Genkit.APIKey == "" {
		return fmt.Errorf("Genkit API密钥不能为空 (GENKIT_API_KEY 或 AI_API_KEY)")
	}
	
	if c.Genkit.Model == "" {
		return fmt.Errorf("Genkit 模型不能为空")
	}
	
	if c.Genkit.DefaultTemperature < 0 || c.Genkit.DefaultTemperature > 2 {
		return fmt.Errorf("默认温度参数必须在0-2之间")
	}
	
	if c.Genkit.DefaultMaxTokens <= 0 {
		return fmt.Errorf("默认最大token数必须大于0")
	}

	// 验证数据库配置
	// 如果设置了 DATABASE_URL，则不需要验证独立配置项
	if c.Database.URL == "" {
		// 未设置 DATABASE_URL，验证独立配置项
		if c.Database.Host == "" {
			return fmt.Errorf("数据库主机不能为空（请设置 DATABASE_URL 或 DB_HOST）")
		}
		
		if c.Database.Port == "" {
			return fmt.Errorf("数据库端口不能为空（请设置 DATABASE_URL 或 DB_PORT）")
		}
		
		dbPort, err := strconv.Atoi(c.Database.Port)
		if err != nil || dbPort < 1 || dbPort > 65535 {
			return fmt.Errorf("数据库端口必须是1-65535之间的有效数字")
		}
		
		if c.Database.User == "" {
			return fmt.Errorf("数据库用户名不能为空（请设置 DATABASE_URL 或 DB_USER）")
		}
		
		if c.Database.DBName == "" {
			return fmt.Errorf("数据库名称不能为空（请设置 DATABASE_URL 或 DB_NAME）")
		}
	}
	
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("最大打开连接数必须大于0")
	}
	
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("最大空闲连接数不能为负数")
	}
	
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("最大空闲连接数不能大于最大打开连接数")
	}
	
	validDBLogLevels := map[string]bool{
		"silent": true,
		"error":  true,
		"warn":   true,
		"info":   true,
	}
	if !validDBLogLevels[c.Database.LogLevel] {
		return fmt.Errorf("数据库日志级别必须是 silent, error, warn 或 info 之一")
	}

	// 验证日志配置
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("日志级别必须是 debug, info, warn 或 error 之一")
	}
	
	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[c.Log.Format] {
		return fmt.Errorf("日志格式必须是 json 或 text")
	}

	// 验证会话配置
	if c.Session.Timeout <= 0 {
		return fmt.Errorf("会话超时时间必须大于0")
	}
	
	if c.Session.CleanupInterval <= 0 {
		return fmt.Errorf("会话清理间隔必须大于0")
	}
	
	if c.Session.SummaryThreshold <= 0 {
		return fmt.Errorf("摘要生成阈值必须大于0")
	}
	
	if c.Session.DefaultPageSize <= 0 {
		return fmt.Errorf("默认分页大小必须大于0")
	}
	
	if c.Session.MaxPageSize <= 0 {
		return fmt.Errorf("最大分页大小必须大于0")
	}
	
	if c.Session.MaxPageSize < c.Session.DefaultPageSize {
		return fmt.Errorf("最大分页大小不能小于默认分页大小")
	}
	
	if c.Session.MaxTitleLength <= 0 {
		return fmt.Errorf("会话标题最大长度必须大于0")
	}

	// 验证模型配置
	if c.Models.Dir == "" {
		return fmt.Errorf("模型目录不能为空")
	}

	// 验证认证配置
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT 签名密钥不能为空 (JWT_SECRET)")
	}

	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT 签名密钥长度必须至少为 32 个字符")
	}

	if c.Auth.JWTIssuer == "" {
		return fmt.Errorf("JWT 签发者不能为空")
	}

	if c.Auth.JWTAudience == "" {
		return fmt.Errorf("JWT 受众不能为空")
	}

	if c.Auth.AccessTokenTTL <= 0 {
		return fmt.Errorf("Access Token 生命周期必须大于0")
	}

	if c.Auth.RefreshTokenTTL <= 0 {
		return fmt.Errorf("Refresh Token 生命周期必须大于0")
	}

	if c.Auth.BcryptCost < 4 || c.Auth.BcryptCost > 31 {
		return fmt.Errorf("Bcrypt cost factor 必须在 4-31 之间")
	}

	if c.Auth.MaxLoginAttempts <= 0 {
		return fmt.Errorf("最大登录尝试次数必须大于0")
	}

	if c.Auth.LoginAttemptWindow <= 0 {
		return fmt.Errorf("登录尝试时间窗口必须大于0")
	}

	if c.Auth.PasswordMinLength < 6 {
		return fmt.Errorf("密码最小长度必须至少为6")
	}

	validStrategies := map[string]bool{
		"header":    true,
		"subdomain": true,
		"path":      true,
		"cookie":    true,
	}
	if !validStrategies[c.Auth.TenantIdentifyStrategy] {
		return fmt.Errorf("租户识别策略必须是 header, subdomain, path 或 cookie 之一")
	}

	// 验证系统初始化配置
	if c.Bootstrap.AdminEmail == "" {
		return fmt.Errorf("平台管理员邮箱不能为空 (PLATFORM_ADMIN_EMAIL)")
	}

	// 验证邮箱格式
	if !isValidEmail(c.Bootstrap.AdminEmail) {
		return fmt.Errorf("平台管理员邮箱格式无效: %s", c.Bootstrap.AdminEmail)
	}

	// 如果提供了密码，验证密码强度
	if c.Bootstrap.AdminPassword != "" {
		if len(c.Bootstrap.AdminPassword) < c.Auth.PasswordMinLength {
			return fmt.Errorf("平台管理员密码长度必须至少为 %d 个字符", c.Auth.PasswordMinLength)
		}
		
		// 验证密码复杂度（至少包含大小写字母、数字）
		if !isStrongPassword(c.Bootstrap.AdminPassword) {
			return fmt.Errorf("平台管理员密码必须包含大写字母、小写字母和数字")
		}
	}

	if c.Bootstrap.AdminDisplayName == "" {
		return fmt.Errorf("平台管理员显示名称不能为空 (PLATFORM_ADMIN_NAME)")
	}

	if c.Bootstrap.TenantName == "" {
		return fmt.Errorf("平台租户名称不能为空 (PLATFORM_TENANT_NAME)")
	}

	if c.Bootstrap.TenantDomain == "" {
		return fmt.Errorf("平台租户域名不能为空 (PLATFORM_TENANT_DOMAIN)")
	}

	// 验证加密配置
	if c.Encryption.SecretKey == "" {
		return fmt.Errorf("加密密钥不能为空 (ENCRYPTION_SECRET_KEY)")
	}

	if len(c.Encryption.SecretKey) < 32 {
		return fmt.Errorf("加密密钥长度必须至少为 32 个字符")
	}

	if c.Encryption.ProviderValidationTimeout <= 0 {
		return fmt.Errorf("模型配置验证超时时间必须大于0")
	}

	return nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// getEnvFloat 获取浮点数类型的环境变量
func getEnvFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// getEnvDuration 获取时间间隔类型的环境变量
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// GetDSN 获取数据库连接字符串
// 优先使用 DATABASE_URL，如果未设置则根据独立配置项构建 DSN
func (c *DatabaseConfig) GetDSN() string {
	// 如果设置了 DATABASE_URL，直接使用
	if c.URL != "" {
		return c.URL
	}
	
	// 否则根据独立配置项构建 DSN
	// 格式：host=localhost user=postgres password=secret dbname=mydb port=5432 sslmode=disable
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host,
		c.User,
		c.Password,
		c.DBName,
		c.Port,
		c.SSLMode,
	)
	
	return dsn
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	// 格式：xxx@xxx.xxx
	if len(email) < 5 || len(email) > 320 {
		return false
	}
	
	// 必须包含 @ 符号
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				// 不能有多个 @ 符号
				return false
			}
			atIndex = i
		}
	}
	
	if atIndex <= 0 || atIndex >= len(email)-1 {
		return false
	}
	
	// @ 后面必须有点号
	domain := email[atIndex+1:]
	dotIndex := -1
	for i, c := range domain {
		if c == '.' {
			dotIndex = i
			break
		}
	}
	
	if dotIndex <= 0 || dotIndex >= len(domain)-1 {
		return false
	}
	
	return true
}

// isStrongPassword 验证密码强度
// 要求：至少包含一个大写字母、一个小写字母和一个数字
func isStrongPassword(password string) bool {
	hasUpper := false
	hasLower := false
	hasDigit := false
	
	for _, c := range password {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		} else if c >= 'a' && c <= 'z' {
			hasLower = true
		} else if c >= '0' && c <= '9' {
			hasDigit = true
		}
		
		if hasUpper && hasLower && hasDigit {
			return true
		}
	}
	
	return hasUpper && hasLower && hasDigit
}
