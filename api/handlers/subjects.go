package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

func GetSubjects(c *gin.Context) {
	subjects, err := service.GetAllSubjects(c.Request.Context())

	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}

func Addsubject(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}

	subjects, err := service.AddSubject(c.Request.Context(), body.Name)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}

// DeleteSubject 处理 DELETE /api/subjects/:name
func DeleteSubject(c *gin.Context) {
	name := c.Param("name") // 从 URL 路径中取参数
	subjects, err := service.DeleteSubject(c.Request.Context(), name)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}
