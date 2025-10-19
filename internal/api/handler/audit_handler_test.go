package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
)

// mockAuditRepository 模拟审计日志仓储
type mockAuditRepository struct {
	audits []*model.AuthAudit
	total  int64
	err    error
}

func (m *mockAuditRepository) Create(ctx context.Context, audit *model.AuthAudit) error {
	return nil
}

func (m *mockAuditRepository) List(ctx context.Context, filter repository.AuditFilter, page, pageSize int) ([]*model.AuthAudit, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.audits, m.total, nil
}

func TestAuditHandler_HandleListAuditLogs(t *testing.T) {
	// 创建测试日志记录器
	log := logger.NewTestLogger()

	// 创建测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	now := time.Now()

	testAudits := []*model.AuthAudit{
		{
			ID:        uuid.New().String(),
			TenantID:  &tenantID,
			UserID:    &userID,
			Event:     "login",
			IP:        "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			CreatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			TenantID:  &tenantID,
			UserID:    &userID,
			Event:     "logout",
			IP:        "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			CreatedAt: now.Add(-1 * time.Hour),
		},
	}

	tests := []struct {
		name           string
		queryParams    string
		mockAudits     []*model.AuthAudit
		mockTotal      int64
		mockErr        error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "成功查询所有审计日志",
			queryParams:    "?page=1&pageSize=10",
			mockAudits:     testAudits,
			mockTotal:      2,
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "按租户ID过滤",
			queryParams:    "?tenantId=" + tenantID + "&page=1&pageSize=10",
			mockAudits:     testAudits,
			mockTotal:      2,
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "按事件类型过滤",
			queryParams:    "?event=login&page=1&pageSize=10",
			mockAudits:     testAudits[:1],
			mockTotal:      1,
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "无效的租户ID",
			queryParams:    "?tenantId=invalid-uuid",
			mockAudits:     nil,
			mockTotal:      0,
			mockErr:        nil,
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "无效的时间格式",
			queryParams:    "?startTime=invalid-time",
			mockAudits:     nil,
			mockTotal:      0,
			mockErr:        nil,
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟仓储
			mockRepo := &mockAuditRepository{
				audits: tt.mockAudits,
				total:  tt.mockTotal,
				err:    tt.mockErr,
			}

			// 创建处理器
			handler := NewAuditHandler(mockRepo, log)

			// 创建测试请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/auth"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// 执行请求
			handler.HandleListAuditLogs(w, req)

			// 验证响应状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 实际得到 %d", tt.expectedStatus, w.Code)
			}

			// 如果期望成功，验证响应内容
			if tt.expectedStatus == http.StatusOK {
				var resp model.ResponsePaginationData[[]*model.AuthAudit]
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("解析响应失败: %v", err)
				}

				if resp.Code != 200 {
					t.Errorf("期望响应码 200, 实际得到 %d", resp.Code)
				}

				if len(resp.Data.Data) != tt.expectedCount {
					t.Errorf("期望返回 %d 条记录, 实际得到 %d 条", tt.expectedCount, len(resp.Data.Data))
				}

				if resp.Data.TotalCount != int(tt.mockTotal) {
					t.Errorf("期望总数 %d, 实际得到 %d", tt.mockTotal, resp.Data.TotalCount)
				}
			}
		})
	}
}

func TestAuditHandler_HandleListAuditLogs_WithTimeFilter(t *testing.T) {
	// 创建测试日志记录器
	log := logger.NewTestLogger()

	// 创建测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	now := time.Now()

	testAudits := []*model.AuthAudit{
		{
			ID:        uuid.New().String(),
			TenantID:  &tenantID,
			UserID:    &userID,
			Event:     "login",
			IP:        "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			CreatedAt: now,
		},
	}

	// 创建模拟仓储
	mockRepo := &mockAuditRepository{
		audits: testAudits,
		total:  1,
		err:    nil,
	}

	// 创建处理器
	handler := NewAuditHandler(mockRepo, log)

	// 创建带时间过滤的请求
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Add(1 * time.Hour).Format(time.RFC3339)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/auth", nil)
	q := req.URL.Query()
	q.Add("startTime", startTime)
	q.Add("endTime", endTime)
	q.Add("page", "1")
	q.Add("pageSize", "10")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	// 执行请求
	handler.HandleListAuditLogs(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际得到 %d", http.StatusOK, w.Code)
	}

	var resp model.ResponsePaginationData[[]*model.AuthAudit]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != 200 {
		t.Errorf("期望响应码 200, 实际得到 %d", resp.Code)
	}

	if len(resp.Data.Data) != 1 {
		t.Errorf("期望返回 1 条记录, 实际得到 %d 条", len(resp.Data.Data))
	}
}
