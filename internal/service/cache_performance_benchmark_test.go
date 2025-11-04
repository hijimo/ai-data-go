package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// 测试数据结构
type BenchmarkData struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`
	Tokens    int                    `json:"tokens"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// BenchmarkCacheService_Set_Small 基准测试：缓存小数据写入
func BenchmarkCacheService_Set_Small(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   "小数据内容",
		Tokens:    10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("small:%d", i)
		_ = cache.Set(ctx, key, data, 5*time.Minute)
	}
}

// BenchmarkCacheService_Set_Medium 基准测试：缓存中等数据写入
func BenchmarkCacheService_Set_Medium(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 中等大小数据（约1KB）
	content := string(make([]byte, 1024))
	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   content,
		Tokens:    100,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("medium:%d", i)
		_ = cache.Set(ctx, key, data, 5*time.Minute)
	}
}

// BenchmarkCacheService_Set_Large 基准测试：缓存大数据写入
func BenchmarkCacheService_Set_Large(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 大数据（约10KB）
	content := string(make([]byte, 10240))
	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   content,
		Tokens:    1000,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("large:%d", i)
		_ = cache.Set(ctx, key, data, 5*time.Minute)
	}
}

// BenchmarkCacheService_Get_Small 基准测试：缓存小数据读取
func BenchmarkCacheService_Get_Small(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 预先设置缓存
	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   "小数据内容",
		Tokens:    10,
	}
	key := "small:test"
	_ = cache.Set(ctx, key, data, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result BenchmarkData
		_ = cache.Get(ctx, key, &result)
	}
}

// BenchmarkCacheService_Get_Medium 基准测试：缓存中等数据读取
func BenchmarkCacheService_Get_Medium(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	content := string(make([]byte, 1024))
	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   content,
		Tokens:    100,
	}
	key := "medium:test"
	_ = cache.Set(ctx, key, data, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result BenchmarkData
		_ = cache.Get(ctx, key, &result)
	}
}

// BenchmarkCacheService_Get_Large 基准测试：缓存大数据读取
func BenchmarkCacheService_Get_Large(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	content := string(make([]byte, 10240))
	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   content,
		Tokens:    1000,
	}
	key := "large:test"
	_ = cache.Set(ctx, key, data, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result BenchmarkData
		_ = cache.Get(ctx, key, &result)
	}
}

// BenchmarkCacheService_SetGet_Mixed 基准测试：混合读写
func BenchmarkCacheService_SetGet_Mixed(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   "混合测试数据",
		Tokens:    50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("mixed:%d", i%100)

		if i%3 == 0 {
			// 33% 写入
			_ = cache.Set(ctx, key, data, 5*time.Minute)
		} else {
			// 67% 读取
			var result BenchmarkData
			_ = cache.Get(ctx, key, &result)
		}
	}
}

// BenchmarkCacheService_Delete 基准测试：缓存删除
func BenchmarkCacheService_Delete(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// 每次迭代前设置缓存
		key := fmt.Sprintf("delete:%d", i)
		data := &BenchmarkData{
			ID:      uuid.New().String(),
			Content: "待删除数据",
		}
		_ = cache.Set(ctx, key, data, 5*time.Minute)
		b.StartTimer()

		_ = cache.Delete(ctx, key)
	}
}

// BenchmarkCacheService_Exists 基准测试：检查缓存存在
func BenchmarkCacheService_Exists(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 预先设置缓存
	key := "exists:test"
	data := &BenchmarkData{
		ID:      uuid.New().String(),
		Content: "存在性测试",
	}
	_ = cache.Set(ctx, key, data, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Exists(ctx, key)
	}
}

// BenchmarkCacheService_Parallel_Set 并发基准测试：缓存写入
func BenchmarkCacheService_Parallel_Set(b *testing.B) {
	cache := NewCacheService(nil, "bench")

	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   "并发写入测试",
		Tokens:    50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		i := 0

		for pb.Next() {
			key := fmt.Sprintf("parallel:set:%d", i)
			_ = cache.Set(ctx, key, data, 5*time.Minute)
			i++
		}
	})
}

// BenchmarkCacheService_Parallel_Get 并发基准测试：缓存读取
func BenchmarkCacheService_Parallel_Get(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 预先设置100个缓存键
	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   "并发读取测试",
		Tokens:    50,
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("parallel:get:%d", i)
		_ = cache.Set(ctx, key, data, 5*time.Minute)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		i := 0

		for pb.Next() {
			key := fmt.Sprintf("parallel:get:%d", i%100)
			var result BenchmarkData
			_ = cache.Get(ctx, key, &result)
			i++
		}
	})
}

// BenchmarkCacheService_Parallel_Mixed 并发基准测试：混合读写
func BenchmarkCacheService_Parallel_Mixed(b *testing.B) {
	cache := NewCacheService(nil, "bench")

	data := &BenchmarkData{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Content:   "并发混合测试",
		Tokens:    50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		i := 0

		for pb.Next() {
			key := fmt.Sprintf("parallel:mixed:%d", i%100)

			if i%3 == 0 {
				_ = cache.Set(ctx, key, data, 5*time.Minute)
			} else {
				var result BenchmarkData
				_ = cache.Get(ctx, key, &result)
			}

			i++
		}
	})
}

// BenchmarkCacheService_TTL_Short 基准测试：短TTL缓存
func BenchmarkCacheService_TTL_Short(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	data := &BenchmarkData{
		ID:      uuid.New().String(),
		Content: "短TTL测试",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("ttl:short:%d", i)
		_ = cache.Set(ctx, key, data, 1*time.Second)
	}
}

// BenchmarkCacheService_TTL_Long 基准测试：长TTL缓存
func BenchmarkCacheService_TTL_Long(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	data := &BenchmarkData{
		ID:      uuid.New().String(),
		Content: "长TTL测试",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("ttl:long:%d", i)
		_ = cache.Set(ctx, key, data, 1*time.Hour)
	}
}

// BenchmarkCacheService_HitRate 基准测试：缓存命中率
func BenchmarkCacheService_HitRate(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 预先设置50个缓存键（50%命中率）
	data := &BenchmarkData{
		ID:      uuid.New().String(),
		Content: "命中率测试",
	}

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("hitrate:%d", i)
		_ = cache.Set(ctx, key, data, 5*time.Minute)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("hitrate:%d", i%100) // 0-99，50%命中
		var result BenchmarkData
		_ = cache.Get(ctx, key, &result)
	}
}
