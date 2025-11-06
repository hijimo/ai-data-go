# Token Manager 实现说明

## 概述

Token Manager 是一个用于计算和估算 AI 模型 Token 使用量的服务组件。它提供了多种方法来计算文本、消息、记忆和摘要的 Token 数量。

## 功能特性

### 1. 文本 Token 计算 (`CalculateTokens`)

使用改进的启发式算法计算文本的 Token 数量：

- **中文文本**：约 1.5 个字符 = 1 个 token
- **英文文本**：约 4 个字符 = 1 个 token（或按单词数计算）
- **混合文本**：根据字符类型动态计算

### 2. 上下文 Token 计算 (`CalculateContextTokens`)

计算完整对话上下文的 Token 总数，包括：

- 短期消息列表
- 长期记忆列表
- 会话摘要

支持使用预计算的 Token 数量或动态计算。

### 3. 快速估算 (`EstimateTokens`)

使用简单的启发式方法快速估算 Token 数量：

- 速度快，适合实时场景
- 精度较低，适合粗略估算
- 使用字符数除以平均比率（约 3 个字符 = 1 个 token）

### 4. 消息列表 Token 计算 (`CalculateMessagesTokens`)

专门用于计算消息列表的 Token 数量：

- 自动处理角色标记开销（每条消息约 4 个 token）
- 支持使用预计算值或动态计算

## 算法实现

### 启发式算法 (`calculateTokensHeuristic`)

改进的算法考虑了中英文混合文本的特点：

1. **字符分类**：
   - 中文字符（CJK 统一表意文字）
   - 英文字母
   - 其他字符（数字、标点等）

2. **Token 计算**：
   - 中文：`(字符数 * 2) / 3`
   - 英文：`(字符数 / 4 + 单词数) / 2`
   - 其他：`字符数 / 3`

3. **特殊处理**：
   - 空文本返回 0
   - 非空文本至少返回 1 个 token

### 字符判断函数

- `isChinese(r rune)`: 判断是否为中文字符（支持 CJK 扩展）
- `isEnglishLetter(r rune)`: 判断是否为英文字母
- `isWhitespace(r rune)`: 判断是否为空白字符

## 使用示例

```go
// 创建 Token Manager
log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
tm := service.NewTokenManager(log)

// 计算文本 Token
ctx := context.Background()
tokens, err := tm.CalculateTokens(ctx, "Hello 世界", "gpt-4")
if err != nil {
    log.Error("计算失败", logger.Fields{"error": err})
}

// 快速估算
estimatedTokens := tm.EstimateTokens("这是一段测试文本")

// 计算上下文 Token
totalTokens, err := tm.CalculateContextTokens(
    ctx,
    messages,
    memories,
    summary,
)
```

## 性能考虑

1. **缓存策略**：
   - 消息、记忆和摘要可以预计算并缓存 Token 数量
   - 避免重复计算相同内容

2. **估算 vs 精确计算**：
   - 使用 `EstimateTokens` 进行快速估算
   - 使用 `CalculateTokens` 进行精确计算

3. **批量处理**：
   - `CalculateContextTokens` 支持批量计算多个组件
   - 减少函数调用开销

## 测试覆盖

实现了全面的单元测试：

- ✅ 空文本处理
- ✅ 纯英文文本
- ✅ 纯中文文本
- ✅ 中英文混合文本
- ✅ 长文本处理
- ✅ 上下文计算（消息、记忆、摘要）
- ✅ 预计算 Token 使用
- ✅ 动态 Token 计算
- ✅ 字符判断函数
- ✅ 性能基准测试

## 未来改进

1. **集成 tiktoken 库**：
   - 对于支持的模型，使用官方 tiktoken 库
   - 提供更精确的 Token 计算

2. **模型特定优化**：
   - 为不同模型（GPT-4, Gemini, Claude）提供专门的计算策略
   - 考虑模型特定的 Token 化规则

3. **缓存优化**：
   - 实现 Token 计算结果缓存
   - 减少重复计算开销

4. **并发支持**：
   - 支持并发计算多个文本的 Token
   - 提高批量处理性能

## 相关文件

- `internal/service/token_manager.go` - 接口定义
- `internal/service/token_manager_impl.go` - 实现代码
- `internal/service/token_manager_test.go` - 单元测试

## 依赖关系

- `internal/logger` - 日志记录
- `internal/model` - 数据模型（ChatMessage, ConversationMemory, ConversationSummary）

## 注意事项

1. **启发式算法的局限性**：
   - 当前实现使用启发式算法，精度有限
   - 对于精确计费场景，建议集成官方 tiktoken 库

2. **模型差异**：
   - 不同模型的 Token 化规则可能不同
   - 当前实现提供通用算法，可能需要针对特定模型调整

3. **性能权衡**：
   - `EstimateTokens` 速度快但精度低
   - `CalculateTokens` 精度较高但计算开销较大
   - 根据场景选择合适的方法
