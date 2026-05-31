package agent

import (
	"errors"
	"strings"
	"time"
)

type Agent struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	Name         string         `json:"name"`
	Code         string         `json:"code"`
	Description  string         `json:"description,omitempty"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	ModelConfig  map[string]any `json:"model_config"`
	ToolPolicy   map[string]any `json:"tool_policy"`
	MemoryPolicy map[string]any `json:"memory_policy"`
	Status       string         `json:"status"`
	CreatedBy    string         `json:"created_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type GenealogyNode struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type GenealogyEdge struct {
	ID            string    `json:"id"`
	ParentAgentID string    `json:"parent_agent_id,omitempty"`
	ParentName    string    `json:"parent_name,omitempty"`
	ChildAgentID  string    `json:"child_agent_id"`
	ChildName     string    `json:"child_name"`
	RelationType  string    `json:"relation_type"`
	CreatedAt     time.Time `json:"created_at"`
}

type GenealogyGraph struct {
	Nodes   []GenealogyNode       `json:"nodes"`
	Edges   []GenealogyEdge       `json:"edges"`
	Summary GenealogyGraphSummary `json:"summary"`
}

type GenealogyGraphSummary struct {
	Nodes         int64                    `json:"nodes"`
	Edges         int64                    `json:"edges"`
	Roots         int64                    `json:"roots"`
	Isolated      int64                    `json:"isolated"`
	RelationTypes []GenealogyRelationCount `json:"relation_types"`
}

type GenealogyRelationCount struct {
	RelationType string `json:"relation_type"`
	Count        int64  `json:"count"`
}

type GenealogyGraphQuery struct {
	Q            string `json:"q"`
	RelationType string `json:"relation_type"`
}

type CreateGenealogyEdgeRequest struct {
	ParentAgentID string `json:"parent_agent_id"`
	ChildAgentID  string `json:"child_agent_id" binding:"required"`
	RelationType  string `json:"relation_type"`
}

type KnowledgeBaseBinding struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	AgentID         string         `json:"agent_id"`
	KnowledgeBaseID string         `json:"knowledge_base_id"`
	KnowledgeBase   string         `json:"knowledge_base,omitempty"`
	Status          string         `json:"status"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type Conversation struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	AgentID   string         `json:"agent_id"`
	UserID    string         `json:"user_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type Message struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	ConversationID string         `json:"conversation_id"`
	Role           string         `json:"role"`
	Content        string         `json:"content"`
	TokenUsage     map[string]any `json:"token_usage"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

type CreateAgentRequest struct {
	Name         string         `json:"name" binding:"required"`
	Code         string         `json:"code" binding:"required"`
	Description  string         `json:"description"`
	SystemPrompt string         `json:"system_prompt"`
	ModelConfig  map[string]any `json:"model_config"`
	ToolPolicy   map[string]any `json:"tool_policy"`
	MemoryPolicy map[string]any `json:"memory_policy"`
}

type UpdateAgentRequest struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	SystemPrompt string         `json:"system_prompt"`
	ModelConfig  map[string]any `json:"model_config"`
	ToolPolicy   map[string]any `json:"tool_policy"`
	MemoryPolicy map[string]any `json:"memory_policy"`
}

type BindKnowledgeBaseRequest struct {
	KnowledgeBaseID string         `json:"knowledge_base_id" binding:"required"`
	Metadata        map[string]any `json:"metadata"`
}

type TestChatRequest struct {
	Message         string  `json:"message" binding:"required"`
	KnowledgeBaseID string  `json:"knowledge_base_id"`
	TopK            int     `json:"top_k"`
	CandidateK      int     `json:"candidate_k"`
	MinScore        float64 `json:"min_score"`
	MaxTokens       int     `json:"max_tokens"`
	Temperature     float64 `json:"temperature"`
}

type TestChatReference struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
}

type TestChatResponse struct {
	ConversationID     string              `json:"conversation_id"`
	UserMessageID      string              `json:"user_message_id"`
	AssistantMessageID string              `json:"assistant_message_id"`
	AgentID            string              `json:"agent_id"`
	KnowledgeBaseID    string              `json:"knowledge_base_id"`
	Answer             string              `json:"answer"`
	References         []TestChatReference `json:"references"`
	EmbeddingModel     string              `json:"embedding_model"`
	GenerationModel    string              `json:"generation_model"`
	GenerationSource   string              `json:"generation_source"`
}

type ChatRequest struct {
	ConversationID  string  `json:"conversation_id"`
	Message         string  `json:"message" binding:"required"`
	KnowledgeBaseID string  `json:"knowledge_base_id"`
	Title           string  `json:"title"`
	TopK            int     `json:"top_k"`
	CandidateK      int     `json:"candidate_k"`
	MinScore        float64 `json:"min_score"`
	MaxTokens       int     `json:"max_tokens"`
	Temperature     float64 `json:"temperature"`
	HistoryLimit    int     `json:"history_limit"`
}

type ChatResponse struct {
	ConversationID     string              `json:"conversation_id"`
	UserMessageID      string              `json:"user_message_id"`
	AssistantMessageID string              `json:"assistant_message_id"`
	AgentID            string              `json:"agent_id"`
	KnowledgeBaseID    string              `json:"knowledge_base_id"`
	Answer             string              `json:"answer"`
	References         []TestChatReference `json:"references"`
	HistoryUsed        int                 `json:"history_used"`
	EmbeddingModel     string              `json:"embedding_model"`
	GenerationModel    string              `json:"generation_model"`
	GenerationSource   string              `json:"generation_source"`
}

type ToolSafetyPolicy struct {
	Enabled              bool             `json:"enabled"`
	DefaultAction        string           `json:"default_action"`
	PermissionRole       string           `json:"permission_role"`
	AuditAction          string           `json:"audit_action"`
	DangerConfirmation   bool             `json:"danger_confirmation"`
	AvailableTools       []ToolPolicyItem `json:"available_tools"`
	DangerousTools       []ToolPolicyItem `json:"dangerous_tools"`
	Guardrails           []string         `json:"guardrails"`
	AgentPolicyHint      string           `json:"agent_policy_hint"`
	FutureExecutionNotes []string         `json:"future_execution_notes"`
}

type ToolPolicyItem struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Category             string `json:"category"`
	Status               string `json:"status"`
	Description          string `json:"description"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

