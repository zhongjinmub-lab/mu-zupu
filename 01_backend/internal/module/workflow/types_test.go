package workflow

import "testing"

// 构造一个合法的线性工作流：start -> llm -> end。
func validLinearGraph() WorkflowGraph {
	return WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_start", Type: "start", Name: "开始"},
			{ID: "n_llm", Type: "llm", Name: "生成"},
			{ID: "n_end", Type: "end", Name: "结束"},
		},
		Edges: []WorkflowEdge{
			{From: "n_start", To: "n_llm"},
			{From: "n_llm", To: "n_end"},
		},
	}
}

func TestValidateWorkflowGraphAcceptsValidLinearGraph(t *testing.T) {
	result := ValidateWorkflowGraph(validLinearGraph())
	if !result.Valid {
		t.Fatalf("expected valid graph, issues=%v", result.Issues)
	}
	if len(result.ExecutionOrder) != 3 || result.ExecutionOrder[0] != "n_start" {
		t.Fatalf("unexpected execution order: %v", result.ExecutionOrder)
	}
}

func TestValidateWorkflowGraphTracksHumanApprovalNodes(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_start", Type: "start"},
			{ID: "n_appr", Type: "human_approval"},
			{ID: "n_end", Type: "end"},
		},
		Edges: []WorkflowEdge{
			{From: "n_start", To: "n_appr"},
			{From: "n_appr", To: "n_end"},
		},
	}
	result := ValidateWorkflowGraph(graph)
	if !result.Valid {
		t.Fatalf("expected valid graph, issues=%v", result.Issues)
	}
	if len(result.HumanApprovalNodes) != 1 || result.HumanApprovalNodes[0] != "n_appr" {
		t.Fatalf("human approval nodes = %v", result.HumanApprovalNodes)
	}
}

func TestValidateWorkflowGraphRejectsMissingStart(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_llm", Type: "llm"},
			{ID: "n_end", Type: "end"},
		},
		Edges: []WorkflowEdge{{From: "n_llm", To: "n_end"}},
	}
	result := ValidateWorkflowGraph(graph)
	if result.Valid {
		t.Fatal("expected invalid graph when start node missing")
	}
}

func TestValidateWorkflowGraphRejectsMissingEnd(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_start", Type: "start"},
			{ID: "n_llm", Type: "llm"},
		},
		Edges: []WorkflowEdge{{From: "n_start", To: "n_llm"}},
	}
	result := ValidateWorkflowGraph(graph)
	if result.Valid {
		t.Fatal("expected invalid graph when end node missing")
	}
}

func TestValidateWorkflowGraphDetectsCycle(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_start", Type: "start"},
			{ID: "a", Type: "llm"},
			{ID: "b", Type: "llm"},
			{ID: "n_end", Type: "end"},
		},
		Edges: []WorkflowEdge{
			{From: "n_start", To: "a"},
			{From: "a", To: "b"},
			{From: "b", To: "a"},
			{From: "b", To: "n_end"},
		},
	}
	result := ValidateWorkflowGraph(graph)
	if result.Valid {
		t.Fatal("expected invalid graph when a cycle exists")
	}
	if len(result.ExecutionOrder) != 0 {
		t.Fatalf("cyclic graph should not produce execution order, got %v", result.ExecutionOrder)
	}
}

func TestValidateWorkflowGraphRejectsDanglingEdge(t *testing.T) {
	graph := validLinearGraph()
	graph.Edges = append(graph.Edges, WorkflowEdge{From: "n_llm", To: "ghost"})
	result := ValidateWorkflowGraph(graph)
	if result.Valid {
		t.Fatal("expected invalid graph when an edge references a missing node")
	}
}

func TestValidateWorkflowGraphRejectsInvalidNodeType(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_start", Type: "start"},
			{ID: "n_x", Type: "magic"},
			{ID: "n_end", Type: "end"},
		},
		Edges: []WorkflowEdge{
			{From: "n_start", To: "n_x"},
			{From: "n_x", To: "n_end"},
		},
	}
	result := ValidateWorkflowGraph(graph)
	if result.Valid {
		t.Fatal("expected invalid graph for unknown node type")
	}
}

func TestValidateWorkflowGraphWarnsOnEndWithOutEdge(t *testing.T) {
	graph := validLinearGraph()
	graph.Edges = append(graph.Edges, WorkflowEdge{From: "n_end", To: "n_llm"})
	result := ValidateWorkflowGraph(graph)
	// end 有出边属于结构错误
	if result.Valid {
		t.Fatal("expected invalid graph when end node has an out edge")
	}
}

func TestDefaultWorkflowOrchestrationPolicyIsSafeByDefault(t *testing.T) {
	policy := DefaultWorkflowOrchestrationPolicy()
	if policy.Enabled {
		t.Fatal("workflow engine must be disabled by default")
	}
	if policy.ExecutionMode != "validate_only" {
		t.Fatalf("execution mode = %q", policy.ExecutionMode)
	}
	if policy.AuditAction != "agent.workflow.run" || len(policy.Guardrails) == 0 {
		t.Fatalf("unexpected policy %#v", policy)
	}
}

