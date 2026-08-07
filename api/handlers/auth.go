package handlers

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/service"
)

func AuthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, service.AuthStatusResponse())
}

func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	result, err := service.Register(c.Request.Context(), req, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	if result.EmailVerificationRequired {
		c.JSON(http.StatusAccepted, models.RegistrationResponse{
			EmailVerificationRequired: true,
			Email:                     result.Email,
		})
		return
	}
	writeAuthCookies(c, result.TokenPair)
	c.JSON(http.StatusOK, models.AuthResponse{User: result.TokenPair.User})
}

func VerifyEmail(c *gin.Context) {
	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	pair, err := service.VerifyEmail(c.Request.Context(), req.Token, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	writeAuthCookies(c, pair)
	c.JSON(http.StatusOK, models.AuthResponse{User: pair.User})
}

func ResendEmailVerification(c *gin.Context) {
	var req models.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if err := service.ResendEmailVerification(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	// Do not reveal whether the address belongs to an unverified account.
	c.JSON(http.StatusAccepted, gin.H{"message": "如果该邮箱对应未验证账号，验证邮件已发送"})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	pair, err := service.Login(c.Request.Context(), req, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": err.Error()})
		return
	}
	writeAuthCookies(c, pair)
	c.JSON(http.StatusOK, models.AuthResponse{User: pair.User})
}

func Refresh(c *gin.Context) {
	token, _ := c.Cookie(service.RefreshCookieName)
	pair, err := service.RefreshLogin(c.Request.Context(), token, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		clearAuthCookies(c)
		c.JSON(http.StatusUnauthorized, gin.H{"detail": err.Error()})
		return
	}
	writeAuthCookies(c, pair)
	c.JSON(http.StatusOK, models.AuthResponse{User: pair.User})
}

func Logout(c *gin.Context) {
	token, _ := c.Cookie(service.RefreshCookieName)
	if err := service.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

func Me(c *gin.Context) {
	user, err := service.CurrentUser(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.AuthResponse{User: user})
}

func writeAuthCookies(c *gin.Context, pair service.TokenPair) {
	setCookie(c, service.AccessCookieName, pair.AccessToken, pair.AccessTokenExpiresAt)
	setCookie(c, service.RefreshCookieName, pair.RefreshToken, pair.RefreshExpiresAt)
}

func clearAuthCookies(c *gin.Context) {
	expired := time.Now().Add(-time.Hour)
	setCookie(c, service.AccessCookieName, "", expired)
	setCookie(c, service.RefreshCookieName, "", expired)
}

func setCookie(c *gin.Context, name string, value string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = -1
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", service.CookieSecure(), true)
}

func clientIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(c.ClientIP())
}