type ConversationOrchestrationPolicy struct {
	HistoryLimitDefault int      `json:"history_limit_default"`
	HistoryLimitMax     int      `json:"history_limit_max"`
	RAGEnabled          bool     `json:"rag_enabled"`
	SSEEnabled          bool     `json:"sse_enabled"`
	ToolPolicy          string   `json:"tool_policy"`
	QuotaMetric         string   `json:"quota_metric"`
	Flow                []string `json:"flow"`
	Events              []string `json:"events"`
	SafetyNotes         []string `json:"safety_notes"`
}

func (r *CreateAgentRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.ToLower(strings.TrimSpace(r.Code))
	r.Description = strings.TrimSpace(r.Description)
	r.SystemPrompt = strings.TrimSpace(r.SystemPrompt)
}

func (r CreateAgentRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Code == "" {
		return errors.New("code is required")
	}
	if len(r.Code) > 64 {
		return errors.New("code must be at most 64 characters")
	}
	for _, ch := range r.Code {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			continue
		}
		return errors.New("code only supports lowercase letters, numbers, hyphen and underscore")
	}
	return nil
}

func (r *UpdateAgentRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.SystemPrompt = strings.TrimSpace(r.SystemPrompt)
}

func (r UpdateAgentRequest) Validate() error {
	if r.Name != "" && len([]rune(r.Name)) > 128 {
		return errors.New("name must be at most 128 characters")
	}
	return nil
}

func (r *CreateGenealogyEdgeRequest) Normalize() {
	r.ParentAgentID = strings.TrimSpace(r.ParentAgentID)
	r.ChildAgentID = strings.TrimSpace(r.ChildAgentID)
	r.RelationType = strings.ToLower(strings.TrimSpace(r.RelationType))
	if r.RelationType == "" {
		r.RelationType = "fork"
	}
}

func (r CreateGenealogyEdgeRequest) Validate() error {
	if r.ChildAgentID == "" {
		return errors.New("child_agent_id is required")
	}
	if r.ParentAgentID != "" && r.ParentAgentID == r.ChildAgentID {
		return errors.New("parent_agent_id and child_agent_id cannot be same")
	}
	switch r.RelationType {
	case "fork", "inherit", "compose", "route":
		return nil
	default:
		return errors.New("relation_type must be fork, inherit, compose or route")
	}
}

func (q *GenealogyGraphQuery) Normalize() {
	q.Q = strings.TrimSpace(q.Q)
	q.RelationType = strings.ToLower(strings.TrimSpace(q.RelationType))
}

func (q GenealogyGraphQuery) Validate() error {
	if len([]rune(q.Q)) > 80 {
		return errors.New("q must be less than 80 characters")
	}
	if q.RelationType == "" {
		return nil
	}
	switch q.RelationType {
	case "fork", "inherit", "compose", "route":
		return nil
	default:
		return errors.New("relation_type must be fork, inherit, compose or route")
	}
}

func (r *BindKnowledgeBaseRequest) Normalize() {
	r.KnowledgeBaseID = strings.TrimSpace(r.KnowledgeBaseID)
}

func (r BindKnowledgeBaseRequest) Validate() error {
	if r.KnowledgeBaseID == "" {
		return errors.New("knowledge_base_id is required")
	}
	return nil
}

