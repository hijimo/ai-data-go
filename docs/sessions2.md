# AI 对话系统会话管理模块 - Genkit Flow 实现方案

## 目录

- [1. 概述](#1-概述)
- [2. Genkit Flow 基础](#2-genkit-flow-基础)
- [3. 上下文构建 Flow](#3-上下文构建-flow)
- [4. 对话生成 Flow](#4-对话生成-flow)
- [5. 摘要生成 Flow](#5-摘要生成-flow)
- [6. 会话记忆策略 Flow](#6-会话记忆策略-flow)
- [7. 流式对话 Flow](#7-流式对话-flow)
- [8. Flow 组合与编排](#8-flow-组合与编排)
- [9. Token 管理策略](#9-token-管理策略)
- [10. 多租户隔离](#10-多租户隔离)
- [11. 监控告警](#11-监控告警)
- [12. 实施路线图](#12-实施路线图)

## 1. 概述

### 1.1 项目背景

本方案基于现有的多租户 AI 对话系统，使用 Google Genkit 构建智能会话管理模块。系统采用三层记忆架构（短期、长期、摘要），结合向量检索技术，实现智能的对话上下文管理和 Token 优化。

### 1.2 设计目标

**核心目标**：

- ✅ 构建基于 Genkit Flow 的会话管理核心流程
- ✅ 实现三层记忆架构（短期、长期、摘要）
- ✅ 智能上下文管理和自适应优化
- ✅ 查询分类和路由机制
- ✅ Token 预算管理和优化
- ✅ 会话健康检查和自动修复
- ✅ 完整的多租户隔离
- ✅ 监控告警和可观测性

**技术特性**：

- 类型安全的 Flow 定义
- 流式响应支持
- 可追踪的执行链路
- 易于测试和调试
- 支持 Flow 组合和编排

### 1.3 架构概览

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
│  ┌──────────────┐  ┌──────────────┐                        │
│  │Stream Chat   │  │Health Check  │                        │
│  │    Flow      │  │    Flow      │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Session Svc   │  │ Memory Svc   │  │ Vector Svc   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Context Svc   │  │Summary Svc   │  │ Token Mgr    │      │
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
│  │ PostgreSQL   │  │   pgvector   │  │    Redis     │      │
│  │  + UUID PK   │  │  (Embedding) │  │   (Cache)    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 1.4 关键特性

**三层记忆架构**：

- 短期记忆：最近 N 条消息，快速访问
- 长期记忆：向量化存储，语义检索
- 摘要记忆：定期压缩，减少 Token 消耗

**智能上下文管理**：

- 自动上下文构建和优化
- 基于查询类型的动态路由
- Token 预算管理和自适应调整
- 上下文压缩和摘要生成

**多租户隔离**：

- 严格的租户数据隔离
- 基于角色的权限控制
- 审计日志记录

**可观测性**：

- Flow 执行追踪
- 性能指标监控
- 告警和异常处理

## 2. Genkit Flow 基础

### 2.1 Genkit 初始化配置

**需求描述**：

- 初始化 Genkit 实例，配置 AI 模型插件
- 支持多种 AI 提供商（Google AI、OpenAI 等）
- 配置日志和追踪
- 设置全局错误处理

**输入参数**：

```go
type GenkitConfig struct {
    // AI 提供商配置
    Provider string `json:"provider"` // "google", "openai"
    
    // API 密钥
    APIKey string `json:"apiKey"`
    
    // 默认模型
    DefaultModel string `json:"defaultModel"`
    
    // 日志级别
    LogLevel string `json:"logLevel"` // "debug", "info", "warn", "error"
    
    // 是否启用追踪
    EnableTracing bool `json:"enableTracing"`
    
    // 超时配置（秒）
    Timeout int `json:"timeout"`
}
```

**实现要点**：

1. 根据配置选择合适的 AI 插件
2. 设置全局超时和重试策略
3. 配置日志输出格式
4. 初始化追踪系统
5. 注册错误处理中间件

**验收标准**：

- 当系统启动时，Genkit 应该成功初始化
- 当配置无效时，系统应该返回明确的错误信息
- 当 API 密钥错误时，系统应该在首次调用时报错
- 日志应该按照配置的级别输出

### 2.2 Flow 定义规范

**需求描述**：

- 定义统一的 Flow 命名规范
- 规范输入输出类型定义
- 统一错误处理机制
- 添加 Flow 元数据和文档

**Flow 命名规范**：

```
{domain}{Action}Flow

示例：
- contextBuildFlow: 上下文构建
- chatGenerateFlow: 对话生成
- summaryGenerateFlow: 摘要生成
- memorySearchFlow: 记忆检索
- queryClassifyFlow: 查询分类
```

**类型定义规范**：

```go
// 输入类型命名：{FlowName}Input
type ContextBuildInput struct {
    SessionID string `json:"sessionId" validate:"required,uuid"`
    UserQuery string `json:"userQuery" validate:"required,max=2000"`
    MaxTokens int    `json:"maxTokens" validate:"min=100,max=32000"`
}

// 输出类型命名：{FlowName}Output
type ContextBuildOutput struct {
    SessionID   string          `json:"sessionId"`
    Context     []MessageItem   `json:"context"`
    TotalTokens int             `json:"totalTokens"`
    Strategy    string          `json:"strategy"`
}
```

**错误处理规范**：

```go
// 自定义错误类型
type FlowError struct {
    Code    string `json:"code"`    // 错误代码
    Message string `json:"message"` // 错误消息
    Details any    `json:"details"` // 详细信息
}

// 错误代码定义
const (
    ErrCodeValidation    = "VALIDATION_ERROR"
    ErrCodePermission    = "PERMISSION_DENIED"
    ErrCodeNotFound      = "NOT_FOUND"
    ErrCodeTokenExceeded = "TOKEN_EXCEEDED"
    ErrCodeAIService     = "AI_SERVICE_ERROR"
)
```

**验收标准**：

- 所有 Flow 命名应该遵循规范
- 输入输出类型应该包含完整的 JSON 标签和验证规则
- 错误应该返回统一的格式
- Flow 应该包含描述性的注释

### 2.3 Flow 测试框架

**需求描述**：

- 提供 Flow 单元测试工具
- 支持 Mock 依赖服务
- 提供测试数据生成器
- 支持性能基准测试

**测试工具需求**：

1. Flow 输入验证测试
2. Flow 输出格式测试
3. 错误场景测试
4. 性能基准测试
5. 集成测试支持

**验收标准**：

- 每个 Flow 应该有对应的单元测试
- 测试覆盖率应该达到 80% 以上
- 应该包含正常场景和异常场景的测试
- 性能测试应该验证响应时间在可接受范围内

## 3. 上下文构建 Flow

### 3.1 上下文构建核心 Flow

**需求描述**：
构建智能对话上下文，整合短期记忆、长期记忆和摘要记忆，为 AI 生成提供完整的上下文信息。

**Flow 名称**：`contextBuildFlow`

**输入定义**：

```go
type ContextBuildInput struct {
    // 会话ID（必填）
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 用户查询（必填，用于检索相关记忆）
    UserQuery string `json:"userQuery" validate:"required,max=2000"`
    
    // 最大 Token 限制
    MaxTokens int `json:"maxTokens" validate:"min=100,max=32000"`
    
    // 上下文策略：auto（自动）、short（仅短期）、full（完整）
    Strategy string `json:"strategy" validate:"oneof=auto short full"`
    
    // 是否包含摘要
    IncludeSummary bool `json:"includeSummary"`
    
    // 是否包含长期记忆
    IncludeLongTerm bool `json:"includeLongTerm"`
    
    // 短期记忆窗口大小
    ShortTermWindow int `json:"shortTermWindow" validate:"min=1,max=50"`
}
```

**输出定义**：

```go
type ContextBuildOutput struct {
    // 会话ID
    SessionID string `json:"sessionId"`
    
    // 摘要上下文
    Summary *SummaryContext `json:"summary,omitempty"`
    
    // 相关长期记忆
    LongTermMemories []MemoryContext `json:"longTermMemories,omitempty"`
    
    // 短期消息列表
    ShortTermMessages []MessageContext `json:"shortTermMessages"`
    
    // 总 Token 数
    TotalTokens int `json:"totalTokens"`
    
    // 使用的策略
    Strategy string `json:"strategy"`
    
    // 上下文质量评分（0-1）
    QualityScore float64 `json:"qualityScore"`
    
    // 构建耗时（毫秒）
    BuildTime int64 `json:"buildTime"`
}

type SummaryContext struct {
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
    Coverage   string `json:"coverage"` // 摘要覆盖的消息范围
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
    Role       string `json:"role"` // "user", "assistant", "system"
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
}
```

**Flow 执行步骤**：

1. **权限验证**
   - 验证用户是否有权访问该会话
   - 检查会话是否属于当前租户
   - 记录访问日志

2. **参数验证和默认值设置**
   - 验证输入参数的有效性
   - 设置默认值（如 MaxTokens、Strategy）
   - 根据查询类型自动调整策略

3. **获取短期记忆**
   - 从数据库查询最近 N 条消息
   - 按时间倒序排列
   - 计算 Token 数量

4. **获取长期记忆（条件执行）**
   - 如果 IncludeLongTerm 为 true 且 UserQuery 不为空
   - 生成查询向量
   - 执行向量相似度搜索
   - 过滤低相似度结果（阈值 0.7）
   - 按相似度和重要性排序

5. **获取摘要记忆（条件执行）**
   - 如果 IncludeSummary 为 true
   - 查询最新的会话摘要
   - 验证摘要的时效性

6. **上下文组合**
   - 按优先级组合：摘要 → 长期记忆 → 短期消息
   - 计算总 Token 数

7. **Token 优化**
   - 如果总 Token 超过 MaxTokens
   - 执行智能裁剪：优先保留短期消息，适当减少长期记忆
   - 确保上下文连贯性

8. **质量评分**
   - 评估上下文的完整性
   - 评估相关性
   - 计算综合质量分数

**依赖服务**：

- SessionRepository：会话数据访问
- MessageRepository：消息数据访问
- MemoryRepository：记忆数据访问
- VectorService：向量检索服务
- TokenManager：Token 计算和优化

**错误处理**：

- 会话不存在：返回 NOT_FOUND 错误
- 权限不足：返回 PERMISSION_DENIED 错误
- 向量服务失败：降级处理，仅使用短期记忆
- Token 超限：返回 TOKEN_EXCEEDED 错误

**性能要求**：

- 平均响应时间 < 200ms
- P95 响应时间 < 500ms
- 支持并发请求 > 100 QPS

**验收标准**：

- 当提供有效的 SessionID 和 UserQuery 时，应该返回完整的上下文
- 当 Token 超限时，应该自动优化上下文
- 当向量服务失败时，应该降级到仅使用短期记忆
- 当用户无权访问会话时，应该返回 403 错误
- 上下文质量评分应该在 0-1 之间

### 3.2 查询分类 Flow

**需求描述**：
分析用户查询的类型和意图，为上下文构建提供决策依据。

**Flow 名称**：`queryClassifyFlow`

**输入定义**：

```go
type QueryClassifyInput struct {
    // 用户查询
    Query string `json:"query" validate:"required,max=2000"`
    
    // 会话ID（可选，用于获取历史上下文）
    SessionID string `json:"sessionId" validate:"omitempty,uuid"`
    
    // 最近的消息（可选，用于理解上下文）
    RecentMessages []string `json:"recentMessages" validate:"max=5"`
}
```

**输出定义**：

```go
type QueryClassifyOutput struct {
    // 查询类型
    QueryType string `json:"queryType"`
    
    // 查询意图
    Intent string `json:"intent"`
    
    // 是否需要历史上下文
    NeedsHistory bool `json:"needsHistory"`
    
    // 是否需要长期记忆
    NeedsLongTerm bool `json:"needsLongTerm"`
    
    // 推荐的上下文策略
    RecommendedStrategy string `json:"recommendedStrategy"`
    
    // 置信度（0-1）
    Confidence float64 `json:"confidence"`
    
    // 关键实体
    Entities []string `json:"entities"`
}
```

**查询类型定义**：

- `simple_question`：简单问题，不需要上下文
- `followup_question`：后续问题，需要短期上下文
- `complex_query`：复杂查询，需要完整上下文
- `reference_query`：引用查询，需要长期记忆
- `summarization`：摘要请求
- `clarification`：澄清请求

**Flow 执行步骤**：

1. **文本预处理**
   - 清理和标准化查询文本
   - 提取关键词和实体

2. **特征提取**
   - 查询长度
   - 是否包含指代词（"它"、"这个"、"那个"）
   - 是否包含时间引用（"之前"、"刚才"）
   - 是否包含比较词（"和"、"与"、"相比"）

3. **AI 分类**
   - 使用轻量级模型进行快速分类
   - 提取查询意图
   - 识别关键实体

4. **策略推荐**
   - 根据查询类型推荐上下文策略
   - 确定是否需要历史上下文
   - 确定是否需要长期记忆

**验收标准**：

- 当查询包含指代词时，应该识别为需要历史上下文
- 当查询是简单问题时，应该推荐 short 策略
- 当查询是复杂查询时，应该推荐 full 策略
- 分类置信度应该在 0.7 以上

### 3.3 上下文优化 Flow

**需求描述**：
对构建的上下文进行智能优化，在保证质量的前提下减少 Token 消耗。

**Flow 名称**：`contextOptimizeFlow`

**输入定义**：

```go
type ContextOptimizeInput struct {
    // 原始上下文
    Context ContextBuildOutput `json:"context" validate:"required"`
    
    // 目标 Token 数
    TargetTokens int `json:"targetTokens" validate:"required,min=100"`
    
    // 优化策略
    Strategy string `json:"strategy" validate:"oneof=aggressive balanced conservative"`
    
    // 是否保留摘要
    PreserveSummary bool `json:"preserveSummary"`
}
```

**输出定义**：

```go
type ContextOptimizeOutput struct {
    // 优化后的上下文
    OptimizedContext ContextBuildOutput `json:"optimizedContext"`
    
    // 原始 Token 数
    OriginalTokens int `json:"originalTokens"`
    
    // 优化后 Token 数
    OptimizedTokens int `json:"optimizedTokens"`
    
    // Token 节省率
    SavedRate float64 `json:"savedRate"`
    
    // 质量损失评分（0-1，越小越好）
    QualityLoss float64 `json:"qualityLoss"`
    
    // 优化操作列表
    Operations []string `json:"operations"`
}
```

**优化策略**：

1. **aggressive（激进）**
   - 大幅减少长期记忆数量
   - 压缩短期消息
   - 可能移除摘要

2. **balanced（平衡）**
   - 适度减少长期记忆
   - 保留关键短期消息
   - 保留摘要

3. **conservative（保守）**
   - 仅移除低相关性长期记忆
   - 完整保留短期消息
   - 必须保留摘要

**优化操作**：

1. 移除低相似度的长期记忆
2. 移除低重要性的长期记忆
3. 压缩长消息（保留关键信息）
4. 合并相似消息
5. 移除系统消息

**验收标准**：

- 优化后的 Token 数应该不超过目标值
- 质量损失应该在可接受范围内（< 0.3）
- 应该记录所有优化操作
- 摘要应该根据配置保留或移除

## 4. 对话生成 Flow

### 4.1 标准对话生成 Flow

**需求描述**：
基于构建的上下文生成 AI 响应，支持多种模型配置和生成策略。

**Flow 名称**：`chatGenerateFlow`

**输入定义**：

```go
type ChatGenerateInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 用户消息
    UserMessage string `json:"userMessage" validate:"required,max=4000"`
    
    // 上下文（可选，如果不提供则自动构建）
    Context *ContextBuildOutput `json:"context,omitempty"`
    
    // 模型配置
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
    
    // 系统提示词（可选）
    SystemPrompt string `json:"systemPrompt" validate:"max=1000"`
    
    // 是否保存消息
    SaveMessage bool `json:"saveMessage"`
}

type ModelConfig struct {
    // 模型名称
    ModelName string `json:"modelName" validate:"required"`
    
    // 温度参数（0-2）
    Temperature float64 `json:"temperature" validate:"min=0,max=2"`
    
    // Top P 参数（0-1）
    TopP float64 `json:"topP" validate:"min=0,max=1"`
    
    // 最大生成 Token 数
    MaxTokens int `json:"maxTokens" validate:"min=1,max=4096"`
    
    // 停止词
    StopSequences []string `json:"stopSequences" validate:"max=4"`
    
    // 频率惩罚（-2 到 2）
    FrequencyPenalty float64 `json:"frequencyPenalty" validate:"min=-2,max=2"`
    
    // 存在惩罚（-2 到 2）
    PresencePenalty float64 `json:"presencePenalty" validate:"min=-2,max=2"`
}
```

**输出定义**：

```go
type ChatGenerateOutput struct {
    // 消息ID
    MessageID string `json:"messageId"`
    
    // AI 响应
    Response string `json:"response"`
    
    // 使用的 Token 数
    TokenUsage TokenUsage `json:"tokenUsage"`
    
    // 完成原因
    FinishReason string `json:"finishReason"`
    
    // 使用的模型
    Model string `json:"model"`
    
    // 生成耗时（毫秒）
    GenerationTime int64 `json:"generationTime"`
    
    // 上下文信息
    ContextInfo ContextInfo `json:"contextInfo"`
}

type TokenUsage struct {
    PromptTokens     int `json:"promptTokens"`
    CompletionTokens int `json:"completionTokens"`
    TotalTokens      int `json:"totalTokens"`
}

type ContextInfo struct {
    ContextTokens int    `json:"contextTokens"`
    Strategy      string `json:"strategy"`
    QualityScore  float64 `json:"qualityScore"`
}
```

**Flow 执行步骤**：

1. **权限验证**
   - 验证用户对会话的访问权限
   - 检查租户配额限制

2. **上下文准备**
   - 如果未提供上下文，调用 contextBuildFlow 构建
   - 验证上下文的有效性
   - 检查 Token 预算

3. **提示词构建**
   - 组合系统提示词
   - 添加上下文信息（摘要、长期记忆、短期消息）
   - 添加用户消息
   - 格式化为模型所需格式

4. **模型配置**
   - 应用默认配置
   - 合并用户提供的配置
   - 验证配置参数的有效性

5. **AI 生成调用**
   - 调用 Genkit Generate API
   - 设置超时和重试策略
   - 处理流式响应（如果支持）

6. **响应处理**
   - 提取生成的文本
   - 记录 Token 使用情况
   - 检查完成原因

7. **消息保存**
   - 如果 SaveMessage 为 true
   - 保存用户消息和 AI 响应到数据库
   - 更新会话统计信息

8. **异步处理**
   - 生成消息向量并存储
   - 检查是否需要生成摘要
   - 更新会话健康状态

**依赖服务**：

- ContextBuildFlow：上下文构建
- MessageRepository：消息存储
- VectorService：向量生成
- TokenManager：Token 管理
- AIService：AI 模型调用

**错误处理**：

- 上下文构建失败：返回详细错误信息
- Token 超限：返回 TOKEN_EXCEEDED 错误
- AI 服务失败：重试 3 次，仍失败则返回错误
- 配额超限：返回 QUOTA_EXCEEDED 错误

**性能要求**：

- 平均响应时间 < 3s（不含 AI 生成时间）
- AI 生成时间取决于模型和响应长度
- 支持并发请求 > 50 QPS

**验收标准**：

- 当提供有效输入时，应该返回 AI 响应
- 当 Token 超限时，应该返回明确的错误
- 当 SaveMessage 为 true 时，消息应该保存到数据库
- Token 使用统计应该准确
- 应该记录完整的生成日志

### 4.2 多轮对话管理 Flow

**需求描述**：
管理多轮对话的状态和上下文，支持对话历史的智能管理。

**Flow 名称**：`multiTurnChatFlow`

**输入定义**：

```go
type MultiTurnChatInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 用户消息
    UserMessage string `json:"userMessage" validate:"required,max=4000"`
    
    // 对话轮次（可选，自动计算）
    TurnNumber int `json:"turnNumber" validate:"min=0"`
    
    // 是否重置上下文
    ResetContext bool `json:"resetContext"`
    
    // 模型配置
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
}
```

**输出定义**：

```go
type MultiTurnChatOutput struct {
    // 基础响应
    ChatResponse ChatGenerateOutput `json:"chatResponse"`
    
    // 当前轮次
    CurrentTurn int `json:"currentTurn"`
    
    // 总轮次
    TotalTurns int `json:"totalTurns"`
    
    // 会话状态
    SessionState string `json:"sessionState"`
    
    // 上下文健康度（0-1）
    ContextHealth float64 `json:"contextHealth"`
    
    // 是否需要压缩
    NeedsCompression bool `json:"needsCompression"`
    
    // 建议操作
    SuggestedActions []string `json:"suggestedActions"`
}
```

**会话状态定义**：

- `active`：活跃状态
- `needs_summary`：需要生成摘要
- `needs_cleanup`：需要清理
- `token_warning`：Token 接近上限
- `healthy`：健康状态

**Flow 执行步骤**：

1. **会话状态检查**
   - 获取会话的当前状态
   - 检查 Token 使用情况
   - 评估上下文健康度

2. **上下文重置（条件执行）**
   - 如果 ResetContext 为 true
   - 清理当前上下文
   - 保留摘要（如果存在）

3. **对话生成**
   - 调用 chatGenerateFlow 生成响应
   - 更新轮次计数

4. **健康度评估**
   - 评估上下文的连贯性
   - 检查 Token 使用率
   - 评估记忆质量

5. **建议生成**
   - 如果 Token 使用率 > 80%，建议压缩
   - 如果轮次 > 20，建议生成摘要
   - 如果上下文质量下降，建议重置

6. **状态更新**
   - 更新会话状态
   - 记录健康度指标

**验收标准**：

- 应该正确跟踪对话轮次
- 应该准确评估上下文健康度
- 当 Token 接近上限时，应该发出警告
- 应该提供合理的建议操作

### 4.3 对话重试和回退 Flow

**需求描述**：
处理 AI 生成失败的情况，提供智能重试和回退机制。

**Flow 名称**：`chatRetryFlow`

**输入定义**：

```go
type ChatRetryInput struct {
    // 原始请求
    OriginalRequest ChatGenerateInput `json:"originalRequest" validate:"required"`
    
    // 失败原因
    FailureReason string `json:"failureReason" validate:"required"`
    
    // 已重试次数
    RetryCount int `json:"retryCount" validate:"min=0,max=5"`
    
    // 重试策略
    RetryStrategy string `json:"retryStrategy" validate:"oneof=simple exponential adaptive"`
}
```

**输出定义**：

```go
type ChatRetryOutput struct {
    // 是否成功
    Success bool `json:"success"`
    
    // 响应（如果成功）
    Response *ChatGenerateOutput `json:"response,omitempty"`
    
    // 错误信息（如果失败）
    Error *FlowError `json:"error,omitempty"`
    
    // 重试次数
    TotalRetries int `json:"totalRetries"`
    
    // 使用的策略
    StrategyUsed string `json:"strategyUsed"`
    
    // 回退操作
    FallbackActions []string `json:"fallbackActions"`
}
```

**重试策略**：

1. **simple（简单重试）**
   - 固定间隔重试
   - 最多重试 3 次

2. **exponential（指数退避）**
   - 指数增长的重试间隔
   - 最多重试 5 次

3. **adaptive（自适应）**
   - 根据失败原因调整策略
   - 可能调整模型参数
   - 可能简化上下文

**回退操作**：

1. 减少上下文 Token 数
2. 降低模型复杂度
3. 使用备用模型
4. 简化用户查询
5. 返回预设响应

**验收标准**：

- 应该根据失败原因选择合适的重试策略
- 应该记录所有重试尝试
- 当所有重试失败时，应该执行回退操作
- 应该避免无限重试

## 5. 摘要生成 Flow

### 5.1 自动摘要生成 Flow

**需求描述**：
自动检测并生成对话摘要，压缩历史对话以减少 Token 消耗。

**Flow 名称**：`summaryGenerateFlow`

**输入定义**：

```go
type SummaryGenerateInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 需要摘要的消息ID列表（可选，不提供则自动选择）
    MessageIDs []string `json:"messageIds" validate:"dive,uuid"`
    
    // 起始消息ID（可选）
    StartMessageID string `json:"startMessageId" validate:"omitempty,uuid"`
    
    // 结束消息ID（可选）
    EndMessageID string `json:"endMessageId" validate:"omitempty,uuid"`
    
    // 之前的摘要（增量摘要）
    PreviousSummary string `json:"previousSummary" validate:"max=2000"`
    
    // 摘要类型
    SummaryType string `json:"summaryType" validate:"oneof=incremental full"`
    
    // 目标长度（字符数）
    TargetLength int `json:"targetLength" validate:"min=50,max=1000"`
}
```

**输出定义**：

```go
type SummaryGenerateOutput struct {
    // 摘要ID
    SummaryID string `json:"summaryId"`
    
    // 摘要内容
    Summary string `json:"summary"`
    
    // Token 数量
    TokenCount int `json:"tokenCount"`
    
    // 摘要的消息数量
    MessageCount int `json:"messageCount"`
    
    // 起始消息ID
    StartMessageID string `json:"startMessageId"`
    
    // 结束消息ID
    EndMessageID string `json:"endMessageId"`
    
    // 摘要质量评分（0-1）
    QualityScore float64 `json:"qualityScore"`
    
    // 压缩率
    CompressionRate float64 `json:"compressionRate"`
    
    // 关键主题
    KeyTopics []string `json:"keyTopics"`
    
    // 生成耗时（毫秒）
    GenerationTime int64 `json:"generationTime"`
}
```

**Flow 执行步骤**：

1. **权限验证**
   - 验证用户对会话的访问权限
   - 检查会话状态

2. **消息选择**
   - 如果提供了 MessageIDs，使用指定消息
   - 如果提供了起止ID，查询范围内的消息
   - 否则，自动选择需要摘要的消息（自上次摘要后的消息）

3. **消息验证**
   - 检查消息数量是否足够（至少 5 条）
   - 验证消息的连续性
   - 计算原始 Token 数

4. **提示词构建**
   - 如果是增量摘要，包含之前的摘要
   - 添加摘要要求和约束
   - 格式化消息内容

5. **AI 摘要生成**
   - 调用 AI 模型生成摘要
   - 使用较低的温度参数（0.3）保证稳定性
   - 限制输出长度

6. **摘要后处理**
   - 提取关键主题
   - 计算质量评分
   - 计算压缩率

7. **摘要保存**
   - 保存摘要到数据库
   - 更新会话的上下文配置
   - 标记已摘要的消息

8. **异步处理**
   - 生成摘要向量
   - 更新会话统计
   - 触发清理任务

**提示词模板**：

```
请对以下对话内容生成简洁的摘要，保留关键信息和上下文。

要求：
1. 摘要长度控制在 {TargetLength} 字以内
2. 保留对话的主题和关键信息
3. 突出重要的结论和决策
4. 使用简洁清晰的语言
5. 保持时间顺序

{如果有之前的摘要}
之前的对话摘要：
{PreviousSummary}

新的对话内容：
{如果没有之前的摘要}
对话内容：

{消息列表}

摘要：
```

**质量评分标准**：

- 信息完整性（30%）
- 关键点覆盖（30%）
- 语言简洁性（20%）
- 逻辑连贯性（20%）

**依赖服务**：

- MessageRepository：消息数据访问
- SummaryRepository：摘要存储
- ContextRepository：上下文配置更新
- AIService：AI 模型调用
- VectorService：向量生成

**错误处理**：

- 消息不足：返回 INSUFFICIENT_MESSAGES 错误
- AI 生成失败：重试 2 次
- 摘要质量过低：重新生成

**性能要求**：

- 平均生成时间 < 5s
- 支持批量摘要生成

**验收标准**：

- 当消息数量足够时，应该生成高质量摘要
- 摘要长度应该在目标范围内
- 压缩率应该 > 50%
- 质量评分应该 > 0.7
- 应该正确提取关键主题

### 5.2 摘要触发策略 Flow

**需求描述**：
智能判断何时需要生成摘要，避免过度或不足的摘要。

**Flow 名称**：`summaryTriggerFlow`

**输入定义**：

```go
type SummaryTriggerInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 检查模式
    CheckMode string `json:"checkMode" validate:"oneof=auto manual force"`
}
```

**输出定义**：

```go
type SummaryTriggerOutput struct {
    // 是否需要摘要
    ShouldSummarize bool `json:"shouldSummarize"`
    
    // 触发原因
    TriggerReason string `json:"triggerReason"`
    
    // 待摘要的消息ID列表
    MessageIDs []string `json:"messageIds"`
    
    // 待摘要的消息数量
    MessageCount int `json:"messageCount"`
    
    // 预估的 Token 节省
    EstimatedTokenSaving int `json:"estimatedTokenSaving"`
    
    // 紧急程度（0-1）
    Urgency float64 `json:"urgency"`
    
    // 建议的摘要类型
    RecommendedType string `json:"recommendedType"`
}
```

**触发条件**：

1. **消息数量触发**
   - 自上次摘要后新增消息 >= 20 条
   - 权重：高

2. **Token 使用触发**
   - 当前上下文 Token 数 > 最大限制的 80%
   - 权重：非常高

3. **时间触发**
   - 距离上次摘要超过 24 小时且有新消息
   - 权重：中

4. **质量触发**
   - 上下文质量评分 < 0.6
   - 权重：高

5. **手动触发**
   - 用户或系统明确请求
   - 权重：最高

**Flow 执行步骤**：

1. **获取会话状态**
   - 查询最后的摘要信息
   - 统计新消息数量
   - 计算当前 Token 使用

2. **评估触发条件**
   - 检查所有触发条件
   - 计算综合得分
   - 确定紧急程度

3. **消息选择**
   - 如果需要摘要，选择待摘要的消息
   - 确保消息的连续性

4. **收益评估**
   - 估算摘要后的 Token 节省
   - 评估摘要的必要性

5. **建议生成**
   - 推荐摘要类型（增量或完整）
   - 提供触发原因说明

**验收标准**：

- 当消息数量达到阈值时，应该触发摘要
- 当 Token 使用率过高时，应该立即触发
- 应该准确估算 Token 节省
- 紧急程度应该合理反映实际情况

### 5.3 摘要质量评估 Flow

**需求描述**：
评估生成的摘要质量，确保摘要的有效性和准确性。

**Flow 名称**：`summaryQualityFlow`

**输入定义**：

```go
type SummaryQualityInput struct {
    // 摘要内容
    Summary string `json:"summary" validate:"required,max=2000"`
    
    // 原始消息列表
    OriginalMessages []string `json:"originalMessages" validate:"required,min=1"`
    
    // 评估维度
    Dimensions []string `json:"dimensions" validate:"dive,oneof=completeness conciseness coherence accuracy"`
}
```

**输出定义**：

```go
type SummaryQualityOutput struct {
    // 总体质量评分（0-1）
    OverallScore float64 `json:"overallScore"`
    
    // 各维度评分
    DimensionScores map[string]float64 `json:"dimensionScores"`
    
    // 是否通过质量检查
    Passed bool `json:"passed"`
    
    // 问题列表
    Issues []QualityIssue `json:"issues"`
    
    // 改进建议
    Suggestions []string `json:"suggestions"`
    
    // 关键信息覆盖率
    KeyInfoCoverage float64 `json:"keyInfoCoverage"`
}

type QualityIssue struct {
    Dimension   string  `json:"dimension"`
    Severity    string  `json:"severity"` // "low", "medium", "high"
    Description string  `json:"description"`
    Score       float64 `json:"score"`
}
```

**评估维度**：

1. **completeness（完整性）**
   - 关键信息是否完整
   - 重要结论是否包含
   - 上下文是否清晰

2. **conciseness（简洁性）**
   - 是否有冗余信息
   - 表达是否简洁
   - 长度是否合适

3. **coherence（连贯性）**
   - 逻辑是否清晰
   - 时间顺序是否正确
   - 主题是否统一

4. **accuracy（准确性）**
   - 是否有事实错误
   - 是否有歧义
   - 是否忠实原文

**质量阈值**：

- 通过标准：总体评分 >= 0.7
- 优秀标准：总体评分 >= 0.85

**验收标准**：

- 应该准确评估摘要质量
- 应该识别具体的质量问题
- 应该提供可操作的改进建议
- 评分应该客观合理

## 6. 会话记忆策略 Flow

### 6.1 长期记忆检索 Flow

**需求描述**：
基于向量相似度检索相关的历史对话记忆，为当前对话提供上下文支持。

**Flow 名称**：`memorySearchFlow`

**输入定义**：

```go
type MemorySearchInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 查询文本
    Query string `json:"query" validate:"required,max=2000"`
    
    // 返回结果数量
    TopK int `json:"topK" validate:"min=1,max=20"`
    
    // 最小相似度阈值（0-1）
    MinSimilarity float32 `json:"minSimilarity" validate:"min=0,max=1"`
    
    // 时间范围过滤（天数，0表示不限制）
    TimeRangeDays int `json:"timeRangeDays" validate:"min=0,max=365"`
    
    // 记忆类型过滤
    MemoryTypes []string `json:"memoryTypes" validate:"dive,oneof=short_term long_term summary"`
    
    // 是否包含其他会话的记忆（同一用户）
    IncludeCrossSessions bool `json:"includeCrossSessions"`
}
```

**输出定义**：

```go
type MemorySearchOutput struct {
    // 检索到的记忆列表
    Memories []MemoryResult `json:"memories"`
    
    // 总找到数量
    TotalFound int `json:"totalFound"`
    
    // 返回数量
    ReturnedCount int `json:"returnedCount"`
    
    // 搜索耗时（毫秒）
    SearchTime int64 `json:"searchTime"`
    
    // 平均相似度
    AverageSimilarity float32 `json:"averageSimilarity"`
    
    // 搜索策略
    SearchStrategy string `json:"searchStrategy"`
}

type MemoryResult struct {
    // 记忆ID
    ID string `json:"id"`
    
    // 会话ID
    SessionID string `json:"sessionId"`
    
    // 记忆类型
    MemoryType string `json:"memoryType"`
    
    // 内容
    Content string `json:"content"`
    
    // Token 数量
    TokenCount int `json:"tokenCount"`
    
    // 相似度分数（0-1）
    Similarity float32 `json:"similarity"`
    
    // 重要性分数（0-1）
    Importance float32 `json:"importance"`
    
    // 综合分数（相似度 * 重要性）
    Score float32 `json:"score"`
    
    // 访问次数
    AccessCount int `json:"accessCount"`
    
    // 创建时间
    CreatedAt string `json:"createdAt"`
    
    // 最后访问时间
    LastAccessAt string `json:"lastAccessAt"`
    
    // 元数据
    Metadata map[string]interface{} `json:"metadata"`
}
```

**Flow 执行步骤**：

1. **权限验证**
   - 验证用户对会话的访问权限
   - 如果 IncludeCrossSessions 为 true，验证用户身份

2. **参数处理**
   - 设置默认值（TopK=5, MinSimilarity=0.7）
   - 验证参数的合理性

3. **查询向量生成**
   - 调用嵌入服务生成查询向量
   - 缓存常见查询的向量

4. **向量检索**
   - 使用 pgvector 进行相似度搜索
   - 应用时间范围过滤
   - 应用记忆类型过滤
   - 应用租户隔离

5. **结果排序**
   - 计算综合分数（相似度 * 重要性）
   - 按分数降序排列
   - 过滤低于阈值的结果

6. **结果增强**
   - 添加元数据信息
   - 计算统计指标

7. **访问统计更新**
   - 异步更新记忆的访问次数
   - 更新最后访问时间
   - 调整重要性分数

**搜索策略**：

1. **精确匹配优先**
   - 优先返回高相似度结果
   - 适用于明确的引用查询

2. **语义扩展**
   - 包含语义相关的记忆
   - 适用于探索性查询

3. **时间加权**
   - 近期记忆权重更高
   - 适用于时间敏感的查询

4. **重要性加权**
   - 高重要性记忆优先
   - 适用于关键信息检索

**依赖服务**：

- MemoryRepository：记忆数据访问
- VectorService：向量生成和检索
- EmbeddingService：文本嵌入
- CacheService：查询缓存

**性能优化**：

- 向量索引优化（IVFFlat）
- 查询结果缓存
- 批量向量生成
- 异步统计更新

**错误处理**：

- 向量生成失败：返回错误，不降级
- 检索超时：返回部分结果
- 数据库错误：重试 2 次

**性能要求**：

- 平均检索时间 < 100ms
- P95 检索时间 < 300ms
- 支持并发检索 > 200 QPS

**验收标准**：

- 应该返回相关度高的记忆
- 相似度计算应该准确
- 应该正确应用过滤条件
- 应该更新访问统计
- 跨会话检索应该正确隔离租户

### 6.2 记忆存储 Flow

**需求描述**：
将对话消息转换为长期记忆并存储，包括向量化和元数据提取。

**Flow 名称**：`memoryStoreFlow`

**输入定义**：

```go
type MemoryStoreInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 消息ID列表
    MessageIDs []string `json:"messageIds" validate:"required,min=1,dive,uuid"`
    
    // 记忆类型
    MemoryType string `json:"memoryType" validate:"required,oneof=short_term long_term summary"`
    
    // 内容（如果不提供则从消息中提取）
    Content string `json:"content" validate:"max=4000"`
    
    // 重要性分数（0-1）
    Importance float32 `json:"importance" validate:"min=0,max=1"`
    
    // 过期时间（天数，0表示不过期）
    ExpirationDays int `json:"expirationDays" validate:"min=0,max=365"`
    
    // 元数据
    Metadata map[string]interface{} `json:"metadata"`
}
```

**输出定义**：

```go
type MemoryStoreOutput struct {
    // 记忆ID
    MemoryID string `json:"memoryId"`
    
    // 存储的内容
    Content string `json:"content"`
    
    // Token 数量
    TokenCount int `json:"tokenCount"`
    
    // 向量维度
    EmbeddingDimension int `json:"embeddingDimension"`
    
    // 提取的关键词
    Keywords []string `json:"keywords"`
    
    // 提取的实体
    Entities []string `json:"entities"`
    
    // 存储耗时（毫秒）
    StoreTime int64 `json:"storeTime"`
    
    // 是否成功
    Success bool `json:"success"`
}
```

**Flow 执行步骤**：

1. **权限验证**
   - 验证用户对会话的访问权限
   - 检查存储配额

2. **内容准备**
   - 如果未提供 Content，从消息中提取
   - 清理和标准化文本
   - 计算 Token 数量

3. **重要性评估**
   - 如果未提供 Importance，自动评估
   - 基于内容长度、关键词、实体等因素

4. **向量生成**
   - 调用嵌入服务生成向量
   - 验证向量维度

5. **元数据提取**
   - 提取关键词
   - 识别命名实体
   - 提取时间信息

6. **记忆存储**
   - 保存到数据库
   - 设置过期时间
   - 建立与消息的关联

7. **索引更新**
   - 更新向量索引
   - 更新全文搜索索引

**重要性评估因素**：

- 内容长度（权重 10%）
- 关键词密度（权重 20%）
- 实体数量（权重 20%）
- 用户反馈（权重 30%）
- 访问频率（权重 20%）

**依赖服务**：

- MessageRepository：消息数据访问
- MemoryRepository：记忆存储
- EmbeddingService：向量生成
- NLPService：关键词和实体提取

**错误处理**：

- 向量生成失败：重试 2 次
- 存储失败：回滚操作
- 配额超限：返回 QUOTA_EXCEEDED 错误

**性能要求**：

- 平均存储时间 < 500ms
- 支持批量存储

**验收标准**：

- 应该成功生成并存储向量
- 应该正确提取元数据
- 重要性评分应该合理
- 应该正确设置过期时间

### 6.3 记忆清理 Flow

**需求描述**：
定期清理过期、低质量或不再需要的记忆，优化存储空间。

**Flow 名称**：`memoryCleanupFlow`

**输入定义**：

```go
type MemoryCleanupInput struct {
    // 会话ID（可选，不提供则清理所有会话）
    SessionID string `json:"sessionId" validate:"omitempty,uuid"`
    
    // 租户ID（可选，用于批量清理）
    TenantID string `json:"tenantId" validate:"omitempty,uuid"`
    
    // 清理策略
    Strategy string `json:"strategy" validate:"required,oneof=expired low_quality unused all"`
    
    // 清理模式
    Mode string `json:"mode" validate:"required,oneof=soft hard"`
    
    // 批量大小
    BatchSize int `json:"batchSize" validate:"min=10,max=1000"`
    
    // 是否执行（false 表示仅预览）
    Execute bool `json:"execute"`
}
```

**输出定义**：

```go
type MemoryCleanupOutput struct {
    // 清理的记忆数量
    CleanedCount int `json:"cleanedCount"`
    
    // 释放的存储空间（字节）
    FreedSpace int64 `json:"freedSpace"`
    
    // 清理详情
    Details []CleanupDetail `json:"details"`
    
    // 清理耗时（毫秒）
    CleanupTime int64 `json:"cleanupTime"`
    
    // 是否完成
    Completed bool `json:"completed"`
    
    // 下一批次的游标
    NextCursor string `json:"nextCursor,omitempty"`
}

type CleanupDetail struct {
    MemoryID   string `json:"memoryId"`
    Reason     string `json:"reason"`
    Size       int64  `json:"size"`
    CreatedAt  string `json:"createdAt"`
    LastAccess string `json:"lastAccess"`
}
```

**清理策略**：

1. **expired（过期清理）**
   - 清理已过期的记忆
   - 检查 ExpiresAt 字段

2. **low_quality（低质量清理）**
   - 清理重要性低且访问少的记忆
   - 阈值：Importance < 0.3 且 AccessCount < 2

3. **unused（未使用清理）**
   - 清理长期未访问的记忆
   - 阈值：LastAccessAt > 90 天前

4. **all（全面清理）**
   - 综合以上所有策略

**清理模式**：

1. **soft（软删除）**
   - 标记为已删除（IsDeleted=true）
   - 保留数据，可恢复
   - 不释放存储空间

2. **hard（硬删除）**
   - 物理删除数据
   - 不可恢复
   - 释放存储空间

**Flow 执行步骤**：

1. **权限验证**
   - 验证清理权限
   - 租户管理员只能清理自己租户的数据

2. **查询待清理记忆**
   - 根据策略查询符合条件的记忆
   - 应用租户隔离
   - 分批查询

3. **预览模式**
   - 如果 Execute=false，仅返回预览信息
   - 不执行实际删除

4. **执行清理**
   - 如果 Execute=true，执行删除操作
   - 根据 Mode 选择软删除或硬删除
   - 批量处理

5. **统计计算**
   - 统计清理数量
   - 计算释放空间

6. **日志记录**
   - 记录清理操作
   - 记录清理详情

**依赖服务**：

- MemoryRepository：记忆数据访问
- AuditService：审计日志

**错误处理**：

- 权限不足：返回 PERMISSION_DENIED 错误
- 清理失败：回滚已清理的记忆

**性能要求**：

- 批量清理性能 > 1000 条/秒
- 支持大规模清理

**验收标准**：

- 应该正确识别待清理的记忆
- 预览模式不应该删除数据
- 软删除应该可恢复
- 硬删除应该释放空间
- 应该记录完整的清理日志

## 7. 流式对话 Flow

### 7.1 流式响应生成 Flow

**需求描述**：
支持流式生成 AI 响应，实时返回生成的内容，提升用户体验。

**Flow 名称**：`chatStreamFlow`

**输入定义**：

```go
type ChatStreamInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 用户消息
    UserMessage string `json:"userMessage" validate:"required,max=4000"`
    
    // 上下文（可选）
    Context *ContextBuildOutput `json:"context,omitempty"`
    
    // 模型配置
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
    
    // 流式配置
    StreamConfig *StreamConfig `json:"streamConfig,omitempty"`
}

type StreamConfig struct {
    // 缓冲区大小
    BufferSize int `json:"bufferSize" validate:"min=1,max=100"`
    
    // 发送间隔（毫秒）
    SendInterval int `json:"sendInterval" validate:"min=10,max=1000"`
    
    // 是否包含 Token 统计
    IncludeTokenStats bool `json:"includeTokenStats"`
    
    // 是否包含中间状态
    IncludeIntermediateStates bool `json:"includeIntermediateStates"`
}
```

**输出定义（流式块）**：

```go
type ChatStreamChunk struct {
    // 块类型
    Type string `json:"type"` // "start", "content", "token_stats", "end", "error"
    
    // 增量内容
    Delta string `json:"delta,omitempty"`
    
    // 累积内容
    Accumulated string `json:"accumulated,omitempty"`
    
    // Token 统计
    TokenStats *TokenStats `json:"tokenStats,omitempty"`
    
    // 中间状态
    State *StreamState `json:"state,omitempty"`
    
    // 错误信息
    Error *FlowError `json:"error,omitempty"`
    
    // 时间戳
    Timestamp int64 `json:"timestamp"`
}

type TokenStats struct {
    PromptTokens     int `json:"promptTokens"`
    CompletionTokens int `json:"completionTokens"`
    TotalTokens      int `json:"totalTokens"`
}

type StreamState struct {
    Status         string  `json:"status"` // "building_context", "generating", "saving"
    Progress       float64 `json:"progress"` // 0-1
    EstimatedTime  int64   `json:"estimatedTime"` // 毫秒
}
```

**最终输出定义**：

```go
type ChatStreamOutput struct {
    // 消息ID
    MessageID string `json:"messageId"`
    
    // 完整响应
    FullResponse string `json:"fullResponse"`
    
    // Token 使用
    TokenUsage TokenUsage `json:"tokenUsage"`
    
    // 完成原因
    FinishReason string `json:"finishReason"`
    
    // 总耗时（毫秒）
    TotalTime int64 `json:"totalTime"`
    
    // 流式块数量
    ChunkCount int `json:"chunkCount"`
}
```

**Flow 执行步骤**：

1. **初始化流式响应**
   - 发送 start 类型的块
   - 初始化缓冲区
   - 设置超时控制

2. **上下文构建**
   - 如果未提供上下文，构建上下文
   - 发送中间状态（building_context）
   - 计算预估时间

3. **流式生成**
   - 调用 Genkit 流式 API
   - 接收增量内容
   - 缓冲和批量发送

4. **内容处理**
   - 累积生成的内容
   - 定期发送 Token 统计
   - 发送进度更新

5. **完成处理**
   - 发送 end 类型的块
   - 保存完整消息
   - 返回最终输出

6. **错误处理**
   - 捕获生成过程中的错误
   - 发送 error 类型的块
   - 清理资源

**流式发送策略**：

1. **固定间隔发送**
   - 每隔 N 毫秒发送一次
   - 适合稳定的网络环境

2. **缓冲区满发送**
   - 缓冲区达到阈值时发送
   - 适合高速生成场景

3. **自适应发送**
   - 根据网络状况动态调整
   - 平衡延迟和吞吐量

**依赖服务**：

- ContextBuildFlow：上下文构建
- MessageRepository：消息存储
- AIService：流式 AI 调用

**错误处理**：

- 连接中断：尝试重连
- 生成超时：返回部分结果
- 缓冲区溢出：增加发送频率

**性能要求**：

- 首字节延迟 < 500ms
- 流式延迟 < 100ms
- 支持并发流式会话 > 50

**验收标准**：

- 应该实时返回生成的内容
- 应该正确发送各类型的块
- 应该处理网络中断
- 应该正确统计 Token 使用
- 最终输出应该完整

### 7.2 流式上下文更新 Flow

**需求描述**：
在流式对话过程中实时更新上下文，支持动态调整。

**Flow 名称**：`streamContextUpdateFlow`

**输入定义**：

```go
type StreamContextUpdateInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 当前生成的内容
    CurrentContent string `json:"currentContent" validate:"required"`
    
    // 更新类型
    UpdateType string `json:"updateType" validate:"oneof=append replace optimize"`
    
    // 是否触发摘要
    TriggerSummary bool `json:"triggerSummary"`
}
```

**输出定义**：

```go
type StreamContextUpdateOutput struct {
    // 更新是否成功
    Success bool `json:"success"`
    
    // 更新的上下文
    UpdatedContext *ContextBuildOutput `json:"updatedContext,omitempty"`
    
    // 触发的操作
    TriggeredActions []string `json:"triggeredActions"`
    
    // 更新耗时（毫秒）
    UpdateTime int64 `json:"updateTime"`
}
```

**验收标准**：

- 应该实时更新上下文
- 应该正确触发摘要
- 更新不应该影响流式生成

### 7.3 流式错误恢复 Flow

**需求描述**：
处理流式对话中的错误，提供恢复机制。

**Flow 名称**：`streamErrorRecoveryFlow`

**输入定义**：

```go
type StreamErrorRecoveryInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 错误类型
    ErrorType string `json:"errorType" validate:"required"`
    
    // 已生成的内容
    GeneratedContent string `json:"generatedContent"`
    
    // 恢复策略
    RecoveryStrategy string `json:"recoveryStrategy" validate:"oneof=retry resume fallback"`
}
```

**输出定义**：

```go
type StreamErrorRecoveryOutput struct {
    // 是否恢复成功
    Recovered bool `json:"recovered"`
    
    // 恢复后的内容
    RecoveredContent string `json:"recoveredContent,omitempty"`
    
    // 使用的策略
    StrategyUsed string `json:"strategyUsed"`
    
    // 恢复耗时（毫秒）
    RecoveryTime int64 `json:"recoveryTime"`
}
```

**恢复策略**：

1. **retry（重试）**
   - 从头重新生成
   - 适用于临时错误

2. **resume（恢复）**
   - 从中断点继续生成
   - 适用于网络中断

3. **fallback（回退）**
   - 返回已生成的内容
   - 适用于不可恢复的错误

**验收标准**：

- 应该正确识别错误类型
- 应该选择合适的恢复策略
- 应该保留已生成的内容
- 恢复应该对用户透明

## 8. Flow 组合与编排

### 8.1 完整对话流程 Flow

**需求描述**：
组合多个 Flow，实现完整的对话流程，包括上下文构建、生成、保存和后处理。

**Flow 名称**：`completeConversationFlow`

**输入定义**：

```go
type CompleteConversationInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 用户消息
    UserMessage string `json:"userMessage" validate:"required,max=4000"`
    
    // 模型配置
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
    
    // 流程配置
    FlowConfig *FlowConfig `json:"flowConfig,omitempty"`
}

type FlowConfig struct {
    // 是否启用查询分类
    EnableQueryClassify bool `json:"enableQueryClassify"`
    
    // 是否自动优化上下文
    AutoOptimizeContext bool `json:"autoOptimizeContext"`
    
    // 是否自动生成摘要
    AutoGenerateSummary bool `json:"autoGenerateSummary"`
    
    // 是否保存记忆
    SaveMemory bool `json:"saveMemory"`
    
    // 是否启用流式响应
    EnableStreaming bool `json:"enableStreaming"`
}
```

**输出定义**：

```go
type CompleteConversationOutput struct {
    // 对话响应
    ChatResponse ChatGenerateOutput `json:"chatResponse"`
    
    // 上下文信息
    Context ContextBuildOutput `json:"context"`
    
    // 查询分类结果
    QueryClassification *QueryClassifyOutput `json:"queryClassification,omitempty"`
    
    // 摘要信息
    Summary *SummaryGenerateOutput `json:"summary,omitempty"`
    
    // 记忆信息
    Memory *MemoryStoreOutput `json:"memory,omitempty"`
    
    // 执行的步骤
    ExecutedSteps []string `json:"executedSteps"`
    
    // 总耗时（毫秒）
    TotalTime int64 `json:"totalTime"`
    
    // 各步骤耗时
    StepTimings map[string]int64 `json:"stepTimings"`
}
```

**Flow 执行步骤**：

1. **查询分类（可选）**
   - 如果 EnableQueryClassify=true
   - 调用 queryClassifyFlow
   - 根据分类结果调整策略

2. **上下文构建**
   - 调用 contextBuildFlow
   - 应用查询分类的建议

3. **上下文优化（可选）**
   - 如果 AutoOptimizeContext=true
   - 调用 contextOptimizeFlow

4. **对话生成**
   - 根据 EnableStreaming 选择 Flow
   - 调用 chatGenerateFlow 或 chatStreamFlow

5. **记忆存储（可选）**
   - 如果 SaveMemory=true
   - 调用 memoryStoreFlow

6. **摘要生成（可选）**
   - 如果 AutoGenerateSummary=true
   - 检查是否需要摘要
   - 调用 summaryGenerateFlow

7. **后处理**
   - 更新会话统计
   - 记录执行日志
   - 触发异步任务

**并行执行优化**：

- 查询分类和上下文构建可以并行
- 记忆存储和摘要生成可以异步执行

**依赖 Flow**：

- queryClassifyFlow
- contextBuildFlow
- contextOptimizeFlow
- chatGenerateFlow / chatStreamFlow
- memoryStoreFlow
- summaryGenerateFlow

**错误处理**：

- 任何步骤失败都应该记录
- 关键步骤失败应该中断流程
- 可选步骤失败不应该影响主流程

**性能要求**：

- 总响应时间 < 5s（不含 AI 生成）
- 支持并发请求 > 50 QPS

**验收标准**：

- 应该正确执行所有配置的步骤
- 应该记录各步骤的耗时
- 可选步骤失败不应该影响主流程
- 应该返回完整的执行信息

### 8.2 批量对话处理 Flow

**需求描述**：
批量处理多个对话请求，优化资源使用和性能。

**Flow 名称**：`batchConversationFlow`

**输入定义**：

```go
type BatchConversationInput struct {
    // 批量请求列表
    Requests []CompleteConversationInput `json:"requests" validate:"required,min=1,max=10"`
    
    // 批处理配置
    BatchConfig *BatchConfig `json:"batchConfig,omitempty"`
}

type BatchConfig struct {
    // 并发数
    Concurrency int `json:"concurrency" validate:"min=1,max=5"`
    
    // 超时时间（秒）
    Timeout int `json:"timeout" validate:"min=10,max=300"`
    
    // 失败策略
    FailureStrategy string `json:"failureStrategy" validate:"oneof=continue abort"`
}
```

**输出定义**：

```go
type BatchConversationOutput struct {
    // 成功的响应
    Responses []CompleteConversationOutput `json:"responses"`
    
    // 失败的请求
    Failures []BatchFailure `json:"failures"`
    
    // 成功数量
    SuccessCount int `json:"successCount"`
    
    // 失败数量
    FailureCount int `json:"failureCount"`
    
    // 总耗时（毫秒）
    TotalTime int64 `json:"totalTime"`
}

type BatchFailure struct {
    Index   int        `json:"index"`
    Request CompleteConversationInput `json:"request"`
    Error   *FlowError `json:"error"`
}
```

**验收标准**：

- 应该并发处理多个请求
- 应该正确处理失败情况
- 应该遵守并发限制
- 应该在超时时间内完成

### 8.3 会话健康检查 Flow

**需求描述**：
定期检查会话的健康状态，识别问题并提供修复建议。

**Flow 名称**：`sessionHealthCheckFlow`

**输入定义**：

```go
type SessionHealthCheckInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 检查项目
    CheckItems []string `json:"checkItems" validate:"dive,oneof=context token memory summary performance"`
    
    // 是否自动修复
    AutoFix bool `json:"autoFix"`
}
```

**输出定义**：

```go
type SessionHealthCheckOutput struct {
    // 会话ID
    SessionID string `json:"sessionId"`
    
    // 整体健康评分（0-1）
    OverallScore float64 `json:"overallScore"`
    
    // 健康状态
    HealthStatus string `json:"healthStatus"` // "healthy", "warning", "critical"
    
    // 各项检查结果
    CheckResults []HealthCheckResult `json:"checkResults"`
    
    // 发现的问题
    Issues []HealthIssue `json:"issues"`
    
    // 修复操作
    FixActions []FixAction `json:"fixActions"`
    
    // 检查耗时（毫秒）
    CheckTime int64 `json:"checkTime"`
}

type HealthCheckResult struct {
    Item   string  `json:"item"`
    Score  float64 `json:"score"`
    Status string  `json:"status"`
    Details string `json:"details"`
}

type HealthIssue struct {
    Severity    string `json:"severity"` // "low", "medium", "high", "critical"
    Category    string `json:"category"`
    Description string `json:"description"`
    Impact      string `json:"impact"`
}

type FixAction struct {
    Action      string `json:"action"`
    Description string `json:"description"`
    Executed    bool   `json:"executed"`
    Result      string `json:"result,omitempty"`
}
```

**检查项目**：

1. **context（上下文健康）**
   - 上下文大小是否合理
   - 上下文质量是否良好
   - 是否有断层

2. **token（Token 使用）**
   - Token 使用率
   - 是否接近上限
   - 增长趋势

3. **memory（记忆健康）**
   - 记忆数量
   - 记忆质量
   - 访问模式

4. **summary（摘要状态）**
   - 是否需要摘要
   - 摘要质量
   - 摘要时效性

5. **performance（性能指标）**
   - 响应时间
   - 错误率
   - 资源使用

**自动修复操作**：

- 压缩上下文
- 生成摘要
- 清理低质量记忆
- 重建索引

**验收标准**：

- 应该准确评估会话健康状态
- 应该识别具体问题
- 自动修复应该有效
- 应该提供可操作的建议

## 9. Token 管理策略

### 9.1 Token 预算管理 Flow

**需求描述**：
管理会话的 Token 预算，防止超限，优化使用效率。

**Flow 名称**：`tokenBudgetFlow`

**输入定义**：

```go
type TokenBudgetInput struct {
    // 会话ID
    SessionID string `json:"sessionId" validate:"required,uuid"`
    
    // 预算类型
    BudgetType string `json:"budgetType" validate:"required,oneof=session daily monthly"`
    
    // 预算限制（0表示使用默认值）
    BudgetLimit int `json:"budgetLimit" validate:"min=0"`
    
    // 预计使用量
    EstimatedUsage int `json:"estimatedUsage" validate:"min=0"`
}
```

**输出定义**：

```go
type TokenBudgetOutput struct {
    // 会话ID
    SessionID string `json:"sessionId"`
    
    // 当前使用量
    CurrentUsage int `json:"currentUsage"`
    
    // 预算限制
    BudgetLimit int `json:"budgetLimit"`
    
    // 剩余预算
    RemainingBudget int `json:"remainingBudget"`
    
    // 使用率（0-1）
    UsageRate float64 `json:"usageRate"`
    
    // 预算状态
    BudgetStatus string `json:"budgetStatus"` // "normal", "warning", "critical", "exceeded"
    
    // 是否允许继续
    AllowContinue bool `json:"allowContinue"`
    
    // 建议操作
    Recommendations []string `json:"recommendations"`
    
    // 预测耗尽时间
    PredictedExhaustion string `json:"predictedExhaustion,omitempty"`
}
```

**预算状态定义**：

- `normal`：使用率 < 70%
- `warning`：使用率 70-90%
- `critical`：使用率 90-100%
- `exceeded`：使用率 > 100%

**Flow 执行步骤**：

1. **获取当前使用量**
   - 查询会话的 Token 使用统计
   - 根据 BudgetType 计算范围内的使用量

2. **获取预算限制**
   - 如果提供了 BudgetLimit，使用该值
   - 否则，从配置中获取默认值
   - 考虑租户级别的配额

3. **计算剩余预算**
   - 剩余预算 = 预算限制 - 当前使用量
   - 如果有预计使用量，减去该值

4. **评估预算状态**
   - 计算使用率
   - 确定预算状态
   - 判断是否允许继续

5. **生成建议**
   - 如果接近上限，建议优化上下文
   - 如果超限，建议升级配额或等待重置

6. **预测耗尽时间**
   - 基于历史使用趋势
   - 预测预算耗尽的时间

**依赖服务**：

- SessionRepository：会话统计
- TenantRepository：租户配额
- UsageRepository：使用记录

**验收标准**：

- 应该准确计算 Token 使用量
- 应该正确判断预算状态
- 应该提供合理的建议
- 预测应该基于实际趋势

### 9.2 Token 优化策略 Flow

**需求描述**：
自动优化 Token 使用，在保证质量的前提下减少消耗。

**Flow 名称**：`tokenOptimizeFlow`

**输入定义**：

```go
type TokenOptimizeInput struct {
    // 原始内容
    Content string `json:"content" validate:"required"`
    
    // 目标 Token 数
    TargetTokens int `json:"targetTokens" validate:"required,min=10"`
    
    // 优化策略
    Strategy string `json:"strategy" validate:"required,oneof=compress summarize truncate smart"`
    
    // 质量阈值（0-1）
    QualityThreshold float64 `json:"qualityThreshold" validate:"min=0,max=1"`
}
```

**输出定义**：

```go
type TokenOptimizeOutput struct {
    // 优化后的内容
    OptimizedContent string `json:"optimizedContent"`
    
    // 原始 Token 数
    OriginalTokens int `json:"originalTokens"`
    
    // 优化后 Token 数
    OptimizedTokens int `json:"optimizedTokens"`
    
    // 节省的 Token 数
    SavedTokens int `json:"savedTokens"`
    
    // 节省率
    SavedRate float64 `json:"savedRate"`
    
    // 质量评分（0-1）
    QualityScore float64 `json:"qualityScore"`
    
    // 使用的策略
    StrategyUsed string `json:"strategyUsed"`
    
    // 优化操作列表
    Operations []string `json:"operations"`
}
```

**优化策略**：

1. **compress（压缩）**
   - 移除冗余信息
   - 简化表达
   - 保留核心内容

2. **summarize（摘要）**
   - 生成内容摘要
   - 保留关键信息
   - 大幅减少 Token

3. **truncate（截断）**
   - 保留前 N 个 Token
   - 简单快速
   - 可能损失信息

4. **smart（智能）**
   - 综合多种策略
   - 自适应选择
   - 平衡质量和效率

**优化操作**：

- 移除停用词
- 合并重复内容
- 简化长句
- 使用缩写
- 移除格式标记

**验收标准**：

- 优化后的 Token 数应该接近目标值
- 质量评分应该高于阈值
- 应该记录所有优化操作
- 不同策略应该有明显差异

### 9.3 Token 使用分析 Flow

**需求描述**：
分析 Token 使用模式，提供优化建议和成本预测。

**Flow 名称**：`tokenAnalysisFlow`

**输入定义**：

```go
type TokenAnalysisInput struct {
    // 会话ID（可选）
    SessionID string `json:"sessionId" validate:"omitempty,uuid"`
    
    // 租户ID（可选）
    TenantID string `json:"tenantId" validate:"omitempty,uuid"`
    
    // 分析时间范围（天数）
    TimeRangeDays int `json:"timeRangeDays" validate:"min=1,max=90"`
    
    // 分析维度
    Dimensions []string `json:"dimensions" validate:"dive,oneof=usage trend cost efficiency"`
}
```

**输出定义**：

```go
type TokenAnalysisOutput struct {
    // 总使用量
    TotalUsage int `json:"totalUsage"`
    
    // 平均每日使用量
    AverageDailyUsage int `json:"averageDailyUsage"`
    
    // 峰值使用量
    PeakUsage int `json:"peakUsage"`
    
    // 使用趋势
    UsageTrend string `json:"usageTrend"` // "increasing", "stable", "decreasing"
    
    // 成本估算
    EstimatedCost float64 `json:"estimatedCost"`
    
    // 效率评分（0-1）
    EfficiencyScore float64 `json:"efficiencyScore"`
    
    // 使用分布
    UsageDistribution map[string]int `json:"usageDistribution"`
    
    // 优化建议
    Recommendations []OptimizationRecommendation `json:"recommendations"`
    
    // 预测
    Predictions *UsagePrediction `json:"predictions,omitempty"`
}

type OptimizationRecommendation struct {
    Category    string  `json:"category"`
    Priority    string  `json:"priority"` // "high", "medium", "low"
    Description string  `json:"description"`
    Impact      string  `json:"impact"`
    Savings     int     `json:"savings"` // 预计节省的 Token 数
}

type UsagePrediction struct {
    NextDayUsage   int     `json:"nextDayUsage"`
    NextWeekUsage  int     `json:"nextWeekUsage"`
    NextMonthUsage int     `json:"nextMonthUsage"`
    Confidence     float64 `json:"confidence"`
}
```

**分析维度**：

1. **usage（使用量分析）**
   - 总使用量统计
   - 时间分布
   - 类型分布（prompt vs completion）

2. **trend（趋势分析）**
   - 使用趋势识别
   - 异常检测
   - 周期性分析

3. **cost（成本分析）**
   - 成本估算
   - 成本分布
   - 成本优化机会

4. **efficiency（效率分析）**
   - Token 利用率
   - 浪费识别
   - 优化空间

**优化建议类别**：

- 上下文优化
- 摘要策略调整
- 模型选择优化
- 缓存策略改进
- 批处理优化

**验收标准**：

- 应该准确统计 Token 使用量
- 应该识别使用趋势
- 应该提供可操作的建议
- 预测应该有合理的置信度

## 10. 多租户隔离

### 10.1 租户权限验证

**需求描述**：
在所有 Flow 中实施严格的租户权限验证，确保数据隔离。

**验证规则**：

1. **会话访问验证**
   - 验证会话是否属于当前租户
   - 平台管理员可以访问所有会话
   - 租户管理员只能访问自己租户的会话
   - 普通用户只能访问自己的会话

2. **记忆访问验证**
   - 验证记忆是否属于当前租户
   - 跨会话检索必须限制在同一租户内

3. **配额验证**
   - 检查租户级别的 Token 配额
   - 检查会话级别的限制

**实现要点**：

```go
// 在每个 Flow 开始时调用
func validateTenantAccess(ctx context.Context, sessionID string) error {
    // 1. 获取 JWT 声明
    claims := middleware.GetJWTClaims(ctx)
    if claims == nil {
        return errors.NewUnauthorizedError("未认证")
    }
    
    // 2. 查询会话
    session, err := sessionRepo.GetByID(ctx, sessionID)
    if err != nil {
        return errors.NewNotFoundError("会话不存在")
    }
    
    // 3. 平台管理员可以访问所有会话
    if hasRole(claims, model.RoleSystemAdmin) {
        return nil
    }
    
    // 4. 获取会话所属用户的租户ID
    sessionUser, err := userRepo.GetByID(ctx, session.UserID.String())
    if err != nil {
        return errors.NewInternalError(err)
    }
    
    // 5. 验证租户ID匹配
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

**审计日志**：

- 记录所有权限验证失败的尝试
- 记录跨租户访问尝试
- 记录配额超限情况

### 10.2 数据隔离策略

**需求描述**：
确保所有数据查询都包含租户过滤条件。

**实施规则**：

1. **数据库查询**
   - 所有查询必须包含租户ID过滤
   - 使用参数化查询防止注入
   - 索引优化（包含租户ID）

2. **向量检索**
   - 向量搜索必须限制在租户范围内
   - 使用复合索引（tenant_id + embedding）

3. **缓存隔离**
   - 缓存键包含租户ID
   - 防止缓存穿透

**示例实现**：

```go
// 向量检索时的租户过滤
func (r *memoryRepository) SearchByVector(
    ctx context.Context,
    tenantID string,
    embedding []float32,
    topK int,
) ([]*model.ConversationMemory, error) {
    var memories []*model.ConversationMemory
    
    err := r.db.WithContext(ctx).
        Where("tenant_id = ?", tenantID).
        Where("is_deleted = ?", false).
        Order(gorm.Expr("embedding <=> ?", embedding)).
        Limit(topK).
        Find(&memories).Error
    
    return memories, err
}
```

### 10.3 配额管理

**需求描述**：
实施租户级别和会话级别的配额管理。

**配额类型**：

1. **租户配额**
   - 每日 Token 限制
   - 每月 Token 限制
   - 会话数量限制
   - 存储空间限制

2. **会话配额**
   - 单次对话 Token 限制
   - 上下文大小限制
   - 消息数量限制

**配额检查**：

```go
func checkTenantQuota(ctx context.Context, tenantID string, estimatedTokens int) error {
    // 1. 获取租户配额配置
    quota, err := quotaRepo.GetByTenantID(ctx, tenantID)
    if err != nil {
        return err
    }
    
    // 2. 获取当前使用量
    usage, err := usageRepo.GetDailyUsage(ctx, tenantID)
    if err != nil {
        return err
    }
    
    // 3. 检查是否超限
    if usage + estimatedTokens > quota.DailyLimit {
        return errors.NewQuotaExceededError("租户每日配额已用尽")
    }
    
    return nil
}
```

**验收标准**：

- 所有 Flow 都应该实施租户验证
- 数据查询应该包含租户过滤
- 配额检查应该在操作前执行
- 应该记录所有权限违规尝试

## 11. 监控告警

### 11.1 Flow 执行监控

**需求描述**：
监控所有 Genkit Flow 的执行情况，收集性能指标和错误信息。

**监控指标**：

1. **执行指标**
   - Flow 执行次数
   - 执行成功率
   - 平均执行时间
   - P50/P95/P99 延迟
   - 并发执行数

2. **错误指标**
   - 错误总数
   - 错误类型分布
   - 错误率趋势
   - 重试次数

3. **资源指标**
   - CPU 使用率
   - 内存使用量
   - 数据库连接数
   - 缓存命中率

4. **业务指标**
   - Token 使用量
   - 上下文大小分布
   - 摘要生成频率
   - 记忆检索效率

**实现方案**：

```go
// Flow 执行监控中间件
type FlowMonitor struct {
    metrics *monitoring.Metrics
    logger  *logger.Logger
}

func (m *FlowMonitor) MonitorFlow(flowName string, fn func() error) error {
    startTime := time.Now()
    
    // 增加执行计数
    m.metrics.IncrementCounter(fmt.Sprintf("flow.%s.executions", flowName))
    
    // 执行 Flow
    err := fn()
    
    // 记录执行时间
    duration := time.Since(startTime)
    m.metrics.RecordDuration(fmt.Sprintf("flow.%s.duration", flowName), duration)
    
    // 记录结果
    if err != nil {
        m.metrics.IncrementCounter(fmt.Sprintf("flow.%s.errors", flowName))
        m.logger.Error("Flow 执行失败",
            "flow", flowName,
            "duration", duration,
            "error", err,
        )
    } else {
        m.metrics.IncrementCounter(fmt.Sprintf("flow.%s.success", flowName))
    }
    
    return err
}
```

**Genkit 内置追踪**：

- 使用 Genkit 的内置追踪功能
- 集成 OpenTelemetry
- 导出到监控系统（Prometheus、Grafana）

**监控面板**：

1. **Flow 概览面板**
   - 所有 Flow 的执行统计
   - 成功率趋势图
   - 响应时间分布

2. **性能分析面板**
   - 各 Flow 的性能对比
   - 慢查询识别
   - 瓶颈分析

3. **错误分析面板**
   - 错误类型分布
   - 错误趋势
   - 错误详情

4. **资源使用面板**
   - CPU/内存使用趋势
   - 数据库性能
   - 缓存效率

**验收标准**：

- 所有 Flow 都应该被监控
- 指标应该实时更新
- 监控面板应该清晰易读
- 应该支持历史数据查询

### 11.2 告警规则配置

**需求描述**：
配置智能告警规则，及时发现和响应系统问题。

**告警级别**：

- `critical`：严重问题，需要立即处理
- `warning`：警告，需要关注
- `info`：信息，仅记录

**告警规则**：

1. **性能告警**

   ```yaml
   - name: flow_slow_execution
     level: warning
     condition: flow.duration.p95 > 5s
     message: "Flow 执行时间过长"
     
   - name: flow_timeout
     level: critical
     condition: flow.timeout.count > 10/min
     message: "Flow 频繁超时"
   ```

2. **错误告警**

   ```yaml
   - name: flow_error_rate_high
     level: critical
     condition: flow.error_rate > 10%
     message: "Flow 错误率过高"
     
   - name: ai_service_failure
     level: critical
     condition: ai.service.errors > 5/min
     message: "AI 服务频繁失败"
   ```

3. **资源告警**

   ```yaml
   - name: token_quota_warning
     level: warning
     condition: tenant.token.usage_rate > 80%
     message: "租户 Token 配额即将用尽"
     
   - name: memory_usage_high
     level: warning
     condition: system.memory.usage > 85%
     message: "系统内存使用率过高"
   ```

4. **业务告警**

   ```yaml
   - name: context_quality_low
     level: warning
     condition: context.quality_score < 0.6
     message: "上下文质量下降"
     
   - name: summary_generation_failed
     level: warning
     condition: summary.generation.failures > 3/hour
     message: "摘要生成频繁失败"
   ```

**告警通知**：

1. **通知渠道**
   - 邮件通知
   - Slack/钉钉通知
   - 短信通知（严重告警）
   - Webhook 通知

2. **通知策略**
   - 告警聚合（避免告警风暴）
   - 告警升级（未处理的告警升级）
   - 告警静默（维护期间）
   - 告警恢复通知

**实现示例**：

```go
type AlertRule struct {
    Name      string
    Level     string
    Condition func(metrics *Metrics) bool
    Message   string
    Cooldown  time.Duration
}

type AlertManager struct {
    rules       []AlertRule
    notifier    Notifier
    lastAlerted map[string]time.Time
}

func (am *AlertManager) CheckAlerts(metrics *Metrics) {
    for _, rule := range am.rules {
        // 检查冷却期
        if lastTime, exists := am.lastAlerted[rule.Name]; exists {
            if time.Since(lastTime) < rule.Cooldown {
                continue
            }
        }
        
        // 检查告警条件
        if rule.Condition(metrics) {
            // 发送告警
            am.notifier.Send(Alert{
                Name:    rule.Name,
                Level:   rule.Level,
                Message: rule.Message,
                Time:    time.Now(),
            })
            
            // 更新最后告警时间
            am.lastAlerted[rule.Name] = time.Now()
        }
    }
}
```

**验收标准**：

- 告警规则应该准确触发
- 告警通知应该及时送达
- 应该避免告警风暴
- 告警恢复应该自动通知

### 11.3 日志管理

**需求描述**：
统一管理所有 Flow 的日志，支持日志查询和分析。

**日志级别**：

- `DEBUG`：调试信息
- `INFO`：一般信息
- `WARN`：警告信息
- `ERROR`：错误信息

**日志内容**：

1. **Flow 执行日志**

   ```json
   {
     "timestamp": "2025-10-29T10:30:00Z",
     "level": "INFO",
     "flow": "contextBuildFlow",
     "session_id": "uuid",
     "user_id": "uuid",
     "tenant_id": "uuid",
     "duration_ms": 150,
     "status": "success",
     "context": {
       "total_tokens": 1500,
       "strategy": "hybrid"
     }
   }
   ```

2. **错误日志**

   ```json
   {
     "timestamp": "2025-10-29T10:30:00Z",
     "level": "ERROR",
     "flow": "chatGenerateFlow",
     "session_id": "uuid",
     "error_code": "AI_SERVICE_ERROR",
     "error_message": "AI 服务超时",
     "stack_trace": "...",
     "retry_count": 2
   }
   ```

3. **权限日志**

   ```json
   {
     "timestamp": "2025-10-29T10:30:00Z",
     "level": "WARN",
     "event": "permission_denied",
     "user_id": "uuid",
     "user_tenant_id": "uuid",
     "target_session_id": "uuid",
     "target_tenant_id": "uuid",
     "reason": "跨租户访问"
   }
   ```

**日志存储**：

- 使用结构化日志（JSON 格式）
- 集中式日志存储（ELK、Loki）
- 日志保留策略（30 天）
- 日志归档和压缩

**日志查询**：

- 按 Flow 名称查询
- 按会话 ID 查询
- 按用户/租户查询
- 按时间范围查询
- 按错误类型查询

**验收标准**：

- 所有 Flow 都应该记录日志
- 日志格式应该统一
- 日志应该包含完整的上下文信息
- 应该支持高效的日志查询

### 11.4 性能追踪

**需求描述**：
追踪 Flow 的执行链路，识别性能瓶颈。

**追踪内容**：

1. **Flow 执行链路**
   - Flow 调用关系
   - 各步骤耗时
   - 依赖服务调用

2. **数据库查询追踪**
   - SQL 语句
   - 执行时间
   - 返回行数

3. **外部服务调用追踪**
   - AI 服务调用
   - 向量服务调用
   - 缓存服务调用

**追踪实现**：

```go
// 使用 OpenTelemetry 进行追踪
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func TraceFlow(ctx context.Context, flowName string, fn func(context.Context) error) error {
    tracer := otel.Tracer("genkit-flows")
    ctx, span := tracer.Start(ctx, flowName)
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("flow.name", flowName),
        attribute.String("session.id", getSessionID(ctx)),
    )
    
    // 执行 Flow
    err := fn(ctx)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    
    return err
}
```

**追踪可视化**：

- Jaeger UI
- Zipkin UI
- Grafana Tempo

**验收标准**：

- 应该追踪完整的执行链路
- 应该识别性能瓶颈
- 追踪数据应该可视化
- 应该支持分布式追踪

### 11.5 健康检查端点

**需求描述**：
提供健康检查端点，用于监控系统状态。

**健康检查项**：

1. **基础健康检查**
   - 服务是否运行
   - 基本功能是否正常

2. **依赖服务检查**
   - 数据库连接
   - Redis 连接
   - AI 服务可用性
   - 向量服务可用性

3. **资源检查**
   - CPU 使用率
   - 内存使用率
   - 磁盘空间

4. **业务检查**
   - Flow 执行状态
   - 错误率
   - 响应时间

**端点定义**：

```go
// GET /health
type HealthResponse struct {
    Status      string            `json:"status"` // "healthy", "degraded", "unhealthy"
    Timestamp   string            `json:"timestamp"`
    Version     string            `json:"version"`
    Checks      map[string]Check  `json:"checks"`
}

type Check struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
    Latency int64  `json:"latency_ms,omitempty"`
}

// 示例响应
{
    "status": "healthy",
    "timestamp": "2025-10-29T10:30:00Z",
    "version": "1.0.0",
    "checks": {
        "database": {
            "status": "healthy",
            "latency_ms": 5
        },
        "redis": {
            "status": "healthy",
            "latency_ms": 2
        },
        "ai_service": {
            "status": "healthy",
            "latency_ms": 150
        },
        "flows": {
            "status": "healthy",
            "message": "所有 Flow 运行正常"
        }
    }
}
```

**健康状态定义**：

- `healthy`：所有检查通过
- `degraded`：部分检查失败，但核心功能可用
- `unhealthy`：核心功能不可用

**验收标准**：

- 健康检查应该快速响应（< 1s）
- 应该检查所有关键依赖
- 应该返回详细的状态信息
- 应该支持 Kubernetes 探针

## 12. 实施路线图

### 12.1 第一阶段：基础设施搭建（2周）

**目标**：搭建 Genkit Flow 基础设施，实现核心 Flow。

**任务清单**：

1. **Genkit 环境配置**
   - [ ] 安装和配置 Genkit
   - [ ] 配置 AI 提供商插件
   - [ ] 设置开发和测试环境
   - [ ] 配置日志和追踪

2. **数据模型实现**
   - [ ] 创建 conversation_memories 表
   - [ ] 创建 conversation_contexts 表
   - [ ] 配置 pgvector 扩展
   - [ ] 创建必要的索引

3. **核心 Repository 实现**
   - [ ] MemoryRepository
   - [ ] ContextRepository
   - [ ] 向量检索方法

4. **基础 Flow 实现**
   - [ ] contextBuildFlow
   - [ ] chatGenerateFlow
   - [ ] memorySearchFlow

**交付物**：

- Genkit 配置文档
- 数据库迁移脚本
- 基础 Flow 实现代码
- 单元测试

**验收标准**：

- Genkit 环境正常运行
- 数据库表创建成功
- 基础 Flow 可以正常执行
- 单元测试覆盖率 > 70%

### 12.2 第二阶段：记忆管理实现（2周）

**目标**：实现三层记忆架构和智能记忆管理。

**任务清单**：

1. **短期记忆实现**
   - [ ] 实现消息查询和缓存
   - [ ] 实现滑动窗口策略
   - [ ] 性能优化

2. **长期记忆实现**
   - [ ] 实现向量生成服务
   - [ ] 实现向量相似度搜索
   - [ ] 实现记忆存储 Flow
   - [ ] 优化向量索引

3. **摘要记忆实现**
   - [ ] 实现摘要生成 Flow
   - [ ] 实现摘要触发策略
   - [ ] 实现摘要质量评估
   - [ ] 实现增量摘要

4. **记忆管理 Flow**
   - [ ] memoryStoreFlow
   - [ ] memoryCleanupFlow
   - [ ] summaryGenerateFlow
   - [ ] summaryTriggerFlow

**交付物**：

- 记忆管理服务代码
- 向量检索实现
- 摘要生成实现
- 集成测试

**验收标准**：

- 三层记忆架构正常工作
- 向量检索准确率 > 85%
- 摘要质量评分 > 0.7
- 记忆清理正常执行

### 12.3 第三阶段：智能优化实现（2周）

**目标**：实现智能上下文优化和 Token 管理。

**任务清单**：

1. **查询分类实现**
   - [ ] queryClassifyFlow
   - [ ] 查询特征提取
   - [ ] 分类模型训练
   - [ ] 策略推荐逻辑

2. **上下文优化实现**
   - [ ] contextOptimizeFlow
   - [ ] Token 优化策略
   - [ ] 质量评估机制
   - [ ] 自适应调整

3. **Token 管理实现**
   - [ ] tokenBudgetFlow
   - [ ] tokenOptimizeFlow
   - [ ] tokenAnalysisFlow
   - [ ] 配额管理

4. **健康检查实现**
   - [ ] sessionHealthCheckFlow
   - [ ] 健康度评估
   - [ ] 自动修复机制
   - [ ] 问题诊断

**交付物**：

- 查询分类服务
- 上下文优化服务
- Token 管理服务
- 健康检查服务

**验收标准**：

- 查询分类准确率 > 80%
- Token 优化节省率 > 30%
- 健康检查覆盖所有关键指标
- 自动修复成功率 > 70%

### 12.4 第四阶段：流式和高级功能（2周）

**目标**：实现流式对话和高级 Flow 组合。

**任务清单**：

1. **流式对话实现**
   - [ ] chatStreamFlow
   - [ ] 流式缓冲和发送
   - [ ] 流式错误处理
   - [ ] 流式上下文更新

2. **Flow 组合实现**
   - [ ] completeConversationFlow
   - [ ] batchConversationFlow
   - [ ] 并行执行优化
   - [ ] 错误恢复机制

3. **重试和回退实现**
   - [ ] chatRetryFlow
   - [ ] streamErrorRecoveryFlow
   - [ ] 重试策略
   - [ ] 回退机制

4. **多轮对话管理**
   - [ ] multiTurnChatFlow
   - [ ] 对话状态管理
   - [ ] 上下文连贯性保证

**交付物**：

- 流式对话实现
- Flow 组合框架
- 错误处理机制
- 性能测试报告

**验收标准**：

- 流式响应延迟 < 100ms
- Flow 组合正确执行
- 错误恢复成功率 > 80%
- 多轮对话上下文连贯

### 12.5 第五阶段：监控和优化（1周）

**目标**：完善监控告警系统，优化性能。

**任务清单**：

1. **监控系统搭建**
   - [ ] 配置 Prometheus
   - [ ] 配置 Grafana
   - [ ] 创建监控面板
   - [ ] 集成 OpenTelemetry

2. **告警规则配置**
   - [ ] 配置性能告警
   - [ ] 配置错误告警
   - [ ] 配置资源告警
   - [ ] 配置业务告警

3. **日志系统完善**
   - [ ] 统一日志格式
   - [ ] 配置日志收集
   - [ ] 实现日志查询
   - [ ] 日志归档策略

4. **性能优化**
   - [ ] 数据库查询优化
   - [ ] 缓存策略优化
   - [ ] 向量检索优化
   - [ ] 并发性能优化

**交付物**：

- 监控面板
- 告警规则配置
- 日志系统文档
- 性能优化报告

**验收标准**：

- 监控指标完整
- 告警及时准确
- 日志可查询分析
- 性能提升 > 30%

### 12.6 第六阶段：测试和上线（1周）

**目标**：完成全面测试，准备生产环境上线。

**任务清单**：

1. **功能测试**
   - [ ] 所有 Flow 功能测试
   - [ ] 集成测试
   - [ ] 端到端测试
   - [ ] 边界条件测试

2. **性能测试**
   - [ ] 压力测试
   - [ ] 并发测试
   - [ ] 长时间运行测试
   - [ ] 资源使用测试

3. **安全测试**
   - [ ] 权限验证测试
   - [ ] 租户隔离测试
   - [ ] 配额限制测试
   - [ ] 注入攻击测试

4. **上线准备**
   - [ ] 生产环境配置
   - [ ] 数据迁移脚本
   - [ ] 回滚方案
   - [ ] 运维文档

**交付物**：

- 测试报告
- 性能基准
- 安全审计报告
- 上线检查清单

**验收标准**：

- 所有测试通过
- 性能达标
- 安全无漏洞
- 文档完整

### 12.7 里程碑和时间线

```
Week 1-2:  基础设施搭建
  ├─ Genkit 环境配置
  ├─ 数据模型实现
  └─ 基础 Flow 实现

Week 3-4:  记忆管理实现
  ├─ 短期记忆
  ├─ 长期记忆
  └─ 摘要记忆

Week 5-6:  智能优化实现
  ├─ 查询分类
  ├─ 上下文优化
  └─ Token 管理

Week 7-8:  流式和高级功能
  ├─ 流式对话
  ├─ Flow 组合
  └─ 错误处理

Week 9:    监控和优化
  ├─ 监控系统
  ├─ 告警配置
  └─ 性能优化

Week 10:   测试和上线
  ├─ 全面测试
  ├─ 上线准备
  └─ 正式发布
```

### 12.8 风险和应对

**技术风险**：

1. **Genkit 学习曲线**
   - 风险：团队不熟悉 Genkit
   - 应对：提前学习，参考官方文档和示例

2. **向量检索性能**
   - 风险：大规模数据下性能下降
   - 应对：优化索引，使用缓存，考虑专业向量数据库

3. **AI 服务稳定性**
   - 风险：AI 服务不稳定或限流
   - 应对：实现重试机制，准备备用服务

**进度风险**：

1. **需求变更**
   - 风险：需求频繁变更影响进度
   - 应对：锁定核心需求，预留缓冲时间

2. **技术难题**
   - 风险：遇到技术难题导致延期
   - 应对：及时寻求帮助，准备备选方案

**资源风险**：

1. **人力不足**
   - 风险：开发人员不足
   - 应对：合理分配任务，考虑外部支持

2. **环境问题**
   - 风险：开发/测试环境不稳定
   - 应对：提前准备环境，做好备份

### 12.9 成功标准

**功能完整性**：

- ✅ 所有规划的 Flow 都已实现
- ✅ 三层记忆架构正常工作
- ✅ 智能优化功能有效
- ✅ 流式对话体验良好

**性能指标**：

- ✅ 上下文构建 < 200ms
- ✅ 对话生成 < 3s（不含 AI）
- ✅ 向量检索 < 100ms
- ✅ 并发支持 > 100 QPS

**质量指标**：

- ✅ 单元测试覆盖率 > 80%
- ✅ 集成测试通过率 100%
- ✅ 代码审查通过
- ✅ 文档完整

**用户体验**：

- ✅ 响应时间满足要求
- ✅ 上下文相关性高
- ✅ Token 使用优化
- ✅ 错误处理友好

**运维指标**：

- ✅ 监控覆盖完整
- ✅ 告警及时准确
- ✅ 日志可追溯
- ✅ 故障恢复快速

## 13. 总结

本方案基于 Google Genkit 设计了完整的 AI 对话系统会话管理模块，包含以下核心特性：

### 13.1 核心优势

1. **类型安全**：Genkit Flow 提供完整的类型检查
2. **可观测性**：内置追踪和监控能力
3. **易于测试**：Flow 可以独立测试和调试
4. **模块化设计**：Flow 可以灵活组合和复用
5. **流式支持**：原生支持流式响应
6. **智能优化**：自适应的上下文和 Token 管理

### 13.2 技术亮点

1. **三层记忆架构**：短期、长期、摘要记忆协同工作
2. **智能上下文管理**：自动构建、优化和压缩
3. **查询分类路由**：根据查询类型动态调整策略
4. **Token 预算管理**：精确控制和优化 Token 使用
5. **会话健康检查**：主动发现和修复问题
6. **完整的多租户隔离**：严格的权限控制和数据隔离

### 13.3 实施建议

1. **分阶段实施**：按照路线图逐步推进
2. **持续测试**：每个阶段都要充分测试
3. **性能优化**：关注性能指标，及时优化
4. **文档完善**：保持文档与代码同步
5. **监控先行**：尽早建立监控体系
6. **用户反馈**：收集用户反馈，持续改进

### 13.4 后续规划

1. **功能增强**
   - 支持多模态对话（图片、语音）
   - 实现对话分支和回溯
   - 支持自定义记忆策略
   - 实现对话模板和预设

2. **性能优化**
   - 引入专业向量数据库
   - 实现分布式缓存
   - 优化数据库查询
   - 实现智能预加载

3. **智能化提升**
   - 引入强化学习优化策略
   - 实现个性化记忆管理
   - 智能对话路由
   - 自动化问题诊断

4. **生态集成**
   - 集成更多 AI 模型
   - 支持插件扩展
   - 提供 SDK 和 API
   - 构建开发者社区

---

**文档版本**：v1.0  
**最后更新**：2025-10-29  
**维护者**：AI 对话系统团队
