package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"study-tracker-go/internal/service"
)

func SearchLearning(c *gin.Context) {
	items, err := service.SearchLearning(c.Request.Context(), c.Query("q"))
	if err != nil { respondError(c, http.StatusBadRequest, err); return }
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}
