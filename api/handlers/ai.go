package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/service"
)

// AIChat 接收面向仅 Harness 学习助手的用户问题。浏览器只会得到回答和引用，绝不会得到提供商密钥或工具能力令牌。
func AIChat(c *gin.Context) {
	var request models.AIChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_chat_request", "聊天请求格式错误")
		return
	}
	response, err := service.ChatWithStudyAI(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeepSeekNotConfigured):
			respondProblem(c, http.StatusBadRequest, "deepseek_not_configured", err.Error())
		case errors.Is(err, service.ErrHarnessRuntimeUnavailable):
			respondProblem(c, http.StatusServiceUnavailable, "harness_runtime_unavailable", err.Error())
		case errors.Is(err, service.ErrAIInvalidScope):
			respondProblem(c, http.StatusBadRequest, "invalid_ai_scope", err.Error())
		default:
			respondError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusOK, response)
}

// AIChatStream 将经过服务端过滤的最终答案以 SSE 快照推送给浏览器；不会发送模型思考、工具调用或内部摘要。
func AIChatStream(c *gin.Context) {
	var request models.AIChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_chat_request", "聊天请求格式错误")
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	writeAIChatStreamEvent(c, "status", gin.H{"status": "generating"})
	response, err := service.ChatWithStudyAIStream(c.Request.Context(), request, func(answer string) {
		writeAIChatStreamEvent(c, "answer", gin.H{"answer": answer})
	})
	if err != nil {
		code, message := aiChatStreamError(err)
		writeAIChatStreamEvent(c, "error", gin.H{"code": code, "message": message})
		return
	}
	writeAIChatStreamEvent(c, "done", response)
}

// StartAITurn 创建可重连的后台 AI 任务。断开本次 HTTP 请求不会终止 Harness 运行。
func StartAITurn(c *gin.Context) {
	var request models.AIChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_chat_request", "聊天请求格式错误")
		return
	}
	turn, err := service.StartAITurn(c.Request.Context(), request)
	if err != nil {
		respondAITurnError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, turn)
}

// GetAITurn 返回一条任务的最终或当前状态，供浏览器刷新后恢复界面。
func GetAITurn(c *gin.Context) {
	turn, err := service.GetAITurn(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondAITurnError(c, err)
		return
	}
	c.JSON(http.StatusOK, turn)
}

// ListAITurns 返回指定对话的任务历史，前端仅需订阅仍在运行的任务。
func ListAITurns(c *gin.Context) {
	turns, err := service.ListAITurns(c.Request.Context(), c.Query("conversation_id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"turns": turns})
}

// StreamAITurn 重放 after 之后的事件并继续订阅实时事件；事件不包含思考过程或资料正文。
func StreamAITurn(c *gin.Context) {
	after, err := aiTurnAfter(c)
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_turn_cursor", "任务事件游标无效")
		return
	}
	_, history, events, unsubscribe, err := service.SubscribeAITurn(c.Request.Context(), c.Param("id"), after)
	if err != nil {
		respondAITurnError(c, err)
		return
	}
	defer unsubscribe()
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	for _, event := range history {
		writeAITurnStreamEvent(c, event)
	}
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			writeAITurnStreamEvent(c, event)
		}
	}
}

// CancelAITurn 由用户显式取消后台任务并使其 Harness 子进程退出。
func CancelAITurn(c *gin.Context) {
	turn, err := service.CancelAITurn(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondAITurnError(c, err)
		return
	}
	c.JSON(http.StatusOK, turn)
}

type aiTurnApprovalRequest struct {
	Approved bool `json:"approved"`
}

// ResolveAITurnWriteApproval 将用户明确的同意或拒绝送回暂停的资料库写入工具。
func ResolveAITurnWriteApproval(c *gin.Context) {
	var request aiTurnApprovalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_write_approval", "写入确认格式错误")
		return
	}
	turn, err := service.ResolveAITurnWriteApproval(c.Request.Context(), c.Param("id"), c.Param("approvalID"), request.Approved)
	if err != nil {
		respondAITurnError(c, err)
		return
	}
	c.JSON(http.StatusOK, turn)
}

func aiTurnAfter(c *gin.Context) (int64, error) {
	raw := c.GetHeader("Last-Event-ID")
	if raw == "" {
		raw = c.Query("after")
	}
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return value, nil
}

func writeAITurnStreamEvent(c *gin.Context, event models.AITurnEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", event.Sequence, event.Type, payload)
	c.Writer.Flush()
}

