package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireRolesAllowsConfiguredRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextTenantKey, Tenant{ID: "tenant", RoleCode: "admin"})
	})
	r.GET("/x", RequireTenantAdmin(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestRequireRolesRejectsDisallowedRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextTenantKey, Tenant{ID: "tenant", RoleCode: "viewer"})
	})
	r.POST("/x", RequireTenantWriter(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestShouldAuditMethod(t *testing.T) {
	if !shouldAuditMethod(http.MethodPost) || !shouldAuditMethod(http.MethodPut) || !shouldAuditMethod(http.MethodPatch) || !shouldAuditMethod(http.MethodDelete) {
		t.Fatal("write methods should be audited")
	}
	if shouldAuditMethod(http.MethodGet) {
		t.Fatal("GET should not be audited")
	}
}

func TestAuditMiddlewareRecordsWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := &fakeAuditLogger{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextTenantKey, Tenant{ID: "tenant", RoleCode: "owner"})
		c.Set("request_id", "rid-1")
	})
	r.Use(AuditMiddleware(logger))
	r.POST("/x", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	r.ServeHTTP(w, req)
	if logger.count != 1 {
		t.Fatalf("audit count = %d", logger.count)
	}
	if logger.tenantID != "tenant" || logger.action != "http.post" {
		t.Fatalf("audit logger = %#v", logger)
	}
}

type fakeAuditLogger struct {
	count    int
	tenantID string
	action   string
}

func (f *fakeAuditLogger) InsertAuditLog(ctx context.Context, tenantID, actorUserID, action, resourceType, resourceID, ip, userAgent string, metadata map[string]any) error {
	f.count++
	f.tenantID = tenantID
	f.action = action
	return nil
}
