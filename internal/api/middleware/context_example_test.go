package middleware_test

import (
	"context"
	"fmt"
	"genkit-ai-service/internal/api/middleware"
)

// ExampleGenerateTraceID 演示如何生成 TraceID
func ExampleGenerateTraceID() {
	traceID := middleware.GenerateTraceID()
	fmt.Printf("生成的 TraceID 格式: trace-{timestamp}-{random}\n")
	fmt.Printf("TraceID 长度: %d\n", len(traceID))
	// Output:
	// 生成的 TraceID 格式: trace-{timestamp}-{random}
	// TraceID 长度: 29
}

// ExampleSetTraceID 演示如何将 TraceID 注入到 Context
func ExampleSetTraceID() {
	ctx := context.Background()
	traceID := "trace-1704067200-a3f9k2b8c1d4"
	
	// 注入 TraceID
	ctx = middleware.SetTraceID(ctx, traceID)
	
	// 验证注入成功
	retrievedID := middleware.GetTraceID(ctx)
	fmt.Printf("注入的 TraceID: %s\n", retrievedID)
	// Output:
	// 注入的 TraceID: trace-1704067200-a3f9k2b8c1d4
}

// ExampleGetTraceID 演示如何从 Context 提取 TraceID
func ExampleGetTraceID() {
	// 场景1：Context 中有 TraceID
	ctx := middleware.SetTraceID(context.Background(), "trace-1704067200-a3f9k2b8c1d4")
	traceID := middleware.GetTraceID(ctx)
	fmt.Printf("提取的 TraceID: %s\n", traceID)
	
	// 场景2：Context 中无 TraceID
	emptyCtx := context.Background()
	emptyTraceID := middleware.GetTraceID(emptyCtx)
	fmt.Printf("空 Context 的 TraceID: '%s'\n", emptyTraceID)
	
	// Output:
	// 提取的 TraceID: trace-1704067200-a3f9k2b8c1d4
	// 空 Context 的 TraceID: ''
}

// Example 演示完整的 TraceID 工作流程
func Example() {
	// 1. 生成 TraceID
	traceID := middleware.GenerateTraceID()
	fmt.Println("步骤1: 生成 TraceID")
	
	// 2. 注入到 Context
	ctx := middleware.SetTraceID(context.Background(), traceID)
	fmt.Println("步骤2: 注入到 Context")
	
	// 3. 在业务逻辑中传递 Context
	processRequest(ctx)
	
	// Output:
	// 步骤1: 生成 TraceID
	// 步骤2: 注入到 Context
	// 步骤3: 处理请求，TraceID 已传递
}

func processRequest(ctx context.Context) {
	// 在业务逻辑中可以随时提取 TraceID
	traceID := middleware.GetTraceID(ctx)
	if traceID != "" {
		fmt.Println("步骤3: 处理请求，TraceID 已传递")
	}
}
