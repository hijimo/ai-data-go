# Context Canceled 错误修复总结

## 问题描述

在流式消息发送过程中，出现 `context canceled` 错误，导致AI消息内容无法正确保存到数据库。

### 错误日志

```json
{
  "timestamp": "2025-11-13T13:48:21Z",
  "level": "ERROR",
  "message": "更新AI消息失败",
  "fields": {
    "error": "更新消息失败: context canceled",
    "messageId": "1633ae8b-ac5c-40e2-9149-fe3195826b41",
    "requestId": "8209a958-5f90-4454-a7e9-630894e84c52",
    "traceId": "trace-1763041689-8e0c018fb0b5"
  }
}
```

## 根本原因

在 `SendMessageStream` 方法中，goroutine 使用的是 HTTP 请求的 context。当发生以下情况时，这个 context 会被取消：

1. 客户端主动断开连接
2. 请求超时
3. 网络中断

当 context 被取消后，所有使用该 context 的数据库操作都会失败，导致：

- AI消息内容无法保存
- 会话信息无法更新
- 消息计数无法增加

## 解决方案

### 核心思路

为数据库操作创建独立的 context，不受 HTTP 请求 context 取消的影响。这样即使客户端断开连接，也能完成消息的持久化。

### 实现细节

1. **创建独立的数据库 context**

   ```go
   // 创建独立的context用于数据库操作
   dbCtx := context.Background()
   // 从原始context中复制traceId等重要信息
   if traceID := ctx.Value("traceId"); traceID != nil {
       dbCtx = context.WithValue(dbCtx, "traceId", traceID)
   }
   if requestID := ctx.Value("requestId"); requestID != nil {
       dbCtx = context.WithValue(dbCtx, "requestId", requestID)
   }
   ```

2. **区分使用场景**
   - **AI服务调用**：继续使用原始 `ctx`，这样可以响应客户端的取消请求
   - **数据库操作**：使用独立的 `dbCtx`，确保数据持久化不受影响

3. **修改的操作**
   - 获取消息序列号：使用 `dbCtx`
   - 保存用户消息：使用 `dbCtx`
   - 创建AI消息记录：使用 `dbCtx`
   - 更新AI消息内容：使用 `dbCtx`
   - 更新会话信息：使用 `dbCtx`
   - 日志记录：使用 `dbCtx`

## 修改文件

- `internal/service/session/message_service.go`

## 优势

1. **数据完整性**：即使客户端断开，消息也能正确保存
2. **可追踪性**：保留了 traceId 和 requestId，便于日志追踪
3. **用户体验**：客户端可以随时取消请求，不影响后端数据一致性
4. **资源管理**：AI服务调用仍然可以被取消，避免资源浪费

## 测试建议

1. 测试客户端主动断开连接的场景
2. 测试网络超时的场景
3. 验证消息是否正确保存到数据库
4. 检查日志中的 traceId 是否正确传递
5. 确认会话信息是否正确更新

## 注意事项

- 独立的 context 不会自动超时，需要确保数据库操作本身有合理的超时设置
- 如果需要，可以为 dbCtx 添加超时控制：`context.WithTimeout(context.Background(), 30*time.Second)`
