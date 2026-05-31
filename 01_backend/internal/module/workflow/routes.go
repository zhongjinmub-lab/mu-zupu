package workflow

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

// RegisterRoutes 注册工作流编排相关路由：
// 只读策略、节点类型与列表/详情对所有租户成员开放；
// 校验、创建、更新、发布、归档为写权限操作，走 RequireTenantWriter。
func RegisterRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/workflows", h.ListWorkflows)
	rg.GET("/workflows/orchestration-policy", h.OrchestrationPolicy)
	rg.GET("/workflow-node-types", h.ListNodeTypes)
	rg.GET("/workflows/:workflow_id", h.GetWorkflow)

	write := rg.Group("")
	write.Use(tenant.RequireTenantWriter())
	write.POST("/workflows/validate", h.ValidateWorkflow)
	write.POST("/workflows", h.CreateWorkflow)
	write.PUT("/workflows/:workflow_id", h.UpdateWorkflow)
	write.POST("/workflows/:workflow_id/publish", h.PublishWorkflow)
	write.DELETE("/workflows/:workflow_id", h.ArchiveWorkflow)
}
