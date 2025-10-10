# 设计文档

## 概述

本文档描述了AI知识管理平台的技术架构设计。平台采用前后端分离架构，支持RAG、模型蒸馏、SFT等功能，优先使用云厂商提供的托管服务，以轻量化MVP方式实现。

### 核心设计原则

- **云原生优先**：优先使用阿里云、微软Azure等云厂商的托管服务
- **适配器模式**：关键组件采用适配器模式，支持未来替换
- **微服务架构**：初期单体部署，预留微服务拆分扩展点
- **异步处理**：长时间任务采用异步队列处理
- **多租户隔离**：项目级数据隔离和权限控制

## 架构

### 高层架构

```mermaid
graph TB
    subgraph "前端层"
        FE[React18 + TypeScript + Ant Design]
    end
    
    subgraph "API网关层"
        GW[API Gateway + JWT认证]
    end
    
    subgraph "应用服务层"
        AUTH[认证服务]
        PROJECT[项目管理服务]
        DOC[文档处理服务]
        VECTOR[向量服务]
        LLM[LLM管理服务]
        AGENT[Agent服务]
        TASK[任务管理服务]
        CHAT[对话服务]
    end
    
    subgraph "数据存储层"
        PG[(PostgreSQL)]
        REDIS[(Redis)]
        OSS[(对象存储)]
        VECTOR_DB[(ADB-PG向量库)]
    end
    
    subgraph "外部服务"
        LLM_API[LLM API提供商]
        EMBED_API[Embedding API]
        TRAIN_API[训练API - 阿里百炼]
    end
    
    FE --> GW
    GW --> AUTH
    GW --> PROJECT
    GW --> DOC
    GW --> VECTOR
    GW --> LLM
    GW --> AGENT
    GW --> TASK
    GW --> CHAT
    
    AUTH --> PG
    PROJECT --> PG
    DOC --> PG
    DOC --> OSS
    VECTOR --> VECTOR_DB
    LLM --> LLM_API
    TASK --> REDIS
    
    DOC --> EMBED_API
    AGENT --> LLM_API
    CHAT --> LLM_API
    TASK --> TRAIN_API
```

### MVP微服务清单

#### 1. 认证服务 (auth-service)

- **职责**：用户认证、JWT令牌管理、RBAC权限控制
- **技术栈**：Go + JWT + PostgreSQL
- **API端点**：`/api/v1/auth/*`

#### 2. 项目管理服务 (project-service)

- **职责**：项目CRUD、成员管理、数据隔离
- **技术栈**：Go + PostgreSQL
- **API端点**：`/api/v1/projects/*`

#### 3. 文档处理服务 (document-service)

- **职责**：文件上传、格式解析、智能分块
- **技术栈**：Go + OSS + 文档解析库
- **API端点**：`/api/v1/documents/*`

#### 4. 向量服务 (vector-service)

- **职责**：向量化、索引管理、相似度检索
- **技术栈**：Go + ADB-PG + Embedding API
- **API端点**：`/api/v1/vectors/*`

#### 5. LLM管理服务 (llm-service)

- **职责**：LLM提供商管理、模型配置、API调用
- **技术栈**：Go + KMS + 多LLM适配器
- **API端点**：`/api/v1/llm/*`

#### 6. Agent服务 (agent-service)

- **职责**：Agent创建管理、工具调用、推理链
- **技术栈**：Go + PostgreSQL + LLM API
- **API端点**：`/api/v1/agents/*`

#### 7. 任务管理服务 (task-service)

- **职责**：异步任务调度、状态跟踪、队列管理
- **技术栈**：Go + Redis + asynq
- **API端点**：`/api/v1/tasks/*`

#### 8. 对话服务 (chat-service)

- **职责**：多轮对话、上下文管理、RAG增强
- **技术栈**：Go + PostgreSQL + LLM API
- **API端点**：`/api/v1/chat/*`

## 组件和接口

### 核心组件设计

#### 1. VectorProvider 抽象接口

