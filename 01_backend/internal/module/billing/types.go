package billing

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	MetricRAGRequests     = "rag_requests"
	MetricAgentMessages   = "agent_messages"
	MetricFileUploadBytes = "file_upload_bytes"
	MetricEmbeddingChunks = "embedding_chunks"
)

type Plan struct {
	ID           string         `json:"id"`
	Code         string         `json:"code"`
	Name         string         `json:"name"`
	PriceCents   int64          `json:"price_cents"`
	BillingCycle string         `json:"billing_cycle"`
	Quota        map[string]any `json:"quota"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type Subscription struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	PlanID    string         `json:"plan_id"`
	PlanCode  string         `json:"plan_code,omitempty"`
	PlanName  string         `json:"plan_name,omitempty"`
	Status    string         `json:"status"`
	StartedAt time.Time      `json:"started_at"`
	ExpiredAt *time.Time     `json:"expired_at,omitempty"`
	AutoRenew bool           `json:"auto_renew"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type UsageRecord struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	SubjectType string         `json:"subject_type"`
	SubjectID   string         `json:"subject_id,omitempty"`
	Metric      string         `json:"metric"`
	Quantity    float64        `json:"quantity"`
	Unit        string         `json:"unit"`
	RequestID   string         `json:"request_id,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	OccurredAt  time.Time      `json:"occurred_at"`
	CreatedAt   time.Time      `json:"created_at"`
}

type UsageSummaryItem struct {
	Metric   string  `json:"metric"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type BusinessOrder struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	OrderNo     string         `json:"order_no"`
	OrderType   string         `json:"order_type"`
	PlanID      string         `json:"plan_id,omitempty"`
	AmountCents int64          `json:"amount_cents"`
	Currency    string         `json:"currency"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type PaymentOrder struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	BusinessOrderID string         `json:"business_order_id"`
	PayNo           string         `json:"pay_no"`
	Channel         string         `json:"channel"`
	AmountCents     int64          `json:"amount_cents"`
	Currency        string         `json:"currency"`
	Status          string         `json:"status"`
	TransactionID   string         `json:"transaction_id,omitempty"`
	PaidAt          *time.Time     `json:"paid_at,omitempty"`
	CallbackPayload map[string]any `json:"callback_payload"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type PaymentCallbackEvent struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	PaymentOrderID string         `json:"payment_order_id,omitempty"`
	PayNo          string         `json:"pay_no"`
	Channel        string         `json:"channel"`
	EventStatus    string         `json:"event_status"`
	TransactionID  string         `json:"transaction_id,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	Payload        map[string]any `json:"payload"`
	ResultStatus   string         `json:"result_status,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

type CreateOrderRequest struct {
	OrderType   string         `json:"order_type"`
	PlanCode    string         `json:"plan_code"`
	AmountCents int64          `json:"amount_cents"`
	Currency    string         `json:"currency"`
	Metadata    map[string]any `json:"metadata"`
}

type CreatePaymentOrderRequest struct {
	BusinessOrderID string `json:"business_order_id" binding:"required"`
	Channel         string `json:"channel"`
}

type PaymentCallbackRequest struct {
	PayNo         string         `json:"pay_no" binding:"required"`
	TransactionID string         `json:"transaction_id"`
	Status        string         `json:"status"`
	Metadata      map[string]any `json:"metadata"`
}

type ChangeOrderStatusRequest struct {
	Reason string `json:"reason"`
}

func (r *CreateOrderRequest) Normalize() {
	r.OrderType = strings.TrimSpace(r.OrderType)
	r.PlanCode = strings.TrimSpace(r.PlanCode)
	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	if r.OrderType == "" {
		r.OrderType = "subscription"
	}
	if r.Currency == "" {
		r.Currency = "CNY"
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
}

func (r CreateOrderRequest) Validate() error {
	if r.OrderType != "subscription" {
		return errors.New("order_type must be subscription")
	}
	if r.PlanCode == "" {
		return errors.New("plan_code is required")
	}
	if r.AmountCents < 0 {
		return errors.New("amount_cents must be non-negative")
	}
	return nil
}

func (r *CreatePaymentOrderRequest) Normalize() {
	r.BusinessOrderID = strings.TrimSpace(r.BusinessOrderID)
	r.Channel = strings.ToLower(strings.TrimSpace(r.Channel))
	if r.Channel == "" {
		r.Channel = "mock"
	}
}

func (r CreatePaymentOrderRequest) Validate() error {
	if r.BusinessOrderID == "" {
		return errors.New("business_order_id is required")
	}
	if r.Channel != "mock" {
		return errors.New("only mock payment channel is supported in MVP")
	}
	return nil
}

func (r *PaymentCallbackRequest) Normalize() {
	r.PayNo = strings.TrimSpace(r.PayNo)
	r.TransactionID = strings.TrimSpace(r.TransactionID)
	r.Status = strings.ToLower(strings.TrimSpace(r.Status))
	if r.Status == "" {
		r.Status = "paid"
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
}

func (r PaymentCallbackRequest) Validate() error {
	if r.PayNo == "" {
		return errors.New("pay_no is required")
	}
	if r.Status != "paid" && r.Status != "failed" {
		return errors.New("status must be paid or failed")
	}
	return nil
}

func normalizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 512 {
		return reason[:512]
	}
	return reason
}

type QuotaCheck struct {
	TenantID  string  `json:"tenant_id"`
	PlanCode  string  `json:"plan_code"`
	Metric    string  `json:"metric"`
	Name      string  `json:"name,omitempty"`
	Limit     float64 `json:"limit"`
	Used      float64 `json:"used"`
	Requested float64 `json:"requested"`
	Remaining float64 `json:"remaining"`
	Allowed   bool    `json:"allowed"`
	Limited   bool    `json:"limited"`
	Unit      string  `json:"unit,omitempty"`
}

func newNo(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + time.Now().UTC().Format("20060102150405")
	}
	return prefix + strings.ToUpper(hex.EncodeToString(b[:]))
}

type RecordUsageInput struct {
	TenantID    string
	SubjectType string
	SubjectID   string
	Metric      string
	Quantity    float64
	Unit        string
	RequestID   string
	Metadata    map[string]any
}

func (in *RecordUsageInput) Normalize() {
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.SubjectType = strings.TrimSpace(in.SubjectType)
	in.SubjectID = strings.TrimSpace(in.SubjectID)
	in.Metric = strings.TrimSpace(in.Metric)
	in.Unit = strings.TrimSpace(in.Unit)
	in.RequestID = strings.TrimSpace(in.RequestID)
	if in.Unit == "" {
		in.Unit = "count"
	}
}

func (in RecordUsageInput) Validate() error {
	if in.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if in.SubjectType == "" {
		return errors.New("subject_type is required")
	}
	if in.Metric == "" {
		return errors.New("metric is required")
	}
	if in.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	return nil
}

func (q QuotaCheck) Error() string {
	return fmt.Sprintf("quota exceeded for %s: used %.0f, requested %.0f, limit %.0f", q.Metric, q.Used, q.Requested, q.Limit)
}

func IsQuotaExceeded(err error) (QuotaCheck, bool) {
	var check QuotaCheck
	if errors.As(err, &check) {
		return check, true
	}
	return QuotaCheck{}, false
}

func quotaLimit(quota map[string]any, metric string) (float64, bool) {
	if quota == nil {
		return 0, false
	}
	value, ok := quota[metric]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case jsonNumber:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func normalizeRemaining(limit, used float64) float64 {
	remaining := limit - used
	if remaining < 0 || math.IsNaN(remaining) {
		return 0
	}
	return remaining
}

type jsonNumber interface {
	Float64() (float64, error)
}
