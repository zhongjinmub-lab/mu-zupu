package agent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) Create(ctx context.Context, tenantID, userID string, req CreateAgentRequest) (Agent, error) {
	modelConfig, err := jsonObject(req.ModelConfig)
	if err != nil {
		return Agent{}, err
	}
	toolPolicy, err := jsonObject(req.ToolPolicy)
	if err != nil {
		return Agent{}, err
	}
	memoryPolicy, err := jsonObject(req.MemoryPolicy)
	if err != nil {
		return Agent{}, err
	}
	const q = `
INSERT INTO agents(
    tenant_id, name, code, description, system_prompt, model_config, tool_policy, memory_policy, created_by
) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6::jsonb, $7::jsonb, $8::jsonb, NULLIF($9, '')::uuid)
RETURNING id::text, tenant_id::text, name, code, COALESCE(description, ''), COALESCE(system_prompt, ''),
          model_config, tool_policy, memory_policy, status, COALESCE(created_by::text, ''), created_at, updated_at`
	var item Agent
	err = r.DB.QueryRow(ctx, q, tenantID, req.Name, req.Code, req.Description, req.SystemPrompt, modelConfig, toolPolicy, memoryPolicy, userID).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Code, &item.Description, &item.SystemPrompt,
		&item.ModelConfig, &item.ToolPolicy, &item.MemoryPolicy, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (r Repository) List(ctx context.Context, tenantID string) ([]Agent, error) {
	const q = `
SELECT id::text, tenant_id::text, name, code, COALESCE(description, ''), COALESCE(system_prompt, ''),
       model_config, tool_policy, memory_policy, status, COALESCE(created_by::text, ''), created_at, updated_at
FROM agents
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Agent, 0)
	for rows.Next() {
		item, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) GenealogyGraph(ctx context.Context, tenantID string) (GenealogyGraph, error) {
	const nodeQ = `
SELECT id::text, name, code, COALESCE(description, ''), status, created_at
FROM agents
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC`
	nodeRows, err := r.DB.Query(ctx, nodeQ, tenantID)
	if err != nil {
		return GenealogyGraph{}, err
	}
	defer nodeRows.Close()
	nodes := make([]GenealogyNode, 0)
	for nodeRows.Next() {
		var item GenealogyNode
		if err := nodeRows.Scan(&item.ID, &item.Name, &item.Code, &item.Description, &item.Status, &item.CreatedAt); err != nil {
			return GenealogyGraph{}, err
		}
		nodes = append(nodes, item)
	}
	if err := nodeRows.Err(); err != nil {
		return GenealogyGraph{}, err
	}

	const edgeQ = `
SELECT g.id::text,
       COALESCE(g.parent_agent_id::text, ''),
       COALESCE(parent.name, ''),
       g.child_agent_id::text,
       child.name,
       g.relation_type,
       g.created_at
FROM agent_genealogy g
JOIN agents child ON child.id = g.child_agent_id
LEFT JOIN agents parent ON parent.id = g.parent_agent_id
WHERE g.tenant_id = $1
  AND child.tenant_id = $1
  AND child.deleted_at IS NULL
  AND (parent.id IS NULL OR (parent.tenant_id = $1 AND parent.deleted_at IS NULL))
ORDER BY g.created_at ASC`
	edgeRows, err := r.DB.Query(ctx, edgeQ, tenantID)
	if err != nil {
		return GenealogyGraph{}, err
	}
	defer edgeRows.Close()
	edges := make([]GenealogyEdge, 0)
	for edgeRows.Next() {
		var item GenealogyEdge
		if err := edgeRows.Scan(&item.ID, &item.ParentAgentID, &item.ParentName, &item.ChildAgentID, &item.ChildName, &item.RelationType, &item.CreatedAt); err != nil {
			return GenealogyGraph{}, err
		}
		edges = append(edges, item)
	}
	if err := edgeRows.Err(); err != nil {
		return GenealogyGraph{}, err
	}

	return GenealogyGraph{Nodes: nodes, Edges: edges}, nil
}

func (r Repository) Get(ctx context.Context, tenantID, agentID string) (Agent, error) {
	const q = `
SELECT id::text, tenant_id::text, name, code, COALESCE(description, ''), COALESCE(system_prompt, ''),
       model_config, tool_policy, memory_policy, status, COALESCE(created_by::text, ''), created_at, updated_at
FROM agents
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	rows, err := r.DB.Query(ctx, q, agentID, tenantID)
	if err != nil {
		return Agent{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Agent{}, rows.Err()
		}
		return Agent{}, ErrAgentNotFound
	}
	item, err := scanAgent(rows)
	if err != nil {
		return Agent{}, err
	}
	return item, rows.Err()
}

func (r Repository) Update(ctx context.Context, tenantID, agentID string, req UpdateAgentRequest) (Agent, error) {
	current, err := r.Get(ctx, tenantID, agentID)
	if err != nil {
		return Agent{}, err
	}
	if req.Name == "" {
		req.Name = current.Name
	}
	if req.Description == "" {
		req.Description = current.Description
	}
	if req.SystemPrompt == "" {
		req.SystemPrompt = current.SystemPrompt
	}
	if req.ModelConfig == nil {
		req.ModelConfig = current.ModelConfig
	}
	if req.ToolPolicy == nil {
		req.ToolPolicy = current.ToolPolicy
	}
	if req.MemoryPolicy == nil {
		req.MemoryPolicy = current.MemoryPolicy
	}
	modelConfig, err := jsonObject(req.ModelConfig)
	if err != nil {
		return Agent{}, err
	}
	toolPolicy, err := jsonObject(req.ToolPolicy)
	if err != nil {
		return Agent{}, err
	}
	memoryPolicy, err := jsonObject(req.MemoryPolicy)
	if err != nil {
		return Agent{}, err
	}
	const q = `
UPDATE agents
SET name = $3,
    description = NULLIF($4, ''),
    system_prompt = NULLIF($5, ''),
    model_config = $6::jsonb,
    tool_policy = $7::jsonb,
    memory_policy = $8::jsonb,
    status = CASE WHEN status = 'published' THEN 'draft' ELSE status END,
    updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, name, code, COALESCE(description, ''), COALESCE(system_prompt, ''),
          model_config, tool_policy, memory_policy, status, COALESCE(created_by::text, ''), created_at, updated_at`
	rows, err := r.DB.Query(ctx, q, agentID, tenantID, req.Name, req.Description, req.SystemPrompt, modelConfig, toolPolicy, memoryPolicy)
	if err != nil {
		return Agent{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Agent{}, ErrAgentNotFound
	}
	item, err := scanAgent(rows)
	if err != nil {
		return Agent{}, err
	}
	return item, rows.Err()
}

func (r Repository) SetStatus(ctx context.Context, tenantID, agentID, status string) (Agent, error) {
	const q = `
UPDATE agents
SET status = $3, updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, name, code, COALESCE(description, ''), COALESCE(system_prompt, ''),
          model_config, tool_policy, memory_policy, status, COALESCE(created_by::text, ''), created_at, updated_at`
	rows, err := r.DB.Query(ctx, q, agentID, tenantID, status)
	if err != nil {
		return Agent{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Agent{}, ErrAgentNotFound
	}
	item, err := scanAgent(rows)
	if err != nil {
		return Agent{}, err
	}
	return item, rows.Err()
}

func (r Repository) Archive(ctx context.Context, tenantID, agentID string) error {
	const q = `
UPDATE agents
SET status = 'archived', deleted_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	tag, err := r.DB.Exec(ctx, q, agentID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentNotFound
	}
	return nil
}

func (r Repository) BindKnowledgeBase(ctx context.Context, tenantID, agentID string, req BindKnowledgeBaseRequest) (KnowledgeBaseBinding, error) {
	if _, err := r.Get(ctx, tenantID, agentID); err != nil {
		return KnowledgeBaseBinding{}, err
	}
	metadata, err := jsonObject(req.Metadata)
	if err != nil {
		return KnowledgeBaseBinding{}, err
	}
	const q = `
INSERT INTO agent_knowledge_bases(tenant_id, agent_id, knowledge_base_id, metadata)
VALUES ($1, $2, $3, $4::jsonb)
ON CONFLICT (tenant_id, agent_id, knowledge_base_id)
DO UPDATE SET status = 'active', metadata = EXCLUDED.metadata, deleted_at = NULL, updated_at = now()
RETURNING id::text, tenant_id::text, agent_id::text, knowledge_base_id::text, status, metadata, created_at, updated_at`
	var item KnowledgeBaseBinding
	err = r.DB.QueryRow(ctx, q, tenantID, agentID, req.KnowledgeBaseID, metadata).Scan(
		&item.ID, &item.TenantID, &item.AgentID, &item.KnowledgeBaseID, &item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (r Repository) ListKnowledgeBases(ctx context.Context, tenantID, agentID string) ([]KnowledgeBaseBinding, error) {
	if _, err := r.Get(ctx, tenantID, agentID); err != nil {
		return nil, err
	}
	const q = `
SELECT b.id::text, b.tenant_id::text, b.agent_id::text, b.knowledge_base_id::text, kb.name,
       b.status, b.metadata, b.created_at, b.updated_at
FROM agent_knowledge_bases b
JOIN knowledge_bases kb ON kb.id = b.knowledge_base_id
WHERE b.tenant_id = $1
  AND b.agent_id = $2
  AND b.deleted_at IS NULL
  AND kb.deleted_at IS NULL
ORDER BY b.created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBaseBinding, 0)
	for rows.Next() {
		var item KnowledgeBaseBinding
		if err := rows.Scan(&item.ID, &item.TenantID, &item.AgentID, &item.KnowledgeBaseID, &item.KnowledgeBase, &item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) ResolveKnowledgeBase(ctx context.Context, tenantID, agentID, kbID string) (KnowledgeBaseBinding, error) {
	if kbID != "" {
		const q = `
SELECT b.id::text, b.tenant_id::text, b.agent_id::text, b.knowledge_base_id::text, kb.name,
       b.status, b.metadata, b.created_at, b.updated_at
FROM agent_knowledge_bases b
JOIN knowledge_bases kb ON kb.id = b.knowledge_base_id
WHERE b.tenant_id = $1
  AND b.agent_id = $2
  AND b.knowledge_base_id = $3
  AND b.status = 'active'
  AND b.deleted_at IS NULL
  AND kb.status = 'active'
  AND kb.deleted_at IS NULL`
		var item KnowledgeBaseBinding
		err := r.DB.QueryRow(ctx, q, tenantID, agentID, kbID).Scan(
			&item.ID, &item.TenantID, &item.AgentID, &item.KnowledgeBaseID, &item.KnowledgeBase,
			&item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == pgx.ErrNoRows {
			return KnowledgeBaseBinding{}, ErrBindingNotFound
		}
		return item, err
	}
	const q = `
SELECT b.id::text, b.tenant_id::text, b.agent_id::text, b.knowledge_base_id::text, kb.name,
       b.status, b.metadata, b.created_at, b.updated_at
FROM agent_knowledge_bases b
JOIN knowledge_bases kb ON kb.id = b.knowledge_base_id
WHERE b.tenant_id = $1
  AND b.agent_id = $2
  AND b.status = 'active'
  AND b.deleted_at IS NULL
  AND kb.status = 'active'
  AND kb.deleted_at IS NULL
ORDER BY b.created_at ASC
LIMIT 1`
	var item KnowledgeBaseBinding
	err := r.DB.QueryRow(ctx, q, tenantID, agentID).Scan(
		&item.ID, &item.TenantID, &item.AgentID, &item.KnowledgeBaseID, &item.KnowledgeBase,
		&item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return KnowledgeBaseBinding{}, ErrBindingNotFound
	}
	return item, err
}

func (r Repository) CreateConversation(ctx context.Context, tenantID, agentID, userID, title string, metadata map[string]any) (string, error) {
	meta, err := jsonObject(metadata)
	if err != nil {
		return "", err
	}
	const q = `
INSERT INTO conversations(tenant_id, agent_id, user_id, title, metadata)
VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, ''), $5::jsonb)
RETURNING id::text`
	var id string
	err = r.DB.QueryRow(ctx, q, tenantID, agentID, userID, title, meta).Scan(&id)
	return id, err
}

func (r Repository) ListConversations(ctx context.Context, tenantID, agentID string, limit int) ([]Conversation, error) {
	if _, err := r.Get(ctx, tenantID, agentID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, COALESCE(agent_id::text, ''), COALESCE(user_id::text, ''),
       COALESCE(title, ''), status, metadata, created_at, updated_at
FROM conversations
WHERE tenant_id = $1
  AND agent_id = $2
  AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $3`
	rows, err := r.DB.Query(ctx, q, tenantID, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Conversation, 0)
	for rows.Next() {
		item, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) GetConversation(ctx context.Context, tenantID, agentID, conversationID string) (Conversation, error) {
	const q = `
SELECT id::text, tenant_id::text, COALESCE(agent_id::text, ''), COALESCE(user_id::text, ''),
       COALESCE(title, ''), status, metadata, created_at, updated_at
FROM conversations
WHERE id = $1
  AND tenant_id = $2
  AND agent_id = $3
  AND deleted_at IS NULL`
	rows, err := r.DB.Query(ctx, q, conversationID, tenantID, agentID)
	if err != nil {
		return Conversation{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Conversation{}, rows.Err()
		}
		return Conversation{}, ErrConversationNotFound
	}
	item, err := scanConversation(rows)
	if err != nil {
		return Conversation{}, err
	}
	return item, rows.Err()
}

func (r Repository) CreateMessage(ctx context.Context, tenantID, conversationID, role, content string, metadata map[string]any) (string, error) {
	meta, err := jsonObject(metadata)
	if err != nil {
		return "", err
	}
	const q = `
INSERT INTO messages(tenant_id, conversation_id, role, content, metadata)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id::text`
	var id string
	err = r.DB.QueryRow(ctx, q, tenantID, conversationID, role, content, meta).Scan(&id)
	return id, err
}

func (r Repository) CreateMessageItem(ctx context.Context, tenantID, conversationID, role, content string, metadata map[string]any) (Message, error) {
	meta, err := jsonObject(metadata)
	if err != nil {
		return Message{}, err
	}
	const q = `
INSERT INTO messages(tenant_id, conversation_id, role, content, metadata)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id::text, tenant_id::text, conversation_id::text, role, content, token_usage, metadata, created_at`
	var item Message
	err = r.DB.QueryRow(ctx, q, tenantID, conversationID, role, content, meta).Scan(
		&item.ID, &item.TenantID, &item.ConversationID, &item.Role, &item.Content, &item.TokenUsage, &item.Metadata, &item.CreatedAt,
	)
	if err != nil {
		return Message{}, err
	}
	if err := r.touchConversation(ctx, tenantID, conversationID); err != nil {
		return Message{}, err
	}
	return item, nil
}

func (r Repository) ListMessages(ctx context.Context, tenantID, agentID, conversationID string, limit int) ([]Message, error) {
	if _, err := r.GetConversation(ctx, tenantID, agentID, conversationID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	const q = `
SELECT id::text, tenant_id::text, conversation_id::text, role, content, token_usage, metadata, created_at
FROM (
    SELECT id, tenant_id, conversation_id, role, content, token_usage, metadata, created_at
    FROM messages
    WHERE tenant_id = $1
      AND conversation_id = $2
    ORDER BY created_at DESC
    LIMIT $3
) m
ORDER BY created_at ASC`
	rows, err := r.DB.Query(ctx, q, tenantID, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0)
	for rows.Next() {
		item, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) touchConversation(ctx context.Context, tenantID, conversationID string) error {
	_, err := r.DB.Exec(ctx, `UPDATE conversations SET updated_at = now() WHERE id = $1 AND tenant_id = $2`, conversationID, tenantID)
	return err
}

func (r Repository) UnbindKnowledgeBase(ctx context.Context, tenantID, agentID, kbID string) error {
	const q = `
UPDATE agent_knowledge_bases
SET status = 'disabled', deleted_at = now(), updated_at = now()
WHERE tenant_id = $1 AND agent_id = $2 AND knowledge_base_id = $3 AND deleted_at IS NULL`
	tag, err := r.DB.Exec(ctx, q, tenantID, agentID, kbID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBindingNotFound
	}
	return nil
}

func scanAgent(rows pgx.Rows) (Agent, error) {
	var item Agent
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Code, &item.Description, &item.SystemPrompt,
		&item.ModelConfig, &item.ToolPolicy, &item.MemoryPolicy, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanConversation(rows pgx.Rows) (Conversation, error) {
	var item Conversation
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.AgentID, &item.UserID,
		&item.Title, &item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanMessage(rows pgx.Rows) (Message, error) {
	var item Message
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.ConversationID, &item.Role,
		&item.Content, &item.TokenUsage, &item.Metadata, &item.CreatedAt,
	)
	return item, err
}

func jsonObject(v map[string]any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}

var (
	ErrAgentNotFound        = errors.New("agent not found")
	ErrBindingNotFound      = errors.New("agent knowledge base binding not found")
	ErrConversationNotFound = errors.New("conversation not found")
)
