package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	models "study-tracker-go/internal/model"
)

const (
	harnessToolGrantLifetime = 35 * time.Minute
	harnessMaxPathResults    = 80
	harnessMaxSearchResults  = 12
	harnessMaxReadTokens     = 80_000
)

var (
	ErrHarnessToolUnauthorized = errors.New("Harness 工具会话未授权或已过期")
	ErrHarnessToolUnavailable  = errors.New("Harness 工具不可用")
)

type harnessToolGrant struct {
	userID  int64
	scope   aiLibraryScope
	expires time.Time
	sources map[int64]models.AIChatSource
}

var harnessToolGrants = struct {
	sync.Mutex
	items map[string]*harnessToolGrant
}{items: make(map[string]*harnessToolGrant)}

// NewHarnessToolGrant creates a short-lived capability for one Harness child
// process. The value only grants access to the current user's current library
// scope, and is never returned to the browser or written to disk.
func NewHarnessToolGrant(ctx context.Context, request models.AIChatRequest) (string, error) {
	scope, err := newAILibraryScope(request)
	if err != nil {
		return "", err
	}
	app, err := appFor(ctx)
	if err != nil {
		return "", err
	}
	userID, hasUserID := UserIDFromContext(ctx)
	if app.AuthEnabled() && (!hasUserID || userID <= 0) {
		return "", ErrHarnessToolUnauthorized
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("生成 Harness 会话授权失败：%w", err)
	}
	token := hex.EncodeToString(raw)
	harnessToolGrants.Lock()
	for existing, grant := range harnessToolGrants.items {
		if time.Now().After(grant.expires) {
			delete(harnessToolGrants.items, existing)
		}
	}
	harnessToolGrants.items[token] = &harnessToolGrant{userID: userID, scope: scope, expires: time.Now().Add(harnessToolGrantLifetime), sources: make(map[int64]models.AIChatSource)}
	harnessToolGrants.Unlock()
	return token, nil
}

func RevokeHarnessToolGrant(token string) {
	harnessToolGrants.Lock()
	delete(harnessToolGrants.items, strings.TrimSpace(token))
	harnessToolGrants.Unlock()
}

// HarnessToolUserID resolves a bearer capability for the HTTP middleware. The
// middleware uses the result to restore the user-bound PostgreSQL repository.
func HarnessToolUserID(token string) (int64, error) {
	grant, err := harnessToolGrantFor(token)
	if err != nil {
		return 0, err
	}
	return grant.userID, nil
}

func harnessToolGrantFor(token string) (*harnessToolGrant, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrHarnessToolUnauthorized
	}
	harnessToolGrants.Lock()
	defer harnessToolGrants.Unlock()
	grant, exists := harnessToolGrants.items[token]
	if !exists || time.Now().After(grant.expires) {
		delete(harnessToolGrants.items, token)
		return nil, ErrHarnessToolUnauthorized
	}
	return grant, nil
}

