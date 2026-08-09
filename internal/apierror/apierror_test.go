package apierror

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestWriteUsesCompatibleAndStructuredErrorEnvelope 在当前模块中验证对应场景的行为与边界条件。
func TestWriteUsesCompatibleAndStructuredErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(RequestIDKey, "request-123")

	Write(ctx, http.StatusBadRequest, "invalid_name", "名称不合法")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Detail != "名称不合法" || response.Error.Code != "invalid_name" || response.Error.Message != "名称不合法" || response.RequestID != "request-123" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

// TestFromErrorDoesNotExposeInternalFailure 在当前模块中验证对应场景的行为与边界条件。
func TestFromErrorDoesNotExposeInternalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	FromError(ctx, http.StatusInternalServerError, errors.New("database password should not be exposed"))

	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "internal_error" || response.Detail == "database password should not be exposed" {
		t.Fatalf("internal error leaked details: %#v", response)
	}
}
