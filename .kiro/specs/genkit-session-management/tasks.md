# 实施计划

## 第一阶段：Repository 层实现

- [x] 1. 实现 MemoryRepository
  - 创建 `internal/repository/memory_repository.go` 接口文件
  - 创建 `internal/repository/memory_repository_impl.go` 实现文件
  - 实现 `Create` 方法：创建记忆记录
  - 实现 `GetByID` 方法：根据ID获取记忆
  - 实现 `SearchByVector` 方法：向量相似度搜索（会话内）
  - 实现 `SearchByVectorCrossSessions` 方法：跨会话向量搜索（租户内）
  - 实现 `UpdateAccessStats` 方法：更新访问统计
  - 实现 `DeleteByStrategy` 方法：按策略删除记忆
  - 实现 `GetExpiredMemories` 方法：获取过期记忆
  - 所有查询必须包含租户ID过滤和软删除过滤
  - _需求: 2.1, 2.2, 2.3, 3.1, 3.2_

- [x] 2. 实现 ContextRepository
  - 创建 `internal/repository/context_repository.go` 接口文件
  - 创建 `internal/repository/context_repository_impl.go` 实现文件
  - 实现 `Create` 方法：创建上下文配置
  - 实现 `GetBySessionID` 方法：根据会话ID获取上下文配置
  - 实现 `Update` 方法：更新上下文配置
  - 实现 `GetLatestSummary` 方法：获取最新摘要
  - 实现 `UpdateTokenUsage` 方法：更新Token使用统计
  - 所有查询必须包含租户ID过滤和软删除过滤
  - _需求: 1.1, 1.2, 1.3_

- [x] 3. 实现 SummaryRepository
  - 创建 `internal/repository/summary_repository.go` 接口文件
  - 创建 `internal/repository/summary_repository_impl.go` 实现文件
  - 实现 `Create` 方法：创建摘要
  - 实现 `GetByID` 方法：根据ID获取摘要
  - 实现 `GetLatestBySessionID` 方法：获取会话最新摘要
  - 实现 `ListBySessionID` 方法：获取会话摘要列表
  - 实现 `Update` 方法：更新摘要
  - 所有查询必须包含租户ID过滤和软删除过滤
  - _需求: 4.1, 4.2_

## 第二阶段：向量服务实现

- [x] 4. 实现 Qdrant 客户端服务
  - 创建 `internal/storage/qdrant_client.go` 接口文件
  - 创建 `internal/storage/qdrant_client_impl.go` 实现文件
  - 实现 `InitializeCollection` 方法：初始化单个共享 Collection
    - Collection 名称：`conversation_memories`
    - 向量维度：1536（text-embedding-ada-002）
    - 距离度量：Cosine
  - 实现 `UpsertVector` 方法：插入或更新向量
    - Payload 包含 `tenant_id` 字段（设置 `is_tenant=true` 索引）
    - Payload 包含 `session_id` 字段
    - Payload 包含 `memory_type` 字段
    - Payload 包含其他元数据
  - 实现 `SearchVectors` 方法：向量检索
    - 支持按 `tenant_id` 过滤（必须）
    - 支持按 `session_id` 过滤（可选）
    - 支持按 `memory_type` 过滤（可选）
  - 实现 `DeleteVector` 方法：删除向量
  - 实现 `DeleteByFilter` 方法：按条件批量删除
  - 配置自定义分片策略（按租户分片）
  - _需求: 2.2, 3.1_

- [x] 5. 实现向量嵌入服务
  - 创建 `internal/service/vector_service.go` 接口文件
  - 创建 `internal/service/vector_service_impl.go` 实现文件
  - 实现 `GenerateEmbedding` 方法：生成文本向量
  - 实现 `GenerateBatchEmbeddings` 方法：批量生成向量
  - 集成 Genkit 的嵌入模型
  - 添加错误处理和重试机制
  - _需求: 2.2, 3.1_

- [x] 6. 实现 Token 管理器
  - 创建 `internal/service/token_manager.go` 接口文件
  - 创建 `internal/service/token_manager_impl.go` 实现文件
  - 实现 `CalculateTokens` 方法：计算文本Token数
  - 实现 `CalculateContextTokens` 方法：计算上下文总Token数
  - 实现 `EstimateTokens` 方法：估算Token数量
  - 使用 tiktoken 或类似库进行Token计算
  - _需求: 1.2, 1.3_

