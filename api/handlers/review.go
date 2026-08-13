package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"study-tracker-go/internal/service"
)

func GetReviewInbox(c *gin.Context) {
	items, err := service.ReviewInbox(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}
