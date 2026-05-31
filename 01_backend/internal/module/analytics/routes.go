package analytics

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/analytics/summary", h.Summary)
	rg.GET("/analytics/summary/export", h.ExportSummary)
}
