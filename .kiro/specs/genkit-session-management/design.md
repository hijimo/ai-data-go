# Genkit 会话管理模块设计文档

## 概述

本文档描述了基于 Google Genkit Go SDK 的会话管理模块的技术设计。该模块实现三层记忆架构（短期、长期、摘要），提供智能上下文管理、Token 优化和完整的多租户隔离能力。

### 设计目标

1. **模块化设计**：使用 Genkit Flow 实现可组合的功能模块
2. **类型安全**：利用 Go 泛型和 Genkit 的类型系统确保类型安全
3. **高性能**：通过缓存、向量索引和异步处理优化性能
4. **可扩展性**：支持水平扩展和功能扩展
5. **多租户隔离**：严格的数据隔离和权限控制
6. **可观测性**：完整的监控、日志和追踪能力

### 技术栈

- **语言**：Go 1.21+
- **AI 框架**：Google Genkit Go SDK
- **数据库**：PostgreSQL 15+
- **向量数据库**：Qdrant（在线服务）
- **缓存**：Redis 7+
- **Web 框架**：Gin
- **ORM**：GORM
- **监控**：Prometheus + Grafana
- **追踪**：OpenTelemetry + Jaeger
- **日志**：结构化日志（JSON 格式）

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      API Layer (Gin)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Session API  │  │ Context API  │  │   Chat API   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Genkit Flow Layer                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Context Build │  │  Chat Gen    │  │Summary Gen   │      │
│  │    Flow      │  │    Flow      │  │    Flow      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Memory Search │  │Query Classify│  │Token Optimize│      │
│  │    Flow      │  │    Flow      │  │    Flow      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Context Svc   │  │ Memory Svc   │  │ Summary Svc  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Token Mgr    │  │ Vector Svc   │  │  Cache Svc   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Session Repo  │  │Message Repo  │  │Memory Repo   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ PostgreSQL   │  │   Qdrant     │  │    Redis     │      │
│  │  + UUID PK   │  │  (Vector DB) │  │   (Cache)    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 分层职责

#### API Layer

- 接收 HTTP 请求
- 参数验证和绑定
- JWT 认证和权限验证
- 调用 Genkit Flow
- 返回标准响应格式

#### Genkit Flow Layer

- 定义可组合的工作流
- 实现业务逻辑编排
- 提供类型安全的输入输出
- 支持 Flow 之间的组合

#### Service Layer

- 实现具体的业务逻辑
- 协调多个 Repository
- 处理事务和缓存
- 实现降级和熔断策略

#### Repository Layer

- 数据访问抽象
- 实现 CRUD 操作
- 应用租户过滤
- 处理软删除

#### Storage Layer

- PostgreSQL：持久化存储（元数据）
- Qdrant：向量数据库（向量存储和检索）
- Redis：缓存和会话

## 数据模型

### 数据库表设计

#### conversation_memories 表

存储长期记忆的元数据。向量数据存储在 Qdrant 中。

```sql
CREATE TABLE conversation_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL,
    memory_type VARCHAR(50) NOT NULL, -- 'short_term', 'long_term', 'summary'
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    importance FLOAT NOT NULL DEFAULT 0.5, -- 0-1
    access_count INTEGER NOT NULL DEFAULT 0,
    last_access_at TIMESTAMP,
    metadata JSONB,
    expires_at TIMESTAMP,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES conversation_sessions(id)
);

-- 索引
CREATE INDEX idx_memories_tenant_session ON conversation_memories(tenant_id, session_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_memories_type ON conversation_memories(memory_type) WHERE is_deleted = FALSE;
CREATE INDEX idx_memories_expires ON conversation_memories(expires_at) WHERE is_deleted = FALSE AND expires_at IS NOT NULL;

-- 注意：向量数据存储在 Qdrant 中，不在 PostgreSQL
-- PostgreSQL 仅存储记忆的元数据
```

#### conversation_contexts 表

存储会话的上下文配置。

```sql
CREATE TABLE conversation_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL UNIQUE,
    max_tokens INTEGER NOT NULL DEFAULT 4000,
    strategy VARCHAR(50) NOT NULL DEFAULT 'auto', -- 'auto', 'short', 'full'
    include_summary BOOLEAN NOT NULL DEFAULT TRUE,
    include_long_term BOOLEAN NOT NULL DEFAULT TRUE,
    short_term_window INTEGER NOT NULL DEFAULT 10,
    last_summary_id UUID,
    last_summary_at TIMESTAMP,
    total_messages INTEGER NOT NULL DEFAULT 0,
    total_tokens_used BIGINT NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES conversation_sessions(id),
    CONSTRAINT fk_last_summary FOREIGN KEY (last_summary_id) REFERENCES conversation_memories(id)
);

-- 索引
CREATE INDEX idx_contexts_tenant ON conversation_contexts(tenant_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_contexts_session ON conversation_contexts(session_id) WHERE is_deleted = FALSE;
```

#### conversation_summaries 表

存储会话摘要。

```sql
CREATE TABLE conversation_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_id UUID NOT NULL,
    summary_type VARCHAR(50) NOT NULL, -- 'incremental', 'full'
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL,
    message_count INTEGER NOT NULL,
    start_message_id UUID,
    end_message_id UUID,
    quality_score FLOAT, -- 0-1
    compression_rate FLOAT, -- 0-1
    key_topics TEXT[], -- 关键主题数组
    previous_summary_id UUID,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES conversation_sessions(id),
    CONSTRAINT fk_previous_summary FOREIGN KEY (previous_summary_id) REFERENCES conversation_summaries(id)
);

-- 索引
CREATE INDEX idx_summaries_tenant_session ON conversation_summaries(tenant_id, session_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_summaries_created ON conversation_summaries(created_at DESC) WHERE is_deleted = FALSE;
```

### Go 数据模型

