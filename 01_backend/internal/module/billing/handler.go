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
	"mu-agent-saas/internal/payment"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo                  Repository
	Hooks                 webhook.Service
	PaymentCallbackSecret string
	Payments              *payment.Registry
	NotifyBaseURL         string
	ReturnURL             string
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

// NewHandlerWithDeps 注入支付渠道注册表与回调基础地址,支持真实第三方渠道接入。
func NewHandlerWithDeps(repo Repository, hooks webhook.Service, paymentCallbackSecret string, payments *payment.Registry, notifyBaseURL, returnURL string) Handler {
	return Handler{
		Repo:                  repo,
		Hooks:                 hooks,
		PaymentCallbackSecret: paymentCallbackSecret,
		Payments:              payments,
		NotifyBaseURL:         strings.TrimRight(strings.TrimSpace(notifyBaseURL), "/"),
		ReturnURL:             strings.TrimSpace(returnURL),
	}
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
	if h.Payments != nil && !h.Payments.IsEnabled(req.Channel) {
		response.Error(c, http.StatusBadRequest, 40002, "不支持的支付渠道: "+req.Channel)
		return
	}
	item, err := h.Repo.CreatePaymentOrder(c.Request.Context(), t.ID, req)
	if err != nil {
		writeBillingHTTPError(c, err)
		return
	}
	resp := CreatePaymentResponse{PaymentOrder: item}
	if provider, ok := h.Payments.Get(req.Channel); ok {
		prepay, perr := provider.CreatePrepay(c.Request.Context(), payment.PrepayInput{
			PayNo:       item.PayNo,
			AmountCents: item.AmountCents,
			Currency:    item.Currency,
			Subject:     "订单支付 " + item.PayNo,
			NotifyURL:   h.notifyURL(req.Channel),
			ReturnURL:   h.ReturnURL,
		})
		if perr == nil {
			resp.Prepay = &prepay
			_ = h.Repo.AttachPrepay(c.Request.Context(), t.ID, item.ID, prepaySummary(prepay))
		}
	}
	if item.Status == "paid" {
		h.emitOrderPaid(c.Request.Context(), t.ID, item)
	}
	response.OK(c, resp)
}

func (h Handler) notifyURL(channel string) string {
	if h.NotifyBaseURL == "" {
		return ""
	}
	return h.NotifyBaseURL + "/api/v1/payment-notify/" + channel
}

// prepaySummary 仅提取非敏感字段用于持久化(签名仅为请求签名,非可复用凭据)。
func prepaySummary(p payment.Prepay) map[string]any {
	summary := map[string]any{
		"channel": p.Channel,
		"method":  p.Method,
	}
	if p.PayURL != "" {
		summary["pay_url"] = p.PayURL
	}
	if p.QRContent != "" {
		summary["qr_content"] = p.QRContent
	}
	if p.Message != "" {
		summary["message"] = p.Message
	}
	return summary
}

// PaymentNotify 处理第三方支付渠道的公开异步通知(无 JWT/租户上下文)。
// 渠道 Provider 负责原生验签;通过 pay_no 反查租户后复用既有回调入账逻辑。
func (h Handler) PaymentNotify(c *gin.Context) {
	channel := strings.ToLower(strings.TrimSpace(c.Param("channel")))
	provider, ok := h.Payments.Get(channel)
	if !ok {
		response.Error(c, http.StatusNotFound, 40463, "支付渠道未启用")
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		writeNotifyResponse(c, provider, false)
		return
	}
	res, err := provider.VerifyNotify(c.Request.Context(), payment.Notify{
		Header:      c.Request.Header,
		Body:        body,
		ContentType: c.ContentType(),
	})
	if err != nil {
		writeNotifyResponse(c, provider, false)
		return
	}
	tenantID, err := h.Repo.GetTenantIDByPayNo(c.Request.Context(), res.PayNo)
	if err != nil {
		writeNotifyResponse(c, provider, false)
		return
	}
	item, err := h.Repo.ApplyPaymentCallback(c.Request.Context(), tenantID, channel, PaymentCallbackRequest{
		PayNo:         res.PayNo,
		TransactionID: res.TransactionID,
		Status:        res.Status,
		Metadata:      res.Raw,
	}, c.GetString("request_id"))
	if err != nil {
		writeNotifyResponse(c, provider, false)
		return
	}
	if item.Status == "paid" {
		h.emitOrderPaid(c.Request.Context(), tenantID, item)
	}
	writeNotifyResponse(c, provider, true)
}

func writeNotifyResponse(c *gin.Context, provider payment.Provider, ok bool) {
	status, contentType, body := provider.NotifyResponse(ok)
	c.Data(status, contentType, []byte(body))
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
	resp := QueryPaymentResponse{PaymentOrder: item}
	// 非终态时尝试向渠道主动查单对账(异步通知丢失时的兜底)。
	if item.Status != "paid" && item.Status != "closed" {
		if provider, ok := h.Payments.Get(item.Channel); ok {
			if querier, ok := provider.(payment.OrderQuerier); ok {
				remote, qerr := querier.QueryOrder(c.Request.Context(), item.PayNo)
				if qerr == nil {
					resp.RemoteStatus = remote.Status
					if remote.Status == payment.QueryStatusPaid || remote.Status == payment.QueryStatusFailed {
						synced, serr := h.Repo.ApplyPaymentCallback(c.Request.Context(), t.ID, item.Channel, PaymentCallbackRequest{
							PayNo:         item.PayNo,
							TransactionID: remote.TransactionID,
							Status:        remote.Status,
							Metadata:      remote.Raw,
						}, c.GetString("request_id"))
						if serr == nil {
							resp.PaymentOrder = synced
							resp.Reconciled = synced.Status != item.Status
							item = synced
							if item.Status == "paid" {
								h.emitOrderPaid(c.Request.Context(), t.ID, item)
							}
						}
					}
				}
			}
		}
	}
	response.OK(c, resp)
}

func (h Handler) emitOrderPaid(ctx context.Context, tenantID string, item PaymentOrder) {
	h.Hooks.Emit(ctx, tenantID, webhook.EventOrderPaid, map[string]any{
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
