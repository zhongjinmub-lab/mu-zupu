package kb

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"mu-agent-saas/internal/embedding"
	"mu-agent-saas/internal/generation"
	"mu-agent-saas/internal/module/billing"
	filemodule "mu-agent-saas/internal/module/file"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
	"mu-agent-saas/pkg/storage"
)

type Handler struct {
	Repo    VectorRepository
	KB      Repository
	Files   filemodule.Repository
	Storage storage.Client
	Embed   embedding.Provider
	Gen     generation.Provider
	Usage   billing.Repository
}

func NewHandler(db *pgxpool.Pool) Handler {
	return Handler{Repo: NewVectorRepository(db), KB: NewRepository(db)}
}

func NewHandlerWithStorage(db *pgxpool.Pool, storageClient storage.Client, embedder embedding.Provider) Handler {
	return NewHandlerWithStorageAndGeneration(db, storageClient, embedder, nil, billing.Repository{})
}

func NewHandlerWithStorageAndGeneration(db *pgxpool.Pool, storageClient storage.Client, embedder embedding.Provider, generator generation.Provider, usage billing.Repository) Handler {
	return Handler{
		Repo:    NewVectorRepository(db),
		KB:      NewRepository(db),
		Files:   filemodule.NewRepository(db),
		Storage: storageClient,
		Embed:   embedder,
		Gen:     generator,
		Usage:   usage,
	}
}

func (h Handler) VectorSearch(c *gin.Context) {
	h.search(c, false)
}

func (h Handler) HybridSearch(c *gin.Context) {
	h.search(c, true)
}

func (h Handler) CreateKnowledgeBase(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req = req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.KB.CreateKnowledgeBase(c.Request.Context(), t.ID, req)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40920, "knowledge base code already exists")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50020, err.Error())
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
	items, err := h.KB.ListKnowledgeBases(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50020, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) CreateDocument(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	item, err := h.KB.CreateDocument(c.Request.Context(), t.ID, c.Param("kb_id"), req)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ListDocuments(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.KB.ListDocuments(c.Request.Context(), t.ID, c.Param("kb_id"), limit)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) GetDocument(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, fileID, err := h.KB.GetDocument(c.Request.Context(), t.ID, c.Param("kb_id"), c.Param("document_id"))
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, gin.H{"document": item, "file_id": fileID})
}

func (h Handler) ListDocumentChunks(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := h.KB.ListDocumentChunks(c.Request.Context(), t.ID, c.Param("kb_id"), c.Param("document_id"), limit)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) ArchiveDocument(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.KB.ArchiveDocument(c.Request.Context(), t.ID, c.Param("kb_id"), c.Param("document_id")); err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, gin.H{"archived": true})
}

func (h Handler) CreateDocumentFromFile(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateDocumentFromFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	f, err := h.Files.Get(c.Request.Context(), t.ID, req.FileID)
	if err != nil {
		if filemodule.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, 40430, "file not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50034, err.Error())
		return
	}
	if !isPlainTextFile(f.MimeType, f.Filename) {
		response.Error(c, http.StatusBadRequest, 40034, "only plain text or markdown files are supported")
		return
	}
	obj, err := h.Storage.Get(c.Request.Context(), f.ObjectKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50035, err.Error())
		return
	}
	defer obj.Close()
	data, err := io.ReadAll(io.LimitReader(obj, 10<<20))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50036, err.Error())
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = f.Filename
	}
	chunks := SplitTextChunks(string(data), req.MaxChars, req.OverlapChars)
	doc, err := h.KB.CreateDocumentFromFile(c.Request.Context(), t.ID, c.Param("kb_id"), f.ID, CreateDocumentRequest{
		Title:         title,
		SourceType:    "file",
		SourceURI:     f.ObjectKey,
		MimeType:      f.MimeType,
		ContentSHA256: f.Checksum,
		Metadata:      req.Metadata,
	}, chunks)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, CreateDocumentFromFileResponse{Document: doc, ChunkCount: len(chunks)})
}

