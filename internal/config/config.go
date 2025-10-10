package config

import (
	"os"
	"strconv"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	OSS      OSSConfig      `json:"oss"`
	KMS      KMSConfig      `json:"kms"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
	Mode string `json:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	URL      string `json:"url"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	SSLMode  string `json:"ssl_mode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// OSSConfig 对象存储配置
type OSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	BucketName      string `json:"bucket_name"`
	Region          string `json:"region"`
}

// KMSConfig 密钥管理服务配置
type KMSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	KeyID           string `json:"key_id"`
	Region          string `json:"region"`
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "ai_knowledge_platform"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		OSS: OSSConfig{
			Endpoint:        getEnv("OSS_ENDPOINT", ""),
			AccessKeyID:     getEnv("OSS_ACCESS_KEY_ID", ""),
			AccessKeySecret: getEnv("OSS_ACCESS_KEY_SECRET", ""),
			BucketName:      getEnv("OSS_BUCKET_NAME", ""),
			Region:          getEnv("OSS_REGION", ""),
		},
		KMS: KMSConfig{
			Endpoint:        getEnv("KMS_ENDPOINT", ""),
			AccessKeyID:     getEnv("KMS_ACCESS_KEY_ID", ""),
			AccessKeySecret: getEnv("KMS_ACCESS_KEY_SECRET", ""),
			KeyID:           getEnv("KMS_KEY_ID", ""),
			Region:          getEnv("KMS_REGION", ""),
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为整数
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}