package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/service"
)

// CreateError 在HTTP 处理层中创建或更新相应状态。
func CreateError(c *gin.Context) {
	var req models.AddErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}

	item, err := service.CreateError(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": item.ID, "message": "添加成功"})
}

// GetErrors 处理 GET /api/errors
func GetErrors(c *gin.Context) {
	errors, err := service.GetAllErrors(
		c.Request.Context(),
		c.Query("subject"),    // ?subject=数学
		c.Query("keyword"),    // ?keyword=导数
		c.Query("tag"),        // ?tag=选择题
		c.Query("reason_tag"), // ?reason_tag=概念混淆
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	// 注意：前端读 res.errors 和 res.total
	c.JSON(http.StatusOK, gin.H{"errors": errors, "total": len(errors)})
}

func GetError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_id", "ID格式错误")
		return
	}
	item, err := service.GetError(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateError 在HTTP 处理层中创建或更新相应状态。
func UpdateError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_id", "ID格式错误")
		return
	}

	var req models.UpdateErrorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
		return
	}

	if err := service.UpdateError(c.Request.Context(), id, req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + "已更新"})
}

// DeleteError 处理 DELETE /api/errors/:id
func DeleteError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_id", "ID格式错误")
		return
	}

	if err := service.DeleteError(c.Request.Context(), id); err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + " 已删除"})
}

// ReviewError 处理 PUT /api/errors/:id/review
func ReviewError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_id", "ID格式错误")
		return
	}

	var request models.ReviewRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			respondProblem(c, http.StatusBadRequest, "invalid_request", "请求格式错误")
			return
		}
	}
	item, err := service.ReviewError(c.Request.Context(), id, request.Rating)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "错题 #" + strconv.Itoa(id) + " 已标记复习",
		"next_review":  item.NextReview,
		"review_count": item.ReviewCount,
		"review_stage": item.ReviewStage,
	})
}

// GetTags 处理 GET /api/tags
func GetTags(c *gin.Context) {
	tags, err := service.GetAllTags(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
