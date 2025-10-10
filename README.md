# AI知识管理平台

一个支持RAG（检索增强生成）、模型蒸馏、SFT（监督微调）等功能的大模型知识管理平台。

## 功能特性

- 📄 **多格式文档处理**: 支持PDF、DOCX、Markdown、TXT、HTML等格式
- 🧠 **智能分块**: 基于语义的智能文档分割
- 🔍 **向量检索**: 高效的语义搜索和相似度检索
- 🤖 **多模型支持**: 集成OpenAI、Azure、千问、Claude等LLM提供商
- 👥 **Agent系统**: 创建和管理专门的AI智能体
- 💬 **对话系统**: 支持多轮对话和上下文管理
- 📊 **数据集生成**: 自动生成训练数据集
- 🎯 **模型训练**: 集成阿里百炼等训练平台
- 🔐 **权限管理**: 基于RBAC的项目级权限控制
- 📈 **监控告警**: 完整的可观测性支持

## 技术架构

- **后端**: Go + Gin + GORM + PostgreSQL + Redis
- **前端**: React 18 + TypeScript + Ant Design 5.x
- **向量存储**: ADB-PG（阿里云分析型数据库PostgreSQL版）
- **对象存储**: 阿里云OSS
- **密钥管理**: 阿里云KMS
- **监控**: Prometheus + Grafana
- **容器化**: Docker + Docker Compose

## 快速开始

### 环境要求

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose（可选）

### 本地开发

1. **克隆项目**

```bash
git clone <repository-url>
cd ai-knowledge-platform
```

2. **安装依赖**

```bash
make deps
```

3. **配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，配置数据库和其他服务连接信息
```

4. **运行数据库迁移**

```bash
make migrate-up
```

5. **生成API文档**

```bash
make swagger
```

6. **启动服务**

```bash
make run
```

服务将在 `http://localhost:8080` 启动。

### Docker部署

1. **使用Docker Compose启动所有服务**

```bash
make docker-run
```

这将启动以下服务：

- API服务 (端口 8080)
- PostgreSQL (端口 5432)
- Redis (端口 6379)
- MinIO (端口 9000, 9001)
- Prometheus (端口 9090)
- Grafana (端口 3000)

2. **查看服务状态**

```bash
docker-compose ps
```

3. **查看日志**

```bash
make docker-logs
```

## API文档

启动服务后，可以通过以下地址访问API文档：

- Swagger UI: <http://localhost:8080/swagger/index.html>
- 健康检查: <http://localhost:8080/health>
- 监控指标: <http://localhost:8080/metrics>

## 开发指南

### 项目结构

```
ai-knowledge-platform/
├── cmd/                    # 应用入口
│   └── server/            # 服务器主程序
├── internal/              # 内部包
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接和迁移
│   ├── cache/            # 缓存管理
│   ├── middleware/       # 中间件
│   ├── router/           # 路由配置
│   └── handler/          # 请求处理器
├── migrations/           # 数据库迁移文件
├── docs/                # API文档
├── monitoring/          # 监控配置
├── docker-compose.yml   # Docker编排文件
├── Dockerfile          # Docker镜像构建文件
├── Makefile           # 构建脚本
└── README.md          # 项目说明
```

### 可用命令

```bash
make help          # 查看所有可用命令
make deps          # 安装依赖
make build         # 构建应用
make run           # 运行应用
make test          # 运行测试
make swagger       # 生成API文档
make migrate-up    # 运行数据库迁移
make migrate-down  # 回滚数据库迁移
make docker-build  # 构建Docker镜像
make docker-run    # 启动Docker服务
```

### 数据库迁移

创建新的迁移文件：

```bash
make migrate-create
```

运行迁移：

```bash
make migrate-up
```

回滚迁移：

```bash
make migrate-down
```

### 测试

运行所有测试：

```bash
make test
```

运行特定包的测试：

```bash
go test -v ./internal/config
```

## 监控和运维

### Prometheus指标

系统暴露以下监控指标：

- HTTP请求数量和延迟
- 数据库连接池状态
- Redis连接状态
- 业务指标（文档处理、向量检索等）

### Grafana仪表板

访问 <http://localhost:3000> 查看监控仪表板：

- 用户名: admin
- 密码: admin123

### 日志

应用使用结构化JSON日志格式，包含以下字段：

- level: 日志级别
- timestamp: 时间戳
- message: 日志消息
- service: 服务名称
- trace_id: 链路追踪ID
- user_id: 用户ID
- project_id: 项目ID

## 部署

### 生产环境部署

1. **构建生产镜像**

```bash
docker build -t ai-knowledge-platform:latest .
```

2. **配置环境变量**
确保生产环境配置了所有必要的环境变量。

3. **运行服务**

```bash
docker run -d \
  --name ai-knowledge-platform \
  -p 8080:8080 \
  --env-file .env.prod \
  ai-knowledge-platform:latest
```

### Kubernetes部署

项目包含Helm Chart配置，支持Kubernetes部署：

```bash
helm install ai-knowledge-platform ./charts/ai-knowledge-platform
```

## 贡献指南

1. Fork项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建Pull Request

## 许可证

本项目采用 Apache 2.0 许可证。详见 [LICENSE](LICENSE) 文件。

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交Issue: [GitHub Issues](https://github.com/your-org/ai-knowledge-platform/issues)
- 邮箱: <support@example.com>

## 更新日志

查看 [CHANGELOG.md](CHANGELOG.md) 了解版本更新历史。
