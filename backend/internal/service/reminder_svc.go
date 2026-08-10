package service

import (
	"fmt"
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type ReminderService struct{ DB *sqlx.DB }

func (s *ReminderService) ListRules() ([]models.ReminderRule, error) {
	var rules []models.ReminderRule
	err := s.DB.Select(&rules, "SELECT * FROM reminder_rules ORDER BY id")
	return rules, err
}

func (s *ReminderService) AddRule(module string, daysBefore int, enabled bool) error {
	now := utils.NowString()
	_, err := s.DB.Exec("INSERT INTO reminder_rules (module, days_before, enabled, created_at, updated_at) VALUES (?,?,?,?,?)",
		module, daysBefore, enabled, now, now)
	return err
}

func (s *ReminderService) UpdateRule(id int, module string, daysBefore int, enabled bool) error {
	now := utils.NowString()
	_, err := s.DB.Exec("UPDATE reminder_rules SET module=?, days_before=?, enabled=?, updated_at=? WHERE id=?",
		module, daysBefore, enabled, now, id)
	return err
}

func (s *ReminderService) DeleteRule(id int) error {
	_, err := s.DB.Exec("DELETE FROM reminder_rules WHERE id=?", id)
	return err
}

func (s *ReminderService) GetLogs(module string, page, pageSize int) ([]models.ReminderLog, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if module != "" {
		where += " AND module=?"
		args = append(args, module)
	}

	var total int
	if err := s.DB.Get(&total, "SELECT COUNT(*) FROM reminder_logs "+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	allArgs := append(args, pageSize, offset)
	var logs []models.ReminderLog
	err := s.DB.Select(&logs, "SELECT * FROM reminder_logs "+where+" ORDER BY id DESC LIMIT ? OFFSET ?", allArgs...)
	return logs, total, err
}

// expiringTarget represents an item nearing expiry for reminder checks
type expiringTarget struct {
	ID         int    `db:"id"`
	Name       string `db:"name"`
	ExpiryDate string `db:"expiry_date"`
}

func (s *ReminderService) RunExpiryCheck() error {
	now := time.Now().Truncate(24 * time.Hour)

	emailSvc := &EmailService{DB: s.DB}

	// Get active admin emails
	var adminEmails []string
	if err := s.DB.Select(&adminEmails, "SELECT email FROM users WHERE role='admin' AND status='active' AND email != ''"); err != nil {
		return fmt.Errorf("查询管理员邮箱失败: %v", err)
	}
	if len(adminEmails) == 0 {
		return nil
	}

	// Get enabled rules
	var rules []models.ReminderRule
	if err := s.DB.Select(&rules, "SELECT * FROM reminder_rules WHERE enabled=1"); err != nil {
		return fmt.Errorf("查询提醒规则失败: %v", err)
	}

	for _, rule := range rules {
		threshold := now.AddDate(0, 0, rule.DaysBefore)
		thresholdStr := threshold.Format("2006-01-02")
		nowStr := now.Format("2006-01-02")

		var items []expiringTarget
		var queryErr error

		switch rule.Module {
		case "personnel_cert":
			queryErr = s.DB.Select(&items, `SELECT pc.id AS id, p.name AS name, pc.expiry_date AS expiry_date
				FROM personnel_certificates pc
				JOIN personnel p ON pc.personnel_id = p.id
				WHERE pc.expiry_date IS NOT NULL AND pc.expiry_date BETWEEN ? AND ? AND p.status='active'`,
				nowStr, thresholdStr)
		case "equipment_cert":
			queryErr = s.DB.Select(&items, `SELECT ec.id AS id, e.name AS name, ec.expiry_date AS expiry_date
				FROM equipment_certificates ec
				JOIN equipment e ON ec.equipment_id = e.id
				WHERE ec.expiry_date IS NOT NULL AND ec.expiry_date != '' AND ec.expiry_date BETWEEN ? AND ? AND e.status='in_use'`,
				nowStr, thresholdStr)
		case "document":
			queryErr = s.DB.Select(&items, `SELECT d.id AS id, d.title AS name, d.expiry_date AS expiry_date
				FROM documents d
				WHERE d.expiry_date IS NOT NULL AND d.expiry_date BETWEEN ? AND ?`,
				nowStr, thresholdStr)
		default:
			continue
		}

		if queryErr != nil {
			return fmt.Errorf("查询 %s 模块到期项失败: %v", rule.Module, queryErr)
		}

		for _, item := range items {
			// Check if already sent today for same module+target (idempotent)
			var sentCount int
			if err := s.DB.Get(&sentCount, `SELECT COUNT(*) FROM reminder_logs
				WHERE module=? AND target_name=? AND date(sent_at)=date(?)`,
				rule.Module, item.Name, utils.NowString()); err != nil {
				continue
			}
			if sentCount > 0 {
				continue
			}

			subject := fmt.Sprintf("【%s】%s 即将到期", getModuleLabel(rule.Module), item.Name)
			body := fmt.Sprintf("%s 的到期日期为 %s，请及时处理。", item.Name, item.ExpiryDate)

			sendErr := emailSvc.SendReminder(adminEmails, subject, body)

			recipients := strings.Join(adminEmails, ",")
			status := "sent"
			var errorMsg *string
			if sendErr != nil {
				status = "failed"
				msg := sendErr.Error()
				errorMsg = &msg
			}

			daysBefore := rule.DaysBefore
			expiryDateStr := item.ExpiryDate
			// expiry_date stored as TEXT "2006-01-02" or "2006-01-02T15:04:05..."
			// Parse just the date part
			if len(expiryDateStr) >= 10 {
				expiryDateStr = expiryDateStr[:10]
			}

			nowFormatted := utils.NowString()
			s.DB.Exec(`INSERT INTO reminder_logs (module, target_name, expiry_date, days_before, status, recipients, error_msg, sent_at, created_at)
				VALUES (?,?,?,?,?,?,?,?,?)`,
				rule.Module, item.Name, expiryDateStr, daysBefore, status, recipients, errorMsg, nowFormatted, nowFormatted)
		}
	}

	return nil
}

func getModuleLabel(module string) string {
	switch module {
	case "personnel_cert":
		return "人员证书"
	case "equipment_cert":
		return "设备证书"
	case "document":
		return "文档"
	default:
		return module
	}
}
