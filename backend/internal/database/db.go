package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// NewDB 打开 SQLite 连接。
//
// 重要：本项目使用 modernc.org/sqlite（纯 Go 实现），它的 DSN 语法与常见的
// mattn/go-sqlite3 **完全不同**：
//
//	mattn   : ?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON
//	modernc : ?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)
//
// 而且 modernc 对**无法识别的 DSN 参数会静默忽略，不会报错**。因此一旦写成
// mattn 风格，程序照常启动，但实际运行在 journal_mode=DELETE、busy_timeout=0、
// 外键关闭的状态下 —— 在 NAS 这类慢盘上会频繁出现 "database is locked"，
// 同时所有 ON DELETE CASCADE 都形同虚设。
//
// 唯一可靠的验证手段是启动后回读实际值，见 VerifyPragmas。
func NewDB(dbPath string) (*sqlx.DB, error) {
	params := strings.Join([]string{
		"_pragma=journal_mode(WAL)",   // 读写不互相阻塞，慢盘必需
		"_pragma=busy_timeout(10000)", // 默认为 0：一遇锁立即失败
		"_pragma=foreign_keys(1)",     // 让 ON DELETE CASCADE / SET NULL 真正生效
		"_pragma=synchronous(NORMAL)", // WAL 模式下的推荐值，兼顾安全与性能
		"_txlock=immediate",           // 事务开始即取写锁，避免升级锁导致的 SQLITE_BUSY
	}, "&")

	db, err := sqlx.Open("sqlite", dbPath+"?"+params)
	if err != nil {
		return nil, err
	}

	// SQLite 是单写者模型，连接数开大并不会提升写入吞吐，反而增加锁竞争。
	// 注意：业务代码中禁止"嵌套游标"（外层 rows 未 Close 时又发起新查询），
	// 否则并发请求会各占一条连接等待第二条，直接把池子锁死。
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return db, nil
}

// PragmaState 是启动时回读到的实际 pragma 值。
type PragmaState struct {
	JournalMode string
	BusyTimeout int
	ForeignKeys int
	Synchronous int
}

func (p PragmaState) String() string {
	return fmt.Sprintf("journal_mode=%s busy_timeout=%d foreign_keys=%d synchronous=%d",
		p.JournalMode, p.BusyTimeout, p.ForeignKeys, p.Synchronous)
}

// OK 表示关键 pragma 均已按预期生效。
func (p PragmaState) OK() bool {
	return strings.EqualFold(p.JournalMode, "wal") && p.BusyTimeout > 0 && p.ForeignKeys == 1
}

// VerifyPragmas 回读实际生效的 pragma。
// 因为驱动会静默忽略写错的 DSN 参数，这是唯一能发现配置失效的办法。
func VerifyPragmas(db *sqlx.DB) (PragmaState, error) {
	var st PragmaState
	if err := db.Get(&st.JournalMode, "PRAGMA journal_mode"); err != nil {
		return st, fmt.Errorf("读取 journal_mode 失败: %w", err)
	}
	if err := db.Get(&st.BusyTimeout, "PRAGMA busy_timeout"); err != nil {
		return st, fmt.Errorf("读取 busy_timeout 失败: %w", err)
	}
	if err := db.Get(&st.ForeignKeys, "PRAGMA foreign_keys"); err != nil {
		return st, fmt.Errorf("读取 foreign_keys 失败: %w", err)
	}
	if err := db.Get(&st.Synchronous, "PRAGMA synchronous"); err != nil {
		return st, fmt.Errorf("读取 synchronous 失败: %w", err)
	}
	return st, nil
}
