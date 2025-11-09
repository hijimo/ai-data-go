package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据库接口
type Database interface {
	// Connect 连接数据库
	Connect(ctx context.Context) error
	// Close 关闭数据库连接
	Close() error
	// Ping 检查数据库连接
	Ping(ctx context.Context) error
	// GetDB 获取 GORM 数据库实例
	GetDB() *gorm.DB
	// AutoMigrate 自动迁移数据库表结构
	AutoMigrate(models ...interface{}) error
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	DSN             string        // 数据库连接字符串（优先使用）
	Host            string        // 数据库主机
	Port            string        // 数据库端口
	User            string        // 数据库用户名
	Password        string        // 数据库密码
	DBName          string        // 数据库名称
	SSLMode         string        // SSL模式
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
	LogLevel        string        // GORM 日志级别 (silent, error, warn, info)
}

// PostgresDatabase PostgreSQL 数据库实现
type PostgresDatabase struct {
	db                *gorm.DB
	config            *PostgresConfig
	queryOptimizer    *QueryOptimizer
	perfMonitor       *QueryPerformanceMonitor
	indexValidator    *IndexValidator
	poolMonitor       *ConnectionPoolMonitor
}

// NewPostgresDatabase 创建新的 PostgreSQL 数据库实例
func NewPostgresDatabase(config *PostgresConfig) *PostgresDatabase {
	return &PostgresDatabase{
		config: config,
	}
}

// Connect 连接数据库
func (p *PostgresDatabase) Connect(ctx context.Context) error {
	// 获取连接字符串
	// 优先使用 DSN，如果未设置则根据独立配置项构建
	var dsn string
	if p.config.DSN != "" {
		dsn = p.config.DSN
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			p.config.Host,
			p.config.Port,
			p.config.User,
			p.config.Password,
			p.config.DBName,
			p.config.SSLMode,
		)
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(ParseLogLevel(string(p.config.LogLevel))),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 打开数据库连接
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 获取底层的 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(p.config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(p.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(p.config.ConnMaxLifetime)

	// 验证连接
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库连接验证失败: %w", err)
	}

	p.db = db

	// 初始化查询优化器
	p.queryOptimizer = NewQueryOptimizer(db, QueryOptimizerConfig{
		BatchSize:          100,
		EnablePreparedStmt: true,
	})

	// 初始化性能监控器
	p.perfMonitor = NewQueryPerformanceMonitor(200*time.Millisecond, 1000)

	// 初始化索引验证器
	p.indexValidator = NewIndexValidator(db)

	// 初始化连接池监控器
	p.poolMonitor = NewConnectionPoolMonitor(db)

	return nil
}

// Close 关闭数据库连接
func (p *PostgresDatabase) Close() error {
	if p.db == nil {
		return nil
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭数据库连接失败: %w", err)
	}

	p.db = nil
	return nil
}

// Ping 检查数据库连接
func (p *PostgresDatabase) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("数据库未连接")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库连接检查失败: %w", err)
	}

	return nil
}

// GetDB 获取 GORM 数据库实例
func (p *PostgresDatabase) GetDB() *gorm.DB {
	return p.db
}

// AutoMigrate 自动迁移数据库表结构
func (p *PostgresDatabase) AutoMigrate(models ...interface{}) error {
	if p.db == nil {
		return fmt.Errorf("数据库未连接")
	}

	if err := p.db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

// GetQueryOptimizer 获取查询优化器
func (p *PostgresDatabase) GetQueryOptimizer() *QueryOptimizer {
	return p.queryOptimizer
}

// GetPerformanceMonitor 获取性能监控器
func (p *PostgresDatabase) GetPerformanceMonitor() *QueryPerformanceMonitor {
	return p.perfMonitor
}

// GetIndexValidator 获取索引验证器
func (p *PostgresDatabase) GetIndexValidator() *IndexValidator {
	return p.indexValidator
}

// GetConnectionPoolMonitor 获取连接池监控器
func (p *PostgresDatabase) GetConnectionPoolMonitor() *ConnectionPoolMonitor {
	return p.poolMonitor
}

// ValidateIndexes 验证所有必需的索引
func (p *PostgresDatabase) ValidateIndexes(ctx context.Context) ([]MissingIndex, error) {
	if p.indexValidator == nil {
		return nil, fmt.Errorf("索引验证器未初始化")
	}
	return p.indexValidator.ValidateIndexes(ctx)
}

// GetConnectionPoolStats 获取连接池统计信息
func (p *PostgresDatabase) GetConnectionPoolStats() (*PoolStats, error) {
	if p.poolMonitor == nil {
		return nil, fmt.Errorf("连接池监控器未初始化")
	}
	return p.poolMonitor.GetPoolStats()
}
