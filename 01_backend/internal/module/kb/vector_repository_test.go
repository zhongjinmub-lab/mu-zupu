package kb

import (
	"strings"
	"testing"
)

func TestBuildSearchSQLKeepsTenantAndKBIsolation(t *testing.T) {
	req := SearchRequest{
		TenantID:        "tenant-1",
		KnowledgeBaseID: "kb-1",
		Embedding:       make([]float32, DefaultEmbeddingDim),
		Query:           "hello",
		TopK:            10,
		CandidateK:      50,
		MinScore:        0.2,
	}
	sql, args := buildSearchSQL(req, VectorProfile{VectorWeight: 0.7, TextWeight: 0.3}, true)
	required := []string{
		"c.tenant_id = $1",
		"c.knowledge_base_id = $2",
		"c.deleted_at IS NULL",
	}
	for _, needle := range required {
		if !strings.Contains(sql, needle) {
			t.Fatalf("search sql missing %q:\n%s", needle, sql)
		}
	}
	if args[0] != "tenant-1" || args[1] != "kb-1" {
		t.Fatalf("unexpected isolation args: %#v", args[:2])
	}
}

func TestVectorLiteralUsesPgvectorFormat(t *testing.T) {
	got := vectorLiteral([]float32{1, 0.5})
	if got != "[1.00000000,0.50000000]" {
		t.Fatalf("vectorLiteral = %q", got)
	}
}

func TestApplyProfileKeepsExplicitZeroMinScore(t *testing.T) {
	req := SearchRequest{TopK: 3, CandidateK: 10}
	req.SetMinScore(0)
	applyProfile(&req, VectorProfile{TopK: 10, CandidateK: 50, MinScore: 0.2})
	if req.MinScore != 0 {
		t.Fatalf("MinScore = %v, want explicit zero", req.MinScore)
	}
}

func TestApplyProfileUsesDefaultMinScoreWhenUnset(t *testing.T) {
	req := SearchRequest{TopK: 3, CandidateK: 10}
	applyProfile(&req, VectorProfile{TopK: 10, CandidateK: 50, MinScore: 0.2})
	if req.MinScore != 0.2 {
		t.Fatalf("MinScore = %v, want profile default", req.MinScore)
	}
}

func TestFallbackTermsSplitsChineseQuestion(t *testing.T) {
	got := fallbackTerms("李四和张三是什么关系？")
	contains := func(v string) bool {
		for _, item := range got {
			if item == v {
				return true
			}
		}
		return false
	}
	if !contains("李四") || !contains("张三") {
		t.Fatalf("fallback terms = %#v", got)
	}
	if contains("什么") || contains("关系") {
		t.Fatalf("weak fallback terms should be filtered: %#v", got)
	}
}
