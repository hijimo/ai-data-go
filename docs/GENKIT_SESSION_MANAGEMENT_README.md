# Genkit 会话管理模块

## 概述

Genkit 会话管理模块是一个基于 Google Genkit Go SDK 构建的智能会话管理系统，提供了完整的对话上下文管理、长期记忆存储、自动摘要生成和 Token 优化功能。

## 核心特性

### 1. 三层记忆架构

- **短期记忆**：最近的对话消息（默认 10 条）
- **长期记忆**：基于向量相似度的重要历史信息
- **摘要记忆**：自动生成的会话摘要

### 2. 智能上下文管理

- 自动查询分类和策略推荐
- 动态上下文构建和优化
- Token 预算管理和优化
- 质量评分和监控

### 3. 向量检索

- 基于 pgvector 的高性能向量存储
- 余弦相似度搜索
- 跨会话记忆检索
- 批量向量生成优化

### 4. 自动摘要

- 智能触发条件检测
- 多种摘要风格（简洁/详细/要点）
- 质量评估和优化建议
- 关键主题提取

### 5. 多租户支持

- 严格的租户隔离
- 基于角色的访问控制
- 审计日志记录
- 配额管理

## 快速开始

### 前置要求

- Go 1.21+
- PostgreSQL 14+ (with pgvector)
- Redis 6.0+
- Google AI API Key

### 安装

```bash
# 克隆代码
git clone https://github.com/your-org/genkit-service.git
cd genkit-service

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入必要的配置

# 运行数据库迁移
go run cmd/migrate/main.go up

# 启动服务
go run cmd/server/main.go
```

### 基本使用

```go
// 1. 构建会话上下文
contextFlow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
    g, "contextBuildFlow",
)

context, err := contextFlow.Run(ctx, flows.ContextBuildInput{
    SessionID:       "session-uuid",
    UserQuery:       "用户查询",
    MaxTokens:       4000,
    Strategy:        "auto",
    IncludeSummary:  true,
    IncludeLongTerm: true,
})

// 2. 生成对话回复
chatFlow := genkit.LookupFlow[flows.ChatGenerateInput, flows.ChatGenerateOutput](
    g, "chatGenerateFlow",
)

response, err := chatFlow.Run(ctx, flows.ChatGenerateInput{
    SessionID:   "session-uuid",
    UserMessage: "用户消息",
    ContextConfig: contextConfig,
    GenerateConfig: generateConfig,
})

// 3. 搜索长期记忆
memoryFlow := genkit.LookupFlow[flows.MemorySearchInput, flows.MemorySearchOutput](
    g, "memorySearchFlow",
)

memories, err := memoryFlow.Run(ctx, flows.MemorySearchInput{
    SessionID:     "session-uuid",
    Query:         "搜索查询",
    TopK:          5,
    MinSimilarity: 0.7,
})
```

## 架构设计

### 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        API Layer                             │
│  (REST API / Genkit Flow Endpoints)                         │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Genkit Flow Layer                       │
│  (Context / Chat / Memory / Summary / Token Flows)          │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  (Business Logic / Permission Control)                      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│  (Data Access / Tenant Filtering)                           │
└─────────────────────────────────────────────────────────────┘
                            │
┌──────────────────┬──────────────────┬──────────────────────┐
│   PostgreSQL     │      Redis       │    Google AI API     │
│   (pgvector)     │    (Cache)       │   (Generation/Embed) │
└──────────────────┴──────────────────┴──────────────────────┘
```

### 数据模型

- **conversation_messages**: 对话消息
- **conversation_memories**: 长期记忆（带向量）
- **conversation_contexts**: 上下文配置
- **conversation_summaries**: 会话摘要

## 文档

### 用户文档

- [API 文档](./API_DOCUMENTATION.md) - 完整的 REST API 参考
- [Flow 使用指南](./FLOW_USAGE_GUIDE.md) - Genkit Flow 使用说明
- [部署指南](./DEPLOYMENT_GUIDE.md) - 部署和配置说明
- [运维指南](./OPERATIONS_GUIDE.md) - 日常运维操作

### 开发文档

- [设计文档](../.kiro/specs/genkit-session-management/design.md) - 详细的技术设计
- [需求文档](../.kiro/specs/genkit-session-management/requirements.md) - 功能需求说明
- [任务列表](../.kiro/specs/genkit-session-management/tasks.md) - 实施任务清单

## API 示例

### 构建上下文

```bash
curl -X POST http://localhost:8080/api/v1/context/build \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-uuid",
    "userQuery": "用户查询",
    "maxTokens": 4000,
    "strategy": "auto",
    "includeSummary": true,
    "includeLongTerm": true,
    "shortTermWindow": 10
  }'
