package database

import (
	"context"
	"testing"
	"time"
)

func TestNewPostgresDatabase(t *testing.T) {
	config := &PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db := NewPostgresDatabase(config)
	if db == nil {
		t.Fatal("NewPostgresDatabase 返回 nil")
	}

	if db.config != config {
		t.Error("配置未正确设置")
	}

	if db.db != nil {
		t.Error("数据库连接应该在 Connect 之前为 nil")
	}
}

func TestPostgresDatabase_GetDB(t *testing.T) {
	config := &PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db := NewPostgresDatabase(config)
	
	// 连接前应该返回 nil
	if db.GetDB() != nil {
		t.Error("连接前 GetDB 应该返回 nil")
	}
}

func TestPostgresDatabase_Close_WhenNotConnected(t *testing.T) {
	config := &PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db := NewPostgresDatabase(config)
	
	// 未连接时关闭应该不报错
	if err := db.Close(); err != nil {
		t.Errorf("未连接时关闭不应该报错: %v", err)
	}
}

func TestPostgresDatabase_Ping_WhenNotConnected(t *testing.T) {
	config := &PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db := NewPostgresDatabase(config)
	ctx := context.Background()
	
	// 未连接时 Ping 应该返回错误
	if err := db.Ping(ctx); err == nil {
		t.Error("未连接时 Ping 应该返回错误")
	}
}

// 注意：以下测试需要实际的 PostgreSQL 数据库才能运行
// 在 CI/CD 环境中，可以使用 Docker 容器来提供测试数据库

// TestPostgresDatabase_Connect_Integration 集成测试
// 需要设置环境变量或跳过此测试
func TestPostgresDatabase_Connect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := &PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "postgres",
		Password:        "postgres",
		DBName:          "postgres",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db := NewPostgresDatabase(config)
	ctx := context.Background()

	// 尝试连接
	err := db.Connect(ctx)
	if err != nil {
		t.Skipf("无法连接到数据库（这是正常的，如果没有运行 PostgreSQL）: %v", err)
		return
	}
	defer db.Close()

	// 验证连接
	if db.GetDB() == nil {
		t.Error("连接后 GetDB 不应该返回 nil")
	}

	// 测试 Ping
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Ping 失败: %v", err)
	}

	// 测试关闭
	if err := db.Close(); err != nil {
		t.Errorf("关闭连接失败: %v", err)
	}

	// 关闭后 GetDB 应该返回 nil
	if db.GetDB() != nil {
		t.Error("关闭后 GetDB 应该返回 nil")
	}
}
