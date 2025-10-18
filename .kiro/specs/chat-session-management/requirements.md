# 会话管理系统需求文档

## 简介

本文档定义了 AI 聊天系统的会话管理模块需求。该模块旨在为用户提供完整的会话生命周期管理能力，包括会话创建、消息历史记录、会话列表管理等功能，同时支持多用户隔离、多模型切换和长会话优化。

## 术语表

- **ChatSessionSystem（会话系统）**: 负责管理用户会话的核心系统模块
- **ChatSession（会话）**: 用户与 AI 进行多轮对话的上下文容器
- **ChatMessage（消息）**: 会话中的单条对话消息，包含用户消息和 AI 回复
- **MessageID（消息ID）**: 唯一标识一条消息的 UUID
- **SessionID（会话ID）**: 唯一标识一个会话的 UUID
- **UserID（用户ID）**: 唯一标识一个用户的 UUID
- **ModelName（模型名称）**: AI 模型的标识符（如 GPT-4、Claude-3 等）
- **SystemPrompt（系统提示词）**: 会话级别的 AI 行为指令
- **SessionSummary（会话摘要）**: 长会话的压缩摘要，用于优化上下文长度
- **SoftDelete（软删除）**: 标记删除而非物理删除数据的操作

## 需求

### 需求 1：会话参数调整

**用户故事：** 作为系统开发者，我希望将现有接口中的 sessionId 参数调整为 messageId，以便更准确地标识消息而非会话

#### 验收标准

1. 当系统接收 /chat 接口请求时，ChatSessionSystem 应该接受 messageId 参数而非 sessionId 参数
2. 当系统接收 /chat/abort 接口请求时，ChatSessionSystem 应该接受 messageId 参数而非 sessionId 参数
3. 当系统处理包含 messageId 的请求时，ChatSessionSystem 应该能够通过 messageId 定位到对应的会话
4. 当系统返回响应时，ChatSessionSystem 应该在响应中包含 messageId 而非 sessionId

### 需求 2：会话创建

**用户故事：** 作为用户，我希望能够创建新的聊天会话，以便开始与 AI 的对话

#### 验收标准

1. 当用户发送创建会话请求时，ChatSessionSystem 应该生成唯一的 SessionID
2. 当用户指定模型名称时，ChatSessionSystem 应该将 ModelName 保存到会话记录中
3. 当用户提供会话标题时，ChatSessionSystem 应该将标题保存到会话记录中
4. 当用户提供系统提示词时，ChatSessionSystem 应该将 SystemPrompt 保存到会话记录中
5. 当用户提供模型参数（temperature、top_p）时，ChatSessionSystem 应该将参数保存到会话记录中
6. 当会话创建成功时，ChatSessionSystem 应该返回包含 SessionID 和创建时间的响应数据
7. 当会话创建时，ChatSessionSystem 应该记录 UserID 以实现用户隔离

### 需求 3：会话列表查询

**用户故事：** 作为用户，我希望能够查看我的所有会话列表，以便快速找到历史对话

#### 验收标准

1. 当用户请求会话列表时，ChatSessionSystem 应该仅返回该用户的会话记录
2. 当用户指定分页参数时，ChatSessionSystem 应该返回符合分页要求的会话列表
3. 当用户请求会话列表时，ChatSessionSystem 应该按更新时间倒序排列会话
4. 当会话列表包含已归档会话时，ChatSessionSystem 应该在响应中标识归档状态
5. 当会话列表包含置顶会话时，ChatSessionSystem 应该将置顶会话排在列表前面
6. 当用户请求会话列表时，ChatSessionSystem 应该排除已软删除的会话
7. 当会话列表返回时，ChatSessionSystem 应该包含每个会话的消息数量统计
8. 当会话列表返回时，ChatSessionSystem 应该包含每个会话的最后一条消息预览

### 需求 4：会话详情查询

**用户故事：** 作为用户，我希望能够查看特定会话的详细信息，以便了解会话的配置和状态

#### 验收标准

1. 当用户请求会话详情时，ChatSessionSystem 应该验证该会话属于请求用户
2. 当会话不存在时，ChatSessionSystem 应该返回 404 错误响应
3. 当会话已被软删除时，ChatSessionSystem 应该返回 404 错误响应
4. 当会话详情查询成功时，ChatSessionSystem 应该返回完整的会话元信息
5. 当会话详情返回时，ChatSessionSystem 应该包含会话的所有配置参数

### 需求 5：消息发送

**用户故事：** 作为用户，我希望能够在会话中发送消息，以便与 AI 进行对话

#### 验收标准

