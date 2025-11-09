package storage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"
)

// Mock CacheService for testing
type mockCacheService struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

func newMockCacheService() *mockCacheService {
	return &mockCacheService{
		data: make(map[string]interface{}),
	}
}

func (m *mockCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if value, exists := m.data[key]; exists {
		// 简单的值复制
		switch v := value.(type) {
		case string:
			if ptr, ok := dest.(*string); ok {
				*ptr = v
			}
		case int:
			if ptr, ok := dest.(*int); ok {
				*ptr = v
			}
		}
		return nil
	}
	return &CacheKeyNotFoundError{Key: key}
}

func (m *mockCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data[key] = value
	return nil
}

func (m *mockCacheService) Delete(ctx context.Context, keys ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *mockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string]interface{})
	return nil
}

func (m *mockCacheService) GetString(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockCacheService) SetString(ctx context.Context, key string, value string, expiration time.Duration) error {
	return nil
}

func (m *mockCacheService) Exists(ctx context.Context, keys ...string) (int64, error) {
	return 0, nil
}

func (m *mockCacheService) Increment(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (m *mockCacheService) GetWithNamespace(ctx context.Context, namespace, key string, dest interface{}) error {
	return nil
}

func (m *mockCacheService) SetWithNamespace(ctx context.Context, namespace, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (m *mockCacheService) DeleteWithNamespace(ctx context.Context, namespace string, keys ...string) error {
	return nil
}

func (m *mockCacheService) DeleteNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (m *mockCacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *mockCacheService) IsEnabled() bool {
	return true
}

// 测试多级缓存
func TestMultiLevelCache_GetSet(t *testing.T) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.DebugLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		LocalMaxEntries:   100,
		LocalDefaultTTL:   5 * time.Minute,
		EnableLocalCache:  true,
		CacheVersion:      "v1",
		NullValueTTL:      1 * time.Minute,
		ExpirationJitter:  5 * time.Second,
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)

	ctx := context.Background()
	key := "test:key"
	value := "test value"

	// 测试设置缓存
	err := mlCache.Set(ctx, key, value, 10*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 测试获取缓存（应该从L1获取）
	var result string
	err = mlCache.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// 验证统计信息
	stats := mlCache.GetStats()
	if stats["l1_hits"].(int64) != 1 {
		t.Errorf("Expected 1 L1 hit, got %d", stats["l1_hits"])
	}
}

// 测试缓存穿透防护
func TestMultiLevelCache_NullValue(t *testing.T) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.DebugLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: true,
		NullValueTTL:     1 * time.Minute,
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)

	ctx := context.Background()
	key := "nonexistent:key"

	// 设置空值
	err := mlCache.SetNullValue(ctx, key)
	if err != nil {
		t.Fatalf("SetNullValue failed: %v", err)
	}

	// 尝试获取（应该返回错误）
	var result string
	err = mlCache.Get(ctx, key, &result)
	if err == nil {
		t.Error("Expected error for null value, got nil")
	}
}

// 测试缓存优化器
func TestCacheOptimizer_GetWithProtection(t *testing.T) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.DebugLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: true,
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)
	optimizer := NewCacheOptimizer(mlCache, log)

	ctx := context.Background()
	key := "user:123"

	loadCount := 0
	loader := func() (interface{}, error) {
		loadCount++
		return "user data", nil
	}

	// 第一次获取（缓存未命中，调用loader）
	var result string
	err := optimizer.GetWithProtection(ctx, key, &result, loader)
	if err != nil {
		t.Fatalf("GetWithProtection failed: %v", err)
	}

	if loadCount != 1 {
		t.Errorf("Expected loader to be called once, got %d", loadCount)
	}

	// 第二次获取（缓存命中，不调用loader）
	err = optimizer.GetWithProtection(ctx, key, &result, loader)
	if err != nil {
		t.Fatalf("GetWithProtection failed: %v", err)
	}

	if loadCount != 1 {
		t.Errorf("Expected loader to be called once, got %d", loadCount)
	}
}

