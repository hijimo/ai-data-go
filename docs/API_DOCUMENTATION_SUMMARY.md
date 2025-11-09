# API 文档更新摘要

## 概述

本次更新为 Genkit AI Service 添加了三个新的功能模块的 API 文档：

1. **上下文管理 (Context Management)**
2. **记忆管理 (Memory Management)**
3. **摘要管理 (Summary Management)**

## 更新内容

### 1. 上下文管理 API

#### 数据模型定义

- `BuildContextRequest` - 构建上下文请求
- `BuildContextResponse` - 构建上下文响应
- `SummaryContext` - 摘要上下文
- `MemoryContext` - 记忆上下文
- `MessageContext` - 消息上下文
- `GetContextConfigResponse` - 获取配置响应
- `UpdateContextConfigRequest` - 更新配置请求
- `BuildContextDataResponse` - 构建上下文数据响应
- `ContextConfigDataResponse` - 配置数据响应

#### API 端点

- `POST /contexts/build` - 构建智能上下文
  - 整合摘要、长期记忆和短期消息
  - 支持 auto、short、full 三种策略
  - 返回上下文质量评分和构建时间

- `GET /contexts/{sessionId}` - 获取上下文配置
  - 返回会话的上下文管理配置
  - 包含策略、token限制、窗口大小等参数

- `PATCH /contexts/{sessionId}` - 更新上下文配置
  - 支持部分更新
  - 更新后立即生效

### 2. 记忆管理 API

#### 数据模型定义

- `SearchMemoriesRequest` - 检索记忆请求
- `SearchMemoriesResponse` - 检索记忆响应
- `MemorySearchResult` - 记忆检索结果
- `StoreMemoryRequest` - 存储记忆请求
- `StoreMemoryResponse` - 存储记忆响应
- `CleanupMemoriesRequest` - 清理记忆请求
- `CleanupMemoriesResponse` - 清理记忆响应
- `CleanupDetail` - 清理详情
- `SearchMemoriesDataResponse` - 检索记忆数据响应
- `StoreMemoryDataResponse` - 存储记忆数据响应
- `CleanupMemoriesDataResponse` - 清理记忆数据响应

#### API 端点

- `POST /memories/search` - 检索长期记忆
  - 使用向量相似度检索
  - 支持按类型、时间范围和相似度过滤
  - 返回综合评分排序的结果

- `POST /memories` - 存储长期记忆
  - 自动生成向量嵌入
  - 支持设置重要性和过期时间
  - 支持元数据存储

- `POST /memories/cleanup` - 清理长期记忆
  - 支持 expired、low_quality、unused、all 四种策略
  - 支持预览模式和软/硬删除
  - 返回清理详情和释放空间

#### 记忆类型

- `fact` - 事实性信息
- `preference` - 用户偏好
- `context` - 上下文信息
- `event` - 事件记录
- `summary` - 摘要信息

### 3. 摘要管理 API

#### 数据模型定义

- `GenerateSummaryRequest` - 生成摘要请求
- `GenerateSummaryResponse` - 生成摘要响应
- `GetSummaryResponse` - 获取摘要响应
- `ListSummariesResponse` - 摘要列表响应
- `SummaryItem` - 摘要列表项
- `CheckTriggerResponse` - 检查触发条件响应
- `GenerateSummaryDataResponse` - 生成摘要数据响应
- `GetSummaryDataResponse` - 获取摘要数据响应
- `ListSummariesDataResponse` - 摘要列表数据响应
- `CheckTriggerDataResponse` - 检查触发数据响应

#### API 端点

- `POST /summaries` - 生成对话摘要
  - 支持增量摘要和完整摘要
  - 自动提取关键要点
  - 返回质量评分

- `GET /summaries/{summaryId}` - 获取摘要详情
  - 返回完整摘要信息
  - 包含关键要点和质量评分
  - 显示消息覆盖范围

- `GET /sessions/{sessionId}/summaries` - 获取会话摘要列表
  - 按创建时间倒序排列
  - 支持限制返回数量

