# AI 对话会话管理系统设计文档

## 目录

- [1. 概述](#1-概述)
- [2. 核心架构](#2-核心架构)
- [3. 数据模型设计](#3-数据模型设计)
- [4. 会话记忆策略](#4-会话记忆策略)
- [5. 上下文管理服务](#5-上下文管理服务)
- [6. 向量存储集成](#6-向量存储集成)
- [7. Token 管理策略](#7-token-管理策略)
- [8. 自动摘要生成](#8-自动摘要生成)
- [9. 多租户权限控制](#9-多租户权限控制)
- [10. API 接口设计](#10-api-接口设计)
- [11. 性能优化](#11-性能优化)
- [12. 实施路线图](#12-实施路线图)

## 1. 概述

本文档描述了基于现有多租户架构的 AI 对话会话管理系统的完整设计方案。系统采用三层记忆架构（短期记忆、长期记忆、摘要记忆），结合向量检索技术，实现智能的对话上下文管理。

### 1.1 设计目标

- **智能上下文管理**：自动管理对话上下文，优化 token 使用
- **多层记忆架构**：短期、长期、摘要三层记忆，平衡性能和效果
- **租户数据隔离**：严格遵循多租户访问控制规范
- **高性能**：支持大规模并发，响应时间 < 200ms
- **可扩展性**：模块化设计，易于扩展新功能

### 1.2 技术栈

- **语言**：Go 1.21+
- **数据库**：PostgreSQL 14+ (with pgvector extension)
- **缓存**：Redis 7+
- **向量检索**：pgvector
- **ORM**：GORM
- **API框架**：Gin

## 2. 核心架构

### 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        API Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Session API  │  │ Context API  │  │ Memory API   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Session Service│ │Context Service│ │Summary Service│     │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Memory Service│  │Vector Service │  │Token Manager │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Session Repo  │  │Message Repo  │  │Memory Repo   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ PostgreSQL   │  │   pgvector   │  │    Redis     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 三层记忆架构

```
┌─────────────────────────────────────────────────────────────┐
│                    短期记忆 (Short-Term)                     │
│  - 最近 N 条消息（默认 10 条）                               │
│  - 直接从数据库查询                                          │
│  - 响应速度快，适合即时对话                                  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    长期记忆 (Long-Term)                      │
│  - 历史对话向量化存储                                        │
│  - 基于语义相似度检索                                        │
│  - 适合跨会话知识关联                                        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    摘要记忆 (Summary)                        │
│  - 定期压缩历史对话                                          │
│  - 保留关键信息，减少 token 消耗                             │
│  - 适合长对话场景                                            │
└─────────────────────────────────────────────────────────────┘
```

## 3. 数据模型设计

### 3.1 会话记忆表 (conversation_memories)

```go
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
    
    // 向量嵌入（1536维，OpenAI text-embedding-ada-002）
    Embedding []float32 `gorm:"type:vector(1536)" json:"-"`
    
    // Token 数量
    TokenCount int `gorm:"default:0" json:"tokenCount"`
    
    // 记忆起始消息ID
    StartMsgID *uuid.UUID `gorm:"type:uuid" json:"startMsgId"`
    
    // 记忆结束消息ID
    EndMsgID *uuid.UUID `gorm:"type:uuid" json:"endMsgId"`
    
    // 重要性评分（0.0-1.0）
    Importance float32 `gorm:"type:float;default:0.5" json:"importance"`
    
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

// TableName 指定表名
func (ConversationMemory) TableName() string {
    return "conversation_memories"
}
```

### 3.2 对话上下文表 (conversation_contexts)

```go
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
    
    // 上下文策略：sliding（滑动窗口）, summary（摘要）, hybrid（混合）
    Strategy string `gorm:"type:varchar(32);default:'hybrid'" json:"strategy"`
    
    // 当前 Token 使用量
    CurrentTokens int `gorm:"default:0" json:"currentTokens"`
    
    // 最后的摘要内容
    LastSummary *string `gorm:"type:text" json:"lastSummary"`
    
    // 最后摘要的消息ID
    LastSummaryMsgID *uuid.UUID `gorm:"type:uuid" json:"lastSummaryMsgId"`
    
    // 压缩次数
    CompressionCount int `gorm:"default:0" json:"compressionCount"`
    
    // 创建时间
    CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    
    // 更新时间
    UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    
    // 配置信息
    Config datatypes.JSON `gorm:"type:jsonb" json:"config"`
}

// TableName 指定表名
func (ConversationContext) TableName() string {
    return "conversation_contexts"
}
```

### 3.3 数据库迁移脚本

```sql
-- 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- 创建对话记忆表
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
    
    CONSTRAINT fk_session FOREIGN KEY (session_id) 
        REFERENCES chat_sessions(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_session_memories ON conversation_memories(session_id) WHERE is_deleted = false;
CREATE INDEX idx_tenant_memories ON conversation_memories(tenant_id) WHERE is_deleted = false;
CREATE INDEX idx_memory_type ON conversation_memories(memory_type);
CREATE INDEX idx_created ON conversation_memories(created_at);
CREATE INDEX idx_expires ON conversation_memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_deleted ON conversation_memories(is_deleted);

-- 创建向量索引（IVFFlat 算法，适合大规模数据）
CREATE INDEX idx_memory_embedding ON conversation_memories 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);

-- 创建对话上下文表
CREATE TABLE IF NOT EXISTS conversation_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    tenant_id UUID NOT NULL,
    window_size INTEGER DEFAULT 10,
    max_tokens INTEGER DEFAULT 4000,
    strategy VARCHAR(32) DEFAULT 'hybrid',
    current_tokens INTEGER DEFAULT 0,
    last_summary TEXT,
    last_summary_msg_id UUID,
    compression_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    config JSONB,
    
    CONSTRAINT fk_session_context FOREIGN KEY (session_id) 
        REFERENCES chat_sessions(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_session_context ON conversation_contexts(session_id);
CREATE INDEX idx_tenant_contexts ON conversation_contexts(tenant_id);

-- 创建更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_conversation_contexts_updated_at 
    BEFORE UPDATE ON conversation_contexts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

## 4. 会话记忆策略

### 4.1 记忆策略接口

```go
// MemoryStrategy 记忆策略接口
type MemoryStrategy interface {
    // GetRelevantContext 获取相关上下文
    GetRelevantContext(ctx context.Context, sessionID string, query string, maxTokens int) ([]ChatMessage, error)
    
    // StoreMessage 存储新消息
    StoreMessage(ctx context.Context, message *ChatMessage) error
    
    // GenerateSummary 生成摘要
    GenerateSummary(ctx context.Context, sessionID string, messages []ChatMessage) (*ChatSummary, error)
    
    // CleanupExpiredMemory 清理过期记忆
    CleanupExpiredMemory(ctx context.Context, sessionID string) error
    
    // UpdateImportance 更新记忆重要性
    UpdateImportance(ctx context.Context, memoryID string, importance float32) error
}
```

### 4.2 短期记忆实现

```go
// ShortTermMemory 短期记忆策略
type ShortTermMemory struct {
    messageRepo repository.MessageRepository
    windowSize  int // 默认 10 条消息
}

// NewShortTermMemory 创建短期记忆实例
func NewShortTermMemory(messageRepo repository.MessageRepository, windowSize int) *ShortTermMemory {
    if windowSize <= 0 {
        windowSize = 10
    }
    return &ShortTermMemory{
        messageRepo: messageRepo,
        windowSize:  windowSize,
    }
}

// GetRecentMessages 获取最近的消息
func (m *ShortTermMemory) GetRecentMessages(ctx context.Context, sessionID string) ([]ChatMessage, error) {
    messages, err := m.messageRepo.GetRecentMessages(ctx, sessionID, m.windowSize)
    if err != nil {
        return nil, fmt.Errorf("获取最近消息失败: %w", err)
    }
    return messages, nil
}

// StoreMessage 存储消息
func (m *ShortTermMemory) StoreMessage(ctx context.Context, message *ChatMessage) error {
    return m.messageRepo.Create(ctx, message)
}
```

### 4.3 长期记忆实现

```go
// LongTermMemory 长期记忆策略（基于向量检索）
type LongTermMemory struct {
    memoryRepo    repository.MemoryRepository
    vectorStore   VectorStore
    embeddingFunc func(text string) ([]float32, error)
    topK          int // 检索 top K 条相关记忆
}

// NewLongTermMemory 创建长期记忆实例
func NewLongTermMemory(
    memoryRepo repository.MemoryRepository,
    vectorStore VectorStore,
    embeddingFunc func(text string) ([]float32, error),
    topK int,
) *LongTermMemory {
    if topK <= 0 {
        topK = 5
    }
    return &LongTermMemory{
        memoryRepo:    memoryRepo,
        vectorStore:   vectorStore,
        embeddingFunc: embeddingFunc,
        topK:          topK,
    }
}

// SearchRelevantMemories 搜索相关记忆
func (m *LongTermMemory) SearchRelevantMemories(
    ctx context.Context,
    sessionID string,
    query string,
) ([]*ConversationMemory, error) {
    // 1. 生成查询向量
    queryEmbedding, err := m.embeddingFunc(query)
    if err != nil {
        return nil, fmt.Errorf("生成查询向量失败: %w", err)
    }
    
    // 2. 向量相似度搜索
    memories, err := m.vectorStore.Search(ctx, sessionID, queryEmbedding, m.topK)
    if err != nil {
        return nil, fmt.Errorf("向量搜索失败: %w", err)
    }
    
    // 3. 更新访问统计
    for _, memory := range memories {
        m.updateAccessStats(ctx, memory.ID)
    }
    
    return memories, nil
}

// StoreMemory 存储长期记忆
func (m *LongTermMemory) StoreMemory(ctx context.Context, memory *ConversationMemory) error {
    // 1. 生成向量嵌入
    embedding, err := m.embeddingFunc(memory.Content)
    if err != nil {
        return fmt.Errorf("生成向量嵌入失败: %w", err)
    }
    memory.Embedding = embedding
    
    // 2. 存储到数据库
    if err := m.memoryRepo.Create(ctx, memory); err != nil {
        return fmt.Errorf("存储记忆失败: %w", err)
    }
    
    return nil
}

// updateAccessStats 更新访问统计
func (m *LongTermMemory) updateAccessStats(ctx context.Context, memoryID uuid.UUID) {
    go func() {
        now := time.Now()
        fields := map[string]interface{}{
            "access_count":   gorm.Expr("access_count + 1"),
            "last_access_at": now,
        }
        m.memoryRepo.UpdateFields(context.Background(), memoryID.String(), fields)
    }()
}
```

### 4.4 摘要记忆实现

```go
// SummaryMemory 摘要记忆策略
type SummaryMemory struct {
    summaryRepo     repository.SummaryRepository
    messageRepo     repository.MessageRepository
    aiService       AIService
    summaryInterval int // 每 N 条消息生成一次摘要
}

// NewSummaryMemory 创建摘要记忆实例
func NewSummaryMemory(
    summaryRepo repository.SummaryRepository,
    messageRepo repository.MessageRepository,
    aiService AIService,
    summaryInterval int,
) *SummaryMemory {
    if summaryInterval <= 0 {
        summaryInterval = 20 // 默认每 20 条消息生成摘要
    }
    return &SummaryMemory{
        summaryRepo:     summaryRepo,
        messageRepo:     messageRepo,
        aiService:       aiService,
        summaryInterval: summaryInterval,
    }
}

// ShouldGenerateSummary 判断是否需要生成摘要
func (m *SummaryMemory) ShouldGenerateSummary(ctx context.Context, sessionID string) (bool, error) {
    // 1. 获取最后的摘要
    lastSummary, err := m.summaryRepo.GetLatest(ctx, sessionID)
    if err != nil && !errors.IsNotFoundError(err) {
        return false, err
    }
    
    // 2. 如果没有摘要，检查消息数量
    if lastSummary == nil {
        count, err := m.messageRepo.CountBySession(ctx, sessionID)
        if err != nil {
            return false, err
        }
        return count >= m.summaryInterval, nil
    }
    
    // 3. 检查自上次摘要后的新消息数量
    newCount, err := m.messageRepo.CountAfterMessage(ctx, sessionID, lastSummary.LastMessageID)
    if err != nil {
        return false, err
    }
    
    return newCount >= m.summaryInterval, nil
}

// GenerateSummary 生成摘要
func (m *SummaryMemory) GenerateSummary(ctx context.Context, sessionID string) (*ChatSummary, error) {
    // 1. 获取需要摘要的消息
    lastSummary, _ := m.summaryRepo.GetLatest(ctx, sessionID)
    
    var startMsgID *uuid.UUID
    var previousSummary string
    if lastSummary != nil {
        startMsgID = &lastSummary.LastMessageID
        previousSummary = lastSummary.Summary
    }
    
    messages, err := m.messageRepo.GetMessagesAfter(ctx, sessionID, startMsgID)
    if err != nil {
        return nil, fmt.Errorf("获取消息失败: %w", err)
    }
    
    if len(messages) < m.summaryInterval {
        return nil, fmt.Errorf("消息数量不足，无需生成摘要")
    }
    
    // 2. 构建摘要提示词
    prompt := m.buildSummaryPrompt(previousSummary, messages)
    
    // 3. 调用 AI 生成摘要
    summaryText, err := m.aiService.GenerateSummary(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("生成摘要失败: %w", err)
    }
    
    // 4. 保存摘要
    summary := &ChatSummary{
        SessionID:     uuid.MustParse(sessionID),
        Summary:       summaryText,
        LastMessageID: messages[len(messages)-1].ID,
        TokenCount:    m.countTokens(summaryText),
        CreatedAt:     time.Now(),
    }
    
    if err := m.summaryRepo.Create(ctx, summary); err != nil {
        return nil, fmt.Errorf("保存摘要失败: %w", err)
    }
    
    return summary, nil
}

// buildSummaryPrompt 构建摘要提示词
func (m *SummaryMemory) buildSummaryPrompt(previousSummary string, messages []ChatMessage) string {
    var prompt strings.Builder
    
    prompt.WriteString("请对以下对话内容生成简洁的摘要，保留关键信息和上下文。\n\n")
    
    if previousSummary != "" {
        prompt.WriteString("之前的对话摘要：\n")
        prompt.WriteString(previousSummary)
        prompt.WriteString("\n\n")
    }
    
    prompt.WriteString("新的对话内容：\n")
    for _, msg := range messages {
        prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
    }
    
    prompt.WriteString("\n请生成摘要（200字以内）：")
    
    return prompt.String()
}

// countTokens 计算 token 数量（简化实现）
func (m *SummaryMemory) countTokens(text string) int {
    // 简化实现：中文按字符数，英文按单词数
    // 实际应使用 tiktoken 等库
    return len([]rune(text))
}
```

## 5. 上下文管理服务

### 5.1 上下文服务接口

```go
// ContextService 上下文管理服务接口
type ContextService interface {
    // BuildContext 构建对话上下文
    BuildContext(ctx context.Context, sessionID, userQuery string) (*ContextResponse, error)
    
    // UpdateContext 更新上下文
    UpdateContext(ctx context.Context, sessionID string, newMessages []ChatMessage) error
    
    // CompressContext 压缩上下文
    CompressContext(ctx context.Context, sessionID string) error
    
    // GetContextConfig 获取上下文配置
    GetContextConfig(ctx context.Context, sessionID string) (*ConversationContext, error)
    
    // UpdateContextConfig 更新上下文配置
    UpdateContextConfig(ctx context.Context, sessionID string, config *ContextConfig) error
}
```

### 5.2 上下文服务实现

```go
// contextService 上下文管理服务实现
type contextService struct {
    sessionRepo     repository.SessionRepository
    messageRepo     repository.MessageRepository
    memoryRepo      repository.MemoryRepository
    contextRepo     repository.ContextRepository
    shortTermMemory *ShortTermMemory
    longTermMemory  *LongTermMemory
    summaryMemory   *SummaryMemory
    tokenManager    *TokenManager
}

// NewContextService 创建上下文服务实例
func NewContextService(
    sessionRepo repository.SessionRepository,
    messageRepo repository.MessageRepository,
    memoryRepo repository.MemoryRepository,
    contextRepo repository.ContextRepository,
    shortTermMemory *ShortTermMemory,
    longTermMemory *LongTermMemory,
    summaryMemory *SummaryMemory,
    tokenManager *TokenManager,
) ContextService {
    return &contextService{
        sessionRepo:     sessionRepo,
        messageRepo:     messageRepo,
        memoryRepo:      memoryRepo,
        contextRepo:     contextRepo,
        shortTermMemory: shortTermMemory,
        longTermMemory:  longTermMemory,
        summaryMemory:   summaryMemory,
        tokenManager:    tokenManager,
    }
}

// BuildContext 构建对话上下文
func (s *contextService) BuildContext(ctx context.Context, sessionID, userQuery string) (*ContextResponse, error) {
    // 1. 权限验证
    if err := s.validateAccess(ctx, sessionID); err != nil {
        return nil, err
    }
    
    // 2. 获取上下文配置
    contextConfig, err := s.getOrCreateContextConfig(ctx, sessionID)
    if err != nil {
        return nil, fmt.Errorf("获取上下文配置失败: %w", err)
    }
    
    // 3. 获取短期记忆（最近的消息）
    recentMessages, err := s.shortTermMemory.GetRecentMessages(ctx, sessionID)
    if err != nil {
        return nil, fmt.Errorf("获取短期记忆失败: %w", err)
    }
    
    // 4. 获取长期记忆（相关的历史对话）
    var relevantMemories []*ConversationMemory
    if userQuery != "" {
        relevantMemories, err = s.longTermMemory.SearchRelevantMemories(ctx, sessionID, userQuery)
        if err != nil {
            logger.WarnContext(ctx, "获取长期记忆失败", "error", err)
            // 不中断流程，继续执行
        }
    }
    
    // 5. 获取摘要记忆
    var summary *ChatSummary
    if contextConfig.Strategy == "summary" || contextConfig.Strategy == "hybrid" {
        summary, _ = s.summaryMemory.summaryRepo.GetLatest(ctx, sessionID)
    }
    
    // 6. 组合上下文
    response := s.combineContext(contextConfig, recentMessages, relevantMemories, summary)
    
    // 7. Token 优化
    response = s.tokenManager.OptimizeContext(response, contextConfig.MaxTokens)
    
    return response, nil
}

// combineContext 组合上下文
func (s *contextService) combineContext(
    config *ConversationContext,
    recentMessages []ChatMessage,
    relevantMemories []*ConversationMemory,
    summary *ChatSummary,
) *ContextResponse {
    response := &ContextResponse{
        SessionID: config.SessionID.String(),
        Strategy:  config.Strategy,
        Messages:  make([]MessageContext, 0),
    }
    
    // 1. 添加摘要（如果存在）
    if summary != nil {
        response.Summary = &SummaryContext{
            Content:    summary.Summary,
            TokenCount: summary.TokenCount,
            CreatedAt:  summary.CreatedAt.Format(time.RFC3339),
        }
    }
    
    // 2. 添加相关的长期记忆
    if len(relevantMemories) > 0 {
        response.RelevantMemories = make([]MemoryContext, 0, len(relevantMemories))
        for _, mem := range relevantMemories {
            response.RelevantMemories = append(response.RelevantMemories, MemoryContext{
                Content:    mem.Content,
                TokenCount: mem.TokenCount,
                Importance: mem.Importance,
                CreatedAt:  mem.CreatedAt.Format(time.RFC3339),
            })
        }
    }
    
    // 3. 添加最近的消息
    for _, msg := range recentMessages {
        response.Messages = append(response.Messages, MessageContext{
            ID:         msg.ID.String(),
            Role:       msg.Role,
            Content:    msg.Content,
            TokenCount: msg.Tokens,
            CreatedAt:  msg.CreatedAt.Format(time.RFC3339),
        })
    }
    
    // 4. 计算总 token 数
    response.TotalTokens = s.calculateTotalTokens(response)
    
    return response
}

// UpdateContext 更新上下文
func (s *contextService) UpdateContext(ctx context.Context, sessionID string, newMessages []ChatMessage) error {
    // 1. 权限验证
    if err := s.validateAccess(ctx, sessionID); err != nil {
        return err
    }
    
    // 2. 存储新消息到短期记忆
    for _, msg := range newMessages {
        if err := s.shortTermMemory.StoreMessage(ctx, &msg); err != nil {
            return fmt.Errorf("存储消息失败: %w", err)
        }
    }
    
    // 3. 异步生成向量并存储到长期记忆
    go s.storeToLongTermMemoryAsync(sessionID, newMessages)
    
    // 4. 检查是否需要生成摘要
    shouldSummarize, _ := s.summaryMemory.ShouldGenerateSummary(ctx, sessionID)
    if shouldSummarize {
        go s.generateSummaryAsync(sessionID)
    }
    
    // 5. 更新上下文配置
    contextConfig, _ := s.contextRepo.GetBySessionID(ctx, sessionID)
    if contextConfig != nil {
        fields := map[string]interface{}{
            "current_tokens": s.calculateCurrentTokens(ctx, sessionID),
            "updated_at":     time.Now(),
        }
        s.contextRepo.UpdateFields(ctx, contextConfig.ID.String(), fields)
    }
    
    return nil
}

// CompressContext 压缩上下文
func (s *contextService) CompressContext(ctx context.Context, sessionID string) error {
    // 1. 权限验证
    if err := s.validateAccess(ctx, sessionID); err != nil {
        return err
    }
    
    // 2. 生成摘要
    summary, err := s.summaryMemory.GenerateSummary(ctx, sessionID)
    if err != nil {
        return fmt.Errorf("生成摘要失败: %w", err)
    }
    
    // 3. 更新上下文配置
    contextConfig, _ := s.contextRepo.GetBySessionID(ctx, sessionID)
    if contextConfig != nil {
        fields := map[string]interface{}{
            "last_summary":        summary.Summary,
            "last_summary_msg_id": summary.LastMessageID,
            "compression_count":   gorm.Expr("compression_count + 1"),
            "updated_at":          time.Now(),
        }
        s.contextRepo.UpdateFields(ctx, contextConfig.ID.String(), fields)
    }
    
    logger.InfoContext(ctx, "上下文压缩成功",
        "session_id", sessionID,
        "summary_tokens", summary.TokenCount,
    )
    
    return nil
}

// validateAccess 验证访问权限（多租户隔离）
func (s *contextService) validateAccess(ctx context.Context, sessionID string) error {
    // 1. 获取 JWT 声明
    claims := middleware.GetJWTClaims(ctx)
    if claims == nil {
        return errors.NewUnauthorizedError("未认证")
    }
    
    // 2. 查询会话
    session, err := s.sessionRepo.GetByID(ctx, sessionID)
    if err != nil {
        return errors.NewSessionNotFoundError(sessionID)
    }
    
    // 3. 平台管理员可以访问所有会话
    if hasRole(claims, model.RoleSystemAdmin) {
        return nil
    }
    
    // 4. 租户管理员和普通用户只能访问自己租户的会话
    user, err := s.userRepo.GetByID(ctx, claims.Subject)
    if err != nil {
        return errors.NewUnauthorizedError("用户不存在")
    }
    
    // 获取会话所属用户的租户ID
    sessionUser, err := s.userRepo.GetByID(ctx, session.UserID.String())
    if err != nil {
        return errors.NewInternalError(fmt.Errorf("获取会话用户失败: %w", err))
    }
    
    if user.TenantID != sessionUser.TenantID {
        logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话",
            "user_id", claims.Subject,
            "user_tenant_id", user.TenantID,
            "session_id", sessionID,
            "session_tenant_id", sessionUser.TenantID,
        )
        return errors.NewForbiddenError("权限不足：无法访问其他租户的会话")
    }
    
    return nil
}

// getOrCreateContextConfig 获取或创建上下文配置
func (s *contextService) getOrCreateContextConfig(ctx context.Context, sessionID string) (*ConversationContext, error) {
    // 尝试获取现有配置
    config, err := s.contextRepo.GetBySessionID(ctx, sessionID)
    if err == nil {
        return config, nil
    }
    
    // 如果不存在，创建默认配置
    session, err := s.sessionRepo.GetByID(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    sessionUser, err := s.userRepo.GetByID(ctx, session.UserID.String())
    if err != nil {
        return nil, err
    }
    
    config = &ConversationContext{
        SessionID:     session.ID,
        TenantID:      sessionUser.TenantID,
        WindowSize:    10,
        MaxTokens:     4000,
        Strategy:      "hybrid",
        CurrentTokens: 0,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }
    
    if err := s.contextRepo.Create(ctx, config); err != nil {
        return nil, err
    }
    
    return config, nil
}

// storeToLongTermMemoryAsync 异步存储到长期记忆
func (s *contextService) storeToLongTermMemoryAsync(sessionID string, messages []ChatMessage) {
    ctx := context.Background()
    
    for _, msg := range messages {
        // 只存储用户和助手的消息
        if msg.Role != "user" && msg.Role != "assistant" {
            continue
        }
        
        memory := &ConversationMemory{
            SessionID:  uuid.MustParse(sessionID),
            MemoryType: "long_term",
            Content:    msg.Content,
            TokenCount: msg.Tokens,
            StartMsgID: &msg.ID,
            EndMsgID:   &msg.ID,
            Importance: 0.5,
            CreatedAt:  time.Now(),
        }
        
        if err := s.longTermMemory.StoreMemory(ctx, memory); err != nil {
            logger.ErrorContext(ctx, "存储长期记忆失败",
                "session_id", sessionID,
                "message_id", msg.ID,
                "error", err,
            )
        }
    }
}

// generateSummaryAsync 异步生成摘要
func (s *contextService) generateSummaryAsync(sessionID string) {
    ctx := context.Background()
    
    if _, err := s.summaryMemory.GenerateSummary(ctx, sessionID); err != nil {
        logger.ErrorContext(ctx, "生成摘要失败",
            "session_id", sessionID,
            "error", err,
        )
    }
}

// calculateTotalTokens 计算总 token 数
func (s *contextService) calculateTotalTokens(response *ContextResponse) int {
    total := 0
    
    if response.Summary != nil {
        total += response.Summary.TokenCount
    }
    
    for _, mem := range response.RelevantMemories {
        total += mem.TokenCount
    }
    
    for _, msg := range response.Messages {
        total += msg.TokenCount
    }
    
    return total
}

// calculateCurrentTokens 计算当前会话的 token 使用量
func (s *contextService) calculateCurrentTokens(ctx context.Context, sessionID string) int {
    messages, err := s.messageRepo.GetBySessionID(ctx, sessionID)
    if err != nil {
        return 0
    }
    
    total := 0
    for _, msg := range messages {
        total += msg.Tokens
    }
    
    return total
}
```

### 5.3 响应数据结构

```go
// ContextResponse 上下文响应
type ContextResponse struct {
    SessionID        string           `json:"sessionId"`
    Strategy         string           `json:"strategy"`
    Summary          *SummaryContext  `json:"summary,omitempty"`
    RelevantMemories []MemoryContext  `json:"relevantMemories,omitempty"`
    Messages         []MessageContext `json:"messages"`
    TotalTokens      int              `json:"totalTokens"`
}

// SummaryContext 摘要上下文
type SummaryContext struct {
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
}

// MemoryContext 记忆上下文
type MemoryContext struct {
    Content    string  `json:"content"`
    TokenCount int     `json:"tokenCount"`
    Importance float32 `json:"importance"`
    CreatedAt  string  `json:"createdAt"`
}

// MessageContext 消息上下文
type MessageContext struct {
    ID         string `json:"id"`
    Role       string `json:"role"`
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
}

// ContextConfig 上下文配置
type ContextConfig struct {
    WindowSize int    `json:"windowSize"`
    MaxTokens  int    `json:"maxTokens"`
    Strategy   string `json:"strategy"`
}
```

## 6. 向量存储集成

### 6.1 向量存储接口

```go
// VectorStore 向量存储接口
type VectorStore interface {
    // Store 存储向量
    Store(ctx context.Context, memory *ConversationMemory) error
    
    // Search 向量相似度搜索
    Search(ctx context.Context, sessionID string, queryEmbedding []float32, topK int) ([]*ConversationMemory, error)
    
    // BatchStore 批量存储向量
    BatchStore(ctx context.Context, memories []*ConversationMemory) error
    
    // Delete 删除向量
    Delete(ctx context.Context, memoryID string) error
    
    // UpdateEmbedding 更新向量
    UpdateEmbedding(ctx context.Context, memoryID string, embedding []float32) error
}
```

### 6.2 pgvector 实现

```go
// pgVectorStore pgvector 向量存储实现
type pgVectorStore struct {
    db            *gorm.DB
    embeddingFunc func(text string) ([]float32, error)
}

// NewPgVectorStore 创建 pgvector 存储实例
func NewPgVectorStore(db *gorm.DB, embeddingFunc func(text string) ([]float32, error)) VectorStore {
    return &pgVectorStore{
        db:            db,
        embeddingFunc: embeddingFunc,
    }
}

// Store 存储向量
func (s *pgVectorStore) Store(ctx context.Context, memory *ConversationMemory) error {
    // 1. 生成向量嵌入
    if memory.Embedding == nil || len(memory.Embedding) == 0 {
        embedding, err := s.embeddingFunc(memory.Content)
        if err != nil {
            return fmt.Errorf("生成向量嵌入失败: %w", err)
        }
        memory.Embedding = embedding
    }
    
    // 2. 存储到数据库
    if err := s.db.WithContext(ctx).Create(memory).Error; err != nil {
        return fmt.Errorf("存储向量失败: %w", err)
    }
    
    return nil
}

// Search 向量相似度搜索
func (s *pgVectorStore) Search(
    ctx context.Context,
    sessionID string,
    queryEmbedding []float32,
    topK int,
) ([]*ConversationMemory, error) {
    var memories []*ConversationMemory
    
    // 使用余弦相似度搜索
    // <=> 是 pgvector 的余弦距离操作符
    // 1 - 余弦距离 = 余弦相似度
    err := s.db.WithContext(ctx).
        Raw(`
            SELECT 
                id, session_id, tenant_id, memory_type, content,
                token_count, start_msg_id, end_msg_id, importance,
                access_count, last_access_at, created_at, expires_at,
                is_deleted, meta,
                1 - (embedding <=> ?::vector) as similarity
            FROM conversation_memories
            WHERE session_id = ?::uuid 
                AND is_deleted = false
                AND (expires_at IS NULL OR expires_at > NOW())
            ORDER BY embedding <=> ?::vector
            LIMIT ?
        `,
            pgvector.NewVector(queryEmbedding),
            sessionID,
            pgvector.NewVector(queryEmbedding),
            topK,
        ).
        Scan(&memories).Error
    
    if err != nil {
        return nil, fmt.Errorf("向量搜索失败: %w", err)
    }
    
    return memories, nil
}

// BatchStore 批量存储向量
func (s *pgVectorStore) BatchStore(ctx context.Context, memories []*ConversationMemory) error {
    // 1. 批量生成向量嵌入
    for i, memory := range memories {
        if memory.Embedding == nil || len(memory.Embedding) == 0 {
            embedding, err := s.embeddingFunc(memory.Content)
            if err != nil {
                logger.WarnContext(ctx, "生成向量嵌入失败",
                    "index", i,
                    "error", err,
                )
                continue
            }
            memories[i].Embedding = embedding
        }
    }
    
    // 2. 批量插入
    if err := s.db.WithContext(ctx).CreateInBatches(memories, 100).Error; err != nil {
        return fmt.Errorf("批量存储向量失败: %w", err)
    }
    
    return nil
}

// Delete 删除向量（软删除）
func (s *pgVectorStore) Delete(ctx context.Context, memoryID string) error {
    err := s.db.WithContext(ctx).
        Model(&ConversationMemory{}).
        Where("id = ?", memoryID).
        Update("is_deleted", true).Error
    
    if err != nil {
        return fmt.Errorf("删除向量失败: %w", err)
    }
    
    return nil
}

// UpdateEmbedding 更新向量
func (s *pgVectorStore) UpdateEmbedding(ctx context.Context, memoryID string, embedding []float32) error {
    err := s.db.WithContext(ctx).
        Model(&ConversationMemory{}).
        Where("id = ?", memoryID).
        Update("embedding", pgvector.NewVector(embedding)).Error
    
    if err != nil {
        return fmt.Errorf("更新向量失败: %w", err)
    }
    
    return nil
}
```

### 6.3 向量嵌入服务

```go
// EmbeddingService 向量嵌入服务接口
type EmbeddingService interface {
    // GenerateEmbedding 生成单个文本的向量
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
    
    // GenerateBatchEmbeddings 批量生成向量
    GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// openAIEmbeddingService OpenAI 向量嵌入服务
type openAIEmbeddingService struct {
    client *openai.Client
    model  string // text-embedding-ada-002 或 text-embedding-3-small
}

// NewOpenAIEmbeddingService 创建 OpenAI 向量嵌入服务
func NewOpenAIEmbeddingService(apiKey, model string) EmbeddingService {
    if model == "" {
        model = "text-embedding-ada-002"
    }
    
    return &openAIEmbeddingService{
        client: openai.NewClient(apiKey),
        model:  model,
    }
}

// GenerateEmbedding 生成单个文本的向量
func (s *openAIEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
    // 文本预处理
    text = strings.TrimSpace(text)
    if text == "" {
        return nil, fmt.Errorf("文本不能为空")
    }
    
    // 限制文本长度（OpenAI 限制 8191 tokens）
    if len(text) > 30000 {
        text = text[:30000]
    }
    
    // 调用 OpenAI API
    resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Input: []string{text},
        Model: openai.EmbeddingModel(s.model),
    })
    
    if err != nil {
        return nil, fmt.Errorf("调用 OpenAI API 失败: %w", err)
    }
    
    if len(resp.Data) == 0 {
        return nil, fmt.Errorf("未返回向量数据")
    }
    
    return resp.Data[0].Embedding, nil
}

// GenerateBatchEmbeddings 批量生成向量
func (s *openAIEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
    if len(texts) == 0 {
        return nil, fmt.Errorf("文本列表不能为空")
    }
    
    // OpenAI 批量限制为 2048 个
    if len(texts) > 2048 {
        return nil, fmt.Errorf("批量文本数量超过限制")
    }
    
    // 文本预处理
    processedTexts := make([]string, 0, len(texts))
    for _, text := range texts {
        text = strings.TrimSpace(text)
        if text == "" {
            continue
        }
        if len(text) > 30000 {
            text = text[:30000]
        }
        processedTexts = append(processedTexts, text)
    }
    
    // 调用 OpenAI API
    resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Input: processedTexts,
        Model: openai.EmbeddingModel(s.model),
    })
    
    if err != nil {
        return nil, fmt.Errorf("调用 OpenAI API 失败: %w", err)
    }
    
    // 提取向量
    embeddings := make([][]float32, len(resp.Data))
    for i, data := range resp.Data {
        embeddings[i] = data.Embedding
    }
    
    return embeddings, nil
}
```

### 6.4 向量索引优化

```sql
-- IVFFlat 索引（适合大规模数据，速度快但精度略低）
CREATE INDEX idx_memory_embedding_ivfflat ON conversation_memories 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);

-- HNSW 索引（精度高但构建慢，适合中等规模数据）
CREATE INDEX idx_memory_embedding_hnsw ON conversation_memories 
USING hnsw (embedding vector_cosine_ops) 
WITH (m = 16, ef_construction = 64);

-- 选择索引策略：
-- 1. 数据量 < 100万：使用 HNSW
-- 2. 数据量 > 100万：使用 IVFFlat
-- 3. 需要实时更新：使用 IVFFlat
-- 4. 查询精度要求高：使用 HNSW
```

## 7. Token 管理策略

### 7.1 Token 管理器

```go
// TokenManager Token 管理器
type TokenManager struct {
    maxContextTokens int     // 最大上下文 token 数
    reservedTokens   int     // 为响应预留的 token 数
    compressionRatio float64 // 压缩比例阈值
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
    }
}

// OptimizeContext 优化上下文 token 使用
func (tm *TokenManager) OptimizeContext(response *ContextResponse, maxTokens int) *ContextResponse {
    if maxTokens <= 0 {
        maxTokens = tm.maxContextTokens
    }
    
    // 减去预留的响应 token
    availableTokens := maxTokens - tm.reservedTokens
    
    if response.TotalTokens <= availableTokens {
        return response // 无需优化
    }
    
    // 优化策略：
    // 1. 保留摘要（如果存在）
    // 2. 减少相关记忆数量
    // 3. 减少消息数量
    
    optimized := &ContextResponse{
        SessionID: response.SessionID,
        Strategy:  response.Strategy,
        Summary:   response.Summary,
    }
    
    currentTokens := 0
    if response.Summary != nil {
        currentTokens += response.Summary.TokenCount
    }
    
    // 添加相关记忆（按重要性排序）
    if len(response.RelevantMemories) > 0 {
        sortedMemories := tm.sortMemoriesByImportance(response.RelevantMemories)
        for _, mem := range sortedMemories {
            if currentTokens+mem.TokenCount > availableTokens {
                break
            }
            optimized.RelevantMemories = append(optimized.RelevantMemories, mem)
            currentTokens += mem.TokenCount
        }
    }
    
    // 添加最近的消息（从最新开始）
    for i := len(response.Messages) - 1; i >= 0; i-- {
        msg := response.Messages[i]
        if currentTokens+msg.TokenCount > availableTokens {
            break
        }
        optimized.Messages = append([]MessageContext{msg}, optimized.Messages...)
        currentTokens += msg.TokenCount
    }
    
    optimized.TotalTokens = currentTokens
    
    return optimized
}

// ShouldCompress 判断是否需要压缩
func (tm *TokenManager) ShouldCompress(currentTokens, maxTokens int) bool {
    if maxTokens <= 0 {
        maxTokens = tm.maxContextTokens
    }
    
    threshold := float64(maxTokens) * tm.compressionRatio
    return float64(currentTokens) > threshold
}

// EstimateTokens 估算文本的 token 数量
func (tm *TokenManager) EstimateTokens(text string) int {
    // 简化实现：
    // 中文：1 字符 ≈ 1.5 tokens
    // 英文：1 单词 ≈ 1.3 tokens
    // 实际应使用 tiktoken 库
    
    runes := []rune(text)
    chineseCount := 0
    englishCount := 0
    
    for _, r := range runes {
        if r >= 0x4e00 && r <= 0x9fff {
            chineseCount++
        } else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
            englishCount++
        }
    }
    
    // 英文单词数估算
    words := len(strings.Fields(text))
    
    return int(float64(chineseCount)*1.5 + float64(words)*1.3)
}

// sortMemoriesByImportance 按重要性排序记忆
func (tm *TokenManager) sortMemoriesByImportance(memories []MemoryContext) []MemoryContext {
    sorted := make([]MemoryContext, len(memories))
    copy(sorted, memories)
    
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].Importance > sorted[j].Importance
    })
    
    return sorted
}

// CalculateModelMaxTokens 根据模型计算最大 token 数
func (tm *TokenManager) CalculateModelMaxTokens(modelName string) int {
    // 不同模型的上下文窗口大小
    modelLimits := map[string]int{
        "gpt-4":              8192,
        "gpt-4-32k":          32768,
        "gpt-4-turbo":        128000,
        "gpt-3.5-turbo":      4096,
        "gpt-3.5-turbo-16k":  16384,
        "claude-2":           100000,
        "claude-instant":     100000,
    }
    
    if limit, ok := modelLimits[modelName]; ok {
        return limit
    }
    
    return tm.maxContextTokens // 默认值
}
```

### 7.2 Token 计数服务

```go
// TokenCounterService Token 计数服务接口
type TokenCounterService interface {
    // CountTokens 计算文本的 token 数量
    CountTokens(text string, model string) (int, error)
    
    // CountMessagesTokens 计算消息列表的 token 数量
    CountMessagesTokens(messages []ChatMessage, model string) (int, error)
}

// tiktokenCounterService 使用 tiktoken 的计数服务
type tiktokenCounterService struct {
    encoders map[string]*tiktoken.Tiktoken
}

// NewTiktokenCounterService 创建 tiktoken 计数服务
func NewTiktokenCounterService() (TokenCounterService, error) {
    service := &tiktokenCounterService{
        encoders: make(map[string]*tiktoken.Tiktoken),
    }
    
    // 预加载常用编码器
    models := []string{"gpt-4", "gpt-3.5-turbo"}
    for _, model := range models {
        encoder, err := tiktoken.EncodingForModel(model)
        if err != nil {
            return nil, fmt.Errorf("加载编码器失败: %w", err)
        }
        service.encoders[model] = encoder
    }
    
    return service, nil
}

// CountTokens 计算文本的 token 数量
func (s *tiktokenCounterService) CountTokens(text string, model string) (int, error) {
    encoder, ok := s.encoders[model]
    if !ok {
        var err error
        encoder, err = tiktoken.EncodingForModel(model)
        if err != nil {
            return 0, fmt.Errorf("获取编码器失败: %w", err)
        }
        s.encoders[model] = encoder
    }
    
    tokens := encoder.Encode(text, nil, nil)
    return len(tokens), nil
}

// CountMessagesTokens 计算消息列表的 token 数量
func (s *tiktokenCounterService) CountMessagesTokens(messages []ChatMessage, model string) (int, error) {
    totalTokens := 0
    
    for _, msg := range messages {
        // 消息格式的额外 token
        totalTokens += 4 // 每条消息的格式开销
        
        // 角色 token
        roleTokens, _ := s.CountTokens(msg.Role, model)
        totalTokens += roleTokens
        
        // 内容 token
        contentTokens, _ := s.CountTokens(msg.Content, model)
        totalTokens += contentTokens
    }
    
    totalTokens += 2 // 对话的起始和结束标记
    
    return totalTokens, nil
}
```

## 8. 自动摘要生成

### 8.1 摘要生成服务

```go
// SummaryGenerationService 摘要生成服务接口
type SummaryGenerationService interface {
    // GenerateSummary 生成摘要
    GenerateSummary(ctx context.Context, prompt string) (string, error)
    
    // GenerateIncrementalSummary 生成增量摘要
    GenerateIncrementalSummary(ctx context.Context, previousSummary string, newContent string) (string, error)
}

// aiSummaryService AI 摘要生成服务
type aiSummaryService struct {
    aiClient AIClient
    model    string
}

// NewAISummaryService 创建 AI 摘要服务
func NewAISummaryService(aiClient AIClient, model string) SummaryGenerationService {
    if model == "" {
        model = "gpt-3.5-turbo"
    }
    return &aiSummaryService{
        aiClient: aiClient,
        model:    model,
    }
}

// GenerateSummary 生成摘要
func (s *aiSummaryService) GenerateSummary(ctx context.Context, prompt string) (string, error) {
    messages := []openai.ChatCompletionMessage{
        {
            Role:    "system",
            Content: "你是一个专业的对话摘要助手。请生成简洁、准确的对话摘要，保留关键信息和上下文。",
        },
        {
            Role:    "user",
            Content: prompt,
        },
    }
    
    resp, err := s.aiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       s.model,
        Messages:    messages,
        Temperature: 0.3, // 较低的温度以保证稳定性
        MaxTokens:   500,
    })
    
    if err != nil {
        return "", fmt.Errorf("生成摘要失败: %w", err)
    }
    
    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("未返回摘要内容")
    }
    
    return resp.Choices[0].Message.Content, nil
}

