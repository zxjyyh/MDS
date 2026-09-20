package scheduler

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zxjyyh/mds/internal/service"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var (
	mu         sync.Mutex
	runner     *cron.Cron
	backupStop chan struct{}
)

// defaultSpec 每天 08:30 跑一次到期检查，可用 REMINDER_CRON 覆盖。
const defaultSpec = "30 8 * * *"

func Start(svc *service.ReminderService, log *zap.Logger) {
	mu.Lock()
	defer mu.Unlock()
	if runner != nil {
		return
	}

	spec := strings.TrimSpace(os.Getenv("REMINDER_CRON"))
	if spec == "" {
		spec = defaultSpec
	}

	// cron.New() 默认使用 time.Local；容器内由 TZ 环境变量决定，
	// 因此 compose 里必须设置 TZ，否则提醒会在 UTC 08:30（北京时间 16:30）触发。
	c := cron.New()
	if _, err := c.AddFunc(spec, func() {
		log.Info("Running scheduled expiry check")
		if err := svc.RunExpiryCheck(); err != nil {
			log.Error("Expiry check failed", zap.Error(err))
		}
	}); err != nil {
		log.Error("Failed to schedule cron", zap.String("spec", spec), zap.Error(err))
		return
	}

	c.Start()
	runner = c
	log.Info("Scheduler started", zap.String("spec", spec))
}

// Stop 等待正在执行的任务结束后停止调度器与备份轮询，供优雅关闭调用。
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if runner != nil {
		<-runner.Stop().Done()
		runner = nil
	}
	if backupStop != nil {
		close(backupStop)
		backupStop = nil
	}
}

// backupCheckInterval 多久检查一次「是否到了该备份的时间」。
//
// 用轮询而不是 cron 表达式：备份间隔是页面上随时可改的设置（小时级），
// 轮询方式改了立刻生效、不用重启服务；cron 需要重新注册规则。
// 代价是实际触发时间最多延后这个间隔，对定时备份无所谓。
const backupCheckInterval = 10 * time.Minute

// StartBackup 启动备份的定时检查（开关与间隔来自 app_settings）。
func StartBackup(svc *service.BackupService, log *zap.Logger) {
	mu.Lock()
	defer mu.Unlock()
	if backupStop != nil {
		return
	}
	stop := make(chan struct{})
	backupStop = stop

	go func() {
		// 启动后先等一会再查，别和迁移/启动流程抢资源
		select {
		case <-time.After(30 * time.Second):
		case <-stop:
			return
		}

		check := func() {
			if !svc.Due() {
				return
			}
			log.Info("Running scheduled backup")
			name, err := svc.RunAndRecord()
			if err != nil {
				log.Error("Scheduled backup failed", zap.Error(err))
				return
			}
			log.Info("Scheduled backup done", zap.String("file", name))
		}

		check()
		ticker := time.NewTicker(backupCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				check()
			case <-stop:
				return
			}
		}
	}()

	log.Info("Backup scheduler started", zap.Duration("checkInterval", backupCheckInterval))
}
