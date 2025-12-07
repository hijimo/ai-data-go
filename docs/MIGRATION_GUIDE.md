# Genkit 多模型支持迁移指南

## 概述

本指南帮助您从单提供商（仅支持 Google AI）平滑迁移到多提供商架构（支持 Google AI、OpenAI、Azure OpenAI、阿里云百炼等）。

**迁移特点**：

- ✅ **完全向后兼容** - 现有 API 接口保持不变
- ✅ **零停机迁移** - 无需停止服务即可完成迁移
- ✅ **渐进式升级** - 可以逐步添加新提供商
- ✅ **数据保留** - 所有现有数据和配置保持不变

## 向后兼容性说明

### 1. API 接口兼容性

**现有接口完全保持不变**，无需修改客户端代码：

```json
// ✅ 原有的 API 调用方式继续有效
POST /api/v1/chat/sessions/{sessionId}/messages
{
  "message": "你好",
  "options": {
    "temperature": 0.7,
    "maxTokens": 2000
  }
}
```

**新增可选参数**，向后兼容：

```json
// ✅ 新增 modelName 参数（可选）
POST /api/v1/chat/sessions/{sessionId}/messages
{
  "message": "你好",
  "options": {
    "modelName": "gpt-4",  // 新增：指定模型名称
    "temperature": 0.7,
    "maxTokens": 2000
  }
}
```

### 2. 配置兼容性

**环境变量配置继续有效**：

```bash
# ✅ 原有的环境变量配置继续工作
GOOGLE_API_KEY=your-google-api-key
```

**新增数据库配置**（可选）：

- 如果不配置数据库中的模型，系统将使用环境变量中的 Google AI 配置
- 配置数据库后，可以支持多个模型和提供商

### 3. 响应格式兼容性

**响应格式完全一致**：

- 非流式响应格式不变
- 流式响应（SSE）格式不变
- Token 统计格式不变
- 错误响应格式不变

### 4. 默认行为

**未指定模型时的行为**：

- 如果请求中未指定 `modelName`，系统将使用会话的默认模型
- 如果会话未配置默认模型，系统将使用租户的默认模型配置
- 如果租户未配置任何模型，系统将回退到环境变量中的 Google AI 配置

## 迁移前准备

### 1. 检查当前环境

```bash
# 检查 Go 版本（需要 1.21+）
go version

# 检查数据库版本（需要 PostgreSQL 13+）
psql --version

# 检查当前配置
echo $GOOGLE_API_KEY
```

### 2. 备份数据

```bash
# 备份数据库
pg_dump -h localhost -U postgres -d your_database > backup_$(date +%Y%m%d).sql

# 备份配置文件
cp .env .env.backup
```

### 3. 准备 API 密钥

如果计划使用新的提供商，请提前准备好 API 密钥：

- **OpenAI**: <https://platform.openai.com/api-keys>
- **Azure OpenAI**: <https://portal.azure.com>
- **阿里云百炼**: <https://bailian.console.aliyun.com/>
- **Anthropic**: <https://console.anthropic.com/>

## 迁移步骤

### 阶段 1：数据库迁移（必需）

#### 步骤 1.1：执行数据库迁移

数据库迁移会自动创建 `model_configurations` 表。

**方法 A：应用启动时自动执行**（推荐）

```bash
# 启动应用，迁移会自动执行
go run cmd/server/main.go
```

**方法 B：手动执行迁移脚本**

```bash
# 执行迁移
go run scripts/init_migration.go
```

#### 步骤 1.2：验证迁移

```sql
-- 检查表是否创建成功
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name = 'model_configurations';

-- 检查表结构
\d model_configurations
```

**预期结果**：

