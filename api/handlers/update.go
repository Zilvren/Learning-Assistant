package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

// GetVersion 在HTTP 处理层中读取并整理所需数据。
func GetVersion(c *gin.Context) {
	if !service.UpdateEnabled(c.Request.Context()) {
		respondError(c, http.StatusNotFound, errors.New("生产环境不提供客户端版本更新"))
		return
	}
	c.JSON(200, service.GetVersionResponse())
}

// CheckUpdate 在HTTP 处理层中完成本文件定义的局部处理。
func CheckUpdate(c *gin.Context) {
	if !service.UpdateEnabled(c.Request.Context()) {
		respondError(c, http.StatusNotFound, errors.New("生产环境不提供客户端版本更新"))
		return
	}
	force := c.Query("force") == "true"
	c.JSON(200, service.CheckUpdate(force))
}

// ApplyUpdate 在HTTP 处理层中执行流程或启动外部操作。
func ApplyUpdate(c *gin.Context) {
	if !service.UpdateEnabled(c.Request.Context()) {
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
