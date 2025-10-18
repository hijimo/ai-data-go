package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

// mockSessionService 模拟会话服务
type mockSessionService struct {
	createSessionFunc  func(ctx context.Context, userID string, req *model.CreateSessionRequest) (*model.SessionResponse, error)
	getSessionFunc     func(ctx context.Context, sessionID, userID string) (*model.SessionResponse, error)
	listSessionsFunc   func(ctx context.Context, userID string, req *model.ListSessionsRequest) ([]*model.SessionResponse, int, error)
	updateSessionFunc  func(ctx context.Context, sessionID, userID string, req *model.UpdateSessionRequest) (*model.SessionResponse, error)
	deleteSessionFunc  func(ctx context.Context, sessionID, userID string) error
	searchSessionsFunc func(ctx context.Context, userID string, req *model.SearchSessionsRequest) ([]*model.SessionResponse, int, error)
	pinSessionFunc     func(ctx context.Context, sessionID, userID string, pinned bool) error
	archiveSessionFunc func(ctx context.Context, sessionID, userID string, archived bool) error
}

func (m *mockSessionService) CreateSession(ctx context.Context, userID string, req *model.CreateSessionRequest) (*model.SessionResponse, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *mockSessionService) GetSession(ctx context.Context, sessionID, userID string) (*model.SessionResponse, error) {
	if m.getSessionFunc != nil {
		return m.getSessionFunc(ctx, sessionID, userID)
	}
	return nil, nil
}

func (m *mockSessionService) ListSessions(ctx context.Context, userID string, req *model.ListSessionsRequest) ([]*model.SessionResponse, int, error) {
	if m.listSessionsFunc != nil {
		return m.listSessionsFunc(ctx, userID, req)
	}
	return nil, 0, nil
}

func (m *mockSessionService) UpdateSession(ctx context.Context, sessionID, userID string, req *model.UpdateSessionRequest) (*model.SessionResponse, error) {
	if m.updateSessionFunc != nil {
		return m.updateSessionFunc(ctx, sessionID, userID, req)
	}
	return nil, nil
}

func (m *mockSessionService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	if m.deleteSessionFunc != nil {
		return m.deleteSessionFunc(ctx, sessionID, userID)
	}
	return nil
}

func (m *mockSessionService) SearchSessions(ctx context.Context, userID string, req *model.SearchSessionsRequest) ([]*model.SessionResponse, int, error) {
	if m.searchSessionsFunc != nil {
		return m.searchSessionsFunc(ctx, userID, req)
	}
	return nil, 0, nil
}

func (m *mockSessionService) PinSession(ctx context.Context, sessionID, userID string, pinned bool) error {
	if m.pinSessionFunc != nil {
		return m.pinSessionFunc(ctx, sessionID, userID, pinned)
	}
	return nil
}

func (m *mockSessionService) ArchiveSession(ctx context.Context, sessionID, userID string, archived bool) error {
	if m.archiveSessionFunc != nil {
		return m.archiveSessionFunc(ctx, sessionID, userID, archived)
	}
	return nil
}

// TestCreateSession 测试创建会话
func TestCreateSession(t *testing.T) {
	// 创建模拟服务
	mockService := &mockSessionService{
		createSessionFunc: func(ctx context.Context, userID string, req *model.CreateSessionRequest) (*model.SessionResponse, error) {
			return &model.SessionResponse{
				ID:        "test-session-id",
				UserID:    userID,
				Title:     req.Title,
				ModelName: req.ModelName,
			}, nil
		},
	}

	// 创建处理器
	handler := NewSessionHandler(mockService, logger.Default())

	// 准备请求
	reqBody := model.CreateSessionRequest{
		Title:     "测试会话",
		ModelName: "gpt-4",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/chat/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "test-user")

	// 执行请求
	w := httptest.NewRecorder()
	handler.CreateSession(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际 %d", http.StatusOK, w.Code)
	}

	var resp model.ResponseData[model.SessionResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望响应码 %d, 实际 %d", errors.CodeSuccess, resp.Code)
	}

	if resp.Data.Title != "测试会话" {
		t.Errorf("期望标题 '测试会话', 实际 '%s'", resp.Data.Title)
	}
}

