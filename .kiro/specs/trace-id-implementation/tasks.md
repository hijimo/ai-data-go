# TraceID 全链路追踪实现任务列表

## 任务概述

本任务列表基于需求文档和设计文档，实现轻量级的全链路追踪功能。通过 TraceID 关联一次请求的所有日志和操作，便于问题排查和性能分析。

## 实现任务

- [x] 1. 实现 TraceID 生成和 Context 管理工具
  - 在 `internal/api/middleware/context.go` 中定义 Context 键和工具函数
  - 实现 TraceID 生成函数（格式：`trace-{timestamp}-{random}`）
  - 实现 `SetTraceID` 和 `GetTraceID` 函数用于 Context 操作
  - 使用对象池优化 TraceID 生成性能
  - _需求: 1.1, 1.2, 1.3, 1.4, 5.1, 5.2_

- [x] 2. 增强 Logger 中间件支持 TraceID
  - 修改 `internal/api/middleware/logger.go` 中的 Logger 中间件
  - 检查请求头 `X-Trace-ID`，如果存在则使用，否则生成新的 TraceID
  - 将 TraceID 注入到请求的 Context 中
  - 在响应头中设置 `X-Trace-ID`
  - 更新日志记录调用，使用包含 TraceID 的 Context
  - _需求: 1.1, 1.2, 1.3, 1.4, 1.5, 6.1, 6.2_

- [x] 3. 更新日志系统自动提取 TraceID
  - 修改 `internal/logger/logger.go` 中的 `extractContextFields` 函数
  - 添加从 Context 提取 TraceID 的逻辑
  - 确保 TraceID 作为结构化字段输出到日志
  - 处理 TraceID 不存在的情况（返回空字符串）
  - _需求: 2.1, 2.2, 2.3, 2.4, 6.4_

- [x] 4. 更新响应工具函数支持 TraceID 注入
  - 修改 `pkg/response/response.go` 中的响应工具函数
  - 新增 `SuccessWithContext`、`SuccessWithMessageContext` 等带 Context 参数的函数
  - 新增 `PaginationWithContext`、`PaginationWithMessageContext` 等分页响应函数
  - 新增 `ErrorWithContext` 等错误响应函数
  - 在新函数中自动从 Context 提取 TraceID 并注入到响应结构
  - 保留原有函数以确保向后兼容
  - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 6.1, 6.5_

- [x] 5. 更新核心 Handler 使用 Context 响应函数
  - 修改 `internal/api/handler/auth_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/tenant_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/user_handler.go` 中的所有响应调用
  - 将 `response.Success` 替换为 `response.SuccessWithContext`
  - 将 `response.Pagination` 替换为 `response.PaginationWithContext`
  - 传递 `r.Context()` 到响应函数
  - _需求: 4.1, 4.2, 4.3, 4.4, 6.1_

- [x] 6. 更新其他 Handler 使用 Context 响应函数
  - 修改 `internal/api/handler/session_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/message_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/chat.go` 中的所有响应调用
  - 修改 `internal/api/handler/provider_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/health.go` 中的所有响应调用
  - 修改 `internal/api/handler/audit_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/monitoring_handler.go` 中的所有响应调用
  - 修改 `internal/api/handler/abort.go` 中的所有响应调用
  - _需求: 4.1, 4.2, 4.3, 4.4, 6.1_

- [x] 7. 编写单元测试
  - 为 TraceID 生成函数编写测试（格式、唯一性、性能）
  - 为 Context 工具函数编写测试（SetTraceID、GetTraceID）
  - 为 Logger 中间件编写测试（TraceID 生成、提取、响应头设置）
  - 为日志系统 TraceID 提取编写测试
  - 为响应工具函数 TraceID 注入编写测试
  - _需求: 5.1, 5.2, 5.3, 5.4, 6.1_

- [x] 8. 编写集成测试
  - 编写端到端追踪测试（请求头 → Context → 日志 → 响应）
  - 编写并发测试（验证 TraceID 唯一性）
  - 编写性能基准测试（TraceID 生成、请求处理）
  - 验证客户端提供 TraceID 的场景
  - 验证客户端未提供 TraceID 的场景
  - _需求: 5.5, 6.1, 6.2, 6.3_

- [x] 9. 性能优化和验证
  - 使用 `sync.Pool` 优化 TraceID 生成的内存分配
  - 验证 TraceID 生成耗时 < 1ms
  - 验证 Context 传递和日志字段添加耗时 < 0.1ms
  - 验证 1000 QPS 下的额外内存开销 < 100KB/s
  - 使用性能分析工具（pprof）验证性能指标
  - _需求: 5.1, 5.2, 5.3, 5.4, 5.5_

## 实施说明

### 任务执行顺序

1. **基础设施层**（任务 1-4）：实现 TraceID 生成、Context 管理、日志集成和响应工具函数
2. **应用层**（任务 5-6）：更新所有 Handler 使用新的响应函数
3. **测试和优化**（任务 7-9）：编写测试并进行性能优化

### 向后兼容性

- 所有原有响应函数保持不变，确保现有代码继续工作
- 新增带 `Context` 后缀的响应函数，Handler 可以渐进式迁移
- TraceID 字段使用 `omitempty` 标签，为空时不影响客户端

### 可选任务说明

- 任务 7-9 标记为可选（后缀 `*`），专注于核心功能实现
- 测试任务虽然重要，但可以在核心功能完成后补充
- 性能优化可以在基础实现完成后进行渐进式改进

## 验收标准

完成所有必需任务后，系统应满足以下条件：

1. ✅ 每个 HTTP 请求都有唯一的 TraceID
2. ✅ 所有日志自动包含 TraceID
3. ✅ 所有 API 响应包含 TraceID（响应头和响应体）
4. ✅ 客户端提供的 TraceID 能够正确传递
5. ✅ TraceID 功能不影响现有功能（向后兼容）
6. ✅ 性能开销可忽略不计（< 1ms）

## 扩展性考虑

当前实现为未来升级到 OpenTelemetry 预留了扩展空间：

- TraceID 格式使用可识别的前缀 `trace-`
- Context 键使用标准化命名
- 追踪逻辑与业务逻辑分离
- 支持渐进式增强而不是全量替换
