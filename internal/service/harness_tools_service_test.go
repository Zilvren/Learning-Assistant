package service

import (
	"errors"
	"testing"

	models "study-tracker-go/internal/model"
)

// TestHarnessToolGrantKeepsReadsAndWritesInsideTheConversationScope 验证当前模块在相应场景下的行为与边界条件。
func TestHarnessToolGrantKeepsReadsAndWritesInsideTheConversationScope(t *testing.T) {
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

	created, err := ExecuteHarnessTool(ctx, token, "create_note", map[string]any{
		"path": "复习清单", "content": "# 导数复习\n\n- 单调性\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	createdResult, ok := created.(map[string]any)
	if !ok || createdResult["written"] != true || createdResult["action"] != "created" {
		t.Fatalf("expected direct create result, got %#v", created)
	}
	createdID, ok := createdResult["item_id"].(int64)
	if !ok || createdID <= 0 {
		t.Fatalf("expected created id, got %#v", createdResult)
	}
	createdBody, createdItem, err := ReadLibraryContent(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if createdItem.ParentID == nil || *createdItem.ParentID != folder.ID || string(createdBody) != "# 导数复习\n\n- 单调性\n" {
		t.Fatalf("unexpected direct create: item=%#v body=%q", createdItem, createdBody)
	}
	if _, err := ExecuteHarnessTool(ctx, token, "create_note", map[string]any{"path": "复习清单", "content": "重复"}); err == nil {
		t.Fatal("expected create to reject an existing path")
	}

	baseVersion, ok := read.(map[string]any)["current_version"].(int)
	if !ok || baseVersion != inside.CurrentVersion {
		t.Fatalf("expected current version in read result, got %#v", read)
	}
	updated, err := ExecuteHarnessTool(ctx, token, "update_note", map[string]any{
		"item_id": float64(inside.ID), "base_version": float64(baseVersion), "content": "# 导数\n\n已由 Harness 更新。",
	})
	if err != nil {
		t.Fatal(err)
	}
	updatedResult, ok := updated.(map[string]any)
	if !ok || updatedResult["written"] != true || updatedResult["action"] != "updated" {
		t.Fatalf("expected direct update result, got %#v", updated)
	}
	updatedBody, updatedItem, err := ReadLibraryContent(ctx, inside.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(updatedBody) != "# 导数\n\n已由 Harness 更新。" || updatedItem.CurrentVersion <= baseVersion {
		t.Fatalf("unexpected direct update: item=%#v body=%q", updatedItem, updatedBody)
	}
	if _, err := ExecuteHarnessTool(ctx, token, "update_note", map[string]any{
		"item_id": float64(inside.ID), "base_version": float64(baseVersion), "content": "不应覆盖新版本",
	}); err == nil {
		t.Fatal("expected stale update to fail instead of overwriting")
	}
	if _, err := ExecuteHarnessTool(ctx, token, "prepare_note_change", map[string]any{}); !errors.Is(err, ErrHarnessToolUnavailable) {
		t.Fatalf("expected retired preview tool to be unavailable, got %v", err)
	}
	if _, err := ExecuteHarnessTool(ctx, token, "delete_note", map[string]any{}); !errors.Is(err, ErrHarnessToolUnavailable) {
		t.Fatalf("expected unavailable tool error, got %v", err)
	}
}

// TestHarnessToolGrantExpiresWhenRevoked 验证当前模块在相应场景下的行为与边界条件。
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
