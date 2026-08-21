package handlers

import (
	"errors"
	"net/http"

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
		if errors.Is(err, service.ErrInvalidAIConversation) {
			respondProblem(c, http.StatusBadRequest, "invalid_ai_conversation", err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// ClearAIConversation 在 HTTP 处理层中完成当前请求的处理与响应。
func ClearAIConversation(c *gin.Context) {
	if err := service.ClearAIConversation(c.Request.Context()); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}
