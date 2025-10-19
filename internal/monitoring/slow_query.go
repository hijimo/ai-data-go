package monitoring

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SlowQueryConfig 慢查询配置
type SlowQueryConfig struct {
	// 慢查询阈值（超过此时间的查询将被记录）
	SlowThreshold time.Duration
	// 是否启用慢查询日志
	Enabled bool
	// 日志记录器
	Logger logger.Interface
}

// SlowQueryLogger 慢查询日志记录器
type SlowQueryLogger struct {
	logger.Interface
	config SlowQueryConfig
}

// NewSlowQueryLogger 创建慢查询日志记录器
func NewSlowQueryLogger(config SlowQueryConfig) *SlowQueryLogger {
	if config.SlowThreshold == 0 {
		config.SlowThreshold = 200 * time.Millisecond // 默认 200ms
	}
	
	return &SlowQueryLogger{
		Interface: config.Logger,
		config:    config,
	}
}

// Trace 实现 gorm logger.Interface 的 Trace 方法
func (l *SlowQueryLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !l.config.Enabled {
		// 如果未启用，调用原始 logger
		if l.Interface != nil {
			l.Interface.Trace(ctx, begin, fc, err)
		}
		return
	}
	
	elapsed := time.Since(begin)
	sql, rows := fc()
	
	// 检查是否为慢查询
	if elapsed > l.config.SlowThreshold {
		// 记录慢查询指标
		GetMetrics().RecordSlowQuery()
		
		// 记录详细日志
		if l.Interface != nil {
			l.Interface.Warn(ctx, "SLOW QUERY [%v] [rows:%v] %s", elapsed, rows, sql)
		}
	}
	
	// 记录数据库错误
	if err != nil && err != gorm.ErrRecordNotFound {
		GetMetrics().RecordDBError()
	}
	
	// 调用原始 logger
	if l.Interface != nil {
		l.Interface.Trace(ctx, begin, fc, err)
	}
}

// SlowQueryMiddleware GORM 慢查询中间件插件
type SlowQueryMiddleware struct {
	config SlowQueryConfig
}

// NewSlowQueryMiddleware 创建慢查询中间件
func NewSlowQueryMiddleware(config SlowQueryConfig) *SlowQueryMiddleware {
	if config.SlowThreshold == 0 {
		config.SlowThreshold = 200 * time.Millisecond
	}
	return &SlowQueryMiddleware{config: config}
}

// Name 插件名称
func (m *SlowQueryMiddleware) Name() string {
	return "slow_query_middleware"
}

// Initialize 初始化插件
func (m *SlowQueryMiddleware) Initialize(db *gorm.DB) error {
	// 注册回调
	return db.Callback().Query().Before("gorm:query").Register("monitoring:before_query", m.beforeQuery)
}

// beforeQuery 查询前回调
func (m *SlowQueryMiddleware) beforeQuery(db *gorm.DB) {
	if !m.config.Enabled {
		return
	}
	
	// 记录开始时间
	db.InstanceSet("monitoring:start_time", time.Now())
}

// afterQuery 查询后回调（需要在 Initialize 中注册）
func (m *SlowQueryMiddleware) afterQuery(db *gorm.DB) {
	if !m.config.Enabled {
		return
	}
	
	// 获取开始时间
	startTime, ok := db.InstanceGet("monitoring:start_time")
	if !ok {
		return
	}
	
	start, ok := startTime.(time.Time)
	if !ok {
		return
	}
	
	elapsed := time.Since(start)
	
	// 检查是否为慢查询
	if elapsed > m.config.SlowThreshold {
		GetMetrics().RecordSlowQuery()
		
		// 记录慢查询日志
		if m.config.Logger != nil {
			m.config.Logger.Warn(db.Statement.Context, 
				"SLOW QUERY [%v] %s", 
				elapsed, 
				db.Statement.SQL.String())
		}
	}
	
	// 记录错误
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		GetMetrics().RecordDBError()
	}
}

// QueryMetrics 查询性能指标
type QueryMetrics struct {
	SQL           string        `json:"sql"`
	Duration      time.Duration `json:"duration"`
	RowsAffected  int64         `json:"rows_affected"`
	Error         string        `json:"error,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}

// SlowQueryCollector 慢查询收集器
type SlowQueryCollector struct {
	queries   []QueryMetrics
	maxSize   int
	threshold time.Duration
}

// NewSlowQueryCollector 创建慢查询收集器
func NewSlowQueryCollector(threshold time.Duration, maxSize int) *SlowQueryCollector {
	if threshold == 0 {
		threshold = 200 * time.Millisecond
	}
	if maxSize == 0 {
		maxSize = 100
	}
	
	return &SlowQueryCollector{
		queries:   make([]QueryMetrics, 0, maxSize),
		maxSize:   maxSize,
		threshold: threshold,
	}
}

// Record 记录查询
func (c *SlowQueryCollector) Record(sql string, duration time.Duration, rowsAffected int64, err error) {
	// 只记录慢查询
	if duration < c.threshold {
		return
	}
	
	metric := QueryMetrics{
		SQL:          sql,
		Duration:     duration,
		RowsAffected: rowsAffected,
		Timestamp:    time.Now(),
	}
	
	if err != nil {
		metric.Error = err.Error()
	}
	
	// 保持最近的 maxSize 条记录
	if len(c.queries) >= c.maxSize {
		c.queries = c.queries[1:]
	}
	c.queries = append(c.queries, metric)
	
	// 记录到全局指标
	GetMetrics().RecordSlowQuery()
}

// GetSlowQueries 获取慢查询列表
func (c *SlowQueryCollector) GetSlowQueries() []QueryMetrics {
	return c.queries
}

// Clear 清空慢查询记录
func (c *SlowQueryCollector) Clear() {
	c.queries = make([]QueryMetrics, 0, c.maxSize)
}
