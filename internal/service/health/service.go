package health

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
	"github.com/redis/go-redis/v9"
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
	redisClient  *redis.Client
	startTime    time.Time
	version      string
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status       string                 `json:"status" example:"healthy"`                                  // 整体状态：healthy, degraded, unhealthy
	Version      string                 `json:"version" example:"1.0.0"`                                   // 服务版本
	Uptime       string                 `json:"uptime" example:"2h30m15s"`                                 // 运行时间
	Timestamp    string                 `json:"timestamp" example:"2025-11-02T10:30:00Z"`                  // 检查时间戳
	Checks       map[string]CheckResult `json:"checks"`                                                    // 各项检查结果
	SystemInfo   *SystemInfo            `json:"systemInfo,omitempty"`                                      // 系统资源信息
}

// CheckResult 单项检查结果
type CheckResult struct {
	Status      string                 `json:"status" example:"healthy"`                // 状态：healthy, degraded, unhealthy
	Message     string                 `json:"message,omitempty" example:"连接正常"`       // 状态消息
	ResponseTime int64                  `json:"responseTime,omitempty" example:"15"`     // 响应时间（毫秒）
	Details     map[string]interface{} `json:"details,omitempty"`                       // 详细信息
}

// SystemInfo 系统资源信息
type SystemInfo struct {
	CPUCores    int     `json:"cpuCores" example:"8"`                    // CPU 核心数
	Goroutines  int     `json:"goroutines" example:"42"`                 // Goroutine 数量
	MemoryAlloc uint64  `json:"memoryAlloc" example:"52428800"`          // 已分配内存（字节）
	MemoryTotal uint64  `json:"memoryTotal" example:"104857600"`         // 总内存（字节）
	MemoryUsage float64 `json:"memoryUsage" example:"50.5"`              // 内存使用率（百分比）
}

// NewService 创建新的健康检查服务
func NewService(genkitClient genkit.Client, db database.Database, redisClient *redis.Client, version string) Service {
	return &service{
		genkitClient: genkitClient,
		database:     db,
		redisClient:  redisClient,
		startTime:    time.Now(),
		version:      version,
	}
}

// Check 执行健康检查
func (s *service) Check(ctx context.Context) (*HealthStatus, error) {
	checks := make(map[string]CheckResult)
	
	// 检查数据库连接
	checks["database"] = s.checkDatabase(ctx)
	
	// 检查 Redis 连接
	checks["redis"] = s.checkRedis(ctx)
	
	// 检查 AI 服务可用性
	checks["ai_service"] = s.checkAIService(ctx)
	
	// 检查系统资源
	checks["system_resources"] = s.checkSystemResources(ctx)
	
	// 计算运行时间
	uptime := s.calculateUptime()
	
	// 获取系统信息
	systemInfo := s.getSystemInfo()
	
	// 确定整体状态
	status := s.determineOverallStatus(checks)
	
	return &HealthStatus{
		Status:     status,
		Version:    s.version,
		Uptime:     uptime,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Checks:     checks,
		SystemInfo: systemInfo,
	}, nil
}

// determineOverallStatus 根据各项检查结果确定整体状态
func (s *service) determineOverallStatus(checks map[string]CheckResult) string {
	hasUnhealthy := false
	hasDegraded := false
	
	for _, check := range checks {
		switch check.Status {
		case "unhealthy":
			hasUnhealthy = true
		case "degraded":
			hasDegraded = true
		}
	}
	
	// 如果有核心服务不健康，整体状态为 unhealthy
	if checks["database"].Status == "unhealthy" {
		return "unhealthy"
	}
	
	// 如果有任何服务不健康，整体状态为 unhealthy
	if hasUnhealthy {
		return "unhealthy"
	}
	
	// 如果有服务降级，整体状态为 degraded
	if hasDegraded {
		return "degraded"
	}
	
	return "healthy"
}

