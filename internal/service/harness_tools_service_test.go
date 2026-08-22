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

	if _, err := ExecuteHarnessTool(ctx, token, "create_note", map[string]any{
		"path": "复习清单", "content": "# 导数复习\n\n- 单调性\n",
	}); !errors.Is(err, ErrAIWriteApprovalRequired) {
		t.Fatalf("expected write approval requirement, got %v", err)
	}
	if _, err := ExecuteHarnessTool(ctx, token, "update_note", map[string]any{
		"item_id": float64(inside.ID), "base_version": float64(inside.CurrentVersion), "content": "不应覆盖",
	}); !errors.Is(err, ErrAIWriteApprovalRequired) {
		t.Fatalf("expected update approval requirement, got %v", err)
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

// TestChatFirstToolGrantStillAllowsExplicitLibraryRequests 验证聊天优先不会撤销资料库能力，只由提示词避免模型主动使用它。
func TestChatFirstToolGrantStillAllowsExplicitLibraryRequests(t *testing.T) {
	ctx := setupAIChatServiceTest(t)
	note, err := CreateLibraryItem(ctx, models.CreateLibraryItemRequest{Kind: "note", Name: "学习记录.md", MimeType: "text/markdown"}, []byte("# 学习记录\n"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := NewHarnessToolGrant(ctx, models.AIChatRequest{Message: "先陪我聊天", ChatOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer RevokeHarnessToolGrant(token)

	result, err := ExecuteHarnessTool(ctx, token, "read_note", map[string]any{"item_id": float64(note.ID)})
	if err != nil {
		t.Fatalf("explicit library request should remain available: %v", err)
	}
	if content, _ := result.(map[string]any)["content"].(string); content != "# 学习记录\n" {
		t.Fatalf("unexpected note content: %q", content)
	}
}
