# 错误日志记录快速参考

## 概述

本文档提供 Genkit 多模型支持中错误日志记录的快速参考，帮助开发者了解如何记录和查看错误日志。

## 日志级别

- **ERROR**：错误级别，用于记录系统错误和异常
- **WARN**：警告级别，用于记录警告信息（如上下文取消）
- **INFO**：信息级别，用于记录正常的操作信息

## 错误日志格式

所有错误日志都遵循以下格式：

```json
{
  "timestamp": "2025-12-01T06:01:42Z",
  "level": "ERROR",
  "message": "错误描述",
  "fields": {
    "key1": "value1",
    "key2": "value2"
  }
}
```

## 常见错误场景

### 1. 初始化错误

#### 配置为空

```json
{
  "level": "ERROR",
  "message": "初始化失败：配置不能为空"
}
```

#### API 密钥为空

```json
{
  "level": "ERROR",
  "message": "初始化失败：API 密钥不能为空"
}
```

#### 模型名称为空

```json
{
  "level": "ERROR",
  "message": "初始化失败：模型名称不能为空"
}
```

### 2. 配置相关错误

#### 配置仓储未初始化

```json
{
  "level": "ERROR",
  "message": "获取 Genkit 实例失败：配置仓储未初始化",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4"
  }
}
```

#### 解析租户ID失败

```json
{
  "level": "ERROR",
  "message": "解析租户ID失败",
  "fields": {
    "tenantId": "invalid-uuid",
    "error": "invalid UUID format"
  }
}
```

#### 获取模型配置失败

```json
{
  "level": "ERROR",
  "message": "获取模型配置失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "error": "record not found"
  }
}
```

#### 解析模型配置失败

```json
{
  "level": "ERROR",
  "message": "解析模型配置失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "error": "json: cannot unmarshal..."
  }
}
```

#### 解析 QueryParams 失败

```json
{
  "level": "ERROR",
  "message": "解析 QueryParams 失败",
  "fields": {
    "queryParams": "{\"temperature\":0.7}",
    "error": "invalid character..."
  }
}
```

### 3. 配置验证错误

#### 配置验证失败

```json
{
  "level": "ERROR",
  "message": "配置验证失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "modelProvider": "azureopenai",
    "error": "missing required field: azureEndpoint"
  }
}
```

#### Azure 配置缺少必需字段

```json
{
  "level": "ERROR",
  "message": "Azure OpenAI 配置缺少必需字段",
  "fields": {
    "missingField": "azureEndpoint"
  }
}
```

### 4. 插件创建错误

#### 创建 Azure OpenAI 插件失败

```json
{
  "level": "ERROR",
  "message": "创建 Azure OpenAI 插件失败",
  "fields": {
    "provider": "azureopenai",
    "error": "invalid endpoint URL"
  }
}
```

#### 创建百炼插件失败

```json
{
  "level": "ERROR",
  "message": "创建百炼插件失败",
  "fields": {
    "provider": "bailian",
    "error": "invalid API key"
  }
}
```

### 5. 提供商初始化错误

#### 初始化提供商失败

```json
{
  "level": "ERROR",
  "message": "初始化提供商失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "provider": "azureopenai",
    "error": "plugin initialization failed"
  }
}
```

#### 不支持的提供商类型

```json
{
  "level": "ERROR",
  "message": "不支持的提供商类型",
  "fields": {
    "provider": "unknown-provider"
  }
}
```

### 6. 生成内容错误

#### 参数验证错误

```json
{
  "level": "ERROR",
  "message": "租户ID不能为空"
}
```

```json
{
  "level": "ERROR",
  "message": "模型名称不能为空",
  "fields": {
    "tenantId": "tenant-123"
  }
}
```

```json
{
  "level": "ERROR",
  "message": "提示词不能为空",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4"
  }
}
```

#### 获取模型实例失败

```json
{
  "level": "ERROR",
  "message": "获取模型实例失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "error": "model configuration not found"
  }
}
```

#### 生成内容失败

```json
{
  "level": "ERROR",
  "message": "生成内容失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "model": "gpt-4-turbo",
    "duration": "5.2s",
    "error": "API call timeout"
  }
}
```

### 7. 流式生成错误

#### 启动流式生成失败

```json
{
  "level": "ERROR",
  "message": "启动流式生成失败",
  "fields": {
    "sessionId": "session-123",
    "tenantId": "tenant-456",
    "modelName": "gpt-4",
    "message": "用户的问题",
    "error": "failed to start stream"
  }
}
```

#### 流式生成出错

