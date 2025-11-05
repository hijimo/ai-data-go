# AI 对话会话管理系统 - 完整实施方案

## 文档说明

本文档整合了架构优化建议和 Genkit Flow，提供完整的实施方案。保持现有的会话接口不变，重新实现底层的 Session 和 Context 管理。

## 目录

- [1. 架构概览](#1-架构概览)
- [2. 数据模型设计](#2-数据模型设计)
- [3. Repository 层实现](#3-repository-层实现)
- [4. Service 层实现](#4-service-层实现)
- [5. Genkit Flow 实现](#5-genkit-flow-实现)
- [6. API 层保持不变](#6-api-层保持不变)
- [7. 实施步骤](#7-实施步骤)

## 1. 架构概览

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    API Layer (保持不变)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Session API  │  │ Context API  │  │   Chat API   │         │
│  │ (现有接口)   │  │  (新增)      │  │  (增强)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                    Service Layer (重新实现)                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Enhanced Session Service                     │  │
│  │  - 保持现有接口                                           │  │
│  │  - 集成上下文管理                                         │  │
│  │  - 集成记忆管理                                           │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Context Builder│ │Memory Service│  │Summary Service│        │
│  │  (新增)      │  │  (新增)      │  │  (新增)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Token Manager │  │Query Classifier│ │Health Checker│        │
│  │  (新增)      │  │  (新增)      │  │  (新增)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                    Genkit Flow Layer (新增)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Context Flow  │  │  Chat Flow   │  │Summary Flow  │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Memory Flow   │  │Importance Flow│ │Adaptive Flow │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                    Repository Layer (扩展)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Session Repo  │  │Message Repo  │  │Summary Repo  │         │
│  │  (现有)      │  │  (现有)      │  │  (现有)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Memory Repo   │  │Context Repo  │  │State Repo    │         │
│  │  (新增)      │  │  (新增)      │  │  (新增)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                      Storage Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ PostgreSQL   │  │   pgvector   │  │    Redis     │         │
│  │  (主存储)    │  │  (向量存储)  │  │   (缓存)     │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 核心设计原则

1. **接口兼容性**：保持现有 SessionService 接口不变
2. **渐进式增强**：在现有基础上逐步添加新功能
3. **模块化设计**：每个功能模块独立，易于测试和维护
4. **性能优先**：使用缓存、批量操作、异步处理
5. **可观测性**：完整的日志、追踪、监控

## 2. 数据模型设计

### 2.1 扩展现有模型

```go
// internal/model/memory.go
package model

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/datatypes"
)

// ConversationMemory 对话记忆实体
type ConversationMemory struct {
    // 记忆ID
    ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    
    // 会话ID
    SessionID uuid.UUID `gorm:"type:uuid;not null;index:idx_session_memories" json:"sessionId"`
    
    // 租户ID（多租户隔离）
    TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_tenant_memories" json:"tenantId"`
    
    // 记忆类型：short_term, long_term, summary
    MemoryType string `gorm:"type:varchar(32);not null;index:idx_memory_type" json:"memoryType"`
    
    // 记忆内容
    Content string `gorm:"type:text;not null" json:"content"`
    
    // 向量嵌入（1536维）
    Embedding []float32 `gorm:"type:vector(1536)" json:"-"`
    
    // Token 数量
    TokenCount int `gorm:"default:0" json:"tokenCount"`
    
    // 记忆起始消息ID
    StartMsgID *uuid.UUID `gorm:"type:uuid" json:"startMsgId"`
    
    // 记忆结束消息ID
    EndMsgID *uuid.UUID `gorm:"type:uuid" json:"endMsgId"`
    
    // 重要性评分（0.0-1.0）
    Importance float32 `gorm:"type:float;default:0.5;index:idx_importance" json:"importance"`
    
    // 访问次数
    AccessCount int `gorm:"default:0" json:"accessCount"`
    
    // 最后访问时间
    LastAccessAt *time.Time `json:"lastAccessAt"`
    
    // 创建时间
    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;index:idx_created" json:"createdAt"`
    
    // 过期时间
    ExpiresAt *time.Time `gorm:"index:idx_expires" json:"expiresAt"`
    
    // 是否删除
    IsDeleted bool `gorm:"default:false;index:idx_deleted" json:"isDeleted"`
    
    // 元数据
    Meta datatypes.JSON `gorm:"type:jsonb" json:"meta"`
}

func (ConversationMemory) TableName() string {
    return "conversation_memories"
}

// ConversationContext 对话上下文实体
type ConversationContext struct {
    // 上下文ID
    ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    
    // 会话ID（唯一）
    SessionID uuid.UUID `gorm:"type:uuid;not null;unique;index:idx_session_context" json:"sessionId"`
    
    // 租户ID
    TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_tenant_contexts" json:"tenantId"`
    
    // 上下文窗口大小（消息数量）
    WindowSize int `gorm:"default:10" json:"windowSize"`
    
    // 最大 Token 数
    MaxTokens int `gorm:"default:4000" json:"maxTokens"`
    
    // 上下文策略：sliding, summary, hybrid, adaptive
    Strategy string `gorm:"type:varchar(32);default:'adaptive'" json:"strategy"`
    
    // 当前 Token 使用量
    CurrentTokens int `gorm:"default:0" json:"currentTokens"`
    
    // 最后的摘要内容
    LastSummary *string `gorm:"type:text" json:"lastSummary"`
    
    // 最后摘要的消息ID
    LastSummaryMsgID *uuid.UUID `gorm:"type:uuid" json:"lastSummaryMsgId"`
    
    // 压缩次数
    CompressionCount int `gorm:"default:0" json:"compressionCount"`
    
    // 质量评分
    QualityScore float32 `gorm:"type:float;default:0.8" json:"qualityScore"`
    
    // 创建时间
    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    
    // 更新时间
    UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    
    // 配置信息
    Config datatypes.JSON `gorm:"type:jsonb" json:"config"`
}

func (ConversationContext) TableName() string {
    return "conversation_contexts"
}

// SessionState 会话状态实体
type SessionState struct {
    // 状态ID
    ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    
    // 会话ID
    SessionID uuid.UUID `gorm:"type:uuid;not null;unique;index:idx_session_state" json:"sessionId"`
    
    // 状态：active, paused, archived, error
    Status string `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
    
    // 最后活动时间
    LastActivityAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"lastActivityAt"`
    
    // 消息数量
    MessageCount int `gorm:"default:0" json:"messageCount"`
    
    // Token 使用总量
    TokenUsageTotal int `gorm:"default:0" json:"tokenUsageTotal"`
    
    // 压缩次数
    CompressionCount int `gorm:"default:0" json:"compressionCount"`
    
    // 健康度评分（0-1）
    HealthScore float32 `gorm:"type:float;default:1.0" json:"healthScore"`
    
    // 错误次数
    ErrorCount int `gorm:"default:0" json:"errorCount"`
    
    // 最后错误信息
    LastError *string `gorm:"type:text" json:"lastError"`
    
    // 创建时间
    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    
    // 更新时间
    UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (SessionState) TableName() string {
    return "session_states"
}

// QueryClassification 查询分类结果
type QueryClassification struct {
    Type       string  `json:"type"`       // simple_qa, complex_reasoning, creative, factual
    Confidence float32 `json:"confidence"` // 置信度
    Complexity float32 `json:"complexity"` // 复杂度 0-1
    RequiresContext bool `json:"requiresContext"` // 是否需要上下文
}
```

### 2.2 数据库迁移脚本

```sql
-- migrations/006_create_conversation_memories.sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS conversation_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    memory_type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),
    token_count INTEGER DEFAULT 0,
    start_msg_id UUID,
    end_msg_id UUID,
    importance FLOAT DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_access_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false,
    meta JSONB,
    
    CONSTRAINT fk_memory_session FOREIGN KEY (session_id) 
        REFERENCES chat_sessions(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_session_memories ON conversation_memories(session_id) WHERE is_deleted = false;
CREATE INDEX idx_tenant_memories ON conversation_memories(tenant_id) WHERE is_deleted = false;
CREATE INDEX idx_memory_type ON conversation_memories(memory_type);
CREATE INDEX idx_importance ON conversation_memories(importance DESC);
CREATE INDEX idx_created ON conversation_memories(created_at DESC);
CREATE INDEX idx_expires ON conversation_memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_deleted ON conversation_memories(is_deleted);

-- 创建向量索引（IVFFlat 算法）
CREATE INDEX idx_memory_embedding ON conversation_memories 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);

-- migrations/007_create_conversation_contexts.sql
CREATE TABLE IF NOT EXISTS conversation_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    tenant_id UUID NOT NULL,
    window_size INTEGER DEFAULT 10,
    max_tokens INTEGER DEFAULT 4000,
    strategy VARCHAR(32) DEFAULT 'adaptive',
    current_tokens INTEGER DEFAULT 0,
    last_summary TEXT,
    last_summary_msg_id UUID,
    compression_count INTEGER DEFAULT 0,
    quality_score FLOAT DEFAULT 0.8,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    config JSONB,
    
    CONSTRAINT fk_context_session FOREIGN KEY (session_id) 
        REFERENCES chat_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_session_context ON conversation_contexts(session_id);
CREATE INDEX idx_tenant_contexts ON conversation_contexts(tenant_id);

-- migrations/008_create_session_states.sql
CREATE TABLE IF NOT EXISTS session_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    message_count INTEGER DEFAULT 0,
    token_usage_total INTEGER DEFAULT 0,
    compression_count INTEGER DEFAULT 0,
    health_score FLOAT DEFAULT 1.0,
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_state_session FOREIGN KEY (session_id) 
        REFERENCES chat_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_session_state ON session_states(session_id);
CREATE INDEX idx_status ON session_states(status);
CREATE INDEX idx_health_score ON session_states(health_score);
```

## 3. Repository 层实现

### 3.1 Memory Repository

```go
// internal/repository/memory_repository.go
package repository

import (
    "context"
    "fmt"
    "time"
    
    "genkit-ai-service/internal/model"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// MemoryRepository 记忆仓储接口
type MemoryRepository interface {
    // Create 创建记忆
    Create(ctx context.Context, memory *model.ConversationMemory) error
    
    // GetByID 根据ID获取记忆（带租户验证）
    GetByID(ctx context.Context, memoryID string, tenantID string) (*model.ConversationMemory, error)
    
    // ListBySession 获取会话的记忆列表（带租户验证）
    ListBySession(ctx context.Context, sessionID string, tenantID string, memoryType string, pageNo, pageSize int) ([]*model.ConversationMemory, int64, error)
    
    // VectorSearch 向量相似度搜索
    VectorSearch(ctx context.Context, sessionID string, tenantID string, embedding []float32, topK int, minScore float32) ([]*model.ConversationMemory, error)
    
    // UpdateImportance 更新重要性评分
    UpdateImportance(ctx context.Context, memoryID string, importance float32) error
    
    // UpdateAccessStats 更新访问统计
    UpdateAccessStats(ctx context.Context, memoryID string) error
    
    // Delete 删除记忆（软删除，带租户验证）
    Delete(ctx context.Context, memoryID string, tenantID string) error
    
    // BatchCreate 批量创建记忆
    BatchCreate(ctx context.Context, memories []*model.ConversationMemory) error
    
    // CleanupExpired 清理过期记忆
    CleanupExpired(ctx context.Context) (int64, error)
}

// memoryRepository 记忆仓储实现
type memoryRepository struct {
    db *gorm.DB
}

// NewMemoryRepository 创建记忆仓储实例
func NewMemoryRepository(db *gorm.DB) MemoryRepository {
    return &memoryRepository{db: db}
}

// Create 创建记忆
func (r *memoryRepository) Create(ctx context.Context, memory *model.ConversationMemory) error {
    return r.db.WithContext(ctx).Create(memory).Error
}

// GetByID 根据ID获取记忆（带租户验证）
func (r *memoryRepository) GetByID(ctx context.Context, memoryID string, tenantID string) (*model.ConversationMemory, error) {
    var memory model.ConversationMemory
    
    err := r.db.WithContext(ctx).
        Where("id = ? AND tenant_id = ? AND is_deleted = false", memoryID, tenantID).
        First(&memory).Error
    
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("记忆不存在")
        }
        return nil, err
    }
    
    return &memory, nil
}

// ListBySession 获取会话的记忆列表（带租户验证）
func (r *memoryRepository) ListBySession(
    ctx context.Context,
    sessionID string,
    tenantID string,
    memoryType string,
    pageNo, pageSize int,
) ([]*model.ConversationMemory, int64, error) {
    var memories []*model.ConversationMemory
    var total int64
    
    query := r.db.WithContext(ctx).
        Model(&model.ConversationMemory{}).
        Where("session_id = ? AND tenant_id = ? AND is_deleted = false", sessionID, tenantID)
    
    if memoryType != "" {
        query = query.Where("memory_type = ?", memoryType)
    }
    
    // 计算总数
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    // 分页查询
    offset := (pageNo - 1) * pageSize
    err := query.
        Order("importance DESC, created_at DESC").
        Limit(pageSize).
        Offset(offset).
        Find(&memories).Error
    
    if err != nil {
        return nil, 0, err
    }
    
    return memories, total, nil
}

// VectorSearch 向量相似度搜索
func (r *memoryRepository) VectorSearch(
    ctx context.Context,
    sessionID string,
    tenantID string,
    embedding []float32,
    topK int,
    minScore float32,
) ([]*model.ConversationMemory, error) {
    var memories []*model.ConversationMemory
    
    // 使用 pgvector 的余弦相似度搜索
    // <=> 是余弦距离操作符，1 - 余弦距离 = 余弦相似度
    err := r.db.WithContext(ctx).
        Raw(`
            SELECT 
                id, session_id, tenant_id, memory_type, content,
                token_count, start_msg_id, end_msg_id, importance,
                access_count, last_access_at, created_at, expires_at,
                is_deleted, meta,
                1 - (embedding <=> ?::vector) as similarity
            FROM conversation_memories
            WHERE session_id = ?::uuid 
                AND tenant_id = ?::uuid
                AND is_deleted = false
                AND (expires_at IS NULL OR expires_at > NOW())
                AND 1 - (embedding <=> ?::vector) >= ?
            ORDER BY embedding <=> ?::vector
            LIMIT ?
        `,
            embedding, sessionID, tenantID, embedding, minScore, embedding, topK,
        ).
        Scan(&memories).Error
    
    if err != nil {
        return nil, err
    }
    
    return memories, nil
}

// UpdateImportance 更新重要性评分
func (r *memoryRepository) UpdateImportance(ctx context.Context, memoryID string, importance float32) error {
    return r.db.WithContext(ctx).
        Model(&model.ConversationMemory{}).
        Where("id = ?", memoryID).
        Update("importance", importance).Error
}

// UpdateAccessStats 更新访问统计
func (r *memoryRepository) UpdateAccessStats(ctx context.Context, memoryID string) error {
    now := time.Now()
    return r.db.WithContext(ctx).
        Model(&model.ConversationMemory{}).
        Where("id = ?", memoryID).
        Updates(map[string]interface{}{
            "access_count":   gorm.Expr("access_count + 1"),
            "last_access_at": now,
        }).Error
}

// Delete 删除记忆（软删除，带租户验证）
func (r *memoryRepository) Delete(ctx context.Context, memoryID string, tenantID string) error {
    return r.db.WithContext(ctx).
        Model(&model.ConversationMemory{}).
        Where("id = ? AND tenant_id = ?", memoryID, tenantID).
        Update("is_deleted", true).Error
}

// BatchCreate 批量创建记忆
func (r *memoryRepository) BatchCreate(ctx context.Context, memories []*model.ConversationMemory) error {
    return r.db.WithContext(ctx).CreateInBatches(memories, 100).Error
}

// CleanupExpired 清理过期记忆
func (r *memoryRepository) CleanupExpired(ctx context.Context) (int64, error) {
    result := r.db.WithContext(ctx).
        Model(&model.ConversationMemory{}).
        Where("expires_at IS NOT NULL AND expires_at < NOW() AND is_deleted = false").
        Update("is_deleted", true)
    
    return result.RowsAffected, result.Error
}
```

### 3.2 Context Repository

```go
// internal/repository/context_repository.go
package repository

import (
    "context"
    "fmt"
    "time"
    
    "genkit-ai-service/internal/model"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ContextRepository 上下文仓储接口
type ContextRepository interface {
    // Create 创建上下文配置
    Create(ctx context.Context, context *model.ConversationContext) error
    
    // GetBySessionID 根据会话ID获取上下文配置
    GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error)
    
    // Update 更新上下文配置
    Update(ctx context.Context, context *model.ConversationContext) error
    
    // UpdateFields 更新指定字段
    UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error
    
    // Delete 删除上下文配置
    Delete(ctx context.Context, sessionID string) error
    
    // GetOrCreate 获取或创建上下文配置
    GetOrCreate(ctx context.Context, sessionID, tenantID string) (*model.ConversationContext, error)
}

// contextRepository 上下文仓储实现
type contextRepository struct {
    db *gorm.DB
}

// NewContextRepository 创建上下文仓储实例
func NewContextRepository(db *gorm.DB) ContextRepository {
    return &contextRepository{db: db}
}

// Create 创建上下文配置
func (r *contextRepository) Create(ctx context.Context, context *model.ConversationContext) error {
    return r.db.WithContext(ctx).Create(context).Error
}

// GetBySessionID 根据会话ID获取上下文配置
func (r *contextRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
    var context model.ConversationContext
    
    err := r.db.WithContext(ctx).
        Where("session_id = ?", sessionID).
        First(&context).Error
    
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("上下文配置不存在")
        }
        return nil, err
    }
    
    return &context, nil
}

// Update 更新上下文配置
func (r *contextRepository) Update(ctx context.Context, context *model.ConversationContext) error {
    context.UpdatedAt = time.Now()
    return r.db.WithContext(ctx).Save(context).Error
}

// UpdateFields 更新指定字段
func (r *contextRepository) UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error {
    fields["updated_at"] = time.Now()
    return r.db.WithContext(ctx).
        Model(&model.ConversationContext{}).
        Where("session_id = ?", sessionID).
        Updates(fields).Error
}

// Delete 删除上下文配置
func (r *contextRepository) Delete(ctx context.Context, sessionID string) error {
    return r.db.WithContext(ctx).
        Where("session_id = ?", sessionID).
        Delete(&model.ConversationContext{}).Error
}

// GetOrCreate 获取或创建上下文配置
func (r *contextRepository) GetOrCreate(ctx context.Context, sessionID, tenantID string) (*model.ConversationContext, error) {
    // 尝试获取现有配置
    context, err := r.GetBySessionID(ctx, sessionID)
    if err == nil {
        return context, nil
    }
    
    // 创建默认配置
    sessionUUID, _ := uuid.Parse(sessionID)
    tenantUUID, _ := uuid.Parse(tenantID)
    
    context = &model.ConversationContext{
        SessionID:     sessionUUID,
        TenantID:      tenantUUID,
        WindowSize:    10,
        MaxTokens:     4000,
        Strategy:      "adaptive",
        CurrentTokens: 0,
        QualityScore:  0.8,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }
    
    if err := r.Create(ctx, context); err != nil {
        return nil, err
    }
    
    return context, nil
}
```

### 3.3 Session State Repository

```go
// internal/repository/session_state_repository.go
package repository

import (
    "context"
    "fmt"
    "time"
    
    "genkit-ai-service/internal/model"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// SessionStateRepository 会话状态仓储接口
type SessionStateRepository interface {
    // Create 创建会话状态
    Create(ctx context.Context, state *model.SessionState) error
    
    // GetBySessionID 根据会话ID获取状态
    GetBySessionID(ctx context.Context, sessionID string) (*model.SessionState, error)
    
    // Update 更新会话状态
    Update(ctx context.Context, state *model.SessionState) error
    
    // UpdateFields 更新指定字段
    UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error
    
    // UpdateActivity 更新活动时间
    UpdateActivity(ctx context.Context, sessionID string) error
    
    // IncrementMessageCount 增加消息计数
    IncrementMessageCount(ctx context.Context, sessionID string) error
    
    // IncrementTokenUsage 增加 token 使用量
    IncrementTokenUsage(ctx context.Context, sessionID string, tokens int) error
    
    // RecordError 记录错误
    RecordError(ctx context.Context, sessionID string, errorMsg string) error
    
    // GetOrCreate 获取或创建会话状态
    GetOrCreate(ctx context.Context, sessionID string) (*model.SessionState, error)
}

// sessionStateRepository 会话状态仓储实现
type sessionStateRepository struct {
    db *gorm.DB
}

// NewSessionStateRepository 创建会话状态仓储实例
func NewSessionStateRepository(db *gorm.DB) SessionStateRepository {
    return &sessionStateRepository{db: db}
}

// Create 创建会话状态
func (r *sessionStateRepository) Create(ctx context.Context, state *model.SessionState) error {
    return r.db.WithContext(ctx).Create(state).Error
}

// GetBySessionID 根据会话ID获取状态
func (r *sessionStateRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.SessionState, error) {
    var state model.SessionState
    
    err := r.db.WithContext(ctx).
        Where("session_id = ?", sessionID).
        First(&state).Error
    
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("会话状态不存在")
        }
        return nil, err
    }
    
    return &state, nil
}

// Update 更新会话状态
func (r *sessionStateRepository) Update(ctx context.Context, state *model.SessionState) error {
    state.UpdatedAt = time.Now()
    return r.db.WithContext(ctx).Save(state).Error
}

// UpdateFields 更新指定字段
func (r *sessionStateRepository) UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error {
    fields["updated_at"] = time.Now()
    return r.db.WithContext(ctx).
        Model(&model.SessionState{}).
        Where("session_id = ?", sessionID).
        Updates(fields).Error
}

// UpdateActivity 更新活动时间
func (r *sessionStateRepository) UpdateActivity(ctx context.Context, sessionID string) error {
    return r.UpdateFields(ctx, sessionID, map[string]interface{}{
        "last_activity_at": time.Now(),
    })
}

// IncrementMessageCount 增加消息计数
func (r *sessionStateRepository) IncrementMessageCount(ctx context.Context, sessionID string) error {
    return r.db.WithContext(ctx).
        Model(&model.SessionState{}).
        Where("session_id = ?", sessionID).
        Updates(map[string]interface{}{
            "message_count":    gorm.Expr("message_count + 1"),
            "last_activity_at": time.Now(),
            "updated_at":       time.Now(),
        }).Error
}

// IncrementTokenUsage 增加 token 使用量
func (r *sessionStateRepository) IncrementTokenUsage(ctx context.Context, sessionID string, tokens int) error {
    return r.db.WithContext(ctx).
        Model(&model.SessionState{}).
        Where("session_id = ?", sessionID).
        Updates(map[string]interface{}{
            "token_usage_total": gorm.Expr("token_usage_total + ?", tokens),
            "updated_at":        time.Now(),
        }).Error
}

// RecordError 记录错误
func (r *sessionStateRepository) RecordError(ctx context.Context, sessionID string, errorMsg string) error {
    return r.db.WithContext(ctx).
        Model(&model.SessionState{}).
        Where("session_id = ?", sessionID).
        Updates(map[string]interface{}{
            "error_count":   gorm.Expr("error_count + 1"),
            "last_error":    errorMsg,
            "health_score":  gorm.Expr("GREATEST(health_score - 0.1, 0)"),
            "updated_at":    time.Now(),
        }).Error
}

// GetOrCreate 获取或创建会话状态
func (r *sessionStateRepository) GetOrCreate(ctx context.Context, sessionID string) (*model.SessionState, error) {
    // 尝试获取现有状态
    state, err := r.GetBySessionID(ctx, sessionID)
    if err == nil {
        return state, nil
    }
    
    // 创建默认状态
    sessionUUID, _ := uuid.Parse(sessionID)
    
    state = &model.SessionState{
        SessionID:      sessionUUID,
        Status:         "active",
        LastActivityAt: time.Now(),
        MessageCount:   0,
        TokenUsageTotal: 0,
        CompressionCount: 0,
        HealthScore:    1.0,
        ErrorCount:     0,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }
    
    if err := r.Create(ctx, state); err != nil {
        return nil, err
    }
    
    return state, nil
}
```

## 4. Service 层实现

### 4.1 Memory Service

```go
// internal/service/ai/memory_service.go
package ai

import (
    "context"
    "fmt"
    "time"
    
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/logger"
)

// MemoryService 记忆服务接口
type MemoryService interface {
    // StoreMemory 存储记忆
    StoreMemory(ctx context.Context, sessionID, tenantID, content string, memoryType string) (*model.ConversationMemory, error)
    
    // SearchRelevantMemories 搜索相关记忆
    SearchRelevantMemories(ctx context.Context, sessionID, tenantID, query string, topK int) ([]*model.ConversationMemory, error)
    
    // EvaluateImportance 评估记忆重要性
    EvaluateImportance(ctx context.Context, memory *model.ConversationMemory) (float32, error)
    
    // ListMemories 获取记忆列表
    ListMemories(ctx context.Context, sessionID, tenantID, memoryType string, pageNo, pageSize int) ([]*model.ConversationMemory, int64, error)
    
    // DeleteMemory 删除记忆
    DeleteMemory(ctx context.Context, memoryID, tenantID string) error
    
    // CleanupExpired 清理过期记忆
    CleanupExpired(ctx context.Context) (int64, error)
}

// memoryService 记忆服务实现
type memoryService struct {
    memoryRepo      repository.MemoryRepository
    embeddingFunc   func(text string) ([]float32, error)
    importanceFlow  *genkit.Flow // 重要性评估 Flow
    logger          logger.Logger
}

// NewMemoryService 创建记忆服务实例
func NewMemoryService(
    memoryRepo repository.MemoryRepository,
    embeddingFunc func(text string) ([]float32, error),
    log logger.Logger,
) MemoryService {
    return &memoryService{
        memoryRepo:    memoryRepo,
        embeddingFunc: embeddingFunc,
        logger:        log,
    }
}

// StoreMemory 存储记忆
func (s *memoryService) StoreMemory(
    ctx context.Context,
    sessionID, tenantID, content string,
    memoryType string,
) (*model.ConversationMemory, error) {
    // 1. 生成向量嵌入
    embedding, err := s.embeddingFunc(content)
    if err != nil {
        s.logger.ErrorContext(ctx, "生成向量嵌入失败", logger.Fields{
            "session_id": sessionID,
            "error":      err.Error(),
        })
        return nil, fmt.Errorf("生成向量嵌入失败: %w", err)
    }
    
    // 2. 创建记忆实体
    memory := &model.ConversationMemory{
        SessionID:  uuid.MustParse(sessionID),
        TenantID:   uuid.MustParse(tenantID),
        MemoryType: memoryType,
        Content:    content,
        Embedding:  embedding,
        TokenCount: s.estimateTokens(content),
        Importance: 0.5, // 默认重要性
        CreatedAt:  time.Now(),
    }
    
    // 3. 评估重要性（异步）
    go func() {
        importance, err := s.EvaluateImportance(context.Background(), memory)
        if err == nil && importance != memory.Importance {
            s.memoryRepo.UpdateImportance(context.Background(), memory.ID.String(), importance)
        }
    }()
    
    // 4. 保存到数据库
    if err := s.memoryRepo.Create(ctx, memory); err != nil {
        return nil, fmt.Errorf("保存记忆失败: %w", err)
    }
    
    s.logger.InfoContext(ctx, "记忆存储成功", logger.Fields{
        "session_id":  sessionID,
        "memory_id":   memory.ID.String(),
        "memory_type": memoryType,
        "token_count": memory.TokenCount,
    })
    
    return memory, nil
}

// SearchRelevantMemories 搜索相关记忆
func (s *memoryService) SearchRelevantMemories(
    ctx context.Context,
    sessionID, tenantID, query string,
    topK int,
) ([]*model.ConversationMemory, error) {
    // 1. 生成查询向量
    queryEmbedding, err := s.embeddingFunc(query)
    if err != nil {
        return nil, fmt.Errorf("生成查询向量失败: %w", err)
    }
    
    // 2. 向量相似度搜索
    memories, err := s.memoryRepo.VectorSearch(ctx, sessionID, tenantID, queryEmbedding, topK, 0.7)
    if err != nil {
        return nil, fmt.Errorf("向量搜索失败: %w", err)
    }
    
    // 3. 异步更新访问统计
    go func() {
        for _, mem := range memories {
            s.memoryRepo.UpdateAccessStats(context.Background(), mem.ID.String())
        }
    }()
    
    return memories, nil
}

// EvaluateImportance 评估记忆重要性
func (s *memoryService) EvaluateImportance(ctx context.Context, memory *model.ConversationMemory) (float32, error) {
    // 使用 AI 评估重要性
    // 这里可以调用 Genkit Flow 或简单的规则
    
    // 简化实现：基于规则评估
    importance := float32(0.5)
    
    // 因素1：内容长度（较长的内容可能更重要）
    if memory.TokenCount > 100 {
        importance += 0.1
    }
    
    // 因素2：访问频率
    if memory.AccessCount > 5 {
        importance += 0.1
    }
    
    // 因素3：关键词检测（简化）
    keywords := []string{"重要", "关键", "决策", "结论", "总结"}
    for _, keyword := range keywords {
        if strings.Contains(memory.Content, keyword) {
            importance += 0.1
            break
        }
    }
    
    // 限制在 0-1 范围内
    if importance > 1.0 {
        importance = 1.0
    }
    
    return importance, nil
}

// ListMemories 获取记忆列表
func (s *memoryService) ListMemories(
    ctx context.Context,
    sessionID, tenantID, memoryType string,
    pageNo, pageSize int,
) ([]*model.ConversationMemory, int64, error) {
    return s.memoryRepo.ListBySession(ctx, sessionID, tenantID, memoryType, pageNo, pageSize)
}

// DeleteMemory 删除记忆
func (s *memoryService) DeleteMemory(ctx context.Context, memoryID, tenantID string) error {
    return s.memoryRepo.Delete(ctx, memoryID, tenantID)
}

// CleanupExpired 清理过期记忆
func (s *memoryService) CleanupExpired(ctx context.Context) (int64, error) {
    return s.memoryRepo.CleanupExpired(ctx)
}

// estimateTokens 估算 token 数量
func (s *memoryService) estimateTokens(text string) int {
    // 简化实现：4个字符约等于1个token
    return len(text) / 4
}
```

### 4.2 Context Builder Service

```go
// internal/service/ai/context_builder.go
package ai

import (
    "context"
    "fmt"
    "sort"
    
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/logger"
)

// ContextBuilder 上下文构建器接口
type ContextBuilder interface {
    // BuildContext 构建对话上下文
    BuildContext(ctx context.Context, sessionID, tenantID, userQuery string) (*ContextBuildOutput, error)
    
    // OptimizeContext 优化上下文
    OptimizeContext(ctx context.Context, context *ContextBuildOutput, maxTokens int, strategy string) (*ContextBuildOutput, error)
    
    // EvaluateQuality 评估上下文质量
    EvaluateQuality(ctx context.Context, context *ContextBuildOutput) (float32, error)
}

// ContextBuildOutput 上下文构建输出
type ContextBuildOutput struct {
    SessionID        string                      `json:"sessionId"`
    Summary          *model.ChatSummary          `json:"summary,omitempty"`
    RelevantMemories []*model.ConversationMemory `json:"relevantMemories,omitempty"`
    Messages         []*model.ChatMessage        `json:"messages"`
    TotalTokens      int                         `json:"totalTokens"`
    QualityScore     float32                     `json:"qualityScore"`
}

// contextBuilder 上下文构建器实现
type contextBuilder struct {
    sessionRepo  repository.SessionRepository
    messageRepo  repository.MessageRepository
    memoryRepo   repository.MemoryRepository
    summaryRepo  repository.SummaryRepository
    contextRepo  repository.ContextRepository
    memoryService MemoryService
    tokenManager *TokenManager
    logger       logger.Logger
}

// NewContextBuilder 创建上下文构建器实例
func NewContextBuilder(
    sessionRepo repository.SessionRepository,
    messageRepo repository.MessageRepository,
    memoryRepo repository.MemoryRepository,
    summaryRepo repository.SummaryRepository,
    contextRepo repository.ContextRepository,
    memoryService MemoryService,
    tokenManager *TokenManager,
    log logger.Logger,
) ContextBuilder {
    return &contextBuilder{
        sessionRepo:   sessionRepo,
        messageRepo:   messageRepo,
        memoryRepo:    memoryRepo,
        summaryRepo:   summaryRepo,
        contextRepo:   contextRepo,
        memoryService: memoryService,
        tokenManager:  tokenManager,
        logger:        log,
    }
}

// BuildContext 构建对话上下文
func (b *contextBuilder) BuildContext(
    ctx context.Context,
    sessionID, tenantID, userQuery string,
) (*ContextBuildOutput, error) {
    startTime := time.Now()
    
    // 1. 获取或创建上下文配置
    contextConfig, err := b.contextRepo.GetOrCreate(ctx, sessionID, tenantID)
    if err != nil {
        return nil, fmt.Errorf("获取上下文配置失败: %w", err)
    }
    
    // 2. 获取短期记忆（最近的消息）
    messages, err := b.messageRepo.GetRecentMessages(ctx, sessionID, contextConfig.WindowSize)
    if err != nil {
        b.logger.WarnContext(ctx, "获取短期记忆失败", logger.Fields{
            "session_id": sessionID,
            "error":      err.Error(),
        })
        messages = []*model.ChatMessage{}
    }
    
    // 3. 获取长期记忆（相关的历史对话）
    var relevantMemories []*model.ConversationMemory
    if userQuery != "" {
        relevantMemories, err = b.memoryService.SearchRelevantMemories(ctx, sessionID, tenantID, userQuery, 5)
        if err != nil {
            b.logger.WarnContext(ctx, "获取长期记忆失败", logger.Fields{
                "session_id": sessionID,
                "error":      err.Error(),
            })
        }
    }
    
    // 4. 获取摘要记忆
    var summary *model.ChatSummary
    if contextConfig.Strategy == "summary" || contextConfig.Strategy == "hybrid" || contextConfig.Strategy == "adaptive" {
        summary, _ = b.summaryRepo.GetLatest(ctx, sessionID)
    }
    
    // 5. 组合上下文
    output := &ContextBuildOutput{
        SessionID:        sessionID,
        Summary:          summary,
        RelevantMemories: relevantMemories,
        Messages:         messages,
    }
    
    // 6. 计算总 token 数
    output.TotalTokens = b.calculateTotalTokens(output)
    
    // 7. 如果超出限制，进行优化
    if output.TotalTokens > contextConfig.MaxTokens {
        output, err = b.OptimizeContext(ctx, output, contextConfig.MaxTokens, contextConfig.Strategy)
        if err != nil {
            b.logger.WarnContext(ctx, "上下文优化失败", logger.Fields{
                "session_id": sessionID,
                "error":      err.Error(),
            })
        }
    }
    
    // 8. 评估质量
    qualityScore, _ := b.EvaluateQuality(ctx, output)
    output.QualityScore = qualityScore
    
    duration := time.Since(startTime)
    b.logger.InfoContext(ctx, "上下文构建完成", logger.Fields{
        "session_id":   sessionID,
        "total_tokens": output.TotalTokens,
        "quality":      qualityScore,
        "duration":     duration.String(),
    })
    
    return output, nil
}

// OptimizeContext 优化上下文
func (b *contextBuilder) OptimizeContext(
    ctx context.Context,
    context *ContextBuildOutput,
    maxTokens int,
    strategy string,
) (*ContextBuildOutput, error) {
    switch strategy {
    case "aggressive":
        return b.optimizeAggressive(context, maxTokens)
    case "conservative":
        return b.optimizeConservative(context, maxTokens)
    case "adaptive":
        return b.optimizeAdaptive(context, maxTokens)
    default: // balanced
        return b.optimizeBalanced(context, maxTokens)
    }
}

// optimizeAggressive 激进优化策略
func (b *contextBuilder) optimizeAggressive(context *ContextBuildOutput, maxTokens int) (*ContextBuildOutput, error) {
    optimized := &ContextBuildOutput{
        SessionID: context.SessionID,
        Messages:  []*model.ChatMessage{},
    }
    
    currentTokens := 0
    
    // 1. 移除所有相关记忆
    // 2. 移除摘要
    // 3. 只保留最近的消息
    
    for i := len(context.Messages) - 1; i >= 0; i-- {
        msg := context.Messages[i]
        if currentTokens+msg.Tokens <= maxTokens {
            optimized.Messages = append([]*model.ChatMessage{msg}, optimized.Messages...)
            currentTokens += msg.Tokens
        } else {
            break
        }
    }
    
    optimized.TotalTokens = currentTokens
    return optimized, nil
}

// optimizeBalanced 平衡优化策略
func (b *contextBuilder) optimizeBalanced(context *ContextBuildOutput, maxTokens int) (*ContextBuildOutput, error) {
    optimized := &ContextBuildOutput{
        SessionID: context.SessionID,
        Summary:   context.Summary,
        Messages:  []*model.ChatMessage{},
    }
    
    currentTokens := 0
    
    // 1. 保留摘要
    if context.Summary != nil {
        currentTokens += context.Summary.TokenCount
    }
    
    // 2. 保留前3个最重要的记忆
    if len(context.RelevantMemories) > 0 {
        sortedMemories := b.sortMemoriesByImportance(context.RelevantMemories)
        keepCount := 3
        if len(sortedMemories) < keepCount {
            keepCount = len(sortedMemories)
        }
        
        for i := 0; i < keepCount; i++ {
            if currentTokens+sortedMemories[i].TokenCount <= maxTokens {
                optimized.RelevantMemories = append(optimized.RelevantMemories, sortedMemories[i])
                currentTokens += sortedMemories[i].TokenCount
            }
        }
    }
    
    // 3. 保留最近5条消息
    keepMsgCount := 5
    if len(context.Messages) < keepMsgCount {
        keepMsgCount = len(context.Messages)
    }
    
    for i := len(context.Messages) - keepMsgCount; i < len(context.Messages); i++ {
        if currentTokens+context.Messages[i].Tokens <= maxTokens {
            optimized.Messages = append(optimized.Messages, context.Messages[i])
            currentTokens += context.Messages[i].Tokens
        }
    }
    
    optimized.TotalTokens = currentTokens
    return optimized, nil
}

// optimizeConservative 保守优化策略
func (b *contextBuilder) optimizeConservative(context *ContextBuildOutput, maxTokens int) (*ContextBuildOutput, error) {
    optimized := &ContextBuildOutput{
        SessionID:        context.SessionID,
        Summary:          context.Summary,
        RelevantMemories: []*model.ConversationMemory{},
        Messages:         context.Messages,
    }
    
    currentTokens := b.calculateTotalTokens(context)
    
    // 只移除低重要性的记忆
    for _, mem := range context.RelevantMemories {
        if mem.Importance >= 0.5 || currentTokens <= maxTokens {
            optimized.RelevantMemories = append(optimized.RelevantMemories, mem)
        } else {
            currentTokens -= mem.TokenCount
        }
    }
    
    optimized.TotalTokens = currentTokens
    return optimized, nil
}

// optimizeAdaptive 自适应优化策略
func (b *contextBuilder) optimizeAdaptive(context *ContextBuildOutput, maxTokens int) (*ContextBuildOutput, error) {
    // 根据当前 token 使用情况动态选择策略
    overageRatio := float64(context.TotalTokens) / float64(maxTokens)
    
    if overageRatio > 1.5 {
        return b.optimizeAggressive(context, maxTokens)
    } else if overageRatio > 1.2 {
        return b.optimizeBalanced(context, maxTokens)
    } else {
        return b.optimizeConservative(context, maxTokens)
    }
}

// EvaluateQuality 评估上下文质量
func (b *contextBuilder) EvaluateQuality(ctx context.Context, context *ContextBuildOutput) (float32, error) {
    score := float32(0.5)
    
    // 因素1：是否有摘要（+0.1）
    if context.Summary != nil {
        score += 0.1
    }
    
    // 因素2：相关记忆数量（+0.1）
    if len(context.RelevantMemories) > 0 {
        score += 0.1
    }
    
    // 因素3：消息数量适中（+0.2）
    if len(context.Messages) >= 3 && len(context.Messages) <= 10 {
        score += 0.2
    }
    
    // 因素4：token 使用率（+0.1）
    // 理想使用率：60-80%
    // 这里需要知道 maxTokens，暂时省略
    
    return score, nil
}

// calculateTotalTokens 计算总 token 数
func (b *contextBuilder) calculateTotalTokens(context *ContextBuildOutput) int {
    total := 0
    
    if context.Summary != nil {
        total += context.Summary.TokenCount
    }
    
    for _, mem := range context.RelevantMemories {
        total += mem.TokenCount
    }
    
    for _, msg := range context.Messages {
        total += msg.Tokens
    }
    
    return total
}

// sortMemoriesByImportance 按重要性排序记忆
func (b *contextBuilder) sortMemoriesByImportance(memories []*model.ConversationMemory) []*model.ConversationMemory {
    sorted := make([]*model.ConversationMemory, len(memories))
    copy(sorted, memories)
    
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].Importance > sorted[j].Importance
    })
    
    return sorted
}
```

### 4.3 Token Manager

```go
// internal/service/ai/token_manager.go
package ai

