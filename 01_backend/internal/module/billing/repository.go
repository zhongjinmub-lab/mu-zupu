package billing

import (
	"context"
	"encoding/json"
	"errors"
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

func (r Repository) ListPlans(ctx context.Context) ([]Plan, error) {
	const q = `
SELECT id::text, code, name, price_cents, billing_cycle, quota, status, created_at, updated_at
FROM plans
WHERE status = 'active' AND deleted_at IS NULL
ORDER BY price_cents ASC, created_at ASC`
	rows, err := r.DB.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Plan, 0)
	for rows.Next() {
		var item Plan
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.PriceCents, &item.BillingCycle, &item.Quota, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) GetPlanByCode(ctx context.Context, code string) (Plan, error) {
	const q = `
SELECT id::text, code, name, price_cents, billing_cycle, quota, status, created_at, updated_at
FROM plans
WHERE code = $1 AND status = 'active' AND deleted_at IS NULL`
	var item Plan
	err := r.DB.QueryRow(ctx, q, code).Scan(
		&item.ID, &item.Code, &item.Name, &item.PriceCents, &item.BillingCycle, &item.Quota, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return Plan{}, ErrPlanNotFound
	}
	return item, err
}

func (r Repository) GetActiveSubscription(ctx context.Context, tenantID string) (Subscription, error) {
	const q = `
SELECT s.id::text, s.tenant_id::text, s.plan_id::text, p.code, p.name, s.status,
       s.started_at, s.expired_at, s.auto_renew, s.metadata, s.created_at, s.updated_at
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
WHERE s.tenant_id = $1
  AND s.status = 'active'
  AND s.deleted_at IS NULL
  AND (s.expired_at IS NULL OR s.expired_at > now())
ORDER BY s.started_at DESC
LIMIT 1`
	var item Subscription
	err := r.DB.QueryRow(ctx, q, tenantID).Scan(
		&item.ID, &item.TenantID, &item.PlanID, &item.PlanCode, &item.PlanName, &item.Status,
		&item.StartedAt, &item.ExpiredAt, &item.AutoRenew, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return Subscription{}, ErrSubscriptionNotFound
	}
	return item, err
}

func (r Repository) EnsureFreeSubscription(ctx context.Context, tenantID string) (Subscription, error) {
	if item, err := r.GetActiveSubscription(ctx, tenantID); err == nil {
		return item, nil
	} else if err != ErrSubscriptionNotFound {
		return Subscription{}, err
	}
	if err := r.EnsureFreePlan(ctx); err != nil {
		return Subscription{}, err
	}
	const q = `
INSERT INTO subscriptions(tenant_id, plan_id, status, metadata)
SELECT $1, id, 'active', '{"source":"auto_free"}'::jsonb
FROM plans
WHERE code = 'free' AND status = 'active' AND deleted_at IS NULL
LIMIT 1`
	if _, err := r.DB.Exec(ctx, q, tenantID); err != nil {
		return Subscription{}, err
	}
	return r.GetActiveSubscription(ctx, tenantID)
}

func (r Repository) EnsureFreePlan(ctx context.Context) error {
	const q = `
INSERT INTO plans(code, name, quota, status)
VALUES (
    'free',
    'Free',
    '{"rag_requests":1000,"agent_messages":1000,"file_upload_bytes":104857600,"embedding_chunks":5000}'::jsonb,
    'active'
)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    quota = EXCLUDED.quota,
    status = 'active',
    deleted_at = NULL,
    updated_at = now()`
	_, err := r.DB.Exec(ctx, q)
	return err
}

func (r Repository) CreateOrder(ctx context.Context, tenantID string, req CreateOrderRequest) (BusinessOrder, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return BusinessOrder{}, err
	}
	plan, err := r.GetPlanByCode(ctx, req.PlanCode)
	if err != nil {
		return BusinessOrder{}, err
	}
	if req.AmountCents == 0 {
		req.AmountCents = plan.PriceCents
	}
	meta := req.Metadata
	meta["plan_code"] = plan.Code
	metaJSON, err := jsonObject(meta)
	if err != nil {
		return BusinessOrder{}, err
	}
	const q = `
INSERT INTO business_orders(tenant_id, order_no, order_type, plan_id, amount_cents, currency, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id::text, tenant_id::text, order_no, order_type, COALESCE(plan_id::text, ''), amount_cents, currency, status, metadata, created_at, updated_at`
	return scanBusinessOrder(r.DB.QueryRow(ctx, q, tenantID, newNo("BO"), req.OrderType, plan.ID, req.AmountCents, req.Currency, metaJSON))
}

