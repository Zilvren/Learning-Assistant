package service

import (
	"testing"

	models "study-tracker-go/internal/model"
	store "study-tracker-go/internal/repository"
)

func TestBackupZipRoundTripPreservesLibraryData(t *testing.T) {
	folderID, noteID := int64(5), int64(6)
	hash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	input := store.BackupData{Library: &store.LibraryBackup{
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
	if len(files) != 2 {
		t.Fatalf("unexpected archive files: %#v", files)
	}
}