import (
    "genkit-ai-service/internal/model"
)

// TokenManager Token 管理器
type TokenManager struct {
    maxContextTokens int
    reservedTokens   int
    compressionRatio float64
    modelLimits      map[string]int
}

// NewTokenManager 创建 Token 管理器
func NewTokenManager(maxContextTokens, reservedTokens int, compressionRatio float64) *TokenManager {
    if maxContextTokens <= 0 {
        maxContextTokens = 4000
    }
    if reservedTokens <= 0 {
        reservedTokens = 1000
    }
    if compressionRatio <= 0 || compressionRatio >= 1 {
        compressionRatio = 0.8
    }
    
    return &TokenManager{
        maxContextTokens: maxContextTokens,
        reservedTokens:   reservedTokens,
        compressionRatio: compressionRatio,
        modelLimits:      initModelLimits(),
    }
}

// EstimateTokens 估算文本的 token 数量
func (tm *TokenManager) EstimateTokens(text string) int {
    // 简化实现：4个字符约等于1个token
    // 实际应使用 tiktoken 库
    return len(text) / 4
}

// CalculateBudget 计算 token 预算
func (tm *TokenManager) CalculateBudget(modelName string, userMessageTokens int) TokenBudget {
    modelMaxTokens := tm.GetModelMaxTokens(modelName)
    reservedForOutput := int(float64(modelMaxTokens) * 0.35)
    contextBudget := modelMaxTokens - reservedForOutput - userMessageTokens
    
    var strategy string
    if contextBudget < 1000 {
        strategy = "aggressive"
    } else if contextBudget < 3000 {
        strategy = "balanced"
    } else {
        strategy = "conservative"
    }
    
    return TokenBudget{
        ModelMaxTokens:    modelMaxTokens,
        ReservedForOutput: reservedForOutput,
        ContextBudget:     contextBudget,
        UserMessageTokens: userMessageTokens,
        Strategy:          strategy,
    }
}

