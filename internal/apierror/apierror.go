package apierror

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/repository"
)

const RequestIDKey = "request_id"

type Detail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Response keeps the legacy detail field for existing clients while exposing a
// stable machine-readable error object for new clients.
type Response struct {
	Detail    string `json:"detail"`
	Error     Detail `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// Write 在当前模块中创建或更新相应状态。
func Write(c *gin.Context, status int, code string, message string) {
	if code == "" {
		code = codeForStatus(status)
	}
	if message == "" {
		message = messageForStatus(status)
	}
	c.JSON(status, Response{
		Detail:    message,
		Error:     Detail{Code: code, Message: message},
		RequestID: RequestID(c),
	})
}

// FromError 在当前模块中完成本文件定义的局部处理。
func FromError(c *gin.Context, status int, err error) {
	if errors.Is(err, repository.ErrDataBusy) {
		c.Header("Retry-After", "1")
		Write(c, http.StatusServiceUnavailable, "data_busy", repository.ErrDataBusy.Error())
		return
	}
	if status >= http.StatusInternalServerError {
		Write(c, status, "internal_error", "服务暂时不可用，请稍后重试")
		return
	}
	message := "请求处理失败"
	if err != nil && err.Error() != "" {
		message = err.Error()
	}
	Write(c, status, codeForStatus(status), message)
}

// RequestID 在当前模块中完成本文件定义的局部处理。
func RequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetString(RequestIDKey)
}

// codeForStatus 在当前模块中完成本文件定义的局部处理。
func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusRequestEntityTooLarge:
		return "payload_too_large"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
			return "invalid_request"
		}
		return "internal_error"
	}
}

// messageForStatus 在当前模块中完成本文件定义的局部处理。
func messageForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "请求格式错误"
	case http.StatusUnauthorized:
		return "未登录"
	case http.StatusForbidden:
		return "无权访问"
	case http.StatusNotFound:
		return "资源不存在"
	case http.StatusConflict:
		return "资源状态冲突"
	case http.StatusRequestEntityTooLarge:
		return "请求数据过大"
	case http.StatusTooManyRequests:
		return "请求过于频繁，请稍后再试"
	case http.StatusServiceUnavailable:
		return "服务暂时不可用，请稍后重试"
	default:
		if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
			return "请求处理失败"
		}
		return "服务暂时不可用，请稍后重试"
	}
}
