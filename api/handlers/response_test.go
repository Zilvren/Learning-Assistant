package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/repository"
)

// TestRespondErrorMapsDataBusyToServiceUnavailable 在HTTP 处理层中验证对应场景的行为与边界条件。
func TestRespondErrorMapsDataBusyToServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	respondError(context, http.StatusBadRequest, fmt.Errorf("wrapped: %w", repository.ErrDataBusy))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if recorder.Header().Get("Retry-After") != "1" {
		t.Fatal("missing Retry-After header")
	}
}
