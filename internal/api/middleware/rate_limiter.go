package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"

	"github.com/gin-gonic/gin"
)

// RateLimiter 速率限制器接口
type RateLimiter interface {
	// Allow 检查是否允许请求
	Allow(key string) bool
	// Reset 重置指定键的限制
	Reset(key string)
}

// TokenBucket 令牌桶算法实现
type TokenBucket struct {
	capacity   int           // 桶容量
	tokens     int           // 当前令牌数
	refillRate int           // 每秒补充的令牌数
	lastRefill time.Time     // 上次补充时间
	mu         sync.Mutex    // 互斥锁
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 补充令牌
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(tb.refillRate))
	
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	// 检查是否有可用令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// Reset 重置令牌桶
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
}

// InMemoryRateLimiter 基于内存的速率限制器
type InMemoryRateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	capacity   int
	refillRate int
}

// NewInMemoryRateLimiter 创建内存速率限制器
func NewInMemoryRateLimiter(capacity, refillRate int) *InMemoryRateLimiter {
	limiter := &InMemoryRateLimiter{
		buckets:    make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
	
	// 启动清理协程
	go limiter.cleanup()
	
	return limiter
}

// Allow 检查是否允许请求
func (rl *InMemoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = NewTokenBucket(rl.capacity, rl.refillRate)
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	return bucket.Allow()
}

// Reset 重置指定键的限制
func (rl *InMemoryRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if bucket, exists := rl.buckets[key]; exists {
		bucket.Reset()
	}
}

// cleanup 定期清理过期的令牌桶
func (rl *InMemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			bucket.mu.Lock()
			// 如果令牌桶超过10分钟未使用，删除它
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.buckets, key)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimiterConfig 速率限制器配置
type RateLimiterConfig struct {
	// 基于IP的限制
	IPCapacity   int // IP令牌桶容量
	IPRefillRate int // IP每秒补充令牌数
	
	// 基于租户的限制
	TenantCapacity   int // 租户令牌桶容量
	TenantRefillRate int // 租户每秒补充令牌数
	
	// 是否启用
	EnableIPLimit     bool
	EnableTenantLimit bool
}

// DefaultRateLimiterConfig 默认速率限制器配置
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		// IP限制：每秒10个请求，桶容量20
		IPCapacity:   20,
		IPRefillRate: 10,
		
		// 租户限制：每秒50个请求，桶容量100
		TenantCapacity:   100,
		TenantRefillRate: 50,
		
		EnableIPLimit:     true,
		EnableTenantLimit: true,
	}
}

// RateLimiterMiddleware 速率限制中间件
type RateLimiterMiddleware struct {
	ipLimiter     RateLimiter
	tenantLimiter RateLimiter
	config        *RateLimiterConfig
	logger        logger.Logger
}

// NewRateLimiterMiddleware 创建速率限制中间件
func NewRateLimiterMiddleware(config *RateLimiterConfig, log logger.Logger) *RateLimiterMiddleware {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}
	
	return &RateLimiterMiddleware{
		ipLimiter:     NewInMemoryRateLimiter(config.IPCapacity, config.IPRefillRate),
		tenantLimiter: NewInMemoryRateLimiter(config.TenantCapacity, config.TenantRefillRate),
		config:        config,
		logger:        log,
	}
}

// RateLimit 返回速率限制中间件处理函数
func (m *RateLimiterMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// 1. 基于IP的速率限制
		if m.config.EnableIPLimit {
			clientIP := c.ClientIP()
			if !m.ipLimiter.Allow(clientIP) {
				m.logger.WarnContext(ctx, "IP速率限制触发", logger.Fields{
					"ip":     clientIP,
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
				})
				
				m.writeRateLimitError(c, "请求过于频繁，请稍后再试")
				return
			}
		}
		
		// 2. 基于租户的速率限制
		if m.config.EnableTenantLimit {
			tenantID, exists := GetTenantID(ctx)
			if exists && tenantID != "" {
				tenantKey := fmt.Sprintf("tenant:%s", tenantID)
				if !m.tenantLimiter.Allow(tenantKey) {
					m.logger.WarnContext(ctx, "租户速率限制触发", logger.Fields{
						"tenantId": tenantID,
						"path":     c.Request.URL.Path,
						"method":   c.Request.Method,
					})
					
					m.writeRateLimitError(c, "租户请求过于频繁，请稍后再试")
					return
				}
			}
		}
		
		c.Next()
	}
}

// RateLimitByIP 仅基于IP的速率限制
func (m *RateLimiterMiddleware) RateLimitByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		clientIP := c.ClientIP()
		
		if !m.ipLimiter.Allow(clientIP) {
			m.logger.WarnContext(ctx, "IP速率限制触发", logger.Fields{
				"ip":     clientIP,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			
			m.writeRateLimitError(c, "请求过于频繁，请稍后再试")
			return
		}
		
		c.Next()
	}
}

// RateLimitByTenant 仅基于租户的速率限制
func (m *RateLimiterMiddleware) RateLimitByTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		tenantID, exists := GetTenantID(ctx)
		
		if !exists || tenantID == "" {
			// 如果没有租户ID，跳过租户限制
			c.Next()
			return
		}
		
		tenantKey := fmt.Sprintf("tenant:%s", tenantID)
		if !m.tenantLimiter.Allow(tenantKey) {
			m.logger.WarnContext(ctx, "租户速率限制触发", logger.Fields{
				"tenantId": tenantID,
				"path":     c.Request.URL.Path,
				"method":   c.Request.Method,
			})
			
			m.writeRateLimitError(c, "租户请求过于频繁，请稍后再试")
			return
		}
		
		c.Next()
	}
}

// writeRateLimitError 写入速率限制错误响应
func (m *RateLimiterMiddleware) writeRateLimitError(c *gin.Context, message string) {
	ctx := c.Request.Context()
	
	// 构建错误响应
	resp := response.ErrorWithContext[any](
		ctx,
		errors.CodeTooManyRequests,
		message,
	)
	
	// 添加 Retry-After 头
	c.Header("Retry-After", "60")
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", m.config.IPCapacity))
	c.Header("X-RateLimit-Remaining", "0")
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(60*time.Second).Unix()))
	
	c.JSON(http.StatusTooManyRequests, resp)
	c.Abort()
}

// ResetIPLimit 重置IP限制（用于测试或管理）
func (m *RateLimiterMiddleware) ResetIPLimit(ip string) {
	m.ipLimiter.Reset(ip)
}

// ResetTenantLimit 重置租户限制（用于测试或管理）
func (m *RateLimiterMiddleware) ResetTenantLimit(tenantID string) {
	tenantKey := fmt.Sprintf("tenant:%s", tenantID)
	m.tenantLimiter.Reset(tenantKey)
}
