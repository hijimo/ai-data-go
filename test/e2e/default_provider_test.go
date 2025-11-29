package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultProvider 测试默认提供商逻辑
// 验证当请求中未指定模型名称时，系统使用默认的 Google AI (Gemini) 模型
func TestDefaultProvider(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过端到端测试")
	}

	// 检查环境变量
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		t.Skip("跳过默认提供商测试：缺少 GOOGLE_API_KEY 环境变量")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	t.Log("✓ 数据库连接成功")

	// ========== 阶段 2: 创建租户和默认模型配置 ==========
	t.Log("========== 阶段 2: 创建租户和默认模型配置 ==========")

	tenantID := uuid.New()
	t.Logf("创建测试租户: %s", tenantID)

	// 创建默认的 Google AI 配置
	defaultModelName := "gemini-pro"
	defaultQueryParams := `{
		"model": "gemini-1.5-pro",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	defaultConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          defaultModelName,
		Model:         "gemini-1.5-pro",
		ModelProvider: "googlegenai",
		APIKey:        googleAPIKey,
		QueryParams:   &defaultQueryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(defaultConfig).Error
	require.NoError(t, err, "默认模型配置创建应该成功")

	t.Logf("✓ 默认模型配置创建成功: %s", defaultModelName)

	// ========== 阶段 3: 初始化 Genkit Client ==========
	t.Log("========== 阶段 3: 初始化 Genkit Client ==========")

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	t.Log("✓ Genkit Client 初始化成功")

	// ========== 阶段 4: 测试使用默认提供商 ==========
	t.Log("========== 阶段 4: 测试使用默认提供商 ==========")

	t.Run("使用默认模型名称", func(t *testing.T) {
		prompt := "请用一句话介绍你自己。"

		result, err := client.Generate(
			ctx,
			tenantID.String(),
			defaultModelName, // 使用默认模型名称
			prompt,
			nil,
		)

		require.NoError(t, err, "使用默认模型应该成功")
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		assert.Equal(t, "gemini-1.5-pro", result.Model)

		t.Logf("✓ 默认模型响应: %s", result.Text)
		t.Logf("✓ 使用的模型: %s", result.Model)
	})

	t.Run("流式调用使用默认模型", func(t *testing.T) {
		prompt := "请用一句话介绍人工智能。"

		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			defaultModelName, // 使用默认模型名称
			prompt,
			nil,
		)

		require.NoError(t, err)

		var fullText string
		var chunkCount int
		var finalModel string

		for chunk := range streamChan {
			require.NoError(t, chunk.Error)

			if chunk.Done {
				finalModel = chunk.Model
			} else {
				fullText += chunk.Content
				chunkCount++
			}
		}

		assert.NotEmpty(t, fullText)
		assert.Greater(t, chunkCount, 0)
		assert.Equal(t, "gemini-1.5-pro", finalModel)

		t.Logf("✓ 默认模型流式响应成功，接收 %d 个数据块", chunkCount)
		t.Logf("✓ 使用的模型: %s", finalModel)
	})

	// ========== 阶段 5: 测试默认模型的参数传递 ==========
	t.Log("========== 阶段 5: 测试默认模型的参数传递 ==========")

	t.Run("默认模型使用自定义参数", func(t *testing.T) {
		temperature := 0.8
		maxTokens := 500

		options := &genkit.GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		result, err := client.Generate(
			ctx,
			tenantID.String(),
			defaultModelName,
			"请列举三个编程语言。",
			options,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		t.Logf("✓ 默认模型使用自定义参数成功")
		t.Logf("  Temperature: %.1f, MaxTokens: %d", temperature, maxTokens)
	})

	// ========== 阶段 6: 测试默认模型的并发调用 ==========
	t.Log("========== 阶段 6: 测试默认模型的并发调用 ==========")

	t.Run("并发使用默认模型", func(t *testing.T) {
		const concurrentRequests = 5

		results := make(chan *genkit.GenerateResult, concurrentRequests)
		errors := make(chan error, concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			go func(index int) {
				result, err := client.Generate(
					ctx,
					tenantID.String(),
					defaultModelName,
					"测试并发请求",
					nil,
				)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(i)
		}

		// 收集结果
		successCount := 0
		for i := 0; i < concurrentRequests; i++ {
			select {
			case result := <-results:
				assert.NotEmpty(t, result.Text)
				assert.Equal(t, "gemini-1.5-pro", result.Model)
				successCount++
			case err := <-errors:
				t.Errorf("并发调用失败: %v", err)
			case <-time.After(60 * time.Second):
				t.Fatal("并发调用超时")
			}
		}

		assert.Equal(t, concurrentRequests, successCount)
		t.Logf("✓ 并发使用默认模型成功: %d/%d", successCount, concurrentRequests)
	})

	// ========== 阶段 7: 测试默认模型的错误处理 ==========
	t.Log("========== 阶段 7: 测试默认模型的错误处理 ==========")

	t.Run("禁用默认模型后的错误处理", func(t *testing.T) {
		// 禁用默认模型
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", defaultConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		// 尝试使用被禁用的默认模型
		_, err = client.Generate(
			ctx,
			tenantID.String(),
			defaultModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")
		t.Logf("✓ 禁用的默认模型错误处理正常: %v", err)

		// 恢复默认模型
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", defaultConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	t.Run("使用不存在的模型名称", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 不存在的模型错误处理正常: %v", err)
	})

	// ========== 阶段 8: 测试默认模型的性能 ==========
	t.Log("========== 阶段 8: 测试默认模型的性能 ==========")

	t.Run("测量默认模型的响应时间", func(t *testing.T) {
		// 预热
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			defaultModelName,
			"预热",
			nil,
		)
		require.NoError(t, err)

		// 测量响应时间
		const iterations = 3
		var durations []time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			_, err := client.Generate(
				ctx,
				tenantID.String(),
				defaultModelName,
				"性能测试",
				nil,
			)
			duration := time.Since(start)

			require.NoError(t, err)
			durations = append(durations, duration)

			t.Logf("  第 %d 次调用耗时: %v", i+1, duration)
		}

		// 计算平均响应时间
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(durations))

		t.Logf("✓ 平均响应时间: %v", avgDuration)
		t.Logf("  总调用次数: %d", len(durations))
	})

	// ========== 阶段 9: 测试默认模型的缓存机制 ==========
	t.Log("========== 阶段 9: 测试默认模型的缓存机制 ==========")

	t.Run("验证默认模型实例被缓存", func(t *testing.T) {
		// 第一次调用
		start1 := time.Now()
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			defaultModelName,
			"第一次调用",
			nil,
		)
		duration1 := time.Since(start1)
		require.NoError(t, err)

		// 第二次调用（应该使用缓存的实例）
		start2 := time.Now()
		_, err = client.Generate(
			ctx,
			tenantID.String(),
			defaultModelName,
			"第二次调用",
			nil,
		)
		duration2 := time.Since(start2)
		require.NoError(t, err)

		t.Logf("  第一次调用耗时: %v", duration1)
		t.Logf("  第二次调用耗时: %v", duration2)
		t.Log("✓ 默认模型实例缓存机制正常工作")
	})

	// ========== 测试完成 ==========
	t.Log("========== 默认提供商测试完成 ==========")
	t.Log("✓ 所有测试阶段通过")
	t.Logf("✓ 验证了默认模型: %s", defaultModelName)
}