func (r *TestChatRequest) Normalize() {
	r.Message = strings.TrimSpace(r.Message)
	r.KnowledgeBaseID = strings.TrimSpace(r.KnowledgeBaseID)
	if r.TopK <= 0 || r.TopK > 20 {
		r.TopK = 5
	}
	if r.CandidateK < r.TopK || r.CandidateK > 100 {
		r.CandidateK = r.TopK * 5
	}
	if r.CandidateK > 100 {
		r.CandidateK = 100
	}
	if r.MinScore < 0 {
		r.MinScore = 0
	}
	if r.MaxTokens <= 0 || r.MaxTokens > 4096 {
		r.MaxTokens = 1024
	}
	if r.Temperature < 0 || r.Temperature > 2 {
		r.Temperature = 0.2
	}
}

func (r TestChatRequest) Validate() error {
	if r.Message == "" {
		return errors.New("message is required")
	}
	return nil
}

func (r *ChatRequest) Normalize() {
	r.ConversationID = strings.TrimSpace(r.ConversationID)
	r.Message = strings.TrimSpace(r.Message)
	r.KnowledgeBaseID = strings.TrimSpace(r.KnowledgeBaseID)
	r.Title = strings.TrimSpace(r.Title)
	if r.TopK <= 0 || r.TopK > 20 {
		r.TopK = 5
	}
	if r.CandidateK < r.TopK || r.CandidateK > 100 {
		r.CandidateK = r.TopK * 5
	}
	if r.CandidateK > 100 {
		r.CandidateK = 100
	}
	if r.MinScore < 0 {
		r.MinScore = 0
	}
	if r.MaxTokens <= 0 || r.MaxTokens > 4096 {
		r.MaxTokens = 1024
	}
	if r.Temperature < 0 || r.Temperature > 2 {
		r.Temperature = 0.2
	}
	if r.HistoryLimit <= 0 || r.HistoryLimit > 50 {
		r.HistoryLimit = 20
	}
}

func (r ChatRequest) Validate() error {
	if r.Message == "" {
		return errors.New("message is required")
	}
	return nil
}

func DefaultToolSafetyPolicy() ToolSafetyPolicy {
	return ToolSafetyPolicy{
		Enabled:            false,
		DefaultAction:      "deny",
		PermissionRole:     "tenant_writer",
		AuditAction:        "agent.tool.call",
		DangerConfirmation: true,
		AvailableTools: []ToolPolicyItem{
			{
				Code:                 "kb_search",
				Name:                 "知识库检索",
				Category:             "read",
				Status:               "planned",
				Description:          "只读检索当前租户已绑定知识库内容。",
				RequiresConfirmation: false,
			},
			{
				Code:                 "file_lookup",
				Name:                 "文件资料查询",
				Category:             "read",
				Status:               "planned",
				Description:          "只读查询当前租户文件和文档元数据。",
				RequiresConfirmation: false,
			},
		},
		DangerousTools: []ToolPolicyItem{
			{
				Code:                 "kb_mutation",
				Name:                 "知识库写入",
				Category:             "write",
				Status:               "blocked",
				Description:          "会创建、更新或归档知识库资料，首版默认禁止。",
				RequiresConfirmation: true,
			},
			{
				Code:                 "billing_operation",
				Name:                 "账单授权操作",
				Category:             "admin",
				Status:               "blocked",
				Description:          "涉及订单、支付、License 或 Webhook，必须由管理员显式操作。",
				RequiresConfirmation: true,
			},
		},
		Guardrails: []string{
			"工具调用默认关闭，Agent 只能执行 RAG 检索和对话生成。",
			"后续启用工具时必须校验当前租户、当前用户角色和 Agent 绑定范围。",
			"危险工具必须先返回待确认状态，不能由模型直接执行。",
			"所有工具调用必须写入审计日志，记录 request_id、agent_id、tool_code、入参摘要和结果状态。",
		},
		AgentPolicyHint: "tool_policy 为空时按 deny 处理；显式启用前不允许模型自主调用外部工具。",
		FutureExecutionNotes: []string{
			"真实工具执行模块上线后复用 tenant writer/admin 权限中间件。",
			"工具入参和响应只保存摘要，敏感字段需要脱敏。",
			"失败、拒绝和人工确认都需要进入审计日志。",
		},
	}
}

func DefaultConversationOrchestrationPolicy() ConversationOrchestrationPolicy {
	return ConversationOrchestrationPolicy{
		HistoryLimitDefault: 20,
		HistoryLimitMax:     50,
		RAGEnabled:          true,
		SSEEnabled:          true,
		ToolPolicy:          "deny",
		QuotaMetric:         "agent_messages",
		Flow: []string{
			"校验租户、用户角色、Agent 和会话归属",
			"选择显式知识库或 Agent 第一个 active 绑定知识库",
			"读取最近历史消息并过滤 tool 角色消息",
			"执行混合检索并生成引用片段",
			"调用生成模型产出回答",
			"写入 user 和 assistant 消息",
			"记录 Agent 消息用量并触发会话完成 Webhook",
		},
		Events: []string{"start", "reference", "delta", "done", "error"},
		SafetyNotes: []string{
			"工具调用首版默认拒绝，不进入模型自主执行链路。",
			"超出套餐额度时返回 402，并保留中文错误摘要。",
			"SSE 兼容非流式模型，会按段输出最终回答。",
		},
	}
}