```
                Table "public.model_configurations"
     Column      |            Type             | Nullable |      Default
-----------------+-----------------------------+----------+-------------------
 id              | uuid                        | not null | gen_random_uuid()
 tenant_id       | uuid                        | not null |
 name            | character varying(255)      | not null |
 model           | character varying(255)      | not null |
 model_provider  | character varying(100)      | not null |
 api_key         | text                        | not null |
 base_url        | character varying(500)      |          |
 query_params    | text                        |          |
 is_enabled      | boolean                     | not null | true
 created_at      | timestamp without time zone | not null | CURRENT_TIMESTAMP
 updated_at      | timestamp without time zone | not null | CURRENT_TIMESTAMP
 is_deleted      | boolean                     | not null | false
```

### 阶段 2：配置迁移（可选）

此阶段是可选的。如果不执行，系统将继续使用环境变量中的 Google AI 配置。

#### 步骤 2.1：为现有租户创建 Google AI 配置

如果您希望将现有的 Google AI 配置迁移到数据库：

```bash
# 使用 API 创建配置
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gemini Pro",
    "model": "gemini-1.5-pro",
    "modelProvider": "googlegenai",
    "apiKey": "'"$GOOGLE_API_KEY"'"
  }'
```

**或者使用 SQL 直接插入**：

```sql
-- 为所有现有租户创建 Google AI 配置
INSERT INTO model_configurations (
    tenant_id, 
    name, 
    model, 
    model_provider, 
    api_key
)
SELECT 
    id as tenant_id,
    'Gemini Pro' as name,
    'gemini-1.5-pro' as model,
    'googlegenai' as model_provider,
    'YOUR_GOOGLE_API_KEY' as api_key
FROM tenants
WHERE is_deleted = false;
```

#### 步骤 2.2：验证配置

```bash
# 查询配置
curl -X GET "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_TENANT_ADMIN_TOKEN"
```

### 阶段 3：测试验证（必需）

#### 步骤 3.1：测试现有功能

**测试非流式调用**：

```bash
# 使用原有的 API（不指定 modelName）
curl -X POST "http://localhost:8080/api/v1/chat/sessions/{sessionId}/messages" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下自己",
    "options": {
      "temperature": 0.7
    }
  }'
```

**预期结果**：

- ✅ 响应格式与之前完全一致
- ✅ 功能正常工作
- ✅ Token 统计正确

**测试流式调用**：

```bash
# 测试流式接口
curl -X POST "http://localhost:8080/api/v1/chat/sessions/{sessionId}/messages/stream" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "写一首关于春天的诗",
    "options": {
      "temperature": 0.8
    }
  }'
```

**预期结果**：

- ✅ SSE 格式与之前一致
- ✅ 实时返回内容
- ✅ 最终返回 Token 统计

#### 步骤 3.2：测试新功能（如果已配置）

**测试指定模型名称**：

```bash
# 使用新的 modelName 参数
curl -X POST "http://localhost:8080/api/v1/chat/sessions/{sessionId}/messages" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好",
    "options": {
      "modelName": "Gemini Pro",
      "temperature": 0.7
    }
  }'
```

### 阶段 4：添加新提供商（可选）

此阶段完全可选，可以在任何时候执行。

#### 步骤 4.1：添加 OpenAI 配置

```bash
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-your-openai-api-key"
  }'
```

#### 步骤 4.2：添加 Azure OpenAI 配置

```bash
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Azure GPT-4",
    "model": "gpt-4",
    "modelProvider": "azureopenai",
    "apiKey": "your-azure-api-key",
    "queryParams": "{\"azureEndpoint\":\"https://your-resource.openai.azure.com\",\"azureDeployment\":\"gpt-4\",\"azureApiVersion\":\"2024-02-15-preview\"}"
  }'
```

#### 步骤 4.3：添加阿里云百炼配置

```bash
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "通义千问",
    "model": "qwen-turbo",
    "modelProvider": "bianlian",
    "apiKey": "sk-your-dashscope-api-key",
    "queryParams": "{\"bailianEndpoint\":\"https://dashscope.aliyuncs.com/compatible-mode/v1\"}"
  }'
```

#### 步骤 4.4：测试新提供商

```bash
# 测试 OpenAI
curl -X POST "http://localhost:8080/api/v1/chat/sessions/{sessionId}/messages" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好",
    "options": {
      "modelName": "GPT-4"
    }
  }'
```

