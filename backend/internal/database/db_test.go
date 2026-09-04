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

// TestRelaxEquipmentCertExpiry 回归 2026-08-21 的 NOT NULL 崩溃（报错 1299）：
// 旧库 equipment_certificates.expiry_date 是 NOT NULL，而前端不强制填到期日期，
// 后端又把空日期置为 nil 插入，于是撞约束报错。迁移须把该列改为可空，
// 且兼容「已存在、带 NOT NULL 的存量库」。
func TestRelaxEquipmentCertExpiry(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	// 删掉 AutoMigrate 建的可空版本，手工重建为「旧库」的 NOT NULL 结构
	db.MustExec(`DROP TABLE IF EXISTS equipment_certificates`)
	db.MustExec(`CREATE TABLE equipment_certificates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		equipment_id INTEGER NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
		cert_type TEXT NOT NULL,
		cert_number TEXT,
		issue_date TEXT,
		expiry_date TEXT NOT NULL,
		file_id INTEGER REFERENCES uploaded_files(id) ON DELETE SET NULL,
		remark TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	db.MustExec(`INSERT INTO equipment (id, name, serial_number) VALUES (1, '设备A', 'SN001')`)
	db.MustExec(`INSERT INTO equipment_certificates (equipment_id, cert_type, expiry_date)
		VALUES (1, '检测', '2026-12-31')`)

	if !columnIsNotNull(db, "equipment_certificates", "expiry_date") {
		t.Fatal("前置条件失败：模拟旧表 expiry_date 应为 NOT NULL")
	}

	if err := relaxEquipmentCertExpiry(db); err != nil {
		t.Fatalf("relaxEquipmentCertExpiry 失败: %v", err)
	}

	if columnIsNotNull(db, "equipment_certificates", "expiry_date") {
		t.Fatal("迁移后 expiry_date 仍标记为 NOT NULL")
	}

	// 关键：空/ NULL 的到期日期现在能插入（复现并修复 1299）
	if _, err := db.Exec(`INSERT INTO equipment_certificates
		(equipment_id, cert_type, expiry_date) VALUES (1, '合格证', NULL)`); err != nil {
		t.Fatalf("迁移后仍无法插入空到期日期: %v", err)
	}

	// 旧数据必须完好
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM equipment_certificates`); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("迁移后数据丢失，期望 2 行，实际 %d", n)
	}

	// 幂等：再跑一次不应报错，也不丢数据
	if err := relaxEquipmentCertExpiry(db); err != nil {
		t.Fatalf("重复运行迁移失败: %v", err)
	}
	var n2 int
	db.Get(&n2, `SELECT COUNT(*) FROM equipment_certificates`)
	if n2 != 2 {
		t.Fatalf("幂等运行后数据异常，期望 2 行，实际 %d", n2)
	}
}

// TestSeedPersonnelReservedCerts 验证每位人员固定预留「安检证/换保证」两行的 seed：
// 1. 无证件的人员补齐 2 行，且名称之外字段为空；
// 2. 已存在同名证件（用户自己添加过的）不重复插入；
// 3. 重复执行幂等，不会越补越多。
func TestSeedPersonnelReservedCerts(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	db.MustExec(`INSERT INTO personnel (id, name) VALUES (1, '张三')`)
	db.MustExec(`INSERT INTO personnel (id, name) VALUES (2, '李四')`)
	// 李四已手工添加过一张「安检证」（历史自由添加场景），seed 不得再造一行
	db.MustExec(`INSERT INTO personnel_certificates (personnel_id, cert_name, cert_number)
		VALUES (2, '安检证', 'X-001')`)

	seedPersonnelReservedCerts(db)

	// 每人恰好一行安检证 + 一行换保证
	for _, pid := range []int{1, 2} {
		var rows []struct {
			CertName string `db:"cert_name"`
			FileID   *int   `db:"file_id"`
		}
		if err := db.Select(&rows, `SELECT cert_name, file_id FROM personnel_certificates WHERE personnel_id=?`, pid); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("人员 %d 期望 2 行预留证件，实际 %d 行", pid, len(rows))
		}
		for _, r := range rows {
			if r.FileID != nil {
				t.Fatalf("预留行不应带附件，人员 %d 的 %s file_id=%v", pid, r.CertName, *r.FileID)
			}
		}
	}

	// 幂等：重复 seed 不新增
	seedPersonnelReservedCerts(db)
	var total int
	if err := db.Get(&total, `SELECT COUNT(*) FROM personnel_certificates`); err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Fatalf("seed 不幂等，期望 4 行（2人×2），实际 %d", total)
	}

	// 李四的「安检证」应仍是原来那条（编号还在），未被覆盖
	var number *string
	if err := db.Get(&number, `SELECT cert_number FROM personnel_certificates WHERE personnel_id=2 AND cert_name='安检证'`); err != nil {
		t.Fatal(err)
	}
	if number == nil || *number != "X-001" {
		t.Fatalf("seed 破坏已有数据，cert_number=%v", number)
	}
}
