# 降级服务 (Degradation Service)

## 概述

降级服务为 Genkit 会话管理模块提供服务降级策略，确保在外部服务故障时系统仍能提供基本功能。

## 功能特性

### 1. AI 服务降级

当 AI 服务不可用时，提供两层降级策略：

- **缓存响应**：从缓存中查找相似查询的历史响应
- **默认响应**：返回预设的友好提示信息

### 2. 向量检索降级

当向量服务故障时，提供两层降级策略：

- **全文搜索**：使用 PostgreSQL 的文本匹配功能
- **空结果**：全文搜索失败时返回空列表

### 3. 摘要生成降级

当 AI 摘要生成失败时，提供三种降级策略：

- **直接返回**：内容已经足够短时直接返回
- **提取关键句子**：智能提取关键信息
- **简单截断**：在完整词或句子处截断

## 使用方法

### 创建服务实例

```go
import (
    "genkit-ai-service/internal/service"
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/logger"
)

// 创建降级服务
degradationSvc := service.NewDegradationService(
    cacheService,
    memoryRepository,
    messageRepository,
    logger,
)
```

### AI 服务降级

```go
// AI 服务不可用时调用
result, err := degradationSvc.DegradeAIService(ctx, sessionID, userQuery)
if err != nil {
    // 处理错误
    return err
}

// 使用降级响应
response := result.Response
source := result.Source  // "cache" 或 "default"
cacheHit := result.CacheHit
degradationTime := result.DegradationTime
```

### 向量检索降级

```go
// 向量服务故障时调用
result, err := degradationSvc.DegradeVectorSearch(ctx, sessionID, query, topK)
if err != nil {
    // 处理错误
    return err
}

// 使用降级结果
memories := result.Memories
source := result.Source  // "fulltext" 或 "empty"
fullTextUsed := result.FullTextUsed
degradationTime := result.DegradationTime
```

### 摘要生成降级

```go
// 摘要生成失败时调用
result, err := degradationSvc.DegradeSummaryGeneration(ctx, messages, targetLength)
if err != nil {
    // 处理错误
    return err
}

// 使用降级摘要
summary := result.Summary
method := result.Method  // "direct", "extract" 或 "truncate"
originalLength := result.OriginalLength
summaryLength := result.SummaryLength
degradationTime := result.DegradationTime
```

## 集成示例

### 在 AI 服务中集成

```go
type AIService struct {
    client         AIClient
    degradationSvc service.DegradationService
}

func (s *AIService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    // 尝试调用 AI 服务
    response, err := s.client.Generate(ctx, req)
    if err != nil {
        // AI 服务失败，使用降级策略
        degradationResult, degradationErr := s.degradationSvc.DegradeAIService(
            ctx,
            req.SessionID,
            req.UserQuery,
        )
        if degradationErr != nil {
            return nil, degradationErr
        }
        
        // 返回降级响应，标记为降级状态
        return &GenerateResponse{
            Content:   degradationResult.Response,
            Degraded:  true,
            Source:    degradationResult.Source,
            CacheHit:  degradationResult.CacheHit,
        }, nil
    }
    
    return response, nil
}
```

### 在记忆服务中集成

```go
type MemoryService struct {
    vectorSvc      VectorService
    degradationSvc service.DegradationService
}

func (s *MemoryService) SearchMemories(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
    // 尝试向量检索
    memories, err := s.vectorSearch(ctx, req)
    if err != nil {
        // 向量检索失败，使用降级策略
        degradationResult, degradationErr := s.degradationSvc.DegradeVectorSearch(
            ctx,
            req.SessionID,
            req.Query,
            req.TopK,
        )
        if degradationErr != nil {
            return nil, degradationErr
        }
        
        // 返回降级结果，标记为降级状态
        return &SearchResult{
            Memories:     degradationResult.Memories,
            Degraded:     true,
            Source:       degradationResult.Source,
            FullTextUsed: degradationResult.FullTextUsed,
        }, nil
    }
    
    return &SearchResult{
        Memories: memories,
        Degraded: false,
    }, nil
}
```

## 降级策略详解

### AI 服务降级策略

#### 1. 缓存响应

- 使用查询哈希查找缓存
- 支持相似查询匹配（未来增强）
- 缓存键格式：`ai:response:{sessionID}:{queryHash}`

#### 2. 默认响应

根据查询类型返回不同的默认响应：

