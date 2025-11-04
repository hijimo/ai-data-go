# AI 对话会话管理系统 - Genkit Flow 集成指南

## 目录

- [1. 概述](#1-概述)
- [2. Genkit Flow 基础](#2-genkit-flow-基础)
- [3. 上下文构建 Flow](#3-上下文构建-flow)
- [4. 对话生成 Flow](#4-对话生成-flow)
- [5. 摘要生成 Flow](#5-摘要生成-flow)
- [6. 记忆检索 Flow](#6-记忆检索-flow)
- [7. 流式对话 Flow](#7-流式对话-flow)
- [8. Flow 组合与编排](#8-flow-组合与编排)
- [9. 部署配置](#10-部署配置)
- [10. 最佳实践](#11-最佳实践)

## 1. 概述

本文档描述如何使用 Google Genkit Flow 构建 AI 对话会话管理系统的核心流程。Genkit Flow 提供了类型安全、可追踪、易部署的 AI 工作流构建能力。

### 1.1 为什么使用 Genkit Flow

- **类型安全**：输入输出类型检查，减少运行时错误
- **可观测性**：内置追踪功能，便于调试和监控
- **流式支持**：原生支持流式响应，提升用户体验
- **简化部署**：可直接部署为 HTTP 端点
- **开发工具**：提供 Developer UI 用于测试和调试

### 1.2 架构集成

```
┌─────────────────────────────────────────────────────────────┐
│                      API Layer (Gin)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Session API  │  │ Context API  │  │   Chat API   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Genkit Flow Layer                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Context Flow  │  │  Chat Flow   │  │Summary Flow  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │Memory Flow   │  │Embedding Flow│                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │Session Svc   │  │ Memory Svc   │  │ Vector Svc   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## 2. Genkit Flow 基础

### 2.1 初始化 Genkit

```go
package main

import (
    "context"
    "log"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
)

var g *genkit.Genkit

func InitGenkit(ctx context.Context) {
    // 初始化 Genkit，配置插件
    g = genkit.Init(ctx, 
        genkit.WithPlugins(
            &googlegenai.GoogleAI{},
        ),
    )
    
    log.Println("Genkit initialized successfully")
}
```

### 2.2 基本 Flow 定义

```go
// 定义输入输出结构
type ChatInput struct {
    SessionID string `json:"sessionId"`
    Message   string `json:"message"`
}

type ChatOutput struct {
    Response  string `json:"response"`
    TokenUsed int    `json:"tokenUsed"`
}

// 定义简单的 Flow
var simpleChatFlow = genkit.DefineFlow(g, "simpleChatFlow",
    func(ctx context.Context, input ChatInput) (ChatOutput, error) {
        // Flow 逻辑
        resp, err := genkit.Generate(ctx, g,
            ai.WithPrompt(input.Message),
        )
        if err != nil {
            return ChatOutput{}, err
        }
        
        return ChatOutput{
            Response:  resp.Text(),
            TokenUsed: resp.Usage().TotalTokens,
        }, nil
    },
)
```

## 3. 上下文构建 Flow

### 3.1 上下文构建输入输出定义

```go
// ContextBuildInput 上下文构建输入
type ContextBuildInput struct {
    SessionID string `json:"sessionId"`
    UserQuery string `json:"userQuery"`
    MaxTokens int    `json:"maxTokens"`
}

// ContextBuildOutput 上下文构建输出
type ContextBuildOutput struct {
    SessionID        string           `json:"sessionId"`
    Summary          *SummaryContext  `json:"summary,omitempty"`
    RelevantMemories []MemoryContext  `json:"relevantMemories,omitempty"`
    Messages         []MessageContext `json:"messages"`
    TotalTokens      int              `json:"totalTokens"`
}

// SummaryContext 摘要上下文
type SummaryContext struct {
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
    CreatedAt  string `json:"createdAt"`
}

// MemoryContext 记忆上下文
type MemoryContext struct {
    Content    string  `json:"content"`
    TokenCount int     `json:"tokenCount"`
    Importance float32 `json:"importance"`
    Similarity float32 `json:"similarity"`
}

// MessageContext 消息上下文
type MessageContext struct {
    ID         string `json:"id"`
    Role       string `json:"role"`
    Content    string `json:"content"`
    TokenCount int    `json:"tokenCount"`
}
```

### 3.2 上下文构建 Flow 实现

```go
// BuildContextFlow 构建对话上下文的 Flow
var BuildContextFlow = genkit.DefineFlow(g, "buildContextFlow",
    func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
        // 1. 验证权限
        if err := validateSessionAccess(ctx, input.SessionID); err != nil {
            return ContextBuildOutput{}, err
        }
        
        // 2. 获取短期记忆（最近的消息）
        recentMessages, err := getRecentMessages(ctx, input.SessionID, 10)
        if err != nil {
            return ContextBuildOutput{}, fmt.Errorf("获取短期记忆失败: %w", err)
        }
        
        // 3. 获取长期记忆（向量检索）
        var relevantMemories []MemoryContext
        if input.UserQuery != "" {
            memories, err := searchRelevantMemories(ctx, input.SessionID, input.UserQuery)
            if err != nil {
                log.Printf("获取长期记忆失败: %v", err)
                // 不中断流程
            } else {
                relevantMemories = memories
            }
        }
        
        // 4. 获取摘要
        summary, _ := getLatestSummary(ctx, input.SessionID)
        
        // 5. 组合上下文
        output := ContextBuildOutput{
            SessionID:        input.SessionID,
            Summary:          summary,
            RelevantMemories: relevantMemories,
            Messages:         convertToMessageContexts(recentMessages),
        }
        
        // 6. 计算总 token 数
        output.TotalTokens = calculateTotalTokens(output)
        
        // 7. Token 优化
        if input.MaxTokens > 0 && output.TotalTokens > input.MaxTokens {
            output = optimizeContext(output, input.MaxTokens)
        }
        
        return output, nil
    },
)

// 辅助函数
func getRecentMessages(ctx context.Context, sessionID string, limit int) ([]ChatMessage, error) {
    // 从数据库获取最近的消息
    // 实现省略
    return nil, nil
}

func searchRelevantMemories(ctx context.Context, sessionID, query string) ([]MemoryContext, error) {
    // 向量检索相关记忆
    // 实现省略
    return nil, nil
}

func getLatestSummary(ctx context.Context, sessionID string) (*SummaryContext, error) {
    // 获取最新摘要
    // 实现省略
    return nil, nil
}

func calculateTotalTokens(output ContextBuildOutput) int {
    total := 0
    if output.Summary != nil {
        total += output.Summary.TokenCount
    }
    for _, mem := range output.RelevantMemories {
        total += mem.TokenCount
    }
    for _, msg := range output.Messages {
        total += msg.TokenCount
    }
    return total
}

func optimizeContext(output ContextBuildOutput, maxTokens int) ContextBuildOutput {
    // Token 优化逻辑
    // 实现省略
    return output
}
```

## 4. 对话生成 Flow

### 4.1 对话生成输入输出定义

```go
// ChatGenerateInput 对话生成输入
type ChatGenerateInput struct {
    SessionID    string                 `json:"sessionId"`
    UserMessage  string                 `json:"userMessage"`
    Context      *ContextBuildOutput    `json:"context,omitempty"`
    ModelConfig  *ModelConfig           `json:"modelConfig,omitempty"`
}

// ChatGenerateOutput 对话生成输出
type ChatGenerateOutput struct {
    MessageID    string `json:"messageId"`
    Response     string `json:"response"`
    TokenUsed    int    `json:"tokenUsed"`
    FinishReason string `json:"finishReason"`
}

// ModelConfig 模型配置
type ModelConfig struct {
    ModelName   string   `json:"modelName"`
    Temperature *float64 `json:"temperature,omitempty"`
    TopP        *float64 `json:"topP,omitempty"`
    MaxTokens   *int     `json:"maxTokens,omitempty"`
}
```

### 4.2 对话生成 Flow 实现

```go
// ChatGenerateFlow 对话生成 Flow
var ChatGenerateFlow = genkit.DefineFlow(g, "chatGenerateFlow",
    func(ctx context.Context, input ChatGenerateInput) (ChatGenerateOutput, error) {
        // 1. 如果没有提供上下文，先构建上下文
        var context ContextBuildOutput
        if input.Context == nil {
            contextInput := ContextBuildInput{
                SessionID: input.SessionID,
                UserQuery: input.UserMessage,
                MaxTokens: 4000,
            }
            
            var err error
            context, err = BuildContextFlow.Run(ctx, contextInput)
            if err != nil {
                return ChatGenerateOutput{}, fmt.Errorf("构建上下文失败: %w", err)
            }
        } else {
            context = *input.Context
        }
        
        // 2. 构建提示词
        prompt := buildPrompt(context, input.UserMessage)
        
        // 3. 配置模型参数
        modelConfig := getModelConfig(input.ModelConfig)
        
        // 4. 调用 AI 生成
        resp, err := genkit.Generate(ctx, g,
            ai.WithPrompt(prompt),
            ai.WithConfig(modelConfig),
        )
        if err != nil {
            return ChatGenerateOutput{}, fmt.Errorf("生成响应失败: %w", err)
        }
        
        // 5. 保存消息到数据库
        messageID, err := saveMessages(ctx, input.SessionID, input.UserMessage, resp.Text())
        if err != nil {
            log.Printf("保存消息失败: %v", err)
            // 不中断流程
        }
        
        // 6. 异步处理：生成向量、更新摘要
        go processMessageAsync(input.SessionID, messageID)
        
        return ChatGenerateOutput{
            MessageID:    messageID,
            Response:     resp.Text(),
            TokenUsed:    resp.Usage().TotalTokens,
            FinishReason: string(resp.FinishReason()),
        }, nil
    },
)

// buildPrompt 构建提示词
func buildPrompt(context ContextBuildOutput, userMessage string) string {
    var prompt strings.Builder
    
    // 添加系统提示
    prompt.WriteString("你是一个智能助手。请基于以下上下文回答用户问题。\n\n")
    
    // 添加摘要
    if context.Summary != nil {
        prompt.WriteString("对话摘要：\n")
        prompt.WriteString(context.Summary.Content)
        prompt.WriteString("\n\n")
    }
    
    // 添加相关记忆
    if len(context.RelevantMemories) > 0 {
        prompt.WriteString("相关历史信息：\n")
        for _, mem := range context.RelevantMemories {
            prompt.WriteString(fmt.Sprintf("- %s\n", mem.Content))
        }
        prompt.WriteString("\n")
    }
    
    // 添加最近的对话
    if len(context.Messages) > 0 {
        prompt.WriteString("最近的对话：\n")
        for _, msg := range context.Messages {
            prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
        }
        prompt.WriteString("\n")
    }
    
    // 添加用户消息
    prompt.WriteString(fmt.Sprintf("用户: %s\n", userMessage))
    prompt.WriteString("助手: ")
    
    return prompt.String()
}

// getModelConfig 获取模型配置
func getModelConfig(config *ModelConfig) ai.GenerateConfig {
    if config == nil {
        return ai.GenerateConfig{
            Temperature: 0.7,
            TopP:        0.9,
            MaxTokens:   2000,
        }
    }
    
    genConfig := ai.GenerateConfig{}
    if config.Temperature != nil {
        genConfig.Temperature = *config.Temperature
    }
    if config.TopP != nil {
        genConfig.TopP = *config.TopP
    }
    if config.MaxTokens != nil {
        genConfig.MaxTokens = *config.MaxTokens
    }
    
    return genConfig
}

// saveMessages 保存消息
func saveMessages(ctx context.Context, sessionID, userMsg, assistantMsg string) (string, error) {
    // 保存用户消息和助手响应到数据库
    // 实现省略
    return uuid.New().String(), nil
}

// processMessageAsync 异步处理消息
func processMessageAsync(sessionID, messageID string) {
    ctx := context.Background()
    
    // 1. 生成向量并存储
    // 2. 检查是否需要生成摘要
    // 3. 更新上下文统计
    
    log.Printf("异步处理消息: session=%s, message=%s", sessionID, messageID)
}
```

## 5. 摘要生成 Flow

### 5.1 摘要生成输入输出定义

```go
// SummaryGenerateInput 摘要生成输入
type SummaryGenerateInput struct {
    SessionID       string   `json:"sessionId"`
    MessageIDs      []string `json:"messageIds"`      // 需要摘要的消息ID列表
    PreviousSummary *string  `json:"previousSummary"` // 之前的摘要（增量摘要）
}

// SummaryGenerateOutput 摘要生成输出
type SummaryGenerateOutput struct {
    SummaryID      string `json:"summaryId"`
    Summary        string `json:"summary"`
    TokenCount     int    `json:"tokenCount"`
    MessageCount   int    `json:"messageCount"`
    LastMessageID  string `json:"lastMessageId"`
}
```

### 5.2 摘要生成 Flow 实现

```go
// SummaryGenerateFlow 摘要生成 Flow
var SummaryGenerateFlow = genkit.DefineFlow(g, "summaryGenerateFlow",
    func(ctx context.Context, input SummaryGenerateInput) (SummaryGenerateOutput, error) {
        // 1. 获取需要摘要的消息
        messages, err := getMessagesByIDs(ctx, input.MessageIDs)
        if err != nil {
            return SummaryGenerateOutput{}, fmt.Errorf("获取消息失败: %w", err)
        }
        
        if len(messages) == 0 {
            return SummaryGenerateOutput{}, fmt.Errorf("没有需要摘要的消息")
        }
        
        // 2. 构建摘要提示词
        prompt := buildSummaryPrompt(input.PreviousSummary, messages)
        
        // 3. 调用 AI 生成摘要
        resp, err := genkit.Generate(ctx, g,
            ai.WithPrompt(prompt),
            ai.WithConfig(ai.GenerateConfig{
                Temperature: 0.3, // 较低温度保证稳定性
                MaxTokens:   500,
            }),
        )
        if err != nil {
            return SummaryGenerateOutput{}, fmt.Errorf("生成摘要失败: %w", err)
        }
        
        summaryText := resp.Text()
        
        // 4. 保存摘要到数据库
        summaryID, err := saveSummary(ctx, input.SessionID, summaryText, messages[len(messages)-1].ID)
        if err != nil {
            return SummaryGenerateOutput{}, fmt.Errorf("保存摘要失败: %w", err)
        }
        
        // 5. 更新会话的上下文配置
        updateContextConfig(ctx, input.SessionID, summaryText, messages[len(messages)-1].ID)
        
        return SummaryGenerateOutput{
            SummaryID:     summaryID,
            Summary:       summaryText,
            TokenCount:    resp.Usage().TotalTokens,
            MessageCount:  len(messages),
            LastMessageID: messages[len(messages)-1].ID,
        }, nil
    },
)

// buildSummaryPrompt 构建摘要提示词
func buildSummaryPrompt(previousSummary *string, messages []ChatMessage) string {
    var prompt strings.Builder
    
    prompt.WriteString("请对以下对话内容生成简洁的摘要，保留关键信息和上下文。\n\n")
    
    if previousSummary != nil && *previousSummary != "" {
        prompt.WriteString("之前的对话摘要：\n")
        prompt.WriteString(*previousSummary)
        prompt.WriteString("\n\n")
        prompt.WriteString("新的对话内容：\n")
    } else {
        prompt.WriteString("对话内容：\n")
    }
    
    for _, msg := range messages {
        prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
    }
    
    prompt.WriteString("\n要求：\n")
    prompt.WriteString("1. 摘要长度控制在200字以内\n")
    prompt.WriteString("2. 保留对话的主题和关键信息\n")
    prompt.WriteString("3. 突出重要的结论和决策\n")
    prompt.WriteString("4. 使用简洁清晰的语言\n\n")
    prompt.WriteString("摘要：")
    
    return prompt.String()
}

// getMessagesByIDs 根据ID列表获取消息
func getMessagesByIDs(ctx context.Context, messageIDs []string) ([]ChatMessage, error) {
    // 从数据库获取消息
    // 实现省略
    return nil, nil
}

// saveSummary 保存摘要
func saveSummary(ctx context.Context, sessionID, summary, lastMessageID string) (string, error) {
    // 保存摘要到数据库
    // 实现省略
    return uuid.New().String(), nil
}

// updateContextConfig 更新上下文配置
func updateContextConfig(ctx context.Context, sessionID, summary, lastMessageID string) error {
    // 更新会话的上下文配置
    // 实现省略
    return nil
}
```

## 6. 记忆检索 Flow

### 6.1 记忆检索输入输出定义

```go
// MemorySearchInput 记忆检索输入
type MemorySearchInput struct {
    SessionID string `json:"sessionId"`
    Query     string `json:"query"`
    TopK      int    `json:"topK"`      // 返回前K个结果
    MinScore  float32 `json:"minScore"` // 最小相似度分数
}

// MemorySearchOutput 记忆检索输出
type MemorySearchOutput struct {
    Memories      []MemoryResult `json:"memories"`
    TotalFound    int            `json:"totalFound"`
    SearchTime    int64          `json:"searchTime"` // 毫秒
}

// MemoryResult 记忆检索结果
type MemoryResult struct {
    ID         string  `json:"id"`
    Content    string  `json:"content"`
    TokenCount int     `json:"tokenCount"`
    Importance float32 `json:"importance"`
    Similarity float32 `json:"similarity"`
    CreatedAt  string  `json:"createdAt"`
}
```

### 6.2 记忆检索 Flow 实现

```go
// MemorySearchFlow 记忆检索 Flow
var MemorySearchFlow = genkit.DefineFlow(g, "memorySearchFlow",
    func(ctx context.Context, input MemorySearchInput) (MemorySearchOutput, error) {
        startTime := time.Now()
        
        // 1. 验证输入
        if input.Query == "" {
            return MemorySearchOutput{}, fmt.Errorf("查询不能为空")
        }
        
        if input.TopK <= 0 {
            input.TopK = 5
        }
        
        if input.MinScore <= 0 {
            input.MinScore = 0.7 // 默认最小相似度
        }
        
        // 2. 生成查询向量
        embedding, err := generateEmbedding(ctx, input.Query)
        if err != nil {
            return MemorySearchOutput{}, fmt.Errorf("生成查询向量失败: %w", err)
        }
        
        // 3. 向量相似度搜索
        memories, err := vectorSearch(ctx, input.SessionID, embedding, input.TopK)
        if err != nil {
            return MemorySearchOutput{}, fmt.Errorf("向量搜索失败: %w", err)
        }
        
        // 4. 过滤低相似度结果
        filteredMemories := make([]MemoryResult, 0)
        for _, mem := range memories {
            if mem.Similarity >= input.MinScore {
                filteredMemories = append(filteredMemories, mem)
            }
        }
        
        // 5. 更新访问统计
        go updateMemoryAccessStats(ctx, filteredMemories)
        
        searchTime := time.Since(startTime).Milliseconds()
        
        return MemorySearchOutput{
            Memories:   filteredMemories,
            TotalFound: len(filteredMemories),
            SearchTime: searchTime,
        }, nil
    },
)

// generateEmbedding 生成向量嵌入
func generateEmbedding(ctx context.Context, text string) ([]float32, error) {
    // 调用嵌入服务生成向量
    // 可以使用 OpenAI、Google AI 等服务
    // 实现省略
    return nil, nil
}

// vectorSearch 向量相似度搜索
func vectorSearch(ctx context.Context, sessionID string, embedding []float32, topK int) ([]MemoryResult, error) {
    // 使用 pgvector 进行相似度搜索
    // 实现省略
    return nil, nil
}

// updateMemoryAccessStats 更新记忆访问统计
func updateMemoryAccessStats(ctx context.Context, memories []MemoryResult) {
    for _, mem := range memories {
        // 更新访问次数和最后访问时间
        // 实现省略
        log.Printf("更新记忆访问统计: %s", mem.ID)
    }
}
```

## 7. 流式对话 Flow

### 7.1 流式对话实现

```go
// ChatStreamInput 流式对话输入
type ChatStreamInput struct {
    SessionID   string       `json:"sessionId"`
    UserMessage string       `json:"userMessage"`
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
}

// ChatStreamChunk 流式响应块
type ChatStreamChunk struct {
    Delta     string `json:"delta"`     // 增量文本
    TokenUsed int    `json:"tokenUsed"` // 已使用的token
}

// ChatStreamOutput 流式对话完整输出
type ChatStreamOutput struct {
    MessageID    string `json:"messageId"`
    FullResponse string `json:"fullResponse"`
    TotalTokens  int    `json:"totalTokens"`
    FinishReason string `json:"finishReason"`
}

// ChatStreamFlow 流式对话 Flow
var ChatStreamFlow = genkit.DefineStreamingFlow(g, "chatStreamFlow",
    func(ctx context.Context, input ChatStreamInput, callback core.StreamCallback[ChatStreamChunk]) (ChatStreamOutput, error) {
        // 1. 构建上下文
        contextInput := ContextBuildInput{
            SessionID: input.SessionID,
            UserQuery: input.UserMessage,
            MaxTokens: 4000,
        }
        
        context, err := BuildContextFlow.Run(ctx, contextInput)
        if err != nil {
            return ChatStreamOutput{}, fmt.Errorf("构建上下文失败: %w", err)
        }
        
        // 2. 构建提示词
        prompt := buildPrompt(context, input.UserMessage)
        
        // 3. 配置模型参数
        modelConfig := getModelConfig(input.ModelConfig)
        
        // 4. 流式生成
        var fullResponse strings.Builder
        var totalTokens int
        var finishReason string
        
        resp, err := genkit.Generate(ctx, g,
            ai.WithPrompt(prompt),
            ai.WithConfig(modelConfig),
            ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
                // 获取增量文本
                delta := chunk.Text()
                fullResponse.WriteString(delta)
                
                // 发送流式响应块
                streamChunk := ChatStreamChunk{
                    Delta:     delta,
                    TokenUsed: chunk.Usage().TotalTokens,
                }
                
                if err := callback(ctx, streamChunk); err != nil {
                    return fmt.Errorf("发送流式响应失败: %w", err)
                }
                
                return nil
            }),
        )
        
        if err != nil {
            return ChatStreamOutput{}, fmt.Errorf("生成响应失败: %w", err)
        }
        
        totalTokens = resp.Usage().TotalTokens
        finishReason = string(resp.FinishReason())
        
        // 5. 保存消息
        messageID, err := saveMessages(ctx, input.SessionID, input.UserMessage, fullResponse.String())
        if err != nil {
            log.Printf("保存消息失败: %v", err)
        }
        
        // 6. 异步处理
        go processMessageAsync(input.SessionID, messageID)
        
        return ChatStreamOutput{
            MessageID:    messageID,
            FullResponse: fullResponse.String(),
            TotalTokens:  totalTokens,
            FinishReason: finishReason,
        }, nil
    },
)
```

### 7.2 调用流式 Flow

```go
// 在 HTTP Handler 中调用流式 Flow
func HandleChatStream(c *gin.Context) {
    var input ChatStreamInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, gin.H{"error": "请求参数错误"})
        return
    }
    
    // 设置 SSE 响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    
    // 调用流式 Flow
    streamCh, err := ChatStreamFlow.Stream(c.Request.Context(), input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 处理流式响应
    for result := range streamCh {
        if result.Err != nil {
            log.Printf("流式响应错误: %v", result.Err)
            break
        }
        
        if result.Done {
            // 发送完成事件
            data, _ := json.Marshal(map[string]interface{}{
                "type":   "done",
                "output": result.Output,
            })
            fmt.Fprintf(c.Writer, "data: %s\n\n", data)
            c.Writer.Flush()
        } else {
            // 发送流式数据块
            data, _ := json.Marshal(map[string]interface{}{
                "type":  "chunk",
                "chunk": result.Stream,
            })
            fmt.Fprintf(c.Writer, "data: %s\n\n", data)
            c.Writer.Flush()
        }
    }
}
```

## 8. Flow 组合与编排

### 8.1 复合 Flow 示例

```go
// CompleteConversationInput 完整对话输入
type CompleteConversationInput struct {
    SessionID   string       `json:"sessionId"`
    UserMessage string       `json:"userMessage"`
    ModelConfig *ModelConfig `json:"modelConfig,omitempty"`
}

// CompleteConversationOutput 完整对话输出
type CompleteConversationOutput struct {
    Context      ContextBuildOutput  `json:"context"`
    ChatResponse ChatGenerateOutput  `json:"chatResponse"`
    Summary      *SummaryGenerateOutput `json:"summary,omitempty"`
}

// CompleteConversationFlow 完整对话流程（组合多个 Flow）
var CompleteConversationFlow = genkit.DefineFlow(g, "completeConversationFlow",
    func(ctx context.Context, input CompleteConversationInput) (CompleteConversationOutput, error) {
        // 1. 构建上下文
        contextInput := ContextBuildInput{
            SessionID: input.SessionID,
            UserQuery: input.UserMessage,
            MaxTokens: 4000,
        }
        
        context, err := BuildContextFlow.Run(ctx, contextInput)
        if err != nil {
            return CompleteConversationOutput{}, fmt.Errorf("构建上下文失败: %w", err)
        }
        
        // 2. 生成对话响应
        chatInput := ChatGenerateInput{
            SessionID:   input.SessionID,
            UserMessage: input.UserMessage,
            Context:     &context,
            ModelConfig: input.ModelConfig,
        }
        
        chatResponse, err := ChatGenerateFlow.Run(ctx, chatInput)
        if err != nil {
            return CompleteConversationOutput{}, fmt.Errorf("生成对话失败: %w", err)
        }
        
        output := CompleteConversationOutput{
            Context:      context,
            ChatResponse: chatResponse,
        }
        
        // 3. 检查是否需要生成摘要
        shouldSummarize, messageIDs := checkShouldSummarize(ctx, input.SessionID)
        if shouldSummarize {
            summaryInput := SummaryGenerateInput{
                SessionID:  input.SessionID,
                MessageIDs: messageIDs,
            }
            
            summary, err := SummaryGenerateFlow.Run(ctx, summaryInput)
            if err != nil {
                log.Printf("生成摘要失败: %v", err)
                // 不中断主流程
            } else {
                output.Summary = &summary
            }
        }
        
        return output, nil
    },
)

// checkShouldSummarize 检查是否需要生成摘要
func checkShouldSummarize(ctx context.Context, sessionID string) (bool, []string) {
    // 检查自上次摘要后的消息数量
    // 如果超过阈值（如20条），返回需要摘要的消息ID列表
    // 实现省略
    return false, nil
}
```

### 8.2 并行执行 Flow

```go
// ParallelProcessInput 并行处理输入
type ParallelProcessInput struct {
    SessionID string `json:"sessionId"`
    Query     string `json:"query"`
}

// ParallelProcessOutput 并行处理输出
type ParallelProcessOutput struct {
    Context  ContextBuildOutput  `json:"context"`
    Memories MemorySearchOutput  `json:"memories"`
    Duration int64               `json:"duration"` // 毫秒
}

// ParallelProcessFlow 并行执行多个 Flow
var ParallelProcessFlow = genkit.DefineFlow(g, "parallelProcessFlow",
    func(ctx context.Context, input ParallelProcessInput) (ParallelProcessOutput, error) {
        startTime := time.Now()
        
        // 使用 goroutine 并行执行
        type contextResult struct {
            context ContextBuildOutput
            err     error
        }
        
        type memoryResult struct {
            memories MemorySearchOutput
            err      error
        }
        
        contextCh := make(chan contextResult, 1)
        memoryCh := make(chan memoryResult, 1)
        
        // 并行执行上下文构建
        go func() {
            contextInput := ContextBuildInput{
                SessionID: input.SessionID,
                UserQuery: input.Query,
                MaxTokens: 4000,
            }
            
            context, err := BuildContextFlow.Run(ctx, contextInput)
            contextCh <- contextResult{context: context, err: err}
        }()
        
        // 并行执行记忆检索
        go func() {
            memoryInput := MemorySearchInput{
                SessionID: input.SessionID,
                Query:     input.Query,
                TopK:      5,
                MinScore:  0.7,
            }
            
            memories, err := MemorySearchFlow.Run(ctx, memoryInput)
            memoryCh <- memoryResult{memories: memories, err: err}
        }()
        
        // 等待结果
        contextRes := <-contextCh
        memoryRes := <-memoryCh
        
        if contextRes.err != nil {
            return ParallelProcessOutput{}, fmt.Errorf("构建上下文失败: %w", contextRes.err)
        }
        
        if memoryRes.err != nil {
            log.Printf("检索记忆失败: %v", memoryRes.err)
            // 不中断流程
        }
        
        duration := time.Since(startTime).Milliseconds()
        
        return ParallelProcessOutput{
            Context:  contextRes.context,
            Memories: memoryRes.memories,
            Duration: duration,
        }, nil
    },
)
```

## 9. 部署配置

### 9.1 HTTP 服务器配置

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
    "github.com/firebase/genkit/go/plugins/server"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    
    // 初始化 Genkit
    g = genkit.Init(ctx, 
        genkit.WithPlugins(
            &googlegenai.GoogleAI{},
        ),
    )
    
    // 初始化所有 Flow
    initFlows(g)
    
    // 创建 Gin 路由
    router := gin.Default()
    
    // 注册 Flow 端点
    registerFlowEndpoints(router, g)
    
    // 注册业务 API 端点
    registerBusinessEndpoints(router)
    
    // 启动服务器
    log.Println("服务器启动在 :3400")
    if err := router.Run(":3400"); err != nil {
        log.Fatal(err)
    }
}

// initFlows 初始化所有 Flow
func initFlows(g *genkit.Genkit) {
    // Flow 已在各自的文件中定义
    log.Println("所有 Flow 已初始化")
}

// registerFlowEndpoints 注册 Flow 端点
func registerFlowEndpoints(router *gin.Engine, g *genkit.Genkit) {
    flowGroup := router.Group("/api/v1/flows")
    flowGroup.Use(AuthMiddleware()) // 认证中间件
    
    // 注册所有 Flow
    for _, flow := range genkit.ListFlows(g) {
        flowName := flow.Name()
        flowGroup.POST("/"+flowName, func(c *gin.Context) {
            genkit.Handler(flow)(c.Writer, c.Request)
        })
        log.Printf("注册 Flow 端点: POST /api/v1/flows/%s", flowName)
    }
}

// registerBusinessEndpoints 注册业务 API 端点
func registerBusinessEndpoints(router *gin.Engine) {
    api := router.Group("/api/v1")
    api.Use(AuthMiddleware())
    
    // 会话管理
    sessions := api.Group("/sessions")
    {
        sessions.POST("", CreateSession)
        sessions.GET("/:sessionId", GetSession)
        sessions.PUT("/:sessionId", UpdateSession)
        sessions.DELETE("/:sessionId", DeleteSession)
        
        // 上下文管理
        sessions.GET("/:sessionId/context", GetContext)
        sessions.POST("/:sessionId/context/compress", CompressContext)
        
        // 对话接口
        sessions.POST("/:sessionId/chat", HandleChat)
        sessions.POST("/:sessionId/chat/stream", HandleChatStream)
    }
    
    // 记忆管理
    memories := api.Group("/memories")
    {
        memories.GET("/search", SearchMemories)
        memories.DELETE("/:memoryId", DeleteMemory)
    }
}
```

### 9.2 业务 API Handler 实现

```go
// GetContext 获取会话上下文
func GetContext(c *gin.Context) {
    sessionID := c.Param("sessionId")
    userQuery := c.Query("query")
    
    // 调用 BuildContextFlow
    input := ContextBuildInput{
        SessionID: sessionID,
        UserQuery: userQuery,
        MaxTokens: 4000,
    }
    
    output, err := BuildContextFlow.Run(c.Request.Context(), input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "code":    200,
        "message": "获取上下文成功",
        "data":    output,
    })
}

// HandleChat 处理对话请求
func HandleChat(c *gin.Context) {
    sessionID := c.Param("sessionId")
    
    var req struct {
        Message     string       `json:"message" binding:"required"`
        ModelConfig *ModelConfig `json:"modelConfig"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "请求参数错误"})
        return
    }
    
    // 调用 ChatGenerateFlow
    input := ChatGenerateInput{
        SessionID:   sessionID,
        UserMessage: req.Message,
        ModelConfig: req.ModelConfig,
    }
    
    output, err := ChatGenerateFlow.Run(c.Request.Context(), input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "code":    200,
        "message": "对话成功",
        "data":    output,
    })
}

// SearchMemories 搜索记忆
func SearchMemories(c *gin.Context) {
    sessionID := c.Query("sessionId")
    query := c.Query("query")
    topK, _ := strconv.Atoi(c.DefaultQuery("topK", "5"))
    
    if sessionID == "" || query == "" {
        c.JSON(400, gin.H{"error": "sessionId 和 query 不能为空"})
        return
    }
    
    // 调用 MemorySearchFlow
    input := MemorySearchInput{
        SessionID: sessionID,
        Query:     query,
        TopK:      topK,
        MinScore:  0.7,
    }
    
    output, err := MemorySearchFlow.Run(c.Request.Context(), input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "code":    200,
        "message": "搜索成功",
        "data":    output,
    })
}
```

### 9.3 环境配置

```bash
# .env 文件
# Genkit 配置
GENKIT_ENV=production
GENKIT_LOG_LEVEL=info

