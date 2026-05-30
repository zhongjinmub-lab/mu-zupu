package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/pkg/response"
)

func AuthMiddleware(jwtSvc JWTService, repo Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			response.Error(c, http.StatusUnauthorized, 40101, "missing bearer token")
			c.Abort()
			return
		}
		claims, err := jwtSvc.Parse(strings.TrimSpace(strings.TrimPrefix(authHeader, prefix)))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 40102, "invalid bearer token")
			c.Abort()
			return
		}
		u, err := repo.FindByID(c.Request.Context(), claims.UserID)
		if err != nil || u.Status != "active" {
			response.Error(c, http.StatusUnauthorized, 40103, "user unavailable")
			c.Abort()
			return
		}
		c.Set(ContextUserKey, u)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (User, bool) {
	v, ok := c.Get(ContextUserKey)
	if !ok {
		return User{}, false
	}
	u, ok := v.(User)
	return u, ok
}
