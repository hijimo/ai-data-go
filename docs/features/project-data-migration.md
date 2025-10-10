# 项目数据迁移功能

## 概述

项目数据迁移功能允许用户在不同项目之间迁移数据，包括文档、Agent、对话会话、问题答案、向量索引和训练任务等。该功能支持完整的数据导出和导入，以及灵活的迁移选项配置。

## 功能特性

### 1. 数据导出

- 支持完整项目数据导出
- 包含所有关联数据（文档版本、文档块、向量记录等）
- 导出数据格式化为JSON结构
- 支持数据完整性验证

### 2. 数据导入

- 支持选择性数据导入
- 可配置覆盖策略
- 支持数据去重和冲突处理
- 事务性操作保证数据一致性

### 3. 迁移任务管理

- 异步任务处理
- 实时进度跟踪
- 任务状态监控
- 支持任务取消

### 4. 权限控制

- 基于RBAC的权限验证
- 源项目读取权限检查
- 目标项目写入权限检查
- 项目级数据隔离

## API接口

### 1. 导出项目数据

```http
GET /api/v1/projects/{project_id}/export
```

**响应示例：**

```json
{
  "project": {
    "id": "uuid",
    "name": "项目名称",
    "description": "项目描述"
  },
  "files": [...],
  "agents": [...],
  "chat_sessions": [...],
  "questions": [...],
  "vector_indexes": [...],
  "training_jobs": [...],
  "exported_at": "2025-01-27T10:00:00Z",
  "export_version": "1.0"
}
```

### 2. 导入项目数据

```http
POST /api/v1/projects/{project_id}/import
```

**请求体：**

```json
{
  "data": {
    "files": [...],
    "agents": [...],
    "chat_sessions": [...],
    "questions": [...],
    "vector_indexes": [...],
    "training_jobs": [...],
    "import_options": {
      "include_files": true,
      "include_agents": true,
      "include_chat_sessions": false,
      "include_questions": true,
      "include_vector_indexes": true,
      "include_training_jobs": false,
      "overwrite_existing": false
    }
  }
}
```

### 3. 获取项目数据统计

```http
GET /api/v1/projects/{project_id}/stats
```

**响应示例：**

```json
{
  "project_id": "uuid",
  "files_count": 150,
  "chunks_count": 3500,
  "agents_count": 5,
  "chat_sessions_count": 25,
  "questions_count": 800,
  "answers_count": 800,
  "vector_indexes_count": 3,
  "training_jobs_count": 2,
  "total_size": 1073741824
}
```

### 4. 创建迁移任务

```http
POST /api/v1/migration/tasks
```

**请求体：**

```json
{
  "source_project_id": "source-uuid",
  "target_project_id": "target-uuid",
  "import_options": {
    "include_files": true,
    "include_agents": true,
    "include_chat_sessions": false,
    "include_questions": true,
    "include_vector_indexes": true,
    "include_training_jobs": false,
    "overwrite_existing": false
  }
}
```

### 5. 获取迁移任务状态

```http
GET /api/v1/migration/tasks/{task_id}
```

**响应示例：**

```json
{
  "id": "task-uuid",
  "task_type": "data_migration",
  "status": 1,
  "progress": 100,
  "input_data": {...},
  "output_data": {
    "migrated_at": "2025-01-27T10:30:00Z",
    "source_project": "source-uuid",
    "target_project": "target-uuid",
    "migration_summary": {
      "files_count": 150,
      "agents_count": 5,
      "chat_sessions_count": 0,
      "questions_count": 800,
      "vector_indexes_count": 3,
      "training_jobs_count": 0
    }
  },
  "started_at": "2025-01-27T10:25:00Z",
  "completed_at": "2025-01-27T10:30:00Z",
  "created_at": "2025-01-27T10:25:00Z"
}
```

### 6. 取消迁移任务

```http
POST /api/v1/migration/tasks/{task_id}/cancel
```

## 数据结构

### 导入选项配置

