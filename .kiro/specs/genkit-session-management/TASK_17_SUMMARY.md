# Task 17: memoryCleanupFlow 实现总结

## 任务概述

实现了 `memoryCleanupFlow`，提供灵活的记忆清理功能，支持多种清理策略、软删除/硬删除模式和预览功能。

## 实现内容

### 1. 类型定义（types.go）

#### MemoryCleanupInput

- `SessionID`: 会话ID（可选，为空则清理租户下所有会话）
- `Strategy`: 清理策略（expired、low_quality、unused、all）
- `Mode`: 清理模式（soft软删除、hard硬删除）
- `BatchSize`: 批量处理大小（1-1000）
- `Execute`: 是否执行删除（false为预览模式）

#### MemoryCleanupOutput

- `SessionID`: 会话ID
- `Strategy`: 清理策略
- `Mode`: 清理模式
- `CleanedCount`: 清理数量
- `FreedSpace`: 释放空间（字节）
- `FreedTokens`: 释放Token数量
- `Details`: 清理详情列表
- `PreviewMode`: 是否为预览模式
- `CleanupTime`: 清理耗时（毫秒）
- `TotalProcessed`: 总处理数量

#### CleanupDetail

- `MemoryID`: 记忆ID
- `SessionID`: 会话ID
- `MemoryType`: 记忆类型
- `Reason`: 清理原因
- `Size`: 大小（字节）
- `TokenCount`: Token数量
- `Importance`: 重要性
- `CreatedAt`: 创建时间
- `LastAccess`: 最后访问时间

### 2. Flow 实现（memory.go）

#### 核心功能

1. **参数验证**
   - 验证会话ID格式（如果提供）
   - 验证清理策略（expired、low_quality、unused、all）
   - 验证清理模式（soft、hard）
   - 验证批量大小（1-1000）

2. **权限验证**
   - 验证JWT认证
   - 提取租户ID
   - 确保多租户隔离

3. **清理策略实现**
   - **expired**: 清理已过期的记忆（expires_at < 当前时间）
   - **low_quality**: 清理低质量记忆（重要性 < 0.3 且访问次数 < 2）
   - **unused**: 清理90天未访问的记忆
   - **all**: 清理所有记忆（谨慎使用）

4. **清理模式**
   - **soft**: 软删除，标记 is_deleted = true，保留数据
   - **hard**: 硬删除，物理删除数据

5. **预览模式**
   - Execute = false 时，只查询不执行删除
   - 返回详细的清理预览信息
   - 帮助用户在执行前评估影响

6. **详细信息收集**
   - 计算每条记忆的大小（内容 + 向量）
   - 统计释放的空间和Token数量
   - 记录清理原因
   - 提供完整的清理详情

7. **批量处理**
   - 支持批量软删除
   - 支持批量硬删除
   - 提高清理效率

### 3. Repository 扩展（genkit_memory_repository.go）

#### 新增接口方法

1. **GetExpiredMemories**: 获取过期记忆
2. **GetLowQualityMemories**: 获取低质量记忆
3. **GetUnusedMemories**: 获取长期未使用的记忆
4. **GetAllMemoriesForCleanup**: 获取所有待清理的记忆
5. **SoftDeleteBatch**: 批量软删除
6. **HardDeleteBatch**: 批量硬删除

#### MemoryCleanupFilters 结构

- `TenantID`: 租户ID（必需）
- `SessionID`: 会话ID（可选）
- `Strategy`: 清理策略
- `BatchSize`: 批量大小

#### 实现特点

1. **租户隔离**
   - 所有查询都包含 tenant_id 过滤
   - 确保不会跨租户清理数据

2. **会话级别控制**
   - 支持指定会话ID清理
   - 支持租户级别批量清理

3. **软删除支持**
   - 标记 is_deleted = true
   - 保留数据用于审计和恢复

4. **批量操作优化**
   - 批量软删除减少数据库操作
   - 批量硬删除提高性能

### 4. 辅助函数

#### validateMemoryCleanupInput

- 验证会话ID格式
- 验证清理策略有效性
- 验证清理模式有效性
- 验证批量大小范围

#### getCleanupReason

- 根据策略和记忆属性生成清理原因
- 提供详细的原因说明
- 包含具体的数值信息

## 清理策略详解

### 1. expired（过期清理）

- **条件**: expires_at IS NOT NULL AND expires_at < NOW()
- **适用场景**: 清理设置了过期时间且已过期的记忆
- **原因示例**: "已过期（过期时间：2024-01-15）"

### 2. low_quality（低质量清理）

- **条件**: importance < 0.3 AND access_count < 2
- **适用场景**: 清理重要性低且很少被访问的记忆
- **原因示例**: "低质量（重要性：0.25，访问次数：1）"

### 3. unused（未使用清理）

- **条件**: last_access_at < (NOW() - 90天) OR (last_access_at IS NULL AND created_at < (NOW() - 90天))
- **适用场景**: 清理长期未被访问的记忆
- **原因示例**: "长期未使用（120天未访问）"

### 4. all（全部清理）

- **条件**: 无额外条件（仅租户和会话过滤）
- **适用场景**: 清理指定范围内的所有记忆
- **原因示例**: "批量清理"
- **警告**: 谨慎使用，建议先使用预览模式

## 使用示例

### 预览模式（推荐先使用）

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "strategy": "low_quality",
  "mode": "soft",
  "batchSize": 100,
  "execute": false
}
```

### 执行软删除

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "strategy": "expired",
  "mode": "soft",
  "batchSize": 100,
  "execute": true
}
```

### 租户级别清理（不指定会话ID）

```json
{
  "strategy": "unused",
  "mode": "soft",
  "batchSize": 500,
  "execute": true
}
```

