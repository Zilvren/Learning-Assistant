package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/service"
)

// HarnessToolAccess 验证为本地 Harness 子进程创建的短时 Bearer 能力令牌。它刻意独立于 Cookie 认证，因为子进程绝不会接收浏览器凭据。
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

// harnessBearerToken 在请求中间件中完成当前局部处理。
func harnessBearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}
