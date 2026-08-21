package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

const aiMaxHarnessNoteBytes = 10 << 20

var (
	ErrAINoteWriteTarget   = errors.New("AI 无法解析或写入这个资料库路径")
	ErrAINoteWriteConflict = errors.New("目标笔记已变化或已存在，请重新读取后再保存")
)

type aiNoteWriteTarget struct {
	item       *models.LibraryItem
	parentID   *int64
	name       string
	targetPath string
}

// createHarnessLibraryNote 执行受限 Harness 工具的创建部分；它刻意不提供 HTTP 处理器，也不触发模型调用。
func createHarnessLibraryNote(ctx context.Context, parentID *int64, name, content string) (models.LibraryItem, error) {
	if len([]byte(content)) > aiMaxHarnessNoteBytes {
		return models.LibraryItem{}, fmt.Errorf("笔记内容不能超过 10MB")
	}
	name, err := aiNormalizeNoteName(name)
	if err != nil {
		return models.LibraryItem{}, err
	}
	if err := aiValidateWriteParent(ctx, parentID); err != nil {
		return models.LibraryItem{}, err
	}
	items, err := ListLibrary(ctx, repository.LibraryFilter{All: true})
	if err != nil {
		return models.LibraryItem{}, err
	}
	if existing := aiFindLibraryChild(items, parentID, name); existing != nil {
		return models.LibraryItem{}, fmt.Errorf("%w：同一路径已经有“%s”", ErrAINoteWriteConflict, existing.Name)
	}
	return CreateLibraryItem(ctx, models.CreateLibraryItemRequest{
		ParentID: parentID,
		Kind:     "note",
		Name:     name,
		MimeType: "text/markdown; charset=utf-8",
	}, []byte(content))
}

// updateHarnessLibraryNote 在 Harness 工具读取到笔记的确切当前版本后保存完整版本。SaveLibraryContent 会拒绝过期版本，因此 Agent 无法强制覆盖更新后的用户修改。
func updateHarnessLibraryNote(ctx context.Context, itemID int64, content string, baseVersion int) (models.LibraryItem, error) {
	if itemID <= 0 || baseVersion <= 0 {
		return models.LibraryItem{}, ErrAINoteWriteConflict
	}
	if len([]byte(content)) > aiMaxHarnessNoteBytes {
		return models.LibraryItem{}, fmt.Errorf("笔记内容不能超过 10MB")
	}
	item, err := GetLibraryItem(ctx, itemID)
	if err != nil {
		return models.LibraryItem{}, err
	}
	if !aiEditableLibraryItem(item) {
		return models.LibraryItem{}, fmt.Errorf("%w：目标不是可编辑笔记", ErrAINoteWriteTarget)
	}
	return SaveLibraryContent(ctx, item.ID, models.SaveLibraryContentRequest{
		Content:     content,
		BaseVersion: baseVersion,
		Checkpoint:  true,
		Force:       false,
	})
}

// aiEditableLibraryItem 在业务层中执行当前流程或局部处理。
func aiEditableLibraryItem(item models.LibraryItem) bool {
	if item.Kind != "note" {
		return false
	}
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(item.MimeType, ";")[0]))
	return mimeType == "" || strings.HasPrefix(mimeType, "text/") || strings.HasSuffix(strings.ToLower(item.Name), ".md") || strings.HasSuffix(strings.ToLower(item.Name), ".txt")
}

// resolveAINoteWriteTarget 在业务层中执行当前流程或局部处理。
func resolveAINoteWriteTarget(ctx context.Context, rawPath string, scopeFolderID *int64) (aiNoteWriteTarget, error) {
	parts, err := aiSplitLibraryPath(rawPath)
	if err != nil {
		return aiNoteWriteTarget{}, err
	}
	items, err := ListLibrary(ctx, repository.LibraryFilter{All: true})
	if err != nil {
		return aiNoteWriteTarget{}, err
	}
	if err := aiValidateWriteParent(ctx, scopeFolderID); err != nil {
		return aiNoteWriteTarget{}, err
	}
	name, err := aiNormalizeNoteName(parts[len(parts)-1])
	if err != nil {
		return aiNoteWriteTarget{}, err
	}
	parentID := aiCopyID(scopeFolderID)
	if len(parts) > 1 {
		parentID, err = aiResolveFolderPath(items, nil, parts[:len(parts)-1])
		if err != nil && scopeFolderID != nil {
			parentID, err = aiResolveFolderPath(items, scopeFolderID, parts[:len(parts)-1])
		}
		if err != nil {
			return aiNoteWriteTarget{}, err
		}
	}
	parentPath := aiLibraryParentPath(items, parentID)
	targetPath := name
	if parentPath != "" {
		targetPath = parentPath + " / " + name
	}
	if item := aiFindLibraryChild(items, parentID, name); item != nil {
		return aiNoteWriteTarget{item: item, parentID: aiCopyID(item.ParentID), name: item.Name, targetPath: targetPath}, nil
	}
	return aiNoteWriteTarget{parentID: parentID, name: name, targetPath: targetPath}, nil
}