## 配置变更说明

### 环境变量变更

**保留的环境变量**（继续有效）：

```bash
# Google AI 配置（作为回退选项）
GOOGLE_API_KEY=your-google-api-key
```

**新增的环境变量**（可选）：

```bash
# 数据库连接（如果使用新的配置管理）
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# 其他提供商的 API 密钥（可选，也可以通过 API 配置）
OPENAI_API_KEY=sk-your-openai-api-key
AZURE_OPENAI_KEY=your-azure-key
BAILIAN_API_KEY=sk-your-bailian-key
```

### 配置文件变更

**无需修改配置文件**。所有新配置都通过数据库管理。

### 代码变更

**无需修改客户端代码**。所有现有的 API 调用方式继续有效。

## 迁移检查清单

### 迁移前检查

- [ ] 已备份数据库
- [ ] 已备份配置文件
- [ ] 已检查 Go 版本（1.21+）
- [ ] 已检查 PostgreSQL 版本（13+）
- [ ] 已准备好新提供商的 API 密钥（如需要）
- [ ] 已通知相关团队成员

### 迁移执行检查

- [ ] 数据库迁移成功执行
- [ ] `model_configurations` 表创建成功
- [ ] 表结构验证通过
- [ ] 索引和约束创建成功

### 迁移后验证

- [ ] 现有 API 接口正常工作
- [ ] 非流式调用功能正常
- [ ] 流式调用功能正常
- [ ] 响应格式与之前一致
- [ ] Token 统计正确
- [ ] 错误处理正常
- [ ] 日志记录正常

### 新功能验证（如果已配置）

- [ ] 可以创建模型配置
- [ ] 可以查询模型配置
- [ ] 可以更新模型配置
- [ ] 可以删除模型配置
- [ ] 可以通过 modelName 指定模型
- [ ] 多提供商切换正常
- [ ] 租户隔离正常

## 回滚方案

如果迁移过程中遇到问题，可以按以下步骤回滚：

### 方案 1：保留数据库表，回退代码

如果只是代码层面的问题：

```bash
# 1. 停止服务
pkill -f "cmd/server/main.go"

# 2. 回退到之前的代码版本
git checkout <previous-commit>

# 3. 重新启动服务
go run cmd/server/main.go
```

**说明**：

- 数据库表保留，不影响系统运行
- 系统将继续使用环境变量中的配置
- 所有现有功能继续正常工作

### 方案 2：完全回滚（包括数据库）

如果需要完全回滚：

```bash
# 1. 停止服务
pkill -f "cmd/server/main.go"

# 2. 回滚数据库
psql -h localhost -U postgres -d your_database < backup_YYYYMMDD.sql

# 3. 回退代码
git checkout <previous-commit>

# 4. 重新启动服务
go run cmd/server/main.go
```

### 方案 3：仅删除新表

如果只想删除新创建的表：

```sql
-- 删除 model_configurations 表
DROP TABLE IF EXISTS model_configurations CASCADE;
```

**注意**：删除表后，系统将自动回退到使用环境变量配置。

## 常见问题

### Q1: 迁移后现有功能是否会受影响？

**A**: 不会。迁移完全向后兼容，所有现有 API 接口和功能保持不变。

### Q2: 必须配置数据库中的模型吗？

**A**: 不必须。如果不配置，系统将继续使用环境变量中的 Google AI 配置。

### Q3: 可以只迁移部分租户吗？

**A**: 可以。您可以为部分租户配置数据库中的模型，其他租户继续使用环境变量配置。

### Q4: 迁移需要停机吗？

**A**: 不需要。数据库迁移可以在服务运行时执行，不影响现有功能。

### Q5: 如何验证迁移是否成功？

**A**: 执行"迁移检查清单"中的所有验证步骤，确保所有功能正常工作。

### Q6: 迁移失败如何处理？

**A**: 按照"回滚方案"中的步骤进行回滚，然后检查错误日志，解决问题后重新迁移。

