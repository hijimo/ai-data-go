# 会话管理系统设计文档

## 概述

本文档描述了 AI 聊天系统会话管理模块的技术设计方案。该模块基于 PostgreSQL 数据库，采用分层架构设计，实现会话的完整生命周期管理、消息历史记录、多用户隔离和长会话优化等功能。

## 架构设计

### 系统分层

```
┌─────────────────────────────────────────┐
│         API Handler Layer               │  HTTP 请求处理
│  (SessionHandler, MessageHandler)       │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│         Service Layer                   │  业务逻辑处理
│  (SessionService, MessageService)       │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│         Repository Layer                │  数据访问层
│  (SessionRepo, MessageRepo, SummaryRepo)│
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│         Database Layer (PostgreSQL)     │  数据持久化
│  (chat_sessions, chat_messages, ...)   │
└─────────────────────────────────────────┘
```

### 模块职责

- **Handler 层**: 处理 HTTP 请求，参数验证，响应构建
- **Service 层**: 实现业务逻辑，事务管理，跨模块协调
- **Repository 层**: 封装数据库操作，提供 CRUD 接口
- **Database 层**: 数据持久化，使用 GORM 作为 ORM 框架

## 数据模型设计

### 1. ChatSession（会话实体）

```go
// ChatSession 会话实体
type ChatSession struct {
 ID            string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
 UserID        string         `gorm:"type:uuid;not null;index:idx_user_sessions" json:"userId"`
 Title         string         `gorm:"type:varchar(255);not null" json:"title"`
 ModelName     string         `gorm:"type:varchar(128);not null" json:"modelName"`
 SystemPrompt  string         `gorm:"type:text" json:"systemPrompt"`
 Temperature   *float64       `gorm:"type:float" json:"temperature"`
 TopP          *float64       `gorm:"type:float" json:"topP"`
 CreatedBy     string         `gorm:"type:uuid;not null" json:"createdBy"`
 CreatedAt     time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
 UpdatedAt     time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
 LastMessageID *string        `gorm:"type:uuid" json:"lastMessageId"`
 MessageCount  int            `gorm:"default:0" json:"messageCount"`
 IsPinned      bool           `gorm:"default:false;index:idx_pinned" json:"isPinned"`
 IsArchived    bool           `gorm:"default:false;index:idx_archived" json:"isArchived"`
 IsDeleted     bool           `gorm:"default:false;index:idx_deleted" json:"isDeleted"`
 Meta          datatypes.JSON `gorm:"type:jsonb" json:"meta"`
}

// TableName 指定表名
func (ChatSession) TableName() string {
 return "chat_sessions"
}
```

**索引设计**:

- `idx_user_sessions`: (user_id, updated_at DESC) - 用户会话列表查询
- `idx_pinned`: (is_pinned, updated_at DESC) - 置顶会话排序
- `idx_archived`: (is_archived) - 归档状态过滤
- `idx_deleted`: (is_deleted) - 软删除过滤

### 2. ChatMessage（消息实体）

```go
// ChatMessage 消息实体
type ChatMessage struct {
 ID        string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
 SessionID string         `gorm:"type:uuid;not null;index:idx_session_messages" json:"sessionId"`
 Role      string         `gorm:"type:varchar(32);not null" json:"role"` // user, assistant, system, function
 Content   string         `gorm:"type:text;not null" json:"content"`
 Tokens    int            `gorm:"default:0" json:"tokens"`
 CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_created" json:"createdAt"`
 Sequence  int            `gorm:"not null" json:"sequence"`
 ToolCalls datatypes.JSON `gorm:"type:jsonb" json:"toolCalls"`
 Error     string         `gorm:"type:text" json:"error"`
 ParentID  *string        `gorm:"type:uuid" json:"parentId"`
 Meta      datatypes.JSON `gorm:"type:jsonb" json:"meta"`
 
 // 关联
 Session   *ChatSession   `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChatMessage) TableName() string {
 return "chat_messages"
}
```

**索引设计**:

- `idx_session_messages`: (session_id, sequence ASC) - 会话消息查询
- `idx_created`: (created_at DESC) - 时间排序

### 3. ChatSummary（摘要实体）

