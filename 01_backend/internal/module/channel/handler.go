package channel

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

// Handler 提供渠道类型目录与渠道接入点的管理接口。
type Handler struct {
	Repo Repository
}

// NewHandler 构造渠道 Handler。
func NewHandler(repo Repository) Handler {
	return Handler{Repo: repo}
}

// ListChannelTypes 返回内置渠道类型目录。
func (h Handler) ListChannelTypes(c *gin.Context) {
	if _, ok := tenant.CurrentTenant(c); !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, gin.H{"items": DefaultChannelTypes()})
}

// ListChannels 返回当前租户的渠道接入点列表。
func (h Handler) ListChannels(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	items, err := h.Repo.List(c.Request.Context(), t.ID)
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

// GetChannel 返回指定渠道详情。
func (h Handler) GetChannel(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("channel_id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, item)
}

// CreateChannel 创建渠道接入点。
func (h Handler) CreateChannel(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	u, _ := auth.CurrentUser(c)
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.Create(c.Request.Context(), t.ID, u.ID, req)
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, item)
}

// EnableChannel 启用渠道。
func (h Handler) EnableChannel(c *gin.Context) {
	h.setChannelStatus(c, StatusEnabled)
}

// DisableChannel 禁用渠道。
func (h Handler) DisableChannel(c *gin.Context) {
	h.setChannelStatus(c, StatusDisabled)
}

func (h Handler) setChannelStatus(c *gin.Context, status string) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.SetStatus(c.Request.Context(), t.ID, c.Param("channel_id"), status)
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, item)
}

// ArchiveChannel 归档渠道。
func (h Handler) ArchiveChannel(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if err := h.Repo.Archive(c.Request.Context(), t.ID, c.Param("channel_id")); err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, gin.H{"status": "archived"})
}

// writeChannelError 统一处理渠道相关错误：未找到 404，外键缺失（agent 不存在）404，唯一冲突 409。
func writeChannelError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrChannelNotFound):
		response.Error(c, http.StatusNotFound, 40445, "channel not found")
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				response.Error(c, http.StatusNotFound, 40446, "bound agent not found")
				return
			case "23505":
				response.Error(c, http.StatusConflict, 40944, "channel key already exists")
				return
			}
		}
		response.Error(c, http.StatusInternalServerError, 50046, err.Error())
	}
}

// ChannelEmbed 返回指定渠道的接入代码与说明（按当前请求推断 baseURL）。
func (h Handler) ChannelEmbed(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("channel_id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + c.Request.Host
	response.OK(c, BuildChannelEmbed(item, baseURL))
}

// ConnectChannel 是面向外部接入方的公开端点：按 channel_key 返回已启用渠道的接入配置，
// 不经登录与租户中间件，仅返回接入所需的最小信息。
func (h Handler) ConnectChannel(c *gin.Context) {
	key := strings.TrimSpace(c.Param("channel_key"))
	if key == "" {
		response.Error(c, http.StatusBadRequest, 40002, "channel_key is required")
		return
	}
	item, err := h.Repo.GetByChannelKey(c.Request.Context(), key)
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, gin.H{
		"channel_key": item.ChannelKey,
		"type":        item.Type,
		"name":        item.Name,
		"agent_id":    item.AgentID,
		"status":      item.Status,
		"connected":   true,
	})
}

// UpdateChannel 更新渠道名称与配置。
func (h Handler) UpdateChannel(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.Update(c.Request.Context(), t.ID, c.Param("channel_id"), req)
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, item)
}