// ConsumeHarnessSources returns the notes whose content or excerpt was sent
// through this child process's tools. They are shown in the normal citation UI
// without exposing anything outside the conversation scope.
func ConsumeHarnessSources(token string) []models.AIChatSource {
	harnessToolGrants.Lock()
	defer harnessToolGrants.Unlock()
	grant, exists := harnessToolGrants.items[strings.TrimSpace(token)]
	if !exists || time.Now().After(grant.expires) || len(grant.sources) == 0 {
		return nil
	}
	result := make([]models.AIChatSource, 0, len(grant.sources))
	for _, source := range grant.sources {
		result = append(result, source)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result
}

// ExecuteHarnessTool is the narrow Go-side implementation behind the custom
// Harness plugin. It has no generic filesystem: it can only create a new note
// or save an exact current version inside the conversation's granted scope.
func ExecuteHarnessTool(ctx context.Context, token, tool string, args map[string]any) (any, error) {
	grant, err := harnessToolGrantFor(token)
	if err != nil {
		return nil, err
	}
	switch strings.TrimSpace(tool) {
	case "list_paths":
		return harnessListPaths(ctx, grant.scope, args)
	case "search":
		return harnessSearch(ctx, token, grant.scope, args)
	case "read_note":
		return harnessReadNote(ctx, token, grant.scope, args)
	case "create_note":
		return harnessCreateNote(ctx, grant.scope, args)
	case "update_note":
		return harnessUpdateNote(ctx, grant.scope, args)
	default:
		return nil, ErrHarnessToolUnavailable
	}
}

func harnessScopedItems(ctx context.Context, scope aiLibraryScope) ([]models.LibraryItem, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	items, err := collectAILibraryItems(ctx, repos.Library)
	if err != nil {
		return nil, err
	}
	if !scope.active() {
		return items, nil
	}
	return filterAILibraryScope(items, scope)
}

func harnessListPaths(ctx context.Context, scope aiLibraryScope, args map[string]any) (map[string]any, error) {
	items, err := harnessScopedItems(ctx, scope)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(harnessStringArg(args, "query"))
	limit := harnessBoundedInt(args, "limit", harnessMaxPathResults, 1, harnessMaxPathResults)
	type row struct {
		ID    int64
		Path  string
		Kind  string
		Tags  []string
		Score int
	}
	rows := make([]row, 0, len(items))
	for _, item := range items {
		if item.Kind != "folder" && !aiReadableLibraryItem(item) {
			continue
		}
		itemPath := harnessLibraryPath(items, item)
		score := aiScore(query, itemPath+" "+strings.Join(item.Tags, " "))
		if query != "" && score == 0 {
			continue
		}
		rows = append(rows, row{ID: item.ID, Path: itemPath, Kind: item.Kind, Tags: append([]string(nil), item.Tags...), Score: score})
	}
	sort.SliceStable(rows, func(left, right int) bool {
		if rows[left].Score != rows[right].Score {
			return rows[left].Score > rows[right].Score
		}
		return strings.Compare(rows[left].Path, rows[right].Path) < 0
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	paths := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		paths = append(paths, map[string]any{"id": item.ID, "path": item.Path, "kind": item.Kind, "tags": item.Tags})
	}
	return map[string]any{"paths": paths, "scope_limited": scope.active()}, nil
}

func harnessSearch(ctx context.Context, token string, scope aiLibraryScope, args map[string]any) (map[string]any, error) {
	query := strings.TrimSpace(harnessStringArg(args, "query"))
	if query == "" {
		return nil, fmt.Errorf("检索关键词不能为空")
	}
	items, err := harnessScopedItems(ctx, scope)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		item  models.LibraryItem
		path  string
		score int
	}
	candidates := make([]candidate, 0, len(items))
	for _, item := range items {
		if item.Kind == "folder" || !aiReadableLibraryItem(item) {
			continue
		}
		score := aiScore(query, item.Name+" "+strings.Join(item.Tags, " "))
		if score == 0 {
			continue
		}
		candidates = append(candidates, candidate{item: item, path: harnessLibraryPath(items, item), score: score})
	}
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].score != candidates[right].score {
			return candidates[left].score > candidates[right].score
		}
		return candidates[left].item.UpdatedAt.After(candidates[right].item.UpdatedAt)
	})
	limit := harnessBoundedInt(args, "limit", harnessMaxSearchResults, 1, harnessMaxSearchResults)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	results := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		content, _, err := ReadLibraryContent(ctx, candidate.item.ID)
		if err != nil {
			continue
		}
		excerpt := aiBoundedText(strings.Join(strings.Fields(string(content)), " "), 800)
		results = append(results, map[string]any{
			"id": candidate.item.ID, "path": candidate.path, "name": candidate.item.Name,
			"tags": candidate.item.Tags, "excerpt": excerpt,
		})
		harnessRecordSource(token, candidate.item)
	}
	return map[string]any{"results": results, "scope_limited": scope.active()}, nil
}

