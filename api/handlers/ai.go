package handlers

import (
	"errors"
	"net/http"

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
