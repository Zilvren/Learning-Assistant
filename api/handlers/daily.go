package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func GetDailyPush(c *gin.Context) {
	result, err := service.GetDailyPush(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
