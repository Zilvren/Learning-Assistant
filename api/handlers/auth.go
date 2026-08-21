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

// AuthStatus 在HTTP 处理层中完成本文件定义的局部处理。
func AuthStatus(c *gin.Context) {
	status, err := service.AuthStatusResponse(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

// Register 在HTTP 处理层中完成本文件定义的局部处理。
func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}
	result, err := service.Register(c.Request.Context(), req, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
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

// VerifyEmail 在HTTP 处理层中完成本文件定义的局部处理。
func VerifyEmail(c *gin.Context) {
	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}
	pair, err := service.VerifyEmail(c.Request.Context(), req.Token, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	writeAuthCookies(c, pair)
	c.JSON(http.StatusOK, models.AuthResponse{User: pair.User})
}

// ResendEmailVerification 在HTTP 处理层中完成本文件定义的局部处理。
func ResendEmailVerification(c *gin.Context) {
	var req models.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}
	if err := service.ResendEmailVerification(c.Request.Context(), req.Email); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	// 不泄露该地址是否属于尚未验证的账户。
	c.JSON(http.StatusAccepted, gin.H{"message": "如果该邮箱对应未验证账号，验证邮件已发送"})
}

// Login 在HTTP 处理层中完成本文件定义的局部处理。
func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}
	pair, err := service.Login(c.Request.Context(), req, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		respondError(c, http.StatusUnauthorized, err)
		return
	}
	writeAuthCookies(c, pair)
	c.JSON(http.StatusOK, models.AuthResponse{User: pair.User})
}

// Refresh 在HTTP 处理层中完成本文件定义的局部处理。
func Refresh(c *gin.Context) {
	token, _ := c.Cookie(service.RefreshCookieName)
	pair, err := service.RefreshLogin(c.Request.Context(), token, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		clearAuthCookies(c)
		respondError(c, http.StatusUnauthorized, err)
		return
	}
	writeAuthCookies(c, pair)
	c.JSON(http.StatusOK, models.AuthResponse{User: pair.User})
}

// Logout 在HTTP 处理层中完成本文件定义的局部处理。
func Logout(c *gin.Context) {
	token, _ := c.Cookie(service.RefreshCookieName)
	if err := service.Logout(c.Request.Context(), token); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// Me 在HTTP 处理层中完成本文件定义的局部处理。
func Me(c *gin.Context) {
	user, err := service.CurrentUser(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusUnauthorized, err)
		return
	}
	c.JSON(http.StatusOK, models.AuthResponse{User: user})
}

// writeAuthCookies 在HTTP 处理层中创建或更新相应状态。
func writeAuthCookies(c *gin.Context, pair service.TokenPair) {
	setCookie(c, service.AccessCookieName, pair.AccessToken, pair.AccessTokenExpiresAt)
	setCookie(c, service.RefreshCookieName, pair.RefreshToken, pair.RefreshExpiresAt)
}

// clearAuthCookies 在HTTP 处理层中删除、清理或撤销相应状态。
func clearAuthCookies(c *gin.Context) {
	expired := time.Now().Add(-time.Hour)
	setCookie(c, service.AccessCookieName, "", expired)
	setCookie(c, service.RefreshCookieName, "", expired)
}

// setCookie 在HTTP 处理层中完成本文件定义的局部处理。
func setCookie(c *gin.Context, name string, value string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = -1
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", service.CookieSecure(c.Request.Context()), true)
}

// clientIP 在HTTP 处理层中完成本文件定义的局部处理。
func clientIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(c.ClientIP())
}
