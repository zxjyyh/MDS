package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/zxjyyh/mds/internal/utils"
)

// UserCompanyService 账号 ↔ 可见公司（数据隔离）。
//
// 表里存的是公司名，规则与设计见 docs/company-data-scope.md。
type UserCompanyService struct {
	DB *sqlx.DB
	// CompanySvc 用于校验公司名是否合法（合法集合 = 公司档案 ∪ 实际出现过的检测站）。
	// **必须注入** —— 缺了它 SetCompanies 会直接报错，而不是跳过校验：
	// 静默跳过校验等于让笔误名称写进权限表，进而导致受限账号静默看不到数据。
	CompanySvc *CompanyService
}

// CompaniesOf 返回某账号已配置的公司名。
func (s *UserCompanyService) CompaniesOf(userID int) ([]string, error) {
	list := []string{}
	err := s.DB.Select(&list,
		`SELECT company_name FROM user_companies WHERE user_id=? ORDER BY id`, userID)
	return list, err
}

// NamesOfUsers 批量取多个账号的可见公司，避免用户列表出现 N+1 查询。
func (s *UserCompanyService) NamesOfUsers(userIDs []int) (map[int][]string, error) {
	out := map[int][]string{}
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		UserID      int    `db:"user_id"`
		CompanyName string `db:"company_name"`
	}
	q, args, err := sqlx.In(
		`SELECT user_id, company_name FROM user_companies WHERE user_id IN (?) ORDER BY user_id, id`,
		userIDs)
	if err != nil {
		return nil, err
	}
	rows := []row{}
	if err := s.DB.Select(&rows, q, args...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], r.CompanyName)
	}
	return out, nil
}

// SetCompanies 覆盖式设置某账号的可见公司（传空列表 = 清空 = 恢复为「不受限」）。
//
// 写入前校验每个名称都落在合法集合内：权限表里的笔误名称不会报错、只会让受限账号
// **静默看不到数据**，是最难发现的一类问题，所以在这里就拦住。
// 事务保证「先删后插」不留下中间态。
func (s *UserCompanyService) SetCompanies(userID int, names []string) error {
	if s.CompanySvc == nil {
		return errors.New("内部错误：未注入 CompanySvc，无法校验公司名")
	}

	list := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		list = append(list, n)
	}

	if len(list) > 0 {
		valid, err := s.CompanySvc.ValidNames()
		if err != nil {
			return err
		}
		var bad []string
		for _, n := range list {
			if !valid[n] {
				bad = append(bad, n)
			}
		}
		if len(bad) > 0 {
			return fmt.Errorf("公司「%s」既不在公司信息中，也没有任何人员或设备归属它，请先在「公司信息管理」中登记",
				strings.Join(bad, "、"))
		}
	}

	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // 已提交后回滚是 no-op

	if _, err := tx.Exec(`DELETE FROM user_companies WHERE user_id=?`, userID); err != nil {
		return err
	}
	now := utils.NowString()
	for _, n := range list {
		if _, err := tx.Exec(
			`INSERT INTO user_companies (user_id, company_name, created_at) VALUES (?,?,?)`,
			userID, n, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