```go
// ChatSummary 会话摘要实体
type ChatSummary struct {
 ID            string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
 SessionID     string    `gorm:"type:uuid;not null;index:idx_session_summary" json:"sessionId"`
 Summary       string    `gorm:"type:text;not null" json:"summary"`
 LastMessageID string    `gorm:"type:uuid;not null" json:"lastMessageId"`
 TokenCount    int       `gorm:"default:0" json:"tokenCount"`
 CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
 
 // 关联
 Session       *ChatSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChatSummary) TableName() string {
 return "chat_summaries"
}
```

**索引设计**:

- `idx_session_summary`: (session_id, created_at DESC) - 会话摘要查询

## 组件设计

### 1. Repository 层

#### SessionRepository 接口

```go
// SessionRepository 会话数据访问接口
type SessionRepository interface {
 // Create 创建会话
 Create(ctx context.Context, session *ChatSession) error
 
 // GetByID 根据ID获取会话
 GetByID(ctx context.Context, sessionID string) (*ChatSession, error)
 
 // GetByUserID 获取用户的会话列表（支持分页）
 GetByUserID(ctx context.Context, userID string, page, pageSize int, filters *SessionFilters) ([]*ChatSession, int, error)
 
 // Update 更新会话
 Update(ctx context.Context, session *ChatSession) error
 
 // UpdateFields 更新指定字段
 UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error
 
 // SoftDelete 软删除会话
 SoftDelete(ctx context.Context, sessionID string) error
 
 // Search 搜索会话
 Search(ctx context.Context, userID, keyword string, page, pageSize int) ([]*ChatSession, int, error)
 
 // IncrementMessageCount 增加消息计数
 IncrementMessageCount(ctx context.Context, sessionID string) error
 
 // UpdateLastMessage 更新最后一条消息
 UpdateLastMessage(ctx context.Context, sessionID, messageID string) error
}

// SessionFilters 会话过滤条件
type SessionFilters struct {
 IsPinned   *bool
 IsArchived *bool
 ModelName  string
}
```

#### MessageRepository 接口

```go
// MessageRepository 消息数据访问接口
type MessageRepository interface {
 // Create 创建消息
 Create(ctx context.Context, message *ChatMessage) error
 
 // GetByID 根据ID获取消息
 GetByID(ctx context.Context, messageID string) (*ChatMessage, error)
 
 // GetBySessionID 获取会话的消息列表（支持分页）
 GetBySessionID(ctx context.Context, sessionID string, page, pageSize int) ([]*ChatMessage, int, error)
 
 // GetLatestMessages 获取最新的N条消息
 GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]*ChatMessage, error)
 
 // GetNextSequence 获取下一个序列号
 GetNextSequence(ctx context.Context, sessionID string) (int, error)
 
 // CountBySessionID 统计会话消息数量
 CountBySessionID(ctx context.Context, sessionID string) (int, error)
 
 // GetMessagesAfter 获取指定消息之后的所有消息
 GetMessagesAfter(ctx context.Context, sessionID string, afterMessageID string) ([]*ChatMessage, error)
}
```

#### SummaryRepository 接口

```go
// SummaryRepository 摘要数据访问接口
type SummaryRepository interface {
 // Create 创建摘要
 Create(ctx context.Context, summary *ChatSummary) error
 
 // GetLatestBySessionID 获取会话的最新摘要
 GetLatestBySessionID(ctx context.Context, sessionID string) (*ChatSummary, error)
 
 // GetBySessionID 获取会话的所有摘要
 GetBySessionID(ctx context.Context, sessionID string) ([]*ChatSummary, error)
}
```

### 2. Service 层

#### SessionService 接口

```go
// SessionService 会话业务逻辑接口
type SessionService interface {
 // CreateSession 创建新会话
 CreateSession(ctx context.Context, req *CreateSessionRequest) (*SessionResponse, error)
 
 // GetSession 获取会话详情
 GetSession(ctx context.Context, sessionID, userID string) (*SessionResponse, error)
 
 // ListSessions 获取会话列表
 ListSessions(ctx context.Context, req *ListSessionsRequest) (*SessionListResponse, error)
 
 // UpdateSession 更新会话
 UpdateSession(ctx context.Context, req *UpdateSessionRequest) (*SessionResponse, error)
 
 // DeleteSession 删除会话
 DeleteSession(ctx context.Context, sessionID, userID string) error
 
 // SearchSessions 搜索会话
 SearchSessions(ctx context.Context, req *SearchSessionsRequest) (*SessionListResponse, error)
 
 // PinSession 置顶/取消置顶会话
 PinSession(ctx context.Context, sessionID, userID string, pinned bool) error
 
 // ArchiveSession 归档/取消归档会话
 ArchiveSession(ctx context.Context, sessionID, userID string, archived bool) error
}
```

