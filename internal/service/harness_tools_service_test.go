package service

import (
	"errors"
	"testing"

	models "study-tracker-go/internal/model"
)

func TestHarnessToolGrantKeepsReadsAndPreviewInsideTheConversationScope(t *testing.T) {
	ctx := setupAIChatServiceTest(t)
	folder, err := CreateLibraryItem(ctx, models.CreateLibraryItemRequest{Kind: "folder", Name: "数学"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inside, err := CreateLibraryItem(ctx, models.CreateLibraryItemRequest{ParentID: &folder.ID, Kind: "note", Name: "导数.md", MimeType: "text/markdown"}, []byte("# 导数\n\n先求导。"))
	if err != nil {
		t.Fatal(err)
	}
	outside, err := CreateLibraryItem(ctx, models.CreateLibraryItemRequest{Kind: "note", Name: "化学.md", MimeType: "text/markdown"}, []byte("# 化学\n\n酸碱中和。"))
	if err != nil {
		t.Fatal(err)
	}

	token, err := NewHarnessToolGrant(ctx, models.AIChatRequest{Message: "整理导数", FolderID: &folder.ID})
	if err != nil {
		t.Fatal(err)
	}
	defer RevokeHarnessToolGrant(token)

	read, err := ExecuteHarnessTool(ctx, token, "read_note", map[string]any{"item_id": float64(inside.ID)})
	if err != nil {
		t.Fatal(err)
	}
	if content, _ := read.(map[string]any)["content"].(string); content != "# 导数\n\n先求导。" {
		t.Fatalf("unexpected scoped note content: %q", content)
	}
	if sources := ConsumeHarnessSources(token); len(sources) != 1 || sources[0].ID != inside.ID {
		t.Fatalf("expected the read note to be cited, got %#v", sources)
	}
	if _, err := ExecuteHarnessTool(ctx, token, "read_note", map[string]any{"item_id": float64(outside.ID)}); err == nil {
		t.Fatal("expected out-of-scope read to fail")
	}

	prepared, err := ExecuteHarnessTool(ctx, token, "prepare_note_change", map[string]any{
		"path": "复习清单", "content": "# 导数复习\n\n- 单调性\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := prepared.(map[string]any)["requires_user_confirmation"].(bool); !ok {
		t.Fatalf("expected preview-only result, got %#v", prepared)
	}
	preview := ConsumeHarnessNoteWritePreview(token)
	if preview == nil || preview.Action != "create" || preview.ParentID == nil || *preview.ParentID != folder.ID {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if _, err := GetLibraryItem(ctx, folder.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := ExecuteHarnessTool(ctx, token, "unknown", map[string]any{}); !errors.Is(err, ErrHarnessToolUnavailable) {
		t.Fatalf("expected unavailable tool error, got %v", err)
	}
}

func TestHarnessToolGrantExpiresWhenRevoked(t *testing.T) {
	ctx := setupAIChatServiceTest(t)
	token, err := NewHarnessToolGrant(ctx, models.AIChatRequest{Message: "你好"})
	if err != nil {
		t.Fatal(err)
	}
	RevokeHarnessToolGrant(token)
	if _, err := ExecuteHarnessTool(ctx, token, "list_paths", map[string]any{}); !errors.Is(err, ErrHarnessToolUnauthorized) {
		t.Fatalf("expected revoked capability to fail, got %v", err)
	}
}
