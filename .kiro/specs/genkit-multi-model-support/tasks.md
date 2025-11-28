# 任务列表：Genkit 多模型支持

## 阶段 1：准备工作

### TASK-1.1: 调研 Genkit 插件支持

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: 无

**描述**:
调研 Genkit 对不同模型提供商的插件支持情况。

**验收标准**:

- [x] 调研 Google AI (Gemini) 插件
- [x] 调研 Azure OpenAI 插件支持情况
- [x] 调研阿里云百炼插件支持情况
- [x] 确定每个提供商的集成方案
- [x] 记录调研结果和决策

**输出文档**:

- `docs/genkit-plugin-research.md`

---

## 阶段 2：核心实现

### TASK-2.1: 扩展 Genkit 配置结构

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-1.1

**描述**:
扩展 Genkit 配置结构，支持从 ModelConfiguration 解析配置。

**验收标准**:

- [x] 定义 GenkitConfig 结构体（用于解析 model_configurations.configuration）
- [x] 支持 Azure 特定配置字段
- [x] 支持百炼特定配置字段
- [x] 添加配置验证方法
- [x] 编写单元测试

**实现文件**:

- `internal/genkit/config.go`
- `internal/genkit/config_test.go`

---

### TASK-2.2: 重构 Genkit Client 支持动态配置

**优先级**: P0  
**预计工时**: 5 小时  
**依赖**: TASK-2.1

**描述**:
重构 Genkit Client，支持从数据库动态获取配置并初始化。

**验收标准**:

- [x] 修改 `client` 结构体，注入 ModelConfigurationRepository
- [x] 实现 `getOrInitGenkit()` 方法（根据租户ID和模型名称）
- [x] 实现 Genkit 实例缓存机制（key: tenantID_modelName）
- [x] 添加并发安全的读写锁
- [x] 实现配置解析逻辑
- [x] 实现插件动态创建逻辑
- [x] 保持向后兼容性
- [x] 编写单元测试

**实现文件**:

- `internal/genkit/client.go`
- `internal/genkit/client_test.go`

---

### TASK-2.3: 扩展 Generate 方法支持租户和模型参数

**优先级**: P0  
**预计工时**: 3 小时  
**依赖**: TASK-2.2

**描述**:
修改 Generate 和 GenerateStream 方法，支持租户ID和模型名称参数。

**验收标准**:

- [x] 修改 `Generate()` 方法签名，添加 tenantID 和 modelName 参数
- [x] 修改 `GenerateStream()` 方法签名，添加 tenantID 和 modelName 参数
- [x] 调用 `getOrInitGenkit()` 获取配置和实例
- [x] 添加配置不存在的错误处理
- [x] 添加模型禁用的错误处理
- [x] 编写单元测试

**实现文件**:

- `internal/genkit/client.go`
- `internal/genkit/client_test.go`

---

## 阶段 3：Azure OpenAI 集成

### TASK-3.1: 调研 Genkit Azure OpenAI 插件

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-2.3

**描述**:
调研 Genkit 是否有官方 Azure OpenAI 插件，确定集成方案。

**验收标准**:

- [x] 检查 Genkit 官方仓库是否有 Azure OpenAI 插件
- [x] 检查是否可以使用 OpenAI 插件 + 自定义 BaseURL
- [x] 确定最终的集成方案
- [x] 记录调研结果和决策

**输出文档**:

- `docs/azure-openai-integration-research.md`

**最终决策**:

✅ **采用方案 A：使用 OpenAI 插件 + 自定义 BaseURL**

**决策理由**：

- 技术可行性已验证（代码实现完成，单元测试通过）
- 无需开发自定义插件，维护成本低
- 与 Genkit 生态系统完全兼容
- 实施风险低，实现简洁

**核心实现**：

```go
plugin := &oai.OpenAI{
    Opts: []option.RequestOption{
        option.WithAPIKey(apiKey),
        option.WithBaseURL(fmt.Sprintf("%s/openai/deployments/%s", 
            azureEndpoint, azureDeployment)),
    },
}
```

---

### TASK-3.2: 实现 Azure OpenAI 插件集成

**优先级**: P0  
**预计工时**: 4 小时  
**依赖**: TASK-3.1

**描述**:
根据调研结果，实现 Azure OpenAI 插件集成。

**验收标准**:

- [x] 实现 `createAzurePlugin()` 函数
- [x] 在 `InitializeProvider()` 中添加 Azure 分支
- [x] 配置正确的模型名称格式
- [x] 处理 Azure 特定的配置参数
- [x] 添加错误处理
- [x] 编写单元测试

**实现文件**:

- `internal/genkit/client.go`
- `internal/genkit/plugins/azure/azure.go` (如果需要自定义插件)