```go
type VectorProvider interface {
    // 创建向量索引
    CreateIndex(ctx context.Context, req *CreateIndexRequest) error
    
    // 插入向量
    InsertVectors(ctx context.Context, vectors []Vector) error
    
    // 相似度检索
    Search(ctx context.Context, query Vector, topK int, filters map[string]interface{}) ([]SearchResult, error)
    
    // 删除向量
    DeleteVectors(ctx context.Context, ids []string) error
    
    // 更新向量
    UpdateVectors(ctx context.Context, vectors []Vector) error
    
    // 获取统计信息
    GetStats(ctx context.Context, indexName string) (*IndexStats, error)
}

// ADB-PG适配器实现
type ADBPGProvider struct {
    client *sql.DB
    config *ADBPGConfig
}

func (p *ADBPGProvider) CreateIndex(ctx context.Context, req *CreateIndexRequest) error {
    // 使用ADB-PG的向量索引创建语法
    query := `CREATE INDEX IF NOT EXISTS %s ON %s USING ann(embedding) WITH (dim=%d, distancemeasure='%s')`
    // 实现细节...
}
```

#### 2. LLMProvider 抽象接口

```go
type LLMProvider interface {
    // 文本生成
    GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
    
    // 流式生成
    GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamResponse, error)
    
    // 获取模型列表
    ListModels(ctx context.Context) ([]Model, error)
    
    // 健康检查
    HealthCheck(ctx context.Context) error
}

// 支持的提供商
type ProviderType string

const (
    ProviderOpenAI    ProviderType = "openai"
    ProviderAzure     ProviderType = "azure"
    ProviderQianwen   ProviderType = "qianwen"
    ProviderClaude    ProviderType = "claude"
)
```

#### 3. DocumentProcessor 接口

```go
type DocumentProcessor interface {
    // 支持的文件类型
    SupportedTypes() []string
    
    // 解析文档
    Parse(ctx context.Context, file io.Reader, metadata *FileMetadata) (*Document, error)
    
    // 智能分块
    Chunk(ctx context.Context, doc *Document, config *ChunkConfig) ([]Chunk, error)
}

// 分块配置
type ChunkConfig struct {
    Strategy    ChunkStrategy `json:"strategy"`    // 分块策略
    MaxSize     int          `json:"max_size"`    // 最大字符数
    Overlap     int          `json:"overlap"`     // 重叠字符数
    Separators  []string     `json:"separators"` // 分隔符
    PreserveContext bool     `json:"preserve_context"` // 保持上下文
}

type ChunkStrategy string

const (
    ChunkByFixedSize   ChunkStrategy = "fixed_size"
    ChunkBySemantic    ChunkStrategy = "semantic"
    ChunkByStructure   ChunkStrategy = "structure"
    ChunkByCode        ChunkStrategy = "code"
)
```

#### 4. TaskWorker 接口

```go
type TaskWorker interface {
    // 处理任务
    Process(ctx context.Context, task *Task) error
    
    // 任务类型
    TaskType() string
    
    // 最大重试次数
    MaxRetries() int
}

// 任务类型定义
const (
    TaskTypeDocumentProcess = "document_process"
    TaskTypeQuestionGen     = "question_generate"
    TaskTypeAnswerGen       = "answer_generate"
    TaskTypeVectorIndex     = "vector_index"
    TaskTypeDatasetExport   = "dataset_export"
    TaskTypeModelTrain      = "model_train"
)
```

## 数据模型

### 核心数据表设计

#### 1. 项目相关表

```sql
-- 项目表
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL,
    settings JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 项目成员表
CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 2. 文档相关表

```sql
-- 文件表
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    oss_path VARCHAR(500) NOT NULL,
    uploader_id UUID NOT NULL,
    status INTEGER DEFAULT 0, -- 0:上传中, 1:已完成, 2:处理失败
    metadata JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 文档版本表
