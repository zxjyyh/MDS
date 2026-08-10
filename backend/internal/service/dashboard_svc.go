package service

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/zxjyyh/mds/internal/models"

	"github.com/jmoiron/sqlx"
)

// ---- SQL 标识符安全校验 ----
//
// 下面这批统计函数通过 fmt.Sprintf 拼接表名/列名（SQL 占位符无法用于标识符）。
// 目前所有调用方传入的都是硬编码常量，不存在注入路径；但这类函数极易在后续迭代中
// 被人接上 c.Query(...)，所以在函数内部做一次强制校验，把安全性钉死在被调用方，
// 而不是依赖调用方自觉。

var errUnsafeSQLIdent = errors.New("unsafe sql identifier")

// allowedStatTables 允许参与统计查询的表白名单。
var allowedStatTables = map[string]bool{
	"personnel":              true,
	"equipment":              true,
	"documents":              true,
	"personnel_certificates": true,
	"equipment_certificates": true,
}

// identRe 合法列名：小写字母/数字/下划线，且以字母或下划线开头。
var identRe = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// filterRe 合法过滤片段：只允许标识符、数字、单引号字符串以及 = != < > AND 等基本符号。
// 分号、注释、括号、逗号一律拒绝，从语法上排除堆叠查询与子查询。
var filterRe = regexp.MustCompile(`^[a-zA-Z0-9_ '=!<>]+$`)

var forbiddenInFilter = []string{";", "--", "/*", "*/", "union", "select", "insert", "update", "delete", "drop", "attach", "pragma"}

func checkIdent(table, col string) error {
	if !allowedStatTables[table] {
		return fmt.Errorf("%w: table %q", errUnsafeSQLIdent, table)
	}
	if col != "" && !identRe.MatchString(col) {
		return fmt.Errorf("%w: column %q", errUnsafeSQLIdent, col)
	}
	return nil
}

func checkFilter(filter string) error {
	if filter == "" {
		return fmt.Errorf("%w: empty filter", errUnsafeSQLIdent)
	}
	if !filterRe.MatchString(filter) {
		return fmt.Errorf("%w: filter %q", errUnsafeSQLIdent, filter)
	}
	lower := strings.ToLower(filter)
	for _, bad := range forbiddenInFilter {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("%w: filter contains %q", errUnsafeSQLIdent, bad)
		}
	}
	return nil
}

// ---- Stats count helpers ----

func CountExpiring(db *sqlx.DB, table, col string, days int) (int, error) {
	if err := checkIdent(table, col); err != nil {
		return 0, err
	}
	var c int
	now := time.Now().Format("2006-01-02")
	future := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND %s BETWEEN ? AND ?", table, col, col), now, future)
	return c, err
}

func CountExpiringRange(db *sqlx.DB, table, col string, from, to int) (int, error) {
	if err := checkIdent(table, col); err != nil {
		return 0, err
	}
	var c int
	now := time.Now()
	start := now.AddDate(0, 0, from).Format("2006-01-02")
	end := now.AddDate(0, 0, to).Format("2006-01-02")
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND %s != '' AND %s BETWEEN ? AND ?", table, col, col, col), start, end)
	return c, err
}

func CountExpired(db *sqlx.DB, table, col string) (int, error) {
	if err := checkIdent(table, col); err != nil {
		return 0, err
	}
	var c int
	now := time.Now().Format("2006-01-02")
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND %s != '' AND %s < ?", table, col, col, col), now)
	return c, err
}

func CountTotal(db *sqlx.DB, table string) (int, error) {
	if err := checkIdent(table, ""); err != nil {
		return 0, err
	}
	var c int
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	return c, err
}

// Filtered versions for documents (by system_type)

func CountExpiringFiltered(db *sqlx.DB, table, col, filter string, days int) (int, error) {
	if err := checkIdent(table, col); err != nil {
		return 0, err
	}
	if err := checkFilter(filter); err != nil {
		return 0, err
	}
	var c int
	now := time.Now().Format("2006-01-02")
	future := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s AND %s IS NOT NULL AND %s BETWEEN ? AND ?", table, filter, col, col), now, future)
	return c, err
}

func CountExpiringRangeFiltered(db *sqlx.DB, table, col, filter string, from, to int) (int, error) {
	if err := checkIdent(table, col); err != nil {
		return 0, err
	}
	if err := checkFilter(filter); err != nil {
		return 0, err
	}
	var c int
	now := time.Now()
	start := now.AddDate(0, 0, from).Format("2006-01-02")
	end := now.AddDate(0, 0, to).Format("2006-01-02")
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s AND %s IS NOT NULL AND %s != '' AND %s BETWEEN ? AND ?", table, filter, col, col, col), start, end)
	return c, err
}

