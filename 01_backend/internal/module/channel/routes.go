package channel

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

// RegisterRoutes 注册渠道接入相关路由：类型目录与列表/详情对所有租户成员开放，
// 创建、启用、禁用、归档为写权限操作，走 RequireTenantWriter。
func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/channel-types", h.ListChannelTypes)
	rg.GET("/channels", h.ListChannels)
	rg.GET("/channels/summary", h.ChannelsSummary)
	rg.GET("/channels/:channel_id", h.GetChannel)
	rg.GET("/channels/:channel_id/embed", h.ChannelEmbed)

	write := rg.Group("")
	write.Use(tenant.RequireTenantWriter())
	write.POST("/channels", h.CreateChannel)
	write.PUT("/channels/:channel_id", h.UpdateChannel)
	write.POST("/channels/:channel_id/enable", h.EnableChannel)
	write.POST("/channels/:channel_id/disable", h.DisableChannel)
	write.DELETE("/channels/:channel_id", h.ArchiveChannel)
}

// RegisterPublicRoutes 注册面向外部接入方的公开渠道端点（不经登录与租户中间件，仅做 IP 限流）。
func RegisterPublicRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/channel-connect/:channel_key", h.ConnectChannel)
}
