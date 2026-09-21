package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/zxjyyh/mds/internal/models"
)

func AutoMigrate(db *sqlx.DB) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS "users" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"username" TEXT NOT NULL UNIQUE,
			"password_hash" TEXT NOT NULL,
			"real_name" TEXT NOT NULL,
			"email" TEXT NOT NULL UNIQUE,
			"role" TEXT NOT NULL DEFAULT 'user',
			"status" TEXT NOT NULL DEFAULT 'active',
			"last_login_at" TEXT,
			"token_epoch" INTEGER NOT NULL DEFAULT 0,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		// 运行时可修改的应用配置（SMTP 等）。
		// 早期版本把 SMTP 写回 .env 文件，既有并发写入问题，也让容器重建后配置丢失。
		`CREATE TABLE IF NOT EXISTS "app_settings" (
			"key" TEXT PRIMARY KEY,
			"value" TEXT NOT NULL DEFAULT '',
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "uploaded_files" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"original_name" TEXT NOT NULL,
			"stored_name" TEXT NOT NULL,
			"mime_type" TEXT NOT NULL,
			"size" INTEGER NOT NULL DEFAULT 0,
			"storage_path" TEXT NOT NULL,
			"owner_type" TEXT,
			"owner_id" INTEGER,
			"uploaded_by" INTEGER NOT NULL REFERENCES "users"("id"),
			"created_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "personnel" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL,
			"gender" TEXT NOT NULL DEFAULT 'unknown',
			"id_card" TEXT,
			"phone" TEXT,
			"position" TEXT,
			"department" TEXT,
			"address" TEXT,
			"entry_date" TEXT,
			"education" TEXT,
			"school" TEXT,
			"major" TEXT,
			"driver_license" TEXT,
			"driver_license_expiry" TEXT,
			"id_card_expiry" TEXT,
			"graduation_date" TEXT,
			"security_cert_date" TEXT,
			"security_cert_number" TEXT,
			"env_cert_date" TEXT,
			"env_cert_number" TEXT,
			"inspection_date" TEXT,
			"inspection_cert_number" TEXT,
			"status" TEXT NOT NULL DEFAULT 'active',
			"photo_file_id" INTEGER REFERENCES "uploaded_files"("id") ON DELETE SET NULL,
			"remark" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "personnel_certificates" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"personnel_id" INTEGER NOT NULL REFERENCES "personnel"("id") ON DELETE CASCADE,
			"cert_name" TEXT NOT NULL,
			"cert_number" TEXT,
			"issue_date" TEXT,
			"expiry_date" TEXT,
			"file_id" INTEGER REFERENCES "uploaded_files"("id") ON DELETE SET NULL,
			"remark" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "equipment" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL,
			"serial_number" TEXT NOT NULL,
			"department" TEXT,
			"line" TEXT,
			"model" TEXT,
			"manufacturer" TEXT,
			"status" TEXT NOT NULL DEFAULT 'in_use',
			"remark" TEXT,
			"inspection_cert_number" TEXT,
			"inspection_date" TEXT,
			"inspection_expiry_date" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "equipment_certificates" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"equipment_id" INTEGER NOT NULL REFERENCES "equipment"("id") ON DELETE CASCADE,
			"cert_type" TEXT NOT NULL,
			"cert_number" TEXT,
			"issue_date" TEXT,
			"expiry_date" TEXT,
			"file_id" INTEGER REFERENCES "uploaded_files"("id") ON DELETE SET NULL,
			"remark" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		// 公司信息（甲方主体资料）：劳动合同、报名材料等对外文书的填充来源。
		`CREATE TABLE IF NOT EXISTS "companies" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL,
			"credit_code" TEXT,
			"qual_cert_no" TEXT,
			"qual_cert_expiry" TEXT,
			"address" TEXT,
			"phone" TEXT,
			"bank_account" TEXT,
			"bank_branch" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "doc_categories" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL,
			"parent_id" INTEGER REFERENCES "doc_categories"("id") ON DELETE SET NULL,
			"system_type" TEXT NOT NULL,
			"sort_order" INTEGER NOT NULL DEFAULT 0,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "documents" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"category_id" INTEGER NOT NULL REFERENCES "doc_categories"("id"),
			"system_type" TEXT NOT NULL,
			"title" TEXT NOT NULL,
			"doc_number" TEXT,
			"version" TEXT,
			"issue_date" TEXT,
			"expiry_date" TEXT,
			"description" TEXT,
			"company_name" TEXT,
			"unified_credit_code" TEXT,
			"legal_person" TEXT,
			"address" TEXT,
			"cert_number" TEXT,
			"file_id" INTEGER REFERENCES "uploaded_files"("id") ON DELETE SET NULL,
			"created_by" INTEGER REFERENCES "users"("id"),
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "reminder_rules" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"module" TEXT NOT NULL,
			"days_before" INTEGER NOT NULL DEFAULT 30,
			"enabled" INTEGER NOT NULL DEFAULT 1,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "reminder_logs" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"module" TEXT NOT NULL,
			"target_name" TEXT,
			"expiry_date" TEXT,
			"days_before" INTEGER,
			"status" TEXT NOT NULL DEFAULT 'sent',
			"recipients" TEXT,
			"error_msg" TEXT,
			"sent_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"created_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "operation_logs" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"user_id" INTEGER REFERENCES "users"("id"),
			"username" TEXT,
			"role" TEXT,
			"action" TEXT NOT NULL,
			"method" TEXT,
			"path" TEXT,
			"target_type" TEXT,
			"target_id" INTEGER,
			"detail" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS "ignored_warnings" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"module" TEXT NOT NULL,
			"target_name" TEXT NOT NULL,
			"related_name" TEXT NOT NULL,
			"expiry_date" TEXT,
			"ignored_by" INTEGER NOT NULL REFERENCES "users"("id"),
			"created_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
	}

	for _, stmt := range ddl {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("Migration error: %v\nSQL: %s", err, stmt)
			return err
		}
	}

	// Upgrade: add education/school/major columns for existing databases
	upgrades := []string{
		`ALTER TABLE personnel ADD COLUMN education TEXT`,
		`ALTER TABLE personnel ADD COLUMN school TEXT`,
		`ALTER TABLE personnel ADD COLUMN major TEXT`,
		`ALTER TABLE personnel ADD COLUMN driver_license TEXT`,
		// Equipment schema upgrade — ignore errors on existing columns
		`ALTER TABLE equipment ADD COLUMN department TEXT`,
		`ALTER TABLE equipment ADD COLUMN line TEXT`,
		// Equipment: rename code to serial_number (SQLite 3.25+)
		`ALTER TABLE equipment RENAME COLUMN code TO serial_number`,
		// Personnel: new cert date fields
		`ALTER TABLE personnel ADD COLUMN security_cert_date TEXT`,
		`ALTER TABLE personnel ADD COLUMN env_cert_date TEXT`,
		`ALTER TABLE personnel ADD COLUMN inspection_date TEXT`,
		`ALTER TABLE personnel ADD COLUMN security_cert_number TEXT`,
		`ALTER TABLE personnel ADD COLUMN env_cert_number TEXT`,
		`ALTER TABLE personnel ADD COLUMN inspection_cert_number TEXT`,
		`ALTER TABLE personnel ADD COLUMN driver_license_expiry TEXT`,
		`ALTER TABLE personnel ADD COLUMN id_card_expiry TEXT`,
		`ALTER TABLE personnel ADD COLUMN graduation_date TEXT`,
		`ALTER TABLE personnel ADD COLUMN address TEXT`,
		// Equipment: standalone cert fields
		`ALTER TABLE equipment ADD COLUMN inspection_cert_number TEXT`,
		`ALTER TABLE equipment ADD COLUMN inspection_date TEXT`,
		`ALTER TABLE equipment ADD COLUMN inspection_expiry_date TEXT`,
		// Personnel: internal auditor cert
		`ALTER TABLE personnel ADD COLUMN internal_auditor_number TEXT`,
		`ALTER TABLE personnel ADD COLUMN internal_auditor_date TEXT`,
		// Users: 令牌世代号。改密 / 重置 / 禁用 / 登出时自增，用于即时吊销已签发的 JWT。
		`ALTER TABLE users ADD COLUMN token_epoch INTEGER NOT NULL DEFAULT 0`,
		// 公司档案（资料库）扩展字段：公司名称 / 统一信用代码 / 法人 / 地址 / 证书编号
		`ALTER TABLE documents ADD COLUMN company_name TEXT`,
		`ALTER TABLE documents ADD COLUMN unified_credit_code TEXT`,
		`ALTER TABLE documents ADD COLUMN legal_person TEXT`,
		`ALTER TABLE documents ADD COLUMN address TEXT`,
		`ALTER TABLE documents ADD COLUMN cert_number TEXT`,
		// 公司信息：公司地址（劳动合同甲方地址自动带出的来源）。
		// 已建过 companies 表的库必须靠这条 ALTER 补列（CREATE TABLE IF NOT EXISTS 不会补）。
		`ALTER TABLE companies ADD COLUMN address TEXT`,
		// 操作日志：补记操作者快照与原始请求。已建过表的库必须靠 ALTER 补列。
		// username / role 冗余存一份：删用户时 user_svc 会把 user_id 置 NULL 以保留审计记录，
		// 那时仍需能看出"是谁做的"。
		`ALTER TABLE operation_logs ADD COLUMN username TEXT`,
		`ALTER TABLE operation_logs ADD COLUMN role TEXT`,
		`ALTER TABLE operation_logs ADD COLUMN method TEXT`,
		`ALTER TABLE operation_logs ADD COLUMN path TEXT`,
	}
	for _, stmt := range upgrades {
		db.Exec(stmt) // ignore error (column may already exist or already renamed)
	}

	// 存量库兼容：equipment_certificates.expiry_date 早期被定义为 NOT NULL，
	// 但后端插入逻辑（空日期置 nil）与 personnel_certificates（可空）不一致，
	// 导致不填到期日期时插入 NULL 触发 SQLITE_CONSTRAINT_NOTNULL(1299)。
	// SQLite 不支持 ALTER COLUMN DROP NOT NULL，只能重建表，故用事务化迁移。
	if err := relaxEquipmentCertExpiry(db); err != nil {
		log.Printf("WARN: 放宽 equipment_certificates.expiry_date 约束失败: %v", err)
	}

	// 预置「资料库」模块的默认分类（公司档案 / 法规标准），幂等。
	seedDocCategories(db)

	// 预置每位人员的固定预留证件（安检证 / 环保证），幂等。
	// 仅补插缺项的"空证件行"，用户已存在同名证件时跳过，不重复插入。
	// 先收敛早期"换保证"笔误写入的行，再按统一名称补缺。
	renameLegacyPersonnelReservedCert(db)
	seedPersonnelReservedCerts(db)

	// 学历枚举收敛为「中专 / 高中 / 大专 / 本科」四档，幂等。
	normalizeEducationValues(db)

	log.Println("Database migration completed successfully")
	return nil
}

