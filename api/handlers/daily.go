package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

// GetDailyPush 在HTTP 处理层中读取并整理所需数据。
func GetDailyPush(c *gin.Context) {
	result, err := service.GetDailyPush(c.Request.Context())

	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