CREATE TABLE document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1,
    chunk_config JSONB NOT NULL,
    chunk_count INTEGER DEFAULT 0,
    status INTEGER DEFAULT 0, -- 0:处理中, 1:已完成, 2:失败
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 文档块表
CREATE TABLE chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_version_id UUID NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    embedding_status INTEGER DEFAULT 0, -- 0:未处理, 1:已完成, 2:失败
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 3. 向量相关表

```sql
-- 向量索引快照表
CREATE TABLE vector_indexes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'adbpg',
    config JSONB NOT NULL,
    status INTEGER DEFAULT 0, -- 0:创建中, 1:可用, 2:错误
    stats JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 向量记录表（用于跟踪和迁移）
CREATE TABLE vector_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    vector_index_id UUID NOT NULL REFERENCES vector_indexes(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL, -- 在向量数据库中的ID
    embedding_model VARCHAR(100) NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 4. LLM和Agent相关表

```sql
-- LLM提供商配置表
CREATE TABLE llm_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL, -- 加密存储的配置
    is_active BOOLEAN DEFAULT true,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- LLM模型表
CREATE TABLE llm_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
    model_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    model_type VARCHAR(50) NOT NULL, -- chat, completion, embedding
    config JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Agent表
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    system_prompt TEXT,
    llm_model_id UUID NOT NULL REFERENCES llm_models(id),
    tools JSONB DEFAULT '[]',
    config JSONB DEFAULT '{}',
    created_by UUID NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 5. 对话和问题相关表

```sql
-- 对话会话表
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    agent_id UUID REFERENCES agents(id),
    title VARCHAR(255),
    context JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 对话消息表
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL, -- user, assistant, system
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 问题表
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    chunk_id UUID REFERENCES chunks(id),
    content TEXT NOT NULL,
    question_type VARCHAR(50), -- factual, reasoning, application
    tags JSONB DEFAULT '[]',
    difficulty INTEGER DEFAULT 1, -- 1-5难度等级
    status INTEGER DEFAULT 0, -- 0:待审核, 1:已审核, 2:已拒绝
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 答案表
CREATE TABLE answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    reasoning TEXT, -- CoT推理过程
    llm_model_id UUID REFERENCES llm_models(id),
    quality_score DECIMAL(3,2), -- 质量评分 0-1
    is_reviewed BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 6. 任务相关表

```sql
-- 任务表
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    task_type VARCHAR(50) NOT NULL,
    status INTEGER DEFAULT 0, -- 0:处理中, 1:已完成, 2:失败, 3:已中断
    progress INTEGER DEFAULT 0, -- 进度百分比
    input_data JSONB NOT NULL,
    output_data JSONB DEFAULT '{}',
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 训练任务表
CREATE TABLE training_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    dataset_path VARCHAR(500) NOT NULL, -- OSS路径
    config JSONB NOT NULL,
    external_job_id VARCHAR(255), -- 外部训练平台的任务ID
    status INTEGER DEFAULT 0, -- 0:提交中, 1:训练中, 2:已完成, 3:失败
    progress INTEGER DEFAULT 0,
    result JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## 错误处理

### 错误分类和处理策略

#### 1. 业务错误

```go
type BusinessError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

// 常见业务错误码
const (
    ErrProjectNotFound     = "PROJECT_NOT_FOUND"
    ErrInsufficientPermission = "INSUFFICIENT_PERMISSION"
    ErrFileFormatNotSupported = "FILE_FORMAT_NOT_SUPPORTED"
    ErrVectorIndexNotReady = "VECTOR_INDEX_NOT_READY"
    ErrLLMProviderUnavailable = "LLM_PROVIDER_UNAVAILABLE"
)
```

#### 2. 系统错误处理

- **重试机制**：指数退避重试，最大重试3次
- **熔断器**：防止级联故障
- **降级策略**：关键服务不可用时的备选方案
- **监控告警**：错误率超过阈值时自动告警

## 测试策略

### 测试层次

#### 1. 单元测试

