package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"study-tracker-go/internal/service"

	"github.com/gin-gonic/gin"
)

// OCRImage 处理 POST /api/ocr
// 注意：前端发的是原始图片 Blob，不是 multipart/form-data
// OCRImage 接收图片并调用 OCR 服务，将识别结果返回给当前用户。
func OCRImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.OCRMaxUploadSize)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			respondProblem(c, http.StatusRequestEntityTooLarge, "payload_too_large", "OCR 文件不能超过 50MB")
			return
		}
		respondProblem(c, http.StatusBadRequest, "missing_file", "未上传 OCR 文件")
		return
	}

	fileName, err := url.QueryUnescape(c.GetHeader("X-OCR-Filename"))
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_filename", "OCR 文件名无效")
		return
	}
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if fileName == "" || fileName == "." {
		fileName = ocrFallbackFilename(c.ContentType())
	}
	task, err := service.StartOCRTask(c.Request.Context(), body, fileName, c.ContentType())
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"task": task})
}

// ListOCRTasks 在 HTTP 处理层中完成当前请求的处理与响应。
func ListOCRTasks(c *gin.Context) {
	tasks, err := service.ListOCRTasks(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// GetOCRTask 在 HTTP 处理层中完成当前请求的处理与响应。
func GetOCRTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ocr_task_id", "OCR 任务 ID 无效")
		return
	}
	task, err := service.GetOCRTask(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

// RetryOCRTask 在 HTTP 处理层中完成当前请求的处理与响应。
func RetryOCRTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ocr_task_id", "OCR 任务 ID 无效")
		return
	}
	task, err := service.RetryOCRTask(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusAccepted, task)
}

// ocrFallbackFilename 在HTTP 处理层中完成本文件定义的局部处理。
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
