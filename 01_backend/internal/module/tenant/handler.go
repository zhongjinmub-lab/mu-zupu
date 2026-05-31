package tenant

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo Repository
}

func jsonString(v map[string]any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func NewHandler(repo Repository) Handler {
	return Handler{Repo: repo}
}

func (h Handler) Create(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	t, err := h.Repo.Create(c.Request.Context(), user.ID, req.Name, normalizeCode(req.Code))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, 40910, "tenant code already exists")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50010, err.Error())
		return
	}
	response.OK(c, t)
}

func (h Handler) List(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	items, err := h.Repo.ListByUser(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50010, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) ListMembers(c *gin.Context) {
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListMembers(c.Request.Context(), t.ID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50011, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) RolePermissions(c *gin.Context) {
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	response.OK(c, RolePermissions(t.RoleCode))
}

func (h Handler) AddMember(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if !CanManageMembers(t.RoleCode) {
		response.Error(c, http.StatusForbidden, 40302, "tenant admin role is required")
		return
	}
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.AddMember(c.Request.Context(), t.ID, req)
	if err != nil {
		writeTenantHTTPError(c, err)
		return
	}
	_ = h.Repo.InsertAuditLog(c.Request.Context(), t.ID, user.ID, "tenant.member.add", "tenant_member", item.ID, c.ClientIP(), c.Request.UserAgent(), map[string]any{"email": item.Email, "role_code": item.RoleCode})
	response.OK(c, item)
}

func (h Handler) UpdateMemberRole(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if !CanManageMembers(t.RoleCode) {
		response.Error(c, http.StatusForbidden, 40302, "tenant admin role is required")
		return
	}
	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.UpdateMemberRole(c.Request.Context(), t.ID, c.Param("member_id"), req)
	if err != nil {
		writeTenantHTTPError(c, err)
		return
	}
	_ = h.Repo.InsertAuditLog(c.Request.Context(), t.ID, user.ID, "tenant.member.role_update", "tenant_member", item.ID, c.ClientIP(), c.Request.UserAgent(), map[string]any{"role_code": item.RoleCode})
	response.OK(c, item)
}

func (h Handler) RemoveMember(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if !CanManageMembers(t.RoleCode) {
		response.Error(c, http.StatusForbidden, 40302, "tenant admin role is required")
		return
	}
	memberID := c.Param("member_id")
	if err := h.Repo.RemoveMember(c.Request.Context(), t.ID, memberID); err != nil {
		writeTenantHTTPError(c, err)
		return
	}
	_ = h.Repo.InsertAuditLog(c.Request.Context(), t.ID, user.ID, "tenant.member.remove", "tenant_member", memberID, c.ClientIP(), c.Request.UserAgent(), nil)
	response.OK(c, gin.H{"id": memberID, "removed": true})
}

func (h Handler) ListAuditLogs(c *gin.Context) {
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	from, err := ParseAuditLogTime(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	to, err := ParseAuditLogTime(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	q := AuditLogQuery{
		TenantID:     t.ID,
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ActorUserID:  c.Query("actor_user_id"),
		From:         from,
		To:           to,
		CursorRaw:    c.Query("cursor"),
		Limit:        limit,
	}
	q.Normalize()
	if err := q.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40003, err.Error())
		return
	}
	items, nextCursor, err := h.Repo.ListAuditLogs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50012, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "next_cursor": nextCursor})
}

func (h Handler) ExportAuditLogs(c *gin.Context) {
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	from, err := ParseAuditLogTime(c.Query("from"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	to, err := ParseAuditLogTime(c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40003, "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	q := AuditLogQuery{
		TenantID:     t.ID,
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ActorUserID:  c.Query("actor_user_id"),
		From:         from,
		To:           to,
		Limit:        1000,
	}
	q.Normalize()
	q.Limit = 1000
	if err := q.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40003, err.Error())
		return
	}
	items, err := h.Repo.ExportAuditLogs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50012, err.Error())
		return
	}
	filename := "audit-logs-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "tenant_id", "actor_user_id", "action", "resource_type", "resource_id", "ip", "user_agent", "metadata", "created_at"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.ID,
			item.TenantID,
			item.ActorUserID,
			item.Action,
			item.ResourceType,
			item.ResourceID,
			item.IP,
			item.UserAgent,
			jsonString(item.Metadata),
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func (h Handler) CreateInvitation(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if !CanManageMembers(t.RoleCode) {
		response.Error(c, http.StatusForbidden, 40302, "tenant admin role is required")
		return
	}
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	item, err := h.Repo.CreateInvitation(c.Request.Context(), t.ID, user.ID, req)
	if err != nil {
		writeTenantHTTPError(c, err)
		return
	}
	_ = h.Repo.InsertAuditLog(c.Request.Context(), t.ID, user.ID, "tenant.invitation.create", "tenant_invitation", item.ID, c.ClientIP(), c.Request.UserAgent(), map[string]any{"email": item.Email, "role_code": item.RoleCode})
	response.OK(c, item)
}

