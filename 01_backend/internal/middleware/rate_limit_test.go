package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/tenant"
)

func TestMemoryRateLimiterLimitsWithinWindow(t *testing.T) {
	limiter := NewMemoryRateLimiter(time.Minute)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		ok, err := limiter.Allow(ctx, "tenant:t1", 2)
		if err != nil || !ok {
			t.Fatalf("request %d allowed=%v err=%v", i+1, ok, err)
		}
	}
	ok, err := limiter.Allow(ctx, "tenant:t1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("third request should be limited")
	}
}

func TestMemoryRateLimiterResetsAfterWindow(t *testing.T) {
	limiter := NewMemoryRateLimiter(10 * time.Millisecond)
	ctx := context.Background()

	ok, err := limiter.Allow(ctx, "user:u1", 1)
	if err != nil || !ok {
		t.Fatalf("first request allowed=%v err=%v", ok, err)
	}
	ok, err = limiter.Allow(ctx, "user:u1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("second request should be limited")
	}
	time.Sleep(20 * time.Millisecond)
	ok, err = limiter.Allow(ctx, "user:u1", 1)
	if err != nil || !ok {
		t.Fatalf("request after reset allowed=%v err=%v", ok, err)
	}
}

func TestRateLimiterFallsBackToMemoryCounter(t *testing.T) {
	limiter := NewRateLimiter(failingCounter{}, NewMemoryRateLimiter(time.Minute))
	ctx := context.Background()

	ok := limiter.allow(ctx, "ip:127.0.0.1", 1)
	if !ok {
		t.Fatal("first fallback request should be allowed")
	}
	ok = limiter.allow(ctx, "ip:127.0.0.1", 1)
	if ok {
		t.Fatal("second fallback request should be limited")
	}
}

func TestIPMiddlewareReturnsChineseRateLimitError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(NewMemoryRateLimiter(30*time.Second), NewMemoryRateLimiter(30*time.Second))
	r := gin.New()
	r.GET("/x", limiter.IP(1), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("first status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("Retry-After = %q", got)
	}
	if !strings.Contains(w.Body.String(), "请求过于频繁，请稍后再试") {
		t.Fatalf("body should contain Chinese summary, got %s", w.Body.String())
	}
}

func TestTenantAndUserMiddlewareUsesSeparateKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(NewMemoryRateLimiter(time.Minute), NewMemoryRateLimiter(time.Minute))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(tenant.ContextTenantKey, tenant.Tenant{ID: "tenant-1"})
		c.Set(auth.ContextUserKey, auth.User{ID: "user-1"})
	})
	r.GET("/x", limiter.TenantAndUser(2, 1), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("first status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d", w.Code)
	}
}

type failingCounter struct{}

func (f failingCounter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	return true, errors.New("counter unavailable")
}
