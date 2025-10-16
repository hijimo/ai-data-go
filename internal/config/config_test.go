package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("SERVER_HOST", "0.0.0.0")
	os.Setenv("GENKIT_API_KEY", "test-api-key")
	os.Setenv("GENKIT_MODEL", "gemini-2.5-flash")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_NAME", "test_db")
	
	defer func() {
		// 清理环境变量
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("GENKIT_API_KEY")
		os.Unsetenv("GENKIT_MODEL")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
	}()

	config, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证服务器配置
	if config.Server.Port != "8080" {
		t.Errorf("期望端口为 8080, 实际为 %s", config.Server.Port)
	}

	// 验证 Genkit 配置
	if config.Genkit.APIKey != "test-api-key" {
		t.Errorf("期望 API Key 为 test-api-key, 实际为 %s", config.Genkit.APIKey)
	}

	// 验证数据库配置
	if config.Database.Host != "localhost" {
		t.Errorf("期望数据库主机为 localhost, 实际为 %s", config.Database.Host)
	}
}

func TestValidate_MissingAPIKey(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port: "8080",
			Host: "0.0.0.0",
		},
		Genkit: GenkitConfig{
			APIKey:             "", // 空的 API Key
			Model:              "gemini-2.5-flash",
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2000,
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         "5432",
			User:         "postgres",
			DBName:       "test_db",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Session: SessionConfig{
			Timeout:         30 * time.Minute,
			CleanupInterval: 5 * time.Minute,
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("期望验证失败，但验证通过了")
	}
}

func TestValidate_InvalidTemperature(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port: "8080",
			Host: "0.0.0.0",
		},
		Genkit: GenkitConfig{
			APIKey:             "test-key",
			Model:              "gemini-2.5-flash",
			DefaultTemperature: 3.0, // 无效的温度值
			DefaultMaxTokens:   2000,
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         "5432",
			User:         "postgres",
			DBName:       "test_db",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Session: SessionConfig{
			Timeout:         30 * time.Minute,
			CleanupInterval: 5 * time.Minute,
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("期望验证失败，但验证通过了")
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// 测试 getEnv
	os.Setenv("TEST_STRING", "test_value")
	defer os.Unsetenv("TEST_STRING")
	
	if value := getEnv("TEST_STRING", "default"); value != "test_value" {
		t.Errorf("期望 test_value, 实际为 %s", value)
	}
	
	if value := getEnv("NON_EXISTENT", "default"); value != "default" {
		t.Errorf("期望 default, 实际为 %s", value)
	}

	// 测试 getEnvInt
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")
	
	if value := getEnvInt("TEST_INT", 0); value != 42 {
		t.Errorf("期望 42, 实际为 %d", value)
	}
	
	if value := getEnvInt("NON_EXISTENT", 10); value != 10 {
		t.Errorf("期望 10, 实际为 %d", value)
	}

	// 测试 getEnvFloat
	os.Setenv("TEST_FLOAT", "3.14")
	defer os.Unsetenv("TEST_FLOAT")
	
	if value := getEnvFloat("TEST_FLOAT", 0.0); value != 3.14 {
		t.Errorf("期望 3.14, 实际为 %f", value)
	}

	// 测试 getEnvDuration
	os.Setenv("TEST_DURATION", "5m")
	defer os.Unsetenv("TEST_DURATION")
	
	if value := getEnvDuration("TEST_DURATION", 0); value != 5*time.Minute {
		t.Errorf("期望 5m, 实际为 %v", value)
	}
}
