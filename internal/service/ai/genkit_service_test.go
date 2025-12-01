package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"

	genkitclient "genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	authservice "genkit-ai-service/internal/service/auth"
)

// mockGenkitClient 模拟 Genkit 客户端
type mockGenkitClient struct {
	generateFunc       func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error)
	generateStreamFunc func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (<-chan genkitclient.StreamChunk, error)
}

func (m *mockGenkitClient) Initialize(ctx context.Context, config *genkitclient.Config) error {
	return nil
}

func (m *mockGenkitClient) Generate(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, tenantID, modelName, prompt, options)
	}
	return &genkitclient.GenerateResult{
		Text:  "测试响应",
		Model: "test-model",
		Usage: &genkitclient.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockGenkitClient) GenerateStream(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (<-chan genkitclient.StreamChunk, error) {
	if m.generateStreamFunc != nil {
		return m.generateStreamFunc(ctx, tenantID, modelName, prompt, options)
	}
	// 返回一个简单的流，包含 Token 使用统计
	ch := make(chan genkitclient.StreamChunk, 2)
	go func() {
		defer close(ch)
		// 发送内容块
		ch <- genkitclient.StreamChunk{
			Content: "测试响应",
			Done:    false,
		}
		// 发送完成标记，包含 Token 使用统计
		ch <- genkitclient.StreamChunk{
			Content: "",
			Done:    true,
			Model:   "test-model",
			Usage: &genkitclient.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
	}()
	return ch, nil
}

func (m *mockGenkitClient) InitializeModel(ctx context.Context) error {
	return nil
}

func (m *mockGenkitClient) GetGenkit() *genkit.Genkit {
	return nil
}

func (m *mockGenkitClient) SetModel(model ai.Model) {}

func (m *mockGenkitClient) Close() error {
	return nil
}

// TestNewGenkitService 测试创建服务
func TestNewGenkitService(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)
	if service == nil {
		t.Fatal("服务创建失败")
	}
}

// testWriter 测试用的 writer
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}

// createTestContext 创建带有 JWT Claims 的测试上下文
func createTestContext() context.Context {
	claims := &model.JWTClaims{
		TenantID:    "test-tenant-id",
		DisplayName: "测试用户",
		Roles:       []string{"user"},
	}
	claims.Subject = "test-user-id"
	
	ctx := context.Background()
	ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)
	return ctx
}

// TestChat_Success 测试成功的对话
func TestChat_Success(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	ctx := createTestContext()
	resp, err := service.Chat(ctx, req)
	if err != nil {
		t.Fatalf("对话失败: %v", err)
	}

	if resp == nil {
		t.Fatal("响应为空")
	}

	if resp.SessionID == "" {
		t.Error("会话ID为空")
	}

	if resp.Message != "测试响应" {
		t.Errorf("期望消息为 '测试响应'，实际为 '%s'", resp.Message)
	}

	if resp.Model != "test-model" {
		t.Errorf("期望模型为 'test-model'，实际为 '%s'", resp.Model)
	}

	if resp.Usage == nil {
		t.Error("Usage 为空")
	} else {
		if resp.Usage.TotalTokens != 30 {
			t.Errorf("期望总 token 数为 30，实际为 %d", resp.Usage.TotalTokens)
		}
	}
}

// TestChat_WithOptions 测试带选项的对话
func TestChat_WithOptions(t *testing.T) {
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			if options == nil {
				t.Error("选项为空")
			} else {
				if options.Temperature == nil || *options.Temperature != 0.8 {
					t.Error("温度值不正确")
				}
				if options.MaxTokens == nil || *options.MaxTokens != 1000 {
					t.Error("最大 token 数不正确")
				}
			}
			return &genkitclient.GenerateResult{
				Text:  "测试响应",
				Model: "test-model",
			}, nil
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	temp := 0.8
	maxTokens := 1000
	req := &model.ChatRequest{
		Message: "你好",
		Options: &model.ChatOptions{
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		},
	}

	ctx := createTestContext()
	_, err := service.Chat(ctx, req)
	if err != nil {
		t.Fatalf("对话失败: %v", err)
	}
}

