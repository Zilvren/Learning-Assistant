package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/service"
)

// ListLearningRelations 在 HTTP 处理层中完成当前请求的处理与响应。
func ListLearningRelations(c *gin.Context) {
	sourceType := strings.TrimSpace(c.Query("source_type"))
	sourceID, err := strconv.ParseInt(c.Query("source_id"), 10, 64)
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_relation_source", "关联来源无效")
		return
	}
	items, err := service.ListLearningRelations(c.Request.Context(), sourceType, sourceID)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// CreateLearningRelation 在 HTTP 处理层中完成当前请求的处理与响应。
func CreateLearningRelation(c *gin.Context) {
	var relation models.LearningRelation
	if err := c.ShouldBindJSON(&relation); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_relation", "关联格式错误")
		return
	}
	created, err := service.CreateLearningRelation(c.Request.Context(), relation)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

// DeleteLearningRelation 在 HTTP 处理层中完成当前请求的处理与响应。
func DeleteLearningRelation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_relation_id", "关联 ID 无效")
		return
	}
	if err := service.DeleteLearningRelation(c.Request.Context(), id); err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "关联已移除"})
}
