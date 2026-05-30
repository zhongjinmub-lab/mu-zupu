package file

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	write := rg.Group("")
	write.Use(tenant.RequireTenantWriter())
	write.POST("/files/upload", h.Upload)
	rg.GET("/files", h.List)
	rg.GET("/files/:file_id/download", h.Download)
}