func harnessReadNote(ctx context.Context, token string, scope aiLibraryScope, args map[string]any) (map[string]any, error) {
	id := int64(harnessBoundedInt(args, "item_id", 0, 1, int(^uint(0)>>1)))
	if id <= 0 {
		return nil, fmt.Errorf("笔记标识无效")
	}
	items, err := harnessScopedItems(ctx, scope)
	if err != nil {
		return nil, err
	}
	var target *models.LibraryItem
	for index := range items {
		if items[index].ID == id {
			target = &items[index]
			break
		}
	}
	if target == nil || target.Kind == "folder" || !aiReadableLibraryItem(*target) {
		return nil, fmt.Errorf("笔记不在当前资料范围内或不可读取")
	}
	content, _, err := ReadLibraryContent(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	harnessRecordSource(token, *target)
	return map[string]any{
		"id": target.ID, "path": harnessLibraryPath(items, *target), "name": target.Name,
		"current_version": target.CurrentVersion,
		"content":         aiBoundedTokens(string(content), harnessMaxReadTokens),
	}, nil
}

func harnessRecordSource(token string, item models.LibraryItem) {
	harnessToolGrants.Lock()
	defer harnessToolGrants.Unlock()
	grant, exists := harnessToolGrants.items[strings.TrimSpace(token)]
	if !exists || time.Now().After(grant.expires) {
		return
	}
	grant.sources[item.ID] = models.AIChatSource{SourceType: "library", ID: item.ID, Title: item.Name}
}

func harnessCreateNote(ctx context.Context, scope aiLibraryScope, args map[string]any) (map[string]any, error) {
	targetPath := strings.TrimSpace(harnessStringArg(args, "path"))
	content := harnessStringArg(args, "content")
	if targetPath == "" {
		return nil, fmt.Errorf("请先明确笔记路径和文件名")
	}
	if len([]byte(content)) > aiMaxEditableNoteBytes {
		return nil, fmt.Errorf("拟写入的笔记超过 10MB，未创建")
	}
	target, err := resolveAINoteWriteTarget(ctx, targetPath, scope.folderID)
	if err != nil {
		return nil, err
	}
	items, err := harnessScopedItems(ctx, scope)
	if err != nil {
		return nil, err
	}
	if !harnessWriteTargetAllowed(items, scope, target) {
		return nil, fmt.Errorf("目标路径不在当前对话的资料范围内")
	}
	if target.item != nil {
		return nil, fmt.Errorf("目标“%s”已存在；请先读取该笔记，再使用更新工具保存新版本", target.targetPath)
	}
	item, err := ApplyAINoteWrite(ctx, models.AINoteWriteApplyRequest{
		Action: "create", ParentID: target.parentID, Name: target.name, Content: content,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"written": true, "action": "created", "item_id": item.ID,
		"target_path": target.targetPath, "current_version": item.CurrentVersion,
	}, nil
}

func harnessUpdateNote(ctx context.Context, scope aiLibraryScope, args map[string]any) (map[string]any, error) {
	itemID := int64(harnessBoundedInt(args, "item_id", 0, 1, int(^uint(0)>>1)))
	baseVersion, validVersion := harnessRequiredPositiveInt(args, "base_version")
	content := harnessStringArg(args, "content")
	if itemID <= 0 || !validVersion {
		return nil, fmt.Errorf("更新笔记必须提供读取结果中的 item_id 和 current_version")
	}
	if len([]byte(content)) > aiMaxEditableNoteBytes {
		return nil, fmt.Errorf("拟写入的笔记超过 10MB，未保存")
	}
	items, err := harnessScopedItems(ctx, scope)
	if err != nil {
		return nil, err
	}
	var target *models.LibraryItem
	for index := range items {
		if items[index].ID == itemID {
			target = &items[index]
			break
		}
	}
	if target == nil || target.Kind != "note" || !aiEditableLibraryItem(*target) {
		return nil, fmt.Errorf("笔记不在当前资料范围内或不可编辑")
	}
	item, err := ApplyAINoteWrite(ctx, models.AINoteWriteApplyRequest{
		Action: "update", ItemID: itemID, Content: content, BaseVersion: baseVersion,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"written": true, "action": "updated", "item_id": item.ID,
		"target_path": harnessLibraryPath(items, item), "current_version": item.CurrentVersion,
	}, nil
}

func harnessWriteTargetAllowed(items []models.LibraryItem, scope aiLibraryScope, target aiNoteWriteTarget) bool {
	if !scope.active() {
		return true
	}
	if target.item != nil {
		for _, item := range items {
			if item.ID == target.item.ID {
				return true
			}
		}
		return false
	}
	if target.parentID == nil {
		return false
	}
	for _, item := range items {
		if item.ID == *target.parentID && item.Kind == "folder" {
			return true
		}
	}
	return false
}

func harnessLibraryPath(items []models.LibraryItem, item models.LibraryItem) string {
	parentPath := aiLibraryParentPath(items, item.ParentID)
	if parentPath == "" {
		return item.Name
	}
	return parentPath + " / " + item.Name
}

func harnessStringArg(args map[string]any, name string) string {
	value, _ := args[name].(string)
	return value
}

func harnessBoundedInt(args map[string]any, name string, fallback, minimum, maximum int) int {
	value, exists := args[name]
	if !exists {
		return fallback
	}
	var number int
	switch current := value.(type) {
	case float64:
		number = int(current)
	case int:
		number = current
	case int64:
		number = int(current)
	default:
		return fallback
	}
	if number < minimum {
		return minimum
	}
	if number > maximum {
		return maximum
	}
	return number
}

func harnessRequiredPositiveInt(args map[string]any, name string) (int, bool) {
	value, exists := args[name]
	if !exists {
		return 0, false
	}
	var number int
	switch current := value.(type) {
	case float64:
		number = int(current)
		if current != float64(number) {
			return 0, false
		}
	case int:
		number = current
	case int64:
		number = int(current)
		if int64(number) != current {
			return 0, false
		}
	default:
		return 0, false
	}
	return number, number > 0
}
