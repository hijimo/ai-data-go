package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// CacheKeyManager 缓存键管理器
type CacheKeyManager struct {
	// 命名空间前缀
	namespace string
}

// NewCacheKeyManager 创建缓存键管理器
func NewCacheKeyManager(namespace string) *CacheKeyManager {
	return &CacheKeyManager{
		namespace: namespace,
	}
}

// 缓存键类型常量
const (
	KeyTypeContext      = "context"
	KeyTypeMemory       = "memory"
	KeyTypeSummary      = "summary"
	KeyTypeVector       = "vector"
	KeyTypeToken        = "token"
	KeyTypeSession      = "session"
	KeyTypeUser         = "user"
	KeyTypeTenant       = "tenant"
)

// BuildContextKey 构建上下文缓存键
// 格式: {namespace}:context:{tenantId}:{sessionId}
func (m *CacheKeyManager) BuildContextKey(tenantID, sessionID string) string {
	return m.buildKey(KeyTypeContext, tenantID, sessionID)
}

// BuildMemoryKey 构建记忆缓存键
// 格式: {namespace}:memory:{tenantId}:{memoryId}
func (m *CacheKeyManager) BuildMemoryKey(tenantID, memoryID string) string {
	return m.buildKey(KeyTypeMemory, tenantID, memoryID)
}

// BuildSummaryKey 构建摘要缓存键
// 格式: {namespace}:summary:{tenantId}:{sessionId}
func (m *CacheKeyManager) BuildSummaryKey(tenantID, sessionID string) string {
	return m.buildKey(KeyTypeSummary, tenantID, sessionID)
}

// BuildVectorSearchKey 构建向量搜索结果缓存键
// 格式: {namespace}:vector:{tenantId}:{queryHash}
func (m *CacheKeyManager) BuildVectorSearchKey(tenantID, query string, limit int) string {
	queryHash := m.hashQuery(query, limit)
	return m.buildKey(KeyTypeVector, tenantID, queryHash)
}

// BuildTokenUsageKey 构建Token使用量缓存键
// 格式: {namespace}:token:{tenantId}:{date}
func (m *CacheKeyManager) BuildTokenUsageKey(tenantID, date string) string {
	return m.buildKey(KeyTypeToken, tenantID, date)
}

// BuildSessionListKey 构建会话列表缓存键
// 格式: {namespace}:session:list:{tenantId}:{userId}
func (m *CacheKeyManager) BuildSessionListKey(tenantID, userID string) string {
	return m.buildKey(KeyTypeSession, "list", tenantID, userID)
}

// BuildUserKey 构建用户缓存键
// 格式: {namespace}:user:{tenantId}:{userId}
func (m *CacheKeyManager) BuildUserKey(tenantID, userID string) string {
	return m.buildKey(KeyTypeUser, tenantID, userID)
}

// BuildTenantKey 构建租户缓存键
// 格式: {namespace}:tenant:{tenantId}
func (m *CacheKeyManager) BuildTenantKey(tenantID string) string {
	return m.buildKey(KeyTypeTenant, tenantID)
}

// BuildPatternForTenant 构建租户相关的所有键模式
// 格式: {namespace}:*:{tenantId}:*
func (m *CacheKeyManager) BuildPatternForTenant(tenantID string) string {
	if m.namespace != "" {
		return fmt.Sprintf("%s:*:%s:*", m.namespace, tenantID)
	}
	return fmt.Sprintf("*:%s:*", tenantID)
}

// BuildPatternForType 构建特定类型的所有键模式
// 格式: {namespace}:{keyType}:*
func (m *CacheKeyManager) BuildPatternForType(keyType string) string {
	if m.namespace != "" {
		return fmt.Sprintf("%s:%s:*", m.namespace, keyType)
	}
	return fmt.Sprintf("%s:*", keyType)
}

// BuildPatternForTenantAndType 构建租户特定类型的所有键模式
// 格式: {namespace}:{keyType}:{tenantId}:*
func (m *CacheKeyManager) BuildPatternForTenantAndType(tenantID, keyType string) string {
	if m.namespace != "" {
		return fmt.Sprintf("%s:%s:%s:*", m.namespace, keyType, tenantID)
	}
	return fmt.Sprintf("%s:%s:*", keyType, tenantID)
}

// ParseKey 解析缓存键
func (m *CacheKeyManager) ParseKey(key string) (keyType, tenantID string, parts []string) {
	// 移除命名空间前缀
	if m.namespace != "" {
		key = strings.TrimPrefix(key, m.namespace+":")
	}

	parts = strings.Split(key, ":")
	if len(parts) >= 2 {
		keyType = parts[0]
		tenantID = parts[1]
	}

	return keyType, tenantID, parts
}

// 内部方法
func (m *CacheKeyManager) buildKey(parts ...string) string {
	if m.namespace != "" {
		parts = append([]string{m.namespace}, parts...)
	}
	return strings.Join(parts, ":")
}

func (m *CacheKeyManager) hashQuery(query string, limit int) string {
	data := fmt.Sprintf("%s:%d", query, limit)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// ValidateKey 验证缓存键格式
func (m *CacheKeyManager) ValidateKey(key string) bool {
	// 移除命名空间前缀
	if m.namespace != "" {
		key = strings.TrimPrefix(key, m.namespace+":")
	}

	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return false
	}

	// 验证键类型
	keyType := parts[0]
	validTypes := []string{
		KeyTypeContext,
		KeyTypeMemory,
		KeyTypeSummary,
		KeyTypeVector,
		KeyTypeToken,
		KeyTypeSession,
		KeyTypeUser,
		KeyTypeTenant,
	}

	for _, validType := range validTypes {
		if keyType == validType {
			return true
		}
	}

	return false
}
