package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/service"
	"study-tracker-go/pkg/logger"
)

// RequestContext 在请求中间件中完成本文件定义的局部处理。
func RequestContext(app *service.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := newRequestID()
		c.Set(apierror.RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(service.ContextWithApp(c.Request.Context(), app))
		c.Next()
	}
}

// RequestAudit records state-changing requests and failures. Request bodies,
// query strings, cookies, and authorization headers are intentionally omitted.
// RequestAudit 为请求附加审计日志，记录耗时、状态和关联请求 ID。
func RequestAudit(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		if !shouldAudit(c.Request.Method, c.Request.URL.Path, c.Writer.Status()) {
			return
		}
		entry := logger.AuditEntry{
			Timestamp:  started,
			RequestID:  apierror.RequestID(c),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Status:     c.Writer.Status(),
			DurationMS: time.Since(started).Milliseconds(),
			ClientIP:   c.ClientIP(),
		}
		if userID, exists := c.Get("user_id"); exists {
			if value, ok := userID.(int64); ok {
				entry.UserID = value
			}
		}
		log.Audit(entry)
	}
}

// Recovery 在请求中间件中完成本文件定义的局部处理。
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Errorf("panic request_id=%s method=%s path=%s: %v\n%s", apierror.RequestID(c), c.Request.Method, c.Request.URL.Path, recovered, debug.Stack())
				apierror.Write(c, http.StatusInternalServerError, "internal_error", "服务暂时不可用，请稍后重试")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// shouldAudit 在请求中间件中完成本文件定义的局部处理。
func shouldAudit(method string, path string, status int) bool {
	if path == "/api/health" {
		return false
	}
	if status >= http.StatusBadRequest {
		return true
	}
	if !strings.HasPrefix(path, "/api/") {
		return false
	}
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

// newRequestID 在请求中间件中完成本文件定义的局部处理。
func newRequestID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
}
