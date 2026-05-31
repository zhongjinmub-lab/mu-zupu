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
	if len(policy.ScopedPolicies) < 4 {
		t.Fatalf("expected scoped rate limit policies: %#v", policy.ScopedPolicies)
	}
	foundWebhook := false
	foundStream := false
	for _, item := range policy.ScopedPolicies {
		if item.Scope == "webhook_test" && item.TenantPerWindow == 120 && item.UserPerWindow == 45 {
			foundWebhook = true
		}
		if item.Scope == "agent_stream" && item.TenantPerWindow == 80 && item.UserPerWindow == 30 {
			foundStream = true
		}
	}
	if !foundWebhook || !foundStream {
		t.Fatalf("expected webhook and stream scoped policies: %#v", policy.ScopedPolicies)
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
	for _, want := range []string{`"code":0`, `"backend":"memory"`, `"tenant_per_window":120`, `"redis_enabled":false`, `"scoped_policies"`, `"webhook_test"`, `"agent_stream"`} {
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

func TestBuildSensitiveFieldSummaryDoesNotExposeSecretValues(t *testing.T) {
	summary := BuildSensitiveFieldSummary(config.Config{
		DatabaseDSN:           "postgres://user:secret@db/prod",
		RedisPass:             "redis-secret",
		JWTSecret:             "jwt-secret-value",
		StorageAccessKey:      "storage-access",
		StorageSecretKey:      "storage-secret",
		EmbeddingAPIKey:       "embed-secret",
		GenerationAPIKey:      "chat-secret",
		PaymentCallbackSecret: "payment-secret",
	})

	if len(summary.EnvironmentSecrets) == 0 || len(summary.StoredSecrets) == 0 || len(summary.ResponseRedactions) == 0 {
		t.Fatalf("expected sensitive field summary sections: %#v", summary)
	}
	text := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(strings.Join(summary.OperationalNotes, " ")), "\n", " "))
	if strings.Contains(text, "jwt-secret-value") || strings.Contains(text, "postgres://") || strings.Contains(text, "storage-secret") {
		t.Fatalf("summary notes should not expose secret values: %#v", summary.OperationalNotes)
	}
	foundPayment := false
	for _, item := range summary.EnvironmentSecrets {
		if strings.Contains(item.Scope, "PAYMENT_CALLBACK_SECRET") && strings.Contains(item.APIExposure, "不返回") && item.Configured {
			foundPayment = true
		}
		if strings.Contains(item.Protection, "secret@db") || strings.Contains(item.Protection, "embed-secret") {
			t.Fatalf("summary should not expose secret values: %#v", item)
		}
	}
	if !foundPayment {
		t.Fatalf("expected configured payment callback secret summary: %#v", summary.EnvironmentSecrets)
	}
}

func TestSensitiveFieldSummaryHandlerReturnsUnifiedResponseWithoutSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(config.Config{
		DatabaseDSN:           "postgres://mu:secret@localhost/db",
		RedisPass:             "redis-secret",
		JWTSecret:             "jwt-secret-value",
		StorageSecretKey:      "storage-secret",
		EmbeddingAPIKey:       "embedding-secret",
		GenerationAPIKey:      "generation-secret",
		PaymentCallbackSecret: "payment-secret",
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/sensitive-fields", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"code":0`, `"environment_secrets"`, `"stored_secrets"`, `"response_redactions"`, `"users.password_hash"`, "has_secret"} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	for _, forbidden := range []string{"postgres://", "jwt-secret-value", "storage-secret", "embedding-secret", "generation-secret", "payment-secret", "redis-secret"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response should not expose %s: %s", forbidden, body)
		}
	}
}

