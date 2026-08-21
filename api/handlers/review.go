package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"study-tracker-go/internal/service"
)

// GetReviewInbox 在 HTTP 处理层中完成当前请求的处理与响应。
func GetReviewInbox(c *gin.Context) {
	items, err := service.ReviewInbox(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}