func (r Repository) ListOrders(ctx context.Context, tenantID string, limit int) ([]BusinessOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, order_no, order_type, COALESCE(plan_id::text, ''), amount_cents, currency, status, metadata, created_at, updated_at
FROM business_orders
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2`
	rows, err := r.DB.Query(ctx, q, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BusinessOrder, 0)
	for rows.Next() {
		item, err := scanBusinessOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) GetOrder(ctx context.Context, tenantID, orderID string) (BusinessOrder, error) {
	const q = `
SELECT id::text, tenant_id::text, order_no, order_type, COALESCE(plan_id::text, ''), amount_cents, currency, status, metadata, created_at, updated_at
FROM business_orders
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	item, err := scanBusinessOrder(r.DB.QueryRow(ctx, q, orderID, tenantID))
	if err == pgx.ErrNoRows {
		return BusinessOrder{}, ErrOrderNotFound
	}
	return item, err
}

func (r Repository) CancelOrder(ctx context.Context, tenantID, orderID, reason string) (BusinessOrder, error) {
	return r.closeOrder(ctx, tenantID, orderID, "cancelled", reason)
}

func (r Repository) CloseOrder(ctx context.Context, tenantID, orderID, reason string) (BusinessOrder, error) {
	return r.closeOrder(ctx, tenantID, orderID, "closed", reason)
}

func (r Repository) closeOrder(ctx context.Context, tenantID, orderID, status, reason string) (BusinessOrder, error) {
	reason = normalizeReason(reason)
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return BusinessOrder{}, err
	}
	defer tx.Rollback(ctx)

	const currentQ = `
SELECT id::text, tenant_id::text, order_no, order_type, COALESCE(plan_id::text, ''), amount_cents, currency, status, metadata, created_at, updated_at
FROM business_orders
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
FOR UPDATE`
	current, err := scanBusinessOrder(tx.QueryRow(ctx, currentQ, orderID, tenantID))
	if err == pgx.ErrNoRows {
		return BusinessOrder{}, ErrOrderNotFound
	}
	if err != nil {
		return BusinessOrder{}, err
	}
	if current.Status == "paid" {
		return BusinessOrder{}, ErrOrderAlreadyPaid
	}
	if current.Status == "cancelled" || current.Status == "closed" {
		if err := tx.Commit(ctx); err != nil {
			return BusinessOrder{}, err
		}
		return current, nil
	}

	const updateOrder = `
UPDATE business_orders
SET status = $3,
    metadata = metadata || jsonb_build_object('close_reason', $4::text, 'closed_at', now()),
    updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'pending' AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, order_no, order_type, COALESCE(plan_id::text, ''), amount_cents, currency, status, metadata, created_at, updated_at`
	item, err := scanBusinessOrder(tx.QueryRow(ctx, updateOrder, orderID, tenantID, status, reason))
	if err == pgx.ErrNoRows {
		return BusinessOrder{}, ErrOrderNotFound
	}
	if err != nil {
		return BusinessOrder{}, err
	}

	const closePayments = `
UPDATE payment_orders
SET status = 'closed',
    callback_payload = callback_payload || jsonb_build_object('closed_by_order_status', $3::text, 'reason', $4::text),
    updated_at = now()
WHERE tenant_id = $1 AND business_order_id = $2 AND status = 'pending' AND deleted_at IS NULL`
	if _, err := tx.Exec(ctx, closePayments, tenantID, orderID, status, reason); err != nil {
		return BusinessOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BusinessOrder{}, err
	}
	return item, nil
}

func (r Repository) CreatePaymentOrder(ctx context.Context, tenantID string, req CreatePaymentOrderRequest) (PaymentOrder, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return PaymentOrder{}, err
	}
	const q = `
WITH bo AS (
    SELECT id, amount_cents, currency
    FROM business_orders
    WHERE id = $2 AND tenant_id = $1 AND status = 'pending' AND deleted_at IS NULL
)
INSERT INTO payment_orders(tenant_id, business_order_id, pay_no, channel, amount_cents, currency)
SELECT $1, id, $3, $4, amount_cents, currency FROM bo
RETURNING id::text, tenant_id::text, business_order_id::text, pay_no, channel, amount_cents, currency, status,
          COALESCE(transaction_id, ''), paid_at, callback_payload, created_at, updated_at`
	item, err := scanPaymentOrder(r.DB.QueryRow(ctx, q, tenantID, req.BusinessOrderID, newNo("PO"), req.Channel))
	if err == pgx.ErrNoRows {
		return PaymentOrder{}, ErrOrderNotFound
	}
	return item, err
}

