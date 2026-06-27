package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"study-tracker-go/models"
	"study-tracker-go/service"
)

func CreateError(c *gin.Context) {
	var req models.AddErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
	}

	item, err := service.CreateError(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": item.ID, "message": "添加成功"})
}

// GetErrors 处理 GET /api/errors
func GetErrors(c *gin.Context) {
	errors, err := service.GetAllErrors(
		c.Query("subject"),    // ?subject=数学
		c.Query("keyword"),    // ?keyword=导数
		c.Query("tag"),        // ?tag=选择题
		c.Query("reason_tag"), // ?reason_tag=概念混淆
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	// 注意：前端读 res.errors 和 res.total
	c.JSON(http.StatusOK, gin.H{"errors": errors, "total": len(errors)})
}

func UpdateError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}

	var req models.UpdateErrorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
	}

	if err := service.UpdateError(id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + "已更新"})
}

// DeleteError 处理 DELETE /api/errors/:id
func DeleteError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}

	if err := service.DeleteError(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + " 已删除"})
}

// ReviewError 处理 PUT /api/errors/:id/review
func ReviewError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}

	item, err := service.ReviewError(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "错题 #" + strconv.Itoa(id) + " 已标记复习",
		"next_review":  item.NextReview,
		"review_count": item.ReviewCount,
	})
}

// GetTags 处理 GET /api/tags
func GetTags(c *gin.Context) {
	tags, err := service.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
