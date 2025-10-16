package health

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
)

// Service 健康检查服务接口
type Service interface {
	// Check 执行健康检查
	Check(ctx context.Context) (*HealthStatus, error)
}

// service 健康检查服务实现
type service struct {
	genkitClient genkit.Client
	database     database.Database
	startTime    time.Time
	version      string
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status       string                 `json:"status"`       // 整体状态：healthy, unhealthy
	Version      string                 `json:"version"`      // 服务版本
	Uptime       string                 `json:"uptime"`       // 运行时间
	Dependencies map[string]string      `json:"dependencies"` // 依赖服务状态
}

// NewService 创建新的健康检查服务
func NewService(genkitClient genkit.Client, db database.Database, version string) Service {
	return &service{
		genkitClient: genkitClient,
		database:     db,
		startTime:    time.Now(),
		version:      version,
	}
}

// Check 执行健康检查
func (s *service) Check(ctx context.Context) (*HealthStatus, error) {
	dependencies := make(map[string]string)
	allHealthy := true

	// 检查 Genkit 连接状态
	genkitStatus := s.checkGenkit(ctx)
	dependencies["genkit"] = genkitStatus
	if genkitStatus != "connected" {
		allHealthy = false
	}

	// 检查数据库连接状态
	dbStatus := s.checkDatabase(ctx)
	dependencies["database"] = dbStatus
	if dbStatus != "connected" {
		allHealthy = false
	}

	// 计算运行时间
	uptime := s.calculateUptime()

	// 确定整体状态
	status := "healthy"
	if !allHealthy {
		status = "unhealthy"
	}

	return &HealthStatus{
		Status:       status,
		Version:      s.version,
		Uptime:       uptime,
		Dependencies: dependencies,
	}, nil
}

// checkGenkit 检查 Genkit 连接状态
func (s *service) checkGenkit(ctx context.Context) string {
	if s.genkitClient == nil {
		return "not_configured"
	}

	// 尝试一个简单的生成请求来验证连接
	// 使用超时上下文避免长时间等待
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.genkitClient.Generate(checkCtx, "test", nil)
	if err != nil {
		return "disconnected"
	}

	return "connected"
}

// checkDatabase 检查数据库连接状态
func (s *service) checkDatabase(ctx context.Context) string {
	if s.database == nil {
		return "not_configured"
	}

	// 使用超时上下文避免长时间等待
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.database.Ping(checkCtx)
	if err != nil {
		return "disconnected"
	}

	return "connected"
}

// calculateUptime 计算运行时间
func (s *service) calculateUptime() string {
	duration := time.Since(s.startTime)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
