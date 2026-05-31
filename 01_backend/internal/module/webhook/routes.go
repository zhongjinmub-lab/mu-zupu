package webhook

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/webhooks", h.ListEndpoints)
	rg.GET("/webhook-deliveries", h.ListDeliveries)

	admin := rg.Group("")
	admin.Use(tenant.RequireTenantAdmin())
	admin.POST("/webhooks", h.CreateEndpoint)
	admin.PUT("/webhooks/:webhook_id", h.UpdateEndpoint)
	admin.DELETE("/webhooks/:webhook_id", h.DeleteEndpoint)
	admin.POST("/webhooks/:webhook_id/test", h.TestEndpoint)
	admin.POST("/webhook-deliveries/:delivery_id/retry", h.RetryDelivery)
}