# Google AI 配置
GOOGLE_AI_API_KEY=your_api_key_here
GOOGLE_AI_MODEL=gemini-1.5-flash

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ai_chat
DB_USER=postgres
DB_PASSWORD=your_password

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# 服务器配置
SERVER_PORT=3400
SERVER_HOST=0.0.0.0

# 向量嵌入配置
EMBEDDING_MODEL=text-embedding-ada-002
EMBEDDING_DIMENSION=1536

# 上下文配置
MAX_CONTEXT_TOKENS=4000
SUMMARY_THRESHOLD=20
MEMORY_RETENTION_DAYS=30
```

### 9.4 Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 复制构建的二进制文件
COPY --from=builder /app/server .

# 复制配置文件
COPY .env .

EXPOSE 3400

CMD ["./server"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "3400:3400"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: ai_chat
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: your_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

## 10. 最佳实践

### 10.1 Flow 设计原则

**1. 单一职责**

```go
// ✅ 好的做法：每个 Flow 只做一件事
var BuildContextFlow = genkit.DefineFlow(g, "buildContextFlow", ...)
var ChatGenerateFlow = genkit.DefineFlow(g, "chatGenerateFlow", ...)
var SummaryGenerateFlow = genkit.DefineFlow(g, "summaryGenerateFlow", ...)

