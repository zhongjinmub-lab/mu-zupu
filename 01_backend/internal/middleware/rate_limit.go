package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

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

func NewMemoryRateLimiter(window time.Duration) *MemoryRateLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &MemoryRateLimiter{
		window:  window,
		buckets: make(map[string]rateWindow),
	}
}

func (l *MemoryRateLimiter) Allow(key string, limit int) bool {
	if key == "" || limit <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	item := l.buckets[key]
	if item.ResetTime.IsZero() || now.After(item.ResetTime) {
		l.buckets[key] = rateWindow{Count: 1, ResetTime: now.Add(l.window)}
		l.cleanupLocked(now)
		return true
	}
	if item.Count >= limit {
		return false
	}
	item.Count++
	l.buckets[key] = item
	return true
}

func (l *MemoryRateLimiter) cleanupLocked(now time.Time) {
	for key, item := range l.buckets {
		if now.After(item.ResetTime.Add(l.window)) {
			delete(l.buckets, key)
		}
	}
}

func (l *MemoryRateLimiter) IP(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("ip:"+c.ClientIP(), limit) {
			writeRateLimitError(c)
			return
		}
		c.Next()
	}
}

func (l *MemoryRateLimiter) TenantAndUser(tenantLimit, userLimit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if t, ok := tenant.CurrentTenant(c); ok {
			if !l.Allow("tenant:"+t.ID, tenantLimit) {
				writeRateLimitError(c)
				return
			}
		}
		if u, ok := auth.CurrentUser(c); ok {
			if !l.Allow("user:"+u.ID, userLimit) {
				writeRateLimitError(c)
				return
			}
		}
		c.Next()
	}
}

func writeRateLimitError(c *gin.Context) {
	c.Header("Retry-After", "60")
	response.Error(c, http.StatusTooManyRequests, 42901, "请求过于频繁，请稍后再试")
	c.Abort()
}
