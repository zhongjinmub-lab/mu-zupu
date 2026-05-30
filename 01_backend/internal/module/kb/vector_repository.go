package kb

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VectorRepository struct {
	DB *pgxpool.Pool
}

func NewVectorRepository(db *pgxpool.Pool) VectorRepository {
	return VectorRepository{DB: db}
}

func (r VectorRepository) LoadProfile(ctx context.Context, tenantID, knowledgeBaseID string) (VectorProfile, error) {
	profile := VectorProfile{
		HNSWEFSearch: 80,
		VectorWeight: 0.7,
		TextWeight:   0.3,
		TopK:         10,
		CandidateK:   50,
		MinScore:     0.2,
	}

	const q = `
SELECT hnsw_ef_search, vector_weight::float8, text_weight::float8, top_k, candidate_k, min_score::float8
FROM vector_index_profiles
WHERE enabled = TRUE
  AND (tenant_id = $1 OR tenant_id IS NULL)
  AND (knowledge_base_id = $2 OR knowledge_base_id IS NULL)
ORDER BY tenant_id NULLS LAST, knowledge_base_id NULLS LAST
LIMIT 1`

	err := r.DB.QueryRow(ctx, q, tenantID, knowledgeBaseID).Scan(
		&profile.HNSWEFSearch,
		&profile.VectorWeight,
		&profile.TextWeight,
		&profile.TopK,
		&profile.CandidateK,
		&profile.MinScore,
	)
	if err != nil && err != pgx.ErrNoRows {
		return profile, err
	}
	return profile, nil
}

func (r VectorRepository) Search(ctx context.Context, req SearchRequest, hybrid bool) ([]SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	profile, err := r.LoadProfile(ctx, req.TenantID, req.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}
	applyProfile(&req, profile)

	start := time.Now()
	rows, err := r.query(ctx, req, profile, hybrid)
	if err == nil && len(rows) == 0 && hybrid {
		rows, err = r.keywordFallback(ctx, req)
	}
	latencyMs := int(time.Since(start).Milliseconds())
	_ = r.InsertSearchLog(context.Background(), req, profile, len(rows), latencyMs)
	return rows, err
}

func (r VectorRepository) query(ctx context.Context, req SearchRequest, profile VectorProfile, hybrid bool) ([]SearchResult, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if profile.HNSWEFSearch > 0 {
		if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL hnsw.ef_search = %d", profile.HNSWEFSearch)); err != nil {
			return nil, err
		}
	}

	sql, args := buildSearchSQL(req, profile, hybrid)

	pgRows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	out := make([]SearchResult, 0, req.TopK)
	for pgRows.Next() {
		var item SearchResult
		if err := pgRows.Scan(&item.ChunkID, &item.DocumentID, &item.Title, &item.Content, &item.Score); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := pgRows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func buildSearchSQL(req SearchRequest, profile VectorProfile, hybrid bool) (string, []any) {
	embedding := vectorLiteral(req.Embedding)
	args := []any{req.TenantID, req.KnowledgeBaseID, embedding, req.CandidateK}
	sql := `
WITH candidates AS (
    SELECT c.id, c.document_id, COALESCE(d.title, '') AS title, c.content,
           (1 - (c.embedding <=> $3::vector)) AS score
    FROM document_chunks c
    LEFT JOIN documents d ON d.id = c.document_id
    WHERE c.tenant_id = $1
      AND c.knowledge_base_id = $2
      AND c.deleted_at IS NULL
      AND c.embedding IS NOT NULL
    ORDER BY c.embedding <=> $3::vector
    LIMIT $4
)
SELECT id::text, document_id::text, title, content, score
FROM candidates
WHERE score >= $5
ORDER BY score DESC
LIMIT $6`
	args = append(args, req.MinScore, req.TopK)

	if hybrid && strings.TrimSpace(req.Query) != "" {
		args = []any{req.TenantID, req.KnowledgeBaseID, embedding, req.Query, profile.VectorWeight, profile.TextWeight, req.CandidateK, req.MinScore, req.TopK}
		sql = `
WITH candidates AS (
    SELECT c.id, c.document_id, COALESCE(d.title, '') AS title, c.content,
           (1 - (c.embedding <=> $3::vector)) AS vector_score,
           ts_rank_cd(c.search_vector, websearch_to_tsquery('simple', $4)) AS text_score
    FROM document_chunks c
    LEFT JOIN documents d ON d.id = c.document_id
    WHERE c.tenant_id = $1
      AND c.knowledge_base_id = $2
      AND c.deleted_at IS NULL
      AND c.embedding IS NOT NULL
      AND (
          c.search_vector @@ websearch_to_tsquery('simple', $4)
          OR c.embedding <=> $3::vector < 0.8
      )
    ORDER BY c.embedding <=> $3::vector
    LIMIT $7
), scored AS (
    SELECT id, document_id, title, content,
           (vector_score * $5::float8 + text_score * $6::float8) AS score
    FROM candidates
)
SELECT id::text, document_id::text, title, content, score
FROM scored
WHERE score >= $8
ORDER BY score DESC
LIMIT $9`
	}
	return sql, args
}

func (r VectorRepository) InsertSearchLog(ctx context.Context, req SearchRequest, profile VectorProfile, resultCount, latencyMs int) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	const q = `
INSERT INTO vector_search_logs(
    tenant_id, knowledge_base_id, agent_id, query, top_k, candidate_k, min_score, result_count, latency_ms, profile
) VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9,
          jsonb_build_object('hnsw_ef_search', $10, 'vector_weight', $11, 'text_weight', $12))`
	_, err := r.DB.Exec(ctx, q,
		req.TenantID,
		req.KnowledgeBaseID,
		req.AgentID,
		req.Query,
		req.TopK,
		req.CandidateK,
		req.MinScore,
		resultCount,
		latencyMs,
		profile.HNSWEFSearch,
		profile.VectorWeight,
		profile.TextWeight,
	)
	return err
}

