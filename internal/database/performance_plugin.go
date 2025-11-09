package database

import (
	"context"
	"time"

	"genkit-ai-service/internal/monitoring"

	"gorm.io/gorm"
)

// PerformancePlugin 数据库性能监控插件
type PerformancePlugin struct {
	perfMonitor       *QueryPerformanceMonitor
	slowQueryThreshold time.Duration
}

// NewPerformancePlugin 创建性能监控插件
func NewPerformancePlugin(perfMonitor *QueryPerformanceMonitor, slowQueryThreshold time.Duration) *PerformancePlugin {
	if slowQueryThreshold == 0 {
		slowQueryThreshold = 200 * time.Millisecond
	}

	return &PerformancePlugin{
		perfMonitor:       perfMonitor,
		slowQueryThreshold: slowQueryThreshold,
	}
}

// Name 插件名称
func (p *PerformancePlugin) Name() string {
	return "performance_monitoring_plugin"
}

// Initialize 初始化插件
func (p *PerformancePlugin) Initialize(db *gorm.DB) error {
	// 注册查询前回调
	if err := db.Callback().Query().Before("gorm:query").Register("performance:before_query", p.beforeQuery); err != nil {
		return err
	}

	// 注册查询后回调
	if err := db.Callback().Query().After("gorm:after_query").Register("performance:after_query", p.afterQuery); err != nil {
		return err
	}

	// 注册创建前回调
	if err := db.Callback().Create().Before("gorm:create").Register("performance:before_create", p.beforeQuery); err != nil {
		return err
	}

	// 注册创建后回调
	if err := db.Callback().Create().After("gorm:after_create").Register("performance:after_create", p.afterQuery); err != nil {
		return err
	}

	// 注册更新前回调
	if err := db.Callback().Update().Before("gorm:update").Register("performance:before_update", p.beforeQuery); err != nil {
		return err
	}

	// 注册更新后回调
	if err := db.Callback().Update().After("gorm:after_update").Register("performance:after_update", p.afterQuery); err != nil {
		return err
	}

	// 注册删除前回调
	if err := db.Callback().Delete().Before("gorm:delete").Register("performance:before_delete", p.beforeQuery); err != nil {
		return err
	}

	// 注册删除后回调
	if err := db.Callback().Delete().After("gorm:after_delete").Register("performance:after_delete", p.afterQuery); err != nil {
		return err
	}

	return nil
}

// beforeQuery 查询前回调
func (p *PerformancePlugin) beforeQuery(db *gorm.DB) {
	// 记录开始时间
	db.InstanceSet("performance:start_time", time.Now())
}

// afterQuery 查询后回调
func (p *PerformancePlugin) afterQuery(db *gorm.DB) {
	// 获取开始时间
	startTime, ok := db.InstanceGet("performance:start_time")
	if !ok {
		return
	}

	start, ok := startTime.(time.Time)
	if !ok {
		return
	}

	// 计算执行时间
	duration := time.Since(start)

	// 获取 SQL 语句
	sql := db.Statement.SQL.String()
	if sql == "" && db.Statement.SQL.Len() > 0 {
		sql = db.Statement.SQL.String()
	}

	// 获取影响的行数
	rowsAffected := db.Statement.RowsAffected

	// 记录到性能监控器
	if p.perfMonitor != nil {
		p.perfMonitor.RecordQuery(sql, duration, rowsAffected, db.Error)
	}

	// 如果是慢查询，记录到全局监控指标
	if duration > p.slowQueryThreshold {
		monitoring.GetMetrics().RecordSlowQuery()
	}

	// 如果有错误，记录到全局监控指标
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		monitoring.GetMetrics().RecordDBError()
	}
}

// PreparedStatementPlugin 预编译语句插件
type PreparedStatementPlugin struct {
	enabled bool
}

// NewPreparedStatementPlugin 创建预编译语句插件
func NewPreparedStatementPlugin(enabled bool) *PreparedStatementPlugin {
	return &PreparedStatementPlugin{
		enabled: enabled,
	}
}

// Name 插件名称
func (p *PreparedStatementPlugin) Name() string {
	return "prepared_statement_plugin"
}

// Initialize 初始化插件
func (p *PreparedStatementPlugin) Initialize(db *gorm.DB) error {
	if !p.enabled {
		return nil
	}

	// 启用预编译语句
	db.Config.PrepareStmt = true

	return nil
}