func (h Handler) ListInvitations(c *gin.Context) {
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if !CanManageMembers(t.RoleCode) {
		response.Error(c, http.StatusForbidden, 40302, "tenant admin role is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.ListInvitations(c.Request.Context(), t.ID, c.Query("status"), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50013, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) RevokeInvitation(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	t, ok := CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	if !CanManageMembers(t.RoleCode) {
		response.Error(c, http.StatusForbidden, 40302, "tenant admin role is required")
		return
	}
	item, err := h.Repo.RevokeInvitation(c.Request.Context(), t.ID, c.Param("invitation_id"))
	if err != nil {
		writeTenantHTTPError(c, err)
		return
	}
	_ = h.Repo.InsertAuditLog(c.Request.Context(), t.ID, user.ID, "tenant.invitation.revoke", "tenant_invitation", item.ID, c.ClientIP(), c.Request.UserAgent(), map[string]any{"email": item.Email})
	response.OK(c, item)
}

func (h Handler) AcceptInvitation(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}
	var req AcceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, 40002, err.Error())
		return
	}
	t, err := h.Repo.AcceptInvitation(c.Request.Context(), user.ID, user.Email, req)
	if err != nil {
		writeTenantHTTPError(c, err)
		return
	}
	_ = h.Repo.InsertAuditLog(c.Request.Context(), t.ID, user.ID, "tenant.invitation.accept", "tenant_invitation", "", c.ClientIP(), c.Request.UserAgent(), map[string]any{"email": user.Email, "role_code": t.RoleCode})
	response.OK(c, t)
}

func writeTenantHTTPError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		response.Error(c, http.StatusNotFound, 40410, "user not found")
	case errors.Is(err, ErrMemberNotFound):
		response.Error(c, http.StatusNotFound, 40411, "tenant member not found")
	case errors.Is(err, ErrCannotRemoveOwner):
		response.Error(c, http.StatusConflict, 40911, "owner member cannot be removed")
	case errors.Is(err, ErrInvitationNotFound):
		response.Error(c, http.StatusNotFound, 40412, "tenant invitation not found")
	case errors.Is(err, ErrInvitationNotPending):
		response.Error(c, http.StatusConflict, 40912, "tenant invitation is not pending")
	case errors.Is(err, ErrInvitationExpired):
		response.Error(c, http.StatusConflict, 40913, "tenant invitation is expired")
	case errors.Is(err, ErrInvitationEmailMismatch):
		response.Error(c, http.StatusForbidden, 40304, "tenant invitation email mismatch")
	default:
		response.Error(c, http.StatusInternalServerError, 50011, err.Error())
	}
}

func RegisterRoutes(private *gin.RouterGroup, h Handler) {
	private.GET("/tenants", h.List)
	private.POST("/tenants", h.Create)
	private.POST("/tenant-invitations/accept", h.AcceptInvitation)
}

func RegisterTenantScopedRoutes(rg *gin.RouterGroup, h Handler) {
	rg.GET("/tenant/members", h.ListMembers)
	rg.GET("/tenant/role-permissions", h.RolePermissions)
	rg.POST("/tenant/members", h.AddMember)
	rg.PUT("/tenant/members/:member_id/role", h.UpdateMemberRole)
	rg.DELETE("/tenant/members/:member_id", h.RemoveMember)
	rg.POST("/tenant/invitations", h.CreateInvitation)
	rg.GET("/tenant/invitations", h.ListInvitations)
	rg.POST("/tenant/invitations/:invitation_id/revoke", h.RevokeInvitation)
	rg.GET("/audit-logs", h.ListAuditLogs)
	rg.GET("/audit-logs/export", h.ExportAuditLogs)
}
