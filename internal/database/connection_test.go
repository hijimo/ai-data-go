package database

import (
	"testing"

	"ai-knowledge-platform/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	// 测试未初始化的数据库连接
	err := HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数据库连接未初始化")
}

func TestNewConnection_InvalidDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		URL: "invalid-dsn",
	}

	_, err := NewConnection(cfg)
	assert.Error(t, err)
}

func TestClose_NilDB(t *testing.T) {
	// 重置DB为nil
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	err := Close()
	assert.NoError(t, err)
}