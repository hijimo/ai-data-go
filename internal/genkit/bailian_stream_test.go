package genkit

import (
	"context"
	"os"
	"testing"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBailianIntegration_Streaming 测试百炼流式调用
// 这是一个集成测试，需要真实的百炼配置
func TestBailianIntegration_Streaming(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 从环境变量获取百炼配置
	bailianAPIKey := os.Getenv("BAILIAN_API_KEY")
	bailianEndpoint := os.Getenv("BAILIAN_ENDPOINT")
	bailianModel := os.Getenv("BAILIAN_MODEL")

	if bailianAPIKey == "" {
		t.Skip("跳过百炼集成测试：缺少必需的环境变量 BAILIAN_API_KEY")
	}

	// 设置默认值
	if bailianEndpoint == "" {
		bailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if bailianModel == "" {
		bailianModel = "qwen-plus"
	}

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err)
	defer cleanupTestDatabase(t, db)

	// 创建测试租户和模型配置
	tenantID := uuid.New()
	modelName := "bailian-qwen-stream-test"

	// 准备 QueryParams JSON
	queryParams := `{
		"model": "` + bailianModel + `",
		"bailianEndpoint": "` + bailianEndpoint + `",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	// 创建模型配置
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
	require.NoError(t, err)

	// 创建 repository 和 client
	repo := repository.NewModelConfigurationRepository(db)
	client := NewClientWithRepo(repo)

	ctx := context.Background()

	t.Run("流式响应接收", func(t *testing.T) {
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
		var chunks []StreamChunk
		var fullText string
		var finalUsage *Usage

		for chunk := range streamChan {
			chunks = append(chunks, chunk)

			if chunk.Error != nil {
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}

			if !chunk.Done {
				fullText += chunk.Content
				t.Logf("接收到流式块: %s", chunk.Content)
			} else {
				// 最后一个块包含使用统计
				finalUsage = chunk.Usage
				t.Logf("流式响应完成")
			}
		}

		// 验证结果
		assert.NotEmpty(t, chunks, "应该接收到至少一个响应块")
		assert.NotEmpty(t, fullText, "完整文本不应为空")

		t.Logf("完整响应: %s", fullText)
		t.Logf("总共接收到 %d 个块", len(chunks))

		// 验证最后一个块是完成标记
		lastChunk := chunks[len(chunks)-1]
		assert.True(t, lastChunk.Done, "最后一个块应该标记为完成")

		// 验证 Token 统计
		if finalUsage != nil {
			assert.Greater(t, finalUsage.PromptTokens, 0, "Prompt tokens 应该大于 0")
			assert.Greater(t, finalUsage.CompletionTokens, 0, "Completion tokens 应该大于 0")
			assert.Greater(t, finalUsage.TotalTokens, 0, "Total tokens 应该大于 0")
			assert.Equal(t, finalUsage.PromptTokens+finalUsage.CompletionTokens, finalUsage.TotalTokens,
				"Total tokens 应该等于 Prompt + Completion")

			t.Logf("Token 使用情况: Prompt=%d, Completion=%d, Total=%d",
				finalUsage.PromptTokens,
				finalUsage.CompletionTokens,
				finalUsage.TotalTokens)
		}
	})

	t.Run("中文流式输出测试", func(t *testing.T) {
		// 测试中文流式生成
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请用中文描述春天的景色，包含花、鸟、风等元素。",
			nil,
		)
		require.NoError(t, err)

		// 收集所有流式块
		var streamText string
		var chunkCount int
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}
			if !chunk.Done {
				streamText += chunk.Content
				chunkCount++
				t.Logf("接收到第 %d 个块: %s", chunkCount, chunk.Content)
			}
		}

		// 验证响应完整性
		assert.NotEmpty(t, streamText, "流式响应不应为空")
		assert.Greater(t, chunkCount, 0, "应该接收到至少一个内容块")

		t.Logf("完整的中文流式响应: %s", streamText)
		t.Logf("总共接收到 %d 个内容块", chunkCount)
	})

	t.Run("流式响应完整性", func(t *testing.T) {
		// 调用流式接口
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请列举三个中国的传统节日。",
			nil,
		)
		require.NoError(t, err)

		// 收集所有流式块
		var streamText string
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}
			if !chunk.Done {
				streamText += chunk.Content
			}
		}

		// 验证响应完整性
		assert.NotEmpty(t, streamText, "流式响应不应为空")
		// 验证包含节日相关内容（至少有一些中文字符）
		assert.Greater(t, len(streamText), 10, "响应应该包含足够的内容")

		t.Logf("流式响应: %s", streamText)
	})

	t.Run("流式中断处理", func(t *testing.T) {
		// 创建可取消的上下文
		ctx, cancel := context.WithCancel(context.Background())

		// 调用流式接口
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请写一篇长文章，详细介绍人工智能的发展历史和未来趋势。",
			nil,
		)
		require.NoError(t, err)

		// 接收几个块后取消
		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				// 如果是上下文取消导致的错误，这是预期的
				if chunk.Error == context.Canceled {
					t.Logf("流式响应被正确取消")
					break
				}
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}

			if !chunk.Done {
				chunkCount++
				t.Logf("接收到第 %d 个块", chunkCount)

				// 接收到 3 个块后取消
				if chunkCount >= 3 {
					cancel()
					t.Logf("取消流式响应")
				}
			}
		}

		// 验证至少接收到了一些块
		assert.GreaterOrEqual(t, chunkCount, 1, "应该至少接收到一个块")
	})

	t.Run("流式参数传递", func(t *testing.T) {
		// 使用自定义选项
		temperature := 0.9
		maxTokens := 500
		options := &GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		// 调用流式接口
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请说一句话。",
			options,
		)
		require.NoError(t, err)

		// 收集响应
		var fullText string
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}
			if !chunk.Done {
				fullText += chunk.Content
			}
		}

		// 验证响应
		assert.NotEmpty(t, fullText, "响应不应为空")
		t.Logf("流式响应（自定义参数）: %s", fullText)
	})

	t.Run("验证SSE格式转换", func(t *testing.T) {
		// 调用流式接口
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请说'你好'。",
			nil,
		)
		require.NoError(t, err)

		// 验证每个块的格式
		chunkCount := 0
		var lastChunk StreamChunk

		for chunk := range streamChan {
			chunkCount++

			// 验证块的基本结构
			if chunk.Done {
				// 完成块应该包含模型信息和使用统计
				assert.NotEmpty(t, chunk.Model, "完成块应该包含模型名称")
				lastChunk = chunk
			} else {
				// 内容块应该包含文本
				assert.NotEmpty(t, chunk.Content, "内容块应该包含文本")
			}

			// 不应该有错误
			assert.Nil(t, chunk.Error, "不应该有错误")
		}

		// 验证至少接收到了一些块
		assert.Greater(t, chunkCount, 1, "应该接收到多个块")

		// 验证最后一个块是完成标记
		assert.True(t, lastChunk.Done, "最后一个块应该是完成标记")
		assert.Equal(t, bailianModel, lastChunk.Model, "模型名称应该正确")

		t.Logf("总共接收到 %d 个块", chunkCount)
	})

	t.Run("错误处理 - 配置不存在", func(t *testing.T) {
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

	t.Run("错误处理 - 租户ID无效", func(t *testing.T) {
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

	t.Run("错误处理 - 模型已禁用", func(t *testing.T) {
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

	t.Run("流式响应性能测试", func(t *testing.T) {
		// 记录首字节时间
		start := time.Now()
		var firstChunkTime time.Duration

		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请说'你好'。",
			nil,
		)
		require.NoError(t, err)

		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}

			if !chunk.Done && chunkCount == 0 {
				// 记录首字节时间
				firstChunkTime = time.Since(start)
				t.Logf("首字节时间: %v", firstChunkTime)
			}

			if !chunk.Done {
				chunkCount++
			}
		}

		totalTime := time.Since(start)
		t.Logf("总耗时: %v", totalTime)
		t.Logf("总共接收到 %d 个内容块", chunkCount)

		// 验证首字节时间在合理范围内（不超过 5 秒）
		assert.Less(t, firstChunkTime.Seconds(), 5.0, "首字节时间应该在 5 秒内")
	})

	t.Run("并发流式调用", func(t *testing.T) {
		// 并发调用流式接口
		concurrency := 3
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				defer func() { done <- true }()

				streamChan, err := client.GenerateStream(
					ctx,
					tenantID.String(),
					modelName,
					"请说一句话。",
					nil,
				)

				if err != nil {
					t.Errorf("并发调用 %d 失败: %v", index, err)
					return
				}

				// 收集响应
				var fullText string
				for chunk := range streamChan {
					if chunk.Error != nil {
						t.Errorf("并发调用 %d 流式响应出错: %v", index, chunk.Error)
						return
					}
					if !chunk.Done {
						fullText += chunk.Content
					}
				}

				t.Logf("并发调用 %d 完成: %s", index, fullText)
			}(i)
		}

		// 等待所有并发调用完成
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})

	t.Run("长文本流式生成", func(t *testing.T) {
		// 测试生成较长的文本
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请详细介绍一下中国的四大发明，每个发明用一段话描述。",
			nil,
		)
		require.NoError(t, err)

		// 收集所有流式块
		var streamText string
		var chunkCount int
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式响应出错: %v", chunk.Error)
			}
			if !chunk.Done {
				streamText += chunk.Content
				chunkCount++
			}
		}

		// 验证响应
		assert.NotEmpty(t, streamText, "流式响应不应为空")
		assert.Greater(t, chunkCount, 5, "长文本应该产生多个流式块")
		assert.Greater(t, len(streamText), 100, "长文本应该有足够的长度")

		t.Logf("长文本流式响应长度: %d 字符", len(streamText))
		t.Logf("总共接收到 %d 个内容块", chunkCount)
	})
}