// ShouldCompress 判断是否需要压缩
func (tm *TokenManager) ShouldCompress(currentTokens, maxTokens int) bool {
    if maxTokens <= 0 {
        maxTokens = tm.maxContextTokens
    }
    
    threshold := float64(maxTokens) * tm.compressionRatio
    return float64(currentTokens) > threshold
}

// GetModelMaxTokens 获取模型的最大 token 限制
func (tm *TokenManager) GetModelMaxTokens(modelName string) int {
    if limit, ok := tm.modelLimits[modelName]; ok {
        return limit
    }
    return tm.maxContextTokens
}

// initModelLimits 初始化模型限制
func initModelLimits() map[string]int {
    return map[string]int{
        "gpt-4":              8192,
        "gpt-4-32k":          32768,
        "gpt-4-turbo":        128000,
        "gpt-4o":             128000,
        "gpt-3.5-turbo":      4096,
        "gpt-3.5-turbo-16k":  16384,
        "gemini-pro":         32768,
        "gemini-1.5-pro":     1048576,
        "gemini-1.5-flash":   1048576,
        "gemini-2.5-flash":   1048576,
        "claude-2":           100000,
        "claude-3-opus":      200000,
        "claude-3-sonnet":    200000,
    }
}

// TokenBudget Token 预算
type TokenBudget struct {
    ModelMaxTokens    int    `json:"modelMaxTokens"`
    ReservedForOutput int    `json:"reservedForOutput"`
    ContextBudget     int    `json:"contextBudget"`
    UserMessageTokens int    `json:"userMessageTokens"`
    Strategy          string `json:"strategy"`
}
```

### 4.4 Query Classifier

```go
// internal/service/ai/query_classifier.go
package ai

