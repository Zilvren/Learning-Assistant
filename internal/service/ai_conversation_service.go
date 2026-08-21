package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

import models "study-tracker-go/internal/model"

const (
	aiConversationMaxActive   = 24
	aiConversationMaxArchived = 100
	aiConversationMaxMessages = 160
	aiConversationMaxRunes    = 16_000
	aiConversationMaxScope    = 240
	aiConversationMaxModel    = 120
	aiConversationMaxSources  = 8
	aiConversationMaxTitle    = 240
	aiConversationMaxExcerpt  = 800
	aiConversationMaxID       = 80
	aiConversationMaxName     = 96
	aiConversationMaxItems    = 60
)

var (
	ErrInvalidAIConversation       = errors.New("AI 对话上下文无效")
	ErrAIConversationNotFound      = errors.New("AI 对话不存在")
	ErrAIConversationActiveLimit   = errors.New("活跃对话已达上限，请先归档一条对话")
	ErrAIConversationArchivedLimit = errors.New("归档对话已达上限，请先永久删除一条归档对话")
)

// GetAIConversation 读取当前用户保存的全部对话。每个对话独占其消息记录和资料范围，因此在界面切换对话后不会使用其他对话的资料上下文。
func GetAIConversation(ctx context.Context) ([]models.AIConversation, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if len(config.AIConversations) == 0 && len(config.AIChatContext) > 0 {
		// 旧版本只保存一份消息记录；将其保留为首个独立对话，避免静默丢弃历史内容。
		legacy := models.AIConversation{ID: "legacy-import", Messages: config.AIChatContext}
		legacy.Title = aiConversationTitle(legacy.Messages)
		config.AIConversations = []models.AIConversation{legacy}
		config.AIChatContext = nil
		if err := saveConfig(ctx, config); err != nil {
			return nil, err
		}
	}
	return cloneAIConversations(config.AIConversations), nil
}

// SaveAIConversation 替换当前用户的独立对话集合，并分别校验活跃和归档对话的数量上限。
func SaveAIConversation(ctx context.Context, conversations []models.AIConversation) ([]models.AIConversation, error) {
	normalized, err := normalizeAIConversations(conversations)
	if err != nil {
		return nil, err
	}
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	config.AIConversations = normalized
	config.AIChatContext = nil
	if err := saveConfig(ctx, config); err != nil {
		return nil, err
	}
	return cloneAIConversations(normalized), nil
}

// ClearAIConversation 在业务层中执行当前流程或局部处理。
func ClearAIConversation(ctx context.Context) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.AIChatContext = nil
	config.AIConversations = nil
	return saveConfig(ctx, config)
}

// ArchiveAIConversation 将指定活跃对话移入归档，并由服务端记录归档时间。
func ArchiveAIConversation(ctx context.Context, id string) ([]models.AIConversation, error) {
	conversations, err := GetAIConversation(ctx)
	if err != nil {
		return nil, err
	}
	index, err := aiConversationIndex(conversations, id)
	if err != nil {
		return nil, err
	}
	if conversations[index].ArchivedAt != nil {
		return conversations, nil
	}
	if countArchivedAIConversations(conversations) >= aiConversationMaxArchived {
		return nil, ErrAIConversationArchivedLimit
	}
	now := time.Now().UTC()
	conversations[index].ArchivedAt = &now
	return SaveAIConversation(ctx, conversations)
}

// RestoreAIConversation 将指定归档对话恢复为活跃对话，并阻止超过活跃对话上限的恢复操作。
func RestoreAIConversation(ctx context.Context, id string) ([]models.AIConversation, error) {
	conversations, err := GetAIConversation(ctx)
	if err != nil {
		return nil, err
	}
	index, err := aiConversationIndex(conversations, id)
	if err != nil {
		return nil, err
	}
	if conversations[index].ArchivedAt == nil {
		return conversations, nil
	}
	if countActiveAIConversations(conversations) >= aiConversationMaxActive {
		return nil, ErrAIConversationActiveLimit
	}
	conversations[index].ArchivedAt = nil
	return SaveAIConversation(ctx, conversations)
}