- **覆盖率要求**：核心业务逻辑 > 80%
- **测试框架**：Go testing + testify
- **Mock工具**：gomock用于外部依赖

#### 2. 集成测试

- **数据库测试**：使用testcontainers启动PostgreSQL容器
- **API测试**：使用httptest测试HTTP接口
- **外部服务测试**：Mock LLM API和云服务

#### 3. 端到端测试

- **关键流程测试**：文档上传→分块→向量化→检索完整流程
- **用户场景测试**：模拟真实用户操作场景

### 测试数据管理

- **测试数据隔离**：每个测试用例使用独立的数据库
- **数据清理**：测试完成后自动清理测试数据
- **种子数据**：提供标准的测试数据集

## 安全与权限

### 认证与授权

#### 1. JWT认证机制

```go
type JWTClaims struct {
    UserID    string   `json:"user_id"`
    ProjectID string   `json:"project_id,omitempty"`
    Roles     []string `json:"roles"`
    jwt.StandardClaims
}

// JWT中间件
func JWTMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        claims, err := validateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Set("user_id", claims.UserID)
        c.Set("roles", claims.Roles)
        c.Next()
    }
}
```

#### 2. RBAC权限模型

```go
type Permission string

const (
    PermProjectRead    Permission = "project:read"
    PermProjectWrite   Permission = "project:write"
    PermProjectDelete  Permission = "project:delete"
    PermDocumentRead   Permission = "document:read"
    PermDocumentWrite  Permission = "document:write"
    PermAgentManage    Permission = "agent:manage"
    PermLLMManage      Permission = "llm:manage"
)

type Role struct {
    Name        string       `json:"name"`
    Permissions []Permission `json:"permissions"`
}

// 预定义角色
var (
    RoleOwner = Role{
        Name: "owner",
        Permissions: []Permission{
            PermProjectRead, PermProjectWrite, PermProjectDelete,
            PermDocumentRead, PermDocumentWrite,
            PermAgentManage, PermLLMManage,
        },
    }
    
    RoleMember = Role{
        Name: "member",
        Permissions: []Permission{
            PermProjectRead, PermDocumentRead, PermDocumentWrite,
        },
    }
    
    RoleViewer = Role{
        Name: "viewer",
        Permissions: []Permission{
            PermProjectRead, PermDocumentRead,
        },
    }
)
```

#### 3. API密钥管理

```go
// 使用阿里云KMS加密存储敏感信息
type KMSManager struct {
    client *kms.Client
    keyID  string
}

func (k *KMSManager) EncryptAPIKey(plaintext string) (string, error) {
    req := &kms.EncryptRequest{
        KeyId:     k.keyID,
        Plaintext: plaintext,
    }
    resp, err := k.client.Encrypt(req)
    return resp.CiphertextBlob, err
}

func (k *KMSManager) DecryptAPIKey(ciphertext string) (string, error) {
    req := &kms.DecryptRequest{
        CiphertextBlob: ciphertext,
    }
    resp, err := k.client.Decrypt(req)
    return resp.Plaintext, err
}
```

### 审计日志

#### 审计事件定义

```go
type AuditEvent struct {
    ID        string                 `json:"id"`
    UserID    string                 `json:"user_id"`
    ProjectID string                 `json:"project_id,omitempty"`
    Action    string                 `json:"action"`
    Resource  string                 `json:"resource"`
    Details   map[string]interface{} `json:"details"`
    IP        string                 `json:"ip"`
    UserAgent string                 `json:"user_agent"`
    Timestamp time.Time              `json:"timestamp"`
}

// 审计中间件
func AuditMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        
        // 记录审计日志
        event := &AuditEvent{
            ID:        generateID(),
            UserID:    c.GetString("user_id"),
            Action:    c.Request.Method,
            Resource:  c.Request.URL.Path,
            IP:        c.ClientIP(),
            UserAgent: c.Request.UserAgent(),
            Timestamp: start,
        }
        
        auditLogger.Log(event)
    }
}
```

## 可观测性

### 监控指标

