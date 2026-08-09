package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/apierror"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/internal/service"
	"study-tracker-go/pkg/config"
	"study-tracker-go/pkg/logger"
)

// TestRequestContextInjectsAppAndRequestID 在请求中间件中验证对应场景的行为与边界条件。
func TestRequestContextInjectsAppAndRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app, err := service.NewApp(config.Config{StorageDriver: "json"}, jsonrepo.NewRepositories(), nil)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(RequestContext(app))
	router.GET("/api/example", func(c *gin.Context) {
		resolved, ok := service.AppFromContext(c.Request.Context())
		if !ok || resolved != app {
			apierror.Write(c, http.StatusInternalServerError, "missing_app", "缺少应用依赖")
			return
		}
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/example", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id header was not set")
	}
}

// TestRecoveryReturnsStandardErrorEnvelope 在请求中间件中验证对应场景的行为与边界条件。
func TestRecoveryReturnsStandardErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestContext(nil), Recovery(logger.New()))
	router.GET("/api/panic", func(*gin.Context) { panic("boom") })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("recovery response must retain the request id")
	}
}

// TestShouldAuditOnlyTracksWritesAndFailures 在请求中间件中验证对应场景的行为与边界条件。
func TestShouldAuditOnlyTracksWritesAndFailures(t *testing.T) {
	if shouldAudit(http.MethodGet, "/api/health", http.StatusInternalServerError) {
		t.Fatal("health checks must not enter the audit log")
	}
	if shouldAudit(http.MethodGet, "/api/library/items", http.StatusOK) {
		t.Fatal("successful API reads must not enter the audit log")
	}
	if !shouldAudit(http.MethodPost, "/api/library/items", http.StatusCreated) {
		t.Fatal("API writes must enter the audit log")
	}
	if !shouldAudit(http.MethodGet, "/api/library/items", http.StatusBadRequest) {
		t.Fatal("failed API reads must enter the audit log")
	}
}
