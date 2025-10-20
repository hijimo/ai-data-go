# 数据库主键规范

## 主键类型要求

所有数据库表的主键必须遵循以下规范：

### 主键定义

- **类型**: 所有主键字段必须使用 `UUID` 类型
- **默认值**: 主键字段必须设置默认值为 `gen_random_uuid()`

### PostgreSQL 实现

```sql
-- 表定义示例
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### GORM 模型定义

```go
type User struct {
    ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    Username  string    `gorm:"type:varchar(255);not null" json:"username"`
    Email     string    `gorm:"type:varchar(255);not null" json:"email"`
    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
}
```

### 迁移文件示例

```go
func (m *Migration) Up() error {
    return m.db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            username VARCHAR(255) NOT NULL,
            email VARCHAR(255) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `).Error
}
```

## 实施要求

### 新表创建

- 创建任何新表时，主键字段必须命名为 `id`
- 主键类型必须为 `UUID`
- 必须设置 `DEFAULT gen_random_uuid()`

### 外键关联

- 外键字段也应使用 `UUID` 类型
- 外键字段命名建议使用 `{关联表名}_id` 格式

```sql
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Go 代码中的使用

```go
import "github.com/google/uuid"

// 创建新记录时，让数据库自动生成 UUID
user := User{
    Username: "john_doe",
    Email:    "john@example.com",
}
db.Create(&user)
// user.ID 会被数据库自动填充

// 手动生成 UUID（不推荐，除非有特殊需求）
user := User{
    ID:       uuid.New(),
    Username: "john_doe",
    Email:    "john@example.com",
}
```

## 优势说明

- **全局唯一性**: UUID 在分布式系统中保证全局唯一
- **安全性**: 不会暴露记录数量和顺序信息
- **可扩展性**: 便于数据迁移和合并
- **性能**: PostgreSQL 对 UUID 有良好的索引支持

## 注意事项

- 确保 PostgreSQL 版本支持 `gen_random_uuid()` 函数（PostgreSQL 13+）
- 如果使用较旧版本，需要启用 `pgcrypto` 扩展：`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`
- 在 Go 代码中使用 `github.com/google/uuid` 包处理 UUID 类型
