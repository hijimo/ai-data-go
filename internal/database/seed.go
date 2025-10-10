package database

import (
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
)

// SeedDatabase 初始化种子数据
func SeedDatabase(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	logrus.Info("开始初始化种子数据...")

	// 检查是否已经有种子数据
	if hasSeededData(db) {
		logrus.Info("种子数据已存在，跳过初始化")
		return nil
	}

	// 初始化默认LLM提供商
	if err := seedLLMProviders(db); err != nil {
		return fmt.Errorf("初始化LLM提供商失败: %w", err)
	}

	// 初始化默认LLM模型
	if err := seedLLMModels(db); err != nil {
		return fmt.Errorf("初始化LLM模型失败: %w", err)
	}

	logrus.Info("种子数据初始化完成")
	return nil
}

// hasSeededData 检查是否已有种子数据
func hasSeededData(db *sql.DB) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM llm_providers WHERE is_deleted = FALSE").Scan(&count)
	if err != nil {
		logrus.Warnf("检查种子数据失败: %v", err)
		return false
	}
	return count > 0
}

// seedLLMProviders 初始化默认LLM提供商
func seedLLMProviders(db *sql.DB) error {
	providers := []struct {
		name         string
		providerType string
		config       string
	}{
		{
			name:         "OpenAI",
			providerType: "openai",
			config:       `{"base_url": "https://api.openai.com/v1", "api_key": "", "organization": ""}`,
		},
		{
			name:         "Azure OpenAI",
			providerType: "azure",
			config:       `{"endpoint": "", "api_key": "", "api_version": "2024-02-01"}`,
		},
		{
			name:         "阿里云千问",
			providerType: "qianwen",
			config:       `{"api_key": "", "endpoint": "https://dashscope.aliyuncs.com/api/v1"}`,
		},
		{
			name:         "Anthropic Claude",
			providerType: "claude",
			config:       `{"api_key": "", "base_url": "https://api.anthropic.com"}`,
		},
		{
			name:         "百川智能",
			providerType: "baichuan",
			config:       `{"api_key": "", "base_url": "https://api.baichuan-ai.com/v1"}`,
		},
		{
			name:         "智谱ChatGLM",
			providerType: "chatglm",
			config:       `{"api_key": "", "base_url": "https://open.bigmodel.cn/api/paas/v4"}`,
		},
	}

	for _, provider := range providers {
		_, err := db.Exec(`
			INSERT INTO llm_providers (name, provider_type, config, is_active, is_deleted, created_at, updated_at)
			VALUES ($1, $2, $3::jsonb, false, false, NOW(), NOW())
			ON CONFLICT DO NOTHING
		`, provider.name, provider.providerType, provider.config)
		
		if err != nil {
			return fmt.Errorf("插入LLM提供商 %s 失败: %w", provider.name, err)
		}
		
		logrus.Infof("已添加LLM提供商: %s", provider.name)
	}

	return nil
}

// seedLLMModels 初始化默认LLM模型
func seedLLMModels(db *sql.DB) error {
	// 获取提供商ID
	providerIDs := make(map[string]string)
	rows, err := db.Query("SELECT id, provider_type FROM llm_providers WHERE is_deleted = FALSE")
	if err != nil {
		return fmt.Errorf("查询LLM提供商失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, providerType string
		if err := rows.Scan(&id, &providerType); err != nil {
			return fmt.Errorf("扫描提供商数据失败: %w", err)
		}
		providerIDs[providerType] = id
	}

	// 定义默认模型
	models := []struct {
		providerType string
		modelName    string
		displayName  string
		modelType    string
		config       string
	}{
		// OpenAI 模型
		{"openai", "gpt-4o", "GPT-4o", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		{"openai", "gpt-4o-mini", "GPT-4o Mini", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		{"openai", "gpt-3.5-turbo", "GPT-3.5 Turbo", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		{"openai", "text-embedding-3-large", "Text Embedding 3 Large", "embedding", `{"dimensions": 3072}`},
		{"openai", "text-embedding-3-small", "Text Embedding 3 Small", "embedding", `{"dimensions": 1536}`},
		
		// Azure OpenAI 模型
		{"azure", "gpt-4o", "Azure GPT-4o", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		{"azure", "gpt-35-turbo", "Azure GPT-3.5 Turbo", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		{"azure", "text-embedding-3-large", "Azure Text Embedding 3 Large", "embedding", `{"dimensions": 3072}`},
		
		// 千问模型
		{"qianwen", "qwen-turbo", "通义千问-Turbo", "chat", `{"max_tokens": 2000, "temperature": 0.7}`},
		{"qianwen", "qwen-plus", "通义千问-Plus", "chat", `{"max_tokens": 8000, "temperature": 0.7}`},
		{"qianwen", "qwen-max", "通义千问-Max", "chat", `{"max_tokens": 8000, "temperature": 0.7}`},
		{"qianwen", "text-embedding-v2", "通义千问文本向量", "embedding", `{"dimensions": 1536}`},
		
		// Claude 模型
		{"claude", "claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		{"claude", "claude-3-haiku-20240307", "Claude 3 Haiku", "chat", `{"max_tokens": 4096, "temperature": 0.7}`},
		
		// 百川模型
		{"baichuan", "Baichuan2-Turbo", "百川2-Turbo", "chat", `{"max_tokens": 2048, "temperature": 0.7}`},
		{"baichuan", "Baichuan2-Turbo-192k", "百川2-Turbo-192k", "chat", `{"max_tokens": 2048, "temperature": 0.7}`},
		
		// ChatGLM 模型
		{"chatglm", "glm-4", "智谱GLM-4", "chat", `{"max_tokens": 4095, "temperature": 0.7}`},
		{"chatglm", "glm-4v", "智谱GLM-4V", "chat", `{"max_tokens": 4095, "temperature": 0.7}`},
		{"chatglm", "embedding-2", "智谱文本向量", "embedding", `{"dimensions": 1024}`},
	}

	for _, model := range models {
		providerID, exists := providerIDs[model.providerType]
		if !exists {
			logrus.Warnf("提供商类型 %s 不存在，跳过模型 %s", model.providerType, model.modelName)
			continue
		}

		_, err := db.Exec(`
			INSERT INTO llm_models (provider_id, model_name, display_name, model_type, config, is_active, is_deleted, created_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, false, false, NOW())
			ON CONFLICT DO NOTHING
		`, providerID, model.modelName, model.displayName, model.modelType, model.config)
		
		if err != nil {
			return fmt.Errorf("插入LLM模型 %s 失败: %w", model.modelName, err)
		}
		
		logrus.Infof("已添加LLM模型: %s (%s)", model.displayName, model.modelName)
	}

	return nil
}

// CleanSeedData 清理种子数据（用于测试）
func CleanSeedData(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	logrus.Info("开始清理种子数据...")

	// 删除模型数据
	if _, err := db.Exec("DELETE FROM llm_models"); err != nil {
		return fmt.Errorf("清理LLM模型失败: %w", err)
	}

	// 删除提供商数据
	if _, err := db.Exec("DELETE FROM llm_providers"); err != nil {
		return fmt.Errorf("清理LLM提供商失败: %w", err)
	}

	logrus.Info("种子数据清理完成")
	return nil
}