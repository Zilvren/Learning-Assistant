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
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	store "study-tracker-go/internal/repository"
)

const BackupMaxFileSize = 10 * 1024 * 1024
const BackupMaxUploadSize = 512 * 1024 * 1024

var backupFileNames = []string{"config.json", "errors.json", "knowledge.json", "subjects.json", "library.json"}

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
	// Blob files live outside PostgreSQL. Store and verify them before creating
	// database rows that reference their hashes, so a failed blob never leaves
	// an imported note or file unreadable.
	for expected, body := range data.Blobs {
		actual, _, blobErr := store.StoreBlob(bytes.NewReader(body))
		if blobErr != nil {
			return ImportBackupResult{}, blobErr
		}
		if actual != expected {
			return ImportBackupResult{}, fmt.Errorf("Blob 校验失败：%s", expected)
		}
	}
	if err := repos.Backup.Import(ctx, data); err != nil {
		return ImportBackupResult{}, err
	}
	// JSON mode keeps its library index in library.json. PostgreSQL mode
	// restores it through BackupRepository.Import into library_items instead.
	if data.Library != nil && currentConfig().StorageDriver != "postgres" {
		if err := store.SaveJSON("library.json", data.Library); err != nil {
			return ImportBackupResult{}, err
		}
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
	data, err := repos.Backup.Export(ctx)
	if err != nil {
		return data, err
	}
	if data.Library != nil {
		data.Blobs = map[string][]byte{}
		for hash := range libraryBlobHashes(*data.Library) {
			body, readErr := store.ReadBlob(hash)
			if readErr != nil {
				return data, fmt.Errorf("读取资料附件失败：%s", hash)
			}
			data.Blobs[hash] = body
		}
	}
	return data, nil
}

func libraryBlobHashes(library store.LibraryBackup) map[string]struct{} {
	hashes := map[string]struct{}{}
	for _, item := range library.Items {
		if len(item.BlobHash) == 64 {
			hashes[item.BlobHash] = struct{}{}
		}
	}
	for _, version := range library.Versions {
		if len(version.BlobHash) == 64 {
			hashes[version.BlobHash] = struct{}{}
		}
	}
	return hashes
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
	if data.Library != nil {
		if err := writeBackupJSON(zw, "library.json", data.Library); err != nil {
			zw.Close()
			return nil, err
		}
	}
	for hash, body := range data.Blobs {
		w, err := zw.Create("blobs/" + hash)
		if err != nil {
			zw.Close()
			return nil, err
		}
		if _, err = w.Write(body); err != nil {
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
		if strings.HasPrefix(filepath.ToSlash(file.Name), "blobs/") {
			if len(name) != 64 || file.UncompressedSize64 > libraryMaxBackupBlobSize {
				return store.BackupData{}, nil, fmt.Errorf("备份包含无效 Blob")
			}
			raw, readErr := readBackupZipFileLimit(file, libraryMaxBackupBlobSize)
			if readErr != nil {
				return store.BackupData{}, nil, readErr
			}
			if data.Blobs == nil {
				data.Blobs = map[string][]byte{}
			}
			data.Blobs[name] = raw
			files = append(files, "blobs/"+name)
			continue
		}
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
	case "library.json":
		var library store.LibraryBackup
		if err := json.Unmarshal(raw, &library); err != nil {
			return fmt.Errorf("library.json 不是有效 JSON 文件")
		}
		if library.Items == nil {
			library.Items = []models.LibraryItem{}
		}
		if library.Versions == nil {
			library.Versions = []models.LibraryVersion{}
		}
		data.Library = &library
	}
	return nil
}

const libraryMaxBackupBlobSize = 200 * 1024 * 1024

func readBackupZipFileLimit(file *zip.File, limit int64) ([]byte, error) {
	rc, e := file.Open()
	if e != nil {
		return nil, e
	}
	defer rc.Close()
	body, e := io.ReadAll(io.LimitReader(rc, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s 文件过大", file.Name)
	}
	return body, nil
}

func isBackupFile(name string) bool {
	for _, allowed := range backupFileNames {
		if name == allowed {
			return true
		}
	}
	return false
}
