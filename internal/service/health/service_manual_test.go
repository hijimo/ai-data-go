package health

import (
	"context"
	"fmt"
	"testing"
	"time"

	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
	"github.com/redis/go-redis/v9"
)

// 这是一个手动测试文件，用于验证健康检查服务的功能
// 由于依赖外部服务，这些测试需要手动运行

// TestHealthCheckWithAllServices 测试所有服务都可用的情况
func TestHealthCheckWithAllServices(t *testing.T) {
	t.Skip("手动测试 - 需要实际的数据库和Redis连接")
	
	// 创建模拟的服务
	mockGenkit := &mockGenkitClient{}
	mockDB := &mockDatabase{}
	mockRedis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// 创建健康检查服务
	svc := NewService(mockGenkit, mockDB, mockRedis, "1.0.0")
	
	// 执行健康检查
	ctx := context.Background()
	status, err := svc.Check(ctx)
	
	if err != nil {
		t.Fatalf("健康检查失败: %v", err)
	}
	
	// 验证结果
	if status.Status != "healthy" && status.Status != "degraded" {
		t.Errorf("期望状态为 healthy 或 degraded，实际为: %s", status.Status)
	}
	
	// 验证各项检查
	if _, ok := status.Checks["database"]; !ok {
		t.Error("缺少数据库检查")
	}
	
	if _, ok := status.Checks["redis"]; !ok {
		t.Error("缺少Redis检查")
	}
	
	if _, ok := status.Checks["ai_service"]; !ok {
		t.Error("缺少AI服务检查")
	}
	
	if _, ok := status.Checks["system_resources"]; !ok {
		t.Error("缺少系统资源检查")
	}
	
	// 验证系统信息
	if status.SystemInfo == nil {
		t.Error("缺少系统信息")
	} else {
		if status.SystemInfo.CPUCores <= 0 {
			t.Error("CPU核心数应该大于0")
		}
		if status.SystemInfo.Goroutines <= 0 {
			t.Error("Goroutine数量应该大于0")
		}
	}
	
	// 打印结果
	fmt.Printf("健康检查结果:\n")
	fmt.Printf("  状态: %s\n", status.Status)
	fmt.Printf("  版本: %s\n", status.Version)
	fmt.Printf("  运行时间: %s\n", status.Uptime)
	fmt.Printf("  时间戳: %s\n", status.Timestamp)
	fmt.Printf("\n检查项:\n")
	for name, check := range status.Checks {
		fmt.Printf("  %s: %s (%dms) - %s\n", name, check.Status, check.ResponseTime, check.Message)
	}
	if status.SystemInfo != nil {
		fmt.Printf("\n系统信息:\n")
		fmt.Printf("  CPU核心数: %d\n", status.SystemInfo.CPUCores)
		fmt.Printf("  Goroutines: %d\n", status.SystemInfo.Goroutines)
		fmt.Printf("  内存使用: %.2f%%\n", status.SystemInfo.MemoryUsage)
	}
}

// TestHealthCheckWithoutRedis 测试没有Redis的情况
func TestHealthCheckWithoutRedis(t *testing.T) {
	t.Skip("手动测试 - 需要实际的数据库连接")
	
	mockGenkit := &mockGenkitClient{}
	mockDB := &mockDatabase{}
	
	// 不传递Redis客户端
	svc := NewService(mockGenkit, mockDB, nil, "1.0.0")
	
	ctx := context.Background()
	status, err := svc.Check(ctx)
	
	if err != nil {
		t.Fatalf("健康检查失败: %v", err)
	}
	
	// Redis应该显示为降级状态
	redisCheck, ok := status.Checks["redis"]
	if !ok {
		t.Error("缺少Redis检查")
	} else if redisCheck.Status != "degraded" {
		t.Errorf("期望Redis状态为 degraded，实际为: %s", redisCheck.Status)
	}
}

// Mock implementations for testing

type mockGenkitClient struct{}

func (m *mockGenkitClient) Initialize(ctx context.Context, config *genkit.Config) error {
	return nil
}

func (m *mockGenkitClient) InitializeModel(ctx context.Context) error {
	return nil
}

func (m *mockGenkitClient) Generate(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResponse, error) {
	// 模拟成功的生成
	return &genkit.GenerateResponse{
		Text: "health check response",
	}, nil
}

func (m *mockGenkitClient) GenerateStream(ctx context.Context, prompt string, options *genkit.GenerateOptions) (<-chan *genkit.StreamChunk, <-chan error) {
	chunks := make(chan *genkit.StreamChunk)
	errs := make(chan error)
	close(chunks)
	close(errs)
	return chunks, errs
}

func (m *mockGenkitClient) AbortGeneration(sessionID string) error {
	return nil
}

type mockDatabase struct{}

func (m *mockDatabase) Connect(ctx context.Context) error {
	return nil
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	// 模拟成功的ping
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (m *mockDatabase) GetDB() interface{} {
	return nil
}
