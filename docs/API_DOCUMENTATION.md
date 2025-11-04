# Genkit 会话管理模块 API 文档

## 概述

本文档描述了 Genkit 会话管理模块提供的所有 REST API 接口。所有接口都遵循统一的响应格式，并实施了严格的多租户访问控制。

## 基础信息

- **基础 URL**: `http://localhost:8080/api/v1`
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`
- **字符编码**: UTF-8

## 通用响应格式

### 成功响应（普通数据）

```json
{
  "code": 10000,
  "message": "操作成功",
  "data": {
    // 实际数据
  }
}
```

### 成功响应（分页数据）

```json
{
  "code": 10000,
  "message": "操作成功",
  "data": {
    "data": [],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 100,
    "totalPage": 10
  }
}
```

### 错误响应

```json
{
  "code": 10101,
  "message": "请求参数错误",
  "details": "sessionId 不能为空"
}
```

## 错误码说明

| 错误码 | 说明 | HTTP 状态码 |
|-------|------|------------|
| 10000 | 成功 | 200 |
| 10101 | 请求参数错误 | 400 |
| 10102 | 未认证 | 401 |
| 10103 | 权限不足 | 403 |
| 10104 | 资源不存在 | 404 |
| 10201 | 内部服务错误 | 500 |
| 30104 | 会话不存在 | 404 |
| 30301 | 会话已过期 | 400 |
| 40201 | 上下文构建失败 | 500 |
| 40302 | Token 超限 | 400 |
| 50104 | 记忆不存在 | 404 |
| 50201 | 向量生成失败 | 500 |
| 60201 | AI 服务超时 | 504 |
| 60202 | AI 服务错误 | 500 |
| 60301 | 配额超限 | 429 |

## 认证

所有 API 请求都需要在 HTTP Header 中携带 JWT Token：

```http
Authorization: Bearer <your_jwt_token>
```

### 获取 Token

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your_password"
}
```

**响应示例**：

```json
{
  "code": 10000,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresAt": "2024-12-31T23:59:59Z",
    "user": {
      "id": "user-uuid",
      "email": "user@example.com",
      "tenantId": "tenant-uuid",
      "roles": ["tenant_admin"]
    }
  }
}
```

## 上下文管理 API

### 1. 构建会话上下文

构建包含短期记忆、长期记忆和摘要的完整会话上下文。

**端点**: `POST /api/v1/context/build`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "userQuery": "用户的当前查询",
  "maxTokens": 4000,
  "strategy": "auto",
  "includeSummary": true,
  "includeLongTerm": true,
  "shortTermWindow": 10
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| sessionId | string | 是 | 会话 ID |
| userQuery | string | 否 | 用户查询（用于向量检索） |
| maxTokens | integer | 否 | 最大 Token 数，默认 4000 |
| strategy | string | 否 | 策略：auto/quality/speed，默认 auto |
| includeSummary | boolean | 否 | 是否包含摘要，默认 true |
| includeLongTerm | boolean | 否 | 是否包含长期记忆，默认 true |
| shortTermWindow | integer | 否 | 短期记忆窗口大小，默认 10 |

**响应示例**:

```json
{
  "code": 10000,
  "message": "上下文构建成功",
  "data": {
    "sessionId": "session-uuid",
    "summary": {
      "id": "summary-uuid",
      "content": "会话摘要内容...",
      "keyTopics": ["主题1", "主题2"],
      "messageCount": 25,
      "qualityScore": 0.85
    },
    "longTermMemories": [
      {
        "id": "memory-uuid",
        "content": "重要的历史信息...",
        "importance": 0.9,
        "similarity": 0.85,
        "createdAt": "2024-01-01T10:00:00Z"
      }
    ],
    "shortTermMessages": [
      {
        "id": "message-uuid",
        "role": "user",
        "content": "用户消息",
        "createdAt": "2024-01-01T10:00:00Z"
      },
      {
        "id": "message-uuid-2",
        "role": "assistant",
        "content": "助手回复",
        "createdAt": "2024-01-01T10:00:05Z"
      }
    ],
    "totalTokens": 3500,
    "strategy": "auto",
    "qualityScore": 0.88,
    "buildTime": 150
  }
}
```

