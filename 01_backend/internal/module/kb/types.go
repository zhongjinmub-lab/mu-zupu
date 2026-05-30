package kb

import (
	"errors"
	"strings"
	"time"
)

const DefaultEmbeddingDim = 1536

type SearchRequest struct {
	TenantID        string    `json:"tenant_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	AgentID         string    `json:"agent_id"`
	Embedding       []float32 `json:"embedding" binding:"required"`
	Query           string    `json:"query"`
	TopK            int       `json:"top_k"`
	CandidateK      int       `json:"candidate_k"`
	MinScore        float64   `json:"min_score"`
	minScoreSet     bool
}

type SearchResult struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
}

type KnowledgeBase struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	Name              string    `json:"name"`
	Code              string    `json:"code"`
	EmbeddingProvider string    `json:"embedding_provider"`
	EmbeddingModel    string    `json:"embedding_model"`
	EmbeddingDim      int       `json:"embedding_dim"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Document struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Title           string    `json:"title"`
	SourceType      string    `json:"source_type"`
	SourceURI       string    `json:"source_uri,omitempty"`
	MimeType        string    `json:"mime_type,omitempty"`
	ContentSHA256   string    `json:"content_sha256,omitempty"`
	ParseStatus     string    `json:"parse_status"`
	ChunkStatus     string    `json:"chunk_status"`
	EmbeddingStatus string    `json:"embedding_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Chunk struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	DocumentID      string    `json:"document_id"`
	ChunkNo         int       `json:"chunk_no"`
	Content         string    `json:"content"`
	ContentTokens   int       `json:"content_tokens"`
	ContentSHA256   string    `json:"content_sha256,omitempty"`
	EmbeddingModel  string    `json:"embedding_model,omitempty"`
	EmbeddingStatus string    `json:"embedding_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PendingChunk struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	KnowledgeBaseID string `json:"knowledge_base_id"`
	DocumentID      string `json:"document_id"`
	ChunkNo         int    `json:"chunk_no"`
	Content         string `json:"content"`
	ContentTokens   int    `json:"content_tokens"`
	EmbeddingStatus string `json:"embedding_status"`
}

type CreateKnowledgeBaseRequest struct {
	Name              string         `json:"name" binding:"required"`
	Code              string         `json:"code" binding:"required"`
	EmbeddingProvider string         `json:"embedding_provider"`
	EmbeddingModel    string         `json:"embedding_model"`
	EmbeddingDim      int            `json:"embedding_dim"`
	ChunkConfig       map[string]any `json:"chunk_config"`
	RetrievalConfig   map[string]any `json:"retrieval_config"`
}

type CreateDocumentRequest struct {
	Title         string         `json:"title" binding:"required"`
	SourceType    string         `json:"source_type"`
	SourceURI     string         `json:"source_uri"`
	MimeType      string         `json:"mime_type"`
	ContentSHA256 string         `json:"content_sha256"`
	Metadata      map[string]any `json:"metadata"`
}

type CreateChunkRequest struct {
	DocumentID     string         `json:"document_id" binding:"required"`
	ChunkNo        int            `json:"chunk_no"`
	Content        string         `json:"content" binding:"required"`
	ContentTokens  int            `json:"content_tokens"`
	ContentSHA256  string         `json:"content_sha256"`
	Embedding      []float32      `json:"embedding" binding:"required"`
	EmbeddingModel string         `json:"embedding_model"`
	Metadata       map[string]any `json:"metadata"`
}

type CreateDocumentFromFileRequest struct {
	FileID       string         `json:"file_id" binding:"required"`
	Title        string         `json:"title"`
	MaxChars     int            `json:"max_chars"`
	OverlapChars int            `json:"overlap_chars"`
	Metadata     map[string]any `json:"metadata"`
}

type CreateDocumentFromFileResponse struct {
	Document   Document `json:"document"`
	ChunkCount int      `json:"chunk_count"`
}

type RebuildDocumentRequest struct {
	MaxChars     int `json:"max_chars"`
	OverlapChars int `json:"overlap_chars"`
}

type RebuildDocumentResponse struct {
	Document   Document `json:"document"`
	ChunkCount int      `json:"chunk_count"`
}

type DocumentJob struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	KnowledgeBaseID string     `json:"knowledge_base_id"`
	DocumentID      string     `json:"document_id,omitempty"`
	FileID          string     `json:"file_id"`
	JobType         string     `json:"job_type"`
	Status          string     `json:"status"`
	MaxChars        int        `json:"max_chars"`
	OverlapChars    int        `json:"overlap_chars"`
	Attempts        int        `json:"attempts"`
	LastError       string     `json:"last_error,omitempty"`
	Title           string     `json:"title,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type EnqueueDocumentJobRequest struct {
	FileID       string `json:"file_id" binding:"required"`
	DocumentID   string `json:"document_id"`
	JobType      string `json:"job_type"`
	Title        string `json:"title"`
	MaxChars     int    `json:"max_chars"`
	OverlapChars int    `json:"overlap_chars"`
}

type RunDocumentJobsRequest struct {
	Limit int `json:"limit"`
}

type RunDocumentJobsResponse struct {
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

type UpdateChunkEmbeddingRequest struct {
	Embedding      []float32      `json:"embedding" binding:"required"`
	EmbeddingModel string         `json:"embedding_model"`
	Metadata       map[string]any `json:"metadata"`
}

type RunEmbeddingRequest struct {
	Limit int `json:"limit"`
}

type RunEmbeddingResponse struct {
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

type AskRequest struct {
	Question    string  `json:"question" binding:"required"`
	TopK        int     `json:"top_k"`
	CandidateK  int     `json:"candidate_k"`
	MinScore    float64 `json:"min_score"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type AskReference struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
}

