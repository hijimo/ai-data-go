package loader

import (
	"os"
	"path/filepath"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/storage"
)

// TestLoadProviders 测试加载提供商
func TestLoadProviders(t *testing.T) {
	// 创建日志记录器
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)

	// 创建存储
	store := storage.NewMemoryStore()

	// 创建加载器
	loader := NewModelLoader(store, log)

	// 获取项目根目录
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}

	// 构建 models 目录路径
	modelsDir := filepath.Join(wd, "..", "..", "models")

	// 检查目录是否存在
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		t.Skipf("models 目录不存在: %s", modelsDir)
	}

	// 加载提供商
	providers, err := loader.LoadProviders(modelsDir)
	if err != nil {
		t.Fatalf("加载提供商失败: %v", err)
	}

	// 验证结果
	if len(providers) == 0 {
		t.Error("未加载到任何提供商")
	}

	t.Logf("成功加载 %d 个提供商", len(providers))

	// 验证提供商数据
	for _, provider := range providers {
		if provider.ID == "" {
			t.Error("提供商 ID 为空")
		}
		if provider.Provider == "" {
			t.Error("提供商标识为空")
		}
		t.Logf("提供商: %s (%s)", provider.ID, provider.Provider)
	}
}

// TestLoadAll 测试完整加载流程
func TestLoadAll(t *testing.T) {
	// 创建日志记录器
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)

	// 创建存储
	store := storage.NewMemoryStore()

	// 创建加载器
	loader := NewModelLoader(store, log)

	// 获取项目根目录
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}

	// 构建 models 目录路径
	modelsDir := filepath.Join(wd, "..", "..", "models")

	// 检查目录是否存在
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		t.Skipf("models 目录不存在: %s", modelsDir)
	}

	// 执行完整加载
	err = loader.LoadAll(modelsDir)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	// 验证提供商数量
	providersCount := store.GetProvidersCount()
	if providersCount == 0 {
		t.Error("未加载到任何提供商")
	}
	t.Logf("成功加载 %d 个提供商", providersCount)

	// 验证模型数量
	modelsCount := store.GetModelsCount()
	if modelsCount == 0 {
		t.Error("未加载到任何模型")
	}
	t.Logf("成功加载 %d 个模型", modelsCount)

	// 验证可以获取提供商
	providers := store.GetProviders()
	if len(providers) != providersCount {
		t.Errorf("提供商数量不匹配: 期望 %d, 实际 %d", providersCount, len(providers))
	}

	// 验证可以获取模型
	for _, provider := range providers {
		models, err := store.GetModels(provider.ID)
		if err != nil {
			t.Errorf("获取提供商 %s 的模型失败: %v", provider.ID, err)
			continue
		}
		t.Logf("提供商 %s 有 %d 个模型", provider.ID, len(models))
	}
}
