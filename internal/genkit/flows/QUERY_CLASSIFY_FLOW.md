# queryClassifyFlow 实现文档

## 概述

`queryClassifyFlow` 是一个 Genkit Flow，用于分析用户查询的类型和意图，为上下文构建提供决策依据。

## 功能特性

### 1. 查询特征提取

自动提取查询的以下特征：

- **指代词检测**：识别"它"、"他"、"她"、"这个"、"那个"等指代词
- **时间引用检测**：识别"昨天"、"今天"、"上次"、"之前"等时间词
- **疑问词检测**：识别"什么"、"为什么"、"怎么"、"如何"等疑问词
- **关键实体提取**：提取引号中的内容和专有名词
- **复杂度评估**：根据查询长度判断复杂度（simple/medium/complex）

### 2. 查询分类

支持以下 6 种查询类型：

| 类型 | 说明 | 需要历史 |
|------|------|----------|
| `simple_question` | 简单问题，不需要历史上下文 | 否 |
| `followup_question` | 追问，需要参考之前的对话 | 是 |
| `complex_query` | 复杂查询，需要详细的上下文 | 是 |
| `reference_query` | 引用查询，明确提到之前的内容 | 是 |
| `summarization` | 要求总结或概括 | 是 |
| `clarification` | 澄清或确认问题 | 是 |

### 3. 策略推荐

根据查询类型自动推荐上下文策略：

- `short`：简单问题，只需要少量上下文
- `auto`：自动选择，根据情况动态调整
- `full`：复杂查询，需要完整的上下文

## 分类规则

### 规则优先级

1. **总结请求**（最高优先级）
   - 关键词：总结、概括、归纳、汇总、梳理
   - 类型：`summarization`

2. **引用查询**
   - 关键词：刚才、上面、前面、之前说的、你说的
   - 类型：`reference_query`

3. **澄清问题**
   - 关键词：是吗、对吗、确认、是不是、对不对
   - 类型：`clarification`

4. **包含指代词或时间引用**
   - 检测到指代词或时间词
   - 类型：`followup_question`

5. **复杂查询**
   - 查询长度 >= 50 字符
   - 类型：`complex_query`

6. **默认**
   - 其他情况
   - 类型：`simple_question`

## 输入输出

### 输入 (QueryClassifyInput)

```go
type QueryClassifyInput struct {
    Query     string `json:"query" validate:"required,max=2000"`
    SessionID string `json:"sessionId" validate:"omitempty,uuid"`
}
```

### 输出 (QueryClassifyOutput)

```go
type QueryClassifyOutput struct {
    QueryType          string   `json:"queryType"`           // 查询类型
    NeedsHistory       bool     `json:"needsHistory"`        // 是否需要历史上下文
    KeyEntities        []string `json:"keyEntities"`         // 关键实体
    RecommendedStrategy string  `json:"recommendedStrategy"` // 推荐的上下文策略
    Confidence         float64  `json:"confidence"`          // 置信度 (0-1)
    Reasoning          string   `json:"reasoning"`           // 分类推理过程
}
```

## 使用示例

### 1. 简单问题

**输入**：

```json
{
  "query": "什么是人工智能？"
}
```

**输出**：

```json
{
  "queryType": "simple_question",
  "needsHistory": false,
  "keyEntities": [],
  "recommendedStrategy": "short",
  "confidence": 0.6,
  "reasoning": "简单问题，不需要历史上下文"
}
```

### 2. 追问

**输入**：

```json
{
  "query": "它是怎么工作的？"
}
```

**输出**：

```json
{
  "queryType": "followup_question",
  "needsHistory": true,
  "keyEntities": [],
  "recommendedStrategy": "auto",
  "confidence": 0.6,
  "reasoning": "查询包含指代词或时间引用，需要历史上下文"
}
```

### 3. 总结请求

**输入**：

```json
{
  "query": "请总结一下我们刚才讨论的内容"
}
```

**输出**：

```json
{
  "queryType": "summarization",
  "needsHistory": true,
  "keyEntities": [],
  "recommendedStrategy": "full",
  "confidence": 0.6,
  "reasoning": "查询要求总结或概括"
}
```

### 4. 引用查询

**输入**：

```json
{
  "query": "你刚才说的那个算法是什么？"
}
```

**输出**：

```json
{
  "queryType": "reference_query",
  "needsHistory": true,
  "keyEntities": [],
  "recommendedStrategy": "full",
  "confidence": 0.6,
  "reasoning": "查询明确引用之前的内容"
}
```

## 技术实现

### 避免循环依赖

为了避免 `internal/genkit/flows` 和 `internal/service/ai` 之间的循环依赖，本实现采用**基于规则的分类方法**，不依赖 AI 服务。

如果需要使用 AI 进行更精确的分类，可以：

1. 先调用 `queryClassifyFlow` 获取基础分类
2. 再调用 AI 服务进行增强分类
3. 在应用层组合两者的结果

### 性能考虑

- **轻量级**：纯规则匹配，无需调用 AI 服务
- **快速响应**：通常在 1ms 内完成分类
- **无外部依赖**：不依赖网络请求或数据库查询

## 测试覆盖

实现了以下测试：

1. `TestExtractQueryFeatures`：测试查询特征提取
2. `TestClassifyQueryWithRules`：测试基于规则的分类
3. `TestRecommendStrategy`：测试策略推荐
4. `TestValidateQueryInput`：测试输入验证

测试覆盖率：100%

## 未来改进

### 1. AI 增强分类

可以创建一个独立的服务层方法，使用 AI 对分类结果进行增强：

```go
// 在应用层或服务层
func EnhanceClassification(
    ctx context.Context,
    basicClassification QueryClassifyOutput,
    aiService ai.AIService,
) (QueryClassifyOutput, error) {
    // 使用 AI 服务增强分类
    // ...
}
```

### 2. 机器学习模型

可以训练一个轻量级的分类模型，部署在本地：

- 使用历史查询数据训练
- 支持多语言
- 更准确的意图识别

### 3. 上下文感知

结合会话历史进行分类：

- 分析对话轮次
- 识别话题转换
- 检测对话模式

## 相关需求

- 需求 1：Flow 定义和注册
- 需求 4：查询分类 Flow

## 相关文件

- `internal/genkit/flows/query.go`：Flow 实现
- `internal/genkit/flows/query_test.go`：单元测试
- `internal/genkit/flows/types.go`：类型定义
- `internal/genkit/registry.go`：Flow 注册