// BatchOperationPlugin 批量操作插件
type BatchOperationPlugin struct {
	defaultBatchSize int
}

// NewBatchOperationPlugin 创建批量操作插件
func NewBatchOperationPlugin(defaultBatchSize int) *BatchOperationPlugin {
	if defaultBatchSize == 0 {
		defaultBatchSize = 100
	}

	return &BatchOperationPlugin{
		defaultBatchSize: defaultBatchSize,
	}
}

// Name 插件名称
func (p *BatchOperationPlugin) Name() string {
	return "batch_operation_plugin"
}

// Initialize 初始化插件
func (p *BatchOperationPlugin) Initialize(db *gorm.DB) error {
	// 设置默认批次大小
	db.Config.CreateBatchSize = p.defaultBatchSize

	return nil
}

// ConnectionPoolHealthCheck 连接池健康检查
type ConnectionPoolHealthCheck struct {
	db                *gorm.DB
	checkInterval     time.Duration
	stopChan          chan struct{}
	poolMonitor       *ConnectionPoolMonitor
}

// NewConnectionPoolHealthCheck 创建连接池健康检查
func NewConnectionPoolHealthCheck(db *gorm.DB, checkInterval time.Duration) *ConnectionPoolHealthCheck {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}

	return &ConnectionPoolHealthCheck{
		db:            db,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
		poolMonitor:   NewConnectionPoolMonitor(db),
	}
}

// Start 启动健康检查
func (h *ConnectionPoolHealthCheck) Start(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.checkHealth()
		}
	}
}

// Stop 停止健康检查
func (h *ConnectionPoolHealthCheck) Stop() {
	close(h.stopChan)
}

// checkHealth 检查连接池健康状态
func (h *ConnectionPoolHealthCheck) checkHealth() {
	stats, err := h.poolMonitor.GetPoolStats()
	if err != nil {
		// 记录错误
		monitoring.GetMetrics().RecordDBError()
		return
	}

	// 检查连接池是否健康
	if !stats.IsHealthy() {
		// 记录警告（这里可以发送告警）
		// 例如：发送到监控系统或日志
	}

	// 检查连接池利用率
	utilization := stats.GetUtilization()
	if utilization > 80 {
		// 高利用率警告
	}
}

// QueryCachePlugin 查询缓存插件（简单实现）
type QueryCachePlugin struct {
	cache   map[string]interface{}
	enabled bool
}

// NewQueryCachePlugin 创建查询缓存插件
func NewQueryCachePlugin(enabled bool) *QueryCachePlugin {
	return &QueryCachePlugin{
		cache:   make(map[string]interface{}),
		enabled: enabled,
	}
}

// Name 插件名称
func (p *QueryCachePlugin) Name() string {
	return "query_cache_plugin"
}

// Initialize 初始化插件
func (p *QueryCachePlugin) Initialize(db *gorm.DB) error {
	if !p.enabled {
		return nil
	}

	// 注册查询前回调
	if err := db.Callback().Query().Before("gorm:query").Register("cache:before_query", p.beforeQuery); err != nil {
		return err
	}

	// 注册查询后回调
	if err := db.Callback().Query().After("gorm:after_query").Register("cache:after_query", p.afterQuery); err != nil {
		return err
	}

	return nil
}

// beforeQuery 查询前回调
func (p *QueryCachePlugin) beforeQuery(db *gorm.DB) {
	if !p.enabled {
		return
	}

	// 检查缓存
	cacheKey := db.Statement.SQL.String()
	if cachedResult, exists := p.cache[cacheKey]; exists {
		// 使用缓存结果
		db.InstanceSet("cache:hit", true)
		db.InstanceSet("cache:result", cachedResult)
		monitoring.RecordCacheHit("query_cache")
	} else {
		monitoring.RecordCacheMiss("query_cache")
	}
}

// afterQuery 查询后回调
func (p *QueryCachePlugin) afterQuery(db *gorm.DB) {
	if !p.enabled {
		return
	}

	// 检查是否命中缓存
	if hit, ok := db.InstanceGet("cache:hit"); ok && hit.(bool) {
		return
	}

	// 缓存查询结果
	cacheKey := db.Statement.SQL.String()
	if db.Error == nil && db.Statement.Dest != nil {
		p.cache[cacheKey] = db.Statement.Dest
	}
}

// Clear 清空缓存
func (p *QueryCachePlugin) Clear() {
	p.cache = make(map[string]interface{})
}
