# 日志持久化功能实现总结

## 实现内容

为 logger 添加了按天存储的文件持久化能力，支持通过 traceId 低成本复查错误。

## 核心功能

### 1. 按天轮转

- 日志文件格式：`logs/app-YYYY-MM-DD.log`
- 每天零点自动创建新文件
- 自动关闭旧文件，无需重启服务

### 2. 双输出模式

- 同时输出到控制台和文件（开发环境）
- 仅输出到文件（生产环境）
- 可通过配置灵活切换

### 3. TraceID 支持

- 所有日志自动包含 traceId 字段
- 便于追踪完整请求链路
- 支持跨天查询

## 修改文件

### 1. internal/logger/logger.go

- 添加 `NewWithFile()` 函数：创建带文件持久化的 logger
- 添加 `InitWithFile()` 函数：初始化默认文件 logger
- 添加 `rotateLogFile()` 方法：实现日志文件轮转
- 添加 `Close()` 方法：关闭文件句柄
- 扩展 `logger` 结构体：添加文件相关字段
- 优化 `write()` 方法：支持自动轮转

### 2. cmd/server/main.go

- 更新日志初始化逻辑
- 支持文件日志配置
- 添加文件句柄自动关闭

### 3. internal/config/config.go

- 扩展 `LogConfig` 结构体
- 添加 `EnableFile`、`LogDir`、`EnableConsole` 配置项
- 更新配置加载逻辑

### 4. .env 和 .env.example

- 添加日志持久化配置项
- 提供详细的配置说明

### 5. .gitignore

- 添加 `logs/` 目录忽略规则

## 配置说明

```bash
# 是否启用文件日志
LOG_ENABLE_FILE=true

# 日志文件目录
LOG_DIR=logs

# 是否同时输出到控制台
LOG_ENABLE_CONSOLE=true
```

## 使用示例

### 开发环境

```bash
LOG_ENABLE_FILE=true
LOG_ENABLE_CONSOLE=true
LOG_LEVEL=debug
LOG_FORMAT=text
```

### 生产环境

```bash
LOG_ENABLE_FILE=true
LOG_ENABLE_CONSOLE=false
LOG_LEVEL=info
LOG_FORMAT=json
```

## TraceID 查询

```bash
# 查询今天的日志
grep "traceId-xxx" logs/app-2025-10-25.log

# 查询所有日志
grep "traceId-xxx" logs/app-*.log

# 使用 jq 解析 JSON 日志
grep "traceId-xxx" logs/app-2025-10-25.log | jq .
```

## 日志管理

```bash
# 删除 30 天前的日志
find logs/ -name "app-*.log" -mtime +30 -delete

# 压缩 7 天前的日志
find logs/ -name "app-*.log" -mtime +7 -exec gzip {} \;

# 查看日志目录大小
du -sh logs/
```

## 测试验证

1. 编译成功：`go build -o bin/server cmd/server/main.go`
2. 无语法错误：所有文件通过 getDiagnostics 检查
3. 提供测试脚本：`./test_log_persistence.sh`

## 优势

1. **低成本**：无需额外日志服务，直接存储到本地文件
2. **高可靠**：自动轮转，不会丢失日志
3. **易查询**：通过 traceId 快速定位问题
4. **零配置**：开箱即用，默认配置适合大多数场景
5. **高性能**：异步写入，不影响业务性能
6. **线程安全**：支持高并发写入

## 文档

- `LOG_PERSISTENCE_README.md`：详细使用文档
- `test_log_persistence.sh`：功能测试脚本
- `.env.example`：配置示例

## 下一步

1. 启动服务测试日志功能
2. 发送 HTTP 请求生成日志
3. 验证 traceId 追踪功能
4. 测试日志文件轮转（跨天）