### Q7: 环境变量配置还需要保留吗？

**A**: 建议保留。环境变量配置作为回退选项，当数据库配置不可用时使用。

### Q8: 可以同时使用环境变量和数据库配置吗？

**A**: 可以。数据库配置优先级更高，如果数据库中没有配置，系统将使用环境变量。

### Q9: 如何为现有租户批量创建配置？

**A**: 使用"阶段 2：配置迁移"中提供的 SQL 脚本批量创建。

### Q10: 新提供商的配置何时生效？

**A**: 立即生效。配置创建后，下次 API 调用时会自动使用新配置。

## 性能影响

### 预期性能变化

- **首次调用延迟**: 增加 10-50ms（用于查询配置和初始化插件）
- **后续调用延迟**: 无影响（使用缓存）
- **内存使用**: 每个模型配置增加约 1-2MB
- **数据库查询**: 每个新模型首次使用时查询一次

### 性能优化建议

1. **预热缓存**: 应用启动后，预先调用常用模型
2. **监控延迟**: 使用日志中的 `durationMs` 字段监控性能
3. **限制模型数量**: 每个租户建议配置不超过 10 个模型
4. **定期清理**: 删除不再使用的模型配置

## 最佳实践

### 1. 渐进式迁移

```
第1周：执行数据库迁移，验证现有功能
第2周：为测试租户配置新提供商
第3周：逐步为生产租户配置新提供商
第4周：监控和优化
```

### 2. 配置管理

- ✅ 为每个环境（开发、测试、生产）使用独立的 API 密钥
- ✅ 定期轮换 API 密钥
- ✅ 使用密钥管理服务（如 AWS Secrets Manager）
- ✅ 在配置中添加描述信息，便于管理

### 3. 监控和告警

- ✅ 监控 API 调用成功率
- ✅ 监控 API 调用延迟
- ✅ 监控 Token 使用量
- ✅ 设置告警阈值

### 4. 成本控制

- ✅ 为不同场景选择合适的模型
- ✅ 设置 Token 使用限制
- ✅ 定期审查 Token 使用情况
- ✅ 优化提示词以减少 Token 消耗

## 技术支持

### 获取帮助

- **文档**: 查看 [多提供商使用指南](./MULTI_PROVIDER_GUIDE.md)
- **配置**: 查看 [配置指南](./CONFIGURATION_GUIDE.md)
- **故障排查**: 查看 [故障排查指南](./TROUBLESHOOTING.md)
- **API 文档**: 访问 `/swagger/index.html`

### 日志分析

迁移过程中的关键日志：

```json
// 数据库迁移日志
{
  "level": "info",
  "message": "执行迁移: 创建模型配置表",
  "migration": "model_configurations"
}

// 模型初始化日志
{
  "level": "info",
  "message": "初始化模型配置",
  "tenantId": "xxx",
  "modelName": "gpt-4",
  "provider": "openai"
}

// API 调用日志
{
  "level": "info",
  "message": "生成内容成功",
  "tenantId": "xxx",
  "modelName": "gpt-4",
  "durationMs": 1500
}
```

## 下一步

迁移完成后，您可以：

1. **探索新功能**: 尝试不同的模型提供商
2. **优化配置**: 根据实际使用情况调整模型配置
3. **监控性能**: 使用日志和监控工具分析性能
4. **成本优化**: 根据使用情况优化模型选择

## 相关文档

- [多提供商使用指南](./MULTI_PROVIDER_GUIDE.md) - 详细的使用说明
- [配置指南](./CONFIGURATION_GUIDE.md) - 配置参数说明
- [故障排查指南](./TROUBLESHOOTING.md) - 常见问题解决
- [API 文档](../docs/swagger.yaml) - API 接口文档

## 版本历史

- **v1.0.0** (2025-12-01): 初始版本，支持多提供商架构
- **v0.9.0** (2025-11-01): 单提供商版本（仅 Google AI）

---

**最后更新**: 2025-12-07  
**维护者**: AI Platform Team
