package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// TestPickPayloadRoot 在命令行工具中验证对应场景的行为与边界条件。
func TestPickPayloadRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "Tracker-2099")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "Tracker.exe"), []byte("exe"), 0644); err != nil {
		t.Fatal(err)
	}

	got := pickPayloadRoot(root, "Tracker.exe")
	if got != nested {
		t.Fatalf("pickPayloadRoot() = %q, want %q", got, nested)
	}
}

// TestSafeZipTargetRejectsTraversal 在命令行工具中验证对应场景的行为与边界条件。
func TestSafeZipTargetRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := safeZipTarget(root, "../escape.txt"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := safeZipTarget(root, "/absolute.txt"); err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

// TestExtractZipSafeRejectsUnsafeEntry 在命令行工具中验证对应场景的行为与边界条件。
func TestExtractZipSafeRejectsUnsafeEntry(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "unsafe.zip")
	if err := writeTestZip(zipPath, map[string]string{
		"../escape.txt": "bad",
	}); err != nil {
		t.Fatal(err)
	}

	if err := extractZipSafe(zipPath, filepath.Join(root, "out")); err == nil {
		t.Fatal("expected unsafe zip to be rejected")
	}
}

// TestExtractZipSafeWritesSafeEntry 在命令行工具中验证对应场景的行为与边界条件。
func TestExtractZipSafeWritesSafeEntry(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "safe.zip")
	if err := writeTestZip(zipPath, map[string]string{
		"Tracker/version.json": `{"version":"2099.01.01-0001"}`,
	}); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "out")
	if err := extractZipSafe(zipPath, out); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(out, "Tracker", "version.json")) {
		t.Fatal("expected safe zip entry to be extracted")
	}
}

// writeTestZip 在命令行工具中创建或更新相应状态。
func writeTestZip(path string, files map[string]string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}
