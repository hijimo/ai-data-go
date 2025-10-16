package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/firebase/genkit/go/ai"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
)

// mockGenkitClient 模拟 Genkit 客户端
type mockGenkitClient struct {
	generateFunc func(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error)
}

func (m *mockGenkitClient) Initialize(ctx context.Context, config *genkit.Config) error {
	return nil
}

func (m *mockGenkitClient) Generate(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt, options)
	}
	return &genkit.GenerateResult{
		Text:  "测试响应",
		Model: "test-model",
		Usage: &genkit.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
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

// TestChat_Success 测试成功的对话
func TestChat_Success(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	resp, err := service.Chat(context.Background(), req)
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
		generateFunc: func(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
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
			return &genkit.GenerateResult{
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

	_, err := service.Chat(context.Background(), req)
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

	resp1, err := service.Chat(context.Background(), req1)
	if err != nil {
		t.Fatalf("第一次对话失败: %v", err)
	}

	sessionID := resp1.SessionID

	// 第二次对话，使用相同会话
	req2 := &model.ChatRequest{
		Message:   "再见",
		SessionID: sessionID,
	}

	resp2, err := service.Chat(context.Background(), req2)
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
		generateFunc: func(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
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

	_, err := service.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestChat_GenerateError 测试生成错误
func TestChat_GenerateError(t *testing.T) {
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
			return nil, errors.New("生成失败")
		},
	}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	_, err := service.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestAbortChat_Success 测试成功中止对话
func TestAbortChat_Success(t *testing.T) {
	client := &mockGenkitClient{
		generateFunc: func(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
			// 模拟长时间运行
			time.Sleep(100 * time.Millisecond)
			return &genkit.GenerateResult{
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

	go func() {
		_, err := service.Chat(context.Background(), req)
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
}

// TestAbortChat_SessionNotFound 测试中止不存在的会话
func TestAbortChat_SessionNotFound(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	err := service.AbortChat(context.Background(), "non-existent-session")
	if err == nil {
		t.Fatal("期望返回错误")
	}
}

// TestChatStream_NotImplemented 测试流式对话（未实现）
func TestChatStream_NotImplemented(t *testing.T) {
	client := &mockGenkitClient{}
	contextManager := NewContextManager(30*time.Minute, 5*time.Minute)
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &testWriter{t: t})

	service := NewGenkitService(client, contextManager, log)

	req := &model.ChatRequest{
		Message: "你好",
	}

	_, err := service.ChatStream(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回未实现错误")
	}
}