- `GET /sessions/{sessionId}/summaries/check-trigger` - 检查摘要触发条件
  - 判断是否需要生成摘要
  - 返回触发原因和紧急程度
  - 提供预计节省的token数量

#### 摘要类型

- `incremental` - 增量摘要（基于前一个摘要和新消息）
- `full` - 完整摘要（对所有指定消息生成全新摘要）

## 错误码说明

所有 API 端点都遵循统一的错误响应格式，包含以下状态码：

- `200` - 成功
- `201` - 创建成功
- `400` - 请求参数错误
- `401` - 未认证
- `403` - 权限不足
- `404` - 资源不存在
- `422` - 参数验证失败
- `500` - 服务器内部错误
- `503` - 服务不可用（AI服务）

## 请求/响应示例

### 构建上下文示例

**请求**:

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "userQuery": "请帮我总结一下之前讨论的内容",
  "maxTokens": 4000,
  "strategy": "auto",
  "includeSummary": true,
  "includeLongTerm": true,
  "shortTermWindow": 10
}
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "summary": {
      "content": "用户询问了关于AI的问题，我们讨论了机器学习的基本概念",
      "tokenCount": 50,
      "createdAt": "2024-01-01T12:00:00Z",
      "coverage": "消息1-20"
    },
    "longTermMemories": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "content": "用户偏好使用Python进行开发",
        "tokenCount": 15,
        "importance": 0.8,
        "similarity": 0.75,
        "createdAt": "2024-01-01T12:00:00Z"
      }
    ],
    "shortTermMessages": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440002",
        "role": "user",
        "content": "你好",
        "tokenCount": 5,
        "createdAt": "2024-01-01T12:00:00Z"
      }
    ],
    "totalTokens": 3500,
    "strategy": "auto",
    "qualityScore": 0.85,
    "buildTime": 150
  },
  "traceId": "trace-1729756800-a1b2c3d4"
}
```

### 检索记忆示例

**请求**:

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "query": "用户的编程偏好",
  "topK": 5,
  "minSimilarity": 0.7,
  "memoryTypes": ["fact", "preference"]
}
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "results": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "sessionId": "550e8400-e29b-41d4-a716-446655440000",
        "memoryType": "preference",
        "content": "用户偏好使用Python进行开发",
        "importance": 0.8,
        "similarity": 0.85,
        "score": 0.82,
        "accessCount": 5,
        "createdAt": "2024-01-01T12:00:00Z"
      }
    ],
    "totalCount": 5,
    "query": "用户的编程偏好",
    "searchTime": 50
  },
  "traceId": "trace-1729756800-a1b2c3d4"
}
```

### 生成摘要示例

**请求**:

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "summaryType": "incremental",
  "targetLength": 500
}
```

**响应**:

```json
{
  "code": 201,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "content": "用户询问了关于AI的问题，我们讨论了机器学习的基本概念和应用场景",
    "summaryType": "incremental",
    "messageCount": 10,
    "tokenCount": 150,
    "qualityScore": 0.85,
    "keyPoints": [
      "讨论了机器学习的定义",
      "介绍了监督学习和无监督学习",
      "分析了实际应用场景"
    ],
    "createdAt": "2024-01-01T12:00:00Z"
  },
  "traceId": "trace-1729756800-a1b2c3d4"
}
```

## 认证要求

所有 API 端点都需要 Bearer Token 认证：

```
Authorization: Bearer {access_token}
```

## 标签分类

- `Context Management` - 上下文管理接口（智能上下文构建和配置管理）
- `Memory Management` - 记忆管理接口（长期记忆检索、存储和清理）
- `Summary Management` - 摘要管理接口（对话摘要生成和查询）

## 文件位置

- Swagger 文档: `docs/swagger.yaml`
- API 文档摘要: `docs/API_DOCUMENTATION_SUMMARY.md`

## 后续工作

1. 使用 Swagger UI 或 Swagger Editor 验证文档格式
2. 生成客户端 SDK（可选）
3. 更新 API 使用指南和最佳实践文档
4. 添加更多的请求/响应示例
5. 补充错误处理和故障排查指南