#### 1. 业务指标

```go
// Prometheus指标定义
var (
    // 文档处理指标
    documentsProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "documents_processed_total",
            Help: "Total number of documents processed",
        },
        []string{"project_id", "status"},
    )
    
    // 向量检索指标
    vectorSearchDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "vector_search_duration_seconds",
            Help: "Duration of vector search operations",
        },
        []string{"project_id", "index_name"},
    )
    
    // LLM调用指标
    llmRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_requests_total",
            Help: "Total number of LLM requests",
        },
        []string{"provider", "model", "status"},
    )
    
    // 任务队列指标
    taskQueueSize = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "task_queue_size",
            Help: "Current size of task queue",
        },
        []string{"task_type"},
    )
)
```

#### 2. 系统指标

- **HTTP请求**：请求数、响应时间、错误率
- **数据库**：连接池使用率、查询耗时、慢查询
- **缓存**：命中率、内存使用率
- **队列**：队列长度、处理速度、失败率

### 日志管理

#### 结构化日志格式

```go
type LogEntry struct {
    Level     string                 `json:"level"`
    Timestamp time.Time              `json:"timestamp"`
    Message   string                 `json:"message"`
    Service   string                 `json:"service"`
    TraceID   string                 `json:"trace_id,omitempty"`
    SpanID    string                 `json:"span_id,omitempty"`
    UserID    string                 `json:"user_id,omitempty"`
    ProjectID string                 `json:"project_id,omitempty"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
}

// 日志记录器配置
func NewLogger(service string) *logrus.Logger {
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: time.RFC3339,
    })
    logger.SetLevel(logrus.InfoLevel)
    
    // 添加服务名称字段
    logger = logger.WithField("service", service)
    return logger
}
```

### 链路追踪

#### OpenTelemetry集成

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

// 追踪中间件
func TracingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tracer := otel.Tracer("ai-knowledge-platform")
        ctx, span := tracer.Start(c.Request.Context(), c.Request.URL.Path)
        defer span.End()
        
        // 设置span属性
        span.SetAttributes(
            attribute.String("http.method", c.Request.Method),
            attribute.String("http.url", c.Request.URL.String()),
            attribute.String("user.id", c.GetString("user_id")),
        )
        
        c.Request = c.Request.WithContext(ctx)
        c.Next()
        
        // 设置响应状态
        span.SetAttributes(
            attribute.Int("http.status_code", c.Writer.Status()),
        )
    }
}
```

### 告警配置

#### Prometheus告警规则

```yaml
groups:
  - name: ai-knowledge-platform
    rules:
      # 错误率告警
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} for the last 5 minutes"
      
      # 响应时间告警
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High response time detected"
          description: "95th percentile response time is {{ $value }}s"
      
      # 队列积压告警
      - alert: TaskQueueBacklog
        expr: task_queue_size > 1000
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Task queue backlog detected"
          description: "Task queue size is {{ $value }}"
      
      # LLM服务不可用告警
      - alert: LLMServiceDown
        expr: up{job="llm-service"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "LLM service is down"
          description: "LLM service has been down for more than 1 minute"
```

## 部署架构

### 容器化部署

#### Dockerfile示例

```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./main"]
```

#### Docker Compose配置

```yaml
version: '3.8'

services:
  # API服务
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/aiplatform
      - REDIS_URL=redis://redis:6379
      - OSS_ENDPOINT=http://minio:9000
    depends_on:
      - postgres
      - redis
      - minio
    volumes:
      - ./configs:/app/configs

  # PostgreSQL数据库
  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=aiplatform
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  # Redis缓存
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  # MinIO对象存储
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data

  # 任务队列Worker
  worker:
    build: .
    command: ["./main", "worker"]
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/aiplatform
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### Kubernetes部署

#### Helm Chart结构

```
charts/ai-knowledge-platform/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   └── hpa.yaml
└── charts/
    ├── postgresql/
    └── redis/
```

#### 部署配置示例

```yaml
# values.yaml
replicaCount: 3

