package workflow

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

// RegisterRoutes 注册工作流编排相关路由：只读策略与节点类型对所有租户成员开放，
// 图校验为写权限操作，走 RequireTenantWriter。
func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/workflows/orchestration-policy", h.OrchestrationPolicy)
	rg.GET("/workflow-node-types", h.ListNodeTypes)

	write := rg.Group("")
	write.Use(tenant.RequireTenantWriter())
	write.POST("/workflows/validate", h.ValidateWorkflow)
}
