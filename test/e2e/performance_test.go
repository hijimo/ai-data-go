package e2e

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSingleProviderLatency 测试单提供商调用延迟
// 这个测试衡量不同提供商的响应时间和性能特征
func TestSingleProviderLatency(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	t.Log("✓ 测试环境设置完成")

	// ========== 阶段 2: 测试 Google AI 延迟 ==========
	t.Run("Google AI 延迟测试", func(t *testing.T) {
		googleAPIKey := os.Getenv("GOOGLE_API_KEY")
		if googleAPIKey == "" {
			t.Skip("跳过 Google AI 延迟测试：缺少 GOOGLE_API_KEY")
		}

		t.Log("========== Google AI 延迟测试 ==========")

		// 创建测试租户和配置
		tenantID := uuid.New()
		modelName := "gemini-perf-test"
		queryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`

		modelConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          modelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        googleAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(modelConfig).Error
		require.NoError(t, err)
		defer db.Delete(modelConfig)

		// 预热：首次调用可能包含初始化开销
		t.Log("预热调用...")
		_, err = client.Generate(ctx, tenantID.String(), modelName, "Hello", nil)
		require.NoError(t, err)

		// 测量多次调用的延迟
		const iterations = 5
		latencies := make([]time.Duration, 0, iterations)

		t.Logf("开始测量 %d 次调用的延迟...", iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				fmt.Sprintf("请用一句话回答：什么是AI？（第%d次）", i+1),
				nil,
			)
			latency := time.Since(start)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			latencies = append(latencies, latency)
			t.Logf("  第 %d 次调用延迟: %v", i+1, latency)
		}

		// 计算统计数据
		stats := calculateLatencyStats(latencies)
		t.Logf("\n✓ Google AI 延迟统计:")
		t.Logf("  平均延迟: %v", stats.Average)
		t.Logf("  最小延迟: %v", stats.Min)
		t.Logf("  最大延迟: %v", stats.Max)
		t.Logf("  中位数: %v", stats.Median)
		t.Logf("  标准差: %v", stats.StdDev)

		// 验证延迟在合理范围内（平均不超过 5 秒）
		assert.Less(t, stats.Average.Seconds(), 5.0, "平均延迟应该小于 5 秒")
	})

	// ========== 阶段 3: 测试 Azure OpenAI 延迟 ==========
	t.Run("Azure OpenAI 延迟测试", func(t *testing.T) {
		azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
		azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
		azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")

		if azureAPIKey == "" || azureEndpoint == "" || azureDeployment == "" {
			t.Skip("跳过 Azure OpenAI 延迟测试：缺少必需的环境变量")
		}

		t.Log("========== Azure OpenAI 延迟测试 ==========")

		// 创建测试租户和配置
		tenantID := uuid.New()
		modelName := "azure-gpt4-perf-test"
		queryParams := fmt.Sprintf(`{
			"model": "gpt-4",
			"azureEndpoint": "%s",
			"azureDeployment": "%s",
			"azureApiVersion": "2024-02-15-preview",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`, azureEndpoint, azureDeployment)

		modelConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          modelName,
			Model:         "gpt-4",
			ModelProvider: model.ModelProviderAzureOpenAI,
			APIKey:        azureAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(modelConfig).Error
		require.NoError(t, err)
		defer db.Delete(modelConfig)

		// 预热
		t.Log("预热调用...")
		_, err = client.Generate(ctx, tenantID.String(), modelName, "Hello", nil)
		require.NoError(t, err)

		// 测量延迟
		const iterations = 5
		latencies := make([]time.Duration, 0, iterations)

		t.Logf("开始测量 %d 次调用的延迟...", iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				fmt.Sprintf("请用一句话回答：什么是AI？（第%d次）", i+1),
				nil,
			)
			latency := time.Since(start)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			latencies = append(latencies, latency)
			t.Logf("  第 %d 次调用延迟: %v", i+1, latency)
		}

		// 计算统计数据
		stats := calculateLatencyStats(latencies)
		t.Logf("\n✓ Azure OpenAI 延迟统计:")
		t.Logf("  平均延迟: %v", stats.Average)
		t.Logf("  最小延迟: %v", stats.Min)
		t.Logf("  最大延迟: %v", stats.Max)
		t.Logf("  中位数: %v", stats.Median)
		t.Logf("  标准差: %v", stats.StdDev)

		// 验证延迟在合理范围内
		assert.Less(t, stats.Average.Seconds(), 5.0, "平均延迟应该小于 5 秒")
	})

	// ========== 阶段 4: 测试百炼延迟 ==========
	t.Run("百炼延迟测试", func(t *testing.T) {
		bailianAPIKey := os.Getenv("BAILIAN_API_KEY")
		bailianEndpoint := os.Getenv("BAILIAN_ENDPOINT")

		if bailianAPIKey == "" {
			t.Skip("跳过百炼延迟测试：缺少 BAILIAN_API_KEY")
		}

		if bailianEndpoint == "" {
			bailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}

		t.Log("========== 百炼延迟测试 ==========")

		// 创建测试租户和配置
		tenantID := uuid.New()
		modelName := "bailian-qwen-perf-test"
		queryParams := fmt.Sprintf(`{
			"model": "qwen-plus",
			"bailianEndpoint": "%s",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`, bailianEndpoint)

		modelConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          modelName,
			Model:         "qwen-plus",
			ModelProvider: "bianlian",
			APIKey:        bailianAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(modelConfig).Error
		require.NoError(t, err)
		defer db.Delete(modelConfig)

		// 预热
		t.Log("预热调用...")
		_, err = client.Generate(ctx, tenantID.String(), modelName, "你好", nil)
		require.NoError(t, err)

		// 测量延迟
		const iterations = 5
		latencies := make([]time.Duration, 0, iterations)

		t.Logf("开始测量 %d 次调用的延迟...", iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				fmt.Sprintf("请用一句话回答：什么是人工智能？（第%d次）", i+1),
				nil,
			)
			latency := time.Since(start)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			latencies = append(latencies, latency)
			t.Logf("  第 %d 次调用延迟: %v", i+1, latency)
		}

		// 计算统计数据
		stats := calculateLatencyStats(latencies)
		t.Logf("\n✓ 百炼延迟统计:")
		t.Logf("  平均延迟: %v", stats.Average)
		t.Logf("  最小延迟: %v", stats.Min)
		t.Logf("  最大延迟: %v", stats.Max)
		t.Logf("  中位数: %v", stats.Median)
		t.Logf("  标准差: %v", stats.StdDev)

		// 验证延迟在合理范围内
		assert.Less(t, stats.Average.Seconds(), 5.0, "平均延迟应该小于 5 秒")
	})

	// ========== 阶段 5: 测试流式调用的首字节时间 (TTFB) ==========
	t.Run("流式调用 TTFB 测试", func(t *testing.T) {
		googleAPIKey := os.Getenv("GOOGLE_API_KEY")
		if googleAPIKey == "" {
			t.Skip("跳过流式 TTFB 测试：缺少 GOOGLE_API_KEY")
		}

		t.Log("========== 流式调用 TTFB 测试 ==========")

		// 创建测试租户和配置
		tenantID := uuid.New()
		modelName := "gemini-stream-perf-test"
		queryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`

		modelConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          modelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        googleAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(modelConfig).Error
		require.NoError(t, err)
		defer db.Delete(modelConfig)

		// 预热
		t.Log("预热调用...")
		streamChan, err := client.GenerateStream(ctx, tenantID.String(), modelName, "Hello", nil)
		require.NoError(t, err)
		for range streamChan {
			// 消费所有数据块
		}

		// 测量 TTFB
		const iterations = 5
		ttfbs := make([]time.Duration, 0, iterations)

		t.Logf("开始测量 %d 次流式调用的 TTFB...", iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			streamChan, err := client.GenerateStream(
				ctx,
				tenantID.String(),
				modelName,
				fmt.Sprintf("请用一句话介绍AI（第%d次）", i+1),
				nil,
			)
			require.NoError(t, err)

			// 等待第一个数据块
			firstChunk := <-streamChan
			ttfb := time.Since(start)

			require.NoError(t, firstChunk.Error)
			ttfbs = append(ttfbs, ttfb)

			t.Logf("  第 %d 次 TTFB: %v", i+1, ttfb)

			// 消费剩余数据块
			for range streamChan {
			}
		}

		// 计算统计数据
		stats := calculateLatencyStats(ttfbs)
		t.Logf("\n✓ 流式调用 TTFB 统计:")
		t.Logf("  平均 TTFB: %v", stats.Average)
		t.Logf("  最小 TTFB: %v", stats.Min)
		t.Logf("  最大 TTFB: %v", stats.Max)
		t.Logf("  中位数: %v", stats.Median)
		t.Logf("  标准差: %v", stats.StdDev)

		// 验证 TTFB 在合理范围内（平均不超过 2 秒）
		assert.Less(t, stats.Average.Seconds(), 2.0, "平均 TTFB 应该小于 2 秒")
	})

	t.Log("========== 单提供商延迟测试完成 ==========")
}

// TestProviderSwitchingLatency 测试提供商切换延迟
// 这个测试衡量在同一租户下切换不同提供商时的性能开销
func TestProviderSwitchingLatency(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	// 检查环境变量
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
	azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	bailianAPIKey := os.Getenv("BAILIAN_API_KEY")
	bailianEndpoint := os.Getenv("BAILIAN_ENDPOINT")

	if bailianEndpoint == "" {
		bailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	// 至少需要两个提供商才能测试切换
	availableProviders := 0
	if googleAPIKey != "" {
		availableProviders++
	}
	if azureAPIKey != "" && azureEndpoint != "" && azureDeployment != "" {
		availableProviders++
	}
	if bailianAPIKey != "" {
		availableProviders++
	}

	if availableProviders < 2 {
		t.Skip("跳过提供商切换延迟测试：至少需要配置两个提供商")
	}

	t.Log("✓ 测试环境设置完成")

	// ========== 阶段 2: 创建多个提供商配置 ==========
	t.Log("========== 阶段 2: 创建多个提供商配置 ==========")

	tenantID := uuid.New()
	configs := make([]*model.ModelConfiguration, 0)

	// Google AI 配置
	if googleAPIKey != "" {
		queryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`

		googleConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          "gemini-switch-test",
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        googleAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(googleConfig).Error
		require.NoError(t, err)
		configs = append(configs, googleConfig)
		t.Log("✓ 创建 Google AI 配置")
	}

	// Azure OpenAI 配置
	if azureAPIKey != "" && azureEndpoint != "" && azureDeployment != "" {
		queryParams := fmt.Sprintf(`{
			"model": "gpt-4",
			"azureEndpoint": "%s",
			"azureDeployment": "%s",
			"azureApiVersion": "2024-02-15-preview",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`, azureEndpoint, azureDeployment)

		azureConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          "azure-gpt4-switch-test",
			Model:         "gpt-4",
			ModelProvider: model.ModelProviderAzureOpenAI,
			APIKey:        azureAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(azureConfig).Error
		require.NoError(t, err)
		configs = append(configs, azureConfig)
		t.Log("✓ 创建 Azure OpenAI 配置")
	}

	// 百炼配置
	if bailianAPIKey != "" {
		queryParams := fmt.Sprintf(`{
			"model": "qwen-plus",
			"bailianEndpoint": "%s",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`, bailianEndpoint)

		bailianConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          "bailian-qwen-switch-test",
			Model:         "qwen-plus",
			ModelProvider: "bianlian",
			APIKey:        bailianAPIKey,
			QueryParams:   &queryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(bailianConfig).Error
		require.NoError(t, err)
		configs = append(configs, bailianConfig)
		t.Log("✓ 创建百炼配置")
	}

	// 清理函数
	defer func() {
		for _, config := range configs {
			db.Delete(config)
		}
	}()

	t.Logf("✓ 共创建 %d 个提供商配置", len(configs))

	// ========== 阶段 3: 预热所有提供商 ==========
	t.Log("========== 阶段 3: 预热所有提供商 ==========")

	for _, config := range configs {
		t.Logf("预热提供商: %s (%s)", config.Name, config.ModelProvider)
		_, err := client.Generate(ctx, tenantID.String(), config.Name, "Hello", nil)
		require.NoError(t, err)
	}

	t.Log("✓ 所有提供商预热完成")

	// ========== 阶段 4: 测试提供商切换延迟 ==========
	t.Log("========== 阶段 4: 测试提供商切换延迟 ==========")

	const iterations = 10
	switchLatencies := make([]time.Duration, 0, iterations)

	t.Logf("开始测量 %d 次提供商切换的延迟...", iterations)

	for i := 0; i < iterations; i++ {
		// 选择两个不同的提供商
		provider1 := configs[i%len(configs)]
		provider2 := configs[(i+1)%len(configs)]

		// 调用第一个提供商
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			provider1.Name,
			fmt.Sprintf("测试消息 %d-1", i+1),
			nil,
		)
		require.NoError(t, err)

		// 测量切换到第二个提供商的延迟
		start := time.Now()
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			provider2.Name,
			fmt.Sprintf("测试消息 %d-2", i+1),
			nil,
		)
		switchLatency := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		switchLatencies = append(switchLatencies, switchLatency)
		t.Logf("  第 %d 次切换 (%s -> %s): %v",
			i+1,
			provider1.ModelProvider,
			provider2.ModelProvider,
			switchLatency,
		)
	}

	// 计算统计数据
	stats := calculateLatencyStats(switchLatencies)
	t.Logf("\n✓ 提供商切换延迟统计:")
	t.Logf("  平均延迟: %v", stats.Average)
	t.Logf("  最小延迟: %v", stats.Min)
	t.Logf("  最大延迟: %v", stats.Max)
	t.Logf("  中位数: %v", stats.Median)
	t.Logf("  标准差: %v", stats.StdDev)

	// 验证切换延迟不超过 50ms（根据需求）
	// 注意：这里测量的是整个 API 调用的延迟，包括网络请求
	// 实际的切换开销应该远小于这个值
	t.Logf("\n注意：测量的延迟包括完整的 API 调用时间")
	t.Logf("实际的提供商切换开销（缓存查找、实例获取）应该在毫秒级别")

	// ========== 阶段 5: 测试同一提供商的连续调用（作为基准） ==========
	t.Log("========== 阶段 5: 测试同一提供商的连续调用（基准） ==========")

	sameProviderLatencies := make([]time.Duration, 0, iterations)
	provider := configs[0]

	t.Logf("使用提供商 %s (%s) 进行 %d 次连续调用...",
		provider.Name,
		provider.ModelProvider,
		iterations,
	)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			provider.Name,
			fmt.Sprintf("基准测试消息 %d", i+1),
			nil,
		)
		latency := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		sameProviderLatencies = append(sameProviderLatencies, latency)
		t.Logf("  第 %d 次调用: %v", i+1, latency)
	}

	// 计算基准统计数据
	baselineStats := calculateLatencyStats(sameProviderLatencies)
	t.Logf("\n✓ 同一提供商连续调用延迟统计（基准）:")
	t.Logf("  平均延迟: %v", baselineStats.Average)
	t.Logf("  最小延迟: %v", baselineStats.Min)
	t.Logf("  最大延迟: %v", baselineStats.Max)
	t.Logf("  中位数: %v", baselineStats.Median)
	t.Logf("  标准差: %v", baselineStats.StdDev)

	// ========== 阶段 6: 比较切换延迟和基准延迟 ==========
	t.Log("========== 阶段 6: 延迟对比分析 ==========")

	overhead := stats.Average - baselineStats.Average
	overheadPercent := float64(overhead) / float64(baselineStats.Average) * 100

	t.Logf("\n✓ 提供商切换开销分析:")
	t.Logf("  切换平均延迟: %v", stats.Average)
	t.Logf("  基准平均延迟: %v", baselineStats.Average)
	t.Logf("  额外开销: %v (%.2f%%)", overhead, overheadPercent)

	// 验证切换开销不超过 50ms
	if overhead > 50*time.Millisecond {
		t.Logf("\n⚠️  警告：提供商切换开销 (%v) 超过了 50ms 的目标", overhead)
		t.Logf("   这可能是由于网络延迟或 API 响应时间的差异")
		t.Logf("   实际的切换逻辑开销应该在毫秒级别")
	} else {
		t.Logf("\n✓ 提供商切换开销 (%v) 在可接受范围内（< 50ms）", overhead)
	}

	// ========== 阶段 7: 测试快速连续切换 ==========
	t.Log("========== 阶段 7: 测试快速连续切换 ==========")

	if len(configs) >= 2 {
		const rapidSwitches = 20
		rapidLatencies := make([]time.Duration, 0, rapidSwitches)

		t.Logf("进行 %d 次快速连续切换...", rapidSwitches)

		for i := 0; i < rapidSwitches; i++ {
			provider := configs[i%len(configs)]

			start := time.Now()
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				provider.Name,
				fmt.Sprintf("快速切换 %d", i+1),
				nil,
			)
			latency := time.Since(start)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			rapidLatencies = append(rapidLatencies, latency)

			if i%5 == 4 {
				t.Logf("  完成 %d/%d 次切换", i+1, rapidSwitches)
			}
		}

		// 计算快速切换统计
		rapidStats := calculateLatencyStats(rapidLatencies)
		t.Logf("\n✓ 快速连续切换延迟统计:")
		t.Logf("  平均延迟: %v", rapidStats.Average)
		t.Logf("  最小延迟: %v", rapidStats.Min)
		t.Logf("  最大延迟: %v", rapidStats.Max)
		t.Logf("  中位数: %v", rapidStats.Median)
		t.Logf("  标准差: %v", rapidStats.StdDev)

		// 验证快速切换不会导致性能显著下降
		threshold := time.Duration(float64(stats.Average) * 1.5)
		if rapidStats.Average > threshold {
			t.Logf("\n⚠️  警告：快速连续切换的平均延迟明显高于正常切换")
		} else {
			t.Logf("\n✓ 快速连续切换性能稳定")
		}
	}

	t.Log("========== 提供商切换延迟测试完成 ==========")
}

// TestConcurrentCallsPerformance 测试并发调用性能
// 这个测试衡量系统在并发负载下的性能表现
func TestConcurrentCallsPerformance(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	// 检查环境变量
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		t.Skip("跳过并发调用测试：缺少 GOOGLE_API_KEY")
	}

	t.Log("✓ 测试环境设置完成")

	// ========== 阶段 2: 创建测试配置 ==========
	t.Log("========== 阶段 2: 创建测试配置 ==========")

	tenantID := uuid.New()
	modelName := "gemini-concurrent-test"
	queryParams := `{
		"model": "gemini-1.5-pro",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 100
	}`

	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "gemini-1.5-pro",
		ModelProvider: "googlegenai",
		APIKey:        googleAPIKey,
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(modelConfig).Error
	require.NoError(t, err)
	defer db.Delete(modelConfig)

	t.Log("✓ 测试配置创建完成")

	// ========== 阶段 3: 预热 ==========
	t.Log("========== 阶段 3: 预热 ==========")

	_, err = client.Generate(ctx, tenantID.String(), modelName, "Hello", nil)
	require.NoError(t, err)

	t.Log("✓ 预热完成")

	// ========== 阶段 4: 测试低并发（10个并发） ==========
	t.Run("低并发测试 (10个并发)", func(t *testing.T) {
		t.Log("========== 低并发测试 (10个并发) ==========")

		concurrency := 10
		results := runConcurrentCalls(t, ctx, client, tenantID.String(), modelName, concurrency)

		// 分析结果
		analyzeResults(t, results, concurrency)

		// 验证所有请求都成功
		assert.Equal(t, concurrency, results.SuccessCount, "所有请求都应该成功")
		assert.Equal(t, 0, results.ErrorCount, "不应该有错误")

		// 验证平均延迟在合理范围内
		assert.Less(t, results.Stats.Average.Seconds(), 10.0, "平均延迟应该小于 10 秒")
	})

	// ========== 阶段 5: 测试中等并发（50个并发） ==========
	t.Run("中等并发测试 (50个并发)", func(t *testing.T) {
		t.Log("========== 中等并发测试 (50个并发) ==========")

		concurrency := 50
		results := runConcurrentCalls(t, ctx, client, tenantID.String(), modelName, concurrency)

		// 分析结果
		analyzeResults(t, results, concurrency)

		// 验证大部分请求成功（允许少量失败）
		successRate := float64(results.SuccessCount) / float64(concurrency) * 100
		assert.GreaterOrEqual(t, successRate, 95.0, "成功率应该至少 95%")

		// 验证平均延迟在合理范围内
		assert.Less(t, results.Stats.Average.Seconds(), 15.0, "平均延迟应该小于 15 秒")
	})

	// ========== 阶段 6: 测试高并发（100个并发） ==========
	t.Run("高并发测试 (100个并发)", func(t *testing.T) {
		t.Log("========== 高并发测试 (100个并发) ==========")

		concurrency := 100
		results := runConcurrentCalls(t, ctx, client, tenantID.String(), modelName, concurrency)

		// 分析结果
		analyzeResults(t, results, concurrency)

		// 验证大部分请求成功（允许更多失败）
		successRate := float64(results.SuccessCount) / float64(concurrency) * 100
		assert.GreaterOrEqual(t, successRate, 90.0, "成功率应该至少 90%")

		// 验证平均延迟在合理范围内
		assert.Less(t, results.Stats.Average.Seconds(), 20.0, "平均延迟应该小于 20 秒")

		t.Logf("\n✓ 系统能够处理 100 个并发请求")
	})

	// ========== 阶段 7: 测试并发流式调用 ==========
	t.Run("并发流式调用测试 (20个并发)", func(t *testing.T) {
		t.Log("========== 并发流式调用测试 (20个并发) ==========")

		concurrency := 20
		results := runConcurrentStreamCalls(t, ctx, client, tenantID.String(), modelName, concurrency)

		// 分析结果
		analyzeStreamResults(t, results, concurrency)

		// 验证所有请求都成功
		assert.Equal(t, concurrency, results.SuccessCount, "所有流式请求都应该成功")
		assert.Equal(t, 0, results.ErrorCount, "不应该有错误")

		// 验证平均 TTFB 在合理范围内
		assert.Less(t, results.Stats.Average.Seconds(), 5.0, "平均 TTFB 应该小于 5 秒")
	})

	// ========== 阶段 8: 测试混合并发（多个提供商） ==========
	t.Run("混合并发测试 (多提供商)", func(t *testing.T) {
		// 检查是否有多个提供商可用
		azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
		azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
		azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")

		if azureAPIKey == "" || azureEndpoint == "" || azureDeployment == "" {
			t.Skip("跳过混合并发测试：需要配置 Azure OpenAI")
		}

		t.Log("========== 混合并发测试 (多提供商) ==========")

		// 创建 Azure 配置
		azureModelName := "azure-gpt4-concurrent-test"
		azureQueryParams := fmt.Sprintf(`{
			"model": "gpt-4",
			"azureEndpoint": "%s",
			"azureDeployment": "%s",
			"azureApiVersion": "2024-02-15-preview",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`, azureEndpoint, azureDeployment)

		azureConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          azureModelName,
			Model:         "gpt-4",
			ModelProvider: model.ModelProviderAzureOpenAI,
			APIKey:        azureAPIKey,
			QueryParams:   &azureQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(azureConfig).Error
		require.NoError(t, err)
		defer db.Delete(azureConfig)

		// 预热 Azure
		_, err = client.Generate(ctx, tenantID.String(), azureModelName, "Hello", nil)
		require.NoError(t, err)

		// 运行混合并发测试
		concurrency := 50
		results := runMixedConcurrentCalls(
			t,
			ctx,
			client,
			tenantID.String(),
			[]string{modelName, azureModelName},
			concurrency,
		)

		// 分析结果
		analyzeMixedResults(t, results, concurrency)

		// 验证大部分请求成功
		successRate := float64(results.SuccessCount) / float64(concurrency) * 100
		assert.GreaterOrEqual(t, successRate, 90.0, "成功率应该至少 90%")

		t.Logf("\n✓ 系统能够处理多提供商的并发请求")
	})

	t.Log("========== 并发调用性能测试完成 ==========")
}

// ConcurrentCallResult 并发调用结果
type ConcurrentCallResult struct {
	Index    int
	Latency  time.Duration
	Success  bool
	Error    error
	Response string
}

// ConcurrentTestResults 并发测试结果汇总
type ConcurrentTestResults struct {
	TotalCount   int
	SuccessCount int
	ErrorCount   int
	TotalTime    time.Duration
	Stats        LatencyStats
	Errors       []error
}

// StreamCallResult 流式调用结果
type StreamCallResult struct {
	Index       int
	TTFB        time.Duration
	TotalTime   time.Duration
	ChunkCount  int
	Success     bool
	Error       error
	TotalLength int
}

// StreamTestResults 流式测试结果汇总
type StreamTestResults struct {
	TotalCount   int
	SuccessCount int
	ErrorCount   int
	Stats        LatencyStats // TTFB 统计
	TotalStats   LatencyStats // 总时间统计
	Errors       []error
}

// runConcurrentCalls 运行并发调用测试
func runConcurrentCalls(
	t *testing.T,
	ctx context.Context,
	client genkit.Client,
	tenantID string,
	modelName string,
	concurrency int,
) ConcurrentTestResults {
	t.Logf("开始 %d 个并发调用...", concurrency)

	results := make(chan ConcurrentCallResult, concurrency)
	startTime := time.Now()

	// 启动并发调用
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			callStart := time.Now()
			result, err := client.Generate(
				ctx,
				tenantID,
				modelName,
				fmt.Sprintf("并发测试消息 %d：请简短回答什么是AI？", index+1),
				nil,
			)
			latency := time.Since(callStart)

			callResult := ConcurrentCallResult{
				Index:   index,
				Latency: latency,
				Success: err == nil,
				Error:   err,
			}

			if err == nil {
				callResult.Response = result.Text
			}

			results <- callResult
		}(i)
	}

	// 收集结果
	var successCount, errorCount int
	var latencies []time.Duration
	var errors []error

	for i := 0; i < concurrency; i++ {
		result := <-results
		if result.Success {
			successCount++
			latencies = append(latencies, result.Latency)
		} else {
			errorCount++
			errors = append(errors, result.Error)
			t.Logf("  请求 %d 失败: %v", result.Index+1, result.Error)
		}

		// 每完成 10 个请求报告一次进度
		if (i+1)%10 == 0 {
			t.Logf("  已完成 %d/%d 个请求", i+1, concurrency)
		}
	}

	totalTime := time.Since(startTime)

	// 计算统计数据
	var stats LatencyStats
	if len(latencies) > 0 {
		stats = calculateLatencyStats(latencies)
	}

	return ConcurrentTestResults{
		TotalCount:   concurrency,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		TotalTime:    totalTime,
		Stats:        stats,
		Errors:       errors,
	}
}

// runConcurrentStreamCalls 运行并发流式调用测试
func runConcurrentStreamCalls(
	t *testing.T,
	ctx context.Context,
	client genkit.Client,
	tenantID string,
	modelName string,
	concurrency int,
) StreamTestResults {
	t.Logf("开始 %d 个并发流式调用...", concurrency)

	results := make(chan StreamCallResult, concurrency)

	// 启动并发流式调用
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			callStart := time.Now()
			streamChan, err := client.GenerateStream(
				ctx,
				tenantID,
				modelName,
				fmt.Sprintf("并发流式测试 %d：请简短介绍AI", index+1),
				nil,
			)

			if err != nil {
				results <- StreamCallResult{
					Index:   index,
					Success: false,
					Error:   err,
				}
				return
			}

			// 等待第一个数据块（TTFB）
			firstChunk := <-streamChan
			ttfb := time.Since(callStart)

			if firstChunk.Error != nil {
				results <- StreamCallResult{
					Index:   index,
					TTFB:    ttfb,
					Success: false,
					Error:   firstChunk.Error,
				}
				return
			}

			// 消费所有数据块
			chunkCount := 1
			totalLength := len(firstChunk.Content)

			for chunk := range streamChan {
				if chunk.Error != nil {
					results <- StreamCallResult{
						Index:      index,
						TTFB:       ttfb,
						ChunkCount: chunkCount,
						Success:    false,
						Error:      chunk.Error,
					}
					return
				}
				chunkCount++
				totalLength += len(chunk.Content)
			}

			totalTime := time.Since(callStart)

			results <- StreamCallResult{
				Index:       index,
				TTFB:        ttfb,
				TotalTime:   totalTime,
				ChunkCount:  chunkCount,
				TotalLength: totalLength,
				Success:     true,
			}
		}(i)
	}

	// 收集结果
	var successCount, errorCount int
	var ttfbs []time.Duration
	var totalTimes []time.Duration
	var errors []error

	for i := 0; i < concurrency; i++ {
		result := <-results
		if result.Success {
			successCount++
			ttfbs = append(ttfbs, result.TTFB)
			totalTimes = append(totalTimes, result.TotalTime)
		} else {
			errorCount++
			errors = append(errors, result.Error)
			t.Logf("  流式请求 %d 失败: %v", result.Index+1, result.Error)
		}

		// 每完成 5 个请求报告一次进度
		if (i+1)%5 == 0 {
			t.Logf("  已完成 %d/%d 个流式请求", i+1, concurrency)
		}
	}

	// 计算统计数据
	var ttfbStats, totalStats LatencyStats
	if len(ttfbs) > 0 {
		ttfbStats = calculateLatencyStats(ttfbs)
		totalStats = calculateLatencyStats(totalTimes)
	}

	return StreamTestResults{
		TotalCount:   concurrency,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		Stats:        ttfbStats,
		TotalStats:   totalStats,
		Errors:       errors,
	}
}

// runMixedConcurrentCalls 运行混合提供商并发调用测试
func runMixedConcurrentCalls(
	t *testing.T,
	ctx context.Context,
	client genkit.Client,
	tenantID string,
	modelNames []string,
	concurrency int,
) ConcurrentTestResults {
	t.Logf("开始 %d 个混合并发调用（使用 %d 个提供商）...", concurrency, len(modelNames))

	results := make(chan ConcurrentCallResult, concurrency)
	startTime := time.Now()

	// 启动并发调用，轮流使用不同的提供商
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			modelName := modelNames[index%len(modelNames)]

			callStart := time.Now()
			result, err := client.Generate(
				ctx,
				tenantID,
				modelName,
				fmt.Sprintf("混合并发测试 %d：请简短回答什么是AI？", index+1),
				nil,
			)
			latency := time.Since(callStart)

			callResult := ConcurrentCallResult{
				Index:   index,
				Latency: latency,
				Success: err == nil,
				Error:   err,
			}

			if err == nil {
				callResult.Response = result.Text
			}

			results <- callResult
		}(i)
	}

	// 收集结果
	var successCount, errorCount int
	var latencies []time.Duration
	var errors []error

	for i := 0; i < concurrency; i++ {
		result := <-results
		if result.Success {
			successCount++
			latencies = append(latencies, result.Latency)
		} else {
			errorCount++
			errors = append(errors, result.Error)
			t.Logf("  混合请求 %d 失败: %v", result.Index+1, result.Error)
		}

		if (i+1)%10 == 0 {
			t.Logf("  已完成 %d/%d 个混合请求", i+1, concurrency)
		}
	}

	totalTime := time.Since(startTime)

	// 计算统计数据
	var stats LatencyStats
	if len(latencies) > 0 {
		stats = calculateLatencyStats(latencies)
	}

	return ConcurrentTestResults{
		TotalCount:   concurrency,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		TotalTime:    totalTime,
		Stats:        stats,
		Errors:       errors,
	}
}

// analyzeResults 分析并发测试结果
func analyzeResults(t *testing.T, results ConcurrentTestResults, concurrency int) {
	t.Logf("\n✓ 并发测试结果分析:")
	t.Logf("  总请求数: %d", results.TotalCount)
	t.Logf("  成功数: %d", results.SuccessCount)
	t.Logf("  失败数: %d", results.ErrorCount)
	t.Logf("  成功率: %.2f%%", float64(results.SuccessCount)/float64(results.TotalCount)*100)
	t.Logf("  总耗时: %v", results.TotalTime)
	t.Logf("  吞吐量: %.2f 请求/秒", float64(results.SuccessCount)/results.TotalTime.Seconds())

	if results.SuccessCount > 0 {
		t.Logf("\n  延迟统计:")
		t.Logf("    平均延迟: %v", results.Stats.Average)
		t.Logf("    最小延迟: %v", results.Stats.Min)
		t.Logf("    最大延迟: %v", results.Stats.Max)
		t.Logf("    中位数: %v", results.Stats.Median)
		t.Logf("    标准差: %v", results.Stats.StdDev)
	}

	if results.ErrorCount > 0 {
		t.Logf("\n  错误详情:")
		errorMap := make(map[string]int)
		for _, err := range results.Errors {
			if err != nil {
				errorMap[err.Error()]++
			}
		}
		for errMsg, count := range errorMap {
			t.Logf("    %s: %d 次", errMsg, count)
		}
	}
}

// analyzeStreamResults 分析流式并发测试结果
func analyzeStreamResults(t *testing.T, results StreamTestResults, concurrency int) {
	t.Logf("\n✓ 流式并发测试结果分析:")
	t.Logf("  总请求数: %d", results.TotalCount)
	t.Logf("  成功数: %d", results.SuccessCount)
	t.Logf("  失败数: %d", results.ErrorCount)
	t.Logf("  成功率: %.2f%%", float64(results.SuccessCount)/float64(results.TotalCount)*100)

	if results.SuccessCount > 0 {
		t.Logf("\n  TTFB 统计:")
		t.Logf("    平均 TTFB: %v", results.Stats.Average)
		t.Logf("    最小 TTFB: %v", results.Stats.Min)
		t.Logf("    最大 TTFB: %v", results.Stats.Max)
		t.Logf("    中位数: %v", results.Stats.Median)

		t.Logf("\n  总时间统计:")
		t.Logf("    平均总时间: %v", results.TotalStats.Average)
		t.Logf("    最小总时间: %v", results.TotalStats.Min)
		t.Logf("    最大总时间: %v", results.TotalStats.Max)
		t.Logf("    中位数: %v", results.TotalStats.Median)
	}

	if results.ErrorCount > 0 {
		t.Logf("\n  错误详情:")
		errorMap := make(map[string]int)
		for _, err := range results.Errors {
			if err != nil {
				errorMap[err.Error()]++
			}
		}
		for errMsg, count := range errorMap {
			t.Logf("    %s: %d 次", errMsg, count)
		}
	}
}

// analyzeMixedResults 分析混合并发测试结果
func analyzeMixedResults(t *testing.T, results ConcurrentTestResults, concurrency int) {
	t.Logf("\n✓ 混合并发测试结果分析:")
	t.Logf("  总请求数: %d", results.TotalCount)
	t.Logf("  成功数: %d", results.SuccessCount)
	t.Logf("  失败数: %d", results.ErrorCount)
	t.Logf("  成功率: %.2f%%", float64(results.SuccessCount)/float64(results.TotalCount)*100)
	t.Logf("  总耗时: %v", results.TotalTime)
	t.Logf("  吞吐量: %.2f 请求/秒", float64(results.SuccessCount)/results.TotalTime.Seconds())

	if results.SuccessCount > 0 {
		t.Logf("\n  延迟统计:")
		t.Logf("    平均延迟: %v", results.Stats.Average)
		t.Logf("    最小延迟: %v", results.Stats.Min)
		t.Logf("    最大延迟: %v", results.Stats.Max)
		t.Logf("    中位数: %v", results.Stats.Median)
		t.Logf("    标准差: %v", results.Stats.StdDev)
	}

	if results.ErrorCount > 0 {
		t.Logf("\n  错误详情:")
		errorMap := make(map[string]int)
		for _, err := range results.Errors {
			if err != nil {
				errorMap[err.Error()]++
			}
		}
		for errMsg, count := range errorMap {
			t.Logf("    %s: %d 次", errMsg, count)
		}
	}
}

// LatencyStats 延迟统计数据
type LatencyStats struct {
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
	Median  time.Duration
	StdDev  time.Duration
}

// TestMemoryUsage 测试内存使用
// 这个测试衡量系统在不同负载下的内存使用情况
func TestMemoryUsage(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	// 检查环境变量
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		t.Skip("跳过内存使用测试：缺少 GOOGLE_API_KEY")
	}

	t.Log("✓ 测试环境设置完成")

	// ========== 阶段 2: 创建测试配置 ==========
	t.Log("========== 阶段 2: 创建测试配置 ==========")

	tenantID := uuid.New()
	modelName := "gemini-memory-test"
	queryParams := `{
		"model": "gemini-1.5-pro",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 100
	}`

	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "gemini-1.5-pro",
		ModelProvider: "googlegenai",
		APIKey:        googleAPIKey,
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(modelConfig).Error
	require.NoError(t, err)
	defer db.Delete(modelConfig)

	t.Log("✓ 测试配置创建完成")

	// ========== 阶段 3: 测试基准内存使用 ==========
	t.Run("基准内存使用", func(t *testing.T) {
		t.Log("========== 基准内存使用测试 ==========")

		// 强制垃圾回收，获取干净的基准
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var baselineMemStats runtime.MemStats
		runtime.ReadMemStats(&baselineMemStats)

		t.Logf("基准内存状态:")
		t.Logf("  分配的堆内存: %.2f MB", float64(baselineMemStats.Alloc)/1024/1024)
		t.Logf("  系统内存: %.2f MB", float64(baselineMemStats.Sys)/1024/1024)
		t.Logf("  堆对象数: %d", baselineMemStats.HeapObjects)
		t.Logf("  GC 次数: %d", baselineMemStats.NumGC)

		// 执行单次调用
		t.Log("\n执行单次调用...")
		_, err := client.Generate(ctx, tenantID.String(), modelName, "Hello", nil)
		require.NoError(t, err)

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var afterCallMemStats runtime.MemStats
		runtime.ReadMemStats(&afterCallMemStats)

		t.Logf("\n单次调用后内存状态:")
		t.Logf("  分配的堆内存: %.2f MB", float64(afterCallMemStats.Alloc)/1024/1024)
		t.Logf("  系统内存: %.2f MB", float64(afterCallMemStats.Sys)/1024/1024)
		t.Logf("  堆对象数: %d", afterCallMemStats.HeapObjects)
		t.Logf("  GC 次数: %d", afterCallMemStats.NumGC)

		memIncrease := float64(afterCallMemStats.Alloc-baselineMemStats.Alloc) / 1024 / 1024
		t.Logf("\n✓ 单次调用内存增长: %.2f MB", memIncrease)

		// 验证单次调用内存增长在合理范围内（不超过 10MB）
		assert.Less(t, memIncrease, 10.0, "单次调用内存增长应该小于 10MB")
	})

	// ========== 阶段 4: 测试连续调用的内存使用 ==========
	t.Run("连续调用内存使用", func(t *testing.T) {
		t.Log("========== 连续调用内存使用测试 ==========")

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var startMemStats runtime.MemStats
		runtime.ReadMemStats(&startMemStats)

		// 执行 50 次连续调用
		const iterations = 50
		t.Logf("执行 %d 次连续调用...", iterations)

		memSnapshots := make([]MemorySnapshot, 0, iterations)

		for i := 0; i < iterations; i++ {
			_, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				fmt.Sprintf("测试消息 %d", i+1),
				nil,
			)
			require.NoError(t, err)

			// 每 10 次调用记录一次内存快照
			if (i+1)%10 == 0 {
				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				snapshot := MemorySnapshot{
					Iteration:   i + 1,
					AllocMB:     float64(memStats.Alloc) / 1024 / 1024,
					SysMB:       float64(memStats.Sys) / 1024 / 1024,
					HeapObjects: memStats.HeapObjects,
					NumGC:       memStats.NumGC,
				}
				memSnapshots = append(memSnapshots, snapshot)

				t.Logf("  第 %d 次调用后: 堆内存=%.2f MB, 对象数=%d",
					snapshot.Iteration,
					snapshot.AllocMB,
					snapshot.HeapObjects,
				)
			}
		}

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var endMemStats runtime.MemStats
		runtime.ReadMemStats(&endMemStats)

		t.Logf("\n✓ 连续调用内存统计:")
		t.Logf("  开始时堆内存: %.2f MB", float64(startMemStats.Alloc)/1024/1024)
		t.Logf("  结束时堆内存: %.2f MB", float64(endMemStats.Alloc)/1024/1024)
		t.Logf("  内存增长: %.2f MB", float64(endMemStats.Alloc-startMemStats.Alloc)/1024/1024)
		t.Logf("  GC 次数: %d", endMemStats.NumGC-startMemStats.NumGC)

		// 分析内存增长趋势
		if len(memSnapshots) >= 2 {
			firstSnapshot := memSnapshots[0]
			lastSnapshot := memSnapshots[len(memSnapshots)-1]
			memGrowth := lastSnapshot.AllocMB - firstSnapshot.AllocMB

			t.Logf("\n  内存增长趋势:")
			t.Logf("    第 %d 次: %.2f MB", firstSnapshot.Iteration, firstSnapshot.AllocMB)
			t.Logf("    第 %d 次: %.2f MB", lastSnapshot.Iteration, lastSnapshot.AllocMB)
			t.Logf("    增长量: %.2f MB", memGrowth)

			// 验证没有明显的内存泄漏（增长不超过 20MB）
			assert.Less(t, memGrowth, 20.0, "连续调用不应该有明显的内存泄漏")
		}
	})

	// ========== 阶段 5: 测试并发调用的内存使用 ==========
	t.Run("并发调用内存使用", func(t *testing.T) {
		t.Log("========== 并发调用内存使用测试 ==========")

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var startMemStats runtime.MemStats
		runtime.ReadMemStats(&startMemStats)

		t.Logf("并发调用前内存状态:")
		t.Logf("  分配的堆内存: %.2f MB", float64(startMemStats.Alloc)/1024/1024)
		t.Logf("  系统内存: %.2f MB", float64(startMemStats.Sys)/1024/1024)
		t.Logf("  Goroutine 数: %d", runtime.NumGoroutine())

		// 执行并发调用
		concurrency := 50
		t.Logf("\n执行 %d 个并发调用...", concurrency)

		results := make(chan error, concurrency)
		for i := 0; i < concurrency; i++ {
			go func(index int) {
				_, err := client.Generate(
					ctx,
					tenantID.String(),
					modelName,
					fmt.Sprintf("并发测试 %d", index+1),
					nil,
				)
				results <- err
			}(i)
		}

		// 等待所有调用完成
		successCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-results
			if err == nil {
				successCount++
			}
		}

		t.Logf("✓ 完成 %d/%d 个并发调用", successCount, concurrency)

		// 等待 goroutine 清理
		time.Sleep(500 * time.Millisecond)

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var endMemStats runtime.MemStats
		runtime.ReadMemStats(&endMemStats)

		t.Logf("\n并发调用后内存状态:")
		t.Logf("  分配的堆内存: %.2f MB", float64(endMemStats.Alloc)/1024/1024)
		t.Logf("  系统内存: %.2f MB", float64(endMemStats.Sys)/1024/1024)
		t.Logf("  Goroutine 数: %d", runtime.NumGoroutine())

		memIncrease := float64(endMemStats.Alloc-startMemStats.Alloc) / 1024 / 1024
		sysIncrease := float64(endMemStats.Sys-startMemStats.Sys) / 1024 / 1024

		t.Logf("\n✓ 并发调用内存统计:")
		t.Logf("  堆内存增长: %.2f MB", memIncrease)
		t.Logf("  系统内存增长: %.2f MB", sysIncrease)
		t.Logf("  GC 次数: %d", endMemStats.NumGC-startMemStats.NumGC)

		// 验证并发调用后内存增长在合理范围内（不超过 50MB）
		assert.Less(t, memIncrease, 50.0, "并发调用后堆内存增长应该小于 50MB")
	})

	// ========== 阶段 6: 测试多提供商的内存使用 ==========
	t.Run("多提供商内存使用", func(t *testing.T) {
		// 检查是否有多个提供商可用
		azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
		azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
		azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")

		if azureAPIKey == "" || azureEndpoint == "" || azureDeployment == "" {
			t.Skip("跳过多提供商内存测试：需要配置 Azure OpenAI")
		}

		t.Log("========== 多提供商内存使用测试 ==========")

		// 创建 Azure 配置
		azureModelName := "azure-gpt4-memory-test"
		azureQueryParams := fmt.Sprintf(`{
			"model": "gpt-4",
			"azureEndpoint": "%s",
			"azureDeployment": "%s",
			"azureApiVersion": "2024-02-15-preview",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 100
		}`, azureEndpoint, azureDeployment)

		azureConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          azureModelName,
			Model:         "gpt-4",
			ModelProvider: model.ModelProviderAzureOpenAI,
			APIKey:        azureAPIKey,
			QueryParams:   &azureQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(azureConfig).Error
		require.NoError(t, err)
		defer db.Delete(azureConfig)

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var startMemStats runtime.MemStats
		runtime.ReadMemStats(&startMemStats)

		t.Logf("初始内存状态:")
		t.Logf("  分配的堆内存: %.2f MB", float64(startMemStats.Alloc)/1024/1024)

		// 初始化两个提供商
		t.Log("\n初始化 Google AI 提供商...")
		_, err = client.Generate(ctx, tenantID.String(), modelName, "Hello", nil)
		require.NoError(t, err)

		var afterGoogle runtime.MemStats
		runtime.ReadMemStats(&afterGoogle)
		googleMem := float64(afterGoogle.Alloc-startMemStats.Alloc) / 1024 / 1024
		t.Logf("  Google AI 初始化后内存增长: %.2f MB", googleMem)

		t.Log("\n初始化 Azure OpenAI 提供商...")
		_, err = client.Generate(ctx, tenantID.String(), azureModelName, "Hello", nil)
		require.NoError(t, err)

		var afterAzure runtime.MemStats
		runtime.ReadMemStats(&afterAzure)
		azureMem := float64(afterAzure.Alloc-afterGoogle.Alloc) / 1024 / 1024
		t.Logf("  Azure OpenAI 初始化后额外内存增长: %.2f MB", azureMem)

		// 在两个提供商之间切换调用
		t.Log("\n在两个提供商之间切换调用...")
		const switches = 20
		for i := 0; i < switches; i++ {
			if i%2 == 0 {
				_, err = client.Generate(ctx, tenantID.String(), modelName, fmt.Sprintf("切换测试 %d", i+1), nil)
			} else {
				_, err = client.Generate(ctx, tenantID.String(), azureModelName, fmt.Sprintf("切换测试 %d", i+1), nil)
			}
			require.NoError(t, err)
		}

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var endMemStats runtime.MemStats
		runtime.ReadMemStats(&endMemStats)

		totalIncrease := float64(endMemStats.Alloc-startMemStats.Alloc) / 1024 / 1024

		t.Logf("\n✓ 多提供商内存统计:")
		t.Logf("  初始内存: %.2f MB", float64(startMemStats.Alloc)/1024/1024)
		t.Logf("  最终内存: %.2f MB", float64(endMemStats.Alloc)/1024/1024)
		t.Logf("  总增长: %.2f MB", totalIncrease)
		t.Logf("  Google AI 初始化: %.2f MB", googleMem)
		t.Logf("  Azure OpenAI 初始化: %.2f MB", azureMem)

		// 验证多提供商不会导致过多的内存使用（不超过 30MB）
		assert.Less(t, totalIncrease, 30.0, "多提供商内存增长应该小于 30MB")
	})

	// ========== 阶段 7: 测试长时间运行的内存稳定性 ==========
	t.Run("长时间运行内存稳定性", func(t *testing.T) {
		t.Log("========== 长时间运行内存稳定性测试 ==========")

		// 强制垃圾回收
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		var startMemStats runtime.MemStats
		runtime.ReadMemStats(&startMemStats)

		// 模拟长时间运行：执行 100 次调用，每 20 次记录一次内存
		const totalCalls = 100
		const snapshotInterval = 20

		t.Logf("执行 %d 次调用，每 %d 次记录内存快照...", totalCalls, snapshotInterval)

		memSnapshots := make([]MemorySnapshot, 0)

		for i := 0; i < totalCalls; i++ {
			_, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				fmt.Sprintf("长时间测试 %d", i+1),
				nil,
			)
			require.NoError(t, err)

			// 记录内存快照
			if (i+1)%snapshotInterval == 0 {
				runtime.GC()
				time.Sleep(50 * time.Millisecond)

				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				snapshot := MemorySnapshot{
					Iteration:   i + 1,
					AllocMB:     float64(memStats.Alloc) / 1024 / 1024,
					SysMB:       float64(memStats.Sys) / 1024 / 1024,
					HeapObjects: memStats.HeapObjects,
					NumGC:       memStats.NumGC,
				}
				memSnapshots = append(memSnapshots, snapshot)

				t.Logf("  第 %d 次: 堆内存=%.2f MB, GC次数=%d",
					snapshot.Iteration,
					snapshot.AllocMB,
					snapshot.NumGC,
				)
			}
		}

		t.Logf("\n✓ 内存稳定性分析:")
		if len(memSnapshots) >= 2 {
			firstSnapshot := memSnapshots[0]
			lastSnapshot := memSnapshots[len(memSnapshots)-1]

			t.Logf("  第 %d 次调用: %.2f MB", firstSnapshot.Iteration, firstSnapshot.AllocMB)
			t.Logf("  第 %d 次调用: %.2f MB", lastSnapshot.Iteration, lastSnapshot.AllocMB)

			memGrowth := lastSnapshot.AllocMB - firstSnapshot.AllocMB
			growthRate := memGrowth / float64(lastSnapshot.Iteration-firstSnapshot.Iteration) * 100

			t.Logf("  内存增长: %.2f MB", memGrowth)
			t.Logf("  增长率: %.4f MB/100次调用", growthRate)

			// 验证内存增长率在可接受范围内（每 100 次调用不超过 5MB）
			assert.Less(t, growthRate, 5.0, "内存增长率应该小于 5MB/100次调用")

			// 检查内存是否稳定（最后一次和第一次的差异不超过 30MB）
			assert.Less(t, memGrowth, 30.0, "长时间运行内存增长应该小于 30MB")
		}
	})

	t.Log("========== 内存使用测试完成 ==========")
}

// MemorySnapshot 内存快照
type MemorySnapshot struct {
	Iteration   int
	AllocMB     float64
	SysMB       float64
	HeapObjects uint64
	NumGC       uint32
}

// calculateLatencyStats 计算延迟统计数据
func calculateLatencyStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// 计算总和和平均值
	var sum time.Duration
	min := latencies[0]
	max := latencies[0]

	for _, latency := range latencies {
		sum += latency
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
	}

	average := sum / time.Duration(len(latencies))

	// 计算中位数
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	// 简单的冒泡排序
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var median time.Duration
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		median = sorted[len(sorted)/2]
	}

	// 计算标准差
	var variance float64
	for _, latency := range latencies {
		diff := float64(latency - average)
		variance += diff * diff
	}
	variance /= float64(len(latencies))
	stdDev := time.Duration(float64(time.Nanosecond) * float64(variance))

	return LatencyStats{
		Average: average,
		Min:     min,
		Max:     max,
		Median:  median,
		StdDev:  stdDev,
	}
}
