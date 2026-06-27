package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"study-tracker-go/store"
)

// 允许备份/导入的文件白名单
var backupFiles = map[string]bool{
	"errors.json":    true,
	"subjects.json":  true,
	"config.json":    true,
	"knowledge.json": true,
}

const backupMaxFileSize = 10 * 1024 * 1024 // 单个文件最大 10MB

// ExportBackup 导出备份 GET /api/backup/export
func ExportBackup(c *gin.Context) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	names := sortedBackupNames()
	for _, name := range names {
		path := store.Path(name)
		if _, err := os.Stat(path); err != nil {
			continue // 文件不存在就跳过
		}
		if err := addFileToZip(zipWriter, path, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			zipWriter.Close()
			return
		}
	}

	if err := zipWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}

	filename := fmt.Sprintf("study-tracker-backup-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/zip", buffer.Bytes())
}

// ImportBackup 导入备份 POST /api/backup/import
// 注意：前端发的是原始 zip 二进制，不是 multipart/form-data
func ImportBackup(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "备份文件不能为空"})
		return
	}

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请上传有效的 zip 备份文件"})
		return
	}

	parsed := map[string]interface{}{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := filepath.Base(file.Name)
		if !backupFiles[name] {
			if filepath.Ext(name) == ".json" {
				c.JSON(http.StatusBadRequest, gin.H{"detail": "备份包包含不支持的数据文件：" + file.Name})
				return
			}
			continue
		}
		if file.UncompressedSize64 > backupMaxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"detail": name + " 文件过大"})
			return
		}

		data, err := readZipFile(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		var value interface{}
		if err := json.Unmarshal(data, &value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": name + " 不是有效 JSON 文件"})
			return
		}
		if err := validateBackupData(name, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}
		parsed[name] = value
	}

	if len(parsed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "备份包中没有可恢复的数据文件"})
		return
	}

	// 导入前先备份当前数据
	snapshot, err := saveCurrentBackupSnapshot("pre-import")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}

	imported := []string{}
	for name, value := range parsed {
		if err := store.SaveJSON(name, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}
		imported = append(imported, name)
	}
	sort.Strings(imported)

	c.JSON(http.StatusOK, gin.H{
		"message":  "备份导入成功",
		"files":    imported,
		"snapshot": filepath.Base(snapshot),
	})
}

// --- 以下是辅助函数 ---

func sortedBackupNames() []string {
	names := []string{}
	for name := range backupFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func addFileToZip(zipWriter *zip.Writer, path string, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func validateBackupData(name string, data interface{}) error {
	switch name {
	case "errors.json":
		list, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("errors.json 数据结构不正确")
		}
		for _, item := range list {
			if _, ok := item.(map[string]interface{}); !ok {
				return fmt.Errorf("errors.json 数据结构不正确")
			}
		}
	case "subjects.json":
		list, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("subjects.json 数据结构不正确")
		}
		for _, item := range list {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("subjects.json 数据结构不正确")
			}
		}
	case "config.json", "knowledge.json":
		if _, ok := data.(map[string]interface{}); !ok {
			return fmt.Errorf("%s 数据结构不正确", name)
		}
	}
	return nil
}

func saveCurrentBackupSnapshot(prefix string) (string, error) {
	backupDir := filepath.Join(store.DataDir(), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	snapshot := filepath.Join(backupDir, fmt.Sprintf("%s-%s.zip", prefix, time.Now().Format("20060102-150405")))
	file, err := os.Create(snapshot)
	if err != nil {
		return "", err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	for _, name := range sortedBackupNames() {
		path := store.Path(name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := addFileToZip(zipWriter, path, name); err != nil {
			zipWriter.Close()
			return "", err
		}
	}
	return snapshot, zipWriter.Close()
}
