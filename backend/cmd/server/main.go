package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/zxjyyh/mds/internal/config"
	"github.com/zxjyyh/mds/internal/database"
	"github.com/zxjyyh/mds/internal/router"
	"github.com/zxjyyh/mds/internal/scheduler"
	"github.com/zxjyyh/mds/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// appVersion 可在构建时注入：
//
//	go build -ldflags "-X main.appVersion=$(git describe --tags --always)"
var appVersion = "2.0.0"

func main() {
	// Production mode: suppress debug route listing
	gin.SetMode(gin.ReleaseMode)

	// Logger
	log := initLogger()
	defer log.Sync()

	// Config（只读：程序不会再往 .env 写任何东西）
	cfg := config.Load()

	// Ensure required directories（按配置建，而不是写死相对路径）
	mustMkdir(log, filepath.Dir(cfg.DBPath), cfg.UploadDir, logDir())

	// Database
	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatal("数据库连接失败", zap.Error(err))
	}
	defer db.Close()

	// 回读实际生效的 PRAGMA。
	// modernc.org/sqlite 对 DSN 里不认识的参数是"静默忽略"，写错一个字母不会报错，
	// 只会在半年后以 "database is locked" 的形式炸出来。所以这里必须显式确认。
	if st, err := database.VerifyPragmas(db); err != nil {
		log.Warn("PRAGMA 回读失败", zap.Error(err))
	} else {
		fields := []zap.Field{
			zap.String("journal_mode", st.JournalMode),
			zap.Int("busy_timeout", st.BusyTimeout),
			zap.Int("foreign_keys", st.ForeignKeys),
			zap.Int("synchronous", st.Synchronous),
		}
		if st.OK() {
			log.Info("SQLite PRAGMA 已生效", fields...)
		} else {
			log.Warn("SQLite PRAGMA 未按预期生效，高并发下可能出现锁等待或外键失效", fields...)
		}
	}

	// Migrate
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal("数据库迁移失败", zap.Error(err))
	}

	// 历史 UTC 时间戳自愈（幂等）
	if n, err := database.NormalizeLegacyTimestamps(db); err != nil {
		log.Warn("时间戳规范化失败", zap.Error(err))
	} else if n > 0 {
		log.Info("已规范化历史时间戳", zap.Int("rows", n))
	}

	// 历史文件存储路径自愈（兼容 Windows 绝对路径等，幂等）
	if n, err := service.NormalizeFilePaths(db, cfg.UploadDir); err != nil {
		log.Warn("文件路径规范化失败", zap.Error(err))
	} else if n > 0 {
		log.Info("已修正历史文件存储路径", zap.Int("rows", n))
	}

	// Seed
	if err := database.Seed(db); err != nil {
		log.Warn("种子数据初始化失败", zap.Error(err))
	}

	// Initialize service config
	service.InitConfig(cfg)

	// 数据库中保存的 SMTP 配置覆盖环境变量（环境变量仅作首次部署的初始值）
	if err := service.LoadSMTPFromDB(db, cfg); err != nil {
		log.Warn("读取 SMTP 配置失败，将使用环境变量中的值", zap.Error(err))
	}

	// Start scheduler
	reminderSvc := &service.ReminderService{DB: db}
	scheduler.Start(reminderSvc, log)

	// Setup router
	r := router.Setup(db, cfg, log)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Info("Server starting",
		zap.String("version", appVersion),
		zap.String("addr", addr),
		zap.String("env", cfg.Env),
		zap.String("db", cfg.DBPath),
	)
	printBanner(cfg)

	// 在独立 goroutine 里监听，主 goroutine 等待退出信号
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("服务启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭：docker stop 发送 SIGTERM，默认只给 10 秒。
	// 不处理的话进程会被 SIGKILL 硬杀，SQLite 的 WAL 可能来不及 checkpoint。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到退出信号，正在关闭服务…")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("优雅关闭超时，强制退出", zap.Error(err))
	}
	scheduler.Stop()
	log.Info("服务已停止")
}

func printBanner(cfg *config.Config) {
	fmt.Printf("\n  内业管理系统 v%s\n", appVersion)
	fmt.Printf("  %s 环境，端口 %d\n", cfg.Env, cfg.Port)
	if cfg.Env == "development" {
		fmt.Println("  [DEV] 前端请另开终端运行: cd frontend && npm run dev")
		fmt.Printf("  [DEV] 前端地址 %s\n", devFrontendHint())
	} else {
		fmt.Printf("  访问 http://localhost:%d\n", cfg.Port)
	}
	fmt.Printf("  Ctrl+C 停止服务\n\n")
}

func devFrontendHint() string {
	if v := os.Getenv("DEV_FRONTEND_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func mustMkdir(log *zap.Logger, dirs ...string) {
	for _, d := range dirs {
		if d == "" || d == "." {
			continue
		}
		if err := os.MkdirAll(d, 0o750); err != nil {
			log.Fatal("无法创建目录", zap.String("dir", d), zap.Error(err))
		}
	}
}

func logDir() string {
	if v := os.Getenv("LOG_DIR"); v != "" {
		return v
	}
	return "logs"
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func initLogger() *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	dir := logDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		// 日志目录建不出来就只写 stdout，不阻断启动
		return zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			zap.InfoLevel,
		))
	}

	// 日志轮转：NAS 上跑一年不做轮转，app.log 能把系统盘撑爆。
	rotator := &lumberjack.Logger{
		Filename:   filepath.Join(dir, "app.log"),
		MaxSize:    envInt("LOG_MAX_SIZE_MB", 10), // 单文件 MB
		MaxBackups: envInt("LOG_MAX_BACKUPS", 5),  // 保留份数
		MaxAge:     envInt("LOG_MAX_AGE_DAYS", 30),
		Compress:   true,
		LocalTime:  true,
	}

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			zap.InfoLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(rotator),
			zap.InfoLevel,
		),
	)
	return zap.New(core)
}