---

### TASK-3.3: 测试 Azure OpenAI 非流式调用

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-3.2

**描述**:
测试 Azure OpenAI 的非流式调用功能。

**验收标准**:

- [x] 编写集成测试用例
- [x] 测试基本的文本生成
- [x] 测试参数传递（temperature, maxTokens）
- [x] 测试 Token 统计
- [x] 测试错误处理
- [x] 验证响应格式正确

**实现文件**:

- `internal/genkit/azure_integration_test.go`
- `test/test_azure_openai.sh`

---

### TASK-3.4: 测试 Azure OpenAI 流式调用

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-3.3

**描述**:
测试 Azure OpenAI 的流式调用功能。

**验收标准**:

- [x] 编写流式调用测试用例
- [x] 测试流式响应接收
- [x] 测试流式响应完整性
- [x] 测试流式中断处理
- [x] 测试 SSE 格式转换
- [x] 验证最终 Token 统计

**实现文件**:

- `internal/genkit/azure_stream_test.go`
- `test/test_azure_stream.sh`

**实现总结**:

- 创建了完整的流式调用测试套件，包含 8 个测试用例
- 测试覆盖：流式响应接收、完整性验证、中断处理、参数传递、格式验证
- 错误处理测试：配置不存在、租户ID无效、模型已禁用
- 性能测试：首字节时间（TTFB）、总耗时、并发调用
- 创建了便捷的测试脚本 `test/test_azure_stream.sh`
- 详细文档：`internal/genkit/TASK-3.4-IMPLEMENTATION-SUMMARY.md`

---

## 阶段 4：百炼集成

### TASK-4.1: 调研百炼 API 和集成方案

**优先级**: P1  
**预计工时**: 3 小时  
**依赖**: TASK-3.4

**描述**:
调研阿里云百炼 API，确定集成方案。

**验收标准**:

- [x] 研究百炼 API 文档
- [x] 检查是否支持 OpenAI 兼容接口
- [x] 确定是否需要自定义插件
- [x] 设计请求/响应格式转换方案
- [x] 记录调研结果和决策

**输出文档**:

- `docs/bailian-integration-research.md`

---

### TASK-4.2: 实现百炼自定义插件

**优先级**: P1  
**预计工时**: 6 小时  
**依赖**: TASK-4.1

**描述**:
实现百炼的 Genkit 自定义插件。

**验收标准**:

- [x] 创建 `BailianPlugin` 结构体
- [x] 实现 `Init()` 方法，注册模型
- [x] 实现 `generate()` 方法，处理非流式调用（通过委托给 OpenAI 插件实现）
- [x] 实现 `generateStream()` 方法，处理流式调用（通过委托给 OpenAI 插件实现）
- [x] 实现请求格式转换（百炼完全兼容 OpenAI 格式，无需转换）
- [x] 实现响应格式转换（百炼完全兼容 OpenAI 格式，无需转换）
- [x] 添加错误处理（由 OpenAI 插件处理，插件层添加了配置验证）
- [x] 编写单元测试

**实现文件**:

- `internal/genkit/plugins/bailian/bailian.go`
- `internal/genkit/plugins/bailian/bailian_test.go`
- `internal/genkit/plugins/bailian/types.go`

---

### TASK-4.3: 集成百炼插件到 Client

**优先级**: P1  
**预计工时**: 2 小时  
**依赖**: TASK-4.2

**描述**:
将百炼插件集成到 Genkit Client。

**验收标准**:

- [x] 实现 `createBailianPlugin()` 函数
- [x] 在 `InitializeProvider()` 中添加百炼分支
- [ ] 配置正确的模型名称格式
- [x] 处理百炼特定的配置参数
- [ ] 添加错误处理
- [ ] 编写单元测试

**实现文件**:

- `internal/genkit/client.go`

---

### TASK-4.4: 测试百炼非流式调用

**优先级**: P1  
**预计工时**: 2 小时  
**依赖**: TASK-4.3

**描述**:
测试百炼的非流式调用功能。

**验收标准**:

- [ ] 编写集成测试用例
- [ ] 测试基本的文本生成
- [ ] 测试中文处理能力
- [ ] 测试参数传递
- [ ] 测试 Token 统计
- [ ] 测试错误处理

**实现文件**:

- `internal/genkit/bailian_integration_test.go`
- `test/test_bailian.sh`

---

### TASK-4.5: 测试百炼流式调用

**优先级**: P1  
**预计工时**: 2 小时  
**依赖**: TASK-4.4

**描述**:
测试百炼的流式调用功能。

**验收标准**:

- [ ] 编写流式调用测试用例
- [ ] 测试流式响应接收
- [ ] 测试中文流式输出
- [ ] 测试流式响应完整性
- [ ] 测试流式中断处理
- [ ] 验证 SSE 格式转换

**实现文件**:

- `internal/genkit/bailian_stream_test.go`
- `test/test_bailian_stream.sh`

---

## 阶段 5：API 层集成

### TASK-5.1: 扩展 ChatOptions 支持模型名称

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-2.4

**描述**:
在 API 请求模型中添加模型名称字段。

**验收标准**:

- [ ] 在 `model.ChatOptions` 中添加 `ModelName` 字段
- [ ] 添加字段验证规则
- [ ] 更新 Swagger 文档注释
- [ ] 保持向后兼容（字段可选）

**实现文件**:

- `internal/model/chat.go`

---

### TASK-5.2: 修改 AI Service 传递租户和模型参数

**优先级**: P0  
**预计工时**: 3 小时  
**依赖**: TASK-5.1

**描述**:
修改 AI Service，将租户ID和模型名称传递给 Genkit Client。

**验收标准**:

- [ ] 从上下文获取当前租户ID
- [ ] 从 `ChatOptions` 中提取 `ModelName` 字段
- [ ] 修改 `Generate()` 调用，传递 tenantID 和 modelName
- [ ] 修改 `GenerateStream()` 调用，传递 tenantID 和 modelName
- [ ] 添加日志记录（包含租户ID和模型名称）
- [ ] 添加错误处理
- [ ] 保持向后兼容

**实现文件**:

- `internal/service/ai/genkit_service.go`

---

## 阶段 6：测试和优化

### TASK-6.1: 端到端测试

**优先级**: P0  
**预计工时**: 4 小时  
**依赖**: TASK-5.2

**描述**:
编写完整的端到端测试，覆盖所有提供商。

**验收标准**:

- [ ] 测试 Google AI 端到端流程
- [ ] 测试 Azure OpenAI 端到端流程
- [ ] 测试百炼端到端流程
- [ ] 测试提供商切换
- [ ] 测试默认提供商逻辑
- [ ] 测试错误场景

**实现文件**:

- `test/e2e/multi_provider_test.go`
- `test/test_multi_provider.sh`

---

### TASK-6.2: 性能测试

**优先级**: P1  
**预计工时**: 3 小时  
**依赖**: TASK-6.1

**描述**:
进行性能测试，确保多提供商支持不影响性能。

**验收标准**:

- [ ] 测试单提供商调用延迟
- [ ] 测试提供商切换延迟
- [ ] 测试并发调用性能
- [ ] 测试内存使用
- [ ] 对比优化前后性能
- [ ] 记录性能测试报告

**输出文档**:

- `docs/performance-test-report.md`

---

### TASK-6.3: 错误处理完善

**优先级**: P0  
**预计工时**: 3 小时  
**依赖**: TASK-6.1

**描述**:
完善各种错误场景的处理。

**验收标准**:

- [ ] 测试配置错误场景
- [ ] 测试 API 密钥错误
- [ ] 测试网络错误
- [ ] 测试提供商不可用
- [ ] 测试超时处理
- [ ] 确保错误信息友好且详细
- [ ] 确保错误日志完整

**实现文件**:

- `internal/genkit/client.go`
- `internal/service/ai/genkit_service.go`

---

### TASK-6.4: 日志和监控完善

**优先级**: P1  
**预计工时**: 2 小时  
**依赖**: TASK-6.1

**描述**:
完善日志记录和监控指标。

**验收标准**:

- [ ] 记录提供商选择日志
- [ ] 记录 API 调用耗时
- [ ] 记录 Token 使用统计
- [ ] 记录错误详情
- [ ] 添加 TraceID 追踪
- [ ] 确保敏感信息脱敏

**实现文件**:

- `internal/genkit/client.go`
- `internal/service/ai/genkit_service.go`

---

## 阶段 7：文档和部署

### TASK-7.1: 编写使用文档

**优先级**: P0  
**预计工时**: 3 小时  
**依赖**: TASK-6.1

**描述**:
编写完整的使用文档和配置指南。

**验收标准**:

- [ ] 编写配置文件说明
- [ ] 编写 API 使用示例
- [ ] 编写各提供商配置指南
- [ ] 编写故障排查指南
- [ ] 添加常见问题解答
- [ ] 提供完整的配置示例

**输出文档**:

- `docs/MULTI_PROVIDER_GUIDE.md`
- `docs/CONFIGURATION_GUIDE.md`
- `docs/TROUBLESHOOTING.md`

---

### TASK-7.2: 编写迁移指南

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-7.1

**描述**:
编写从单提供商到多提供商的迁移指南。

