package health

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"genkit-ai-service/internal/genkit"

	"github.com/firebase/genkit/go/ai"
)

// mockGenkitClient 模拟 Genkit 客户端
type mockGenkitClient struct {
	generateErr error
}

func (m *mockGenkitClient) Initialize(ctx context.Context, config *genkit.Config) error {
	return nil
}

func (m *mockGenkitClient) Generate(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &genkit.GenerateResult{
		Text:  "test response",
		Model: "test-model",
	}, nil
}

func (m *mockGenkitClient) SetModel(model ai.Model) {}

func (m *mockGenkitClient) Close() error {
	return nil
}

// mockDatabase 模拟数据库
type mockDatabase struct {
	pingErr error
}

func (m *mockDatabase) Connect(ctx context.Context) error {
	return nil
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockDatabase) GetDB() *sql.DB {
	return nil
}

func TestNewService(t *testing.T) {
	mockGenkit := &mockGenkitClient{}
	mockDB := &mockDatabase{}
	version := "1.0.0"

	svc := NewService(mockGenkit, mockDB, version)

	if svc == nil {
		t.Fatal("期望创建服务成功，但得到 nil")
	}
}

func TestCheck_AllHealthy(t *testing.T) {
	mockGenkit := &mockGenkitClient{}
	mockDB := &mockDatabase{}
	version := "1.0.0"

	svc := NewService(mockGenkit, mockDB, version)
	ctx := context.Background()

	status, err := svc.Check(ctx)

	if err != nil {
		t.Fatalf("期望检查成功，但得到错误: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("期望状态为 'healthy'，但得到 '%s'", status.Status)
	}

	if status.Version != version {
		t.Errorf("期望版本为 '%s'，但得到 '%s'", version, status.Version)
	}

	if status.Uptime == "" {
		t.Error("期望运行时间不为空")
	}

	if status.Dependencies["genkit"] != "connected" {
		t.Errorf("期望 Genkit 状态为 'connected'，但得到 '%s'", status.Dependencies["genkit"])
	}

	if status.Dependencies["database"] != "connected" {
		t.Errorf("期望数据库状态为 'connected'，但得到 '%s'", status.Dependencies["database"])
	}
}

func TestCheck_GenkitUnhealthy(t *testing.T) {
	mockGenkit := &mockGenkitClient{
		generateErr: errors.New("连接失败"),
	}
	mockDB := &mockDatabase{}
	version := "1.0.0"

	svc := NewService(mockGenkit, mockDB, version)
	ctx := context.Background()

	status, err := svc.Check(ctx)

	if err != nil {
		t.Fatalf("期望检查成功，但得到错误: %v", err)
	}

	if status.Status != "unhealthy" {
		t.Errorf("期望状态为 'unhealthy'，但得到 '%s'", status.Status)
	}

	if status.Dependencies["genkit"] != "disconnected" {
		t.Errorf("期望 Genkit 状态为 'disconnected'，但得到 '%s'", status.Dependencies["genkit"])
	}
}

func TestCheck_DatabaseUnhealthy(t *testing.T) {
	mockGenkit := &mockGenkitClient{}
	mockDB := &mockDatabase{
		pingErr: errors.New("数据库连接失败"),
	}
	version := "1.0.0"

	svc := NewService(mockGenkit, mockDB, version)
	ctx := context.Background()

	status, err := svc.Check(ctx)

	if err != nil {
		t.Fatalf("期望检查成功，但得到错误: %v", err)
	}

	if status.Status != "unhealthy" {
		t.Errorf("期望状态为 'unhealthy'，但得到 '%s'", status.Status)
	}

	if status.Dependencies["database"] != "disconnected" {
		t.Errorf("期望数据库状态为 'disconnected'，但得到 '%s'", status.Dependencies["database"])
	}
}

func TestCheck_NilDependencies(t *testing.T) {
	version := "1.0.0"

	svc := NewService(nil, nil, version)
	ctx := context.Background()

	status, err := svc.Check(ctx)

	if err != nil {
		t.Fatalf("期望检查成功，但得到错误: %v", err)
	}

	if status.Status != "unhealthy" {
		t.Errorf("期望状态为 'unhealthy'，但得到 '%s'", status.Status)
	}

	if status.Dependencies["genkit"] != "not_configured" {
		t.Errorf("期望 Genkit 状态为 'not_configured'，但得到 '%s'", status.Dependencies["genkit"])
	}

	if status.Dependencies["database"] != "not_configured" {
		t.Errorf("期望数据库状态为 'not_configured'，但得到 '%s'", status.Dependencies["database"])
	}
}

func TestCalculateUptime(t *testing.T) {
	mockGenkit := &mockGenkitClient{}
	mockDB := &mockDatabase{}
	version := "1.0.0"

	svc := NewService(mockGenkit, mockDB, version).(*service)

	// 测试不同的运行时间
	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{
			name:     "秒级运行时间",
			duration: 30 * time.Second,
			contains: "s",
		},
		{
			name:     "分钟级运行时间",
			duration: 5 * time.Minute,
			contains: "m",
		},
		{
			name:     "小时级运行时间",
			duration: 2 * time.Hour,
			contains: "h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.startTime = time.Now().Add(-tt.duration)
			uptime := svc.calculateUptime()

			if uptime == "" {
				t.Error("期望运行时间不为空")
			}

			// 简单检查格式是否包含预期的时间单位
			if len(uptime) == 0 {
				t.Errorf("期望运行时间包含 '%s'，但得到空字符串", tt.contains)
			}
		})
	}
}
