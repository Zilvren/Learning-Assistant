package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/service"
)

func GetVersion(c *gin.Context) {
	c.JSON(200, service.GetVersionResponse())
}

func CheckUpdate(c *gin.Context) {
	force := c.Query("force") == "true"
	c.JSON(200, service.CheckUpdate(force))
}

func ApplyUpdate(c *gin.Context) {
	result, err := service.ApplyUpdate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