func TestBuildRateLimitAuditSummaryIncludesCoverage(t *testing.T) {
	summary := BuildRateLimitAuditSummary(config.Config{
		RateLimitBackend:         "memory",
		RateLimitWindowSeconds:   60,
		RateLimitTenantPerMinute: 120,
		RateLimitUserPerMinute:   60,
		RateLimitAuthIPPerMinute: 20,
		RedisPass:                "redis-secret",
	})

	if summary.RateLimit.Backend != "memory" || summary.RateLimit.TenantPerWindow != 120 {
		t.Fatalf("unexpected rate limit summary: %#v", summary.RateLimit)
	}
	if len(summary.Audit.AutomaticActions) < 4 || len(summary.Audit.BusinessActions) == 0 {
		t.Fatalf("expected audit coverage: %#v", summary.Audit)
	}
	if !summary.Audit.ExportEnabled {
		t.Fatalf("audit export should be enabled: %#v", summary.Audit)
	}
	foundPost := false
	for _, item := range summary.Audit.AutomaticActions {
		if item.Action == "http.post" && item.ResourceType == "http_request" {
			foundPost = true
		}
	}
	if !foundPost {
		t.Fatalf("expected http.post audit coverage: %#v", summary.Audit.AutomaticActions)
	}
}

func TestRateLimitAuditSummaryHandlerReturnsUnifiedResponseWithoutRedisSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(config.Config{
		RateLimitBackend:         "redis",
		RateLimitWindowSeconds:   30,
		RateLimitTenantPerMinute: 240,
		RateLimitUserPerMinute:   90,
		RateLimitAuthIPPerMinute: 25,
		RedisAddr:                "redis.internal:6379",
		RedisPass:                "redis-secret",
		RedisDB:                  3,
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/rate-limit-audit", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"code":0`, `"rate_limit"`, `"audit"`, `"http.post"`, `"tenant.member.add"`, `"export_enabled":true`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	for _, forbidden := range []string{"redis-secret", "redis.internal:6379"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response should not expose %s: %s", forbidden, body)
		}
	}
}

func TestBuildVectorSearchSummaryIncludesIsolationAndIndexChecks(t *testing.T) {
	summary := BuildVectorSearchSummary(config.Config{
		EmbeddingProvider: " openai_compatible ",
		EmbeddingModel:    "text-embedding-3-small",
		EmbeddingAPIKey:   "embedding-secret",
		DatabaseDSN:       "postgres://mu:secret@localhost/db",
	})

	if summary.Status != "ready" || summary.EmbeddingProvider != "openai_compatible" {
		t.Fatalf("unexpected vector search summary: %#v", summary)
	}
	if summary.EmbeddingDimension != 1536 || summary.IndexProfile.Extension != "pgvector" || summary.IndexProfile.IndexMethod != "HNSW" {
		t.Fatalf("unexpected index profile: %#v", summary.IndexProfile)
	}
	if len(summary.IsolationChecks) < 3 || len(summary.RetrievalChecks) < 3 || len(summary.OperationsChecks) < 3 {
		t.Fatalf("expected complete vector checks: %#v", summary)
	}
	foundTenantIsolation := false
	foundSearchLog := false
	for _, item := range summary.IsolationChecks {
		if item.Name == "租户隔离" && item.Status == "covered" {
			foundTenantIsolation = true
		}
	}
	for _, item := range summary.RetrievalChecks {
		if item.Name == "检索日志" && item.Status == "covered" {
			foundSearchLog = true
		}
	}
	if !foundTenantIsolation || !foundSearchLog {
		t.Fatalf("expected tenant isolation and search log checks: %#v %#v", summary.IsolationChecks, summary.RetrievalChecks)
	}
}

func TestVectorSearchSummaryHandlerReturnsUnifiedResponseWithoutSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(config.Config{
		DatabaseDSN:     "postgres://mu:secret@localhost/db",
		EmbeddingModel:  "local-hash-1536",
		EmbeddingAPIKey: "embedding-secret",
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/vector-search", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"code":0`, `"embedding_dimension":1536`, `"extension":"pgvector"`, `"index_method":"HNSW"`, `"租户隔离"`, `"检索日志"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	for _, forbidden := range []string{"postgres://", "secret@localhost", "embedding-secret"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response should not expose %s: %s", forbidden, body)
		}
	}
}