// checkDatabase 检查数据库连接状态
func (s *service) checkDatabase(ctx context.Context) CheckResult {
	startTime := time.Now()
	
	if s.database == nil {
		return CheckResult{
			Status:  "unhealthy",
			Message: "数据库未配置",
		}
	}

	// 使用超时上下文避免长时间等待
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.database.Ping(checkCtx)
	responseTime := time.Since(startTime).Milliseconds()
	
	if err != nil {
		return CheckResult{
			Status:       "unhealthy",
			Message:      fmt.Sprintf("数据库连接失败: %v", err),
			ResponseTime: responseTime,
		}
	}

	// 检查响应时间
	status := "healthy"
	message := "数据库连接正常"
	if responseTime > 1000 {
		status = "degraded"
		message = "数据库响应缓慢"
	}

	return CheckResult{
		Status:       status,
		Message:      message,
		ResponseTime: responseTime,
		Details: map[string]interface{}{
			"type": "PostgreSQL",
		},
	}
}

// checkRedis 检查 Redis 连接状态
func (s *service) checkRedis(ctx context.Context) CheckResult {
	startTime := time.Now()
	
	if s.redisClient == nil {
		return CheckResult{
			Status:  "degraded",
			Message: "Redis 未配置（可选服务）",
		}
	}

	// 使用超时上下文避免长时间等待
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 执行 PING 命令
	err := s.redisClient.Ping(checkCtx).Err()
	responseTime := time.Since(startTime).Milliseconds()
	
	if err != nil {
		return CheckResult{
			Status:       "degraded",
			Message:      fmt.Sprintf("Redis 连接失败: %v", err),
			ResponseTime: responseTime,
		}
	}

	// 检查响应时间
	status := "healthy"
	message := "Redis 连接正常"
	if responseTime > 500 {
		status = "degraded"
		message = "Redis 响应缓慢"
	}

	return CheckResult{
		Status:       status,
		Message:      message,
		ResponseTime: responseTime,
	}
}

// checkAIService 检查 AI 服务可用性
func (s *service) checkAIService(ctx context.Context) CheckResult {
	startTime := time.Now()
	
	if s.genkitClient == nil {
		return CheckResult{
			Status:  "unhealthy",
			Message: "AI 服务未配置",
		}
	}

	// 使用超时上下文避免长时间等待
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 尝试一个简单的生成请求来验证连接
	// 使用最小的 token 数量以加快检查速度
	_, err := s.genkitClient.Generate(checkCtx, "health check", nil)
	responseTime := time.Since(startTime).Milliseconds()
	
	if err != nil {
		// 检查是否是超时错误
		if checkCtx.Err() == context.DeadlineExceeded {
			return CheckResult{
				Status:       "unhealthy",
				Message:      "AI 服务响应超时",
				ResponseTime: responseTime,
			}
		}
		
		return CheckResult{
			Status:       "unhealthy",
			Message:      fmt.Sprintf("AI 服务不可用: %v", err),
			ResponseTime: responseTime,
		}
	}

	// 检查响应时间
	status := "healthy"
	message := "AI 服务正常"
	if responseTime > 5000 {
		status = "degraded"
		message = "AI 服务响应缓慢"
	}

	return CheckResult{
		Status:       status,
		Message:      message,
		ResponseTime: responseTime,
		Details: map[string]interface{}{
			"provider": "Google Genkit",
		},
	}
}

// checkSystemResources 检查系统资源
func (s *service) checkSystemResources(ctx context.Context) CheckResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// 计算内存使用率
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100
	
	// 获取 Goroutine 数量
	goroutines := runtime.NumGoroutine()
	
	// 确定状态
	status := "healthy"
	message := "系统资源正常"
	
	// 检查内存使用率
	if memoryUsage > 85 {
		status = "degraded"
		message = "内存使用率较高"
	}
	
	// 检查 Goroutine 数量
	if goroutines > 10000 {
		status = "degraded"
		message = "Goroutine 数量过多"
	}
	
	return CheckResult{
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"cpuCores":    runtime.NumCPU(),
			"goroutines":  goroutines,
			"memoryAlloc": m.Alloc,
			"memoryTotal": m.Sys,
			"memoryUsage": fmt.Sprintf("%.2f%%", memoryUsage),
		},
	}
}

// getSystemInfo 获取系统信息
func (s *service) getSystemInfo() *SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100
	
	return &SystemInfo{
		CPUCores:    runtime.NumCPU(),
		Goroutines:  runtime.NumGoroutine(),
		MemoryAlloc: m.Alloc,
		MemoryTotal: m.Sys,
		MemoryUsage: memoryUsage,
	}
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
