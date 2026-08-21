package main

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/internal/service"
	"study-tracker-go/pkg/config"
)

// TestListenWithFallback 在当前模块中验证对应场景的行为与边界条件。
func TestListenWithFallback(t *testing.T) {
	previousListen := listenTCP
	t.Cleanup(func() { listenTCP = previousListen })
	const occupiedPort = 52730
	attempts := []string{}
	listener := &testListener{address: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: occupiedPort + 1}}
	listenTCP = func(network, address string) (net.Listener, error) {
		attempts = append(attempts, network+":"+address)
		if len(attempts) == 1 {
			return nil, errors.New("address already in use")
		}
		return listener, nil
	}

	gotListener, port, err := listenWithFallback("127.0.0.1", occupiedPort)
	if err != nil {
		t.Fatal(err)
	}
	if gotListener != listener {
		t.Fatal("expected listener returned by the second bind attempt")
	}
	if port != occupiedPort+1 || len(attempts) != 2 {
		t.Fatalf("fallback port %d is outside expected range", port)
	}
}

type testListener struct{ address net.Addr }

// Accept 验证当前模块在相应场景下的行为与边界条件。
func (listener *testListener) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
// Close 验证当前模块在相应场景下的行为与边界条件。
func (listener *testListener) Close() error              { return nil }
// Addr 验证当前模块在相应场景下的行为与边界条件。
func (listener *testListener) Addr() net.Addr            { return listener.address }

// TestBusinessRoutesAuthPolicy 在当前模块中验证对应场景的行为与边界条件。
func TestBusinessRoutesAuthPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(gin.DebugMode) })

	if err := service.InitApp(config.Config{StorageDriver: "json", AuthEnabled: false}, jsonrepo.NewRepositories(), nil); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	registerRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("json mode should not require login")
	}

	if err := service.InitApp(config.Config{StorageDriver: "postgres", AuthEnabled: true, JWTSecret: "test-secret"}, jsonrepo.NewRepositories(), nil); err != nil {
		t.Fatal(err)
	}
	r = gin.New()
	registerRoutes(r)

	req = httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("postgres auth mode should require login, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("production mode should hide client updates, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing API route should return 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":{"code":"not_found"`) || w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("missing API route must use the standard error envelope: %s", w.Body.String())
	}
}

// TestFrontendMissingAssetDoesNotFallbackToHTML 在当前模块中验证对应场景的行为与边界条件。
func TestFrontendMissingAssetDoesNotFallbackToHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/assets/missing-chunk.js", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	serveFrontend(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("missing frontend asset should return 404, got %d", w.Code)
	}
	if contentType := w.Header().Get("Content-Type"); contentType == "text/html; charset=utf-8" {
		t.Fatalf("missing frontend asset should not return HTML content type")
	}
	if cacheControl := w.Header().Get("Cache-Control"); cacheControl != "no-store, max-age=0" {
		t.Fatalf("missing frontend asset should disable cache, got %q", cacheControl)
	}
}

// TestFrontendRouteFallsBackToIndex 在当前模块中验证对应场景的行为与边界条件。
func TestFrontendRouteFallsBackToIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/errors/4", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	serveFrontend(c)

	if w.Code != http.StatusOK {
		t.Fatalf("frontend route should return index.html, got %d", w.Code)
	}
	if contentType := w.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("frontend route should return HTML, got %q", contentType)
	}
	if cacheControl := w.Header().Get("Cache-Control"); cacheControl != "no-store, max-age=0" {
		t.Fatalf("frontend route should disable cache, got %q", cacheControl)
	}
}
