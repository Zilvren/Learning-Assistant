package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	retryCount = 20
	retryDelay = 500 * time.Millisecond
)

var skipNames = map[string]bool{
	"data":        true,
	".git":        true,
	"__pycache__": true,
}

// main 在命令行工具中组装应用依赖并启动服务。
func main() {
	packagePath := flag.String("package", "", "update package zip")
	appDir := flag.String("app-dir", "", "application directory")
	appExe := flag.String("app-exe", "", "application executable")
	pid := flag.Int("pid", 0, "process id to wait for")
	flag.Parse()

	if *packagePath == "" || *appDir == "" || *appExe == "" {
		fmt.Fprintln(os.Stderr, "--package, --app-dir and --app-exe are required")
		os.Exit(2)
	}

	if err := run(*packagePath, *appDir, *appExe, *pid); err != nil {
		os.Exit(1)
	}
}

// run 在命令行工具中执行流程或启动外部操作。
func run(packagePath, appDir, appExe string, pid int) error {
	appDir, err := filepath.Abs(appDir)
	if err != nil {
		return err
	}
	updatesDir := ensureDir(filepath.Join(appDir, "data", "updates"))
	logPath := filepath.Join(updatesDir, "update.log")
	stamp := time.Now().Format("20060102-150405")
	extractDir := filepath.Join(updatesDir, "extract-"+stamp)
	rollbackDir := filepath.Join(updatesDir, "rollback-"+stamp)

	if err := runUpdate(packagePath, appDir, appExe, pid, extractDir, rollbackDir, logPath); err != nil {
		writeLog(logPath, "Update failed: "+err.Error())
		if restoreErr := restoreRollback(rollbackDir, appDir, logPath); restoreErr != nil {
			writeLog(logPath, "Rollback failed: "+restoreErr.Error())
		}
		if launchErr := launchApp(appDir, appExe, logPath); launchErr != nil {
			writeLog(logPath, "Relaunch failed: "+launchErr.Error())
		}
		return err
	}
	return nil
}

// runUpdate 在命令行工具中执行流程或启动外部操作。
func runUpdate(packagePath, appDir, appExe string, pid int, extractDir, rollbackDir, logPath string) error {
	writeLog(logPath, "Updater started")
	writeLog(logPath, fmt.Sprintf("Waiting for process %d to exit", pid))
	waitForPID(pid, 60*time.Second)

	ensureDir(extractDir)
	ensureDir(rollbackDir)
	if err := extractZipSafe(packagePath, extractDir); err != nil {
		return err
	}

	payloadRoot := pickPayloadRoot(extractDir, appExe)
	payloadVersion := readPayloadVersion(payloadRoot)
	if payloadVersion != "" {
		writeLog(logPath, "Payload version "+payloadVersion)
	}

	currentExe, _ := os.Executable()
	if err := replaceFromPayload(payloadRoot, appDir, rollbackDir, currentExe, logPath); err != nil {
		return err
	}
	if err := ensureVersionFile(payloadRoot, appDir, rollbackDir, logPath); err != nil {
		return err
	}
	if payloadVersion != "" {
		installedVersion := readPayloadVersion(appDir)
		writeLog(logPath, "Installed version "+installedVersion)
		if installedVersion != payloadVersion {
			return fmt.Errorf("version.json was not updated: expected %s, got %s", payloadVersion, installedVersion)
		}
	}

	writeLog(logPath, "Update installed successfully")
	return launchApp(appDir, appExe, logPath)
}

// pickPayloadRoot 在命令行工具中完成本文件定义的局部处理。
func pickPayloadRoot(extractDir, appExe string) string {
	if fileExists(filepath.Join(extractDir, appExe)) {
		return extractDir
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return extractDir
	}
	children := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		child := filepath.Join(extractDir, entry.Name())
		if fileExists(filepath.Join(child, appExe)) {
			children = append(children, child)
		}
	}
	if len(children) == 1 {
		return children[0]
	}
	return extractDir
}

