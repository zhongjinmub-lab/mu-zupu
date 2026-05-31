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