func TestCreateWorkflowRequestNormalizeAndValidate(t *testing.T) {
	req := CreateWorkflowRequest{
		Name:       "  订单审批流  ",
		Code:       "  Order_Approve-1 ",
		Definition: validLinearGraph(),
	}
	req.Normalize()
	if req.Name != "订单审批流" || req.Code != "order_approve-1" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}

	bad := CreateWorkflowRequest{Name: "x", Code: "bad code", Definition: validLinearGraph()}
	bad.Normalize()
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid code error")
	}

	empty := CreateWorkflowRequest{Name: "x", Code: "ok"}
	empty.Normalize()
	if err := empty.Validate(); err == nil {
		t.Fatal("expected error when nodes are empty")
	}
}

func TestUpdateWorkflowRequestValidate(t *testing.T) {
	graph := validLinearGraph()
	req := UpdateWorkflowRequest{Name: "新名称", Definition: &graph}
	req.Normalize()
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid update: %v", err)
	}

	emptyGraph := WorkflowGraph{Nodes: []WorkflowNode{}, Edges: []WorkflowEdge{}}
	bad := UpdateWorkflowRequest{Definition: &emptyGraph}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error when definition has no nodes")
	}

	// Definition 为 nil 时不校验图，仅做名称长度校验。
	ok := UpdateWorkflowRequest{Name: "只改名"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected nil-definition update to be valid: %v", err)
	}
}

func TestMarshalDefinitionFillsEmptySlices(t *testing.T) {
	raw, err := marshalDefinition(WorkflowGraph{})
	if err != nil {
		t.Fatalf("marshal definition: %v", err)
	}
	if raw != `{"nodes":[],"edges":[]}` {
		t.Fatalf("unexpected marshalled definition: %s", raw)
	}
}

func TestSimulateWorkflowRunCompletesLinear(t *testing.T) {
	result := SimulateWorkflowRun(validLinearGraph())
	if result.Status != RunStatusCompletedDryRun {
		t.Fatalf("status = %q, want completed_dry_run", result.Status)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(result.Steps))
	}
	var llmSimulated bool
	for _, step := range result.Steps {
		if step.NodeID == "n_llm" && step.Status == "simulated" {
			llmSimulated = true
		}
	}
	if !llmSimulated {
		t.Fatalf("llm node should be simulated, steps=%#v", result.Steps)
	}
}

func TestSimulateWorkflowRunPausesAtHumanApproval(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_start", Type: "start"},
			{ID: "n_appr", Type: "human_approval"},
			{ID: "n_llm", Type: "llm"},
			{ID: "n_end", Type: "end"},
		},
		Edges: []WorkflowEdge{
			{From: "n_start", To: "n_appr"},
			{From: "n_appr", To: "n_llm"},
			{From: "n_llm", To: "n_end"},
		},
	}
	result := SimulateWorkflowRun(graph)
	if result.Status != RunStatusAwaitingApproval {
		t.Fatalf("status = %q, want awaiting_approval", result.Status)
	}
	if result.AwaitingApprovalNode != "n_appr" {
		t.Fatalf("awaiting node = %q", result.AwaitingApprovalNode)
	}
	// 人工确认节点之后的节点应为 pending。
	statusByID := map[string]string{}
	for _, step := range result.Steps {
		statusByID[step.NodeID] = step.Status
	}
	if statusByID["n_llm"] != "pending" || statusByID["n_end"] != "pending" {
		t.Fatalf("nodes after approval should be pending, steps=%#v", result.Steps)
	}
}

func TestSimulateWorkflowRunBlockedOnInvalidGraph(t *testing.T) {
	graph := WorkflowGraph{
		Nodes: []WorkflowNode{
			{ID: "n_llm", Type: "llm"},
			{ID: "n_end", Type: "end"},
		},
		Edges: []WorkflowEdge{{From: "n_llm", To: "n_end"}},
	}
	result := SimulateWorkflowRun(graph)
	if result.Status != RunStatusBlocked {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if len(result.Steps) != 0 {
		t.Fatalf("blocked run should have no steps, got %#v", result.Steps)
	}
	if len(result.Issues) == 0 {
		t.Fatal("blocked run should carry validation issues")
	}
}

func TestSummarizeWorkflows(t *testing.T) {
	workflows := []Workflow{
		{Status: StatusDraft},
		{Status: StatusDraft},
		{Status: StatusPublished},
	}
	summary := SummarizeWorkflows(workflows)
	if summary.Total != 3 || summary.Draft != 2 || summary.Published != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.ByStatus[StatusDraft] != 2 || summary.ByStatus[StatusPublished] != 1 {
		t.Fatalf("unexpected by-status distribution: %#v", summary.ByStatus)
	}
}

func TestSummarizeWorkflowsEmpty(t *testing.T) {
	summary := SummarizeWorkflows(nil)
	if summary.Total != 0 || len(summary.ByStatus) != 0 {
		t.Fatalf("empty summary should be zero-valued: %#v", summary)
	}
}