1. 当用户在会话中发送消息时，ChatSessionSystem 应该生成唯一的 MessageID
2. 当用户消息保存时，ChatSessionSystem 应该记录消息角色为 user
3. 当用户消息保存时，ChatSessionSystem 应该记录消息的创建时间戳
4. 当用户消息保存时，ChatSessionSystem 应该递增消息序列号
5. 当 AI 回复生成后，ChatSessionSystem 应该保存 AI 消息并记录角色为 assistant
6. 当消息保存成功时，ChatSessionSystem 应该更新会话的 last_message_id 字段
7. 当消息保存成功时，ChatSessionSystem 应该递增会话的 message_count 字段
8. 当消息保存成功时，ChatSessionSystem 应该更新会话的 updated_at 时间戳

### 需求 6：消息历史查询

**用户故事：** 作为用户，我希望能够查看会话的历史消息，以便回顾之前的对话内容

#### 验收标准

1. 当用户请求消息历史时，ChatSessionSystem 应该验证该会话属于请求用户
2. 当用户指定分页参数时，ChatSessionSystem 应该返回符合分页要求的消息列表
3. 当消息历史返回时，ChatSessionSystem 应该按消息序列号正序排列
4. 当消息历史返回时，ChatSessionSystem 应该包含每条消息的完整内容和元数据
5. 当消息包含错误信息时，ChatSessionSystem 应该在响应中包含错误详情

### 需求 7：会话更新

**用户故事：** 作为用户，我希望能够修改会话的标题和配置，以便更好地组织和管理会话

#### 验收标准

1. 当用户更新会话标题时，ChatSessionSystem 应该保存新的标题到会话记录
2. 当用户更新模型参数时，ChatSessionSystem 应该保存新的参数到会话记录
3. 当用户更新系统提示词时，ChatSessionSystem 应该保存新的提示词到会话记录
4. 当用户切换会话的置顶状态时，ChatSessionSystem 应该更新 is_pinned 字段
5. 当用户切换会话的归档状态时，ChatSessionSystem 应该更新 is_archived 字段
6. 当会话更新成功时，ChatSessionSystem 应该更新 updated_at 时间戳
7. 当用户尝试更新不属于自己的会话时，ChatSessionSystem 应该返回 403 错误响应

### 需求 8：会话删除

**用户故事：** 作为用户，我希望能够删除不需要的会话，以便保持会话列表的整洁

#### 验收标准

1. 当用户删除会话时，ChatSessionSystem 应该将 is_deleted 字段设置为 true
2. 当用户删除会话时，ChatSessionSystem 应该保留会话和消息的物理数据
3. 当会话被软删除后，ChatSessionSystem 应该在会话列表查询中排除该会话
4. 当会话被软删除后，ChatSessionSystem 应该在会话详情查询中返回 404 错误
5. 当用户尝试删除不属于自己的会话时，ChatSessionSystem 应该返回 403 错误响应

### 需求 9：多用户隔离

**用户故事：** 作为系统管理员，我希望系统能够隔离不同用户的会话数据，以便保护用户隐私

#### 验收标准

1. 当系统创建会话时，ChatSessionSystem 应该记录 UserID 到会话记录
2. 当用户查询会话列表时，ChatSessionSystem 应该仅返回该用户创建的会话
3. 当用户访问会话详情时，ChatSessionSystem 应该验证会话所有权
4. 当用户访问消息历史时，ChatSessionSystem 应该验证会话所有权
5. 当用户尝试访问其他用户的会话时，ChatSessionSystem 应该返回 403 错误响应

### 需求 10：多模型支持

**用户故事：** 作为用户，我希望能够在不同会话中使用不同的 AI 模型，以便根据需求选择合适的模型

#### 验收标准

1. 当用户创建会话时，ChatSessionSystem 应该允许指定 ModelName 参数
2. 当用户发送消息时，ChatSessionSystem 应该使用会话配置的模型处理请求
3. 当会话返回时，ChatSessionSystem 应该在响应中包含当前使用的 ModelName
4. 当用户更新会话时，ChatSessionSystem 应该允许切换 ModelName

### 需求 11：会话搜索

**用户故事：** 作为用户，我希望能够搜索会话，以便快速找到特定的对话

#### 验收标准

1. 当用户提供搜索关键词时，ChatSessionSystem 应该在会话标题中进行模糊匹配
2. 当用户提供搜索关键词时，ChatSessionSystem 应该在最后一条消息内容中进行模糊匹配
3. 当搜索结果返回时，ChatSessionSystem 应该仅包含匹配的会话记录
4. 当搜索结果为空时，ChatSessionSystem 应该返回空列表而非错误

### 需求 12：会话摘要（长会话优化）

**用户故事：** 作为系统，我希望能够为长会话生成摘要，以便优化上下文长度和性能

#### 验收标准

1. 当会话消息数量超过配置阈值时，ChatSessionSystem 应该触发摘要生成流程
2. 当生成摘要时，ChatSessionSystem 应该保存摘要内容到 chat_summaries 表
3. 当生成摘要时，ChatSessionSystem 应该记录摘要截止的 last_message_id
4. 当生成摘要时，ChatSessionSystem 应该记录摘要的 token_count
5. 当用户发送新消息时，ChatSessionSystem 应该使用摘要替代早期消息构建上下文