### 2. 查询分类

分析用户查询并推荐最佳上下文策略。

**端点**: `POST /api/v1/context/classify`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "userQuery": "用户的查询内容"
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "查询分类成功",
  "data": {
    "queryType": "complex",
    "needsHistory": true,
    "needsMemory": true,
    "complexity": 0.8,
    "recommendedStrategy": "quality",
    "confidence": 0.9,
    "reasoning": "查询涉及历史信息，需要完整上下文"
  }
}
```

### 3. 上下文优化

优化现有上下文以减少 Token 使用。

**端点**: `POST /api/v1/context/optimize`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "currentContext": {
    "summary": {...},
    "longTermMemories": [...],
    "shortTermMessages": [...]
  },
  "targetTokens": 3000,
  "strategy": "balanced"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| sessionId | string | 是 | 会话 ID |
| currentContext | object | 是 | 当前上下文 |
| targetTokens | integer | 是 | 目标 Token 数 |
| strategy | string | 否 | 优化策略：aggressive/balanced/conservative |

**响应示例**:

```json
{
  "code": 10000,
  "message": "上下文优化成功",
  "data": {
    "optimizedContext": {
      "summary": {...},
      "longTermMemories": [...],
      "shortTermMessages": [...]
    },
    "originalTokens": 4500,
    "optimizedTokens": 2800,
    "tokensSaved": 1700,
    "qualityLoss": 0.05,
    "strategy": "balanced"
  }
}
```

## 对话生成 API

### 4. 生成对话回复

生成 AI 对话回复（非流式）。

**端点**: `POST /api/v1/chat/generate`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "userMessage": "用户的消息内容",
  "contextConfig": {
    "maxTokens": 4000,
    "strategy": "auto",
    "includeSummary": true,
    "includeLongTerm": true
  },
  "generateConfig": {
    "temperature": 0.7,
    "maxOutputTokens": 1000,
    "topP": 0.9,
    "topK": 40
  }
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "生成成功",
  "data": {
    "messageId": "message-uuid",
    "content": "AI 生成的回复内容...",
    "tokenStats": {
      "inputTokens": 3500,
      "outputTokens": 250,
      "totalTokens": 3750
    },
    "contextUsed": {
      "summaryIncluded": true,
      "longTermCount": 5,
      "shortTermCount": 10
    },
    "generatedAt": "2024-01-01T10:00:00Z"
  }
}
```

### 5. 流式对话生成

生成 AI 对话回复（流式）。

**端点**: `POST /api/v1/chat/stream`

**权限**: 租户管理员、平台管理员

**请求参数**: 与 `/chat/generate` 相同

**响应**: Server-Sent Events (SSE) 流

**事件类型**:

1. **start** - 流开始

```json
{
  "type": "start",
  "messageId": "message-uuid",
  "timestamp": "2024-01-01T10:00:00Z"
}
```

2. **content** - 内容块

```json
{
  "type": "content",
  "content": "生成的文本片段",
  "index": 0
}
```

3. **token_stats** - Token 统计

```json
{
  "type": "token_stats",
  "inputTokens": 3500,
  "outputTokens": 250,
  "totalTokens": 3750
}
```

4. **end** - 流结束

```json
{
  "type": "end",
  "messageId": "message-uuid",
  "totalChunks": 15
}
```

5. **error** - 错误

```json
{
  "type": "error",
  "code": 60202,
  "message": "AI 服务错误"
}
```

### 6. 多轮对话管理

管理多轮对话并评估会话健康度。

