package service

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/zxjyyh/mds/internal/config"
	"github.com/zxjyyh/mds/internal/utils"
)

// BackupService 数据备份：把「数据库 + 上传附件」打包成一个 zip。
//
// 备份内容与目录结构（刻意与部署目录一致，恢复时直接解压覆盖 data/ 与 uploads/ 即可）：
//
//	data/data.db          数据库（用一致性快照导出，不是直接拷文件）
//	data/.secrets.env     JWT 密钥持久化文件（存在才有；漏了它恢复后所有人会被登出）
//	uploads/**            上传的附件
//
// 设置存 app_settings 表（与 SMTP 配置同样的做法）：启用开关 / 间隔小时 / 保留份数 / 上次执行时间。
type BackupService struct {
	DB  *sqlx.DB
	Cfg *config.Config
}

const (
	KeyBackupEnabled   = "backup.enabled"
	KeyBackupInterval  = "backup.interval_hours"
	KeyBackupRetention = "backup.retention"
	KeyBackupLastRunAt = "backup.last_run_at"

	defaultBackupInterval = 24 // 小时
	// 默认只留 5 份：实测本项目每份备份约等于 uploads 的体积（扫描件为主，压缩率很低，
	// 372MB 的上传目录压出来 342MB），10 份就是 3.4GB，对 NAS 太激进。
	defaultBackupRetention = 5 // 份

	// 与 config.secretsFileName 保持一致（那边未导出，这里跟着数据目录一起打包）
	backupSecretsName = ".secrets.env"
	backupNamePrefix  = "备份_"
)

// BackupSettings 备份设置（对外返回结构）。
type BackupSettings struct {
	Enabled       bool   `json:"enabled"`
	IntervalHours int    `json:"intervalHours"`
	Retention     int    `json:"retention"`
	LastRunAt     string `json:"lastRunAt"`
	NextRunAt     string `json:"nextRunAt"`
	Dir           string `json:"dir"`
}

// BackupFile 一个备份文件。
type BackupFile struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

// Dir 返回备份输出目录（相对路径按进程工作目录解析）。
func (s *BackupService) Dir() string {
	dir := strings.TrimSpace(s.Cfg.BackupDir)
	if dir == "" {
		dir = "./backups"
	}
	return dir
}

func (s *BackupService) LoadSettings() (BackupSettings, error) {
	out := BackupSettings{
		IntervalHours: defaultBackupInterval,
		Retention:     defaultBackupRetention,
		Dir:           s.Dir(),
	}
	if v, err := GetSetting(s.DB, KeyBackupEnabled); err == nil {
		out.Enabled = strings.TrimSpace(v) == "1"
	}
	if v, err := GetSetting(s.DB, KeyBackupInterval); err == nil {
		if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil && n > 0 {
			out.IntervalHours = n
		}
	}
	if v, err := GetSetting(s.DB, KeyBackupRetention); err == nil {
		if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil && n >= 1 {
			out.Retention = n
		}
	}
	if v, err := GetSetting(s.DB, KeyBackupLastRunAt); err == nil {
		out.LastRunAt = v
	}
	out.NextRunAt = s.nextRunAt(out)
	return out, nil
}

func (s *BackupService) SaveSettings(enabled bool, intervalHours, retention int) error {
	if intervalHours < 1 {
		intervalHours = defaultBackupInterval
	}
	if intervalHours > 24*30 { // 上限 30 天，防手滑填出个天文数字
		intervalHours = 24 * 30
	}
	if retention < 1 {
		retention = 1
	}
	if retention > 200 {
		retention = 200
	}
	e := "0"
	if enabled {
		e = "1"
	}
	return SetSettings(s.DB, map[string]string{
		KeyBackupEnabled:   e,
		KeyBackupInterval:  strconv.Itoa(intervalHours),
		KeyBackupRetention: strconv.Itoa(retention),
	})
}

func (s *BackupService) nextRunAt(st BackupSettings) string {
	if !st.Enabled || st.IntervalHours <= 0 {
		return ""
	}
	base := time.Now()
	if t, err := time.Parse(time.RFC3339, st.LastRunAt); err == nil {
		base = t
	}
	return base.Add(time.Duration(st.IntervalHours) * time.Hour).Format(time.RFC3339)
}

// Due 判断此刻是否该执行备份（供定时器轮询；间隔改了不用重启）。
// 从未备份过时返回 true —— 刚开启就能尽快看到第一份备份落盘。
func (s *BackupService) Due() bool {
	st, err := s.LoadSettings()
	if err != nil || !st.Enabled || st.IntervalHours <= 0 {
		return false
	}
	if strings.TrimSpace(st.LastRunAt) == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, st.LastRunAt)
	if err != nil {
		return true
	}
	return time.Since(last) >= time.Duration(st.IntervalHours)*time.Hour
}

// List 列出备份目录下的 zip（按文件名倒序 = 最新在前；文件名带零填充时间戳可直接比大小）。
func (s *BackupService) List() ([]BackupFile, error) {
	dir := s.Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupFile{}, nil // 还没备份过
		}
		return nil, err
	}
	out := []BackupFile{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".zip") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupFile{
			Name:      e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out, nil
}