func (h Handler) RebuildDocument(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req RebuildDocumentRequest
	_ = c.ShouldBindJSON(&req)
	req.Normalize()
	doc, fileID, err := h.KB.GetDocument(c.Request.Context(), t.ID, c.Param("kb_id"), c.Param("document_id"))
	if err != nil {
		writeKBError(c, err)
		return
	}
	if fileID == "" || doc.SourceType != "file" {
		response.Error(c, http.StatusBadRequest, 40037, "only file-backed documents can be rebuilt")
		return
	}
	f, err := h.Files.Get(c.Request.Context(), t.ID, fileID)
	if err != nil {
		if filemodule.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, 40430, "file not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50034, err.Error())
		return
	}
	if !isPlainTextFile(f.MimeType, f.Filename) {
		response.Error(c, http.StatusBadRequest, 40034, "only plain text or markdown files are supported")
		return
	}
	obj, err := h.Storage.Get(c.Request.Context(), f.ObjectKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50035, err.Error())
		return
	}
	defer obj.Close()
	data, err := io.ReadAll(io.LimitReader(obj, 10<<20))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50036, err.Error())
		return
	}
	chunks := SplitTextChunks(string(data), req.MaxChars, req.OverlapChars)
	doc, err = h.KB.RebuildDocumentChunks(c.Request.Context(), t.ID, c.Param("kb_id"), c.Param("document_id"), chunks)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, RebuildDocumentResponse{Document: doc, ChunkCount: len(chunks)})
}

func (h Handler) EnqueueDocumentJob(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req EnqueueDocumentJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if _, err := h.Files.Get(c.Request.Context(), t.ID, req.FileID); err != nil {
		if filemodule.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, 40430, "file not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50034, err.Error())
		return
	}
	if req.JobType == "rebuild" {
		doc, fileID, err := h.KB.GetDocument(c.Request.Context(), t.ID, c.Param("kb_id"), req.DocumentID)
		if err != nil {
			writeKBError(c, err)
			return
		}
		if doc.SourceType != "file" || fileID != req.FileID {
			response.Error(c, http.StatusBadRequest, 40037, "document and file do not match")
			return
		}
	}
	job, err := h.KB.CreateDocumentJob(c.Request.Context(), t.ID, c.Param("kb_id"), req)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, job)
}

func (h Handler) ListDocumentJobs(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.KB.ListDocumentJobs(c.Request.Context(), t.ID, c.Param("kb_id"), limit)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) RunDocumentJobs(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req RunDocumentJobsRequest
	_ = c.ShouldBindJSON(&req)
	if req.Limit <= 0 || req.Limit > 20 {
		req.Limit = 5
	}
	jobs, err := h.KB.ClaimDocumentJobs(c.Request.Context(), t.ID, c.Param("kb_id"), req.Limit)
	if err != nil {
		writeKBError(c, err)
		return
	}
	runner := NewDocumentJobRunner(h.KB, h.Files, h.Storage)
	response.OK(c, runner.Run(c.Request.Context(), jobs))
}

