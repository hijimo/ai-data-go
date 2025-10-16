# Genkit AI Service

基于 Firebase Genkit 构建的标准分布式架构 Go 语言 AI 服务。

## 项目结构

```
genkit-ai-service/
├── cmd/                    # 应用程序入口
│   └── server/            # 服务器主程序
├── internal/              # 内部包（不对外暴露）
│   ├── api/              # API 层
│   │   ├── handler/      # 请求处理器
│   │   └── middleware/   # 中间件
│   ├── service/          # 服务层
│   │   ├── ai/          # AI 服务
│   │   └── health/      # 健康检查服务
│   ├── repository/       # 数据访问层
│   ├── genkit/          # Genkit 封装
│   ├── database/        # 数据库连接管理
│   ├── model/           # 数据模型
│   ├── config/          # 配置管理
│   └── logger/          # 日志管理
└── pkg/                  # 公共包（可对外暴露）
    ├── response/        # 统一响应构建器
    ├── validator/       # 参数验证器
    └── errors/          # 错误定义
```

## 快速开始

### 环境要求

- Go 1.21 或更高版本
- PostgreSQL 数据库
- Firebase Genkit API 密钥

### 安装依赖

```bash
go mod download
```

### 配置环境变量

复制 `.env.example` 文件为 `.env` 并填写相应的配置：

```bash
cp .env.example .env
```

编辑 `.env` 文件，填写必要的配置信息。

### 运行服务

```bash
go run cmd/server/main.go
```

或编译后运行：

```bash
go build -o bin/server cmd/server/main.go
./bin/server
```

## 主要依赖

- **Firebase Genkit**: AI 模型集成
- **gorilla/mux**: HTTP 路由
- **lib/pq**: PostgreSQL 驱动
- **logrus**: 结构化日志
- **validator**: 参数验证
- **godotenv**: 环境变量管理

## 开发状态

项目当前处于初始化阶段，基础架构已搭建完成。

## 许可证

MIT
