package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL 驱动
)

// Database 数据库接口
type Database interface {
	// Connect 连接数据库
	Connect(ctx context.Context) error
	// Close 关闭数据库连接
	Close() error
	// Ping 检查数据库连接
	Ping(ctx context.Context) error
	// GetDB 获取数据库实例
	GetDB() *sql.DB
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	Host            string        // 数据库主机
	Port            string        // 数据库端口
	User            string        // 数据库用户名
	Password        string        // 数据库密码
	DBName          string        // 数据库名称
	SSLMode         string        // SSL模式
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
}

// PostgresDatabase PostgreSQL 数据库实现
type PostgresDatabase struct {
	db     *sql.DB
	config *PostgresConfig
}

// NewPostgresDatabase 创建新的 PostgreSQL 数据库实例
func NewPostgresDatabase(config *PostgresConfig) *PostgresDatabase {
	return &PostgresDatabase{
		config: config,
	}
}

// Connect 连接数据库
func (p *PostgresDatabase) Connect(ctx context.Context) error {
	// 构建连接字符串
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.config.Host,
		p.config.Port,
		p.config.User,
		p.config.Password,
		p.config.DBName,
		p.config.SSLMode,
	)

	// 打开数据库连接
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(p.config.MaxOpenConns)
	db.SetMaxIdleConns(p.config.MaxIdleConns)
	db.SetConnMaxLifetime(p.config.ConnMaxLifetime)

	// 验证连接
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("数据库连接验证失败: %w", err)
	}

	p.db = db
	return nil
}

// Close 关闭数据库连接
func (p *PostgresDatabase) Close() error {
	if p.db == nil {
		return nil
	}

	if err := p.db.Close(); err != nil {
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

	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库连接检查失败: %w", err)
	}

	return nil
}

// GetDB 获取数据库实例
func (p *PostgresDatabase) GetDB() *sql.DB {
	return p.db
}
