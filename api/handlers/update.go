package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func GetVersion(c *gin.Context) {
	c.JSON(200, service.GetVersionResponse())
}

func CheckUpdate(c *gin.Context) {
	force := c.Query("force") == "true"
	c.JSON(200, service.CheckUpdate(force))
}

func ApplyUpdate(c *gin.Context) {
	result, err := service.ApplyUpdate(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
