package settings

type RateLimitPolicy struct {
	Backend            string `json:"backend"`
	WindowSeconds      int    `json:"window_seconds"`
	TenantPerWindow    int    `json:"tenant_per_window"`
	UserPerWindow      int    `json:"user_per_window"`
	AuthIPPerWindow    int    `json:"auth_ip_per_window"`
	RedisEnabled       bool   `json:"redis_enabled"`
	RedisFallbackLabel string `json:"redis_fallback_label"`
}

type RuntimeSummary struct {
	Env                           string `json:"env"`
	UploadMaxMB                   int64  `json:"upload_max_mb"`
	StorageMode                   string `json:"storage_mode"`
	StoragePublicEnabled          bool   `json:"storage_public_enabled"`
	EmbeddingProvider             string `json:"embedding_provider"`
	EmbeddingModel                string `json:"embedding_model"`
	EmbeddingExternalConfigured   bool   `json:"embedding_external_configured"`
	GenerationProvider            string `json:"generation_provider"`
	GenerationModel               string `json:"generation_model"`
	GenerationExternalConfigured  bool   `json:"generation_external_configured"`
	DocumentWorkerIntervalSeconds int    `json:"document_worker_interval_seconds"`
	DocumentWorkerBatchSize       int    `json:"document_worker_batch_size"`
	WebhookWorkerIntervalSeconds  int    `json:"webhook_worker_interval_seconds"`
	WebhookWorkerBatchSize        int    `json:"webhook_worker_batch_size"`
	WebhookMaxRetries             int    `json:"webhook_max_retries"`
	WebhookRetryBaseSeconds       int    `json:"webhook_retry_base_seconds"`
}

type MonitoringSnapshot struct {
	Status           string `json:"status"`
	CheckedAt        string `json:"checked_at"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	Goroutines       int    `json:"goroutines"`
	HeapAllocMB      uint64 `json:"heap_alloc_mb"`
	HeapSysMB        uint64 `json:"heap_sys_mb"`
	HeapObjects      uint64 `json:"heap_objects"`
	GCCount          uint32 `json:"gc_count"`
	LastGCAgoSeconds int64  `json:"last_gc_ago_seconds"`
}

type SensitiveFieldSummary struct {
	EnvironmentSecrets []SensitiveFieldItem `json:"environment_secrets"`
	StoredSecrets      []SensitiveFieldItem `json:"stored_secrets"`
	ResponseRedactions []SensitiveFieldItem `json:"response_redactions"`
	OperationalNotes   []string             `json:"operational_notes"`
}

type SensitiveFieldItem struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Protection  string `json:"protection"`
	APIExposure string `json:"api_exposure"`
	Configured   bool   `json:"configured"`
}