// ❌ 不好的做法：一个 Flow 做太多事情
var DoEverythingFlow = genkit.DefineFlow(g, "doEverythingFlow", ...)
```

**2. 类型安全**

```go
// ✅ 好的做法：使用明确的类型定义
type ChatInput struct {
    SessionID string `json:"sessionId"`
    Message   string `json:"message"`
}

var chatFlow = genkit.DefineFlow(g, "chatFlow",
    func(ctx context.Context, input ChatInput) (ChatOutput, error) {
        // ...
    },
)

// ❌ 不好的做法：使用 map 或 interface{}
var chatFlow = genkit.DefineFlow(g, "chatFlow",
    func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
        // ...
    },
)
```

**3. 错误处理**

```go
// ✅ 好的做法：明确的错误处理和日志
func(ctx context.Context, input ChatInput) (ChatOutput, error) {
    context, err := BuildContextFlow.Run(ctx, contextInput)
    if err != nil {
        log.Printf("构建上下文失败: session=%s, error=%v", input.SessionID, err)
        return ChatOutput{}, fmt.Errorf("构建上下文失败: %w", err)
    }
    // ...
}

// ❌ 不好的做法：忽略错误
func(ctx context.Context, input ChatInput) (ChatOutput, error) {
    context, _ := BuildContextFlow.Run(ctx, contextInput)
    // ...
}
```

### 10.2 性能优化

**1. 使用缓存**

```go
// 缓存上下文构建结果
var cachedContextCache = make(map[string]ContextBuildOutput)
var cacheMutex sync.RWMutex