func CountExpiredFiltered(db *sqlx.DB, table, col, filter string) (int, error) {
	if err := checkIdent(table, col); err != nil {
		return 0, err
	}
	if err := checkFilter(filter); err != nil {
		return 0, err
	}
	var c int
	now := time.Now().Format("2006-01-02")
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s AND %s IS NOT NULL AND %s != '' AND %s < ?", table, filter, col, col, col), now)
	return c, err
}

func CountTotalFiltered(db *sqlx.DB, table, filter string) (int, error) {
	if err := checkIdent(table, ""); err != nil {
		return 0, err
	}
	if err := checkFilter(filter); err != nil {
		return 0, err
	}
	var c int
	err := db.Get(&c, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, filter))
	return c, err
}

// ---- Expiring list for dashboard ----

type ExpiringItem struct {
	Module      string         `db:"module" json:"module"`
	TargetName  string         `db:"target_name" json:"targetName"`
	CertNumber  sql.NullString `db:"cert_number" json:"certNumber"`
	ExpiryDate  sql.NullString `db:"expiry_date" json:"expiryDate"`
	RelatedName sql.NullString `db:"related_name" json:"relatedName"`
	Department  sql.NullString `db:"department" json:"department"`
	FileID      sql.NullInt64  `db:"file_id" json:"fileId"`
}

// ToJSON converts ExpiringItem to a map for JSON response with proper nil handling
func (e ExpiringItem) ToJSON() map[string]interface{} {
	result := map[string]interface{}{
		"module":     e.Module,
		"targetName": e.TargetName,
	}

	if e.CertNumber.Valid {
		result["certNumber"] = e.CertNumber.String
	} else {
		result["certNumber"] = nil
	}

	if e.ExpiryDate.Valid {
		result["expiryDate"] = e.ExpiryDate.String
	} else {
		result["expiryDate"] = nil
	}

	if e.RelatedName.Valid {
		result["relatedName"] = e.RelatedName.String
	} else {
		result["relatedName"] = nil
	}

	if e.Department.Valid {
		result["department"] = e.Department.String
	} else {
		result["department"] = nil
	}

	if e.FileID.Valid {
		v := int(e.FileID.Int64)
		result["fileId"] = v
	} else {
		result["fileId"] = nil
	}

	return result
}

// ---- Ignored warnings ----

func AddIgnoredWarning(db *sqlx.DB, module, targetName, relatedName, expiryDate string, ignoredBy int) (*models.IgnoredWarning, error) {
	result, err := db.Exec(`INSERT INTO ignored_warnings (module, target_name, related_name, expiry_date, ignored_by) VALUES (?,?,?,?,?)`,
		module, targetName, relatedName, expiryDate, ignoredBy)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	var w models.IgnoredWarning
	err = db.Get(&w, "SELECT * FROM ignored_warnings WHERE id=?", id)
	return &w, err
}

func RemoveIgnoredWarning(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM ignored_warnings WHERE id=?", id)
	return err
}

func ListIgnoredWarnings(db *sqlx.DB) ([]models.IgnoredWarning, error) {
	var list []models.IgnoredWarning
	err := db.Select(&list, "SELECT * FROM ignored_warnings ORDER BY created_at DESC")
	return list, err
}

func CountIgnored(db *sqlx.DB) (int, error) {
	var c int
	err := db.Get(&c, "SELECT COUNT(*) FROM ignored_warnings")
	return c, err
}

// getIgnoredKeys returns a set of "module|target_name|related_name" keys to exclude
func getIgnoredKeys(db *sqlx.DB) (map[string]bool, error) {
	items, err := ListIgnoredWarnings(db)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]bool, len(items))
	for _, item := range items {
		k := item.Module + "|" + item.TargetName + "|" + item.RelatedName
		keys[k] = true
	}
	return keys, nil
}