// 测试单飞模式（防止缓存击穿）
func TestCacheOptimizer_SingleFlight(t *testing.T) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.DebugLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: false, // 禁用本地缓存以测试单飞
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)
	optimizer := NewCacheOptimizer(mlCache, log)

	ctx := context.Background()
	key := "hot:key"

	loadCount := 0
	loader := func() (interface{}, error) {
		loadCount++
		time.Sleep(100 * time.Millisecond) // 模拟慢查询
		return "hot data", nil
	}

	// 并发请求同一个键
	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result string
			optimizer.GetWithProtection(ctx, key, &result, loader)
		}()
	}

	wg.Wait()

	// 验证loader只被调用一次（单飞模式）
	if loadCount != 1 {
		t.Errorf("Expected loader to be called once, got %d", loadCount)
	}
}

// 测试热点数据识别
func TestCacheOptimizer_HotKeyDetection(t *testing.T) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.DebugLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: true,
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)
	optimizer := NewCacheOptimizer(mlCache, log)

	key := "hot:key"

	// 模拟多次访问
	for i := 0; i < 15; i++ {
		optimizer.recordAccess(key)
	}

	// 验证是否被识别为热点
	if !optimizer.isHotKey(key) {
		t.Error("Expected key to be identified as hot")
	}

	hotKeys := optimizer.GetHotKeys()
	if len(hotKeys) != 1 || hotKeys[0] != key {
		t.Errorf("Expected hot keys to contain %s, got %v", key, hotKeys)
	}
}

// 测试缓存键管理器
func TestCacheKeyManager(t *testing.T) {
	keyMgr := NewCacheKeyManager("genkit")

	// 测试构建各类键
	contextKey := keyMgr.BuildContextKey("tenant1", "session1")
	expected := "genkit:context:tenant1:session1"
	if contextKey != expected {
		t.Errorf("Expected %s, got %s", expected, contextKey)
	}

	memoryKey := keyMgr.BuildMemoryKey("tenant1", "memory1")
	expected = "genkit:memory:tenant1:memory1"
	if memoryKey != expected {
		t.Errorf("Expected %s, got %s", expected, memoryKey)
	}

	// 测试解析键
	keyType, tenantID, parts := keyMgr.ParseKey(contextKey)
	if keyType != "context" {
		t.Errorf("Expected keyType 'context', got %s", keyType)
	}
	if tenantID != "tenant1" {
		t.Errorf("Expected tenantID 'tenant1', got %s", tenantID)
	}
	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}

	// 测试模式构建
	pattern := keyMgr.BuildPatternForTenant("tenant1")
	expected = "genkit:*:tenant1:*"
	if pattern != expected {
		t.Errorf("Expected %s, got %s", expected, pattern)
	}

	// 测试键验证
	if !keyMgr.ValidateKey(contextKey) {
		t.Error("Expected key to be valid")
	}

	if keyMgr.ValidateKey("invalid:key") {
		t.Error("Expected key to be invalid")
	}
}

// 测试缓存版本控制
func TestMultiLevelCache_VersionControl(t *testing.T) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.DebugLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: true,
		CacheVersion:     "v1",
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)

	ctx := context.Background()
	key := "test:key"
	value := "test value"

	// 设置缓存
	mlCache.Set(ctx, key, value, 10*time.Minute)

	// 使版本失效
	mlCache.InvalidateVersion(ctx)

	// 尝试获取（应该失败，因为版本已变）
	var result string
	err := mlCache.Get(ctx, key, &result)
	if err == nil {
		t.Error("Expected error after version invalidation")
	}
}

// 基准测试：多级缓存性能
func BenchmarkMultiLevelCache_Get(b *testing.B) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.ErrorLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: true,
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)

	ctx := context.Background()
	key := "bench:key"
	value := "bench value"

	mlCache.Set(ctx, key, value, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		mlCache.Get(ctx, key, &result)
	}
}

// 基准测试：缓存优化器性能
func BenchmarkCacheOptimizer_GetWithProtection(b *testing.B) {
	mockRedis := newMockCacheService()
	log := logger.New(logger.ErrorLevel, logger.TextFormat, os.Stdout)

	config := MultiLevelCacheConfig{
		EnableLocalCache: true,
	}

	mlCache := NewMultiLevelCache(mockRedis, config, log)
	optimizer := NewCacheOptimizer(mlCache, log)

	ctx := context.Background()
	key := "bench:key"

	loader := func() (interface{}, error) {
		return "bench value", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		optimizer.GetWithProtection(ctx, fmt.Sprintf("%s:%d", key, i%100), &result, loader)
	}
}