// ResolvePath 校验并返回备份文件路径。只允许 backups 目录下的 .zip，防目录穿越。
func (s *BackupService) ResolvePath(name string) (string, error) {
	raw := strings.TrimSpace(name)
	base := filepath.Base(raw)
	if raw == "" || base != raw || !strings.HasSuffix(base, ".zip") {
		return "", errors.New("非法的备份文件名")
	}
	p := filepath.Join(s.Dir(), base)
	if _, err := os.Stat(p); err != nil {
		return "", errors.New("备份文件不存在")
	}
	return p, nil
}

func (s *BackupService) Remove(name string) error {
	p, err := s.ResolvePath(name)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

// RunAndRecord 完整跑一次备份：导出快照 → 打包 → 按保留份数清理旧的 → 记录执行时间。
func (s *BackupService) RunAndRecord() (string, error) {
	name, err := s.Run()
	if err != nil {
		return "", err
	}
	st, _ := s.LoadSettings()
	if n, perr := s.prune(st.Retention); perr != nil {
		log.Printf("WARN: 清理旧备份失败: %v", perr)
	} else if n > 0 {
		log.Printf("Backup retention: removed %d old file(s)", n)
	}
	if err := SetSettings(s.DB, map[string]string{KeyBackupLastRunAt: utils.NowString()}); err != nil {
		log.Printf("WARN: 记录备份执行时间失败: %v", err)
	}
	return name, nil
}

// Run 执行一次备份，返回生成的 zip 文件名。
func (s *BackupService) Run() (string, error) {
	dir := s.Dir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("创建备份目录失败：%w", err)
	}

	tmpDir, err := os.MkdirTemp("", "mds-backup-")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败：%w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1) 先把数据库导成一致快照。直接拷 .db 文件在 WAL 模式下可能拿到残缺副本。
	snap := filepath.Join(tmpDir, "data.db")
	if err := s.snapshotDB(snap); err != nil {
		return "", fmt.Errorf("导出数据库快照失败：%w", err)
	}

	name := backupNamePrefix + time.Now().Format("2006-01-02_150405") + ".zip"
	outPath := filepath.Join(dir, name)
	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("创建备份文件失败：%w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	if err := backupAddFile(zw, snap, "data/data.db"); err != nil {
		zw.Close()
		os.Remove(outPath)
		return "", err
	}

	// 2) 数据目录里除数据库本体与 WAL/SHM 之外的其它文件（.secrets.env 等）
	dbDir := filepath.Dir(s.Cfg.DBPath)
	dbBase := filepath.Base(s.Cfg.DBPath)
	if entries, err := os.ReadDir(dbDir); err == nil {
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || n == dbBase || n == dbBase+"-wal" || n == dbBase+"-shm" {
				continue
			}
			if err := backupAddFile(zw, filepath.Join(dbDir, n), "data/"+n); err != nil {
				zw.Close()
				os.Remove(outPath)
				return "", err
			}
		}
	}

	// 3) 上传附件
	if err := backupAddDir(zw, s.Cfg.UploadDir, "uploads"); err != nil {
		zw.Close()
		os.Remove(outPath)
		return "", err
	}

	if err := zw.Close(); err != nil {
		os.Remove(outPath)
		return "", fmt.Errorf("写入压缩包失败：%w", err)
	}
	log.Printf("Backup created: %s", outPath)
	return name, nil
}

// snapshotDB 导出一份一致的数据库快照。
// 首选 `VACUUM INTO`（SQLite 保证是一致快照，且不需要停写）；
// 万一当前驱动不支持，则回退「wal_checkpoint 把 WAL 合并回主库 + 拷文件」。
func (s *BackupService) snapshotDB(dst string) error {
	if _, err := s.DB.Exec("VACUUM INTO ?", dst); err == nil {
		return nil
	} else {
		log.Printf("WARN: VACUUM INTO 不可用（%v），回退 checkpoint + 拷贝", err)
		_ = os.Remove(dst)
	}
	if _, err := s.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("WARN: wal_checkpoint 失败: %v", err)
	}
	return backupCopyFile(s.Cfg.DBPath, dst)
}

// prune 只保留最近 retention 份，返回删除数量。
func (s *BackupService) prune(retention int) (int, error) {
	if retention <= 0 {
		return 0, nil
	}
	list, err := s.List()
	if err != nil {
		return 0, err
	}
	deleted := 0
	for i := retention; i < len(list); i++ {
		p, err := s.ResolvePath(list[i].Name)
		if err != nil {
			continue
		}
		if err := os.Remove(p); err != nil {
			continue
		}
		deleted++
	}
	return deleted, nil
}

// ---- 打包小工具（名字前缀避免与其它 service 里的同名函数冲突）----

func backupCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func backupAddFile(zw *zip.Writer, src, nameInZip string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(st)
	if err != nil {
		return err
	}
	hdr.Name = nameInZip
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

// backupAddDir 把目录下所有文件加进 zip，zip 内路径为 <prefix>/<相对路径>。
// 目录不存在直接跳过（例如还没上传过任何附件）。
func backupAddDir(zw *zip.Writer, base, prefix string) error {
	if base == "" {
		return nil
	}
	if _, err := os.Stat(base); err != nil {
		return nil
	}
	return filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		return backupAddFile(zw, p, filepath.ToSlash(filepath.Join(prefix, rel)))
	})
}
