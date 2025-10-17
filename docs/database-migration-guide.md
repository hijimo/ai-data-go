# 数据库迁移指南

本指南介绍如何在项目中使用 GORM 进行数据库迁移。

## 快速开始

### 1. 安装依赖

依赖已经添加到项目中：

```bash
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
```

### 2. 配置环境变量

在 `.env` 文件中配置数据库连接信息：

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=genkit_ai_service
DB_SSLMODE=disable
DB_LOG_LEVEL=warn
```

### 3. 定义数据模型

在 `internal/model` 目录下创建你的模型文件，例如 `user.go`：

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

// User 用户模型
type User struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
    Email     string         `gorm:"size:100;uniqueIndex;not null" json:"email"`
    Password  string         `gorm:"size:255;not null" json:"-"`
    Active    bool           `gorm:"default:true" json:"active"`
    CreatedAt time.Time      `json:"createdAt"`
    UpdatedAt time.Time      `json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
    return "users"
}
```

### 4. 执行迁移

#### 方式一：使用迁移脚本（推荐）

1. 编辑 `scripts/migrate.go`，添加你的模型：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "genkit-ai-service/internal/config"
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/model"  // 导入你的模型
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

    // 创建数据库实例
    db := database.NewPostgresDatabase(dbConfig)

    // 连接数据库
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := db.Connect(ctx); err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer db.Close()

    fmt.Println("数据库连接成功")

    // 创建迁移器
    migrator := database.NewMigrator(db)

    // 执行迁移
    fmt.Println("开始执行数据库迁移...")
    if err := migrator.Migrate(
        &model.User{},
        // 在这里添加更多模型...
    ); err != nil {
        log.Fatalf("数据库迁移失败: %v", err)
    }
    
    fmt.Println("数据库迁移成功完成")
    os.Exit(0)
}
```

2. 运行迁移脚本：

```bash
go run scripts/migrate.go
```

#### 方式二：在应用启动时自动迁移

在 `main.go` 中添加迁移逻辑：

```go
package main

import (
    "context"
    "log"
    
    "genkit-ai-service/internal/config"
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/model"
)

func main() {
    // 加载配置
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }
    
    // 初始化数据库
    db := initDatabase(cfg)
    defer db.Close()
    
    // 执行迁移
    if err := db.AutoMigrate(
        &model.User{},
        // 添加更多模型...
    ); err != nil {
        log.Fatalf("数据库迁移失败: %v", err)
    }
    
    log.Println("数据库迁移完成")
    
    // 启动应用...
}

func initDatabase(cfg *config.Config) database.Database {
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
    
    db := database.NewPostgresDatabase(dbConfig)
    ctx := context.Background()
    
    if err := db.Connect(ctx); err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    
    return db
}
```

## GORM 模型标签说明

### 常用标签

```go
type Example struct {
    // 主键
    ID uint `gorm:"primaryKey"`
    
    // 字段大小限制
    Name string `gorm:"size:100"`
    
    // 非空约束
    Email string `gorm:"not null"`
    
    // 唯一索引
    Username string `gorm:"uniqueIndex"`
    
    // 普通索引
    Status string `gorm:"index"`
    
    // 默认值
    Active bool `gorm:"default:true"`
    
    // 自动更新时间
    CreatedAt time.Time
    UpdatedAt time.Time
    
    // 软删除
    DeletedAt gorm.DeletedAt `gorm:"index"`
    
    // 忽略字段（不映射到数据库）
    TempField string `gorm:"-"`
    
    // JSON 序列化时忽略
    Password string `json:"-"`
}
```

### 关联关系

```go
// 一对一
type User struct {
    ID      uint
    Profile Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    ID     uint
    UserID uint
    Bio    string
}

// 一对多
type User struct {
    ID       uint
    Articles []Article `gorm:"foreignKey:AuthorID"`
}

type Article struct {
    ID       uint
    AuthorID uint
    Title    string
}

// 多对多
type User struct {
    ID    uint
    Roles []Role `gorm:"many2many:user_roles;"`
}

type Role struct {
    ID    uint
    Name  string
    Users []User `gorm:"many2many:user_roles;"`
}
```

## 常见问题

### Q: 如何回滚迁移？

GORM 的 AutoMigrate 只会添加新表和新字段，不会删除或修改现有字段。如果需要回滚，需要手动执行 SQL：

```go
// 删除表
db.GetDB().Migrator().DropTable(&User{})

// 删除列
db.GetDB().Migrator().DropColumn(&User{}, "column_name")
```

### Q: 如何查看生成的 SQL？

设置日志级别为 `info`：

```env
DB_LOG_LEVEL=info
```

### Q: 如何处理数据库迁移冲突？

1. 使用版本控制管理迁移脚本
2. 在团队中协调数据库结构变更
3. 考虑使用专业的迁移工具如 `golang-migrate`

### Q: 生产环境如何安全迁移？

1. 先在测试环境验证迁移
2. 备份生产数据库
3. 在低峰期执行迁移
4. 准备回滚方案
5. 监控迁移过程

## 最佳实践

1. **模型定义**：将所有模型放在 `internal/model` 目录下
2. **迁移时机**：在应用启动时自动执行迁移（开发环境）或使用独立脚本（生产环境）
3. **日志级别**：开发环境使用 `info`，生产环境使用 `warn` 或 `error`
4. **软删除**：对重要数据使用软删除而非物理删除
5. **索引优化**：为常用查询字段添加索引
6. **字段约束**：合理使用 `not null`、`unique` 等约束
7. **命名规范**：使用清晰的表名和字段名

## 参考资料

- [GORM 官方文档](https://gorm.io/zh_CN/docs/)
- [GORM 迁移指南](https://gorm.io/zh_CN/docs/migration.html)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
