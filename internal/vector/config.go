package vector

import (
	"fmt"
	"time"
)

// ProviderType 向量存储提供商类型
type ProviderType string

const (
	ProviderADBPG     ProviderType = "adbpg"     // 阿里云AnalyticDB PostgreSQL
	ProviderPinecone  ProviderType = "pinecone"  // Pinecone
	ProviderWeaviate  ProviderType = "weaviate"  // Weaviate
	ProviderChroma    ProviderType = "chroma"    // Chroma
	ProviderMilvus    ProviderType = "milvus"    // Milvus
)

// Config 向量存储配置
type Config struct {
	Provider ProviderType           `json:"provider" yaml:"provider"`
	Settings map[string]interface{} `json:"settings" yaml:"settings"`
}

// ADBPGConfig 阿里云AnalyticDB PostgreSQL配置
type ADBPGConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	Database     string        `json:"database" yaml:"database"`
	Username     string        `json:"username" yaml:"username"`
	Password     string        `json:"password" yaml:"password"`
	SSLMode      string        `json:"ssl_mode" yaml:"ssl_mode"`
	MaxOpenConns int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLife  time.Duration `json:"conn_max_life" yaml:"conn_max_life"`
}

// PineconeConfig Pinecone配置
type PineconeConfig struct {
	APIKey      string `json:"api_key" yaml:"api_key"`
	Environment string `json:"environment" yaml:"environment"`
	ProjectID   string `json:"project_id" yaml:"project_id"`
}

// WeaviateConfig Weaviate配置
type WeaviateConfig struct {
	Host   string `json:"host" yaml:"host"`
	Scheme string `json:"scheme" yaml:"scheme"`
	APIKey string `json:"api_key" yaml:"api_key"`
}

// ChromaConfig Chroma配置
type ChromaConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
}

// MilvusConfig Milvus配置
type MilvusConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	
	if c.Settings == nil {
		return fmt.Errorf("settings is required")
	}
	
	switch c.Provider {
	case ProviderADBPG:
		return c.validateADBPGConfig()
	case ProviderPinecone:
		return c.validatePineconeConfig()
	case ProviderWeaviate:
		return c.validateWeaviateConfig()
	case ProviderChroma:
		return c.validateChromaConfig()
	case ProviderMilvus:
		return c.validateMilvusConfig()
	default:
		return fmt.Errorf("unsupported provider: %s", c.Provider)
	}
}

func (c *Config) validateADBPGConfig() error {
	required := []string{"host", "port", "database", "username", "password"}
	for _, field := range required {
		if _, exists := c.Settings[field]; !exists {
			return fmt.Errorf("adbpg config missing required field: %s", field)
		}
	}
	return nil
}

func (c *Config) validatePineconeConfig() error {
	required := []string{"api_key", "environment"}
	for _, field := range required {
		if _, exists := c.Settings[field]; !exists {
			return fmt.Errorf("pinecone config missing required field: %s", field)
		}
	}
	return nil
}

func (c *Config) validateWeaviateConfig() error {
	required := []string{"host"}
	for _, field := range required {
		if _, exists := c.Settings[field]; !exists {
			return fmt.Errorf("weaviate config missing required field: %s", field)
		}
	}
	return nil
}

func (c *Config) validateChromaConfig() error {
	required := []string{"host", "port"}
	for _, field := range required {
		if _, exists := c.Settings[field]; !exists {
			return fmt.Errorf("chroma config missing required field: %s", field)
		}
	}
	return nil
}

func (c *Config) validateMilvusConfig() error {
	required := []string{"host", "port"}
	for _, field := range required {
		if _, exists := c.Settings[field]; !exists {
			return fmt.Errorf("milvus config missing required field: %s", field)
		}
	}
	return nil
}

// GetADBPGConfig 获取ADBPG配置
func (c *Config) GetADBPGConfig() (*ADBPGConfig, error) {
	if c.Provider != ProviderADBPG {
		return nil, fmt.Errorf("not an ADBPG config")
	}
	
	config := &ADBPGConfig{
		SSLMode:      "require",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		ConnMaxLife:  5 * time.Minute,
	}
	
	if host, ok := c.Settings["host"].(string); ok {
		config.Host = host
	}
	if port, ok := c.Settings["port"].(int); ok {
		config.Port = port
	}
	if database, ok := c.Settings["database"].(string); ok {
		config.Database = database
	}
	if username, ok := c.Settings["username"].(string); ok {
		config.Username = username
	}
	if password, ok := c.Settings["password"].(string); ok {
		config.Password = password
	}
	if sslMode, ok := c.Settings["ssl_mode"].(string); ok {
		config.SSLMode = sslMode
	}
	if maxOpenConns, ok := c.Settings["max_open_conns"].(int); ok {
		config.MaxOpenConns = maxOpenConns
	}
	if maxIdleConns, ok := c.Settings["max_idle_conns"].(int); ok {
		config.MaxIdleConns = maxIdleConns
	}
	if connMaxLife, ok := c.Settings["conn_max_life"].(time.Duration); ok {
		config.ConnMaxLife = connMaxLife
	}
	
	return config, nil
}