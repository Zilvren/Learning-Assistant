package handlers

import (
	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
)

// respondError 在HTTP 处理层中完成本文件定义的局部处理。
func respondError(c *gin.Context, status int, err error) {
	apierror.FromError(c, status, err)
}

// respondProblem 在HTTP 处理层中完成本文件定义的局部处理。
func respondProblem(c *gin.Context, status int, code string, message string) {
	apierror.Write(c, status, code, message)
}
