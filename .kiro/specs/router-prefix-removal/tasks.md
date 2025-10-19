# 实施计划

- [x] 1. 修改认证相关 Handler 的 @Router 注解
  - 修改 `internal/api/handler/auth_handler.go` 文件中所有包含 `/api/v1` 前缀的 `@Router` 注解
  - 移除 `/api/v1` 前缀，保留后续路径和 HTTP 方法
  - 涉及的路由：register, login, refresh, logout, change-password, unlock-account, me, verify-email, resend-verification
  - _需求: 1.1, 1.2, 1.3, 1.5_

- [x] 2. 修改审计日志 Handler 的 @Router 注解
  - 修改 `internal/api/handler/audit_handler.go` 文件中的 `@Router` 注解
  - 将 `/api/v1/audit/auth` 修改为 `/audit/auth`
  - _需求: 1.1, 1.2, 1.3, 1.5_

- [x] 3. 修改监控 Handler 的 @Router 注解
  - 修改 `internal/api/handler/monitoring_handler.go` 文件中所有包含 `/api/v1` 前缀的 `@Router` 注解
  - 移除 `/api/v1` 前缀，保留 `/monitoring/*` 路径
  - _需求: 1.1, 1.2, 1.3, 1.5_

- [x] 4. 扫描并修改其他可能遗漏的 Handler 文件
  - 使用 grep 搜索项目中所有包含 `@Router /api/v1` 的 Go 文件
  - 修改发现的任何其他包含 `/api/v1` 前缀的 `@Router` 注解
  - _需求: 1.1, 1.4_

- [ ] 5. 验证修改结果
- [ ] 5.1 运行编译验证
  - 执行 `go build ./...` 确保所有包编译通过
  - 执行 `go vet ./...` 检查代码问题
  - _需求: 2.1_

- [ ] 5.2 重新生成 Swagger 文档
  - 运行 `swag init` 命令重新生成 Swagger 文档
  - 检查生成的 `docs/swagger.json` 和 `docs/swagger.yaml` 文件
  - 确认路由路径格式正确（不包含 `/api/v1` 前缀）
  - _需求: 2.3_

- [ ]* 5.3 启动服务并进行功能测试
  - 启动服务验证能否正常运行
  - 访问 Swagger UI 确认文档显示正确
  - 使用 curl 或 Swagger UI 测试几个关键 API 端点
  - _需求: 2.2, 2.4_

- [ ] 6. 生成修改报告
  - 统计修改的文件数量
  - 列出所有被修改的 `@Router` 注解
  - 提供修改前后的对比
  - _需求: 3.1, 3.2, 3.3_
