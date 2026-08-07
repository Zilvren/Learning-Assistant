package service

import (
	"context"
	"testing"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

func TestPurgeLegacyReviewDoesNotReappear(t *testing.T) {
	previousDir := base.DataDir()
	defaultMu.RLock()
	previousRepos, previousConfig, previousPool := defaultRepos, appConfig, pgPool
	defaultMu.RUnlock()
	t.Cleanup(func() {
		base.SetDataDir(previousDir)
		defaultMu.Lock()
		defaultRepos, appConfig, pgPool = previousRepos, previousConfig, previousPool
		defaultMu.Unlock()
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
	legacy, err := repos.Errors.List(ctx, base.ErrorFilter{})
	if err != nil || len(legacy) != 0 {
		t.Fatalf("legacy source should be removed with permanent deletion: %#v, %v", legacy, err)
	}
}
