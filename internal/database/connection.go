package database

import (
	"database/sql"
	"fmt"
	"time"

	"ai-knowledge-platform/internal/config"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库连接实例
var DB *gorm.DB

// NewConnection 创建新的数据库连接
func NewConnection(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dsn string
	
	// 如果提供了完整的DATABASE_URL，直接使用
	if cfg.URL != "" {
		dsn = cfg.URL
	} else {
		// 否则根据配置参数构建DSN
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	}

	// 配置GORM日志
	gormLogger := logger.New(
		logrus.StandardLogger(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层的sql.DB对象来配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(10)                // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)               // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)      // 连接最大生存时间

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	DB = db
	logrus.Info("数据库连接成功")
	return db, nil
}

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// HealthCheck 数据库健康检查
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接检查失败: %w", err)
	}

	return nil
}