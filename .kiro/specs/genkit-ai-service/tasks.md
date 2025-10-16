# 实现计划

- [x] 1. 初始化项目结构和基础配置
  - 创建标准的 Go 项目目录结构（cmd、internal、pkg）
  - 初始化 go.mod 并添加必要的依赖（Genkit SDK、PostgreSQL 驱动、日志库等）
  - 创建 .env.example 文件，包含所有必需的环境变量模板
  - _需求: 1.1, 1.2, 1.3_

- [x] 2. 实现配置管理模块
  - 创建 `internal/config/config.go`，定义配置结构体（ServerConfig、GenkitConfig、DatabaseConfig 等）
  - 实现从环境变量加载配置的函数
  - 实现配置验证逻辑，确保必需参数存在且有效
  - _需求: 1.5_

- [x] 3. 实现日志管理模块
  - 创建 `internal/logger/logger.go`，封装结构化日志功能
  - 支持不同日志级别（DEBUG、INFO、WARN、ERROR）
  - 支持 JSON 格式输出
  - 实现日志上下文信息注入（如 sessionId、requestId）
  - _需求: 5.3, 5.5_

- [x] 4. 实现统一响应和错误处理
  - 创建 `internal/model/response.go`，定义 ResponseData 和 ResponsePaginationData 泛型结构
  - 创建 `pkg/response/response.go`，实现响应构建辅助函数
  - 创建 `pkg/errors/errors.go`，定义错误码常量和自定义错误类型
  - 实现错误到响应的转换逻辑
  - _需求: 5.1, 5.2_

- [x] 5. 实现数据库连接管理
  - 创建 `internal/database/postgres.go`，实现 PostgreSQL 连接管理
  - 实现数据库连接初始化、连接池配置
  - 实现 Ping 方法用于健康检查
  - 实现优雅关闭数据库连接
  - _需求: 1.5_

- [x] 6. 实现 Genkit 客户端封装
  - 创建 `internal/genkit/client.go`，封装 Genkit SDK 初始化逻辑
  - 创建 `internal/genkit/config.go`，定义 Genkit 配置结构
  - 实现 Generate 方法，支持基本的文本生成
  - 实现参数映射，将 ChatOptions 转换为 Genkit 的生成选项
  - _需求: 2.2, 4.2_

- [x] 7. 实现上下文管理器
  - 创建 `internal/service/ai/context_manager.go`，实现会话上下文管理
  - 实现 CreateSession 方法，生成唯一的 sessionId 并创建可取消的 context
  - 实现 GetSession 和 CancelSession 方法
  - 实现会话超时和自动清理机制
  - _需求: 3.2, 3.5_

- [x] 8. 实现 AI 服务层
  - 创建 `internal/service/ai/service.go`，定义 AIService 接口
  - 创建 `internal/service/ai/genkit_service.go`，实现基于 Genkit 的 AI 服务
  - 实现 Chat 方法，处理对话请求并调用 Genkit 生成响应
  - 实现 AbortChat 方法，通过上下文管理器取消正在进行的对话
  - 集成日志记录，记录每次对话的关键信息
  - _需求: 2.1, 2.2, 2.6, 3.1, 3.2_

- [x] 9. 实现请求参数验证
  - 创建 `pkg/validator/validator.go`，封装参数验证逻辑
  - 使用 validator 库实现结构体标签验证
  - 实现自定义验证规则（如温度范围、token 数量限制）
  - 实现验证错误的格式化输出
  - _需求: 4.1, 4.4_

- [x] 10. 实现对话接口处理器
  - 创建 `internal/model/request.go`，定义 ChatRequest 和 ChatOptions 结构
  - 创建 `internal/model/ai.go`，定义 ChatResponse 和 Usage 结构
  - 创建 `internal/api/handler/chat.go`，实现对话接口处理器
  - 实现请求参数解析和验证
  - 调用 AI 服务处理对话请求
  - 构建标准响应并返回
  - _需求: 2.1, 2.2, 2.3, 2.4, 4.1, 4.2, 4.5_

- [x] 11. 实现中止接口处理器
  - 创建 `internal/api/handler/abort.go`，实现中止接口处理器
  - 实现请求参数验证（sessionId 必填）
  - 调用 AI 服务的 AbortChat 方法
  - 处理会话不存在或已完成的情况
  - _需求: 3.1, 3.2, 3.3, 3.4_

- [x] 12. 实现健康检查服务和处理器
  - 创建 `internal/service/health/service.go`，实现健康检查服务
  - 检查 Genkit 连接状态
  - 检查数据库连接状态（调用 Ping）
  - 收集服务版本和运行时间信息
  - 创建 `internal/api/handler/health.go`，实现健康检查接口处理器
  - _需求: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 13. 实现 HTTP 中间件
  - 创建 `internal/api/middleware/logger.go`，实现请求日志中间件
  - 创建 `internal/api/middleware/recovery.go`，实现 panic 恢复中间件
  - 创建 `internal/api/middleware/cors.go`，实现 CORS 中间件
  - 记录请求方法、路径、耗时、状态码等信息
  - _需求: 5.3_

- [x] 14. 实现路由配置
  - 创建 `internal/api/router.go`，配置所有 HTTP 路由
  - 注册中间件（日志、恢复、CORS）
  - 注册 API 路由（/api/v1/chat、/api/v1/chat/abort）
  - 注册健康检查路由（/health）
  - _需求: 2.1, 3.1, 7.1_

- [x] 15. 实现应用入口和服务启动
  - 创建 `cmd/server/main.go`，实现应用主函数
  - 加载配置
  - 初始化日志
  - 初始化数据库连接
  - 初始化 Genkit 客户端
  - 初始化服务和处理器
  - 启动 HTTP 服务器
  - 实现优雅关闭（监听系统信号，清理资源）
  - _需求: 1.1, 1.2, 1.4_

- [ ] 16. 创建项目文档
  - 创建 README.md，包含项目介绍、快速开始、API 文档
  - 说明环境变量配置
  - 提供 API 使用示例
  - 说明如何运行和部署项目
  - _需求: 1.1_
