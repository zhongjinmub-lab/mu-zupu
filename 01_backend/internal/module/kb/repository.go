package kb

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) EnsureAccess(ctx context.Context, tenantID, kbID string) error {
	const q = `
SELECT 1
FROM knowledge_bases
WHERE id = $1
  AND tenant_id = $2
  AND status = 'active'
  AND deleted_at IS NULL`
	var ok int
	err := r.DB.QueryRow(ctx, q, kbID, tenantID).Scan(&ok)
	if err == pgx.ErrNoRows {
		return ErrKnowledgeBaseNotFound
	}
	return err
}

func (r Repository) CreateKnowledgeBase(ctx context.Context, tenantID string, req CreateKnowledgeBaseRequest) (KnowledgeBase, error) {
	chunkConfig, err := jsonObject(req.ChunkConfig)
	if err != nil {
		return KnowledgeBase{}, err
	}
	retrievalConfig, err := jsonObject(req.RetrievalConfig)
	if err != nil {
		return KnowledgeBase{}, err
	}
	const q = `
INSERT INTO knowledge_bases(
    tenant_id, name, code, embedding_provider, embedding_model, embedding_dim, chunk_config, retrieval_config
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)
RETURNING id::text, tenant_id::text, name, code, embedding_provider, embedding_model, embedding_dim, status, created_at, updated_at`
	var item KnowledgeBase
	err = r.DB.QueryRow(ctx, q,
		tenantID,
		req.Name,
		req.Code,
		req.EmbeddingProvider,
		req.EmbeddingModel,
		req.EmbeddingDim,
		chunkConfig,
		retrievalConfig,
	).Scan(&item.ID, &item.TenantID, &item.Name, &item.Code, &item.EmbeddingProvider, &item.EmbeddingModel, &item.EmbeddingDim, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r Repository) ListKnowledgeBases(ctx context.Context, tenantID string) ([]KnowledgeBase, error) {
	const q = `
SELECT id::text, tenant_id::text, name, code, embedding_provider, embedding_model, embedding_dim, status, created_at, updated_at
FROM knowledge_bases
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBase, 0)
	for rows.Next() {
		var item KnowledgeBase
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Name, &item.Code, &item.EmbeddingProvider, &item.EmbeddingModel, &item.EmbeddingDim, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) CreateDocument(ctx context.Context, tenantID, kbID string, req CreateDocumentRequest) (Document, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return Document{}, err
	}
	metadata, err := jsonObject(req.Metadata)
	if err != nil {
		return Document{}, err
	}
	if req.SourceType == "" {
		req.SourceType = "manual"
	}
	const q = `
INSERT INTO documents(
    tenant_id, knowledge_base_id, title, source_type, source_uri, mime_type, content_sha256, metadata
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8::jsonb)
RETURNING id::text, tenant_id::text, knowledge_base_id::text, title, source_type, COALESCE(source_uri, ''),
          COALESCE(mime_type, ''), COALESCE(content_sha256, ''), parse_status, chunk_status, embedding_status, created_at, updated_at`
	var item Document
	err = r.DB.QueryRow(ctx, q, tenantID, kbID, req.Title, req.SourceType, req.SourceURI, req.MimeType, req.ContentSHA256, metadata).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.Title, &item.SourceType, &item.SourceURI,
		&item.MimeType, &item.ContentSHA256, &item.ParseStatus, &item.ChunkStatus, &item.EmbeddingStatus, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (r Repository) CreateDocumentFromFile(ctx context.Context, tenantID, kbID, fileID string, req CreateDocumentRequest, chunks []string) (Document, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return Document{}, err
	}
	metadata, err := jsonObject(req.Metadata)
	if err != nil {
		return Document{}, err
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	const createDoc = `
INSERT INTO documents(
    tenant_id, knowledge_base_id, file_id, title, source_type, source_uri, mime_type, content_sha256, metadata,
    parse_status, chunk_status, embedding_status
) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), $9::jsonb,
          'success', 'success', 'pending')
RETURNING id::text, tenant_id::text, knowledge_base_id::text, title, source_type, COALESCE(source_uri, ''),
          COALESCE(mime_type, ''), COALESCE(content_sha256, ''), parse_status, chunk_status, embedding_status, created_at, updated_at`
	var item Document
	if err := tx.QueryRow(ctx, createDoc, tenantID, kbID, fileID, req.Title, req.SourceType, req.SourceURI, req.MimeType, req.ContentSHA256, metadata).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.Title, &item.SourceType, &item.SourceURI,
		&item.MimeType, &item.ContentSHA256, &item.ParseStatus, &item.ChunkStatus, &item.EmbeddingStatus, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return Document{}, err
	}

	const insertChunk = `
