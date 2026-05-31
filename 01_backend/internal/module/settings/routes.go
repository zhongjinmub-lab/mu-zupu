package settings

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/settings/rate-limit", h.RateLimitPolicy)
	rg.GET("/settings/runtime", h.RuntimeSummary)
	rg.GET("/settings/monitoring", h.MonitoringSnapshot)
}
