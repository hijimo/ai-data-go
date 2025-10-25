package main

import (
"context"
"encoding/json"
"fmt"
"genkit-ai-service/pkg/errors"
"genkit-ai-service/pkg/response"
)

func main() {
	// 创建一个带 traceId 的 context
	ctx := context.WithValue(context.Background(), "traceId", "test-trace-123")
	
	// 测试错误响应
	resp := response.ErrorWithContext[any](ctx, errors.CodeUnauthorized, "测试错误")
	
	// 序列化为 JSON
	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(jsonData))
}