- **问候语**（你好、hello、hi）：返回友好的问候
- **帮助请求**（帮助、help、怎么、如何）：返回帮助提示
- **其他查询**：返回通用错误提示

### 向量检索降级策略

#### 1. 全文搜索

- 使用 PostgreSQL 的 ILIKE 操作符
- 不区分大小写的模糊匹配
- 按重要性和创建时间排序
- 支持租户隔离和软删除过滤

#### 2. 空结果

- 全文搜索失败时返回空列表
- 不会中断主流程
- 记录详细的错误日志

### 摘要生成降级策略

#### 1. 直接返回

- 条件：原始内容长度 ≤ 目标长度
- 操作：直接返回原始内容
- 适用场景：内容已经足够短

#### 2. 提取关键句子

- 条件：目标长度 ≥ 200 字符
- 操作：智能提取关键句子
- 特性：
  - 优先提取用户问题和助手回答
  - 跳过系统消息
  - 支持中英文句子分割
  - 保持句子完整性

#### 3. 简单截断

- 条件：目标长度 < 200 字符
- 操作：在完整词或句子处截断
- 特性：
  - 尝试在句子结束符处截断
  - 否则在最后一个空格处截断
  - 自动添加省略号

## 性能监控

所有降级操作都记录了处理耗时，便于：

- 性能分析和优化
- 监控降级策略的效率
- 识别性能瓶颈

## 日志记录

每个降级操作都记录了详细的日志信息：

- **降级触发**：记录触发原因和上下文信息
- **降级执行**：记录使用的降级策略
- **降级结果**：记录处理结果和耗时

日志级别：

- `WARN`：降级触发
- `INFO`：降级成功
- `ERROR`：降级失败

## 最佳实践

### 1. 及时降级

在检测到服务故障时立即触发降级，避免长时间等待：

```go
// 设置合理的超时时间
ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()

response, err := aiService.Generate(ctx, req)
if err != nil {
    // 立即降级
    return degradationSvc.DegradeAIService(ctx, sessionID, query)
}
```

### 2. 标记降级状态

在响应中明确标记降级状态，便于前端展示和监控：

```go
type Response struct {
    Content  string `json:"content"`
    Degraded bool   `json:"degraded"`
    Source   string `json:"source,omitempty"`
}
```

### 3. 监控降级率

定期监控降级次数和降级率，及时发现服务问题：

```go
// 记录降级指标
metrics.RecordDegradation("ai_service", source)
metrics.IncrementDegradationCount("ai_service")
```

### 4. 优化降级策略

根据实际使用情况优化降级策略：

- 调整缓存 TTL
- 优化全文搜索查询
- 改进默认响应内容
- 增强摘要提取算法

## 注意事项

### 1. 缓存一致性

- 确保缓存的响应内容是准确的
- 定期清理过期的缓存
- 避免缓存敏感信息

### 2. 全文搜索性能

- ILIKE 查询可能较慢，考虑使用 PostgreSQL 全文搜索
- 添加适当的索引优化查询性能
- 限制搜索结果数量

### 3. 摘要质量

- 降级摘要质量可能不如 AI 生成
- 在响应中明确标记降级状态
- 提供用户反馈机制

### 4. 错误处理

- 降级操作本身也可能失败
- 实现多层降级策略
- 记录详细的错误日志

## 未来增强

### 1. 智能缓存匹配

- 实现基于相似度的缓存查找
- 支持语义相似查询匹配
- 添加缓存预热机制

### 2. 高级全文搜索

- 使用 PostgreSQL 的全文搜索功能
- 添加中文分词支持
- 实现搜索结果排序优化

### 3. NLP 摘要提取

- 使用 NLP 工具进行句子分割
- 实现关键句子识别算法
- 支持多语言摘要生成

### 4. 配置化降级策略

- 支持通过配置文件定义降级策略
- 允许动态调整降级参数
- 实现降级策略的 A/B 测试

## 相关文档

- [需求文档](../../.kiro/specs/genkit-session-management/requirements.md) - 需求 32
- [设计文档](../../.kiro/specs/genkit-session-management/design.md) - 降级策略设计
- [任务总结](../../.kiro/specs/genkit-session-management/TASK_32_SUMMARY.md) - 实现总结

## 测试

运行测试：

```bash
go test -v ./internal/service -run TestDegrade
```

测试覆盖：

- AI 服务降级（缓存命中、默认响应）
- 向量检索降级（全文搜索成功、空结果）
- 摘要生成降级（截断、提取、直接返回）
