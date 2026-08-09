package jsonrepo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

// TestConcurrentCreatesAndReviews 在存储层中验证对应场景的行为与边界条件。
func TestConcurrentCreatesAndReviews(t *testing.T) {
	restoreDataDir(t)
	base.SetDataDir(t.TempDir())
	repos := NewRepositories()
	ctx := context.Background()
	if err := repos.Subjects.Replace(ctx, []string{"数学"}); err != nil {
		t.Fatal(err)
	}

	const createCount = 50
	var wg sync.WaitGroup
	errs := make(chan error, createCount)
	for i := 0; i < createCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := repos.Errors.Create(ctx, testProblem(fmt.Sprintf("题目-%d", index)))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	items, err := repos.Errors.List(ctx, base.ErrorFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != createCount {
		t.Fatalf("expected %d errors, got %d", createCount, len(items))
	}
	ids := map[int]bool{}
	for _, item := range items {
		if ids[item.ID] {
			t.Fatalf("duplicate id %d", item.ID)
		}
		ids[item.ID] = true
	}

	const reviewCount = 25
	errs = make(chan error, reviewCount)
	reviewedAt := time.Date(2026, 7, 23, 12, 0, 0, 0, time.Local)
	for i := 0; i < reviewCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repos.Errors.Review(ctx, items[0].ID, reviewedAt, []int{0, 1, 2, 4, 7, 15})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	reviewed, err := repos.Errors.Get(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.ReviewCount != reviewCount {
		t.Fatalf("expected review count %d, got %d", reviewCount, reviewed.ReviewCount)
	}
}

// TestCrossProcessCreates 在存储层中验证对应场景的行为与边界条件。
func TestCrossProcessCreates(t *testing.T) {
	if os.Getenv("TRACKER_JSON_HELPER") == "1" {
		runJSONProcessHelper(t)
		return
	}

	restoreDataDir(t)
	dir := t.TempDir()
	base.SetDataDir(dir)
	repos := NewRepositories()
	if err := repos.Subjects.Replace(context.Background(), []string{"数学"}); err != nil {
		t.Fatal(err)
	}

	const perProcess = 20
	commands := []*exec.Cmd{
		jsonHelperCommand(dir, "left", perProcess),
		jsonHelperCommand(dir, "right", perProcess),
	}
	outputs := make([][]byte, len(commands))
	errs := make([]error, len(commands))
	var wg sync.WaitGroup
	for i := range commands {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			outputs[index], errs[index] = commands[index].CombinedOutput()
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("helper %d failed: %v\n%s", i, err, outputs[i])
		}
	}

	items, err := repos.Errors.List(context.Background(), base.ErrorFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != perProcess*len(commands) {
		t.Fatalf("expected %d errors, got %d", perProcess*len(commands), len(items))
	}
	ids := map[int]bool{}
	for _, item := range items {
		if ids[item.ID] {
			t.Fatalf("duplicate id %d", item.ID)
		}
		ids[item.ID] = true
	}
}

// TestBackupExportWaitsForWriteTransaction 在存储层中验证对应场景的行为与边界条件。
func TestBackupExportWaitsForWriteTransaction(t *testing.T) {
	store := base.NewJSONStore(t.TempDir())
	backup := &BackupRepository{store: store}
	locked := make(chan struct{})
	release := make(chan struct{})
	writerDone := make(chan error, 1)
	go func() {
		writerDone <- store.Write(context.Background(), func(tx *base.JSONTx) error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	exportDone := make(chan error, 1)
	go func() {
		_, err := backup.Export(context.Background())
		exportDone <- err
	}()
	select {
	case err := <-exportDone:
		t.Fatalf("backup export returned while a write transaction was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-writerDone; err != nil {
		t.Fatal(err)
	}
	if err := <-exportDone; err != nil {
		t.Fatal(err)
	}
}

// runJSONProcessHelper 在存储层中执行流程或启动外部操作。
func runJSONProcessHelper(t *testing.T) {
	dir := os.Getenv("TRACKER_JSON_DATA_DIR")
	prefix := os.Getenv("TRACKER_JSON_PREFIX")
	count, err := strconv.Atoi(os.Getenv("TRACKER_JSON_COUNT"))
	if err != nil {
		t.Fatal(err)
	}
	base.SetDataDir(dir)
	repos := NewRepositories()
	for i := 0; i < count; i++ {
		if _, err := repos.Errors.Create(context.Background(), testProblem(fmt.Sprintf("%s-%d", prefix, i))); err != nil {
			t.Fatal(err)
		}
	}
}

// jsonHelperCommand 在存储层中完成本文件定义的局部处理。
func jsonHelperCommand(dir string, prefix string, count int) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestCrossProcessCreates$")
	cmd.Env = append(os.Environ(),
		"TRACKER_JSON_HELPER=1",
		"TRACKER_JSON_DATA_DIR="+dir,
		"TRACKER_JSON_PREFIX="+prefix,
		"TRACKER_JSON_COUNT="+strconv.Itoa(count),
	)
	return cmd
}

// testProblem 在存储层中完成本文件定义的局部处理。
func testProblem(question string) models.ErrorProblem {
	return models.ErrorProblem{
		Subject:    "数学",
		Title:      question,
		Question:   question,
		Wrong:      "未记录",
		Correct:    "未记录",
		Reason:     "未记录",
		Tags:       []string{},
		ReasonTags: []string{},
		Created:    "2026-07-23 12:00:00",
		NextReview: "2026-07-23",
	}
}

// restoreDataDir 在存储层中完成本文件定义的局部处理。
func restoreDataDir(t *testing.T) {
	previous := base.DataDir()
	t.Cleanup(func() { base.SetDataDir(previous) })
}
