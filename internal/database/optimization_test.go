package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestQueryOptimizer 测试查询优化器
func TestQueryOptimizer(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建查询优化器
	optimizer := NewQueryOptimizer(db, QueryOptimizerConfig{
		BatchSize:          50,
		EnablePreparedStmt: true,
	})

	assert.NotNil(t, optimizer)
	assert.Equal(t, 50, optimizer.GetBatchSize())

	// 测试设置批次大小
	optimizer.SetBatchSize(100)
	assert.Equal(t, 100, optimizer.GetBatchSize())
}

// TestQueryPerformanceMonitor 测试查询性能监控器
func TestQueryPerformanceMonitor(t *testing.T) {
	monitor := NewQueryPerformanceMonitor(200*time.Millisecond, 100)
	assert.NotNil(t, monitor)

	// 记录一些查询
	monitor.RecordQuery("SELECT * FROM users", 100*time.Millisecond, 10, nil)
	monitor.RecordQuery("SELECT * FROM posts", 300*time.Millisecond, 20, nil)
	monitor.RecordQuery("SELECT * FROM comments", 150*time.Millisecond, 5, nil)

	// 获取所有查询
	queries := monitor.GetAllQueries()
	assert.Equal(t, 3, len(queries))

	// 获取慢查询
	slowQueries := monitor.GetSlowQueries()
	assert.Equal(t, 1, len(slowQueries))
	assert.True(t, slowQueries[0].IsSlow)

	// 获取查询统计
	stats := monitor.GetQueryStats()
	assert.Equal(t, 3, stats.TotalQueries)
	assert.Equal(t, 1, stats.SlowQueries)
	assert.Greater(t, stats.SlowQueryRate, 0.0)

	// 清空查询记录
	monitor.Clear()
	queries = monitor.GetAllQueries()
	assert.Equal(t, 0, len(queries))
}

// TestConnectionPoolMonitor 测试连接池监控器
func TestConnectionPoolMonitor(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	monitor := NewConnectionPoolMonitor(db)
	assert.NotNil(t, monitor)

	// 获取连接池统计
	stats, err := monitor.GetPoolStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// 检查健康状态
	assert.True(t, stats.IsHealthy())

	// 获取利用率
	utilization := stats.GetUtilization()
	assert.GreaterOrEqual(t, utilization, 0.0)
	assert.LessOrEqual(t, utilization, 100.0)
}

// TestOptimizationConfig 测试优化配置
func TestOptimizationConfig(t *testing.T) {
	config := DefaultOptimizationConfig()

	assert.True(t, config.EnablePreparedStmt)
	assert.Equal(t, 100, config.DefaultBatchSize)
	assert.Equal(t, 200*time.Millisecond, config.SlowQueryThreshold)
	assert.True(t, config.EnablePerformanceMonitoring)
	assert.False(t, config.EnableQueryCache)
	assert.Equal(t, 30*time.Second, config.PoolHealthCheckInterval)
	assert.Equal(t, 1000, config.MaxQueryRecords)
}

// TestApplyOptimizations 测试应用优化
func TestApplyOptimizations(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	config := DefaultOptimizationConfig()
	err = ApplyOptimizations(db, config)
	assert.NoError(t, err)

	// 验证预编译语句已启用
	assert.True(t, db.Config.PrepareStmt)

	// 验证批次大小已设置
	assert.Equal(t, config.DefaultBatchSize, db.Config.CreateBatchSize)
}

// TestOptimizationManager 测试优化管理器
func TestOptimizationManager(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	config := DefaultOptimizationConfig()
	manager := NewOptimizationManager(db, config)
	assert.NotNil(t, manager)

	// 初始化（跳过索引验证，因为 SQLite 不支持完整的 PostgreSQL 功能）
	// ctx := context.Background()
	// err = manager.Initialize(ctx)
	// assert.NoError(t, err)

	// 获取性能报告
	report := manager.GetPerformanceReport()
	assert.NotNil(t, report)

	// 检查报告摘要
	summary := report.GetSummary()
	assert.NotEmpty(t, summary)

	// 关闭管理器
	manager.Shutdown()
}

// TestPerformanceReport 测试性能报告
func TestPerformanceReport(t *testing.T) {
	report := &PerformanceReport{
		QueryStats: QueryStats{
			TotalQueries:  100,
			SlowQueries:   5,
			ErrorQueries:  2,
			AvgDuration:   150 * time.Millisecond,
			SlowQueryRate: 5.0,
		},
		SlowQueries: []QueryMetric{
			{
				SQL:      "SELECT * FROM large_table",
				Duration: 500 * time.Millisecond,
				IsSlow:   true,
			},
		},
		PoolStats: &PoolStats{
			MaxOpenConnections: 25,
			OpenConnections:    20,
			InUse:              15,
			Idle:               5,
		},
	}

	// 获取摘要
	summary := report.GetSummary()
	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "查询总数")
	assert.Contains(t, summary, "慢查询")
	assert.Contains(t, summary, "连接池")

	// 检查是否有问题
	hasIssues := report.HasIssues()
	assert.False(t, hasIssues) // 5% 慢查询率不算问题
}

// TestPerformanceReportWithIssues 测试有问题的性能报告
func TestPerformanceReportWithIssues(t *testing.T) {
	report := &PerformanceReport{
		QueryStats: QueryStats{
			TotalQueries:  100,
			SlowQueries:   15, // 15% 慢查询率
			ErrorQueries:  2,
			AvgDuration:   150 * time.Millisecond,
			SlowQueryRate: 15.0,
		},
	}

	// 检查是否有问题
	hasIssues := report.HasIssues()
	assert.True(t, hasIssues) // 15% 慢查询率算问题
}

// TestPoolStatsHealthy 测试连接池健康状态
func TestPoolStatsHealthy(t *testing.T) {
	// 健康的连接池
	healthyStats := &PoolStats{
		MaxOpenConnections: 25,
		OpenConnections:    20,
		InUse:              10,
		Idle:               10,
		WaitCount:          0,
		WaitDuration:       0,
	}
	assert.True(t, healthyStats.IsHealthy())

	// 不健康的连接池（所有连接都在使用，没有空闲）
	unhealthyStats := &PoolStats{
		MaxOpenConnections: 25,
		OpenConnections:    25,
		InUse:              25,
		Idle:               0,
		WaitCount:          10,
		WaitDuration:       10 * time.Second,
	}
	assert.False(t, unhealthyStats.IsHealthy())
}

// TestPoolStatsUtilization 测试连接池利用率
func TestPoolStatsUtilization(t *testing.T) {
	stats := &PoolStats{
		MaxOpenConnections: 25,
		InUse:              20,
	}

	utilization := stats.GetUtilization()
	assert.Equal(t, 80.0, utilization)
}

// BenchmarkQueryPerformanceMonitor 基准测试查询性能监控器
func BenchmarkQueryPerformanceMonitor(b *testing.B) {
	monitor := NewQueryPerformanceMonitor(200*time.Millisecond, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordQuery("SELECT * FROM users", 100*time.Millisecond, 10, nil)
	}
}

// BenchmarkQueryOptimizer 基准测试查询优化器
func BenchmarkQueryOptimizer(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	optimizer := NewQueryOptimizer(db, QueryOptimizerConfig{
		BatchSize:          100,
		EnablePreparedStmt: true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.GetBatchSize()
	}
}