// DeleteArchivedAIConversation 永久删除指定归档对话，避免活跃对话被删除接口误删。
func DeleteArchivedAIConversation(ctx context.Context, id string) ([]models.AIConversation, error) {
	conversations, err := GetAIConversation(ctx)
	if err != nil {
		return nil, err
	}
	index, err := aiConversationIndex(conversations, id)
	if err != nil {
		return nil, err
	}
	if conversations[index].ArchivedAt == nil {
		return nil, fmt.Errorf("%w：请先归档后再永久删除", ErrInvalidAIConversation)
	}
	conversations = append(conversations[:index:index], conversations[index+1:]...)
	return SaveAIConversation(ctx, conversations)
}

// aiConversationIndex 根据会话标识定位一条已保存的对话，并统一处理非法或缺失标识。
func aiConversationIndex(conversations []models.AIConversation, id string) (int, error) {
	id = strings.TrimSpace(id)
	if !validAIConversationID(id) {
		return -1, ErrAIConversationNotFound
	}
	for index, conversation := range conversations {
		if conversation.ID == id {
			return index, nil
		}
	}
	return -1, ErrAIConversationNotFound
}

// countActiveAIConversations 返回尚未归档的对话数量。
func countActiveAIConversations(conversations []models.AIConversation) int {
	count := 0
	for _, conversation := range conversations {
		if conversation.ArchivedAt == nil {
			count++
		}
	}
	return count
}

// countArchivedAIConversations 返回已归档的对话数量。
func countArchivedAIConversations(conversations []models.AIConversation) int {
	return len(conversations) - countActiveAIConversations(conversations)
}

// normalizeAIConversations 在业务层中执行当前流程或局部处理。
func normalizeAIConversations(conversations []models.AIConversation) ([]models.AIConversation, error) {
	result := make([]models.AIConversation, 0, len(conversations))
	seenIDs := make(map[string]struct{}, len(conversations))
	activeCount := 0
	archivedCount := 0
	for _, conversation := range conversations {
		id := strings.TrimSpace(conversation.ID)
		if !validAIConversationID(id) {
			return nil, fmt.Errorf("%w：会话标识无效", ErrInvalidAIConversation)
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("%w：会话标识不能重复", ErrInvalidAIConversation)
		}
		seenIDs[id] = struct{}{}

		messages, err := normalizeAIConversation(conversation.Messages)
		if err != nil {
			return nil, err
		}
		folderID := conversation.FolderID
		if folderID != nil && *folderID <= 0 {
			folderID = nil
		}
		archivedAt := normalizeAIConversationArchivedAt(conversation.ArchivedAt)
		if archivedAt == nil {
			activeCount++
			if activeCount > aiConversationMaxActive {
				return nil, ErrAIConversationActiveLimit
			}
		} else {
			archivedCount++
			if archivedCount > aiConversationMaxArchived {
				return nil, ErrAIConversationArchivedLimit
			}
		}
		result = append(result, models.AIConversation{
			ID:               id,
			Title:            aiConversationTitleWithFallback(conversation.Title, messages),
			FolderID:         folderID,
			ItemIDs:          normalizeAIConversationItemIDs(conversation.ItemIDs),
			Messages:         messages,
			HarnessSessionID: normalizeAIHarnessSessionID(conversation.HarnessSessionID),
			ArchivedAt:       archivedAt,
		})
	}
	return result, nil
}

