package analytics

import "time"

type Summary struct {
	TenantID      string             `json:"tenant_id"`
	GeneratedAt   time.Time          `json:"generated_at"`
	Resource      ResourceSummary    `json:"resource"`
	Business      BusinessSummary    `json:"business"`
	UsageTrend    []UsageTrendItem   `json:"usage_trend"`
	RecentActions []RecentActionItem `json:"recent_actions"`
	Risks         []RiskItem         `json:"risks"`
}

type ResourceSummary struct {
	KnowledgeBases int64 `json:"knowledge_bases"`
	Files          int64 `json:"files"`
	Documents      int64 `json:"documents"`
	DocumentChunks int64 `json:"document_chunks"`
	PendingChunks  int64 `json:"pending_chunks"`
	DocumentJobs   int64 `json:"document_jobs"`
	Agents         int64 `json:"agents"`
	Conversations  int64 `json:"conversations"`
	Messages       int64 `json:"messages"`
	Members        int64 `json:"members"`
	Invitations    int64 `json:"invitations"`
}

type BusinessSummary struct {
	Orders          []StatusCount `json:"orders"`
	Payments        []StatusCount `json:"payments"`
	Licenses        []StatusCount `json:"licenses"`
	TotalRevenueCNY float64       `json:"total_revenue_cny"`
	PaidOrders      int64         `json:"paid_orders"`
	PendingOrders   int64         `json:"pending_orders"`
	FailedPayments  int64         `json:"failed_payments"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type UsageTrendItem struct {
	Date     string  `json:"date"`
	Metric   string  `json:"metric"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type RecentActionItem struct {
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type,omitempty"`
	ActorUserID  string    `json:"actor_user_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type RiskItem struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Count   int64  `json:"count"`
}
