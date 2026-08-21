package service

import (
	"context"
	"testing"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

// TestPurgeLegacyReviewDoesNotReappear 在业务层中验证对应场景的行为与边界条件。
func TestPurgeLegacyReviewDoesNotReappear(t *testing.T) {
	previousDir := base.DataDir()
	previousApp := DefaultApp()
	t.Cleanup(func() {
		base.SetDataDir(previousDir)
		legacyApp.Store(previousApp)
	})

	base.SetDataDir(t.TempDir())
	repos := jsonrepo.NewRepositories()
	if err := InitApp(config.Config{StorageDriver: "json"}, repos, nil); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := repos.Errors.Create(ctx, models.ErrorProblem{Title: "旧错题"}); err != nil {
		t.Fatal(err)
	}
	legacy, err := repos.Errors.List(ctx, base.ErrorFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Library.EnsureLegacy(ctx, legacy, nil); err != nil {
		t.Fatal(err)
	}

	due, err := DueLibraryReviews(ctx)
	if err != nil || len(due) != 1 {
		t.Fatalf("expected one migrated review note, got %#v, %v", due, err)
	}
	noteID := due[0].ID
	if err = TrashLibraryItem(ctx, noteID); err != nil {
		t.Fatal(err)
	}
	if due, err = DueLibraryReviews(ctx); err != nil || len(due) != 0 {
		t.Fatalf("trashed note must not be reviewable: %#v, %v", due, err)
	}
	if err = PurgeLibraryItem(ctx, noteID); err != nil {
		t.Fatal(err)
	}
	if due, err = DueLibraryReviews(ctx); err != nil || len(due) != 0 {
		t.Fatalf("purged legacy note must not reappear in reviews: %#v, %v", due, err)
	}
	legacy, err = repos.Errors.List(ctx, base.ErrorFilter{})
	if err != nil || len(legacy) != 1 {
		t.Fatalf("permanently deleting a note must preserve its source error: %#v, %v", legacy, err)
	}
}

// TestPostgresLibraryBrowsingDoesNotMigrateLegacyErrors 在业务层中验证对应场景的行为与边界条件。
func TestPostgresLibraryBrowsingDoesNotMigrateLegacyErrors(t *testing.T) {
	previousDir := base.DataDir()
	previousApp := DefaultApp()
	t.Cleanup(func() {
		base.SetDataDir(previousDir)
		legacyApp.Store(previousApp)
	})

	base.SetDataDir(t.TempDir())
	repos := jsonrepo.NewRepositories()
	// 在测试中保留内存仓储，同时验证 PostgreSQL 模式的 Service 行为：浏览不能写入旧版错题笔记。
	if err := InitApp(config.Config{StorageDriver: "postgres"}, repos, nil); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := repos.Errors.Create(ctx, models.ErrorProblem{Title: "旧错题"}); err != nil {
		t.Fatal(err)
	}

	items, err := ListLibrary(ctx, base.LibraryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("browsing PostgreSQL-mode library must not migrate legacy errors: %#v", items)
	}
}