// normalizeAIConversationArchivedAt 复制归档时间并统一为 UTC，零值按未归档处理。
func normalizeAIConversationArchivedAt(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

// normalizeAIHarnessSessionID 在业务层中执行当前流程或局部处理。
func normalizeAIHarnessSessionID(value string) string {
	value = strings.TrimSpace(value)
	if !validAIConversationID(value) {
		return ""
	}
	return value
}

// validAIConversationID 在业务层中执行当前流程或局部处理。
func validAIConversationID(value string) bool {
	if value == "" || len([]rune(value)) > aiConversationMaxID {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

// aiConversationTitleWithFallback 在业务层中执行当前流程或局部处理。
func aiConversationTitleWithFallback(value string, messages []models.AIConversationMessage) string {
	if title := boundedAIConversationText(value, aiConversationMaxName); title != "" {
		return title
	}
	return aiConversationTitle(messages)
}

// aiConversationTitle 在业务层中执行当前流程或局部处理。
func aiConversationTitle(messages []models.AIConversationMessage) string {
	for _, message := range messages {
		if message.Role == "user" {
			if title := boundedAIConversationText(strings.Join(strings.Fields(message.Content), " "), aiConversationMaxName); title != "" {
				return title
			}
		}
	}
	return "新对话"
}

// normalizeAIConversationItemIDs 在业务层中执行当前流程或局部处理。
func normalizeAIConversationItemIDs(values []int64) []int64 {
	result := make([]int64, 0, min(len(values), aiConversationMaxItems))
	seen := make(map[int64]struct{}, len(values))
	for _, id := range values {
		if id <= 0 || len(result) == aiConversationMaxItems {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// cloneAIConversations 在业务层中执行当前流程或局部处理。
func cloneAIConversations(conversations []models.AIConversation) []models.AIConversation {
	result := make([]models.AIConversation, 0, len(conversations))
	for _, conversation := range conversations {
		result = append(result, models.AIConversation{
			ID:               conversation.ID,
			Title:            conversation.Title,
			FolderID:         conversation.FolderID,
			ItemIDs:          append([]int64(nil), conversation.ItemIDs...),
			Messages:         append([]models.AIConversationMessage(nil), conversation.Messages...),
			HarnessSessionID: conversation.HarnessSessionID,
			ArchivedAt:       normalizeAIConversationArchivedAt(conversation.ArchivedAt),
		})
	}
	return result
}

// normalizeAIConversation 在业务层中执行当前流程或局部处理。
func normalizeAIConversation(messages []models.AIConversationMessage) ([]models.AIConversationMessage, error) {
	if len(messages) > aiConversationMaxMessages {
		messages = messages[len(messages)-aiConversationMaxMessages:]
	}
	result := make([]models.AIConversationMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role != "user" && role != "assistant" {
			return nil, fmt.Errorf("%w：消息角色必须是 user 或 assistant", ErrInvalidAIConversation)
		}
		content := boundedAIConversationText(message.Content, aiConversationMaxRunes)
		if content == "" {
			return nil, fmt.Errorf("%w：消息内容不能为空", ErrInvalidAIConversation)
		}
		normalized := models.AIConversationMessage{Role: role, Content: content}
		if role == "user" {
			normalized.Scope = boundedAIConversationText(message.Scope, aiConversationMaxScope)
		} else {
			normalized.Model = boundedAIConversationText(message.Model, aiConversationMaxModel)
			normalized.Sources = normalizeAIConversationSources(message.Sources)
			normalized.Incomplete = message.Incomplete
		}
		result = append(result, normalized)
	}
	return result, nil
}

// normalizeAIConversationSources 在业务层中执行当前流程或局部处理。
func normalizeAIConversationSources(sources []models.AIChatSource) []models.AIChatSource {
	result := make([]models.AIChatSource, 0, min(len(sources), aiConversationMaxSources))
	for _, source := range sources {
		if len(result) == aiConversationMaxSources || (source.SourceType != "library" && source.SourceType != "error") || source.ID <= 0 {
			continue
		}
		result = append(result, models.AIChatSource{
			SourceType: source.SourceType,
			ID:         source.ID,
			Title:      boundedAIConversationText(source.Title, aiConversationMaxTitle),
			Excerpt:    boundedAIConversationText(source.Excerpt, aiConversationMaxExcerpt),
		})
	}
	return result
}

// boundedAIConversationText 在业务层中执行当前流程或局部处理。
func boundedAIConversationText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
