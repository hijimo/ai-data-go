# TASK-4.3 快速参考

## 实现内容

### 新增函数

#### `createBailianPlugin()`

**位置**: `internal/genkit/client.go`

```go
func createBailianPlugin(apiKey string, genkitConfig *GenkitConfig) (*oai.OpenAI, error)
```

**功能**: 创建阿里云百炼插件实例

**参数**:

- `apiKey`: 百炼 API 密钥
- `genkitConfig`: Genkit 配置对象

**返回**:

- `*oai.OpenAI`: OpenAI 插件实例
- `error`: 错误信息

**特性**:

- 支持默认端点（北京地域）
- 支持自定义端点配置
- 使用 OpenAI 插件作为底层实现

### 修改的函数

#### `initializeProvider()`

**位置**: `internal/genkit/client.go`

**修改内容**: 更新 `bianlian` 分支，使用 `createBailianPlugin()` 函数

**修改前**:

```go
case "bianlian":
    bailianBaseURL := "https://dashscope.aliyuncs.com/compatible-mode/v1"
    if genkitConfig.BailianEndpoint != "" {
        bailianBaseURL = genkitConfig.BailianEndpoint
    }
    plugin := &oai.OpenAI{
        Opts: []option.RequestOption{
            option.WithAPIKey(tempConfig.APIKey),
            option.WithBaseURL(bailianBaseURL),
        },
    }
```

**修改后**:

```go
case "bianlian":
    plugin, err := createBailianPlugin(tempConfig.APIKey, genkitConfig)
    if err != nil {
        return nil, fmt.Errorf("创建百炼插件失败: %w", err)
    }
```

### 新增测试

#### `TestCreateBailianPlugin`

**位置**: `internal/genkit/client_test.go`

**测试用例**:

1. 使用默认端点
2. 使用自定义端点（新加坡地域）
3. 使用金融云端点

## 配置格式

### 数据库配置

```json
{
    "model": "qwen-plus",
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
}
```

### 字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 百炼模型名称 |
| `bailianEndpoint` | string | ❌ | API 端点（默认：北京地域） |
| `defaultTemperature` | float | ❌ | 默认温度参数 |
| `defaultMaxTokens` | int | ❌ | 默认最大 token 数 |

## 支持的端点

| 地域 | 端点 URL |
|------|----------|
| 北京（默认） | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| 新加坡 | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` |
| 金融云 | `https://dashscope-finance.aliyuncs.com/compatible-mode/v1` |

## 使用示例

### Go 代码

```go
// 创建客户端
client := genkit.NewClientWithRepo(configRepo)

// 调用百炼模型
result, err := client.Generate(
    ctx,
    "tenant-id",
    "qwen-plus",
    "你好",
    nil,
)
```

### HTTP API

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{id}/messages \
  -H "Authorization: Bearer TOKEN" \
  -d '{"message": "你好", "options": {"modelName": "qwen-plus"}}'
```

## 测试结果

```
✅ TestCreateBailianPlugin/使用默认端点
✅ TestCreateBailianPlugin/使用自定义端点
✅ TestCreateBailianPlugin/使用金融云端点
✅ TestInitializeProvider_Bianlian
✅ TestInitializeProvider_Bianlian_CustomEndpoint
```

## 文件清单

### 修改的文件

- `internal/genkit/client.go` - 添加 `createBailianPlugin()` 函数，更新 `initializeProvider()`
- `internal/genkit/client_test.go` - 添加 `TestCreateBailianPlugin` 测试

### 新增的文档

- `internal/genkit/TASK-4.3-IMPLEMENTATION-SUMMARY.md` - 详细实现总结
- `internal/genkit/BAILIAN_INTEGRATION_GUIDE.md` - 百炼集成使用指南
- `internal/genkit/TASK-4.3-QUICK-REFERENCE.md` - 本文档

## 验收标准

- ✅ 实现 `createBailianPlugin()` 函数
- ✅ 在 `InitializeProvider()` 中添加百炼分支
- ✅ 配置正确的模型名称格式
- ✅ 处理百炼特定的配置参数
- ✅ 添加错误处理
- ✅ 编写单元测试

## 下一步

继续执行 **TASK-4.4**: 测试百炼非流式调用
