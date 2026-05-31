package webhook

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo    Repository
	Service Service
}

func NewHandler(repo Repository, service Service) Handler {
	return Handler{Repo: repo, Service: service}
}

func (h Handler) ListEndpoints(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.ListEndpoints(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50090, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) CreateEndpoint(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.CreateEndpoint(c.Request.Context(), t.ID, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50091, err.Error())
		return
	}
	response.OK(c, item)
}

func (h Handler) UpdateEndpoint(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req UpdateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.UpdateEndpoint(c.Request.Context(), t.ID, c.Param("webhook_id"), req)
	if err != nil {
		writeWebhookError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) DeleteEndpoint(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.Repo.DeleteEndpoint(c.Request.Context(), t.ID, c.Param("webhook_id")); err != nil {
		writeWebhookError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h Handler) TestEndpoint(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Service.Test(c.Request.Context(), t.ID, c.Param("webhook_id"))
	if err != nil {
		writeWebhookError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) ListDeliveries(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	from, err := parseDeliveryTime(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40002, "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	to, err := parseDeliveryTime(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40002, "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	query := DeliveryQuery{
		TenantID:   t.ID,
		EndpointID: c.Query("endpoint_id"),
		EventType:  c.Query("event_type"),
		Status:     c.Query("status"),
		From:       from,
		To:         to,
		Limit:      limit,
	}
	query.Normalize()
	if err := query.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	items, err := h.Repo.ListDeliveries(c.Request.Context(), query)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50092, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) DeliverySummary(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	query, err := h.deliveryQueryFromRequest(c, t.ID, 50)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	query.Normalize()
	if err := query.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.DeliverySummary(c.Request.Context(), query)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50092, err.Error())
		return
	}
	response.OK(c, item)
}

func (h Handler) ExportDeliveries(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	from, err := parseDeliveryTime(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40002, "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	to, err := parseDeliveryTime(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40002, "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	query := DeliveryQuery{
		TenantID:   t.ID,
		EndpointID: c.Query("endpoint_id"),
		EventType:  c.Query("event_type"),
		Status:     c.Query("status"),
		From:       from,
		To:         to,
		Limit:      limit,
	}
	query.NormalizeForExport()
	if err := query.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	items, err := h.Repo.ExportDeliveries(c.Request.Context(), query)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50092, err.Error())
		return
	}
	filename := "webhook-deliveries-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "tenant_id", "endpoint_id", "event_type", "target_url", "status", "http_status", "duration_ms", "retry_count", "next_retry_at", "last_attempt_at", "error_message", "response_body", "request_body", "created_at"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.ID,
			item.TenantID,
			item.EndpointID,
			item.EventType,
			item.TargetURL,
			item.Status,
			strconv.Itoa(item.HTTPStatus),
			strconv.FormatInt(item.DurationMS, 10),
			strconv.Itoa(item.RetryCount),
			formatOptionalTime(item.NextRetryAt),
			formatOptionalTime(item.LastAttemptAt),
			item.ErrorMessage,
			item.ResponseBody,
			jsonMapString(item.RequestBody),
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func (h Handler) RetryDelivery(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	deliveryID := c.Param("delivery_id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		writeWebhookError(c, ErrDeliveryIDMustBeUUID)
		return
	}
	item, err := h.Service.RetryDelivery(c.Request.Context(), t.ID, deliveryID)
	if err != nil {
		writeWebhookError(c, err)
		return
	}
	response.OK(c, item)
}

func (h Handler) deliveryQueryFromRequest(c *gin.Context, tenantID string, defaultLimit int) (DeliveryQuery, error) {
	from, err := parseDeliveryTime(c.Query("from"))
	if err != nil {
		return DeliveryQuery{}, errors.New("from must be RFC3339 or YYYY-MM-DD")
	}
	to, err := parseDeliveryTime(c.Query("to"))
	if err != nil {
		return DeliveryQuery{}, errors.New("to must be RFC3339 or YYYY-MM-DD")
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	return DeliveryQuery{
		TenantID:   tenantID,
		EndpointID: c.Query("endpoint_id"),
		EventType:  c.Query("event_type"),
		Status:     c.Query("status"),
		From:       from,
		To:         to,
		Limit:      limit,
	}, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func jsonMapString(value map[string]any) string {
	if value == nil {
		return "{}"
	}
	b, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func parseDeliveryTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, errors.New("invalid time")
}

func writeWebhookError(c *gin.Context, err error) {
	if errors.Is(err, ErrEndpointNotFound) {
		response.Error(c, http.StatusNotFound, 40490, "webhook endpoint not found")
		return
	}
	if errors.Is(err, ErrDeliveryNotFound) {
		response.Error(c, http.StatusNotFound, 40491, "webhook delivery not found")
		return
	}
	if errors.Is(err, ErrDeliveryNotRetryable) {
		response.Error(c, http.StatusBadRequest, 40093, "webhook delivery is not retryable")
		return
	}
	if errors.Is(err, ErrDeliveryIDMustBeUUID) || errors.Is(err, ErrEndpointIDMustBeUUID) {
		response.Error(c, http.StatusBadRequest, 40094, err.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, 50090, err.Error())
}
