package workflow

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrWorkflowNotFound 表示当前租户下不存在指定工作流。
var ErrWorkflowNotFound = errors.New("workflow not found")

// Repository 提供工作流定义的持久化访问。
type Repository struct {
	DB *pgxpool.Pool
}

// NewRepository 构造工作流 Repository。
func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

const workflowColumns = `id::text, tenant_id::text, name, code, COALESCE(description, ''), definition, status, version, COALESCE(created_by::text, ''), created_at, updated_at`

// Create 创建工作流定义。
func (r Repository) Create(ctx context.Context, tenantID, userID string, req CreateWorkflowRequest) (Workflow, error) {
	definition, err := marshalDefinition(req.Definition)
	if err != nil {
		return Workflow{}, err
	}
	const q = `
INSERT INTO workflows(tenant_id, name, code, description, definition, created_by)
VALUES ($1, $2, $3, NULLIF($4, ''), $5::jsonb, NULLIF($6, '')::uuid)
RETURNING ` + workflowColumns
	rows, err := r.DB.Query(ctx, q, tenantID, req.Name, req.Code, req.Description, definition, userID)
	if err != nil {
		return Workflow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Workflow{}, rows.Err()
		}
		return Workflow{}, ErrWorkflowNotFound
	}
	return scanWorkflow(rows)
}

// List 返回当前租户未归档（未软删除）的工作流列表。
func (r Repository) List(ctx context.Context, tenantID string) ([]Workflow, error) {
	const q = `
SELECT ` + workflowColumns + `
FROM workflows
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Workflow, 0)
	for rows.Next() {
		item, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Get 返回指定工作流（按当前租户隔离）。
func (r Repository) Get(ctx context.Context, tenantID, workflowID string) (Workflow, error) {
	const q = `
SELECT ` + workflowColumns + `
FROM workflows
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	rows, err := r.DB.Query(ctx, q, workflowID, tenantID)
	if err != nil {
		return Workflow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Workflow{}, rows.Err()
		}
		return Workflow{}, ErrWorkflowNotFound
	}
	return scanWorkflow(rows)
}

// Update 更新工作流的名称、描述与定义；更新后若原为已发布则回到草稿，并递增版本号。
func (r Repository) Update(ctx context.Context, tenantID, workflowID string, req UpdateWorkflowRequest) (Workflow, error) {
	current, err := r.Get(ctx, tenantID, workflowID)
	if err != nil {
		return Workflow{}, err
	}
	name := req.Name
	if name == "" {
		name = current.Name
	}
	description := req.Description
	if description == "" {
		description = current.Description
	}
	definitionGraph := current.Definition
	if req.Definition != nil {
		definitionGraph = *req.Definition
	}
	definition, err := marshalDefinition(definitionGraph)
	if err != nil {
		return Workflow{}, err
	}
	const q = `
UPDATE workflows
SET name = $3,
    description = NULLIF($4, ''),
    definition = $5::jsonb,
    status = CASE WHEN status = 'published' THEN 'draft' ELSE status END,
    version = version + 1,
    updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING ` + workflowColumns
	rows, err := r.DB.Query(ctx, q, workflowID, tenantID, name, description, definition)
	if err != nil {
		return Workflow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Workflow{}, ErrWorkflowNotFound
	}
	return scanWorkflow(rows)
}

// SetStatus 设置工作流状态（如发布）。
func (r Repository) SetStatus(ctx context.Context, tenantID, workflowID, status string) (Workflow, error) {
	const q = `
UPDATE workflows
SET status = $3, updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING ` + workflowColumns
	rows, err := r.DB.Query(ctx, q, workflowID, tenantID, status)
	if err != nil {
		return Workflow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Workflow{}, ErrWorkflowNotFound
	}
	return scanWorkflow(rows)
}

// Archive 归档（软删除）工作流。
func (r Repository) Archive(ctx context.Context, tenantID, workflowID string) error {
	const q = `
UPDATE workflows
SET status = 'archived', deleted_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	tag, err := r.DB.Exec(ctx, q, workflowID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrWorkflowNotFound
	}
	return nil
}

// scanWorkflow 从查询行扫描工作流记录，definition 列以原始 JSON 字节解码为图结构。
func scanWorkflow(rows pgx.Rows) (Workflow, error) {
	var item Workflow
	var definition []byte
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Code, &item.Description,
		&definition, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return Workflow{}, err
	}
	if len(definition) > 0 {
		if err := json.Unmarshal(definition, &item.Definition); err != nil {
			return Workflow{}, err
		}
	}
	if item.Definition.Nodes == nil {
		item.Definition.Nodes = []WorkflowNode{}
	}
	if item.Definition.Edges == nil {
		item.Definition.Edges = []WorkflowEdge{}
	}
	return item, nil
}

const workflowRunColumns = `id::text, tenant_id::text, workflow_id::text, status, mode, input, steps, cost_ms, COALESCE(created_by::text, ''), created_at`

// InsertWorkflowRun 写入一条工作流运行记录（执行日志）并返回它。
func (r Repository) InsertWorkflowRun(ctx context.Context, tenantID, workflowID, userID, status, mode string, input map[string]any, steps []WorkflowRunStep, costMS int) (WorkflowRun, error) {
	if input == nil {
		input = map[string]any{}
	}
	if steps == nil {
		steps = []WorkflowRunStep{}
	}
	inputJSON, err := marshalJSONValue(input)
	if err != nil {
		return WorkflowRun{}, err
	}
	stepsJSON, err := marshalJSONValue(steps)
	if err != nil {
		return WorkflowRun{}, err
	}
	const q = `
INSERT INTO workflow_runs(tenant_id, workflow_id, status, mode, input, steps, cost_ms, created_by)
VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, NULLIF($8, '')::uuid)
RETURNING ` + workflowRunColumns
	rows, err := r.DB.Query(ctx, q, tenantID, workflowID, status, mode, inputJSON, stepsJSON, costMS, userID)
	if err != nil {
		return WorkflowRun{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return WorkflowRun{}, rows.Err()
		}
		return WorkflowRun{}, ErrWorkflowNotFound
	}
	return scanWorkflowRun(rows)
}

// ListWorkflowRuns 返回指定工作流的运行记录（按时间倒序），按当前租户隔离。
func (r Repository) ListWorkflowRuns(ctx context.Context, tenantID, workflowID string, limit int) ([]WorkflowRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	const q = `
SELECT ` + workflowRunColumns + `
FROM workflow_runs
WHERE tenant_id = $1 AND workflow_id = $2
ORDER BY created_at DESC, id DESC
LIMIT $3`
	rows, err := r.DB.Query(ctx, q, tenantID, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkflowRun, 0)
	for rows.Next() {
		item, err := scanWorkflowRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// scanWorkflowRun 扫描运行记录，input/steps 列以原始 JSON 字节解码。
func scanWorkflowRun(rows pgx.Rows) (WorkflowRun, error) {
	var item WorkflowRun
	var input []byte
	var steps []byte
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.WorkflowID, &item.Status, &item.Mode,
		&input, &steps, &item.CostMS, &item.CreatedBy, &item.CreatedAt,
	)
	if err != nil {
		return WorkflowRun{}, err
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &item.Input); err != nil {
			return WorkflowRun{}, err
		}
	}
	if len(steps) > 0 {
		if err := json.Unmarshal(steps, &item.Steps); err != nil {
			return WorkflowRun{}, err
		}
	}
	if item.Input == nil {
		item.Input = map[string]any{}
	}
	if item.Steps == nil {
		item.Steps = []WorkflowRunStep{}
	}
	return item, nil
}
