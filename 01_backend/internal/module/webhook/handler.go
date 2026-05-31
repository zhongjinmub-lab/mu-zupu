package webhook

import (
	"errors"
	"net/http"
	"strconv"

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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	query := DeliveryQuery{
		TenantID:   t.ID,
		EndpointID: c.Query("endpoint_id"),
		EventType:  c.Query("event_type"),
		Status:     c.Query("status"),
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
