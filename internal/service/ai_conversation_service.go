package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

import models "study-tracker-go/internal/model"

const (
	aiConversationMaxMessages = 32
	aiConversationMaxRunes    = 8_000
	aiConversationMaxScope    = 240
	aiConversationMaxModel    = 120
	aiConversationMaxSources  = 8
	aiConversationMaxTitle    = 240
	aiConversationMaxExcerpt  = 800
)

var ErrInvalidAIConversation = errors.New("AI 对话上下文无效")

// GetAIConversation loads the current user's saved chat. It is intentionally
// separate from provider history so a browser reload can restore the visible
// conversation while ChatWithStudyAI still receives a bounded recent context.
func GetAIConversation(ctx context.Context) ([]models.AIConversationMessage, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return append([]models.AIConversationMessage(nil), config.AIChatContext...), nil
}

// SaveAIConversation replaces the current user's saved chat after validating
// its shape and limiting its size. Keeping the latest 16 turns avoids turning
// a settings record into an unbounded transcript.
func SaveAIConversation(ctx context.Context, messages []models.AIConversationMessage) ([]models.AIConversationMessage, error) {
	normalized, err := normalizeAIConversation(messages)
	if err != nil {
		return nil, err
	}
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	config.AIChatContext = normalized
	if err := saveConfig(ctx, config); err != nil {
		return nil, err
	}
	return normalized, nil
}

func ClearAIConversation(ctx context.Context) error {
	config, err := loadConfig(ctx)
	if err != nil {
		return err
	}
	config.AIChatContext = nil
	return saveConfig(ctx, config)
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
