package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	models "study-tracker-go/internal/model"
	store "study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

// TestBackupZipRoundTripPreservesLibraryData 在业务层中验证对应场景的行为与边界条件。
func TestBackupZipRoundTripPreservesLibraryData(t *testing.T) {
	folderID, noteID := int64(5), int64(6)
	hash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	config := models.Config{Username: "student"}
	errors := []models.ErrorProblem{}
	subjects := []string{}
	knowledge := map[string][]string{}
	input := store.BackupData{Config: &config, Errors: &errors, Subjects: &subjects, Knowledge: &knowledge, Library: &store.LibraryBackup{
		SchemaVersion: 2,
		NextID:        7,
		NextVersionID: 4,
		Items: []models.LibraryItem{
			{ID: folderID, Kind: "folder", Name: "物理"},
			{ID: noteID, ParentID: &folderID, Kind: "note", Name: "力学", CurrentVersion: 1, BlobHash: hash},
		},
		Versions: []models.LibraryVersion{{ID: 3, ItemID: noteID, Version: 1, BlobHash: hash}},
	}, Blobs: map[string][]byte{hash: []byte("content")}}

	archive, err := encodeBackupZip(input)
	if err != nil {
		t.Fatal(err)
	}
	actual, files, err := decodeBackupZip(archive)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Library == nil || len(actual.Library.Items) != 2 || len(actual.Library.Versions) != 1 {
		t.Fatalf("library data was not restored: %#v", actual.Library)
	}
	if actual.Library.Items[1].ParentID == nil || *actual.Library.Items[1].ParentID != folderID {
		t.Fatalf("library hierarchy was not restored: %#v", actual.Library.Items)
	}
	if string(actual.Blobs[hash]) != "content" {
		t.Fatalf("library blob was not restored: %#v", actual.Blobs)
	}
	if len(files) != 6 {
		t.Fatalf("unexpected archive files: %#v", files)
	}
}

// TestImportBackupZipRestoresVisibleLibraryItemsInJSONMode 在业务层中验证对应场景的行为与边界条件。
func TestImportBackupZipRestoresVisibleLibraryItemsInJSONMode(t *testing.T) {
	previousDir := store.DataDir()
	previousApp := DefaultApp()
	t.Cleanup(func() {
		store.SetDataDir(previousDir)
		legacyApp.Store(previousApp)
	})

	store.SetDataDir(t.TempDir())
	repos := jsonrepo.NewRepositories()
	if err := InitApp(config.Config{StorageDriver: "json"}, repos, nil); err != nil {
		t.Fatal(err)
	}

	content := []byte("# 备份后的笔记\n")
	digest := sha256.Sum256(content)
	hash := hex.EncodeToString(digest[:])
	folderID, noteID := int64(1), int64(2)
	config := models.Config{Username: "student"}
	errors := []models.ErrorProblem{}
	subjects := []string{}
	knowledge := map[string][]string{}
	archive, err := encodeBackupZip(store.BackupData{Config: &config, Errors: &errors, Subjects: &subjects, Knowledge: &knowledge, Library: &store.LibraryBackup{
		SchemaVersion: 2,
		NextID:        3,
		NextVersionID: 2,
		Items: []models.LibraryItem{
			{ID: folderID, Kind: "folder", Name: "Note"},
			{ID: noteID, ParentID: &folderID, Kind: "note", Name: "20260808.md", MimeType: "text/markdown", CurrentVersion: 1, BlobHash: hash, Size: int64(len(content))},
		},
		Versions: []models.LibraryVersion{{ID: 1, ItemID: noteID, Version: 1, BlobHash: hash, Size: int64(len(content))}},
	}, Blobs: map[string][]byte{hash: content}})
	if err != nil {
		t.Fatal(err)
	}

	result, err := ImportBackupZip(context.Background(), archive)
	if err != nil {
		t.Fatal(err)
	}
	if result.LibraryItems != 2 {
		t.Fatalf("expected two restored library items, got %#v", result)
	}
	roots, err := repos.Library.List(context.Background(), store.LibraryFilter{})
	if err != nil || len(roots) != 1 || roots[0].Name != "Note" {
		t.Fatalf("restored root folder is not visible: %#v, %v", roots, err)
	}
	children, err := repos.Library.List(context.Background(), store.LibraryFilter{ParentID: &roots[0].ID})
	if err != nil || len(children) != 1 || children[0].Name != "20260808.md" {
		t.Fatalf("restored note is not visible: %#v, %v", children, err)
	}
	restored, _, err := repos.Library.ReadContent(context.Background(), children[0].ID)
	if err != nil || string(restored) != string(content) {
		t.Fatalf("restored note content is not readable: %q, %v", restored, err)
	}
}
