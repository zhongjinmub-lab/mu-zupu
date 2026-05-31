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
	Nodes []GenealogyNode `json:"nodes"`
	Edges []GenealogyEdge `json:"edges"`
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
