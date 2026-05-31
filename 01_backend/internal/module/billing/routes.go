package billing

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/billing/plans", h.ListPlans)
	rg.GET("/billing/subscription", h.CurrentSubscription)
	rg.GET("/billing/usage/summary", h.UsageSummary)
	rg.GET("/billing/quota/status", h.QuotaStatus)
	rg.GET("/orders", h.ListOrders)
	rg.GET("/payment-orders", h.ListPaymentOrders)
	rg.POST("/payments/:payment_id/query", h.QueryPayment)
	rg.GET("/payment-callback-events", h.ListPaymentCallbackEvents)

	admin := rg.Group("")
	admin.Use(tenant.RequireTenantAdmin())
	admin.POST("/orders", h.CreateOrder)
	admin.POST("/orders/:order_id/cancel", h.CancelOrder)
	admin.POST("/orders/:order_id/close", h.CloseOrder)
	admin.POST("/payment-orders", h.CreatePaymentOrder)
	admin.POST("/payments/:payment_id/close", h.ClosePayment)
	admin.POST("/payment-callbacks/:channel", h.PaymentCallback)
}

// RegisterPublicRoutes 注册无需鉴权的第三方支付异步通知端点。
// 该端点由渠道 Provider 做原生验签,并通过 pay_no 反查租户,因此不在租户作用域内。
func RegisterPublicRoutes(rg *gin.RouterGroup, h Handler) {
	rg.POST("/payment-notify/:channel", h.PaymentNotify)
}
