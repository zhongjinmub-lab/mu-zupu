package config

import "testing"

func TestLoadReadsJWTConfig(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("JWT_ISSUER", "test-issuer")
	t.Setenv("JWT_TTL_HOURS", "2")
	t.Setenv("STORAGE_ENDPOINT", "minio:9000")
	t.Setenv("STORAGE_ACCESS_KEY", "access")
	t.Setenv("STORAGE_SECRET_KEY", "secret")
	t.Setenv("STORAGE_BUCKET", "files")
	t.Setenv("STORAGE_USE_SSL", "true")
	t.Setenv("UPLOAD_MAX_BYTES", "1024")
	t.Setenv("EMBEDDING_PROVIDER", "local")
	t.Setenv("EMBEDDING_MODEL", "test-embedding")
	t.Setenv("EMBEDDING_BASE_URL", "https://embedding.example/v1")
	t.Setenv("EMBEDDING_API_KEY", "test-key")
	t.Setenv("EMBEDDING_TIMEOUT_SECONDS", "7")
	t.Setenv("GENERATION_PROVIDER", "openai_compatible")
	t.Setenv("GENERATION_MODEL", "test-chat")
	t.Setenv("GENERATION_BASE_URL", "https://llm.example/v1")
	t.Setenv("GENERATION_API_KEY", "chat-key")
	t.Setenv("GENERATION_TIMEOUT_SECONDS", "17")
	t.Setenv("RATE_LIMIT_BACKEND", "redis")
	t.Setenv("RATE_LIMIT_WINDOW_SECONDS", "30")
	t.Setenv("RATE_LIMIT_TENANT_PER_MINUTE", "240")
	t.Setenv("RATE_LIMIT_USER_PER_MINUTE", "90")
	t.Setenv("RATE_LIMIT_AUTH_IP_PER_MINUTE", "25")
	t.Setenv("DOCUMENT_WORKER_INTERVAL_SECONDS", "3")
	t.Setenv("DOCUMENT_WORKER_BATCH_SIZE", "9")
	t.Setenv("PAYMENT_CALLBACK_SECRET", "payment-callback-test-secret")
	t.Setenv("PAYMENT_CHANNELS", "mock,alipay")
	t.Setenv("PAYMENT_NOTIFY_BASE_URL", "https://api.example.com")
	t.Setenv("PAYMENT_RETURN_URL", "https://app.example.com/return")
	t.Setenv("ALIPAY_APP_ID", "2021000000000000")
	t.Setenv("ALIPAY_PRIVATE_KEY", "alipay-private-key")
	t.Setenv("ALIPAY_PUBLIC_KEY", "alipay-public-key")
	t.Setenv("ALIPAY_GATEWAY", "https://openapi.alipaydev.com/gateway.do")
	t.Setenv("LICENSE_PUBLIC_KEYS", `{"default":"pubkey"}`)

	cfg := Load()
	if cfg.Env != "production" {
		t.Fatalf("Env = %q", cfg.Env)
	}
	if cfg.JWTSecret != "12345678901234567890123456789012" {
		t.Fatal("JWTSecret was not loaded from env")
	}
	if cfg.JWTIssuer != "test-issuer" {
		t.Fatalf("JWTIssuer = %q", cfg.JWTIssuer)
	}
	if cfg.JWTTTLHours != 2 {
		t.Fatalf("JWTTTLHours = %d", cfg.JWTTTLHours)
	}
	if cfg.StorageEndpoint != "minio:9000" || cfg.StorageBucket != "files" || !cfg.StorageUseSSL {
		t.Fatalf("storage config = %#v", cfg)
	}
	if cfg.UploadMaxBytes != 1024 {
		t.Fatalf("UploadMaxBytes = %d", cfg.UploadMaxBytes)
	}
	if cfg.EmbeddingProvider != "local" || cfg.EmbeddingModel != "test-embedding" {
		t.Fatalf("embedding config = %#v", cfg)
	}
	if cfg.EmbeddingBaseURL != "https://embedding.example/v1" || cfg.EmbeddingAPIKey != "test-key" || cfg.EmbeddingTimeout != 7 {
		t.Fatalf("embedding http config = %#v", cfg)
	}
	if cfg.GenerationProvider != "openai_compatible" || cfg.GenerationModel != "test-chat" {
		t.Fatalf("generation config = %#v", cfg)
	}
	if cfg.GenerationBaseURL != "https://llm.example/v1" || cfg.GenerationAPIKey != "chat-key" || cfg.GenerationTimeout != 17 {
		t.Fatalf("generation http config = %#v", cfg)
	}
	if cfg.RateLimitBackend != "redis" || cfg.RateLimitWindowSeconds != 30 {
		t.Fatalf("rate limit backend config = %#v", cfg)
	}
	if cfg.RateLimitTenantPerMinute != 240 || cfg.RateLimitUserPerMinute != 90 || cfg.RateLimitAuthIPPerMinute != 25 {
		t.Fatalf("rate limit threshold config = %#v", cfg)
	}
	if cfg.DocumentWorkerIntervalSeconds != 3 || cfg.DocumentWorkerBatchSize != 9 {
		t.Fatalf("document worker config = %#v", cfg)
	}
	if cfg.PaymentCallbackSecret != "payment-callback-test-secret" {
		t.Fatalf("PaymentCallbackSecret = %q", cfg.PaymentCallbackSecret)
	}
	if cfg.PaymentChannels != "mock,alipay" || cfg.PaymentNotifyBaseURL != "https://api.example.com" {
		t.Fatalf("payment channel config = %#v", cfg)
	}
	if cfg.PaymentReturnURL != "https://app.example.com/return" {
		t.Fatalf("PaymentReturnURL = %q", cfg.PaymentReturnURL)
	}
	if cfg.AlipayAppID != "2021000000000000" || cfg.AlipayPrivateKey != "alipay-private-key" || cfg.AlipayPublicKey != "alipay-public-key" {
		t.Fatalf("alipay credential config = %#v", cfg)
	}
	if cfg.AlipayGateway != "https://openapi.alipaydev.com/gateway.do" || cfg.AlipaySignType != "RSA2" {
		t.Fatalf("alipay gateway config = %#v", cfg)
	}
	if cfg.LicensePublicKeys != `{"default":"pubkey"}` {
		t.Fatalf("LicensePublicKeys = %q", cfg.LicensePublicKeys)
	}
}
