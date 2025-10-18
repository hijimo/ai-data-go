package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

// mockAIService 模拟 AI 服务
type mockAIService struct {
	chatFunc       func(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error)
	chatStreamFunc func(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error)
	abortChatFunc  func(ctx context.Context, messageID string) error
}

func (m *mockAIService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAIService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error) {
	if m.chatStreamFunc != nil {
		return m.chatStreamFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAIService) AbortChat(ctx context.Context, messageID string) error {
	if m.abortChatFunc != nil {
		return m.abortChatFunc(ctx, messageID)
	}
	return nil
}

// TestHandleChat_Success 测试成功的对话请求
func TestHandleChat_Success(t *testing.T) {
	// 创建模拟服务
	mockService := &mockAIService{
		chatFunc: func(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
			return &model.ChatResponse{
				SessionID: "test-session-123",
				Message:   "这是 AI 的回复",
				Model:     "gemini-2.5-flash",
				Usage: &model.Usage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			}, nil
		},
	}

	// 创建处理器
	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)
	handler := NewChatHandler(mockService, log)

	// 创建请求
	reqBody := model.ChatRequest{
		Message: "你好",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用处理器
	handler.HandleChat(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp model.ResponseData[model.ChatResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望响应码 %d, 得到 %d", errors.CodeSuccess, resp.Code)
	}

	if resp.Data == nil {
		t.Fatal("响应数据为空")
	}

	if resp.Data.SessionID != "test-session-123" {
		t.Errorf("期望 sessionId 为 test-session-123, 得到 %s", resp.Data.SessionID)
	}
}

// TestHandleChat_ValidationError 测试参数验证失败
func TestHandleChat_ValidationError(t *testing.T) {
	mockService := &mockAIService{}
	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)
	handler := NewChatHandler(mockService, log)

	// 创建无效请求（缺少必填字段 message）
	reqBody := model.ChatRequest{
		Message: "", // 空消息
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChat(w, req)

	// 验证响应
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp model.ResponseData[map[string]interface{}]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeValidationError {
		t.Errorf("期望响应码 %d, 得到 %d", errors.CodeValidationError, resp.Code)
	}
}

// TestHandleChat_AIServiceError 测试 AI 服务错误
func TestHandleChat_AIServiceError(t *testing.T) {
	mockService := &mockAIService{
		chatFunc: func(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
			return nil, errors.NewAIServiceError(nil)
		},
	}

	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)
	handler := NewChatHandler(mockService, log)

	reqBody := model.ChatRequest{
		Message: "你好",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChat(w, req)

	// 验证响应
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusInternalServerError, w.Code)
	}

	var resp model.ResponseData[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeAIServiceError {
		t.Errorf("期望响应码 %d, 得到 %d", errors.CodeAIServiceError, resp.Code)
	}
}

// TestHandleChat_WithOptions 测试带高级参数的请求
func TestHandleChat_WithOptions(t *testing.T) {
	mockService := &mockAIService{
		chatFunc: func(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
			// 验证参数是否正确传递
			if req.Options == nil {
				t.Error("期望收到 options 参数")
			}
			return &model.ChatResponse{
				SessionID: "test-session-456",
				Message:   "带参数的回复",
				Model:     "gemini-2.5-flash",
			}, nil
		},
	}

	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)
	handler := NewChatHandler(mockService, log)

	temp := 0.7
	maxTokens := 1000
	reqBody := model.ChatRequest{
		Message: "你好",
		Options: &model.ChatOptions{
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChat(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}
}
