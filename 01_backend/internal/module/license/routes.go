package license

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/licenses", h.List)
	rg.POST("/licenses/:license_id/verify", h.Verify)

	admin := rg.Group("")
	admin.Use(tenant.RequireTenantAdmin())
	admin.POST("/licenses", h.Create)
	admin.POST("/licenses/:license_id/activate", h.Activate)
	admin.POST("/licenses/:license_id/revoke", h.Revoke)
}
