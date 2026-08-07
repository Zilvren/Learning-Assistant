package postgres

import (
	"reflect"
	"testing"
)

func TestNormalizeLibraryTagsAlwaysReturnsNonNilSlice(t *testing.T) {
	tags := normalizeLibraryTags(nil)
	if tags == nil {
		t.Fatal("expected an empty slice, not nil")
	}
	if len(tags) != 0 {
		t.Fatalf("expected no tags, got %#v", tags)
	}
}

func TestNormalizeLibraryTagsTrimsAndDeduplicates(t *testing.T) {
	tags := normalizeLibraryTags([]string{" 数学 ", "", "数学", "英语", "英语 "})
	want := []string{"数学", "英语"}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("unexpected tags: got %#v, want %#v", tags, want)
	}
}