## 第三阶段：核心服务层实现

- [x] 7. 实现 ContextService
  - 创建 `internal/service/context_service.go` 接口文件
  - 创建 `internal/service/context_service_impl.go` 实现文件
  - 实现 `BuildContext` 方法：构建会话上下文
    - 获取短期记忆（最近N条消息）
    - 获取长期记忆（向量检索）
    - 获取摘要（如果启用）
    - 计算Token数量
    - Token优化（如果超限）
    - 计算质量评分
  - 实现 `OptimizeContext` 方法：优化上下文Token使用
  - 实现 `validateAccess` 方法：验证租户访问权限
  - 集成缓存服务
  - 添加审计日志
  - _需求: 1.1, 1.2, 1.3, 2.1, 2.2, 4.1_

- [x] 8. 实现 MemoryService
  - 创建 `internal/service/memory_service.go` 接口文件
  - 创建 `internal/service/memory_service_impl.go` 实现文件
  - 实现 `SearchMemories` 方法：检索记忆
    - 生成查询向量
    - 执行向量检索
    - 更新访问统计
    - 支持跨会话检索
  - 实现 `StoreMemory` 方法：存储记忆
    - 生成向量嵌入
    - 计算重要性评分
    - 设置过期时间
  - 实现 `CleanupMemories` 方法：清理记忆
    - 支持多种清理策略（过期、低质量、未使用）
    - 支持软删除和硬删除
    - 批量处理
  - 实现 `UpdateMemoryAccess` 方法：更新记忆访问统计
  - 实现租户权限验证
  - _需求: 2.1, 2.2, 2.3, 3.1, 3.2_

- [x] 9. 实现 SummaryService
  - 创建 `internal/service/summary_service.go` 接口文件
  - 创建 `internal/service/summary_service_impl.go` 实现文件
  - 实现 `GenerateSummary` 方法：生成摘要
    - 获取消息列表
    - 调用 Genkit AI 生成摘要
    - 计算质量评分
    - 计算压缩率
    - 提取关键主题
  - 实现 `CheckSummaryTrigger` 方法：检查是否需要生成摘要
    - 检查消息数量阈值
    - 检查Token使用量
    - 计算紧急程度
  - 实现 `EvaluateSummaryQuality` 方法：评估摘要质量
    - 多维度评分
    - 关键信息覆盖率
    - 生成改进建议
  - 实现租户权限验证
  - _需求: 4.1, 4.2, 4.3_

## 第四阶段：缓存服务实现

- [x] 10. 实现缓存服务
  - 创建 `internal/storage/cache_service.go` 接口文件
  - 创建 `internal/storage/cache_service_impl.go` 实现文件
  - 实现 `Get` 方法：获取缓存
  - 实现 `Set` 方法：设置缓存
  - 实现 `Delete` 方法：删除缓存
  - 实现 `DeletePattern` 方法：按模式删除缓存
  - 实现 `Exists` 方法：检查缓存是否存在
  - 实现 `Increment` 方法：增量操作
  - 使用 Redis 作为缓存后端
  - 实现命名空间隔离
  - _需求: 1.3, 5.1_

- [x] 11. 实现缓存预热
  - 创建 `internal/storage/cache_warmer.go` 文件
  - 实现 `WarmupOnStartup` 方法：启动时预热
  - 实现 `warmupActiveSessions` 方法：预热活跃会话
  - 实现 `StartPeriodicWarmup` 方法：定期预热
  - 预热会话上下文配置
  - 预热常用摘要
  - _需求: 5.1_

## 第五阶段：Genkit Flow 实现

- [x] 12. 实现上下文构建 Flow
  - 创建 `internal/genkit/flows/context_flows.go` 文件
  - 定义 `ContextBuildInput` 结构体
  - 定义 `ContextBuildOutput` 结构体
  - 实现 `contextBuildFlow` Flow：构建会话上下文
    - 调用 ContextService.BuildContext
    - 记录执行时间
    - 记录监控指标
  - 实现 `RegisterContextFlows` 函数：注册Flow
  - 添加错误处理
  - _需求: 1.1, 1.2, 1.3_

