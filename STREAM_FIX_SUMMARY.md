# 流式聊天主键冲突问题修复总结

## 问题描述

在流式聊天接口中，当 AI 响应结束时，系统尝试创建一个新的消息记录，导致主键冲突错误：

```
data: {"event":"error","error":"更新AI消息失败: 创建消息失败: ERROR: duplicate key value violates unique constraint \"chat_messages_pkey\" (SQLSTATE 23505)"}
```

## 问题原因

在 `internal/service/session/message_service.go` 的 `SendMessageStream` 方法中存在两个问题：

### 1. 重复调用 Create 方法

代码在两个地方调用了 `messageRepo.Create(ctx, aiMessage)`：

- **第一次**（第 587 行）：创建初始的空 AI 消息记录
- **第二次**（第 668 行）：尝试"更新" AI 消息内容，但错误地使用了 `Create` 方法

第二次调用应该使用 `Update` 方法而不是 `Create`，因为消息记录已经存在。

### 2. 未正确处理流结束事件

在处理流式响应时，代码没有正确处理 `chunk.Done` 标志，导致：

- 在流结束时发送了空的 content 事件
- 没有在检测到 `Done` 标志时立即跳出循环
- 空内容块也被发送到客户端

## 修复方案

### 1. 添加 Update 方法到消息仓储

在 `internal/repository/message_repository.go` 中：

```go
// MessageRepository 接口添加 Update 方法
type MessageRepository interface {
    // ... 其他方法
    Update(ctx context.Context, message *model.ChatMessage) error
}

// 实现 Update 方法
func (r *messageRepository) Update(ctx context.Context, message *model.ChatMessage) error {
    if err := r.db.WithContext(ctx).Save(message).Error; err != nil {
        return fmt.Errorf("更新消息失败: %w", err)
    }
    return nil
}
```

### 2. 修复流式响应处理逻辑

在 `internal/service/session/message_service.go` 中修改流式响应处理：

```go
for chunk := range streamChan {
    // 处理错误
    if chunk.Error != nil {
        // ... 错误处理
        return
    }

    // 检查是否完成（在处理内容之前）
    if chunk.Done {
        // 保存模型和使用信息
        if chunk.Model != "" {
            lastModel = chunk.Model
        }
        if chunk.Usage != nil {
            lastUsage = chunk.Usage
        }
        break  // 立即跳出循环
    }

    // 跳过空内容块
    if chunk.Content == "" {
        continue
    }

    // 发送内容块
    // ...
}

// 使用 Update 而不是 Create
if err := s.messageRepo.Update(ctx, aiMessage); err != nil {
    // ... 错误处理
}
```

### 关键改进点

1. **提前检查 Done 标志**：在处理内容之前检查 `chunk.Done`，避免处理空内容块
2. **跳过空内容**：添加 `if chunk.Content == "" { continue }` 跳过空内容块
3. **使用 Update 方法**：将 `messageRepo.Create` 改为 `messageRepo.Update`
4. **保存元数据**：在 Done 块中保存模型和使用信息后立即 break

## 修复的文件

1. `internal/repository/message_repository.go`
   - 添加 `Update` 方法到接口定义
   - 实现 `Update` 方法

2. `internal/service/session/message_service.go`
   - 修改流式响应处理逻辑
   - �� `Create` 改为 `Update`
   - 优化 Done 标志和空内容的处理

## 测试验证

运行测试脚本验证修复：

```bash
./test_stream_fix.sh
```

测试内容：

1. 创建测试会话
2. 发送流式消息
3. 检查是否有主键冲突错误
4. 验证消息正确保存（用户消息 + AI 消息）
5. 统计事件类型

## 预期结果

修复后，流式聊天应该：

- ✅ 不再出现主键冲突错误
- ✅ 正确保存用户消息和 AI 消息
- ✅ 流式响应正常完成
- ✅ 不发送空的 content 事件
- ✅ 正确发送 done 事件

## 注意事项

1. **数据库事务**：Update 操作在事务外执行，如果失败会发送错误事件
2. **幂等性**：Update 方法使用 GORM 的 `Save`，会更新所有字段
3. **并发安全**：流式处理在 goroutine 中执行，需要注意上下文取消
4. **错误处理**：所有错误都通过流式通道发送 error 事件

## 相关代码位置

- 消息服务：`internal/service/session/message_service.go:SendMessageStream`
- 消息仓储：`internal/repository/message_repository.go:Update`
- 消息处理器：`internal/api/handler/message_handler.go:SendMessageStream`
