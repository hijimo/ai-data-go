# 会话管理系统实施任务列表

- [x] 1. 创建数据模型和数据库迁移
  - 创建 `internal/model/session.go` 定义 ChatSession、ChatMessage、ChatSummary 实体
  - 创建 `internal/database/migrations/` 目录并添加数据库迁移脚本
  - 实现数据库表创建和索引创建的迁移代码
  - _需求: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12_

- [x] 2. 实现 Repository 层
  - 创建 `internal/repository/session_repository.go` 实现 SessionRepository 接口
  - 实现会话的 CRUD 操作（Create, GetByID, GetByUserID, Update, SoftDelete）
  - 实现会话搜索、过滤和分页查询功能
  - 实现消息计数更新和最后消息更新方法
  - _需求: 2, 3, 4, 7, 8, 9, 11_

- [x] 3. 实现消息 Repository
  - 创建 `internal/repository/message_repository.go` 实现 MessageRepository 接口
  - 实现消息的创建、查询和分页功能
  - 实现序列号生成和消息统计功能
  - 实现获取指定消息之后的消息列表功能
  - _需求: 5, 6_

- [x] 4. 实现摘要 Repository
  - 创建 `internal/repository/summary_repository.go` 实现 SummaryRepository 接口
  - 实现摘要的创建和查询功能
  - 实现获取会话最新摘要的方法
  - _需求: 12_

- [x] 5. 实现 SessionService 业务逻辑
  - 创建 `internal/service/session/session_service.go` 实现 SessionService 接口
  - 实现创建会话功能，包含参数验证和默认值设置
  - 实现获取会话详情功能，包含权限验证
  - 实现会话列表查询功能，支持分页、排序和过滤
  - 实现会话更新功能，支持部分字段更新
  - 实现会话软删除功能
  - 实现会话搜索功能
  - 实现置顶和归档功能
  - _需求: 2, 3, 4, 7, 8, 9, 11_

- [x] 6. 实现 MessageService 业务逻辑
  - 创建 `internal/service/session/message_service.go` 实现 MessageService 接口
  - 实现发送消息功能，包含用户消息保存和 AI 回复生成
  - 实现消息历史查询功能，支持分页
  - 实现获取单条消息详情功能
  - 实现中止消息生成功能
  - 集成现有的 AI 服务进行消息处理
  - 实现事务管理确保消息和会话状态一致性
  - _需求: 5, 6, 9_

- [x] 7. 实现 SummaryService 业务逻辑
  - 创建 `internal/service/session/summary_service.go` 实现 SummaryService 接口
  - 实现摘要生成逻辑，调用 AI 服务生成会话摘要
  - 实现判断是否需要生成摘要的逻辑（基于消息数量阈值）
  - 实现获取会话摘要功能
  - _需求: 12_

- [x] 8. 创建请求和响应模型
  - 在 `internal/model/request.go` 中添加会话管理相关的请求结构体
  - 添加 CreateSessionRequest、ListSessionsRequest、UpdateSessionRequest 等
  - 添加 SendMessageRequest、GetMessagesRequest、AbortMessageRequest 等
  - 在 `internal/model/response.go` 中添加响应结构体
  - 添加 SessionResponse、MessageResponse、MessageDetailResponse 等
  - 添加验证标签和 JSON 标签
  - _需求: 2, 3, 4, 5, 6, 7, 8, 11_

- [x] 9. 实现 SessionHandler
  - 创建 `internal/api/handler/session_handler.go` 实现 SessionHandler
  - 实现 CreateSession 处理器（POST /chat/sessions）
  - 实现 ListSessions 处理器（GET /chat/sessions）
  - 实现 GetSession 处理器（GET /chat/sessions/:id）
  - 实现 UpdateSession 处理器（PATCH /chat/sessions/:id）
  - 实现 DeleteSession 处理器（DELETE /chat/sessions/:id）
  - 实现 SearchSessions 处理器（GET /chat/sessions/search）
  - 实现 PinSession 处理器（POST /chat/sessions/:id/pin）
  - 实现 ArchiveSession 处理器（POST /chat/sessions/:id/archive）
  - 添加参数验证、错误处理和日志记录
  - 添加 Swagger 注释
  - _需求: 2, 3, 4, 7, 8, 11_