- [x] 13. 实现记忆管理 Flow
  - 创建 `internal/genkit/flows/memory_flows.go` 文件
  - 定义 `MemorySearchInput` 结构体
  - 定义 `MemorySearchOutput` 结构体
  - 定义 `MemoryStoreInput` 结构体
  - 定义 `MemoryStoreOutput` 结构体
  - 实现 `memorySearchFlow` Flow：检索记忆
  - 实现 `memoryStoreFlow` Flow：存储记忆
  - 实现 `memoryCleanupFlow` Flow：清理记忆
  - 实现 `RegisterMemoryFlows` 函数：注册Flow
  - 添加错误处理
  - _需求: 2.1, 2.2, 2.3, 3.1, 3.2_

- [x] 14. 实现摘要生成 Flow
  - 创建 `internal/genkit/flows/summary_flows.go` 文件
  - 定义 `SummaryGenerateInput` 结构体
  - 定义 `SummaryGenerateOutput` 结构体
  - 实现 `summaryGenerateFlow` Flow：生成摘要
  - 实现 `summaryTriggerCheckFlow` Flow：检查摘要触发条件
  - 实现 `RegisterSummaryFlows` 函数：注册Flow
  - 添加错误处理
  - _需求: 4.1, 4.2, 4.3_

## 第六阶段：API Handler 实现

- [x] 15. 实现上下文管理 Handler
  - 创建 `internal/handler/context_handler.go` 文件
  - 实现 `HandleBuildContext` 方法：构建上下文API
    - 参数验证
    - 调用 contextBuildFlow
    - 返回标准响应格式
  - 实现 `HandleGetContextConfig` 方法：获取上下文配置API
  - 实现 `HandleUpdateContextConfig` 方法：更新上下文配置API
  - 添加JWT认证中间件
  - 添加租户权限验证
  - 添加请求日志
  - _需求: 1.1, 1.2, 1.3_

- [x] 16. 实现记忆管理 Handler
  - 创建 `internal/handler/memory_handler.go` 文件
  - 实现 `HandleSearchMemories` 方法：检索记忆API
  - 实现 `HandleStoreMemory` 方法：存储记忆API
  - 实现 `HandleCleanupMemories` 方法：清理记忆API
  - 实现 `HandleGetMemory` 方法：获取记忆详情API
  - 添加JWT认证中间件
  - 添加租户权限验证
  - 添加请求日志
  - _需求: 2.1, 2.2, 2.3, 3.1, 3.2_

- [x] 17. 实现摘要管理 Handler
  - 创建 `internal/handler/summary_handler.go` 文件
  - 实现 `HandleGenerateSummary` 方法：生成摘要API
  - 实现 `HandleGetSummary` 方法：获取摘要详情API
  - 实现 `HandleListSummaries` 方法：获取摘要列表API
  - 实现 `HandleCheckTrigger` 方法：检查摘要触发条件API
  - 添加JWT认证中间件
  - 添加租户权限验证
  - 添加请求日志
  - _需求: 4.1, 4.2, 4.3_

## 第七阶段：路由配置

- [x] 18. 配置上下文管理路由
  - 在 `internal/router/router.go` 中添加上下文管理路由组
  - `POST /api/v1/contexts/build` - 构建上下文
  - `GET /api/v1/contexts/:sessionId` - 获取上下文配置
  - `PUT /api/v1/contexts/:sessionId` - 更新上下文配置
  - 应用 JWT 认证中间件
  - 应用租户管理员权限中间件
  - _需求: 1.1, 1.2, 1.3_

- [x] 19. 配置记忆管理路由
  - 在 `internal/router/router.go` 中添加记忆管理路由组
  - `POST /api/v1/memories/search` - 检索记忆
  - `POST /api/v1/memories` - 存储记忆
  - `POST /api/v1/memories/cleanup` - 清理记忆
  - `GET /api/v1/memories/:id` - 获取记忆详情
  - 应用 JWT 认证中间件
  - 应用租户管理员权限中间件
  - _需求: 2.1, 2.2, 2.3, 3.1, 3.2_

- [x] 20. 配置摘要管理路由
  - 在 `internal/router/router.go` 中添加摘要管理路由组
  - `POST /api/v1/summaries` - 生成摘要
  - `GET /api/v1/summaries/:id` - 获取摘要详情
  - `GET /api/v1/summaries/session/:sessionId` - 获取会话摘要列表
  - `POST /api/v1/summaries/check-trigger` - 检查摘要触发条件
  - 应用 JWT 认证中间件
  - 应用租户管理员权限中间件
  - _需求: 4.1, 4.2, 4.3_

