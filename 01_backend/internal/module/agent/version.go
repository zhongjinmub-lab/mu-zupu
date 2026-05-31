package agent

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

// ========== 类型定义 ==========

// AgentVersion 智能体版本快照
type AgentVersion struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	AgentID         string         `json:"agent_id"`
	VersionNo       string         `json:"version_no"`
	Prompt          string         `json:"prompt,omitempty"`
	ModelConfig     map[string]any `json:"model_config"`
	ToolConfig      map[string]any `json:"tool_config"`
	KnowledgeConfig map[string]any `json:"knowledge_config"`
	Channel         string         `json:"channel"`
	Status          string         `json:"status"`
	PublishNote     string         `json:"publish_note,omitempty"`
	CreatedBy       string         `json:"created_by,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
}

// CreateVersionRequest 创建版本请求
type CreateVersionRequest struct {
	VersionNo       string         `json:"version_no" binding:"required"`
	Prompt          string         `json:"prompt"`
	ModelConfig     map[string]any `json:"model_config"`
	ToolConfig      map[string]any `json:"tool_config"`
	KnowledgeConfig map[string]any `json:"knowledge_config"`
	Channel         string         `json:"channel"`
	PublishNote     string         `json:"publish_note"`
}

// PublishVersionRequest 发布版本请求
type PublishVersionRequest struct {
	PublishNote string `json:"publish_note"`
}

func (r *CreateVersionRequest) Normalize() {
	r.VersionNo = strings.TrimSpace(r.VersionNo)
	r.Prompt = strings.TrimSpace(r.Prompt)
	r.PublishNote = strings.TrimSpace(r.PublishNote)
	r.Channel = strings.ToLower(strings.TrimSpace(r.Channel))
	if r.Channel == "" {
		r.Channel = "web"
	}
}

func (r CreateVersionRequest) Validate() error {
	if r.VersionNo == "" {
		return errors.New("version_no is required")
	}
	if len(r.VersionNo) > 32 {
		return errors.New("version_no must be at most 32 characters")
	}
	switch r.Channel {
	case "web", "wechat", "api", "miniapp", "h5", "enterprise_wechat":
		// 合法渠道
	default:
		return errors.New("channel must be one of: web, wechat, api, miniapp, h5, enterprise_wechat")
	}
	return nil
}

// ========== 错误定义 ==========

var (
	ErrVersionNotFound = errors.New("agent version not found")
	ErrVersionExists   = errors.New("agent version already exists")
)

// ========== Repository 方法 ==========

// CreateVersion 创建版本快照
func (r Repository) CreateVersion(ctx context.Context, tenantID, agentID, userID string, req CreateVersionRequest) (AgentVersion, error) {
	modelConfig, err := jsonObject(req.ModelConfig)
	if err != nil {
		return AgentVersion{}, err
	}
	toolConfig, err := jsonObject(req.ToolConfig)
	if err != nil {
		return AgentVersion{}, err
	}
	knowledgeConfig, err := jsonObject(req.KnowledgeConfig)
	if err != nil {
		return AgentVersion{}, err
	}
	const q = `
