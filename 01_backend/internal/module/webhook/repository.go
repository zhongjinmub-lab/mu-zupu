package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	if req.Secret == "" {
		// 未传 secret 时保留原密钥，避免启停或普通编辑时清空签名。
		req.Secret = current.Secret
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
    response_body, error_message, duration_ms, retry_count, next_retry_at, last_attempt_at
) VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, 0), $7::jsonb, NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12, now())
RETURNING id::text, tenant_id::text, COALESCE(endpoint_id::text, ''), event_type, target_url, status,
          COALESCE(http_status, 0), request_body, COALESCE(response_body, ''), COALESCE(error_message, ''),
          duration_ms, retry_count, next_retry_at, last_attempt_at, created_at`
	return r.scanDeliveryRow(ctx, q, item.TenantID, item.EndpointID, item.EventType, item.TargetURL, item.Status, item.HTTPStatus, string(body), item.ResponseBody, item.ErrorMessage, item.DurationMS, item.RetryCount, item.NextRetryAt)
}

func (r Repository) ListDeliveries(ctx context.Context, query DeliveryQuery) ([]Delivery, error) {
	query.Normalize()
	return r.listDeliveries(ctx, query)
}

func (r Repository) ExportDeliveries(ctx context.Context, query DeliveryQuery) ([]Delivery, error) {
	query.NormalizeForExport()
	return r.listDeliveries(ctx, query)
}

func (r Repository) listDeliveries(ctx context.Context, query DeliveryQuery) ([]Delivery, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	args := []any{query.TenantID}
	filters := []string{"tenant_id = $1"}
	if query.EndpointID != "" {
		args = append(args, query.EndpointID)
		filters = append(filters, fmt.Sprintf("endpoint_id = $%d::uuid", len(args)))
	}
	if query.EventType != "" {
		args = append(args, query.EventType)
		filters = append(filters, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if query.Status != "" {
		args = append(args, query.Status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		filters = append(filters, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		filters = append(filters, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	args = append(args, query.Limit)
	limitArg := len(args)
	q := `
SELECT id::text, tenant_id::text, COALESCE(endpoint_id::text, ''), event_type, target_url, status,
       COALESCE(http_status, 0), request_body, COALESCE(response_body, ''), COALESCE(error_message, ''),
       duration_ms, retry_count, next_retry_at, last_attempt_at, created_at
FROM webhook_deliveries
WHERE ` + strings.Join(filters, " AND ") + `
ORDER BY created_at DESC
LIMIT $` + fmt.Sprint(limitArg)
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

func (r Repository) DeliverySummary(ctx context.Context, query DeliveryQuery) (DeliverySummary, error) {
	query.Normalize()
	if err := query.Validate(); err != nil {
		return DeliverySummary{}, err
	}
	args := []any{query.TenantID}
	filters := []string{"tenant_id = $1"}
	if query.EndpointID != "" {
		args = append(args, query.EndpointID)
		filters = append(filters, fmt.Sprintf("endpoint_id = $%d::uuid", len(args)))
	}
	if query.EventType != "" {
		args = append(args, query.EventType)
		filters = append(filters, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if query.Status != "" {
		args = append(args, query.Status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		filters = append(filters, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		filters = append(filters, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	q := `
SELECT
    COUNT(*)::int,
    COUNT(*) FILTER (WHERE status = 'success')::int,
    COUNT(*) FILTER (WHERE status = 'failed')::int,
    COUNT(*) FILTER (WHERE status = 'failed' AND next_retry_at IS NOT NULL)::int,
    COUNT(*) FILTER (WHERE status = 'failed' AND next_retry_at IS NOT NULL AND next_retry_at <= now())::int,
    COUNT(*) FILTER (WHERE status = 'failed' AND next_retry_at IS NULL)::int,
    MAX(last_attempt_at)
FROM webhook_deliveries
WHERE ` + strings.Join(filters, " AND ")
	var item DeliverySummary
	if err := r.DB.QueryRow(ctx, q, args...).Scan(
		&item.Total,
		&item.Success,
		&item.Failed,
		&item.RetryScheduled,
		&item.RetryDue,
		&item.ManualReview,
		&item.LastAttemptAt,
	); err != nil {
		return DeliverySummary{}, err
	}
	return item, nil
}

func (r Repository) ClaimRetryJobs(ctx context.Context, limit int) ([]RetryJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	const q = `
WITH due AS (
    SELECT d.id
    FROM webhook_deliveries d
    JOIN webhook_endpoints e ON e.id = d.endpoint_id
    WHERE d.status = 'failed'
      AND d.next_retry_at IS NOT NULL
      AND d.next_retry_at <= now()
      AND e.status = 'active'
      AND e.deleted_at IS NULL
    ORDER BY d.next_retry_at ASC, d.created_at ASC
    LIMIT $1
    FOR UPDATE OF d SKIP LOCKED
)
UPDATE webhook_deliveries d
SET next_retry_at = now() + interval '5 minutes',
    last_attempt_at = now()
FROM due, webhook_endpoints e
WHERE d.id = due.id
  AND e.id = d.endpoint_id
