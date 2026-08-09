package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

// ExportBackup 在HTTP 处理层中完成本文件定义的局部处理。
func ExportBackup(c *gin.Context) {
	content, filename, err := service.ExportBackupZip(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/zip", content)
}

// ImportBackup 在HTTP 处理层中完成本文件定义的局部处理。
func ImportBackup(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.BackupMaxUploadSize)
	result, err := service.ImportBackupReader(c.Request.Context(), c.Request.Body)
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			respondProblem(c, http.StatusRequestEntityTooLarge, "payload_too_large", "备份文件不能超过 512MB")
			return
		}
		respondError(c, http.StatusBadRequest, err)
		return
	}
	respondImportedBackup(c, result, nil)
}

// respondImportedBackup 在HTTP 处理层中完成本文件定义的局部处理。
func respondImportedBackup(c *gin.Context, result service.ImportBackupResult, err error) {
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":       "备份导入成功",
		"files":         result.Files,
		"snapshot":      result.Snapshot,
		"library_items": result.LibraryItems,
	})
}