import (
    "context"
    "strings"
    
    "genkit-ai-service/internal/model"
)

// QueryClassifier 查询分类器接口
type QueryClassifier interface {
    // Classify 分类查询
    Classify(ctx context.Context, query string) (*model.QueryClassification, error)
}

// queryClassifier 查询分类器实现
type queryClassifier struct {
    // 可以集成 AI 模型进行分类
}

// NewQueryClassifier 创建查询分类器实例
func NewQueryClassifier() QueryClassifier {
    return &queryClassifier{}
}

// Classify 分类查询
func (qc *queryClassifier) Classify(ctx context.Context, query string) (*model.QueryClassification, error) {
    // 简化实现：基于规则分类
    // 实际应使用 AI 模型
    
    queryLower := strings.ToLower(query)
    queryLen := len(query)
    
    // 默认分类
    classification := &model.QueryClassification{
        Type:            "simple_qa",
        Confidence:      0.7,
        Complexity:      0.3,
        RequiresContext: false,
    }
    
    // 规则1：长查询通常更复杂
    if queryLen > 200 {
        classification.Complexity = 0.8
        classification.Type = "complex_reasoning"
        classification.RequiresContext = true
    }
    
    // 规则2：包含特定关键词
    creativeKeywords := []string{"创作", "写", "生成", "设计", "想象"}
    for _, keyword := range creativeKeywords {
        if strings.Contains(queryLower, keyword) {
            classification.Type = "creative"
            classification.Complexity = 0.6
            classification.RequiresContext = true
            break
        }
    }
    
    // 规则3：事实查询
    factualKeywords := []string{"什么是", "如何", "为什么", "哪里", "谁"}
    for _, keyword := range factualKeywords {
        if strings.Contains(queryLower, keyword) {
            classification.Type = "factual"
            classification.Complexity = 0.4
            classification.RequiresContext = false
            break
        }
    }
    
    // 规则4：复杂推理
    reasoningKeywords := []string{"分析", "比较", "评估", "解释", "推理"}
    for _, keyword := range reasoningKeywords {
        if strings.Contains(queryLower, keyword) {
            classification.Type = "complex_reasoning"
            classification.Complexity = 0.9
            classification.RequiresContext = true
            break
        }
    }
    
    return classification, nil
}
```

### 4.5 Session Health Checker

```go
// internal/service/ai/session_health_checker.go
package ai

