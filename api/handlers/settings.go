package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

// GetToken 在HTTP 处理层中读取并整理所需数据。
func GetToken(c *gin.Context) {
	info, err := service.GetTokenInfo(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, info) // 直接用结构体序列化，JSON 标签跟前端期望一致
}

// SetToken 在HTTP 处理层中完成本文件定义的局部处理。
func SetToken(c *gin.Context) {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}
	if err := service.SetToken(c.Request.Context(), body.Token); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token saved"})
}

// DeleteToken 在HTTP 处理层中删除、清理或撤销相应状态。
func DeleteToken(c *gin.Context) {
	if err := service.ClearToken(c.Request.Context()); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token cleared"})
}

// SetUsername 在HTTP 处理层中完成本文件定义的局部处理。
func SetUsername(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}
	if err := service.SetUsername(c.Request.Context(), body.Name); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Username saved"})
}
