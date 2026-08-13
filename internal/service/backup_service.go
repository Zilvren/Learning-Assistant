package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	models "study-tracker-go/internal/model"
	store "study-tracker-go/internal/repository"
)

const BackupMaxFileSize = 10 * 1024 * 1024
const BackupMaxUploadSize = 512 * 1024 * 1024
const BackupMaxUncompressedSize = 512 * 1024 * 1024
const BackupMaxArchiveEntries = 1000
const BackupMaxCompressionRatio = 100

var backupFileNames = []string{"config.json", "errors.json", "knowledge.json", "subjects.json", "library.json", "activity.json", "relations.json"}
var requiredBackupFileNames = []string{"config.json", "errors.json", "knowledge.json", "subjects.json", "library.json"}
var backupSnapshotMu sync.Mutex

type ImportBackupResult struct {
	Files        []string `json:"files"`
	Snapshot     string   `json:"snapshot"`
	LibraryItems int      `json:"library_items"`
}

// ExportBackupZip 在业务层中完成本文件定义的局部处理。
func ExportBackupZip(ctx context.Context) ([]byte, string, error) {
	var content bytes.Buffer
	if err := WriteBackupZip(ctx, &content); err != nil {
		return nil, "", err
	}
	return content.Bytes(), BackupFilename(), nil
}

// BackupFilename returns a portable, timestamped archive name for browser
// downloads. Streaming callers can set this before the first ZIP byte is sent.
func BackupFilename() string {
	return fmt.Sprintf("study-tracker-backup-%s.zip", time.Now().Format("20060102-150405"))
}

// WriteBackupZip streams the portable backup to writer. Unlike ExportBackupZip
// it does not retain a second full ZIP buffer in process memory.
func WriteBackupZip(ctx context.Context, writer io.Writer) error {
	data, err := loadBackupData(ctx)
	if err != nil {
		return err
	}
	// Credentials are deliberately not portable. Restoring a backup should
	// never leak or overwrite the OCR token; users can reconnect it in settings.
	redactBackupCredentials(&data)
	return writeBackupZip(writer, data)
}

// ImportBackupZip 在业务层中完成本文件定义的局部处理。
func ImportBackupZip(ctx context.Context, body []byte) (ImportBackupResult, error) {
	data, files, err := decodeBackupZip(body)
	if err != nil {
		return ImportBackupResult{}, err
	}
	return restoreBackupData(ctx, data, files)
}

// ImportBackupReader keeps uploaded archives out of process memory. zip.Reader
// needs random access, so the bounded stream is staged in an OS temp file and
// removed immediately after validation and import.
// ImportBackupReader 从备份流导入用户数据，并返回导入结果摘要。
func ImportBackupReader(ctx context.Context, input io.Reader) (ImportBackupResult, error) {
	temp, err := os.CreateTemp("", "study-tracker-backup-*.zip")
	if err != nil {
		return ImportBackupResult{}, err
	}
	name := temp.Name()
	defer os.Remove(name)
	defer temp.Close()

	written, err := io.Copy(temp, io.LimitReader(input, BackupMaxUploadSize+1))
	if err != nil {
		return ImportBackupResult{}, err
	}
	if written == 0 {
		return ImportBackupResult{}, fmt.Errorf("备份文件不能为空")
	}
	if written > BackupMaxUploadSize {
		return ImportBackupResult{}, fmt.Errorf("备份文件不能超过 512MB")
	}
	reader, err := zip.NewReader(temp, written)
	if err != nil {
		return ImportBackupResult{}, fmt.Errorf("请上传有效的 zip 备份文件")
	}
	data, files, err := decodeBackupZipReader(reader)
	if err != nil {
		return ImportBackupResult{}, err
	}
	return restoreBackupData(ctx, data, files)
}

