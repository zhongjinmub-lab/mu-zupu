package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/internal/module/webhook"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo                  Repository
	Hooks                 webhook.Service
	PaymentCallbackSecret string
}

func NewHandler(repo Repository) Handler {
	return Handler{Repo: repo}
}

func NewHandlerWithWebhook(repo Repository, hooks webhook.Service) Handler {
	return Handler{Repo: repo, Hooks: hooks}
}

func NewHandlerWithWebhookAndPaymentSecret(repo Repository, hooks webhook.Service, paymentCallbackSecret string) Handler {
	return Handler{Repo: repo, Hooks: hooks, PaymentCallbackSecret: paymentCallbackSecret}
}

func (h Handler) ListPlans(c *gin.Context) {
	items, err := h.Repo.ListPlans(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50060, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) CurrentSubscription(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.EnsureFreeSubscription(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50061, err.Error())
		return
	}
	response.OK(c, item)
}

func (h Handler) UsageSummary(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	from, err := parseTimeQuery(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40060, "invalid from time")
		return
	}
	to, err := parseTimeQuery(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40061, "invalid to time")
		return
	}
	items, err := h.Repo.Summary(c.Request.Context(), t.ID, from, to)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50062, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "from": nullableTime(from), "to": nullableTime(to)})
}

func (h Handler) QuotaStatus(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.QuotaStatus(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50066, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) CreateOrder(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.CreateOrder(c.Request.Context(), t.ID, req)
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ListOrders(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListOrders(c.Request.Context(), t.ID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50063, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) CancelOrder(c *gin.Context) {
	h.changeOrderStatus(c, h.Repo.CancelOrder)
}

func (h Handler) CloseOrder(c *gin.Context) {
	h.changeOrderStatus(c, h.Repo.CloseOrder)
}

func (h Handler) changeOrderStatus(c *gin.Context, fn func(ctx context.Context, tenantID, orderID, reason string) (BusinessOrder, error)) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req ChangeOrderStatusRequest
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
	}
	item, err := fn(c.Request.Context(), t.ID, c.Param("order_id"), req.Reason)
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) CreatePaymentOrder(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreatePaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.CreatePaymentOrder(c.Request.Context(), t.ID, req)
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	if item.Status == "paid" {
		h.Hooks.Emit(c.Request.Context(), t.ID, webhook.EventOrderPaid, map[string]any{
			"payment_order_id":  item.ID,
			"business_order_id": item.BusinessOrderID,
			"pay_no":            item.PayNo,
			"channel":           item.Channel,
			"amount_cents":      item.AmountCents,
			"currency":          item.Currency,
			"transaction_id":    item.TransactionID,
			"paid_at":           item.PaidAt,
		})
	}
	response.OK(c, item)
}

func (h Handler) ListPaymentOrders(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListPaymentOrders(c.Request.Context(), t.ID, c.Query("business_order_id"), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50064, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) QueryPayment(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.QueryPaymentOrder(c.Request.Context(), t.ID, c.Param("payment_id"))
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ClosePayment(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req ChangeOrderStatusRequest
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
	}
	item, err := h.Repo.ClosePaymentOrder(c.Request.Context(), t.ID, c.Param("payment_id"), req.Reason)
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) PaymentCallback(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if h.PaymentCallbackSecret != "" {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.Error(c, http.StatusBadRequest, 40003, "读取支付回调请求体失败")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if !verifyPaymentCallbackSignature(h.PaymentCallbackSecret, body, c.GetHeader("X-Payment-Signature")) {
			response.Error(c, http.StatusUnauthorized, 40160, "支付回调签名校验失败")
			return
		}
	}
	var req PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	item, err := h.Repo.ApplyPaymentCallback(c.Request.Context(), t.ID, c.Param("channel"), req, c.GetString("request_id"))
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	response.OK(c, item)
}

func paymentCallbackSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func verifyPaymentCallbackSignature(secret string, body []byte, signature string) bool {
	if strings.TrimSpace(secret) == "" {
		return true
	}
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	expected := paymentCallbackSignature(secret, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (h Handler) ListPaymentCallbackEvents(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListPaymentCallbackEvents(c.Request.Context(), t.ID, c.Query("pay_no"), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50065, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func writeBillingHTTPError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPlanNotFound):
		response.Error(c, http.StatusNotFound, 40460, "plan not found")
	case errors.Is(err, ErrOrderNotFound):
		response.Error(c, http.StatusNotFound, 40461, "order not found")
	case errors.Is(err, ErrPaymentNotFound):
		response.Error(c, http.StatusNotFound, 40462, "payment order not found")
	case errors.Is(err, ErrOrderAlreadyPaid):
		response.Error(c, http.StatusConflict, 40960, "paid order cannot be changed")
	case errors.Is(err, ErrPaymentAlreadyPaid):
		response.Error(c, http.StatusConflict, 40961, "paid payment order cannot be closed")
	default:
		response.Error(c, http.StatusInternalServerError, 50063, err.Error())
	}
}

func parseTimeQuery(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
