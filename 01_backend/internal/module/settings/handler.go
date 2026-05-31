package settings

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/config"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	cfg config.Config
}

func NewHandler(cfg config.Config) Handler {
	return Handler{cfg: cfg}
}

func (h Handler) RateLimitPolicy(c *gin.Context) {
	response.OK(c, BuildRateLimitPolicy(h.cfg))
}

func BuildRateLimitPolicy(cfg config.Config) RateLimitPolicy {
	backend := strings.ToLower(strings.TrimSpace(cfg.RateLimitBackend))
	if backend != "redis" {
		backend = "memory"
	}
	redisEnabled := backend == "redis"
	return RateLimitPolicy{
		Backend:            backend,
		WindowSeconds:      cfg.RateLimitWindowSeconds,
		TenantPerWindow:    cfg.RateLimitTenantPerMinute,
		UserPerWindow:      cfg.RateLimitUserPerMinute,
		AuthIPPerWindow:    cfg.RateLimitAuthIPPerMinute,
		RedisEnabled:       redisEnabled,
		RedisFallbackLabel: redisFallbackLabel(redisEnabled),
	}
}

func redisFallbackLabel(enabled bool) string {
	if enabled {
		return "Redis 计数异常时自动回退内存限流"
	}
	return "当前使用内存限流，适合单实例或本地环境"
}
