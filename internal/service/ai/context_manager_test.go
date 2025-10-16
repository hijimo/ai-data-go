package ai

import (
	"context"
	"testing"
	"time"
)

func TestNewContextManager(t *testing.T) {
	timeout := 30 * time.Minute
	cleanupInterval := 5 * time.Minute

	cm := NewContextManager(timeout, cleanupInterval)
	if cm == nil {
		t.Fatal("NewContextManager 返回 nil")
	}
}

func TestCreateSession(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	sessionID, sessionCtx, cancel := cm.CreateSession(ctx)
	defer cancel()

	// 验证会话ID不为空
	if sessionID == "" {
		t.Error("会话ID为空")
	}

	// 验证上下文不为nil
	if sessionCtx == nil {
		t.Error("会话上下文为nil")
	}

	// 验证取消函数不为nil
	if cancel == nil {
		t.Error("取消函数为nil")
	}

	// 验证可以获取会话
	retrievedCtx, exists := cm.GetSession(sessionID)
	if !exists {
		t.Error("无法获取刚创建的会话")
	}
	if retrievedCtx != sessionCtx {
		t.Error("获取的上下文与创建的不一致")
	}
}

func TestGetSession(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	// 测试获取不存在的会话
	_, exists := cm.GetSession("non-existent-session")
	if exists {
		t.Error("不应该找到不存在的会话")
	}

	// 创建会话并获取
	sessionID, _, cancel := cm.CreateSession(ctx)
	defer cancel()

	retrievedCtx, exists := cm.GetSession(sessionID)
	if !exists {
		t.Error("应该找到存在的会话")
	}
	if retrievedCtx == nil {
		t.Error("获取的上下文不应该为nil")
	}
}

func TestCancelSession(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	// 测试取消不存在的会话
	err := cm.CancelSession("non-existent-session")
	if err == nil {
		t.Error("取消不存在的会话应该返回错误")
	}

	// 创建会话并取消
	sessionID, sessionCtx, _ := cm.CreateSession(ctx)

	err = cm.CancelSession(sessionID)
	if err != nil {
		t.Errorf("取消会话失败: %v", err)
	}

	// 验证上下文已被取消
	if sessionCtx.Err() == nil {
		t.Error("上下文应该已被取消")
	}

	// 验证会话已被清理
	_, exists := cm.GetSession(sessionID)
	if exists {
		t.Error("会话应该已被清理")
	}
}

func TestCleanupSession(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	sessionID, sessionCtx, _ := cm.CreateSession(ctx)

	// 清理会话
	cm.CleanupSession(sessionID)

	// 验证上下文已被取消
	if sessionCtx.Err() == nil {
		t.Error("上下文应该已被取消")
	}

	// 验证会话已被删除
	_, exists := cm.GetSession(sessionID)
	if exists {
		t.Error("会话应该已被删除")
	}
}

func TestSessionTimeout(t *testing.T) {
	// 使用较短的超时时间进行测试
	timeout := 100 * time.Millisecond
	cleanupInterval := 50 * time.Millisecond

	cm := NewContextManager(timeout, cleanupInterval)
	cm.Start()
	defer cm.Stop()

	ctx := context.Background()
	sessionID, _, _ := cm.CreateSession(ctx)

	// 等待超时
	time.Sleep(200 * time.Millisecond)

	// 验证会话已被自动清理
	_, exists := cm.GetSession(sessionID)
	if exists {
		t.Error("超时的会话应该已被自动清理")
	}
}

func TestMultipleSessions(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	// 创建多个会话
	sessions := make(map[string]context.Context)
	for i := 0; i < 5; i++ {
		sessionID, sessionCtx, _ := cm.CreateSession(ctx)
		sessions[sessionID] = sessionCtx
	}

	// 验证所有会话都可以获取
	for sessionID := range sessions {
		_, exists := cm.GetSession(sessionID)
		if !exists {
			t.Errorf("会话 %s 应该存在", sessionID)
		}
	}

	// 取消一个会话
	var firstSessionID string
	for sessionID := range sessions {
		firstSessionID = sessionID
		break
	}
	cm.CancelSession(firstSessionID)

	// 验证被取消的会话不存在
	_, exists := cm.GetSession(firstSessionID)
	if exists {
		t.Error("被取消的会话不应该存在")
	}

	// 验证其他会话仍然存在
	count := 0
	for sessionID := range sessions {
		if sessionID != firstSessionID {
			_, exists := cm.GetSession(sessionID)
			if exists {
				count++
			}
		}
	}
	if count != 4 {
		t.Errorf("应该有4个会话存在，实际有 %d 个", count)
	}
}

func TestStopCleansUpAllSessions(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	cm.Start()

	ctx := context.Background()

	// 创建多个会话
	sessionIDs := make([]string, 0)
	for i := 0; i < 3; i++ {
		sessionID, _, _ := cm.CreateSession(ctx)
		sessionIDs = append(sessionIDs, sessionID)
	}

	// 停止管理器
	cm.Stop()

	// 验证所有会话都被清理
	for _, sessionID := range sessionIDs {
		_, exists := cm.GetSession(sessionID)
		if exists {
			t.Errorf("会话 %s 应该已被清理", sessionID)
		}
	}
}

func TestWrappedCancelFunction(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	sessionID, sessionCtx, cancel := cm.CreateSession(ctx)

	// 调用包装的取消函数
	cancel()

	// 验证上下文已被取消
	if sessionCtx.Err() == nil {
		t.Error("上下文应该已被取消")
	}

	// 验证会话已被清理
	_, exists := cm.GetSession(sessionID)
	if exists {
		t.Error("会话应该已被清理")
	}
}

func TestConcurrentAccess(t *testing.T) {
	cm := NewContextManager(30*time.Minute, 5*time.Minute)
	ctx := context.Background()

	// 并发创建会话
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sessionID, _, cancel := cm.CreateSession(ctx)
			defer cancel()

			// 获取会话
			_, exists := cm.GetSession(sessionID)
			if !exists {
				t.Error("应该能获取到会话")
			}

			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