// aiSplitLibraryPath 在业务层中执行当前流程或局部处理。
func aiSplitLibraryPath(rawPath string) ([]string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "资料库/")
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" {
		return nil, ErrAINoteWriteTarget
	}
	parts := make([]string, 0, strings.Count(normalized, "/")+1)
	for _, part := range strings.Split(normalized, "/") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("%w：路径不能包含空目录或上级目录", ErrAINoteWriteTarget)
		}
		if strings.ContainsAny(part, "\x00\r\n") || utf8.RuneCountInString(part) > 160 {
			return nil, fmt.Errorf("%w：路径名称无效", ErrAINoteWriteTarget)
		}
		parts = append(parts, part)
	}
	return parts, nil
}

// aiNormalizeNoteName 在业务层中执行当前流程或局部处理。
func aiNormalizeNoteName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || strings.ContainsAny(name, "/\\\x00\r\n") || utf8.RuneCountInString(name) > 160 || name == "." || name == ".." {
		return "", fmt.Errorf("%w：文件名无效", ErrAINoteWriteTarget)
	}
	if path.Ext(name) == "" {
		name += ".md"
	}
	if extension := strings.ToLower(path.Ext(name)); extension != ".md" && extension != ".txt" {
		return "", fmt.Errorf("%w：只能创建 Markdown 或纯文本笔记", ErrAINoteWriteTarget)
	}
	return name, nil
}

// aiValidateWriteParent 在业务层中执行当前流程或局部处理。
func aiValidateWriteParent(ctx context.Context, parentID *int64) error {
	if parentID == nil {
		return nil
	}
	if *parentID <= 0 {
		return fmt.Errorf("%w：资料路径无效", ErrAINoteWriteTarget)
	}
	item, err := GetLibraryItem(ctx, *parentID)
	if err != nil {
		return fmt.Errorf("%w：资料路径不存在", ErrAINoteWriteTarget)
	}
	if item.Kind != "folder" {
		return fmt.Errorf("%w：目标父级不是文件夹", ErrAINoteWriteTarget)
	}
	return nil
}

// aiResolveFolderPath 在业务层中执行当前流程或局部处理。
func aiResolveFolderPath(items []models.LibraryItem, parentID *int64, folders []string) (*int64, error) {
	current := aiCopyID(parentID)
	for _, segment := range folders {
		matches := make([]*models.LibraryItem, 0, 1)
		for index := range items {
			item := &items[index]
			if item.Kind == "folder" && aiSameParent(item.ParentID, current) && strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(segment)) {
				matches = append(matches, item)
			}
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("%w：找不到唯一的文件夹“%s”", ErrAINoteWriteTarget, segment)
		}
		current = aiCopyID(&matches[0].ID)
	}
	return current, nil
}

// aiFindLibraryChild 在业务层中执行当前流程或局部处理。
func aiFindLibraryChild(items []models.LibraryItem, parentID *int64, name string) *models.LibraryItem {
	for index := range items {
		item := &items[index]
		if !aiSameParent(item.ParentID, parentID) || !strings.EqualFold(item.Name, name) {
			continue
		}
		return item
	}
	return nil
}

// aiLibraryParentPath 在业务层中执行当前流程或局部处理。
func aiLibraryParentPath(items []models.LibraryItem, parentID *int64) string {
	if parentID == nil {
		return ""
	}
	byID := make(map[int64]models.LibraryItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	parts := make([]string, 0, 4)
	seen := map[int64]struct{}{}
	current := parentID
	for current != nil {
		if _, loop := seen[*current]; loop {
			return ""
		}
		seen[*current] = struct{}{}
		item, exists := byID[*current]
		if !exists {
			return ""
		}
		parts = append([]string{item.Name}, parts...)
		current = item.ParentID
	}
	return strings.Join(parts, " / ")
}

// aiSameParent 在业务层中执行当前流程或局部处理。
func aiSameParent(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// aiCopyID 在业务层中执行当前流程或局部处理。
func aiCopyID(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// aiHistoryWithinTokenBudget 在业务层中执行当前流程或局部处理。
func aiHistoryWithinTokenBudget(history []models.AIChatMessage, budget int) []models.AIChatMessage {
	if budget <= 0 || len(history) == 0 {
		return nil
	}
	start := len(history)
	used := 0
	for index := len(history) - 1; index >= 0; index-- {
		cost := aiApproxTokens(history[index].Content) + 8
		if used+cost > budget && start < len(history) {
			break
		}
		used += cost
		start = index
	}
	return history[start:]
}
