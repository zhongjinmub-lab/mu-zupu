package license

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo     Repository
	Verifier Verifier
}

func NewHandler(repo Repository, verifier Verifier) Handler {
	return Handler{Repo: repo, Verifier: verifier}
}

func (h Handler) List(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.List(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50070, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) Create(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req CreateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	if req.Signature != "" || req.PublicKeyID != "" {
		item := License{
			TenantID:    t.ID,
			LicenseNo:   req.LicenseNo,
			LicenseType: req.LicenseType,
			Subject:     req.Subject,
			Limits:      req.Limits,
			PublicKeyID: req.PublicKeyID,
			Signature:   req.Signature,
			ExpiredAt:   req.ExpiredAt,
		}
		if item.LicenseNo == "" {
			response.Error(c, http.StatusBadRequest, 40073, "license_no is required for signed license")
			return
		}
		result := h.Verifier.Verify(item)
		if !result.Valid {
			response.Error(c, http.StatusBadRequest, 40074, result.Message)
			return
		}
	}
	item, err := h.Repo.Create(c.Request.Context(), t.ID, req)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40970, "license_no already exists")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50071, err.Error())
		return
	}
	response.OK(c, item)
}

func (h Handler) Verify(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("license_id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(c, http.StatusNotFound, 40470, "license not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50072, err.Error())
		return
	}
	response.OK(c, h.Verifier.Verify(item))
}

func (h Handler) Activate(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("license_id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(c, http.StatusNotFound, 40470, "license not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50072, err.Error())
		return
	}
	if item.Signature != "" || item.PublicKeyID != "" {
		result := h.Verifier.Verify(item)
		if !result.Valid {
			response.Error(c, http.StatusBadRequest, 40074, result.Message)
			return
		}
	}
	item, err = h.Repo.Activate(c.Request.Context(), t.ID, item.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(c, http.StatusNotFound, 40470, "license not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50072, err.Error())
		return
	}
	response.OK(c, item)
}

func (h Handler) Revoke(c *gin.Context) {
	h.changeStatus(c, h.Repo.Revoke)
}

func (h Handler) changeStatus(c *gin.Context, fn func(ctx context.Context, tenantID, licenseID string) (License, error)) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := fn(c.Request.Context(), t.ID, c.Param("license_id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(c, http.StatusNotFound, 40470, "license not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50072, err.Error())
		return
	}
	response.OK(c, item)
}
