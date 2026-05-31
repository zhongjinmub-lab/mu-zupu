package agent

import (
	"testing"
	"time"
)

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

func TestToolCallLogQueryNormalizeAndValidate(t *testing.T) {
	q := ToolCallLogQuery{
		TenantID: " tenant-1 ",
		AgentID:  "7b7d45a7-10f7-4aa8-b068-d90c4e35f5dc",
		ToolName: " http_get ",
		Status:   " dry_run_ok ",
		Limit:    500,
	}
	q.Normalize()
	if q.TenantID != "tenant-1" || q.ToolName != "http_get" || q.Status != "dry_run_ok" || q.Limit != 50 {
		t.Fatalf("normalized query = %#v", q)
	}
	if err := q.Validate(); err != nil {
		t.Fatalf("expected valid query: %v", err)
	}
}

func TestToolCallLogQueryRejectsInvalidAgentID(t *testing.T) {
	q := ToolCallLogQuery{TenantID: "tenant-1", AgentID: "bad-agent-id", Limit: 10}
	q.Normalize()
	if err := q.Validate(); err == nil {
		t.Fatal("expected agent_id validation error")
	}
}

func TestToolCallLogQueryRejectsInvalidTimeRange(t *testing.T) {
	q := ToolCallLogQuery{
		TenantID: "tenant-1",
		From:     time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
		Limit:    10,
	}
	q.Normalize()
	if err := q.Validate(); err == nil {
		t.Fatal("expected time range validation error")
	}
}

func TestToolCallLogQueryRequiresTenant(t *testing.T) {
	q := ToolCallLogQuery{Limit: 10}
	q.Normalize()
	if err := q.Validate(); err == nil {
		t.Fatal("expected tenant_id validation error")
	}
}

func TestParseToolCallLogTime(t *testing.T) {
	cases := []string{
		"2026-05-28T10:11:12Z",
		"2026-05-28T10:11:12.123456789Z",
		"2026-05-28",
	}
	for _, tc := range cases {
		if _, err := ParseToolCallLogTime(tc); err != nil {
			t.Fatalf("parse %q: %v", tc, err)
		}
	}
	if _, err := ParseToolCallLogTime("2026/05/28"); err == nil {
		t.Fatal("expected invalid time format error")
	}
}

func TestToolCallLogCursorEncodeDecode(t *testing.T) {
	item := ToolCallLog{
		ID:        "7b7d45a7-10f7-4aa8-b068-d90c4e35f5dc",
		CreatedAt: time.Date(2026, 5, 28, 10, 11, 12, 123456789, time.UTC),
	}
	raw := EncodeToolCallLogCursor(item)
	if raw == "" {
		t.Fatal("expected cursor")
	}
	cursor, err := DecodeToolCallLogCursor(raw)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if cursor.ID != item.ID || !cursor.CreatedAt.Equal(item.CreatedAt) {
		t.Fatalf("decoded cursor = %#v", cursor)
	}
}

func TestToolCallLogCursorDecodeRejectsInvalidCursor(t *testing.T) {
	for _, raw := range []string{"bad", "bm90LWVub3VnaA", "MjAyNi0wNS0yOFQxMDoxMToxMlo=|bad"} {
		if _, err := DecodeToolCallLogCursor(raw); err == nil {
			t.Fatalf("expected invalid cursor error for %q", raw)
		}
	}
}

func TestDefaultMCPGatewayPolicyIsSafeByDefault(t *testing.T) {
	policy := DefaultMCPGatewayPolicy()
	if policy.Enabled {
		t.Fatal("MCP gateway must be disabled by default")
	}
	if policy.DefaultAction != "deny" {
		t.Fatalf("default action = %q, want deny", policy.DefaultAction)
	}
	if policy.AuditAction != "agent.mcp.call" {
		t.Fatalf("audit action = %q", policy.AuditAction)
	}
	if !policy.DangerConfirmation {
		t.Fatal("danger confirmation must be required")
	}
	if len(policy.AvailableServers) == 0 || len(policy.DangerousServers) == 0 {
		t.Fatal("expected both available and dangerous servers")
	}
}

func TestDefaultMCPServerCatalogHasUniqueCodes(t *testing.T) {
	items := DefaultMCPServerCatalog()
	if len(items) == 0 {
		t.Fatal("expected non-empty MCP catalog")
	}
	seen := map[string]bool{}
	for _, item := range items {
		if item.Code == "" {
			t.Fatal("catalog item missing code")
		}
		if seen[item.Code] {
			t.Fatalf("duplicate MCP server code %q", item.Code)
		}
		seen[item.Code] = true
		if item.AuditAction != "agent.mcp.call" {
			t.Fatalf("catalog item %q audit action = %q", item.Code, item.AuditAction)
		}
	}
}

func TestFindMCPCatalogItem(t *testing.T) {
	if _, ok := FindMCPCatalogItem("  KB_Resource "); !ok {
		t.Fatal("expected to find kb_resource ignoring case and spaces")
	}
	if _, ok := FindMCPCatalogItem("does-not-exist"); ok {
		t.Fatal("expected missing server to be not found")
	}
}

func TestBuildMCPTestResultBlocksDangerousServer(t *testing.T) {
	item, ok := FindMCPCatalogItem("external_http")
	if !ok {
		t.Fatal("external_http server should exist")
	}
	result := BuildMCPTestResult(item, MCPTestRequest{})
	if result.Allowed || result.Status != "blocked" {
		t.Fatalf("dangerous server result = %#v", result)
	}
	if !result.DryRun {
		t.Fatal("result must be dry run")
	}
}

func TestBuildMCPTestResultAllowsReadServer(t *testing.T) {
	item, ok := FindMCPCatalogItem("kb_resource")
	if !ok {
		t.Fatal("kb_resource server should exist")
	}
	result := BuildMCPTestResult(item, MCPTestRequest{Input: map[string]any{"query": "hello"}})
	if !result.Allowed || result.Status != "dry_run_ok" {
		t.Fatalf("read server result = %#v", result)
	}
	if result.InputSummary == "" {
		t.Fatal("expected input summary")
	}
}
