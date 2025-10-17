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

#### 主要配置项

- **SERVER_PORT**: 服务器端口（默认：8080）
- **SERVER_HOST**: 服务器主机地址（默认：0.0.0.0）
- **GENKIT_API_KEY**: Genkit API 密钥（必需）
- **GENKIT_MODEL**: 默认使用的模型（默认：gemini-2.5-flash）
- **MODELS_DIR**: 模型配置文件目录（默认：./models）
- **DB_HOST**: 数据库主机（默认：localhost）
- **DB_PORT**: 数据库端口（默认：5432）
- **DB_USER**: 数据库用户名（默认：postgres）
- **DB_PASSWORD**: 数据库密码
- **DB_NAME**: 数据库名称（默认：genkit_ai_service）
- **LOG_LEVEL**: 日志级别（默认：info）
- **LOG_FORMAT**: 日志格式（默认：json）

#### 模型配置目录

服务启动时会从 `MODELS_DIR` 指定的目录加载所有模型提供商和模型的配置信息。目录结构应如下：

```
models/
├── gemini/
│   ├── provider/
│   │   └── gemini.yaml
│   └── models/
│       ├── llm/
│       │   ├── _position.yaml
│       │   ├── gemini-2.5-flash.yaml
│       │   └── ...
│       └── text_embedding/
│           └── ...
└── tongyi/
    ├── provider/
    │   └── tongyi.yaml
    └── models/
        └── ...
```

如果需要使用自定义的模型配置目录，可以通过环境变量指定：

```bash
export MODELS_DIR=/path/to/your/models
```

### 运行服务

```bash
go run cmd/server/main.go
```

或编译后运行：

```bash
go build -o bin/server cmd/server/main.go
./bin/server
```

## API 接口

### 📚 API 文档

项目已集成 Swagger UI，提供交互式 API 文档：

**访问地址**: <http://localhost:8080/swagger/index.html>

启动服务后，在浏览器中打开上述地址即可查看完整的 API 文档，包括：

- 所有接口的详细说明
- 请求参数和响应格式
- 在线测试功能
- 数据模型定义

详细的 Swagger 使用指南请参考：[docs/swagger-guide.md](docs/swagger-guide.md)

### 模型提供商 API

服务提供了一套完整的模型提供商查询接口：

#### 1. 获取所有提供商列表

```http
GET /api/v1/providers
```

#### 2. 获取指定提供商详情

```http
GET /api/v1/providers/{providerId}
```

#### 3. 获取提供商的所有模型列表

```http
GET /api/v1/providers/{providerId}/models
```

#### 4. 获取指定模型详情

```http
GET /api/v1/providers/{providerId}/models/{modelId}
```

#### 5. 获取模型的参数规则

```http
GET /api/v1/providers/{providerId}/models/{modelId}/parameter-rules
```

### AI 对话 API

#### 发送对话消息

```http
POST /api/v1/chat
```

#### 中止对话

```http
POST /api/v1/abort
```

### 健康检查

```http
GET /api/v1/health
```

## 主要依赖

- **Firebase Genkit**: AI 模型集成
- **gorilla/mux**: HTTP 路由
- **lib/pq**: PostgreSQL 驱动
- **logrus**: 结构化日志
- **validator**: 参数验证
- **godotenv**: 环境变量管理
- **gopkg.in/yaml.v3**: YAML 配置解析
- **swaggo/swag**: OpenAPI/Swagger 文档生成
- **swaggo/http-swagger**: Swagger UI 集成

## 开发状态

项目当前处于初始化阶段，基础架构已搭建完成。

## 许可证

MIT
