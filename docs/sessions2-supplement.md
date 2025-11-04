# AI 对话系统会话管理模块 - 实现细节补充

本文档是 `sessions2.md` 的补充，提供更详细的实现细节和技术规范。

## 目录

- [1. Genkit 具体实现细节](#1-genkit-具体实现细节)
- [2. 缓存策略](#2-缓存策略)
- [3. 错误处理和异常场景](#3-错误处理和异常场景)
- [4. 安全性设计](#4-安全性设计)
- [5. 性能优化](#5-性能优化)

---

## 1. Genkit 具体实现细节

### 1.1 Genkit 在 Go 中的基础使用

**项目结构**：

```
internal/
├── genkit/
│   ├── client.go          # Genkit 客户端封装
│   ├── config.go          # Genkit 配置
│   ├── flows/             # Flow 定义
│   │   ├── context.go     # 上下文相关 Flow
│   │   ├── chat.go        # 对话相关 Flow
│   │   ├── memory.go      # 记忆相关 Flow
│   │   └── summary.go     # 摘要相关 Flow
│   ├── middleware.go      # Flow 中间件
│   └── registry.go        # Flow 注册器
```

**Genkit 初始化示例**：

```go
// internal/genkit/client.go
package genkit

import (
    "context"
    "fmt"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googleai"
    "github.com/firebase/genkit/go/ai"
)

// Client Genkit 客户端
type Client struct {
    genkit *genkit.Genkit
    config *Config
}

// Config Genkit 配置
type Config struct {
    Provider     string
    APIKey       string
    DefaultModel string
    Timeout      int
    LogLevel     string
}

// NewClient 创建 Genkit 客户端
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
    // 初始化 Genkit
    g := genkit.Init(ctx, &genkit.Options{
        FlowAddr: ":3400",
        LogLevel: cfg.LogLevel,
    })
    
    // 配置 AI 插件
    if err := configureAIPlugin(g, cfg); err != nil {
        return nil, fmt.Errorf("配置 AI 插件失败: %w", err)
    }
    
    return &Client{
        genkit: g,
        config: cfg,
    }, nil
}

// configureAIPlugin 配置 AI 插件
func configureAIPlugin(g *genkit.Genkit, cfg *Config) error {
    switch cfg.Provider {
    case "google":
        return googleai.Init(context.Background(), &googleai.Config{
            APIKey: cfg.APIKey,
        })
    case "openai":
        // TODO: 配置 OpenAI 插件
        return fmt.Errorf("OpenAI 插件暂未实现")
    default:
        return fmt.Errorf("不支持的 AI 提供商: %s", cfg.Provider)
    }
}

// GetGenkit 获取 Genkit 实例
func (c *Client) GetGenkit() *genkit.Genkit {
    return c.genkit
}
```

### 1.2 Flow 注册和调用示例

**Flow 定义示例**：

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

// ContextBuildInput 上下文构建输入
type ContextBuildInput struct {
    SessionID   string `json:"sessionId"`
    UserQuery   string `json:"userQuery"`
    MaxTokens   int    `json:"maxTokens"`
    Strategy    string `json:"strategy"`
}

// ContextBuildOutput 上下文构建输出
type ContextBuildOutput struct {
    SessionID        string          `json:"sessionId"`
    Summary          *SummaryContext `json:"summary,omitempty"`
    LongTermMemories []MemoryContext `json:"longTermMemories,omitempty"`
    ShortTermMessages []MessageContext `json:"shortTermMessages"`
    TotalTokens      int             `json:"totalTokens"`
    Strategy         string          `json:"strategy"`
    QualityScore     float64         `json:"qualityScore"`
    BuildTime        int64           `json:"buildTime"`
}

// RegisterContextFlows 注册上下文相关的 Flow
func RegisterContextFlows(g *genkit.Genkit, contextSvc service.ContextService) {
    // 注册上下文构建 Flow
    genkit.DefineFlow(
        g,
        "contextBuildFlow",
        func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
            startTime := time.Now()
            
            // 1. 参数验证
            if err := validateContextInput(input); err != nil {
                return ContextBuildOutput{}, err
            }
            
            // 2. 调用服务层构建上下文
            result, err := contextSvc.BuildContext(ctx, input.SessionID, input.UserQuery)
            if err != nil {
                return ContextBuildOutput{}, fmt.Errorf("构建上下文失败: %w", err)
            }
            
            // 3. 转换为输出格式
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

// validateContextInput 验证输入参数
func validateContextInput(input ContextBuildInput) error {
    if input.SessionID == "" {
        return fmt.Errorf("sessionId 不能为空")
    }
    if input.UserQuery == "" {
        return fmt.Errorf("userQuery 不能为空")
    }
    if input.MaxTokens < 100 || input.MaxTokens > 32000 {
        return fmt.Errorf("maxTokens 必须在 100-32000 之间")
    }
    return nil
}
```

**Flow 注册器**：

```go
// internal/genkit/registry.go
package genkit

import (
    "context"
    
    "github.com/firebase/genkit/go/genkit"
    "genkit-ai-service/internal/genkit/flows"
    "genkit-ai-service/internal/service"
)

// Registry Flow 注册器
type Registry struct {
    client     *Client
    services   *Services
}

// Services 服务依赖
type Services struct {
    ContextService service.ContextService
    ChatService    service.ChatService
    MemoryService  service.MemoryService
    SummaryService service.SummaryService
}

// NewRegistry 创建注册器
func NewRegistry(client *Client, services *Services) *Registry {
    return &Registry{
        client:   client,
        services: services,
    }
}

// RegisterAllFlows 注册所有 Flow
func (r *Registry) RegisterAllFlows(ctx context.Context) error {
    g := r.client.GetGenkit()
    
    // 注册上下文相关 Flow
    flows.RegisterContextFlows(g, r.services.ContextService)
    
    // 注册对话相关 Flow
    flows.RegisterChatFlows(g, r.services.ChatService)
    
    // 注册记忆相关 Flow
    flows.RegisterMemoryFlows(g, r.services.MemoryService)
    
    // 注册摘要相关 Flow
    flows.RegisterSummaryFlows(g, r.services.SummaryService)
    
    return nil
}
```

### 1.3 Genkit 与现有服务层集成

**服务层接口定义**：

```go
// internal/service/context_service.go
package service

import (
    "context"
    "genkit-ai-service/internal/model"
)

// ContextService 上下文服务接口
type ContextService interface {
    // BuildContext 构建上下文
    BuildContext(ctx context.Context, sessionID, userQuery string) (*model.ContextResult, error)
    
    // OptimizeContext 优化上下文
    OptimizeContext(ctx context.Context, sessionID string, targetTokens int) (*model.ContextResult, error)
    
    // GetContextConfig 获取上下文配置
    GetContextConfig(ctx context.Context, sessionID string) (*model.ContextConfig, error)
}
```

**服务层实现**：

```go
// internal/service/context_service_impl.go
package service

import (
    "context"
    "fmt"
    
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
)

type contextServiceImpl struct {
    sessionRepo repository.SessionRepository
    messageRepo repository.MessageRepository
    memoryRepo  repository.MemoryRepository
    contextRepo repository.ContextRepository
}

// NewContextService 创建上下文服务
func NewContextService(
    sessionRepo repository.SessionRepository,
    messageRepo repository.MessageRepository,
    memoryRepo repository.MemoryRepository,
    contextRepo repository.ContextRepository,
) ContextService {
    return &contextServiceImpl{
        sessionRepo: sessionRepo,
        messageRepo: messageRepo,
        memoryRepo:  memoryRepo,
        contextRepo: contextRepo,
    }
}

// BuildContext 实现上下文构建
func (s *contextServiceImpl) BuildContext(
    ctx context.Context,
    sessionID, userQuery string,
) (*model.ContextResult, error) {
    // 1. 验证权限
    if err := s.validateAccess(ctx, sessionID); err != nil {
        return nil, err
    }
    
    // 2. 获取短期记忆
    messages, err := s.messageRepo.GetRecentMessages(ctx, sessionID, 10)
    if err != nil {
        return nil, fmt.Errorf("获取短期记忆失败: %w", err)
    }
    
    // 3. 获取长期记忆
    memories, err := s.memoryRepo.SearchByQuery(ctx, sessionID, userQuery, 5)
    if err != nil {
        // 记录错误但不中断流程
        logger.WarnContext(ctx, "获取长期记忆失败", "error", err)
    }
    
    // 4. 组合上下文
    result := &model.ContextResult{
        SessionID:         sessionID,
        ShortTermMessages: messages,
        LongTermMemories:  memories,
        TotalTokens:       s.calculateTokens(messages, memories),
    }
    
    return result, nil
}
```

**在 HTTP Handler 中调用 Flow**：

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
    genkitClient *genkit.Genkit
}

// HandleBuildContext 处理上下文构建请求
func (h *ContextHandler) HandleBuildContext(c *gin.Context) {
    var input flows.ContextBuildInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
        return
    }
    
    // 调用 Genkit Flow
    flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
        h.genkitClient,
        "contextBuildFlow",
    )
    
    output, err := flow.Run(c.Request.Context(), input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

### 1.4 Genkit 配置文件示例

**配置文件结构**：

```yaml
# config/genkit.yaml
genkit:
  # AI 提供商配置
  provider: "google"  # google, openai
  
  # API 配置
  api:
    key: "${GENAI_API_KEY}"
    endpoint: "https://generativelanguage.googleapis.com"
    timeout: 30  # 秒
    
  # 默认模型配置
  model:
    name: "gemini-1.5-flash"
    temperature: 0.7
    top_p: 0.9
    max_tokens: 2048
    
  # Flow 服务器配置
  server:
    addr: ":3400"
    enable_dev_ui: true  # 开发环境启用
    
  # 日志配置
  logging:
    level: "info"  # debug, info, warn, error
    format: "json"
    
  # 追踪配置
  tracing:
    enabled: true
    exporter: "otlp"  # otlp, jaeger, zipkin
    endpoint: "localhost:4317"
    
  # 重试配置
  retry:
    max_attempts: 3
    initial_interval: 1  # 秒
    max_interval: 10
    multiplier: 2.0
```

**配置加载代码**：

```go
// internal/genkit/config.go
package genkit

import (
    "fmt"
    "os"
    
    "gopkg.in/yaml.v3"
)

// LoadConfig 加载 Genkit 配置
func LoadConfig(configPath string) (*Config, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("读取配置文件失败: %w", err)
    }
    
    var cfg struct {
        Genkit Config `yaml:"genkit"`
    }
    
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("解析配置文件失败: %w", err)
    }
    
    // 替换环境变量
    cfg.Genkit.APIKey = os.ExpandEnv(cfg.Genkit.APIKey)
    
    return &cfg.Genkit, nil
}
```

**主程序集成**：

```go
// cmd/server/main.go
package main

import (
    "context"
    "log"
    
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/service"
)

func main() {
    ctx := context.Background()
    
    // 1. 加载配置
    cfg, err := genkit.LoadConfig("config/genkit.yaml")
    if err != nil {
        log.Fatal("加载配置失败:", err)
    }
    
    // 2. 初始化 Genkit 客户端
    genkitClient, err := genkit.NewClient(ctx, cfg)
    if err != nil {
        log.Fatal("初始化 Genkit 失败:", err)
    }
    
    // 3. 初始化服务层
    services := initServices()
    
    // 4. 注册所有 Flow
    registry := genkit.NewRegistry(genkitClient, services)
    if err := registry.RegisterAllFlows(ctx); err != nil {
        log.Fatal("注册 Flow 失败:", err)
    }
    
    // 5. 启动 HTTP 服务器
    startHTTPServer(genkitClient, services)
}
```

## 2. 缓存策略

### 2.1 Redis 缓存使用场景

**缓存场景分类**：

1. **会话上下文缓存**
   - 缓存最近构建的上下文
   - 避免重复查询数据库
   - TTL: 5 分钟

2. **向量查询结果缓存**
   - 缓存常见查询的向量检索结果
   - 减少向量计算开销
   - TTL: 30 分钟

3. **摘要缓存**
   - 缓存会话的最新摘要
   - 快速获取摘要信息
   - TTL: 1 小时

4. **用户会话列表缓存**
   - 缓存用户的会话列表
   - 减少列表查询压力
   - TTL: 10 分钟

5. **Token 使用统计缓存**
   - 缓存租户和会话的 Token 使用量
   - 实时更新配额信息
   - TTL: 5 分钟

### 2.2 缓存键设计规范

**命名规范**：

```
{namespace}:{resource}:{identifier}:{sub_key}

示例：
- session:context:uuid:latest
- session:summary:uuid:v1
- memory:vector:uuid:query_hash
- user:sessions:uuid:list
- tenant:quota:uuid:daily
```

**缓存键定义**：

```go
// internal/storage/cache_keys.go
package storage

import (
    "crypto/md5"
    "encoding/hex"
    "fmt"
)

// CacheKeys 缓存键管理
type CacheKeys struct {
    namespace string
}

// NewCacheKeys 创建缓存键管理器
func NewCacheKeys(namespace string) *CacheKeys {
    return &CacheKeys{namespace: namespace}
}

// SessionContext 会话上下文缓存键
func (k *CacheKeys) SessionContext(sessionID string) string {
    return fmt.Sprintf("%s:session:context:%s:latest", k.namespace, sessionID)
}

// SessionSummary 会话摘要缓存键
func (k *CacheKeys) SessionSummary(sessionID string) string {
    return fmt.Sprintf("%s:session:summary:%s:v1", k.namespace, sessionID)
}

// MemoryVectorQuery 向量查询缓存键
func (k *CacheKeys) MemoryVectorQuery(sessionID, query string) string {
    hash := k.hashQuery(query)
    return fmt.Sprintf("%s:memory:vector:%s:%s", k.namespace, sessionID, hash)
}

// UserSessions 用户会话列表缓存键
func (k *CacheKeys) UserSessions(userID string) string {
    return fmt.Sprintf("%s:user:sessions:%s:list", k.namespace, userID)
}

// TenantQuota 租户配额缓存键
func (k *CacheKeys) TenantQuota(tenantID, quotaType string) string {
    return fmt.Sprintf("%s:tenant:quota:%s:%s", k.namespace, tenantID, quotaType)
}

// TokenUsage Token 使用量缓存键
func (k *CacheKeys) TokenUsage(sessionID string) string {
    return fmt.Sprintf("%s:session:tokens:%s:usage", k.namespace, sessionID)
}

// hashQuery 生成查询哈希
func (k *CacheKeys) hashQuery(query string) string {
    hash := md5.Sum([]byte(query))
    return hex.EncodeToString(hash[:])
}
```

### 2.3 缓存失效策略

**失效策略类型**：

1. **TTL 自动过期**
   - 设置合理的过期时间
   - 根据数据更新频率调整

2. **主动失效**
   - 数据更新时主动删除缓存
   - 确保数据一致性

3. **版本控制**
   - 使用版本号管理缓存
   - 版本变更时自动失效

4. **标签失效**
   - 使用标签关联相关缓存
   - 批量失效相关缓存

**缓存服务实现**：

```go
// internal/storage/cache_service.go
package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
)

// CacheService 缓存服务
type CacheService struct {
    client *redis.Client
    keys   *CacheKeys
}

// NewCacheService 创建缓存服务
func NewCacheService(client *redis.Client, namespace string) *CacheService {
    return &CacheService{
        client: client,
        keys:   NewCacheKeys(namespace),
    }
}

// Get 获取缓存
func (s *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
    data, err := s.client.Get(ctx, key).Bytes()
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

// Set 设置缓存
func (s *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("序列化缓存失败: %w", err)
    }
    
    if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
        return fmt.Errorf("设置缓存失败: %w", err)
    }
    
    return nil
}

// Delete 删除缓存
func (s *CacheService) Delete(ctx context.Context, keys ...string) error {
    if len(keys) == 0 {
        return nil
    }
    
    if err := s.client.Del(ctx, keys...).Err(); err != nil {
        return fmt.Errorf("删除缓存失败: %w", err)
    }
    
    return nil
}

// DeletePattern 按模式删除缓存
func (s *CacheService) DeletePattern(ctx context.Context, pattern string) error {
    iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
    
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

// Exists 检查缓存是否存在
func (s *CacheService) Exists(ctx context.Context, key string) (bool, error) {
    n, err := s.client.Exists(ctx, key).Result()
    if err != nil {
        return false, fmt.Errorf("检查缓存失败: %w", err)
    }
    return n > 0, nil
}

// Increment 增加计数
func (s *CacheService) Increment(ctx context.Context, key string, delta int64) (int64, error) {
    val, err := s.client.IncrBy(ctx, key, delta).Result()
    if err != nil {
        return 0, fmt.Errorf("增加计数失败: %w", err)
    }
    return val, nil
}

// GetSessionContext 获取会话上下文缓存
func (s *CacheService) GetSessionContext(ctx context.Context, sessionID string) (*ContextCache, error) {
    key := s.keys.SessionContext(sessionID)
    var cache ContextCache
    if err := s.Get(ctx, key, &cache); err != nil {
        return nil, err
    }
    return &cache, nil
}

// SetSessionContext 设置会话上下文缓存
func (s *CacheService) SetSessionContext(ctx context.Context, sessionID string, context *ContextCache) error {
    key := s.keys.SessionContext(sessionID)
    return s.Set(ctx, key, context, 5*time.Minute)
}

// InvalidateSessionCache 失效会话相关的所有缓存
func (s *CacheService) InvalidateSessionCache(ctx context.Context, sessionID string) error {
    pattern := fmt.Sprintf("%s:session:*:%s:*", s.keys.namespace, sessionID)
    return s.DeletePattern(ctx, pattern)
}
```

### 2.4 缓存预热机制

**预热场景**：

1. **系统启动预热**
   - 预加载热门会话的上下文
   - 预加载常用配置

2. **定时预热**
   - 定期刷新即将过期的缓存
   - 预测性加载可能访问的数据

3. **触发式预热**
   - 用户登录时预热其会话列表
   - 会话创建时预热相关数据

**预热实现**：

```go
// internal/storage/cache_warmer.go
package storage

import (
    "context"
    "log"
    "time"
    
    "genkit-ai-service/internal/repository"
)

// CacheWarmer 缓存预热器
type CacheWarmer struct {
    cache       *CacheService
    sessionRepo repository.SessionRepository
    contextRepo repository.ContextRepository
}

// NewCacheWarmer 创建缓存预热器
func NewCacheWarmer(
    cache *CacheService,
    sessionRepo repository.SessionRepository,
    contextRepo repository.ContextRepository,
) *CacheWarmer {
    return &CacheWarmer{
        cache:       cache,
        sessionRepo: sessionRepo,
        contextRepo: contextRepo,
    }
}

// WarmupOnStartup 系统启动时预热
func (w *CacheWarmer) WarmupOnStartup(ctx context.Context) error {
    log.Println("开始缓存预热...")
    
    // 1. 预热活跃会话
    if err := w.warmupActiveSessions(ctx); err != nil {
        log.Printf("预热活跃会话失败: %v", err)
    }
    
    // 2. 预热热门查询
    if err := w.warmupPopularQueries(ctx); err != nil {
        log.Printf("预热热门查询失败: %v", err)
    }
    
    log.Println("缓存预热完成")
    return nil
}

// warmupActiveSessions 预热活跃会话
func (w *CacheWarmer) warmupActiveSessions(ctx context.Context) error {
    // 获取最近活跃的会话
    sessions, err := w.sessionRepo.GetRecentActive(ctx, 100)
    if err != nil {
        return err
    }
    
    for _, session := range sessions {
        // 预热会话上下文
        context, err := w.contextRepo.GetBySessionID(ctx, session.ID.String())
        if err != nil {
            continue
        }
        
        // 缓存上下文
        w.cache.SetSessionContext(ctx, session.ID.String(), &ContextCache{
            SessionID:   session.ID.String(),
            MaxTokens:   context.MaxTokens,
            Strategy:    context.Strategy,
            LastSummary: context.LastSummary,
        })
    }
    
    return nil
}

// StartPeriodicWarmup 启动定期预热
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

// WarmupUserSessions 预热用户会话
func (w *CacheWarmer) WarmupUserSessions(ctx context.Context, userID string) error {
    sessions, err := w.sessionRepo.GetByUserID(ctx, userID)
    if err != nil {
        return err
    }
    
    // 缓存会话列表
    key := w.cache.keys.UserSessions(userID)
    return w.cache.Set(ctx, key, sessions, 10*time.Minute)
}
```

**缓存使用示例**：

```go
// 在服务层使用缓存
func (s *contextServiceImpl) BuildContext(
    ctx context.Context,
    sessionID, userQuery string,
) (*model.ContextResult, error) {
    // 1. 尝试从缓存获取
    cached, err := s.cache.GetSessionContext(ctx, sessionID)
    if err == nil && cached != nil {
        // 缓存命中，检查是否需要更新
        if time.Since(cached.UpdatedAt) < 5*time.Minute {
            return cached.ToContextResult(), nil
        }
    }
    
    // 2. 缓存未命中，从数据库查询
    result, err := s.buildContextFromDB(ctx, sessionID, userQuery)
    if err != nil {
        return nil, err
    }
    
    // 3. 更新缓存
    go func() {
        cacheCtx := context.Background()
        s.cache.SetSessionContext(cacheCtx, sessionID, result.ToCache())
    }()
    
    return result, nil
}
```

## 3. 错误处理和异常场景

### 3.1 统一错误码定义

**错误码结构**：

```
{模块代码}{错误类型}{序号}

模块代码：
- 10: 通用错误
- 20: 认证授权
- 30: 会话管理
- 40: 上下文管理
- 50: 记忆管理
- 60: AI 服务
- 70: 数据库
- 80: 缓存
- 90: 外部服务

错误类型：
- 0: 成功
- 1: 客户端错误（4xx）
- 2: 服务端错误（5xx）
- 3: 业务逻辑错误
```

**错误码定义**：

```go
// internal/model/errors.go
package model

import (
    "fmt"
    "net/http"
)

// 错误码常量
const (
    // 通用错误 (10xxx)
    ErrCodeSuccess          = 10000
    ErrCodeBadRequest       = 10101
    ErrCodeUnauthorized     = 10102
    ErrCodeForbidden        = 10103
    ErrCodeNotFound         = 10104
    ErrCodeInternalError    = 10201
    ErrCodeServiceUnavailable = 10202
    
    // 认证授权错误 (20xxx)
    ErrCodeInvalidToken     = 20101
    ErrCodeTokenExpired     = 20102
    ErrCodePermissionDenied = 20103
    ErrCodeTenantMismatch   = 20104
    
    // 会话管理错误 (30xxx)
    ErrCodeSessionNotFound  = 30104
    ErrCodeSessionExpired   = 30301
    ErrCodeSessionLocked    = 30302
    ErrCodeSessionLimitExceeded = 30303
    
    // 上下文管理错误 (40xxx)
    ErrCodeContextBuildFailed = 40201
    ErrCodeContextTooLarge    = 40301
    ErrCodeTokenExceeded      = 40302
    
    // 记忆管理错误 (50xxx)
    ErrCodeMemoryNotFound     = 50104
    ErrCodeVectorGenerationFailed = 50201
    ErrCodeMemoryStorageFailed = 50202
    
    // AI 服务错误 (60xxx)
    ErrCodeAIServiceTimeout   = 60201
    ErrCodeAIServiceError     = 60202
    ErrCodeModelNotAvailable  = 60203
    ErrCodeQuotaExceeded      = 60301
    
    // 数据库错误 (70xxx)
    ErrCodeDatabaseError      = 70201
    ErrCodeDatabaseTimeout    = 70202
    ErrCodeDuplicateKey       = 70301
    
    // 缓存错误 (80xxx)
    ErrCodeCacheError         = 80201
    ErrCodeCacheNotFound      = 80104
    
    // 外部服务错误 (90xxx)
    ErrCodeExternalServiceError = 90201
)

// AppError 应用错误
type AppError struct {
    Code       int    `json:"code"`
    Message    string `json:"message"`
    Details    string `json:"details,omitempty"`
    HTTPStatus int    `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
    if e.Details != "" {
        return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
    }
    return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewAppError 创建应用错误
func NewAppError(code int, message, details string) *AppError {
    return &AppError{
        Code:       code,
        Message:    message,
        Details:    details,
        HTTPStatus: getHTTPStatus(code),
    }
}

// getHTTPStatus 根据错误码获取 HTTP 状态码
func getHTTPStatus(code int) int {
    errorType := (code / 100) % 10
    switch errorType {
    case 1: // 客户端错误
        switch code {
        case ErrCodeUnauthorized, ErrCodeInvalidToken, ErrCodeTokenExpired:
            return http.StatusUnauthorized
        case ErrCodeForbidden, ErrCodePermissionDenied:
            return http.StatusForbidden
        case ErrCodeNotFound, ErrCodeSessionNotFound, ErrCodeMemoryNotFound:
            return http.StatusNotFound
        default:
            return http.StatusBadRequest
        }
    case 2: // 服务端错误
        return http.StatusInternalServerError
    case 3: // 业务逻辑错误
        return http.StatusUnprocessableEntity
    default:
        return http.StatusInternalServerError
    }
}

// 预定义错误
var (
    ErrBadRequest       = NewAppError(ErrCodeBadRequest, "请求参数错误", "")
    ErrUnauthorized     = NewAppError(ErrCodeUnauthorized, "未认证", "")
    ErrForbidden        = NewAppError(ErrCodeForbidden, "权限不足", "")
    ErrNotFound         = NewAppError(ErrCodeNotFound, "资源不存在", "")
    ErrInternalError    = NewAppError(ErrCodeInternalError, "内部服务器错误", "")
    ErrSessionNotFound  = NewAppError(ErrCodeSessionNotFound, "会话不存在", "")
    ErrTokenExceeded    = NewAppError(ErrCodeTokenExceeded, "Token 超限", "")
    ErrQuotaExceeded    = NewAppError(ErrCodeQuotaExceeded, "配额已用尽", "")
)
```

### 3.2 异常场景处理流程

**异常场景分类**：

1. **网络异常**
   - 连接超时
   - 连接中断
   - DNS 解析失败

2. **服务异常**
   - AI 服务不可用
   - 数据库连接失败
   - 缓存服务故障

3. **业务异常**
   - Token 超限
   - 配额不足
   - 权限不足

4. **数据异常**
   - 数据不存在
   - 数据格式错误
   - 数据一致性问题

**异常处理器**：

```go
// internal/middleware/error_handler.go
package middleware

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "genkit-ai-service/internal/model"
)

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        // 检查是否有错误
        if len(c.Errors) == 0 {
            return
        }
        
        // 获取最后一个错误
        err := c.Errors.Last().Err
        
        // 转换为应用错误
        var appErr *model.AppError
        switch e := err.(type) {
        case *model.AppError:
            appErr = e
        default:
            // 未知错误，转换为内部错误
            appErr = model.NewAppError(
                model.ErrCodeInternalError,
                "内部服务器错误",
                err.Error(),
            )
        }
        
        // 记录错误日志
        logError(c, appErr)
        
        // 返回错误响应
        c.JSON(appErr.HTTPStatus, gin.H{
            "code":    appErr.Code,
            "message": appErr.Message,
            "details": appErr.Details,
        })
    }
}

// logError 记录错误日志
func logError(c *gin.Context, err *model.AppError) {
    log.Printf("[ERROR] %s %s - Code: %d, Message: %s, Details: %s",
        c.Request.Method,
        c.Request.URL.Path,
        err.Code,
        err.Message,
        err.Details,
    )
}
```

**Flow 错误处理**：

```go
// internal/genkit/flows/error_handler.go
package flows

import (
    "context"
    "fmt"
    
    "genkit-ai-service/internal/model"
)

// HandleFlowError 处理 Flow 错误
func HandleFlowError(ctx context.Context, flowName string, err error) error {
    // 1. 记录错误
    logger.ErrorContext(ctx, "Flow 执行失败",
        "flow", flowName,
        "error", err,
    )
    
    // 2. 转换错误类型
    switch {
    case isTimeoutError(err):
        return model.NewAppError(
            model.ErrCodeAIServiceTimeout,
            "AI 服务超时",
            err.Error(),
        )
    case isQuotaError(err):
        return model.NewAppError(
            model.ErrCodeQuotaExceeded,
            "配额已用尽",
            err.Error(),
        )
    case isPermissionError(err):
        return model.NewAppError(
            model.ErrCodePermissionDenied,
            "权限不足",
            err.Error(),
        )
    default:
        return model.NewAppError(
            model.ErrCodeInternalError,
            "Flow 执行失败",
            err.Error(),
        )
    }
}

// isTimeoutError 判断是否为超时错误
func isTimeoutError(err error) bool {
    // 实现超时错误判断逻辑
    return false
}
```

### 3.3 降级策略

**降级场景**：

1. **AI 服务降级**
   - 主服务不可用时切换到备用服务
   - 使用缓存的历史响应
   - 返回预设的默认响应

2. **向量检索降级**
   - 向量服务故障时使用全文搜索
   - 使用缓存的检索结果
   - 跳过长期记忆检索

3. **摘要生成降级**
   - 摘要生成失败时使用简单截断
   - 使用历史摘要
   - 跳过摘要步骤

**降级实现**：

```go
// internal/service/degradation.go
package service

import (
    "context"
    "log"
    
    "genkit-ai-service/internal/model"
)

// DegradationService 降级服务
type DegradationService struct {
    cache CacheService
}

// NewDegradationService 创建降级服务
func NewDegradationService(cache CacheService) *DegradationService {
    return &DegradationService{cache: cache}
}

// DegradeAIService AI 服务降级
func (s *DegradationService) DegradeAIService(
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

// DegradeVectorSearch 向量检索降级
func (s *DegradationService) DegradeVectorSearch(
    ctx context.Context,
    sessionID string,
    query string,
) ([]model.Memory, error) {
    log.Printf("向量检索降级: session=%s", sessionID)
    
    // 1. 尝试使用全文搜索
    memories, err := s.fullTextSearch(ctx, sessionID, query)
    if err == nil {
        return memories, nil
    }
    
    // 2. 返回空结果
    return []model.Memory{}, nil
}

// DegradeSummaryGeneration 摘要生成降级
func (s *DegradationService) DegradeSummaryGeneration(
    ctx context.Context,
    messages []model.Message,
) (string, error) {
    log.Printf("摘要生成降级: messages=%d", len(messages))
    
    // 使用简单截断策略
    if len(messages) == 0 {
        return "", nil
    }
    
    // 取最后几条消息的内容
    var summary string
    for i := len(messages) - 1; i >= 0 && i >= len(messages)-5; i-- {
        summary = messages[i].Content + "\n" + summary
    }
    
    return summary, nil
}
```

### 3.4 熔断机制

**熔断器实现**：

```go
// internal/middleware/circuit_breaker.go
package middleware

import (
    "context"
    "errors"
    "sync"
    "time"
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
    mu              sync.RWMutex
    state           State
    failureCount    int
    successCount    int
    lastFailureTime time.Time
    
    // 配置
    maxFailures     int           // 最大失败次数
    timeout         time.Duration // 熔断超时时间
    halfOpenSuccess int           // 半开状态需要的成功次数
}

// State 熔断器状态
type State int

const (
    StateClosed State = iota // 关闭状态（正常）
    StateOpen                 // 打开状态（熔断）
    StateHalfOpen            // 半开状态（尝试恢复）
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        state:           StateClosed,
        maxFailures:     maxFailures,
        timeout:         timeout,
        halfOpenSuccess: 3,
    }
}

// Execute 执行操作
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
    // 检查熔断器状态
    if !cb.canExecute() {
        return errors.New("熔断器已打开")
    }
    
    // 执行操作
    err := fn()
    
    // 记录结果
    cb.recordResult(err)
    
    return err
}

// canExecute 检查是否可以执行
func (cb *CircuitBreaker) canExecute() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        // 检查是否可以进入半开状态
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

// recordResult 记录执行结果
func (cb *CircuitBreaker) recordResult(err error) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if err != nil {
        cb.failureCount++
        cb.lastFailureTime = time.Now()
        
        // 检查是否需要打开熔断器
        if cb.failureCount >= cb.maxFailures {
            cb.state = StateOpen
            cb.successCount = 0
        }
    } else {
        cb.successCount++
        
        // 半开状态下，成功次数达到阈值则关闭熔断器
        if cb.state == StateHalfOpen && cb.successCount >= cb.halfOpenSuccess {
            cb.state = StateClosed
            cb.failureCount = 0
            cb.successCount = 0
        }
    }
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() State {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}
```

**熔断器使用示例**：

```go
// 在服务中使用熔断器
type AIService struct {
    client         *genkit.Client
    circuitBreaker *CircuitBreaker
}

func (s *AIService) Generate(ctx context.Context, prompt string) (string, error) {
    var result string
    
    err := s.circuitBreaker.Execute(ctx, func() error {
        var err error
        result, err = s.client.Generate(ctx, prompt)
        return err
    })
    
    if err != nil {
        // 熔断器打开，执行降级逻辑
        if err.Error() == "熔断器已打开" {
            return s.degradationService.DegradeAIService(ctx, "", prompt)
        }
        return "", err
    }
    
    return result, nil
}
```
