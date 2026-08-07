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
