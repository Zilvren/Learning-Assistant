package service

import (
	"archive/zip"
	"bytes"
	"testing"
)

// TestZipNamesUsesNaturalNumericOrder 验证当前模块在相应场景下的行为与边界条件。
func TestZipNamesUsesNaturalNumericOrder(t *testing.T) {
	archive := officePreviewArchive(t, map[string]string{
		"ppt/slides/slide10.xml": "<p/>",
		"ppt/slides/slide2.xml":  "<p/>",
		"ppt/slides/slide1.xml":  "<p/>",
	})
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	names := zipNames(reader, "ppt/slides/slide", ".xml")
	if len(names) != 3 || names[0] != "ppt/slides/slide1.xml" || names[1] != "ppt/slides/slide2.xml" || names[2] != "ppt/slides/slide10.xml" {
		t.Fatalf("unexpected preview order: %#v", names)
	}
}

// TestValidateOfficeArchiveRejectsSuspiciousCompression 验证当前模块在相应场景下的行为与边界条件。
func TestValidateOfficeArchiveRejectsSuspiciousCompression(t *testing.T) {
	archive := officePreviewArchive(t, map[string]string{"word/document.xml": string(bytes.Repeat([]byte("a"), 1<<20))})
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateOfficeArchive(reader); err == nil {
		t.Fatal("expected compressed document to be rejected")
	}
}

// officePreviewArchive 验证当前模块在相应场景下的行为与边界条件。
func officePreviewArchive(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
