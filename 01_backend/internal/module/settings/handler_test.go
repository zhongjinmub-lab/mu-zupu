package settings

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestBuildRuntimeSummaryDoesNotExposeSensitiveValues(t *testing.T) {
	summary := BuildRuntimeSummary(config.Config{
		Env:                           "prod",
		DatabaseDSN:                   "postgres://user:secret@db/prod",
		StorageEndpoint:               "minio.internal:9000",
		StorageAccessKey:              "access-key",
		StorageSecretKey:              "storage-secret",
		StorageUseSSL:                 true,
		StoragePublicBase:             "https://files.example.com",
		UploadMaxBytes:                100 << 20,
		EmbeddingProvider:             "openai_compatible",
		EmbeddingModel:                "embed-model",
		EmbeddingBaseURL:              "https://embed.example.com",
		EmbeddingAPIKey:               "embed-secret",
		GenerationProvider:            "http",
		GenerationModel:               "chat-model",
		GenerationBaseURL:             "https://chat.example.com",
		GenerationAPIKey:              "chat-secret",
		DocumentWorkerIntervalSeconds: 12,
		DocumentWorkerBatchSize:       7,
		WebhookWorkerIntervalSeconds:  15,
		WebhookWorkerBatchSize:        20,
		WebhookMaxRetries:             3,
		WebhookRetryBaseSeconds:       60,
	})

	if summary.UploadMaxMB != 100 || summary.StorageMode != "s3/minio https" || !summary.StoragePublicEnabled {
		t.Fatalf("unexpected storage summary: %#v", summary)
	}
	if !summary.EmbeddingExternalConfigured || !summary.GenerationExternalConfigured {
		t.Fatalf("external providers should be marked configured: %#v", summary)
	}
	if summary.DocumentWorkerIntervalSeconds != 12 || summary.DocumentWorkerBatchSize != 7 {
		t.Fatalf("unexpected document worker summary: %#v", summary)
	}
}

func TestRuntimeSummaryHandlerReturnsUnifiedResponseWithoutSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(config.Config{
		Env:                     "dev",
		DatabaseDSN:             "postgres://mu:secret@localhost/db",
		StorageSecretKey:        "storage-secret",
		EmbeddingAPIKey:         "embedding-secret",
		GenerationAPIKey:        "generation-secret",
		EmbeddingProvider:       "local",
		GenerationProvider:      "local",
		UploadMaxBytes:          50 << 20,
		StoragePublicBase:       "",
		StorageUseSSL:           false,
		WebhookMaxRetries:       3,
		WebhookRetryBaseSeconds: 60,
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/runtime", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"code":0`, `"env":"dev"`, `"upload_max_mb":50`, `"storage_mode":"s3/minio http"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	for _, forbidden := range []string{"secret", "postgres://", "embedding-secret", "generation-secret", "storage-secret"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response should not expose %s: %s", forbidden, body)
		}
	}
}

func TestBuildMonitoringSnapshotReturnsRuntimeMetrics(t *testing.T) {
	snapshot := BuildMonitoringSnapshot(processStartedAt.Add(2 * time.Hour))

	if snapshot.Status != "ok" {
		t.Fatalf("unexpected monitoring status: %#v", snapshot)
	}
	if snapshot.CheckedAt == "" || snapshot.Goroutines <= 0 {
		t.Fatalf("missing runtime metrics: %#v", snapshot)
	}
}

func TestMonitoringSnapshotHandlerReturnsUnifiedResponseWithoutSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(config.Config{
		DatabaseDSN:      "postgres://mu:secret@localhost/db",
		StorageSecretKey: "storage-secret",
		EmbeddingAPIKey:  "embedding-secret",
		GenerationAPIKey: "generation-secret",
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/monitoring", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"code":0`, `"status":"ok"`, `"goroutines":`, `"heap_alloc_mb":`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	for _, forbidden := range []string{"secret", "postgres://", "embedding-secret", "generation-secret", "storage-secret"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response should not expose %s: %s", forbidden, body)
		}
	}
}
