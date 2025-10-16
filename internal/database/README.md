# 数据库连接管理

本模块提供 PostgreSQL 数据库连接管理功能，支持连接池配置、健康检查和优雅关闭。

## 功能特性

- ✅ PostgreSQL 数据库连接管理
- ✅ 连接池配置（最大连接数、空闲连接数、连接生命周期）
- ✅ 健康检查（Ping）
- ✅ 优雅关闭
- ✅ 上下文支持（超时控制）

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

### 健康检查

```go
func healthCheck(db database.Database) error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    if err := db.Ping(ctx); err != nil {
        return fmt.Errorf("数据库健康检查失败: %w", err)
    }
    
    return nil
}
```

### 执行查询

```go
func queryUsers(db database.Database) error {
    sqlDB := db.GetDB()
    if sqlDB == nil {
        return fmt.Errorf("数据库未连接")
    }
    
    ctx := context.Background()
    rows, err := sqlDB.QueryContext(ctx, "SELECT id, name FROM users")
    if err != nil {
        return fmt.Errorf("查询失败: %w", err)
    }
    defer rows.Close()
    
    for rows.Next() {
        var id int
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            return fmt.Errorf("扫描行失败: %w", err)
        }
        fmt.Printf("用户: ID=%d, Name=%s\n", id, name)
    }
    
    return rows.Err()
}
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
    
    // GetDB 获取数据库实例
    GetDB() *sql.DB
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

if err := db.Close(); err != nil {
    // 错误信息格式: "关闭数据库连接失败: <原始错误>"
    log.Printf("关闭连接错误: %v", err)
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

## 测试

运行单元测试：

```bash
# 运行所有测试（跳过集成测试）
go test -v ./internal/database/... -short

# 运行包括集成测试（需要 PostgreSQL 运行）
go test -v ./internal/database/...

# 查看测试覆盖率
go test -v ./internal/database/... -cover -short
```

## 注意事项

1. **密码安全**：不要在代码中硬编码数据库密码，使用环境变量
2. **连接池配置**：根据应用负载和数据库资源合理配置连接池参数
3. **错误处理**：始终检查并处理错误，特别是连接和查询错误
4. **资源清理**：使用 `defer` 确保数据库连接和查询结果被正确关闭
5. **上下文使用**：为所有数据库操作设置合理的超时时间

## 未来扩展

当前实现为基础版本，未来可以扩展以下功能：

- 数据库迁移管理
- 读写分离支持
- 连接重试机制
- 连接池监控指标
- 慢查询日志
- 事务管理辅助函数
