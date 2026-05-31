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

func TestCreateGenealogyEdgeRequestNormalizeAndValidate(t *testing.T) {
	req := CreateGenealogyEdgeRequest{
		ParentAgentID: " parent-1 ",
		ChildAgentID:  " child-1 ",
		RelationType:  " INHERIT ",
	}
	req.Normalize()
	if req.ParentAgentID != "parent-1" || req.ChildAgentID != "child-1" || req.RelationType != "inherit" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid genealogy edge request: %v", err)
	}

	req.RelationType = ""
	req.Normalize()
	if req.RelationType != "fork" {
		t.Fatalf("expected default relation_type fork, got %q", req.RelationType)
	}

	req.ChildAgentID = ""
	if err := req.Validate(); err == nil {
		t.Fatal("expected missing child_agent_id error")
	}

	req.ChildAgentID = "agent-1"
	req.ParentAgentID = "agent-1"
	if err := req.Validate(); err == nil {
		t.Fatal("expected same parent and child error")
	}

	req.ParentAgentID = "agent-2"
	req.RelationType = "unknown"
	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid relation_type error")
	}
}

func TestGenealogyGraphQueryNormalizeAndValidate(t *testing.T) {
	req := GenealogyGraphQuery{
		Q:            "  客服助手  ",
		RelationType: " ROUTE ",
	}
	req.Normalize()
	if req.Q != "客服助手" || req.RelationType != "route" {
		t.Fatalf("normalized query = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid genealogy graph query: %v", err)
	}

	req.RelationType = ""
	if err := req.Validate(); err != nil {
		t.Fatalf("expected empty relation type to be valid: %v", err)
	}

	req.RelationType = "unknown"
	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid relation_type error")
	}

	req.RelationType = ""
	req.Q = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	if err := req.Validate(); err == nil {
		t.Fatal("expected q length validation error")
	}
}

func TestBuildGenealogyGraphSummary(t *testing.T) {
	nodes := []GenealogyNode{
		{ID: "root"},
		{ID: "child"},
		{ID: "route"},
		{ID: "isolated"},
	}
	edges := []GenealogyEdge{
		{ParentAgentID: "root", ChildAgentID: "child", RelationType: "fork"},
		{ParentAgentID: "child", ChildAgentID: "route", RelationType: "route"},
	}

	summary := buildGenealogyGraphSummary(nodes, edges)
	if summary.Nodes != 4 || summary.Edges != 2 || summary.Roots != 2 || summary.Isolated != 1 {
		t.Fatalf("unexpected summary = %#v", summary)
	}
	if len(summary.RelationTypes) != 2 {
		t.Fatalf("unexpected relation type count = %#v", summary.RelationTypes)
	}
	if summary.RelationTypes[0].RelationType != "fork" || summary.RelationTypes[0].Count != 1 {
		t.Fatalf("unexpected first relation count = %#v", summary.RelationTypes[0])
	}
	if summary.RelationTypes[1].RelationType != "route" || summary.RelationTypes[1].Count != 1 {
		t.Fatalf("unexpected second relation count = %#v", summary.RelationTypes[1])
	}
}

func TestBuildGenealogyGraphSummaryRootMarkerIsNotIsolated(t *testing.T) {
	nodes := []GenealogyNode{{ID: "root"}}
	edges := []GenealogyEdge{{ChildAgentID: "root", RelationType: "fork"}}

	summary := buildGenealogyGraphSummary(nodes, edges)
	if summary.Roots != 1 || summary.Isolated != 0 {
		t.Fatalf("root marker summary = %#v", summary)
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
