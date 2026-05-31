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
