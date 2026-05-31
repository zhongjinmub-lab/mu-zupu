package agent

import (
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
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

type ToolCatalogItem struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	Category             string   `json:"category"`
	Status               string   `json:"status"`
	Description          string   `json:"description"`
	Scope                string   `json:"scope"`
	PermissionRole       string   `json:"permission_role"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	AuditAction          string   `json:"audit_action"`
	SupportedActions     []string `json:"supported_actions"`
	OperationalNotes     []string `json:"operational_notes"`
}

type ToolTestRequest struct {
	Input map[string]any `json:"input"`
}

type ToolTestResult struct {
	ToolCode             string   `json:"tool_code"`
	ToolName             string   `json:"tool_name"`
	Status               string   `json:"status"`
	Allowed              bool     `json:"allowed"`
	DryRun               bool     `json:"dry_run"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	Message              string   `json:"message"`
	InputSummary         string   `json:"input_summary"`
	NextSteps            []string `json:"next_steps"`
	AuditAction          string   `json:"audit_action"`
}

type ToolCallLog struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	AgentID        string         `json:"agent_id,omitempty"`
	ConversationID string         `json:"conversation_id,omitempty"`
	ToolName       string         `json:"tool_name"`
	Input          map[string]any `json:"input"`
	Output         map[string]any `json:"output"`
	Status         string         `json:"status"`
	CostMS         int            `json:"cost_ms"`
	CreatedAt      time.Time      `json:"created_at"`
}

type ToolCallLogQuery struct {
	TenantID  string
	AgentID   string
	ToolName  string
	Status    string
	From      time.Time
	To        time.Time
	Cursor    ToolCallLogCursor
	CursorRaw string
	Limit     int
}

type ToolCallLogCursor struct {
	CreatedAt time.Time
	ID        string
}

func (q *ToolCallLogQuery) Normalize() {
	q.TenantID = strings.TrimSpace(q.TenantID)
	q.AgentID = strings.TrimSpace(q.AgentID)
	q.ToolName = strings.TrimSpace(q.ToolName)
	q.Status = strings.TrimSpace(q.Status)
	q.CursorRaw = strings.TrimSpace(q.CursorRaw)
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
}

func (q *ToolCallLogQuery) Validate() error {
	if q.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if q.AgentID != "" {
		if _, err := uuid.Parse(q.AgentID); err != nil {
			return errors.New("agent_id must be a valid uuid")
		}
	}
	if !q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To) {
		return errors.New("from must be before or equal to to")
	}
	if q.CursorRaw != "" {
		cursor, err := DecodeToolCallLogCursor(q.CursorRaw)
		if err != nil {
			return err
		}
		q.Cursor = cursor
	}
	return nil
}

func ParseToolCallLogTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

