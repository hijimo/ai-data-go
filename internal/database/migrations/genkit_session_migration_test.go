package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestGenkitSessionMigration 测试 Genkit 会话管理迁移
func TestGenkitSessionMigration(t *testing.T) {
	// 跳过集成测试（需要真实数据库）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 注意：此测试需要配置测试数据库
	// 可以通过环境变量设置测试数据库连接
	dsn := "host=localhost user=postgres password=postgres dbname=genkit_test port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("无法连接测试数据库: %v", err)
		return
	}

	// 创建迁移实例
	migration := NewGenkitSessionMigration(db)

	// 测试 GetName
	t.Run("GetName", func(t *testing.T) {
		name := migration.GetName()
		assert.Equal(t, "genkit_session_migration", name)
	})

	// 测试 Up 迁移
	t.Run("Up", func(t *testing.T) {
		// 先清理可能存在的表
		_ = migration.Down()

		// 执行迁移
		err := migration.Up()
		assert.NoError(t, err)

		// 验证表是否创建
		tables := []string{
			"conversation_memories",
			"conversation_contexts",
			"conversation_summaries",
		}

		for _, table := range tables {
			var exists bool
			err := db.Raw(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables 
					WHERE table_schema = 'public' 
					AND table_name = ?
				)
			`, table).Scan(&exists).Error

			assert.NoError(t, err)
			assert.True(t, exists, "表 %s 应该存在", table)
		}
	})

	// 测试 Down 迁移
	t.Run("Down", func(t *testing.T) {
		// 执行回滚
		err := migration.Down()
		assert.NoError(t, err)

		// 验证表是否删除
		tables := []string{
			"conversation_memories",
			"conversation_contexts",
			"conversation_summaries",
		}

		for _, table := range tables {
			var exists bool
			err := db.Raw(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables 
					WHERE table_schema = 'public' 
					AND table_name = ?
				)
			`, table).Scan(&exists).Error

			assert.NoError(t, err)
			assert.False(t, exists, "表 %s 不应该存在", table)
		}
	})
}

// TestGenkitSessionMigrationGetName 测试获取迁移名称
func TestGenkitSessionMigrationGetName(t *testing.T) {
	// 这个测试不需要数据库连接
	migration := &GenkitSessionMigration{}
	name := migration.GetName()
	assert.Equal(t, "genkit_session_migration", name)
}
