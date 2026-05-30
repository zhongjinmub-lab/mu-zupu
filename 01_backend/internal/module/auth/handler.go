package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Service Service
}

func NewHandler(service Service) Handler {
	return Handler{Service: service}
}

func (h Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	res, err := h.Service.Register(c.Request.Context(), req)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40901, "email already registered")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}
	response.OK(c, res)
}

func (h Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	res, err := h.Service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, 40104, "invalid email or password")
			return
		}
		response.Error(c, http.StatusUnauthorized, 40105, err.Error())
		return
	}
	response.OK(c, res)
}

func (h Handler) Me(c *gin.Context) {
	u, ok := CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	response.OK(c, u)
}

func RegisterRoutes(public, private *gin.RouterGroup, h Handler) {
	g := public.Group("/auth")
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
	private.GET("/auth/me", h.Me)
}
