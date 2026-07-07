package handlers

import (
	"errors"
	"io"
	"net/http"

	"study-tracker-go/internal/service"

	"github.com/gin-gonic/gin"
)

const ocrMaxUploadSize = 200 * 1024 * 1024

// OCRImage 处理 POST /api/ocr
// 注意：前端发的是原始图片 Blob，不是 multipart/form-data
func OCRImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ocrMaxUploadSize)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "图片文件不能超过 200MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"detail": "No file uploaded"})
		return
	}

	markdown, err := service.OCRImageBytes(c.Request.Context(), body, "ocr_upload.png")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"markdown": markdown})
}
