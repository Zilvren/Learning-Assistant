package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

const (
	aiMaxMessageRunes        = 2_000
	aiMaxHistoryMessages     = 160
	aiMaxHistoryMessageRunes = 16_000
	aiMaxScopedItems         = 60
	aiCompactSummaryRunes    = 24_000
	aiRequestTimeout         = 30 * time.Minute
	deepSeekFlashModel       = "deepseek-v4-flash"
	deepSeekProModel         = "deepseek-v4-pro"
	deepSeekDefaultModel     = deepSeekFlashModel
)

var (
	ErrDeepSeekNotConfigured    = errors.New("请先在设置中心配置 DeepSeek API Key")
	ErrUnsupportedDeepSeekModel = errors.New("不支持的 DeepSeek 默认模型")
	ErrAIInvalidScope           = errors.New("AI 资料范围无效")
)

// aiLibraryScope is the user-selected boundary that is copied into the
// short-lived Harness capability token. The Node agent never receives direct
// repository access.
type aiLibraryScope struct {
	folderID *int64
	itemIDs  []int64
}

func (scope aiLibraryScope) active() bool {
	return scope.folderID != nil || len(scope.itemIDs) > 0
}

// ChatWithStudyAI is deliberately Harness-only. There is no direct
// OpenAI-compatible DeepSeek fallback: unavailable Harness dependencies are
// returned to the caller as ErrHarnessRuntimeUnavailable.
func ChatWithStudyAI(ctx context.Context, request models.AIChatRequest) (models.AIChatResponse, error) {
	return chatWithHarnessStudyAI(ctx, request)
}

func newAILibraryScope(request models.AIChatRequest) (aiLibraryScope, error) {
	scope := aiLibraryScope{folderID: request.FolderID}
	if scope.folderID != nil && *scope.folderID <= 0 {
		return aiLibraryScope{}, fmt.Errorf("%w：资料路径无效", ErrAIInvalidScope)
	}
	seen := make(map[int64]struct{}, len(request.ItemIDs))
	for _, id := range request.ItemIDs {
		if id <= 0 {
			return aiLibraryScope{}, fmt.Errorf("%w：资料 ID 无效", ErrAIInvalidScope)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		scope.itemIDs = append(scope.itemIDs, id)
	}
	if len(scope.itemIDs) > aiMaxScopedItems {
		return aiLibraryScope{}, fmt.Errorf("%w：一次最多选择 %d 项资料", ErrAIInvalidScope, aiMaxScopedItems)
	}
	return scope, nil
}

func deepSeekAPIKey(ctx context.Context) (string, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return "", err
	}
	if key := strings.TrimSpace(config.DeepSeekToken); key != "" {
		return key, nil
	}
	if key := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")); key != "" {
		return key, nil
	}
	return "", ErrDeepSeekNotConfigured
}

func deepSeekModel(ctx context.Context) (string, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return "", err
	}
	if model := strings.TrimSpace(config.DeepSeekModel); model != "" {
		return normalizeDeepSeekModel(model)
	}
	if model := strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL")); model != "" {
		return model, nil
	}
	return deepSeekDefaultModel, nil
}

func normalizeDeepSeekModel(model string) (string, error) {
	switch strings.TrimSpace(model) {
	case deepSeekFlashModel:
		return deepSeekFlashModel, nil
	case deepSeekProModel:
		return deepSeekProModel, nil
	default:
		return "", fmt.Errorf("%w：仅支持 %s 或 %s", ErrUnsupportedDeepSeekModel, deepSeekFlashModel, deepSeekProModel)
	}
}

func filterAILibraryScope(items []models.LibraryItem, scope aiLibraryScope) ([]models.LibraryItem, error) {
	byID := make(map[int64]models.LibraryItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	if scope.folderID != nil {
		folder, exists := byID[*scope.folderID]
		if !exists || folder.Kind != "folder" {
			return nil, fmt.Errorf("%w：资料路径不存在或不可用", ErrAIInvalidScope)
		}
	}
	selected := make(map[int64]struct{}, len(scope.itemIDs))
	for _, id := range scope.itemIDs {
		if _, exists := byID[id]; !exists {
			return nil, fmt.Errorf("%w：所选资料不存在或不可用", ErrAIInvalidScope)
		}
		selected[id] = struct{}{}
	}

	inFolder := func(item models.LibraryItem) bool {
		if scope.folderID == nil {
			return false
		}
		for cursor := item; ; {
			if cursor.ID == *scope.folderID {
				return true
			}
			if cursor.ParentID == nil {
				return false
			}
			parent, exists := byID[*cursor.ParentID]
			if !exists {
				return false
			}
			cursor = parent
		}
	}

	result := make([]models.LibraryItem, 0, len(items))
	for _, item := range items {
		if inFolder(item) {
			result = append(result, item)
			continue
		}
		for cursor := item; ; {
			if _, exists := selected[cursor.ID]; exists {
				result = append(result, item)
				break
			}
			if cursor.ParentID == nil {
				break
			}
			parent, exists := byID[*cursor.ParentID]
			if !exists {
				break
			}
			cursor = parent
		}
	}
	return result, nil
}

func collectAILibraryItems(ctx context.Context, repo repository.LibraryRepository) ([]models.LibraryItem, error) {
	result := []models.LibraryItem{}
	queue := []*int64{nil}
	visited := map[int64]bool{}
	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]
		items, err := repo.List(ctx, repository.LibraryFilter{ParentID: parentID})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if visited[item.ID] {
				continue
			}
			visited[item.ID] = true
			result = append(result, item)
			if item.Kind == "folder" {
				id := item.ID
				queue = append(queue, &id)
			}
		}
	}
	return result, nil
}