type AskResponse struct {
	Answer           string         `json:"answer"`
	References       []AskReference `json:"references"`
	Retrieval        AskRetrieval   `json:"retrieval"`
	EmbeddingModel   string         `json:"embedding_model"`
	GenerationModel  string         `json:"generation_model"`
	GenerationSource string         `json:"generation_source"`
}

type AskRetrieval struct {
	TopK       int     `json:"top_k"`
	CandidateK int     `json:"candidate_k"`
	MinScore   float64 `json:"min_score"`
}

type VectorProfile struct {
	HNSWEFSearch int
	VectorWeight float64
	TextWeight   float64
	TopK         int
	CandidateK   int
	MinScore     float64
}

func (r SearchRequest) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(r.KnowledgeBaseID) == "" {
		return errors.New("knowledge_base_id is required")
	}
	if len(r.Embedding) != DefaultEmbeddingDim {
		return errors.New("embedding dimension must be 1536")
	}
	return nil
}

func (r CreateKnowledgeBaseRequest) Normalize() CreateKnowledgeBaseRequest {
	r.Code = strings.ToLower(strings.TrimSpace(r.Code))
	r.Name = strings.TrimSpace(r.Name)
	if r.EmbeddingProvider == "" {
		r.EmbeddingProvider = "openai_compatible"
	}
	if r.EmbeddingModel == "" {
		r.EmbeddingModel = "text-embedding-3-small"
	}
	if r.EmbeddingDim == 0 {
		r.EmbeddingDim = DefaultEmbeddingDim
	}
	return r
}

func (r CreateKnowledgeBaseRequest) Validate() error {
	if r.EmbeddingDim != DefaultEmbeddingDim {
		return errors.New("embedding_dim must be 1536")
	}
	return nil
}

func (r CreateChunkRequest) Validate() error {
	if len(r.Embedding) != DefaultEmbeddingDim {
		return errors.New("embedding dimension must be 1536")
	}
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	if r.ContentTokens < 0 {
		return errors.New("content_tokens must be non-negative")
	}
	return nil
}

func (r UpdateChunkEmbeddingRequest) Validate() error {
	if len(r.Embedding) != DefaultEmbeddingDim {
		return errors.New("embedding dimension must be 1536")
	}
	return nil
}

func (r *CreateDocumentFromFileRequest) Normalize() {
	normalizeChunkParams(&r.MaxChars, &r.OverlapChars)
}

func (r *RebuildDocumentRequest) Normalize() {
	normalizeChunkParams(&r.MaxChars, &r.OverlapChars)
}

func (r *EnqueueDocumentJobRequest) Normalize() {
	r.FileID = strings.TrimSpace(r.FileID)
	r.DocumentID = strings.TrimSpace(r.DocumentID)
	r.JobType = strings.TrimSpace(r.JobType)
	if r.JobType == "" {
		r.JobType = "parse_chunk"
	}
	r.Title = strings.TrimSpace(r.Title)
	normalizeChunkParams(&r.MaxChars, &r.OverlapChars)
}

func (r EnqueueDocumentJobRequest) Validate() error {
	if r.FileID == "" {
		return errors.New("file_id is required")
	}
	if r.JobType != "parse_chunk" && r.JobType != "rebuild" {
		return errors.New("job_type must be parse_chunk or rebuild")
	}
	if r.JobType == "rebuild" && r.DocumentID == "" {
		return errors.New("document_id is required for rebuild job")
	}
	return nil
}

func normalizeChunkParams(maxChars, overlapChars *int) {
	if *maxChars <= 0 || *maxChars > 8000 {
		*maxChars = 1200
	}
	if *overlapChars < 0 {
		*overlapChars = 0
	}
	if *overlapChars >= *maxChars {
		*overlapChars = *maxChars / 10
	}
}

func (r *SearchRequest) Normalize() {
	if r.TopK <= 0 || r.TopK > 50 {
		r.TopK = 10
	}
	if r.CandidateK < r.TopK || r.CandidateK > 200 {
		r.CandidateK = r.TopK * 5
	}
	if r.CandidateK > 200 {
		r.CandidateK = 200
	}
	if r.MinScore < 0 {
		r.MinScore = 0
	}
}

func (r *SearchRequest) SetMinScore(score float64) {
	r.MinScore = score
	r.minScoreSet = true
}

func (r SearchRequest) MinScoreSet() bool {
	return r.minScoreSet
}

func (r *AskRequest) Normalize() {
	r.Question = strings.TrimSpace(r.Question)
	if r.TopK <= 0 || r.TopK > 20 {
		r.TopK = 5
	}
	if r.CandidateK < r.TopK || r.CandidateK > 100 {
		r.CandidateK = r.TopK * 5
	}
	if r.CandidateK > 100 {
		r.CandidateK = 100
	}
	if r.MinScore < 0 {
		r.MinScore = 0
	}
	if r.MaxTokens <= 0 || r.MaxTokens > 4096 {
		r.MaxTokens = 1024
	}
	if r.Temperature < 0 || r.Temperature > 2 {
		r.Temperature = 0.2
	}
}

func (r AskRequest) Validate() error {
	if r.Question == "" {
		return errors.New("question is required")
	}
	return nil
}