// TestGetSession 测试获取会话详情
func TestGetSession(t *testing.T) {
	// 创建模拟服务
	mockService := &mockSessionService{
		getSessionFunc: func(ctx context.Context, sessionID, userID string) (*model.SessionResponse, error) {
			if sessionID == "not-found" {
				return nil, errors.NewSessionNotFoundError(sessionID)
			}
			return &model.SessionResponse{
				ID:        sessionID,
				UserID:    userID,
				Title:     "测试会话",
				ModelName: "gpt-4",
			}, nil
		},
	}

	// 创建处理器
	handler := NewSessionHandler(mockService, logger.Default())

	// 测试成功获取
	t.Run("成功获取会话", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions/test-id", nil)
		req.Header.Set("X-User-ID", "test-user")

		w := httptest.NewRecorder()
		handler.GetSession(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 %d, 实际 %d", http.StatusOK, w.Code)
		}
	})

	// 测试会话不存在
	t.Run("会话不存在", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions/not-found", nil)
		req.Header.Set("X-User-ID", "test-user")

		w := httptest.NewRecorder()
		handler.GetSession(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("期望状态码 %d, 实际 %d", http.StatusNotFound, w.Code)
		}
	})
}

// TestListSessions 测试获取会话列表
func TestListSessions(t *testing.T) {
	// 创建模拟服务
	mockService := &mockSessionService{
		listSessionsFunc: func(ctx context.Context, userID string, req *model.ListSessionsRequest) ([]*model.SessionResponse, int, error) {
			sessions := []*model.SessionResponse{
				{
					ID:        "session-1",
					UserID:    userID,
					Title:     "会话1",
					ModelName: "gpt-4",
				},
				{
					ID:        "session-2",
					UserID:    userID,
					Title:     "会话2",
					ModelName: "gpt-4",
				},
			}
			return sessions, 2, nil
		},
	}

	// 创建处理器
	handler := NewSessionHandler(mockService, logger.Default())

	// 准备请求
	req := httptest.NewRequest(http.MethodGet, "/chat/sessions?pageNo=1&pageSize=20", nil)
	req.Header.Set("X-User-ID", "test-user")

	// 执行请求
	w := httptest.NewRecorder()
	handler.ListSessions(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际 %d", http.StatusOK, w.Code)
	}

	var resp model.ResponsePaginationData[[]*model.SessionResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望响应码 %d, 实际 %d", errors.CodeSuccess, resp.Code)
	}

	if len(resp.Data.Data) != 2 {
		t.Errorf("期望返回 2 个会话, 实际 %d", len(resp.Data.Data))
	}
}

// TestDeleteSession 测试删除会话
func TestDeleteSession(t *testing.T) {
	// 创建模拟服务
	mockService := &mockSessionService{
		deleteSessionFunc: func(ctx context.Context, sessionID, userID string) error {
			if sessionID == "not-found" {
				return errors.NewSessionNotFoundError(sessionID)
			}
			return nil
		},
	}

	// 创建处理器
	handler := NewSessionHandler(mockService, logger.Default())

	// 测试成功删除
	t.Run("成功删除会话", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chat/sessions/test-id", nil)
		req.Header.Set("X-User-ID", "test-user")

		w := httptest.NewRecorder()
		handler.DeleteSession(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 %d, 实际 %d", http.StatusOK, w.Code)
		}
	})

	// 测试会话不存在
	t.Run("会话不存在", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chat/sessions/not-found", nil)
		req.Header.Set("X-User-ID", "test-user")

		w := httptest.NewRecorder()
		handler.DeleteSession(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("期望状态码 %d, 实际 %d", http.StatusNotFound, w.Code)
		}
	})
}

// TestPinSession 测试置顶会话
func TestPinSession(t *testing.T) {
	// 创建模拟服务
	mockService := &mockSessionService{
		pinSessionFunc: func(ctx context.Context, sessionID, userID string, pinned bool) error {
			return nil
		},
	}

	// 创建处理器
	handler := NewSessionHandler(mockService, logger.Default())

	// 准备请求
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test-id/pin?pinned=true", nil)
	req.Header.Set("X-User-ID", "test-user")

	// 执行请求
	w := httptest.NewRecorder()
	handler.PinSession(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际 %d", http.StatusOK, w.Code)
	}
}

// TestArchiveSession 测试归档会话
func TestArchiveSession(t *testing.T) {
	// 创建模拟服务
	mockService := &mockSessionService{
		archiveSessionFunc: func(ctx context.Context, sessionID, userID string, archived bool) error {
			return nil
		},
	}

	// 创建处理器
	handler := NewSessionHandler(mockService, logger.Default())

	// 准备请求
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test-id/archive?archived=true", nil)
	req.Header.Set("X-User-ID", "test-user")

	// 执行请求
	w := httptest.NewRecorder()
	handler.ArchiveSession(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际 %d", http.StatusOK, w.Code)
	}
}
