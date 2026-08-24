package router

import (
	"github.com/zxjyyh/mds/internal/config"
	"github.com/zxjyyh/mds/internal/handler"
	"github.com/zxjyyh/mds/internal/middleware"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func Setup(db *sqlx.DB, cfg *config.Config, log *zap.Logger) *gin.Engine {
	r := gin.New()

	// 默认不信任任何反向代理提供的 X-Forwarded-For / X-Real-IP。
	// 否则任何客户端都能伪造 ClientIP，直接绕过登录限流。
	// 如果确实部署在可信反代之后，用 TRUSTED_PROXIES=172.16.0.0/12 显式声明。
	if err := r.SetTrustedProxies(parseTrustedProxies()); err != nil {
		log.Warn("SetTrustedProxies failed, falling back to no trusted proxy", zap.Error(err))
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(gin.Recovery(), middleware.Logger(log))
	r.Use(middleware.CORS(), middleware.SecurityHeaders())

	h := handler.NewHandler(db, cfg, log)

	// ---- Static files ----
	r.GET("/favicon.svg", func(c *gin.Context) {
		c.File(filepath.Join(cfg.FrontendDir, "favicon.svg"))
	})
	r.Static("/assets", filepath.Join(cfg.FrontendDir, "assets"))

	// ---- Health ----
	r.GET("/health", h.HealthCheck)

	api := r.Group("/api")

	// Auth
	auth := api.Group("/auth")
	auth.POST("/login", middleware.LoginRateLimit(), h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.GET("/profile", middleware.Auth(db, cfg.JWTSecret), h.GetProfile)
	auth.PUT("/password", middleware.Auth(db, cfg.JWTSecret), h.ChangePassword)
	auth.POST("/logout", middleware.Auth(db, cfg.JWTSecret), h.Logout)
	auth.POST("/file-token", middleware.Auth(db, cfg.JWTSecret), h.IssueFileToken)

	// Files
	// 预览/下载走 AuthWithQuery：浏览器的 <img src> 和 window.open 无法携带
	// Authorization 头，只能用 ?token= 传专用文件令牌（typ=file，30 分钟有效）。
	files := api.Group("/files")
	files.POST("/upload", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin, h.UploadFile)
	files.GET("/:id/preview", middleware.AuthWithQuery(db, cfg.JWTSecret), h.PreviewFile)
	files.GET("/:id/download", middleware.AuthWithQuery(db, cfg.JWTSecret), h.DownloadFile)
	files.DELETE("/:id", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin, h.DeleteFile)

	// Users (all admin only)
	users := api.Group("/users", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	users.GET("", h.ListUsers)
	users.GET("/:id", h.GetUser)
	users.POST("", h.CreateUser)
	users.PUT("/:id", h.UpdateUser)
	users.DELETE("/:id", h.DeleteUser)
	users.PUT("/:id/password", h.ResetPassword)
	users.PATCH("/:id/status", h.SetStatus)

	// Personnel
	personnel := api.Group("/personnel", middleware.Auth(db, cfg.JWTSecret))
	personnel.GET("", h.ListPersonnel)
	personnel.GET("/options", h.PersonnelOptions)
	personnel.POST("/:id/certificates", middleware.RequireAdmin, h.AddPersonnelCert)
	personnel.PUT("/certificates/:certId", middleware.RequireAdmin, h.UpdatePersonnelCert)
	personnel.DELETE("/certificates/:certId", middleware.RequireAdmin, h.DeletePersonnelCert)
	personnel.GET("/:id", h.GetPersonnel)
	personnel.POST("", middleware.RequireAdmin, h.CreatePersonnel)
	personnel.PUT("/:id", middleware.RequireAdmin, h.UpdatePersonnel)
	personnel.DELETE("/:id", middleware.RequireAdmin, h.DeletePersonnel)

	// Equipment
	equipment := api.Group("/equipment", middleware.Auth(db, cfg.JWTSecret))
	equipment.GET("", h.ListEquipment)
	equipment.GET("/options", h.EquipmentOptions)
	equipment.POST("/:id/certificates", middleware.RequireAdmin, h.AddEquipmentCert)
	equipment.PUT("/certificates/:certId", middleware.RequireAdmin, h.UpdateEquipmentCert)
	equipment.DELETE("/certificates/:certId", middleware.RequireAdmin, h.DeleteEquipmentCert)
	equipment.GET("/:id", h.GetEquipment)
	equipment.POST("", middleware.RequireAdmin, h.CreateEquipment)
	equipment.PUT("/:id", middleware.RequireAdmin, h.UpdateEquipment)
	equipment.DELETE("/:id", middleware.RequireAdmin, h.DeleteEquipment)

	// Categories
	categories := api.Group("/categories", middleware.Auth(db, cfg.JWTSecret))
	categories.GET("", h.GetCategoryTree)
	categories.POST("", middleware.RequireAdmin, h.CreateCategory)
	categories.PUT("/:id", middleware.RequireAdmin, h.UpdateCategory)
	categories.DELETE("/:id", middleware.RequireAdmin, h.DeleteCategory)

	// Documents
	documents := api.Group("/documents", middleware.Auth(db, cfg.JWTSecret))
	documents.GET("", h.ListDocuments)
	documents.GET("/:id", h.GetDocument)
	documents.POST("", middleware.RequireAdmin, h.CreateDocument)
	documents.PUT("/:id", middleware.RequireAdmin, h.UpdateDocument)
	documents.DELETE("/:id", middleware.RequireAdmin, h.DeleteDocument)

	// Dashboard
	dashboard := api.Group("/dashboard", middleware.Auth(db, cfg.JWTSecret))
	dashboard.GET("/stats", h.DashboardStats)
	dashboard.GET("/expiring", h.DashboardExpiringList)
	dashboard.GET("/ignored", h.ListIgnoredWarnings)
	dashboard.POST("/ignore", h.IgnoreWarning)
	dashboard.DELETE("/ignored/:id", h.UnignoreWarning)

	// Reminders
	reminders := api.Group("/reminders", middleware.Auth(db, cfg.JWTSecret))
	reminders.GET("/rules", middleware.RequireAdmin, h.ListRules)
	reminders.POST("/rules", middleware.RequireAdmin, h.AddRule)
	reminders.PUT("/rules/:id", middleware.RequireAdmin, h.UpdateRule)
	reminders.DELETE("/rules/:id", middleware.RequireAdmin, h.DeleteRule)
	reminders.GET("/logs", h.GetLogs)
	reminders.POST("/test", middleware.RequireAdmin, h.TestRun)

	// Settings (admin only)
	settings := api.Group("/settings", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	settings.GET("/smtp", h.GetSMTPConfig)
	settings.PUT("/smtp", h.UpdateSMTPConfig)

	// ---- SPA fallback ----
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 404
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"code": -1, "message": "not found"})
			return
		}

		// Dev mode: proxy to Vite
		if cfg.Env == "development" {
			c.Redirect(http.StatusTemporaryRedirect, devFrontendURL()+path)
			return
		}

		// Production: serve index.html from frontend directory
		indexPath := filepath.Join(cfg.FrontendDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			return
		}

		// No frontend built
		c.String(http.StatusOK, "Frontend not built. Run: cd frontend && npm install && npm run build")
	})

	return r
}

// parseTrustedProxies 读取 TRUSTED_PROXIES（逗号分隔的 IP 或 CIDR）。
// 返回 nil 表示不信任任何代理，ClientIP 直接取 TCP 对端地址。
func parseTrustedProxies() []string {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}
	var list []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			list = append(list, s)
		}
	}
	return list
}

// devFrontendURL 开发模式下 Vite dev server 地址，可用 DEV_FRONTEND_URL 覆盖。
func devFrontendURL() string {
	if v := strings.TrimSpace(os.Getenv("DEV_FRONTEND_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8080"
}
