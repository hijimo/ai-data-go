package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

// mockAIServiceForAbort 用于测试的 mock AI 服务
type mockAIServiceForAbort struct {
	abortChatFunc func(ctx context.Context, sessionID string) error
}

func (m *mockAIServiceForAbort) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	return nil, nil
}

func (m *mockAIServiceForAbort) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error) {
	return nil, nil
}

func (m *mockAIServiceForAbort) AbortChat(ctx context.Context, sessionID string) error {
	if m.abortChatFunc != nil {
		return m.abortChatFunc(ctx, sessionID)
	}
	return nil
}

func TestAbortHandler_HandleAbort_Success(t *testing.T) {
	// 创建 mock 服务
	mockService := &mockAIServiceForAbort{
		abortChatFunc: func(ctx context.Context, sessionID string) error {
			if sessionID == "test-session-123" {
				return nil
			}
			return errors.NewNotFoundError("会话不存在")
		},
	}

	// 创建处理器
	log := logger.New(logger.ErrorLevel, logger.JSONFormat, io.Discard)
	handler := NewAbortHandler(mockService, log)

	// 创建请求
	reqBody := model.AbortRequest{
		SessionID: "test-session-123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/abort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用处理器
	handler.HandleAbort(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际得到 %d", http.StatusOK, w.Code)
	}

	var resp model.ResponseData[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望响应码 %d, 实际得到 %d", errors.CodeSuccess, resp.Code)
	}

	if resp.Message != "对话已成功中止" {
		t.Errorf("期望消息 '对话已成功中止', 实际得到 '%s'", resp.Message)
	}
}

func TestAbortHandler_HandleAbort_SessionNotFound(t *testing.T) {
	// 创建 mock 服务
	mockService := &mockAIServiceForAbort{
		abortChatFunc: func(ctx context.Context, sessionID string) error {
			return errors.NewNotFoundError("会话不存在或已完成")
		},
	}

	// 创建处理器
	log := logger.New(logger.ErrorLevel, logger.JSONFormat, io.Discard)
	handler := NewAbortHandler(mockService, log)

	// 创建请求
	reqBody := model.AbortRequest{
		SessionID: "non-existent-session",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/abort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用处理器
	handler.HandleAbort(w, req)

	// 验证响应
	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 %d, 实际得到 %d", http.StatusNotFound, w.Code)
	}

	var resp model.ResponseData[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeNotFound {
		t.Errorf("期望响应码 %d, 实际得到 %d", errors.CodeNotFound, resp.Code)
	}
}

func TestAbortHandler_HandleAbort_MissingSessionID(t *testing.T) {
	// 创建 mock 服务
	mockService := &mockAIServiceForAbort{}

	// 创建处理器
	log := logger.New(logger.ErrorLevel, logger.JSONFormat, io.Discard)
	handler := NewAbortHandler(mockService, log)

	// 创建请求（缺少 sessionId）
	reqBody := model.AbortRequest{
		SessionID: "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/abort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用处理器
	handler.HandleAbort(w, req)

	// 验证响应
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 %d, 实际得到 %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp model.ResponseData[map[string]interface{}]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeValidationError {
		t.Errorf("期望响应码 %d, 实际得到 %d", errors.CodeValidationError, resp.Code)
	}
}

func TestAbortHandler_HandleAbort_InvalidJSON(t *testing.T) {
	// 创建 mock 服务
	mockService := &mockAIServiceForAbort{}

	// 创建处理器
	log := logger.New(logger.ErrorLevel, logger.JSONFormat, io.Discard)
	handler := NewAbortHandler(mockService, log)

	// 创建无效的 JSON 请求
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/abort", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用处理器
	handler.HandleAbort(w, req)

	// 验证响应
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 %d, 实际得到 %d", http.StatusBadRequest, w.Code)
	}

	var resp model.ResponseData[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeBadRequest {
		t.Errorf("期望响应码 %d, 实际得到 %d", errors.CodeBadRequest, resp.Code)
	}
}
