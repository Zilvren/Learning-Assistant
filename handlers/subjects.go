package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/service"
)

func GetSubjects(c *gin.Context) {
	subjects, err := service.GetAllSubjects()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
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

	subjects, err := service.AddSubject(body.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}

// DeleteSubject 处理 DELETE /api/subjects/:name
func DeleteSubject(c *gin.Context) {
	name := c.Param("name") // 从 URL 路径中取参数
	subjects, err := service.DeleteSubject(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}
