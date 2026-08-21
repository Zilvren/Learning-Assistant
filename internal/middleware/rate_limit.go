package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
)

type rateWindow struct {
	started time.Time
	count   int
}

// RateLimit 为公开认证和邮件接口提供轻量的进程内保护。它按客户端 IP 采取默认拒绝策略，并定期清理空闲条目以避免映射无限增长。
// RateLimit 创建按客户端 IP 限制请求频率的 Gin 中间件。
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	var mu sync.Mutex
	entries := map[string]rateWindow{}
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		mu.Lock()
		entry := entries[key]
		if entry.started.IsZero() || now.Sub(entry.started) >= window {
			entry = rateWindow{started: now}
		}
		entry.count++
		entries[key] = entry
		for staleKey, stale := range entries {
			if now.Sub(stale.started) > 2*window {
				delete(entries, staleKey)
			}
		}
		exceeded := entry.count > limit
		mu.Unlock()
		if exceeded {
			c.Header("Retry-After", "60")
			apierror.Write(c, http.StatusTooManyRequests, "rate_limited", "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
