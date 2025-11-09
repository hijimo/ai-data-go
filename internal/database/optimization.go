package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// OptimizationConfig 数据库优化配置
type OptimizationConfig struct {
	// 启用预编译语句
	EnablePreparedStmt bool
	// 默认批次大小
	DefaultBatchSize int
	// 慢查询阈值
	SlowQueryThreshold time.Duration
	// 启用性能监控
	EnablePerformanceMonitoring bool
	// 启用查询缓存
	EnableQueryCache bool
	// 连接池健康检查间隔
	PoolHealthCheckInterval time.Duration
	// 最大查询记录数
	MaxQueryRecords int
}

// DefaultOptimizationConfig 默认优化配置
func DefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		EnablePreparedStmt:          true,
		DefaultBatchSize:            100,
		SlowQueryThreshold:          200 * time.Millisecond,
		EnablePerformanceMonitoring: true,
		EnableQueryCache:            false, // 默认关闭查询缓存
		PoolHealthCheckInterval:     30 * time.Second,
		MaxQueryRecords:             1000,
	}
}

// ApplyOptimizations 应用数据库优化
func ApplyOptimizations(db *gorm.DB, config OptimizationConfig) error {
	// 1. 启用预编译语句
	if config.EnablePreparedStmt {
		preparedStmtPlugin := NewPreparedStatementPlugin(true)
		if err := db.Use(preparedStmtPlugin); err != nil {
			return fmt.Errorf("启用预编译语句插件失败: %w", err)
		}
	}

	// 2. 配置批量操作
	if config.DefaultBatchSize > 0 {
		batchPlugin := NewBatchOperationPlugin(config.DefaultBatchSize)
		if err := db.Use(batchPlugin); err != nil {
			return fmt.Errorf("启用批量操作插件失败: %w", err)
		}
	}

	// 3. 启用性能监控
	if config.EnablePerformanceMonitoring {
		perfMonitor := NewQueryPerformanceMonitor(config.SlowQueryThreshold, config.MaxQueryRecords)
		perfPlugin := NewPerformancePlugin(perfMonitor, config.SlowQueryThreshold)
		if err := db.Use(perfPlugin); err != nil {
			return fmt.Errorf("启用性能监控插件失败: %w", err)
		}
	}

	// 4. 启用查询缓存（可选）
	if config.EnableQueryCache {
		cachePlugin := NewQueryCachePlugin(true)
		if err := db.Use(cachePlugin); err != nil {
			return fmt.Errorf("启用查询缓存插件失败: %w", err)
		}
	}

	return nil
}

// OptimizationManager 优化管理器
type OptimizationManager struct {
	db                *gorm.DB
	config            OptimizationConfig
	perfMonitor       *QueryPerformanceMonitor
	indexValidator    *IndexValidator
	poolMonitor       *ConnectionPoolMonitor
	poolHealthCheck   *ConnectionPoolHealthCheck
}

// NewOptimizationManager 创建优化管理器
func NewOptimizationManager(db *gorm.DB, config OptimizationConfig) *OptimizationManager {
	return &OptimizationManager{
		db:              db,
		config:          config,
		perfMonitor:     NewQueryPerformanceMonitor(config.SlowQueryThreshold, config.MaxQueryRecords),
		indexValidator:  NewIndexValidator(db),
		poolMonitor:     NewConnectionPoolMonitor(db),
		poolHealthCheck: NewConnectionPoolHealthCheck(db, config.PoolHealthCheckInterval),
	}
}

// Initialize 初始化优化管理器
func (m *OptimizationManager) Initialize(ctx context.Context) error {
	// 应用优化配置
	if err := ApplyOptimizations(m.db, m.config); err != nil {
		return fmt.Errorf("应用数据库优化失败: %w", err)
	}

	// 验证索引
	missingIndexes, err := m.indexValidator.ValidateIndexes(ctx)
	if err != nil {
		return fmt.Errorf("验证索引失败: %w", err)
	}

	// 如果有缺失的索引，记录警告
	if len(missingIndexes) > 0 {
		fmt.Printf("警告: 发现 %d 个缺失的索引\n", len(missingIndexes))
		for _, idx := range missingIndexes {
			fmt.Printf("  - 表: %s, 列: %v, 原因: %s\n", idx.TableName, idx.Columns, idx.Reason)
		}
	}

	// 启动连接池健康检查
	if m.config.PoolHealthCheckInterval > 0 {
		go m.poolHealthCheck.Start(ctx)
	}

	return nil
}

// GetPerformanceReport 获取性能报告
func (m *OptimizationManager) GetPerformanceReport() *PerformanceReport {
	report := &PerformanceReport{
		QueryStats: m.perfMonitor.GetQueryStats(),
	}

	// 获取慢查询
	report.SlowQueries = m.perfMonitor.GetSlowQueries()

	// 获取连接池统计
	if poolStats, err := m.poolMonitor.GetPoolStats(); err == nil {
		report.PoolStats = poolStats
	}

	return report
}