import (
    "context"
    "time"
    
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/logger"
)

// SessionHealthChecker 会话健康检查器接口
type SessionHealthChecker interface {
    // CheckHealth 检查会话健康状态
    CheckHealth(ctx context.Context, sessionID string) (*HealthReport, error)
    
    // AutoHeal 自动修复
    AutoHeal(ctx context.Context, sessionID string) error
}

// HealthReport 健康报告
type HealthReport struct {
    SessionID        string    `json:"sessionId"`
    HealthScore      float32   `json:"healthScore"`
    Status           string    `json:"status"`
    Issues           []string  `json:"issues"`
    Recommendations  []string  `json:"recommendations"`
    LastCheckedAt    time.Time `json:"lastCheckedAt"`
}

// sessionHealthChecker 会话健康检查器实现
type sessionHealthChecker struct {
    stateRepo       repository.SessionStateRepository
    contextRepo     repository.ContextRepository
    tokenManager    *TokenManager
    logger          logger.Logger
}

// NewSessionHealthChecker 创建会话健康检查器实例
func NewSessionHealthChecker(
    stateRepo repository.SessionStateRepository,
    contextRepo repository.ContextRepository,
    tokenManager *TokenManager,
    log logger.Logger,
) SessionHealthChecker {
    return &sessionHealthChecker{
        stateRepo:    stateRepo,
        contextRepo:  contextRepo,
        tokenManager: tokenManager,
        logger:       log,
    }
}

// CheckHealth 检查会话健康状态
func (hc *sessionHealthChecker) CheckHealth(ctx context.Context, sessionID string) (*HealthReport, error) {
    report := &HealthReport{
        SessionID:     sessionID,
        Issues:        []string{},
        Recommendations: []string{},
        LastCheckedAt: time.Now(),
    }
    
    // 1. 获取会话状态
    state, err := hc.stateRepo.GetBySessionID(ctx, sessionID)
    if err != nil {
        report.Status = "error"
        report.HealthScore = 0
        report.Issues = append(report.Issues, "无法获取会话状态")
        return report, nil
    }
    
    report.HealthScore = state.HealthScore
    
    // 2. 检查错误率
    if state.ErrorCount > 5 {
        report.Issues = append(report.Issues, "错误次数过多")
        report.Recommendations = append(report.Recommendations, "建议检查会话配置或重新创建会话")
    }
    
    // 3. 检查 token 使用
    contextConfig, _ := hc.contextRepo.GetBySessionID(ctx, sessionID)
    if contextConfig != nil {
        if hc.tokenManager.ShouldCompress(contextConfig.CurrentTokens, contextConfig.MaxTokens) {
            report.Issues = append(report.Issues, "Token 使用量接近上限")
            report.Recommendations = append(report.Recommendations, "建议生成摘要压缩上下文")
        }
    }
    
    // 4. 检查活动时间
    if time.Since(state.LastActivityAt) > 30*time.Minute {
        report.Issues = append(report.Issues, "会话长时间未活动")
        report.Recommendations = append(report.Recommendations, "考虑归档或删除会话")
    }
    
    // 5. 确定状态
    if report.HealthScore >= 0.8 {
        report.Status = "healthy"
    } else if report.HealthScore >= 0.5 {
        report.Status = "warning"
    } else {
        report.Status = "critical"
    }
    
    return report, nil
}

