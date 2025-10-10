package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "5433")
	
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
	}()

	cfg := Load()

	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "testhost", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
}

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Server.Mode)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "ai_knowledge_platform", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Equal(t, 0, cfg.Redis.DB)
}

func TestGetEnv(t *testing.T) {
	// 测试存在的环境变量
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")
	
	result := getEnv("TEST_VAR", "default")
	assert.Equal(t, "test_value", result)

	// 测试不存在的环境变量
	result = getEnv("NON_EXISTENT_VAR", "default")
	assert.Equal(t, "default", result)
}

func TestGetEnvAsInt(t *testing.T) {
	// 测试有效的整数环境变量
	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")
	
	result := getEnvAsInt("TEST_INT", 456)
	assert.Equal(t, 123, result)

	// 测试无效的整数环境变量
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT")
	
	result = getEnvAsInt("TEST_INVALID_INT", 456)
	assert.Equal(t, 456, result)

	// 测试不存在的环境变量
	result = getEnvAsInt("NON_EXISTENT_INT", 789)
	assert.Equal(t, 789, result)
}