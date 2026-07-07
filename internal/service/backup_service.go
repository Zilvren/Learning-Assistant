package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	models "study-tracker-go/internal/model"
	store "study-tracker-go/internal/repository"
)

const BackupMaxFileSize = 10 * 1024 * 1024
const BackupMaxUploadSize = 50 * 1024 * 1024

var backupFileNames = []string{"config.json", "errors.json", "knowledge.json", "subjects.json"}

type ImportBackupResult struct {
	Files    []string `json:"files"`
	Snapshot string   `json:"snapshot"`
}

func ExportBackupZip(ctx context.Context) ([]byte, string, error) {
	data, err := loadBackupData(ctx)
	if err != nil {
		return nil, "", err
	}
	content, err := encodeBackupZip(data)
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("study-tracker-backup-%s.zip", time.Now().Format("20060102-150405"))
	return content, filename, nil
}

func ImportBackupZip(ctx context.Context, body []byte) (ImportBackupResult, error) {
	data, files, err := decodeBackupZip(body)
	if err != nil {
		return ImportBackupResult{}, err
	}
	snapshot, err := SaveCurrentBackupSnapshot(ctx, "pre-import")
	if err != nil {
		return ImportBackupResult{}, err
	}
	repos, err := repositories(ctx)
	if err != nil {
		return ImportBackupResult{}, err
	}
	if err := repos.Backup.Import(ctx, data); err != nil {
		return ImportBackupResult{}, err
	}
	sort.Strings(files)
	return ImportBackupResult{
		Files:    files,
		Snapshot: filepath.Base(snapshot),
	}, nil
}

func SaveCurrentBackupSnapshot(ctx context.Context, prefix string) (string, error) {
	content, _, err := ExportBackupZip(ctx)
	if err != nil {
		return "", err
	}
	backupDir := filepath.Join(store.DataDir(), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	snapshot := filepath.Join(backupDir, fmt.Sprintf("%s-%s.zip", prefix, time.Now().Format("20060102-150405")))
	return snapshot, os.WriteFile(snapshot, content, 0644)
}

func loadBackupData(ctx context.Context) (store.BackupData, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return store.BackupData{}, err
	}
	return repos.Backup.Export(ctx)
}

func encodeBackupZip(data store.BackupData) ([]byte, error) {
	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	if data.Config != nil {
		if err := writeBackupJSON(zw, "config.json", data.Config); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if data.Errors != nil {
		if err := writeBackupJSON(zw, "errors.json", data.Errors); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if data.Knowledge != nil {
		if err := writeBackupJSON(zw, "knowledge.json", data.Knowledge); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if data.Subjects != nil {
		if err := writeBackupJSON(zw, "subjects.json", data.Subjects); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeBackupJSON(zw *zip.Writer, name string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func decodeBackupZip(body []byte) (store.BackupData, []string, error) {
	if len(body) == 0 {
		return store.BackupData{}, nil, fmt.Errorf("备份文件不能为空")
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return store.BackupData{}, nil, fmt.Errorf("请上传有效的 zip 备份文件")
	}

	data := store.BackupData{}
	files := []string{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(file.Name)
		if !isBackupFile(name) {
			if filepath.Ext(name) == ".json" {
				return store.BackupData{}, nil, fmt.Errorf("备份包包含不支持的数据文件：%s", file.Name)
			}
			continue
		}
		if file.UncompressedSize64 > BackupMaxFileSize {
			return store.BackupData{}, nil, fmt.Errorf("%s 文件过大", name)
		}
		raw, err := readBackupZipFile(file)
		if err != nil {
			return store.BackupData{}, nil, err
		}
		if err := decodeBackupJSON(name, raw, &data); err != nil {
			return store.BackupData{}, nil, err
		}
		files = append(files, name)
	}
	if len(files) == 0 {
		return store.BackupData{}, nil, fmt.Errorf("备份包中没有可恢复的数据文件")
	}
	return data, files, nil
}

func readBackupZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, BackupMaxFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > BackupMaxFileSize {
		return nil, fmt.Errorf("%s 文件过大", filepath.Base(file.Name))
	}
	return data, nil
}

func decodeBackupJSON(name string, raw []byte, data *store.BackupData) error {
	switch name {
	case "errors.json":
		var errors []models.ErrorProblem
		if err := json.Unmarshal(raw, &errors); err != nil {
			return fmt.Errorf("errors.json 不是有效 JSON 文件")
		}
		data.Errors = &errors
	case "subjects.json":
		var subjects []string
		if err := json.Unmarshal(raw, &subjects); err != nil {
			return fmt.Errorf("subjects.json 不是有效 JSON 文件")
		}
		data.Subjects = &subjects
	case "config.json":
		var config models.Config
		if err := json.Unmarshal(raw, &config); err != nil {
			return fmt.Errorf("config.json 不是有效 JSON 文件")
		}
		data.Config = &config
	case "knowledge.json":
		var knowledge map[string][]string
		if err := json.Unmarshal(raw, &knowledge); err != nil {
			return fmt.Errorf("knowledge.json 不是有效 JSON 文件")
		}
		data.Knowledge = &knowledge
	}
	return nil
}

func isBackupFile(name string) bool {
	for _, allowed := range backupFileNames {
		if name == allowed {
			return true
		}
	}
	return false
}
