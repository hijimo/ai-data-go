package genkit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
	"unicode/utf8"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoogleAIIntegration_NonStreaming 测试 Google AI (Gemini) 非流式调用
// 这是一个集成测试，需要真实的 Google AI 配置
func TestGoogleAIIntegration_NonStreaming(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 从环境变量获取 Google AI 配置
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	googleModel := os.Getenv("GOOGLE_MODEL")

	if googleAPIKey == "" {
		t.Skip("跳过 Google AI 集成测试：缺少必需的环境变量 GOOGLE_API_KEY")
	}

	// 设置默认模型
	if googleModel == "" {
		googleModel = "gemini-2.0-flash-exp"
	}

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err)
	defer cleanupTestDatabase(t, db)

	// 创建测试租户和模型配置
	tenantID := uuid.New()
	modelName := "google-gemini-test"

	// 准备 QueryParams JSON
	queryParams := `{
		"model": "` + googleModel + `",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	// 创建模型配置
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         googleModel,
		ModelProvider: model.ModelProviderGoogleGenAI,
		APIKey:        googleAPIKey,
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(modelConfig).Error
	require.NoError(t, err)

	// 创建 repository 和 client
	repo := repository.NewModelConfigurationRepository(db)
	client := NewClientWithRepo(repo)

	ctx := context.Background()

	t.Run("基本文本生成", func(t *testing.T) {
		// 调用 Generate 方法
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"你好，请用一句话介绍你自己。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		assert.Equal(t, googleModel, result.Model)

		t.Logf("生成的文本: %s", result.Text)
	})

	t.Run("参数传递测试", func(t *testing.T) {
		// 使用自定义选项
		temperature := 0.8
		maxTokens := 1000
		options := &GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请列举三个编程语言的名称。",
			options,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		t.Logf("生成的文本: %s", result.Text)
	})

	t.Run("Token 统计测试", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"计算 1+1 等于多少？",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		// 验证 Token 统计
		if result.Usage != nil {
			assert.Greater(t, result.Usage.PromptTokens, 0)
			assert.Greater(t, result.Usage.CompletionTokens, 0)
			assert.Greater(t, result.Usage.TotalTokens, 0)
			assert.Equal(t, result.Usage.PromptTokens+result.Usage.CompletionTokens, result.Usage.TotalTokens)

			t.Logf("Token 使用情况: Prompt=%d, Completion=%d, Total=%d",
				result.Usage.PromptTokens,
				result.Usage.CompletionTokens,
				result.Usage.TotalTokens)
		}
	})

	t.Run("中文处理能力测试", func(t *testing.T) {
		// 测试中文理解和生成
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请用中文解释什么是人工智能，不超过50字。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		// 验证返回的是中文内容
		assert.Contains(t, result.Text, "人工智能")

		t.Logf("中文生成结果: %s", result.Text)
	})

	t.Run("错误处理 - 配置不存在", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
	})

	t.Run("错误处理 - 租户ID无效", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			"invalid-uuid",
			modelName,
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的租户ID")
	})

	t.Run("错误处理 - 模型已禁用", func(t *testing.T) {
		// 禁用模型
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")

		// 恢复模型状态
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	t.Run("响应格式验证", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请说'你好'。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证响应格式
		assert.NotEmpty(t, result.Text)
		assert.NotEmpty(t, result.Model)
		assert.Contains(t, result.Text, "你好")

		t.Logf("响应格式验证通过: Text=%s, Model=%s", result.Text, result.Model)
	})

	t.Run("缓存机制测试", func(t *testing.T) {
		// 第一次调用（初始化实例）
		start1 := time.Now()
		result1, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"第一次调用",
			nil,
		)
		duration1 := time.Since(start1)
		require.NoError(t, err)
		assert.NotNil(t, result1)

		// 第二次调用（使用缓存的实例）
		start2 := time.Now()
		result2, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"第二次调用",
			nil,
		)
		duration2 := time.Since(start2)
		require.NoError(t, err)
		assert.NotNil(t, result2)

		t.Logf("第一次调用耗时: %v", duration1)
		t.Logf("第二次调用耗时: %v", duration2)

		// 注意：由于网络延迟等因素，第二次调用不一定更快
		// 这里只是记录日志，不做断言
	})

	t.Run("多轮对话测试", func(t *testing.T) {
		// 测试多轮对话能力
		prompts := []string{
			"你好，我想了解一下 Go 语言。",
			"Go 语言有哪些特点？",
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
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Text)

			t.Logf("第 %d 轮对话 - 提示: %s", i+1, prompt)
			t.Logf("第 %d 轮对话 - 响应: %s", i+1, result.Text)
		}
	})

	t.Run("长文本生成测试", func(t *testing.T) {
		// 测试生成较长的文本
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请详细介绍一下 Go 语言的并发模型，包括 goroutine 和 channel 的使用。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		// 验证文本长度
		assert.Greater(t, len(result.Text), 100, "长文本生成应该返回较长的内容")

		t.Logf("长文本生成结果长度: %d 字符", len(result.Text))
		t.Logf("长文本生成结果: %s", result.Text)
	})

	t.Run("特殊字符处理测试", func(t *testing.T) {
		// 测试包含特殊字符的输入
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请解释这个表达式的含义：a && b || c",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		t.Logf("特殊字符处理结果: %s", result.Text)
	})

	t.Run("并发调用测试", func(t *testing.T) {
		// 测试并发调用的安全性
		const concurrency = 5
		results := make(chan *GenerateResult, concurrency)
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

		// 收集结果
		successCount := 0
		errorCount := 0

		for i := 0; i < concurrency; i++ {
			select {
			case result := <-results:
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Text)
				successCount++
			case err := <-errors:
				t.Logf("并发调用错误: %v", err)
				errorCount++
			case <-time.After(30 * time.Second):
				t.Fatal("并发调用超时")
			}
		}

		t.Logf("并发调用结果: 成功=%d, 失败=%d", successCount, errorCount)
		assert.Equal(t, concurrency, successCount, "所有并发调用都应该成功")
	})
}

// TestGoogleAIIntegration_Streaming 测试 Google AI (Gemini) 流式调用
func TestGoogleAIIntegration_Streaming(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 从环境变量获取 Google AI 配置
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	googleModel := os.Getenv("GOOGLE_MODEL")

	if googleAPIKey == "" {
		t.Skip("跳过 Google AI 集成测试：缺少必需的环境变量 GOOGLE_API_KEY")
	}

	// 设置默认模型
	if googleModel == "" {
		googleModel = "gemini-2.0-flash-exp"
	}

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err)
	defer cleanupTestDatabase(t, db)

	// 创建测试租户和模型配置
	tenantID := uuid.New()
	modelName := "google-gemini-stream-test"

	// 准备 QueryParams JSON
	queryParams := `{
		"model": "` + googleModel + `",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	// 创建模型配置
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         googleModel,
		ModelProvider: model.ModelProviderGoogleGenAI,
		APIKey:        googleAPIKey,
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(modelConfig).Error
	require.NoError(t, err)

	// 创建 repository 和 client
	repo := repository.NewModelConfigurationRepository(db)
	client := NewClientWithRepo(repo)

	ctx := context.Background()

	t.Run("基本流式生成", func(t *testing.T) {
		// 调用 GenerateStream 方法
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请用一句话介绍你自己。",
			nil,
		)

		// 验证没有错误
		require.NoError(t, err)
		assert.NotNil(t, streamChan)

		// 接收流式响应
		var fullText string
		var chunkCount int
		var finalUsage *Usage

		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应错误: %v", chunk.Error)
			}

			if !chunk.Done {
				fullText += chunk.Content
				chunkCount++
				t.Logf("接收到第 %d 个数据块: %s", chunkCount, chunk.Content)
			} else {
				// 最后一个块
				if chunk.Usage != nil {
					finalUsage = chunk.Usage
				}
			}
		}

		// 验证结果
		assert.NotEmpty(t, fullText, "流式响应应该返回内容")
		assert.Greater(t, chunkCount, 0, "应该至少接收到一个数据块")

		t.Logf("完整文本: %s", fullText)
		t.Logf("总共接收到 %d 个数据块", chunkCount)

		if finalUsage != nil {
			t.Logf("Token 使用情况: Prompt=%d, Completion=%d, Total=%d",
				finalUsage.PromptTokens,
				finalUsage.CompletionTokens,
				finalUsage.TotalTokens)
		}
	})

	t.Run("流式响应完整性测试", func(t *testing.T) {
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请列举三个编程语言。",
			nil,
		)

		require.NoError(t, err)

		var fullText string
		var gotDone bool

		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应错误: %v", chunk.Error)
			}

			if !chunk.Done {
				fullText += chunk.Content
			} else {
				gotDone = true
			}
		}

		// 验证完整性
		assert.True(t, gotDone, "应该接收到完成标记")
		assert.NotEmpty(t, fullText, "应该接收到完整的文本")

		t.Logf("完整文本: %s", fullText)
	})

	t.Run("流式中断处理测试", func(t *testing.T) {
		// 创建可取消的上下文
		cancelCtx, cancel := context.WithCancel(ctx)

		streamChan, err := client.GenerateStream(
			cancelCtx,
			tenantID.String(),
			modelName,
			"请详细介绍一下 Go 语言的历史和发展。",
			nil,
		)

		require.NoError(t, err)

		// 接收几个数据块后取消
		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				// 取消后可能会收到错误，这是正常的
				t.Logf("流式响应错误（预期）: %v", chunk.Error)
				break
			}

			if !chunk.Done {
				chunkCount++
				t.Logf("接收到第 %d 个数据块", chunkCount)

				// 接收到 2 个数据块后取消
				if chunkCount >= 2 {
					cancel()
					t.Log("取消流式请求")
				}
			}
		}

		t.Logf("在取消前接收到 %d 个数据块", chunkCount)
	})

	t.Run("流式参数传递测试", func(t *testing.T) {
		temperature := 0.9
		maxTokens := 500
		options := &GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请简单介绍一下人工智能。",
			options,
		)

		require.NoError(t, err)

		var fullText string
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应错误: %v", chunk.Error)
			}

			if !chunk.Done {
				fullText += chunk.Content
			}
		}

		assert.NotEmpty(t, fullText)
		t.Logf("使用自定义参数生成的文本: %s", fullText)
	})

	t.Run("流式中文输出测试", func(t *testing.T) {
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请用中文介绍一下春天的景色。",
			nil,
		)

		require.NoError(t, err)

		var fullText string
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应错误: %v", chunk.Error)
			}

			if !chunk.Done {
				fullText += chunk.Content
				// 验证每个块都是有效的 UTF-8
				assert.True(t, utf8.ValidString(chunk.Content), "数据块应该是有效的 UTF-8")
			}
		}

		assert.NotEmpty(t, fullText)
		assert.Contains(t, fullText, "春")
		t.Logf("中文流式输出: %s", fullText)
	})

	t.Run("流式错误处理 - 配置不存在", func(t *testing.T) {
		_, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
	})

	t.Run("流式错误处理 - 租户ID无效", func(t *testing.T) {
		_, err := client.GenerateStream(
			ctx,
			"invalid-uuid",
			modelName,
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的租户ID")
	})

	t.Run("流式错误处理 - 模型已禁用", func(t *testing.T) {
		// 禁用模型
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		_, err = client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")

		// 恢复模型状态
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	t.Run("流式性能测试 - TTFB", func(t *testing.T) {
		// 测试首字节时间（Time To First Byte）
		start := time.Now()

		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"你好",
			nil,
		)

		require.NoError(t, err)

		// 等待第一个数据块
		firstChunk := <-streamChan
		ttfb := time.Since(start)

		assert.False(t, firstChunk.Done, "第一个块不应该是完成标记")
		assert.NoError(t, firstChunk.Error)

		t.Logf("首字节时间 (TTFB): %v", ttfb)

		// 消费剩余的数据块
		for range streamChan {
		}
	})

	t.Run("流式并发调用测试", func(t *testing.T) {
		const concurrency = 3
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				streamChan, err := client.GenerateStream(
					ctx,
					tenantID.String(),
					modelName,
					fmt.Sprintf("这是第 %d 个并发流式请求", index+1),
					nil,
				)

				if err != nil {
					t.Errorf("并发流式调用错误: %v", err)
					done <- false
					return
				}

				var fullText string
				for chunk := range streamChan {
					if chunk.Error != nil {
						t.Errorf("流式响应错误: %v", chunk.Error)
						done <- false
						return
					}

					if !chunk.Done {
						fullText += chunk.Content
					}
				}

				if fullText == "" {
					t.Error("流式响应为空")
					done <- false
					return
				}

				t.Logf("并发请求 %d 完成，文本长度: %d", index+1, len(fullText))
				done <- true
			}(i)
		}

		// 等待所有并发请求完成
		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case success := <-done:
				if success {
					successCount++
				}
			case <-time.After(30 * time.Second):
				t.Fatal("并发流式调用超时")
			}
		}

		assert.Equal(t, concurrency, successCount, "所有并发流式调用都应该成功")
	})
}