func (r Repository) ListPaymentOrders(ctx context.Context, tenantID, businessOrderID string, limit int) ([]PaymentOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args := []any{tenantID, limit}
	where := `WHERE tenant_id = $1 AND deleted_at IS NULL`
	if businessOrderID != "" {
		args = []any{tenantID, businessOrderID, limit}
		where += ` AND business_order_id = $2`
	}
	limitPlaceholder := "$2"
	if businessOrderID != "" {
		limitPlaceholder = "$3"
	}
	q := `
SELECT id::text, tenant_id::text, business_order_id::text, pay_no, channel, amount_cents, currency, status,
       COALESCE(transaction_id, ''), paid_at, callback_payload, created_at, updated_at
FROM payment_orders
` + where + `
ORDER BY created_at DESC
LIMIT ` + limitPlaceholder
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PaymentOrder, 0)
	for rows.Next() {
		item, err := scanPaymentOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) QueryPaymentOrder(ctx context.Context, tenantID, paymentID string) (PaymentOrder, error) {
	const q = `
SELECT id::text, tenant_id::text, business_order_id::text, pay_no, channel, amount_cents, currency, status,
       COALESCE(transaction_id, ''), paid_at, callback_payload, created_at, updated_at
FROM payment_orders
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	item, err := scanPaymentOrder(r.DB.QueryRow(ctx, q, paymentID, tenantID))
	if err == pgx.ErrNoRows {
		return PaymentOrder{}, ErrPaymentNotFound
	}
	return item, err
}

func (r Repository) ClosePaymentOrder(ctx context.Context, tenantID, paymentID, reason string) (PaymentOrder, error) {
	reason = normalizeReason(reason)
	const currentQ = `
SELECT id::text, tenant_id::text, business_order_id::text, pay_no, channel, amount_cents, currency, status,
       COALESCE(transaction_id, ''), paid_at, callback_payload, created_at, updated_at
FROM payment_orders
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	current, err := scanPaymentOrder(r.DB.QueryRow(ctx, currentQ, paymentID, tenantID))
	if err == pgx.ErrNoRows {
		return PaymentOrder{}, ErrPaymentNotFound
	}
	if err != nil {
		return PaymentOrder{}, err
	}
	if current.Status == "paid" {
		return PaymentOrder{}, ErrPaymentAlreadyPaid
	}
	if current.Status == "closed" {
		return current, nil
	}
	const q = `
UPDATE payment_orders
SET status = 'closed',
    callback_payload = callback_payload || jsonb_build_object('close_reason', $3::text, 'closed_at', now()),
    updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND status <> 'paid' AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, business_order_id::text, pay_no, channel, amount_cents, currency, status,
          COALESCE(transaction_id, ''), paid_at, callback_payload, created_at, updated_at`
	item, err := scanPaymentOrder(r.DB.QueryRow(ctx, q, paymentID, tenantID, reason))
	if err == pgx.ErrNoRows {
		return PaymentOrder{}, ErrPaymentNotFound
	}
	if err != nil {
		return PaymentOrder{}, err
	}
	return item, nil
}

func (r Repository) ApplyPaymentCallback(ctx context.Context, tenantID, channel string, req PaymentCallbackRequest, requestID string) (PaymentOrder, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return PaymentOrder{}, err
	}
	payload, err := jsonObject(map[string]any{
		"status":         req.Status,
		"transaction_id": req.TransactionID,
		"metadata":       req.Metadata,
	})
	if err != nil {
		return PaymentOrder{}, err
	}
	requestID = normalizeReason(requestID)
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return PaymentOrder{}, err
	}
	defer tx.Rollback(ctx)
	const updatePayment = `
UPDATE payment_orders
SET status = CASE WHEN status = 'paid' THEN status ELSE $4 END,
    transaction_id = COALESCE(NULLIF(transaction_id, ''), NULLIF($5, '')),
    paid_at = CASE WHEN status = 'paid' THEN paid_at WHEN $4 = 'paid' THEN now() ELSE paid_at END,
    callback_payload = $6::jsonb,
    updated_at = now()
WHERE tenant_id = $1 AND channel = $2 AND pay_no = $3 AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, business_order_id::text, pay_no, channel, amount_cents, currency, status,
          COALESCE(transaction_id, ''), paid_at, callback_payload, created_at, updated_at`
	item, err := scanPaymentOrder(tx.QueryRow(ctx, updatePayment, tenantID, channel, req.PayNo, req.Status, req.TransactionID, payload))
	if err == pgx.ErrNoRows {
		if eventErr := r.insertPaymentCallbackEvent(ctx, tx, tenantID, "", req.PayNo, channel, req.Status, req.TransactionID, requestID, payload, "", "payment order not found"); eventErr != nil {
			return PaymentOrder{}, eventErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return PaymentOrder{}, commitErr
		}
		return PaymentOrder{}, ErrPaymentNotFound
	}
	if err != nil {
		return PaymentOrder{}, err
	}
	if item.Status == "paid" {
		if err := r.markOrderPaidAndSubscribe(ctx, tx, item.BusinessOrderID); err != nil {
			_ = r.insertPaymentCallbackEvent(ctx, tx, tenantID, item.ID, req.PayNo, channel, req.Status, req.TransactionID, requestID, payload, item.Status, err.Error())
			return PaymentOrder{}, err
		}
	}
	if err := r.insertPaymentCallbackEvent(ctx, tx, tenantID, item.ID, req.PayNo, channel, req.Status, req.TransactionID, requestID, payload, item.Status, ""); err != nil {
		return PaymentOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PaymentOrder{}, err
	}
	return item, nil
}

