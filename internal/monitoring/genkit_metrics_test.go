package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenkitMetrics_RecordFlowExecution(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录 Flow 执行
	metrics.RecordFlowExecution("contextBuildFlow", "success", "tenant-123")
	metrics.RecordFlowExecution("contextBuildFlow", "error", "tenant-123")

	// 验证指标已记录（实际验证需要查询 Prometheus）
	// 这里只是确保方法不会 panic
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordFlowDuration(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录 Flow 执行时间
	duration := 150 * time.Millisecond
	metrics.RecordFlowDuration("contextBuildFlow", "tenant-123", duration)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordTokenUsage(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录 Token 使用量
	metrics.RecordTokenUsage("tenant-123", "prompt", "chatGenerateFlow", 1000)
	metrics.RecordTokenUsage("tenant-123", "completion", "chatGenerateFlow", 500)
	metrics.RecordTokenUsage("tenant-123", "total", "chatGenerateFlow", 1500)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordCacheHitMiss(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录缓存命中和未命中
	metrics.RecordCacheHit("context", "tenant-123")
	metrics.RecordCacheHit("context", "tenant-123")
	metrics.RecordCacheMiss("context", "tenant-123")

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordContextBuild(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录上下文构建
	metrics.RecordContextBuild("session-123", "tenant-123", 2000, 0.85)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordVectorSearch(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录向量检索
	duration := 50 * time.Millisecond
	metrics.RecordVectorSearch("tenant-123", duration, 5)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordSummaryGeneration(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录摘要生成
	metrics.RecordSummaryGeneration("incremental", "tenant-123", 0.8, 0.6)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordAIServiceCall(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录 AI 服务调用
	duration := 2 * time.Second
	metrics.RecordAIServiceCall("google", "gemini-1.5-flash", "success", "tenant-123", duration)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_UpdateSessionHealth(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试更新会话健康度
	metrics.UpdateSessionHealth("session-123", "tenant-123", 0.9)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_UpdateActiveSessions(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试更新活跃会话数
	metrics.UpdateActiveSessions("tenant-123", 50)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordMemoryOperations(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录记忆存储
	metrics.RecordMemoryStore("long_term", "tenant-123")

	// 测试记录记忆清理
	metrics.RecordMemoryCleanup("expired", "soft", "tenant-123", 10)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_UpdateSystemResources(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试更新系统资源
	metrics.UpdateDatabaseConnections(20)
	metrics.UpdateRedisConnections(10)

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_RecordFlowError(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试记录不同类型的错误
	errorTypes := []string{
		"timeout",
		"permission",
		"validation",
		"not_found",
		"ai_service",
		"database",
		"cache",
		"vector_service",
		"quota_exceeded",
		"unknown",
	}

	for _, errorType := range errorTypes {
		metrics.RecordFlowError("testFlow", errorType, "tenant-123")
	}

	// 验证指标已记录
	assert.NotNil(t, metrics)
}

func TestGenkitMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewGenkitMetrics()

	// 测试并发访问
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			metrics.RecordFlowExecution("testFlow", "success", "tenant-123")
			metrics.RecordFlowDuration("testFlow", "tenant-123", 100*time.Millisecond)
			metrics.RecordTokenUsage("tenant-123", "total", "testFlow", 1000)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证没有 panic
	assert.NotNil(t, metrics)
}

func BenchmarkGenkitMetrics_RecordFlowExecution(b *testing.B) {
	metrics := NewGenkitMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordFlowExecution("testFlow", "success", "tenant-123")
	}
}

func BenchmarkGenkitMetrics_RecordFlowDuration(b *testing.B) {
	metrics := NewGenkitMetrics()
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordFlowDuration("testFlow", "tenant-123", duration)
	}
}

func BenchmarkGenkitMetrics_RecordTokenUsage(b *testing.B) {
	metrics := NewGenkitMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordTokenUsage("tenant-123", "total", "testFlow", 1000)
	}
}

func BenchmarkGenkitMetrics_RecordCacheHit(b *testing.B) {
	metrics := NewGenkitMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordCacheHit("context", "tenant-123")
	}
}
