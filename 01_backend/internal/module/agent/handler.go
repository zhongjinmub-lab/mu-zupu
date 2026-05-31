package agent

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/internal/embedding"
	"mu-agent-saas/internal/generation"
	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/billing"
	"mu-agent-saas/internal/module/kb"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/internal/module/webhook"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo   Repository
	KB     kb.Repository
	Vector kb.VectorRepository
	Embed  embedding.Provider
	Gen    generation.Provider
	Usage  billing.Repository
	Hooks  webhook.Service
}

func NewHandler(repo Repository, kbRepo kb.Repository) Handler {
	return Handler{Repo: repo, KB: kbRepo}
}

func NewHandlerWithRuntime(repo Repository, kbRepo kb.Repository, vectorRepo kb.VectorRepository, embedder embedding.Provider, generator generation.Provider, usage billing.Repository) Handler {
	return Handler{Repo: repo, KB: kbRepo, Vector: vectorRepo, Embed: embedder, Gen: generator, Usage: usage}
}

func NewHandlerWithRuntimeAndWebhook(repo Repository, kbRepo kb.Repository, vectorRepo kb.VectorRepository, embedder embedding.Provider, generator generation.Provider, usage billing.Repository, hooks webhook.Service) Handler {
	return Handler{Repo: repo, KB: kbRepo, Vector: vectorRepo, Embed: embedder, Gen: generator, Usage: usage, Hooks: hooks}
}

func (h Handler) CreateAgent(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.Create(c.Request.Context(), t.ID, u.ID, req)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ListAgents(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.List(c.Request.Context(), t.ID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) ToolSafetyPolicy(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, DefaultToolSafetyPolicy())
}

func (h Handler) ListTools(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, gin.H{"items": DefaultToolCatalog()})
}

func (h Handler) TestTool(c *gin.Context) {
	started := time.Now()
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	t, _ := tenant.CurrentTenant(c)
	item, ok := FindToolCatalogItem(c.Param("tool_id"))
	if !ok {
		response.Error(c, http.StatusNotFound, 40420, "工具不存在")
		return
	}
	var req ToolTestRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	result := BuildToolTestResult(item, req)
	_ = h.Repo.InsertToolCallLog(c.Request.Context(), t.ID, "", "", item.Code, result.Status, map[string]any{
		"input_summary": result.InputSummary,
		"dry_run":       true,
	}, map[string]any{
		"allowed":               result.Allowed,
		"requires_confirmation": result.RequiresConfirmation,
		"message":               result.Message,
	}, int(time.Since(started).Milliseconds()))
	response.OK(c, result)
}

func (h Handler) ListToolCallLogs(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	from, err := ParseToolCallLogTime(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	to, err := ParseToolCallLogTime(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	q := ToolCallLogQuery{
		TenantID:  t.ID,
		AgentID:   c.Query("agent_id"),
		ToolName:  c.Query("tool_name"),
		Status:    c.Query("status"),
		From:      from,
		To:        to,
		CursorRaw: c.Query("cursor"),
		Limit:     limit,
	}
	q.Normalize()
	if err := q.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40003, err.Error())
		return
	}
	items, nextCursor, err := h.Repo.ListToolCallLogs(c.Request.Context(), q)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items, "next_cursor": nextCursor})
}

func (h Handler) ExportToolCallLogs(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	from, err := ParseToolCallLogTime(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	to, err := ParseToolCallLogTime(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	q := ToolCallLogQuery{
		TenantID: t.ID,
		AgentID:  c.Query("agent_id"),
		ToolName: c.Query("tool_name"),
		Status:   c.Query("status"),
		From:     from,
		To:       to,
		Limit:    1000,
	}
	q.Normalize()
	q.Limit = 1000
	if err := q.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40003, err.Error())
		return
	}
	items, err := h.Repo.ExportToolCallLogs(c.Request.Context(), q)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	filename := "tool-call-logs-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "tenant_id", "agent_id", "conversation_id", "tool_name", "status", "cost_ms", "input", "output", "created_at"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.ID,
			item.TenantID,
			item.AgentID,
			item.ConversationID,
			item.ToolName,
			item.Status,
			strconv.Itoa(item.CostMS),
			toolCallLogJSON(item.Input),
			toolCallLogJSON(item.Output),
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func toolCallLogJSON(v map[string]any) string {
	if len(v) == 0 {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (h Handler) ConversationOrchestrationPolicy(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, DefaultConversationOrchestrationPolicy())
}

func (h Handler) MCPGatewayPolicy(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, DefaultMCPGatewayPolicy())
}

func (h Handler) ListMCPServers(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, gin.H{"items": DefaultMCPServerCatalog()})
}