func (r Repository) insertPaymentCallbackEvent(ctx context.Context, tx pgx.Tx, tenantID, paymentOrderID, payNo, channel, eventStatus, transactionID, requestID, payload, resultStatus, errorMessage string) error {
	const q = `
INSERT INTO payment_callback_events(tenant_id, payment_order_id, pay_no, channel, event_status, transaction_id, request_id, payload, result_status, error_message)
VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8::jsonb, NULLIF($9, ''), NULLIF($10, ''))`
	_, err := tx.Exec(ctx, q, tenantID, paymentOrderID, payNo, channel, eventStatus, transactionID, requestID, payload, resultStatus, errorMessage)
	return err
}

func (r Repository) ListPaymentCallbackEvents(ctx context.Context, tenantID, payNo string, limit int) ([]PaymentCallbackEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args := []any{tenantID, limit}
	where := `WHERE tenant_id = $1`
	limitPlaceholder := "$2"
	if payNo != "" {
		args = []any{tenantID, payNo, limit}
		where += ` AND pay_no = $2`
		limitPlaceholder = "$3"
	}
	q := `
SELECT id::text, tenant_id::text, COALESCE(payment_order_id::text, ''), pay_no, channel, event_status,
       COALESCE(transaction_id, ''), COALESCE(request_id, ''), payload, COALESCE(result_status, ''),
       COALESCE(error_message, ''), created_at
FROM payment_callback_events
` + where + `
ORDER BY created_at DESC
LIMIT ` + limitPlaceholder
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PaymentCallbackEvent, 0)
	for rows.Next() {
		item, err := scanPaymentCallbackEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) markOrderPaidAndSubscribe(ctx context.Context, tx pgx.Tx, orderID string) error {
	const updateOrder = `
UPDATE business_orders
SET status = 'paid', updated_at = now()
WHERE id = $1 AND status <> 'paid'`
	if _, err := tx.Exec(ctx, updateOrder, orderID); err != nil {
		return err
	}
	const createSub = `
INSERT INTO subscriptions(tenant_id, plan_id, status, started_at, expired_at, auto_renew, metadata)
SELECT tenant_id, plan_id, 'active', now(), now() + interval '1 month', false,
       jsonb_build_object('source', 'payment_order', 'business_order_id', id::text)
FROM business_orders
WHERE id = $1 AND plan_id IS NOT NULL
ON CONFLICT DO NOTHING`
	_, err := tx.Exec(ctx, createSub, orderID)
	return err
}

func (r Repository) Record(ctx context.Context, in RecordUsageInput) error {
	if r.DB == nil {
		return nil
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return err
	}
	meta, err := jsonObject(in.Metadata)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO usage_records(tenant_id, subject_type, subject_id, metric, quantity, unit, request_id, metadata)
VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, NULLIF($7, ''), $8::jsonb)`
	_, err = r.DB.Exec(ctx, q, in.TenantID, in.SubjectType, in.SubjectID, in.Metric, in.Quantity, in.Unit, in.RequestID, meta)
	return err
}

func (r Repository) CheckQuota(ctx context.Context, tenantID, metric string, requested float64) (QuotaCheck, error) {
	if r.DB == nil {
		return QuotaCheck{TenantID: tenantID, Metric: metric, Requested: requested, Allowed: true}, nil
	}
	sub, err := r.EnsureFreeSubscription(ctx, tenantID)
	if err != nil {
		return QuotaCheck{}, err
	}
	const q = `
