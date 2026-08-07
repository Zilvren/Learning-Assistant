package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func GetVersion(c *gin.Context) {
	if !service.UpdateEnabled() {
		respondError(c, http.StatusNotFound, errors.New("生产环境不提供客户端版本更新"))
		return
	}
	c.JSON(200, service.GetVersionResponse())
}

func CheckUpdate(c *gin.Context) {
	if !service.UpdateEnabled() {
		respondError(c, http.StatusNotFound, errors.New("生产环境不提供客户端版本更新"))
		return
	}
	force := c.Query("force") == "true"
	c.JSON(200, service.CheckUpdate(force))
}

func ApplyUpdate(c *gin.Context) {
	if !service.UpdateEnabled() {
		respondError(c, http.StatusNotFound, errors.New("生产环境不提供客户端版本更新"))
		return
	}
	result, err := service.ApplyUpdate(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
