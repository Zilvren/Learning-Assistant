package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	models "study-tracker-go/internal/model"
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

// GetDailyPlan 在 HTTP 处理层中完成当前请求的处理与响应。
func GetDailyPlan(c *gin.Context) {
	result, err := service.GetDailyPlan(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// SetDailyGoal 在 HTTP 处理层中完成当前请求的处理与响应。
func SetDailyGoal(c *gin.Context) {
	var goal models.DailyGoalSettings
	if err := c.ShouldBindJSON(&goal); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_goal", "目标格式错误")
		return
	}
	result, err := service.SetDailyGoal(c.Request.Context(), goal)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// RecordFocusSession 在 HTTP 处理层中完成当前请求的处理与响应。
func RecordFocusSession(c *gin.Context) {
	var body struct {
		Minutes   int    `json:"minutes"`
		ClientKey string `json:"client_key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondProblem(c, http.StatusBadRequest, "invalid_focus_session", "专注记录格式错误")
		return
	}
	result, err := service.RecordFocusSession(c.Request.Context(), body.Minutes, body.ClientKey)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// GetWeeklyReport 在 HTTP 处理层中完成当前请求的处理与响应。
func GetWeeklyReport(c *gin.Context) {
	result, err := service.GetWeeklyReport(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
