# 日志持久化功能说明

## 功能概述

系统已实现日志持久化功能，支持按天自动轮转日志文件，便于通过 traceId 复查错误和追踪请求链路。

## 核心特性

### 1. 按天存储

- 日志文件按日期自动分割，格式：`app-YYYY-MM-DD.log`
- 每天零点自动创建新的日志文件
- 历史日志文件自动保留，便于回溯查询

### 2. 双输出模式

- **控制台 + 文件**：同时输出到控制台和文件（开发环境推荐）
- **仅文件**：只输出到文件（生产环境推荐）

### 3. TraceID 支持

- 所有日志自动包含 traceId 字段
- 通过 traceId 可以追踪完整的请求链路
- 便于定位和分析错误

### 4. 自动轮转

- 日期变更时自动关闭旧文件，创建新文件
- 无需手动干预，零停机时间
- 线程安全，支持高并发写入

## 配置说明

### 环境变量配置

在 `.env` 文件中添加以下配置：

```bash
# 日志配置
# 日志级别: debug, info, warn, error
LOG_LEVEL=info

# 日志格式: json, text
LOG_FORMAT=json

# 是否启用文件日志（按天存储）
LOG_ENABLE_FILE=true

# 日志文件目录
LOG_DIR=logs

# 是否同时输出到控制台
LOG_ENABLE_CONSOLE=true
```

### 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `LOG_LEVEL` | string | `info` | 日志级别：debug, info, warn, error |
| `LOG_FORMAT` | string | `json` | 日志格式：json（结构化）, text（纯文本） |
| `LOG_ENABLE_FILE` | bool | `true` | 是否启用文件日志 |
| `LOG_DIR` | string | `logs` | 日志文件存储目录 |
| `LOG_ENABLE_CONSOLE` | bool | `true` | 是否同时输出到控制台 |

## 使用示例

### 1. 开发环境配置

```bash
# 同时输出到控制台和文件，便于实时查看
LOG_ENABLE_FILE=true
LOG_ENABLE_CONSOLE=true
LOG_LEVEL=debug
LOG_FORMAT=text
```

### 2. 生产环境配置

```bash
# 仅输出到文件，减少控制台噪音
LOG_ENABLE_FILE=true
LOG_ENABLE_CONSOLE=false
LOG_LEVEL=info
LOG_FORMAT=json
```

### 3. 测试环境配置

```bash
# 仅输出到控制台，不保存文件
LOG_ENABLE_FILE=false
LOG_ENABLE_CONSOLE=true
LOG_LEVEL=debug
LOG_FORMAT=text
```

## 日志文件结构

### 目录结构

```
logs/
├── app-2025-10-25.log    # 今天的日志
├── app-2025-10-24.log    # 昨天的日志
├── app-2025-10-23.log    # 前天的日志
└── ...
```

### JSON 格式示例

```json
{
  "timestamp": "2025-10-25T10:30:45Z",
  "level": "INFO",
  "message": "用户登录成功",
  "fields": {
    "traceId": "abc123def456",
    "userId": "user-uuid-123",
    "tenantId": "tenant-uuid-456",
    "ip": "192.168.1.100"
  }
}
```

### Text 格式示例

```
2025-10-25T10:30:45Z [INFO] 用户登录成功 traceId=abc123def456 userId=user-uuid-123 tenantId=tenant-uuid-456 ip=192.168.1.100
```

## TraceID 使用指南

### 1. 自动注入

所有通过 HTTP 请求进入的日志都会自动包含 traceId：

```go
// 在 Handler 中使用
logger.InfoContext(ctx, "处理用户请求", logger.Fields{
    "action": "create_user",
})
// 输出: {"timestamp":"...","level":"INFO","message":"处理用户请求","fields":{"traceId":"xxx","action":"create_user"}}
```

### 2. 错误追踪

当发生错误时，记录 traceId 便于后续查询：

```go
logger.ErrorContext(ctx, "创建用户失败", logger.Fields{
    "error": err.Error(),
    "email": email,
})
```

### 3. 日志查询

使用 grep 或其他工具查询特定 traceId 的所有日志：

```bash
# 查询今天的日志
grep "abc123def456" logs/app-2025-10-25.log

# 查询最近3天的日志
grep "abc123def456" logs/app-2025-10-*.log

# 使用 jq 解析 JSON 格式日志
grep "abc123def456" logs/app-2025-10-25.log | jq .
```