**端点**: `POST /api/v1/chat/multi-turn`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "checkHealth": true,
  "autoOptimize": true
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "会话状态检查成功",
  "data": {
    "sessionId": "session-uuid",
    "turnCount": 15,
    "healthScore": 0.85,
    "issues": [],
    "suggestions": [
      "建议在 5 轮对话后生成摘要"
    ],
    "contextStatus": {
      "tokenUsage": 3500,
      "tokenLimit": 4000,
      "utilizationRate": 0.875
    }
  }
}
```

### 7. 对话重试

重试失败的对话生成。

**端点**: `POST /api/v1/chat/retry`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "messageId": "failed-message-uuid",
  "strategy": "exponential",
  "maxRetries": 3
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| sessionId | string | 是 | 会话 ID |
| messageId | string | 是 | 失败的消息 ID |
| strategy | string | 否 | 重试策略：simple/exponential/adaptive |
| maxRetries | integer | 否 | 最大重试次数，默认 3 |

**响应示例**:

```json
{
  "code": 10000,
  "message": "重试成功",
  "data": {
    "messageId": "new-message-uuid",
    "content": "重试后的回复内容",
    "retryCount": 2,
    "strategy": "exponential",
    "success": true
  }
}
```

## 记忆管理 API

### 8. 搜索记忆

基于向量相似度搜索长期记忆。

**端点**: `POST /api/v1/memory/search`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "query": "搜索查询",
  "topK": 5,
  "minSimilarity": 0.7,
  "crossSession": false
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| sessionId | string | 是 | 会话 ID |
| query | string | 是 | 搜索查询 |
| topK | integer | 否 | 返回结果数量，默认 5 |
| minSimilarity | float | 否 | 最小相似度，默认 0.7 |
| crossSession | boolean | 否 | 是否跨会话搜索，默认 false |

**响应示例**:

```json
{
  "code": 10000,
  "message": "搜索成功",
  "data": {
    "memories": [
      {
        "id": "memory-uuid",
        "sessionId": "session-uuid",
        "content": "记忆内容...",
        "importance": 0.9,
        "similarity": 0.85,
        "accessCount": 5,
        "createdAt": "2024-01-01T10:00:00Z",
        "lastAccessAt": "2024-01-02T15:30:00Z"
      }
    ],
    "totalCount": 1,
    "searchTime": 50
  }
}
```

### 9. 存储记忆

创建新的长期记忆。

**端点**: `POST /api/v1/memory/store`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "content": "要存储的记忆内容",
  "metadata": {
    "source": "user_highlight",
    "tags": ["重要", "技术"]
  },
  "expiresAt": "2025-01-01T00:00:00Z"
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "记忆存储成功",
  "data": {
    "id": "memory-uuid",
    "sessionId": "session-uuid",
    "content": "要存储的记忆内容",
    "importance": 0.85,
    "metadata": {
      "source": "user_highlight",
      "tags": ["重要", "技术"],
      "keyPhrases": ["技术方案", "架构设计"]
    },
    "createdAt": "2024-01-01T10:00:00Z"
  }
}
```

### 10. 清理记忆

批量清理过期或低质量的记忆。

**端点**: `POST /api/v1/memory/cleanup`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "tenantId": "tenant-uuid",
  "strategy": "expired",
  "mode": "soft",
  "batchSize": 100,
  "dryRun": false
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| tenantId | string | 是 | 租户 ID（平台管理员可指定） |
| strategy | string | 是 | 清理策略：expired/low_quality/unused/all |
| mode | string | 否 | 删除模式：soft/hard，默认 soft |
| batchSize | integer | 否 | 批量大小，默认 100 |
| dryRun | boolean | 否 | 预览模式，默认 false |

**响应示例**:

```json
{
  "code": 10000,
  "message": "清理成功",
  "data": {
    "deletedCount": 45,
    "strategy": "expired",
    "mode": "soft",
    "estimatedSpaceSaved": "2.5MB"
  }
}
```

## 摘要管理 API

### 11. 生成摘要

为会话生成摘要。

**端点**: `POST /api/v1/summary/generate`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "messageRange": {
    "start": 0,
    "end": 50
  },
  "style": "concise"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| sessionId | string | 是 | 会话 ID |
| messageRange | object | 否 | 消息范围 |
| style | string | 否 | 摘要风格：concise/detailed/bullet_points |