## 第八阶段：依赖注入和初始化

- [ ] 21. 配置依赖注入
  - 在 `cmd/server/main.go` 或 DI 容器中配置依赖注入
  - 初始化 Redis 客户端
  - 初始化 Genkit 客户端
  - 创建 Repository 实例
  - 创建 Service 实例
  - 创建 Handler 实例
  - 注册 Genkit Flows
  - 启动缓存预热
  - _需求: 所有需求_

- [ ] 22. 配置环境变量和配置文件
  - 在 `config/config.yaml` 中添加 Genkit 配置
    - API Key
    - 模型名称
    - 默认参数
  - 添加 Redis 配置
    - 主机地址
    - 端口
    - 密码
    - 数据库编号
  - 添加 Qdrant 配置
    - 主机地址
    - 端口
    - API Key（如果需要）
    - Collection 名称
    - 向量维度
    - 分片策略
  - 添加缓存配置
    - TTL 设置
    - 命名空间
  - _需求: 5.1_

## 第九阶段：监控和可观测性

- [ ] 23. 实现监控指标
  - 创建 `internal/monitoring/metrics.go` 文件
  - 定义 Prometheus 指标
    - Flow 执行次数
    - Flow 执行时间
    - Token 使用量
    - 缓存命中率
  - 实现 `RecordFlowExecution` 方法
  - 实现 `RecordFlowDuration` 方法
  - 实现 `RecordTokenUsage` 方法
  - 实现 `RecordCacheHit` 方法
  - 实现 `RecordCacheMiss` 方法
  - _需求: 5.2_

- [ ] 24. 实现 Flow 监控中间件
  - 创建 `internal/genkit/middleware.go` 文件
  - 实现 `FlowMonitor` 结构体
  - 实现 `MonitorFlow` 方法：包装Flow执行
    - 记录开始时间
    - 执行Flow
    - 记录执行时间
    - 记录执行结果
  - 在所有Flow中应用监控中间件
  - _需求: 5.2_

- [ ] 25. 实现日志记录
  - 创建 `internal/logger/logger.go` 文件（如果不存在）
  - 实现结构化日志
    - `InfoContext` 方法
    - `ErrorContext` 方法
    - `WarnContext` 方法
  - 实现 `LogEntry` 结构体
  - 实现 `buildLogEntry` 方法：从上下文提取信息
  - 在所有服务层方法中添加日志记录
  - 记录权限验证失败的审计日志
  - _需求: 5.2, 6.1_

- [ ] 26. 实现性能追踪
  - 创建 `internal/tracing/tracer.go` 文件
  - 集成 OpenTelemetry
  - 实现 `TraceFlow` 方法：追踪Flow执行
  - 在所有Flow中添加追踪
  - 配置 Jaeger 导出器
  - _需求: 5.2_

## 第十阶段：错误处理和降级

- [ ] 27. 实现统一错误处理
  - 在 `internal/model/errors.go` 中定义错误码
    - 上下文管理错误码（40xxx）
    - 记忆管理错误码（50xxx）
    - AI 服务错误码（60xxx）
  - 实现 `AppError` 结构体
  - 实现错误构造函数
  - 在所有服务层方法中使用统一错误
  - _需求: 5.3_

- [ ] 28. 实现降级策略
  - 创建 `internal/service/degradation_service.go` 接口文件
  - 创建 `internal/service/degradation_service_impl.go` 实现文件
  - 实现 `DegradeAIService` 方法：AI服务降级
    - 尝试从缓存获取响应
    - 返回默认响应
  - 实现 `DegradeVectorSearch` 方法：向量检索降级
    - 使用全文搜索
    - 返回空结果
  - 实现 `DegradeSummaryGeneration` 方法：摘要生成降级
  - _需求: 5.3_

- [ ] 29. 实现熔断机制
  - 创建 `internal/middleware/circuit_breaker.go` 文件
  - 实现 `CircuitBreaker` 结构体
  - 实现 `Execute` 方法：执行带熔断的操作
  - 实现 `canExecute` 方法：检查是否可以执行
  - 实现 `recordResult` 方法：记录执行结果
  - 支持三种状态：Closed、Open、HalfOpen
  - 在 AI 服务调用中应用熔断器
  - _需求: 5.3_

## 第十一阶段：安全和权限