func aiReadableLibraryItem(item models.LibraryItem) bool {
	if item.Kind == "note" {
		return true
	}
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(item.MimeType, ";")[0]))
	if strings.HasPrefix(mimeType, "text/") || mimeType == "application/json" {
		return true
	}
	if strings.HasSuffix(strings.ToLower(item.Name), ".md") || strings.HasSuffix(strings.ToLower(item.Name), ".txt") {
		return true
	}
	return mimeType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		mimeType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
		mimeType == "application/vnd.openxmlformats-officedocument.presentationml.presentation"
}

func aiScore(question, text string) int {
	text = strings.ToLower(text)
	if text == "" {
		return 0
	}
	score := 0
	for _, term := range aiSearchTerms(question) {
		if term == "" {
			continue
		}
		count := strings.Count(text, term)
		if count > 0 {
			score += count * max(1, len([]rune(term))-1)
		}
	}
	return score
}

func aiSearchTerms(question string) []string {
	clean := strings.ToLower(strings.TrimSpace(question))
	seen := map[string]bool{}
	add := func(values *[]string, value string) {
		value = strings.TrimSpace(value)
		if len([]rune(value)) >= 2 && !seen[value] {
			seen[value] = true
			*values = append(*values, value)
		}
	}
	terms := []string{}
	add(&terms, clean)
	for _, value := range strings.FieldsFunc(clean, func(r rune) bool {
		return r == ' ' || r == '，' || r == '。' || r == '？' || r == '?' || r == '、' || r == '！' || r == '!'
	}) {
		add(&terms, value)
	}
	runes := []rune(clean)
	for index := 0; index+1 < len(runes); index++ {
		add(&terms, string(runes[index:index+2]))
	}
	return terms
}

func normalizeAIHistory(history []models.AIChatMessage) []models.AIChatMessage {
	if len(history) > aiMaxHistoryMessages {
		history = history[len(history)-aiMaxHistoryMessages:]
	}
	result := make([]models.AIChatMessage, 0, len(history))
	for _, message := range history {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := aiBoundedText(strings.TrimSpace(message.Content), aiMaxHistoryMessageRunes)
		if content == "" {
			continue
		}
		result = append(result, models.AIChatMessage{Role: role, Content: content})
	}
	return result
}

func aiHistoryTranscript(history []models.AIChatMessage) string {
	parts := make([]string, 0, len(history))
	for _, item := range history {
		parts = append(parts, aiHistoryLine(item))
	}
	return strings.Join(parts, "\n\n")
}

func aiHistoryLine(item models.AIChatMessage) string {
	role := "用户"
	if item.Role == "assistant" {
		role = "助手"
	}
	return role + "：" + item.Content
}

func aiApproxTokens(value string) int {
	asciiRun := 0
	tokens := 0
	flushASCII := func() {
		if asciiRun > 0 {
			tokens += (asciiRun + 3) / 4
			asciiRun = 0
		}
	}
	for _, char := range value {
		if char <= 0x7f && ((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			asciiRun++
			continue
		}
		flushASCII()
		tokens++
	}
	flushASCII()
	return tokens
}

func aiBoundedText(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	if maximum == 1 {
		return "…"
	}
	return string(runes[:maximum-1]) + "…"
}

func aiBoundedTokens(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	if aiApproxTokens(value) <= maximum {
		return value
	}
	runes := []rune(value)
	low, high := 0, len(runes)
	for low < high {
		mid := (low + high + 1) / 2
		if aiApproxTokens(string(runes[:mid])) <= maximum-1 {
			low = mid
		} else {
			high = mid - 1
		}
	}
	if low == 0 {
		return "…"
	}
	return string(runes[:low]) + "…"
}