// 预留证件历史名称笔误修复："换保证" → "环保证"（同音笔误，幂等，每次启动执行）。
// 早期版本把第二个预留证件误写为"换保证"，统一术语后在此收敛存量行。
func renameLegacyPersonnelReservedCert(db *sqlx.DB) {
	if _, err := db.Exec(`UPDATE personnel_certificates SET cert_name = '环保证' WHERE cert_name = '换保证'`); err != nil {
		log.Printf("WARN: 修正历史预留证件名称(换保证→环保证)失败: %v", err)
	}
}

// 学历枚举收敛（2026-09-21，幂等，每次启动执行）。
//
// 学历原先只有三档「高中/中专」「大专」「本科及以上」，现改为四档
// 「中专」「高中」「大专」「本科」，前两档语义重复（高中与中专混在一格里），
// 且「本科及以上」无法区分本科/硕士。存量按下列规则收敛：
//
//	高中/中专  → 高中
//	大专       → 大专（不变）
//	本科及以上 → 本科
//
// 收敛后不再有匹配行，重复执行无副作用。
// 空值保持为空：理工报名时会要求手工选择，不在这里臆造。
func normalizeEducationValues(db *sqlx.DB) {
	for _, m := range []struct{ from, to string }{
		{"高中/中专", "高中"},
		{"本科及以上", "本科"},
	} {
		res, err := db.Exec(`UPDATE personnel SET education = ? WHERE education = ?`, m.to, m.from)
		if err != nil {
			log.Printf("WARN: 学历枚举收敛(%s→%s)失败: %v", m.from, m.to, err)
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			log.Printf("学历枚举收敛：%s → %s，影响 %d 条", m.from, m.to, n)
		}
	}
}

