package main

import (
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

func TestListenWithFallback(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()

	occupiedPort := occupied.Addr().(*net.TCPAddr).Port
	if occupiedPort > 65515 {
		t.Skipf("occupied port %d leaves too little room for fallback range", occupiedPort)
	}

	listener, port, err := listenWithFallback("127.0.0.1", occupiedPort)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	if port == occupiedPort {
		t.Fatalf("expected fallback port, got occupied port %d", port)
	}
	if port < occupiedPort || port >= occupiedPort+20 {
		t.Fatalf("fallback port %d is outside expected range", port)
	}
}

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
}

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

func TestFrontendEmbedIncludesViteUnderscoreChunks(t *testing.T) {
	entries, err := frontendFS.ReadDir("frontend/dist/assets")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "_plugin-vue_export-helper-") && strings.HasSuffix(entry.Name(), ".js") {
			return
		}
	}

	t.Fatal("embedded frontend is missing Vite's underscore-prefixed helper chunk")
}

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
