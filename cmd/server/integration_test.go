package main

import (
	"os"
	"testing"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/logger"
)

// TestProviderServiceIntegration 测试模型提供商服务的完整集成
func TestProviderServiceIntegration(t *testing.T) {
	// 检查 models 目录是否存在
	if _, err := os.Stat("../../models"); os.IsNotExist(err) {
		t.Skip("跳过集成测试：models 目录不存在")
	}

	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)

	t.Run("完整的模型提供商服务初始化", func(t *testing.T) {
		cfg := &config.Config{
			Models: config.ModelsConfig{
				Dir: "../../models",
			},
		}

		service, err := initProviderService(cfg, log)
		if err != nil {
			t.Fatalf("模型提供商服务初始化失败: %v", err)
		}

		if service == nil {
			t.Fatal("期望返回服务实例，但得到 nil")
		}

		// 测试获取所有提供商
		providers := service.GetAllProviders()
		if len(providers) == 0 {
			t.Error("期望至少有一个提供商，但得到空列表")
		}

		t.Logf("成功加载 %d 个提供商", len(providers))

		// 测试获取每个提供商的详情
		for _, p := range providers {
			provider, err := service.GetProviderByID(p.ID)
			if err != nil {
				t.Errorf("获取提供商 %s 详情失败: %v", p.ID, err)
				continue
			}

			if provider.ID != p.ID {
				t.Errorf("提供商 ID 不匹配: 期望 %s, 得到 %s", p.ID, provider.ID)
			}

			// 测试获取提供商的模型列表
			models, err := service.GetProviderModels(p.ID)
			if err != nil {
				t.Errorf("获取提供商 %s 的模型列表失败: %v", p.ID, err)
				continue
			}

			t.Logf("提供商 %s 有 %d 个模型", p.ID, len(models))

			// 测试获取第一个模型的详情（如果存在）
			if len(models) > 0 {
				firstModel := models[0]
				model, err := service.GetProviderModel(p.ID, firstModel.Model)
				if err != nil {
					t.Errorf("获取模型 %s/%s 详情失败: %v", p.ID, firstModel.Model, err)
					continue
				}

				if model.Model != firstModel.Model {
					t.Errorf("模型名称不匹配: 期望 %s, 得到 %s", firstModel.Model, model.Model)
				}

				// 测试获取模型的参数规则
				rules, err := service.GetModelParameterRules(p.ID, firstModel.Model)
				if err != nil {
					t.Errorf("获取模型 %s/%s 的参数规则失败: %v", p.ID, firstModel.Model, err)
					continue
				}

				t.Logf("模型 %s/%s 有 %d 个参数规则", p.ID, firstModel.Model, len(rules))
			}
		}
	})

	t.Run("测试不存在的提供商", func(t *testing.T) {
		cfg := &config.Config{
			Models: config.ModelsConfig{
				Dir: "../../models",
			},
		}

		service, err := initProviderService(cfg, log)
		if err != nil {
			t.Fatalf("模型提供商服务初始化失败: %v", err)
		}

		// 测试获取不存在的提供商
		_, err = service.GetProviderByID("nonexistent-provider")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}

		// 测试获取不存在提供商的模型
		_, err = service.GetProviderModels("nonexistent-provider")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("测试不存在的模型", func(t *testing.T) {
		cfg := &config.Config{
			Models: config.ModelsConfig{
				Dir: "../../models",
			},
		}

		service, err := initProviderService(cfg, log)
		if err != nil {
			t.Fatalf("模型提供商服务初始化失败: %v", err)
		}

		// 获取第一个提供商
		providers := service.GetAllProviders()
		if len(providers) == 0 {
			t.Skip("没有可用的提供商")
		}

		firstProvider := providers[0]

		// 测试获取不存在的模型
		_, err = service.GetProviderModel(firstProvider.ID, "nonexistent-model")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}

		// 测试获取不存在模型的参数规则
		_, err = service.GetModelParameterRules(firstProvider.ID, "nonexistent-model")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}
