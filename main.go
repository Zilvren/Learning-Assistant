package main

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"study-tracker-go/api/handlers"
	"study-tracker-go/internal/middleware"
	"study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	postgresrepo "study-tracker-go/internal/repository/postgres"
	"study-tracker-go/internal/service"
	"study-tracker-go/pkg/config"
	"study-tracker-go/pkg/logger"
)

func main() {
	cfg := config.Load(os.Args[1:])
	log := logger.New()
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	repos, pool, cleanup, err := setupRepositories(cfg)
	if err != nil {
		log.Errorf("failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer cleanup()
	if err := service.InitApp(cfg, repos, pool); err != nil {
		log.Errorf("failed to initialize services: %v", err)
		os.Exit(1)
	}
	if cfg.AuthEnabled && os.Getenv("TRACKER_JWT_SECRET") == "" {
		log.Infof("TRACKER_JWT_SECRET is empty; using a temporary in-memory auth secret for this run.")
	}

	listener, port, err := listenWithFallback(cfg.Host, cfg.Port)
	if err != nil {
		log.Errorf("failed to start server: %v", err)
		os.Exit(1)
	}

	r := gin.Default()
	r.Use(middleware.SecurityHeaders(), middleware.LocalCORS(), middleware.CookieOriginGuard())
	registerRoutes(r)

	url := fmt.Sprintf("http://%s:%d/", cfg.Host, port)
	if port != cfg.Port {
		log.Infof("Port %d is occupied, using %d instead.", cfg.Port, port)
	}
	log.Infof("Study tracker is running at %s", url)

	if !cfg.NoBrowser {
		openBrowserLater(url)
	}

	if err := r.RunListener(listener); err != nil && err != http.ErrServerClosed {
		log.Errorf("server stopped: %v", err)
		os.Exit(1)
	}
}

func setupRepositories(cfg config.Config) (repository.Repositories, *pgxpool.Pool, func(), error) {
	switch cfg.StorageDriver {
	case "", "json":
		return jsonrepo.NewRepositories(), nil, func() {}, nil
	case "postgres":
		ctx := context.Background()
		pool, err := postgresrepo.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return repository.Repositories{}, nil, func() {}, err
		}
		userID, err := postgresrepo.EnsureLocalUser(ctx, pool)
		if err != nil {
			pool.Close()
			return repository.Repositories{}, nil, func() {}, err
		}
		return postgresrepo.NewRepositories(pool, userID), pool, pool.Close, nil
	default:
		return repository.Repositories{}, nil, func() {}, fmt.Errorf("未知存储类型：%s", cfg.StorageDriver)
	}
}

func listenWithFallback(host string, preferredPort int) (net.Listener, int, error) {
	const attempts = 20
	for offset := 0; offset < attempts; offset++ {
		port := preferredPort + offset
		if port > 65535 {
			break
		}
		address := fmt.Sprintf("%s:%d", host, port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			return listener, port, nil
		}
	}
	return nil, 0, fmt.Errorf("ports %d-%d are unavailable", preferredPort, preferredPort+attempts-1)
}

func registerRoutes(r *gin.Engine) {
	//健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	//认证接口
	r.GET("/api/auth/status", handlers.AuthStatus)
	r.POST("/api/auth/register", handlers.Register)
	r.POST("/api/auth/login", handlers.Login)
	r.POST("/api/auth/refresh", handlers.Refresh)
	r.POST("/api/auth/logout", handlers.Logout)

	//公开更新接口
	r.GET("/api/version", handlers.GetVersion)
	r.GET("/api/update/check", handlers.CheckUpdate)

	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		api.GET("/auth/me", handlers.Me)

		//科目接口
		api.GET("/subjects", handlers.GetSubjects)
		api.POST("/subjects", handlers.Addsubject)
		api.DELETE("/subjects/:name", handlers.DeleteSubject)

		//错题接口
		api.GET("/errors", handlers.GetErrors)
		api.POST("/errors", handlers.CreateError)
		api.PUT("/errors/:id", handlers.UpdateError)
		api.DELETE("/errors/:id", handlers.DeleteError)

		api.PUT("/errors/:id/review", handlers.ReviewError)
		api.GET("/tags", handlers.GetTags)

		//每日推送接口
		api.GET("/daily-push", handlers.GetDailyPush)

		//设置接口
		api.GET("/settings/token", handlers.GetToken)
		api.PUT("/settings/token", handlers.SetToken)
		api.DELETE("/settings/token", handlers.DeleteToken)
		api.PUT("/settings/username", handlers.SetUsername)

		//备份接口
		api.GET("/backup/export", handlers.ExportBackup)
		api.POST("/backup/import", handlers.ImportBackup)

		//OCR接口
		api.POST("/ocr", handlers.OCRImage)

		// 个人学习资料库
		api.GET("/library/items", handlers.ListLibraryItems)
		api.POST("/library/items", handlers.CreateLibraryItem)
		api.GET("/library/search", handlers.SearchLibrary)
		api.GET("/library/tags", handlers.ListLibraryTags)
		api.GET("/library/reviews", handlers.ListLibraryReviews)
		api.POST("/library/uploads", handlers.UploadLibraryFile)
		api.POST("/library/batch", handlers.BatchLibraryItems)
		api.GET("/library/items/:id", handlers.GetLibraryItem)
		api.PATCH("/library/items/:id", handlers.UpdateLibraryItem)
		api.DELETE("/library/items/:id", handlers.DeleteLibraryItem)
		api.GET("/library/items/:id/content", handlers.GetLibraryContent)
		api.PUT("/library/items/:id/content", handlers.SaveLibraryContent)
		api.POST("/library/items/:id/restore", handlers.RestoreLibraryItem)
		api.DELETE("/library/items/:id/purge", handlers.PurgeLibraryItem)
		api.POST("/library/items/:id/duplicate", handlers.DuplicateLibraryItem)
		api.POST("/library/items/:id/review", handlers.ReviewLibraryNote)
		api.GET("/library/items/:id/versions", handlers.ListLibraryVersions)
		api.POST("/library/items/:id/versions/:versionId/restore", handlers.RestoreLibraryVersion)

		//更新应用
		api.POST("/update/apply", handlers.ApplyUpdate)
	}

	//接口失败
	r.NoRoute(serveFrontend)
}