func (r VectorRepository) keywordFallback(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	terms := fallbackTerms(req.Query)
	if len(terms) == 0 {
		return []SearchResult{}, nil
	}
	args := []any{req.TenantID, req.KnowledgeBaseID}
	conditions := make([]string, 0, len(terms))
	for _, term := range terms {
		args = append(args, "%"+term+"%")
		conditions = append(conditions, fmt.Sprintf("c.content ILIKE $%d", len(args)))
	}
	args = append(args, req.TopK)
	sql := `
SELECT c.id::text, c.document_id::text, COALESCE(d.title, '') AS title, c.content, 0.01::float8 AS score
FROM document_chunks c
LEFT JOIN documents d ON d.id = c.document_id
WHERE c.tenant_id = $1
  AND c.knowledge_base_id = $2
  AND c.deleted_at IS NULL
  AND c.embedding IS NOT NULL
  AND (` + strings.Join(conditions, " OR ") + `)
ORDER BY c.created_at DESC
LIMIT $` + fmt.Sprintf("%d", len(args))
	rows, err := r.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SearchResult, 0, req.TopK)
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.ChunkID, &item.DocumentID, &item.Title, &item.Content, &item.Score); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

var fallbackTermPattern = regexp.MustCompile(`[\p{Han}A-Za-z0-9]{2,}`)

func fallbackTerms(query string) []string {
	matches := fallbackTermPattern.FindAllString(strings.TrimSpace(query), -1)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, item := range matches {
		candidates := []string{item}
		if hasHan(item) {
			candidates = append(candidates, hanWindows(item)...)
		}
		for _, term := range candidates {
			term = strings.TrimSpace(term)
			if len([]rune(term)) < 2 || isWeakFallbackTerm(term) {
				continue
			}
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			out = append(out, term)
			if len(out) >= 12 {
				return out
			}
		}
	}
	return out
}

func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func hanWindows(s string) []string {
	runes := []rune(s)
	out := make([]string, 0)
	for size := 2; size <= 6; size++ {
		if len(runes) < size {
			continue
		}
		for i := 0; i+size <= len(runes); i++ {
			out = append(out, string(runes[i:i+size]))
		}
	}
	return out
}

func isWeakFallbackTerm(term string) bool {
	switch term {
	case "什么", "关系", "是什么", "什么关系", "为什么", "怎么", "如何", "哪个", "哪些":
		return true
	default:
		return false
	}
}

func applyProfile(req *SearchRequest, profile VectorProfile) {
	if req.TopK <= 0 {
		req.TopK = profile.TopK
	}
	if req.CandidateK <= 0 {
		req.CandidateK = profile.CandidateK
	}
	if !req.MinScoreSet() && req.MinScore <= 0 {
		req.MinScore = profile.MinScore
	}
	req.Normalize()
}

func vectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = fmt.Sprintf("%.8f", x)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
