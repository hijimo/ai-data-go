# 数据库迁移

本目录包含会话管理系统的数据库迁移脚本。

## 文件说明

- `migration_manager.go`: 迁移管理器，负责统一管理和执行所有迁移
- `session_migration.go`: 会话管理相关表的迁移脚本

## 使用方法

### 在应用启动时执行迁移

```go
import (
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/database/migrations"
)

// 连接数据库
db := database.NewPostgresDatabase(config)
if err := db.Connect(ctx); err != nil {
    log.Fatal(err)
}

// 执行会话管理迁移
if err := migrations.RunSessionMigrations(db.GetDB()); err != nil {
    log.Fatal(err)
}
```

### 手动执行迁移

可以创建一个独立的迁移命令：

```go
// cmd/migrate/main.go
package main

import (
    "context"
    "log"
    
    "genkit-ai-service/internal/config"
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/database/migrations"
)

func main() {
    // 加载配置
    cfg := config.Load()
    
    // 连接数据库
    db := database.NewPostgresDatabase(&database.PostgresConfig{
        Host:     cfg.Database.Host,
        Port:     cfg.Database.Port,
        User:     cfg.Database.User,
        Password: cfg.Database.Password,
        DBName:   cfg.Database.DBName,
        SSLMode:  cfg.Database.SSLMode,
    })
    
    if err := db.Connect(context.Background()); err != nil {
        log.Fatalf("数据库连接失败: %v", err)
    }
    defer db.Close()
    
    // 执行迁移
    if err := migrations.RunSessionMigrations(db.GetDB()); err != nil {
        log.Fatalf("迁移执行失败: %v", err)
    }
    
    log.Println("迁移执行成功")
}
```

## 迁移内容

### ChatSession 表

会话表，存储用户的聊天会话信息。

**字段**:

- `id`: 会话ID (UUID)
- `user_id`: 用户ID (UUID)
- `title`: 会话标题
- `model_name`: 使用的模型名称
- `system_prompt`: 系统提示词
- `temperature`: 温度参数
- `top_p`: TopP参数
- `created_by`: 创建者ID
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `last_message_id`: 最后一条消息ID
- `message_count`: 消息数量
- `is_pinned`: 是否置顶
- `is_archived`: 是否归档
- `is_deleted`: 是否删除（软删除）
- `meta`: 元数据 (JSONB)

**索引**:

- `idx_user_sessions`: (user_id, updated_at DESC) - 用户会话列表查询
- `idx_pinned`: (is_pinned, updated_at DESC) - 置顶会话排序
- `idx_archived`: (is_archived) - 归档状态过滤
- `idx_deleted`: (is_deleted) - 软删除过滤

### ChatMessage 表

消息表，存储会话中的所有消息。

**字段**:

- `id`: 消息ID (UUID)
- `session_id`: 会话ID (UUID)
- `role`: 角色 (user, assistant, system, function)
- `content`: 消息内容
- `tokens`: Token数量
- `created_at`: 创建时间
- `sequence`: 消息序列号
- `tool_calls`: 工具调用信息 (JSONB)
- `error`: 错误信息
- `parent_id`: 父消息ID
- `meta`: 元数据 (JSONB)

**索引**:

- `idx_session_messages`: (session_id, sequence ASC) - 会话消息查询
- `idx_created`: (created_at DESC) - 时间排序

**外键**:

- `session_id` -> `chat_sessions.id` (ON DELETE CASCADE)

### ChatSummary 表

摘要表，存储长会话的摘要信息。

**字段**:

- `id`: 摘要ID (UUID)
- `session_id`: 会话ID (UUID)
- `summary`: 摘要内容
- `last_message_id`: 最后一条消息ID
- `token_count`: Token数量
- `created_at`: 创建时间

**索引**:

- `idx_session_summary`: (session_id, created_at DESC) - 会话摘要查询

**外键**:

- `session_id` -> `chat_sessions.id` (ON DELETE CASCADE)

## 注意事项

1. 迁移会自动创建表和索引，如果表已存在则会跳过
2. 使用 `gen_random_uuid()` 作为UUID的默认值，需要PostgreSQL支持
3. 所有时间字段使用 `CURRENT_TIMESTAMP` 作为默认值
4. 外键设置了 `ON DELETE CASCADE`，删除会话时会自动删除相关消息和摘要
5. 使用JSONB类型存储元数据，支持高效的JSON查询
