package tenant

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/pkg/response"
)

const TenantIDHeader = "X-Tenant-ID"

func ContextMiddleware(repo Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUser(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
			c.Abort()
			return
		}
		tenantID := c.GetHeader(TenantIDHeader)
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}
		if tenantID == "" {
			response.Error(c, http.StatusBadRequest, 40010, "X-Tenant-ID is required")
			c.Abort()
			return
		}
		t, err := repo.GetForUser(c.Request.Context(), tenantID, user.ID)
		if err != nil {
			if IsNotFound(err) {
				response.Error(c, http.StatusForbidden, 40301, "tenant access denied")
				c.Abort()
				return
			}
			response.Error(c, http.StatusInternalServerError, 50010, err.Error())
			c.Abort()
			return
		}
		c.Set(ContextTenantKey, t)
		c.Next()
	}
}

func CurrentTenant(c *gin.Context) (Tenant, bool) {
	v, ok := c.Get(ContextTenantKey)
	if !ok {
		return Tenant{}, false
	}
	t, ok := v.(Tenant)
	return t, ok
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		t, ok := CurrentTenant(c)
		if !ok {
			response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
			c.Abort()
			return
		}
		if _, ok := allowed[t.RoleCode]; !ok {
			response.Error(c, http.StatusForbidden, 40303, "tenant role is not allowed")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireTenantWriter() gin.HandlerFunc {
	return RequireRoles("owner", "admin", "member")
}

func RequireTenantAdmin() gin.HandlerFunc {
	return RequireRoles("owner", "admin")
}

type AuditLogger interface {
	InsertAuditLog(ctx context.Context, tenantID, actorUserID, action, resourceType, resourceID, ip, userAgent string, metadata map[string]any) error
}

func AuditMiddleware(logger AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if logger == nil || !shouldAuditMethod(c.Request.Method) {
			return
		}
		t, ok := CurrentTenant(c)
		if !ok {
			return
		}
		u, _ := auth.CurrentUser(c)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = logger.InsertAuditLog(ctx, t.ID, u.ID, "http."+strings.ToLower(c.Request.Method), "http_request", "", c.ClientIP(), c.Request.UserAgent(), map[string]any{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"route":      c.FullPath(),
			"status":     c.Writer.Status(),
			"request_id": c.GetString("request_id"),
			"role_code":  t.RoleCode,
		})
	}
}

func shouldAuditMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
