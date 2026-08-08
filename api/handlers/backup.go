package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func ExportBackup(c *gin.Context) {
	content, filename, err := service.ExportBackupZip(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/zip", content)
}

func ImportBackup(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.BackupMaxUploadSize)
	result, err := service.ImportBackupReader(c.Request.Context(), c.Request.Body)
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "备份文件不能超过 512MB"})
			return
		}
		respondError(c, http.StatusBadRequest, err)
		return
	}
	respondImportedBackup(c, result, nil)
}

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