// GetExpiringList now excludes ignored items
func GetExpiringList(db *sqlx.DB, module string, days, page, pageSize int) ([]map[string]interface{}, int, error) {
	now := time.Now().Format("2006-01-02")
	var future string
	if days == 0 {
		future = now
	} else {
		future = time.Now().AddDate(0, 0, days).Format("2006-01-02")
	}

	var sqlStr string
	var isUnion bool
	switch module {
	case "personnel_idcard":
		fields := []struct{ label, num, date string }{
			{"身份证", "p.id_card", "p.id_card_expiry"},
			{"驾驶证", "p.driver_license", "p.driver_license_expiry"},
		}
		var parts []string
		for _, f := range fields {
			sqlStr = fmt.Sprintf("SELECT 'personnel_idcard' as module,'%s' as target_name,%s as cert_number,%s as expiry_date,p.name as related_name,p.department as department,0 as file_id FROM personnel p WHERE %s IS NOT NULL AND %s != '' AND %s ",
				f.label, f.num, f.date, f.date, f.date, f.date)
			if days == 0 {
				sqlStr += fmt.Sprintf("< '%s'", now)
			} else {
				sqlStr += fmt.Sprintf("BETWEEN '%s' AND '%s'", now, future)
			}
			parts = append(parts, sqlStr)
		}
		sqlStr = strings.Join(parts, " UNION ALL ")
		isUnion = true
	case "personnel_license":
		fields := []struct{ label, num, date string }{
			{"安检证", "p.security_cert_number", "p.security_cert_date"},
			{"环保证", "p.env_cert_number", "p.env_cert_date"},
			{"查验证", "p.inspection_cert_number", "p.inspection_date"},
			{"内审员证", "p.internal_auditor_number", "p.internal_auditor_date"},
		}
		var parts []string
		for _, f := range fields {
			sqlStr = fmt.Sprintf("SELECT 'personnel_license' as module,'%s' as target_name,%s as cert_number,%s as expiry_date,p.name as related_name,p.department as department,0 as file_id FROM personnel p WHERE %s IS NOT NULL AND %s != '' AND %s ",
				f.label, f.num, f.date, f.date, f.date, f.date)
			if days == 0 {
				sqlStr += fmt.Sprintf("< '%s'", now)
			} else {
				sqlStr += fmt.Sprintf("BETWEEN '%s' AND '%s'", now, future)
			}
			parts = append(parts, sqlStr)
		}
		sqlStr = strings.Join(parts, " UNION ALL ")
		isUnion = true
	case "equipment_cert":
		sqlStr = `SELECT 'equipment_cert' as module, ec.cert_type as target_name, ec.cert_number, ec.expiry_date, e.name as related_name, ec.file_id
			FROM equipment_certificates ec LEFT JOIN equipment e ON e.id=ec.equipment_id
			WHERE ec.expiry_date IS NOT NULL AND ec.expiry_date != '' AND ec.expiry_date `
	case "qualification_doc":
		sqlStr = `SELECT 'qualification_doc' as module, d.title as target_name, d.doc_number as cert_number, d.expiry_date, '' as related_name, '' as department, d.file_id
			FROM documents d WHERE d.system_type='qualification' AND d.expiry_date IS NOT NULL AND d.expiry_date != '' AND d.expiry_date `
	case "safety_doc":
		sqlStr = `SELECT 'safety_doc' as module, d.title as target_name, d.doc_number as cert_number, d.expiry_date, '' as related_name, '' as department, d.file_id
			FROM documents d WHERE d.system_type='safety' AND d.expiry_date IS NOT NULL AND d.expiry_date != '' AND d.expiry_date `
	case "equipment_inspection":
		sqlStr = `SELECT 'equipment_inspection' as module, '检定有效期' as target_name, e.inspection_cert_number as cert_number, e.inspection_expiry_date as expiry_date, e.name as related_name, e.department as department, 0 as file_id
			FROM equipment e WHERE e.inspection_expiry_date IS NOT NULL AND e.inspection_expiry_date != '' AND e.inspection_expiry_date `
	default:
		return []map[string]interface{}{}, 0, nil
	}

	if !isUnion {
		if days == 0 {
			sqlStr += "< ?"
		} else {
			sqlStr += "BETWEEN ? AND ?"
		}
	}

	// Count total
	var total int
	countSQL := "SELECT COUNT(*) FROM (" + sqlStr + ") AS sub"
	if !isUnion {
		if days == 0 {
			if err := db.Get(&total, countSQL, now); err != nil {
				return nil, 0, err
			}
		} else {
			if err := db.Get(&total, countSQL, now, future); err != nil {
				return nil, 0, err
			}
		}
	} else {
		if err := db.Get(&total, countSQL); err != nil {
			return nil, 0, err
		}
	}

	// Paginated query
	offset := (page - 1) * pageSize
	if !isUnion {
		sqlStr += " ORDER BY expiry_date LIMIT ? OFFSET ?"
	} else {
		sqlStr = "SELECT * FROM (" + sqlStr + ") AS sub ORDER BY expiry_date LIMIT ? OFFSET ?"
	}
	var items []ExpiringItem
	if !isUnion {
		if days == 0 {
			if err := db.Select(&items, sqlStr, now, pageSize, offset); err != nil {
				return nil, 0, err
			}
		} else {
			if err := db.Select(&items, sqlStr, now, future, pageSize, offset); err != nil {
				return nil, 0, err
			}
		}
	} else {
		if err := db.Select(&items, sqlStr, pageSize, offset); err != nil {
			return nil, 0, err
		}
	}

	// Convert to JSON-friendly format, excluding ignored items
	ignored, _ := getIgnoredKeys(db)
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		relName := ""
		if item.RelatedName.Valid {
			relName = item.RelatedName.String
		}
		tgtName := item.TargetName
		key := item.Module + "|" + tgtName + "|" + relName
		if ignored[key] {
			total--
			continue
		}
		result = append(result, item.ToJSON())
	}
	return result, total, nil
}
