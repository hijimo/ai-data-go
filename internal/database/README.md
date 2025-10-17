# 数据库连接管理

本模块提供 PostgreSQL 数据库连接管理功能，使用 GORM 作为 ORM 框架，支持连接池配置、健康检查、自动迁移和优雅关闭。

## 功能特性

- ✅ PostgreSQL 数据库连接管理（基于 GORM）
- ✅ 连接池配置（最大连接数、空闲连接数、连接生命周期）
- ✅ 健康检查（Ping）
- ✅ 自动数据库迁移
- ✅ 灵活的日志级别配置
- ✅ 优雅关闭
- ✅ 上下文支持（超时控制）

## 依赖

- `gorm.io/gorm` - GORM ORM 框架
- `gorm.io/driver/postgres` - PostgreSQL 驱动

## 使用方法

### 基本使用

```go
package main

import (
    "context"
    "log"
    "time"
    
    "genkit-ai-service/internal/database"
)

func main() {
    // 创建数据库配置
    config := &database.PostgresConfig{
        Host:            "localhost",
        Port:            "5432",
        User:            "postgres",
        Password:        "your-password",
        DBName:          "genkit_ai_service",
        SSLMode:         "disable",
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
        LogLevel:        "warn", // silent, error, warn, info
    }
    
    // 创建数据库实例
    db := database.NewPostgresDatabase(config)
    
    // 连接数据库
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer db.Close()
    
    // 检查连接
    if err := db.Ping(ctx); err != nil {
        log.Fatalf("数据库连接检查失败: %v", err)
    }
    
    // 获取 GORM 数据库实例
    gormDB := db.GetDB()
    
    log.Println("数据库连接成功")
}
```

### 与配置模块集成

```go
package main

import (
    "context"
    "log"
    
    "genkit-ai-service/internal/config"
    "genkit-ai-service/internal/database"
)

func main() {
    // 加载配置
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }
    
    // 创建数据库配置
    dbConfig := &database.PostgresConfig{
        Host:            cfg.Database.Host,
        Port:            cfg.Database.Port,
        User:            cfg.Database.User,
        Password:        cfg.Database.Password,
        DBName:          cfg.Database.DBName,
        SSLMode:         cfg.Database.SSLMode,
        MaxOpenConns:    cfg.Database.MaxOpenConns,
        MaxIdleConns:    cfg.Database.MaxIdleConns,
        ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
        LogLevel:        cfg.Database.LogLevel,
    }
    
    // 连接数据库
    db := database.NewPostgresDatabase(dbConfig)
    ctx := context.Background()
    
    if err := db.Connect(ctx); err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer db.Close()
    
    log.Println("数据库连接成功")
}
```

## 数据库迁移

### 方式一：使用 AutoMigrate

```go
// 定义模型
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string    `gorm:"size:100;not null"`
    Email     string    `gorm:"size:100;uniqueIndex;not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// 执行迁移
if err := db.AutoMigrate(&User{}); err != nil {
    log.Fatal(err)
}
```

### 方式二：使用 Migrator

```go
// 创建迁移器
migrator := database.NewMigrator(db)

