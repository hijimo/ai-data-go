# Task 20: summaryQualityFlow 实现总结

## 任务概述

实现了 `summaryQualityFlow`，用于评估生成的摘要质量，包括四个维度的质量评估、问题识别和改进建议生成。

## 实现内容

### 1. 类型定义（types.go）

#### SummaryQualityInput

- `summaryId`: 摘要ID（可选，如果提供则从数据库加载）
- `summary`: 摘要内容（必需，除非提供summaryId）
- `originalMessages`: 原始消息列表（必需）
- `dimensions`: 评估维度列表（可选，默认评估所有维度）

#### SummaryQualityOutput

- `summaryId`: 摘要ID
- `overallScore`: 总体质量评分（0-1）
- `dimensionScores`: 各维度评分映射
- `passed`: 是否通过质量检查（>= 0.7）
- `issues`: 质量问题列表
- `suggestions`: 改进建议列表
- `keyInfoCoverage`: 关键信息覆盖率（0-1）
- `redundancyScore`: 冗余度评分（0-1，越低越好）
- `evaluationTime`: 评估耗时（毫秒）

#### QualityIssue

- `dimension`: 维度名称
- `severity`: 严重程度（low、medium、high）
- `description`: 问题描述
- `score`: 该维度的评分
- `impact`: 影响说明

### 2. Flow 实现（summary.go）

#### summaryQualityFlow 主流程

1. **参数验证**
   - 验证摘要内容或摘要ID必须提供其中之一
   - 验证原始消息列表不为空

2. **摘要加载**
   - 如果提供了摘要ID，从数据库加载摘要内容

3. **维度确定**
   - 默认评估所有四个维度：completeness、conciseness、coherence、accuracy

4. **四维度评估**
   - 完整性（Completeness）：检查摘要是否涵盖原始消息的关键信息
   - 简洁性（Conciseness）：检查摘要是否简洁，避免冗余
   - 连贯性（Coherence）：检查摘要的逻辑结构和语言流畅性
   - 准确性（Accuracy）：检查摘要是否准确反映原始内容

5. **总体评分计算**
   - 计算各维度评分的平均值

6. **质量判定**
   - 总体评分 >= 0.7 视为通过质量检查

7. **附加指标计算**
   - 关键信息覆盖率
   - 冗余度评分

8. **结果输出**
   - 返回完整的质量评估结果

### 3. 质量评估函数

#### evaluateCompleteness（完整性评估）

- 提取原始消息和摘要中的关键词
- 计算关键词覆盖率
- 识别遗漏的重要内容
- 评分标准：覆盖率 >= 70% 为良好

#### evaluateConciseness（简洁性评估）

- 计算压缩率（摘要长度 vs 原文长度）
- 理想压缩率：50%-80%
- 识别过长或过短的问题
- 评分标准：压缩率在理想范围内为满分

#### evaluateCoherence（连贯性评估）

- 检查摘要长度合理性
- 统计句子数量（2-20个为理想）
- 检查连接词使用情况
- 评分标准：结构清晰、逻辑连贯

#### evaluateAccuracy（准确性评估）

- 提取摘要中的关键实体
- 验证实体是否在原始消息中出现
- 计算准确率
- 评分标准：准确率 >= 90% 为良好

### 4. 文本分析辅助函数

实现了一系列文本分析工具函数：

- `extractKeywords`: 提取关键词
- `extractWords`: 提取词汇
- `isStopWord`: 判断停用词
- `countSentences`: 统计句子数量
- `checkConnectors`: 检查连接词使用
- `extractEntities`: 提取实体
- `containsEntity`: 检查实体包含
- `extractKeyPoints`: 提取关键信息点
- `containsKeyPoint`: 检查关键点包含
- `calculateKeyInfoCoverage`: 计算关键信息覆盖率
- `calculateRedundancyScore`: 计算冗余度评分

## 技术特点

### 1. 多维度评估

- 从四个不同角度全面评估摘要质量
- 每个维度独立评分，便于定位具体问题

### 2. 问题识别

- 自动识别质量问题
- 按严重程度分级（low、medium、high）
- 提供详细的问题描述和影响说明

### 3. 改进建议

- 针对每个问题提供具体的改进建议
- 帮助用户或系统优化摘要生成策略

### 4. 灵活配置

- 支持选择性评估特定维度
- 支持从数据库加载摘要或直接提供内容

### 5. 性能优化

- 使用高效的文本分析算法
- 记录评估耗时，便于性能监控

## 使用示例

### 评估已生成的摘要

