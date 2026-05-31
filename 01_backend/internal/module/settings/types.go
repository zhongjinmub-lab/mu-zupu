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
