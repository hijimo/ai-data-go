package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/storage"
	"genkit-ai-service/pkg/validator"

	"gopkg.in/yaml.v3"
)

// ModelLoader 模型加载器接口
type ModelLoader interface {
	// LoadAll 加载所有提供商和模型数据
	LoadAll(modelsDir string) error

	// LoadProviders 加载所有提供商配置
	LoadProviders(modelsDir string) ([]model.Provider, error)

	// LoadModels 加载指定提供商的所有模型
	LoadModels(providerDir string, providerID string, modelTypes map[string]model.ModelTypeInfo) ([]model.Model, error)
}

// modelLoader 模型加载器实现
type modelLoader struct {
	store  storage.Store
	logger logger.Logger
}

// NewModelLoader 创建新的模型加载器
func NewModelLoader(store storage.Store, log logger.Logger) ModelLoader {
	return &modelLoader{
		store:  store,
		logger: log,
	}
}

// LoadAll 加载所有提供商和模型数据
func (l *modelLoader) LoadAll(modelsDir string) error {
	l.logger.Info("开始加载模型提供商数据", logger.Fields{"dir": modelsDir})

	// 清理并验证基础路径
	cleanModelsDir := filepath.Clean(modelsDir)
	
	// 加载提供商
	providers, err := l.LoadProviders(cleanModelsDir)
	if err != nil {
		l.logger.Error("加载提供商失败", logger.Fields{"error": err.Error()})
		return fmt.Errorf("加载提供商失败: %w", err)
	}

	// 存储提供商数据
	l.store.SetProviders(providers)

	// 加载每个提供商的模型
	totalModels := 0
	for _, provider := range providers {
		providerDir := filepath.Join(cleanModelsDir, provider.ID)
		models, err := l.LoadModels(providerDir, provider.ID, provider.Models)
		if err != nil {
			l.logger.Error("加载提供商模型失败", logger.Fields{
				"provider": provider.ID,
				"error":    err.Error(),
			})
			// 继续处理其他提供商
			continue
		}

		// 存储模型数据
		l.store.SetModels(provider.ID, models)
		totalModels += len(models)

		l.logger.Info("提供商模型加载完成", logger.Fields{
			"provider": provider.ID,
			"count":    len(models),
		})
	}

	l.logger.Info("模型提供商数据加载完成", logger.Fields{
		"providers":    len(providers),
		"total_models": totalModels,
	})

	return nil
}

// LoadProviders 加载所有提供商配置
func (l *modelLoader) LoadProviders(modelsDir string) ([]model.Provider, error) {
	var providers []model.Provider

	// 读取 models 目录
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("读取 models 目录失败: %w", err)
	}

	// 遍历每个提供商文件夹
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		providerID := entry.Name()

		// 跳过特殊文件夹
		if strings.HasPrefix(providerID, ".") || strings.HasPrefix(providerID, "_") {
			continue
		}

		// 验证提供商ID的安全性
		if err := validator.ValidateProviderID(providerID); err != nil {
			l.logger.Warn("提供商ID验证失败，跳过", logger.Fields{
				"provider": providerID,
				"error":    err.Error(),
			})
			continue
		}

		// 构建提供商目录路径并验证安全性
		providerDir := filepath.Join(modelsDir, providerID, "provider")
		
		// 验证路径安全性
		if err := l.validatePathSafety(modelsDir, providerDir); err != nil {
			l.logger.Error("提供商目录路径不安全", logger.Fields{
				"provider": providerID,
				"path":     providerDir,
				"error":    err.Error(),
			})
			continue
		}

		providerEntries, err := os.ReadDir(providerDir)
		if err != nil {
			l.logger.Error("读取提供商目录失败", logger.Fields{
				"provider": providerID,
				"path":     providerDir,
				"error":    err.Error(),
			})
			continue
		}

		// 查找第一个 yaml 文件
		var yamlPath string
		for _, pe := range providerEntries {
			if !pe.IsDir() && strings.HasSuffix(pe.Name(), ".yaml") {
				yamlPath = filepath.Join(providerDir, pe.Name())
				break
			}
		}

		if yamlPath == "" {
			l.logger.Error("未找到提供商配置文件", logger.Fields{
				"provider": providerID,
				"path":     providerDir,
			})
			continue
		}

		// 验证YAML文件路径安全性
		if err := l.validatePathSafety(modelsDir, yamlPath); err != nil {
			l.logger.Error("提供商配置文件路径不安全", logger.Fields{
				"provider": providerID,
				"path":     yamlPath,
				"error":    err.Error(),
			})
			continue
		}

		// 读取 provider yaml
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			l.logger.Error("读取提供商配置文件失败", logger.Fields{
				"provider": providerID,
				"path":     yamlPath,
				"error":    err.Error(),
			})
			continue
		}

		// 解析 YAML
		var provider model.Provider
		if err := yaml.Unmarshal(data, &provider); err != nil {
			l.logger.Error("解析提供商配置文件失败", logger.Fields{
				"provider": providerID,
				"path":     yamlPath,
				"error":    err.Error(),
			})
			continue
		}

		// 设置 ID（使用文件夹名称作为 ID）
		provider.ID = providerID

		providers = append(providers, provider)

		l.logger.Info("提供商配置加载成功", logger.Fields{"provider": providerID})
	}

	return providers, nil
}

