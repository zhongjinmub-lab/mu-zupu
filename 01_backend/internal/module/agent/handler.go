package agent

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/internal/embedding"
	"mu-agent-saas/internal/generation"
	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/billing"
	"mu-agent-saas/internal/module/kb"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo   Repository
	KB     kb.Repository
	Vector kb.VectorRepository
	Embed  embedding.Provider
	Gen    generation.Provider
	Usage  billing.Repository
}

func NewHandler(repo Repository, kbRepo kb.Repository) Handler {
	return Handler{Repo: repo, KB: kbRepo}
}

func NewHandlerWithRuntime(repo Repository, kbRepo kb.Repository, vectorRepo kb.VectorRepository, embedder embedding.Provider, generator generation.Provider, usage billing.Repository) Handler {
	return Handler{Repo: repo, KB: kbRepo, Vector: vectorRepo, Embed: embedder, Gen: generator, Usage: usage}
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
	if h.Embed == nil {
		response.Error(c, http.StatusInternalServerError, 50040, "embedding provider is not configured")
		return
	}
	if h.Gen == nil {
		response.Error(c, http.StatusInternalServerError, 50041, "generation provider is not configured")
		return
	}
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
	conversationID := req.ConversationID
	if conversationID != "" {
		if _, err := h.Repo.GetConversation(c.Request.Context(), t.ID, agentItem.ID, conversationID); err != nil {
			writeAgentError(c, err)
			return
		}
	} else {
		title := req.Title
		if title == "" {
			title = conversationTitle(req.Message)
		}
		conversationID, err = h.Repo.CreateConversation(c.Request.Context(), t.ID, agentItem.ID, u.ID, title, map[string]any{
			"knowledge_base_id": binding.KnowledgeBaseID,
			"mode":              "chat",
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 50044, err.Error())
			return
		}
	}
	history, err := h.Repo.ListMessages(c.Request.Context(), t.ID, agentItem.ID, conversationID, req.HistoryLimit)
	if err != nil {
		writeAgentError(c, err)
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
	messages := buildChatMessages(agentItem, rows, history, req.Message)
	genResp, err := h.Gen.Generate(c.Request.Context(), generation.Request{
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50043, err.Error())
		return
	}
	userMessage, err := h.Repo.CreateMessageItem(c.Request.Context(), t.ID, conversationID, "user", req.Message, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50045, err.Error())
		return
	}
	assistantMessage, err := h.Repo.CreateMessageItem(c.Request.Context(), t.ID, conversationID, "assistant", genResp.Answer, map[string]any{
		"knowledge_base_id": binding.KnowledgeBaseID,
		"generation_model":  genResp.Model,
		"generation_source": h.Gen.Name(),
		"reference_count":   len(rows),
		"history_used":      len(history),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50045, err.Error())
		return
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
			"mode":              "chat",
			"references":        len(rows),
			"history_used":      len(history),
		},
	})
	response.OK(c, out)
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
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40940, "agent code or binding already exists")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50040, err.Error())
	}
}

func writeBillingError(c *gin.Context, err error) {
	if check, ok := billing.IsQuotaExceeded(err); ok {
		response.Error(c, http.StatusPaymentRequired, 40201, check.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, 50060, err.Error())
}