func EncodeToolCallLogCursor(item ToolCallLog) string {
	if item.ID == "" || item.CreatedAt.IsZero() {
		return ""
	}
	payload := item.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + item.ID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeToolCallLogCursor(raw string) (ToolCallLogCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ToolCallLogCursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return ToolCallLogCursor{}, errors.New("cursor is invalid")
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return ToolCallLogCursor{}, errors.New("cursor is invalid")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return ToolCallLogCursor{}, errors.New("cursor is invalid")
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return ToolCallLogCursor{}, errors.New("cursor is invalid")
	}
	return ToolCallLogCursor{CreatedAt: createdAt, ID: parts[1]}, nil
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

func DefaultToolCatalog() []ToolCatalogItem {
	policy := DefaultToolSafetyPolicy()
	items := make([]ToolCatalogItem, 0, len(policy.AvailableTools)+len(policy.DangerousTools))
	for _, tool := range policy.AvailableTools {
		items = append(items, toolCatalogItemFromPolicy(tool, policy))
	}
	for _, tool := range policy.DangerousTools {
		items = append(items, toolCatalogItemFromPolicy(tool, policy))
	}
	return items
}

func FindToolCatalogItem(code string) (ToolCatalogItem, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, item := range DefaultToolCatalog() {
		if item.Code == code {
			return item, true
		}
	}
	return ToolCatalogItem{}, false
}

func BuildToolTestResult(item ToolCatalogItem, req ToolTestRequest) ToolTestResult {
	summary := summarizeToolInput(req.Input)
	if item.Status == "blocked" || item.RequiresConfirmation {
		return ToolTestResult{
			ToolCode:             item.Code,
			ToolName:             item.Name,
			Status:               "blocked",
			Allowed:              false,
			DryRun:               true,
			RequiresConfirmation: item.RequiresConfirmation,
			Message:              "工具测试已被安全策略阻断，当前版本不会执行真实外部动作。",
			InputSummary:         summary,
			NextSteps: []string{
				"由租户管理员在管理台完成对应业务操作。",
				"后续启用真实工具执行前，需要补充人工确认和审计日志。",
			},
			AuditAction: item.AuditAction,
		}
	}
	return ToolTestResult{
		ToolCode:             item.Code,
		ToolName:             item.Name,
		Status:               "dry_run_ok",
		Allowed:              true,
		DryRun:               true,
		RequiresConfirmation: item.RequiresConfirmation,
		Message:              "工具测试通过安全预检；当前版本仅返回 dry-run 摘要，不执行真实查询或写入。",
		InputSummary:         summary,
		NextSteps: []string{
			"可在 Agent 对话中继续使用 RAG 和知识库问答能力。",
			"真实工具执行器上线后，将复用当前权限、审计和脱敏要求。",
		},
		AuditAction: item.AuditAction,
	}
}

func toolCatalogItemFromPolicy(tool ToolPolicyItem, policy ToolSafetyPolicy) ToolCatalogItem {
	permissionRole := policy.PermissionRole
	if tool.Category == "admin" {
		permissionRole = "tenant_admin"
	}
	scope := "当前租户"
	if tool.Category == "read" {
		scope = "当前租户已授权资源"
	}
	return ToolCatalogItem{
		Code:                 tool.Code,
		Name:                 tool.Name,
		Category:             tool.Category,
		Status:               tool.Status,
		Description:          tool.Description,
		Scope:                scope,
		PermissionRole:       permissionRole,
		RequiresConfirmation: tool.RequiresConfirmation,
		AuditAction:          policy.AuditAction,
		SupportedActions:     supportedToolActions(tool),
		OperationalNotes:     toolOperationalNotes(tool),
	}
}

func supportedToolActions(tool ToolPolicyItem) []string {
	switch tool.Code {
	case "kb_search":
		return []string{"检索知识库片段", "返回引用摘要"}
	case "file_lookup":
		return []string{"查询文件元数据", "定位已归档文档"}
	case "kb_mutation":
		return []string{"创建文档", "更新 Chunk", "归档文档"}
	case "billing_operation":
		return []string{"查询订单状态", "关闭支付单", "触发授权变更"}
	default:
		return []string{"安全预检"}
	}
}

func toolOperationalNotes(tool ToolPolicyItem) []string {
	if tool.Status == "blocked" {
		return []string{
			"当前版本默认阻断，不允许模型直接执行。",
			"启用前必须加入人工确认、角色校验和审计记录。",
		}
	}
	return []string{
		"当前版本仅开放 dry-run 测试，不执行真实外部动作。",
		"后续接入真实执行器时必须校验 Agent 绑定范围。",
	}
}

func summarizeToolInput(input map[string]any) string {
	if len(input) == 0 {
		return "未提供测试参数"
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "测试参数为空"
	}
	if len(keys) > 8 {
		keys = keys[:8]
		return "已接收参数：" + strings.Join(keys, "、") + " 等"
	}
	return "已接收参数：" + strings.Join(keys, "、")
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

type MCPGatewayPolicy struct {
	Enabled              bool            `json:"enabled"`
	DefaultAction        string          `json:"default_action"`
	PermissionRole       string          `json:"permission_role"`
	AuditAction          string          `json:"audit_action"`
	DangerConfirmation   bool            `json:"danger_confirmation"`
	TransportSupport     []string        `json:"transport_support"`
	AvailableServers     []MCPServerItem `json:"available_servers"`
	DangerousServers     []MCPServerItem `json:"dangerous_servers"`
	Guardrails           []string        `json:"guardrails"`
	AgentPolicyHint      string          `json:"agent_policy_hint"`
	FutureExecutionNotes []string        `json:"future_execution_notes"`
}

type MCPServerItem struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	Transport            string   `json:"transport"`
	Category             string   `json:"category"`
	Status               string   `json:"status"`
	Description          string   `json:"description"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	Capabilities         []string `json:"capabilities"`
}

type MCPCatalogItem struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	Transport            string   `json:"transport"`
	Category             string   `json:"category"`
	Status               string   `json:"status"`
	Description          string   `json:"description"`
	Scope                string   `json:"scope"`
	PermissionRole       string   `json:"permission_role"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	AuditAction          string   `json:"audit_action"`
	Capabilities         []string `json:"capabilities"`
	OperationalNotes     []string `json:"operational_notes"`
}

type MCPTestRequest struct {
	Input map[string]any `json:"input"`
}

type MCPTestResult struct {
	ServerCode           string   `json:"server_code"`
	ServerName           string   `json:"server_name"`
	Transport            string   `json:"transport"`
	Status               string   `json:"status"`
	Allowed              bool     `json:"allowed"`
	DryRun               bool     `json:"dry_run"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	Message              string   `json:"message"`
	InputSummary         string   `json:"input_summary"`
	NextSteps            []string `json:"next_steps"`
	AuditAction          string   `json:"audit_action"`
}

func DefaultMCPGatewayPolicy() MCPGatewayPolicy {
	return MCPGatewayPolicy{
		Enabled:            false,
		DefaultAction:      "deny",
		PermissionRole:     "tenant_writer",
		AuditAction:        "agent.mcp.call",
		DangerConfirmation: true,
		TransportSupport:   []string{"stdio", "sse", "http"},
		AvailableServers: []MCPServerItem{
			{
				Code:                 "kb_resource",
				Name:                 "知识库资源服务",
				Transport:            "sse",
				Category:             "read",
				Status:               "planned",
				Description:          "以 MCP resources 暴露当前租户已授权知识库的只读片段。",
				RequiresConfirmation: false,
				Capabilities:         []string{"resources/list", "resources/read"},
			},
			{
				Code:                 "agent_catalog",
				Name:                 "Agent 工具目录服务",
				Transport:            "http",
				Category:             "read",
				Status:               "planned",
				Description:          "以 MCP tools 暴露当前租户 Agent 工具目录，仅支持 dry-run 调用。",
				RequiresConfirmation: false,
				Capabilities:         []string{"tools/list", "tools/call(dry-run)"},
			},
		},
		DangerousServers: []MCPServerItem{
			{
				Code:                 "external_http",
				Name:                 "外部 HTTP MCP 服务",
				Transport:            "http",
				Category:             "write",
				Status:               "blocked",
				Description:          "连接租户自定义的外部 MCP Server，首版默认禁止，防止 SSRF 和数据外泄。",
				RequiresConfirmation: true,
				Capabilities:         []string{"tools/call", "resources/read"},
			},
			{
				Code:                 "local_stdio",
				Name:                 "本地命令 MCP 服务",
				Transport:            "stdio",
				Category:             "admin",
				Status:               "blocked",
				Description:          "通过 stdio 启动本地进程执行命令，风险极高，默认阻断。",
				RequiresConfirmation: true,
				Capabilities:         []string{"tools/call"},
			},
		},
		Guardrails: []string{
			"MCP 网关默认关闭，未显式启用前不允许连接任何 MCP Server。",
			"启用 MCP Server 时必须校验当前租户、当前用户角色和 Agent 绑定范围。",
			"外部和本地命令类 MCP Server 必须先返回待确认状态，不能由模型直接连接。",
			"所有 MCP 调用必须写入审计日志，记录 request_id、agent_id、server_code、入参摘要和结果状态。",
		},
		AgentPolicyHint: "mcp_policy 为空时按 deny 处理；显式启用前不允许模型自主连接外部 MCP Server。",
		FutureExecutionNotes: []string{
			"真实 MCP 网关上线后复用 tenant writer/admin 权限中间件。",
			"MCP 入参和响应只保存摘要，敏感字段需要脱敏。",
			"连接失败、拒绝和人工确认都需要进入审计日志。",
			"外部 MCP Server 需要配置出站白名单，防止 SSRF 与数据外泄。",
		},
	}
}

func DefaultMCPServerCatalog() []MCPCatalogItem {
	policy := DefaultMCPGatewayPolicy()
	items := make([]MCPCatalogItem, 0, len(policy.AvailableServers)+len(policy.DangerousServers))
	for _, server := range policy.AvailableServers {
		items = append(items, mcpCatalogItemFromPolicy(server, policy))
	}
	for _, server := range policy.DangerousServers {
		items = append(items, mcpCatalogItemFromPolicy(server, policy))
	}
	return items
}

func FindMCPCatalogItem(code string) (MCPCatalogItem, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, item := range DefaultMCPServerCatalog() {
		if item.Code == code {
			return item, true
		}
	}
	return MCPCatalogItem{}, false
}

func BuildMCPTestResult(item MCPCatalogItem, req MCPTestRequest) MCPTestResult {
	summary := summarizeToolInput(req.Input)
	if item.Status == "blocked" || item.RequiresConfirmation {
		return MCPTestResult{
			ServerCode:           item.Code,
			ServerName:           item.Name,
			Transport:            item.Transport,
			Status:               "blocked",
			Allowed:              false,
			DryRun:               true,
			RequiresConfirmation: item.RequiresConfirmation,
			Message:              "MCP 连通性测试已被安全策略阻断，当前版本不会建立真实连接或执行外部动作。",
			InputSummary:         summary,
			NextSteps: []string{
				"由租户管理员配置出站白名单并完成人工确认后再启用。",
				"后续启用真实 MCP 网关前，需要补充人工确认和审计日志。",
			},
			AuditAction: item.AuditAction,
		}
	}
	return MCPTestResult{
		ServerCode:           item.Code,
		ServerName:           item.Name,
		Transport:            item.Transport,
		Status:               "dry_run_ok",
		Allowed:              true,
		DryRun:               true,
		RequiresConfirmation: item.RequiresConfirmation,
		Message:              "MCP 连通性测试通过安全预检；当前版本仅返回 dry-run 摘要，不建立真实连接。",
		InputSummary:         summary,
		NextSteps: []string{
			"可继续在 Agent 对话中使用 RAG 和知识库问答能力。",
			"真实 MCP 网关上线后，将复用当前权限、审计和脱敏要求。",
		},
		AuditAction: item.AuditAction,
	}
}

func mcpCatalogItemFromPolicy(server MCPServerItem, policy MCPGatewayPolicy) MCPCatalogItem {
	permissionRole := policy.PermissionRole
	if server.Category == "admin" {
		permissionRole = "tenant_admin"
	}
	scope := "当前租户"
	if server.Category == "read" {
		scope = "当前租户已授权资源"
	}
	return MCPCatalogItem{
		Code:                 server.Code,
		Name:                 server.Name,
		Transport:            server.Transport,
		Category:             server.Category,
		Status:               server.Status,
		Description:          server.Description,
		Scope:                scope,
		PermissionRole:       permissionRole,
		RequiresConfirmation: server.RequiresConfirmation,
		AuditAction:          policy.AuditAction,
		Capabilities:         server.Capabilities,
		OperationalNotes:     mcpOperationalNotes(server),
	}
}

func mcpOperationalNotes(server MCPServerItem) []string {
	if server.Status == "blocked" {
		return []string{
			"当前版本默认阻断，不允许模型直接连接。",
			"启用前必须加入出站白名单、人工确认、角色校验和审计记录。",
		}
	}
	return []string{
		"当前版本仅开放 dry-run 测试，不建立真实连接。",
		"后续接入真实 MCP 网关时必须校验 Agent 绑定范围。",
	}
}
