package workflow

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

// Handler 提供工作流编排相关的策略、图校验与定义持久化接口。
type Handler struct {
	Repo Repository
}

// NewHandler 构造工作流 Handler。
func NewHandler(repo Repository) Handler {
	return Handler{Repo: repo}
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

// ListWorkflows 返回当前租户的工作流定义列表。
func (h Handler) ListWorkflows(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.List(c.Request.Context(), t.ID)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

// GetWorkflow 返回指定工作流定义。
func (h Handler) GetWorkflow(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("workflow_id"))
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, item)
}

// CreateWorkflow 创建工作流定义，写入前对图结构做校验。
func (h Handler) CreateWorkflow(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if validation := ValidateWorkflowGraph(req.Definition); !validation.Valid {
		response.Error(c, http.StatusBadRequest, 40046, "工作流图结构校验未通过："+joinIssues(validation.Issues))
		return
	}
	item, err := h.Repo.Create(c.Request.Context(), t.ID, u.ID, req)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, item)
}

// UpdateWorkflow 更新工作流定义，若更新定义则同样做图结构校验。
func (h Handler) UpdateWorkflow(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if req.Definition != nil {
		if validation := ValidateWorkflowGraph(*req.Definition); !validation.Valid {
			response.Error(c, http.StatusBadRequest, 40046, "工作流图结构校验未通过："+joinIssues(validation.Issues))
			return
		}
	}
	item, err := h.Repo.Update(c.Request.Context(), t.ID, c.Param("workflow_id"), req)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, item)
}

// PublishWorkflow 发布工作流，发布前要求当前定义通过图结构校验。
func (h Handler) PublishWorkflow(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	workflowID := c.Param("workflow_id")
	current, err := h.Repo.Get(c.Request.Context(), t.ID, workflowID)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	if validation := ValidateWorkflowGraph(current.Definition); !validation.Valid {
		response.Error(c, http.StatusBadRequest, 40046, "工作流图结构校验未通过，无法发布："+joinIssues(validation.Issues))
		return
	}
	item, err := h.Repo.SetStatus(c.Request.Context(), t.ID, workflowID, StatusPublished)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, item)
}

// ArchiveWorkflow 归档工作流。
func (h Handler) ArchiveWorkflow(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.Repo.Archive(c.Request.Context(), t.ID, c.Param("workflow_id")); err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, gin.H{"status": "archived"})
}

// joinIssues 把校验错误列表拼成中文摘要。
func joinIssues(issues []string) string {
	if len(issues) == 0 {
		return "未知错误"
	}
	return strings.Join(issues, "；")
}

// writeWorkflowError 统一处理工作流相关错误，未知错误返回 500。
func writeWorkflowError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrWorkflowNotFound):
		response.Error(c, http.StatusNotFound, 40444, "workflow not found")
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40943, "workflow code already exists")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50045, err.Error())
	}
}

// RunWorkflow 对指定工作流做一次 dry-run 模拟执行，并写入执行日志。
func (h Handler) RunWorkflow(c *gin.Context) {
	started := time.Now()
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	workflowID := c.Param("workflow_id")
	wf, err := h.Repo.Get(c.Request.Context(), t.ID, workflowID)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	var body struct {
		Input map[string]any `json:"input"`
	}
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	result := SimulateWorkflowRun(wf.Definition)
	run, err := h.Repo.InsertWorkflowRun(c.Request.Context(), t.ID, workflowID, u.ID, result.Status, "dry_run", body.Input, result.Steps, int(time.Since(started).Milliseconds()))
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, gin.H{
		"run":                    run,
		"execution_order":        result.ExecutionOrder,
		"awaiting_approval_node": result.AwaitingApprovalNode,
		"issues":                 result.Issues,
	})
}

// ListWorkflowRuns 返回指定工作流的运行记录（执行日志）。
func (h Handler) ListWorkflowRuns(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.Repo.ListWorkflowRuns(c.Request.Context(), t.ID, c.Param("workflow_id"), limit)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

// WorkflowsSummary 返回当前租户工作流定义的概览统计（复用 List 后做纯函数聚合）。
func (h Handler) WorkflowsSummary(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.List(c.Request.Context(), t.ID)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, SummarizeWorkflows(items))
}

// DuplicateWorkflow 将已有工作流复制为一个新的草稿工作流。
func (h Handler) DuplicateWorkflow(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	src, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("workflow_id"))
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	req := DuplicateWorkflowRequest(src, time.Now().Format("150405"))
	req.Normalize()
	item, err := h.Repo.Create(c.Request.Context(), t.ID, u.ID, req)
	if err != nil {
		writeWorkflowError(c, err)
		return
	}
	response.OK(c, item)
}
