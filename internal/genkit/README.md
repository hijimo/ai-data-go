# Genkit 客户端封装

本模块封装了 Firebase Genkit SDK，提供了简洁的接口用于 AI 内容生成。

## 功能特性

- ✅ 配置管理：支持从配置结构初始化客户端
- ✅ 参数映射：将 ChatOptions 转换为 Genkit 的生成选项
- ✅ 文本生成：支持基本的文本生成功能
- ✅ Token 统计：返回 token 使用情况
- ✅ 灵活配置：支持自定义温度、最大 token 数、TopP、TopK 等参数

## 使用方法

### 1. 创建和初始化客户端

```go
import (
    "context"
    "genkit-ai-service/internal/genkit"
)

// 创建客户端
client := genkit.NewClient()

// 配置客户端
config := &genkit.Config{
    APIKey:             "your-api-key",
    Model:              "gemini-2.5-flash",
    DefaultTemperature: 0.7,
    DefaultMaxTokens:   2000,
}

// 初始化客户端
err := client.Initialize(context.Background(), config)
if err != nil {
    log.Fatalf("初始化失败: %v", err)
}
```

### 2. 设置模型

在实际使用中，需要通过具体的 Genkit 插件（如 googleai）来创建模型实例：

```go
import (
    "github.com/firebase/genkit/go/plugins/googleai"
)

// 创建模型实例（示例）
model := googleai.Model("gemini-2.5-flash")

// 设置到客户端
client.SetModel(model)
```

### 3. 生成内容

#### 使用默认配置

```go
result, err := client.Generate(
    context.Background(),
    "你好，请介绍一下 Firebase",
    nil, // 使用默认配置
)
if err != nil {
    log.Fatalf("生成失败: %v", err)
}

fmt.Println("生成的内容:", result.Text)
fmt.Printf("使用的 token: %d\n", result.Usage.TotalTokens)
```

#### 使用自定义选项

```go
temp := 0.9
maxTokens := 1000
topP := 0.95
topK := 40

options := &genkit.GenerateOptions{
    Temperature: &temp,
    MaxTokens:   &maxTokens,
    TopP:        &topP,
    TopK:        &topK,
}

result, err := client.Generate(
    context.Background(),
    "你好，请介绍一下 Firebase",
    options,
)
if err != nil {
    log.Fatalf("生成失败: %v", err)
}

fmt.Println("生成的内容:", result.Text)
```

## 配置说明

### Config 结构

```go
type Config struct {
    APIKey             string  // API 密钥（必填）
    Model              string  // 模型名称（必填）
    DefaultTemperature float64 // 默认温度值
    DefaultMaxTokens   int     // 默认最大 token 数
}
```

### GenerateOptions 结构

```go
type GenerateOptions struct {
    Temperature *float64 // 温度值，控制输出的随机性 (0-2)
    MaxTokens   *int     // 最大 token 数
    TopP        *float64 // Top-p 采样参数 (0-1)
    TopK        *int     // Top-k 采样参数
}
```

### GenerateResult 结构

```go
type GenerateResult struct {
    Text  string // 生成的文本内容
    Model string // 使用的模型
    Usage *Usage // Token 使用情况
}

type Usage struct {
    PromptTokens     int // 提示词 token 数
    CompletionTokens int // 生成内容 token 数
    TotalTokens      int // 总 token 数
}
```

## 参数说明

### Temperature（温度）

- 范围：0-2
- 默认值：0.7
- 说明：控制输出的随机性，值越高输出越随机，值越低输出越确定

### MaxTokens（最大 token 数）

- 范围：> 0
- 默认值：2000
- 说明：限制生成内容的最大长度

### TopP

- 范围：0-1
- 说明：核采样参数，控制从累积概率达到 TopP 的最小 token 集合中采样

### TopK

- 范围：> 0
- 说明：从概率最高的 K 个 token 中采样

## 错误处理

客户端会返回以下类型的错误：

- 配置错误：配置为空、API 密钥为空、模型名称为空
- 初始化错误：客户端未初始化、模型未设置
- 参数错误：提示词为空
- 生成错误：AI 模型生成失败

## 注意事项

1. 在调用 `Generate` 方法之前，必须先调用 `Initialize` 方法初始化客户端
2. 必须通过 `SetModel` 方法设置实际的模型实例
3. 所有可选参数（Temperature、MaxTokens 等）如果不设置，将使用配置中的默认值
4. 建议在生产环境中使用 context 的超时控制，避免请求长时间挂起

## 测试

运行单元测试：

```bash
go test ./internal/genkit/... -v
```

查看测试覆盖率：

```bash
go test ./internal/genkit/... -cover
```
