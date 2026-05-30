package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDUsesIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		if got := c.GetString("request_id"); got != "rid-1" {
			t.Fatalf("request_id context = %q", got)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "rid-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get(RequestIDHeader) != "rid-1" {
		t.Fatalf("response request id = %q", w.Header().Get(RequestIDHeader))
	}
}

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Header().Get(RequestIDHeader) == "" {
		t.Fatal("expected generated request id")
	}
}
