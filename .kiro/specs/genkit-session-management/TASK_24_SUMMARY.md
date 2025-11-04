# Task 24: tokenAnalysisFlow 实现总结

## 任务概述

实现 tokenAnalysisFlow，提供 Token 使用分析功能，支持四种分析维度（usage、trend、cost、efficiency），并生成优化建议和使用量预测。

## 实现内容

### 1. 类型定义

在 `internal/genkit/flows/token.go` 中定义了：

#### TokenAnalysisInput

```go
type TokenAnalysisInput struct {
    TenantID      string   `json:"tenantId" validate:"required,uuid"`
    SessionID     string   `json:"sessionId" validate:"omitempty,uuid"`
    TimeRangeDays int      `json:"timeRangeDays" validate:"required,min=1,max=365"`
    Dimensions    []string `json:"dimensions" validate:"dive,oneof=usage trend cost efficiency"`
}
```

#### TokenAnalysisOutput

```go
type TokenAnalysisOutput struct {
    TotalUsage        int                              `json:"totalUsage"`
    AverageDailyUsage int                              `json:"averageDailyUsage"`
    PeakUsage         int                              `json:"peakUsage"`
    Trend             string                           `json:"trend"`
    EstimatedCost     float64                          `json:"estimatedCost"`
    EfficiencyScore   float64                          `json:"efficiencyScore"`
    Suggestions       []service.OptimizationSuggestion `json:"suggestions"`
    Predictions       service.TokenPredictions         `json:"predictions"`
    AnalysisTime      string                           `json:"analysisTime"`
}
```

### 2. Flow 定义

在 `internal/genkit/flows/token.go` 的 `RegisterTokenFlows` 函数中实现了 tokenAnalysisFlow：

**主要功能：**

- 参数验证（租户ID、时间范围、分析维度）
- 设置默认分析维度（如果未指定，使用全部四个维度）
- 调用服务层的 `AnalyzeUsage` 方法
- 构建输出结果，包含分析时间戳

**验证逻辑：**

- 租户ID必填
- 时间范围必须在 1-365 天之间
- 分析维度必须是 usage、trend、cost、efficiency 之一

### 3. 服务层实现

在 `internal/service/token_manager_impl.go` 中实现了 `AnalyzeUsage` 方法：

#### 四种分析维度实现

**1. Usage（使用量分析）**

- 计算指定时间范围内的总使用量
- 计算平均每日使用量
- 获取峰值使用量（按天统计的最大值）
- 支持会话级别和租户级别分析

**2. Trend（趋势分析）**

- 将时间段分为两半，比较前后使用量
- 计算变化率：
  - 变化率 > 20%：increasing（增长）
  - 变化率 < -20%：decreasing（下降）
  - 其他：stable（稳定）

**3. Cost（成本分析）**

- 基于使用量估算成本
- 定价模型：每 1000 tokens = $0.002
- 公式：`estimatedCost = totalUsage / 1000.0 * 0.002`

**4. Efficiency（效率分析）**

- 基于峰值与平均值的比率计算效率评分
- 理想比率：1.5-2.0（评分 0.9）
- 比率过低或过高都会降低效率评分
- 评分范围：0.3-0.9

#### 优化建议生成

根据分析结果生成优化建议，包括：

**基于趋势的建议：**

- 使用量持续增长 → 启用上下文优化和摘要功能（预计节省 25%）

**基于效率的建议：**

- 效率评分 < 0.6 → 优化上下文策略和减少冗余内容（预计节省 33%）

**通用优化建议：**

- 总使用量 > 100,000 → 启用缓存机制（预计节省 10%）
- 总使用量 > 100,000 → 定期清理低质量记忆（预计节省 5%）

每个建议包含：

- 优先级（high、medium、low）
- 建议内容
- 预计节省量

#### 使用量预测

基于平均每日使用量和趋势预测未来使用量：

**预测时间点：**

- 次日（NextDay）
- 次周（NextWeek）
- 次月（NextMonth）

**趋势调整：**

- increasing：假设每天增长 5%，月度增长 10%
- decreasing：假设每天减少 3%，月度减少 10%
- stable：使用平均值直接计算

### 4. 数据库查询

实现了多个辅助方法支持分析功能：

**getSessionTokenUsage**

- 查询会话在指定时间范围内的 Token 使用量
- 从 `conversation_contexts` 表的 `total_tokens_used` 字段统计

**getTenantTokenUsage**

- 查询租户在指定时间范围内的 Token 使用量
- 通过 JOIN `conversation_sessions` 表关联租户ID

**getPeakUsage**

- 按天统计使用量，找出最大值
- 使用 SQL 的 `DATE()` 函数和 `GROUP BY` 实现

**analyzeTrend**

- 将时间段分为两半，分别统计使用量
- 计算变化率判断趋势

## 技术特点

### 1. 灵活的分析维度

