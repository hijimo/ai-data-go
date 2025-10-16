package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/health"
)

// mockHealthService 模拟健康检查服务
type mockHealthService struct {
	checkResult *health.HealthStatus
	checkErr    error
}

func (m *mockHealthService) Check(ctx context.Context) (*health.HealthStatus, error) {
	if m.checkErr != nil {
		return nil, m.checkErr
	}
	return m.checkResult, nil
}

func TestNewHealthHandler(t *testing.T) {
	mockService := &mockHealthService{}
	var buf bytes.Buffer
	mockLogger := logger.New(logger.InfoLevel, logger.JSONFormat, &buf)

	handler := NewHealthHandler(mockService, mockLogger)

	if handler == nil {
		t.Fatal("期望创建处理器成功，但得到 nil")
	}

	if handler.healthService == nil {
		t.Error("期望健康检查服务不为 nil")
	}

	if handler.logger == nil {
		t.Error("期望日志记录器不为 nil")
	}
}

func TestHealthHandler_Handle_Success(t *testing.T) {
	mockService := &mockHealthService{
		checkResult: &health.HealthStatus{
			Status:  "healthy",
			Version: "1.0.0",
			Uptime:  "1h30m",
			Dependencies: map[string]string{
				"genkit":   "connected",
				"database": "connected",
			},
		},
	}
	var buf bytes.Buffer
	mockLogger := logger.New(logger.InfoLevel, logger.JSONFormat, &buf)

	handler := NewHealthHandler(mockService, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	// 检查状态码
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d，但得到 %d", http.StatusOK, w.Code)
	}

	// 检查响应头
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type 为 'application/json'，但得到 '%s'", contentType)
	}

	// 解析响应
	var resp model.ResponseData[health.HealthStatus]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应
	if resp.Code != 200 {
		t.Errorf("期望响应码为 200，但得到 %d", resp.Code)
	}

	if resp.Data == nil {
		t.Fatal("期望响应数据不为 nil")
	}

	if resp.Data.Status != "healthy" {
		t.Errorf("期望状态为 'healthy'，但得到 '%s'", resp.Data.Status)
	}

	if resp.Data.Version != "1.0.0" {
		t.Errorf("期望版本为 '1.0.0'，但得到 '%s'", resp.Data.Version)
	}
}

func TestHealthHandler_Handle_Unhealthy(t *testing.T) {
	mockService := &mockHealthService{
		checkResult: &health.HealthStatus{
			Status:  "unhealthy",
			Version: "1.0.0",
			Uptime:  "1h30m",
			Dependencies: map[string]string{
				"genkit":   "disconnected",
				"database": "connected",
			},
		},
	}
	var buf bytes.Buffer
	mockLogger := logger.New(logger.InfoLevel, logger.JSONFormat, &buf)

	handler := NewHealthHandler(mockService, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	// 检查状态码应该是 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望状态码为 %d，但得到 %d", http.StatusServiceUnavailable, w.Code)
	}

	// 解析响应
	var resp model.ResponseData[health.HealthStatus]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应
	if resp.Code != 200 {
		t.Errorf("期望响应码为 200，但得到 %d", resp.Code)
	}

	if resp.Data == nil {
		t.Fatal("期望响应数据不为 nil")
	}

	if resp.Data.Status != "unhealthy" {
		t.Errorf("期望状态为 'unhealthy'，但得到 '%s'", resp.Data.Status)
	}
}

func TestHealthHandler_Handle_ServiceError(t *testing.T) {
	mockService := &mockHealthService{
		checkErr: errors.New("服务检查失败"),
	}
	var buf bytes.Buffer
	mockLogger := logger.New(logger.InfoLevel, logger.JSONFormat, &buf)

	handler := NewHealthHandler(mockService, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	// 检查状态码应该是 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码为 %d，但得到 %d", http.StatusInternalServerError, w.Code)
	}

	// 解析响应
	var resp model.ResponseData[health.HealthStatus]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应
	if resp.Code != 500 {
		t.Errorf("期望响应码为 500，但得到 %d", resp.Code)
	}

	if resp.Message != "健康检查失败" {
		t.Errorf("期望消息为 '健康检查失败'，但得到 '%s'", resp.Message)
	}

	if resp.Data != nil {
		t.Error("期望响应数据为 nil")
	}
}
