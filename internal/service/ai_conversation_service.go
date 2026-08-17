package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

import models "study-tracker-go/internal/model"

const (
	aiConversationMaxConversations = 24
	aiConversationMaxMessages      = 160
	aiConversationMaxRunes         = 16_000
	aiConversationMaxScope         = 240
	aiConversationMaxModel         = 120
	aiConversationMaxSources       = 8
	aiConversationMaxTitle         = 240
	aiConversationMaxExcerpt       = 800
	aiConversationMaxID            = 80
	aiConversationMaxName          = 96
	aiConversationMaxItems         = 60
)

var ErrInvalidAIConversation = errors.New("AI 对话上下文无效")

// GetAIConversation loads every saved chat for the current user. A chat owns
// both its transcript and scope, so one conversation can never use another
// conversation's library context after the user switches in the UI.
func GetAIConversation(ctx context.Context) ([]models.AIConversation, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if len(config.AIConversations) == 0 && len(config.AIChatContext) > 0 {
		// Existing installs stored a single transcript. Preserve it as the first
		// independent conversation rather than silently discarding its history.
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

// SaveAIConversation replaces the current user's bounded collection of
// independent chats. The browser submits the full ordered list so switching
// and creating chats remains equally safe in JSON and PostgreSQL modes.
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

func ClearAIConversation(ctx context.Context) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.AIChatContext = nil
	config.AIConversations = nil
	return saveConfig(ctx, config)
}

func normalizeAIConversations(conversations []models.AIConversation) ([]models.AIConversation, error) {
	if len(conversations) > aiConversationMaxConversations {
		conversations = conversations[:aiConversationMaxConversations]
	}
	result := make([]models.AIConversation, 0, len(conversations))
	seenIDs := make(map[string]struct{}, len(conversations))
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
		result = append(result, models.AIConversation{
			ID:               id,
			Title:            aiConversationTitleWithFallback(conversation.Title, messages),
			FolderID:         folderID,
			ItemIDs:          normalizeAIConversationItemIDs(conversation.ItemIDs),
			Messages:         messages,
			HarnessSessionID: normalizeAIHarnessSessionID(conversation.HarnessSessionID),
		})
	}
	return result, nil
}

func normalizeAIHarnessSessionID(value string) string {
	value = strings.TrimSpace(value)
	if !validAIConversationID(value) {
		return ""
	}
	return value
}

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

func aiConversationTitleWithFallback(value string, messages []models.AIConversationMessage) string {
	if title := boundedAIConversationText(value, aiConversationMaxName); title != "" {
		return title
	}
	return aiConversationTitle(messages)
}

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
		})
	}
	return result
}

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

func boundedAIConversationText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