// 迁移多个模型
if err := migrator.Migrate(
    &User{},
    &ChatSession{},
    &Message{},
); err != nil {
    log.Fatal(err)
}
```

### 方式三：使用迁移脚本

1. 编辑 `scripts/migrate.go` 添加你的模型
2. 运行迁移脚本：

```bash
go run scripts/migrate.go
```

## GORM 使用示例

### 模型定义

```go
type User struct {
    ID        uint           `gorm:"primaryKey"`
    Name      string         `gorm:"size:100;not null"`
    Email     string         `gorm:"size:100;uniqueIndex;not null"`
    Age       int            `gorm:"default:0"`
    Active    bool           `gorm:"default:true"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除
}
```

### 常用操作

```go
gormDB := db.GetDB()

// 创建
user := User{Name: "张三", Email: "zhangsan@example.com"}
gormDB.Create(&user)

// 查询
var user User
gormDB.First(&user, 1) // 根据主键查询
gormDB.Where("email = ?", "zhangsan@example.com").First(&user)

// 更新
gormDB.Model(&user).Update("name", "李四")
gormDB.Model(&user).Updates(User{Name: "李四", Age: 30})

// 删除
gormDB.Delete(&user, 1) // 软删除
gormDB.Unscoped().Delete(&user, 1) // 永久删除

// 批量操作
var users []User
gormDB.Where("age > ?", 18).Find(&users)
```

## 配置说明

### PostgresConfig 结构

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| Host | string | 数据库主机地址 | localhost |
| Port | string | 数据库端口 | 5432 |
| User | string | 数据库用户名 | postgres |
| Password | string | 数据库密码 | - |
| DBName | string | 数据库名称 | genkit_ai_service |
| SSLMode | string | SSL 模式 | disable |
| MaxOpenConns | int | 最大打开连接数 | 25 |
| MaxIdleConns | int | 最大空闲连接数 | 5 |
| ConnMaxLifetime | time.Duration | 连接最大生命周期 | 5m |
| LogLevel | string | GORM 日志级别 | warn |

### 日志级别

GORM 支持以下日志级别（通过 `DB_LOG_LEVEL` 环境变量配置）：

- `silent` - 不输出任何日志
- `error` - 只输出错误日志
- `warn` - 输出警告和错误日志（推荐用于生产环境）
- `info` - 输出所有日志，包括 SQL 语句（推荐用于开发环境）

### 环境变量

在 `.env` 文件中配置以下环境变量：

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=genkit_ai_service
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
DB_LOG_LEVEL=warn
```

## 接口定义

### Database 接口

```go
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
```

## 错误处理

所有方法都返回详细的错误信息，便于调试和日志记录：

```go
if err := db.Connect(ctx); err != nil {
    // 错误信息格式: "打开数据库连接失败: <原始错误>"
    // 或: "数据库连接验证失败: <原始错误>"
    log.Printf("数据库连接错误: %v", err)
}

if err := db.Ping(ctx); err != nil {
    // 错误信息格式: "数据库未连接"
    // 或: "数据库连接检查失败: <原始错误>"
    log.Printf("健康检查错误: %v", err)
}

if err := db.AutoMigrate(&User{}); err != nil {
    // 错误信息格式: "数据库迁移失败: <原始错误>"
    log.Printf("迁移错误: %v", err)
}
```

## 最佳实践

### 1. 使用 defer 确保连接关闭

```go
db := database.NewPostgresDatabase(config)
if err := db.Connect(ctx); err != nil {
    return err
}
defer db.Close() // 确保连接被关闭
```

### 2. 使用上下文超时

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := db.Connect(ctx); err != nil {
    return err
}
```

### 3. 合理配置连接池

```go
config := &database.PostgresConfig{
    MaxOpenConns:    25,  // 根据应用负载调整
    MaxIdleConns:    5,   // 保持一定数量的空闲连接
    ConnMaxLifetime: 5 * time.Minute, // 定期回收连接
}
```

### 4. 定期健康检查

```go
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker.C {
    if err := db.Ping(context.Background()); err != nil {
        log.Printf("数据库健康检查失败: %v", err)
        // 可以触发告警或重连逻辑
    }
}
```

### 5. 使用软删除

```go
type User struct {
    ID        uint
    Name      string
    DeletedAt gorm.DeletedAt `gorm:"index"` // 启用软删除
}

// 软删除（记录不会真正删除）
db.GetDB().Delete(&user)

// 查询时自动排除已软删除的记录
db.GetDB().Find(&users)

// 包含软删除的记录
db.GetDB().Unscoped().Find(&users)

// 永久删除
db.GetDB().Unscoped().Delete(&user)
```

## 注意事项

1. **密码安全**：不要在代码中硬编码数据库密码，使用环境变量
2. **连接池配置**：根据应用负载和数据库资源合理配置连接池参数
3. **日志级别**：生产环境建议使用 `warn` 或 `error`，开发环境使用 `info`
4. **错误处理**：始终检查并处理错误，特别是连接和查询错误
5. **资源清理**：使用 `defer` 确保数据库连接被正确关闭
6. **上下文使用**：为所有数据库操作设置合理的超时时间
7. **迁移时机**：建议在应用启动时执行数据库迁移
8. **模型定义**：合理使用 GORM 标签定义字段约束和索引

## 参考资料

- [GORM 官方文档](https://gorm.io/zh_CN/docs/)
- [GORM PostgreSQL 驱动](https://gorm.io/zh_CN/docs/connecting_to_the_database.html#PostgreSQL)