**响应示例**:

```json
{
  "code": 10000,
  "message": "摘要生成成功",
  "data": {
    "id": "summary-uuid",
    "sessionId": "session-uuid",
    "content": "会话摘要内容...",
    "keyTopics": ["主题1", "主题2", "主题3"],
    "messageCount": 50,
    "qualityScore": 0.88,
    "createdAt": "2024-01-01T10:00:00Z"
  }
}
```

### 12. 检查摘要触发条件

检查是否应该生成摘要。

**端点**: `POST /api/v1/summary/trigger`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid"
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "检查完成",
  "data": {
    "shouldGenerate": true,
    "score": 0.85,
    "reasons": [
      "消息数量达到阈值（50条）",
      "Token 使用率较高（85%）"
    ],
    "estimatedBenefit": {
      "tokenSavings": 1500,
      "qualityImprovement": 0.1
    }
  }
}
```

### 13. 评估摘要质量

评估现有摘要的质量。

**端点**: `POST /api/v1/summary/quality`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "summaryId": "summary-uuid"
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "质量评估完成",
  "data": {
    "summaryId": "summary-uuid",
    "overallScore": 0.85,
    "dimensions": {
      "completeness": 0.9,
      "accuracy": 0.85,
      "conciseness": 0.8,
      "relevance": 0.85
    },
    "issues": [
      "部分细节信息缺失"
    ],
    "suggestions": [
      "建议补充技术细节"
    ]
  }
}
```

## Token 管理 API

### 14. Token 预算管理

管理和监控 Token 使用预算。

**端点**: `POST /api/v1/token/budget`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "budget": 10000
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "预算检查完成",
  "data": {
    "sessionId": "session-uuid",
    "budget": 10000,
    "used": 7500,
    "remaining": 2500,
    "utilizationRate": 0.75,
    "status": "warning",
    "suggestions": [
      "建议生成摘要以节省 Token"
    ],
    "prediction": {
      "estimatedTurnsRemaining": 5,
      "estimatedTimeRemaining": "30分钟"
    }
  }
}
```

### 15. Token 优化

优化 Token 使用。

**端点**: `POST /api/v1/token/optimize`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "content": "要优化的内容...",
  "targetReduction": 0.3,
  "strategy": "smart"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| sessionId | string | 是 | 会话 ID |
| content | string | 是 | 要优化的内容 |
| targetReduction | float | 否 | 目标减少比例，默认 0.3 |
| strategy | string | 否 | 优化策略：compress/summarize/truncate/smart |

**响应示例**:

```json
{
  "code": 10000,
  "message": "优化成功",
  "data": {
    "originalTokens": 1000,
    "optimizedTokens": 700,
    "tokensSaved": 300,
    "reductionRate": 0.3,
    "optimizedContent": "优化后的内容...",
    "qualityScore": 0.9,
    "strategy": "smart"
  }
}
```

### 16. Token 使用分析

分析 Token 使用情况。

**端点**: `POST /api/v1/token/analysis`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "tenantId": "tenant-uuid",
  "timeRange": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  },
  "dimension": "usage"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| tenantId | string | 否 | 租户 ID（租户管理员自动使用当前租户） |
| timeRange | object | 是 | 时间范围 |
| dimension | string | 否 | 分析维度：usage/trend/cost/efficiency |

**响应示例**:

```json
{
  "code": 10000,
  "message": "分析完成",
  "data": {
    "dimension": "usage",
    "totalTokens": 1000000,
    "breakdown": {
      "input": 600000,
      "output": 400000
    },
    "topSessions": [
      {
        "sessionId": "session-uuid",
        "tokens": 50000,
        "percentage": 5.0
      }
    ],
    "trends": {
      "dailyAverage": 32258,
      "peakDay": "2024-01-15",
      "peakDayTokens": 45000
    },
    "suggestions": [
      "建议启用自动摘要以减少 Token 使用"
    ]
  }
}
```

## 会话健康检查 API

### 17. 会话健康检查

全面检查会话健康状态。

**端点**: `POST /api/v1/session/health-check`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "sessionId": "session-uuid",
  "checks": ["context", "token", "memory", "summary", "performance"],
  "autoFix": true
}
```

