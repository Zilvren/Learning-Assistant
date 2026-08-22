package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	models "study-tracker-go/internal/model"
)

const (
	aiTurnMaxStored        = 80
	aiTurnMaxEvents        = 160
	aiTurnSubscriberBuffer = 24
)

var (
	ErrAITurnNotFound          = errors.New("AI 任务不存在")
	ErrAITurnAlreadyRunning    = errors.New("当前对话已有正在执行的 AI 任务")
	ErrAITurnNotCancellable    = errors.New("当前 AI 任务不能取消")
	ErrAIWriteApprovalRequired = errors.New("写入资料库前需要用户确认；请通过可恢复的 AI 任务发起写入")
	ErrAIWriteApprovalNotFound = errors.New("写入确认不存在或已失效")
	ErrAIWriteApprovalResolved = errors.New("写入确认已经处理")
	ErrAIWriteRejected         = errors.New("用户未批准本次资料库写入")
)

type aiTurnRuntime struct {
	cancel      context.CancelFunc
	subscribers map[chan models.AITurnEvent]struct{}
}

var aiTurnRuntimes = struct {
	sync.Mutex
	items map[string]*aiTurnRuntime
}{items: make(map[string]*aiTurnRuntime)}

type aiWriteApprovalRuntime struct {
	turnID   string
	decision chan bool
}

var aiWriteApprovalRuntimes = struct {
	sync.Mutex
	items map[string]aiWriteApprovalRuntime
}{items: make(map[string]aiWriteApprovalRuntime)}

// StartAITurn 创建可恢复的应用侧任务。模型运行在后台，浏览器断开不会取消该任务。
func StartAITurn(ctx context.Context, request models.AIChatRequest) (models.AITurn, error) {
	if strings.TrimSpace(request.Message) == "" {
		return models.AITurn{}, fmt.Errorf("请输入想问 AI 的学习问题")
	}
	if !validAIConversationID(strings.TrimSpace(request.ConversationID)) {
		return models.AITurn{}, fmt.Errorf("%w：对话标识无效", ErrAIInvalidScope)
	}
	if _, err := newAILibraryScope(request); err != nil {
		return models.AITurn{}, err
	}

	now := time.Now().UTC()
	turn := models.AITurn{
		ID:             newAITurnID(),
		ConversationID: strings.TrimSpace(request.ConversationID),
		Status:         models.AITurnQueued,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	turn.Events = []models.AITurnEvent{{Sequence: 1, Type: "turn.started", Status: turn.Status, CreatedAt: now}}
	if _, err := mutateAIConfig(ctx, func(config *models.Config) error {
		for _, existing := range config.AITurns {
			if existing.ConversationID == turn.ConversationID && aiTurnActive(existing.Status) {
				return ErrAITurnAlreadyRunning
			}
		}
		config.AITurns = append(config.AITurns, turn)
		if len(config.AITurns) > aiTurnMaxStored {
			config.AITurns = config.AITurns[len(config.AITurns)-aiTurnMaxStored:]
		}
		return nil
	}); err != nil {
		return models.AITurn{}, err
	}

	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	aiTurnRuntimes.Lock()
	aiTurnRuntimes.items[turn.ID] = &aiTurnRuntime{cancel: cancel, subscribers: make(map[chan models.AITurnEvent]struct{})}
	aiTurnRuntimes.Unlock()
	request.TurnID = turn.ID
	go runAITurn(runCtx, turn.ID, request)
	return cloneAITurn(turn), nil
}

// GetAITurn 返回当前用户拥有的一条 AI 任务和其可重放事件。
func GetAITurn(ctx context.Context, id string) (models.AITurn, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return models.AITurn{}, ErrAITurnNotFound
	}
	config, err := loadConfig(ctx)
	if err != nil {
		return models.AITurn{}, err
	}
	for _, turn := range config.AITurns {
		if turn.ID == id {
			return cloneAITurn(turn), nil
		}
	}
	return models.AITurn{}, ErrAITurnNotFound
}

// ListAITurns 返回指定对话的最近任务，供刷新页面后恢复正在生成的消息。
func ListAITurns(ctx context.Context, conversationID string) ([]models.AITurn, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]models.AITurn, 0, len(config.AITurns))
	for _, turn := range config.AITurns {
		if conversationID == "" || turn.ConversationID == conversationID {
			result = append(result, cloneAITurn(turn))
		}
	}
	return result, nil
}