func getCachedContext(sessionID string) (ContextBuildOutput, bool) {
    cacheMutex.RLock()
    defer cacheMutex.RUnlock()
    
    context, ok := cachedContextCache[sessionID]
    return context, ok
}

func setCachedContext(sessionID string, context ContextBuildOutput) {
    cacheMutex.Lock()
    defer cacheMutex.Unlock()
    
    cachedContextCache[sessionID] = context
}
```

**2. 并行执行**

```go
// 并行执行独立的 Flow
go func() {
    // 异步生成向量
    generateEmbedding(ctx, message)
}()

go func() {
    // 异步检查摘要
    checkAndGenerateSummary(ctx, sessionID)
}()
```

**3. 流式响应**

```go
// 对于长响应，使用流式 Flow
var streamFlow = genkit.DefineStreamingFlow(g, "streamFlow",
    func(ctx context.Context, input Input, callback core.StreamCallback[Chunk]) (Output, error) {
        // 实时发送响应块
        callback(ctx, chunk)
    },
)
```

### 10.3 监控和调试

**1. 使用 Genkit Developer UI**

```bash
# 启动开发服务器
genkit start -- go run .

# 访问 http://localhost:4000
# 可以测试和调试所有 Flow
```

**2. 添加追踪日志**

```go
func(ctx context.Context, input ChatInput) (ChatOutput, error) {
    log.Printf("[TRACE] 开始处理对话: session=%s", input.SessionID)
    
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        log.Printf("[TRACE] 对话处理完成: session=%s, duration=%v", input.SessionID, duration)
    }()
    
    // Flow 逻辑
}
```

**3. 性能指标收集**

```go
// 使用 Prometheus 收集指标
var (
    flowDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "genkit_flow_duration_seconds",
            Help: "Flow 执行时间",
        },
        []string{"flow_name"},
    )
    
    flowErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "genkit_flow_errors_total",
            Help: "Flow 错误次数",
        },
        []string{"flow_name"},
    )
)

