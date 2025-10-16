# 启动 Genkit AI 服务

## 快速启动

服务已经构建完成！请按照以下步骤启动：

### 1. 检查环境变量

确保 `.env` 文件中的配置正确：

```bash
# 查看当前配置
cat .env
```

关键配置项：

- `GEMINI_API_KEY` - 已设置 ✓
- `DB_HOST` - localhost
- `DB_PORT` - 5432
- `DB_NAME` - ai_service
- `DB_USER` - postgres
- `DB_PASSWORD` - password
- `SERVER_PORT` - 8080

### 2. 启动数据库（如果还未启动）

如果你使用 Docker：

```bash
docker run -d \
  --name postgres-ai \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=ai_service \
  -p 5432:5432 \
  postgres:15
```

如果你使用本地 PostgreSQL：

```bash
# macOS (Homebrew)
brew services start postgresql@15

# 创建数据库
createdb ai_service
```

### 3. 启动服务

**方式一：直接运行二进制文件**

```bash
./bin/server
```

**方式二：使用 go run**

```bash
go run ./cmd/server/main.go
```

**方式三：使用启动脚本**

```bash
chmod +x bin/start.sh
./bin/start.sh
```

### 4. 验证服务

服务启动后，你应该看到类似的日志输出：

```json
{"timestamp":"2025-10-16T...","level":"INFO","message":"服务启动中...","fields":{"version":"1.0.0","port":"8080"}}
{"timestamp":"2025-10-16T...","level":"INFO","message":"初始化数据库连接...","fields":{"host":"localhost","port":"5432","name":"ai_service"}}
{"timestamp":"2025-10-16T...","level":"INFO","message":"数据库连接成功","fields":{"host":"localhost"}}
{"timestamp":"2025-10-16T...","level":"INFO","message":"初始化 Genkit 客户端...","fields":{"model":"gemini-2.5-flash"}}
{"timestamp":"2025-10-16T...","level":"INFO","message":"Genkit 客户端初始化成功","fields":{"model":"gemini-2.5-flash"}}
{"timestamp":"2025-10-16T...","level":"INFO","message":"初始化 AI 服务..."}
{"timestamp":"2025-10-16T...","level":"INFO","message":"AI 服务初始化成功"}
{"timestamp":"2025-10-16T...","level":"INFO","message":"HTTP 服务器启动","fields":{"address":"0.0.0.0:8080"}}
```

### 5. 测试 API

**健康检查：**

```bash
curl http://localhost:8080/health
```

预期响应：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime": "5s",
    "dependencies": {
      "database": "connected",
      "genkit": "connected"
    }
  }
}
```

**对话接口：**

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下自己",
    "sessionId": "test-session-001"
  }'
```

### 6. 优雅关闭

按 `Ctrl+C` 或发送 SIGTERM 信号：

```bash
# 如果在后台运行
kill -TERM <pid>
```

服务会优雅关闭，清理所有资源。

## 故障排查

### 数据库连接失败

如果看到错误：`连接数据库失败`

1. 检查 PostgreSQL 是否运行：

   ```bash
   # macOS
   brew services list | grep postgresql
   
   # 或尝试连接
   psql -h localhost -U postgres -d ai_service
   ```

2. 检查数据库配置：

   ```bash
   echo $DB_HOST $DB_PORT $DB_NAME $DB_USER
   ```

3. 创建数据库（如果不存在）：

   ```bash
   createdb -U postgres ai_service
   ```

### Genkit 初始化失败

如果看到错误：`初始化 Genkit 客户端失败`

1. 检查 API 密钥：

   ```bash
   echo $GEMINI_API_KEY
   ```

2. 验证 API 密钥是否有效（访问 Google AI Studio）

### 端口被占用

如果看到错误：`bind: address already in use`

1. 查找占用端口的进程：

   ```bash
   lsof -i :8080
   ```

2. 修改端口：

   ```bash
   export SERVER_PORT=8081
   ./bin/server
   ```

## 开发模式

如果需要实时重载，可以使用 `air`：

```bash
# 安装 air
go install github.com/cosmtrek/air@latest

# 运行
air
```

## 生产部署

生产环境建议：

1. 使用环境变量而不是 .env 文件
2. 配置反向代理（Nginx/Caddy）
3. 使用进程管理器（systemd/supervisor）
4. 启用 HTTPS
5. 配置日志收集
6. 设置监控和告警

---

**服务已准备就绪！** 🚀

现在请在终端中运行 `./bin/server` 启动服务。
