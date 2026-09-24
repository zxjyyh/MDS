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

	// 操作日志：全局注册（要包住路由上的 Auth 中间件才能拿到操作者），
	// 中间件内部先 c.Next() 让 Auth 执行，返回后 context 里才有 userId/username/userRole。
	r.Use(middleware.OperationLog(h.OperationLogSvc))

	// ---- Static files ----
	r.GET("/favicon.svg", func(c *gin.Context) {
		c.File(filepath.Join(cfg.FrontendDir, "favicon.svg"))
	})
	r.Static("/assets", filepath.Join(cfg.FrontendDir, "assets"))

	// ---- Health ----
	r.GET("/health", h.HealthCheck)

	api := r.Group("/api")

	// ---- 角色权限策略（2026-09-18 按用户要求调整）----
	// 普通用户与管理员**功能完全一致**，可以正常增删改业务数据；
	// 只有两个模块整体限管理员（菜单/路由/接口三层一起挡）：
	//   ① 公司信息管理 → /api/companies
	//   ② 系统管理     → /api/users、/api/settings、/api/operation-logs、/api/backups、
	//                     /api/reminders/rules（提醒设置）、/api/reminders/logs（提醒日志）、
	//                     /api/reminders/test（发送测试提醒）
	// 例外：「发送测试提醒」按钮在仪表盘上，但属于提醒设置这套系统功能，仍限管理员。
	// 例外：GET /api/companies/options（只读、仅 id/name/address）对所有登录用户开放 ——
	//       劳动合同弹窗要用它按甲方名称带出公司地址，而公司信息管理模块本身仍限管理员。

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
	files.POST("/upload", middleware.Auth(db, cfg.JWTSecret), h.UploadFile)
	files.GET("/:id/preview", middleware.AuthWithQuery(db, cfg.JWTSecret), h.PreviewFile)
	files.GET("/:id/download", middleware.AuthWithQuery(db, cfg.JWTSecret), h.DownloadFile)
	files.DELETE("/:id", middleware.Auth(db, cfg.JWTSecret), h.DeleteFile)

	// Users (all admin only)
	users := api.Group("/users", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	users.GET("", h.ListUsers)
	users.GET("/:id", h.GetUser)
	users.POST("", h.CreateUser)
	users.PUT("/:id", h.UpdateUser)
	users.DELETE("/:id", h.DeleteUser)
	users.PUT("/:id/password", h.ResetPassword)
	users.PATCH("/:id/status", h.SetStatus)
	// 账号可见公司（数据隔离）：覆盖式设置，传空数组 = 清空 = 恢复为不受限。
	// 仅管理员 —— 整组已挂 RequireAdmin。
	users.PUT("/:id/companies", h.SetUserCompanies)

	// Personnel（普通用户可读写）
	personnel := api.Group("/personnel", middleware.Auth(db, cfg.JWTSecret))
	personnel.GET("", h.ListPersonnel)
	personnel.GET("/options", h.PersonnelOptions)
	// 证件查漏：按检测站/在职状态 + 证书类型列出人员的证件状态
	personnel.GET("/cert-gap", h.PersonnelCertGap)
	personnel.GET("/certs/export", h.ExportPersonnelCerts)
	// 文档生成：劳动合同（占位符模板直出 docx）
	personnel.POST("/docs/contract", h.GenerateContract)
	// 理工人员报名材料：选项 + 生成前预检 + 整套材料 ZIP（汇总表 / 安全责任书 / 证件 / 劳动合同）
	personnel.GET("/docs/ligong-options", h.LigongOptions)
	personnel.POST("/docs/ligong-precheck", h.LigongPrecheck)
	personnel.POST("/docs/ligong-enroll", h.GenerateLigongEnroll)
	personnel.POST("/:id/certificates", h.AddPersonnelCert)
	personnel.PUT("/certificates/:certId", h.UpdatePersonnelCert)
	personnel.DELETE("/certificates/:certId", h.DeletePersonnelCert)
	// 固定预留证件行（安检证/环保证）的清空：保留行本身，清掉已填信息与已上传附件
	personnel.POST("/certificates/:certId/clear", h.ClearPersonnelCert)
	personnel.GET("/:id", h.GetPersonnel)
	personnel.POST("", h.CreatePersonnel)
	personnel.PUT("/:id", h.UpdatePersonnel)
	personnel.DELETE("/:id", h.DeletePersonnel)

	// Equipment（普通用户可读写）
	equipment := api.Group("/equipment", middleware.Auth(db, cfg.JWTSecret))
	equipment.GET("", h.ListEquipment)
	equipment.GET("/options", h.EquipmentOptions)
	equipment.POST("/:id/certificates", h.AddEquipmentCert)
	equipment.PUT("/certificates/:certId", h.UpdateEquipmentCert)
	equipment.DELETE("/certificates/:certId", h.DeleteEquipmentCert)
	equipment.GET("/:id", h.GetEquipment)
	equipment.POST("", h.CreateEquipment)
	equipment.PUT("/:id", h.UpdateEquipment)
	equipment.DELETE("/:id", h.DeleteEquipment)

	// 公司下拉选项：**对所有登录用户开放**。
	// 劳动合同弹窗（普通用户也能用）需要按「甲方名称」自动带出公司地址，而这个弹窗
	// 一进「人员材料」页面就已挂载、会立刻请求本接口 —— 若限管理员，普通用户一进页面
	// 就会看到「需要管理员权限」的报错。
	// 只暴露 id / name / address（不含信用代码与银行账号）；公司信息的增删改查仍整组限管理员。
	api.GET("/companies/options", middleware.Auth(db, cfg.JWTSecret), h.CompanyOptions)

	// 合法的公司名列表（公司档案 ∪ 人员/设备里出现过的检测站，去重）。
	// 对**所有登录用户**开放：人员/设备表单的「所属检测站」下拉要用它，
	// 而这两个模块普通用户可读写。只返回名称，不含任何公司敏感信息。
	api.GET("/companies/names", middleware.Auth(db, cfg.JWTSecret), h.CompanyNames)

	// Companies（公司信息 / 甲方主体资料）—— 其余接口限管理员：
	// 普通用户已在菜单与路由上隐藏该模块，接口层同步收紧做纵深防御
	companies := api.Group("/companies", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	companies.GET("", h.ListCompanies)
	companies.GET("/:id", h.GetCompany)
	companies.POST("", h.CreateCompany)
	companies.PUT("/:id", h.UpdateCompany)
	companies.DELETE("/:id", h.DeleteCompany)

	// Categories（资料库分类，普通用户可读写）
	categories := api.Group("/categories", middleware.Auth(db, cfg.JWTSecret))
	categories.GET("", h.GetCategoryTree)
	categories.POST("", h.CreateCategory)
	categories.PUT("/:id", h.UpdateCategory)
	categories.DELETE("/:id", h.DeleteCategory)

	// Documents（资料库文档，普通用户可读写）
	documents := api.Group("/documents", middleware.Auth(db, cfg.JWTSecret))
	documents.GET("", h.ListDocuments)
	documents.GET("/:id", h.GetDocument)
	documents.POST("", h.CreateDocument)
	documents.PUT("/:id", h.UpdateDocument)
	documents.DELETE("/:id", h.DeleteDocument)

	// Dashboard（普通用户可用；忽略预警属于仪表盘上的个人化操作，两种角色都可做）
	dashboard := api.Group("/dashboard", middleware.Auth(db, cfg.JWTSecret))
	dashboard.GET("/stats", h.DashboardStats)
	dashboard.GET("/expiring", h.DashboardExpiringList)
	dashboard.GET("/ignored", h.ListIgnoredWarnings)
	dashboard.POST("/ignore", h.IgnoreWarning)
	dashboard.DELETE("/ignored/:id", h.UnignoreWarning)

	// Reminders（提醒设置 / 提醒日志 / 测试发送 都属于「系统管理」，限管理员）
	reminders := api.Group("/reminders", middleware.Auth(db, cfg.JWTSecret))
	reminders.GET("/rules", middleware.RequireAdmin, h.ListRules)
	reminders.POST("/rules", middleware.RequireAdmin, h.AddRule)
	reminders.PUT("/rules/:id", middleware.RequireAdmin, h.UpdateRule)
	reminders.DELETE("/rules/:id", middleware.RequireAdmin, h.DeleteRule)
	reminders.GET("/logs", middleware.RequireAdmin, h.GetLogs)
	reminders.POST("/test", middleware.RequireAdmin, h.TestRun)

	// Settings (admin only)
	settings := api.Group("/settings", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	settings.GET("/smtp", h.GetSMTPConfig)
	settings.PUT("/smtp", h.UpdateSMTPConfig)

	// Operation logs（操作日志，仅管理员）：由全局 OperationLog 中间件自动写入
	opLogs := api.Group("/operation-logs", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	opLogs.GET("", h.ListOperationLogs)
	opLogs.DELETE("", h.ClearOperationLogs)

	// Backups（数据备份，仅管理员）
	backups := api.Group("/backups", middleware.Auth(db, cfg.JWTSecret), middleware.RequireAdmin)
	backups.GET("/settings", h.GetBackupSettings)
	backups.PUT("/settings", h.UpdateBackupSettings)
	backups.GET("", h.ListBackups)
	backups.POST("/run", h.RunBackupNow)
	backups.GET("/download", h.DownloadBackup)
	backups.DELETE("", h.DeleteBackup)

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
