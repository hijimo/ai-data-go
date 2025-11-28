# ModelName 字段验证规则更新

## 更新日期

2025-11-28

## 更新内容

### 验证规则改进

为 `ChatOptions.ModelName` 字段添加了更严格的验证规则：

**之前**：

```go
ModelName *string `json:"modelName,omitempty" validate:"omitempty,max=128"`
```

**现在**：

```go
ModelName *string `json:"modelName,omitempty" validate:"omitempty,min=1,max=128"`
```

### 改进说明

1. **添加最小长度验证（min=1）**
   - 防止客户端提交空字符串
   - 确保如果提供了模型名称，它必须是有效的非空字符串
   - 提高数据质量和系统健壮性

2. **保持最大长度限制（max=128）**
   - 与数据库 `model_configurations.model_name` 字段长度一致
   - 防止过长的输入导致数据库错误

3. **保持字段可选性（omitempty）**
   - 字段仍然是可选的
   - 客户端可以不提供 modelName，系统将使用默认模型
   - 完全向后兼容

## 验证行为

### 有效输入

- `null` 或不提供字段 ✅
- `"gpt-4"` ✅
- `"gpt-4-turbo"` ✅
- `"qwen_turbo"` ✅
- `"azure-gpt4"` ✅
- 任何 1-128 字符的非空字符串 ✅

### 无效输入

- `""` (空字符串) ❌ - 违反 min=1
- 超过 128 字符的字符串 ❌ - 违反 max=128

## 错误消息

当验证失败时，系统会返回友好的错误消息：

```json
{
  "code": 400,
  "message": "参数验证失败",
  "data": {
    "errors": [
      {
        "field": "modelName",
        "message": "modelName 长度必须至少为 1"
      }
    ]
  }
}
```

## 测试覆盖

添加了完整的验证测试套件（`internal/model/request_validation_test.go`），包括：

- ✅ 有效的模型名称（各种格式）
- ✅ 空指针（nil）
- ❌ 空字符串
- ❌ 超长字符串
- ✅ 其他字段的验证
- ✅ 组合验证

所有测试都通过 ✅

## 影响范围

### 不受影响

- 现有的 API 调用（不提供 modelName 或提供有效值）
- 向后兼容性完全保持

### 可能受影响

- 如果客户端尝试提交空字符串作为 modelName，将收到验证错误
- 这是预期的行为改进，有助于及早发现客户端错误

## 相关文件

- 实现文件：`internal/model/request.go`
- 测试文件：`internal/model/request_validation_test.go`
- 实现总结：`internal/model/TASK-5.1-IMPLEMENTATION-SUMMARY.md`

## 后续步骤

此改进为 TASK-5.2 做准备，下一步将：

1. 修改 AI Service 从 ChatOptions 中提取 ModelName
2. 将 ModelName 传递给 Genkit Client
3. 实现基于租户ID和模型名称的动态配置查询