**响应示例**:

```json
{
  "code": 10000,
  "message": "健康检查完成",
  "data": {
    "sessionId": "session-uuid",
    "overallHealth": 0.85,
    "status": "healthy",
    "checks": {
      "context": {
        "status": "healthy",
        "score": 0.9,
        "issues": []
      },
      "token": {
        "status": "warning",
        "score": 0.75,
        "issues": ["Token 使用率较高"]
      },
      "memory": {
        "status": "healthy",
        "score": 0.9,
        "issues": []
      },
      "summary": {
        "status": "healthy",
        "score": 0.85,
        "issues": []
      },
      "performance": {
        "status": "healthy",
        "score": 0.9,
        "issues": []
      }
    },
    "autoFixApplied": true,
    "fixedIssues": [
      "已生成摘要以优化 Token 使用"
    ],
    "recommendations": [
      "建议定期清理低质量记忆"
    ]
  }
}
```

## 批量操作 API

### 18. 批量对话处理

批量处理多个对话请求。

**端点**: `POST /api/v1/batch/conversations`

**权限**: 租户管理员、平台管理员

**请求参数**:

```json
{
  "conversations": [
    {
      "sessionId": "session-uuid-1",
      "userMessage": "消息1"
    },
    {
      "sessionId": "session-uuid-2",
      "userMessage": "消息2"
    }
  ],
  "concurrency": 3,
  "failureStrategy": "continue"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| conversations | array | 是 | 对话请求列表 |
| concurrency | integer | 否 | 并发数，默认 3 |
| failureStrategy | string | 否 | 失败策略：continue/abort，默认 continue |

**响应示例**:

```json
{
  "code": 10000,
  "message": "批量处理完成",
  "data": {
    "total": 2,
    "successful": 2,
    "failed": 0,
    "results": [
      {
        "sessionId": "session-uuid-1",
        "success": true,
        "messageId": "message-uuid-1",
        "content": "回复1"
      },
      {
        "sessionId": "session-uuid-2",
        "success": true,
        "messageId": "message-uuid-2",
        "content": "回复2"
      }
    ],
    "processingTime": 1500
  }
}
```

## 权限说明

### 角色定义

| 角色 | 说明 | 权限范围 |
|-----|------|---------|
| system_admin | 平台管理员 | 可访问所有租户的所有资源 |
| tenant_admin | 租户管理员 | 只能访问自己租户的资源 |
| user | 普通用户 | 只能访问自己的会话 |

### 多租户隔离

所有 API 都实施了严格的多租户隔离：

1. **租户管理员**：
   - 只能访问自己租户下的资源
   - 尝试访问其他租户资源将返回 403 错误
   - 系统自动过滤查询结果，只返回当前租户的数据

2. **平台管理员**：
   - 可以访问所有租户的资源
   - 可以在请求中指定 `tenantId` 参数
   - 用于系统管理和跨租户操作

### 审计日志

所有权限验证失败的尝试都会被记录到审计日志，包括：

- 用户 ID 和租户 ID
- 目标资源类型和 ID
- 失败原因
- 客户端 IP 和 User-Agent
- 时间戳

## 速率限制

为保护系统资源，所有 API 都实施了速率限制：

| 限制类型 | 限制值 | 说明 |
|---------|-------|------|
| 每用户每分钟 | 60 次 | 单个用户的请求频率 |
| 每租户每分钟 | 300 次 | 单个租户的总请求频率 |
| 每 IP 每分钟 | 100 次 | 单个 IP 的请求频率 |
| Token 配额 | 按租户配置 | 每月 Token 使用配额 |

超过限制将返回 429 错误：

```json
{
  "code": 60301,
  "message": "请求频率超限",
  "details": "请在 30 秒后重试",
  "retryAfter": 30
}
```

## 最佳实践

### 1. 错误处理

始终检查响应的 `code` 字段：

```javascript
const response = await fetch('/api/v1/chat/generate', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(requestData)
});

