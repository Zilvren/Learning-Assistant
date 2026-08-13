package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/service"
)

// AIChat accepts a user question and keeps the provider key on the server.
// The browser receives only the model answer and titles of the context used.
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
		case errors.Is(err, service.ErrDeepSeekClientUnavailable):
			respondProblem(c, http.StatusServiceUnavailable, "deepseek_client_unavailable", err.Error())
		case errors.Is(err, service.ErrAIInvalidScope):
			respondProblem(c, http.StatusBadRequest, "invalid_ai_scope", err.Error())
		default:
			respondError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusOK, response)
}

// PreviewAIEdit prepares a change set for one explicitly selected note. It is
// deliberately separate from apply so the browser can always show a preview.
func PreviewAIEdit(c *gin.Context) {
	var request models.AIEditPreviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_edit_request", "AI 编辑请求格式错误")
		return
	}
	response, err := service.PreviewAIEdit(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeepSeekNotConfigured):
			respondProblem(c, http.StatusBadRequest, "deepseek_not_configured", err.Error())
		case errors.Is(err, service.ErrDeepSeekClientUnavailable):
			respondProblem(c, http.StatusServiceUnavailable, "deepseek_client_unavailable", err.Error())
		case errors.Is(err, service.ErrAIEditTarget):
			respondProblem(c, http.StatusBadRequest, "invalid_ai_edit_target", err.Error())
		default:
			respondError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusOK, response)
}

// ApplyAIEdit persists a preview only after the user explicitly confirms it.
func ApplyAIEdit(c *gin.Context) {
	var request models.AIEditApplyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_edit_apply", "AI 编辑确认格式错误")
		return
	}
	item, err := service.ApplyAIEdit(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIEditTarget):
			respondProblem(c, http.StatusBadRequest, "invalid_ai_edit_target", err.Error())
		case strings.Contains(err.Error(), "版本冲突"):
			respondProblem(c, http.StatusConflict, "version_conflict", "笔记已被更新，请重新生成修改预览")
		default:
			respondError(c, http.StatusBadRequest, err)
		}
		return
	}
	c.JSON(http.StatusOK, item)
}

// PreviewAINoteWrite resolves a natural-language target path and generates a
// create or update preview without changing the user's library.
func PreviewAINoteWrite(c *gin.Context) {
	var request models.AINoteWritePreviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_note_write_request", "AI 写入请求格式错误")
		return
	}
	response, err := service.PreviewAINoteWrite(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeepSeekNotConfigured):
			respondProblem(c, http.StatusBadRequest, "deepseek_not_configured", err.Error())
		case errors.Is(err, service.ErrDeepSeekClientUnavailable):
			respondProblem(c, http.StatusServiceUnavailable, "deepseek_client_unavailable", err.Error())
		case errors.Is(err, service.ErrAINoteWriteIntent):
			respondProblem(c, http.StatusBadRequest, "not_ai_note_write_intent", err.Error())
		case errors.Is(err, service.ErrAINoteWriteTarget):
			respondProblem(c, http.StatusBadRequest, "invalid_ai_note_write_target", err.Error())
		default:
			respondError(c, http.StatusBadGateway, err)
		}
		return
	}
	c.JSON(http.StatusOK, response)
}

// ApplyAINoteWrite creates or updates exactly the note shown in an approved
// preview. It is deliberately separate from preview generation.
func ApplyAINoteWrite(c *gin.Context) {
	var request models.AINoteWriteApplyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_note_write_apply", "AI 写入确认格式错误")
		return
	}
	item, err := service.ApplyAINoteWrite(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAINoteWriteConflict), strings.Contains(err.Error(), "版本冲突"):
			respondProblem(c, http.StatusConflict, "ai_note_write_conflict", "目标笔记已变化，请重新生成写入预览")
		case errors.Is(err, service.ErrAINoteWriteTarget):
			respondProblem(c, http.StatusBadRequest, "invalid_ai_note_write_target", err.Error())
		default:
			respondError(c, http.StatusBadRequest, err)
		}
		return
	}
	c.JSON(http.StatusOK, item)
}

type aiConversationRequest struct {
	Conversations []models.AIConversation `json:"conversations"`
}

// GetAIConversation restores the signed-in user's independently scoped chats.
func GetAIConversation(c *gin.Context) {
	conversations, err := service.GetAIConversation(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// SaveAIConversation persists a bounded collection of user/assistant chats.
// API keys and provider prompts are never included in this payload.
func SaveAIConversation(c *gin.Context) {
	var request aiConversationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_ai_conversation", "对话上下文格式错误")
		return
	}
	conversations, err := service.SaveAIConversation(c.Request.Context(), request.Conversations)
	if err != nil {
		if errors.Is(err, service.ErrInvalidAIConversation) {
			respondProblem(c, http.StatusBadRequest, "invalid_ai_conversation", err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

func ClearAIConversation(c *gin.Context) {
	if err := service.ClearAIConversation(c.Request.Context()); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}