## 日志管理建议

### 1. 日志保留策略

建议使用 logrotate 或定时任务清理旧日志：

```bash
# 保留最近 30 天的日志
find logs/ -name "app-*.log" -mtime +30 -delete
```

### 2. 日志压缩

压缩旧日志以节省磁盘空间：

```bash
# 压缩 7 天前的日志
find logs/ -name "app-*.log" -mtime +7 -exec gzip {} \;
```

### 3. 日志备份

定期备份重要日志到远程存储：

```bash
# 备份到 S3 或其他对象存储
aws s3 sync logs/ s3://my-bucket/logs/
```

### 4. 日志监控

使用日志分析工具监控错误率和性能：

- **ELK Stack**：Elasticsearch + Logstash + Kibana
- **Grafana Loki**：轻量级日志聚合系统
- **CloudWatch Logs**：AWS 云原生日志服务

## 性能优化

### 1. 异步写入

日志写入已经过优化，不会阻塞主业务逻辑。

### 2. 缓冲写入

使用 Go 标准库的缓冲 I/O，减少系统调用次数。

### 3. 文件句柄管理

- 自动管理文件句柄的打开和关闭
- 日期变更时自动轮转，无需重启服务
- 程序退出时自动关闭文件句柄

## 故障排查

### 问题1：日志文件未创建

**可能原因**：

- 日志目录不存在或无写入权限
- `LOG_ENABLE_FILE` 设置为 `false`

**解决方案**：

```bash
# 检查目录权限
ls -la logs/

# 手动创建目录
mkdir -p logs
chmod 755 logs

# 检查配置
grep LOG_ENABLE_FILE .env
```

### 问题2：日志未轮转

**可能原因**：

- 服务未跨越日期边界运行
- 系统时区配置错误

**解决方案**：

```bash
# 检查系统时间
date

# 检查时区设置
timedatectl
```

### 问题3：磁盘空间不足

**可能原因**：

- 日志文件过多未清理
- 日志级别设置过低（debug）

**解决方案**：

```bash
# 检查磁盘使用情况
df -h

# 检查日志目录大小
du -sh logs/

# 清理旧日志
find logs/ -name "app-*.log" -mtime +30 -delete
```

## 最佳实践

### 1. 生产环境

- ✅ 启用文件日志
- ✅ 禁用控制台输出（减少 I/O）
- ✅ 使用 JSON 格式（便于解析）
- ✅ 设置 INFO 或 WARN 级别
- ✅ 定期清理旧日志
- ✅ 配置日志监控告警

### 2. 开发环境

- ✅ 启用文件日志
- ✅ 启用控制台输出（便于调试）
- ✅ 使用 TEXT 格式（便于阅读）
- ✅ 设置 DEBUG 级别
- ✅ 不需要清理日志

### 3. 测试环境

- ✅ 可选文件日志
- ✅ 启用控制台输出
- ✅ 使用 TEXT 格式
- ✅ 设置 DEBUG 级别
- ✅ 测试后清理日志

## 代码示例

### 基本使用

```go
import "genkit-ai-service/internal/logger"

// 记录普通日志
logger.Info("服务启动", logger.Fields{
    "port": 8080,
    "version": "1.0.0",
})

// 记录带上下文的日志（自动包含 traceId）
logger.InfoContext(ctx, "处理请求", logger.Fields{
    "method": "POST",
    "path": "/api/v1/users",
})

// 记录错误日志
logger.ErrorContext(ctx, "数据库连接失败", logger.Fields{
    "error": err.Error(),
    "host": "localhost",
})
```

### 创建子 Logger

```go
// 创建带预设字段的 logger
userLogger := logger.WithFields(logger.Fields{
    "module": "user_service",
    "version": "v1",
})

userLogger.Info("用户服务初始化完成")
// 输出: {"timestamp":"...","level":"INFO","message":"用户服务初始化完成","fields":{"module":"user_service","version":"v1"}}
```

## 总结

日志持久化功能提供了：

1. ✅ **低成本**：无需额外的日志服务，直接存储到本地文件
2. ✅ **高可靠**：自动轮转，不会丢失日志
3. ✅ **易查询**：通过 traceId 快速定位问题
4. ✅ **零配置**：开箱即用，默认配置适合大多数场景
5. ✅ **高性能**：异步写入，不影响业务性能

通过合理配置和使用日志持久化功能，可以大大提升系统的可观测性和问题排查效率。