const result = await response.json();

if (result.code !== 10000) {
  // 处理错误
  console.error(`错误 ${result.code}: ${result.message}`);
  if (result.details) {
    console.error(`详情: ${result.details}`);
  }
  return;
}

// 处理成功响应
const data = result.data;
```

### 2. Token 管理

定期检查 Token 使用情况：

```javascript
// 在生成对话前检查预算
const budgetCheck = await checkTokenBudget(sessionId);
if (budgetCheck.status === 'critical') {
  // 先生成摘要
  await generateSummary(sessionId);
}

// 然后生成对话
const response = await generateChat(sessionId, userMessage);
```

### 3. 缓存利用

利用缓存提高性能：

```javascript
// 构建上下文时，相同的查询会被缓存 5 分钟
const context1 = await buildContext(sessionId, query);
// 5 分钟内的相同请求会直接返回缓存结果
const context2 = await buildContext(sessionId, query);
```

### 4. 流式响应处理

正确处理 SSE 流：

```javascript
const eventSource = new EventSource('/api/v1/chat/stream');

eventSource.addEventListener('start', (e) => {
  const data = JSON.parse(e.data);
  console.log('开始生成:', data.messageId);
});

eventSource.addEventListener('content', (e) => {
  const data = JSON.parse(e.data);
  // 逐步显示内容
  appendContent(data.content);
});

eventSource.addEventListener('end', (e) => {
  const data = JSON.parse(e.data);
  console.log('生成完成:', data.totalChunks);
  eventSource.close();
});

eventSource.addEventListener('error', (e) => {
  const data = JSON.parse(e.data);
  console.error('生成错误:', data.message);
  eventSource.close();
});
```

### 5. 批量操作

使用批量 API 提高效率：

```javascript
// 不推荐：逐个处理
for (const session of sessions) {
  await generateChat(session.id, message);
}

// 推荐：批量处理
const batchResult = await batchConversations(
  sessions.map(s => ({
    sessionId: s.id,
    userMessage: message
  })),
  { concurrency: 3 }
);
```

## 附录

### A. 完整的错误码列表

请参考本文档开头的"错误码说明"部分。

### B. 数据模型

#### ConversationMessage

```typescript
interface ConversationMessage {
  id: string;
  sessionId: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  tokenCount: number;
  createdAt: string;
}
```

#### ConversationMemory

```typescript
interface ConversationMemory {
  id: string;
  sessionId: string;
  content: string;
  embedding: number[];
  importance: number;
  similarity?: number;
  accessCount: number;
  metadata: Record<string, any>;
  createdAt: string;
  lastAccessAt: string;
  expiresAt?: string;
}
```

#### ConversationSummary

```typescript
interface ConversationSummary {
  id: string;
  sessionId: string;
  content: string;
  keyTopics: string[];
  messageCount: number;
  qualityScore: number;
  createdAt: string;
}
```

### C. 环境配置

#### 开发环境

```bash
API_BASE_URL=http://localhost:8080/api/v1
JWT_SECRET=your-dev-secret
DB_HOST=localhost
DB_PORT=5432
REDIS_HOST=localhost
REDIS_PORT=6379
GENAI_API_KEY=your-api-key
```

#### 生产环境

```bash
API_BASE_URL=https://api.example.com/api/v1
JWT_SECRET=${JWT_SECRET}
DB_HOST=${DB_HOST}
DB_PORT=5432
REDIS_HOST=${REDIS_HOST}
REDIS_PORT=6379
GENAI_API_KEY=${GENAI_API_KEY}
```

### D. 联系支持

如有问题，请联系：

- 技术支持邮箱: <support@example.com>
- 文档反馈: <docs@example.com>
- GitHub Issues: <https://github.com/your-org/genkit-service/issues>

---

**文档版本**: v1.0.0  
**最后更新**: 2024-01-01  
**维护者**: 开发团队