// seedPersonnelReservedCerts 为每位人员补齐两个固定预留证件行（安检证 / 环保证）。
// 预留行只有固定名称，编号/到期日期/附件等字段保持 NULL，等操作者上传证书后再填写。
// 幂等：仅当该人员不存在同名证件时插入，可随每次启动重复执行。
func seedPersonnelReservedCerts(db *sqlx.DB) {
	for _, name := range models.PersonnelReservedCertNames {
		_, err := db.Exec(`INSERT INTO personnel_certificates (personnel_id, cert_name)
			SELECT p.id, ? FROM personnel p
			WHERE NOT EXISTS (
				SELECT 1 FROM personnel_certificates pc
				WHERE pc.personnel_id = p.id AND pc.cert_name = ?
			)`, name, name)
		if err != nil {
			log.Printf("WARN: 预置人员预留证件 %q 失败: %v", name, err)
		}
	}
}

// seedDocCategories 幂等预置「资料库」模块的默认分类：
//   - company_archive（公司档案）：营业执照、资质证书、法人身份证、公司合同、章程内资、整改材料、相关通知
//   - legal_standard（法规标准）：公安专网标准、环保专网标准
//
// 仅当对应 system_type 下不存在同名顶级分类时才插入，可重复运行；
// 用户后续删除/改名分类后不会被强制还原。
func seedDocCategories(db *sqlx.DB) {
	presets := []struct {
		systemType string
		names      []string
	}{
		{"company_archive", []string{"营业执照", "资质证书", "法人身份证", "公司合同", "章程内资", "整改材料", "相关通知"}},
		{"legal_standard", []string{"公安专网标准", "环保专网标准"}},
	}
	for _, p := range presets {
		for i, name := range p.names {
			var count int
			if err := db.Get(&count, "SELECT COUNT(*) FROM doc_categories WHERE system_type=? AND name=? AND parent_id IS NULL", p.systemType, name); err != nil {
				continue
			}
			if count > 0 {
				continue
			}
			db.Exec("INSERT INTO doc_categories (name, system_type, sort_order) VALUES (?,?,?)", name, p.systemType, i+1)
		}
	}
}

