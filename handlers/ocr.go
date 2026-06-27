package handlers

import (
	"io"
	"net/http"

	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

// OCRImage 处理 POST /api/ocr
// 注意：前端发的是原始图片 Blob，不是 multipart/form-data
func OCRImage(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "No file uploaded"})
		return
	}

	markdown, err := service.OCRImageBytes(body, "ocr_upload.png")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"markdown": markdown})
}