func (h Handler) TestMCPServer(c *gin.Context) {
	started := time.Now()
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, ok := FindMCPCatalogItem(c.Param("server_id"))
	if !ok {
		response.Error(c, http.StatusNotFound, 40421, "MCP 服务不存在")
		return
	}
	var req MCPTestRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	result := BuildMCPTestResult(item, req)
	_ = h.Repo.InsertToolCallLog(c.Request.Context(), t.ID, "", "", "mcp:"+item.Code, result.Status, map[string]any{
		"input_summary": result.InputSummary,
		"transport":     result.Transport,
		"dry_run":       true,
	}, map[string]any{
		"allowed":               result.Allowed,
		"requires_confirmation": result.RequiresConfirmation,
		"message":               result.Message,
	}, int(time.Since(started).Milliseconds()))
	response.OK(c, result)
}

func (h Handler) GenealogyGraph(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	query := genealogyGraphQueryFromRequest(c)
	if err := query.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40045, err.Error())
		return
	}
	graph, err := h.Repo.GenealogyGraph(c.Request.Context(), t.ID, query)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, graph)
}

func (h Handler) ExportGenealogyGraph(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	query := genealogyGraphQueryFromRequest(c)
	if err := query.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40045, err.Error())
		return
	}
	graph, err := h.Repo.GenealogyGraph(c.Request.Context(), t.ID, query)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	filename := "agent-genealogy-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"section", "id", "name", "code", "status", "description", "parent_agent_id", "parent_name", "child_agent_id", "child_name", "relation_type", "created_at"})
	for _, node := range graph.Nodes {
		_ = writer.Write([]string{
			"node",
			node.ID,
			node.Name,
			node.Code,
			node.Status,
			node.Description,
			"",
			"",
			"",
			"",
			"",
			node.CreatedAt.Format(time.RFC3339),
		})
	}
	for _, edge := range graph.Edges {
		_ = writer.Write([]string{
			"edge",
			edge.ID,
			"",
			"",
			"",
			"",
			edge.ParentAgentID,
			edge.ParentName,
			edge.ChildAgentID,
			edge.ChildName,
			edge.RelationType,
			edge.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func genealogyGraphQueryFromRequest(c *gin.Context) GenealogyGraphQuery {
	query := GenealogyGraphQuery{
		Q:            c.Query("q"),
		RelationType: c.Query("relation_type"),
	}
	query.Normalize()
	return query
}

func (h Handler) CreateGenealogyEdge(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateGenealogyEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.CreateGenealogyEdge(c.Request.Context(), t.ID, req)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) DeleteGenealogyEdge(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.Repo.DeleteGenealogyEdge(c.Request.Context(), t.ID, c.Param("edge_id")); err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h Handler) GetAgent(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("agent_id"))
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) UpdateAgent(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.Update(c.Request.Context(), t.ID, c.Param("agent_id"), req)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) PublishAgent(c *gin.Context) {
	h.setStatus(c, "published")
}

func (h Handler) RollbackAgent(c *gin.Context) {
	h.setStatus(c, "draft")
}

func (h Handler) ArchiveAgent(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.Repo.Archive(c.Request.Context(), t.ID, c.Param("agent_id")); err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"archived": true})
}

