package settings

import (
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/config"
	"mu-agent-saas/pkg/response"
)

var processStartedAt = time.Now()

type Handler struct {
	cfg config.Config
}

func NewHandler(cfg config.Config) Handler {
	return Handler{cfg: cfg}
}

func (h Handler) RateLimitPolicy(c *gin.Context) {
	response.OK(c, BuildRateLimitPolicy(h.cfg))
}

func (h Handler) RuntimeSummary(c *gin.Context) {
	response.OK(c, BuildRuntimeSummary(h.cfg))
}

func (h Handler) MonitoringSnapshot(c *gin.Context) {
	response.OK(c, BuildMonitoringSnapshot(time.Now()))
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

func BuildRuntimeSummary(cfg config.Config) RuntimeSummary {
	return RuntimeSummary{
		Env:                           strings.TrimSpace(cfg.Env),
		UploadMaxMB:                   cfg.UploadMaxBytes / (1024 * 1024),
		StorageMode:                   storageMode(cfg),
		StoragePublicEnabled:          strings.TrimSpace(cfg.StoragePublicBase) != "",
		EmbeddingProvider:             normalizedProvider(cfg.EmbeddingProvider, "local"),
		EmbeddingModel:                strings.TrimSpace(cfg.EmbeddingModel),
		EmbeddingExternalConfigured:   strings.TrimSpace(cfg.EmbeddingBaseURL) != "" && strings.TrimSpace(cfg.EmbeddingAPIKey) != "",
		GenerationProvider:            normalizedProvider(cfg.GenerationProvider, "local"),
		GenerationModel:               strings.TrimSpace(cfg.GenerationModel),
		GenerationExternalConfigured:  strings.TrimSpace(cfg.GenerationBaseURL) != "" && strings.TrimSpace(cfg.GenerationAPIKey) != "",
		DocumentWorkerIntervalSeconds: cfg.DocumentWorkerIntervalSeconds,
		DocumentWorkerBatchSize:       cfg.DocumentWorkerBatchSize,
		WebhookWorkerIntervalSeconds:  cfg.WebhookWorkerIntervalSeconds,
		WebhookWorkerBatchSize:        cfg.WebhookWorkerBatchSize,
		WebhookMaxRetries:             cfg.WebhookMaxRetries,
		WebhookRetryBaseSeconds:       cfg.WebhookRetryBaseSeconds,
	}
}

func BuildMonitoringSnapshot(now time.Time) MonitoringSnapshot {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	lastGCAgoSeconds := int64(-1)
	if stats.LastGC > 0 {
		lastGC := time.Unix(0, int64(stats.LastGC))
		lastGCAgoSeconds = int64(now.Sub(lastGC).Seconds())
		if lastGCAgoSeconds < 0 {
			lastGCAgoSeconds = 0
		}
	}
	uptimeSeconds := int64(now.Sub(processStartedAt).Seconds())
	if uptimeSeconds < 0 {
		uptimeSeconds = 0
	}
	return MonitoringSnapshot{
		Status:           "ok",
		CheckedAt:        now.UTC().Format(time.RFC3339),
		UptimeSeconds:    uptimeSeconds,
		Goroutines:       runtime.NumGoroutine(),
		HeapAllocMB:      bytesToMB(stats.HeapAlloc),
		HeapSysMB:        bytesToMB(stats.HeapSys),
		HeapObjects:      stats.HeapObjects,
		GCCount:          stats.NumGC,
		LastGCAgoSeconds: lastGCAgoSeconds,
	}
}

func normalizedProvider(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func bytesToMB(value uint64) uint64 {
	return value / 1024 / 1024
}

func storageMode(cfg config.Config) string {
	if cfg.StorageUseSSL {
		return "s3/minio https"
	}
	return "s3/minio http"
}

func redisFallbackLabel(enabled bool) string {
	if enabled {
		return "Redis 计数异常时自动回退内存限流"
	}
	return "当前使用内存限流，适合单实例或本地环境"
}
