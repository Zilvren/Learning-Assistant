package service

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	models "study-tracker-go/internal/model"
)

const (
	deepSeekBaseURLDefault = "https://api.deepseek.com"
	// A 1,200-token cap often ends structured summaries in the middle of a
	// list or Markdown link. 4,096 is sufficient for normal library reports;
	// the completion finish reason still tells the UI when a continuation is
	// required for unusually long answers.
	deepSeekMaxCompletionTokens = 4_096
)

// deepSeekBaseURL is a package variable only so the SDK adapter can be verified
// against a local HTTP server. Production always uses DeepSeek's OpenAI-compatible
// endpoint above.
var deepSeekBaseURL = deepSeekBaseURLDefault

func init() {
	runDeepSeekChat = chatWithDeepSeekOpenAI
}

// chatWithDeepSeekOpenAI calls DeepSeek through its OpenAI-compatible Chat
// Completions endpoint. Request and response shapes are owned by openai-go rather
// than being duplicated as provider-specific application structs.
func chatWithDeepSeekOpenAI(ctx context.Context, apiKey, modelName, systemPrompt string, history []models.AIChatMessage, message string) (string, string, bool, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+2)
	messages = append(messages, openai.SystemMessage(systemPrompt))
	for _, item := range history {
		switch item.Role {
		case "assistant":
			messages = append(messages, openai.AssistantMessage(item.Content))
		case "user":
			messages = append(messages, openai.UserMessage(item.Content))
		}
	}
	messages = append(messages, openai.UserMessage(message))

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(deepSeekBaseURL),
		// A completed POST can be billable. The UI can offer an explicit retry, so
		// do not issue an invisible second inference request after a network error.
		option.WithMaxRetries(0),
	)
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     modelName,
		Messages:  messages,
		MaxTokens: openai.Int(deepSeekMaxCompletionTokens),
	})
	if err != nil {
		return "", "", false, fmt.Errorf("DeepSeek 请求失败：%w", err)
	}
	if len(completion.Choices) == 0 {
		return "", "", false, fmt.Errorf("DeepSeek 没有返回回答，请重试")
	}
	answer := strings.TrimSpace(completion.Choices[0].Message.Content)
	if answer == "" {
		return "", "", false, fmt.Errorf("DeepSeek 没有返回可显示的内容，请重试")
	}
	model := strings.TrimSpace(completion.Model)
	if model == "" {
		model = modelName
	}
	incomplete := strings.EqualFold(strings.TrimSpace(completion.Choices[0].FinishReason), "length")
	return answer, model, incomplete, nil
}
