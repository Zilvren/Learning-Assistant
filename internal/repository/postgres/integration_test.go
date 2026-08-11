package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

// TestPostgresRepositoriesIntegration 在存储层中验证对应场景的行为与边界条件。
func TestPostgresRepositoriesIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	username := fmt.Sprintf("test_%d", time.Now().UnixNano())
	var userID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, status)
		VALUES ($1, 'test-only', 'active')
		RETURNING id
	`, username).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	repos := NewRepositories(pool, userID)
	previousDataDir := base.DataDir()
	base.SetDataDir(t.TempDir())
	t.Cleanup(func() { base.SetDataDir(previousDataDir) })

	note, err := repos.Library.Create(ctx, models.CreateLibraryItemRequest{Kind: "note", Name: "正文搜索"}, []byte("这是一段只存在于笔记正文中的检索词。"))
	if err != nil {
		t.Fatal(err)
	}
	matchedNotes, err := repos.Library.List(ctx, base.LibraryFilter{Query: "检索词"})
	if err != nil || len(matchedNotes) != 1 || matchedNotes[0].ID != note.ID {
		t.Fatalf("expected note body search result: %#v %v", matchedNotes, err)
	}

	subjects, err := repos.Subjects.Create(ctx, "数学")
	if err != nil {
		t.Fatal(err)
	}
	if len(subjects) != 1 || subjects[0] != "数学" {
		t.Fatalf("unexpected subjects: %#v", subjects)
	}

	item, err := repos.Errors.Create(ctx, models.ErrorProblem{
		Subject:     "数学",
		Title:       "函数单调性",
		Question:    "判断 f(x)=x^2 在 R 上是否单调",
		Wrong:       "单调递增",
		Correct:     "不是单调函数",
		Reason:      "忽略负半轴",
		Tags:        []string{"函数"},
		ReasonTags:  []string{"概念不清"},
		Created:     "2026-07-01 12:00:00",
		NextReview:  "2026-07-01",
		ReviewCount: 0,
		ReviewStage: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.ID == 0 {
		t.Fatal("expected generated id")
	}

	filtered, err := repos.Errors.List(ctx, base.ErrorFilter{Tag: "函数"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].Tags[0] != "函数" {
		t.Fatalf("unexpected filtered errors: %#v", filtered)
	}

	reviewedAt := time.Date(2026, 7, 1, 12, 10, 0, 0, time.Local)
	reviewed, err := repos.Errors.Review(ctx, item.ID, reviewedAt, []int{0, 1, 2, 4, 7, 15})
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.ReviewCount != 1 || reviewed.NextReview != "2026-07-02" {
		t.Fatalf("unexpected reviewed item: %#v", reviewed)
	}

	if err := repos.Settings.Save(ctx, models.Config{Username: "Tester", MineruToken: "token-123"}); err != nil {
		t.Fatal(err)
	}
	config, err := repos.Settings.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if config.Username != "Tester" || config.MineruToken != "token-123" {
		t.Fatalf("unexpected config: %#v", config)
	}

	if err := repos.Knowledge.Replace(ctx, map[string][]string{"数学": []string{"函数要分区间讨论"}}); err != nil {
		t.Fatal(err)
	}
	knowledge, err := repos.Knowledge.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(knowledge["数学"]) != 1 {
		t.Fatalf("unexpected knowledge: %#v", knowledge)
	}

	taskID, err := repos.OCRTasks.Create(ctx, base.OCRTask{
		Provider:       "mineru",
		Status:         "pending",
		SourceFilename: "ocr.png",
		FileSize:       100,
	})
	if err != nil {
		t.Fatal(err)
	}
	finishedAt := time.Now()
	if err := repos.OCRTasks.Update(ctx, taskID, base.OCRTask{Status: "succeeded", ResultMarkdown: "# ok", FinishedAt: &finishedAt}); err != nil {
		t.Fatal(err)
	}

	backup, err := repos.Backup.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if backup.Errors == nil || len(*backup.Errors) != 1 {
		t.Fatalf("unexpected backup: %#v", backup)
	}
}

// TestPostgresImportAllowsDuplicateLegacyIDsAcrossUsers 在存储层中验证对应场景的行为与边界条件。
func TestPostgresImportAllowsDuplicateLegacyIDsAcrossUsers(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var firstUserID int64
	var secondUserID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, status)
		VALUES ($1, 'test-only', 'active')
		RETURNING id
	`, fmt.Sprintf("legacy_a_%d", time.Now().UnixNano())).Scan(&firstUserID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, status)
		VALUES ($1, 'test-only', 'active')
		RETURNING id
	`, fmt.Sprintf("legacy_b_%d", time.Now().UnixNano())).Scan(&secondUserID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = ANY($1)`, []int64{firstUserID, secondUserID})
	})

	firstRepos := NewRepositories(pool, firstUserID)
	if _, err := firstRepos.Subjects.Create(ctx, "数学"); err != nil {
		t.Fatal(err)
	}
	first, err := firstRepos.Errors.Create(ctx, models.ErrorProblem{
		Subject:    "数学",
		Title:      "已有错题",
		Question:   "A",
		Wrong:      "A",
		Correct:    "B",
		Reason:     "测试",
		Created:    "2026-07-01 12:00:00",
		NextReview: "2026-07-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	secondRepos := NewRepositories(pool, secondUserID)
	data := base.BackupData{
		Subjects: &[]string{"数学"},
		Errors: &[]models.ErrorProblem{{
			ID:         first.ID,
			Subject:    "数学",
			Title:      "导入错题",
			Question:   "C",
			Wrong:      "C",
			Correct:    "D",
			Reason:     "测试",
			Created:    "2026-07-01 12:00:00",
			NextReview: "2026-07-01",
		}},
	}
	if err := secondRepos.Backup.Import(ctx, data); err != nil {
		t.Fatal(err)
	}
	imported, err := secondRepos.Errors.List(ctx, base.ErrorFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 1 {
		t.Fatalf("expected one imported problem, got %#v", imported)
	}
	if imported[0].ID == first.ID {
		t.Fatalf("expected duplicate legacy id to be regenerated, got %d", imported[0].ID)
	}
}

// TestPostgresBackupImportRestoresLibraryItemsAndVersions 在存储层中验证对应场景的行为与边界条件。
func TestPostgresBackupImportRestoresLibraryItemsAndVersions(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var userID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, status)
		VALUES ($1, 'test-only', 'active')
		RETURNING id
	`, fmt.Sprintf("library_backup_%d", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	folderID, noteID := int64(41), int64(42)
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	repos := NewRepositories(pool, userID)
	data := base.BackupData{Library: &base.LibraryBackup{
		SchemaVersion: 2,
		Items: []models.LibraryItem{
			{ID: folderID, Kind: "folder", Name: "微积分", Tags: []string{"数学"}},
			{ID: noteID, ParentID: &folderID, Kind: "note", Name: "极限笔记", MimeType: "text/markdown", Size: 12, Tags: []string{"极限"}, CurrentVersion: 1, BlobHash: hash, ReviewEnabled: true},
		},
		Versions: []models.LibraryVersion{{ID: 9, ItemID: noteID, Version: 1, BlobHash: hash, Size: 12}},
	}}
	if err := repos.Backup.Import(ctx, data); err != nil {
		t.Fatal(err)
	}
	var importedActivity int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM user_activity_events
		WHERE user_id = $1 AND event_type = 'library_import'
	`, userID).Scan(&importedActivity); err != nil {
		t.Fatal(err)
	}
	if importedActivity != 1 {
		t.Fatalf("expected one import activity for the restored note, got %d", importedActivity)
	}

	roots, err := repos.Library.List(ctx, base.LibraryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0].Name != "微积分" {
		t.Fatalf("unexpected restored library roots: %#v", roots)
	}
	children, err := repos.Library.List(ctx, base.LibraryFilter{ParentID: &roots[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 || children[0].Name != "极限笔记" || children[0].BlobHash != hash {
		t.Fatalf("unexpected restored library children: %#v", children)
	}
	versions, err := repos.Library.Versions(ctx, children[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].Version != 1 || versions[0].BlobHash != hash {
		t.Fatalf("unexpected restored library versions: %#v", versions)
	}

	exported, err := repos.Backup.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Library == nil || len(exported.Library.Items) != 2 || len(exported.Library.Versions) != 1 {
		t.Fatalf("library was not included in backup export: %#v", exported.Library)
	}
	if err := repos.Library.Trash(ctx, roots[0].ID); err != nil {
		t.Fatal(err)
	}
	trashed, err := repos.Library.List(ctx, base.LibraryFilter{Trashed: true})
	if err != nil || len(trashed) != 1 || trashed[0].ID != roots[0].ID {
		t.Fatalf("expected only trashed library root: %#v %v", trashed, err)
	}
	if _, err := repos.Library.Restore(ctx, roots[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := repos.Backup.Import(ctx, data); err != nil {
		t.Fatalf("restoring over an existing folder tree failed: %v", err)
	}
	roots, err = repos.Library.List(ctx, base.LibraryFilter{})
	if err != nil || len(roots) != 1 || roots[0].Name != "微积分" {
		t.Fatalf("restored ZIP library root is not visible: %#v, %v", roots, err)
	}
}

// TestPostgresPoolReopenDoesNotCreateActivity 在存储层中验证对应场景的行为与边界条件。
func TestPostgresPoolReopenDoesNotCreateActivity(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	username := fmt.Sprintf("restart_activity_%d", time.Now().UnixNano())
	var userID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, status)
		VALUES ($1, 'test-only', 'active')
		RETURNING id
	`, username).Scan(&userID); err != nil {
		pool.Close()
		t.Fatal(err)
	}

	repos := NewRepositories(pool, userID)
	if _, err := repos.Library.Create(ctx, models.CreateLibraryItemRequest{Kind: "note", Name: "重启测试"}, []byte("# 测试")); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_activity_events WHERE user_id = $1`, userID).Scan(&before); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	pool.Close()

	reopened, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	t.Cleanup(func() {
		_, _ = reopened.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	var after int
	if err := reopened.QueryRow(ctx, `SELECT count(*) FROM user_activity_events WHERE user_id = $1`, userID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("reopening the pool wrote activity rows: before=%d after=%d", before, after)
	}
}