#### MessageService 接口

```go
// MessageService 消息业务逻辑接口
type MessageService interface {
 // SendMessage 发送消息（包含AI回复）
 SendMessage(ctx context.Context, req *SendMessageRequest) (*MessageResponse, error)
 
 // GetMessages 获取消息历史
 GetMessages(ctx context.Context, req *GetMessagesRequest) (*MessageListResponse, error)
 
 // GetMessageByID 获取单条消息
 GetMessageByID(ctx context.Context, messageID, userID string) (*MessageDetailResponse, error)
 
 // AbortMessage 中止消息生成
 AbortMessage(ctx context.Context, messageID, userID string) error
}
```

#### SummaryService 接口

```go
// SummaryService 摘要业务逻辑接口
type SummaryService interface {
 // GenerateSummary 生成会话摘要
 GenerateSummary(ctx context.Context, sessionID string) (*ChatSummary, error)
 
 // GetSummary 获取会话摘要
 GetSummary(ctx context.Context, sessionID string) (*ChatSummary, error)
 
 // ShouldGenerateSummary 判断是否需要生成摘要
 ShouldGenerateSummary(ctx context.Context, sessionID string) (bool, error)
}
```

### 3. Handler 层

#### SessionHandler

```go
// SessionHandler 会话处理器
type SessionHandler struct {
 sessionService SessionService
 logger         logger.Logger
 validator      *validator.Validator
}

// 路由方法
// POST   /chat/sessions          - CreateSession
// GET    /chat/sessions          - ListSessions
// GET    /chat/sessions/:id      - GetSession
// PATCH  /chat/sessions/:id      - UpdateSession
// DELETE /chat/sessions/:id      - DeleteSession
// GET    /chat/sessions/search   - SearchSessions
// POST   /chat/sessions/:id/pin  - PinSession
// POST   /chat/sessions/:id/archive - ArchiveSession
```

#### MessageHandler

```go
// MessageHandler 消息处理器
type MessageHandler struct {
 messageService MessageService
 logger         logger.Logger
 validator      *validator.Validator
}

// 路由方法
// POST /chat/sessions/:id/messages     - SendMessage
// GET  /chat/sessions/:id/messages     - GetMessages
// GET  /chat/messages/:id              - GetMessageByID
// POST /chat/messages/:id/abort        - AbortMessage
```

## 接口设计

### 1. 创建会话

**请求**: `POST /chat/sessions`

```go
type CreateSessionRequest struct {
 Title        string   `json:"title" validate:"required,max=255"`
 ModelName    string   `json:"modelName" validate:"required,max=128"`
 SystemPrompt string   `json:"systemPrompt,omitempty"`
 Temperature  *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
 TopP         *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1"`
 Meta         map[string]interface{} `json:"meta,omitempty"`
}
```

**响应**: `ResponseData[SessionResponse]`

```go
type SessionResponse struct {
 ID            string                 `json:"id"`
 UserID        string                 `json:"userId"`
 Title         string                 `json:"title"`
 ModelName     string                 `json:"modelName"`
 SystemPrompt  string                 `json:"systemPrompt"`
 Temperature   *float64               `json:"temperature"`
 TopP          *float64               `json:"topP"`
 CreatedAt     time.Time              `json:"createdAt"`
 UpdatedAt     time.Time              `json:"updatedAt"`
 MessageCount  int                    `json:"messageCount"`
 IsPinned      bool                   `json:"isPinned"`
 IsArchived    bool                   `json:"isArchived"`
 LastMessage   *MessagePreview        `json:"lastMessage,omitempty"`
 Meta          map[string]interface{} `json:"meta,omitempty"`
}