// AutoHeal 自动修复
func (hc *sessionHealthChecker) AutoHeal(ctx context.Context, sessionID string) error {
    report, err := hc.CheckHealth(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // 根据问题自动修复
    for _, issue := range report.Issues {
        switch issue {
        case "Token 使用量接近上限":
            // 触发摘要生成
            hc.logger.InfoContext(ctx, "自动触发摘要生成", logger.Fields{
                "session_id": sessionID,
            })
            // 这里应该调用 SummaryService
            
        case "错误次数过多":
            // 重置错误计数
            hc.stateRepo.UpdateFields(ctx, sessionID, map[string]interface{}{
                "error_count": 0,
                "health_score": 0.8,
            })
        }
    }
    
    return nil
}
```

### 4.6 Enhanced Session Service

```go
// internal/service/session/enhanced_session_service.go
package session

import (
    "context"
    "fmt"
    "time"
    
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/service/ai"
    "genkit-ai-service/internal/service/auth"
    "genkit-ai-service/pkg/errors"
    
    "github.com/google/uuid"
)

// enhancedSessionService 增强的会话服务实现
// 保持原有接口，内部集成新功能
type enhancedSessionService struct {
    // 原有依赖
    sessionRepo repository.SessionRepository
    messageRepo repository.MessageRepository
    
    // 新增依赖
    stateRepo       repository.SessionStateRepository
    contextRepo     repository.ContextRepository
    contextBuilder  ai.ContextBuilder
    healthChecker   ai.SessionHealthChecker
}

// NewEnhancedSessionService 创建增强的会话服务实例
func NewEnhancedSessionService(
    sessionRepo repository.SessionRepository,
    messageRepo repository.MessageRepository,
    stateRepo repository.SessionStateRepository,
    contextRepo repository.ContextRepository,
    contextBuilder ai.ContextBuilder,
    healthChecker ai.SessionHealthChecker,
) SessionService {
    return &enhancedSessionService{
        sessionRepo:    sessionRepo,
        messageRepo:    messageRepo,
        stateRepo:      stateRepo,
        contextRepo:    contextRepo,
        contextBuilder: contextBuilder,
        healthChecker:  healthChecker,
    }
}

// CreateSession 创建新会话（保持接口不变，内部增强）
func (s *enhancedSessionService) CreateSession(
    ctx context.Context,
    userID string,
    req *model.CreateSessionRequest,
) (*model.SessionResponse, error) {
    userUUID, err := uuid.Parse(userID)
    if err != nil {
        return nil, errors.NewBadRequestError("用户ID格式无效")
    }
    
    // 从 Context 获取创建者信息
    createdByUUID, createdByName := auth.GetCreatorInfoFromContext(ctx)
    
    var createdBy uuid.UUID
    if createdByUUID != nil {
        createdBy = *createdByUUID
    } else {
        createdBy = userUUID
    }
    
    // 创建会话实体
    session := &model.ChatSession{
        UserID:        userUUID,
        Title:         req.Title,
        ModelName:     req.ModelName,
        SystemPrompt:  req.SystemPrompt,
        Temperature:   req.Temperature,
        TopP:          req.TopP,
        CreatedBy:     createdBy,
        CreatedByName: createdByName,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
        MessageCount:  0,
        IsPinned:      false,
        IsArchived:    false,
        IsDeleted:     false,
    }
    
    // 保存到数据库
    if err := s.sessionRepo.Create(ctx, session); err != nil {
        return nil, errors.NewInternalError(fmt.Errorf("创建会话失败: %w", err))
    }
    
    // 【新增】创建会话状态
    go func() {
        state := &model.SessionState{
            SessionID:      session.ID,
            Status:         "active",
            LastActivityAt: time.Now(),
            HealthScore:    1.0,
            CreatedAt:      time.Now(),
            UpdatedAt:      time.Now(),
        }
        s.stateRepo.Create(context.Background(), state)
    }()
    
    // 【新增】创建上下文配置
    go func() {
        // 获取租户ID
        user, _ := getUserByID(context.Background(), userID)
        if user != nil {
            contextConfig := &model.ConversationContext{
                SessionID:  session.ID,
                TenantID:   user.TenantID,
                WindowSize: 10,
                MaxTokens:  4000,
                Strategy:   "adaptive",
                CreatedAt:  time.Now(),
                UpdatedAt:  time.Now(),
            }
            s.contextRepo.Create(context.Background(), contextConfig)
        }
    }()
    
    return s.toSessionResponse(session, nil), nil
}

// GetSession 获取会话详情（保持接口不变）
func (s *enhancedSessionService) GetSession(
    ctx context.Context,
    sessionID, userID string,
) (*model.SessionResponse, error) {
    // 原有逻辑
    session, err := s.sessionRepo.GetByID(ctx, sessionID)
    if err != nil {
        return nil, errors.NewSessionNotFoundError(sessionID)
    }
    
    if session.UserID.String() != userID {
        return nil, errors.NewSessionAccessDeniedError()
    }
    
    var lastMessage *model.ChatMessage
    if session.LastMessageID != nil {
        lastMessage, _ = s.messageRepo.GetByID(ctx, session.LastMessageID.String())
    }
    
    // 【新增】更新活动时间
    go s.stateRepo.UpdateActivity(context.Background(), sessionID)
    
    // 【新增】健康检查
    go s.healthChecker.CheckHealth(context.Background(), sessionID)
    
    return s.toSessionResponse(session, lastMessage), nil
}

// 其他方法保持不变，只在内部增加状态更新...
// ListSessions, UpdateSession, DeleteSession, SearchSessions, PinSession, ArchiveSession
// 实现省略，与原有实现类似，只是增加状态更新的异步调用
```

## 5. Genkit Flow 实现

### 5.1 Flow 初始化

```go
// internal/service/ai/flows.go
package ai

import (
    "context"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/ai"
)

var (
    // 全局 Genkit 实例
    g *genkit.Genkit
    
    // Flow 实例
    BuildContextFlow      *genkit.Flow
    ChatGenerateFlow      *genkit.Flow
    SummaryGenerateFlow   *genkit.Flow
    MemorySearchFlow      *genkit.Flow
    ImportanceEvalFlow    *genkit.Flow
    QueryClassifyFlow     *genkit.Flow
    TokenEstimateFlow     *genkit.Flow
    ContextOptimizeFlow   *genkit.Flow
    SmartChatFlow         *genkit.Flow
)

// InitializeFlows 初始化所有 Flow
func InitializeFlows(ctx context.Context, genkitInstance *genkit.Genkit, deps *FlowDependencies) {
    g = genkitInstance
    
    // 初始化各个 Flow
    BuildContextFlow = defineBuildContextFlow(deps)
    ChatGenerateFlow = defineChatGenerateFlow(deps)
    SummaryGenerateFlow = defineSummaryGenerateFlow(deps)
    MemorySearchFlow = defineMemorySearchFlow(deps)
    ImportanceEvalFlow = defineImportanceEvalFlow(deps)
    QueryClassifyFlow = defineQueryClassifyFlow(deps)
    TokenEstimateFlow = defineTokenEstimateFlow(deps)
    ContextOptimizeFlow = defineContextOptimizeFlow(deps)
    SmartChatFlow = defineSmartChatFlow(deps)
}

// FlowDependencies Flow 依赖
type FlowDependencies struct {
    ContextBuilder  ContextBuilder
    MemoryService   MemoryService
    SummaryService  SummaryService
    TokenManager    *TokenManager
    QueryClassifier QueryClassifier
    MessageRepo     repository.MessageRepository
    SessionRepo     repository.SessionRepository
}
```

### 5.2 上下文构建 Flow

```go
// internal/service/ai/flow_context.go
package ai

import (
    "context"
    "fmt"
    
    "github.com/firebase/genkit/go/genkit"
)

// ContextBuildInput 上下文构建输入
type ContextBuildInput struct {
    SessionID string `json:"sessionId"`
    TenantID  string `json:"tenantId"`
    UserQuery string `json:"userQuery"`
    MaxTokens int    `json:"maxTokens"`
}

// defineBuildContextFlow 定义上下文构建 Flow
func defineBuildContextFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "buildContextFlow",
        func(ctx context.Context, input ContextBuildInput) (*ContextBuildOutput, error) {
            // 验证输入
            if input.SessionID == "" {
                return nil, fmt.Errorf("sessionId 不能为空")
            }
            
            if input.MaxTokens <= 0 {
                input.MaxTokens = 4000
            }
            
            // 调用 ContextBuilder
            output, err := deps.ContextBuilder.BuildContext(
                ctx,
                input.SessionID,
                input.TenantID,
                input.UserQuery,
            )
            
            if err != nil {
                return nil, fmt.Errorf("构建上下文失败: %w", err)
            }
            
            // 如果超出限制，优化上下文
            if output.TotalTokens > input.MaxTokens {
                output, err = deps.ContextBuilder.OptimizeContext(
                    ctx,
                    output,
                    input.MaxTokens,
                    "adaptive",
                )
                if err != nil {
                    return nil, fmt.Errorf("优化上下文失败: %w", err)
                }
            }
            
            return output, nil
        },
    )
}
```

### 5.3 对话生成 Flow

```go
// internal/service/ai/flow_chat.go
package ai

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/ai"
)

// ChatGenerateInput 对话生成输入
type ChatGenerateInput struct {
    SessionID   string                 `json:"sessionId"`
    TenantID    string                 `json:"tenantId"`
    UserMessage string                 `json:"userMessage"`
    Context     *ContextBuildOutput    `json:"context,omitempty"`
    ModelConfig *ModelConfig           `json:"modelConfig,omitempty"`
}

// ChatGenerateOutput 对话生成输出
type ChatGenerateOutput struct {
    MessageID    string      `json:"messageId"`
    Response     string      `json:"response"`
    TokenUsed    int         `json:"tokenUsed"`
    FinishReason string      `json:"finishReason"`
    Model        string      `json:"model"`
}

// ModelConfig 模型配置
type ModelConfig struct {
    ModelName   string   `json:"modelName"`
    Temperature *float64 `json:"temperature,omitempty"`
    TopP        *float64 `json:"topP,omitempty"`
    MaxTokens   *int     `json:"maxTokens,omitempty"`
}

// defineChatGenerateFlow 定义对话生成 Flow
func defineChatGenerateFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "chatGenerateFlow",
        func(ctx context.Context, input ChatGenerateInput) (*ChatGenerateOutput, error) {
            // 1. 如果没有提供上下文，先构建上下文
            var context *ContextBuildOutput
            if input.Context == nil {
                contextInput := ContextBuildInput{
                    SessionID: input.SessionID,
                    TenantID:  input.TenantID,
                    UserQuery: input.UserMessage,
                    MaxTokens: 4000,
                }
                
                var err error
                context, err = BuildContextFlow.Run(ctx, contextInput)
                if err != nil {
                    return nil, fmt.Errorf("构建上下文失败: %w", err)
                }
            } else {
                context = input.Context
            }
            
            // 2. 构建提示词
            prompt := buildPrompt(context, input.UserMessage)
            
            // 3. 配置模型参数
            modelConfig := getModelConfig(input.ModelConfig)
            
            // 4. 调用 AI 生成
            resp, err := genkit.Generate(ctx, g,
                ai.WithPrompt(prompt),
                ai.WithConfig(modelConfig),
            )
            if err != nil {
                return nil, fmt.Errorf("生成响应失败: %w", err)
            }
            
            // 5. 保存消息到数据库（异步）
            go func() {
                saveMessages(
                    context.Background(),
                    input.SessionID,
                    input.UserMessage,
                    resp.Text(),
                    deps.MessageRepo,
                )
            }()
            
            // 6. 异步处理：生成向量、更新摘要
            go processMessageAsync(input.SessionID, resp.Text(), deps)
            
            return &ChatGenerateOutput{
                MessageID:    generateMessageID(),
                Response:     resp.Text(),
                TokenUsed:    int(resp.Usage().TotalTokens),
                FinishReason: string(resp.FinishReason()),
                Model:        getModelName(input.ModelConfig),
            }, nil
        },
    )
}