SELECT p.quota, COALESCE((
    SELECT SUM(quantity)::float8
    FROM usage_records u
    WHERE u.tenant_id = s.tenant_id
      AND u.metric = $2
      AND u.occurred_at >= s.started_at
      AND (s.expired_at IS NULL OR u.occurred_at < s.expired_at)
), 0)::float8
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
WHERE s.id = $1`
	var quota map[string]any
	var used float64
	if err := r.DB.QueryRow(ctx, q, sub.ID, metric).Scan(&quota, &used); err != nil {
		return QuotaCheck{}, err
	}
	limit, limited := quotaLimit(quota, metric)
	check := QuotaCheck{
		TenantID:  tenantID,
		PlanCode:  sub.PlanCode,
		Metric:    metric,
		Limit:     limit,
		Used:      used,
		Requested: requested,
		Remaining: normalizeRemaining(limit, used),
		Allowed:   true,
		Limited:   limited && limit > 0,
	}
	if !limited || limit <= 0 {
		return check, nil
	}
	check.Allowed = used+requested <= limit
	check.Remaining = normalizeRemaining(limit, used+requested)
	if !check.Allowed {
		return check, check
	}
	return check, nil
}

func (r Repository) QuotaStatus(ctx context.Context, tenantID string) ([]QuotaCheck, error) {
	metrics := []struct {
		metric string
		name   string
		unit   string
	}{
		{MetricRAGRequests, "RAG 问答请求", "次"},
		{MetricAgentMessages, "Agent 会话消息", "条"},
		{MetricFileUploadBytes, "文件上传容量", "字节"},
		{MetricEmbeddingChunks, "知识切片向量化", "片"},
	}
	items := make([]QuotaCheck, 0, len(metrics))
	for _, metric := range metrics {
		check, err := r.CheckQuota(ctx, tenantID, metric.metric, 0)
		if err != nil {
			return nil, err
		}
		check.Name = metric.name
		check.Unit = metric.unit
		items = append(items, check)
	}
	return items, nil
}

func (r Repository) EnsureQuota(ctx context.Context, tenantID, metric string, requested float64) error {
	if requested <= 0 {
		return nil
	}
	check, err := r.CheckQuota(ctx, tenantID, metric, requested)
	if err != nil {
		return err
	}
	if !check.Allowed {
		return check
	}
	return nil
}

func (r Repository) Summary(ctx context.Context, tenantID string, from, to time.Time) ([]UsageSummaryItem, error) {
	args := []any{tenantID}
	where := `WHERE tenant_id = $1`
	if !from.IsZero() {
		args = append(args, from)
		where += ` AND occurred_at >= $2`
	}
	if !to.IsZero() {
		args = append(args, to)
		if len(args) == 2 {
			where += ` AND occurred_at < $2`
		} else {
			where += ` AND occurred_at < $3`
		}
	}
	q := `
SELECT metric, COALESCE(SUM(quantity), 0)::float8, MIN(unit)
FROM usage_records
` + where + `
GROUP BY metric
ORDER BY metric ASC`
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]UsageSummaryItem, 0)
	for rows.Next() {
		var item UsageSummaryItem
		if err := rows.Scan(&item.Metric, &item.Quantity, &item.Unit); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func jsonObject(v map[string]any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBusinessOrder(row rowScanner) (BusinessOrder, error) {
	var item BusinessOrder
	err := row.Scan(
		&item.ID, &item.TenantID, &item.OrderNo, &item.OrderType, &item.PlanID,
		&item.AmountCents, &item.Currency, &item.Status, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanPaymentOrder(row rowScanner) (PaymentOrder, error) {
	var item PaymentOrder
	err := row.Scan(
		&item.ID, &item.TenantID, &item.BusinessOrderID, &item.PayNo, &item.Channel,
		&item.AmountCents, &item.Currency, &item.Status, &item.TransactionID, &item.PaidAt,
		&item.CallbackPayload, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanPaymentCallbackEvent(row rowScanner) (PaymentCallbackEvent, error) {
	var item PaymentCallbackEvent
	err := row.Scan(
		&item.ID, &item.TenantID, &item.PaymentOrderID, &item.PayNo, &item.Channel,
		&item.EventStatus, &item.TransactionID, &item.RequestID, &item.Payload,
		&item.ResultStatus, &item.ErrorMessage, &item.CreatedAt,
	)
	return item, err
}

var ErrSubscriptionNotFound = pgx.ErrNoRows
var ErrPlanNotFound = errors.New("plan not found")
var ErrOrderNotFound = errors.New("order not found")
var ErrPaymentNotFound = errors.New("payment order not found")
var ErrOrderAlreadyPaid = errors.New("paid order cannot be changed")
var ErrPaymentAlreadyPaid = errors.New("paid payment order cannot be closed")