func (h Handler) CreateChunk(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateChunkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.KB.CreateChunk(c.Request.Context(), t.ID, c.Param("kb_id"), req)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ListPendingChunks(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.KB.ListPendingChunks(c.Request.Context(), t.ID, c.Param("kb_id"), limit)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) UpdateChunkEmbedding(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req UpdateChunkEmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.KB.UpdateChunkEmbedding(c.Request.Context(), t.ID, c.Param("kb_id"), c.Param("chunk_id"), req)
	if err != nil {
		writeKBError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) RunEmbedding(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if h.Embed == nil {
		response.Error(c, http.StatusInternalServerError, 50040, "embedding provider is not configured")
		return
	}
	var req RunEmbeddingRequest
	_ = c.ShouldBindJSON(&req)
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	items, err := h.KB.ListPendingChunks(c.Request.Context(), t.ID, c.Param("kb_id"), req.Limit)
	if err != nil {
		writeKBError(c, err)
		return
	}
	if len(items) > 0 {
		if err := h.Usage.EnsureQuota(c.Request.Context(), t.ID, billing.MetricEmbeddingChunks, float64(len(items))); err != nil {
			writeBillingError(c, err)
			return
		}
	}
	var out RunEmbeddingResponse
	for _, item := range items {
		vec, err := h.Embed.Embed(c.Request.Context(), item.Content)
		if err != nil {
			out.Failed++
			continue
		}
		_, err = h.KB.UpdateChunkEmbedding(c.Request.Context(), t.ID, c.Param("kb_id"), item.ID, UpdateChunkEmbeddingRequest{
			Embedding:      vec,
			EmbeddingModel: h.Embed.Model(),
			Metadata:       map[string]any{"embedding_provider": h.Embed.Name()},
		})
		if err != nil {
			out.Failed++
			continue
		}
		out.Processed++
	}
	if out.Processed > 0 {
		_ = h.Usage.Record(c.Request.Context(), billing.RecordUsageInput{
			TenantID:    t.ID,
			SubjectType: "knowledge_base",
			SubjectID:   c.Param("kb_id"),
			Metric:      billing.MetricEmbeddingChunks,
			Quantity:    float64(out.Processed),
			Unit:        "chunks",
			RequestID:   c.GetString("request_id"),
		})
	}
	response.OK(c, out)
}

func (h Handler) SearchKnowledgeBase(c *gin.Context) {
	h.search(c, true)
}

func (h Handler) AskKnowledgeBase(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if h.Embed == nil {
		response.Error(c, http.StatusInternalServerError, 50040, "embedding provider is not configured")
		return
	}
	if h.Gen == nil {
		response.Error(c, http.StatusInternalServerError, 50041, "generation provider is not configured")
		return
	}
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if err := h.Usage.EnsureQuota(c.Request.Context(), t.ID, billing.MetricRAGRequests, 1); err != nil {
		writeBillingError(c, err)
		return
	}
	kbID := c.Param("kb_id")
	if err := h.KB.EnsureAccess(c.Request.Context(), t.ID, kbID); err != nil {
		writeKBError(c, err)
		return
	}
	vec, err := h.Embed.Embed(c.Request.Context(), req.Question)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50042, err.Error())
		return
	}
	searchReq := SearchRequest{
		TenantID:        t.ID,
		KnowledgeBaseID: kbID,
		Embedding:       vec,
		Query:           req.Question,
		TopK:            req.TopK,
		CandidateK:      req.CandidateK,
	}
	searchReq.SetMinScore(req.MinScore)
	rows, err := h.Repo.Search(c.Request.Context(), searchReq, true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}
	genResp, err := h.Gen.Generate(c.Request.Context(), generation.Request{
		Messages: []generation.Message{
			{Role: "system", Content: buildRAGContext(rows)},
			{Role: "user", Content: req.Question},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50043, err.Error())
		return
	}
	out := AskResponse{
		Answer: genResp.Answer,
		Retrieval: AskRetrieval{
			TopK:       req.TopK,
			CandidateK: req.CandidateK,
			MinScore:   req.MinScore,
		},
		EmbeddingModel:   h.Embed.Model(),
		GenerationModel:  genResp.Model,
		GenerationSource: h.Gen.Name(),
	}
	out.References = make([]AskReference, 0, len(rows))
	for _, row := range rows {
		out.References = append(out.References, AskReference(row))
	}
	_ = h.Usage.Record(c.Request.Context(), billing.RecordUsageInput{
		TenantID:    t.ID,
		SubjectType: "knowledge_base",
		SubjectID:   kbID,
		Metric:      billing.MetricRAGRequests,
		Quantity:    1,
		Unit:        "requests",
		RequestID:   c.GetString("request_id"),
		Metadata: map[string]any{
			"references": len(rows),
			"top_k":      req.TopK,
		},
	})
	response.OK(c, out)
}

func (h Handler) search(c *gin.Context, hybrid bool) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, 40001, err.Error())
		return
	}
	req.TenantID = t.ID
	if kbID := c.Param("kb_id"); kbID != "" {
		req.KnowledgeBaseID = kbID
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, 400, 40002, err.Error())
		return
	}
	if err := h.KB.EnsureAccess(c.Request.Context(), req.TenantID, req.KnowledgeBaseID); err != nil {
		writeKBError(c, err)
		return
	}
	rows, err := h.Repo.Search(c.Request.Context(), req, hybrid)
	if err != nil {
		response.Error(c, 500, 50001, err.Error())
		return
	}
	response.OK(c, rows)
}

func buildRAGContext(rows []SearchResult) string {
	if len(rows) == 0 {
		return "请仅基于知识库上下文回答；当前未检索到相关上下文，如无法回答请明确说明。"
	}
	var b strings.Builder
	b.WriteString("请仅基于以下知识库上下文回答，并在需要时引用片段编号。上下文：\n")
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

func writeKBError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrKnowledgeBaseNotFound):
		response.Error(c, http.StatusNotFound, 40420, "knowledge base not found")
	case errors.Is(err, ErrDocumentNotFound):
		response.Error(c, http.StatusNotFound, 40421, "document not found")
	case errors.Is(err, ErrChunkNotFound):
		response.Error(c, http.StatusNotFound, 40422, "chunk not found")
	case errors.Is(err, ErrDocumentJobNotFound):
		response.Error(c, http.StatusNotFound, 40423, "document job not found")
	default:
		response.Error(c, http.StatusInternalServerError, 50020, err.Error())
	}
}

func writeBillingError(c *gin.Context, err error) {
	if check, ok := billing.IsQuotaExceeded(err); ok {
		response.Error(c, http.StatusPaymentRequired, 40201, check.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, 50060, err.Error())
}

func isPlainTextFile(mimeType, filename string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	filename = strings.ToLower(filename)
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	return strings.HasSuffix(filename, ".txt") || strings.HasSuffix(filename, ".md") || strings.HasSuffix(filename, ".markdown")
}