func recordFlowMetrics(flowName string, duration time.Duration, err error) {
    flowDuration.WithLabelValues(flowName).Observe(duration.Seconds())
    if err != nil {
        flowErrors.WithLabelValues(flowName).Inc()
    }
}
```

### 10.4 安全最佳实践

**1. 输入验证**

```go
func(ctx context.Context, input ChatInput) (ChatOutput, error) {
    // 验证输入
    if input.SessionID == "" {
        return ChatOutput{}, fmt.Errorf("sessionId 不能为空")
    }
    
    if len(input.Message) > 10000 {
        return ChatOutput{}, fmt.Errorf("消息长度超过限制")
    }
    
    // 清理输入
    input.Message = sanitizeInput(input.Message)
    
    // ...
}
```

**2. 权限验证**

```go
func(ctx context.Context, input ChatInput) (ChatOutput, error) {
    // 从上下文获取用户信息
    claims := middleware.GetJWTClaims(ctx)
    if claims == nil {
        return ChatOutput{}, fmt.Errorf("未认证")
    }
    
    // 验证会话访问权限
    if err := validateSessionAccess(ctx, input.SessionID, claims.Subject); err != nil {
        return ChatOutput{}, fmt.Errorf("权限不足: %w", err)
    }
    
    // ...
}
```

**3. 速率限制**

```go
// 使用 Redis 实现速率限制
func checkRateLimit(ctx context.Context, userID string) error {
    key := fmt.Sprintf("rate_limit:%s", userID)
    
    count, err := redisClient.Incr(ctx, key).Result()
    if err != nil {
        return err
    }
    
    if count == 1 {
        redisClient.Expire(ctx, key, time.Minute)
    }
    
    if count > 60 { // 每分钟最多 60 次请求
        return fmt.Errorf("请求过于频繁，请稍后再试")
    }
    
    return nil
}
```

## 11. 测试

### 11.1 单元测试

```go
package flows

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestBuildContextFlow(t *testing.T) {
    ctx := context.Background()
    
    input := ContextBuildInput{
        SessionID: "test-session-id",
        UserQuery: "测试查询",
        MaxTokens: 4000,
    }
    
    output, err := BuildContextFlow.Run(ctx, input)
    
    assert.NoError(t, err)
    assert.Equal(t, input.SessionID, output.SessionID)
    assert.NotEmpty(t, output.Messages)
    assert.LessOrEqual(t, output.TotalTokens, input.MaxTokens)
}