**验收标准**:

- [ ] 说明向后兼容性
- [ ] 提供迁移步骤
- [ ] 说明配置变更
- [ ] 提供迁移检查清单
- [ ] 说明回滚方案

**输出文档**:

- `docs/MIGRATION_GUIDE.md`

---

### TASK-7.3: 更新 Swagger 文档

**优先级**: P1  
**预计工时**: 2 小时  
**依赖**: TASK-5.1

**描述**:
更新 Swagger API 文档，说明新增的提供商参数。

**验收标准**:

- [ ] 更新 ChatOptions 定义
- [ ] 添加 provider 字段说明
- [ ] 添加使用示例
- [ ] 重新生成 Swagger 文档
- [ ] 验证文档正确性

**实现文件**:

- `internal/model/chat.go`
- `docs/swagger.yaml`

---

### TASK-7.4: 部署到测试环境

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-7.2

**描述**:
将多提供商支持部署到测试环境。

**验收标准**:

- [ ] 准备测试环境配置
- [ ] 配置所有 API 密钥
- [ ] 部署应用
- [ ] 验证所有提供商可用
- [ ] 进行冒烟测试
- [ ] 记录部署日志

**输出文档**:

- `docs/DEPLOYMENT_LOG.md`

---

## 任务依赖关系图

```
TASK-1.1 ──┬──> TASK-2.1 ──> TASK-2.2 ──┐
           │                            │
TASK-1.2 ──┘                            ├──> TASK-5.3 ──┐
                                        │                │
           TASK-2.1 ──> TASK-2.3 ──┬──> TASK-2.4 ──┐   │
                                   │                │   │
                                   └──> TASK-3.1 ──> TASK-3.2 ──> TASK-3.3 ──> TASK-3.4
                                                                                  │
                                   TASK-3.4 ──> TASK-4.1 ──> TASK-4.2 ──> TASK-4.3 ──> TASK-4.4 ──> TASK-4.5
                                                                                                              │
           TASK-2.4 ──> TASK-5.1 ──> TASK-5.2 ──────────────────────────────────────────────────────────────┤
                                                                                                              │
                                                                                                              ├──> TASK-6.1 ──┬──> TASK-6.2
                                                                                                              │                │
                                                                                                              │                ├──> TASK-6.3
                                                                                                              │                │
                                                                                                              │                └──> TASK-6.4
                                                                                                              │                     │
                                                                                                              └─────────────────────┤
                                                                                                                                    │
                                                                                                                                    ├──> TASK-7.1 ──> TASK-7.2
                                                                                                                                    │                     │
                                                                                                                                    └──> TASK-7.3 ────────┤
                                                                                                                                                          │
                                                                                                                                                          └──> TASK-7.4
```

### TASK-5.4: 更新应用初始化逻辑

**优先级**: P0  
**预计工时**: 2 小时  
**依赖**: TASK-2.2, TASK-5.2

**描述**:
更新应用启动时的初始化逻辑，注入 ModelConfigurationRepository。

**验收标准**:

- [ ] 修改 Genkit Client 初始化，注入 ModelConfigurationRepository
- [ ] 更新依赖注入配置
- [ ] 添加启动日志
- [ ] 处理初始化失败情况

**实现文件**:

- `cmd/server/main.go`
- `internal/app/app.go`

---

## 工时统计

- **阶段 1**: 2 小时
- **阶段 2**: 10 小时
- **阶段 3**: 10 小时
- **阶段 4**: 15 小时
- **阶段 5**: 5 小时
- **阶段 6**: 12 小时
- **阶段 7**: 9 小时

**总计**: 63 小时（约 8 个工作日）

## 说明

**已实现的功能**：

- model_configurations 表已创建
- ModelConfiguration 数据模型已实现
- ModelConfigurationRepository 已实现
- ModelConfigurationService 已实现（含权限验证）
- ModelConfigurationHandler 已实现（含完整的 CRUD 接口）
- 测试租户（738dbb1f-83e6-4bf5-935c-f0498236440d）的测试数据已配置

**本次任务重点**：

- 将 Genkit Client 与 model_configurations 表集成
- 实现根据租户ID和模型名称动态获取配置
- 支持 Google AI、Azure OpenAI、阿里云百炼三种提供商
- 实现配置缓存和懒加载机制

## 里程碑

- **M1**: 核心框架完成（TASK-2.4 完成）- 第 2 天
- **M2**: Azure OpenAI 集成完成（TASK-3.4 完成）- 第 4 天
- **M3**: 百炼集成完成（TASK-4.5 完成）- 第 7 天
- **M4**: 测试和文档完成（TASK-7.4 完成）- 第 9 天
