package webhook

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) CreateEndpoint(ctx context.Context, tenantID string, req CreateEndpointRequest) (Endpoint, error) {
	events, err := json.Marshal(req.Events)
	if err != nil {
		return Endpoint{}, err
	}
	const q = `
INSERT INTO webhook_endpoints(tenant_id, name, url, secret, events, status)
VALUES ($1, $2, $3, NULLIF($4, ''), $5::jsonb, $6)
RETURNING id::text, tenant_id::text, name, url, COALESCE(secret, ''), events, status, created_at, updated_at`
	return r.scanEndpointRow(ctx, q, tenantID, req.Name, req.URL, req.Secret, string(events), req.Status)
}

func (r Repository) UpdateEndpoint(ctx context.Context, tenantID, endpointID string, req UpdateEndpointRequest) (Endpoint, error) {
	current, err := r.GetEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return Endpoint{}, err
	}
	if req.Name == "" {
		req.Name = current.Name
	}
	if req.URL == "" {
		req.URL = current.URL
	}
	if req.Status == "" {
		req.Status = current.Status
	}
	if len(req.Events) == 0 {
		req.Events = current.Events
	}
	events, err := json.Marshal(req.Events)
	if err != nil {
		return Endpoint{}, err
	}
	const q = `
UPDATE webhook_endpoints
SET name = $3, url = $4, secret = NULLIF($5, ''), events = $6::jsonb, status = $7, updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, name, url, COALESCE(secret, ''), events, status, created_at, updated_at`
	return r.scanEndpointRow(ctx, q, endpointID, tenantID, req.Name, req.URL, req.Secret, string(events), req.Status)
}

func (r Repository) GetEndpoint(ctx context.Context, tenantID, endpointID string) (Endpoint, error) {
	const q = `
SELECT id::text, tenant_id::text, name, url, COALESCE(secret, ''), events, status, created_at, updated_at
FROM webhook_endpoints
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	return r.scanEndpointRow(ctx, q, endpointID, tenantID)
}

func (r Repository) ListEndpoints(ctx context.Context, tenantID string) ([]Endpoint, error) {
	const q = `
SELECT id::text, tenant_id::text, name, url, COALESCE(secret, ''), events, status, created_at, updated_at
FROM webhook_endpoints
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Endpoint, 0)
	for rows.Next() {
		item, err := scanEndpoint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) ListActiveEndpointsForEvent(ctx context.Context, tenantID, eventType string) ([]Endpoint, error) {
	const q = `
SELECT id::text, tenant_id::text, name, url, COALESCE(secret, ''), events, status, created_at, updated_at
FROM webhook_endpoints
WHERE tenant_id = $1
  AND status = 'active'
  AND deleted_at IS NULL
  AND events ? $2
ORDER BY created_at ASC`
	rows, err := r.DB.Query(ctx, q, tenantID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Endpoint, 0)
	for rows.Next() {
		item, err := scanEndpoint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) DeleteEndpoint(ctx context.Context, tenantID, endpointID string) error {
	tag, err := r.DB.Exec(ctx, `UPDATE webhook_endpoints SET deleted_at = now(), updated_at = now() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, endpointID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrEndpointNotFound
	}
	return nil
}

func (r Repository) CreateDelivery(ctx context.Context, item Delivery) (Delivery, error) {
	body, err := json.Marshal(item.RequestBody)
	if err != nil {
		return Delivery{}, err
	}
	const q = `
INSERT INTO webhook_deliveries(
    tenant_id, endpoint_id, event_type, target_url, status, http_status, request_body,
    response_body, error_message, duration_ms, retry_count
) VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, 0), $7::jsonb, NULLIF($8, ''), NULLIF($9, ''), $10, $11)
RETURNING id::text, tenant_id::text, COALESCE(endpoint_id::text, ''), event_type, target_url, status,
          COALESCE(http_status, 0), request_body, COALESCE(response_body, ''), COALESCE(error_message, ''),
          duration_ms, retry_count, created_at`
	return r.scanDeliveryRow(ctx, q, item.TenantID, item.EndpointID, item.EventType, item.TargetURL, item.Status, item.HTTPStatus, string(body), item.ResponseBody, item.ErrorMessage, item.DurationMS, item.RetryCount)
}

func (r Repository) ListDeliveries(ctx context.Context, tenantID, endpointID string, limit int) ([]Delivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{tenantID, limit}
	filter := ""
	if endpointID != "" {
		filter = " AND endpoint_id = $3::uuid"
		args = append(args, endpointID)
	}
	q := `
SELECT id::text, tenant_id::text, COALESCE(endpoint_id::text, ''), event_type, target_url, status,
       COALESCE(http_status, 0), request_body, COALESCE(response_body, ''), COALESCE(error_message, ''),
       duration_ms, retry_count, created_at
FROM webhook_deliveries
WHERE tenant_id = $1` + filter + `
ORDER BY created_at DESC
LIMIT $2`
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Delivery, 0)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) scanEndpointRow(ctx context.Context, q string, args ...any) (Endpoint, error) {
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return Endpoint{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Endpoint{}, rows.Err()
		}
		return Endpoint{}, ErrEndpointNotFound
	}
	item, err := scanEndpoint(rows)
	if err != nil {
		return Endpoint{}, err
	}
	return item, rows.Err()
}

func scanEndpoint(rows pgx.Rows) (Endpoint, error) {
	var item Endpoint
	var events []byte
	if err := rows.Scan(&item.ID, &item.TenantID, &item.Name, &item.URL, &item.Secret, &events, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Endpoint{}, err
	}
	_ = json.Unmarshal(events, &item.Events)
	return item, nil
}

func (r Repository) scanDeliveryRow(ctx context.Context, q string, args ...any) (Delivery, error) {
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return Delivery{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Delivery{}, rows.Err()
		}
		return Delivery{}, ErrEndpointNotFound
	}
	item, err := scanDelivery(rows)
	if err != nil {
		return Delivery{}, err
	}
	return item, rows.Err()
}

func scanDelivery(rows pgx.Rows) (Delivery, error) {
	var item Delivery
	var body []byte
	if err := rows.Scan(
		&item.ID, &item.TenantID, &item.EndpointID, &item.EventType, &item.TargetURL, &item.Status,
		&item.HTTPStatus, &body, &item.ResponseBody, &item.ErrorMessage, &item.DurationMS, &item.RetryCount, &item.CreatedAt,
	); err != nil {
		return Delivery{}, err
	}
	_ = json.Unmarshal(body, &item.RequestBody)
	if item.RequestBody == nil {
		item.RequestBody = map[string]any{}
	}
	return item, nil
}
