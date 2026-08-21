package postgres

import (
	"bytes"
	"reflect"
	"testing"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

// TestPostgresLibraryMatchesQueryIncludesNoteBody 验证即使笔记正文未出现在文件名和标签中，仍可被搜索到。
func TestPostgresLibraryMatchesQueryIncludesNoteBody(t *testing.T) {
	previousDataDir := base.DataDir()
	base.SetDataDir(t.TempDir())
	t.Cleanup(func() { base.SetDataDir(previousDataDir) })

	hash, _, err := base.StoreBlob(bytes.NewReader([]byte("只出现在笔记正文里的关键词")))
	if err != nil {
		t.Fatal(err)
	}
	item := models.LibraryItem{Kind: "note", Name: "Redis 笔记", Tags: []string{"缓存"}, BlobHash: hash}
	if !postgresLibraryMatchesQuery(item, "关键词") {
		t.Fatal("expected note body to match the library query")
	}
	if !postgresLibraryMatchesQuery(item, "缓存") {
		t.Fatal("expected tag to match the library query")
	}
}

// TestNormalizeLibraryTagsAlwaysReturnsNonNilSlice 在存储层中验证对应场景的行为与边界条件。
func TestNormalizeLibraryTagsAlwaysReturnsNonNilSlice(t *testing.T) {
	tags := normalizeLibraryTags(nil)
	if tags == nil {
		t.Fatal("expected an empty slice, not nil")
	}
	if len(tags) != 0 {
		t.Fatalf("expected no tags, got %#v", tags)
	}
}

// TestNormalizeLibraryTagsTrimsAndDeduplicates 在存储层中验证对应场景的行为与边界条件。
func TestNormalizeLibraryTagsTrimsAndDeduplicates(t *testing.T) {
	tags := normalizeLibraryTags([]string{" 数学 ", "", "数学", "英语", "英语 "})
	want := []string{"数学", "英语"}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("unexpected tags: got %#v, want %#v", tags, want)
	}
}