```

### 生成对话

```bash
curl -X POST http://localhost:8080/api/v1/chat/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-uuid",
    "userMessage": "请解释量子计算",
    "contextConfig": {
      "maxTokens": 4000,
      "strategy": "auto"
    },
    "generateConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 1000
    }
  }'
```

### 搜索记忆

```bash
curl -X POST http://localhost:8080/api/v1/memory/search \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-uuid",
    "query": "关于数据库的讨论",
    "topK": 5,
    "minSimilarity": 0.7
  }'
```

## 性能指标

### 响应时间

- 上下文构建: < 200ms
- 对话生成: < 2s
- 向量检索: < 50ms
- 摘要生成: < 3s

### 吞吐量

- 支持 1000+ QPS
- 支持 10000+ 并发会话
- 支持百万级记忆存储

### 资源使用

- CPU: 2-4 核（生产环境）
- 内存: 4-8 GB（生产环境）
- 存储: 根据数据量动态扩展

## 监控和告警

### 关键指标

- 服务可用性: > 99.9%
- API 响应时间: P95 < 1s
- 错误率: < 0.1%
- Token 使用量: 实时监控
- 缓存命中率: > 80%

### 监控工具

- Prometheus: 指标收集
- Grafana: 可视化仪表板
- Jaeger: 分布式追踪
- ELK Stack: 日志分析

## 安全性

### 认证和授权

- JWT Token 认证
- 基于角色的访问控制（RBAC）
- 多租户严格隔离
- API 密钥管理

### 数据安全

- 数据库连接加密（SSL/TLS）
- 敏感信息加密存储
- 审计日志记录
- 定期安全扫描

## 最佳实践

### 1. 上下文管理

- 使用查询分类自动选择策略
- 定期生成摘要以节省 Token
- 合理设置短期记忆窗口大小
- 监控上下文质量评分

### 2. 记忆管理

- 为重要信息创建长期记忆
- 定期清理过期和低质量记忆
- 使用跨会话检索共享知识
- 优化向量索引参数

### 3. Token 优化

- 监控 Token 使用预算
- 使用智能优化策略
- 启用缓存减少重复计算
- 批量处理提高效率

### 4. 性能优化

- 使用连接池管理数据库连接
- 启用多级缓存
- 合理配置向量索引
- 使用批量操作

## 故障排查

### 常见问题

1. **服务启动失败**
   - 检查数据库连接
   - 验证环境变量配置
   - 查看启动日志

2. **响应时间慢**
   - 检查数据库查询性能
   - 查看缓存命中率
   - 分析慢查询日志

3. **Token 超限**
   - 生成摘要优化上下文
   - 减少短期记忆窗口
   - 使用更激进的优化策略

## 贡献指南

欢迎贡献代码、报告问题或提出建议！

### 开发流程

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 代码规范

- 遵循 Go 代码规范
- 编写单元测试（覆盖率 > 80%）
- 添加必要的注释
- 更新相关文档

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 联系方式

- 项目主页: <https://github.com/your-org/genkit-service>
- 问题反馈: <https://github.com/your-org/genkit-service/issues>
- 技术支持: <support@example.com>
- 文档反馈: <docs@example.com>

## 致谢

- [Google Genkit](https://github.com/firebase/genkit) - AI 工作流框架
- [pgvector](https://github.com/pgvector/pgvector) - PostgreSQL 向量扩展
- [GORM](https://gorm.io/) - Go ORM 库

---

**版本**: v1.0.0  
**最后更新**: 2024-01-01  
**维护者**: 开发团队
