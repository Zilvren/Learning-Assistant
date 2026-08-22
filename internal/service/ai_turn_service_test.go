package service

import (
	"context"
	"testing"
	"time"

	models "study-tracker-go/internal/model"
)

func TestAITurnWriteApprovalPausesAndResumesTask(t *testing.T) {
	ctx := setupAIChatServiceTest(t)
	turn := seedRunningAITurn(t, ctx, "turn-approval")
	result := make(chan error, 1)
	go func() {
		approved, err := RequestAITurnWriteApproval(ctx, turn.ID, models.AIWriteApproval{
			Tool: "update_note", Path: "数学/导数.md", ItemID: 8, BaseVersion: 3,
			OriginalContent: "# 导数\n旧内容", Content: "# 导数\n新内容",
		})
		if err != nil {
			result <- err
			return
		}
		if !approved {
			result <- ErrAIWriteRejected
			return
		}
		result <- nil
	}()

	pending := waitForAITurn(t, ctx, turn.ID, func(value models.AITurn) bool {
		return value.Status == models.AITurnWaitingApproval && value.Approval != nil
	})
	if pending.Approval.Tool != "update_note" || pending.Approval.OriginalContent == "" || pending.Approval.Content == "" {
		t.Fatalf("unexpected pending approval: %#v", pending.Approval)
	}
	updated, err := ResolveAITurnWriteApproval(ctx, turn.ID, pending.Approval.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != models.AITurnRunning || updated.Approval == nil || updated.Approval.Status != "approved" {
		t.Fatalf("unexpected resolved turn: %#v", updated)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("approval waiter did not resume")
	}
	if len(updated.Events) < 3 || updated.Events[len(updated.Events)-2].Type != "approval.required" || updated.Events[len(updated.Events)-1].Type != "approval.resolved" {
		t.Fatalf("expected replayable approval events, got %#v", updated.Events)
	}
}

func TestAITurnWriteApprovalRejectsWrite(t *testing.T) {
	ctx := setupAIChatServiceTest(t)
	turn := seedRunningAITurn(t, ctx, "turn-rejection")
	result := make(chan error, 1)
	go func() {
		_, err := RequestAITurnWriteApproval(ctx, turn.ID, models.AIWriteApproval{Tool: "create_note", Path: "复习清单.md", Content: "# 清单"})
		result <- err
	}()
	pending := waitForAITurn(t, ctx, turn.ID, func(value models.AITurn) bool {
		return value.Approval != nil && value.Approval.Status == "pending"
	})
	if _, err := ResolveAITurnWriteApproval(ctx, turn.ID, pending.Approval.ID, false); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != ErrAIWriteRejected {
			t.Fatalf("expected rejection error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("rejection waiter did not resume")
	}
}

func seedRunningAITurn(t *testing.T, ctx context.Context, id string) models.AITurn {
	t.Helper()
	now := time.Now().UTC()
	turn := models.AITurn{
		ID: id, ConversationID: id, Status: models.AITurnRunning, CreatedAt: now, UpdatedAt: now,
		Events: []models.AITurnEvent{{Sequence: 1, Type: "turn.running", Status: models.AITurnRunning, CreatedAt: now}},
	}
	if _, err := mutateAIConfig(ctx, func(config *models.Config) error {
		config.AITurns = append(config.AITurns, turn)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return turn
}

func waitForAITurn(t *testing.T, ctx context.Context, id string, ready func(models.AITurn) bool) models.AITurn {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		turn, err := GetAITurn(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if ready(turn) {
			return turn
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("AI task did not reach expected state")
	return models.AITurn{}
}
