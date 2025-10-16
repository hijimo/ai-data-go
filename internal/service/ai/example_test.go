package ai_test

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/service/ai"
)

// ExampleContextManager_CreateSession 演示如何创建会话
func ExampleContextManager_CreateSession() {
	// 创建上下文管理器，设置30分钟超时和5分钟清理间隔
	cm := ai.NewContextManager(30*time.Minute, 5*time.Minute)

	// 创建新会话
	ctx := context.Background()
	sessionID, sessionCtx, cancel := cm.CreateSession(ctx)
	defer cancel()

	// 验证会话ID不为空
	fmt.Printf("会话ID不为空: %v\n", sessionID != "")
	fmt.Printf("会话上下文有效: %v\n", sessionCtx.Err() == nil)

	// Output:
	// 会话ID不为空: true
	// 会话上下文有效: true
}

// ExampleContextManager_GetSession 演示如何获取会话
func ExampleContextManager_GetSession() {
	cm := ai.NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	// 创建会话
	sessionID, _, cancel := cm.CreateSession(ctx)
	defer cancel()

	// 获取会话
	sessionCtx, exists := cm.GetSession(sessionID)
	if exists {
		fmt.Printf("会话存在: %v\n", sessionCtx != nil)
	}

	// 尝试获取不存在的会话
	_, exists = cm.GetSession("non-existent-id")
	fmt.Printf("不存在的会话: %v\n", exists)

	// Output:
	// 会话存在: true
	// 不存在的会话: false
}

// ExampleContextManager_CancelSession 演示如何取消会话
func ExampleContextManager_CancelSession() {
	cm := ai.NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	// 创建会话
	sessionID, sessionCtx, _ := cm.CreateSession(ctx)

	// 取消会话
	err := cm.CancelSession(sessionID)
	if err != nil {
		fmt.Printf("取消失败: %v\n", err)
		return
	}

	fmt.Printf("会话已取消: %v\n", sessionCtx.Err() != nil)

	// 验证会话已被清理
	_, exists := cm.GetSession(sessionID)
	fmt.Printf("会话已清理: %v\n", !exists)

	// Output:
	// 会话已取消: true
	// 会话已清理: true
}

// ExampleContextManager_lifecycle 演示完整的会话生命周期
func ExampleContextManager_lifecycle() {
	// 创建管理器并启动自动清理
	cm := ai.NewContextManager(30*time.Minute, 5*time.Minute)
	cm.Start()
	defer cm.Stop()

	ctx := context.Background()

	// 创建会话
	sessionID, sessionCtx, cancel := cm.CreateSession(ctx)

	// 使用会话进行操作
	select {
	case <-sessionCtx.Done():
		fmt.Println("会话已取消")
	case <-time.After(100 * time.Millisecond):
		fmt.Println("会话正常运行")
	}

	// 手动取消会话
	cancel()

	// 验证会话已清理
	_, exists := cm.GetSession(sessionID)
	fmt.Printf("会话已清理: %v\n", !exists)

	// Output:
	// 会话正常运行
	// 会话已清理: true
}
