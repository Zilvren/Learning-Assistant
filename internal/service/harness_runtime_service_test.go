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

// TestHarnessSessionIDKeepsOneConversationSeparate 验证当前模块在相应场景下的行为与边界条件。
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

// TestHarnessPromptKeepsClientContinuityAfterSessionCompaction 验证已有 Harness 会话也会携带客户端连续记录，
// 防止提供商压缩旧上下文后让用户看到像“新对话”一样的回答。
func TestHarnessPromptKeepsClientContinuityAfterSessionCompaction(t *testing.T) {
	prompt := harnessPrompt(models.AIChatRequest{
		HarnessSessionID: "stored-session",
		History: []models.AIChatMessage{
			{Role: "user", Content: "我们先聊天，最后把内容整理进 20260822.md。"},
			{Role: "assistant", Content: "好的，我会在最后整理。"},
			{Role: "user", Content: "继续聊今天的学习计划。"},
		},
	}, "请记住最后要做什么？")
	for _, expected := range []string{
		"Client-backed continuity record",
		"Return only the final answer for the user.",
		"我们先聊天，最后把内容整理进 20260822.md。",
		"继续聊今天的学习计划。",
		"Current user message:\n请记住最后要做什么？",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt is missing %q:\n%s", expected, prompt)
		}
	}
}

// TestHarnessContinuityHistoryKeepsOpeningGoalAndRecentTurns 验证当记录超出预算时，仍同时保留开场目标和末尾对话。
func TestHarnessContinuityHistoryKeepsOpeningGoalAndRecentTurns(t *testing.T) {
	history := []models.AIChatMessage{
		{Role: "user", Content: "开场目标"},
		{Role: "assistant", Content: strings.Repeat("旧", 30)},
		{Role: "user", Content: strings.Repeat("近", 30)},
	}
	continuity := harnessContinuityHistory(history, 44)
	if len(continuity) != 2 || continuity[0].Content != "开场目标" || continuity[1].Content != strings.Repeat("近", 30) {
		t.Fatalf("expected opening goal plus newest turn, got %#v", continuity)
	}
}

// TestHarnessAssistantFragmentsReadDurableAssistantMessages 验证当前模块在相应场景下的行为与边界条件。
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

// TestSanitizeHarnessAnswerRemovesHiddenReasoning 验证模型意外混入的思考标签不会展示给用户。
func TestSanitizeHarnessAnswerRemovesHiddenReasoning(t *testing.T) {
	raw := "<think>先分析资料和工具调用</think>\n\n最终答案：今天先复习导数。\n<analysis>不应展示</analysis>"
	if got := sanitizeHarnessAnswer(raw); got != "最终答案：今天先复习导数。" {
		t.Fatalf("unexpected visible answer: %q", got)
	}
	if got := sanitizeHarnessAnswer("<think>尚未完成的思考"); got != "" {
		t.Fatalf("expected unfinished reasoning to be hidden, got %q", got)
	}
}

// 此测试为可选项，因为它会启动固定版本的真实 Harness 运行时。它使用本地伪造的 DeepSeek SSE 端点，不会把请求或密钥发送到网络；执行 npm install 后可设置 RUN_HARNESS_INTEGRATION_TEST=1 运行。
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