- [x] 10. 实现 MessageHandler
  - 创建 `internal/api/handler/message_handler.go` 实现 MessageHandler
  - 实现 SendMessage 处理器（POST /chat/sessions/:id/messages）
  - 实现 GetMessages 处理器（GET /chat/sessions/:id/messages）
  - 实现 GetMessageByID 处理器（GET /chat/messages/:id）
  - 实现 AbortMessage 处理器（POST /chat/messages/:id/abort）
  - 添加参数验证、错误处理和日志记录
  - 添加 Swagger 注释
  - _需求: 5, 6_

- [x] 11. 调整现有 Chat 接口
  - 修改 `internal/model/request.go` 中的 ChatRequest，将 SessionID 改为 MessageID
  - 修改 `internal/model/request.go` 中的 AbortRequest，将 SessionID 改为 MessageID
  - 修改 `internal/api/handler/chat.go` 中的处理逻辑，适配 MessageID 参数
  - 修改 `internal/api/handler/abort.go` 中的处理逻辑，适配 MessageID 参数
  - 更新相关的 Swagger 注释
  - 更新日志记录中的字段名称
  - _需求: 1_

- [x] 12. 配置路由
  - 在 `internal/api/routes/` 中添加会话管理相关路由
  - 注册 SessionHandler 的所有路由
  - 注册 MessageHandler 的所有路由
  - 配置路由中间件（如身份验证、日志记录）
  - 更新路由文档
  - _需求: 2, 3, 4, 5, 6, 7, 8, 11_

- [x] 13. 添加错误码定义
  - 在 `pkg/errors/errors.go` 中添加会话管理相关的错误码
  - 添加 CodeSessionNotFound、CodeSessionAccessDenied 等错误码
  - 添加 CodeMessageNotFound、CodeMessageSendFailed 等错误码
  - 添加对应的错误消息常量
  - 添加错误构造函数
  - _需求: 4, 6, 7, 8, 9_

- [x] 14. 实现用户上下文中间件
  - 创建 `internal/api/middleware/user_context.go` 实现用户身份提取
  - 从请求头或 Token 中提取 UserID
  - 将 UserID 存入请求上下文
  - 添加错误处理（未认证用户）
  - _需求: 9_

- [x] 15. 添加配置项
  - 在 `internal/config/config.go` 中添加会话管理配置
  - 添加摘要生成阈值配置
  - 添加默认分页大小和最大分页大小配置
  - 添加会话标题最大长度配置
  - _需求: 12_

- [x] 16. 数据库初始化和迁移
  - 在 `cmd/migrate/main.go` 或启动流程中添加数据库迁移逻辑
  - 注册所有模型到 AutoMigrate
  - 执行数据库迁移创建表和索引
  - 添加迁移日志记录
  - _需求: 2, 5, 12_

- [x] 17. 依赖注入和服务初始化
  - 在主程序中初始化 Repository 实例
  - 初始化 Service 实例并注入依赖
  - 初始化 Handler 实例并注入依赖
  - 配置服务生命周期管理
  - _需求: 2, 3, 4, 5, 6, 7, 8, 9, 10, 11_

- [ ] 18. 集成测试
- [ ]* 18.1 编写会话创建和查询的集成测试
  - 测试创建会话的完整流程
  - 测试会话列表查询和分页
  - 测试会话详情查询和权限验证
  - _需求: 2, 3, 4, 9_

- [ ]* 18.2 编写消息发送和查询的集成测试
  - 测试发送消息的完整流程
  - 测试消息历史查询和分页
  - 测试会话状态更新（消息计数、最后消息）
  - _需求: 5, 6_

- [ ]* 18.3 编写会话更新和删除的集成测试
  - 测试会话更新功能
  - 测试会话软删除功能
  - 测试置顶和归档功能
  - _需求: 7, 8_

- [ ]* 18.4 编写多用户隔离的集成测试
  - 测试不同用户的数据隔离
  - 测试跨用户访问的权限验证
  - _需求: 9_

- [ ] 19. 更新 API 文档
  - 更新 Swagger 文档，添加所有新接口
  - 更新 README 文档，说明会话管理功能
  - 添加接口使用示例
  - 更新错误码文档
  - _需求: 2, 3, 4, 5, 6, 7, 8, 11_