func (h Handler) BindKnowledgeBase(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req BindKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if err := h.KB.EnsureAccess(c.Request.Context(), t.ID, req.KnowledgeBaseID); err != nil {
		if errors.Is(err, kb.ErrKnowledgeBaseNotFound) {
			response.Error(c, http.StatusNotFound, 40420, "knowledge base not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50020, err.Error())
		return
	}
	item, err := h.Repo.BindKnowledgeBase(c.Request.Context(), t.ID, c.Param("agent_id"), req)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ListKnowledgeBases(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.ListKnowledgeBases(c.Request.Context(), t.ID, c.Param("agent_id"))
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) UnbindKnowledgeBase(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.Repo.UnbindKnowledgeBase(c.Request.Context(), t.ID, c.Param("agent_id"), c.Param("kb_id")); err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"unbound": true})
}

func (h Handler) TestChat(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	if h.Embed == nil {
		response.Error(c, http.StatusInternalServerError, 50040, "embedding provider is not configured")
		return
	}
	if h.Gen == nil {
		response.Error(c, http.StatusInternalServerError, 50041, "generation provider is not configured")
		return
	}
	var req TestChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if err := h.Usage.EnsureQuota(c.Request.Context(), t.ID, billing.MetricAgentMessages, 1); err != nil {
		writeBillingError(c, err)
		return
	}
	agentItem, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("agent_id"))
	if err != nil {
		writeAgentError(c, err)
		return
	}
	binding, err := h.Repo.ResolveKnowledgeBase(c.Request.Context(), t.ID, agentItem.ID, req.KnowledgeBaseID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	if err := h.KB.EnsureAccess(c.Request.Context(), t.ID, binding.KnowledgeBaseID); err != nil {
		response.Error(c, http.StatusNotFound, 40420, "knowledge base not found")
		return
	}
	vec, err := h.Embed.Embed(c.Request.Context(), req.Message)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50042, err.Error())
		return
	}
	searchReq := kb.SearchRequest{
		TenantID:        t.ID,
		KnowledgeBaseID: binding.KnowledgeBaseID,
		AgentID:         agentItem.ID,
		Embedding:       vec,
		Query:           req.Message,
		TopK:            req.TopK,
		CandidateK:      req.CandidateK,
	}
	searchReq.SetMinScore(req.MinScore)
	rows, err := h.Vector.Search(c.Request.Context(), searchReq, true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}
	genResp, err := h.Gen.Generate(c.Request.Context(), generation.Request{
		Messages: []generation.Message{
			{Role: "system", Content: buildAgentRAGContext(agentItem, rows)},
			{Role: "user", Content: req.Message},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50043, err.Error())
		return
	}
	conversationID, err := h.Repo.CreateConversation(c.Request.Context(), t.ID, agentItem.ID, u.ID, req.Message, map[string]any{
		"knowledge_base_id": binding.KnowledgeBaseID,
		"mode":              "test_chat",
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50044, err.Error())
		return
	}
	userMessageID, err := h.Repo.CreateMessage(c.Request.Context(), t.ID, conversationID, "user", req.Message, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50045, err.Error())
		return
	}
	assistantMessageID, err := h.Repo.CreateMessage(c.Request.Context(), t.ID, conversationID, "assistant", genResp.Answer, map[string]any{
		"knowledge_base_id": binding.KnowledgeBaseID,
		"generation_model":  genResp.Model,
		"generation_source": h.Gen.Name(),
		"reference_count":   len(rows),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50045, err.Error())
		return
	}
	out := TestChatResponse{
		ConversationID:     conversationID,
		UserMessageID:      userMessageID,
		AssistantMessageID: assistantMessageID,
		AgentID:            agentItem.ID,
		KnowledgeBaseID:    binding.KnowledgeBaseID,
		Answer:             genResp.Answer,
		EmbeddingModel:     h.Embed.Model(),
		GenerationModel:    genResp.Model,
		GenerationSource:   h.Gen.Name(),
		References:         make([]TestChatReference, 0, len(rows)),
	}
	for _, row := range rows {
		out.References = append(out.References, TestChatReference(row))
	}
	_ = h.Usage.Record(c.Request.Context(), billing.RecordUsageInput{
		TenantID:    t.ID,
		SubjectType: "agent",
		SubjectID:   agentItem.ID,
		Metric:      billing.MetricAgentMessages,
		Quantity:    1,
		Unit:        "messages",
		RequestID:   c.GetString("request_id"),
		Metadata: map[string]any{
			"conversation_id":   conversationID,
			"knowledge_base_id": binding.KnowledgeBaseID,
			"mode":              "test_chat",
			"references":        len(rows),
		},
	})
	response.OK(c, out)
}

func (h Handler) Chat(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	out, err := h.executeChat(c, t.ID, u.ID, c.Param("agent_id"), req)
	if err != nil {
		writeChatExecutionError(c, err)
		return
	}
	response.OK(c, out)
}

func (h Handler) ChatStream(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	writeSSE(c, "start", gin.H{"message": "流式会话已开始"})
	out, err := h.executeChat(c, t.ID, u.ID, c.Param("agent_id"), req)
	if err != nil {
		writeSSE(c, "error", gin.H{"message": chatExecutionMessage(err)})
		return
	}
	for _, ref := range out.References {
		writeSSE(c, "reference", ref)
	}
	for _, part := range splitAnswerDeltas(out.Answer) {
		if c.Request.Context().Err() != nil {
			return
		}
		writeSSE(c, "delta", gin.H{"content": part})
	}
	writeSSE(c, "done", out)
}