INSERT INTO document_chunks(
    tenant_id, knowledge_base_id, document_id, chunk_no, content, content_tokens, metadata, embedding_status
) VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb, 'pending')`
	for i, chunk := range chunks {
		if _, err := tx.Exec(ctx, insertChunk, tenantID, kbID, item.ID, i+1, chunk, len([]rune(chunk))); err != nil {
			return Document{}, err
		}
	}
	if len(chunks) == 0 {
		if _, err := tx.Exec(ctx, `UPDATE documents SET chunk_status = 'skipped', embedding_status = 'skipped', updated_at = now() WHERE id = $1`, item.ID); err != nil {
			return Document{}, err
		}
		item.ChunkStatus = "skipped"
		item.EmbeddingStatus = "skipped"
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return item, nil
}

func (r Repository) GetDocument(ctx context.Context, tenantID, kbID, documentID string) (Document, string, error) {
	const q = `
SELECT id::text, tenant_id::text, knowledge_base_id::text, title, source_type, COALESCE(source_uri, ''),
       COALESCE(mime_type, ''), COALESCE(content_sha256, ''), parse_status, chunk_status, embedding_status,
       COALESCE(file_id::text, ''), created_at, updated_at
FROM documents
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL`
	var item Document
	var fileID string
	err := r.DB.QueryRow(ctx, q, documentID, tenantID, kbID).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.Title, &item.SourceType, &item.SourceURI,
		&item.MimeType, &item.ContentSHA256, &item.ParseStatus, &item.ChunkStatus, &item.EmbeddingStatus,
		&fileID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return Document{}, "", ErrDocumentNotFound
	}
	return item, fileID, err
}

func (r Repository) ListDocuments(ctx context.Context, tenantID, kbID string, limit int) ([]Document, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, knowledge_base_id::text, title, source_type, COALESCE(source_uri, ''),
       COALESCE(mime_type, ''), COALESCE(content_sha256, ''), parse_status, chunk_status, embedding_status,
       created_at, updated_at
FROM documents
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3`
	rows, err := r.DB.Query(ctx, q, tenantID, kbID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Document, 0)
	for rows.Next() {
		var item Document
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.Title, &item.SourceType, &item.SourceURI,
			&item.MimeType, &item.ContentSHA256, &item.ParseStatus, &item.ChunkStatus, &item.EmbeddingStatus,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) ListDocumentChunks(ctx context.Context, tenantID, kbID, documentID string, limit int) ([]Chunk, error) {
	if err := r.EnsureDocumentAccess(ctx, tenantID, kbID, documentID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `
SELECT id::text, tenant_id::text, knowledge_base_id::text, document_id::text, chunk_no, content, content_tokens,
       COALESCE(content_sha256, ''), COALESCE(embedding_model, ''), embedding_status, created_at, updated_at
FROM document_chunks
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND document_id = $3
  AND deleted_at IS NULL
ORDER BY chunk_no ASC, created_at ASC
LIMIT $4`
	rows, err := r.DB.Query(ctx, q, tenantID, kbID, documentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Chunk, 0)
	for rows.Next() {
		var item Chunk
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.ChunkNo, &item.Content, &item.ContentTokens,
			&item.ContentSHA256, &item.EmbeddingModel, &item.EmbeddingStatus, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) ArchiveDocument(ctx context.Context, tenantID, kbID, documentID string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const archiveDoc = `
UPDATE documents
SET deleted_at = now(), updated_at = now()
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL`
	tag, err := tx.Exec(ctx, archiveDoc, documentID, tenantID, kbID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}

	const archiveChunks = `
UPDATE document_chunks
SET deleted_at = now(), updated_at = now()
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND document_id = $3
  AND deleted_at IS NULL`
	if _, err := tx.Exec(ctx, archiveChunks, tenantID, kbID, documentID); err != nil {
		return err
	}

	const archiveJobs = `
UPDATE document_jobs
SET deleted_at = now(), updated_at = now()
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND document_id = $3
  AND deleted_at IS NULL`
	if _, err := tx.Exec(ctx, archiveJobs, tenantID, kbID, documentID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r Repository) RebuildDocumentChunks(ctx context.Context, tenantID, kbID, documentID string, chunks []string) (Document, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	const lockDoc = `
SELECT id::text, tenant_id::text, knowledge_base_id::text, title, source_type, COALESCE(source_uri, ''),
       COALESCE(mime_type, ''), COALESCE(content_sha256, ''), parse_status, chunk_status, embedding_status,
       created_at, updated_at
FROM documents
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL
FOR UPDATE`
	var item Document
	if err := tx.QueryRow(ctx, lockDoc, documentID, tenantID, kbID).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.Title, &item.SourceType, &item.SourceURI,
		&item.MimeType, &item.ContentSHA256, &item.ParseStatus, &item.ChunkStatus, &item.EmbeddingStatus,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return Document{}, ErrDocumentNotFound
		}
		return Document{}, err
	}

	const softDeleteChunks = `
UPDATE document_chunks
SET deleted_at = now(), updated_at = now()
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND document_id = $3
  AND deleted_at IS NULL`
	if _, err := tx.Exec(ctx, softDeleteChunks, tenantID, kbID, documentID); err != nil {
		return Document{}, err
	}

	const insertChunk = `
INSERT INTO document_chunks(
    tenant_id, knowledge_base_id, document_id, chunk_no, content, content_tokens, metadata, embedding_status
) VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb, 'pending')`
	for i, chunk := range chunks {
		if _, err := tx.Exec(ctx, insertChunk, tenantID, kbID, documentID, i+1, chunk, len([]rune(chunk))); err != nil {
			return Document{}, err
		}
	}

	chunkStatus := "success"
	embeddingStatus := "pending"
	if len(chunks) == 0 {
		chunkStatus = "skipped"
		embeddingStatus = "skipped"
	}
	const updateDoc = `
UPDATE documents
SET parse_status = 'success',
    chunk_status = $4,
    embedding_status = $5,
    updated_at = now()
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
RETURNING id::text, tenant_id::text, knowledge_base_id::text, title, source_type, COALESCE(source_uri, ''),
          COALESCE(mime_type, ''), COALESCE(content_sha256, ''), parse_status, chunk_status, embedding_status, created_at, updated_at`
	if err := tx.QueryRow(ctx, updateDoc, documentID, tenantID, kbID, chunkStatus, embeddingStatus).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.Title, &item.SourceType, &item.SourceURI,
		&item.MimeType, &item.ContentSHA256, &item.ParseStatus, &item.ChunkStatus, &item.EmbeddingStatus,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return item, nil
}

func (r Repository) CreateDocumentJob(ctx context.Context, tenantID, kbID string, req EnqueueDocumentJobRequest) (DocumentJob, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return DocumentJob{}, err
	}
	metadata, err := jsonObject(map[string]any{"title": req.Title})
	if err != nil {
		return DocumentJob{}, err
	}
	const q = `
INSERT INTO document_jobs(
    tenant_id, knowledge_base_id, document_id, file_id, job_type, max_chars, overlap_chars, metadata
) VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8::jsonb)
RETURNING id::text, tenant_id::text, knowledge_base_id::text, COALESCE(document_id::text, ''), file_id::text,
          job_type, status, max_chars, overlap_chars, attempts, COALESCE(last_error, ''),
          COALESCE(metadata->>'title', ''),
          created_at, updated_at, started_at, finished_at`
	var item DocumentJob
	err = r.DB.QueryRow(ctx, q, tenantID, kbID, req.DocumentID, req.FileID, req.JobType, req.MaxChars, req.OverlapChars, metadata).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.FileID,
		&item.JobType, &item.Status, &item.MaxChars, &item.OverlapChars, &item.Attempts, &item.LastError,
		&item.Title, &item.CreatedAt, &item.UpdatedAt, &item.StartedAt, &item.FinishedAt,
	)
	return item, err
}

