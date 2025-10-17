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
	Server   ServerConfig
	Genkit   GenkitConfig
	Database DatabaseConfig
	Log      LogConfig
	Session  SessionConfig
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
	Level  string // 日志级别 (debug, info, warn, error)
	Format string // 日志格式 (json, text)
}

// SessionConfig 会话配置
type SessionConfig struct {
	Timeout         time.Duration // 会话超时时间
	CleanupInterval time.Duration // 会话清理间隔
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
	apiKey := os.Getenv("GENKIT_API_KEY")
	if apiKey == "" {
		// 尝试从其他可能的环境变量名获取
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	
	config.Genkit = GenkitConfig{
		APIKey:             apiKey,
		Model:              getEnv("GENKIT_MODEL", "gemini-2.5-flash"),
		DefaultTemperature: getEnvFloat("GENKIT_DEFAULT_TEMPERATURE", 0.7),
		DefaultMaxTokens:   getEnvInt("GENKIT_DEFAULT_MAX_TOKENS", 2000),
	}

	// 加载数据库配置
	config.Database = DatabaseConfig{
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
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
	}

	// 加载会话配置
	config.Session = SessionConfig{
		Timeout:         getEnvDuration("SESSION_TIMEOUT", 30*time.Minute),
		CleanupInterval: getEnvDuration("SESSION_CLEANUP_INTERVAL", 5*time.Minute),
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
		return fmt.Errorf("Genkit API密钥不能为空 (GENKIT_API_KEY 或 GEMINI_API_KEY)")
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
	if c.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}
	
	if c.Database.Port == "" {
		return fmt.Errorf("数据库端口不能为空")
	}
	
	dbPort, err := strconv.Atoi(c.Database.Port)
	if err != nil || dbPort < 1 || dbPort > 65535 {
		return fmt.Errorf("数据库端口必须是1-65535之间的有效数字")
	}
	
	if c.Database.User == "" {
		return fmt.Errorf("数据库用户名不能为空")
	}
	
	if c.Database.DBName == "" {
		return fmt.Errorf("数据库名称不能为空")
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
