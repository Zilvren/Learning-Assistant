package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	models "study-tracker-go/internal/model"
)

func TestChatWithDeepSeekOpenAIUsesCompatibleChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected SDK request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-local-test" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if got := payload["model"]; got != deepSeekProModel {
			t.Fatalf("unexpected model: %#v", got)
		}
		if got := payload["max_tokens"]; got != float64(deepSeekMaxCompletionTokens) {
			t.Fatalf("unexpected completion token limit: %#v", got)
		}
		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) != 4 {
			t.Fatalf("unexpected messages: %#v", payload["messages"])
		}
		last, ok := messages[len(messages)-1].(map[string]any)
		if !ok || last["role"] != "user" || last["content"] != "新的学习问题" {
			t.Fatalf("expected final user message, got %#v", last)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-local","object":"chat.completion","created":0,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"先复习导数的符号表。"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	previousBaseURL := deepSeekBaseURL
	deepSeekBaseURL = server.URL
	t.Cleanup(func() { deepSeekBaseURL = previousBaseURL })
	answer, model, err := chatWithDeepSeekOpenAI(context.Background(), "sk-local-test", deepSeekProModel, "系统约束", []models.AIChatMessage{
		{Role: "user", Content: "上一轮问题"},
		{Role: "assistant", Content: "上一轮回答"},
	}, "新的学习问题")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "先复习导数的符号表。" || model != "deepseek-chat" {
		t.Fatalf("unexpected completion: answer=%q model=%q", answer, model)
	}
}
