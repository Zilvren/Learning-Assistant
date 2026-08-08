package jsonrepo

import (
	"context"
	"strings"
	"testing"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

func TestLibraryFoldersVersionsTrashAndMigration(t *testing.T) {
	previous := base.DataDir()
	base.SetDataDir(t.TempDir())
	t.Cleanup(func() { base.SetDataDir(previous) })
	repo := &LibraryRepository{store: base.DefaultJSONStore()}
	ctx := context.Background()
	if err := repo.EnsureLegacy(ctx, []models.ErrorProblem{{ID: 7, Subject: "数学", Title: "导数", Tags: []string{"函数"}}}, []string{"数学"}); err != nil {
		t.Fatal(err)
	}
	root, err := repo.List(ctx, base.LibraryFilter{})
	if err != nil || len(root) != 1 || root[0].Kind != "note" || !root[0].ReviewEnabled || !containsTag(root[0].Tags, "数学") {
		t.Fatalf("unexpected root: %#v %v", root, err)
	}
	body, _, err := repo.ReadContent(ctx, root[0].ID)
	if err != nil || !strings.Contains(string(body), "## 题目") {
		t.Fatalf("legacy error was not converted to markdown: %q %v", body, err)
	}
	tags, err := repo.ListTags(ctx)
	if err != nil || len(tags) != 2 {
		t.Fatalf("unexpected tags: %#v %v", tags, err)
	}
	if _, err = repo.Review(ctx, root[0].ID, time.Now(), []int{0, 1, 2}); err != nil {
		t.Fatal(err)
	}
	folder, err := repo.Create(ctx, models.CreateLibraryItemRequest{Kind: "folder", Name: "笔记"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	note, err := repo.Create(ctx, models.CreateLibraryItemRequest{ParentID: &folder.ID, Kind: "note", Name: "高数.md", MimeType: "text/markdown"}, []byte("第一版"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := repo.SaveContent(ctx, note.ID, []byte("第二版"), note.CurrentVersion, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SaveContent(ctx, note.ID, []byte("冲突"), note.CurrentVersion, true, false); err == nil {
		t.Fatal("expected version conflict")
	}
	body, _, err = repo.ReadContent(ctx, note.ID)
	if err != nil || string(body) != "第二版" {
		t.Fatalf("unexpected content %q %v", body, err)
	}
	versions, err := repo.Versions(ctx, note.ID)
	if err != nil || len(versions) != 2 || updated.CurrentVersion != 2 {
		t.Fatalf("unexpected versions %#v", versions)
	}
	var firstVersion models.LibraryVersion
	for _, version := range versions {
		if version.Version == 1 {
			firstVersion = version
			break
		}
	}
	if firstVersion.ID == 0 {
		t.Fatal("first version not found")
	}
	restored, err := repo.RestoreVersion(ctx, note.ID, firstVersion.ID)
	if err != nil {
		t.Fatal(err)
	}
	versionsAfterRestore, err := repo.Versions(ctx, note.ID)
	if err != nil || len(versionsAfterRestore) != len(versions) {
		t.Fatalf("restore should not create history: %#v %v", versionsAfterRestore, err)
	}
	body, _, err = repo.ReadContent(ctx, note.ID)
	if err != nil || string(body) != "第一版" || restored.CurrentVersion <= updated.CurrentVersion {
		t.Fatalf("unexpected restored content/version %q %#v %v", body, restored, err)
	}
	if _, err = repo.Update(ctx, folder.ID, models.UpdateLibraryItemRequest{ParentID: &note.ID}); err == nil {
		t.Fatal("folder cycle/invalid parent should fail")
	}
	if err = repo.Trash(ctx, folder.ID); err != nil {
		t.Fatal(err)
	}
	trash, err := repo.List(ctx, base.LibraryFilter{Trashed: true})
	if err != nil || len(trash) != 1 || trash[0].ID != folder.ID {
		t.Fatalf("expected only trashed folder root: %#v %v", trash, err)
	}
	if _, err = repo.Restore(ctx, folder.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.Trash(ctx, note.ID); err != nil {
		t.Fatal(err)
	}
	trash, err = repo.List(ctx, base.LibraryFilter{Trashed: true})
	if err != nil || len(trash) != 1 || trash[0].ID != note.ID {
		t.Fatalf("expected individually trashed note: %#v %v", trash, err)
	}
}
