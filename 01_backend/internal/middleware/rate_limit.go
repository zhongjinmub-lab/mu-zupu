package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

type rateWindow struct {
	Count     int
	ResetTime time.Time
}

type MemoryRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	buckets map[string]rateWindow
}

type RateCounter interface {
	Allow(ctx context.Context, key string, limit int) (bool, error)
}

type RateLimiter struct {
	counter           RateCounter
	fallback          *MemoryRateLimiter
	retryAfterSeconds string
}

func NewMemoryRateLimiter(window time.Duration) *MemoryRateLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &MemoryRateLimiter{
		window:  window,
		buckets: make(map[string]rateWindow),
	}
}

func NewRateLimiter(counter RateCounter, fallback *MemoryRateLimiter) *RateLimiter {
	if fallback == nil {
		fallback = NewMemoryRateLimiter(time.Minute)
	}
	if counter == nil {
		counter = fallback
	}
	return &RateLimiter{
		counter:           counter,
		fallback:          fallback,
		retryAfterSeconds: retryAfterSeconds(fallback.window),
	}
}

type RedisRateLimiter struct {
	client *redis.Client
	window time.Duration
	prefix string
}

func NewRedisRateLimiter(client *redis.Client, window time.Duration, prefix string) *RedisRateLimiter {
	if window <= 0 {
		window = time.Minute
	}
	if prefix == "" {
		prefix = "rate_limit:"
	}
	return &RedisRateLimiter{
		client: client,
		window: window,
		prefix: prefix,
	}
}

func (l *MemoryRateLimiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	if key == "" || limit <= 0 {
		return true, nil
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	item := l.buckets[key]
	if item.ResetTime.IsZero() || now.After(item.ResetTime) {
		l.buckets[key] = rateWindow{Count: 1, ResetTime: now.Add(l.window)}
		l.cleanupLocked(now)
		return true, nil
	}
	if item.Count >= limit {
		return false, nil
	}
	item.Count++
	l.buckets[key] = item
	return true, nil
}

func (l *MemoryRateLimiter) cleanupLocked(now time.Time) {
	for key, item := range l.buckets {
		if now.After(item.ResetTime.Add(l.window)) {
			delete(l.buckets, key)
		}
	}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	if key == "" || limit <= 0 {
		return true, nil
	}
	if l == nil || l.client == nil {
		return true, nil
	}
	count, err := l.client.Incr(ctx, l.prefix+key).Result()
	if err != nil {
		return true, err
	}
	if count == 1 {
		if err := l.client.Expire(ctx, l.prefix+key, l.window).Err(); err != nil {
			return true, err
		}
	}
	return count <= int64(limit), nil
}

func (l *RateLimiter) allow(ctx context.Context, key string, limit int) bool {
	ok, err := l.counter.Allow(ctx, key, limit)
	if err == nil {
		return ok
	}
	ok, fallbackErr := l.fallback.Allow(ctx, key, limit)
	return fallbackErr == nil && ok
}

func (l *RateLimiter) IP(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(c.Request.Context(), "ip:"+c.ClientIP(), limit) {
			l.writeRateLimitError(c)
			return
		}
		c.Next()
	}
}

func (l *RateLimiter) TenantAndUser(tenantLimit, userLimit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if t, ok := tenant.CurrentTenant(c); ok {
			if !l.allow(c.Request.Context(), "tenant:"+t.ID, tenantLimit) {
				l.writeRateLimitError(c)
				return
			}
		}
		if u, ok := auth.CurrentUser(c); ok {
			if !l.allow(c.Request.Context(), "user:"+u.ID, userLimit) {
				l.writeRateLimitError(c)
				return
			}
		}
		c.Next()
	}
}

func (l *RateLimiter) writeRateLimitError(c *gin.Context) {
	c.Header("Retry-After", l.retryAfterSeconds)
	response.Error(c, http.StatusTooManyRequests, 42901, "请求过于频繁，请稍后再试")
	c.Abort()
}

func retryAfterSeconds(window time.Duration) string {
	seconds := int(window.Seconds())
	if seconds < 1 {
		seconds = int(time.Minute.Seconds())
	}
	return strconv.Itoa(seconds)
}
