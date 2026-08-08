package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"study-tracker-go/internal/service"

	"github.com/gin-gonic/gin"
)

// OCRImage 处理 POST /api/ocr
// 注意：前端发的是原始图片 Blob，不是 multipart/form-data
func OCRImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.OCRMaxUploadSize)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "OCR 文件不能超过 50MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"detail": "No file uploaded"})
		return
	}

	fileName, err := url.QueryUnescape(c.GetHeader("X-OCR-Filename"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "OCR 文件名无效"})
		return
	}
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if fileName == "" || fileName == "." {
		fileName = ocrFallbackFilename(c.ContentType())
	}
	markdown, err := service.OCRFileBytes(c.Request.Context(), body, fileName, c.ContentType())
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"markdown": markdown})
}

func ocrFallbackFilename(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "application/pdf":
		return "ocr-upload.pdf"
	case "image/jpeg":
		return "ocr-upload.jpg"
	case "image/webp":
		return "ocr-upload.webp"
	case "image/gif":
		return "ocr-upload.gif"
	default:
		return "ocr-upload.png"
	}
}
