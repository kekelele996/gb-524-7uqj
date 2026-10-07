package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"spectrum-interference-triangulation/backend/internal/service"
	"spectrum-interference-triangulation/backend/pkg/api"
)

func Auth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			api.Fail(c, api.NewError(401, "AUTH_REQUIRED", "请先登录后再访问该资源"))
			return
		}
		claims, err := authService.ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			api.Fail(c, err)
			return
		}
		// Re-read the account so deactivation and role changes take effect
		// immediately instead of waiting for the JWT to expire.
		user, err := authService.Me(c.Request.Context(), claims.UserID)
		if err != nil {
			api.Fail(c, err)
			return
		}
		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("role", user.Role)
		c.Next()
	}
}
