package agent

import "testing"

func TestCreateAgentRequestNormalizeAndValidate(t *testing.T) {
	req := CreateAgentRequest{Name: " Agent ", Code: " My_Agent-1 "}
	req.Normalize()
	if req.Name != "Agent" || req.Code != "my_agent-1" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}

	req.Code = "bad code"
	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid code")
	}
}

func TestBindKnowledgeBaseRequestValidate(t *testing.T) {
	req := BindKnowledgeBaseRequest{KnowledgeBaseID: " kb-1 "}
	req.Normalize()
	if req.KnowledgeBaseID != "kb-1" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid binding request: %v", err)
	}

	req.KnowledgeBaseID = ""
	if err := req.Validate(); err == nil {
		t.Fatal("expected missing knowledge_base_id error")
	}
}

func TestTestChatRequestNormalizeAndValidate(t *testing.T) {
	req := TestChatRequest{Message: " hello ", KnowledgeBaseID: " kb-1 ", TopK: 99, CandidateK: 1, MinScore: -1, MaxTokens: 9000, Temperature: 9}
	req.Normalize()
	if req.Message != "hello" || req.KnowledgeBaseID != "kb-1" || req.TopK != 5 || req.CandidateK != 25 || req.MinScore != 0 || req.MaxTokens != 1024 || req.Temperature != 0.2 {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid test chat request: %v", err)
	}

	req.Message = ""
	if err := req.Validate(); err == nil {
		t.Fatal("expected message validation error")
	}
}

func TestChatRequestNormalizeAndValidate(t *testing.T) {
	req := ChatRequest{
		ConversationID:  " conv-1 ",
		Message:         " hello ",
		KnowledgeBaseID: " kb-1 ",
		Title:           " title ",
		TopK:            99,
		CandidateK:      1,
		MinScore:        -1,
		MaxTokens:       9000,
		Temperature:     9,
		HistoryLimit:    999,
	}
	req.Normalize()
	if req.ConversationID != "conv-1" || req.Message != "hello" || req.KnowledgeBaseID != "kb-1" || req.Title != "title" {
		t.Fatalf("trimmed request = %#v", req)
	}
	if req.TopK != 5 || req.CandidateK != 25 || req.MinScore != 0 || req.MaxTokens != 1024 || req.Temperature != 0.2 || req.HistoryLimit != 20 {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid chat request: %v", err)
	}

	req.Message = ""
	if err := req.Validate(); err == nil {
		t.Fatal("expected message validation error")
	}
}