func TestChatGenerateFlow(t *testing.T) {
    ctx := context.Background()
    
    input := ChatGenerateInput{
        SessionID:   "test-session-id",
        UserMessage: "你好",
    }
    
    output, err := ChatGenerateFlow.Run(ctx, input)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, output.Response)
    assert.NotEmpty(t, output.MessageID)
    assert.Greater(t, output.TokenUsed, 0)
}
```

### 11.2 集成测试

```go
func TestCompleteConversationFlow(t *testing.T) {
    ctx := context.Background()
    
    // 创建测试会话
    sessionID := createTestSession(t)
    defer cleanupTestSession(t, sessionID)
    
    // 测试完整对话流程
    input := CompleteConversationInput{
        SessionID:   sessionID,
        UserMessage: "介绍一下人工智能",
    }
    
    output, err := CompleteConversationFlow.Run(ctx, input)
    
    assert.NoError(t, err)
    assert.NotNil(t, output.Context)
    assert.NotEmpty(t, output.ChatResponse.Response)
    
    // 验证消息已保存
    messages := getSessionMessages(t, sessionID)
    assert.GreaterOrEqual(t, len(messages), 2) // 用户消息 + 助手响应
}
```

## 12. 总结

通过集成 Google Genkit Flow，我们的 AI 对话会话管理系统获得了以下优势：

1. **类型安全**：编译时和运行时的类型检查
2. **可观测性**：内置的追踪和调试功能
3. **流式支持**：原生的流式响应能力
4. **易于部署**：简化的 HTTP 端点部署
5. **开发体验**：Developer UI 提供的测试和调试工具

### 关键要点

- 使用 Flow 封装 AI 逻辑，提高可维护性
- 合理组合和编排 Flow，实现复杂功能
- 利用流式 Flow 提升用户体验
- 遵循最佳实践，确保性能和安全
- 充分利用 Genkit 的开发工具

### 下一步

1. 实现更多专用 Flow（如意图识别、情感分析等）
2. 添加更多的监控和告警
3. 优化 Flow 的性能和资源使用
4. 扩展 Flow 的功能（如多模态支持）

---

**文档版本**: v1.0  
**最后更新**: 2025-10-29  
**维护者**: AI Platform Team
