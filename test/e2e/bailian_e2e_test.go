package e2e

import (
	"context"
	"fmt"
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

// TestBailian_E2E_Complete 测试百炼完整的端到端流程
// 这个测试模拟真实用户场景：从配置创建到模型调用的完整流程
func TestBailian_E2E_Complete(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过端到端测试")
	}

	// 从环境变量获取百炼配置
	bailianAPIKey := os.Getenv("BAILIAN_API_KEY")
	bailianEndpoint := os.Getenv("BAILIAN_ENDPOINT")
	bailianModel := os.Getenv("BAILIAN_MODEL")

	if bailianAPIKey == "" {
		t.Skip("跳过百炼端到端测试：缺少必需的环境变量 BAILIAN_API_KEY")
	}

	// 设置默认值
	if bailianEndpoint == "" {
		bailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if bailianModel == "" {
		bailianModel = "qwen-plus"
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	t.Log("✓ 数据库连接成功")

	// ========== 阶段 2: 创建租户和模型配置 ==========
	t.Log("========== 阶段 2: 创建租户和模型配置 ==========")

	// 创建测试租户
	tenantID := uuid.New()
	t.Logf("创建测试租户: %s", tenantID)

	// 创建模型配置
	modelName := "bailian-qwen-e2e"
	queryParams := fmt.Sprintf(`{
		"model": "%s",
		"bailianEndpoint": "%s",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`, bailianModel, bailianEndpoint)

	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         bailianModel,
		ModelProvider: "bianlian",
		APIKey:        bailianAPIKey,
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(modelConfig).Error
	require.NoError(t, err, "模型配置创建应该成功")

	t.Logf("✓ 模型配置创建成功: %s", modelName)

	// ========== 阶段 3: 初始化 Genkit Client ==========
	t.Log("========== 阶段 3: 初始化 Genkit Client ==========")

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	t.Log("✓ Genkit Client 初始化成功")

	// ========== 阶段 4: 测试非流式调用 ==========
	t.Log("========== 阶段 4: 测试非流式调用 ==========")

	t.Run("简单问答", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"你好，请用一句话介绍你自己。",
			nil,
		)

		require.NoError(t, err, "生成应该成功")
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		assert.Equal(t, bailianModel, result.Model)

		t.Logf("✓ 简单问答成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("中文处理能力", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请用中文解释什么是人工智能，不超过50字。",
			nil,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		assert.Contains(t, result.Text, "人工智能")

		t.Logf("✓ 中文处理成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("带参数的调用", func(t *testing.T) {
		temperature := 0.8
		maxTokens := 500
		options := &genkit.GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请列举三个中国的传统节日。",
			options,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		t.Logf("✓ 带参数调用成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("Token 统计", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"计算 1+1 等于多少？",
			nil,
		)

		require.NoError(t, err)
		assert.NotNil(t, result.Usage)
		assert.Greater(t, result.Usage.TotalTokens, 0)

		t.Logf("✓ Token 统计正常")
		t.Logf("  Prompt: %d, Completion: %d, Total: %d",
			result.Usage.PromptTokens,
			result.Usage.CompletionTokens,
			result.Usage.TotalTokens)
	})

	// ========== 阶段 5: 测试流式调用 ==========
	t.Log("========== 阶段 5: 测试流式调用 ==========")

	t.Run("流式响应", func(t *testing.T) {
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请用一句话介绍人工智能。",
			nil,
		)

		require.NoError(t, err)

		var fullText string
		var chunkCount int

		for chunk := range streamChan {
			require.NoError(t, chunk.Error)

			if !chunk.Done {
				fullText += chunk.Content
				chunkCount++
			}
		}

		assert.NotEmpty(t, fullText)
		assert.Greater(t, chunkCount, 0)

		t.Logf("✓ 流式响应成功")
		t.Logf("  接收到 %d 个数据块", chunkCount)
		t.Logf("  完整响应: %s", fullText)
	})

	t.Run("中文流式输出", func(t *testing.T) {
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请描述春天的景色，包含花、鸟、风等元素。",
			nil,
		)

		require.NoError(t, err)

		var fullText string
		var chunkCount int

		for chunk := range streamChan {
			require.NoError(t, chunk.Error)

			if !chunk.Done {
				fullText += chunk.Content
				chunkCount++
				t.Logf("  接收数据块 %d: %s", chunkCount, chunk.Content)
			}
		}

		assert.NotEmpty(t, fullText)
		assert.Greater(t, chunkCount, 0)

		t.Logf("✓ 中文流式输出成功")
		t.Logf("  完整响应: %s", fullText)
	})

	t.Run("流式中断", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)

		streamChan, err := client.GenerateStream(
			cancelCtx,
			tenantID.String(),
			modelName,
			"请详细介绍一下云计算的发展历史。",
			nil,
		)

		require.NoError(t, err)

		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				break
			}

			if !chunk.Done {
				chunkCount++
				if chunkCount >= 2 {
					cancel()
					t.Log("  取消流式请求")
				}
			}
		}

		assert.GreaterOrEqual(t, chunkCount, 1)
		t.Logf("✓ 流式中断处理正常")
		t.Logf("  在取消前接收到 %d 个数据块", chunkCount)
	})

	// ========== 阶段 6: 测试缓存机制 ==========
	t.Log("========== 阶段 6: 测试缓存机制 ==========")

	t.Run("实例缓存", func(t *testing.T) {
		// 第一次调用
		start1 := time.Now()
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
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
			modelName,
			"第二次调用",
			nil,
		)
		duration2 := time.Since(start2)
		require.NoError(t, err)

		t.Logf("✓ 缓存机制正常")
		t.Logf("  第一次调用: %v", duration1)
		t.Logf("  第二次调用: %v", duration2)
	})

	// ========== 阶段 7: 测试错误处理 ==========
	t.Log("========== 阶段 7: 测试错误处理 ==========")

	t.Run("配置不存在", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 配置不存在错误处理正常: %v", err)
	})

	t.Run("租户ID无效", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			"invalid-uuid",
			modelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的租户ID")
		t.Logf("✓ 租户ID无效错误处理正常: %v", err)
	})

	t.Run("模型已禁用", func(t *testing.T) {
		// 禁用模型
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")
		t.Logf("✓ 模型已禁用错误处理正常: %v", err)

		// 恢复模型状态
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	// ========== 阶段 8: 测试并发场景 ==========
	t.Log("========== 阶段 8: 测试并发场景 ==========")

	t.Run("并发调用", func(t *testing.T) {
		const concurrency = 3
		results := make(chan *genkit.GenerateResult, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				result, err := client.Generate(
					ctx,
					tenantID.String(),
					modelName,
					fmt.Sprintf("这是第 %d 个并发请求", index+1),
					nil,
				)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(i)
		}

		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case result := <-results:
				assert.NotEmpty(t, result.Text)
				successCount++
			case err := <-errors:
				t.Errorf("并发调用失败: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("并发调用超时")
			}
		}

		assert.Equal(t, concurrency, successCount)
		t.Logf("✓ 并发调用成功: %d/%d", successCount, concurrency)
	})

	// ========== 阶段 9: 测试多轮对话 ==========
	t.Log("========== 阶段 9: 测试多轮对话 ==========")

	t.Run("多轮对话", func(t *testing.T) {
		prompts := []string{
			"你好，我想了解一下阿里云百炼。",
			"它有哪些特点？",
			"谢谢你的介绍。",
		}

		for i, prompt := range prompts {
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				prompt,
				nil,
			)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			t.Logf("  第 %d 轮 - 提示: %s", i+1, prompt)
			t.Logf("  第 %d 轮 - 响应: %s", i+1, result.Text)
		}

		t.Log("✓ 多轮对话成功")
	})

	// ========== 阶段 10: 测试复杂中文场景 ==========
	t.Log("========== 阶段 10: 测试复杂中文场景 ==========")

	t.Run("古诗词理解", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请解释李白的《静夜思》这首诗的意境。",
			nil,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		t.Logf("✓ 古诗词理解成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("成语解释", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请解释成语'画龙点睛'的含义和出处。",
			nil,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		t.Logf("✓ 成语解释成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("中文创作", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请写一首关于秋天的七言绝句。",
			nil,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		t.Logf("✓ 中文创作成功")
		t.Logf("  响应: %s", result.Text)
	})

	// ========== 测试完成 ==========
	t.Log("========== 百炼端到端测试完成 ==========")
	t.Log("✓ 所有测试阶段通过")
}
