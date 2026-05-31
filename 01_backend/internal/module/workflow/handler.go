package workflow

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

// Handler 提供工作流编排相关的只读策略与图校验接口，当前不依赖数据库。
type Handler struct{}

// NewHandler 构造工作流 Handler。
func NewHandler() Handler {
	return Handler{}
}

// OrchestrationPolicy 返回工作流编排安全默认策略。
func (h Handler) OrchestrationPolicy(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, DefaultWorkflowOrchestrationPolicy())
}

// ListNodeTypes 返回内置工作流节点类型目录。
func (h Handler) ListNodeTypes(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, gin.H{"items": DefaultWorkflowNodeTypes()})
}

// ValidateWorkflow 对提交的工作流图做结构校验，返回中文诊断与执行顺序，不执行真实动作。
func (h Handler) ValidateWorkflow(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req ValidateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	response.OK(c, ValidateWorkflowGraph(req.Definition))
}
