package tracing_test

import (
	"context"
	"errors"
	"fmt"

	"genkit-ai-service/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Example_flowTracing 演示如何追踪 Flow 执行
func Example_flowTracing() {
	ctx := context.Background()

	// 追踪 Flow 执行
	err := tracing.TraceFlow(ctx, "contextBuildFlow", func(ctx context.Context) error {
		// Flow 执行逻辑
		fmt.Println("执行 contextBuildFlow")
		return nil
	})

	if err != nil {
		fmt.Printf("Flow 执行失败: %v\n", err)
	}

	// Output:
	// 执行 contextBuildFlow
}

// Example_serviceTracing 演示如何追踪服务层方法
func Example_serviceTracing() {
	ctx := context.Background()

	// 追踪服务方法
	err := tracing.TraceService(ctx, "ContextService", "BuildContext", func(ctx context.Context) error {
		// 服务逻辑
		fmt.Println("构建上下文")
		return nil
	})

	if err != nil {
		fmt.Printf("服务执行失败: %v\n", err)
	}

	// Output:
	// 构建上下文
}

// Example_repositoryTracing 演示如何追踪数据库操作
func Example_repositoryTracing() {
	ctx := context.Background()

	// 追踪数据库操作
	err := tracing.TraceRepository(ctx, "MemoryRepository", "SearchByVector", func(ctx context.Context) error {
		// 数据库查询
		fmt.Println("执行向量检索")
		return nil
	})

	if err != nil {
		fmt.Printf("数据库操作失败: %v\n", err)
	}

	// Output:
	// 执行向量检索
}

// Example_externalCallTracing 演示如何追踪外部服务调用
func Example_externalCallTracing() {
	ctx := context.Background()

	// 追踪外部服务调用
	err := tracing.TraceExternalCall(ctx, "OpenAI", "ChatCompletion", func(ctx context.Context) error {
		// 调用外部 API
		fmt.Println("调用 OpenAI API")
		return nil
	})

	if err != nil {
		fmt.Printf("外部调用失败: %v\n", err)
	}

	// Output:
	// 调用 OpenAI API
}

// Example_customAttributes 演示如何添加自定义属性
func Example_customAttributes() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.StartSpan(ctx, "processData")
	defer span.End()

	// 添加自定义属性
	tracing.AddSpanAttributes(ctx,
		attribute.String("data.id", "data-123"),
		attribute.Int("data.size", 1024),
		attribute.String("data.type", "json"),
	)

	fmt.Println("处理数据")

	// Output:
	// 处理数据
}

// Example_spanEvents 演示如何添加事件
func Example_spanEvents() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.StartSpan(ctx, "multiStepProcess")
	defer span.End()

	// 步骤 1
	tracing.AddSpanEvent(ctx, "step1.started")
	fmt.Println("执行步骤 1")
	tracing.AddSpanEvent(ctx, "step1.completed")

	// 步骤 2
	tracing.AddSpanEvent(ctx, "step2.started")
	fmt.Println("执行步骤 2")
	tracing.AddSpanEvent(ctx, "step2.completed")

	// Output:
	// 执行步骤 1
	// 执行步骤 2
}

// Example_errorHandling 演示如何处理错误
func Example_errorHandling() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.StartSpan(ctx, "operationWithError")
	defer span.End()

	// 模拟错误
	err := errors.New("操作失败")
	if err != nil {
		tracing.RecordError(ctx, err)
		fmt.Printf("发生错误: %v\n", err)
	}

	// Output:
	// 发生错误: 操作失败
}

// Example_tokenUsage 演示如何记录 Token 使用
func Example_tokenUsage() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.TraceAIGeneration(ctx, "gpt-4", 1500)
	defer span.End()

	// 模拟 AI 生成
	fmt.Println("生成 AI 响应")

	// 记录 Token 使用
	tracing.AddTokenUsage(ctx, 1500, 800, 2300)
	tracing.SetSpanSuccess(span)

	// Output:
	// 生成 AI 响应
}

// Example_contextMetrics 演示如何记录上下文指标
func Example_contextMetrics() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.StartSpan(ctx, "buildContext")
	defer span.End()

	// 模拟构建上下文
	fmt.Println("构建上下文")

	// 记录指标
	tracing.AddContextMetrics(ctx, 2000, 0.85, "auto")
	tracing.SetSpanSuccess(span)

	// Output:
	// 构建上下文
}

// Example_memoryMetrics 演示如何记录记忆指标
func Example_memoryMetrics() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.TraceVectorSearch(ctx, "session-123", 10)
	defer span.End()

	// 模拟向量检索
	fmt.Println("检索记忆")

	// 记录指标
	tracing.AddMemoryMetrics(ctx, 5, 0.78)
	tracing.SetSpanSuccess(span)

	// Output:
	// 检索记忆
}

// Example_nestedSpans 演示如何创建嵌套 Span
func Example_nestedSpans() {
	ctx := context.Background()

	// 父 Span
	ctx, parentSpan := tracing.StartSpan(ctx, "parentOperation")
	defer parentSpan.End()

	fmt.Println("开始父操作")

	// 子 Span 1
	ctx, childSpan1 := tracing.StartSpan(ctx, "childOperation1")
	fmt.Println("执行子操作 1")
	tracing.SetSpanSuccess(childSpan1)
	childSpan1.End()

	// 子 Span 2
	ctx, childSpan2 := tracing.StartSpan(ctx, "childOperation2")
	fmt.Println("执行子操作 2")
	tracing.SetSpanSuccess(childSpan2)
	childSpan2.End()

	fmt.Println("完成父操作")
	tracing.SetSpanSuccess(parentSpan)

	// Output:
	// 开始父操作
	// 执行子操作 1
	// 执行子操作 2
	// 完成父操作
}

// Example_contextValues 演示如何使用上下文值
func Example_contextValues() {
	ctx := context.Background()

	// 设置上下文值
	ctx = context.WithValue(ctx, "session_id", "session-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	ctx = context.WithValue(ctx, "tenant_id", "tenant-789")

	// 追踪 Flow（会自动提取上下文值）
	err := tracing.TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		fmt.Println("执行 Flow")
		return nil
	})

	if err != nil {
		fmt.Printf("Flow 执行失败: %v\n", err)
	}

	// Output:
	// 执行 Flow
}

// Example_getTraceID 演示如何获取 TraceID
func Example_getTraceID() {
	ctx := context.Background()

	// 创建 Span
	ctx, span := tracing.StartSpan(ctx, "testSpan")
	defer span.End()

	// 获取 TraceID
	traceID := tracing.GetTraceID(ctx)
	if traceID != "" {
		fmt.Println("TraceID 已生成")
	}

	// Output:
	// TraceID 已生成
}

// Example_cacheOperation 演示如何追踪缓存操作
func Example_cacheOperation() {
	ctx := context.Background()

	// 追踪缓存获取
	ctx, span := tracing.TraceCacheOperation(ctx, "get", "context:session-123")
	defer span.End()

	// 模拟缓存操作
	fmt.Println("从缓存获取数据")
	tracing.SetSpanSuccess(span)

	// Output:
	// 从缓存获取数据
}

// Example_dbQuery 演示如何追踪数据库查询
func Example_dbQuery() {
	ctx := context.Background()

	// 追踪数据库查询
	ctx, span := tracing.TraceDBQuery(ctx, "SELECT * FROM users WHERE id = $1", "user-123")
	defer span.End()

	// 模拟数据库查询
	fmt.Println("执行数据库查询")
	tracing.SetSpanSuccess(span)

	// Output:
	// 执行数据库查询
}
