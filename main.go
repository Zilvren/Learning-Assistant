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
	"study-tracker-go/internal/apierror"
	"study-tracker-go/internal/middleware"
	"study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	postgresrepo "study-tracker-go/internal/repository/postgres"
	"study-tracker-go/internal/service"
	"study-tracker-go/pkg/config"
	"study-tracker-go/pkg/logger"
)

// main 按“配置 → 存储 → 服务容器 → 路由 → 监听”的顺序组装应用；任一步失败都会阻止服务带着错误配置启动。
func main() {
	cfg := config.Load(os.Args[1:])
	log := logger.New()
	if err := cfg.Validate(); err != nil {
		log.Errorf("invalid configuration: %v", err)
		os.Exit(1)
	}
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	repository.SetDataDir(cfg.DataDir)
	repos, pool, cleanup, err := setupRepositories(cfg)
	if err != nil {
		log.Errorf("failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer cleanup()
	app, err := service.NewApp(cfg, repos, pool)
	if err != nil {
		log.Errorf("failed to initialize services: %v", err)
		os.Exit(1)
	}
	stopAutomaticBackup := service.StartAutomaticBackup(app)
	defer stopAutomaticBackup()
	if cfg.AuthEnabled && os.Getenv("TRACKER_JWT_SECRET") == "" {
		log.Infof("TRACKER_JWT_SECRET is empty; a stable local secret is stored in the data directory for this installation.")
	}

	listener, port, err := listenWithFallback(cfg.Host, cfg.Port)
	if err != nil {
		log.Errorf("failed to start server: %v", err)
		os.Exit(1)
	}

	r := gin.New()
	registerRoutes(r, app)

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

// setupRepositories 根据存储模式创建对应的 Repository 集合；PostgreSQL 模式还会建立连接池、执行迁移并准备本地导入用户。
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

// listenWithFallback 从首选端口开始尝试监听，开发环境中端口被占用时自动寻找后续可用端口。
var listenTCP = net.Listen

func listenWithFallback(host string, preferredPort int) (net.Listener, int, error) {
	const attempts = 20
	for offset := 0; offset < attempts; offset++ {
		port := preferredPort + offset
		if port > 65535 {
			break
		}
		address := fmt.Sprintf("%s:%d", host, port)
		listener, err := listenTCP("tcp", address)
		if err == nil {
			return listener, port, nil
		}
	}
	return nil, 0, fmt.Errorf("ports %d-%d are unavailable", preferredPort, preferredPort+attempts-1)
}

// registerRoutes 安装统一的请求上下文、审计、恢复与安全中间件，然后注册公开和受认证保护的 API。
func registerRoutes(r *gin.Engine, apps ...*service.App) {
	app := service.DefaultApp()
	if len(apps) > 0 && apps[0] != nil {
		app = apps[0]
	}
	if app == nil {
		panic("registerRoutes requires a service.App")
	}
	cfg := app.Config()
	requestLog := logger.New()
	r.Use(
		gin.Logger(),
		middleware.RequestContext(app),
		middleware.RequestAudit(requestLog),
		middleware.Recovery(requestLog),
		middleware.SecurityHeaders(),
		middleware.LocalCORS(),
		middleware.CookieOriginGuard(app),
	)
	//健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"storage": cfg.StorageDriver,
		})
	})

	//认证接口
	r.GET("/api/auth/status", handlers.AuthStatus)
	publicAuthLimit := middleware.RateLimit(10, time.Minute)
	r.POST("/api/auth/register", publicAuthLimit, handlers.Register)
	r.POST("/api/auth/verify-email", publicAuthLimit, handlers.VerifyEmail)
	r.POST("/api/auth/resend-verification", publicAuthLimit, handlers.ResendEmailVerification)
	r.POST("/api/auth/login", publicAuthLimit, handlers.Login)
	r.POST("/api/auth/refresh", handlers.Refresh)
	r.POST("/api/auth/logout", handlers.Logout)

	//公开更新接口
	r.GET("/api/version", handlers.GetVersion)
	r.GET("/api/update/check", handlers.CheckUpdate)

	api := r.Group("/api")
	api.Use(middleware.AuthRequired(app))
	{
		api.GET("/auth/me", handlers.Me)

		//科目接口
		api.GET("/subjects", handlers.GetSubjects)
		api.POST("/subjects", handlers.Addsubject)
		api.DELETE("/subjects/:name", handlers.DeleteSubject)

		//错题接口
		api.GET("/errors", handlers.GetErrors)
		api.POST("/errors", handlers.CreateError)
		api.GET("/errors/:id", handlers.GetError)
		api.PUT("/errors/:id", handlers.UpdateError)
		api.DELETE("/errors/:id", handlers.DeleteError)

		api.PUT("/errors/:id/review", handlers.ReviewError)
		api.GET("/tags", handlers.GetTags)

		//每日推送接口
		api.GET("/daily-push", handlers.GetDailyPush)
		api.GET("/dashboard/activity", handlers.GetLearningActivity)
		api.GET("/dashboard/plan", handlers.GetDailyPlan)
		api.PUT("/dashboard/plan", handlers.SetDailyGoal)
		api.POST("/dashboard/focus-sessions", handlers.RecordFocusSession)
		api.GET("/dashboard/weekly-report", handlers.GetWeeklyReport)

		//设置接口
		api.GET("/settings/token", handlers.GetToken)
		api.PUT("/settings/token", handlers.SetToken)
		api.DELETE("/settings/token", handlers.DeleteToken)
		api.GET("/settings/deepseek", handlers.GetDeepSeekToken)
		api.PUT("/settings/deepseek", handlers.SetDeepSeekToken)
		api.DELETE("/settings/deepseek", handlers.DeleteDeepSeekToken)
		api.PUT("/settings/deepseek/model", handlers.SetDeepSeekModel)
		api.PUT("/settings/username", handlers.SetUsername)

		// AI 学习助手：DeepSeek Key 保持在服务端，资料上下文按请求即时生成。
		api.POST("/ai/chat", middleware.RateLimit(20, time.Minute), handlers.AIChat)
		api.POST("/ai/edits/preview", middleware.RateLimit(10, time.Minute), handlers.PreviewAIEdit)
		api.POST("/ai/edits/apply", handlers.ApplyAIEdit)
		api.GET("/ai/conversation", handlers.GetAIConversation)
		api.PUT("/ai/conversation", handlers.SaveAIConversation)
		api.DELETE("/ai/conversation", handlers.ClearAIConversation)

		//备份接口
		api.GET("/backup/export", handlers.ExportBackup)
		api.POST("/backup/import", handlers.ImportBackup)

		//OCR接口
		api.POST("/ocr", handlers.OCRImage)
		api.GET("/ocr/tasks", handlers.ListOCRTasks)
		api.GET("/ocr/tasks/:id", handlers.GetOCRTask)
		api.POST("/ocr/tasks/:id/retry", handlers.RetryOCRTask)

		// 个人学习资料库
		api.GET("/library/items", handlers.ListLibraryItems)
		api.POST("/library/items", handlers.CreateLibraryItem)
		api.GET("/library/search", handlers.SearchLibrary)
		api.GET("/library/tags", handlers.ListLibraryTags)
		api.GET("/library/reviews", handlers.ListLibraryReviews)
		api.GET("/review/inbox", handlers.GetReviewInbox)
		api.GET("/search", handlers.SearchLearning)
		api.POST("/library/uploads", handlers.UploadLibraryFile)
		api.POST("/library/batch", handlers.BatchLibraryItems)
		api.GET("/library/items/:id", handlers.GetLibraryItem)
		api.PATCH("/library/items/:id", handlers.UpdateLibraryItem)
		api.DELETE("/library/items/:id", handlers.DeleteLibraryItem)
		api.GET("/library/items/:id/content", handlers.GetLibraryContent)
		api.GET("/library/items/:id/preview", handlers.GetLibraryPreview)
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

// openBrowserLater 让 HTTP 监听先完成，再异步唤起默认浏览器，避免浏览器抢在服务就绪前访问。
func openBrowserLater(url string) {
	go func() {
		time.Sleep(1200 * time.Millisecond)
		_ = openBrowser(url)
	}()
}

// openBrowser 根据当前操作系统调用对应命令打开 URL。
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
	// /api 开头但没匹配到路由 → 404
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		apierror.Write(c, http.StatusNotFound, "not_found", "接口不存在")
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
			setFrontendCacheHeaders(c, false)
			c.String(http.StatusNotFound, "frontend asset not found")
			return
		}

		// 文件不存在 → 回退到 index.html（Vue Router 是前端路由）
		data, err = fs.ReadFile(frontendFS, "frontend/dist/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "frontend/dist/index.html not found")
			return
		}
		setFrontendCacheHeaders(c, false)
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)

		return
	}

	contentType := mime.TypeByExtension(path.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	setFrontendCacheHeaders(c, strings.HasPrefix(requestPath, "assets/"))
	c.Data(http.StatusOK, contentType, data)
}

// isFrontendAssetRequest 判断不存在的路径是否应被视为静态资源，而不是 Vue Router 的页面路由。
func isFrontendAssetRequest(requestPath string) bool {
	return strings.HasPrefix(requestPath, "assets/") || path.Ext(requestPath) != ""
}

// setFrontendCacheHeaders 为带哈希的构建资源设置长期缓存，为 HTML 和路由回退禁用缓存。
func setFrontendCacheHeaders(c *gin.Context, immutable bool) {
	if immutable {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	c.Header("Cache-Control", "no-store, max-age=0")
	c.Header("Pragma", "no-cache")
}
