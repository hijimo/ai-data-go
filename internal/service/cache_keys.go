package service

import (
	"fmt"
	"time"
)

// CacheKeys 缓存键管理
type CacheKeys struct{}

// NewCacheKeys 创建缓存键管理实例
func NewCacheKeys() *CacheKeys {
	return &CacheKeys{}
}

// ContextKey 构建上下文缓存键
// 格式: context:{sessionId}:{queryHash}
// TTL: 5分钟
func (k *CacheKeys) ContextKey(sessionID, queryHash string) string {
	return fmt.Sprintf("context:%s:%s", sessionID, queryHash)
}

// ContextTTL 返回上下文缓存的 TTL
func (k *CacheKeys) ContextTTL() time.Duration {
	return 5 * time.Minute
}

// VectorSearchKey 构建向量检索结果缓存键
// 格式: vector:{sessionId}:{queryHash}
// TTL: 30分钟
func (k *CacheKeys) VectorSearchKey(sessionID, queryHash string) string {
	return fmt.Sprintf("vector:%s:%s", sessionID, queryHash)
}

// VectorSearchTTL 返回向量检索缓存的 TTL
func (k *CacheKeys) VectorSearchTTL() time.Duration {
	return 30 * time.Minute
}

// SummaryKey 构建会话摘要缓存键
// 格式: summary:{sessionId}:latest
// TTL: 1小时
func (k *CacheKeys) SummaryKey(sessionID string) string {
	return fmt.Sprintf("summary:%s:latest", sessionID)
}

// SummaryTTL 返回摘要缓存的 TTL
func (k *CacheKeys) SummaryTTL() time.Duration {
	return 1 * time.Hour
}

// SessionListKey 构建用户会话列表缓存键
// 格式: sessions:{userId}:list
// TTL: 10分钟
func (k *CacheKeys) SessionListKey(userID string) string {
	return fmt.Sprintf("sessions:%s:list", userID)
}

// SessionListTTL 返回会话列表缓存的 TTL
func (k *CacheKeys) SessionListTTL() time.Duration {
	return 10 * time.Minute
}

// TokenUsageKey 构建 Token 使用统计缓存键
// 格式: tokens:{sessionId}:usage
// TTL: 5分钟
func (k *CacheKeys) TokenUsageKey(sessionID string) string {
	return fmt.Sprintf("tokens:%s:usage", sessionID)
}

// TokenUsageTTL 返回 Token 使用统计缓存的 TTL
func (k *CacheKeys) TokenUsageTTL() time.Duration {
	return 5 * time.Minute
}

// QuotaKey 构建租户配额缓存键
// 格式: quota:{tenantId}:{type}
// type: daily, monthly, session
// TTL: 5分钟
func (k *CacheKeys) QuotaKey(tenantID, quotaType string) string {
	return fmt.Sprintf("quota:%s:%s", tenantID, quotaType)
}

// QuotaTTL 返回配额缓存的 TTL
func (k *CacheKeys) QuotaTTL() time.Duration {
	return 5 * time.Minute
}

// ContextConfigKey 构建上下文配置缓存键
// 格式: context:config:{sessionId}
// TTL: 10分钟
func (k *CacheKeys) ContextConfigKey(sessionID string) string {
	return fmt.Sprintf("context:config:%s", sessionID)
}

// ContextConfigTTL 返回上下文配置缓存的 TTL
func (k *CacheKeys) ContextConfigTTL() time.Duration {
	return 10 * time.Minute
}

// MessageKey 构建消息缓存键
// 格式: message:{messageId}
// TTL: 30分钟
func (k *CacheKeys) MessageKey(messageID string) string {
	return fmt.Sprintf("message:%s", messageID)
}

// MessageTTL 返回消息缓存的 TTL
func (k *CacheKeys) MessageTTL() time.Duration {
	return 30 * time.Minute
}

// MemoryKey 构建记忆缓存键
// 格式: memory:{memoryId}
// TTL: 1小时
func (k *CacheKeys) MemoryKey(memoryID string) string {
	return fmt.Sprintf("memory:%s", memoryID)
}

// MemoryTTL 返回记忆缓存的 TTL
func (k *CacheKeys) MemoryTTL() time.Duration {
	return 1 * time.Hour
}

// SessionPattern 构建会话相关缓存的模式
// 用于批量删除会话相关的所有缓存
func (k *CacheKeys) SessionPattern(sessionID string) string {
	return fmt.Sprintf("*:%s:*", sessionID)
}

// UserPattern 构建用户相关缓存的模式
// 用于批量删除用户相关的所有缓存
func (k *CacheKeys) UserPattern(userID string) string {
	return fmt.Sprintf("*:%s:*", userID)
}

// TenantPattern 构建租户相关缓存的模式
// 用于批量删除租户相关的所有缓存
func (k *CacheKeys) TenantPattern(tenantID string) string {
	return fmt.Sprintf("*:%s:*", tenantID)
}

// AIResponseKey 构建 AI 响应缓存键（用于降级）
// 格式: ai:response:{sessionId}:{queryHash}
// TTL: 1小时
func (k *CacheKeys) AIResponseKey(sessionID, queryHash string) string {
	return fmt.Sprintf("ai:response:%s:%s", sessionID, queryHash)
}

// AIResponseTTL 返回 AI 响应缓存的 TTL
func (k *CacheKeys) AIResponseTTL() time.Duration {
	return 1 * time.Hour
}

// RateLimitKey 构建速率限制缓存键
// 格式: ratelimit:{tenantId}:{endpoint}
// TTL: 1分钟
func (k *CacheKeys) RateLimitKey(tenantID, endpoint string) string {
	return fmt.Sprintf("ratelimit:%s:%s", tenantID, endpoint)
}

// RateLimitTTL 返回速率限制缓存的 TTL
func (k *CacheKeys) RateLimitTTL() time.Duration {
	return 1 * time.Minute
}

// CircuitBreakerKey 构建熔断器状态缓存键
// 格式: circuit:{service}
// TTL: 5分钟
func (k *CacheKeys) CircuitBreakerKey(service string) string {
	return fmt.Sprintf("circuit:%s", service)
}

// CircuitBreakerTTL 返回熔断器状态缓存的 TTL
func (k *CacheKeys) CircuitBreakerTTL() time.Duration {
	return 5 * time.Minute
}