func (r Repository) ListDocumentJobs(ctx context.Context, tenantID, kbID string, limit int) ([]DocumentJob, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, knowledge_base_id::text, COALESCE(document_id::text, ''), file_id::text,
       job_type, status, max_chars, overlap_chars, attempts, COALESCE(last_error, ''),
       COALESCE(metadata->>'title', ''),
       created_at, updated_at, started_at, finished_at
FROM document_jobs
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3`
	rows, err := r.DB.Query(ctx, q, tenantID, kbID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DocumentJob, 0)
	for rows.Next() {
		var item DocumentJob
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.FileID,
			&item.JobType, &item.Status, &item.MaxChars, &item.OverlapChars, &item.Attempts, &item.LastError,
			&item.Title, &item.CreatedAt, &item.UpdatedAt, &item.StartedAt, &item.FinishedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) ClaimDocumentJobs(ctx context.Context, tenantID, kbID string, limit int) ([]DocumentJob, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	const q = `
WITH picked AS (
    SELECT id
    FROM document_jobs
    WHERE tenant_id = $1
      AND knowledge_base_id = $2
      AND deleted_at IS NULL
      AND status IN ('pending', 'failed')
    ORDER BY created_at ASC
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
UPDATE document_jobs j
SET status = 'processing',
    attempts = attempts + 1,
    started_at = now(),
    updated_at = now(),
    last_error = NULL
FROM picked
WHERE j.id = picked.id
RETURNING j.id::text, j.tenant_id::text, j.knowledge_base_id::text, COALESCE(j.document_id::text, ''), j.file_id::text,
          j.job_type, j.status, j.max_chars, j.overlap_chars, j.attempts, COALESCE(j.last_error, ''),
          COALESCE(j.metadata->>'title', ''),
          j.created_at, j.updated_at, j.started_at, j.finished_at`
	rows, err := r.DB.Query(ctx, q, tenantID, kbID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DocumentJob, 0)
	for rows.Next() {
		var item DocumentJob
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.FileID,
			&item.JobType, &item.Status, &item.MaxChars, &item.OverlapChars, &item.Attempts, &item.LastError,
			&item.Title, &item.CreatedAt, &item.UpdatedAt, &item.StartedAt, &item.FinishedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) ClaimAnyDocumentJobs(ctx context.Context, limit int) ([]DocumentJob, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	const q = `
WITH picked AS (
    SELECT id
    FROM document_jobs
    WHERE deleted_at IS NULL
      AND status IN ('pending', 'failed')
    ORDER BY created_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE document_jobs j
SET status = 'processing',
    attempts = attempts + 1,
    started_at = now(),
    updated_at = now(),
    last_error = NULL
FROM picked
WHERE j.id = picked.id
RETURNING j.id::text, j.tenant_id::text, j.knowledge_base_id::text, COALESCE(j.document_id::text, ''), j.file_id::text,
          j.job_type, j.status, j.max_chars, j.overlap_chars, j.attempts, COALESCE(j.last_error, ''),
          COALESCE(j.metadata->>'title', ''),
          j.created_at, j.updated_at, j.started_at, j.finished_at`
	rows, err := r.DB.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DocumentJob, 0)
	for rows.Next() {
		var item DocumentJob
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.FileID,
			&item.JobType, &item.Status, &item.MaxChars, &item.OverlapChars, &item.Attempts, &item.LastError,
			&item.Title, &item.CreatedAt, &item.UpdatedAt, &item.StartedAt, &item.FinishedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) MarkDocumentJobSuccess(ctx context.Context, tenantID, kbID, jobID, documentID string) error {
	const q = `
UPDATE document_jobs
SET status = 'success',
    document_id = NULLIF($4, '')::uuid,
    finished_at = now(),
    updated_at = now(),
    last_error = NULL
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL`
	tag, err := r.DB.Exec(ctx, q, jobID, tenantID, kbID, documentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentJobNotFound
	}
	return nil
}

func (r Repository) MarkDocumentJobFailed(ctx context.Context, tenantID, kbID, jobID string, cause error) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if len(msg) > 1000 {
		msg = msg[:1000]
	}
	const q = `
UPDATE document_jobs
SET status = 'failed',
    last_error = NULLIF($4, ''),
    finished_at = now(),
    updated_at = now()
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL`
	tag, err := r.DB.Exec(ctx, q, jobID, tenantID, kbID, msg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentJobNotFound
	}
	return nil
}

func (r Repository) CreateChunk(ctx context.Context, tenantID, kbID string, req CreateChunkRequest) (Chunk, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := r.EnsureDocumentAccess(ctx, tenantID, kbID, req.DocumentID); err != nil {
		return Chunk{}, err
	}
	metadata, err := jsonObject(req.Metadata)
	if err != nil {
		return Chunk{}, err
	}
	if req.EmbeddingModel == "" {
		req.EmbeddingModel = "text-embedding-3-small"
	}
	const q = `
INSERT INTO document_chunks(
    tenant_id, knowledge_base_id, document_id, chunk_no, content, content_tokens, content_sha256,
    metadata, embedding, embedding_model, embedding_status
) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb, $9::vector, $10, 'success')
RETURNING id::text, tenant_id::text, knowledge_base_id::text, document_id::text, chunk_no, content, content_tokens,
          COALESCE(content_sha256, ''), COALESCE(embedding_model, ''), embedding_status, created_at, updated_at`
	var item Chunk
	err = r.DB.QueryRow(ctx, q,
		tenantID,
		kbID,
		req.DocumentID,
		req.ChunkNo,
		req.Content,
		req.ContentTokens,
		req.ContentSHA256,
		metadata,
		vectorLiteral(req.Embedding),
		req.EmbeddingModel,
	).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.ChunkNo, &item.Content, &item.ContentTokens,
		&item.ContentSHA256, &item.EmbeddingModel, &item.EmbeddingStatus, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return Chunk{}, err
	}
	_, _ = r.DB.Exec(ctx, `
UPDATE documents
SET chunk_status = 'success', embedding_status = 'success', updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND knowledge_base_id = $3`, req.DocumentID, tenantID, kbID)
	return item, nil
}

func (r Repository) EnsureDocumentAccess(ctx context.Context, tenantID, kbID, documentID string) error {
	const q = `
SELECT 1
FROM documents
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL`
	var ok int
	err := r.DB.QueryRow(ctx, q, documentID, tenantID, kbID).Scan(&ok)
	if err == pgx.ErrNoRows {
		return ErrDocumentNotFound
	}
	return err
}

func (r Repository) ListPendingChunks(ctx context.Context, tenantID, kbID string, limit int) ([]PendingChunk, error) {
	if err := r.EnsureAccess(ctx, tenantID, kbID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, knowledge_base_id::text, document_id::text, chunk_no, content, content_tokens, embedding_status
FROM document_chunks
WHERE tenant_id = $1
  AND knowledge_base_id = $2
  AND deleted_at IS NULL
  AND embedding_status IN ('pending', 'failed')
ORDER BY created_at ASC
LIMIT $3`
	rows, err := r.DB.Query(ctx, q, tenantID, kbID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PendingChunk, 0)
	for rows.Next() {
		var item PendingChunk
		if err := rows.Scan(&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.ChunkNo, &item.Content, &item.ContentTokens, &item.EmbeddingStatus); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) UpdateChunkEmbedding(ctx context.Context, tenantID, kbID, chunkID string, req UpdateChunkEmbeddingRequest) (Chunk, error) {
	metadata, err := jsonObject(req.Metadata)
	if err != nil {
		return Chunk{}, err
	}
	if req.EmbeddingModel == "" {
		req.EmbeddingModel = "text-embedding-3-small"
	}
	const q = `
UPDATE document_chunks
SET embedding = $4::vector,
    embedding_model = $5,
    embedding_status = 'success',
    metadata = metadata || $6::jsonb,
    updated_at = now()
WHERE id = $1
  AND tenant_id = $2
  AND knowledge_base_id = $3
  AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, knowledge_base_id::text, document_id::text, chunk_no, content, content_tokens,
          COALESCE(content_sha256, ''), COALESCE(embedding_model, ''), embedding_status, created_at, updated_at`
	var item Chunk
	err = r.DB.QueryRow(ctx, q, chunkID, tenantID, kbID, vectorLiteral(req.Embedding), req.EmbeddingModel, metadata).Scan(
		&item.ID, &item.TenantID, &item.KnowledgeBaseID, &item.DocumentID, &item.ChunkNo, &item.Content, &item.ContentTokens,
		&item.ContentSHA256, &item.EmbeddingModel, &item.EmbeddingStatus, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return Chunk{}, ErrChunkNotFound
	}
	if err != nil {
		return Chunk{}, err
	}
	_, _ = r.DB.Exec(ctx, `
UPDATE documents
SET embedding_status = CASE
    WHEN EXISTS (
        SELECT 1 FROM document_chunks
        WHERE document_id = $1 AND tenant_id = $2 AND knowledge_base_id = $3 AND deleted_at IS NULL AND embedding_status IN ('pending', 'failed')
    ) THEN 'processing'
    ELSE 'success'
END,
updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND knowledge_base_id = $3`, item.DocumentID, tenantID, kbID)
	return item, nil
}

var (
	ErrKnowledgeBaseNotFound = errors.New("knowledge base not found")
	ErrDocumentNotFound      = errors.New("document not found")
	ErrChunkNotFound         = errors.New("chunk not found")
	ErrDocumentJobNotFound   = errors.New("document job not found")
)

func jsonObject(v map[string]any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}
