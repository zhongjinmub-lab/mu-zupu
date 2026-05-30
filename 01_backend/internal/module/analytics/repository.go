package analytics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) Summary(ctx context.Context, tenantID string) (Summary, error) {
	resource, err := r.resourceSummary(ctx, tenantID)
	if err != nil {
		return Summary{}, err
	}
	business, err := r.businessSummary(ctx, tenantID)
	if err != nil {
		return Summary{}, err
	}
	usageTrend, err := r.usageTrend(ctx, tenantID)
	if err != nil {
		return Summary{}, err
	}
	recentActions, err := r.recentActions(ctx, tenantID)
	if err != nil {
		return Summary{}, err
	}
	risks := buildRisks(resource, business)
	return Summary{
		TenantID:      tenantID,
		GeneratedAt:   time.Now().UTC(),
		Resource:      resource,
		Business:      business,
		UsageTrend:    usageTrend,
		RecentActions: recentActions,
		Risks:         risks,
	}, nil
}

func (r Repository) resourceSummary(ctx context.Context, tenantID string) (ResourceSummary, error) {
	const q = `
SELECT
  (SELECT COUNT(*) FROM knowledge_bases WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM documents WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM document_chunks WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM document_chunks WHERE tenant_id = $1 AND deleted_at IS NULL AND embedding_status <> 'done'),
  (SELECT COUNT(*) FROM document_jobs WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM agents WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM messages WHERE tenant_id = $1),
  (SELECT COUNT(*) FROM tenant_members WHERE tenant_id = $1 AND deleted_at IS NULL AND status = 'active'),
  (SELECT COUNT(*) FROM tenant_invitations WHERE tenant_id = $1 AND deleted_at IS NULL AND status = 'pending')`
	var item ResourceSummary
	err := r.DB.QueryRow(ctx, q, tenantID).Scan(
		&item.KnowledgeBases,
		&item.Files,
		&item.Documents,
		&item.DocumentChunks,
		&item.PendingChunks,
		&item.DocumentJobs,
		&item.Agents,
		&item.Conversations,
		&item.Messages,
		&item.Members,
		&item.Invitations,
	)
	return item, err
}

func (r Repository) businessSummary(ctx context.Context, tenantID string) (BusinessSummary, error) {
	orders, err := r.statusCounts(ctx, `business_orders`, tenantID)
	if err != nil {
		return BusinessSummary{}, err
	}
	payments, err := r.statusCounts(ctx, `payment_orders`, tenantID)
	if err != nil {
		return BusinessSummary{}, err
	}
	licenses, err := r.statusCounts(ctx, `licenses`, tenantID)
	if err != nil {
		return BusinessSummary{}, err
	}
	const totalsQ = `
SELECT
  COALESCE(SUM(CASE WHEN status = 'paid' THEN amount_cents ELSE 0 END), 0)::float8 / 100.0,
  COUNT(*) FILTER (WHERE status = 'paid'),
  COUNT(*) FILTER (WHERE status = 'pending')
FROM business_orders
WHERE tenant_id = $1 AND deleted_at IS NULL`
	var item BusinessSummary
	item.Orders = orders
	item.Payments = payments
	item.Licenses = licenses
	if err := r.DB.QueryRow(ctx, totalsQ, tenantID).Scan(&item.TotalRevenueCNY, &item.PaidOrders, &item.PendingOrders); err != nil {
		return BusinessSummary{}, err
	}
	const failedQ = `SELECT COUNT(*) FROM payment_orders WHERE tenant_id = $1 AND deleted_at IS NULL AND status IN ('failed', 'closed')`
	if err := r.DB.QueryRow(ctx, failedQ, tenantID).Scan(&item.FailedPayments); err != nil {
		return BusinessSummary{}, err
	}
	return item, nil
}

func (r Repository) statusCounts(ctx context.Context, table, tenantID string) ([]StatusCount, error) {
	q := `SELECT status, COUNT(*) FROM ` + table + ` WHERE tenant_id = $1 AND deleted_at IS NULL GROUP BY status ORDER BY status ASC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]StatusCount, 0)
	for rows.Next() {
		var item StatusCount
		if err := rows.Scan(&item.Status, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) usageTrend(ctx context.Context, tenantID string) ([]UsageTrendItem, error) {
	const q = `
SELECT to_char(date_trunc('day', occurred_at), 'YYYY-MM-DD') AS day,
       metric,
       COALESCE(SUM(quantity), 0)::float8,
       MIN(unit)
FROM usage_records
WHERE tenant_id = $1 AND occurred_at >= now() - interval '6 days'
GROUP BY day, metric
ORDER BY day ASC, metric ASC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]UsageTrendItem, 0)
	for rows.Next() {
		var item UsageTrendItem
		if err := rows.Scan(&item.Date, &item.Metric, &item.Quantity, &item.Unit); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) recentActions(ctx context.Context, tenantID string) ([]RecentActionItem, error) {
	const q = `
SELECT action, COALESCE(resource_type, ''), COALESCE(actor_user_id::text, ''), created_at
FROM audit_logs
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT 10`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RecentActionItem, 0)
	for rows.Next() {
		var item RecentActionItem
		if err := rows.Scan(&item.Action, &item.ResourceType, &item.ActorUserID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func buildRisks(resource ResourceSummary, business BusinessSummary) []RiskItem {
	risks := make([]RiskItem, 0, 4)
	if resource.PendingChunks > 0 {
		risks = append(risks, RiskItem{Level: "warn", Code: "pending_chunks", Message: "存在待向量化 Chunk", Count: resource.PendingChunks})
	}
	if resource.Invitations > 0 {
		risks = append(risks, RiskItem{Level: "info", Code: "pending_invitations", Message: "存在待处理租户邀请", Count: resource.Invitations})
	}
	if business.PendingOrders > 0 {
		risks = append(risks, RiskItem{Level: "info", Code: "pending_orders", Message: "存在待支付订单", Count: business.PendingOrders})
	}
	if business.FailedPayments > 0 {
		risks = append(risks, RiskItem{Level: "warn", Code: "failed_payments", Message: "存在异常或关闭支付单", Count: business.FailedPayments})
	}
	if len(risks) == 0 {
		risks = append(risks, RiskItem{Level: "ok", Code: "healthy", Message: "当前租户无明显运营风险", Count: 0})
	}
	return risks
}