// CancelAITurn 显式终止后台 Agent；已结束任务不会被覆盖为取消状态。
func CancelAITurn(ctx context.Context, id string) (models.AITurn, error) {
	turn, changed, err := updateAITurn(ctx, id, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
		if !aiTurnActive(turn.Status) {
			return models.AITurnEvent{}, false, ErrAITurnNotCancellable
		}
		turn.Status = models.AITurnCancelled
		turn.Error = "已由用户取消"
		return models.AITurnEvent{Type: "turn.cancelled", Status: turn.Status, Error: turn.Error}, true, nil
	})
	if err != nil {
		return models.AITurn{}, err
	}
	if changed {
		cancelAITurnRuntime(id)
		finishAITurnRuntime(id)
	}
	return turn, nil
}

// RequestAITurnWriteApproval 将一次资料库副作用暂停在应用边界，直到用户明确同意或拒绝。
// 该函数只接受由后台任务创建的 TurnID，因此旧的同步聊天接口不能绕过确认直接写入。
func RequestAITurnWriteApproval(ctx context.Context, turnID string, approval models.AIWriteApproval) (bool, error) {
	turnID = strings.TrimSpace(turnID)
	if turnID == "" {
		return false, ErrAIWriteApprovalRequired
	}
	approval.ID = newAIWriteApprovalID()
	approval.Status = "pending"
	approval.CreatedAt = time.Now().UTC()
	runtime := aiWriteApprovalRuntime{turnID: turnID, decision: make(chan bool, 1)}
	aiWriteApprovalRuntimes.Lock()
	aiWriteApprovalRuntimes.items[approval.ID] = runtime
	aiWriteApprovalRuntimes.Unlock()

	turn, _, err := updateAITurn(ctx, turnID, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
		if turn.Status == models.AITurnCancelled {
			return models.AITurnEvent{}, false, context.Canceled
		}
		if turn.Status != models.AITurnRunning {
			return models.AITurnEvent{}, false, fmt.Errorf("AI 任务当前不能等待写入确认")
		}
		turn.Status = models.AITurnWaitingApproval
		turn.Approval = cloneAIWriteApproval(&approval)
		return models.AITurnEvent{Type: "approval.required", Status: turn.Status, Approval: cloneAIWriteApproval(&approval)}, true, nil
	})
	if err != nil {
		removeAIWriteApprovalRuntime(approval.ID)
		return false, err
	}
	if turn.Status != models.AITurnWaitingApproval {
		removeAIWriteApprovalRuntime(approval.ID)
		return false, ErrAIWriteApprovalNotFound
	}

	select {
	case approved := <-runtime.decision:
		if !approved {
			return false, ErrAIWriteRejected
		}
		return true, nil
	case <-ctx.Done():
		removeAIWriteApprovalRuntime(approval.ID)
		return false, ctx.Err()
	}
}

// ResolveAITurnWriteApproval 接收用户对一项具体写入的确认，并让等待中的 Harness 工具继续或返回拒绝。
func ResolveAITurnWriteApproval(ctx context.Context, turnID, approvalID string, approved bool) (models.AITurn, error) {
	approvalID = strings.TrimSpace(approvalID)
	aiWriteApprovalRuntimes.Lock()
	runtime, exists := aiWriteApprovalRuntimes.items[approvalID]
	if !exists || runtime.turnID != strings.TrimSpace(turnID) {
		aiWriteApprovalRuntimes.Unlock()
		return models.AITurn{}, ErrAIWriteApprovalNotFound
	}
	aiWriteApprovalRuntimes.Unlock()

	turn, changed, err := updateAITurn(ctx, turnID, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
		if turn.Approval == nil || turn.Approval.ID != approvalID {
			return models.AITurnEvent{}, false, ErrAIWriteApprovalNotFound
		}
		if turn.Approval.Status != "pending" {
			return models.AITurnEvent{}, false, ErrAIWriteApprovalResolved
		}
		now := time.Now().UTC()
		resolved := cloneAIWriteApproval(turn.Approval)
		if approved {
			resolved.Status = "approved"
		} else {
			resolved.Status = "rejected"
		}
		resolved.ResolvedAt = &now
		turn.Approval = resolved
		turn.Status = models.AITurnRunning
		return models.AITurnEvent{Type: "approval.resolved", Status: turn.Status, Approval: cloneAIWriteApproval(resolved)}, true, nil
	})
	if err != nil {
		return models.AITurn{}, err
	}
	if !changed {
		return turn, ErrAIWriteApprovalResolved
	}
	removeAIWriteApprovalRuntime(approvalID)
	// 带缓冲通道保证 HTTP 响应不需要等 Harness 已开始下一步。
	runtime.decision <- approved
	return turn, nil
}

