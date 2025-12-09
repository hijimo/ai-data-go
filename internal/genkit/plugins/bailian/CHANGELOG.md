# Bailian 插件更新日志

## 2024-12-09 - 重构版本

### 重大变更

完全重构了 Bailian 插件实现，参照 Anthropic 插件的架构：

#### 新实现特点

1. **使用 OpenAI 兼容模式**
   - 复用 `compat_oai.OpenAICompatible` 基础设施
   - 简化代码实现，提高可维护性
   - 完全兼容 OpenAI API 规范

2. **简化配置**
   - 只需要 `API Key` 和 `Base URL` 两个配置项
   - 移除了复杂的 `BailianPlugin`、`Config` 等自定义结构
   - 支持环境变量配置（`BAILIAN_API_KEY`、`BAILIAN_BASE_URL`）

3. **标准化模型定义**
   - 预定义 5 个支持的模型
   - 明确标注每个模型的能力（多轮对话、工具调用、系统角色、多模态）
   - 使用 `bailian/` 作为提供商前缀

#### 文件结构

```
internal/genkit/plugins/bailian/
├── bailian.go           # 核心插件实现
├── bailian_test.go      # 单元测试
├── example_test.go      # 使用示例
├── README.md            # 使用文档
├── INTEGRATION.md       # 集成说明
└── CHANGELOG.md         # 更新日志（本文件）
```

#### 支持的模型

| 模型 ID | 模型名称 | 多轮对话 | 工具调用 | 系统角色 | 多模态 |
|---------|---------|---------|---------|---------|--------|
| qwen-turbo | 通义千问 Turbo | ✓ | ✓ | ✓ | ✗ |
| qwen-plus | 通义千问 Plus | ✓ | ✓ | ✓ | ✗ |
| qwen-max | 通义千问 Max | ✓ | ✓ | ✓ | ✗ |
| qwen3-max | 通义千问 3 Max | ✓ | ✓ | ✓ | ✗ |
| qwen-vl-plus | 通义千问 VL Plus | ✓ | ✗ | ✓ | ✓ |
| qwen-vl-max | 通义千问 VL Max | ✓ | ✗ | ✓ | ✓ |

#### Client 集成变更

**旧实现**（已移除）：
```go
// 使用自定义的 BailianPlugin
plugin, err := bailian.NewBailianPlugin(config)
```

**新实现**：
```go
// 使用标准的 Bailian 结构
plugin := &bailian.Bailian{
    Opts: []option.RequestOption{
        option.WithAPIKey(apiKey),
        option.WithBaseURL(baseURL),
    },
}
```

#### 模型名称变更

- **旧格式**: `openai/qwen-turbo`
- **新格式**: `bailian/qwen-turbo`

这样可以更清晰地区分不同的提供商。

### 向后兼容性

#### 数据库配置

现有的数据库配置**无需修改**，插件会自动处理：

- `model_provider`: 保持 `"bianlian"`
- `base_url`: 继续使用现有的端点 URL
- `api_key`: 保持不变
- `query_params`: 可选，用于存储额外配置（向后兼容）

#### 配置传递

**不使用环境变量**，所有配置通过代码传入：

```go
plugin := &bailian.Bailian{
    Opts: []option.RequestOption{
        option.WithAPIKey("sk-your-api-key"),
        option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
    },
}
```

**认证方式**：
- API Key 自动设置为 `Authorization: Bearer {apiKey}` header
- 符合百炼 API 的认证规范

### 迁移指南

#### 对于新项目

直接使用新的插件实现，参考 `README.md` 和 `INTEGRATION.md`。

#### 对于现有项目

1. **无需修改数据库配置**
   - 现有的 `model_configurations` 表数据继续有效
   - 插件会自动从 `base_url` 字段读取端点

2. **更新代码引用**（如果有直接使用插件）
   ```go
   // 旧代码
   import "genkit-ai-service/internal/genkit/plugins/bailian"
   plugin, err := bailian.NewBailianPlugin(config)
   
   // 新代码
   import "genkit-ai-service/internal/genkit/plugins/bailian"
   plugin := &bailian.Bailian{
       Opts: []option.RequestOption{
           option.WithAPIKey(apiKey),
           option.WithBaseURL(baseURL),
       },
   }
   ```

3. **测试验证**
   - 运行现有的集成测试
   - 验证 API 调用是否正常工作
   - 检查日志中的模型名称格式

### 性能改进

1. **减少依赖**
   - 移除了自定义的 HTTP 客户端实现
   - 复用 OpenAI SDK 的连接池和重试机制

2. **更好的错误处理**
   - 统一的错误格式
   - 详细的日志记录

3. **缓存优化**
   - Client 层的实例缓存继续有效
   - 减少重复初始化的开销

### 测试覆盖

新增测试：

- ✅ 插件名称测试
- ✅ 插件初始化测试
- ✅ 模型定义测试
- ✅ 模型获取测试
- ✅ 支持的模型配置验证
- ✅ 集成测试（需要真实 API Key）
- ✅ 流式响应测试

### 文档更新

新增文档：

- ✅ `README.md` - 完整的使用文档
- ✅ `INTEGRATION.md` - Client 集成说明
- ✅ `example_test.go` - 8 个使用示例
- ✅ `CHANGELOG.md` - 更新日志（本文件）

### 已知问题

无

### 下一步计划

1. 添加更多模型支持（根据阿里云百炼平台更新）
2. 优化错误处理和重试机制
3. 添加性能监控和指标收集
4. 支持更多高级功能（如函数调用、图像生成等）

### 贡献者

- 初始实现：参照 Anthropic 插件架构
- 测试和文档：完整的测试覆盖和使用文档

### 参考资料

- [阿里云百炼平台](https://bailian.console.aliyun.com/)
- [百炼 API 文档](https://help.aliyun.com/zh/model-studio/developer-reference/api-details)
- [Genkit 文档](https://firebase.google.com/docs/genkit)
- [OpenAI 兼容模式](https://help.aliyun.com/zh/model-studio/developer-reference/compatibility-of-openai-with-dashscope)
