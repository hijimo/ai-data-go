package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// ModelConfigurationMigration 模型配置表迁移
type ModelConfigurationMigration struct {
	db *gorm.DB
}

// NewModelConfigurationMigration 创建模型配置迁移实例
func NewModelConfigurationMigration(db *gorm.DB) *ModelConfigurationMigration {
	return &ModelConfigurationMigration{
		db: db,
	}
}

// Up 执行迁移（创建 model_configurations 表）
func (m *ModelConfigurationMigration) Up() error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 创建 model_configurations 表
		if err := m.createModelConfigurationsTable(tx); err != nil {
			return fmt.Errorf("创建model_configurations表失败: %w", err)
		}

		return nil
	})
}

// Down 回滚迁移（删除 model_configurations 表）
func (m *ModelConfigurationMigration) Down() error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DROP TABLE IF EXISTS model_configurations CASCADE").Error; err != nil {
			return fmt.Errorf("删除model_configurations表失败: %w", err)
		}
		return nil
	})
}

// GetName 获取迁移名称
func (m *ModelConfigurationMigration) GetName() string {
	return "model_configuration_migration"
}

// createModelConfigurationsTable 创建 model_configurations 表
func (m *ModelConfigurationMigration) createModelConfigurationsTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS model_configurations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		name VARCHAR(255) NOT NULL,
		model VARCHAR(255) NOT NULL,
		model_provider VARCHAR(50) NOT NULL,
		base_url VARCHAR(500),
		api_key TEXT NOT NULL,
		query_params JSONB,
		is_enabled BOOLEAN NOT NULL DEFAULT true,
		is_deleted BOOLEAN NOT NULL DEFAULT false,
		created_by UUID NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by UUID,
		updated_at TIMESTAMP WITH TIME ZONE,
		deleted_by UUID,
		deleted_at TIMESTAMP WITH TIME ZONE,
		
		-- 外键约束
		CONSTRAINT fk_model_configurations_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_model_configurations_created_by FOREIGN KEY (created_by) REFERENCES users(id),
		CONSTRAINT fk_model_configurations_updated_by FOREIGN KEY (updated_by) REFERENCES users(id),
		CONSTRAINT fk_model_configurations_deleted_by FOREIGN KEY (deleted_by) REFERENCES users(id),
		
		-- 检查约束：确保 model_provider 为有效值
		CONSTRAINT chk_model_provider CHECK (
			model_provider IN ('openai', 'anthropic', 'googlegenai', 'azureopenai', 'bianlian', 'custom_openai')
		)
	);

	-- 创建索引
	-- 复合索引：租户ID和模型提供商
	CREATE INDEX IF NOT EXISTS idx_model_configs_tenant_provider 
		ON model_configurations(tenant_id, model_provider) 
		WHERE is_deleted = false;
	
	-- 单列索引：软删除标记
	CREATE INDEX IF NOT EXISTS idx_model_configs_deleted 
		ON model_configurations(is_deleted);
	
	-- 部分索引：已启用且未删除的配置
	CREATE INDEX IF NOT EXISTS idx_model_configs_enabled 
		ON model_configurations(is_enabled) 
		WHERE is_deleted = false;
	
	-- 索引：租户ID（用于列表查询）
	CREATE INDEX IF NOT EXISTS idx_model_configs_tenant_id 
		ON model_configurations(tenant_id) 
		WHERE is_deleted = false;
	
	-- 索引：创建时间（用于排序）
	CREATE INDEX IF NOT EXISTS idx_model_configs_created_at 
		ON model_configurations(created_at DESC);

	-- 添加表注释
	COMMENT ON TABLE model_configurations IS '模型配置表，存储租户的AI模型提供商配置信息';
	
	-- 添加列注释
	COMMENT ON COLUMN model_configurations.id IS '模型配置唯一标识符（UUID）';
	COMMENT ON COLUMN model_configurations.tenant_id IS '所属租户ID';
	COMMENT ON COLUMN model_configurations.name IS '配置名称';
	COMMENT ON COLUMN model_configurations.model IS '模型标识（如：gpt-4、claude-3-opus等）';
	COMMENT ON COLUMN model_configurations.model_provider IS '模型提供商：openai、anthropic、googlegenai、azureopenai、bianlian、custom_openai';
	COMMENT ON COLUMN model_configurations.base_url IS 'API基础URL（可选，用于自定义端点）';
	COMMENT ON COLUMN model_configurations.api_key IS 'API密钥（加密存储）';
	COMMENT ON COLUMN model_configurations.query_params IS '查询参数（JSONB格式，可选）';
	COMMENT ON COLUMN model_configurations.is_enabled IS '是否启用';
	COMMENT ON COLUMN model_configurations.is_deleted IS '软删除标记';
	COMMENT ON COLUMN model_configurations.created_by IS '创建者用户ID';
	COMMENT ON COLUMN model_configurations.created_at IS '创建时间';
	COMMENT ON COLUMN model_configurations.updated_by IS '更新者用户ID';
	COMMENT ON COLUMN model_configurations.updated_at IS '更新时间';
	COMMENT ON COLUMN model_configurations.deleted_by IS '删除者用户ID';
	COMMENT ON COLUMN model_configurations.deleted_at IS '删除时间';
	`

	return tx.Exec(sql).Error
}