// restoreBackupData 在业务层中完成本文件定义的局部处理。
func restoreBackupData(ctx context.Context, data store.BackupData, files []string) (ImportBackupResult, error) {
	snapshot, err := SaveCurrentBackupSnapshot(ctx, "pre-import")
	if err != nil {
		return ImportBackupResult{}, err
	}
	repos, err := repositories(ctx)
	if err != nil {
		return ImportBackupResult{}, err
	}
	if data.Config != nil {
		current, configErr := loadConfig(ctx)
		if configErr != nil {
			return ImportBackupResult{}, configErr
		}
		portable := *data.Config
		if portable.MineruToken == "" {
			portable.MineruToken = current.MineruToken
		}
		if portable.DeepSeekToken == "" {
			portable.DeepSeekToken = current.DeepSeekToken
		}
		data.Config = &portable
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
	sort.Strings(files)
	libraryItems := 0
	if data.Library != nil {
		libraryItems = len(data.Library.Items)
	}
	return ImportBackupResult{
		Files:        files,
		Snapshot:     filepath.Base(snapshot),
		LibraryItems: libraryItems,
	}, nil
}

// SaveCurrentBackupSnapshot 在业务层中创建或更新相应状态。
func SaveCurrentBackupSnapshot(ctx context.Context, prefix string) (string, error) {
	backupDir := filepath.Join(store.DataDir(), "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", err
	}
	snapshot := filepath.Join(backupDir, fmt.Sprintf("%s-%s.zip", prefix, time.Now().Format("20060102-150405")))
	if err := writeBackupArchive(ctx, snapshot); err != nil {
		return "", err
	}
	return snapshot, nil
}

// SaveAutomaticBackupSnapshot makes at most one scheduled snapshot per day
// and keeps only the requested number of automatic restore points.
func SaveAutomaticBackupSnapshot(ctx context.Context, keep int) (string, error) {
	backupSnapshotMu.Lock()
	defer backupSnapshotMu.Unlock()
	if keep < 1 {
		keep = 1
	}
	backupDir := filepath.Join(store.DataDir(), "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", err
	}
	prefix := "auto-" + time.Now().Format("20060102") + "-"
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".zip") {
			return filepath.Join(backupDir, entry.Name()), nil
		}
	}
	snapshot := filepath.Join(backupDir, fmt.Sprintf("%s%s.zip", prefix, time.Now().Format("150405")))
	if err := writeBackupArchive(ctx, snapshot); err != nil {
		return "", err
	}
	if err := pruneAutomaticBackupSnapshots(backupDir, keep); err != nil {
		return "", err
	}
	return snapshot, nil
}

func writeBackupArchive(ctx context.Context, target string) (err error) {
	temp, err := os.CreateTemp(filepath.Dir(target), ".backup-*.zip")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}()
	if err = temp.Chmod(0600); err != nil {
		return err
	}
	if err = WriteBackupZip(ctx, temp); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return replaceBackupFile(tempName, target)
}

func replaceBackupFile(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(source, target)
}