// buildPrompt 构建提示词
func buildPrompt(context *ContextBuildOutput, userMessage string) string {
    var prompt strings.Builder
    
    prompt.WriteString("你是一个智能助手。请基于以下上下文回答用户问题。\n\n")
    
    // 添加摘要
    if context.Summary != nil {
        prompt.WriteString("对话摘要：\n")
        prompt.WriteString(context.Summary.Summary)
        prompt.WriteString("\n\n")
    }
    
    // 添加相关记忆
    if len(context.RelevantMemories) > 0 {
        prompt.WriteString("相关历史信息：\n")
        for _, mem := range context.RelevantMemories {
            prompt.WriteString(fmt.Sprintf("- %s\n", mem.Content))
        }
        prompt.WriteString("\n")
    }
    
    // 添加最近的对话
    if len(context.Messages) > 0 {
        prompt.WriteString("最近的对话：\n")
        for _, msg := range context.Messages {
            prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
        }
        prompt.WriteString("\n")
    }
    
    // 添加用户消息
    prompt.WriteString(fmt.Sprintf("用户: %s\n", userMessage))
    prompt.WriteString("助手: ")
    
    return prompt.String()
}

// getModelConfig 获取模型配置
func getModelConfig(config *ModelConfig) ai.GenerateConfig {
    if config == nil {
        return ai.GenerateConfig{
            Temperature: 0.7,
            TopP:        0.9,
            MaxTokens:   2000,
        }
    }
    
    genConfig := ai.GenerateConfig{}
    if config.Temperature != nil {
        genConfig.Temperature = *config.Temperature
    }
    if config.TopP != nil {
        genConfig.TopP = *config.TopP
    }
    if config.MaxTokens != nil {
        genConfig.MaxTokens = *config.MaxTokens
    }
    
    return genConfig
}
```

### 5.4 智能对话 Flow（集成所有功能）

```go
// internal/service/ai/flow_smart_chat.go
package ai

import (
    "context"
    "fmt"
    
    "github.com/firebase/genkit/go/genkit"
)

// SmartChatInput 智能对话输入
type SmartChatInput struct {
    SessionID   string       `json:"sessionId"`
    TenantID    string       `json:"tenantId"`
    UserMessage string       `json:"userMessage"`
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
}

// SmartChatOutput 智能对话输出
type SmartChatOutput struct {
    Response        string             `json:"response"`
    MessageID       string             `json:"messageId"`
    TokenUsage      TokenUsageInfo     `json:"tokenUsage"`
    QueryType       string             `json:"queryType"`
    ContextStrategy string             `json:"contextStrategy"`
    OptimizationLog *OptimizationLog   `json:"optimizationLog,omitempty"`
}

// TokenUsageInfo Token 使用信息
type TokenUsageInfo struct {
    PromptTokens     int `json:"promptTokens"`
    CompletionTokens int `json:"completionTokens"`
    TotalTokens      int `json:"totalTokens"`
}

// OptimizationLog 优化日志
type OptimizationLog struct {
    OriginalTokens  int      `json:"originalTokens"`
    OptimizedTokens int      `json:"optimizedTokens"`
    Strategy        string   `json:"strategy"`
    RemovedItems    []string `json:"removedItems"`
}

// defineSmartChatFlow 定义智能对话 Flow
func defineSmartChatFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "smartChatFlow",
        func(ctx context.Context, input SmartChatInput) (*SmartChatOutput, error) {
            modelName := "gemini-1.5-flash"
            if input.ModelConfig != nil && input.ModelConfig.ModelName != "" {
                modelName = input.ModelConfig.ModelName
            }
            
            // 1. 查询分类
            classification, err := QueryClassifyFlow.Run(ctx, input.UserMessage)
            if err != nil {
                // 分类失败不影响主流程
                classification = &model.QueryClassification{
                    Type:       "simple_qa",
                    Complexity: 0.5,
                }
            }
            
            // 2. 计算 Token 预算
            userMsgTokens := deps.TokenManager.EstimateTokens(input.UserMessage)
            budget := deps.TokenManager.CalculateBudget(modelName, userMsgTokens)
            
            // 3. 根据查询类型调整上下文策略
            contextStrategy := budget.Strategy
            if classification.Type == "simple_qa" {
                contextStrategy = "aggressive" // 简单问答用更少上下文
            } else if classification.Type == "complex_reasoning" {
                contextStrategy = "conservative" // 复杂推理需要更多上下文
            }
            
            // 4. 构建上下文
            contextInput := ContextBuildInput{
                SessionID: input.SessionID,
                TenantID:  input.TenantID,
                UserQuery: input.UserMessage,
                MaxTokens: budget.ContextBudget,
            }
            
            context, err := BuildContextFlow.Run(ctx, contextInput)
            if err != nil {
                return nil, fmt.Errorf("构建上下文失败: %w", err)
            }
            
            originalTokens := context.TotalTokens
            
            // 5. 如果上下文超出预算，进行优化
            var optimizationLog *OptimizationLog
            if context.TotalTokens > budget.ContextBudget {
                optimizeInput := ContextOptimizeInput{
                    Context:   context,
                    MaxTokens: budget.ContextBudget,
                    Strategy:  contextStrategy,
                }
                
                optimizeResult, err := ContextOptimizeFlow.Run(ctx, optimizeInput)
                if err == nil {
                    context = optimizeResult.OptimizedContext
                    optimizationLog = &OptimizationLog{
                        OriginalTokens:  originalTokens,
                        OptimizedTokens: context.TotalTokens,
                        Strategy:        contextStrategy,
                        RemovedItems:    optimizeResult.RemovedItems,
                    }
                }
            }
            
            // 6. 生成对话
            chatInput := ChatGenerateInput{
                SessionID:   input.SessionID,
                TenantID:    input.TenantID,
                UserMessage: input.UserMessage,
                Context:     context,
                ModelConfig: input.ModelConfig,
            }
            
            chatOutput, err := ChatGenerateFlow.Run(ctx, chatInput)
            if err != nil {
                return nil, fmt.Errorf("生成对话失败: %w", err)
            }
            
            return &SmartChatOutput{
                Response:        chatOutput.Response,
                MessageID:       chatOutput.MessageID,
                TokenUsage: TokenUsageInfo{
                    PromptTokens:     context.TotalTokens + userMsgTokens,
                    CompletionTokens: chatOutput.TokenUsed,
                    TotalTokens:      context.TotalTokens + userMsgTokens + chatOutput.TokenUsed,
                },
                QueryType:       classification.Type,
                ContextStrategy: contextStrategy,
                OptimizationLog: optimizationLog,
            }, nil
        },
    )
}
```

### 5.5 其他辅助 Flow

```go
// internal/service/ai/flow_helpers.go
package ai

import (
    "context"
    
    "github.com/firebase/genkit/go/genkit"
)

// ContextOptimizeInput 上下文优化输入
type ContextOptimizeInput struct {
    Context   *ContextBuildOutput `json:"context"`
    MaxTokens int                 `json:"maxTokens"`
    Strategy  string              `json:"strategy"`
}

// ContextOptimizeOutput 上下文优化输出
type ContextOptimizeOutput struct {
    OptimizedContext *ContextBuildOutput `json:"optimizedContext"`
    OriginalTokens   int                 `json:"originalTokens"`
    OptimizedTokens  int                 `json:"optimizedTokens"`
    RemovedItems     []string            `json:"removedItems"`
}

// defineContextOptimizeFlow 定义上下文优化 Flow
func defineContextOptimizeFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "contextOptimizeFlow",
        func(ctx context.Context, input ContextOptimizeInput) (*ContextOptimizeOutput, error) {
            originalTokens := input.Context.TotalTokens
            
            optimized, err := deps.ContextBuilder.OptimizeContext(
                ctx,
                input.Context,
                input.MaxTokens,
                input.Strategy,
            )
            
            if err != nil {
                return nil, err
            }
            
            return &ContextOptimizeOutput{
                OptimizedContext: optimized,
                OriginalTokens:   originalTokens,
                OptimizedTokens:  optimized.TotalTokens,
                RemovedItems:     []string{}, // 可以记录被移除的项目
            }, nil
        },
    )
}

// defineQueryClassifyFlow 定义查询分类 Flow
func defineQueryClassifyFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "queryClassifyFlow",
        func(ctx context.Context, query string) (*model.QueryClassification, error) {
            return deps.QueryClassifier.Classify(ctx, query)
        },
    )
}

// defineTokenEstimateFlow 定义 Token 估算 Flow
func defineTokenEstimateFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "tokenEstimateFlow",
        func(ctx context.Context, text string) (int, error) {
            return deps.TokenManager.EstimateTokens(text), nil
        },
    )
}

// defineMemorySearchFlow 定义记忆搜索 Flow
func defineMemorySearchFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "memorySearchFlow",
        func(ctx context.Context, input MemorySearchInput) ([]*model.ConversationMemory, error) {
            return deps.MemoryService.SearchRelevantMemories(
                ctx,
                input.SessionID,
                input.TenantID,
                input.Query,
                input.TopK,
            )
        },
    )
}

// MemorySearchInput 记忆搜索输入
type MemorySearchInput struct {
    SessionID string `json:"sessionId"`
    TenantID  string `json:"tenantId"`
    Query     string `json:"query"`
    TopK      int    `json:"topK"`
}

// defineSummaryGenerateFlow 定义摘要生成 Flow
func defineSummaryGenerateFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "summaryGenerateFlow",
        func(ctx context.Context, input SummaryGenerateInput) (*model.ChatSummary, error) {
            return deps.SummaryService.GenerateSummary(ctx, input.SessionID)
        },
    )
}

// SummaryGenerateInput 摘要生成输入
type SummaryGenerateInput struct {
    SessionID string `json:"sessionId"`
}

// defineImportanceEvalFlow 定义重要性评估 Flow
func defineImportanceEvalFlow(deps *FlowDependencies) *genkit.Flow {
    return genkit.DefineFlow(g, "importanceEvalFlow",
        func(ctx context.Context, memory *model.ConversationMemory) (float32, error) {
            return deps.MemoryService.EvaluateImportance(ctx, memory)
        },
    )
}
```

## 6. API 层保持不变

### 6.1 集成新服务到现有 Handler

```go
// internal/api/handler/session_handler.go
// 保持现有接口不变，只需替换服务实例

package handler

import (
    "genkit-ai-service/internal/service/session"
    // ... 其他导入
)

