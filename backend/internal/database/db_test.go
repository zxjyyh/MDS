package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

// openTestDB 在临时目录里开一个全新库，测试结束自动清理。
func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewDB 失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestPragmasActuallyApplied 是本仓库最重要的一个回归测试。
//
// modernc.org/sqlite 对无法识别的 DSN 参数**静默忽略**，写错时程序照常启动，
// 但实际运行在 journal_mode=DELETE / busy_timeout=0 / 外键关闭的状态下。
// 一旦有人把 DSN 改回 mattn 风格（?_journal_mode=WAL&...），这里会立刻报错。
func TestPragmasActuallyApplied(t *testing.T) {
	db := openTestDB(t)

	st, err := VerifyPragmas(db)
	if err != nil {
		t.Fatalf("VerifyPragmas 失败: %v", err)
	}
	if !st.OK() {
		t.Fatalf("关键 pragma 未生效: %s", st)
	}
	if st.BusyTimeout < 10000 {
		t.Errorf("busy_timeout 期望 >=10000，实际 %d", st.BusyTimeout)
	}
}

// TestForeignKeysEnforced 验证外键约束真的在起作用，
// 否则所有 ON DELETE CASCADE / SET NULL 都是摆设。
func TestForeignKeysEnforced(t *testing.T) {
	db := openTestDB(t)
	db.MustExec(`CREATE TABLE fk_parent (id INTEGER PRIMARY KEY)`)
	db.MustExec(`CREATE TABLE fk_child (
		id INTEGER PRIMARY KEY,
		pid INTEGER REFERENCES fk_parent(id) ON DELETE CASCADE
	)`)

	if _, err := db.Exec(`INSERT INTO fk_child (pid) VALUES (999)`); err == nil {
		t.Fatal("外键未生效：插入了不存在的父行引用")
	}

	db.MustExec(`INSERT INTO fk_parent (id) VALUES (1)`)
	db.MustExec(`INSERT INTO fk_child (pid) VALUES (1)`)
	db.MustExec(`DELETE FROM fk_parent WHERE id=1`)

	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM fk_child`); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("ON DELETE CASCADE 未生效，仍剩 %d 行", n)
	}
}

func TestAutoMigrateIdempotent(t *testing.T) {
	db := openTestDB(t)
	for i := 0; i < 2; i++ {
		if err := AutoMigrate(db); err != nil {
			t.Fatalf("第 %d 次 AutoMigrate 失败: %v", i+1, err)
		}
	}
	if !tableExists(db, "users") {
		t.Fatal("users 表未创建")
	}
	if !columnExists(db, "users", "token_epoch") {
		t.Fatal("users.token_epoch 列缺失（令牌吊销依赖该列）")
	}
}

// TestNormalizeLegacyTimestamps 覆盖 UTC 遗留格式自愈：
// 建表时 DEFAULT (datetime('now')) 写入的是 UTC 且不带时区，
// 而 Go 侧写入的是 RFC3339 本地时间，混在一列里会导致排序和显示都错乱。
func TestNormalizeLegacyTimestamps(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	const legacy = "2026-01-02 03:04:05" // UTC
	modern := time.Date(2026, 3, 4, 5, 6, 7, 0, time.Local).Format(time.RFC3339)

	db.MustExec(`INSERT INTO users (username,password_hash,real_name,email,created_at,updated_at)
		VALUES ('legacy','x','旧数据','legacy@example.com',?,?)`, legacy, legacy)
	db.MustExec(`INSERT INTO users (username,password_hash,real_name,email,created_at,updated_at)
		VALUES ('modern','x','新数据','modern@example.com',?,?)`, modern, modern)

	n, err := NormalizeLegacyTimestamps(db)
	if err != nil {
		t.Fatalf("NormalizeLegacyTimestamps 失败: %v", err)
	}
	if n == 0 {
		t.Fatal("应至少改写 1 个遗留时间戳，实际为 0")
	}

	var got string
	if err := db.Get(&got, `SELECT created_at FROM users WHERE username='legacy'`); err != nil {
		t.Fatal(err)
	}
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("改写后的值不是合法 RFC3339: %q (%v)", got, err)
	}
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if !parsed.Equal(want) {
		t.Fatalf("时刻发生了偏移：期望 %s，实际 %s", want, parsed.UTC())
	}

	// 已是 RFC3339 的值不能被二次改写
	var untouched string
	if err := db.Get(&untouched, `SELECT created_at FROM users WHERE username='modern'`); err != nil {
		t.Fatal(err)
	}
	if untouched != modern {
		t.Fatalf("新格式数据被误改：%q → %q", modern, untouched)
	}

	// 幂等：再跑一次应当 0 改动
	again, err := NormalizeLegacyTimestamps(db)
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Fatalf("不幂等，第二次仍改写了 %d 行", again)
	}
}