// RecordAITurnToolAudit 仅保留用户可理解的工具动作与结果，供任务恢复和事后核对。
func RecordAITurnToolAudit(ctx context.Context, turnID, tool, outcome string) {
	if strings.TrimSpace(turnID) == "" {
		return
	}
	_, _, _ = updateAITurn(ctx, turnID, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
		audit := models.AIToolAudit{Tool: strings.TrimSpace(tool), Outcome: strings.TrimSpace(outcome), CreatedAt: time.Now().UTC()}
		turn.Audit = append(turn.Audit, audit)
		if len(turn.Audit) > 64 {
			turn.Audit = turn.Audit[len(turn.Audit)-64:]
		}
		return models.AITurnEvent{Type: "tool.completed", Status: turn.Status, Audit: &audit}, true, nil
	})
}

// SubscribeAITurn 先返回持久化的缺失事件，再订阅运行中任务的后续事件。
func SubscribeAITurn(ctx context.Context, id string, after int64) (models.AITurn, []models.AITurnEvent, <-chan models.AITurnEvent, func(), error) {
	turn, err := GetAITurn(ctx, id)
	if err != nil {
		return models.AITurn{}, nil, nil, nil, err
	}
	history := make([]models.AITurnEvent, 0, len(turn.Events))
	for _, event := range turn.Events {
		if event.Sequence > after {
			history = append(history, event)
		}
	}
	channel := make(chan models.AITurnEvent, aiTurnSubscriberBuffer)
	cleanup := func() { removeAITurnSubscriber(id, channel) }
	if !aiTurnActive(turn.Status) {
		close(channel)
		return turn, history, channel, cleanup, nil
	}
	aiTurnRuntimes.Lock()
	runtime := aiTurnRuntimes.items[id]
	if runtime != nil {
		runtime.subscribers[channel] = struct{}{}
	} else {
		close(channel)
	}
	aiTurnRuntimes.Unlock()
	return turn, history, channel, cleanup, nil
}

func runAITurn(ctx context.Context, id string, request models.AIChatRequest) {
	_, _, _ = updateAITurn(ctx, id, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
		if turn.Status == models.AITurnCancelled {
			return models.AITurnEvent{}, false, nil
		}
		turn.Status = models.AITurnRunning
		return models.AITurnEvent{Type: "turn.running", Status: turn.Status}, true, nil
	})
	response, err := ChatWithStudyAIStream(ctx, request, func(answer string) {
		_, _, _ = updateAITurn(ctx, id, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
			if turn.Status == models.AITurnCancelled {
				return models.AITurnEvent{}, false, nil
			}
			turn.Answer = answer
			return models.AITurnEvent{Type: "answer.updated", Status: turn.Status, Answer: answer}, true, nil
		})
	})
	if err != nil {
		_, _, _ = updateAITurn(ctx, id, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
			if turn.Status == models.AITurnCancelled {
				return models.AITurnEvent{}, false, nil
			}
			turn.Status = models.AITurnFailed
			turn.Error = aiTurnPublicError(err)
			return models.AITurnEvent{Type: "turn.failed", Status: turn.Status, Error: turn.Error}, true, nil
		})
		finishAITurnRuntime(id)
		return
	}
	_, _, _ = updateAITurn(ctx, id, func(turn *models.AITurn) (models.AITurnEvent, bool, error) {
		if turn.Status == models.AITurnCancelled {
			return models.AITurnEvent{}, false, nil
		}
		turn.Status = models.AITurnCompleted
		turn.Answer = response.Answer
		turn.Model = response.Model
		turn.Sources = append([]models.AIChatSource(nil), response.Sources...)
		turn.ContextSummary = response.ContextSummary
		turn.ContextMemory = cloneAIConversationMemory(response.ContextMemory)
		turn.HarnessSessionID = response.HarnessSessionID
		return models.AITurnEvent{Type: "turn.completed", Status: turn.Status, Answer: turn.Answer, Sources: turn.Sources}, true, nil
	})
	finishAITurnRuntime(id)
}