// GenerateIncrementalSummary 生成增量摘要
func (s *aiSummaryService) GenerateIncrementalSummary(
    ctx context.Context,
    previousSummary string,
    newContent string,
) (string, error) {
    prompt := fmt.Sprintf(`
之前的对话摘要：
%s

新的对话内容：
%s

请基于之前的摘要和新的对话内容，生成更新后的完整摘要。要求：
1. 保留之前摘要中的关键信息
2. 整合新对话的重要内容
3. 保持摘要简洁（200字以内）
4. 突出对话的主题和结论
`, previousSummary, newContent)
    
    return s.GenerateSummary(ctx, prompt)
}
```

### 8.2 摘要策略配置

```go
// SummaryConfig 摘要配置
type SummaryConfig struct {
    // 触发摘要的消息数量阈值
    MessageThreshold int `json:"messageThreshold"`
    
    // 触发摘要的 token 数量阈值
    TokenThreshold int `json:"tokenThreshold"`
    
    // 摘要的最大长度（字符数）
    MaxSummaryLength int `json:"maxSummaryLength"`
    
    // 是否启用增量摘要
    EnableIncremental bool `json:"enableIncremental"`
    
    // 摘要保留时间（小时）
    RetentionHours int `json:"retentionHours"`
}

// DefaultSummaryConfig 默认摘要配置
func DefaultSummaryConfig() *SummaryConfig {
    return &SummaryConfig{
        MessageThreshold:  20,
        TokenThreshold:    3000,
        MaxSummaryLength:  500,
        EnableIncremental: true,
        RetentionHours:    720, // 30 天
    }
}
```

## 9. 多租户权限控制

### 9.1 权限验证中间件

```go
// RequireSessionAccess 会话访问权限验证中间件
func RequireSessionAccess() gin.HandlerFunc {
    return func(c *gin.Context) {
        sessionID := c.Param("sessionId")
        if sessionID == "" {
            response.Error(c, errors.NewBadRequestError("会话ID不能为空"))
            c.Abort()
            return
        }
        
        // 获取 JWT 声明
        claims := GetJWTClaims(c)
        if claims == nil {
            response.Error(c, errors.NewUnauthorizedError("未认证"))
            c.Abort()
            return
        }
        
        // 验证会话访问权限
        sessionService := c.MustGet("sessionService").(session.SessionService)
        if err := sessionService.ValidateAccess(c.Request.Context(), sessionID, claims.Subject); err != nil {
            logger.WarnContext(c.Request.Context(), "会话访问权限验证失败",
                "session_id", sessionID,
                "user_id", claims.Subject,
                "error", err,
            )
            response.Error(c, err)
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### 9.2 租户隔离查询

```go
// MemoryRepository 记忆仓储接口
type MemoryRepository interface {
    // Create 创建记忆
    Create(ctx context.Context, memory *ConversationMemory) error
    
    // GetByID 根据ID获取记忆（带租户验证）
    GetByID(ctx context.Context, memoryID string, tenantID string) (*ConversationMemory, error)
    
    // ListBySession 获取会话的记忆列表（带租户验证）
    ListBySession(ctx context.Context, sessionID string, tenantID string, pageNo, pageSize int) ([]*ConversationMemory, int64, error)
    
    // Delete 删除记忆（软删除，带租户验证）
    Delete(ctx context.Context, memoryID string, tenantID string) error
}

// memoryRepository 记忆仓储实现
type memoryRepository struct {
    db *gorm.DB
}

// GetByID 根据ID获取记忆（带租户验证）
func (r *memoryRepository) GetByID(ctx context.Context, memoryID string, tenantID string) (*ConversationMemory, error) {
    var memory ConversationMemory
    
    err := r.db.WithContext(ctx).
        Where("id = ? AND tenant_id = ? AND is_deleted = false", memoryID, tenantID).
        First(&memory).Error
    
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.NewNotFoundError("记忆不存在")
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
    pageNo, pageSize int,
) ([]*ConversationMemory, int64, error) {
    var memories []*ConversationMemory
    var total int64
    
    query := r.db.WithContext(ctx).
        Model(&ConversationMemory{}).
        Where("session_id = ? AND tenant_id = ? AND is_deleted = false", sessionID, tenantID)
    
    // 计算总数
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    // 分页查询
    offset := (pageNo - 1) * pageSize
    err := query.
        Order("created_at DESC").
        Limit(pageSize).
        Offset(offset).
        Find(&memories).Error
    
    if err != nil {
        return nil, 0, err
    }
    
    return memories, total, nil
}
```

### 9.3 审计日志

```go
// LogMemoryAccess 记录记忆访问日志
func LogMemoryAccess(ctx context.Context, action string, memoryID string, success bool, err error) {
    claims := middleware.GetJWTClaims(ctx)
    
    fields := []interface{}{
        "event", "memory_access",
        "action", action,
        "memory_id", memoryID,
        "success", success,
    }
    
    if claims != nil {
        fields = append(fields,
            "user_id", claims.Subject,
            "tenant_id", claims.TenantID,
        )
    }
    
    if err != nil {
        fields = append(fields, "error", err.Error())
        logger.WarnContext(ctx, "记忆访问失败", fields...)
    } else {
        logger.InfoContext(ctx, "记忆访问成功", fields...)
    }
}

// LogContextBuild 记录上下文构建日志
func LogContextBuild(ctx context.Context, sessionID string, tokenCount int, duration time.Duration) {
    claims := middleware.GetJWTClaims(ctx)
    
    fields := []interface{}{
        "event", "context_build",
        "session_id", sessionID,
        "token_count", tokenCount,
        "duration_ms", duration.Milliseconds(),
    }
    
    if claims != nil {
        fields = append(fields,
            "user_id", claims.Subject,
            "tenant_id", claims.TenantID,
        )
    }
    
    logger.InfoContext(ctx, "上下文构建完成", fields...)
}
```

## 10. API 接口设计

### 10.1 上下文管理接口

```go
// ContextHandler 上下文处理器
type ContextHandler struct {
    contextService ContextService
}

// NewContextHandler 创建上下文处理器
func NewContextHandler(contextService ContextService) *ContextHandler {
    return &ContextHandler{
        contextService: contextService,
    }
}

// GetContext 获取会话上下文
// @Summary 获取会话上下文
// @Description 获取会话的对话上下文，包括短期记忆、长期记忆和摘要
// @Tags 上下文管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Param query query string false "用户查询（用于检索相关记忆）"
// @Success 200 {object} response.ResponseData[ContextResponse]
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Failure 404 {object} response.ResponseData[any] "会话不存在"
// @Router /api/v1/sessions/{sessionId}/context [get]
func (h *ContextHandler) GetContext(c *gin.Context) {
    sessionID := c.Param("sessionId")
    userQuery := c.Query("query")
    
    startTime := time.Now()
    
    context, err := h.contextService.BuildContext(c.Request.Context(), sessionID, userQuery)
    if err != nil {
        response.Error(c, err)
        return
    }
    
    duration := time.Since(startTime)
    LogContextBuild(c.Request.Context(), sessionID, context.TotalTokens, duration)
    
    response.Success(c, context)
}

// CompressContext 压缩会话上下文
// @Summary 压缩会话上下文
// @Description 生成会话摘要，压缩历史对话以节省 token
// @Tags 上下文管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Success 200 {object} response.ResponseData[string]
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Failure 404 {object} response.ResponseData[any] "会话不存在"
// @Router /api/v1/sessions/{sessionId}/context/compress [post]
func (h *ContextHandler) CompressContext(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    if err := h.contextService.CompressContext(c.Request.Context(), sessionID); err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, "上下文压缩成功")
}

// GetContextConfig 获取上下文配置
// @Summary 获取上下文配置
// @Description 获取会话的上下文管理配置
// @Tags 上下文管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Success 200 {object} response.ResponseData[ConversationContext]
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Failure 404 {object} response.ResponseData[any] "会话不存在"
// @Router /api/v1/sessions/{sessionId}/context/config [get]
func (h *ContextHandler) GetContextConfig(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    config, err := h.contextService.GetContextConfig(c.Request.Context(), sessionID)
    if err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, config)
}

// UpdateContextConfig 更新上下文配置
// @Summary 更新上下文配置
// @Description 更新会话的上下文管理配置
// @Tags 上下文管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Param config body ContextConfig true "上下文配置"
// @Success 200 {object} response.ResponseData[string]
// @Failure 400 {object} response.ResponseData[any] "请求参数错误"
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Failure 404 {object} response.ResponseData[any] "会话不存在"
// @Router /api/v1/sessions/{sessionId}/context/config [put]
func (h *ContextHandler) UpdateContextConfig(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    var config ContextConfig
    if err := c.ShouldBindJSON(&config); err != nil {
        response.Error(c, errors.NewBadRequestError("请求参数错误"))
        return
    }
    
    if err := h.contextService.UpdateContextConfig(c.Request.Context(), sessionID, &config); err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, "配置更新成功")
}
```

### 10.2 记忆管理接口

```go
// MemoryHandler 记忆处理器
type MemoryHandler struct {
    memoryService MemoryService
}

// ListMemories 获取会话记忆列表
// @Summary 获取会话记忆列表
// @Description 获取指定会话的记忆列表，支持分页和类型过滤
// @Tags 记忆管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Param memoryType query string false "记忆类型" Enums(short_term, long_term, summary)
// @Param pageNo query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} response.ResponsePaginationData[[]ConversationMemory]
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Router /api/v1/sessions/{sessionId}/memories [get]
func (h *MemoryHandler) ListMemories(c *gin.Context) {
    sessionID := c.Param("sessionId")
    memoryType := c.Query("memoryType")
    pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
    
    memories, total, err := h.memoryService.ListMemories(
        c.Request.Context(),
        sessionID,
        memoryType,
        pageNo,
        pageSize,
    )
    
    if err != nil {
        response.Error(c, err)
        return
    }
    
    response.SuccessWithPagination(c, memories, pageNo, pageSize, int(total))
}

// SearchMemories 搜索相关记忆
// @Summary 搜索相关记忆
// @Description 基于语义相似度搜索相关的历史记忆
// @Tags 记忆管理
// @Security BearerAuth
// @Param sessionId path string true "会话ID"
// @Param query query string true "搜索查询"
// @Param topK query int false "返回数量" default(5)
// @Success 200 {object} response.ResponseData[[]ConversationMemory]
// @Failure 400 {object} response.ResponseData[any] "请求参数错误"
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Router /api/v1/sessions/{sessionId}/memories/search [get]
func (h *MemoryHandler) SearchMemories(c *gin.Context) {
    sessionID := c.Param("sessionId")
    query := c.Query("query")
    topK, _ := strconv.Atoi(c.DefaultQuery("topK", "5"))
    
    if query == "" {
        response.Error(c, errors.NewBadRequestError("搜索查询不能为空"))
        return
    }
    
    memories, err := h.memoryService.SearchMemories(c.Request.Context(), sessionID, query, topK)
    if err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, memories)
}

// DeleteMemory 删除记忆
// @Summary 删除记忆
// @Description 删除指定的记忆（软删除）
// @Tags 记忆管理
// @Security BearerAuth
// @Param memoryId path string true "记忆ID"
// @Success 200 {object} response.ResponseData[string]
// @Failure 401 {object} response.ResponseData[any] "未认证"
// @Failure 403 {object} response.ResponseData[any] "权限不足"
// @Failure 404 {object} response.ResponseData[any] "记忆不存在"
// @Router /api/v1/memories/{memoryId} [delete]
func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
    memoryID := c.Param("memoryId")
    
    if err := h.memoryService.DeleteMemory(c.Request.Context(), memoryID); err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, "记忆删除成功")
}
```

### 10.3 路由注册

```go
// RegisterContextRoutes 注册上下文管理路由
func RegisterContextRoutes(r *gin.RouterGroup, handler *ContextHandler) {
    context := r.Group("/sessions/:sessionId/context")
    context.Use(middleware.RequireAuth())
    context.Use(middleware.RequireSessionAccess())
    {
        context.GET("", handler.GetContext)
        context.POST("/compress", handler.CompressContext)
        context.GET("/config", handler.GetContextConfig)
        context.PUT("/config", handler.UpdateContextConfig)
    }
}

// RegisterMemoryRoutes 注册记忆管理路由
func RegisterMemoryRoutes(r *gin.RouterGroup, handler *MemoryHandler) {
    // 会话记忆路由
    sessionMemories := r.Group("/sessions/:sessionId/memories")
    sessionMemories.Use(middleware.RequireAuth())
    sessionMemories.Use(middleware.RequireSessionAccess())
    {
        sessionMemories.GET("", handler.ListMemories)
        sessionMemories.GET("/search", handler.SearchMemories)
    }
    
    // 记忆操作路由
    memories := r.Group("/memories")
    memories.Use(middleware.RequireAuth())
    {
        memories.DELETE("/:memoryId", handler.DeleteMemory)
    }
}
```

## 11. 性能优化

### 11.1 缓存策略

```go
// CachedContextService 带缓存的上下文服务
type CachedContextService struct {
    contextService ContextService
    cache          *redis.Client
    cacheTTL       time.Duration
}

// NewCachedContextService 创建带缓存的上下文服务
func NewCachedContextService(
    contextService ContextService,
    cache *redis.Client,
    cacheTTL time.Duration,
) ContextService {
    if cacheTTL <= 0 {
        cacheTTL = 5 * time.Minute
    }
    
    return &CachedContextService{
        contextService: contextService,
        cache:          cache,
        cacheTTL:       cacheTTL,
    }
}

// BuildContext 构建上下文（带缓存）
func (s *CachedContextService) BuildContext(
    ctx context.Context,
    sessionID, userQuery string,
) (*ContextResponse, error) {
    // 1. 尝试从缓存获取
    cacheKey := fmt.Sprintf("context:%s", sessionID)
    if userQuery == "" {
        cached, err := s.cache.Get(ctx, cacheKey).Result()
        if err == nil {
            var context ContextResponse
            if json.Unmarshal([]byte(cached), &context) == nil {
                logger.DebugContext(ctx, "从缓存获取上下文", "session_id", sessionID)
                return &context, nil
            }
        }
    }
    
    // 2. 缓存未命中，构建上下文
    context, err := s.contextService.BuildContext(ctx, sessionID, userQuery)
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存（仅当没有查询时）
    if userQuery == "" {
        data, _ := json.Marshal(context)
        s.cache.Set(ctx, cacheKey, data, s.cacheTTL)
    }
    
    return context, nil
}

// UpdateContext 更新上下文（清除缓存）
func (s *CachedContextService) UpdateContext(
    ctx context.Context,
    sessionID string,
    newMessages []ChatMessage,
) error {
    // 1. 更新上下文
    if err := s.contextService.UpdateContext(ctx, sessionID, newMessages); err != nil {
        return err
    }
    
    // 2. 清除缓存
    cacheKey := fmt.Sprintf("context:%s", sessionID)
    s.cache.Del(ctx, cacheKey)
    
    return nil
}
```

### 11.2 异步处理

```go
// AsyncMemoryProcessor 异步记忆处理器
type AsyncMemoryProcessor struct {
    longTermMemory *LongTermMemory
    summaryMemory  *SummaryMemory
    workerPool     *WorkerPool
}

// NewAsyncMemoryProcessor 创建异步记忆处理器
func NewAsyncMemoryProcessor(
    longTermMemory *LongTermMemory,
    summaryMemory *SummaryMemory,
    workerCount int,
) *AsyncMemoryProcessor {
    return &AsyncMemoryProcessor{
        longTermMemory: longTermMemory,
        summaryMemory:  summaryMemory,
        workerPool:     NewWorkerPool(workerCount),
    }
}

// ProcessMessageAsync 异步处理消息
func (p *AsyncMemoryProcessor) ProcessMessageAsync(sessionID string, message *ChatMessage) {
    p.workerPool.Submit(func() {
        ctx := context.Background()
        
        // 1. 存储到长期记忆
        memory := &ConversationMemory{
            SessionID:  uuid.MustParse(sessionID),
            MemoryType: "long_term",
            Content:    message.Content,
            TokenCount: message.Tokens,
            StartMsgID: &message.ID,
            EndMsgID:   &message.ID,
            Importance: 0.5,
            CreatedAt:  time.Now(),
        }
        
        if err := p.longTermMemory.StoreMemory(ctx, memory); err != nil {
            logger.ErrorContext(ctx, "存储长期记忆失败",
                "session_id", sessionID,
                "message_id", message.ID,
                "error", err,
            )
        }
        
        // 2. 检查是否需要生成摘要
        shouldSummarize, _ := p.summaryMemory.ShouldGenerateSummary(ctx, sessionID)
        if shouldSummarize {
            if _, err := p.summaryMemory.GenerateSummary(ctx, sessionID); err != nil {
                logger.ErrorContext(ctx, "生成摘要失败",
                    "session_id", sessionID,
                    "error", err,
                )
            }
        }
    })
}

// WorkerPool 工作池
type WorkerPool struct {
    tasks chan func()
    wg    sync.WaitGroup
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workerCount int) *WorkerPool {
    pool := &WorkerPool{
        tasks: make(chan func(), 1000),
    }
    
    for i := 0; i < workerCount; i++ {
        pool.wg.Add(1)
        go pool.worker()
    }
    
    return pool
}

// Submit 提交任务
func (p *WorkerPool) Submit(task func()) {
    p.tasks <- task
}

// worker 工作协程
func (p *WorkerPool) worker() {
    defer p.wg.Done()
    
    for task := range p.tasks {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    logger.Error("工作协程panic", "error", r)
                }
            }()
            task()
        }()
    }
}

// Close 关闭工作池
func (p *WorkerPool) Close() {
    close(p.tasks)
    p.wg.Wait()
}
```

### 11.3 批量操作优化

```go
// BatchMemoryStore 批量记忆存储
type BatchMemoryStore struct {
    vectorStore VectorStore
    batchSize   int
    flushTicker *time.Ticker
    buffer      []*ConversationMemory
    mu          sync.Mutex
}

// NewBatchMemoryStore 创建批量记忆存储
func NewBatchMemoryStore(vectorStore VectorStore, batchSize int, flushInterval time.Duration) *BatchMemoryStore {
    store := &BatchMemoryStore{
        vectorStore: vectorStore,
        batchSize:   batchSize,
        flushTicker: time.NewTicker(flushInterval),
        buffer:      make([]*ConversationMemory, 0, batchSize),
    }
    
    go store.autoFlush()
    
    return store
}

// Add 添加记忆到缓冲区
func (s *BatchMemoryStore) Add(memory *ConversationMemory) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.buffer = append(s.buffer, memory)
    
    if len(s.buffer) >= s.batchSize {
        s.flush()
    }
}

// flush 刷新缓冲区
func (s *BatchMemoryStore) flush() {
    if len(s.buffer) == 0 {
        return
    }
    
    ctx := context.Background()
    if err := s.vectorStore.BatchStore(ctx, s.buffer); err != nil {
        logger.ErrorContext(ctx, "批量存储记忆失败",
            "count", len(s.buffer),
            "error", err,
        )
    }
    
    s.buffer = s.buffer[:0]
}

// autoFlush 自动刷新
func (s *BatchMemoryStore) autoFlush() {
    for range s.flushTicker.C {
        s.mu.Lock()
        s.flush()
        s.mu.Unlock()
    }
}

// Close 关闭批量存储
func (s *BatchMemoryStore) Close() {
    s.flushTicker.Stop()
    s.mu.Lock()
    s.flush()
    s.mu.Unlock()
}
```

### 11.4 数据库查询优化

```sql
-- 创建复合索引优化查询
CREATE INDEX idx_memories_session_type_deleted 
ON conversation_memories(session_id, memory_type, is_deleted) 
WHERE is_deleted = false;

-- 创建部分索引优化过期数据查询
CREATE INDEX idx_memories_expires 
ON conversation_memories(expires_at) 
WHERE expires_at IS NOT NULL AND is_deleted = false;

-- 创建索引优化访问统计查询
CREATE INDEX idx_memories_access 
ON conversation_memories(last_access_at DESC) 
WHERE is_deleted = false;

-- 分区表优化（按租户分区）
CREATE TABLE conversation_memories_partitioned (
    LIKE conversation_memories INCLUDING ALL
) PARTITION BY HASH (tenant_id);

-- 创建分区
CREATE TABLE conversation_memories_p0 PARTITION OF conversation_memories_partitioned
    FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE conversation_memories_p1 PARTITION OF conversation_memories_partitioned
    FOR VALUES WITH (MODULUS 4, REMAINDER 1);
CREATE TABLE conversation_memories_p2 PARTITION OF conversation_memories_partitioned
    FOR VALUES WITH (MODULUS 4, REMAINDER 2);
CREATE TABLE conversation_memories_p3 PARTITION OF conversation_memories_partitioned
    FOR VALUES WITH (MODULUS 4, REMAINDER 3);
```

## 12. 实施路线图

### 阶段1：基础功能（第1-2周）

**目标**：实现基本的会话上下文管理

**任务清单**：
- [x] 完善会话和消息数据模型
- [ ] 实现短期记忆（滑动窗口）
- [ ] 实现 Token 计数和限制
- [ ] 实现基本的上下文构建服务
- [ ] 添加上下文管理 API 接口
- [ ] 编写单元测试

**验收标准**：
- 能够获取最近 N 条消息作为上下文
- Token 数量控制在指定范围内
- API 接口正常工作

### 阶段2：摘要功能（第3周）

**目标**：实现自动摘要生成

**任务清单**：
- [ ] 创建摘要数据模型和表
- [ ] 实现摘要生成服务
- [ ] 实现增量摘要更新
- [ ] 添加摘要触发策略
- [ ] 实现摘要存储和查询
- [ ] 添加摘要管理 API

**验收标准**：
- 每 20 条消息自动生成摘要
- 摘要内容准确简洁
- 支持增量更新摘要

### 阶段3：向量检索（第4-5周）

**目标**：实现基于向量的长期记忆

**任务清单**：
- [ ] 安装配置 pgvector 扩展
- [ ] 创建记忆数据模型和表
- [ ] 集成向量嵌入服务（OpenAI）
- [ ] 实现向量存储和检索
- [ ] 实现相似度搜索
- [ ] 添加记忆管理 API
- [ ] 性能测试和优化

**验收标准**：
- 向量检索准确率 > 85%
- 检索响应时间 < 100ms
- 支持跨会话知识关联

### 阶段4：优化和监控（第6周）

**目标**：性能优化和生产就绪

**任务清单**：
- [ ] 添加 Redis 缓存层
- [ ] 实现异步处理
- [ ] 实现批量操作
- [ ] 添加性能监控指标
- [ ] 优化数据库查询
- [ ] 添加清理任务
- [ ] 压力测试

**验收标准**：
- 上下文构建响应时间 < 200ms
- 支持 1000+ 并发请求
- 缓存命中率 > 70%

### 阶段5：高级功能（第7-8周）

**目标**：扩展功能和用户体验

**任务清单**：
- [ ] 实现记忆重要性评分
- [ ] 实现记忆过期策略
- [ ] 添加记忆搜索功能
- [ ] 实现上下文策略配置
- [ ] 添加数据导出功能
- [ ] 完善文档和示例

**验收标准**：
- 支持多种上下文策略
- 用户可自定义配置
- 完整的 API 文档

## 13. 监控指标

### 13.1 性能指标

```go
// ContextMetrics 上下文性能指标
type ContextMetrics struct {
    // 上下文构建次数
    BuildCount prometheus.Counter
    
    // 上下文构建耗时
    BuildDuration prometheus.Histogram
    
    // Token 使用量
    TokenUsage prometheus.Histogram
    
    // 缓存命中率
    CacheHitRate prometheus.Gauge
    
    // 向量检索耗时
    VectorSearchDuration prometheus.Histogram
    
    // 摘要生成次数
    SummaryGenerationCount prometheus.Counter
}

// RecordContextBuild 记录上下文构建
func (m *ContextMetrics) RecordContextBuild(duration time.Duration, tokenCount int, cacheHit bool) {
    m.BuildCount.Inc()
    m.BuildDuration.Observe(duration.Seconds())
    m.TokenUsage.Observe(float64(tokenCount))
    
    if cacheHit {
        m.CacheHitRate.Set(1)
    } else {
        m.CacheHitRate.Set(0)
    }
}
```

### 13.2 业务指标

- 平均上下文 Token 数
- 摘要生成频率
- 向量检索准确率
- 记忆存储增长率
- 用户活跃会话数

### 13.3 告警规则

```yaml
# Prometheus 告警规则
groups:
  - name: context_alerts
    rules:
      - alert: HighContextBuildLatency
        expr: histogram_quantile(0.95, context_build_duration_seconds) > 1
        for: 5m
        annotations:
          summary: "上下文构建延迟过高"
          
      - alert: LowCacheHitRate
        expr: avg_over_time(context_cache_hit_rate[5m]) < 0.5
        for: 10m
        annotations:
          summary: "缓存命中率过低"
          
      - alert: HighTokenUsage
        expr: avg(context_token_usage) > 3500
        for: 5m
        annotations:
          summary: "Token 使用量接近上限"
```

## 14. 最佳实践

### 14.1 上下文策略选择

- **滑动窗口（sliding）**：适合短对话，响应快
- **摘要（summary）**：适合长对话，节省 token
- **混合（hybrid）**：推荐，平衡性能和效果

### 14.2 Token 优化建议

1. 合理设置窗口大小（默认 10 条）
2. 及时生成摘要（每 20 条消息）
3. 使用向量检索替代全量历史
4. 根据模型调整 token 限制

### 14.3 向量检索优化

1. 选择合适的嵌入模型
2. 定期更新向量索引
3. 控制检索数量（topK=5）
4. 过滤低相似度结果

### 14.4 安全注意事项

1. 严格执行租户隔离
2. 记录所有访问日志
3. 定期清理过期数据
4. 加密敏感信息

## 15. 常见问题

### Q1: 如何选择合适的上下文策略？

**A**: 根据对话长度和 token 预算选择：
- 短对话（< 20 条）：使用 sliding
- 长对话（> 50 条）：使用 summary 或 hybrid
- 需要历史关联：使用 hybrid

### Q2: 向量检索的准确率如何保证？

**A**: 
- 使用高质量的嵌入模型
- 合理设置相似度阈值
- 定期评估检索效果
- 结合用户反馈优化

### Q3: 如何处理大量历史消息？

**A**:
- 定期生成摘要压缩
- 设置记忆过期时间
- 使用分区表存储
- 实施数据归档策略

### Q4: 性能瓶颈在哪里？

**A**: 主要瓶颈：
- 向量嵌入生成（异步处理）
- 向量相似度搜索（优化索引）
- 摘要生成（批量处理）
- 数据库查询（添加索引）

## 16. 参考资源

### 文档
- [pgvector 官方文档](https://github.com/pgvector/pgvector)
- [OpenAI Embeddings API](https://platform.openai.com/docs/guides/embeddings)
- [GORM 文档](https://gorm.io/docs/)

### 相关论文
- "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks"
- "Long-Term Memory in Conversational AI"

### 开源项目
- LangChain Memory
- LlamaIndex
- Semantic Kernel

---

**文档版本**: v1.0  
**最后更新**: 2025-10-29  
**维护者**: AI Platform Team