INSERT INTO agent_versions(
    tenant_id, agent_id, version_no, prompt, model_config, tool_config, knowledge_config, channel, publish_note, created_by
) VALUES ($1, $2, $3, NULLIF($4, ''), $5::jsonb, $6::jsonb, $7::jsonb, $8, NULLIF($9, ''), NULLIF($10, '')::uuid)
RETURNING id::text, tenant_id::text, agent_id::text, version_no, COALESCE(prompt, ''),
          model_config, tool_config, knowledge_config, channel, status,
          COALESCE(publish_note, ''), COALESCE(created_by::text, ''), created_at, published_at`
	var item AgentVersion
	err = r.DB.QueryRow(ctx, q,
		tenantID, agentID, req.VersionNo, req.Prompt,
		modelConfig, toolConfig, knowledgeConfig,
		req.Channel, req.PublishNote, userID,
	).Scan(
		&item.ID, &item.TenantID, &item.AgentID, &item.VersionNo, &item.Prompt,
		&item.ModelConfig, &item.ToolConfig, &item.KnowledgeConfig,
		&item.Channel, &item.Status, &item.PublishNote, &item.CreatedBy,
		&item.CreatedAt, &item.PublishedAt,
	)
	return item, err
}

// ListVersions 列出某个 Agent 的所有版本
func (r Repository) ListVersions(ctx context.Context, tenantID, agentID string, limit int) ([]AgentVersion, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, agent_id::text, version_no, COALESCE(prompt, ''),
       model_config, tool_config, knowledge_config, channel, status,
       COALESCE(publish_note, ''), COALESCE(created_by::text, ''), created_at, published_at
FROM agent_versions
WHERE tenant_id = $1 AND agent_id = $2
ORDER BY created_at DESC
LIMIT $3`
	rows, err := r.DB.Query(ctx, q, tenantID, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AgentVersion, 0)
	for rows.Next() {
		item, err := scanAgentVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetVersion 获取指定版本
func (r Repository) GetVersion(ctx context.Context, tenantID, agentID, versionID string) (AgentVersion, error) {
	const q = `
SELECT id::text, tenant_id::text, agent_id::text, version_no, COALESCE(prompt, ''),
       model_config, tool_config, knowledge_config, channel, status,
       COALESCE(publish_note, ''), COALESCE(created_by::text, ''), created_at, published_at
FROM agent_versions
WHERE id = $1 AND tenant_id = $2 AND agent_id = $3`
	rows, err := r.DB.Query(ctx, q, versionID, tenantID, agentID)
	if err != nil {
		return AgentVersion{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return AgentVersion{}, rows.Err()
		}
		return AgentVersion{}, ErrVersionNotFound
	}
	item, err := scanAgentVersion(rows)
	if err != nil {
		return AgentVersion{}, err
	}
	return item, rows.Err()
}

// PublishVersion 发布版本：将指定版本状态设为 published，同 Agent 其他 published 版本变为 archived
func (r Repository) PublishVersion(ctx context.Context, tenantID, agentID, versionID, publishNote string) (AgentVersion, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return AgentVersion{}, err
	}
	defer tx.Rollback(ctx)

	// 将同 Agent 当前 published 版本归档
	_, err = tx.Exec(ctx, `
UPDATE agent_versions
SET status = 'archived'
WHERE tenant_id = $1 AND agent_id = $2 AND status = 'published'`, tenantID, agentID)
	if err != nil {
		return AgentVersion{}, err
	}

	// 发布目标版本
	const q = `
UPDATE agent_versions
SET status = 'published', published_at = now(), publish_note = COALESCE(NULLIF($4, ''), publish_note)
WHERE id = $1 AND tenant_id = $2 AND agent_id = $3
RETURNING id::text, tenant_id::text, agent_id::text, version_no, COALESCE(prompt, ''),
          model_config, tool_config, knowledge_config, channel, status,
          COALESCE(publish_note, ''), COALESCE(created_by::text, ''), created_at, published_at`
	rows, err := tx.Query(ctx, q, versionID, tenantID, agentID, publishNote)
	if err != nil {
		return AgentVersion{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return AgentVersion{}, rows.Err()
		}
		return AgentVersion{}, ErrVersionNotFound
	}
	item, err := scanAgentVersion(rows)
	if err != nil {
		return AgentVersion{}, err
	}
	rows.Close()

	// 同步更新 Agent 主表状态为 published
	_, err = tx.Exec(ctx, `
UPDATE agents SET status = 'published', updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, agentID, tenantID)
	if err != nil {
		return AgentVersion{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AgentVersion{}, err
	}
	return item, nil
}

// RollbackVersion 回滚版本：将指定版本状态标记为 rollback
func (r Repository) RollbackVersion(ctx context.Context, tenantID, agentID, versionID string) (AgentVersion, error) {
	const q = `
UPDATE agent_versions
SET status = 'rollback'
WHERE id = $1 AND tenant_id = $2 AND agent_id = $3 AND status = 'published'
RETURNING id::text, tenant_id::text, agent_id::text, version_no, COALESCE(prompt, ''),
          model_config, tool_config, knowledge_config, channel, status,
          COALESCE(publish_note, ''), COALESCE(created_by::text, ''), created_at, published_at`
	rows, err := r.DB.Query(ctx, q, versionID, tenantID, agentID)
	if err != nil {
		return AgentVersion{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return AgentVersion{}, rows.Err()
		}
		return AgentVersion{}, ErrVersionNotFound
	}
	item, err := scanAgentVersion(rows)
	if err != nil {
		return AgentVersion{}, err
	}
	return item, rows.Err()
}

func scanAgentVersion(rows pgx.Rows) (AgentVersion, error) {
	var item AgentVersion
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.AgentID, &item.VersionNo, &item.Prompt,
		&item.ModelConfig, &item.ToolConfig, &item.KnowledgeConfig,
		&item.Channel, &item.Status, &item.PublishNote, &item.CreatedBy,
		&item.CreatedAt, &item.PublishedAt,
	)
	return item, err
}

// ========== Handler 方法 ==========

// CreateVersion 创建 Agent 版本快照
func (h Handler) CreateVersion(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	agentID := c.Param("agent_id")

	// 校验 Agent 存在
	if _, err := h.Repo.Get(c.Request.Context(), t.ID, agentID); err != nil {
		writeAgentError(c, err)
		return
	}

	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}

	item, err := h.Repo.CreateVersion(c.Request.Context(), t.ID, agentID, u.ID, req)
	if err != nil {
		writeVersionError(c, err)
		return
	}
	response.OK(c, item)
}

// ListVersions 列出 Agent 版本
func (h Handler) ListVersions(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	agentID := c.Param("agent_id")

	if _, err := h.Repo.Get(c.Request.Context(), t.ID, agentID); err != nil {
		writeAgentError(c, err)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListVersions(c.Request.Context(), t.ID, agentID, limit)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

// GetVersion 获取单个版本详情
func (h Handler) GetVersion(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	agentID := c.Param("agent_id")
	versionID := c.Param("version_id")

	item, err := h.Repo.GetVersion(c.Request.Context(), t.ID, agentID, versionID)
	if err != nil {
		writeVersionError(c, err)
		return
	}
	response.OK(c, item)
}

// PublishVersion 发布版本
func (h Handler) PublishVersion(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	agentID := c.Param("agent_id")
	versionID := c.Param("version_id")

	var req PublishVersionRequest
	_ = c.ShouldBindJSON(&req)

	item, err := h.Repo.PublishVersion(c.Request.Context(), t.ID, agentID, versionID, req.PublishNote)
	if err != nil {
		writeVersionError(c, err)
		return
	}
	response.OK(c, item)
}

// RollbackVersion 回滚版本
func (h Handler) RollbackVersion(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	agentID := c.Param("agent_id")
	versionID := c.Param("version_id")

	item, err := h.Repo.RollbackVersion(c.Request.Context(), t.ID, agentID, versionID)
	if err != nil {
		writeVersionError(c, err)
		return
	}
	response.OK(c, item)
}

func writeVersionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrVersionNotFound):
		response.Error(c, http.StatusNotFound, 40450, "agent version not found")
	case errors.Is(err, ErrVersionExists):
		response.Error(c, http.StatusConflict, 40950, "agent version already exists")
	case errors.Is(err, ErrAgentNotFound):
		writeAgentError(c, err)
	default:
		response.Error(c, http.StatusInternalServerError, 50050, err.Error())
	}
}
