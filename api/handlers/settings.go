package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func GetToken(c *gin.Context) {
	info, err := service.GetTokenInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info) // 直接用结构体序列化，JSON 标签跟前端期望一致
}

func SetToken(c *gin.Context) {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if err := service.SetToken(c.Request.Context(), body.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token saved"})
}

func DeleteToken(c *gin.Context) {
	if err := service.ClearToken(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token cleared"})
}

func SetUsername(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if err := service.SetUsername(c.Request.Context(), body.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Username saved"})
}