- 支持指定分析维度，也可以使用默认的全部维度
- 每个维度独立计算，互不影响

### 2. 多级别分析

- 支持会话级别分析（指定 sessionID）
- 支持租户级别分析（不指定 sessionID）

### 3. 智能建议生成

- 基于实际数据生成针对性建议
- 提供预计节省量，帮助决策

### 4. 准确的趋势预测

- 考虑历史趋势进行预测
- 提供短期、中期、长期预测

### 5. 成本估算

- 基于实际定价模型估算成本
- 帮助租户了解使用成本

## 符合需求

✅ **需求 1**：Flow 定义和注册

- 使用 `genkit.DefineFlow` 定义 Flow
- 在 `RegisterTokenFlows` 中注册
- 提供类型安全的输入输出

✅ **需求 18**：Token 使用分析 Flow

- 支持四种分析维度（usage、trend、cost、efficiency）
- 计算总使用量、平均每日使用量、峰值使用量
- 识别使用趋势（increasing、stable、decreasing）
- 估算使用成本
- 计算效率评分（0-1 范围）
- 提供优化建议列表
- 预测未来使用量（次日、次周、次月）
- 为每个建议标注优先级和预计节省量

## 使用示例

### 租户级别分析（全部维度）

```go
input := TokenAnalysisInput{
    TenantID:      "tenant-uuid",
    TimeRangeDays: 30,
    Dimensions:    []string{"usage", "trend", "cost", "efficiency"},
}

output, err := flow.Run(ctx, input)
// output.TotalUsage: 1500000
// output.AverageDailyUsage: 50000
// output.PeakUsage: 75000
// output.Trend: "increasing"
// output.EstimatedCost: 3.0
// output.EfficiencyScore: 0.85
// output.Suggestions: [...]
// output.Predictions: {NextDay: 52500, NextWeek: 367500, NextMonth: 1650000}
```

### 会话级别分析（指定维度）

```go
input := TokenAnalysisInput{
    TenantID:      "tenant-uuid",
    SessionID:     "session-uuid",
    TimeRangeDays: 7,
    Dimensions:    []string{"usage", "trend"},
}

output, err := flow.Run(ctx, input)
// 只包含 usage 和 trend 相关的分析结果
```

### 默认维度分析

```go
input := TokenAnalysisInput{
    TenantID:      "tenant-uuid",
    TimeRangeDays: 14,
    // Dimensions 为空，自动使用全部四个维度
}

output, err := flow.Run(ctx, input)
// 包含所有四个维度的完整分析
```

## 输出示例

```json
{
  "totalUsage": 1500000,
  "averageDailyUsage": 50000,
  "peakUsage": 75000,
  "trend": "increasing",
  "estimatedCost": 3.0,
  "efficiencyScore": 0.85,
  "suggestions": [
    {
      "priority": "high",
      "suggestion": "Token使用量持续增长，建议启用上下文优化和摘要功能",
      "estimatedSaving": 12500
    },
    {
      "priority": "medium",
      "suggestion": "启用缓存机制以减少重复查询的Token消耗",
      "estimatedSaving": 5000
    }
  ],
  "predictions": {
    "nextDay": 52500,
    "nextWeek": 367500,
    "nextMonth": 1650000
  },
  "analysisTime": "2025-11-01T10:30:00Z"
}
```

## 性能考虑

1. **数据库查询优化**
   - 使用索引加速时间范围查询
   - 使用聚合函数减少数据传输
   - 支持会话级别和租户级别的灵活查询

2. **计算效率**
   - Token 计算使用简单估算方法
   - 趋势分析只需两次查询
   - 峰值查询使用 SQL 聚合

3. **可扩展性**
   - 支持自定义分析维度
   - 易于添加新的分析指标
   - 建议生成逻辑可配置

## 后续优化建议

1. **缓存分析结果**
   - 对于相同的查询参数，可以缓存分析结果
   - TTL 设置为 5-10 分钟

2. **异步分析**
   - 对于大时间范围的分析，可以考虑异步处理
   - 提供分析任务状态查询接口

3. **更精确的 Token 计算**
   - 集成实际的 tokenizer（如 tiktoken）
   - 支持不同模型的 Token 计算

4. **更丰富的可视化数据**
   - 提供按天的使用量数组
   - 支持导出图表数据

5. **机器学习预测**
   - 使用时间序列模型进行更准确的预测
   - 考虑季节性和周期性因素

## 总结

Task 24 已完整实现，tokenAnalysisFlow 提供了全面的 Token 使用分析功能，包括：

✅ 四种分析维度（usage、trend、cost、efficiency）
✅ 智能优化建议生成
✅ 准确的使用量预测
✅ 灵活的分析级别（会话/租户）
✅ 完整的类型定义和验证
✅ 高效的数据库查询
✅ 符合所有需求规范

该实现为租户管理员提供了强大的 Token 使用分析工具，帮助他们了解使用模式、控制成本并优化使用策略。