func pruneAutomaticBackupSnapshots(backupDir string, keep int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "auto-") && strings.HasSuffix(entry.Name(), ".zip") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files[:max(0, len(files)-keep)] {
		if err := os.Remove(filepath.Join(backupDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// loadBackupData 在业务层中读取并整理所需数据。
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

// libraryBlobHashes 在业务层中完成本文件定义的局部处理。
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

// encodeBackupZip 在业务层中构造、编码或标准化数据。
func encodeBackupZip(data store.BackupData) ([]byte, error) {
	var buffer bytes.Buffer
	if err := writeBackupZip(&buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeBackupZip(writer io.Writer, data store.BackupData) error {
	zw := zip.NewWriter(writer)
	if data.Config != nil {
		if err := writeBackupJSON(zw, "config.json", data.Config); err != nil {
			zw.Close()
			return err
		}
	}
	if data.Errors != nil {
		if err := writeBackupJSON(zw, "errors.json", data.Errors); err != nil {
			zw.Close()
			return err
		}
	}
	if data.Knowledge != nil {
		if err := writeBackupJSON(zw, "knowledge.json", data.Knowledge); err != nil {
			zw.Close()
			return err
		}
	}
	if data.Subjects != nil {
		if err := writeBackupJSON(zw, "subjects.json", data.Subjects); err != nil {
			zw.Close()
			return err
		}
	}
	if data.Library != nil {
		if err := writeBackupJSON(zw, "library.json", data.Library); err != nil {
			zw.Close()
			return err
		}
	}
	if data.Activity != nil {
		if err := writeBackupJSON(zw, "activity.json", data.Activity); err != nil {
			zw.Close()
			return err
		}
	}
	if data.Relations != nil {
		if err := writeBackupJSON(zw, "relations.json", data.Relations); err != nil {
			zw.Close()
			return err
		}
	}
	for hash, body := range data.Blobs {
		w, err := zw.Create("blobs/" + hash)
		if err != nil {
			zw.Close()
			return err
		}
		if _, err = w.Write(body); err != nil {
			zw.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return nil
}

// writeBackupJSON 在业务层中创建或更新相应状态。
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

// decodeBackupZip 在业务层中解析外部输入为内部数据。
func decodeBackupZip(body []byte) (store.BackupData, []string, error) {
	if len(body) == 0 {
		return store.BackupData{}, nil, fmt.Errorf("备份文件不能为空")
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return store.BackupData{}, nil, fmt.Errorf("请上传有效的 zip 备份文件")
	}

	return decodeBackupZipReader(reader)
}

// decodeBackupZipReader 在业务层中解析外部输入为内部数据。
func decodeBackupZipReader(reader *zip.Reader) (store.BackupData, []string, error) {
	if len(reader.File) > BackupMaxArchiveEntries {
		return store.BackupData{}, nil, fmt.Errorf("备份包中的文件数量超过限制")
	}
	data := store.BackupData{}
	files := []string{}
	seen := map[string]struct{}{}
	var totalSize uint64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > BackupMaxUncompressedSize || totalSize > BackupMaxUncompressedSize-file.UncompressedSize64 {
			return store.BackupData{}, nil, fmt.Errorf("备份解压后的总大小超过限制")
		}
		if file.UncompressedSize64 > 0 && (file.CompressedSize64 == 0 || file.UncompressedSize64/file.CompressedSize64 > BackupMaxCompressionRatio) {
			return store.BackupData{}, nil, fmt.Errorf("备份压缩比例异常")
		}
		totalSize += file.UncompressedSize64
		name := backupInputName(file.Name)
		if strings.HasPrefix(filepath.ToSlash(file.Name), "blobs/") {
			if !isBackupBlobName(name) || file.UncompressedSize64 > libraryMaxBackupBlobSize {
				return store.BackupData{}, nil, fmt.Errorf("备份包含无效 Blob")
			}
			if _, exists := seen["blobs/"+name]; exists {
				return store.BackupData{}, nil, fmt.Errorf("备份包含重复文件：blobs/%s", name)
			}
			raw, readErr := readBackupZipFileLimit(file, libraryMaxBackupBlobSize)
			if readErr != nil {
				return store.BackupData{}, nil, readErr
			}
			if data.Blobs == nil {
				data.Blobs = map[string][]byte{}
			}
			data.Blobs[name] = raw
			seen["blobs/"+name] = struct{}{}
			files = append(files, "blobs/"+name)
			continue
		}
		if !isBackupFile(name) {
			if filepath.Ext(name) == ".json" {
				return store.BackupData{}, nil, fmt.Errorf("备份包包含不支持的数据文件：%s", file.Name)
			}
			continue
		}
		if _, exists := seen[name]; exists {
			return store.BackupData{}, nil, fmt.Errorf("备份包含重复文件：%s", name)
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
		seen[name] = struct{}{}
		files = append(files, name)
	}
	if len(files) == 0 {
		return store.BackupData{}, nil, fmt.Errorf("备份包中没有可恢复的数据文件")
	}
	for _, required := range requiredBackupFileNames {
		if _, exists := seen[required]; !exists {
			return store.BackupData{}, nil, fmt.Errorf("备份不完整，缺少 %s", required)
		}
	}
	if err := validateBackupBlobs(data); err != nil {
		return store.BackupData{}, nil, err
	}
	return data, files, nil
}

// redactBackupCredentials 在业务层中完成本文件定义的局部处理。
func redactBackupCredentials(data *store.BackupData) {
	if data.Config == nil {
		return
	}
	config := *data.Config
	config.MineruToken = ""
	config.DeepSeekToken = ""
	data.Config = &config
}

// readBackupZipFile 在业务层中读取并整理所需数据。
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

// decodeBackupJSON 在业务层中解析外部输入为内部数据。
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
	case "activity.json":
		var activity store.ActivityBackup
		if err := json.Unmarshal(raw, &activity); err != nil {
			return fmt.Errorf("activity.json 不是有效 JSON 文件")
		}
		if activity.Events == nil {
			activity.Events = []models.ActivityEvent{}
		}
		data.Activity = &activity
	case "relations.json":
		var relations store.RelationBackup
		if err := json.Unmarshal(raw, &relations); err != nil {
			return fmt.Errorf("relations.json 不是有效 JSON 文件")
		}
		if relations.Relations == nil {
			relations.Relations = []models.LearningRelation{}
		}
		data.Relations = &relations
	}
	return nil
}

const libraryMaxBackupBlobSize = 200 * 1024 * 1024

// readBackupZipFileLimit 在业务层中读取并整理所需数据。
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

// isBackupFile 在业务层中校验输入或判断当前条件。
func isBackupFile(name string) bool {
	for _, allowed := range backupFileNames {
		if name == allowed {
			return true
		}
	}
	return false
}

// backupInputName 在业务层中完成本文件定义的局部处理。
func backupInputName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if slash := strings.LastIndex(value, "/"); slash >= 0 {
		value = value[slash+1:]
	}
	return strings.ToLower(value)
}

// isBackupBlobName 在业务层中校验输入或判断当前条件。
func isBackupBlobName(name string) bool {
	if len(name) != 64 {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
}

// validateBackupBlobs 在业务层中校验输入或判断当前条件。
func validateBackupBlobs(data store.BackupData) error {
	if data.Library == nil {
		return nil
	}
	for hash := range libraryBlobHashes(*data.Library) {
		if _, exists := data.Blobs[hash]; !exists {
			return fmt.Errorf("资料库附件缺失：blobs/%s。请选择完整 ZIP 备份", hash)
		}
	}
	return nil
}