func openBrowserLater(url string) {
	go func() {
		time.Sleep(1200 * time.Millisecond)
		_ = openBrowser(url)
	}()
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// serveFrontend 处理前端页面请求
func serveFrontend(c *gin.Context) {
	setFrontendCacheHeaders(c)

	// /api 开头但没匹配到路由 → 404
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		c.JSON(http.StatusNotFound, gin.H{"detail": "接口不存在"})
		return
	}

	// 请求路径 → 文件路径
	requestPath := strings.TrimPrefix(c.Request.URL.Path, "/")
	if requestPath == "" {
		requestPath = "index.html"
	}

	filePath := path.Join("frontend/dist", requestPath)
	data, err := fs.ReadFile(frontendFS, filePath)
	if err != nil {
		if isFrontendAssetRequest(requestPath) {
			c.String(http.StatusNotFound, "frontend asset not found")
			return
		}

		// 文件不存在 → 回退到 index.html（Vue Router 是前端路由）
		data, err = fs.ReadFile(frontendFS, "frontend/dist/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "frontend/dist/index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)

		return
	}

	contentType := mime.TypeByExtension(path.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Data(http.StatusOK, contentType, data)
}

func isFrontendAssetRequest(requestPath string) bool {
	return strings.HasPrefix(requestPath, "assets/") || path.Ext(requestPath) != ""
}

func setFrontendCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, max-age=0")
	c.Header("Pragma", "no-cache")
}