- [ ] 30. 实现租户隔离验证
  - 在所有 Service 层方法中实现租户权限验证
  - 从 JWT 上下文获取租户ID
  - 验证目标资源是否属于当前租户
  - 平台管理员可以访问所有租户资源
  - 租户管理员只能访问自己租户的资源
  - 记录权限验证失败的审计日志
  - _需求: 6.1_

- [ ] 31. 实现 API 安全措施
  - 在所有 Handler 中添加输入验证
    - 参数类型验证
    - 长度限制
    - 格式验证
  - 防止 SQL 注入（使用参数化查询）
  - 防止 XSS 攻击（输出转义）
  - 实现速率限制中间件
    - 基于租户的速率限制
    - 基于 IP 的速率限制
  - _需求: 6.1_

## 第十二阶段：性能优化

- [ ] 32. 实现数据库查询优化
  - 验证所有索引已创建（在迁移文件中）
  - 使用预编译语句
  - 实现批量操作
  - 配置数据库连接池
  - 添加查询性能监控
  - _需求: 5.1_

- [ ] 33. 实现缓存优化
  - 实现多级缓存策略（本地 + Redis）
  - 实现缓存穿透防护
  - 实现缓存雪崩防护
  - 优化缓存键设计
  - 实现缓存版本控制
  - _需求: 5.1_

- [ ] 34. 实现 Qdrant 向量检索优化
  - 优化 Qdrant Collection 配置
    - 调整分片数量（按租户数量）
    - 配置副本数量
    - 优化索引参数（HNSW: m, ef_construction）
  - 实现批量向量生成和插入
  - 实现向量查询结果缓存
  - 实现异步向量生成
  - 配置租户级别的 Payload 索引（`is_tenant=true`）
  - 定期优化 Collection（compact、reindex）
  - _需求: 2.2, 3.1_

## 第十三阶段：集成和端到端测试

- [ ] 35. 编写 Repository 层集成测试
  - 创建 `internal/repository/memory_repository_test.go`
  - 创建 `internal/repository/context_repository_test.go`
  - 创建 `internal/repository/summary_repository_test.go`
  - 测试所有 CRUD 操作
  - 测试向量检索功能
  - 测试租户隔离
  - 使用测试数据库
  - _需求: 所有需求_

- [ ] 36. 编写 Service 层集成测试
  - 创建 `internal/service/context_service_test.go`
  - 创建 `internal/service/memory_service_test.go`
  - 创建 `internal/service/summary_service_test.go`
  - 使用 Mock 对象测试业务逻辑
  - 测试权限验证
  - 测试错误处理
  - _需求: 所有需求_

- [ ] 37. 编写 Flow 集成测试
  - 创建 `internal/genkit/flows/context_flows_test.go`
  - 创建 `internal/genkit/flows/memory_flows_test.go`
  - 创建 `internal/genkit/flows/summary_flows_test.go`
  - 测试 Flow 执行
  - 测试错误处理
  - 测试监控指标记录
  - _需求: 所有需求_

- [ ] 38. 编写 API 端到端测试
  - 创建 `test/integration/context_api_test.go`
  - 创建 `test/integration/memory_api_test.go`
  - 创建 `test/integration/summary_api_test.go`
  - 测试完整的 API 调用流程
  - 测试认证和授权
  - 测试租户隔离
  - 测试错误响应
  - _需求: 所有需求_

## 第十四阶段：文档和部署

- [ ] 39. 编写 API 文档
  - 更新 Swagger/OpenAPI 规范
  - 添加上下文管理 API 文档
  - 添加记忆管理 API 文档
  - 添加摘要管理 API 文档
  - 添加请求/响应示例
  - 添加错误码说明
  - _需求: 所有需求_

- [ ] 40. 编写部署文档
  - 创建部署指南
  - 添加环境变量配置说明
  - 添加 Redis 配置说明
  - 添加 PostgreSQL 配置说明（pgvector）
  - 添加 Qdrant 配置说明
    - 单 Collection 多租户架构
    - 分片策略配置
    - 租户索引配置
  - 添加 Genkit API Key 配置说明
  - 添加监控配置说明
  - _需求: 所有需求_

- [ ] 41. 配置 CI/CD 流水线
  - 添加单元测试步骤
  - 添加集成测试步骤
  - 添加代码覆盖率检查
  - 添加代码质量检查
  - 配置自动部署
  - _需求: 所有需求_