```go
input := SummaryQualityInput{
    SummaryID: "summary-uuid",
    OriginalMessages: []string{
        "用户询问了产品价格",
        "客服回复价格为99元",
        "用户表示满意并下单",
    },
}

output, err := summaryQualityFlow.Run(ctx, input)
if err != nil {
    // 处理错误
}

if output.Passed {
    fmt.Printf("摘要质量良好，总体评分: %.2f\n", output.OverallScore)
} else {
    fmt.Printf("摘要质量不佳，发现 %d 个问题\n", len(output.Issues))
    for _, issue := range output.Issues {
        fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.Dimension, issue.Description)
    }
}
```

### 评估新摘要内容

```go
input := SummaryQualityInput{
    Summary: "用户咨询产品价格，客服报价99元，用户下单购买。",
    OriginalMessages: []string{
        "用户询问了产品价格",
        "客服回复价格为99元",
        "用户表示满意并下单",
    },
    Dimensions: []string{"completeness", "accuracy"}, // 只评估特定维度
}

output, err := summaryQualityFlow.Run(ctx, input)
```

## 质量标准

### 通过标准

- 总体评分 >= 0.7
- 各维度评分均衡，无严重问题

### 各维度标准

#### 完整性（Completeness）

- 优秀（>= 0.8）：覆盖80%以上的关键信息
- 良好（>= 0.7）：覆盖70%以上的关键信息
- 需改进（< 0.7）：关键信息遗漏较多

#### 简洁性（Conciseness）

- 优秀（>= 0.8）：压缩率在50%-80%之间
- 良好（>= 0.7）：压缩率在40%-85%之间
- 需改进（< 0.7）：过长或过短

#### 连贯性（Coherence）

- 优秀（>= 0.8）：结构清晰，逻辑连贯，使用连接词
- 良好（>= 0.7）：基本连贯，结构合理
- 需改进（< 0.7）：结构混乱或逻辑不清

#### 准确性（Accuracy）

- 优秀（>= 0.9）：所有信息准确无误
- 良好（>= 0.8）：大部分信息准确
- 需改进（< 0.8）：存在不准确或错误信息

## 与其他 Flow 的集成

### 与 summaryGenerateFlow 集成

```go
// 1. 生成摘要
generateOutput, err := summaryGenerateFlow.Run(ctx, generateInput)

// 2. 评估质量
qualityInput := SummaryQualityInput{
    SummaryID: generateOutput.SummaryID,
    OriginalMessages: originalMessages,
}
qualityOutput, err := summaryQualityFlow.Run(ctx, qualityInput)

// 3. 如果质量不佳，重新生成
if !qualityOutput.Passed {
    // 根据建议调整生成策略
    // 重新生成摘要
}
```

### 与 summaryTriggerFlow 集成

```go
// 1. 检查是否需要生成摘要
triggerOutput, err := summaryTriggerFlow.Run(ctx, triggerInput)

if triggerOutput.ShouldSummarize {
    // 2. 生成摘要
    generateOutput, err := summaryGenerateFlow.Run(ctx, generateInput)
    
    // 3. 评估质量
    qualityOutput, err := summaryQualityFlow.Run(ctx, qualityInput)
    
    // 4. 只有质量合格的摘要才保存
    if qualityOutput.Passed {
        // 保存摘要
    }
}
```

## 监控指标

建议监控以下指标：

1. **评估通过率**：通过质量检查的摘要比例
2. **平均评分**：各维度和总体的平均评分
3. **常见问题**：最常出现的质量问题类型
4. **评估耗时**：质量评估的平均耗时

## 未来优化方向

1. **AI 辅助评估**
   - 使用 AI 模型进行更深入的语义分析
   - 提高准确性和连贯性评估的精度

2. **自定义评估标准**
   - 支持租户自定义质量标准
   - 支持不同场景的评估权重配置

3. **历史数据分析**
   - 分析历史摘要质量趋势
   - 自动优化生成策略

4. **多语言支持**
   - 支持不同语言的文本分析
   - 适配不同语言的质量标准

## 相关需求

- 需求 1：Flow 定义和注册
- 需求 11：摘要质量评估 Flow

## 文件变更

- `internal/genkit/flows/types.go`：新增类型定义
- `internal/genkit/flows/summary.go`：实现 summaryQualityFlow 和辅助函数

## 测试建议

1. **单元测试**
   - 测试各维度评估函数
   - 测试文本分析辅助函数
   - 测试边界情况（空摘要、超长摘要等）

2. **集成测试**
   - 测试完整的质量评估流程
   - 测试与数据库的集成
   - 测试不同质量水平的摘要

3. **性能测试**
   - 测试大量消息的评估性能
   - 测试并发评估场景

## 总结

成功实现了 `summaryQualityFlow`，提供了全面的摘要质量评估能力。该 Flow 通过四个维度的评估，能够准确识别摘要质量问题并提供改进建议，为摘要生成系统提供了重要的质量保障机制。