```go
type ImportOptions struct {
    IncludeFiles         bool `json:"include_files"`         // 是否包含文件
    IncludeAgents        bool `json:"include_agents"`        // 是否包含Agent
    IncludeChatSessions  bool `json:"include_chat_sessions"` // 是否包含对话会话
    IncludeQuestions     bool `json:"include_questions"`     // 是否包含问题
    IncludeVectorIndexes bool `json:"include_vector_indexes"` // 是否包含向量索引
    IncludeTrainingJobs  bool `json:"include_training_jobs"`  // 是否包含训练任务
    OverwriteExisting    bool `json:"overwrite_existing"`     // 是否覆盖已存在的数据
}
```

### 任务状态

- `0`: 处理中 (TaskStatusProcessing)
- `1`: 已完成 (TaskStatusCompleted)
- `2`: 失败 (TaskStatusFailed)
- `3`: 已取消 (TaskStatusCancelled)

## 使用场景

### 1. 项目备份与恢复

```bash
# 导出项目数据作为备份
curl -X GET "https://api.example.com/api/v1/projects/{project_id}/export" \
  -H "Authorization: Bearer {token}" \
  > project_backup.json

# 恢复数据到新项目
curl -X POST "https://api.example.com/api/v1/projects/{new_project_id}/import" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d @project_backup.json
```

### 2. 项目合并

```bash
# 将项目A的数据迁移到项目B
curl -X POST "https://api.example.com/api/v1/migration/tasks" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "source_project_id": "project-a-uuid",
    "target_project_id": "project-b-uuid",
    "import_options": {
      "include_files": true,
      "include_agents": true,
      "include_chat_sessions": false,
      "include_questions": true,
      "include_vector_indexes": false,
      "include_training_jobs": false,
      "overwrite_existing": false
    }
  }'
```

### 3. 选择性数据迁移

```bash
# 只迁移文档和问题数据
curl -X POST "https://api.example.com/api/v1/migration/tasks" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "source_project_id": "source-uuid",
    "target_project_id": "target-uuid",
    "import_options": {
      "include_files": true,
      "include_agents": false,
      "include_chat_sessions": false,
      "include_questions": true,
      "include_vector_indexes": false,
      "include_training_jobs": false,
      "overwrite_existing": true
    }
  }'
```

## 注意事项

### 1. 权限要求

- 导出数据需要源项目的读取权限
- 导入数据需要目标项目的写入权限
- 创建迁移任务需要同时具备源项目读取和目标项目写入权限

### 2. 数据一致性

- 所有导入操作都在数据库事务中执行
- 如果导入过程中出现错误，会自动回滚所有更改
- 支持数据完整性验证和冲突检测

### 3. 性能考虑

- 大量数据的迁移会创建异步任务
- 可以通过任务状态API监控进度
- 建议在低峰期执行大规模数据迁移

### 4. 数据去重

- 文件基于SHA256哈希值去重
- Agent基于名称去重
- 问题基于内容去重
- 向量索引基于名称去重
- 训练任务基于名称去重

### 5. 外部依赖

- 向量数据的迁移不包含实际向量内容，只迁移元数据
- OSS文件路径保持不变，需要确保目标项目能访问相同的存储
- LLM模型ID需要在目标环境中存在

## 错误处理

### 常见错误码

- `INSUFFICIENT_PERMISSION`: 权限不足
- `PROJECT_NOT_FOUND`: 项目不存在
- `INVALID_REQUEST`: 请求参数无效
- `EXPORT_FAILED`: 导出失败
- `IMPORT_FAILED`: 导入失败
- `TASK_NOT_FOUND`: 任务不存在
- `INVALID_TASK_STATUS`: 任务状态无效

### 错误响应格式

```json
{
  "error": "IMPORT_FAILED",
  "message": "导入项目数据失败",
  "details": "具体错误信息"
}
```

## 监控和日志

### 1. 任务监控

- 实时进度跟踪
- 任务执行时间统计
- 成功率监控

### 2. 审计日志

- 记录所有迁移操作
- 包含用户信息和操作时间
- 支持操作回溯

### 3. 性能指标

- 数据迁移速度
- 资源使用情况
- 错误率统计

## 扩展性

### 1. 新数据类型支持

- 通过扩展导出/导入接口支持新的数据类型
- 保持向后兼容性

### 2. 自定义迁移策略

- 支持插件化的数据处理逻辑
- 可配置的数据转换规则

### 3. 批量操作

- 支持多项目批量迁移
- 支持定时迁移任务
