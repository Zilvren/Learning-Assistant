package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/service"
)

// HarnessTool executes exactly one operation exposed by the local learning
// library plugin. The endpoint is loopback-only in practice and additionally
// guarded by a per-chat bearer capability.
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

// HarnessStatus lets the browser select the agent workflow only when its
// local runtime has been installed and explicitly enabled by the operator.
func HarnessStatus(c *gin.Context) {
	c.JSON(http.StatusOK, service.HarnessRuntimeStatus(c.Request.Context()))
}