func respondAITurnError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAITurnNotFound):
		respondProblem(c, http.StatusNotFound, "ai_turn_not_found", err.Error())
	case errors.Is(err, service.ErrAITurnAlreadyRunning):
		respondProblem(c, http.StatusConflict, "ai_turn_already_running", err.Error())
	case errors.Is(err, service.ErrAITurnNotCancellable):
		respondProblem(c, http.StatusConflict, "ai_turn_not_cancellable", err.Error())
	case errors.Is(err, service.ErrAIWriteApprovalNotFound):
		respondProblem(c, http.StatusNotFound, "ai_write_approval_not_found", err.Error())
	case errors.Is(err, service.ErrAIWriteApprovalResolved):
		respondProblem(c, http.StatusConflict, "ai_write_approval_resolved", err.Error())
	case errors.Is(err, service.ErrDeepSeekNotConfigured):
		respondProblem(c, http.StatusBadRequest, "deepseek_not_configured", err.Error())
	case errors.Is(err, service.ErrHarnessRuntimeUnavailable):
		respondProblem(c, http.StatusServiceUnavailable, "harness_runtime_unavailable", err.Error())
	case errors.Is(err, service.ErrAIInvalidScope):
		respondProblem(c, http.StatusBadRequest, "invalid_ai_scope", err.Error())
	default:
		respondError(c, http.StatusInternalServerError, err)
	}
}

// writeAIChatStreamEvent 在当前 HTTP 流中写入一条完整 SSE 事件，并立即刷新到浏览器。
func writeAIChatStreamEvent(c *gin.Context, event string, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, payload)
	c.Writer.Flush()
}

// aiChatStreamError 将无法再通过 HTTP 状态码表达的流内错误压缩为稳定的浏览器提示。
func aiChatStreamError(err error) (code, message string) {
	switch {
	case errors.Is(err, service.ErrDeepSeekNotConfigured):
		return "deepseek_not_configured", err.Error()
	case errors.Is(err, service.ErrHarnessRuntimeUnavailable):
		return "harness_runtime_unavailable", err.Error()
	case errors.Is(err, service.ErrAIInvalidScope):
		return "invalid_ai_scope", err.Error()
	default:
		return "ai_chat_failed", "AI 暂时无法回答，请稍后重试"
	}
}

type aiConversationRequest struct {
	Conversations []models.AIConversation `json:"conversations"`
}

// GetAIConversation 恢复已登录用户范围独立的对话。
func GetAIConversation(c *gin.Context) {
	conversations, err := service.GetAIConversation(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// SaveAIConversation 持久化数量受限的用户和助手对话集合，同时保留 Harness 会话标识而不暴露任何工具授权。
func SaveAIConversation(c *gin.Context) {
	var request aiConversationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_conversation", "对话上下文格式错误")
		return
	}
	conversations, err := service.SaveAIConversation(c.Request.Context(), request.Conversations)
	if err != nil {
		respondAIConversationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// ArchiveAIConversation 将一条活跃对话归档，归档后不会占用活跃对话数量。
func ArchiveAIConversation(c *gin.Context) {
	conversations, err := service.ArchiveAIConversation(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondAIConversationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// RestoreAIConversation 恢复一条归档对话；活跃对话达到上限时返回明确提示。
func RestoreAIConversation(c *gin.Context) {
	conversations, err := service.RestoreAIConversation(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondAIConversationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// DeleteAIConversation 永久删除一条已经归档的对话。
func DeleteAIConversation(c *gin.Context) {
	conversations, err := service.DeleteArchivedAIConversation(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondAIConversationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// respondAIConversationError 将对话操作的领域错误转换为前端可读且稳定的 HTTP 响应。
func respondAIConversationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidAIConversation):
		respondProblem(c, http.StatusBadRequest, "invalid_ai_conversation", err.Error())
	case errors.Is(err, service.ErrAIConversationNotFound):
		respondProblem(c, http.StatusNotFound, "ai_conversation_not_found", err.Error())
	case errors.Is(err, service.ErrAIConversationActiveLimit), errors.Is(err, service.ErrAIConversationArchivedLimit):
		respondProblem(c, http.StatusConflict, "ai_conversation_limit", err.Error())
	default:
		respondError(c, http.StatusInternalServerError, err)
	}
}

// ClearAIConversation 在 HTTP 处理层中完成当前请求的处理与响应。
func ClearAIConversation(c *gin.Context) {
	if err := service.ClearAIConversation(c.Request.Context()); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}
