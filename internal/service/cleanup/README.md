# 数据库清理服务

## 概述

数据库清理服务（Cleanup Service）负责定期清理数据库中的过期数据，包括：

- 过期的 Refresh Token
- 审计日志归档（可选，未来实现）

## 功能特性

### 1. Refresh Token 清理

- 定期扫描 `refresh_tokens` 表
- 删除 `expires_at` 早于当前时间的记录
- 默认清理间隔：1 小时（可配置）

### 2. 自动启动和停止

- 应用启动时自动启动清理服务
- 应用关闭时优雅停止清理服务
- 支持上下文取消

## 配置

### 环境变量

在 `.env` 文件中配置清理间隔：

```bash
# Token 清理间隔（默认 1 小时）
TOKEN_CLEANUP_INTERVAL=1h
```

支持的时间单位：

- `s` - 秒
- `m` - 分钟
- `h` - 小时

示例：

- `30m` - 30 分钟
- `2h` - 2 小时
- `1h30m` - 1 小时 30 分钟

### 代码配置

```go
import "genkit-ai-service/internal/service/cleanup"

// 创建清理服务配置
config := cleanup.CleanupConfig{
    TokenCleanupInterval: 1 * time.Hour,
}

// 创建清理服务实例
cleanupService := cleanup.NewCleanupService(
    refreshTokenRepo,
    logger,
    config,
)

// 启动清理服务
ctx := context.Background()
cleanupService.Start(ctx)

// 停止清理服务（应用关闭时）
cleanupService.Stop()
```

## 使用方式

### 在应用中集成

清理服务已经在 `cmd/server/main.go` 中自动集成：

1. **启动时**：在认证服务初始化后自动启动
2. **运行时**：后台定期执行清理任务
3. **关闭时**：收到关闭信号时优雅停止

### 手动触发清理

如果需要手动触发清理（例如在管理接口中）：

```go
ctx := context.Background()
if err := cleanupService.CleanExpiredTokens(ctx); err != nil {
    log.Error("手动清理失败", map[string]interface{}{
        "error": err.Error(),
    })
}
```

## 工作原理

### 清理流程

1. **初始清理**：服务启动时立即执行一次清理
2. **定时清理**：使用 `time.Ticker` 定期触发清理
3. **数据库操作**：调用 `RefreshTokenRepository.DeleteExpired()` 删除过期记录
4. **日志记录**：记录清理开始、完成和错误信息

### 清理逻辑

```sql
-- 删除过期的 Refresh Token
DELETE FROM refresh_tokens 
WHERE expires_at < NOW();
```

### 并发安全

- 使用 goroutine 运行后台清理任务
- 通过 channel 实现优雅停止
- 支持上下文取消

## 监控和日志

### 日志输出

清理服务会记录以下日志：

```json
// 启动日志
{
  "level": "INFO",
  "message": "启动数据库清理服务",
  "fields": {
    "interval": "1h0m0s"
  }
}

// 清理开始
{
  "level": "INFO",
  "message": "开始清理过期的 Refresh Token"
}

// 清理完成
{
  "level": "INFO",
  "message": "清理过期 Token 完成",
  "fields": {
    "duration": "15.234ms"
  }
}

// 清理失败
{
  "level": "ERROR",
  "message": "清理过期 Token 失败",
  "fields": {
    "error": "database connection lost"
  }
}

// 停止日志
{
  "level": "INFO",
  "message": "停止数据库清理服务"
}
```

### 监控指标（建议）

可以添加以下监控指标：

- 清理执行次数
- 清理删除的记录数
- 清理执行时间
- 清理失败次数

## 性能考虑

### 数据库索引

确保 `refresh_tokens` 表有以下索引：

```sql
CREATE INDEX idx_refresh_tokens_expires_at 
ON refresh_tokens(expires_at);
```

### 清理间隔建议

- **开发环境**：15-30 分钟（快速清理测试数据）
- **生产环境**：1-2 小时（平衡性能和及时性）
- **高负载环境**：可以延长到 4-6 小时

### 性能优化

1. **批量删除**：当前实现使用单个 DELETE 语句
2. **分批处理**：如果记录数量很大，可以考虑分批删除
3. **非高峰期执行**：可以配置在业务低峰期执行

## 扩展功能

### 审计日志归档（未来实现）

```go
// 归档旧的审计日志到归档表
func (s *cleanupService) ArchiveAuditLogs(ctx context.Context) error {
    // 将 30 天前的审计日志移动到归档表
    // ...
}
```

### 会话清理（未来实现）

```go
// 清理过期的会话数据
func (s *cleanupService) CleanExpiredSessions(ctx context.Context) error {
    // 删除超过保留期的会话记录
    // ...
}
```

## 故障排查

### 清理服务未启动

检查：

1. 数据库连接是否正常
2. 配置是否正确加载
3. 日志中是否有错误信息

### 清理失败

可能原因：

1. 数据库连接丢失
2. 权限不足
3. 表锁定

解决方法：

1. 检查数据库连接状态
2. 验证数据库用户权限
3. 检查是否有长时间运行的事务

### 性能问题

如果清理操作影响性能：

1. 增加清理间隔
2. 在业务低峰期执行
3. 考虑分批删除
4. 优化数据库索引

## 测试

运行清理服务测试：

```bash
# 运行所有测试
go test ./internal/service/cleanup/...

# 运行特定测试
go test -v ./internal/service/cleanup/... -run TestCleanupService_CleanExpiredTokens

# 运行测试并显示覆盖率
go test -cover ./internal/service/cleanup/...
```

## 相关文档

- [认证服务设计文档](../../specs/multi-tenant-auth/design.md)
- [Refresh Token Repository](../../repository/refresh_token_repository.go)
- [配置管理](../../config/config.go)
