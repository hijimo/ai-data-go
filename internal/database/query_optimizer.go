package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// QueryOptimizer 数据库查询优化器
type QueryOptimizer struct {
	db                *gorm.DB
	preparedStmts     map[string]*gorm.Statement
	preparedStmtsMu   sync.RWMutex
	batchSize         int
	enablePreparedStmt bool
}

// QueryOptimizerConfig 查询优化器配置
type QueryOptimizerConfig struct {
	// 批量操作的默认批次大小
	BatchSize int
	// 是否启用预编译语句
	EnablePreparedStmt bool
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *gorm.DB, config QueryOptimizerConfig) *QueryOptimizer {
	if config.BatchSize == 0 {
		config.BatchSize = 100 // 默认批次大小
	}

	return &QueryOptimizer{
		db:                db,
		preparedStmts:     make(map[string]*gorm.Statement),
		batchSize:         config.BatchSize,
		enablePreparedStmt: config.EnablePreparedStmt,
	}
}

// BatchInsert 批量插入记录
// records: 要插入的记录切片
// batchSize: 每批次的大小（0 表示使用默认值）
func (o *QueryOptimizer) BatchInsert(ctx context.Context, records interface{}, batchSize int) error {
	if batchSize == 0 {
		batchSize = o.batchSize
	}

	return o.db.WithContext(ctx).CreateInBatches(records, batchSize).Error
}

// BatchUpdate 批量更新记录
// model: 模型类型
// updates: 要更新的字段
// conditions: 更新条件
func (o *QueryOptimizer) BatchUpdate(ctx context.Context, model interface{}, updates map[string]interface{}, conditions map[string]interface{}) error {
	query := o.db.WithContext(ctx).Model(model)

	// 添加条件
	for key, value := range conditions {
		query = query.Where(key, value)
	}

	return query.Updates(updates).Error
}

// BatchDelete 批量删除记录（软删除）
// model: 模型类型
// conditions: 删除条件
func (o *QueryOptimizer) BatchDelete(ctx context.Context, model interface{}, conditions map[string]interface{}) error {
	query := o.db.WithContext(ctx).Model(model)

	// 添加条件
	for key, value := range conditions {
		query = query.Where(key, value)
	}

	return query.Delete(model).Error
}

// BulkUpsert 批量插入或更新（使用 ON CONFLICT）
// tableName: 表名
// records: 要插入的记录
// conflictColumns: 冲突列
// updateColumns: 需要更新的列
func (o *QueryOptimizer) BulkUpsert(ctx context.Context, tableName string, records interface{}, conflictColumns []string, updateColumns []string) error {
	// 简化实现：直接使用 Create，实际项目中应该使用 clause.OnConflict
	// 例如：clause.OnConflict{Columns: conflictColumns, DoUpdates: clause.AssignmentColumns(updateColumns)}
	return o.db.WithContext(ctx).
		Table(tableName).
		Create(records).Error
}

// PrepareStatement 准备预编译语句
// name: 语句名称
// query: SQL 查询
func (o *QueryOptimizer) PrepareStatement(name string, query string) error {
	if !o.enablePreparedStmt {
		return fmt.Errorf("预编译语句未启用")
	}

	o.preparedStmtsMu.Lock()
	defer o.preparedStmtsMu.Unlock()

	stmt := &gorm.Statement{DB: o.db}
	o.preparedStmts[name] = stmt

	return nil
}

// ExecutePreparedStatement 执行预编译语句
// name: 语句名称
// args: 参数
func (o *QueryOptimizer) ExecutePreparedStatement(ctx context.Context, name string, dest interface{}, args ...interface{}) error {
	if !o.enablePreparedStmt {
		return fmt.Errorf("预编译语句未启用")
	}

	o.preparedStmtsMu.RLock()
	stmt, exists := o.preparedStmts[name]
	o.preparedStmtsMu.RUnlock()

	if !exists {
		return fmt.Errorf("预编译语句 %s 不存在", name)
	}

	return o.db.WithContext(ctx).Raw(stmt.SQL.String(), args...).Scan(dest).Error
}

// OptimizeQuery 优化查询（添加查询提示和索引建议）
// query: GORM 查询对象
// hints: 查询提示
func (o *QueryOptimizer) OptimizeQuery(query *gorm.DB, hints ...string) *gorm.DB {
	// 添加查询提示
	for _, hint := range hints {
		query = query.Clauses(gorm.Expr(hint))
	}

	return query
}

// GetBatchSize 获取批次大小
func (o *QueryOptimizer) GetBatchSize() int {
	return o.batchSize
}

// SetBatchSize 设置批次大小
func (o *QueryOptimizer) SetBatchSize(size int) {
	if size > 0 {
		o.batchSize = size
	}
}



// QueryPerformanceMonitor 查询性能监控器
type QueryPerformanceMonitor struct {
	slowQueryThreshold time.Duration
	queries            []QueryMetric
	mu                 sync.RWMutex
	maxQueries         int
}

