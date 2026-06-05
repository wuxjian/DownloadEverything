package server

import (
	"database/sql"
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"download-everything/internal/config"
	"download-everything/internal/downloader"
	"download-everything/internal/service"

	"github.com/gin-gonic/gin"
)

// AppHandler 应用Handler聚合
type AppHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Manager *downloader.Manager
	DownSvc *service.DownloadService
	AISvc   *service.AIService
	Tmpl    *template.Template
}

// NewRouter 创建路由
func NewRouter(db *sql.DB, cfg *config.Config, mgr *downloader.Manager, webFS embed.FS) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 解析模板 (webFS路径: web/templates/xxx.html)
	tmpl := template.Must(template.ParseFS(webFS, "web/templates/*.html"))

	h := &AppHandler{
		DB:      db,
		Cfg:     cfg,
		Manager: mgr,
		DownSvc: service.NewDownloadService(mgr),
		AISvc:   service.NewAIService(cfg, db),
		Tmpl:    tmpl,
	}

	// 静态文件 (webFS路径: web/static/...)
	staticFS, _ := fs.Sub(webFS, "web/static")
	r.StaticFS("/static", http.FS(staticFS))

	// 页面路由
	r.GET("/", h.pageIndex)
	r.GET("/search", h.pageSearch)
	r.GET("/settings", h.pageSettings)

	// API - 任务管理
	api := r.Group("/api")
	{
		api.POST("/tasks", h.createTask)
		api.GET("/tasks", h.listTasks)
		api.GET("/tasks/:id", h.getTask)
		api.POST("/tasks/:id/pause", h.pauseTask)
		api.POST("/tasks/:id/resume", h.resumeTask)
		api.POST("/tasks/:id/retry", h.retryTask)
		api.DELETE("/tasks", h.clearTasks)
		api.DELETE("/tasks/:id", h.deleteTask)
		api.GET("/tasks/events", h.taskEvents)
		api.POST("/open-dir", h.openDownloadDir)
	}

	// API - AI搜索
	{
		api.POST("/ai/search", h.aiSearch)
		api.POST("/ai/parse-url", h.aiParseURL)
		api.POST("/ai/download", h.aiDownload)
	}

	// API - 配置
	{
		api.GET("/settings", h.getSettings)
		api.PUT("/settings", h.updateSettings)
	}

	return r
}

// --- 页面路由 ---

func (h *AppHandler) pageIndex(c *gin.Context) {
	h.Tmpl.ExecuteTemplate(c.Writer, "index.html", gin.H{
		"Page": "index",
	})
}

func (h *AppHandler) pageSearch(c *gin.Context) {
	h.Tmpl.ExecuteTemplate(c.Writer, "search.html", gin.H{
		"Page": "search",
	})
}

func (h *AppHandler) pageSettings(c *gin.Context) {
	h.Tmpl.ExecuteTemplate(c.Writer, "settings.html", gin.H{
		"Page": "settings",
		"Cfg":  h.Cfg,
	})
}