type MessagePreview struct {
 ID        string    `json:"id"`
 Role      string    `json:"role"`
 Content   string    `json:"content"`
 CreatedAt time.Time `json:"createdAt"`
}
```

### 2. 获取会话列表

**请求**: `GET /chat/sessions?pageNo=1&pageSize=20&isPinned=true&isArchived=false`

```go
type ListSessionsRequest struct {
 PageNo     int    `json:"pageNo" validate:"required,min=1"`
 PageSize   int    `json:"pageSize" validate:"required,min=1,max=100"`
 IsPinned   *bool  `json:"isPinned,omitempty"`
 IsArchived *bool  `json:"isArchived,omitempty"`
 ModelName  string `json:"modelName,omitempty"`
}
```

**响应**: `ResponsePaginationData[[]SessionResponse]`

### 3. 发送消息

**请求**: `POST /chat/sessions/:id/messages`

```go
type SendMessageRequest struct {
 SessionID string       `json:"sessionId" validate:"required,uuid"`
 Message   string       `json:"message" validate:"required"`
 Options   *ChatOptions `json:"options,omitempty"`
}
```

**响应**: `ResponseData[MessageResponse]`

```go
type MessageResponse struct {
 MessageID     string    `json:"messageId"`
 SessionID     string    `json:"sessionId"`
 UserMessage   *Message  `json:"userMessage"`
 AIMessage     *Message  `json:"aiMessage"`
 Model         string    `json:"model"`
 Usage         *Usage    `json:"usage,omitempty"`
}

type Message struct {
 ID        string    `json:"id"`
 Role      string    `json:"role"`
 Content   string    `json:"content"`
 Sequence  int       `json:"sequence"`
 CreatedAt time.Time `json:"createdAt"`
}
```

### 4. 获取消息历史

**请求**: `GET /chat/sessions/:id/messages?pageNo=1&pageSize=50`

```go
type GetMessagesRequest struct {
 SessionID string `json:"sessionId" validate:"required,uuid"`
 PageNo    int    `json:"pageNo" validate:"required,min=1"`
 PageSize  int    `json:"pageSize" validate:"required,min=1,max=100"`
}
```

**响应**: `ResponsePaginationData[[]MessageDetailResponse]`

```go
type MessageDetailResponse struct {
 ID        string                 `json:"id"`
 SessionID string                 `json:"sessionId"`
 Role      string                 `json:"role"`
 Content   string                 `json:"content"`
 Tokens    int                    `json:"tokens"`
 Sequence  int                    `json:"sequence"`
 CreatedAt time.Time              `json:"createdAt"`
 ToolCalls map[string]interface{} `json:"toolCalls,omitempty"`
 Error     string                 `json:"error,omitempty"`
 Meta      map[string]interface{} `json:"meta,omitempty"`
}
```

### 5. 更新会话

**请求**: `PATCH /chat/sessions/:id`

```go
type UpdateSessionRequest struct {
 SessionID    string   `json:"sessionId" validate:"required,uuid"`
 Title        *string  `json:"title,omitempty" validate:"omitempty,max=255"`
 SystemPrompt *string  `json:"systemPrompt,omitempty"`
 Temperature  *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
 TopP         *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1"`
 ModelName    *string  `json:"modelName,omitempty" validate:"omitempty,max=128"`
}
```

**响应**: `ResponseData[SessionResponse]`

### 6. 删除会话

**请求**: `DELETE /chat/sessions/:id`

**响应**: `ResponseData[EmptyData]`

### 7. 中止消息生成

**请求**: `POST /chat/messages/:id/abort`

```go
type AbortMessageRequest struct {
 MessageID string `json:"messageId" validate:"required,uuid"`
}
```

**响应**: `ResponseData[EmptyData]`

## 错误处理

### 错误码定义

```go
const (
 // 会话相关错误
 CodeSessionNotFound      = 40401 // 会话不存在
 CodeSessionAccessDenied  = 40301 // 无权访问会话
 CodeSessionAlreadyExists = 40901 // 会话已存在
 
 // 消息相关错误
 CodeMessageNotFound     = 40402 // 消息不存在
 CodeMessageAccessDenied = 40302 // 无权访问消息
 CodeMessageSendFailed   = 50001 // 消息发送失败
 
 // 摘要相关错误
 CodeSummaryGenerationFailed = 50002 // 摘要生成失败
)
```

### 错误处理策略

1. **权限验证**: 所有操作都需验证 UserID 与资源所有权
2. **事务管理**: 涉及多表操作使用数据库事务
3. **并发控制**: 使用乐观锁处理并发更新
4. **错误日志**: 记录详细的错误上下文信息

## 测试策略

### 单元测试

- Repository 层: 测试 CRUD 操作，使用测试数据库
- Service 层: 测试业务逻辑，使用 mock repository
- Handler 层: 测试请求处理，使用 httptest

### 集成测试

- 端到端测试: 测试完整的会话创建到消息发送流程
- 数据库测试: 测试数据一致性和事务处理
- 并发测试: 测试高并发场景下的数据正确性

### 测试覆盖率目标

- Repository 层: ≥ 80%
- Service 层: ≥ 85%
- Handler 层: ≥ 75%

## 性能优化

### 数据库优化

1. **索引优化**: 为常用查询字段创建合适的索引
2. **查询优化**: 使用分页查询，避免全表扫描
3. **连接池**: 配置合理的数据库连接池参数
4. **预加载**: 使用 GORM 的 Preload 减少 N+1 查询

### 缓存策略

1. **会话缓存**: 缓存热点会话信息（可选）
2. **消息缓存**: 缓存最近的消息列表（可选）
3. **缓存失效**: 更新操作时主动失效相关缓存

### 长会话优化

1. **摘要生成**: 当消息数超过阈值（如 50 条）时生成摘要
2. **上下文截断**: 使用摘要 + 最近消息构建 AI 上下文
3. **异步处理**: 摘要生成使用异步任务处理

## 安全考虑

### 数据安全

1. **用户隔离**: 严格验证资源所有权
2. **SQL 注入**: 使用 GORM 参数化查询
3. **敏感数据**: SystemPrompt 等敏感字段考虑加密存储（可选）

### 访问控制

1. **身份验证**: 从请求上下文获取 UserID
2. **权限验证**: 每个操作验证用户权限
3. **审计日志**: 记录关键操作的审计日志

## 迁移策略

### 数据库迁移

使用 GORM AutoMigrate 或独立的迁移工具（如 golang-migrate）

```sql
-- 创建会话表
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    model_name VARCHAR(128) NOT NULL,
    system_prompt TEXT,
    temperature FLOAT,
    top_p FLOAT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_message_id UUID,
    message_count INT DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    meta JSONB
);

