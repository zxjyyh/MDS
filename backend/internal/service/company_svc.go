package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
)

type CompanyService struct{ DB *sqlx.DB }

// List 按关键字（公司名 / 信用代码 / 资质证书编号）模糊查询，分页返回。
func (s *CompanyService) List(keyword string, page, pageSize int) ([]models.Company, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if kw := strings.TrimSpace(keyword); kw != "" {
		where += " AND (c.name LIKE ? OR c.credit_code LIKE ? OR c.qual_cert_no LIKE ?)"
		like := "%" + kw + "%"
		args = append(args, like, like, like)
	}

	var total int
	if err := s.DB.Get(&total, "SELECT COUNT(*) FROM companies c "+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQ := fmt.Sprintf("SELECT * FROM companies c %s ORDER BY c.id DESC LIMIT ? OFFSET ?", where)
	rows, err := s.DB.Queryx(selectQ, append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := []models.Company{}
	for rows.Next() {
		var c models.Company
		if err := rows.StructScan(&c); err != nil {
			return nil, 0, fmt.Errorf("failed to scan company: %w", err)
		}
		list = append(list, c)
	}
	return list, total, nil
}

func (s *CompanyService) GetByID(id int) (*models.Company, error) {
	var c models.Company
	if err := s.DB.Get(&c, "SELECT * FROM companies WHERE id=?", id); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CompanyService) Create(data map[string]interface{}) (*models.Company, error) {
	name := strings.TrimSpace(asString(data["name"]))
	if name == "" {
		return nil, errors.New("公司名不能为空")
	}
	now := utils.NowString()
	res, err := s.DB.Exec(`INSERT INTO companies
		(name, credit_code, qual_cert_no, qual_cert_expiry, address, phone, bank_account, bank_branch, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		name,
		nilIfBlank(data["creditCode"]),
		nilIfBlank(data["qualCertNo"]),
		nilIfBlank(data["qualCertExpiry"]),
		nilIfBlank(data["address"]),
		nilIfBlank(data["phone"]),
		nilIfBlank(data["bankAccount"]),
		nilIfBlank(data["bankBranch"]),
		now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetByID(int(id))
}

func (s *CompanyService) Update(id int, data map[string]interface{}) error {
	name := strings.TrimSpace(asString(data["name"]))
	if name == "" {
		return errors.New("公司名不能为空")
	}
	_, err := s.DB.Exec(`UPDATE companies SET
		name=?, credit_code=?, qual_cert_no=?, qual_cert_expiry=?, address=?, phone=?, bank_account=?, bank_branch=?, updated_at=?
		WHERE id=?`,
		name,
		nilIfBlank(data["creditCode"]),
		nilIfBlank(data["qualCertNo"]),
		nilIfBlank(data["qualCertExpiry"]),
		nilIfBlank(data["address"]),
		nilIfBlank(data["phone"]),
		nilIfBlank(data["bankAccount"]),
		nilIfBlank(data["bankBranch"]),
		utils.NowString(), id)
	return err
}

func (s *CompanyService) Delete(id int) error {
	_, err := s.DB.Exec("DELETE FROM companies WHERE id=?", id)
	return err
}

// Options 返回公司下拉选项（id / name / address），供劳动合同选择甲方并自动带出公司地址。
func (s *CompanyService) Options() ([]models.Company, error) {
	list := []models.Company{}
	if err := s.DB.Select(&list, "SELECT id, name, address FROM companies ORDER BY name"); err != nil {
		return nil, err
	}
	return list, nil
}

// AddressByName 按公司名精确匹配返回公司地址（找不到或地址为空时返回空串）。
// 用途：劳动合同生成时以「甲方名称」回查公司信息，避免公司在两处重复维护地址。
func (s *CompanyService) AddressByName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var row struct {
		Address *string `db:"address"`
	}
	if err := s.DB.Get(&row, "SELECT address FROM companies WHERE name=? ORDER BY id LIMIT 1", name); err != nil || row.Address == nil {
		return ""
	}
	return strings.TrimSpace(*row.Address)
}

// ---- 公司名集合（「公司档案 ∪ 实际出现过的检测站」）----

// companyNameUnionSQL 返回公司名的并集。
//
// 为什么是并集而不是只取 companies：personnel.department / equipment.department 存的是
// 自由填写的公司名，历史数据里可能存在没在公司档案登记过的名称
// （本项目实际出现过"某公司只在人员数据里、公司档案中没有"的情况）。
// 权限配置与「所属检测站」下拉都以此并集为取值来源，才能覆盖历史数据。
const companyNameUnionSQL = `
	SELECT name FROM companies WHERE name IS NOT NULL AND name != ''
	UNION
	SELECT department FROM personnel WHERE department IS NOT NULL AND department != ''
	UNION
	SELECT department FROM equipment WHERE department IS NOT NULL AND department != ''`

// ValidNames 返回合法公司名的集合，供写入前校验（权限表、检测站字段）。
func (s *CompanyService) ValidNames() (map[string]bool, error) {
	names := []string{}
	if err := s.DB.Select(&names, companyNameUnionSQL); err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			set[n] = true
		}
	}
	return set, nil
}

// NameOptions 返回合法公司名的**有序**列表（去重）。
// 顺序：公司档案里的优先（按名称），其余（仅数据里出现过的）排在其后。
func (s *CompanyService) NameOptions() ([]string, error) {
	type row struct {
		Name    string `db:"name"`
		InFirms int    `db:"in_firms"`
	}
	q := `SELECT name, MAX(in_firms) AS in_firms FROM (
			SELECT name, 1 AS in_firms FROM companies WHERE name IS NOT NULL AND name != ''
			UNION ALL
			SELECT department AS name, 0 AS in_firms FROM personnel
			  WHERE department IS NOT NULL AND department != ''
			UNION ALL
			SELECT department AS name, 0 AS in_firms FROM equipment
			  WHERE department IS NOT NULL AND department != ''
		  ) GROUP BY name
		  ORDER BY in_firms DESC, name ASC`
	rows := []row{}
	if err := s.DB.Select(&rows, q); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if n := strings.TrimSpace(r.Name); n != "" {
			out = append(out, n)
		}
	}
	return out, nil
}

// ---- 内部小工具 ----

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

// nilIfBlank 把空字符串归一化为 NULL，避免库里混入 ” 让前端的判空逻辑出现两种空值。
func nilIfBlank(v interface{}) interface{} {
	if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
		return nil
	}
	return v
}
