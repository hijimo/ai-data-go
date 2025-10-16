package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ContextManager 上下文管理器接口
type ContextManager interface {
	// CreateSession 创建新会话
	CreateSession(ctx context.Context) (string, context.Context, context.CancelFunc)

	// GetSession 获取会话上下文
	GetSession(sessionID string) (context.Context, bool)

	// CancelSession 取消会话
	CancelSession(sessionID string) error

	// CleanupSession 清理会话
	CleanupSession(sessionID string)

	// Start 启动自动清理
	Start()

	// Stop 停止自动清理
	Stop()
}

// sessionInfo 会话信息
type sessionInfo struct {
	ctx        context.Context
	cancel     context.CancelFunc
	createdAt  time.Time
	lastAccess time.Time
}

// contextManager 上下文管理器实现
type contextManager struct {
	sessions        map[string]*sessionInfo
	mu              sync.RWMutex
	timeout         time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// NewContextManager 创建新的上下文管理器
func NewContextManager(timeout, cleanupInterval time.Duration) ContextManager {
	return &contextManager{
		sessions:        make(map[string]*sessionInfo),
		timeout:         timeout,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

// CreateSession 创建新会话
func (cm *contextManager) CreateSession(ctx context.Context) (string, context.Context, context.CancelFunc) {
	// 生成唯一的会话ID
	sessionID := uuid.New().String()

	// 创建可取消的上下文
	sessionCtx, cancel := context.WithCancel(ctx)

	// 创建会话信息
	now := time.Now()
	info := &sessionInfo{
		ctx:        sessionCtx,
		cancel:     cancel,
		createdAt:  now,
		lastAccess: now,
	}

	// 存储会话信息
	cm.mu.Lock()
	cm.sessions[sessionID] = info
	cm.mu.Unlock()

	// 返回包装的取消函数，确保在取消时清理会话
	wrappedCancel := func() {
		cancel()
		cm.CleanupSession(sessionID)
	}

	return sessionID, sessionCtx, wrappedCancel
}

// GetSession 获取会话上下文
func (cm *contextManager) GetSession(sessionID string) (context.Context, bool) {
	cm.mu.RLock()
	info, exists := cm.sessions[sessionID]
	cm.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// 更新最后访问时间
	cm.mu.Lock()
	info.lastAccess = time.Now()
	cm.mu.Unlock()

	return info.ctx, true
}

// CancelSession 取消会话
func (cm *contextManager) CancelSession(sessionID string) error {
	cm.mu.RLock()
	info, exists := cm.sessions[sessionID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 取消上下文
	info.cancel()

	// 清理会话
	cm.CleanupSession(sessionID)

	return nil
}

// CleanupSession 清理会话
func (cm *contextManager) CleanupSession(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if info, exists := cm.sessions[sessionID]; exists {
		// 确保取消函数被调用
		info.cancel()
		// 从映射中删除
		delete(cm.sessions, sessionID)
	}
}

// Start 启动自动清理
func (cm *contextManager) Start() {
	cm.wg.Add(1)
	go cm.cleanupLoop()
}

// Stop 停止自动清理
func (cm *contextManager) Stop() {
	close(cm.stopChan)
	cm.wg.Wait()

	// 清理所有剩余会话
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for sessionID, info := range cm.sessions {
		info.cancel()
		delete(cm.sessions, sessionID)
	}
}

// cleanupLoop 自动清理循环
func (cm *contextManager) cleanupLoop() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.cleanupExpiredSessions()
		case <-cm.stopChan:
			return
		}
	}
}

// cleanupExpiredSessions 清理过期会话
func (cm *contextManager) cleanupExpiredSessions() {
	now := time.Now()
	expiredSessions := make([]string, 0)

	// 查找过期会话
	cm.mu.RLock()
	for sessionID, info := range cm.sessions {
		// 检查会话是否超时
		if now.Sub(info.lastAccess) > cm.timeout {
			expiredSessions = append(expiredSessions, sessionID)
		}
		// 检查上下文是否已取消
		if info.ctx.Err() != nil {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}
	cm.mu.RUnlock()

	// 清理过期会话
	for _, sessionID := range expiredSessions {
		cm.CleanupSession(sessionID)
	}
}
