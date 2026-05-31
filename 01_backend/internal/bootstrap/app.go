package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"mu-agent-saas/internal/config"
	"mu-agent-saas/internal/embedding"
	"mu-agent-saas/internal/generation"
	"mu-agent-saas/internal/middleware"
	"mu-agent-saas/internal/module/agent"
	"mu-agent-saas/internal/module/analytics"
	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/billing"
	filemodule "mu-agent-saas/internal/module/file"
	"mu-agent-saas/internal/module/kb"
	"mu-agent-saas/internal/module/license"
	"mu-agent-saas/internal/module/settings"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/internal/module/webhook"
	"mu-agent-saas/internal/module/workflow"
	"mu-agent-saas/pkg/database"
	"mu-agent-saas/pkg/response"
	"mu-agent-saas/pkg/storage"
)

type App struct {
	Router      *gin.Engine
	DB          *pgxpool.Pool
	RedisClient *redis.Client
}

func NewApp(cfg config.Config) (*App, error) {
	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	if cfg.Env == "prod" || cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID())

	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			response.Error(c, http.StatusServiceUnavailable, 50301, "database unavailable")
			return
		}
		response.OK(c, gin.H{"status": "ok", "service": "智能体族谱SAAS"})
	})
	v1.GET("/ready", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ready"})
	})

	authRepo := auth.NewRepository(db)
	jwtSvc, err := auth.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTLHours)
	if err != nil {
		return nil, err
	}
	authHandler := auth.NewHandler(auth.NewService(authRepo, jwtSvc))
	rateLimitWindow := time.Duration(cfg.RateLimitWindowSeconds) * time.Second
	memoryRateLimiter := middleware.NewMemoryRateLimiter(rateLimitWindow)
	rateLimiter := middleware.NewRateLimiter(memoryRateLimiter, memoryRateLimiter)
	var redisClient *redis.Client
	if strings.EqualFold(strings.TrimSpace(cfg.RateLimitBackend), "redis") {
		redisClient = redis.NewClient(&redis.Options{
			Addr:         cfg.RedisAddr,
			Password:     cfg.RedisPass,
			DB:           cfg.RedisDB,
			DialTimeout:  300 * time.Millisecond,
			ReadTimeout:  300 * time.Millisecond,
			WriteTimeout: 300 * time.Millisecond,
		})
		rateLimitPrefix := "mu:" + cfg.Env + ":rate:global:"
		rateLimiter = middleware.NewRateLimiter(middleware.NewRedisRateLimiter(redisClient, rateLimitWindow, rateLimitPrefix), memoryRateLimiter)
	}
	private := v1.Group("")
	private.Use(auth.AuthMiddleware(jwtSvc, authRepo))
	authPublic := v1.Group("")
	authPublic.Use(rateLimiter.IP(cfg.RateLimitAuthIPPerMinute))
	auth.RegisterRoutes(authPublic, private, authHandler)

	tenantRepo := tenant.NewRepository(db)
	tenantHandler := tenant.NewHandler(tenantRepo)
	tenant.RegisterRoutes(private, tenantHandler)
	settings.RegisterRoutes(private, settings.NewHandler(cfg))

	tenantScoped := private.Group("")
	tenantScoped.Use(tenant.ContextMiddleware(tenantRepo))
	tenantScoped.Use(rateLimiter.TenantAndUser(cfg.RateLimitTenantPerMinute, cfg.RateLimitUserPerMinute))
	tenantScoped.Use(tenant.AuditMiddleware(tenantRepo))
	tenant.RegisterTenantScopedRoutes(tenantScoped, tenantHandler)
	storageClient, err := storage.NewMinIO(storage.Config{
		Endpoint:   cfg.StorageEndpoint,
		AccessKey:  cfg.StorageAccessKey,
		SecretKey:  cfg.StorageSecretKey,
		Bucket:     cfg.StorageBucket,
		UseSSL:     cfg.StorageUseSSL,
		PublicBase: cfg.StoragePublicBase,
	})
	if err != nil {
		return nil, err
	}
	providerName := strings.ToLower(strings.TrimSpace(cfg.EmbeddingProvider))
	var embedder embedding.Provider
	switch providerName {
	case "", "local":
		embedder, err = embedding.NewProvider(providerName, cfg.EmbeddingModel)
	case "openai_compatible", "http":
		embedder, err = embedding.NewHTTPProvider(embedding.HTTPConfig{
			Provider:       providerName,
			Model:          cfg.EmbeddingModel,
			BaseURL:        cfg.EmbeddingBaseURL,
			APIKey:         cfg.EmbeddingAPIKey,
			TimeoutSeconds: cfg.EmbeddingTimeout,
		})
	default:
		embedder, err = embedding.NewProvider(providerName, cfg.EmbeddingModel)
	}
	if err != nil {
		return nil, err
	}
	generatorName := strings.ToLower(strings.TrimSpace(cfg.GenerationProvider))
	var generator generation.Provider
	switch generatorName {
	case "", "local":
		generator, err = generation.NewProvider(generatorName, cfg.GenerationModel)
	case "openai_compatible", "http":
		generator, err = generation.NewHTTPProvider(generation.HTTPConfig{
			Provider:       generatorName,
			Model:          cfg.GenerationModel,
			BaseURL:        cfg.GenerationBaseURL,
			APIKey:         cfg.GenerationAPIKey,
			TimeoutSeconds: cfg.GenerationTimeout,
		})
	default:
		generator, err = generation.NewProvider(generatorName, cfg.GenerationModel)
	}
	if err != nil {
		return nil, err
	}
	billingRepo := billing.NewRepository(db)
	webhookRepo := webhook.NewRepository(db)
	webhookService := webhook.NewService(webhookRepo, webhook.ServiceOptions{
		MaxRetries:       cfg.WebhookMaxRetries,
		RetryBaseSeconds: cfg.WebhookRetryBaseSeconds,
	})
	analytics.RegisterRoutes(tenantScoped, analytics.NewHandler(analytics.NewRepository(db)))
	billing.RegisterRoutes(tenantScoped, billing.NewHandlerWithWebhookAndPaymentSecret(billingRepo, webhookService, cfg.PaymentCallbackSecret))
	licenseVerifier, err := license.NewVerifierFromConfig(cfg.LicensePublicKeys)
	if err != nil {
		return nil, err
	}
	license.RegisterRoutes(tenantScoped, license.NewHandlerWithWebhook(license.NewRepository(db), licenseVerifier, webhookService))
	webhook.RegisterRoutes(tenantScoped, webhook.NewHandler(webhookRepo, webhookService), rateLimiter.TenantAndUserScoped("webhook_test", rateLimitShare(cfg.RateLimitTenantPerMinute, 2), rateLimitShare(cfg.RateLimitUserPerMinute, 2)))
	filemodule.RegisterRoutes(tenantScoped, filemodule.NewHandler(filemodule.NewRepository(db), storageClient, cfg.UploadMaxBytes, billingRepo))
	kb.RegisterRoutesWithHandler(tenantScoped, kb.NewHandlerWithStorageAndGeneration(db, storageClient, embedder, generator, billingRepo))
	agent.RegisterRoutes(tenantScoped, agent.NewHandlerWithRuntimeAndWebhook(agent.NewRepository(db), kb.NewRepository(db), kb.NewVectorRepository(db), embedder, generator, billingRepo, webhookService), rateLimiter.TenantAndUserScoped("agent_stream", rateLimitShare(cfg.RateLimitTenantPerMinute, 3), rateLimitShare(cfg.RateLimitUserPerMinute, 3)))
	workflow.RegisterRoutes(tenantScoped, workflow.NewHandler())
	return &App{Router: r, DB: db, RedisClient: redisClient}, nil
}

func (a *App) Close() {
	if a != nil && a.DB != nil {
		a.DB.Close()
	}
	if a != nil && a.RedisClient != nil {
		_ = a.RedisClient.Close()
	}
}

func rateLimitShare(base, divisor int) int {
	if base <= 0 {
		return 0
	}
	if divisor <= 1 {
		return base
	}
	value := base / divisor
	if value < 1 {
		return 1
	}
	return value
}
