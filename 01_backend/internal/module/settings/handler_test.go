package settings

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/config"
)

func TestBuildRateLimitPolicyDoesNotExposeRedisConnection(t *testing.T) {
	policy := BuildRateLimitPolicy(config.Config{
		RateLimitBackend:         " redis ",
		RateLimitWindowSeconds:   30,
		RateLimitTenantPerMinute: 240,
		RateLimitUserPerMinute:   90,
		RateLimitAuthIPPerMinute: 25,
		RedisAddr:                "redis.internal:6379",
		RedisPass:                "secret",
		RedisDB:                  2,
	})

	if policy.Backend != "redis" || !policy.RedisEnabled {
		t.Fatalf("unexpected backend policy: %#v", policy)
	}
	if policy.WindowSeconds != 30 || policy.TenantPerWindow != 240 || policy.UserPerWindow != 90 || policy.AuthIPPerWindow != 25 {
		t.Fatalf("unexpected policy limits: %#v", policy)
	}
}

func TestRateLimitPolicyHandlerReturnsUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(config.Config{
		RateLimitBackend:         "memory",
		RateLimitWindowSeconds:   60,
		RateLimitTenantPerMinute: 120,
		RateLimitUserPerMinute:   60,
		RateLimitAuthIPPerMinute: 20,
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/rate-limit", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"code":0`, `"backend":"memory"`, `"tenant_per_window":120`, `"redis_enabled":false`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, "RedisAddr") || strings.Contains(body, "secret") {
		t.Fatalf("response should not expose redis connection details: %s", body)
	}
}