func (h Handler) executeChat(c *gin.Context, tenantID, userID, agentID string, req ChatRequest) (ChatResponse, error) {
	if h.Embed == nil {
		return ChatResponse{}, errors.New("embedding provider is not configured")
	}
	if h.Gen == nil {
		return ChatResponse{}, errors.New("generation provider is not configured")
	}
	if err := h.Usage.EnsureQuota(c.Request.Context(), tenantID, billing.MetricAgentMessages, 1); err != nil {
		return ChatResponse{}, err
	}
	agentItem, err := h.Repo.Get(c.Request.Context(), tenantID, agentID)
	if err != nil {
		return ChatResponse{}, err
	}
	binding, err := h.Repo.ResolveKnowledgeBase(c.Request.Context(), tenantID, agentItem.ID, req.KnowledgeBaseID)
	if err != nil {
		return ChatResponse{}, err
	}
	if err := h.KB.EnsureAccess(c.Request.Context(), tenantID, binding.KnowledgeBaseID); err != nil {
		return ChatResponse{}, err
	}
	conversationID := req.ConversationID
	if conversationID != "" {
		if _, err := h.Repo.GetConversation(c.Request.Context(), tenantID, agentItem.ID, conversationID); err != nil {
			return ChatResponse{}, err
		}
	} else {
		title := req.Title
		if title == "" {
			title = conversationTitle(req.Message)
		}
		conversationID, err = h.Repo.CreateConversation(c.Request.Context(), tenantID, agentItem.ID, userID, title, map[string]any{
			"knowledge_base_id": binding.KnowledgeBaseID,
			"mode":              "chat",
		})
		if err != nil {
			return ChatResponse{}, err
		}
	}
	history, err := h.Repo.ListMessages(c.Request.Context(), tenantID, agentItem.ID, conversationID, req.HistoryLimit)
	if err != nil {
		return ChatResponse{}, err
	}
	vec, err := h.Embed.Embed(c.Request.Context(), req.Message)
	if err != nil {
		return ChatResponse{}, err
	}
	searchReq := kb.SearchRequest{
		TenantID:        tenantID,
		KnowledgeBaseID: binding.KnowledgeBaseID,
		AgentID:         agentItem.ID,
		Embedding:       vec,
		Query:           req.Message,
		TopK:            req.TopK,
		CandidateK:      req.CandidateK,
	}
	searchReq.SetMinScore(req.MinScore)
	rows, err := h.Vector.Search(c.Request.Context(), searchReq, true)
	if err != nil {
		return ChatResponse{}, err
	}
	messages := buildChatMessages(agentItem, rows, history, req.Message)
	genResp, err := h.Gen.Generate(c.Request.Context(), generation.Request{
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		return ChatResponse{}, err
	}
	userMessage, err := h.Repo.CreateMessageItem(c.Request.Context(), tenantID, conversationID, "user", req.Message, nil)
	if err != nil {
		return ChatResponse{}, err
	}
	assistantMessage, err := h.Repo.CreateMessageItem(c.Request.Context(), tenantID, conversationID, "assistant", genResp.Answer, map[string]any{
		"knowledge_base_id": binding.KnowledgeBaseID,
		"generation_model":  genResp.Model,
		"generation_source": h.Gen.Name(),
		"reference_count":   len(rows),
		"history_used":      len(history),
	})
	if err != nil {
		return ChatResponse{}, err
	}
	out := ChatResponse{
		ConversationID:     conversationID,
		UserMessageID:      userMessage.ID,
		AssistantMessageID: assistantMessage.ID,
		AgentID:            agentItem.ID,
		KnowledgeBaseID:    binding.KnowledgeBaseID,
		Answer:             genResp.Answer,
		HistoryUsed:        len(history),
		EmbeddingModel:     h.Embed.Model(),
		GenerationModel:    genResp.Model,
		GenerationSource:   h.Gen.Name(),
		References:         make([]TestChatReference, 0, len(rows)),
	}
	for _, row := range rows {
		out.References = append(out.References, TestChatReference(row))
	}
	_ = h.Usage.Record(c.Request.Context(), billing.RecordUsageInput{
		TenantID:    tenantID,
		SubjectType: "agent",
		SubjectID:   agentItem.ID,
		Metric:      billing.MetricAgentMessages,
		Quantity:    1,
		Unit:        "messages",
		RequestID:   c.GetString("request_id"),
		Metadata: map[string]any{
			"conversation_id":   conversationID,
			"knowledge_base_id": binding.KnowledgeBaseID,
			"mode":              "chat",
			"references":        len(rows),
			"history_used":      len(history),
		},
	})
	h.Hooks.Emit(c.Request.Context(), tenantID, webhook.EventAgentChatFinished, map[string]any{
		"agent_id":             agentItem.ID,
		"conversation_id":      conversationID,
		"user_message_id":      userMessage.ID,
		"assistant_message_id": assistantMessage.ID,
		"knowledge_base_id":    binding.KnowledgeBaseID,
		"history_used":         len(history),
		"reference_count":      len(rows),
	})
	return out, nil
}

func (h Handler) ListConversations(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListConversations(c.Request.Context(), t.ID, c.Param("agent_id"), limit)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) ListMessages(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := h.Repo.ListMessages(c.Request.Context(), t.ID, c.Param("agent_id"), c.Param("conversation_id"), limit)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) setStatus(c *gin.Context, status string) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.SetStatus(c.Request.Context(), t.ID, c.Param("agent_id"), status)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, item)
}

