package main

import (
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"study-tracker-go/handlers"
)

func main() {
	r := gin.Default()

	//健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	//科目接口
	r.GET("/api/subjects", handlers.GetSubjects)
	r.POST("/api/subjects", handlers.Addsubject)
	r.DELETE("/api/subjects/:name", handlers.DeleteSubject)

	//错题接口
	r.GET("/api/errors", handlers.GetErrors)
	r.POST("/api/errors", handlers.CreateError)
	r.PUT("/api/errors/:id", handlers.UpdateError)
	r.DELETE("/api/errors/:id", handlers.DeleteError)

	r.PUT("/api/errors/:id/review", handlers.ReviewError)
	r.GET("/api/tags", handlers.GetTags)

	//每日推送接口
	r.GET("/api/daily-push", handlers.GetDailyPush)

	//设置接口
	r.GET("/api/settings/token", handlers.GetToken)
	r.PUT("/api/settings/token", handlers.SetToken)
	r.DELETE("/api/settings/token", handlers.DeleteToken)
	r.PUT("/api/settings/username", handlers.SetUsername)

	//备份接口】
	r.GET("/api/backup/export", handlers.ExportBackup)
	r.POST("/api/backup/import", handlers.ImportBackup)

	//OCR接口
	r.POST("/api/ocr", handlers.OCRImage)

	//更新接口
	r.GET("/api/version", handlers.GetVersion)
	r.GET("/api/update/check", handlers.CheckUpdate)
	r.POST("/api/update/apply", handlers.ApplyUpdate)

	//接口失败
	r.NoRoute(serveFrontend)

	if !hasArg("--no-browser") {
		openBrowserLater("http://127.0.0.1:8000/")
	}

	r.Run("127.0.0.1:8000")
}

func hasArg(name string) bool {
	for _, arg := range os.Args[1:] {
		if arg == name {
			return true
		}
	}
	return false
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