```json
{
  "level": "ERROR",
  "message": "流式生成出错",
  "fields": {
    "sessionId": "session-123",
    "tenantId": "tenant-456",
    "modelName": "gpt-4",
    "chunkCount": 150,
    "error": "connection reset",
    "errorType": "*net.OpError"
  }
}
```

#### 流式生成失败

```json
{
  "level": "ERROR",
  "message": "流式生成失败",
  "fields": {
    "tenantId": "tenant-123",
    "modelName": "gpt-4",
    "model": "gpt-4-turbo",
    "duration": "3.5s",
    "chunkCount": 10,
    "error": "stream interrupted"
  }
}
```

### 8. 上下文取消（警告级别）

#### 对话请求被取消

```json
{
  "level": "WARN",
  "message": "对话请求被取消",
  "fields": {
    "sessionId": "session-123",
    "tenantId": "tenant-456",
    "modelName": "gpt-4",
    "error": "context canceled"
  }
}
```

#### 流式对话请求被取消

```json
{
  "level": "WARN",
  "message": "流式对话请求被取消",
  "fields": {
    "sessionId": "session-123",
    "tenantId": "tenant-456",
    "modelName": "gpt-4",
    "chunkCount": 50
  }
}
```

### 9. Service 层错误

#### 无法从上下文获取JWT Claims

```json
{
  "level": "ERROR",
  "message": "无法从上下文获取JWT Claims",
  "fields": {
    "sessionId": "session-123"
  }
}
```

#### JWT Claims 中缺少租户ID

```json
{
  "level": "ERROR",
  "message": "JWT Claims 中缺少租户ID",
  "fields": {
    "sessionId": "session-123",
    "userId": "user-456"
  }
}
```

#### AI 生成失败

```json
{
  "level": "ERROR",
  "message": "AI 生成失败",
  "fields": {
    "sessionId": "session-123",
    "tenantId": "tenant-456",
    "modelName": "gpt-4",
    "message": "用户的问题",
    "error": "generation failed"
  }
}
```

## 日志字段说明

### 通用字段

- **timestamp**: 日志时间戳（ISO 8601 格式）
- **level**: 日志级别（ERROR、WARN、INFO）
- **message**: 错误描述信息

### 上下文字段

- **tenantId**: 租户ID，用于多租户隔离
- **modelName**: 模型名称，如 "gpt-4"、"gemini-pro"
- **model**: 实际使用的模型标识，如 "gpt-4-turbo"
- **provider**: 提供商类型，如 "azureopenai"、"bailian"
- **sessionId**: 会话ID，用于追踪对话
- **userId**: 用户ID

### 错误相关字段

- **error**: 错误详细信息
- **errorType**: 错误类型（Go 类型）
- **duration**: 操作耗时
- **chunkCount**: 流式传输已接收的块数或内容长度

### 配置相关字段

- **queryParams**: 查询参数内容
- **missingField**: 缺少的配置字段名
- **azureEndpoint**: Azure OpenAI 端点
- **azureDeployment**: Azure OpenAI 部署名称

## 日志查询示例

### 查询特定租户的错误

```bash
grep '"tenantId":"tenant-123"' app.log | grep '"level":"ERROR"'
```

### 查询特定模型的错误

```bash
grep '"modelName":"gpt-4"' app.log | grep '"level":"ERROR"'
```

### 查询配置相关错误

```bash
grep "配置" app.log | grep '"level":"ERROR"'
```

### 查询流式生成错误

```bash
grep "流式" app.log | grep '"level":"ERROR"'
```

### 统计错误类型

```bash
grep '"level":"ERROR"' app.log | jq -r '.message' | sort | uniq -c | sort -rn
```

## 最佳实践

### 1. 错误日志记录原则

- 记录所有错误场景
- 包含足够的上下文信息
- 使用结构化日志格式
- 避免记录敏感信息（API密钥、密码等）

### 2. 日志字段选择

- 必须包含：tenantId、modelName（如果适用）
- 建议包含：sessionId、userId（如果适用）
- 错误信息：error、errorType
- 性能指标：duration、chunkCount

### 3. 错误处理流程

1. 捕获错误
2. 记录详细日志
3. 返回友好的错误信息
4. 不在日志中暴露敏感信息

### 4. 日志监控

- 设置错误率告警
- 监控特定错误类型
- 按租户和模型维度分析
- 定期审查高频错误

## 相关文档

- [完整实施文档](./TASK-6.4-ERROR-LOGGING-COMPLETION.md)
- [日志规范](../../logger/README.md)
- [错误处理规范](../../model/ERRORS_USAGE.md)
- [任务列表](.kiro/specs/genkit-multi-model-support/tasks.md)