func buildAgentRAGContext(agentItem Agent, rows []kb.SearchResult) string {
	var b strings.Builder
	if agentItem.SystemPrompt != "" {
		b.WriteString(agentItem.SystemPrompt)
		b.WriteString("\n\n")
	}
	b.WriteString("请仅基于绑定知识库上下文回答；如上下文不足，请明确说明。\n")
	for i, row := range rows {
		b.WriteString("\n[")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("] ")
		if row.Title != "" {
			b.WriteString(row.Title)
			b.WriteString("\n")
		}
		b.WriteString(row.Content)
		b.WriteString("\n")
	}
	return b.String()
}

func buildChatMessages(agentItem Agent, rows []kb.SearchResult, history []Message, userMessage string) []generation.Message {
	messages := []generation.Message{{Role: "system", Content: buildAgentRAGContext(agentItem, rows)}}
	for _, item := range history {
		if item.Role != "user" && item.Role != "assistant" {
			continue
		}
		messages = append(messages, generation.Message{Role: item.Role, Content: item.Content})
	}
	messages = append(messages, generation.Message{Role: "user", Content: userMessage})
	return messages
}

func writeSSE(c *gin.Context, event string, data any) {
	payload, _ := json.Marshal(data)
	_, _ = c.Writer.WriteString("event: " + event + "\n")
	_, _ = c.Writer.WriteString("data: " + string(payload) + "\n\n")
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func splitAnswerDeltas(answer string) []string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return []string{""}
	}
	runes := []rune(answer)
	out := make([]string, 0, len(runes)/60+1)
	for len(runes) > 0 {
		n := 60
		if len(runes) < n {
			n = len(runes)
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}

func conversationTitle(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "New conversation"
	}
	runes := []rune(message)
	if len(runes) > 80 {
		return string(runes[:80])
	}
	return message
}

func writeAgentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAgentNotFound):
		response.Error(c, http.StatusNotFound, 40440, "agent not found")
	case errors.Is(err, ErrBindingNotFound):
		response.Error(c, http.StatusNotFound, 40441, "agent knowledge base binding not found")
	case errors.Is(err, ErrConversationNotFound):
		response.Error(c, http.StatusNotFound, 40442, "conversation not found")
	case errors.Is(err, ErrGenealogyEdgeNotFound):
		response.Error(c, http.StatusNotFound, 40443, "agent genealogy edge not found")
	case errors.Is(err, ErrGenealogyEdgeExists):
		response.Error(c, http.StatusConflict, 40941, "agent genealogy edge already exists")
	case errors.Is(err, ErrGenealogyEdgeCycle):
		response.Error(c, http.StatusBadRequest, 40044, "族谱关系会形成循环，请调整父子节点")
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40940, "agent code, binding or genealogy edge already exists")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50040, err.Error())
	}
}

func writeChatExecutionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, kb.ErrKnowledgeBaseNotFound):
		response.Error(c, http.StatusNotFound, 40420, "knowledge base not found")
	case errors.Is(err, ErrAgentNotFound), errors.Is(err, ErrBindingNotFound), errors.Is(err, ErrConversationNotFound):
		writeAgentError(c, err)
	default:
		if _, ok := billing.IsQuotaExceeded(err); ok {
			writeBillingError(c, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50043, err.Error())
	}
}

func chatExecutionMessage(err error) string {
	switch {
	case errors.Is(err, kb.ErrKnowledgeBaseNotFound):
		return "知识库不存在或不可访问"
	case errors.Is(err, ErrAgentNotFound):
		return "智能体不存在"
	case errors.Is(err, ErrBindingNotFound):
		return "智能体未绑定可用知识库"
	case errors.Is(err, ErrConversationNotFound):
		return "会话不存在"
	default:
		if _, ok := billing.IsQuotaExceeded(err); ok {
			return "用量配额不足，请检查订阅或 License"
		}
		if err != nil {
			return err.Error()
		}
		return "流式会话失败"
	}
}

func writeBillingError(c *gin.Context, err error) {
	if check, ok := billing.IsQuotaExceeded(err); ok {
		response.Error(c, http.StatusPaymentRequired, 40201, check.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, 50060, err.Error())
}
