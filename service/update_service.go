package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"study-tracker-go/store"
)

const versionFile = "version.json"

var dataBackupFiles = []string{"config.json", "errors.json", "knowledge.json", "subjects.json"}

type UpdateApplyError struct {
	Message string
}

func (e UpdateApplyError) Error() string {
	return e.Message
}

type VersionInfo struct {
	Version   string `json:"version"`
	Repo      string `json:"repo"`
	AssetName string `json:"asset_name"`
	AppExe    string `json:"app_exe"`

	AppDir        string `json:"-"`
	CanAutoUpdate bool   `json:"-"`
	UpdaterPath   string `json:"-"`
}

type releaseInfo struct {
	CurrentVersion string
	LatestVersion  string
	TagName        string
	HasUpdate      bool
	Repo           string
	AssetName      string
	AssetFound     bool
	AssetSize      int64
	PublishedAt    string
	HTMLURL        string
	Notes          string
	CanAutoUpdate  bool
	DownloadURL    string
}

type githubReleaseResponse struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	Assets      []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func GetVersionResponse() map[string]interface{} {
	info := loadVersionInfo()
	return map[string]interface{}{
		"version":         info.Version,
		"repo":            info.Repo,
		"asset_name":      info.AssetName,
		"app_exe":         info.AppExe,
		"can_auto_update": info.CanAutoUpdate,
	}
}

func CheckUpdate(force bool) map[string]interface{} {
	_ = force
	checkedAt := time.Now().Format("2006-01-02 15:04:05")
	release, err := fetchLatestRelease()
	if err != nil {
		response := GetVersionResponse()
		response["ok"] = false
		response["message"] = "检查更新失败：" + err.Error()
		response["checked_at"] = checkedAt
		return response
	}

	result := releaseToMap(release)
	result["ok"] = true
	result["checked_at"] = checkedAt
	delete(result, "download_url")
	return result
}

func ApplyUpdate() (map[string]interface{}, error) {
	info := loadVersionInfo()
	if !info.CanAutoUpdate {
		return nil, UpdateApplyError{Message: "当前运行环境不支持自动替换，请使用打包后的 Tracker.exe"}
	}

	release, err := fetchLatestRelease()
	if err != nil {
		return nil, UpdateApplyError{Message: "检查更新失败：" + err.Error()}
	}
	if compareVersions(release.LatestVersion, info.Version) <= 0 {
		result := releaseToMap(release)
		result["message"] = "当前已是最新版本"
		result["ok"] = true
		delete(result, "download_url")
		return result, nil
	}
	if !release.AssetFound || strings.TrimSpace(release.DownloadURL) == "" {
		return nil, UpdateApplyError{Message: fmt.Sprintf("最新 Release 中没有找到 %s", info.AssetName)}
	}

	snapshot, err := savePreUpdateSnapshot()
	if err != nil {
		return nil, err
	}
	packagePath, err := downloadUpdatePackage(release)
	if err != nil {
		return nil, UpdateApplyError{Message: "下载更新失败：" + err.Error()}
	}
	updaterRunPath, err := copyUpdaterForRun(info.UpdaterPath)
	if err != nil {
		return nil, err
	}
	if err := startUpdater(updaterRunPath, packagePath, info.AppDir, info.AppExe); err != nil {
		return nil, err
	}

	go func() {
		time.Sleep(time.Second)
		os.Exit(0)
	}()

	return map[string]interface{}{
		"message":        "更新包已下载，程序即将重启并安装更新",
		"latest_version": release.LatestVersion,
		"package":        packagePath,
		"snapshot":       filepath.Base(snapshot),
	}, nil
}

func loadVersionInfo() VersionInfo {
	appDir := appDirectory()
	info := VersionInfo{
		Version:   "0.0.0-dev",
		Repo:      "Zilvren/Learning-Assitant",
		AssetName: "Tracker.zip",
		AppExe:    "Tracker.exe",
		AppDir:    appDir,
	}

	if data, err := os.ReadFile(filepath.Join(appDir, versionFile)); err == nil {
		_ = json.Unmarshal(data, &info)
	}
	normalizeVersionInfo(&info)
	info.AppDir = appDir
	info.UpdaterPath = filepath.Join(appDir, "Updater.exe")
	info.CanAutoUpdate = fileExists(info.UpdaterPath)
	return info
}

func normalizeVersionInfo(info *VersionInfo) {
	if strings.TrimSpace(info.Version) == "" {
		info.Version = "0.0.0-dev"
	}
	if strings.TrimSpace(info.Repo) == "" {
		info.Repo = "Zilvren/Learning-Assitant"
	}
	if strings.TrimSpace(info.AssetName) == "" {
		info.AssetName = "Tracker.zip"
	}
	if strings.TrimSpace(info.AppExe) == "" {
		info.AppExe = "Tracker.exe"
	}
}

func appDirectory() string {
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func fetchLatestRelease() (releaseInfo, error) {
	info := loadVersionInfo()
	url := "https://api.github.com/repos/" + info.Repo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return releaseInfo{}, fmt.Errorf("GitHub 返回 HTTP %d", resp.StatusCode)
	}

	var payload githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return releaseInfo{}, err
	}
	return parseReleasePayload(info, payload), nil
}

