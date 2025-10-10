package database

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"ai-knowledge-platform/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationManager(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=true 来运行")
	}

	// 加载测试配置
	cfg, err := config.Load()
	require.NoError(t, err)

	// 使用测试数据库
	testDBName := cfg.Database.DBName + "_test"
	testDatabaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		testDBName,
		cfg.Database.SSLMode,
	)

	// 创建测试数据库
	createTestDB(t, cfg, testDBName)
	defer dropTestDB(t, cfg, testDBName)

	t.Run("创建迁移管理器", func(t *testing.T) {
		mm, err := NewMigrationManager(testDatabaseURL, "../../migrations")
		require.NoError(t, err)
		defer mm.Close()

		assert.NotNil(t, mm.migrate)
		assert.NotNil(t, mm.db)
	})

	t.Run("执行迁移", func(t *testing.T) {
		mm, err := NewMigrationManager(testDatabaseURL, "../../migrations")
		require.NoError(t, err)
		defer mm.Close()

		// 执行迁移
		err = mm.Up()
		assert.NoError(t, err)

		// 检查版本
		version, dirty, err := mm.Version()
		assert.NoError(t, err)
		assert.False(t, dirty)
		assert.Greater(t, version, uint(0))
	})

	t.Run("回滚迁移", func(t *testing.T) {
		mm, err := NewMigrationManager(testDatabaseURL, "../../migrations")
		require.NoError(t, err)
		defer mm.Close()

		// 先执行迁移
		err = mm.Up()
		require.NoError(t, err)

		// 获取当前版本
		version, _, err := mm.Version()
		require.NoError(t, err)

		// 回滚一个版本
		err = mm.Down()
		assert.NoError(t, err)

		// 检查版本是否减少了
		newVersion, dirty, err := mm.Version()
		assert.NoError(t, err)
		assert.False(t, dirty)
		assert.Less(t, newVersion, version)
	})
}

func TestSeedDatabase(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=true 来运行")
	}

	// 加载测试配置
	cfg, err := config.Load()
	require.NoError(t, err)

	// 使用测试数据库
	testDBName := cfg.Database.DBName + "_seed_test"
	testDatabaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		testDBName,
		cfg.Database.SSLMode,
	)

	// 创建测试数据库
	createTestDB(t, cfg, testDBName)
	defer dropTestDB(t, cfg, testDBName)

	// 先运行迁移
	err = RunMigrations(testDatabaseURL)
	require.NoError(t, err)

	t.Run("初始化种子数据", func(t *testing.T) {
		err := SeedDatabase(testDatabaseURL)
		assert.NoError(t, err)

		// 验证种子数据
		db, err := sql.Open("postgres", testDatabaseURL)
		require.NoError(t, err)
		defer db.Close()

		// 检查LLM提供商数量
		var providerCount int
		err = db.QueryRow("SELECT COUNT(*) FROM llm_providers WHERE is_deleted = FALSE").Scan(&providerCount)
		assert.NoError(t, err)
		assert.Greater(t, providerCount, 0)

		// 检查LLM模型数量
		var modelCount int
		err = db.QueryRow("SELECT COUNT(*) FROM llm_models WHERE is_deleted = FALSE").Scan(&modelCount)
		assert.NoError(t, err)
		assert.Greater(t, modelCount, 0)
	})

	t.Run("清理种子数据", func(t *testing.T) {
		// 先初始化种子数据
		err := SeedDatabase(testDatabaseURL)
		require.NoError(t, err)

		// 清理种子数据
		err = CleanSeedData(testDatabaseURL)
		assert.NoError(t, err)

		// 验证数据已清理
		db, err := sql.Open("postgres", testDatabaseURL)
		require.NoError(t, err)
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM llm_providers").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)

		err = db.QueryRow("SELECT COUNT(*) FROM llm_models").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// createTestDB 创建测试数据库
func createTestDB(t *testing.T, cfg *config.Config, dbName string) {
	adminURL := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", adminURL)
	require.NoError(t, err)
	defer db.Close()

	// 删除测试数据库（如果存在）
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	require.NoError(t, err)

	// 创建测试数据库
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err)
}

// dropTestDB 删除测试数据库
func dropTestDB(t *testing.T, cfg *config.Config, dbName string) {
	adminURL := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", adminURL)
	require.NoError(t, err)
	defer db.Close()

	// 删除测试数据库
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	require.NoError(t, err)
}