// GetIndexReport 获取索引报告
func (m *OptimizationManager) GetIndexReport(ctx context.Context) (*IndexReport, error) {
	return m.indexValidator.GenerateIndexReport(ctx)
}

// CreateMissingIndexes 创建缺失的索引
func (m *OptimizationManager) CreateMissingIndexes(ctx context.Context) error {
	missingIndexes, err := m.indexValidator.ValidateIndexes(ctx)
	if err != nil {
		return fmt.Errorf("验证索引失败: %w", err)
	}

	if len(missingIndexes) == 0 {
		return nil
	}

	return m.indexValidator.CreateMissingIndexes(ctx, missingIndexes)
}

// Shutdown 关闭优化管理器
func (m *OptimizationManager) Shutdown() {
	if m.poolHealthCheck != nil {
		m.poolHealthCheck.Stop()
	}
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	QueryStats  QueryStats     `json:"queryStats"`
	SlowQueries []QueryMetric  `json:"slowQueries"`
	PoolStats   *PoolStats     `json:"poolStats,omitempty"`
}

// GetSummary 获取报告摘要
func (r *PerformanceReport) GetSummary() string {
	summary := fmt.Sprintf(
		"查询总数: %d, 慢查询: %d (%.2f%%), 错误查询: %d, 平均耗时: %v",
		r.QueryStats.TotalQueries,
		r.QueryStats.SlowQueries,
		r.QueryStats.SlowQueryRate,
		r.QueryStats.ErrorQueries,
		r.QueryStats.AvgDuration,
	)

	if r.PoolStats != nil {
		summary += fmt.Sprintf(
			"\n连接池: 打开连接 %d/%d, 使用中 %d, 空闲 %d, 利用率 %.2f%%",
			r.PoolStats.OpenConnections,
			r.PoolStats.MaxOpenConnections,
			r.PoolStats.InUse,
			r.PoolStats.Idle,
			r.PoolStats.GetUtilization(),
		)
	}

	return summary
}

// HasIssues 检查是否有性能问题
func (r *PerformanceReport) HasIssues() bool {
	// 检查慢查询率
	if r.QueryStats.SlowQueryRate > 10 {
		return true
	}

	// 检查错误率
	if r.QueryStats.TotalQueries > 0 {
		errorRate := float64(r.QueryStats.ErrorQueries) / float64(r.QueryStats.TotalQueries) * 100
		if errorRate > 5 {
			return true
		}
	}

	// 检查连接池健康状态
	if r.PoolStats != nil && !r.PoolStats.IsHealthy() {
		return true
	}

	return false
}

// OptimizationRecommendations 优化建议
type OptimizationRecommendations struct {
	Recommendations []string `json:"recommendations"`
}

// GenerateRecommendations 生成优化建议
func (m *OptimizationManager) GenerateRecommendations(ctx context.Context) (*OptimizationRecommendations, error) {
	recommendations := &OptimizationRecommendations{
		Recommendations: make([]string, 0),
	}

	// 1. 检查索引
	missingIndexes, err := m.indexValidator.ValidateIndexes(ctx)
	if err != nil {
		return nil, err
	}

	if len(missingIndexes) > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("发现 %d 个缺失的索引，建议创建以提高查询性能", len(missingIndexes)))
	}

	// 2. 检查未使用的索引
	unusedIndexes, err := m.indexValidator.GetUnusedIndexes(ctx)
	if err == nil && len(unusedIndexes) > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("发现 %d 个未使用的索引，建议删除以减少维护开销", len(unusedIndexes)))
	}

	// 3. 检查慢查询
	slowQueries := m.perfMonitor.GetSlowQueries()
	if len(slowQueries) > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("发现 %d 个慢查询，建议优化查询语句或添加索引", len(slowQueries)))
	}

	// 4. 检查连接池
	poolStats, err := m.poolMonitor.GetPoolStats()
	if err == nil {
		utilization := poolStats.GetUtilization()
		if utilization > 80 {
			recommendations.Recommendations = append(recommendations.Recommendations,
				fmt.Sprintf("连接池利用率较高 (%.2f%%)，建议增加最大连接数", utilization))
		}

		if poolStats.WaitCount > 100 {
			recommendations.Recommendations = append(recommendations.Recommendations,
				"连接等待次数较多，建议增加连接池大小或优化查询性能")
		}
	}

	// 5. 检查查询统计
	queryStats := m.perfMonitor.GetQueryStats()
	if queryStats.SlowQueryRate > 10 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("慢查询率较高 (%.2f%%)，建议优化查询或添加索引", queryStats.SlowQueryRate))
	}

	if len(recommendations.Recommendations) == 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			"数据库性能良好，暂无优化建议")
	}

	return recommendations, nil
}