### 硬删除（永久删除）

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "strategy": "expired",
  "mode": "hard",
  "batchSize": 100,
  "execute": true
}
```

## 输出示例

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "strategy": "low_quality",
  "mode": "soft",
  "cleanedCount": 15,
  "freedSpace": 245760,
  "freedTokens": 3420,
  "details": [
    {
      "memoryId": "123e4567-e89b-12d3-a456-426614174000",
      "sessionId": "550e8400-e29b-41d4-a716-446655440000",
      "memoryType": "long_term",
      "reason": "低质量（重要性：0.25，访问次数：1）",
      "size": 16384,
      "tokenCount": 228,
      "importance": 0.25,
      "createdAt": "2024-01-15T10:30:00Z",
      "lastAccess": "2024-01-16T14:20:00Z"
    }
  ],
  "previewMode": false,
  "cleanupTime": 156,
  "totalProcessed": 15
}
```

## 安全特性

### 1. 多租户隔离

- 所有操作都限制在当前租户范围内
- 无法跨租户清理数据
- 租户ID从JWT中提取，不信任客户端输入

### 2. 权限验证

- 必须通过JWT认证
- 验证租户ID有效性
- 记录所有清理操作

### 3. 预览模式

- 默认不执行删除
- 先预览再执行
- 降低误操作风险

### 4. 软删除优先

- 推荐使用软删除模式
- 保留数据用于审计
- 支持数据恢复

### 5. 批量限制

- 单次最多处理1000条
- 防止一次性删除过多数据
- 可多次调用处理大量数据

## 性能优化

### 1. 批量操作

- 批量查询待清理记忆
- 批量执行删除操作
- 减少数据库往返次数

### 2. 索引利用

- 利用 tenant_id 索引
- 利用 session_id 索引
- 利用 expires_at 索引
- 利用 is_deleted 索引

### 3. 分批处理

- 支持设置批量大小
- 避免一次性处理过多数据
- 降低数据库压力

## 日志记录

### 关键日志点

1. **开始清理**: 记录租户ID、会话ID、策略、模式
2. **查询结果**: 记录找到的待清理记忆数量
3. **执行清理**: 记录实际清理的数量
4. **完成清理**: 记录释放的空间、Token、耗时

### 日志级别

- **Info**: 正常操作流程
- **Warn**: 未认证请求
- **Error**: 参数验证失败、查询失败、删除失败

## 监控指标

建议监控以下指标：

1. **清理频率**: 各策略的调用次数
2. **清理数量**: 每次清理的记忆数量
3. **释放空间**: 累计释放的存储空间
4. **执行时间**: 清理操作的耗时
5. **失败率**: 清理失败的比例

## 最佳实践

### 1. 使用预览模式

```
先执行 execute=false 查看影响 → 确认无误后执行 execute=true
```

### 2. 优先软删除

```
使用 mode=soft 保留数据 → 确认无需恢复后再使用 mode=hard
```

### 3. 分批清理

```
设置合理的 batchSize → 多次调用处理大量数据
```

### 4. 定期清理

```
设置定时任务 → 定期清理过期和低质量记忆 → 保持系统性能
```

### 5. 监控告警

```
监控清理指标 → 设置异常告警 → 及时发现问题
```

## 需求覆盖

本实现完全覆盖了需求14的所有验收标准：

✅ 1. 支持四种清理策略（expired、low_quality、unused、all）
✅ 2. expired策略清理已过期的记忆
✅ 3. low_quality策略清理重要性低于0.3且访问次数少于2的记忆
✅ 4. unused策略清理90天未访问的记忆
✅ 5. 支持两种清理模式（soft软删除、hard硬删除）
✅ 6. soft模式标记is_deleted为true但保留数据
✅ 7. hard模式物理删除数据
✅ 8. Execute为false时仅返回预览信息不执行删除
✅ 9. 应用租户隔离过滤
✅ 10. 分批处理清理操作
✅ 11. 统计清理数量和释放空间
✅ 12. 记录完整的清理日志

## 文件变更

1. **internal/genkit/flows/types.go**
   - 新增 MemoryCleanupInput 类型
   - 新增 MemoryCleanupOutput 类型
   - 新增 CleanupDetail 类型

2. **internal/genkit/flows/memory.go**
   - 实现 memoryCleanupFlow
   - 新增 validateMemoryCleanupInput 函数
   - 新增 getCleanupReason 函数

3. **internal/repository/genkit_memory_repository.go**
   - 新增 MemoryCleanupFilters 类型
   - 更新 GetExpiredMemories 方法签名和实现
   - 更新 GetLowQualityMemories 方法签名和实现
   - 更新 GetUnusedMemories 方法签名和实现
   - 新增 GetAllMemoriesForCleanup 方法
   - 新增 SoftDeleteBatch 方法
   - 新增 HardDeleteBatch 方法

## 测试建议

### 单元测试

1. 测试各种清理策略的过滤逻辑
2. 测试软删除和硬删除的执行
3. 测试预览模式不执行删除
4. 测试批量大小限制
5. 测试租户隔离

### 集成测试

1. 创建测试数据（不同状态的记忆）
2. 执行各种清理策略
3. 验证清理结果
4. 验证释放空间计算
5. 验证日志记录

### 性能测试

1. 测试大批量清理性能
2. 测试并发清理
3. 测试数据库压力

## 总结

成功实现了功能完整、安全可靠的记忆清理Flow，支持多种清理策略、预览模式和批量操作，完全满足需求14的所有验收标准。实现遵循了多租户隔离、权限验证、日志记录等最佳实践，为系统提供了强大的记忆管理能力。
