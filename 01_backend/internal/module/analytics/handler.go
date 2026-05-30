package analytics

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo Repository
}

func NewHandler(repo Repository) Handler {
	return Handler{Repo: repo}
}

func (h Handler) Summary(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Summary(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50090, err.Error())
		return
	}
	response.OK(c, item)
}