// SessionHandler 会话处理器
type SessionHandler struct {
    // 使用增强的服务，但接口相同
    sessionService session.SessionService
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(sessionService session.SessionService) *SessionHandler {
    return &SessionHandler{
        sessionService: sessionService,
    }
}

// 所有现有的 Handler 方法保持不变
// CreateSession, GetSession, ListSessions, UpdateSession, DeleteSession 等
// 实现完全不需要修改，因为接口没有变化
```

### 6.2 新增上下文管理 Handler

```go
// internal/api/handler/context_handler.go
package handler

import (
    "net/http"
    "strconv"
    
    "genkit-ai-service/internal/service/ai"
    "genkit-ai-service/pkg/response"
    
    "github.com/gin-gonic/gin"
)

// ContextHandler 上下文处理器
type ContextHandler struct {
    contextBuilder ai.ContextBuilder
}

// NewContextHandler 创建上下文处理器
func NewContextHandler(contextBuilder ai.ContextBuilder) *ContextHandler {
    return &ContextHandler{
        contextBuilder: contextBuilder,
    }
}

// GetContext 获取会话上下文
// @Summary 获取会话上下文
// @Tags 上下文管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Param query query string false "用户查询"
// @Success 200 {object} response.ResponseData[ai.ContextBuildOutput]
// @Router /api/v1/sessions/{sessionId}/context [get]
func (h *ContextHandler) GetContext(c *gin.Context) {
    sessionID := c.Param("sessionId")
    userQuery := c.Query("query")
    
    // 获取租户ID（从JWT中）
    tenantID := getTenantIDFromContext(c)
    
    // 调用 BuildContextFlow
    input := ai.ContextBuildInput{
        SessionID: sessionID,
        TenantID:  tenantID,
        UserQuery: userQuery,
        MaxTokens: 4000,
    }
    
    output, err := ai.BuildContextFlow.Run(c.Request.Context(), input)
    if err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, output)
}
```

### 6.3 路由注册

```go
// internal/api/routes/routes.go
package routes

import (
    "genkit-ai-service/internal/api/handler"
    "genkit-ai-service/internal/api/middleware"
    
    "github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(
    r *gin.Engine,
    sessionHandler *handler.SessionHandler,
    contextHandler *handler.ContextHandler,
    // ... 其他 handler
) {
    api := r.Group("/api/v1")
    api.Use(middleware.RequireAuth())
    
    // 会话管理路由（保持不变）
    sessions := api.Group("/sessions")
    {
        sessions.POST("", sessionHandler.CreateSession)
        sessions.GET("/:sessionId", sessionHandler.GetSession)
        sessions.PUT("/:sessionId", sessionHandler.UpdateSession)
        sessions.DELETE("/:sessionId", sessionHandler.DeleteSession)
        sessions.GET("", sessionHandler.ListSessions)
        
        // 新增：上下文管理
        sessions.GET("/:sessionId/context", contextHandler.GetContext)
        
        // 新增：对话接口（使用 SmartChatFlow）
        sessions.POST("/:sessionId/chat", sessionHandler.Chat)
        sessions.POST("/:sessionId/chat/stream", sessionHandler.ChatStream)
    }
}
```

## 7. 实施步骤

### 7.1 阶段 1：基础设施（第 1 周）

**目标**：搭建基础数据模型和 Repository 层

**任务清单**：
- [ ] 创建数据库迁移脚本
  - `006_create_conversation_memories.sql`
  - `007_create_conversation_contexts.sql`
  - `008_create_session_states.sql`
- [ ] 执行数据库迁移
- [ ] 创建数据模型
  - `internal/model/memory.go`
  - 扩展 `internal/model/session.go`
- [ ] 实现 Repository 层
  - `internal/repository/memory_repository.go`
  - `internal/repository/context_repository.go`
  - `internal/repository/session_state_repository.go`
- [ ] 编写 Repository 单元测试

**验收标准**：
- 所有数据库表创建成功
- Repository 测试通过率 100%
- 可以正常 CRUD 操作

### 7.2 阶段 2：核心服务（第 2 周）

**目标**：实现核心业务逻辑服务

**任务清单**：
- [ ] 实现 MemoryService
  - 向量嵌入集成
  - 记忆存储和检索
- [ ] 实现 ContextBuilder
  - 上下文构建逻辑
  - 优化策略
- [ ] 实现 TokenManager
  - Token 估算
  - 预算管理
- [ ] 实现 QueryClassifier
  - 查询分类逻辑
- [ ] 实现 SessionHealthChecker
  - 健康检查
  - 自动修复
- [ ] 编写服务层单元测试

**验收标准**：
- 所有服务接口实现完成
- 服务测试通过率 > 90%
- 可以独立测试每个服务

### 7.3 阶段 3：Genkit Flow 集成（第 3 周）

**目标**：将服务封装为 Genkit Flow

**任务清单**：
- [ ] 初始化 Genkit Flow 框架
- [ ] 实现所有 Flow 定义
  - BuildContextFlow
  - ChatGenerateFlow
  - SmartChatFlow
  - 其他辅助 Flow
- [ ] Flow 集成测试
- [ ] 使用 Genkit Developer UI 测试

**验收标准**：
- 所有 Flow 可以在 Developer UI 中运行
- Flow 测试通过率 > 95%
- 追踪功能正常工作

### 7.4 阶段 4：服务集成（第 4 周）

**目标**：集成到现有系统

**任务清单**：
- [ ] 实现 EnhancedSessionService
- [ ] 替换现有 SessionService 实例
- [ ] 新增 ContextHandler
- [ ] 更新路由配置
- [ ] 集成测试
- [ ] API 文档更新

**验收标准**：
- 现有 API 接口保持兼容
- 新增接口正常工作
- 集成测试通过率 > 90%

### 7.5 阶段 5：性能优化和监控（第 5 周）

**目标**：优化性能，完善监控

**任务清单**：
- [ ] 添加 Redis 缓存
- [ ] 实现批量操作
- [ ] 异步处理优化
- [ ] 添加性能监控指标
- [ ] 压力测试
- [ ] 优化数据库查询

**验收标准**：
- 上下文构建响应时间 < 200ms
- 支持 1000+ 并发请求
- 缓存命中率 > 70%

### 7.6 阶段 6：文档和部署（第 6 周）

**目标**：完善文档，准备生产部署

**任务清单**：
- [ ] 完善 API 文档
- [ ] 编写使用指南
- [ ] 准备部署脚本
- [ ] 配置监控告警
- [ ] 生产环境测试
- [ ] 培训和交接

**验收标准**：
- 文档完整清晰
- 部署流程自动化
- 监控告警配置完成

## 8. 测试策略

### 8.1 单元测试

```go
// internal/service/ai/memory_service_test.go
package ai

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestMemoryService_StoreMemory(t *testing.T) {
    // 创建 mock
    mockRepo := new(MockMemoryRepository)
    mockEmbedding := func(text string) ([]float32, error) {
        return []float32{0.1, 0.2, 0.3}, nil
    }
    
    service := NewMemoryService(mockRepo, mockEmbedding, logger.Default())
    
    // 设置期望
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
    
    // 执行测试
    memory, err := service.StoreMemory(
        context.Background(),
        "session-id",
        "tenant-id",
        "test content",
        "long_term",
    )
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, memory)
    assert.Equal(t, "test content", memory.Content)
    
    mockRepo.AssertExpectations(t)
}
```

### 8.2 集成测试

```go
// internal/service/ai/integration_test.go
package ai

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestSmartChatFlow_Integration(t *testing.T) {
    // 设置测试环境
    ctx := context.Background()
    
    // 创建测试会话
    sessionID := createTestSession(t)
    defer cleanupTestSession(t, sessionID)
    
    // 测试智能对话
    input := SmartChatInput{
        SessionID:   sessionID,
        TenantID:    "test-tenant",
        UserMessage: "介绍一下人工智能",
    }
    
    output, err := SmartChatFlow.Run(ctx, input)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotEmpty(t, output.Response)
    assert.NotEmpty(t, output.MessageID)
    assert.Greater(t, output.TokenUsage.TotalTokens, 0)
}
```

## 9. 监控和告警

### 9.1 关键指标

```go
// internal/monitoring/ai_metrics.go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    // 上下文构建指标
    ContextBuildDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "context_build_duration_seconds",
            Help: "上下文构建耗时",
        },
        []string{"session_id", "strategy"},
    )
    
    // Token 使用指标
    TokenUsage = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "token_usage_total",
            Help: "Token 使用量",
        },
        []string{"session_id", "model"},
    )
    
    // 记忆检索指标
    MemorySearchDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "memory_search_duration_seconds",
            Help: "记忆检索耗时",
        },
    )
    
    // 会话健康度
    SessionHealthScore = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "session_health_score",
            Help: "会话健康度评分",
        },
        []string{"session_id"},
    )
)
```

### 9.2 告警规则

```yaml
# prometheus/alerts.yml
groups:
  - name: ai_service_alerts
    rules:
      - alert: HighContextBuildLatency
        expr: histogram_quantile(0.95, context_build_duration_seconds) > 1
        for: 5m
        annotations:
          summary: "上下文构建延迟过高"
          
      - alert: HighTokenUsage
        expr: avg(token_usage_total) > 3500
        for: 5m
        annotations:
          summary: "Token 使用量接近上限"
          
      - alert: LowSessionHealth
        expr: session_health_score < 0.5
        for: 10m
        annotations:
          summary: "会话健康度过低"
```

## 10. 总结

本实施方案提供了完整的架构设计和实施路线：

### 核心优势

1. **接口兼容**：保持现有 SessionService 接口不变
2. **渐进式增强**：在现有基础上逐步添加功能
3. **模块化设计**：每个模块独立，易于测试和维护
4. **性能优先**：使用缓存、批量操作、异步处理
5. **可观测性**：完整的日志、追踪、监控

### 关键特性

- ✅ 三层记忆架构（短期、长期、摘要）
- ✅ 智能上下文管理
- ✅ 自适应优化策略
- ✅ 查询分类和路由
- ✅ Token 预算管理
- ✅ 会话健康检查
- ✅ Genkit Flow 集成
- ✅ 多租户隔离
- ✅ 完整的监控告警

### 实施时间线

- **第 1 周**：基础设施
- **第 2 周**：核心服务
- **第 3 周**：Genkit Flow
- **第 4 周**：服务集成
- **第 5 周**：性能优化
- **第 6 周**：文档部署

---

**文档版本**: v1.0  
**最后更新**: 2025-10-29  
**维护者**: AI Platform Team
