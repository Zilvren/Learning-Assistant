package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !service.AuthEnabled() {
			c.Next()
			return
		}
		token, err := c.Cookie(service.AccessCookieName)
		if err != nil || strings.TrimSpace(token) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "未登录"})
			c.Abort()
			return
		}
		user, err := service.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "未登录"})
			c.Abort()
			return
		}
		ctx := service.ContextWithUserID(c.Request.Context(), user.ID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("user_id", user.ID)
		c.Next()
	}
}

func CookieOriginGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !service.AuthEnabled() || !isUnsafeMethod(c.Request.Method) {
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
		c.JSON(http.StatusForbidden, gin.H{"detail": "请求来源不受信任"})
		c.Abort()
	}
}

func isUnsafeMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func sameHost(rawURL string, host string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}

func allowedDevOrigin(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" && (parsed.Host == "127.0.0.1:5173" || parsed.Host == "localhost:5173")
}
