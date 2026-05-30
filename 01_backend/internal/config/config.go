package config

import (
	"os"
	"strconv"
)

type Config struct {
	Env                           string
	HTTPAddr                      string
	DatabaseDSN                   string
	RedisAddr                     string
	RedisPass                     string
	RedisDB                       int
	RateLimitBackend              string
	RateLimitWindowSeconds        int
	RateLimitTenantPerMinute      int
	RateLimitUserPerMinute        int
	RateLimitAuthIPPerMinute      int
	JWTSecret                     string
	JWTIssuer                     string
	JWTTTLHours                   int
	StorageEndpoint               string
	StorageAccessKey              string
	StorageSecretKey              string
	StorageBucket                 string
	StorageUseSSL                 bool
	StoragePublicBase             string
	UploadMaxBytes                int64
	EmbeddingProvider             string
	EmbeddingModel                string
	EmbeddingBaseURL              string
	EmbeddingAPIKey               string
	EmbeddingTimeout              int
	GenerationProvider            string
	GenerationModel               string
	GenerationBaseURL             string
	GenerationAPIKey              string
	GenerationTimeout             int
	DocumentWorkerIntervalSeconds int
	DocumentWorkerBatchSize       int
	WebhookWorkerIntervalSeconds  int
	WebhookWorkerBatchSize        int
	WebhookMaxRetries             int
	WebhookRetryBaseSeconds       int
	LicensePublicKeys             string
}

func Load() Config {
	return Config{
		Env:                           env("APP_ENV", "dev"),
		HTTPAddr:                      env("HTTP_ADDR", ":8080"),
		DatabaseDSN:                   env("DATABASE_DSN", "postgres://mu:mu_password@localhost:5432/mu_agent_saas?sslmode=disable"),
		RedisAddr:                     env("REDIS_ADDR", "localhost:6379"),
		RedisPass:                     env("REDIS_PASS", ""),
		RedisDB:                       envInt("REDIS_DB", 0),
		RateLimitBackend:              env("RATE_LIMIT_BACKEND", "memory"),
		RateLimitWindowSeconds:        envInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitTenantPerMinute:      envInt("RATE_LIMIT_TENANT_PER_MINUTE", 120),
		RateLimitUserPerMinute:        envInt("RATE_LIMIT_USER_PER_MINUTE", 60),
		RateLimitAuthIPPerMinute:      envInt("RATE_LIMIT_AUTH_IP_PER_MINUTE", 20),
		JWTSecret:                     env("JWT_SECRET", ""),
		JWTIssuer:                     env("JWT_ISSUER", "mu-agent-saas"),
		JWTTTLHours:                   envInt("JWT_TTL_HOURS", 24),
		StorageEndpoint:               env("STORAGE_ENDPOINT", "127.0.0.1:19000"),
		StorageAccessKey:              env("STORAGE_ACCESS_KEY", "muadmin"),
		StorageSecretKey:              env("STORAGE_SECRET_KEY", ""),
		StorageBucket:                 env("STORAGE_BUCKET", "mu-agent-files"),
		StorageUseSSL:                 envBool("STORAGE_USE_SSL", false),
		StoragePublicBase:             env("STORAGE_PUBLIC_BASE", ""),
		UploadMaxBytes:                envInt64("UPLOAD_MAX_BYTES", 50<<20),
		EmbeddingProvider:             env("EMBEDDING_PROVIDER", "local"),
		EmbeddingModel:                env("EMBEDDING_MODEL", "local-hash-1536"),
		EmbeddingBaseURL:              env("EMBEDDING_BASE_URL", ""),
		EmbeddingAPIKey:               env("EMBEDDING_API_KEY", ""),
		EmbeddingTimeout:              envInt("EMBEDDING_TIMEOUT_SECONDS", 30),
		GenerationProvider:            env("GENERATION_PROVIDER", "local"),
		GenerationModel:               env("GENERATION_MODEL", "local-rag"),
		GenerationBaseURL:             env("GENERATION_BASE_URL", ""),
		GenerationAPIKey:              env("GENERATION_API_KEY", ""),
		GenerationTimeout:             envInt("GENERATION_TIMEOUT_SECONDS", 60),
		DocumentWorkerIntervalSeconds: envInt("DOCUMENT_WORKER_INTERVAL_SECONDS", 10),
		DocumentWorkerBatchSize:       envInt("DOCUMENT_WORKER_BATCH_SIZE", 5),
		WebhookWorkerIntervalSeconds:  envInt("WEBHOOK_WORKER_INTERVAL_SECONDS", 15),
		WebhookWorkerBatchSize:        envInt("WEBHOOK_WORKER_BATCH_SIZE", 20),
		WebhookMaxRetries:             envInt("WEBHOOK_MAX_RETRIES", 3),
		WebhookRetryBaseSeconds:       envInt("WEBHOOK_RETRY_BASE_SECONDS", 60),
		LicensePublicKeys:             env("LICENSE_PUBLIC_KEYS", ""),
	}
}

func env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func envInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return i
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