func parseReleasePayload(info VersionInfo, payload githubReleaseResponse) releaseInfo {
	latest := normalizeVersion(payload.TagName)
	result := releaseInfo{
		CurrentVersion: info.Version,
		LatestVersion:  latest,
		TagName:        payload.TagName,
		HasUpdate:      latest != "" && compareVersions(latest, info.Version) > 0,
		Repo:           info.Repo,
		AssetName:      info.AssetName,
		PublishedAt:    payload.PublishedAt,
		HTMLURL:        payload.HTMLURL,
		Notes:          payload.Body,
		CanAutoUpdate:  info.CanAutoUpdate,
	}
	for _, asset := range payload.Assets {
		if asset.Name == info.AssetName {
			result.AssetFound = true
			result.AssetSize = asset.Size
			result.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}
	return result
}

func releaseToMap(release releaseInfo) map[string]interface{} {
	return map[string]interface{}{
		"current_version": release.CurrentVersion,
		"latest_version":  release.LatestVersion,
		"tag_name":        release.TagName,
		"has_update":      release.HasUpdate,
		"repo":            release.Repo,
		"asset_name":      release.AssetName,
		"asset_found":     release.AssetFound,
		"asset_size":      release.AssetSize,
		"published_at":    release.PublishedAt,
		"html_url":        release.HTMLURL,
		"notes":           release.Notes,
		"can_auto_update": release.CanAutoUpdate,
		"download_url":    release.DownloadURL,
	}
}

func savePreUpdateSnapshot() (string, error) {
	backupDir := filepath.Join(store.DataDir(), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	snapshot := filepath.Join(backupDir, "pre-update-"+time.Now().Format("20060102-150405")+".zip")
	out, err := os.Create(snapshot)
	if err != nil {
		return "", err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	for _, name := range dataBackupFiles {
		path := store.Path(name)
		if !fileExists(path) {
			continue
		}
		if err := addFileToZip(zw, path, name); err != nil {
			_ = zw.Close()
			return "", err
		}
	}
	return snapshot, zw.Close()
}

func addFileToZip(zw *zip.Writer, source string, name string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

func downloadUpdatePackage(release releaseInfo) (string, error) {
	updatesDir := filepath.Join(store.DataDir(), "updates")
	if err := os.MkdirAll(updatesDir, 0755); err != nil {
		return "", err
	}
	stamp := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("Tracker-%s-%s.zip", safeFilenamePart(release.LatestVersion), stamp)
	target := filepath.Join(updatesDir, name)
	tmp := target + ".download"

	req, err := http.NewRequest(http.MethodGet, release.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("下载地址返回 HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := validateZip(target); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	return target, nil
}

func copyUpdaterForRun(updaterPath string) (string, error) {
	updatesDir := filepath.Join(store.DataDir(), "updates")
	if err := os.MkdirAll(updatesDir, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(updatesDir, "updater-run-"+time.Now().Format("20060102-150405")+".exe")
	src, err := os.Open(updaterPath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return "", err
	}
	if err := dst.Close(); err != nil {
		return "", err
	}
	return target, nil
}

func startUpdater(updaterPath, packagePath, appDir, appExe string) error {
	cmd := exec.Command(
		updaterPath,
		"--package", packagePath,
		"--app-dir", appDir,
		"--app-exe", appExe,
		"--pid", strconv.Itoa(os.Getpid()),
	)
	cmd.Dir = appDir
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func validateZip(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("下载的更新包不是有效 zip 文件")
	}
	return reader.Close()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func safeFilenamePart(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "unknown"
	}
	re := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	return re.ReplaceAllString(text, "_")
}

func normalizeVersion(value string) string {
	return strings.TrimLeft(strings.TrimSpace(value), "vV")
}

func compareVersions(left, right string) int {
	left = normalizeVersion(left)
	right = normalizeVersion(right)
	leftKind, leftInts, leftText := versionKey(left)
	rightKind, rightInts, rightText := versionKey(right)
	if leftKind != rightKind {
		return compareInt(leftKind, rightKind)
	}
	if leftKind == 0 {
		return strings.Compare(leftText, rightText)
	}
	for i := 0; i < len(leftInts) && i < len(rightInts); i++ {
		if c := compareInt(leftInts[i], rightInts[i]); c != 0 {
			return c
		}
	}
	return compareInt(len(leftInts), len(rightInts))
}

func versionKey(value string) (int, []int, string) {
	dateRe := regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}-\d{4}$`)
	if dateRe.MatchString(value) {
		parts := []int{}
		for _, piece := range regexp.MustCompile(`[.-]`).Split(value, -1) {
			number, _ := strconv.Atoi(piece)
			if len(piece) == 4 && len(parts) == 3 {
				hour, _ := strconv.Atoi(piece[:2])
				minute, _ := strconv.Atoi(piece[2:])
				parts = append(parts, hour, minute)
				continue
			}
			parts = append(parts, number)
		}
		return 2, parts, ""
	}

	numericRe := regexp.MustCompile(`^\d+(\.\d+)*$`)
	if numericRe.MatchString(value) {
		pieces := strings.Split(value, ".")
		parts := make([]int, 0, 6)
		for _, piece := range pieces {
			number, _ := strconv.Atoi(piece)
			parts = append(parts, number)
		}
		for len(parts) < 6 {
			parts = append(parts, 0)
		}
		return 1, parts[:6], ""
	}

	return 0, nil, value
}

func compareInt(left, right int) int {
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}