func updateAITurn(ctx context.Context, id string, apply func(*models.AITurn) (models.AITurnEvent, bool, error)) (models.AITurn, bool, error) {
	var result models.AITurn
	var emitted models.AITurnEvent
	changed := false
	_, err := mutateAIConfig(ctx, func(config *models.Config) error {
		for index := range config.AITurns {
			if config.AITurns[index].ID != id {
				continue
			}
			event, nextChanged, err := apply(&config.AITurns[index])
			if err != nil {
				return err
			}
			if nextChanged {
				now := time.Now().UTC()
				config.AITurns[index].UpdatedAt = now
				event.Sequence = nextAITurnEventSequence(config.AITurns[index].Events)
				event.CreatedAt = now
				config.AITurns[index].Events = append(config.AITurns[index].Events, event)
				if len(config.AITurns[index].Events) > aiTurnMaxEvents {
					config.AITurns[index].Events = config.AITurns[index].Events[len(config.AITurns[index].Events)-aiTurnMaxEvents:]
				}
				emitted = event
				changed = true
			}
			result = cloneAITurn(config.AITurns[index])
			return nil
		}
		return ErrAITurnNotFound
	})
	if err != nil {
		return models.AITurn{}, false, err
	}
	if changed {
		publishAITurnEvent(id, emitted)
	}
	return result, changed, nil
}

func aiTurnActive(status models.AITurnStatus) bool {
	return status == models.AITurnQueued || status == models.AITurnRunning || status == models.AITurnWaitingApproval
}

func nextAITurnEventSequence(events []models.AITurnEvent) int64 {
	if len(events) == 0 {
		return 1
	}
	return events[len(events)-1].Sequence + 1
}

func newAITurnID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "turn-" + hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("turn-%d", time.Now().UnixNano())
}

func newAIWriteApprovalID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "approval-" + hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("approval-%d", time.Now().UnixNano())
}

func aiTurnPublicError(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "任务已取消"
	case errors.Is(err, ErrDeepSeekNotConfigured), errors.Is(err, ErrHarnessRuntimeUnavailable), errors.Is(err, ErrAIInvalidScope):
		return err.Error()
	default:
		return "AI 任务未能完成，请重试"
	}
}

func cloneAITurn(turn models.AITurn) models.AITurn {
	clone := turn
	clone.Sources = append([]models.AIChatSource(nil), turn.Sources...)
	clone.Audit = append([]models.AIToolAudit(nil), turn.Audit...)
	clone.ContextMemory = cloneAIConversationMemory(turn.ContextMemory)
	clone.Events = make([]models.AITurnEvent, len(turn.Events))
	for index, event := range turn.Events {
		clone.Events[index] = event
		clone.Events[index].Approval = cloneAIWriteApproval(event.Approval)
		if event.Audit != nil {
			audit := *event.Audit
			clone.Events[index].Audit = &audit
		}
	}
	clone.Approval = cloneAIWriteApproval(turn.Approval)
	return clone
}

func cloneAIWriteApproval(approval *models.AIWriteApproval) *models.AIWriteApproval {
	if approval == nil {
		return nil
	}
	clone := *approval
	if approval.ResolvedAt != nil {
		resolvedAt := *approval.ResolvedAt
		clone.ResolvedAt = &resolvedAt
	}
	return &clone
}

func removeAIWriteApprovalRuntime(approvalID string) {
	aiWriteApprovalRuntimes.Lock()
	delete(aiWriteApprovalRuntimes.items, approvalID)
	aiWriteApprovalRuntimes.Unlock()
}

func publishAITurnEvent(id string, event models.AITurnEvent) {
	aiTurnRuntimes.Lock()
	defer aiTurnRuntimes.Unlock()
	runtime := aiTurnRuntimes.items[id]
	if runtime == nil {
		return
	}
	for subscriber := range runtime.subscribers {
		select {
		case subscriber <- event:
		default:
			// 订阅者可借助持久化事件补齐；不让慢浏览器阻塞模型执行。
		}
	}
}

func removeAITurnSubscriber(id string, subscriber chan models.AITurnEvent) {
	aiTurnRuntimes.Lock()
	defer aiTurnRuntimes.Unlock()
	if runtime := aiTurnRuntimes.items[id]; runtime != nil {
		if _, exists := runtime.subscribers[subscriber]; exists {
			delete(runtime.subscribers, subscriber)
			close(subscriber)
		}
	}
}

func cancelAITurnRuntime(id string) {
	aiTurnRuntimes.Lock()
	defer aiTurnRuntimes.Unlock()
	if runtime := aiTurnRuntimes.items[id]; runtime != nil && runtime.cancel != nil {
		runtime.cancel()
	}
}

func finishAITurnRuntime(id string) {
	aiTurnRuntimes.Lock()
	runtime := aiTurnRuntimes.items[id]
	delete(aiTurnRuntimes.items, id)
	aiTurnRuntimes.Unlock()
	if runtime == nil {
		return
	}
	for subscriber := range runtime.subscribers {
		close(subscriber)
	}
}