// relaxEquipmentCertExpiry 把 equipment_certificates.expiry_date 从 NOT NULL 改为可空。
// 通过 PRAGMA table_info 探测当前约束，已是可空则直接返回（幂等，可重复运行）。
func relaxEquipmentCertExpiry(db *sqlx.DB) error {
	if !columnIsNotNull(db, "equipment_certificates", "expiry_date") {
		return nil // 新建库或已修复的库：无需处理
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	stmts := []string{
		`CREATE TABLE equipment_certificates_new (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"equipment_id" INTEGER NOT NULL REFERENCES "equipment"("id") ON DELETE CASCADE,
			"cert_type" TEXT NOT NULL,
			"cert_number" TEXT,
			"issue_date" TEXT,
			"expiry_date" TEXT,
			"file_id" INTEGER REFERENCES "uploaded_files"("id") ON DELETE SET NULL,
			"remark" TEXT,
			"created_at" TEXT NOT NULL DEFAULT (datetime('now')),
			"updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`INSERT INTO equipment_certificates_new (id, equipment_id, cert_type, cert_number, issue_date, expiry_date, file_id, remark, created_at, updated_at)
			SELECT id, equipment_id, cert_type, cert_number, issue_date, expiry_date, file_id, remark, created_at, updated_at
			FROM equipment_certificates`,
		`DROP TABLE equipment_certificates`,
		`ALTER TABLE equipment_certificates_new RENAME TO equipment_certificates`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			tx.Rollback()
			return fmt.Errorf("relax equipment_certificates.expiry_date: %w", err)
		}
	}
	return tx.Commit()
}

// columnIsNotNull 通过 PRAGMA table_info 判断某列当前是否被标记为 NOT NULL。
func columnIsNotNull(db *sqlx.DB, table, col string) bool {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == col {
			return notnull == 1
		}
	}
	return false
}

// legacyTimestampLayout 是 SQLite `DEFAULT (datetime('now'))` 产生的格式，且为 UTC。
const legacyTimestampLayout = "2006-01-02 15:04:05"

// timestampColumns 列出所有存放"时刻"（而非"日期"）的列。
// 注意不要把 issue_date / expiry_date 这类纯日期列放进来。
var timestampColumns = map[string][]string{
	"users":                  {"created_at", "updated_at", "last_login_at"},
	"app_settings":           {"updated_at"},
	"uploaded_files":         {"created_at"},
	"personnel":              {"created_at", "updated_at"},
	"personnel_certificates": {"created_at", "updated_at"},
	"equipment":              {"created_at", "updated_at"},
	"equipment_certificates": {"created_at", "updated_at"},
	"doc_categories":         {"created_at"},
	"documents":              {"created_at", "updated_at"},
	"companies":              {"created_at", "updated_at"},
	"reminder_rules":         {"created_at", "updated_at"},
	"reminder_logs":          {"created_at", "sent_at"},
	"operation_logs":         {"created_at"},
	"ignored_warnings":       {"created_at"},
}

// NormalizeLegacyTimestamps 把历史遗留的 "2006-01-02 15:04:05"（UTC）时间戳
// 统一改写为 RFC3339 本地时间，例如 "2026-08-10T11:02:00+08:00"。
//
// 为什么需要：建表时用了 `DEFAULT (datetime('now'))`，SQLite 会写入 **UTC** 且不带时区标记；
// 而 Go 代码写入时用的是 `time.RFC3339` 本地时间。两种格式混在同一列里会造成两个可见 Bug：
//  1. 列表按 created_at 字符串排序时，"2026-08-10 03:02:00" 恒小于 "2026-08-10T11:02:00+08:00"
//     （空格 0x20 < 'T' 0x54），新旧数据交错，排序结果错乱；
//  2. 前端按 ISO 解析 UTC 字符串会当成本地时间，界面上时间少 8 小时。
//
// 该函数幂等，每次启动都跑一遍，把新混入的脏数据一并收敛。
func NormalizeLegacyTimestamps(db *sqlx.DB) (int, error) {
	total := 0

	for table, cols := range timestampColumns {
		if !tableExists(db, table) {
			continue
		}
		// 各表主键名不一致（多数表为 id，app_settings 为 key），
		// 必须按表动态取主键列，否则 SELECT id 在 app_settings 上直接报 no such column。
		pk := primaryKeyOf(db, table)
		if pk == "" {
			continue
		}
		for _, col := range cols {
			if !columnExists(db, table, col) {
				continue
			}

			type row struct {
				PK  int    `db:"pk"`
				Val string `db:"val"`
			}
			var rows []row
			// GLOB 区分大小写且支持 [0-9] 字符类，比 LIKE 更精确地锁定遗留格式
			q := fmt.Sprintf(
				`SELECT %s AS pk, %s AS val FROM %s WHERE %s IS NOT NULL AND %s GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]'`,
				pk, col, table, col, col)
			if err := db.Select(&rows, q); err != nil {
				return total, fmt.Errorf("scan %s.%s: %w", table, col, err)
			}
			if len(rows) == 0 {
				continue
			}

			tx, err := db.Beginx()
			if err != nil {
				return total, err
			}
			upd := fmt.Sprintf("UPDATE %s SET %s=? WHERE %s=?", table, col, pk)
			for _, r := range rows {
				t, perr := time.ParseInLocation(legacyTimestampLayout, r.Val, time.UTC)
				if perr != nil {
					continue
				}
				if _, err := tx.Exec(upd, t.Local().Format(time.RFC3339), r.PK); err != nil {
					tx.Rollback()
					return total, fmt.Errorf("update %s.%s pk=%d: %w", table, col, r.PK, err)
				}
				total++
			}
			if err := tx.Commit(); err != nil {
				return total, err
			}
		}
	}

	if total > 0 {
		log.Printf("Normalized %d legacy UTC timestamps to RFC3339 local time", total)
	}
	return total, nil
}

func tableExists(db *sqlx.DB, table string) bool {
	var n int
	err := db.Get(&n, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
	return err == nil && n > 0
}

// primaryKeyOf 返回表的主键列名（取第一个 pk 列）。
// 用于 NormalizeLegacyTimestamps 动态拼 SELECT/UPDATE，避免对 app_settings（主键为 key）硬编码 id。
func primaryKeyOf(db *sqlx.DB, table string) string {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid         int
			name, ctype string
			notnull, pk int
			dflt        sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return ""
		}
		if pk > 0 {
			return name
		}
	}
	return ""
}

func columnExists(db *sqlx.DB, table, col string) bool {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid         int
			name, ctype string
			notnull, pk int
			dflt        sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == col {
			return true
		}
	}
	return false
}
