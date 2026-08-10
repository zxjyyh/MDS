package service

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/zxjyyh/mds/internal/config"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/jmoiron/sqlx"
)

// 应用配置存放在 app_settings 表，而不是回写 .env 文件。
//
// 旧实现把 SMTP 配置写回 .env：
//   - 容器里 .env 位于镜像层，`docker compose up --force-recreate` 后配置全丢；
//   - 多个请求并发写同一个文件没有任何加锁，可能把文件写坏；
//   - 只读挂载（read_only: true）或非 root 运行时直接失败。
//
// 放进 SQLite 后，配置和业务数据在同一个卷里，备份/迁移只需要拷一个 .db 文件。

const (
	KeySMTPHost = "smtp.host"
	KeySMTPPort = "smtp.port"
	KeySMTPUser = "smtp.user"
	KeySMTPPass = "smtp.pass"
	KeySMTPFrom = "smtp.from"
)

// GetSetting 读取单个配置项，不存在返回空串。
func GetSetting(db *sqlx.DB, key string) (string, error) {
	var v string
	err := db.Get(&v, "SELECT value FROM app_settings WHERE key=?", key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetSettings 批量写入配置项（UPSERT）。
func SetSettings(db *sqlx.DB, kv map[string]string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := utils.NowString()
	for k, v := range kv {
		if _, err := tx.Exec(
			`INSERT INTO app_settings (key, value, updated_at) VALUES (?,?,?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
			k, v, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// LoadSMTPFromDB 用数据库中保存的值覆盖内存配置。
// 优先级：数据库 > 环境变量 / .env（环境变量只作为首次部署的初始值）。
// 在 main 中于 AutoMigrate 之后调用。
func LoadSMTPFromDB(db *sqlx.DB, cfg *config.Config) error {
	rows := []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}{}
	if err := db.Select(&rows, "SELECT key, value FROM app_settings WHERE key LIKE 'smtp.%'"); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	for _, r := range rows {
		switch r.Key {
		case KeySMTPHost:
			cfg.SMTP.Host = r.Value
		case KeySMTPUser:
			cfg.SMTP.User = r.Value
		case KeySMTPPass:
			cfg.SMTP.Pass = r.Value
		case KeySMTPFrom:
			cfg.SMTP.From = r.Value
		case KeySMTPPort:
			if n, err := strconv.Atoi(strings.TrimSpace(r.Value)); err == nil && n > 0 {
				cfg.SMTP.Port = n
			}
		}
	}
	return nil
}

// SaveSMTP 持久化 SMTP 配置并同步内存中的 cfg（email_svc 直接读 cfg）。
func SaveSMTP(db *sqlx.DB, cfg *config.Config, host string, port int, user, pass, from string) error {
	if port <= 0 || port > 65535 {
		port = 465
	}
	if err := SetSettings(db, map[string]string{
		KeySMTPHost: host,
		KeySMTPPort: strconv.Itoa(port),
		KeySMTPUser: user,
		KeySMTPPass: pass,
		KeySMTPFrom: from,
	}); err != nil {
		return err
	}

	cfg.SMTP.Host = host
	cfg.SMTP.Port = port
	cfg.SMTP.User = user
	cfg.SMTP.Pass = pass
	cfg.SMTP.From = from
	return nil
}
