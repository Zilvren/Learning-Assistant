package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	models "study-tracker-go/internal/model"
)

func TestHarnessSessionIDKeepsOneConversationSeparate(t *testing.T) {
	if got := harnessSessionID(models.AIChatRequest{HarnessSessionID: "stored-session", ConversationID: "chat-other"}); got != "stored-session" {
		t.Fatalf("expected stored session, got %q", got)
	}
	if got := harnessSessionID(models.AIChatRequest{ConversationID: "chat-123"}); got != "chat-123" {
		t.Fatalf("expected conversation id fallback, got %q", got)
	}
	if got := harnessSessionID(models.AIChatRequest{ConversationID: "bad id"}); !strings.HasPrefix(got, "harness-") || !validAIConversationID(got) {
		t.Fatalf("expected generated safe session id, got %q", got)
	}
}

func TestHarnessAssistantFragmentsReadDurableAssistantMessages(t *testing.T) {
	value := map[string]any{
		"sessionId": "chat-1",
		"event": map[string]any{
			"type": "message/created",
			"message": map[string]any{
				"role":    "assistant",
				"content": []any{map[string]any{"type": "text", "text": "你好，已找到笔记。"}},
			},
		},
	}
	fragments := harnessAssistantFragments(value, false)
	if len(fragments) != 1 || fragments[0] != "你好，已找到笔记。" {
		t.Fatalf("unexpected assistant fragments: %#v", fragments)
	}
	if got := mergeHarnessText("你好", "你好，世界"); got != "你好，世界" {
		t.Fatalf("expected cumulative chunk to replace prior content, got %q", got)
	}
}

// This is opt-in because it boots the real pinned Harness runtime. It uses a
// local fake DeepSeek SSE endpoint, so it never sends a request or key to the
// network. Run with RUN_HARNESS_INTEGRATION_TEST=1 after npm install.
func TestHarnessAgentRoundTripWithLocalProvider(t *testing.T) {
	if os.Getenv("RUN_HARNESS_INTEGRATION_TEST") != "1" {
		t.Skip("set RUN_HARNESS_INTEGRATION_TEST=1 to run the local Harness round trip")
	}
	nodePath := strings.TrimSpace(os.Getenv("STUDY_HARNESS_NODE"))
	if nodePath == "" {
		t.Skip("STUDY_HARNESS_NODE must point to Node.js 22.19+")
	}
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir := filepath.Join(projectRoot, "harness")
	capabilityToken := "a-valid-local-capability-token-which-is-never-used"
	var toolCalls atomic.Int32
	bridge := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/internal/harness/tools/list_paths" || request.Header.Get("Authorization") != "Bearer "+capabilityToken {
			http.Error(writer, "unexpected local tool request", http.StatusForbidden)
			return
		}
		toolCalls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"result":{"paths":[{"id":1,"path":"数学 / 导数.md","kind":"note"}]}}`)
	}))
	defer bridge.Close()
	runtime := harnessRuntimeConfig{
		nodePath:    nodePath,
		agentPath:   filepath.Join(runtimeDir, "node_modules", "@deepseek-ai", "dsh-sdk-jsonrpc-demo", "lib", "bin.js"),
		configPath:  filepath.Join(runtimeDir, "learning-agent.cordis.yml"),
		sessionRoot: t.TempDir(),
		bridgeURL:   bridge.URL,
	}
	for _, path := range []string{runtime.nodePath, runtime.agentPath, runtime.configPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Harness test dependency is missing at %s: %v", path, err)
		}
	}
	var providerCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/chat/completions" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		if providerCalls.Add(1) == 1 {
			_, _ = fmt.Fprint(writer, "data: {\"id\":\"local-test\",\"object\":\"chat.completion.chunk\",\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-list\",\"type\":\"function\",\"function\":{\"name\":\"list_library_paths\",\"arguments\":\"{\\\"query\\\":\\\"导数\\\"}\"}}]},\"finish_reason\":null}]}\n\n")
			_, _ = fmt.Fprint(writer, "data: {\"id\":\"local-test\",\"object\":\"chat.completion.chunk\",\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
			_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
			return
		}
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"local-test\",\"object\":\"chat.completion.chunk\",\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Harness 工具回环成功。\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"local-test\",\"object\":\"chat.completion.chunk\",\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":6,\"total_tokens\":16}}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}))
	defer provider.Close()
	t.Setenv("DEEPSEEK_BASE_URL", provider.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	answer, err := runHarnessAgent(ctx, runtime, "sk-local-test", deepSeekFlashModel, "harness-local-test", capabilityToken, "请简短确认运行状态。")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "Harness 工具回环成功。" {
		t.Fatalf("unexpected Harness answer: %q", answer)
	}
	if providerCalls.Load() != 2 || toolCalls.Load() != 1 {
		t.Fatalf("expected one local tool call and two model requests, got tools=%d model=%d", toolCalls.Load(), providerCalls.Load())
	}
}