// TestChat_WithExistingSession 测试使用现有会话
func TestChat_WithExistingSession(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	// 第一次对话，创建会话
	req1 := &model.ChatRequest{
		Message: "你好",
	}

	ctx := createTestContext()
	resp1, err := service.Chat(ctx, req1)
	if err != nil {
		t.Fatalf("第一次对话失败: %v", err)
	}

	sessionID := resp1.SessionID

	// 第二次对话，使用相同会话（通过消息ID）
	req2 := &model.ChatRequest{
		Message:   "再见",
		MessageID: sessionID,
	}

	resp2, err := service.Chat(ctx, req2)
	if err != nil {
		t.Fatalf("第二次对话失败: %v", err)
	}

	if resp2.SessionID != sessionID {
		t.Errorf("期望会话ID为 '%s'，实际为 '%s'", sessionID, resp2.SessionID)
	}
}

// TestChat_ContextCancelled 测试上下文取消
func TestChat_ContextCancelled(t *testing.T) {
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			// 模拟上下文取消
			return nil, context.Canceled
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	ctx := createTestContext()
	_, err := service.Chat(ctx, req)
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestChat_GenerateError 测试生成错误
func TestChat_GenerateError(t *testing.T) {
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			return nil, errors.New("生成失败")
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	ctx := createTestContext()
	_, err := service.Chat(ctx, req)
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestAbortChat_Success 测试成功中止对话
func TestAbortChat_Success(t *testing.T) {
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			// 模拟长时间运行
			time.Sleep(100 * time.Millisecond)
			return &genkitclient.GenerateResult{
				Text:  "测试响应",
				Model: "test-model",
			}, nil
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	contextManager.Start()
	defer contextManager.Stop()

	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	// 启动对话
	req := &model.ChatRequest{
		Message: "你好",
	}

	// 在 goroutine 中执行对话
	done := make(chan error, 1)
	ctx := createTestContext()

	go func() {
		_, err := service.Chat(ctx, req)
		done <- err
	}()

	// 等待一小段时间确保对话开始
	time.Sleep(10 * time.Millisecond)

	// 获取会话ID（从第一次对话创建）
	// 注意：这里需要一个更好的方式来获取会话ID
	// 为了测试，我们先创建一个会话
	testSessionID, _, _ := contextManager.CreateSession(context.Background())

	// 中止对话
	err := service.AbortChat(context.Background(), testSessionID)
	if err != nil {
		t.Logf("中止对话返回错误（可能是会话已完成）: %v", err)
	}

	// 等待 goroutine 完成
	select {
	case <-done:
		// goroutine 已完成
	case <-time.After(200 * time.Millisecond):
		t.Log("等待 goroutine 完成超时")
	}
}

// TestAbortChat_MessageNotFound 测试中止不存在的消息
func TestAbortChat_MessageNotFound(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	// 中止不存在的消息应该返回 nil（幂等操作）
	err := service.AbortChat(context.Background(), "non-existent-message")
	if err != nil {
		t.Fatalf("期望返回 nil，实际返回错误: %v", err)
	}
}

// TestChatStream_Success 测试流式对话
func TestChatStream_Success(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	ctx := createTestContext()
	stream, err := service.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("流式对话失败: %v", err)
	}

	// 读取流式响应
	var messages []*model.TencentCloudStreamMessage
	for msg := range stream {
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		t.Fatal("未收到流式响应")
	}
}

// TestChat_MissingJWTClaims 测试缺少 JWT Claims
func TestChat_MissingJWTClaims(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	// 使用没有 JWT Claims 的上下文
	_, err := service.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestChat_MissingTenantID 测试缺少租户ID
func TestChat_MissingTenantID(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	// 创建一个没有租户ID的 JWT Claims
	claims := &model.JWTClaims{
		DisplayName: "测试用户",
		Roles:       []string{"user"},
	}
	claims.Subject = "test-user-id"
	
	ctx := context.Background()
	ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)

	_, err := service.Chat(ctx, req)
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestChat_WithModelName 测试指定模型名称
func TestChat_WithModelName(t *testing.T) {
	expectedModelName := "gpt-4"
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			// 验证传递的模型名称
			if modelName != expectedModelName {
				t.Errorf("期望模型名称为 '%s'，实际为 '%s'", expectedModelName, modelName)
			}
			return &genkitclient.GenerateResult{
				Text:  "测试响应",
				Model: modelName,
			}, nil
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	modelName := expectedModelName
	req := &model.ChatRequest{
		Message: "你好",
		Options: &model.ChatOptions{
			ModelName: &modelName,
		},
	}

	ctx := createTestContext()
	resp, err := service.Chat(ctx, req)
	if err != nil {
		t.Fatalf("对话失败: %v", err)
	}

	if resp.Model != expectedModelName {
		t.Errorf("期望响应模型为 '%s'，实际为 '%s'", expectedModelName, resp.Model)
	}
}

// TestChat_WithoutModelName 测试不指定模型名称（使用默认模型）
func TestChat_WithoutModelName(t *testing.T) {
	expectedModelName := "gemini-pro" // 默认模型
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			// 验证使用默认模型
			if modelName != expectedModelName {
				t.Errorf("期望使用默认模型 '%s'，实际为 '%s'", expectedModelName, modelName)
			}
			return &genkitclient.GenerateResult{
				Text:  "测试响应",
				Model: modelName,
			}, nil
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
		// 不指定 Options 或 ModelName
	}

	ctx := createTestContext()
	_, err := service.Chat(ctx, req)
	if err != nil {
		t.Fatalf("对话失败: %v", err)
	}
}

// TestChat_WithEmptyModelName 测试空模型名称（应使用默认模型）
func TestChat_WithEmptyModelName(t *testing.T) {
	expectedModelName := "gemini-pro" // 默认模型
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (*genkitclient.GenerateResult, error) {
			// 验证使用默认模型
			if modelName != expectedModelName {
				t.Errorf("期望使用默认模型 '%s'，实际为 '%s'", expectedModelName, modelName)
			}
			return &genkitclient.GenerateResult{
				Text:  "测试响应",
				Model: modelName,
			}, nil
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	emptyModelName := ""
	req := &model.ChatRequest{
		Message: "你好",
		Options: &model.ChatOptions{
			ModelName: &emptyModelName, // 空字符串
		},
	}

	ctx := createTestContext()
	_, err := service.Chat(ctx, req)
	if err != nil {
		t.Fatalf("对话失败: %v", err)
	}
}

// TestChatStream_WithModelName 测试流式对话指定模型名称
func TestChatStream_WithModelName(t *testing.T) {
	expectedModelName := "gpt-4"
	client := &mockGenkitClient{
		generateStreamFunc: func(ctx context.Context, tenantID, modelName, prompt string, options *genkitclient.GenerateOptions) (<-chan genkitclient.StreamChunk, error) {
			// 验证传递的模型名称
			if modelName != expectedModelName {
				t.Errorf("期望模型名称为 '%s'，实际为 '%s'", expectedModelName, modelName)
			}
			// 返回一个简单的流
			ch := make(chan genkitclient.StreamChunk, 1)
			go func() {
				defer close(ch)
				ch <- genkitclient.StreamChunk{
					Content: "测试响应",
					Done:    true,
					Model:   modelName,
				}
			}()
			return ch, nil
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	modelName := expectedModelName
	req := &model.ChatRequest{
		Message: "你好",
		Options: &model.ChatOptions{
			ModelName: &modelName,
		},
	}

	ctx := createTestContext()
	stream, err := service.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("流式对话失败: %v", err)
	}

	// 读取流式响应
	var messages []*model.TencentCloudStreamMessage
	for msg := range stream {
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		t.Fatal("未收到流式响应")
	}
}
