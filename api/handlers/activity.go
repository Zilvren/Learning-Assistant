package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/service"
)

// GetLearningActivity 在HTTP 处理层中读取并整理所需数据。
func GetLearningActivity(c *gin.Context) {
	year := 0
	if rawYear := strings.TrimSpace(c.Query("year")); rawYear != "" {
		parsedYear, parseErr := strconv.Atoi(rawYear)
		if parseErr != nil {
			respondProblem(c, http.StatusBadRequest, "invalid_year", "年份格式错误")
			return
		}
		year = parsedYear
	}
	if year != 0 && (year < 2000 || year > time.Now().Year()) {
		respondProblem(c, http.StatusBadRequest, "invalid_year", "请选择有效的学习记录年份")
		return
	}
	result, err := service.GetLearningActivity(c.Request.Context(), year)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
