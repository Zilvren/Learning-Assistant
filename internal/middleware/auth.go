package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/service"
)

// AuthRequired 在请求中间件中完成本文件定义的局部处理。
func AuthRequired(app *service.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if app == nil || !app.AuthEnabled() {
			c.Next()
			return
		}
		token, err := c.Cookie(service.AccessCookieName)
		if err != nil || strings.TrimSpace(token) == "" {
			apierror.Write(c, http.StatusUnauthorized, "unauthorized", "未登录")
			c.Abort()
			return
		}
		user, err := service.ValidateAccessToken(c.Request.Context(), token)
		if err != nil {
			apierror.Write(c, http.StatusUnauthorized, "unauthorized", "未登录")
			c.Abort()
			return
		}
		ctx := service.ContextWithUserID(c.Request.Context(), user.ID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("user_id", user.ID)
		c.Next()
	}
}

// CookieOriginGuard 在请求中间件中完成本文件定义的局部处理。
func CookieOriginGuard(app *service.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if app == nil || !app.AuthEnabled() || !isUnsafeMethod(c.Request.Method) {
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = c.GetHeader("Referer")
		}
		if origin == "" || sameHost(origin, c.Request.Host) || allowedDevOrigin(origin) {
			c.Next()
			return
		}
		apierror.Write(c, http.StatusForbidden, "origin_not_allowed", "请求来源不受信任")
		c.Abort()
	}
}

// isUnsafeMethod 在请求中间件中校验输入或判断当前条件。
func isUnsafeMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

// sameHost 在请求中间件中完成本文件定义的局部处理。
func sameHost(rawURL string, host string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}

// allowedDevOrigin 在请求中间件中校验输入或判断当前条件。
func allowedDevOrigin(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" && (parsed.Host == "127.0.0.1:5173" || parsed.Host == "localhost:5173")
}
