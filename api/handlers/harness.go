package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/service"
)

// HarnessTool 执行本地学习资料库插件暴露的一项操作。该端点实际上仅允许回环访问，并额外由每个对话的 Bearer 能力令牌保护。
func HarnessTool(c *gin.Context) {
	var args map[string]any
	if err := c.ShouldBindJSON(&args); err != nil {
		apierror.Write(c, http.StatusBadRequest, "invalid_harness_tool_request", "Harness 工具参数格式错误")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	result, err := service.ExecuteHarnessTool(c.Request.Context(), token, c.Param("tool"), args)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrHarnessToolUnauthorized):
			apierror.Write(c, http.StatusUnauthorized, "harness_unauthorized", err.Error())
		case errors.Is(err, service.ErrHarnessToolUnavailable):
			apierror.Write(c, http.StatusNotFound, "harness_tool_not_found", "该 Harness 工具不可用")
		default:
			apierror.Write(c, http.StatusBadRequest, "harness_tool_failed", err.Error())
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}

// HarnessStatus 使浏览器只在所需本地 Harness 运行时已安装且就绪时启用 AI 对话。
func HarnessStatus(c *gin.Context) {
	c.JSON(http.StatusOK, service.HarnessRuntimeStatus(c.Request.Context()))
}