RETURNING d.id::text, d.tenant_id::text, COALESCE(d.endpoint_id::text, ''), d.event_type, d.target_url, d.status,
          COALESCE(d.http_status, 0), d.request_body, COALESCE(d.response_body, ''), COALESCE(d.error_message, ''),
          d.duration_ms, d.retry_count, d.next_retry_at, d.last_attempt_at, d.created_at,
          e.id::text, e.tenant_id::text, e.name, e.url, COALESCE(e.secret, ''), e.events, e.status, e.created_at, e.updated_at`
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	jobs := make([]RetryJob, 0)
	for rows.Next() {
		job, err := scanRetryJob(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r Repository) GetRetryJob(ctx context.Context, tenantID, deliveryID string) (RetryJob, error) {
	const q = `
SELECT d.id::text, d.tenant_id::text, COALESCE(d.endpoint_id::text, ''), d.event_type, d.target_url, d.status,
       COALESCE(d.http_status, 0), d.request_body, COALESCE(d.response_body, ''), COALESCE(d.error_message, ''),
       d.duration_ms, d.retry_count, d.next_retry_at, d.last_attempt_at, d.created_at,
       e.id::text, e.tenant_id::text, e.name, e.url, COALESCE(e.secret, ''), e.events, e.status, e.created_at, e.updated_at
FROM webhook_deliveries d
JOIN webhook_endpoints e ON e.id = d.endpoint_id
WHERE d.id = $1
  AND d.tenant_id = $2
  AND e.tenant_id = $2
  AND e.deleted_at IS NULL`
	rows, err := r.DB.Query(ctx, q, deliveryID, tenantID)
	if err != nil {
		return RetryJob{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return RetryJob{}, rows.Err()
		}
		return RetryJob{}, ErrDeliveryNotFound
	}
	job, err := scanRetryJob(rows)
	if err != nil {
		return RetryJob{}, err
	}
	return job, rows.Err()
}

func (r Repository) UpdateDeliveryAttempt(ctx context.Context, deliveryID string, result Delivery) (Delivery, error) {
	const q = `
UPDATE webhook_deliveries
SET status = $2,
    http_status = NULLIF($3, 0),
    response_body = NULLIF($4, ''),
    error_message = NULLIF($5, ''),
    duration_ms = $6,
    retry_count = $7,
    next_retry_at = $8,
    last_attempt_at = now()
WHERE id = $1
RETURNING id::text, tenant_id::text, COALESCE(endpoint_id::text, ''), event_type, target_url, status,
          COALESCE(http_status, 0), request_body, COALESCE(response_body, ''), COALESCE(error_message, ''),
          duration_ms, retry_count, next_retry_at, last_attempt_at, created_at`
	return r.scanDeliveryRow(ctx, q, deliveryID, result.Status, result.HTTPStatus, result.ResponseBody, result.ErrorMessage, result.DurationMS, result.RetryCount, result.NextRetryAt)
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
	item.HasSecret = item.Secret != ""
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
		&item.HTTPStatus, &body, &item.ResponseBody, &item.ErrorMessage, &item.DurationMS, &item.RetryCount, &item.NextRetryAt, &item.LastAttemptAt, &item.CreatedAt,
	); err != nil {
		return Delivery{}, err
	}
	_ = json.Unmarshal(body, &item.RequestBody)
	if item.RequestBody == nil {
		item.RequestBody = map[string]any{}
	}
	return item, nil
}

func scanRetryJob(rows pgx.Rows) (RetryJob, error) {
	var delivery Delivery
	var requestBody []byte
	var endpoint Endpoint
	var endpointEvents []byte
	if err := rows.Scan(
		&delivery.ID, &delivery.TenantID, &delivery.EndpointID, &delivery.EventType, &delivery.TargetURL, &delivery.Status,
		&delivery.HTTPStatus, &requestBody, &delivery.ResponseBody, &delivery.ErrorMessage, &delivery.DurationMS, &delivery.RetryCount,
		&delivery.NextRetryAt, &delivery.LastAttemptAt, &delivery.CreatedAt,
		&endpoint.ID, &endpoint.TenantID, &endpoint.Name, &endpoint.URL, &endpoint.Secret, &endpointEvents, &endpoint.Status, &endpoint.CreatedAt, &endpoint.UpdatedAt,
	); err != nil {
		return RetryJob{}, err
	}
	_ = json.Unmarshal(requestBody, &delivery.RequestBody)
	if delivery.RequestBody == nil {
		delivery.RequestBody = map[string]any{}
	}
	_ = json.Unmarshal(endpointEvents, &endpoint.Events)
	endpoint.HasSecret = endpoint.Secret != ""
	return RetryJob{
		Delivery: delivery,
		Endpoint: endpoint,
		Event: Event{
			Type:      delivery.EventType,
			TenantID:  delivery.TenantID,
			Payload:   eventPayload(delivery.RequestBody),
			CreatedAt: eventCreatedAt(delivery.RequestBody, delivery.CreatedAt),
		},
	}, nil
}

func eventPayload(body map[string]any) map[string]any {
	payload, ok := body["payload"].(map[string]any)
	if !ok || payload == nil {
		return map[string]any{}
	}
	return payload
}

func eventCreatedAt(body map[string]any, fallback time.Time) time.Time {
	raw, _ := body["created_at"].(string)
	if raw == "" {
		return fallback
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return fallback
	}
	return t
}