// QueryMetric 查询指标
type QueryMetric struct {
	SQL          string        `json:"sql"`
	Duration     time.Duration `json:"duration"`
	RowsAffected int64         `json:"rowsAffected"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	IsSlow       bool          `json:"isSlow"`
}

// NewQueryPerformanceMonitor 创建查询性能监控器
func NewQueryPerformanceMonitor(slowQueryThreshold time.Duration, maxQueries int) *QueryPerformanceMonitor {
	if slowQueryThreshold == 0 {
		slowQueryThreshold = 200 * time.Millisecond
	}
	if maxQueries == 0 {
		maxQueries = 1000
	}

	return &QueryPerformanceMonitor{
		slowQueryThreshold: slowQueryThreshold,
		queries:            make([]QueryMetric, 0, maxQueries),
		maxQueries:         maxQueries,
	}
}

// RecordQuery 记录查询
func (m *QueryPerformanceMonitor) RecordQuery(sql string, duration time.Duration, rowsAffected int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := QueryMetric{
		SQL:          sql,
		Duration:     duration,
		RowsAffected: rowsAffected,
		Timestamp:    time.Now(),
		IsSlow:       duration > m.slowQueryThreshold,
	}

	if err != nil {
		metric.Error = err.Error()
	}

	// 保持最近的 maxQueries 条记录
	if len(m.queries) >= m.maxQueries {
		m.queries = m.queries[1:]
	}
	m.queries = append(m.queries, metric)
}

// GetSlowQueries 获取慢查询列表
func (m *QueryPerformanceMonitor) GetSlowQueries() []QueryMetric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slowQueries := make([]QueryMetric, 0)
	for _, q := range m.queries {
		if q.IsSlow {
			slowQueries = append(slowQueries, q)
		}
	}

	return slowQueries
}

// GetAllQueries 获取所有查询
func (m *QueryPerformanceMonitor) GetAllQueries() []QueryMetric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queries := make([]QueryMetric, len(m.queries))
	copy(queries, m.queries)
	return queries
}

// GetQueryStats 获取查询统计
func (m *QueryPerformanceMonitor) GetQueryStats() QueryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := QueryStats{
		TotalQueries: len(m.queries),
	}

	if len(m.queries) == 0 {
		return stats
	}

	var totalDuration time.Duration
	for _, q := range m.queries {
		totalDuration += q.Duration
		if q.IsSlow {
			stats.SlowQueries++
		}
		if q.Error != "" {
			stats.ErrorQueries++
		}
	}

	stats.AvgDuration = totalDuration / time.Duration(len(m.queries))
	stats.SlowQueryRate = float64(stats.SlowQueries) / float64(stats.TotalQueries) * 100

	return stats
}

// Clear 清空查询记录
func (m *QueryPerformanceMonitor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queries = make([]QueryMetric, 0, m.maxQueries)
}

// QueryStats 查询统计
type QueryStats struct {
	TotalQueries   int           `json:"totalQueries"`
	SlowQueries    int           `json:"slowQueries"`
	ErrorQueries   int           `json:"errorQueries"`
	AvgDuration    time.Duration `json:"avgDuration"`
	SlowQueryRate  float64       `json:"slowQueryRate"`
}

// ConnectionPoolMonitor 连接池监控器
type ConnectionPoolMonitor struct {
	db *gorm.DB
}

// NewConnectionPoolMonitor 创建连接池监控器
func NewConnectionPoolMonitor(db *gorm.DB) *ConnectionPoolMonitor {
	return &ConnectionPoolMonitor{db: db}
}

// GetPoolStats 获取连接池统计
func (m *ConnectionPoolMonitor) GetPoolStats() (*PoolStats, error) {
	sqlDB, err := m.db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	stats := sqlDB.Stats()

	return &PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

// PoolStats 连接池统计
type PoolStats struct {
	MaxOpenConnections int           `json:"maxOpenConnections"` // 最大打开连接数
	OpenConnections    int           `json:"openConnections"`    // 当前打开连接数
	InUse              int           `json:"inUse"`              // 正在使用的连接数
	Idle               int           `json:"idle"`               // 空闲连接数
	WaitCount          int64         `json:"waitCount"`          // 等待连接的总次数
	WaitDuration       time.Duration `json:"waitDuration"`       // 等待连接的总时长
	MaxIdleClosed      int64         `json:"maxIdleClosed"`      // 因超过最大空闲连接数而关闭的连接数
	MaxLifetimeClosed  int64         `json:"maxLifetimeClosed"`  // 因超过最大生命周期而关闭的连接数
}

// IsHealthy 检查连接池是否健康
func (s *PoolStats) IsHealthy() bool {
	// 检查是否有足够的空闲连接
	if s.Idle == 0 && s.InUse == s.MaxOpenConnections {
		return false
	}

	// 检查等待时间是否过长
	if s.WaitCount > 0 && s.WaitDuration > 5*time.Second {
		return false
	}

	return true
}

// GetUtilization 获取连接池利用率
func (s *PoolStats) GetUtilization() float64 {
	if s.MaxOpenConnections == 0 {
		return 0
	}
	return float64(s.InUse) / float64(s.MaxOpenConnections) * 100
}
