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

func (h Handler) SensitiveFieldSummary(c *gin.Context) {
	response.OK(c, BuildSensitiveFieldSummary(h.cfg))
}

func (h Handler) RateLimitAuditSummary(c *gin.Context) {
	response.OK(c, BuildRateLimitAuditSummary(h.cfg))
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

func BuildSensitiveFieldSummary(cfg config.Config) SensitiveFieldSummary {
	return SensitiveFieldSummary{
		EnvironmentSecrets: []SensitiveFieldItem{
			{
				Name:        "JWT 签名密钥",
				Scope:       "JWT_SECRET",
				Protection:  "仅从环境变量读取，用于签发和校验登录令牌",
				APIExposure: "不通过任何 API 返回原文",
				Configured:   strings.TrimSpace(cfg.JWTSecret) != "",
			},
			{
				Name:        "数据库连接串",
				Scope:       "DATABASE_DSN",
				Protection:  "仅在后端进程内使用，设置页只展示运行摘要",
				APIExposure: "不返回 DSN、用户名或密码",
				Configured:   strings.TrimSpace(cfg.DatabaseDSN) != "",
			},
			{
				Name:        "对象存储密钥",
				Scope:       "STORAGE_ACCESS_KEY / STORAGE_SECRET_KEY",
				Protection:  "仅用于服务端访问 MinIO/S3，前端通过业务接口访问文件",
				APIExposure: "不返回 AccessKey 或 SecretKey",
				Configured:   strings.TrimSpace(cfg.StorageSecretKey) != "",
			},
			{
				Name:        "模型服务 API Key",
				Scope:       "EMBEDDING_API_KEY / GENERATION_API_KEY",
				Protection:  "仅用于服务端调用外部模型 Provider",
				APIExposure: "只返回是否已配置，不返回 Key 原文",
				Configured:   strings.TrimSpace(cfg.EmbeddingAPIKey) != "" || strings.TrimSpace(cfg.GenerationAPIKey) != "",
			},
			{
				Name:        "支付回调验签密钥",
				Scope:       "PAYMENT_CALLBACK_SECRET",
				Protection:  "仅从环境变量读取，用于 HMAC 验证支付回调",
				APIExposure: "不入库、不返回、不展示",
				Configured:   strings.TrimSpace(cfg.PaymentCallbackSecret) != "",
			},
		},
		StoredSecrets: []SensitiveFieldItem{
			{
				Name:        "用户密码",
				Scope:       "users.password_hash",
				Protection:  "数据库仅保存密码哈希，登录时做哈希校验",
				APIExposure: "用户接口不返回 password_hash",
				Configured:   true,
			},
			{
				Name:        "租户邀请 Token",
				Scope:       "tenant_invitations.token_hash",
				Protection:  "数据库仅保存 token_hash，明文 Token 只在创建响应中返回一次",
				APIExposure: "列表接口不返回明文 Token",
				Configured:   true,
			},
			{
				Name:        "Webhook 签名密钥",
				Scope:       "webhook_endpoints.secret",
				Protection:  "数据库保留密钥用于 HMAC 投递签名，编辑留空时保留原密钥",
				APIExposure: "Webhook 响应只返回 has_secret",
				Configured:   true,
			},
			{
				Name:        "License 离线签名",
				Scope:       "licenses.signature",
				Protection:  "数据库保留签名用于离线验签",
				APIExposure: "License 响应只返回 has_signature",
				Configured:   true,
			},
		},
		ResponseRedactions: []SensitiveFieldItem{
			{
				Name:        "运行配置摘要",
				Scope:       "GET /settings/runtime",
				Protection:  "按 Provider、存储模式和 Worker 参数输出中文摘要",
				APIExposure: "不返回 DSN、Redis 密码、对象存储密钥或模型 Key",
				Configured:   true,
			},
			{
				Name:        "限流策略摘要",
				Scope:       "GET /settings/rate-limit",
				Protection:  "只展示限流后端与窗口阈值",
				APIExposure: "不返回 Redis 地址、密码或数据库编号",
				Configured:   true,
			},
			{
				Name:        "运行监控摘要",
				Scope:       "GET /settings/monitoring",
				Protection:  "只返回进程健康和内存指标",
				APIExposure: "不返回任何连接串或密钥",
				Configured:   true,
			},
		},
		OperationalNotes: []string{
			"生产密钥通过环境变量或部署密钥注入，不写入前端静态文件。",
			"接口响应使用 has_secret、has_signature、configured 等布尔摘要替代敏感原文。",
			"当前版本已覆盖哈希存储、环境变量隔离和响应脱敏；如需字段级 KMS 加密，可在后续版本扩展。",
		},
	}
}

func BuildRateLimitAuditSummary(cfg config.Config) RateLimitAuditSummary {
	return RateLimitAuditSummary{
		RateLimit: BuildRateLimitPolicy(cfg),
		Audit: AuditCoverage{
			Scope: "租户级接口在鉴权和租户上下文校验后统一进入限流与审计链路",
			AutomaticActions: []AuditActionPolicy{
				{
					Action:       "http.post",
					ResourceType: "http_request",
					Description:  "记录租户级 POST 写操作，包含路由、状态码、请求 ID 和角色",
				},
				{
					Action:       "http.put",
					ResourceType: "http_request",
					Description:  "记录租户级 PUT 写操作，包含路由、状态码、请求 ID 和角色",
				},
				{
					Action:       "http.patch",
					ResourceType: "http_request",
					Description:  "记录租户级 PATCH 写操作，包含路由、状态码、请求 ID 和角色",
				},
				{
					Action:       "http.delete",
					ResourceType: "http_request",
					Description:  "记录租户级 DELETE 写操作，包含路由、状态码、请求 ID 和角色",
				},
			},
			BusinessActions: []AuditActionPolicy{
				{
					Action:       "tenant.member.add",
					ResourceType: "tenant_member",
					Description:  "记录成员添加及角色信息",
				},
				{
					Action:       "tenant.member.role_update",
					ResourceType: "tenant_member",
					Description:  "记录成员角色调整",
				},
				{
					Action:       "tenant.member.remove",
					ResourceType: "tenant_member",
					Description:  "记录成员移除",
				},
				{
					Action:       "tenant.invitation.create",
					ResourceType: "tenant_invitation",
					Description:  "记录邀请创建，明文 Token 不写入审计元数据",
				},
				{
					Action:       "tenant.invitation.revoke",
					ResourceType: "tenant_invitation",
					Description:  "记录邀请撤销",
				},
				{
					Action:       "tenant.invitation.accept",
					ResourceType: "tenant_invitation",
					Description:  "记录邀请接受",
				},
			},
			QueryCapabilities: []string{
				"按 action 筛选",
				"按 resource_type 筛选",
				"按 actor_user_id 筛选",
				"按创建时间范围筛选",
				"cursor 分页",
				"CSV 导出",
			},
			ExportEnabled: true,
			MetadataFields: []string{
				"method",
				"path",
				"route",
				"status",
				"request_id",
				"role_code",
			},
		},
		Notes: []string{
			"登录和注册按 IP 限流，租户级 API 按租户和用户双维度限流。",
			"Redis 限流异常时回退内存计数，保障接口可用性；内存限流适合单实例或本地环境。",
			"审计日志按 tenant_id 隔离，管理台支持筛选、分页和 CSV 导出。",
		},
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
