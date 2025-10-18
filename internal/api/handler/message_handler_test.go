package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/session"
	pkgErrors "genkit-ai-service/pkg/errors"
)

// mockMessageService 模拟消息服务
type mockMessageService struct {
	sendMessageFunc      func(ctx context.Context, req *session.SendMessageRequest) (*session.MessageResponse, error)
	getMessagesFunc      func(ctx context.Context, req *session.GetMessagesRequest) (*session.MessageListResponse, error)
	getMessageByIDFunc   func(ctx context.Context, messageID, userID string) (*session.MessageDetailResponse, error)
	abortMessageFunc     func(ctx context.Context, messageID, userID string) error
}

func (m *mockMessageService) SendMessage(ctx context.Context, req *session.SendMessageRequest) (*session.MessageResponse, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, req)
	}
	return nil, errors.New("未实现")
}

func (m *mockMessageService) GetMessages(ctx context.Context, req *session.GetMessagesRequest) (*session.MessageListResponse, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, req)
	}
	return nil, errors.New("未实现")
}

func (m *mockMessageService) GetMessageByID(ctx context.Context, messageID, userID string) (*session.MessageDetailResponse, error) {
	if m.getMessageByIDFunc != nil {
		return m.getMessageByIDFunc(ctx, messageID, userID)
	}
	return nil, errors.New("未实现")
}

func (m *mockMessageService) AbortMessage(ctx context.Context, messageID, userID string) error {
	if m.abortMessageFunc != nil {
		return m.abortMessageFunc(ctx, messageID, userID)
	}
	return errors.New("未实现")
}

