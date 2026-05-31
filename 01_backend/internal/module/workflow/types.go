package workflow

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// WorkflowNode 表示工作流图中的一个节点。
type WorkflowNode struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

// WorkflowEdge 表示工作流图中的一条有向边，Condition 用于条件分支说明。
type WorkflowEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition"`
}

// WorkflowGraph 表示一个工作流编排图定义。
type WorkflowGraph struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

// ValidateWorkflowRequest 是图校验接口的请求体。
type ValidateWorkflowRequest struct {
	Name       string        `json:"name"`
	Code       string        `json:"code"`
	Definition WorkflowGraph `json:"definition"`
}

// WorkflowValidationResult 是图校验结果，包含错误、警告、人工确认节点与拓扑执行顺序。
type WorkflowValidationResult struct {
	Valid              bool     `json:"valid"`
	NodeCount          int      `json:"node_count"`
	EdgeCount          int      `json:"edge_count"`
	Issues             []string `json:"issues"`
	Warnings           []string `json:"warnings"`
	HumanApprovalNodes []string `json:"human_approval_nodes"`
	ExecutionOrder     []string `json:"execution_order"`
}

// WorkflowNodeType 表示一种内置节点类型定义。
type WorkflowNodeType struct {
	Type                 string `json:"type"`
	Name                 string `json:"name"`
	Category             string `json:"category"`
	Description          string `json:"description"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	Terminal             bool   `json:"terminal"`
}

// WorkflowOrchestrationPolicy 描述工作流编排的安全默认策略。
type WorkflowOrchestrationPolicy struct {
	Enabled         bool     `json:"enabled"`
	ExecutionMode   string   `json:"execution_mode"`
	PermissionRole  string   `json:"permission_role"`
	AuditAction     string   `json:"audit_action"`
	MaxNodes        int      `json:"max_nodes"`
	Guardrails      []string `json:"guardrails"`
	AgentPolicyHint string   `json:"agent_policy_hint"`
}

// Normalize 归一化校验请求的字符串字段。
func (r *ValidateWorkflowRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.ToLower(strings.TrimSpace(r.Code))
}

// Validate 校验请求基础字段（图结构由 ValidateWorkflowGraph 单独诊断）。
func (r ValidateWorkflowRequest) Validate() error {
	if len(r.Definition.Nodes) == 0 {
		return errors.New("definition.nodes is required")
	}
	if len(r.Definition.Nodes) > maxWorkflowNodes {
		return errors.New("workflow nodes exceed the maximum limit")
	}
	return nil
}

const maxWorkflowNodes = 100

// DefaultWorkflowNodeTypes 返回内置节点类型目录。
func DefaultWorkflowNodeTypes() []WorkflowNodeType {
	return []WorkflowNodeType{
		{Type: "start", Name: "开始", Category: "control", Description: "工作流入口，必须且只能有一个。"},
		{Type: "llm", Name: "模型生成", Category: "compute", Description: "调用生成模型产出内容。"},
		{Type: "tool", Name: "工具调用", Category: "compute", Description: "调用已启用的工具或插件，遵循工具安全策略。"},
		{Type: "condition", Name: "条件分支", Category: "control", Description: "按条件路由到不同后继节点，建议至少两个分支。"},
		{Type: "human_approval", Name: "人工确认", Category: "control", Description: "暂停等待人工确认后再继续，危险操作必经此节点。", RequiresConfirmation: true},
		{Type: "end", Name: "结束", Category: "control", Description: "工作流出口，至少需要一个。", Terminal: true},
	}
}

// DefaultWorkflowOrchestrationPolicy 返回工作流编排安全默认策略。
func DefaultWorkflowOrchestrationPolicy() WorkflowOrchestrationPolicy {
	return WorkflowOrchestrationPolicy{
		Enabled:        false,
		ExecutionMode:  "validate_only",
		PermissionRole: "tenant_writer",
		AuditAction:    "agent.workflow.run",
		MaxNodes:       maxWorkflowNodes,
		Guardrails: []string{
			"工作流执行引擎默认关闭，当前版本仅提供图结构校验，不执行真实节点动作。",
			"工具与插件节点必须遵循工具安全策略和插件启用状态，未启用按拒绝处理。",
			"危险操作必须经过 human_approval 人工确认节点。",
			"工作流定义、发布和执行都按当前租户隔离，并写入审计日志。",
		},
		AgentPolicyHint: "工作流未发布或引擎未启用时，不允许 Agent 自主触发工作流执行。",
	}
}

// ValidateWorkflowGraph 对工作流图做结构校验：
// 校验节点 id 唯一与类型合法、起止节点数量、边引用、出边规则，并通过拓扑排序检测环、生成执行顺序。
// 该校验为纯函数，不执行任何真实节点动作。
func ValidateWorkflowGraph(graph WorkflowGraph) WorkflowValidationResult {
	result := WorkflowValidationResult{
		NodeCount:          len(graph.Nodes),
		EdgeCount:          len(graph.Edges),
		Issues:             []string{},
		Warnings:           []string{},
		HumanApprovalNodes: []string{},
		ExecutionOrder:     []string{},
	}

	validTypes := map[string]bool{}
	terminalTypes := map[string]bool{}
	for _, nt := range DefaultWorkflowNodeTypes() {
		validTypes[nt.Type] = true
		if nt.Terminal {
			terminalTypes[nt.Type] = true
		}
	}

	// 节点校验：id 唯一非空、类型合法，统计起止与人工确认节点。
	nodeIDs := map[string]bool{}
	nodeType := map[string]string{}
	nodeOrder := make([]string, 0, len(graph.Nodes))
	startCount := 0
	endCount := 0
	for _, n := range graph.Nodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			result.Issues = append(result.Issues, "存在缺少 id 的节点")
			continue
		}
		if nodeIDs[id] {
			result.Issues = append(result.Issues, "节点 id 重复："+id)
			continue
		}
		nodeIDs[id] = true
		nodeType[id] = n.Type
		nodeOrder = append(nodeOrder, id)
		if !validTypes[n.Type] {
			result.Issues = append(result.Issues, "节点 "+id+" 的类型非法："+n.Type)
		}
		switch n.Type {
		case "start":
			startCount++
		case "end":
			endCount++
		case "human_approval":
			result.HumanApprovalNodes = append(result.HumanApprovalNodes, id)
		}
	}
	if startCount != 1 {
		result.Issues = append(result.Issues, "工作流必须且只能有一个 start 节点")
	}
	if endCount < 1 {
		result.Issues = append(result.Issues, "工作流至少需要一个 end 节点")
	}

	// 边校验：引用的节点必须存在，重复边给出警告并只计一次。
	adj := map[string][]string{}
	indeg := map[string]int{}
	outCount := map[string]int{}
	for id := range nodeIDs {
		indeg[id] = 0
	}
	edgeSeen := map[string]bool{}
	for _, e := range graph.Edges {
		from := strings.TrimSpace(e.From)
		to := strings.TrimSpace(e.To)
		if !nodeIDs[from] {
			result.Issues = append(result.Issues, "边引用了不存在的起点节点："+from)
			continue
		}
		if !nodeIDs[to] {
			result.Issues = append(result.Issues, "边引用了不存在的终点节点："+to)
			continue
		}
		key := from + "->" + to
		if edgeSeen[key] {
			result.Warnings = append(result.Warnings, "存在重复边："+key)
			continue
		}
		edgeSeen[key] = true
		adj[from] = append(adj[from], to)
		indeg[to]++
		outCount[from]++
	}

	// 出边规则：end 节点不应有出边；condition 建议多分支；非终止节点应有后继。
	for _, id := range nodeOrder {
		switch {
		case terminalTypes[nodeType[id]]:
			if outCount[id] > 0 {
				result.Issues = append(result.Issues, "end 节点不应有出边："+id)
			}
		case nodeType[id] == "condition":
			if outCount[id] < 2 {
				result.Warnings = append(result.Warnings, "condition 节点建议至少两个分支出边："+id)
			}
		default:
			if outCount[id] == 0 {
				result.Warnings = append(result.Warnings, "节点没有后继，可能是断点："+id)
			}
		}
	}

	// 拓扑排序（Kahn）检测环并生成执行顺序。
	order, acyclic := topoSort(nodeOrder, adj, indeg)
	if !acyclic {
		result.Issues = append(result.Issues, "工作流存在环，无法生成执行顺序")
	} else {
		result.ExecutionOrder = order
	}

	result.Valid = len(result.Issues) == 0
	return result
}

// topoSort 基于节点出现顺序做 Kahn 拓扑排序，返回执行顺序与是否无环。
func topoSort(nodeOrder []string, adj map[string][]string, indeg map[string]int) ([]string, bool) {
	queue := make([]string, 0, len(nodeOrder))
	for _, id := range nodeOrder {
		if indeg[id] == 0 {
			queue = append(queue, id)
		}
	}
	order := make([]string, 0, len(nodeOrder))
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		for _, nb := range adj[cur] {
			indeg[nb]--
			if indeg[nb] == 0 {
				queue = append(queue, nb)
			}
		}
	}
	return order, len(order) == len(nodeOrder)
}

// 工作流状态常量。
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

// Workflow 表示一个持久化的工作流定义。
type Workflow struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"`
	Name        string        `json:"name"`
	Code        string        `json:"code"`
	Description string        `json:"description"`
	Definition  WorkflowGraph `json:"definition"`
	Status      string        `json:"status"`
	Version     int           `json:"version"`
	CreatedBy   string        `json:"created_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CreateWorkflowRequest 是创建工作流的请求体。
type CreateWorkflowRequest struct {
	Name        string        `json:"name" binding:"required"`
	Code        string        `json:"code" binding:"required"`
	Description string        `json:"description"`
	Definition  WorkflowGraph `json:"definition"`
}

// UpdateWorkflowRequest 是更新工作流的请求体，Definition 为 nil 时不更新定义。
type UpdateWorkflowRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Definition  *WorkflowGraph `json:"definition"`
}

// Normalize 归一化创建请求字段。
func (r *CreateWorkflowRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.ToLower(strings.TrimSpace(r.Code))
	r.Description = strings.TrimSpace(r.Description)
	if r.Definition.Nodes == nil {
		r.Definition.Nodes = []WorkflowNode{}
	}
	if r.Definition.Edges == nil {
		r.Definition.Edges = []WorkflowEdge{}
	}
}

// Validate 校验创建请求的基础字段（图结构由 ValidateWorkflowGraph 单独诊断）。
func (r CreateWorkflowRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if err := validateWorkflowCode(r.Code); err != nil {
		return err
	}
	if len(r.Definition.Nodes) == 0 {
		return errors.New("definition.nodes is required")
	}
	if len(r.Definition.Nodes) > maxWorkflowNodes {
		return errors.New("workflow nodes exceed the maximum limit")
	}
	return nil
}

// Normalize 归一化更新请求字段。
func (r *UpdateWorkflowRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
}

// Validate 校验更新请求字段。
func (r UpdateWorkflowRequest) Validate() error {
	if r.Name != "" && len([]rune(r.Name)) > 128 {
		return errors.New("name must be at most 128 characters")
	}
	if r.Definition != nil {
		if len(r.Definition.Nodes) == 0 {
			return errors.New("definition.nodes is required")
		}
		if len(r.Definition.Nodes) > maxWorkflowNodes {
			return errors.New("workflow nodes exceed the maximum limit")
		}
	}
	return nil
}

// validateWorkflowCode 校验工作流编码：非空、不超过 64 字符、仅限小写字母数字与 - _。
func validateWorkflowCode(code string) error {
	if code == "" {
		return errors.New("code is required")
	}
	if len(code) > 64 {
		return errors.New("code must be at most 64 characters")
	}
	for _, ch := range code {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			continue
		}
		return errors.New("code only supports lowercase letters, numbers, hyphen and underscore")
	}
	return nil
}

// marshalDefinition 将工作流图序列化为 JSON 字符串，用于写入 JSONB 列。
func marshalDefinition(graph WorkflowGraph) (string, error) {
	if graph.Nodes == nil {
		graph.Nodes = []WorkflowNode{}
	}
	if graph.Edges == nil {
		graph.Edges = []WorkflowEdge{}
	}
	b, err := json.Marshal(graph)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
