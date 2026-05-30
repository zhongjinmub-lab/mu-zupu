package kb

import "testing"

func TestSearchRequestValidateEmbeddingDimension(t *testing.T) {
	req := SearchRequest{TenantID: "t1", KnowledgeBaseID: "kb1", Embedding: make([]float32, DefaultEmbeddingDim-1)}
	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid embedding dimension")
	}
	req.Embedding = make([]float32, DefaultEmbeddingDim)
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}
}

func TestCreateKnowledgeBaseRequires1536Dim(t *testing.T) {
	req := CreateKnowledgeBaseRequest{Name: "kb", Code: "kb", EmbeddingDim: 768}
	if err := req.Normalize().Validate(); err == nil {
		t.Fatal("expected embedding_dim validation error")
	}
}

func TestCreateChunkRequires1536Dim(t *testing.T) {
	req := CreateChunkRequest{DocumentID: "doc", Content: "hello", Embedding: make([]float32, 10)}
	if err := req.Validate(); err == nil {
		t.Fatal("expected embedding dimension validation error")
	}
}

func TestUpdateChunkEmbeddingRequires1536Dim(t *testing.T) {
	req := UpdateChunkEmbeddingRequest{Embedding: make([]float32, 10)}
	if err := req.Validate(); err == nil {
		t.Fatal("expected embedding dimension validation error")
	}
	req.Embedding = make([]float32, DefaultEmbeddingDim)
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid embedding: %v", err)
	}
}

func TestRebuildDocumentRequestNormalize(t *testing.T) {
	req := RebuildDocumentRequest{MaxChars: 100, OverlapChars: 100}
	req.Normalize()
	if req.MaxChars != 100 || req.OverlapChars != 10 {
		t.Fatalf("normalized request = %#v", req)
	}

	req = RebuildDocumentRequest{MaxChars: -1, OverlapChars: -1}
	req.Normalize()
	if req.MaxChars != 1200 || req.OverlapChars != 0 {
		t.Fatalf("default normalized request = %#v", req)
	}
}

func TestEnqueueDocumentJobRequestValidate(t *testing.T) {
	req := EnqueueDocumentJobRequest{FileID: " file-1 ", JobType: ""}
	req.Normalize()
	if req.FileID != "file-1" || req.JobType != "parse_chunk" || req.MaxChars != 1200 {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid parse job: %v", err)
	}

	req = EnqueueDocumentJobRequest{FileID: "file-1", JobType: "rebuild"}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected rebuild document_id validation error")
	}

	req.DocumentID = "doc-1"
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid rebuild job: %v", err)
	}
}

func TestAskRequestNormalizeAndValidate(t *testing.T) {
	req := AskRequest{Question: "  hello  ", TopK: 100, CandidateK: 1, MinScore: -1, MaxTokens: 9000, Temperature: 9}
	req.Normalize()
	if req.Question != "hello" || req.TopK != 5 || req.CandidateK != 25 || req.MinScore != 0 || req.MaxTokens != 1024 || req.Temperature != 0.2 {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid ask request: %v", err)
	}

	req.Question = ""
	if err := req.Validate(); err == nil {
		t.Fatal("expected question validation error")
	}
}