// LoadModels 加载指定提供商的所有模型
func (l *modelLoader) LoadModels(providerDir string, providerID string, modelTypes map[string]model.ModelTypeInfo) ([]model.Model, error) {
	var allModels []model.Model

	// 获取基础目录（用于路径安全验证）
	baseDir := filepath.Dir(filepath.Dir(providerDir)) // 回到 models 目录

	// 遍历每个模型类型
	for modelType, typeInfo := range modelTypes {
		modelsDir := filepath.Join(providerDir, "models", modelType)

		// 验证路径安全性
		if err := l.validatePathSafety(baseDir, modelsDir); err != nil {
			l.logger.Error("模型类型目录路径不安全", logger.Fields{
				"provider": providerID,
				"type":     modelType,
				"path":     modelsDir,
				"error":    err.Error(),
			})
			continue
		}

		// 检查目录是否存在
		if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
			l.logger.Warn("模型类型目录不存在", logger.Fields{
				"provider": providerID,
				"type":     modelType,
				"path":     modelsDir,
			})
			continue
		}

		var modelNames []string

		// 尝试读取 _position.yaml
		if typeInfo.Position != "" {
			positionPath := filepath.Join(providerDir, typeInfo.Position)
			
			// 验证position文件路径安全性
			if err := l.validatePathSafety(baseDir, positionPath); err != nil {
				l.logger.Error("position文件路径不安全", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"path":     positionPath,
					"error":    err.Error(),
				})
				continue
			}
			
			data, err := os.ReadFile(positionPath)
			if err == nil {
				// 解析 position 文件
				if err := yaml.Unmarshal(data, &modelNames); err != nil {
					l.logger.Error("解析 position 文件失败", logger.Fields{
						"provider": providerID,
						"type":     modelType,
						"path":     positionPath,
						"error":    err.Error(),
					})
				}
			} else {
				l.logger.Warn("position 文件不存在，将扫描目录", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"path":     positionPath,
				})
			}
		}

		// 如果没有 position 文件或解析失败，扫描目录
		if len(modelNames) == 0 {
			entries, err := os.ReadDir(modelsDir)
			if err != nil {
				l.logger.Error("读取模型目录失败", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"path":     modelsDir,
					"error":    err.Error(),
				})
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
					continue
				}
				// 排除特殊文件
				if strings.HasPrefix(entry.Name(), "_") {
					continue
				}
				modelName := strings.TrimSuffix(entry.Name(), ".yaml")
				modelNames = append(modelNames, modelName)
			}
		}

		// 加载每个模型
		for _, modelName := range modelNames {
			// 验证模型名称的安全性
			if err := validator.ValidateModelID(modelName); err != nil {
				l.logger.Warn("模型名称验证失败，跳过", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"model":    modelName,
					"error":    err.Error(),
				})
				continue
			}

			modelPath := filepath.Join(modelsDir, modelName+".yaml")
			
			// 验证模型文件路径安全性
			if err := l.validatePathSafety(baseDir, modelPath); err != nil {
				l.logger.Error("模型文件路径不安全", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"model":    modelName,
					"path":     modelPath,
					"error":    err.Error(),
				})
				continue
			}
			
			data, err := os.ReadFile(modelPath)
			if err != nil {
				l.logger.Error("读取模型配置文件失败", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"model":    modelName,
					"path":     modelPath,
					"error":    err.Error(),
				})
				continue
			}

			var mdl model.Model
			if err := yaml.Unmarshal(data, &mdl); err != nil {
				l.logger.Error("解析模型配置文件失败", logger.Fields{
					"provider": providerID,
					"type":     modelType,
					"model":    modelName,
					"path":     modelPath,
					"error":    err.Error(),
				})
				continue
			}

			// 设置 model_type
			mdl.ModelType = modelType

			allModels = append(allModels, mdl)
		}

		l.logger.Info("模型类型加载完成", logger.Fields{
			"provider": providerID,
			"type":     modelType,
			"count":    len(modelNames),
		})
	}

	return allModels, nil
}

// validatePathSafety 验证路径安全性，防止目录遍历攻击
func (l *modelLoader) validatePathSafety(baseDir, targetPath string) error {
	// 清理路径
	cleanBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("无法解析基础目录: %w", err)
	}

	cleanTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("无法解析目标路径: %w", err)
	}

	// 确保目标路径在基础目录内
	if !strings.HasPrefix(cleanTarget, cleanBase) {
		return fmt.Errorf("路径 %s 不在允许的基础目录 %s 内", cleanTarget, cleanBase)
	}

	// 额外的路径遍历检查
	relPath, err := filepath.Rel(cleanBase, cleanTarget)
	if err != nil {
		return fmt.Errorf("无法计算相对路径: %w", err)
	}

	// 检查相对路径是否包含 ".."
	if strings.Contains(relPath, "..") {
		return fmt.Errorf("路径包含非法的父目录引用")
	}

	return nil
}
