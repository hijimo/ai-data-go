# GORM 集成完成说明

## 已完成的工作

### 1. 依赖安装

已成功添加以下依赖到项目：

- `gorm.io/gorm` v1.31.0 - GORM ORM 框架
- `gorm.io/driver/postgres` v1.6.0 - PostgreSQL 驱动

### 2. 代码更新

#### 数据库层 (`internal/database/`)

- **postgres.go**: 更新为使用 GORM 而非原生 `database/sql`
  - 修改 `Database` 接口，添加 `AutoMigrate` 方法
  - 更新 `PostgresConfig` 添加 `LogLevel` 配置
  - 重构 `Connect` 方法使用 GORM
  - 修改 `GetDB` 返回 `*gorm.DB` 而非 `*sql.DB`
  - 添加 `AutoMigrate` 方法支持数据库迁移

- **migrate.go**: 新增迁移工具
  - `ParseLogLevel` 函数：解析日志级别字符串
  - `Migrator` 结构体：封装迁移逻辑
  - `Migrate` 方法：执行批量模型迁移

#### 配置层 (`internal/config/`)

- **config.go**: 更新数据库配置
  - `DatabaseConfig` 添加 `LogLevel` 字段
  - 添加日志级别验证逻辑
  - 支持通过 `DB_LOG_LEVEL` 环境变量配置

#### 脚本 (`scripts/`)

- **migrate.go**: 新增独立迁移脚本
  - 可独立运行的数据库迁移工具
  - 支持批量迁移多个模型
  - 包含完整的错误处理和日志输出

### 3. 配置文件更新

- **.env.example**: 添加 `DB_LOG_LEVEL` 配置项
  - 支持的值：`silent`, `error`, `warn`, `info`
  - 默认值：`warn`

### 4. 文档

创建了以下文档：

- **internal/database/README.md**: 数据库模块使用文档
  - GORM 基本使用方法
  - 迁移指南
  - 配置说明
  - 最佳实践

- **docs/database-migration-guide.md**: 详细的迁移指南
  - 快速开始教程
  - 模型定义示例
  - GORM 标签说明
  - 常见问题解答

- **docs/gorm-integration.md**: 本文档

## 使用方法

### 环境配置

在 `.env` 文件中添加：

```env
DB_LOG_LEVEL=warn
```

### 定义模型

在 `internal/model` 目录下创建模型文件：

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

type User struct {
    ID        uint           `gorm:"primaryKey"`
    Username  string         `gorm:"size:50;uniqueIndex;not null"`
    Email     string         `gorm:"size:100;uniqueIndex;not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 执行迁移

#### 方式一：使用迁移脚本

```bash
# 编辑 scripts/migrate.go 添加模型
# 然后运行
go run scripts/migrate.go
```

#### 方式二：在应用中自动迁移

```go
// 在 main.go 中
if err := db.AutoMigrate(&model.User{}); err != nil {
    log.Fatal(err)
}
```

### 使用 GORM 进行数据操作

```go
// 获取 GORM 实例
gormDB := db.GetDB()

// 创建
user := User{Username: "zhangsan", Email: "zhangsan@example.com"}
gormDB.Create(&user)

// 查询
var user User
gormDB.First(&user, 1)
gormDB.Where("email = ?", "zhangsan@example.com").First(&user)

// 更新
gormDB.Model(&user).Update("username", "lisi")

// 删除
gormDB.Delete(&user)
```

## 日志级别说明

- **silent**: 不输出任何日志
- **error**: 只输出错误日志
- **warn**: 输出警告和错误（推荐生产环境）
- **info**: 输出所有日志包括 SQL 语句（推荐开发环境）

## 兼容性说明

### 向后兼容

虽然底层从 `database/sql` 切换到 GORM，但接口保持兼容：

- `Database` 接口新增了 `AutoMigrate` 方法
- `GetDB()` 返回类型从 `*sql.DB` 改为 `*gorm.DB`
- 如果需要访问底层 `*sql.DB`，可以使用：

  ```go
  sqlDB, err := gormDB.DB()
  ```

### 迁移建议

如果项目中有使用原生 SQL 的代码：

1. 可以继续使用，通过 `gormDB.DB()` 获取 `*sql.DB`
2. 逐步迁移到 GORM 的 API
3. 或者保持混用（GORM 支持原生 SQL）

```go
// 使用原生 SQL
var result []map[string]interface{}
db.GetDB().Raw("SELECT * FROM users WHERE age > ?", 18).Scan(&result)

// 执行原生 SQL
db.GetDB().Exec("UPDATE users SET active = ? WHERE id = ?", true, 1)
```

## 下一步

现在你可以：

1. 创建数据模型（在 `internal/model` 目录）
2. 运行迁移脚本创建数据库表
3. 在业务代码中使用 GORM 进行数据操作

## 参考资料

- [GORM 官方文档](https://gorm.io/zh_CN/docs/)
- [GORM GitHub](https://github.com/go-gorm/gorm)
- [PostgreSQL 驱动文档](https://gorm.io/zh_CN/docs/connecting_to_the_database.html#PostgreSQL)