// TestSendMessage 测试发送消息
func TestSendMessage(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		requestBody    interface{}
		mockService    *mockMessageService
		expectedStatus int
		expectedCode   int
	}{
		{
			name:      "成功发送消息",
			sessionID: "550e8400-e29b-41d4-a716-446655440000",
			requestBody: model.SendMessageRequest{
				Message: "你好",
			},
			mockService: &mockMessageService{
				sendMessageFunc: func(ctx context.Context, req *session.SendMessageRequest) (*session.MessageResponse, error) {
					return &session.MessageResponse{
						MessageID: "550e8400-e29b-41d4-a716-446655440001",
						SessionID: "550e8400-e29b-41d4-a716-446655440000",
						UserMessage: &session.Message{
							ID:        "user-msg-id",
							Role:      "user",
							Content:   "你好",
							Sequence:  1,
							CreatedAt: time.Now(),
						},
						AIMessage: &session.Message{
							ID:        "ai-msg-id",
							Role:      "assistant",
							Content:   "你好！有什么可以帮助你的吗？",
							Sequence:  2,
							CreatedAt: time.Now(),
						},
						Model: "gpt-4",
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   pkgErrors.CodeSuccess,
		},
		{
			name:      "会话不存在",
			sessionID: "550e8400-e29b-41d4-a716-446655440002",
			requestBody: model.SendMessageRequest{
				Message: "你好",
			},
			mockService: &mockMessageService{
				sendMessageFunc: func(ctx context.Context, req *session.SendMessageRequest) (*session.MessageResponse, error) {
					return nil, pkgErrors.NewSessionNotFoundError("550e8400-e29b-41d4-a716-446655440002")
				},
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   pkgErrors.CodeSessionNotFound,
		},
		{
			name:      "无权访问会话",
			sessionID: "550e8400-e29b-41d4-a716-446655440003",
			requestBody: model.SendMessageRequest{
				Message: "你好",
			},
			mockService: &mockMessageService{
				sendMessageFunc: func(ctx context.Context, req *session.SendMessageRequest) (*session.MessageResponse, error) {
					return nil, pkgErrors.NewSessionAccessDeniedError()
				},
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   pkgErrors.CodeSessionAccessDenied,
		},
		{
			name:           "请求参数为空",
			sessionID:      "550e8400-e29b-41d4-a716-446655440004",
			requestBody:    model.SendMessageRequest{},
			mockService:    &mockMessageService{},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   pkgErrors.CodeValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建处理器
			handler := NewMessageHandler(tt.mockService, logger.NewTestLogger())

			// 创建请求
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/"+tt.sessionID+"/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", "test-user-id")

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 调用处理器
			handler.SendMessage(w, req)

			// 验证状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 实际 %d", tt.expectedStatus, w.Code)
			}

			// 验证响应码
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); ok {
				if int(code) != tt.expectedCode {
					t.Errorf("期望响应码 %d, 实际 %d", tt.expectedCode, int(code))
				}
			}
		})
	}
}

// TestGetMessages 测试获取消息历史
func TestGetMessages(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		queryParams    string
		mockService    *mockMessageService
		expectedStatus int
		expectedCode   int
	}{
		{
			name:        "成功获取消息历史",
			sessionID:   "550e8400-e29b-41d4-a716-446655440000",
			queryParams: "?pageNo=1&pageSize=50",
			mockService: &mockMessageService{
				getMessagesFunc: func(ctx context.Context, req *session.GetMessagesRequest) (*session.MessageListResponse, error) {
					return &session.MessageListResponse{
						Messages: []*session.MessageDetailResponse{
							{
								ID:        "550e8400-e29b-41d4-a716-446655440001",
								SessionID: "550e8400-e29b-41d4-a716-446655440000",
								Role:      "user",
								Content:   "你好",
								Sequence:  1,
								CreatedAt: time.Now(),
							},
							{
								ID:        "550e8400-e29b-41d4-a716-446655440002",
								SessionID: "550e8400-e29b-41d4-a716-446655440000",
								Role:      "assistant",
								Content:   "你好！",
								Sequence:  2,
								CreatedAt: time.Now(),
							},
						},
						PageNo:     1,
						PageSize:   50,
						TotalCount: 2,
						TotalPage:  1,
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   pkgErrors.CodeSuccess,
		},
		{
			name:        "会话不存在",
			sessionID:   "550e8400-e29b-41d4-a716-446655440010",
			queryParams: "?pageNo=1&pageSize=50",
			mockService: &mockMessageService{
				getMessagesFunc: func(ctx context.Context, req *session.GetMessagesRequest) (*session.MessageListResponse, error) {
					return nil, pkgErrors.NewSessionNotFoundError("550e8400-e29b-41d4-a716-446655440010")
				},
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   pkgErrors.CodeSessionNotFound,
		},
		{
			name:        "无权访问会话",
			sessionID:   "550e8400-e29b-41d4-a716-446655440011",
			queryParams: "?pageNo=1&pageSize=50",
			mockService: &mockMessageService{
				getMessagesFunc: func(ctx context.Context, req *session.GetMessagesRequest) (*session.MessageListResponse, error) {
					return nil, pkgErrors.NewSessionAccessDeniedError()
				},
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   pkgErrors.CodeSessionAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建处理器
			handler := NewMessageHandler(tt.mockService, logger.NewTestLogger())

			// 创建请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions/"+tt.sessionID+"/messages"+tt.queryParams, nil)
			req.Header.Set("X-User-ID", "test-user-id")

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 调用处理器
			handler.GetMessages(w, req)

			// 验证状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 实际 %d", tt.expectedStatus, w.Code)
			}

			// 验证响应码
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); ok {
				if int(code) != tt.expectedCode {
					t.Errorf("期望响应码 %d, 实际 %d", tt.expectedCode, int(code))
				}
			}
		})
	}
}

// TestGetMessageByID 测试获取单条消息详情
func TestGetMessageByID(t *testing.T) {
	tests := []struct {
		name           string
		messageID      string
		mockService    *mockMessageService
		expectedStatus int
		expectedCode   int
	}{
		{
			name:      "成功获取消息详情",
			messageID: "550e8400-e29b-41d4-a716-446655440020",
			mockService: &mockMessageService{
				getMessageByIDFunc: func(ctx context.Context, messageID, userID string) (*session.MessageDetailResponse, error) {
					return &session.MessageDetailResponse{
						ID:        "550e8400-e29b-41d4-a716-446655440020",
						SessionID: "550e8400-e29b-41d4-a716-446655440000",
						Role:      "user",
						Content:   "你好",
						Sequence:  1,
						CreatedAt: time.Now(),
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   pkgErrors.CodeSuccess,
		},
		{
			name:      "消息不存在",
			messageID: "550e8400-e29b-41d4-a716-446655440021",
			mockService: &mockMessageService{
				getMessageByIDFunc: func(ctx context.Context, messageID, userID string) (*session.MessageDetailResponse, error) {
					return nil, pkgErrors.NewMessageNotFoundError("550e8400-e29b-41d4-a716-446655440021")
				},
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   pkgErrors.CodeMessageNotFound,
		},
		{
			name:      "无权访问消息",
			messageID: "550e8400-e29b-41d4-a716-446655440022",
			mockService: &mockMessageService{
				getMessageByIDFunc: func(ctx context.Context, messageID, userID string) (*session.MessageDetailResponse, error) {
					return nil, pkgErrors.NewMessageAccessDeniedError()
				},
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   pkgErrors.CodeMessageAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建处理器
			handler := NewMessageHandler(tt.mockService, logger.NewTestLogger())

			// 创建请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/messages/"+tt.messageID, nil)
			req.Header.Set("X-User-ID", "test-user-id")

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 调用处理器
			handler.GetMessageByID(w, req)

			// 验证状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 实际 %d", tt.expectedStatus, w.Code)
			}

			// 验证响应码
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); ok {
				if int(code) != tt.expectedCode {
					t.Errorf("期望响应码 %d, 实际 %d", tt.expectedCode, int(code))
				}
			}
		})
	}
}

// TestAbortMessage 测试中止消息生成
func TestAbortMessage(t *testing.T) {
	tests := []struct {
		name           string
		messageID      string
		mockService    *mockMessageService
		expectedStatus int
		expectedCode   int
	}{
		{
			name:      "成功中止消息生成",
			messageID: "550e8400-e29b-41d4-a716-446655440030",
			mockService: &mockMessageService{
				abortMessageFunc: func(ctx context.Context, messageID, userID string) error {
					return nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   pkgErrors.CodeSuccess,
		},
		{
			name:      "消息不存在（幂等操作）",
			messageID: "550e8400-e29b-41d4-a716-446655440031",
			mockService: &mockMessageService{
				abortMessageFunc: func(ctx context.Context, messageID, userID string) error {
					return nil // 幂等操作，返回成功
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   pkgErrors.CodeSuccess,
		},
		{
			name:      "无权访问消息",
			messageID: "550e8400-e29b-41d4-a716-446655440032",
			mockService: &mockMessageService{
				abortMessageFunc: func(ctx context.Context, messageID, userID string) error {
					return pkgErrors.NewMessageAccessDeniedError()
				},
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   pkgErrors.CodeMessageAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建处理器
			handler := NewMessageHandler(tt.mockService, logger.NewTestLogger())

			// 创建请求
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/messages/"+tt.messageID+"/abort", nil)
			req.Header.Set("X-User-ID", "test-user-id")

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 调用处理器
			handler.AbortMessage(w, req)

			// 验证状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 实际 %d", tt.expectedStatus, w.Code)
			}

			// 验证响应码
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); ok {
				if int(code) != tt.expectedCode {
					t.Errorf("期望响应码 %d, 实际 %d", tt.expectedCode, int(code))
				}
			}
		})
	}
}
