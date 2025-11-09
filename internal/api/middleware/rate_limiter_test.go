package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTokenBucket(t *testing.T) {
	// 创建令牌桶：容量5，每秒补充2个令牌
	tb := NewTokenBucket(5, 2)

	// 测试初始状态
	assert.True(t, tb.Allow(), "初始应该有令牌")
	assert.True(t, tb.Allow(), "应该还有令牌")

	// 消耗所有令牌
	for i := 0; i < 3; i++ {
		tb.Allow()
	}

	// 应该没有令牌了
	assert.False(t, tb.Allow(), "令牌应该用完")

	// 等待补充
	time.Sleep(1 * time.Second)

	// 应该补充了2个令牌
	assert.True(t, tb.Allow(), "应该补充了令牌")
	assert.True(t, tb.Allow(), "应该还有补充的令牌")
	assert.False(t, tb.Allow(), "补充的令牌应该用完")
}

func TestTokenBucketReset(t *testing.T) {
	tb := NewTokenBucket(5, 2)

	// 消耗所有令牌
	for i := 0; i < 5; i++ {
		tb.Allow()
	}

	assert.False(t, tb.Allow(), "令牌应该用完")

	// 重置
	tb.Reset()

	// 应该恢复到满容量
	assert.True(t, tb.Allow(), "重置后应该有令牌")
}

func TestInMemoryRateLimiter(t *testing.T) {
	limiter := NewInMemoryRateLimiter(3, 1)

	// 测试不同的键
	assert.True(t, limiter.Allow("key1"), "key1 应该允许")
	assert.True(t, limiter.Allow("key2"), "key2 应该允许")

	// 消耗 key1 的所有令牌
	limiter.Allow("key1")
	limiter.Allow("key1")

	assert.False(t, limiter.Allow("key1"), "key1 应该被限制")
	assert.True(t, limiter.Allow("key2"), "key2 应该不受影响")
}

func TestRateLimiterMiddleware_IPLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.NewTestLogger()

	config := &RateLimiterConfig{
		IPCapacity:        3,
		IPRefillRate:      1,
		EnableIPLimit:     true,
		EnableTenantLimit: false,
	}

	middleware := NewRateLimiterMiddleware(config, log)

	// 创建测试路由
	router := gin.New()
	router.Use(middleware.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 前3个请求应该成功
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "请求 %d 应该成功", i+1)
	}

	// 第4个请求应该被限制
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第4个请求应该被限制")
	assert.Contains(t, w.Header().Get("Retry-After"), "60", "应该包含 Retry-After 头")
}

func TestRateLimiterMiddleware_TenantLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.NewTestLogger()

	config := &RateLimiterConfig{
		TenantCapacity:    2,
		TenantRefillRate:  1,
		EnableIPLimit:     false,
		EnableTenantLimit: true,
	}

	middleware := NewRateLimiterMiddleware(config, log)

	// 创建测试路由
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// 模拟设置租户ID到上下文
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, TenantIDKey, "tenant-123")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(middleware.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 前2个请求应该成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "请求 %d 应该成功", i+1)
	}

	// 第3个请求应该被限制
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第3个请求应该被限制")
}

func TestRateLimiterMiddleware_Reset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.NewTestLogger()

	config := &RateLimiterConfig{
		IPCapacity:        2,
		IPRefillRate:      1,
		EnableIPLimit:     true,
		EnableTenantLimit: false,
	}

	middleware := NewRateLimiterMiddleware(config, log)

	// 创建测试路由
	router := gin.New()
	router.Use(middleware.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 消耗所有令牌
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 应该被限制
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// 重置限制
	middleware.ResetIPLimit("192.168.1.1")

	// 应该可以再次请求
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "重置后应该可以请求")
}

func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()

	assert.Equal(t, 20, config.IPCapacity, "默认IP容量应该是20")
	assert.Equal(t, 10, config.IPRefillRate, "默认IP补充率应该是10")
	assert.Equal(t, 100, config.TenantCapacity, "默认租户容量应该是100")
	assert.Equal(t, 50, config.TenantRefillRate, "默认租户补充率应该是50")
	assert.True(t, config.EnableIPLimit, "默认应该启用IP限制")
	assert.True(t, config.EnableTenantLimit, "默认应该启用租户限制")
}