// replaceFromPayload 在命令行工具中创建或更新相应状态。
func replaceFromPayload(payloadRoot, appDir, rollbackDir, currentExe, logPath string) error {
	entries, err := os.ReadDir(payloadRoot)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		leftExe := strings.HasSuffix(strings.ToLower(entries[i].Name()), ".exe")
		rightExe := strings.HasSuffix(strings.ToLower(entries[j].Name()), ".exe")
		if leftExe != rightExe {
			return !leftExe
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, entry := range entries {
		name := entry.Name()
		if shouldSkipName(name) {
			writeLog(logPath, "Skipping "+name)
			continue
		}
		if strings.EqualFold(name, "version.json") {
			continue
		}
		source := filepath.Join(payloadRoot, name)
		target := filepath.Join(appDir, name)
		if samePath(target, currentExe) {
			writeLog(logPath, "Skipping running updater "+name)
			continue
		}
		if err := ensureWithin(appDir, target); err != nil {
			return err
		}
		if err := backupTarget(target, rollbackDir, appDir); err != nil {
			return err
		}
		if entry.IsDir() {
			if pathExists(target) {
				if err := removePathWithRetry(target, logPath); err != nil {
					return err
				}
			}
			if err := copyTree(source, target); err != nil {
				return err
			}
		} else {
			if err := copyFileWithRetry(source, target, logPath); err != nil {
				return err
			}
		}
		writeLog(logPath, "Replaced "+name)
	}
	return nil
}

// ensureVersionFile 在命令行工具中完成本文件定义的局部处理。
func ensureVersionFile(payloadRoot, appDir, rollbackDir, logPath string) error {
	source := filepath.Join(payloadRoot, "version.json")
	if !fileExists(source) {
		return nil
	}
	target := filepath.Join(appDir, "version.json")
	if err := backupTarget(target, rollbackDir, appDir); err != nil {
		return err
	}
	if err := copyFileWithRetry(source, target, logPath); err != nil {
		return err
	}
	writeLog(logPath, "Replaced version.json")
	return nil
}

// extractZipSafe 在命令行工具中完成本文件定义的局部处理。
func extractZipSafe(packagePath, targetDir string) error {
	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		target, err := safeZipTarget(targetDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			ensureDir(target)
			continue
		}
		if err := ensureWithin(targetDir, target); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

// safeZipTarget 在命令行工具中完成本文件定义的局部处理。
func safeZipTarget(root, name string) (string, error) {
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", fmt.Errorf("zip contains absolute path: %s", name)
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == string(filepath.Separator) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("zip contains unsafe path: %s", name)
	}
	target := filepath.Join(root, clean)
	if err := ensureWithin(root, target); err != nil {
		return "", err
	}
	return target, nil
}

// backupTarget 在命令行工具中完成本文件定义的局部处理。
func backupTarget(target, rollbackDir, appDir string) error {
	if !pathExists(target) {
		return nil
	}
	rel, err := filepath.Rel(appDir, target)
	if err != nil {
		return err
	}
	backup := filepath.Join(rollbackDir, rel)
	if isDir(target) {
		return copyTree(target, backup)
	}
	return copyFile(target, backup)
}

// restoreRollback 在命令行工具中完成本文件定义的局部处理。
func restoreRollback(rollbackDir, appDir, logPath string) error {
	if !isDir(rollbackDir) {
		return nil
	}
	writeLog(logPath, "Restoring rollback files")
	return copyTree(rollbackDir, appDir)
}

// copyTree 在命令行工具中完成本文件定义的局部处理。
func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		return copyFile(path, dst)
	})
}

// copyFile 在命令行工具中完成本文件定义的局部处理。
func copyFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	return dst.Close()
}

// copyFileWithRetry 在命令行工具中完成本文件定义的局部处理。
func copyFileWithRetry(source, target, logPath string) error {
	var lastErr error
	for attempt := 1; attempt <= retryCount; attempt++ {
		if err := copyFile(source, target); err != nil {
			lastErr = err
			writeLog(logPath, fmt.Sprintf("Retry %d/%d copying %s: %v", attempt, retryCount, filepath.Base(target), err))
			time.Sleep(retryDelay)
			continue
		}
		return nil
	}
	return lastErr
}

// removePathWithRetry 在命令行工具中删除、清理或撤销相应状态。
func removePathWithRetry(path, logPath string) error {
	var lastErr error
	for attempt := 1; attempt <= retryCount; attempt++ {
		if err := os.RemoveAll(path); err != nil {
			lastErr = err
			writeLog(logPath, fmt.Sprintf("Retry %d/%d removing %s: %v", attempt, retryCount, filepath.Base(path), err))
			time.Sleep(retryDelay)
			continue
		}
		return nil
	}
	return lastErr
}

// readPayloadVersion 在命令行工具中读取并整理所需数据。
func readPayloadVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "version.json"))
	if err != nil {
		return ""
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Version)
}

// launchApp 在命令行工具中完成本文件定义的局部处理。
func launchApp(appDir, appExe, logPath string) error {
	appPath := filepath.Join(appDir, appExe)
	if !fileExists(appPath) {
		return fmt.Errorf("App exe not found: %s", appPath)
	}
	writeLog(logPath, "Launching "+appPath+" --no-browser")
	cmd := exec.Command(appPath, "--no-browser")
	cmd.Dir = appDir
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// writeLog 在命令行工具中创建或更新相应状态。
func writeLog(logPath, message string) {
	ensureDir(filepath.Dir(logPath))
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
}

// ensureDir 在命令行工具中完成本文件定义的局部处理。
func ensureDir(path string) string {
	_ = os.MkdirAll(path, 0755)
	return path
}

// ensureWithin 在命令行工具中完成本文件定义的局部处理。
func ensureWithin(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return nil
	}
	return fmt.Errorf("path escapes target directory: %s", target)
}

// shouldSkipName 在命令行工具中完成本文件定义的局部处理。
func shouldSkipName(name string) bool {
	return skipNames[strings.ToLower(name)]
}

// samePath 在命令行工具中完成本文件定义的局部处理。
func samePath(left, right string) bool {
	leftAbs, err1 := filepath.Abs(left)
	rightAbs, err2 := filepath.Abs(right)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(leftAbs, rightAbs)
}

// pathExists 在命令行工具中完成本文件定义的局部处理。
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// fileExists 在命令行工具中完成本文件定义的局部处理。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// isDir 在命令行工具中校验输入或判断当前条件。
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