#### ConversationMemory 模型

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type ConversationMemory struct {
    ID            uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID      uuid.UUID       `gorm:"type:uuid;not null;index:idx_memories_tenant_session" json:"tenantId"`
    SessionID     uuid.UUID       `gorm:"type:uuid;not null;index:idx_memories_tenant_session" json:"sessionId"`
    MemoryType    string          `gorm:"type:varchar(50);not null;index:idx_memories_type" json:"memoryType"`
    Content       string          `gorm:"type:text;not null" json:"content"`
    // 注意：向量数据存储在 Qdrant 中，不在 PostgreSQL
    TokenCount    int             `gorm:"not null;default:0" json:"tokenCount"`
    Importance    float32         `gorm:"not null;default:0.5" json:"importance"`
    AccessCount   int             `gorm:"not null;default:0" json:"accessCount"`
    LastAccessAt  *time.Time      `json:"lastAccessAt,omitempty"`
    Metadata      map[string]interface{} `gorm:"type:jsonb" json:"metadata,omitempty"`
    ExpiresAt     *time.Time      `gorm:"index:idx_memories_expires" json:"expiresAt,omitempty"`
    IsDeleted     bool            `gorm:"not null;default:false" json:"-"`
    CreatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (ConversationMemory) TableName() string {
    return "conversation_memories"
}
```

#### ConversationContext 模型

```go
type ConversationContext struct {
    ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_contexts_tenant" json:"tenantId"`
    SessionID         uuid.UUID  `gorm:"type:uuid;not null;unique;index:idx_contexts_session" json:"sessionId"`
    MaxTokens         int        `gorm:"not null;default:4000" json:"maxTokens"`
    Strategy          string     `gorm:"type:varchar(50);not null;default:'auto'" json:"strategy"`
    IncludeSummary    bool       `gorm:"not null;default:true" json:"includeSummary"`
    IncludeLongTerm   bool       `gorm:"not null;default:true" json:"includeLongTerm"`
    ShortTermWindow   int        `gorm:"not null;default:10" json:"shortTermWindow"`
    LastSummaryID     *uuid.UUID `json:"lastSummaryId,omitempty"`
    LastSummaryAt     *time.Time `json:"lastSummaryAt,omitempty"`
    TotalMessages     int        `gorm:"not null;default:0" json:"totalMessages"`
    TotalTokensUsed   int64      `gorm:"not null;default:0" json:"totalTokensUsed"`
    IsDeleted         bool       `gorm:"not null;default:false" json:"-"`
    CreatedAt         time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt         time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (ConversationContext) TableName() string {
    return "conversation_contexts"
}
```

#### ConversationSummary 模型

```go
type ConversationSummary struct {
    ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_summaries_tenant_session" json:"tenantId"`
    SessionID         uuid.UUID  `gorm:"type:uuid;not null;index:idx_summaries_tenant_session" json:"sessionId"`
    SummaryType       string     `gorm:"type:varchar(50);not null" json:"summaryType"`
    Content           string     `gorm:"type:text;not null" json:"content"`
    TokenCount        int        `gorm:"not null" json:"tokenCount"`
    MessageCount      int        `gorm:"not null" json:"messageCount"`
    StartMessageID    *uuid.UUID `json:"startMessageId,omitempty"`
    EndMessageID      *uuid.UUID `json:"endMessageId,omitempty"`
    QualityScore      *float64   `json:"qualityScore,omitempty"`
    CompressionRate   *float64   `json:"compressionRate,omitempty"`
    KeyTopics         []string   `gorm:"type:text[]" json:"keyTopics,omitempty"`
    PreviousSummaryID *uuid.UUID `json:"previousSummaryId,omitempty"`
    IsDeleted         bool       `gorm:"not null;default:false" json:"-"`
    CreatedAt         time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_summaries_created" json:"createdAt"`
    UpdatedAt         time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (ConversationSummary) TableName() string {
    return "conversation_summaries"
}
```

### Qdrant 向量数据库设计

#### 多租户架构设计

根据 Qdrant 官方文档和最佳实践（参考：Qdrant v1.11+ 多租户支持），本系统采用**单个共享 Collection + Payload 分区**的多租户架构：

- 使用单个 Collection：`conversation_memories`
- 在 Payload 中添加 `tenant_id` 字段并设置 `is_tenant=true` 索引
- 通过 Payload 过滤实现租户隔离
- 配置自定义分片策略按租户分布数据

**优势**：

- 简化管理：无需为每个租户创建和维护独立的 Collection
- 性能优化：Qdrant 针对租户标识字段进行了专门优化（v1.11+）
- 资源效率：共享索引结构，减少内存占用
- 扩展性好：支持大量租户而不会产生管理开销

#### Collection 结构

```json
{
  "name": "conversation_memories",
  "vectors": {
    "size": 1536,
    "distance": "Cosine"
  },
  "shard_number": 4,
  "replication_factor": 2,
  "payload_schema": {
    "memory_id": "keyword",
    "tenant_id": {
      "type": "keyword",
      "is_tenant": true
    },
    "session_id": "keyword",
    "memory_type": "keyword",
    "importance": "float",
    "created_at": "datetime",
    "expires_at": "datetime"
  },
  "hnsw_config": {
    "m": 16,
    "ef_construction": 100
  }
}
```

#### Payload 结构

每个向量点的 payload 包含：

```json
{
  "memory_id": "uuid",
  "tenant_id": "uuid",
  "session_id": "uuid",
  "memory_type": "short_term|long_term|summary",
  "importance": 0.8,
  "created_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

#### 索引配置

```json
{
  "field_name": "tenant_id",
  "field_schema": "keyword"
},
{
  "field_name": "session_id",
  "field_schema": "keyword"
},
{
  "field_name": "memory_type",
  "field_schema": "keyword"
}
```

#### 向量服务接口

```go
// internal/service/vector_service.go
package service

import (
    "context"
    "github.com/google/uuid"
    qdrant "github.com/qdrant/go-client/qdrant"
)

type VectorService interface {
    // InitializeCollection 初始化共享 Collection（仅在启动时调用一次）
    InitializeCollection(ctx context.Context) error
    
    // StoreVector 存储向量（自动包含租户ID在payload中）
    StoreVector(ctx context.Context, req StoreVectorRequest) error
    
    // SearchVectors 搜索相似向量（强制按租户ID过滤）
    SearchVectors(ctx context.Context, req SearchVectorRequest) ([]VectorSearchResult, error)
    
    // DeleteVector 删除向量（验证租户权限）
    DeleteVector(ctx context.Context, tenantID, memoryID uuid.UUID) error
    
    // DeleteByFilter 按条件批量删除（验证租户权限）
    DeleteByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error
    
    // UpdatePayload 更新 payload（验证租户权限）
    UpdatePayload(ctx context.Context, tenantID, memoryID uuid.UUID, payload map[string]interface{}) error
}

type StoreVectorRequest struct {
    TenantID   uuid.UUID
    MemoryID   uuid.UUID
    SessionID  uuid.UUID
    MemoryType string
    Vector     []float32
    Importance float32
    ExpiresAt  *time.Time
}

type SearchVectorRequest struct {
    TenantID      uuid.UUID
    SessionID     *uuid.UUID  // 可选，用于会话内搜索
    QueryVector   []float32
    TopK          int
    MinScore      float32
    MemoryType    *string     // 可选过滤
    TimeRange     *TimeRange  // 可选时间范围
}

type VectorSearchResult struct {
    MemoryID   uuid.UUID
    Score      float32
    Payload    map[string]interface{}
}

type TimeRange struct {
    Start time.Time
    End   time.Time
}
```

#### Qdrant 客户端实现

```go
// internal/storage/qdrant_client.go
package storage

import (
    "context"
    "fmt"
    
    "github.com/google/uuid"
    qdrant "github.com/qdrant/go-client/qdrant"
)

type QdrantClient struct {
    client *qdrant.Client
    config *QdrantConfig
}

type QdrantConfig struct {
    Host   string
    Port   int
    APIKey string
    UseTLS bool
}

func NewQdrantClient(config *QdrantConfig) (*QdrantClient, error) {
    client, err := qdrant.NewClient(&qdrant.Config{
        Host:   config.Host,
        Port:   config.Port,
        APIKey: config.APIKey,
        UseTLS: config.UseTLS,
    })
    
    if err != nil {
        return nil, fmt.Errorf("创建 Qdrant 客户端失败: %w", err)
    }
    
    return &QdrantClient{
        client: client,
        config: config,
    }, nil
}

const CollectionName = "conversation_memories"

func (c *QdrantClient) InitializeCollection(ctx context.Context) error {
    // 检查 Collection 是否已存在
    exists, err := c.client.CollectionExists(ctx, CollectionName)
    if err != nil {
        return fmt.Errorf("检查 collection 失败: %w", err)
    }
    
    if exists {
        return nil // Collection 已存在，跳过创建
    }
    
    // 创建共享 Collection
    err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
        CollectionName: CollectionName,
        VectorsConfig: &qdrant.VectorsConfig{
            Params: &qdrant.VectorParams{
                Size:     1536,
                Distance: qdrant.Distance_Cosine,
            },
        },
        ShardNumber:        4,  // 根据租户数量调整
        ReplicationFactor:  2,  // 高可用配置
        HnswConfig: &qdrant.HnswConfigDiff{
            M:              16,
            EfConstruction: 100,
        },
    })
    
    if err != nil {
        return fmt.Errorf("创建 collection 失败: %w", err)
    }
    
    // 创建租户标识索引（is_tenant=true）
    err = c.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
        CollectionName: CollectionName,
        FieldName:      "tenant_id",
        FieldType:      qdrant.FieldType_FieldTypeKeyword,
        IsTenant:       true,  // 关键：标记为租户字段
    })
    
    if err != nil {
        return fmt.Errorf("创建租户索引失败: %w", err)
    }
    
    // 创建其他 payload 索引
    indexes := []string{"session_id", "memory_type"}
    for _, field := range indexes {
        err = c.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
            CollectionName: CollectionName,
            FieldName:      field,
            FieldType:      qdrant.FieldType_FieldTypeKeyword,
        })
        
        if err != nil {
            return fmt.Errorf("创建索引 %s 失败: %w", field, err)
        }
    }
    
    return nil
}

func (c *QdrantClient) Upsert(ctx context.Context, tenantID uuid.UUID, points []*qdrant.PointStruct) error {
    // 验证所有 points 都包含正确的 tenant_id
    for _, point := range points {
        if point.Payload == nil {
            point.Payload = make(map[string]interface{})
        }
        point.Payload["tenant_id"] = tenantID.String()
    }
    
    _, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
        CollectionName: CollectionName,
        Points:         points,
    })
    
    return err
}

func (c *QdrantClient) Search(ctx context.Context, tenantID uuid.UUID, req *qdrant.SearchPoints) ([]*qdrant.ScoredPoint, error) {
    req.CollectionName = CollectionName
    
    // 强制添加租户过滤条件
    tenantFilter := &qdrant.Filter{
        Must: []*qdrant.Condition{
            {
                Field: "tenant_id",
                Match: &qdrant.Match{
                    Value: tenantID.String(),
                },
            },
        },
    }
    
    // 合并用户提供的过滤条件
    if req.Filter != nil {
        if req.Filter.Must == nil {
            req.Filter.Must = []*qdrant.Condition{}
        }
        req.Filter.Must = append(req.Filter.Must, tenantFilter.Must...)
    } else {
        req.Filter = tenantFilter
    }
    
    result, err := c.client.Search(ctx, req)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

func (c *QdrantClient) DeleteByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error {
    // 构建删除过滤条件，强制包含租户ID
    conditions := []*qdrant.Condition{
        {
            Field: "tenant_id",
            Match: &qdrant.Match{
                Value: tenantID.String(),
            },
        },
    }
    
    // 添加用户提供的过滤条件
    for field, value := range filter {
        conditions = append(conditions, &qdrant.Condition{
            Field: field,
            Match: &qdrant.Match{
                Value: value,
            },
        })
    }
    
    _, err := c.client.Delete(ctx, &qdrant.DeletePoints{
        CollectionName: CollectionName,
        Filter: &qdrant.Filter{
            Must: conditions,
        },
    })
    
    return err
}
```

## Genkit Flow 设计

### Flow 组织结构

```
internal/genkit/
├── client.go              # Genkit 客户端封装
├── config.go              # 配置加载
├── registry.go            # Flow 注册器
├── middleware.go          # Flow 中间件
├── flows/
│   ├── context.go         # 上下文相关 Flow
│   ├── chat.go            # 对话相关 Flow
│   ├── memory.go          # 记忆相关 Flow
│   ├── summary.go         # 摘要相关 Flow
│   ├── token.go           # Token 管理 Flow
│   ├── health.go          # 健康检查 Flow
│   └── types.go           # 共享类型定义
└── errors/
    └── errors.go          # 错误处理
```

### 核心 Flow 定义

#### 1. contextBuildFlow

**功能**：构建智能对话上下文

**输入输出定义**：

```go
// internal/genkit/flows/types.go
package flows

type ContextBuildInput struct {
    SessionID       string `json:"sessionId" validate:"required,uuid"`
    UserQuery       string `json:"userQuery" validate:"required,max=2000"`
    MaxTokens       int    `json:"maxTokens" validate:"min=100,max=32000"`
    Strategy        string `json:"strategy" validate:"oneof=auto short full"`
    IncludeSummary  bool   `json:"includeSummary"`
    IncludeLongTerm bool   `json:"includeLongTerm"`
    ShortTermWindow int    `json:"shortTermWindow" validate:"min=1,max=50"`
}

type ContextBuildOutput struct {
    SessionID         string           `json:"sessionId"`
    Summary           *SummaryContext  `json:"summary,omitempty"`
    LongTermMemories  []MemoryContext  `json:"longTermMemories,omitempty"`
    ShortTermMessages []MessageContext `json:"shortTermMessages"`
    TotalTokens       int              `json:"totalTokens"`
    Strategy          string           `json:"strategy"`
    QualityScore      float64          `json:"qualityScore"`
    BuildTime         int64            `json:"buildTime"`
}

type SummaryContext struct {
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
    Coverage   string `json:"coverage"`
}

type MemoryContext struct {
    ID         string  `json:"id"`
    Content    string  `json:"content"`
    TokenCount int     `json:"tokenCount"`
    Importance float32 `json:"importance"`
    Similarity float32 `json:"similarity"`
    CreatedAt  string  `json:"createdAt"`
}

type MessageContext struct {
    ID         string `json:"id"`
    Role       string `json:"role"`
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
}
```

**Flow 实现**：

```go
// internal/genkit/flows/context.go
package flows

import (
    "context"
    "fmt"
    "time"
    
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/service"
)

func RegisterContextFlows(g *genkit.Genkit, contextSvc service.ContextService) {
    genkit.DefineFlow(
        g,
        "contextBuildFlow",
        func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
            startTime := time.Now()
            
            // 1. 参数验证
            if err := validateContextInput(input); err != nil {
                return ContextBuildOutput{}, err
            }
            
            // 2. 权限验证
            if err := validateTenantAccess(ctx, input.SessionID); err != nil {
                return ContextBuildOutput{}, err
            }
            
            // 3. 调用服务层构建上下文
            result, err := contextSvc.BuildContext(ctx, service.BuildContextRequest{
                SessionID:       input.SessionID,
                UserQuery:       input.UserQuery,
                MaxTokens:       input.MaxTokens,
                Strategy:        input.Strategy,
                IncludeSummary:  input.IncludeSummary,
                IncludeLongTerm: input.IncludeLongTerm,
                ShortTermWindow: input.ShortTermWindow,
            })
            if err != nil {
                return ContextBuildOutput{}, fmt.Errorf("构建上下文失败: %w", err)
            }
            
            // 4. 转换为输出格式
            output := ContextBuildOutput{
                SessionID:         result.SessionID,
                Summary:           convertSummary(result.Summary),
                LongTermMemories:  convertMemories(result.LongTermMemories),
                ShortTermMessages: convertMessages(result.ShortTermMessages),
                TotalTokens:       result.TotalTokens,
                Strategy:          result.Strategy,
                QualityScore:      result.QualityScore,
                BuildTime:         time.Since(startTime).Milliseconds(),
            }
            
            return output, nil
        },
    )
}
```

#### 2. chatGenerateFlow

**功能**：生成 AI 对话响应

**输入输出定义**：

```go
type ChatGenerateInput struct {
    SessionID    string        `json:"sessionId" validate:"required,uuid"`
    UserMessage  string        `json:"userMessage" validate:"required,max=4000"`
    Context      *ContextBuildOutput `json:"context,omitempty"`
    ModelConfig  *ModelConfig  `json:"modelConfig,omitempty"`
    SystemPrompt string        `json:"systemPrompt" validate:"max=1000"`
    SaveMessage  bool          `json:"saveMessage"`
}

type ModelConfig struct {
    ModelName        string   `json:"modelName" validate:"required"`
    Temperature      float64  `json:"temperature" validate:"min=0,max=2"`
    TopP             float64  `json:"topP" validate:"min=0,max=1"`
    MaxTokens        int      `json:"maxTokens" validate:"min=1,max=4096"`
    StopSequences    []string `json:"stopSequences" validate:"max=4"`
    FrequencyPenalty float64  `json:"frequencyPenalty" validate:"min=-2,max=2"`
    PresencePenalty  float64  `json:"presencePenalty" validate:"min=-2,max=2"`
}

type ChatGenerateOutput struct {
    MessageID      string      `json:"messageId"`
    Response       string      `json:"response"`
    TokenUsage     TokenUsage  `json:"tokenUsage"`
    FinishReason   string      `json:"finishReason"`
    Model          string      `json:"model"`
    GenerationTime int64       `json:"generationTime"`
    ContextInfo    ContextInfo `json:"contextInfo"`
}

type TokenUsage struct {
    PromptTokens     int `json:"promptTokens"`
    CompletionTokens int `json:"completionTokens"`
    TotalTokens      int `json:"totalTokens"`
}

type ContextInfo struct {
    ContextTokens int     `json:"contextTokens"`
    Strategy      string  `json:"strategy"`
    QualityScore  float64 `json:"qualityScore"`
}
```

#### 3. memorySearchFlow

**功能**：基于向量相似度检索记忆

**输入输出定义**：

```go
type MemorySearchInput struct {
    SessionID            string   `json:"sessionId" validate:"required,uuid"`
    Query                string   `json:"query" validate:"required,max=2000"`
    TopK                 int      `json:"topK" validate:"min=1,max=20"`
    MinSimilarity        float32  `json:"minSimilarity" validate:"min=0,max=1"`
    TimeRangeDays        int      `json:"timeRangeDays" validate:"min=0,max=365"`
    MemoryTypes          []string `json:"memoryTypes" validate:"dive,oneof=short_term long_term summary"`
    IncludeCrossSessions bool     `json:"includeCrossSessions"`
}

type MemorySearchOutput struct {
    Memories          []MemoryResult `json:"memories"`
    TotalFound        int            `json:"totalFound"`
    ReturnedCount     int            `json:"returnedCount"`
    SearchTime        int64          `json:"searchTime"`
    AverageSimilarity float32        `json:"averageSimilarity"`
    SearchStrategy    string         `json:"searchStrategy"`
}

type MemoryResult struct {
    ID           string                 `json:"id"`
    SessionID    string                 `json:"sessionId"`
    MemoryType   string                 `json:"memoryType"`
    Content      string                 `json:"content"`
    TokenCount   int                    `json:"tokenCount"`
    Similarity   float32                `json:"similarity"`
    Importance   float32                `json:"importance"`
    Score        float32                `json:"score"`
    AccessCount  int                    `json:"accessCount"`
    CreatedAt    string                 `json:"createdAt"`
    LastAccessAt string                 `json:"lastAccessAt"`
    Metadata     map[string]interface{} `json:"metadata"`
}
```

#### 4. summaryGenerateFlow

**功能**：生成对话摘要

**输入输出定义**：

```go
type SummaryGenerateInput struct {
    SessionID       string   `json:"sessionId" validate:"required,uuid"`
    MessageIDs      []string `json:"messageIds" validate:"dive,uuid"`
    StartMessageID  string   `json:"startMessageId" validate:"omitempty,uuid"`
    EndMessageID    string   `json:"endMessageId" validate:"omitempty,uuid"`
    PreviousSummary string   `json:"previousSummary" validate:"max=2000"`
    SummaryType     string   `json:"summaryType" validate:"oneof=incremental full"`
    TargetLength    int      `json:"targetLength" validate:"min=50,max=1000"`
}

type SummaryGenerateOutput struct {
    SummaryID       string   `json:"summaryId"`
    Summary         string   `json:"summary"`
    TokenCount      int      `json:"tokenCount"`
    MessageCount    int      `json:"messageCount"`
    StartMessageID  string   `json:"startMessageId"`
    EndMessageID    string   `json:"endMessageId"`
    QualityScore    float64  `json:"qualityScore"`
    CompressionRate float64  `json:"compressionRate"`
    KeyTopics       []string `json:"keyTopics"`
    GenerationTime  int64    `json:"generationTime"`
}
```

### Flow 注册和调用

#### Flow 注册器

```go
// internal/genkit/registry.go
package genkit

import (
    "context"
    
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit/flows"
    "genkit-ai-service/internal/service"
)

type Registry struct {
    genkit   *genkit.Genkit
    services *Services
}

type Services struct {
    ContextService service.ContextService
    ChatService    service.ChatService
    MemoryService  service.MemoryService
    SummaryService service.SummaryService
    TokenService   service.TokenService
}

func NewRegistry(g *genkit.Genkit, services *Services) *Registry {
    return &Registry{
        genkit:   g,
        services: services,
    }
}

func (r *Registry) RegisterAllFlows(ctx context.Context) error {
    // 注册上下文相关 Flow
    flows.RegisterContextFlows(r.genkit, r.services.ContextService)
    
    // 注册对话相关 Flow
    flows.RegisterChatFlows(r.genkit, r.services.ChatService)
    
    // 注册记忆相关 Flow
    flows.RegisterMemoryFlows(r.genkit, r.services.MemoryService)
    
    // 注册摘要相关 Flow
    flows.RegisterSummaryFlows(r.genkit, r.services.SummaryService)
    
    // 注册 Token 管理 Flow
    flows.RegisterTokenFlows(r.genkit, r.services.TokenService)
    
    return nil
}
```

#### Flow 调用示例

```go
// internal/api/handler/context_handler.go
package handler

import (
    "net/http"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/gin-gonic/gin"
    "genkit-ai-service/internal/genkit/flows"
)

type ContextHandler struct {
    genkit *genkit.Genkit
}

func NewContextHandler(g *genkit.Genkit) *ContextHandler {
    return &ContextHandler{genkit: g}
}

func (h *ContextHandler) HandleBuildContext(c *gin.Context) {
    var input flows.ContextBuildInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "code":    400,
            "message": "请求参数错误",
            "details": err.Error(),
        })
        return
    }
    
    // 查找并调用 Flow
    flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
        h.genkit,
        "contextBuildFlow",
    )
    
    output, err := flow.Run(c.Request.Context(), input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "code":    500,
            "message": "构建上下文失败",
            "details": err.Error(),
        })
        return
    }
    
    // 返回标准响应格式
    c.JSON(http.StatusOK, gin.H{
        "code":    200,
        "message": "上下文构建成功",
        "data":    output,
    })
}
```

## 服务层设计

### 服务接口定义

#### ContextService

```go
// internal/service/context_service.go
package service

import (
    "context"
    "genkit-ai-service/internal/model"
)

type ContextService interface {
    // BuildContext 构建上下文
    BuildContext(ctx context.Context, req BuildContextRequest) (*ContextResult, error)
    
    // OptimizeContext 优化上下文
    OptimizeContext(ctx context.Context, req OptimizeContextRequest) (*ContextResult, error)
    
    // GetContextConfig 获取上下文配置
    GetContextConfig(ctx context.Context, sessionID string) (*model.ConversationContext, error)
    
    // UpdateContextConfig 更新上下文配置
    UpdateContextConfig(ctx context.Context, sessionID string, config *model.ConversationContext) error
}

type BuildContextRequest struct {
    SessionID       string
    UserQuery       string
    MaxTokens       int
    Strategy        string
    IncludeSummary  bool
    IncludeLongTerm bool
    ShortTermWindow int
}

type ContextResult struct {
    SessionID         string
    Summary           *model.ConversationSummary
    LongTermMemories  []*model.ConversationMemory
    ShortTermMessages []*model.ConversationMessage
    TotalTokens       int
    Strategy          string
    QualityScore      float64
}

type OptimizeContextRequest struct {
    Context         *ContextResult
    TargetTokens    int
    Strategy        string
    PreserveSummary bool
}
```

#### MemoryService

```go
type MemoryService interface {
    // SearchMemories 检索记忆
    SearchMemories(ctx context.Context, req SearchMemoriesRequest) ([]*MemorySearchResult, error)
    
    // StoreMemory 存储记忆
    StoreMemory(ctx context.Context, req StoreMemoryRequest) (*model.ConversationMemory, error)
    
    // CleanupMemories 清理记忆
    CleanupMemories(ctx context.Context, req CleanupMemoriesRequest) (*CleanupResult, error)
    
    // UpdateMemoryAccess 更新记忆访问统计
    UpdateMemoryAccess(ctx context.Context, memoryID string) error
}

type SearchMemoriesRequest struct {
    SessionID            string
    Query                string
    TopK                 int
    MinSimilarity        float32
    TimeRangeDays        int
    MemoryTypes          []string
    IncludeCrossSessions bool
    TenantID             string
}

type MemorySearchResult struct {
    Memory     *model.ConversationMemory
    Similarity float32
    Score      float32
}

type StoreMemoryRequest struct {
    SessionID      string
    MessageIDs     []string
    MemoryType     string
    Content        string
    Importance     float32
    ExpirationDays int
    Metadata       map[string]interface{}
    TenantID       string
}

type CleanupMemoriesRequest struct {
    SessionID  string
    TenantID   string
    Strategy   string
    Mode       string
    BatchSize  int
    Execute    bool
}

type CleanupResult struct {
    CleanedCount int
    FreedSpace   int64
    Details      []CleanupDetail
}

type CleanupDetail struct {
    MemoryID   string
    Reason     string
    Size       int64
    CreatedAt  string
    LastAccess string
}
```

#### SummaryService

```go
type SummaryService interface {
    // GenerateSummary 生成摘要
    GenerateSummary(ctx context.Context, req GenerateSummaryRequest) (*model.ConversationSummary, error)
    
    // CheckSummaryTrigger 检查是否需要生成摘要
    CheckSummaryTrigger(ctx context.Context, sessionID string) (*SummaryTriggerResult, error)
    
    // EvaluateSummaryQuality 评估摘要质量
    EvaluateSummaryQuality(ctx context.Context, req EvaluateSummaryRequest) (*SummaryQualityResult, error)
}

type GenerateSummaryRequest struct {
    SessionID       string
    MessageIDs      []string
    StartMessageID  string
    EndMessageID    string
    PreviousSummary string
    SummaryType     string
    TargetLength    int
    TenantID        string
}

type SummaryTriggerResult struct {
    ShouldSummarize       bool
    TriggerReason         string
    MessageIDs            []string
    MessageCount          int
    EstimatedTokenSaving  int
    Urgency               float64
    RecommendedType       string
}

type EvaluateSummaryRequest struct {
    Summary          string
    OriginalMessages []string
    Dimensions       []string
}

type SummaryQualityResult struct {
    OverallScore     float64
    DimensionScores  map[string]float64
    Passed           bool
    Issues           []QualityIssue
    Suggestions      []string
    KeyInfoCoverage  float64
}

type QualityIssue struct {
    Dimension   string
    Severity    string
    Description string
    Score       float64
}
```

### 服务实现示例

#### ContextService 实现

```go
// internal/service/context_service_impl.go
package service

import (
    "context"
    "fmt"
    
    "genkit-ai-service/internal/middleware"
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
)

type contextServiceImpl struct {
    sessionRepo repository.SessionRepository
    messageRepo repository.MessageRepository
    memoryRepo  repository.MemoryRepository
    contextRepo repository.ContextRepository
    vectorSvc   VectorService
    tokenMgr    TokenManager
    cache       CacheService
}

func NewContextService(
    sessionRepo repository.SessionRepository,
    messageRepo repository.MessageRepository,
    memoryRepo repository.MemoryRepository,
    contextRepo repository.ContextRepository,
    vectorSvc VectorService,
    tokenMgr TokenManager,
    cache CacheService,
) ContextService {
    return &contextServiceImpl{
        sessionRepo: sessionRepo,
        messageRepo: messageRepo,
        memoryRepo:  memoryRepo,
        contextRepo: contextRepo,
        vectorSvc:   vectorSvc,
        tokenMgr:    tokenMgr,
        cache:       cache,
    }
}

func (s *contextServiceImpl) BuildContext(
    ctx context.Context,
    req BuildContextRequest,
) (*ContextResult, error) {
    // 1. 权限验证
    if err := s.validateAccess(ctx, req.SessionID); err != nil {
        return nil, err
    }
    
    // 2. 尝试从缓存获取
    cacheKey := fmt.Sprintf("context:%s:%s", req.SessionID, req.UserQuery)
    var cached *ContextResult
    if err := s.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
        return cached, nil
    }
    
    // 3. 获取短期记忆
    messages, err := s.messageRepo.GetRecentMessages(
        ctx,
        req.SessionID,
        req.ShortTermWindow,
    )
    if err != nil {
        return nil, fmt.Errorf("获取短期记忆失败: %w", err)
    }
    
    // 4. 获取长期记忆（如果启用）
    var memories []*model.ConversationMemory
    if req.IncludeLongTerm && req.UserQuery != "" {
        // 生成查询向量
        embedding, err := s.vectorSvc.GenerateEmbedding(ctx, req.UserQuery)
        if err != nil {
            // 记录错误但不中断流程
            logger.WarnContext(ctx, "生成查询向量失败", "error", err)
        } else {
            // 执行向量检索
            memories, err = s.memoryRepo.SearchByVector(
                ctx,
                req.SessionID,
                embedding,
                5,
                0.7,
            )
            if err != nil {
                logger.WarnContext(ctx, "向量检索失败", "error", err)
            }
        }
    }
    
    // 5. 获取摘要（如果启用）
    var summary *model.ConversationSummary
    if req.IncludeSummary {
        summary, err = s.contextRepo.GetLatestSummary(ctx, req.SessionID)
        if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
            logger.WarnContext(ctx, "获取摘要失败", "error", err)
        }
    }
    
    // 6. 计算 Token 数量
    totalTokens := s.tokenMgr.CalculateContextTokens(messages, memories, summary)
    
    // 7. Token 优化（如果超限）
    if totalTokens > req.MaxTokens {
        messages, memories, summary = s.optimizeContext(
            messages,
            memories,
            summary,
            req.MaxTokens,
        )
        totalTokens = s.tokenMgr.CalculateContextTokens(messages, memories, summary)
    }
    
    // 8. 计算质量评分
    qualityScore := s.calculateQualityScore(messages, memories, summary)
    
    // 9. 构建结果
    result := &ContextResult{
        SessionID:         req.SessionID,
        Summary:           summary,
        LongTermMemories:  memories,
        ShortTermMessages: messages,
        TotalTokens:       totalTokens,
        Strategy:          req.Strategy,
        QualityScore:      qualityScore,
    }
    
    // 10. 缓存结果
    go func() {
        cacheCtx := context.Background()
        s.cache.Set(cacheCtx, cacheKey, result, 5*time.Minute)
    }()
    
    return result, nil
}

func (s *contextServiceImpl) validateAccess(ctx context.Context, sessionID string) error {
    // 获取 JWT 声明
    claims := middleware.GetJWTClaims(ctx)
    if claims == nil {
        return errors.NewUnauthorizedError("未认证")
    }
    
    // 查询会话
    session, err := s.sessionRepo.GetByID(ctx, sessionID)
    if err != nil {
        return errors.NewNotFoundError("会话不存在")
    }
    
    // 平台管理员可以访问所有会话
    if hasRole(claims, model.RoleSystemAdmin) {
        return nil
    }
    
    // 获取会话所属用户的租户ID
    sessionUser, err := s.userRepo.GetByID(ctx, session.UserID.String())
    if err != nil {
        return errors.NewInternalError(err)
    }
    
    // 验证租户ID匹配
    if claims.TenantID != sessionUser.TenantID.String() {
        logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话",
            "user_id", claims.Subject,
            "user_tenant_id", claims.TenantID,
            "session_id", sessionID,
            "session_tenant_id", sessionUser.TenantID,
        )
        return errors.NewForbiddenError("权限不足：无法访问其他租户的会话")
    }
    
    return nil
}
```

## Repository 层设计

### Repository 接口定义

#### MemoryRepository

```go
// internal/repository/memory_repository.go
package repository

import (
    "context"
    "genkit-ai-service/internal/model"
    "github.com/pgvector/pgvector-go"
)

type MemoryRepository interface {
    // Create 创建记忆
    Create(ctx context.Context, memory *model.ConversationMemory) error
    
    // GetByID 根据ID获取记忆
    GetByID(ctx context.Context, id string) (*model.ConversationMemory, error)
    
    // SearchByVector 向量相似度搜索
    SearchByVector(
        ctx context.Context,
        sessionID string,
        embedding pgvector.Vector,
        topK int,
        minSimilarity float32,
    ) ([]*model.ConversationMemory, error)
    
    // SearchByVectorCrossSessions 跨会话向量搜索
    SearchByVectorCrossSessions(
        ctx context.Context,
        tenantID string,
        embedding pgvector.Vector,
        topK int,
        minSimilarity float32,
    ) ([]*model.ConversationMemory, error)
    
    // UpdateAccessStats 更新访问统计
    UpdateAccessStats(ctx context.Context, id string) error
    
    // DeleteByStrategy 按策略删除记忆
    DeleteByStrategy(
        ctx context.Context,
        tenantID string,
        strategy string,
        mode string,
        batchSize int,
    ) (int, error)
    
    // GetExpiredMemories 获取过期记忆
    GetExpiredMemories(ctx context.Context, batchSize int) ([]*model.ConversationMemory, error)
}
```

#### ContextRepository

```go
type ContextRepository interface {
    // Create 创建上下文配置
    Create(ctx context.Context, context *model.ConversationContext) error
    
    // GetBySessionID 根据会话ID获取上下文配置
    GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error)
    
    // Update 更新上下文配置
    Update(ctx context.Context, context *model.ConversationContext) error
    
    // GetLatestSummary 获取最新摘要
    GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error)
    
    // UpdateTokenUsage 更新Token使用统计
    UpdateTokenUsage(ctx context.Context, sessionID string, tokens int) error
}
```

### Repository 实现示例

#### MemoryRepository 实现

```go
// internal/repository/memory_repository_impl.go
package repository

import (
    "context"
    "fmt"
    "time"
    
    "gorm.io/gorm"
    "github.com/pgvector/pgvector-go"
    "genkit-ai-service/internal/model"
)

type memoryRepositoryImpl struct {
    db *gorm.DB
}

func NewMemoryRepository(db *gorm.DB) MemoryRepository {
    return &memoryRepositoryImpl{db: db}
}

func (r *memoryRepositoryImpl) SearchByVector(
    ctx context.Context,
    sessionID string,
    embedding pgvector.Vector,
    topK int,
    minSimilarity float32,
) ([]*model.ConversationMemory, error) {
    var memories []*model.ConversationMemory
    
    // 使用余弦相似度搜索
    // <=> 是 pgvector 的余弦距离操作符
    // 1 - 余弦距离 = 余弦相似度
    err := r.db.WithContext(ctx).
        Where("session_id = ?", sessionID).
        Where("is_deleted = ?", false).
        Where("(1 - (embedding <=> ?)) >= ?", embedding, minSimilarity).
        Order(gorm.Expr("embedding <=> ?", embedding)).
        Limit(topK).
        Find(&memories).Error
    
    if err != nil {
        return nil, fmt.Errorf("向量检索失败: %w", err)
    }
    
    return memories, nil
}

func (r *memoryRepositoryImpl) SearchByVectorCrossSessions(
    ctx context.Context,
    tenantID string,
    embedding pgvector.Vector,
    topK int,
    minSimilarity float32,
) ([]*model.ConversationMemory, error) {
    var memories []*model.ConversationMemory
    
    // 跨会话检索，但限制在同一租户内
    err := r.db.WithContext(ctx).
        Where("tenant_id = ?", tenantID).
        Where("is_deleted = ?", false).
        Where("(1 - (embedding <=> ?)) >= ?", embedding, minSimilarity).
        Order(gorm.Expr("embedding <=> ?", embedding)).
        Limit(topK).
        Find(&memories).Error
    
    if err != nil {
        return nil, fmt.Errorf("跨会话向量检索失败: %w", err)
    }
    
    return memories, nil
}

func (r *memoryRepositoryImpl) UpdateAccessStats(ctx context.Context, id string) error {
    now := time.Now()
    
    return r.db.WithContext(ctx).
        Model(&model.ConversationMemory{}).
        Where("id = ?", id).
        Updates(map[string]interface{}{
            "access_count":   gorm.Expr("access_count + 1"),
            "last_access_at": now,
        }).Error
}

func (r *memoryRepositoryImpl) DeleteByStrategy(
    ctx context.Context,
    tenantID string,
    strategy string,
    mode string,
    batchSize int,
) (int, error) {
    query := r.db.WithContext(ctx).
        Where("tenant_id = ?", tenantID).
        Where("is_deleted = ?", false)
    
    // 根据策略添加过滤条件
    switch strategy {
    case "expired":
        query = query.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now())
    case "low_quality":
        query = query.Where("importance < ? AND access_count < ?", 0.3, 2)
    case "unused":
        cutoff := time.Now().AddDate(0, 0, -90)
        query = query.Where("last_access_at < ?", cutoff)
    }
    
    query = query.Limit(batchSize)
    
    // 根据模式执行删除
    var result *gorm.DB
    if mode == "soft" {
        // 软删除
        result = query.Update("is_deleted", true)
    } else {
        // 硬删除
        result = query.Delete(&model.ConversationMemory{})
    }
    
    if result.Error != nil {
        return 0, fmt.Errorf("删除记忆失败: %w", result.Error)
    }
    
    return int(result.RowsAffected), nil
}
```

## 缓存设计

### 缓存策略

#### 缓存场景和 TTL

| 缓存类型 | 缓存键格式 | TTL | 说明 |
|---------|-----------|-----|------|
| 会话上下文 | `context:{sessionId}:{queryHash}` | 5分钟 | 缓存构建的上下文 |
| 向量查询结果 | `vector:{sessionId}:{queryHash}` | 30分钟 | 缓存向量检索结果 |
| 会话摘要 | `summary:{sessionId}:latest` | 1小时 | 缓存最新摘要 |
| 用户会话列表 | `sessions:{userId}:list` | 10分钟 | 缓存会话列表 |
| Token使用统计 | `tokens:{sessionId}:usage` | 5分钟 | 缓存Token统计 |
| 租户配额 | `quota:{tenantId}:{type}` | 5分钟 | 缓存配额信息 |

### 缓存服务实现

```go
// internal/storage/cache_service.go
package storage

import (
    "context"
    "crypto/md5"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type CacheService interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    DeletePattern(ctx context.Context, pattern string) error
    Exists(ctx context.Context, key string) (bool, error)
    Increment(ctx context.Context, key string, delta int64) (int64, error)
}

type cacheServiceImpl struct {
    client    *redis.Client
    namespace string
}

func NewCacheService(client *redis.Client, namespace string) CacheService {
    return &cacheServiceImpl{
        client:    client,
        namespace: namespace,
    }
}

func (s *cacheServiceImpl) Get(ctx context.Context, key string, dest interface{}) error {
    fullKey := s.buildKey(key)
    
    data, err := s.client.Get(ctx, fullKey).Bytes()
    if err != nil {
        if err == redis.Nil {
            return ErrCacheNotFound
        }
        return fmt.Errorf("获取缓存失败: %w", err)
    }
    
    if err := json.Unmarshal(data, dest); err != nil {
        return fmt.Errorf("反序列化缓存失败: %w", err)
    }
    
    return nil
}

func (s *cacheServiceImpl) Set(
    ctx context.Context,
    key string,
    value interface{},
    ttl time.Duration,
) error {
    fullKey := s.buildKey(key)
    
    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("序列化缓存失败: %w", err)
    }
    
    if err := s.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
        return fmt.Errorf("设置缓存失败: %w", err)
    }
    
    return nil
}

func (s *cacheServiceImpl) DeletePattern(ctx context.Context, pattern string) error {
    fullPattern := s.buildKey(pattern)
    
    iter := s.client.Scan(ctx, 0, fullPattern, 0).Iterator()
    
    var keys []string
    for iter.Next(ctx) {
        keys = append(keys, iter.Val())
    }
    
    if err := iter.Err(); err != nil {
        return fmt.Errorf("扫描缓存失败: %w", err)
    }
    
    if len(keys) > 0 {
        return s.Delete(ctx, keys...)
    }
    
    return nil
}

func (s *cacheServiceImpl) buildKey(key string) string {
    return fmt.Sprintf("%s:%s", s.namespace, key)
}

func (s *cacheServiceImpl) hashQuery(query string) string {
    hash := md5.Sum([]byte(query))
    return hex.EncodeToString(hash[:])
}
```

### 缓存预热

```go
// internal/storage/cache_warmer.go
package storage

import (
    "context"
    "log"
    "time"
    
    "genkit-ai-service/internal/repository"
)

type CacheWarmer struct {
    cache       CacheService
    sessionRepo repository.SessionRepository
    contextRepo repository.ContextRepository
}

func NewCacheWarmer(
    cache CacheService,
    sessionRepo repository.SessionRepository,
    contextRepo repository.ContextRepository,
) *CacheWarmer {
    return &CacheWarmer{
        cache:       cache,
        sessionRepo: sessionRepo,
        contextRepo: contextRepo,
    }
}

func (w *CacheWarmer) WarmupOnStartup(ctx context.Context) error {
    log.Println("开始缓存预热...")
    
    // 预热活跃会话
    if err := w.warmupActiveSessions(ctx); err != nil {
        log.Printf("预热活跃会话失败: %v", err)
    }
    
    log.Println("缓存预热完成")
    return nil
}

func (w *CacheWarmer) warmupActiveSessions(ctx context.Context) error {
    // 获取最近活跃的会话
    sessions, err := w.sessionRepo.GetRecentActive(ctx, 100)
    if err != nil {
        return err
    }
    
    for _, session := range sessions {
        // 预热会话上下文配置
        context, err := w.contextRepo.GetBySessionID(ctx, session.ID.String())
        if err != nil {
            continue
        }
        
        // 缓存上下文配置
        key := fmt.Sprintf("context:config:%s", session.ID.String())
        w.cache.Set(ctx, key, context, 10*time.Minute)
    }
    
    return nil
}

func (w *CacheWarmer) StartPeriodicWarmup(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := w.WarmupOnStartup(ctx); err != nil {
                log.Printf("定期预热失败: %v", err)
            }
        }
    }
}
```

## 错误处理设计

### 统一错误码

```go
// internal/model/errors.go
package model

const (
    // 通用错误 (10xxx)
    ErrCodeSuccess          = 10000
    ErrCodeBadRequest       = 10101
    ErrCodeUnauthorized     = 10102
    ErrCodeForbidden        = 10103
    ErrCodeNotFound         = 10104
    ErrCodeInternalError    = 10201
    
    // 会话管理错误 (30xxx)
    ErrCodeSessionNotFound  = 30104
    ErrCodeSessionExpired   = 30301
    
    // 上下文管理错误 (40xxx)
    ErrCodeContextBuildFailed = 40201
    ErrCodeTokenExceeded      = 40302
    
    // 记忆管理错误 (50xxx)
    ErrCodeMemoryNotFound     = 50104
    ErrCodeVectorGenerationFailed = 50201
    
    // AI 服务错误 (60xxx)
    ErrCodeAIServiceTimeout   = 60201
    ErrCodeAIServiceError     = 60202
    ErrCodeQuotaExceeded      = 60301
)

type AppError struct {
    Code       int    `json:"code"`
    Message    string `json:"message"`
    Details    string `json:"details,omitempty"`
    HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string {
    if e.Details != "" {
        return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
    }
    return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func NewAppError(code int, message, details string) *AppError {
    return &AppError{
        Code:       code,
        Message:    message,
        Details:    details,
        HTTPStatus: getHTTPStatus(code),
    }
}
```

### 降级策略

```go
// internal/service/degradation_service.go
package service

type DegradationService interface {
    DegradeAIService(ctx context.Context, sessionID, userQuery string) (string, error)
    DegradeVectorSearch(ctx context.Context, sessionID, query string) ([]model.Memory, error)
    DegradeSummaryGeneration(ctx context.Context, messages []model.Message) (string, error)
}

type degradationServiceImpl struct {
    cache CacheService
}

func (s *degradationServiceImpl) DegradeAIService(
    ctx context.Context,
    sessionID string,
    userQuery string,
) (string, error) {
    log.Printf("AI 服务降级: session=%s", sessionID)
    
    // 1. 尝试从缓存获取相似查询的响应
    cachedResponse, err := s.getCachedResponse(ctx, sessionID, userQuery)
    if err == nil && cachedResponse != "" {
        return cachedResponse, nil
    }
    
    // 2. 返回默认响应
    return "抱歉，服务暂时不可用，请稍后重试。", nil
}

func (s *degradationServiceImpl) DegradeVectorSearch(
    ctx context.Context,
    sessionID string,
    query string,
) ([]model.Memory, error) {
    log.Printf("向量检索降级: session=%s", sessionID)
    
    // 使用全文搜索作为降级方案
    memories, err := s.fullTextSearch(ctx, sessionID, query)
    if err == nil {
        return memories, nil
    }
    
    // 返回空结果
    return []model.Memory{}, nil
}
```

### 熔断机制

```go
// internal/middleware/circuit_breaker.go
package middleware

import (
    "context"
    "errors"
    "sync"
    "time"
)

type CircuitBreaker struct {
    mu              sync.RWMutex
    state           State
    failureCount    int
    successCount    int
    lastFailureTime time.Time
    maxFailures     int
    timeout         time.Duration
    halfOpenSuccess int
}

type State int

const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        state:           StateClosed,
        maxFailures:     maxFailures,
        timeout:         timeout,
        halfOpenSuccess: 3,
    }
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
    if !cb.canExecute() {
        return errors.New("熔断器已打开")
    }
    
    err := fn()
    cb.recordResult(err)
    
    return err
}

func (cb *CircuitBreaker) canExecute() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Since(cb.lastFailureTime) > cb.timeout {
            cb.mu.RUnlock()
            cb.mu.Lock()
            cb.state = StateHalfOpen
            cb.mu.Unlock()
            cb.mu.RLock()
            return true
        }
        return false
    case StateHalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) recordResult(err error) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if err != nil {
        cb.failureCount++
        cb.lastFailureTime = time.Now()
        
        if cb.failureCount >= cb.maxFailures {
            cb.state = StateOpen
            cb.successCount = 0
        }
    } else {
        cb.successCount++
        
        if cb.state == StateHalfOpen && cb.successCount >= cb.halfOpenSuccess {
            cb.state = StateClosed
            cb.failureCount = 0
            cb.successCount = 0
        }
    }
}
```

## 监控和可观测性设计

### 监控指标

#### Flow 执行指标

```go
// internal/monitoring/metrics.go
package monitoring

import (
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Flow 执行次数
    flowExecutions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "genkit_flow_executions_total",
            Help: "Total number of flow executions",
        },
        []string{"flow_name", "status"},
    )
    
    // Flow 执行时间
    flowDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "genkit_flow_duration_seconds",
            Help:    "Flow execution duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"flow_name"},
    )
    
    // Token 使用量
    tokenUsage = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "genkit_token_usage_total",
            Help: "Total token usage",
        },
        []string{"tenant_id", "type"},
    )
    
    // 缓存命中率
    cacheHits = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "genkit_cache_hits_total",
            Help: "Total cache hits",
        },
        []string{"cache_type"},
    )
    
    cacheMisses = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "genkit_cache_misses_total",
            Help: "Total cache misses",
        },
        []string{"cache_type"},
    )
)

type Metrics struct{}

func NewMetrics() *Metrics {
    return &Metrics{}
}

func (m *Metrics) RecordFlowExecution(flowName string, status string) {
    flowExecutions.WithLabelValues(flowName, status).Inc()
}

func (m *Metrics) RecordFlowDuration(flowName string, duration time.Duration) {
    flowDuration.WithLabelValues(flowName).Observe(duration.Seconds())
}

func (m *Metrics) RecordTokenUsage(tenantID string, tokenType string, count int) {
    tokenUsage.WithLabelValues(tenantID, tokenType).Add(float64(count))
}

func (m *Metrics) RecordCacheHit(cacheType string) {
    cacheHits.WithLabelValues(cacheType).Inc()
}

func (m *Metrics) RecordCacheMiss(cacheType string) {
    cacheMisses.WithLabelValues(cacheType).Inc()
}
```

### Flow 监控中间件

```go
// internal/genkit/middleware.go
package genkit

import (
    "context"
    "time"
    
    "genkit-ai-service/internal/monitoring"
)

type FlowMonitor struct {
    metrics *monitoring.Metrics
}

func NewFlowMonitor(metrics *monitoring.Metrics) *FlowMonitor {
    return &FlowMonitor{metrics: metrics}
}

func (m *FlowMonitor) MonitorFlow(
    flowName string,
    fn func(context.Context) error,
) func(context.Context) error {
    return func(ctx context.Context) error {
        startTime := time.Now()
        
        // 执行 Flow
        err := fn(ctx)
        
        // 记录执行时间
        duration := time.Since(startTime)
        m.metrics.RecordFlowDuration(flowName, duration)
        
        // 记录执行结果
        status := "success"
        if err != nil {
            status = "error"
        }
        m.metrics.RecordFlowExecution(flowName, status)
        
        return err
    }
}
```

### 日志设计

```go
// internal/logger/logger.go
package logger

import (
    "context"
    "encoding/json"
    "log"
    "time"
)

type LogEntry struct {
    Timestamp string                 `json:"timestamp"`
    Level     string                 `json:"level"`
    Flow      string                 `json:"flow,omitempty"`
    SessionID string                 `json:"session_id,omitempty"`
    UserID    string                 `json:"user_id,omitempty"`
    TenantID  string                 `json:"tenant_id,omitempty"`
    Duration  int64                  `json:"duration_ms,omitempty"`
    Status    string                 `json:"status,omitempty"`
    Message   string                 `json:"message"`
    Error     string                 `json:"error,omitempty"`
    Context   map[string]interface{} `json:"context,omitempty"`
}

func InfoContext(ctx context.Context, message string, fields ...interface{}) {
    entry := buildLogEntry(ctx, "INFO", message, fields...)
    writeLog(entry)
}

func ErrorContext(ctx context.Context, message string, fields ...interface{}) {
    entry := buildLogEntry(ctx, "ERROR", message, fields...)
    writeLog(entry)
}

func WarnContext(ctx context.Context, message string, fields ...interface{}) {
    entry := buildLogEntry(ctx, "WARN", message, fields...)
    writeLog(entry)
}

func buildLogEntry(ctx context.Context, level string, message string, fields ...interface{}) *LogEntry {
    entry := &LogEntry{
        Timestamp: time.Now().Format(time.RFC3339),
        Level:     level,
        Message:   message,
        Context:   make(map[string]interface{}),
    }
    
    // 从上下文提取信息
    if sessionID := ctx.Value("session_id"); sessionID != nil {
        entry.SessionID = sessionID.(string)
    }
    if userID := ctx.Value("user_id"); userID != nil {
        entry.UserID = userID.(string)
    }
    if tenantID := ctx.Value("tenant_id"); tenantID != nil {
        entry.TenantID = tenantID.(string)
    }
    
    // 添加额外字段
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fields[i].(string)
            value := fields[i+1]
            entry.Context[key] = value
        }
    }
    
    return entry
}

func writeLog(entry *LogEntry) {
    data, _ := json.Marshal(entry)
    log.Println(string(data))
}
```

### 性能追踪

```go
// internal/tracing/tracer.go
package tracing

import (
    "context"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

func TraceFlow(
    ctx context.Context,
    flowName string,
    fn func(context.Context) error,
) error {
    tracer := otel.Tracer("genkit-flows")
    ctx, span := tracer.Start(ctx, flowName)
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("flow.name", flowName),
    )
    
    // 从上下文提取信息
    if sessionID := ctx.Value("session_id"); sessionID != nil {
        span.SetAttributes(attribute.String("session.id", sessionID.(string)))
    }
    if tenantID := ctx.Value("tenant_id"); tenantID != nil {
        span.SetAttributes(attribute.String("tenant.id", tenantID.(string)))
    }
    
    // 执行 Flow
    err := fn(ctx)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    
    return err
}
```

## 测试策略

### 单元测试

#### Flow 测试示例

```go
// internal/genkit/flows/context_test.go
package flows_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "genkit-ai-service/internal/genkit/flows"
    "genkit-ai-service/internal/service/mocks"
)

func TestContextBuildFlow(t *testing.T) {
    // 准备
    mockContextSvc := new(mocks.MockContextService)
    mockContextSvc.On("BuildContext", mock.Anything, mock.Anything).
        Return(&service.ContextResult{
            SessionID:   "test-session-id",
            TotalTokens: 1000,
            Strategy:    "auto",
            QualityScore: 0.85,
        }, nil)
    
    // 执行
    input := flows.ContextBuildInput{
        SessionID:       "test-session-id",
        UserQuery:       "测试查询",
        MaxTokens:       4000,
        Strategy:        "auto",
        IncludeSummary:  true,
        IncludeLongTerm: true,
        ShortTermWindow: 10,
    }
    
    // 这里需要实际调用 Flow，但为了测试我们直接调用服务层
    result, err := mockContextSvc.BuildContext(context.Background(), service.BuildContextRequest{
        SessionID:       input.SessionID,
        UserQuery:       input.UserQuery,
        MaxTokens:       input.MaxTokens,
        Strategy:        input.Strategy,
        IncludeSummary:  input.IncludeSummary,
        IncludeLongTerm: input.IncludeLongTerm,
        ShortTermWindow: input.ShortTermWindow,
    })
    
    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "test-session-id", result.SessionID)
    assert.Equal(t, 1000, result.TotalTokens)
    assert.Greater(t, result.QualityScore, 0.7)
    
    mockContextSvc.AssertExpectations(t)
}
```

#### Service 测试示例

```go
// internal/service/context_service_test.go
package service_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "genkit-ai-service/internal/service"
)

func TestBuildContext(t *testing.T) {
    // 准备 Mock
    mockSessionRepo := new(mocks.MockSessionRepository)
    mockMessageRepo := new(mocks.MockMessageRepository)
    mockMemoryRepo := new(mocks.MockMemoryRepository)
    mockContextRepo := new(mocks.MockContextRepository)
    mockVectorSvc := new(mocks.MockVectorService)
    mockTokenMgr := new(mocks.MockTokenManager)
    mockCache := new(mocks.MockCacheService)
    
    // 创建服务
    svc := service.NewContextService(
        mockSessionRepo,
        mockMessageRepo,
        mockMemoryRepo,
        mockContextRepo,
        mockVectorSvc,
        mockTokenMgr,
        mockCache,
    )
    
    // 设置 Mock 行为
    mockMessageRepo.On("GetRecentMessages", mock.Anything, "session-id", 10).
        Return([]*model.ConversationMessage{}, nil)
    
    mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).
        Return(1000)
    
    // 执行
    result, err := svc.BuildContext(context.Background(), service.BuildContextRequest{
        SessionID:       "session-id",
        UserQuery:       "测试",
        MaxTokens:       4000,
        Strategy:        "auto",
        ShortTermWindow: 10,
    })
    
    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "session-id", result.SessionID)
}
```

### 集成测试

```go
// test/integration/context_flow_test.go
package integration_test

import (
    "context"
    "testing"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/stretchr/testify/assert"
    "genkit-ai-service/internal/genkit/flows"
)

func TestContextBuildFlowIntegration(t *testing.T) {
    // 跳过短测试
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    // 初始化测试环境
    ctx := context.Background()
    g := setupTestGenkit(t)
    defer teardownTestGenkit(t)
    
    // 注册 Flow
    flows.RegisterContextFlows(g, testContextService)
    
    // 查找 Flow
    flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
        g,
        "contextBuildFlow",
    )
    
    // 执行 Flow
    input := flows.ContextBuildInput{
        SessionID:       createTestSession(t),
        UserQuery:       "测试查询",
        MaxTokens:       4000,
        Strategy:        "auto",
        IncludeSummary:  true,
        IncludeLongTerm: true,
        ShortTermWindow: 10,
    }
    
    output, err := flow.Run(ctx, input)
    
    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, output)
    assert.NotEmpty(t, output.SessionID)
    assert.Greater(t, output.TotalTokens, 0)
    assert.Greater(t, output.QualityScore, 0.0)
    assert.Less(t, output.BuildTime, int64(500))
}
```

### 性能测试

```go
// test/benchmark/context_flow_bench_test.go
package benchmark_test

import (
    "context"
    "testing"
    
    "genkit-ai-service/internal/genkit/flows"
)

func BenchmarkContextBuildFlow(b *testing.B) {
    ctx := context.Background()
    g := setupBenchmarkGenkit(b)
    defer teardownBenchmarkGenkit(b)
    
    flows.RegisterContextFlows(g, benchContextService)
    
    flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
        g,
        "contextBuildFlow",
    )
    
    input := flows.ContextBuildInput{
        SessionID:       "bench-session-id",
        UserQuery:       "基准测试查询",
        MaxTokens:       4000,
        Strategy:        "auto",
        ShortTermWindow: 10,
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := flow.Run(ctx, input)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## 部署考虑

### 环境配置

#### 开发环境

```yaml
# config/dev.yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 5432
  database: genkit_dev
  user: postgres
  password: postgres
  max_connections: 10

redis:
  host: localhost
  port: 6379
  database: 0

genkit:
  provider: google
  api_key: ${GENAI_API_KEY}
  model: gemini-1.5-flash
  log_level: debug
```

#### 生产环境

```yaml
# config/prod.yaml
server:
  port: 8080
  mode: release

database:
  host: ${DB_HOST}
  port: 5432
  database: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  max_connections: 50
  ssl_mode: require

redis:
  host: ${REDIS_HOST}
  port: 6379
  password: ${REDIS_PASSWORD}
  database: 0

genkit:
  provider: google
  api_key: ${GENAI_API_KEY}
  model: gemini-1.5-flash
  log_level: info

monitoring:
  prometheus_port: 9090
  jaeger_endpoint: ${JAEGER_ENDPOINT}
```

### Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o /genkit-service ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 复制二进制文件
COPY --from=builder /genkit-service .

# 复制配置文件
COPY config ./config

EXPOSE 8080

CMD ["./genkit-service"]
```

### Kubernetes 部署

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: genkit-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: genkit-service
  template:
    metadata:
      labels:
        app: genkit-service
    spec:
      containers:
      - name: genkit-service
        image: genkit-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: host
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        - name: GENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: genai-secret
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 安全考虑

### 多租户隔离

1. **数据库层隔离**
   - 所有查询必须包含 tenant_id 过滤
   - 使用 Row Level Security (RLS) 策略
   - 定期审计跨租户访问尝试

2. **应用层隔离**
   - 在所有 Flow 中验证租户权限
   - 使用中间件强制租户过滤
   - 记录所有权限验证失败

3. **缓存层隔离**
   - 缓存键包含租户 ID
   - 防止缓存穿透攻击

### API 安全

1. **认证和授权**
   - JWT Token 验证
   - 基于角色的访问控制 (RBAC)
   - API 密钥管理

2. **输入验证**
   - 参数类型验证
   - 长度限制
   - SQL 注入防护
   - XSS 防护

3. **速率限制**
   - 基于租户的速率限制
   - 基于 IP 的速率限制
   - Token 配额管理

## 性能优化

### 数据库优化

1. **索引策略**
   - 复合索引：(tenant_id, session_id)
   - 部分索引：WHERE is_deleted = FALSE
   - 时间索引：created_at, expires_at

2. **查询优化**
   - 使用预编译语句
   - 批量操作
   - 连接池管理

3. **分区策略**
   - 按租户分区
   - 按时间分区（历史数据）

### 缓存优化

1. **缓存策略**
   - 多级缓存（本地 + Redis）
   - 缓存预热
   - 缓存穿透防护

2. **缓存失效**
   - 主动失效
   - TTL 管理
   - 版本控制

### Qdrant 向量检索优化

1. **Collection 配置优化**
   - 使用 HNSW 索引（Qdrant 默认，性能优于 IVFFlat）
   - 调整 HNSW 参数：
     - `m`: 16-32（连接数，影响召回率和内存）
     - `ef_construction`: 100-200（构建时搜索深度）
   - 配置分片数量：根据租户数量和数据量调整（建议 4-8 个分片）
   - 配置副本数量：生产环境建议 2-3 个副本

2. **租户隔离优化**
   - 使用 `is_tenant=true` 标记租户字段（Qdrant v1.11+ 优化）
   - 配置自定义分片策略按租户分布数据
   - 所有查询强制包含 `tenant_id` 过滤条件
   - 定期监控租户数据分布，必要时重新平衡

3. **查询优化**
   - 批量向量生成和插入
   - 实现向量查询结果缓存
   - 使用异步处理生成向量
   - 调整查询参数 `ef`（搜索时深度，默认等于 `ef_construction`）
   - 使用 Payload 索引加速过滤

4. **维护优化**
   - 定期执行 Collection 优化（compact）
   - 监控索引质量指标
   - 清理过期向量数据
   - 备份和恢复策略

## 总结

本设计文档描述了基于 Google Genkit Go SDK 的会话管理模块的完整技术设计，包括：

1. **架构设计**：分层架构，职责清晰
2. **数据模型**：支持三层记忆架构和向量检索
3. **Genkit Flow**：类型安全的工作流定义
4. **服务层**：业务逻辑实现和权限控制
5. **Repository 层**：数据访问抽象和租户过滤
6. **缓存设计**：多场景缓存策略和预热机制
7. **错误处理**：统一错误码、降级策略和熔断机制
8. **监控和可观测性**：指标收集、日志记录和链路追踪
9. **测试策略**：单元测试、集成测试和性能测试
10. **部署考虑**：环境配置、容器化和 Kubernetes 部署
11. **安全考虑**：多租户隔离、API 安全和输入验证
12. **性能优化**：数据库优化、缓存优化和 Qdrant 向量检索优化

该设计确保了系统的可扩展性、可维护性和高性能，同时提供了完整的多租户隔离和安全保障。

**Qdrant 多租户架构亮点**：

- 采用单 Collection + Payload 分区的官方推荐架构
- 使用 `is_tenant=true` 标记实现租户级别优化（Qdrant v1.11+）
- 自定义分片策略确保租户数据均匀分布
- 所有查询强制租户过滤，确保数据隔离
- HNSW 索引提供高性能向量检索