image:
  repository: ai-knowledge-platform
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80
  targetPort: 8080

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: ai-platform.example.com
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

postgresql:
  enabled: true
  auth:
    database: aiplatform
    username: user
    password: password

redis:
  enabled: true
  auth:
    enabled: false
```

## 性能优化

### 数据库优化

#### 1. 逻辑删除策略

所有核心业务表都采用逻辑删除机制，包含以下字段：

- `is_deleted BOOLEAN DEFAULT FALSE`：删除标记
- `deleted_at TIMESTAMP WITH TIME ZONE`：删除时间

**逻辑删除的优势：**

- 数据安全：避免误删除造成的数据丢失
- 审计追踪：保留完整的数据变更历史
- 关联完整性：保持外键关系的完整性
- 恢复能力：支持数据恢复功能

**查询规范：**

```go
// 标准查询模式 - 始终过滤已删除记录
const BaseQuery = "WHERE is_deleted = FALSE"

// 查询包含已删除记录（仅用于管理功能）
const AdminQuery = "WHERE 1=1" // 不过滤删除状态

// 只查询已删除记录（用于回收站功能）
const DeletedQuery = "WHERE is_deleted = TRUE"
```

**定期清理策略：**

```go
// 定期清理超过30天的已删除记录
func (r *Repository) CleanupDeletedRecords(ctx context.Context, days int) error {
    query := `
        DELETE FROM %s 
        WHERE is_deleted = TRUE 
        AND deleted_at < NOW() - INTERVAL '%d days'
    `
    // 执行清理...
}
```

#### 2. 索引策略

```sql
-- 项目相关索引（包含逻辑删除）
CREATE INDEX idx_projects_not_deleted ON projects(id) WHERE is_deleted = FALSE;
CREATE INDEX idx_files_project_id_not_deleted ON files(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_files_status ON files(status) WHERE is_deleted = FALSE;
CREATE INDEX idx_chunks_document_version_id ON chunks(document_version_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chunks_embedding_status ON chunks(embedding_status) WHERE is_deleted = FALSE;

-- 向量检索优化索引
CREATE INDEX idx_vector_records_chunk_id ON vector_records(chunk_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_vector_records_index_id ON vector_records(vector_index_id) WHERE is_deleted = FALSE;

-- 对话相关索引
CREATE INDEX idx_chat_sessions_project_user ON chat_sessions(project_id, user_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id) WHERE is_deleted = FALSE;

-- 问题答案相关索引
CREATE INDEX idx_questions_project_id ON questions(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_answers_question_id ON answers(question_id) WHERE is_deleted = FALSE;

-- Agent相关索引
CREATE INDEX idx_agents_project_id ON agents(project_id) WHERE is_deleted = FALSE;

-- 任务相关索引
CREATE INDEX idx_tasks_project_type_status ON tasks(project_id, task_type, status) WHERE is_deleted = FALSE;
CREATE INDEX idx_tasks_created_at ON tasks(created_at) WHERE is_deleted = FALSE;
```

#### 2. 查询优化

```go
// 分页查询优化（包含逻辑删除）
func (r *DocumentRepository) ListDocuments(ctx context.Context, projectID string, limit, offset int) ([]*Document, error) {
    query := `
        SELECT id, name, size, status, created_at
        FROM files 
        WHERE project_id = $1 AND is_deleted = FALSE
        ORDER BY created_at DESC 
        LIMIT $2 OFFSET $3
    `
    // 使用预编译语句
    stmt, err := r.db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()
    
    rows, err := stmt.QueryContext(ctx, projectID, limit, offset)
    // 处理结果...
}

// 逻辑删除方法
func (r *DocumentRepository) SoftDeleteDocument(ctx context.Context, documentID string) error {
    query := `
        UPDATE files 
        SET is_deleted = TRUE, deleted_at = NOW(), updated_at = NOW()
        WHERE id = $1 AND is_deleted = FALSE
    `
    result, err := r.db.ExecContext(ctx, query, documentID)
    if err != nil {
        return err
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    
    if rowsAffected == 0 {
        return errors.New("document not found or already deleted")
    }
    
    return nil
}

// 恢复删除的文档
func (r *DocumentRepository) RestoreDocument(ctx context.Context, documentID string) error {
    query := `
        UPDATE files 
        SET is_deleted = FALSE, deleted_at = NULL, updated_at = NOW()
        WHERE id = $1 AND is_deleted = TRUE
    `
    result, err := r.db.ExecContext(ctx, query, documentID)
    if err != nil {
        return err
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    
    if rowsAffected == 0 {
        return errors.New("document not found or not deleted")
    }
    
    return nil
}
```

### 缓存策略

#### 1. Redis缓存层

```go
type CacheManager struct {
    client *redis.Client
}

// 文档元数据缓存
func (c *CacheManager) CacheDocumentMetadata(ctx context.Context, docID string, metadata *DocumentMetadata) error {
    key := fmt.Sprintf("doc:meta:%s", docID)
    data, _ := json.Marshal(metadata)
    return c.client.Set(ctx, key, data, 30*time.Minute).Err()
}

// 向量检索结果缓存
func (c *CacheManager) CacheSearchResults(ctx context.Context, queryHash string, results []SearchResult) error {
    key := fmt.Sprintf("search:%s", queryHash)
    data, _ := json.Marshal(results)
    return c.client.Set(ctx, key, data, 10*time.Minute).Err()
}

// LLM响应缓存
func (c *CacheManager) CacheLLMResponse(ctx context.Context, requestHash string, response string) error {
    key := fmt.Sprintf("llm:%s", requestHash)
    return c.client.Set(ctx, key, response, 1*time.Hour).Err()
}
```

#### 2. 应用层缓存

```go
// 内存缓存用于热点数据
type MemoryCache struct {
    cache *sync.Map
    ttl   time.Duration
}

type CacheItem struct {
    Value     interface{}
    ExpiresAt time.Time
}

func (m *MemoryCache) Set(key string, value interface{}) {
    m.cache.Store(key, &CacheItem{
        Value:     value,
        ExpiresAt: time.Now().Add(m.ttl),
    })
}

func (m *MemoryCache) Get(key string) (interface{}, bool) {
    item, ok := m.cache.Load(key)
    if !ok {
        return nil, false
    }
    
    cacheItem := item.(*CacheItem)
    if time.Now().After(cacheItem.ExpiresAt) {
        m.cache.Delete(key)
        return nil, false
    }
    
    return cacheItem.Value, true
}
```

### 异步处理优化

#### 1. 任务队列配置

```go
// asynq配置优化
func NewTaskQueue() *asynq.Server {
    cfg := asynq.Config{
        Concurrency: 20,  // 并发worker数量
        Queues: map[string]int{
            "critical": 6,  // 高优先级队列
            "default":  3,  // 默认队列
            "low":      1,  // 低优先级队列
        },
        RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
            // 指数退避重试
            return time.Duration(n*n) * time.Second
        },
    }
    
    return asynq.NewServer(asynq.RedisClientOpt{Addr: "localhost:6379"}, cfg)
}
```

#### 2. 批处理优化

```go
// 批量向量插入
func (v *VectorService) BatchInsertVectors(ctx context.Context, vectors []Vector) error {
    const batchSize = 100
    
    for i := 0; i < len(vectors); i += batchSize {
        end := i + batchSize
        if end > len(vectors) {
            end = len(vectors)
        }
        
        batch := vectors[i:end]
        if err := v.provider.InsertVectors(ctx, batch); err != nil {
            return fmt.Errorf("batch insert failed at index %d: %w", i, err)
        }
    }
    
    return nil
}
```

这个设计文档涵盖了系统的核心架构、数据模型、安全机制、可观测性、部署方案和性能优化策略。设计遵循了云原生最佳实践，支持水平扩展和高可用部署。
