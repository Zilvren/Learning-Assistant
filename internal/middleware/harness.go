package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/service"
)

// HarnessToolAccess authenticates a short-lived bearer capability created for
// one local Harness child process. It is deliberately separate from cookie
// authentication: the child process never receives browser credentials.
func HarnessToolAccess(app *service.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := harnessBearerToken(c.GetHeader("Authorization"))
		if !ok {
			apierror.Write(c, http.StatusUnauthorized, "harness_unauthorized", "Harness 工具会话未授权")
			c.Abort()
			return
		}
		userID, err := service.HarnessToolUserID(token)
		if err != nil {
			apierror.Write(c, http.StatusUnauthorized, "harness_unauthorized", err.Error())
			c.Abort()
			return
		}
		if app != nil && app.AuthEnabled() {
			c.Request = c.Request.WithContext(service.ContextWithUserID(c.Request.Context(), userID))
			c.Set("user_id", userID)
		}
		c.Next()
	}
}

func harnessBearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}
