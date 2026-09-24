package service

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/scope"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/jmoiron/sqlx"
)

// strOf 从 data map（handler 传来的 map[string]interface{}）里安全取字符串。
// 各 service 共用：置空/非字符串一律当空串处理，避免断言 panic。
func strOf(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

type DocumentService struct{ DB *sqlx.DB }

// 文档类型（与前端 system_type / doc_categories.system_type 一致）。
const (
	DocSystemCompanyArchive = "company_archive" // 公司档案
	DocSystemLegalStandard  = "legal_standard"  // 法规标准
	DocSystemQualification  = "qualification"   // 资质体系
	DocSystemSafety         = "safety"          // 安全生产
	DocSystemIP             = "ip"              // IP 信息
)

// DocIsGlobal 该类型文档是否为**全局共享**（不受公司隔离）。
//
// 用户 2026-09-22 确认：只有「法规标准」是全局的；
// 其余（公司档案 / IP 信息 / 资质体系 / 安全生产）都按 documents.company_name 归属公司。
func DocIsGlobal(systemType string) bool {
	return systemType == DocSystemLegalStandard
}

// docVisible 判断某文档对当前账号是否可见。
func docVisible(sc *scope.Scope, systemType, companyName string) bool {
	if DocIsGlobal(systemType) {
		return true
	}
	return sc.Allows(companyName)
}

// assertDoc 确认该文档存在且在当前账号的可见范围内。
func (s *DocumentService) assertDoc(sc *scope.Scope, id int) error {
	var row struct {
		SystemType  string  `db:"system_type"`
		CompanyName *string `db:"company_name"`
	}
	if err := s.DB.Get(&row, "SELECT system_type, company_name FROM documents WHERE id=?", id); err != nil {
		return err
	}
	c := ""
	if row.CompanyName != nil {
		c = *row.CompanyName
	}
	if !docVisible(sc, row.SystemType, c) {
		return fmt.Errorf("%w：该文档不属于你可访问的公司", scope.ErrForbidden)
	}
	return nil
}

func (s *DocumentService) List(sc *scope.Scope, systemType string, categoryID *int, keyword string, expiryStatus string, page, pageSize int) ([]models.Document, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if systemType != "" {
		where += " AND d.system_type=?"
		args = append(args, systemType)
	}
	if categoryID != nil {
		where += " AND d.category_id=?"
		args = append(args, *categoryID)
	}
	if keyword != "" {
		where += " AND (d.title LIKE ? OR d.doc_number LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}

	// Expiry status filter (frontend sends: expired|expiring_30|expiring_60|expiring_90|normal)
	now := time.Now().Format("2006-01-02")
	switch expiryStatus {
	case "expired":
		where += " AND d.expiry_date IS NOT NULL AND d.expiry_date < ?"
		args = append(args, now)
	case "expiring_30":
		future := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		where += " AND d.expiry_date IS NOT NULL AND d.expiry_date BETWEEN ? AND ?"
		args = append(args, now, future)
	case "expiring_60":
		start := time.Now().AddDate(0, 0, 31).Format("2006-01-02")
		end := time.Now().AddDate(0, 0, 60).Format("2006-01-02")
		where += " AND d.expiry_date IS NOT NULL AND d.expiry_date BETWEEN ? AND ?"
		args = append(args, start, end)
	case "expiring_90":
		start := time.Now().AddDate(0, 0, 61).Format("2006-01-02")
		end := time.Now().AddDate(0, 0, 90).Format("2006-01-02")
		where += " AND d.expiry_date IS NOT NULL AND d.expiry_date BETWEEN ? AND ?"
		args = append(args, start, end)
	case "normal":
		future := time.Now().AddDate(0, 0, 90).Format("2006-01-02")
		where += " AND (d.expiry_date IS NULL OR d.expiry_date > ?)"
		args = append(args, future)
	}

	// 数据隔离：法规标准全局共享，其余按所属公司过滤。
	// 注意这里是**一个整体条件**（OR 包在一起），不能拆成两条 AND ——
	// 否则「法规标准」也会被公司白名单挡掉。
	if f := sc.Filter(); len(f) > 0 {
		ph := make([]string, len(f))
		for i := range f {
			ph[i] = "?"
		}
		where += " AND (d.system_type = ? OR d.company_name IN (" + strings.Join(ph, ",") + "))"
		args = append(args, DocSystemLegalStandard)
		for _, c := range f {
			args = append(args, c)
		}
	}

	var total int
	if err := s.DB.Get(&total, "SELECT COUNT(*) FROM documents d "+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQ := `SELECT d.* FROM documents d ` + where + ` ORDER BY d.id DESC LIMIT ? OFFSET ?`
	allArgs := append(args, pageSize, offset)

	var docs []models.Document
	if err := s.DB.Select(&docs, selectQ, allArgs...); err != nil {
		return nil, 0, err
	}

	// Join category, file, createdBy for each document
	for i := range docs {
		var cat models.DocCategory
		if err := s.DB.Get(&cat, "SELECT * FROM doc_categories WHERE id=?", docs[i].CategoryID); err == nil {
			docs[i].Category = &cat
		}
		if docs[i].FileID != nil {
			var f models.UploadedFile
			if err := s.DB.Get(&f, "SELECT * FROM uploaded_files WHERE id=?", *docs[i].FileID); err == nil {
				docs[i].File = &f
			}
		}
		if docs[i].CreatedByID != nil {
			var u models.User
			if err := s.DB.Get(&u, "SELECT id, username, real_name, email, role, status FROM users WHERE id=?", *docs[i].CreatedByID); err == nil {
				docs[i].CreatedBy = &u
			}
		}
	}

	return docs, total, nil
}

func (s *DocumentService) GetByID(sc *scope.Scope, id int) (*models.Document, error) {
	if err := s.assertDoc(sc, id); err != nil {
		return nil, err
	}
	var doc models.Document
	if err := s.DB.Get(&doc, "SELECT * FROM documents WHERE id=?", id); err != nil {
		return nil, err
	}

	var cat models.DocCategory
	if err := s.DB.Get(&cat, "SELECT * FROM doc_categories WHERE id=?", doc.CategoryID); err == nil {
		doc.Category = &cat
	}
	if doc.FileID != nil {
		var f models.UploadedFile
		if err := s.DB.Get(&f, "SELECT * FROM uploaded_files WHERE id=?", *doc.FileID); err == nil {
			doc.File = &f
		}
	}
	if doc.CreatedByID != nil {
		var u models.User
		if err := s.DB.Get(&u, "SELECT id, username, real_name, email, role, status FROM users WHERE id=?", *doc.CreatedByID); err == nil {
			doc.CreatedBy = &u
		}
	}

	return &doc, nil
}

func (s *DocumentService) Create(sc *scope.Scope, data map[string]interface{}, userID int) error {
	categoryID, ok := data["categoryId"]
	if !ok {
		return fmt.Errorf("缺少categoryId参数")
	}
	// Determine systemType from category
	var sysType string
	s.DB.Get(&sysType, "SELECT system_type FROM doc_categories WHERE id=?", categoryID)

	// 归属校验：除法规标准（全局共享）外的文档必须归入当前账号可见的公司，
	// 否则新文档对创建者自己都不可见，等于造了一条"看不见的孤儿数据"。
	if !DocIsGlobal(sysType) && !sc.Allows(strOf(data["companyName"])) {
		return fmt.Errorf("%w：只能把文档归入你有权限的公司", scope.ErrForbidden)
	}

	now := utils.NowString()
	res, err := s.DB.Exec(`INSERT INTO documents (category_id, system_type, title, doc_number, version, issue_date, expiry_date, description, company_name, unified_credit_code, legal_person, address, cert_number, file_id, created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		categoryID, sysType, data["title"], data["docNumber"], data["version"],
		data["issueDate"], data["expiryDate"], data["description"],
		data["companyName"], data["unifiedCreditCode"], data["legalPerson"], data["address"], data["certNumber"],
		data["fileId"], userID, now, now)
	if err != nil {
		return err
	}
	docID, _ := res.LastInsertId()

	// Update file owner
	if data["fileId"] != nil {
		s.DB.Exec("UPDATE uploaded_files SET owner_type='document', owner_id=? WHERE id=?", docID, data["fileId"])
	}
	// 同步文件名（若有）
	s.applyFileName(data)

	return nil
}

func (s *DocumentService) Update(sc *scope.Scope, id int, data map[string]interface{}) error {
	if err := s.assertDoc(sc, id); err != nil {
		return err
	}
	if _, has := data["companyName"]; has {
		if !sc.Allows(strOf(data["companyName"])) {
			return fmt.Errorf("%w：不能把文档改到你没有权限的公司", scope.ErrForbidden)
		}
	}
	now := utils.NowString()
	_, err := s.DB.Exec(`UPDATE documents SET title=?, doc_number=?, version=?, issue_date=?, expiry_date=?, description=?, company_name=?, unified_credit_code=?, legal_person=?, address=?, cert_number=?, file_id=?, updated_at=? WHERE id=?`,
		data["title"], data["docNumber"], data["version"], data["issueDate"], data["expiryDate"],
		data["description"], data["companyName"], data["unifiedCreditCode"], data["legalPerson"], data["address"], data["certNumber"],
		data["fileId"], now, id)
	if err != nil {
		return err
	}
	// 同步文件名（若有）
	s.applyFileName(data)
	return nil
}

// applyFileName 当请求携带 fileName 时，把其写入对应文件的 original_name，
// 实现「修改文件名时同步修改已上传文件名」。
func (s *DocumentService) applyFileName(data map[string]interface{}) {
	fn, ok := data["fileName"].(string)
	if !ok || fn == "" {
		return
	}
	fileID, ok := data["fileId"].(float64)
	if !ok || fileID <= 0 {
		return
	}
	s.DB.Exec("UPDATE uploaded_files SET original_name=? WHERE id=?", fn, int(fileID))
}

func (s *DocumentService) Delete(sc *scope.Scope, id int) error {
	if err := s.assertDoc(sc, id); err != nil {
		return err
	}
	var fileID *int
	s.DB.Get(&fileID, "SELECT file_id FROM documents WHERE id=?", id)
	if fileID != nil {
		var path string
		s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *fileID)
		if path != "" {
			os.Remove(path)
		}
		s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *fileID)
	}
	_, err := s.DB.Exec("DELETE FROM documents WHERE id=?", id)
	return err
}