-- 创建索引
CREATE INDEX idx_user_sessions ON chat_sessions(user_id, updated_at DESC);
CREATE INDEX idx_pinned ON chat_sessions(is_pinned, updated_at DESC);
CREATE INDEX idx_archived ON chat_sessions(is_archived);
CREATE INDEX idx_deleted ON chat_sessions(is_deleted);

-- 创建消息表
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    tokens INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sequence INT NOT NULL,
    tool_calls JSONB,
    error TEXT,
    parent_id UUID,
    meta JSONB
);

-- 创建索引
CREATE INDEX idx_session_messages ON chat_messages(session_id, sequence ASC);
CREATE INDEX idx_created ON chat_messages(created_at DESC);

-- 创建摘要表
CREATE TABLE chat_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    summary TEXT NOT NULL,
    last_message_id UUID NOT NULL,
    token_count INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_session_summary ON chat_summaries(session_id, created_at DESC);
```

### 现有接口调整

1. **参数重命名**: 将 `/chat` 和 `/chat/abort` 接口的 `sessionId` 改为 `messageId`
2. **兼容性**: 考虑提供过渡期的兼容处理
3. **文档更新**: 更新 API 文档和 Swagger 注释

## 部署考虑

### 配置管理

```go
type SessionConfig struct {
 // 摘要生成阈值
 SummaryThreshold int `json:"summaryThreshold" default:"50"`
 // 默认分页大小
 DefaultPageSize int `json:"defaultPageSize" default:"20"`
 // 最大分页大小
 MaxPageSize int `json:"maxPageSize" default:"100"`
 // 会话标题最大长度
 MaxTitleLength int `json:"maxTitleLength" default:"255"`
}
```

### 监控指标

1. **业务指标**: 会话创建数、消息发送数、活跃会话数
2. **性能指标**: 接口响应时间、数据库查询时间
3. **错误指标**: 错误率、失败请求数

## 未来扩展

### 可能的功能扩展

1. **会话分享**: 支持会话分享给其他用户
2. **会话导出**: 导出会话为 Markdown/PDF 格式
3. **会话模板**: 预设会话模板和系统提示词
4. **多模态支持**: 支持图片、音频等多模态消息
5. **分支对话**: 支持从历史消息创建分支对话
6. **协作会话**: 支持多用户协作对话

### 技术优化方向

1. **读写分离**: 使用主从数据库提升查询性能
2. **分库分表**: 按用户或时间分片存储海量数据
3. **消息队列**: 使用消息队列处理异步任务
4. **全文搜索**: 集成 Elasticsearch 提升搜索能力
