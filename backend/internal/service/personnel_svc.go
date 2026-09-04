package service

import (
	"fmt"
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
	"os"

	"github.com/jmoiron/sqlx"
)

type PersonnelService struct{ DB *sqlx.DB }

func (s *PersonnelService) List(keyword, department, status string, page, pageSize int) ([]models.Personnel, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if keyword != "" {
		where += " AND p.name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	if department != "" {
		where += " AND p.department=?"
		args = append(args, department)
	}
	if status != "" {
		where += " AND p.status=?"
		args = append(args, status)
	}

	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM personnel p `+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQ := fmt.Sprintf(`SELECT p.*,
		COALESCE((SELECT COUNT(*) FROM personnel_certificates WHERE personnel_id=p.id), 0) AS cert_count
		FROM personnel p %s ORDER BY p.id DESC LIMIT ? OFFSET ?`, where)
	allArgs := append(args, pageSize, offset)

	rows, err := s.DB.Queryx(selectQ, allArgs...)
	if err != nil {
		return nil, 0, err
	}

	var list []models.Personnel
	photoIDs := make([]int, 0, pageSize)
	for rows.Next() {
		var p models.Personnel
		if err := rows.StructScan(&p); err != nil {
			rows.Close()
			return nil, 0, fmt.Errorf("扫描人员数据失败: %v", err)
		}
		p.Count = &models.Count{Certificates: p.CertCount}
		if p.PhotoFileID != nil {
			photoIDs = append(photoIDs, *p.PhotoFileID)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	// 关键：必须先关闭游标把连接还给池，再发起下一次查询。
	// 原实现在 rows 未关闭时循环调用 s.DB.Get 查照片（嵌套游标），
	// 每个并发请求都会占住一条连接再去抢第二条，连接池极易死锁。
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}

	// 批量取头像文件，顺带消除 N+1 查询
	if len(photoIDs) > 0 {
		photos, err := s.filesByIDs(photoIDs)
		if err != nil {
			return nil, 0, err
		}
		for i := range list {
			if list[i].PhotoFileID == nil {
				continue
			}
			if f, ok := photos[*list[i].PhotoFileID]; ok {
				list[i].PhotoFile = f
			}
		}
	}

	return list, total, nil
}

// Options 返回用于筛选下拉框的检测站去重列表（取自人员表自身）。
func (s *PersonnelService) Options() ([]string, error) {
	departments := []string{}
	if err := s.DB.Select(&departments, `SELECT DISTINCT department FROM personnel WHERE department IS NOT NULL AND department != '' ORDER BY department`); err != nil {
		return nil, err
	}
	return departments, nil
}

// filesByIDs 一次性按 ID 批量取回文件记录，返回 id -> 文件 的映射。
func (s *PersonnelService) filesByIDs(ids []int) (map[int]*models.UploadedFile, error) {
	query, args, err := sqlx.In("SELECT * FROM uploaded_files WHERE id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	var files []models.UploadedFile
	if err := s.DB.Select(&files, s.DB.Rebind(query), args...); err != nil {
		return nil, err
	}
	m := make(map[int]*models.UploadedFile, len(files))
	for i := range files {
		m[files[i].ID] = &files[i]
	}
	return m, nil
}

func (s *PersonnelService) GetByID(id int) (*models.Personnel, error) {
	var p models.Personnel
	if err := s.DB.Get(&p, "SELECT * FROM personnel WHERE id=?", id); err != nil {
		return nil, err
	}

	if p.PhotoFileID != nil {
		var photo models.UploadedFile
		if err := s.DB.Get(&photo, "SELECT * FROM uploaded_files WHERE id=?", *p.PhotoFileID); err == nil {
			p.PhotoFile = &photo
		}
	}

	var certs []models.PersonnelCertificate
	s.DB.Select(&certs, "SELECT * FROM personnel_certificates WHERE personnel_id=? ORDER BY id", id)
	for i := range certs {
		if certs[i].FileID != nil {
			var f models.UploadedFile
			if err := s.DB.Get(&f, "SELECT * FROM uploaded_files WHERE id=?", *certs[i].FileID); err == nil {
				certs[i].File = &f
			}
		}
	}
	p.Certificates = certs
	p.CertCount = len(certs)
	p.Count = &models.Count{Certificates: p.CertCount}
	return &p, nil
}

func (s *PersonnelService) Create(data map[string]interface{}, userID int) (*models.Personnel, error) {
	now := utils.NowString()

	res, err := s.DB.Exec(`INSERT INTO personnel (name, gender, id_card, phone, position, department, address, entry_date, education, school, major, driver_license, driver_license_expiry, id_card_expiry, graduation_date, security_cert_date, security_cert_number, env_cert_date, env_cert_number, inspection_date, inspection_cert_number, internal_auditor_number, internal_auditor_date, status, photo_file_id, remark, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		data["name"], data["gender"], data["idCard"], data["phone"], data["position"],
		data["department"], data["address"], data["entryDate"], data["education"], data["school"], data["major"], data["driverLicense"],
		data["driverLicenseExpiry"], data["idCardExpiry"], data["graduationDate"],
		data["securityCertDate"], data["securityCertNumber"], data["envCertDate"], data["envCertNumber"],
		data["inspectionDate"], data["inspectionCertNumber"], data["internalAuditorNumber"], data["internalAuditorDate"],
		data["status"], data["photoFileId"], data["remark"], now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	// 新人员自动补齐两个固定预留证件（安检证/环保证），
	// 保证任何入口创建的人员打开详情就有固定证书槽位。
	s.seedReservedCerts(int(id))

	var p models.Personnel
	s.DB.Get(&p, "SELECT * FROM personnel WHERE id=?", id)
	return &p, nil
}

// seedReservedCerts 给指定人员补插固定预留证件行（仅名称，其余字段留空）。
// 仅应在人员刚创建（确定无任何证件）时调用，无需判重。
func (s *PersonnelService) seedReservedCerts(personnelID int) {
	now := utils.NowString()
	for _, name := range models.PersonnelReservedCertNames {
		s.DB.Exec(`INSERT INTO personnel_certificates (personnel_id, cert_name, created_at, updated_at)
			VALUES (?,?,?,?)`, personnelID, name, now, now)
	}
}

func (s *PersonnelService) Update(id int, data map[string]interface{}) error {
	now := utils.NowString()
	_, err := s.DB.Exec(`UPDATE personnel SET name=?, gender=?, id_card=?, phone=?, position=?, department=?, address=?, entry_date=?, education=?, school=?, major=?, driver_license=?, driver_license_expiry=?, id_card_expiry=?, graduation_date=?, security_cert_date=?, security_cert_number=?, env_cert_date=?, env_cert_number=?, inspection_date=?, inspection_cert_number=?, internal_auditor_number=?, internal_auditor_date=?, status=?, photo_file_id=?, remark=?, updated_at=? WHERE id=?`,
		data["name"], data["gender"], data["idCard"], data["phone"], data["position"],
		data["department"], data["address"], data["entryDate"], data["education"], data["school"], data["major"], data["driverLicense"],
		data["driverLicenseExpiry"], data["idCardExpiry"], data["graduationDate"],
		data["securityCertDate"], data["securityCertNumber"], data["envCertDate"], data["envCertNumber"],
		data["inspectionDate"], data["inspectionCertNumber"], data["internalAuditorNumber"], data["internalAuditorDate"],
		data["status"], data["photoFileId"], data["remark"], now, id)
	return err
}

func (s *PersonnelService) Delete(id int) error {
	// Get all certificate file IDs and delete physical files
	var certFileIDs []*int
	s.DB.Select(&certFileIDs, "SELECT file_id FROM personnel_certificates WHERE personnel_id=? AND file_id IS NOT NULL", id)
	for _, fid := range certFileIDs {
		if fid != nil {
			var path string
			s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *fid)
			if path != "" {
				os.Remove(path)
			}
			s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *fid)
		}
	}

	// Get photo file ID and delete physical file
	var photoID *int
	s.DB.Get(&photoID, "SELECT photo_file_id FROM personnel WHERE id=?", id)
	if photoID != nil {
		var path string
		s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *photoID)
		if path != "" {
			os.Remove(path)
		}
		s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *photoID)
	}

	_, err := s.DB.Exec("DELETE FROM personnel WHERE id=?", id)
	return err
}

// ---- Certificate operations ----

func (s *PersonnelService) AddCert(personnelID int, data map[string]interface{}) (*models.PersonnelCertificate, error) {
	now := utils.NowString()
	name, _ := data["certName"].(string)
	if models.IsPersonnelReservedCertName(name) {
		var count int
		if err := s.DB.Get(&count, `SELECT COUNT(*) FROM personnel_certificates WHERE personnel_id=? AND cert_name=?`, personnelID, name); err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("「%s」为固定预留证件，已存在该证书行，请直接点击对应行「编辑」上传", name)
		}
	}
	if data["expiryDate"] == "" {
		data["expiryDate"] = nil
	}
	res, err := s.DB.Exec(`INSERT INTO personnel_certificates (personnel_id, cert_name, cert_number, expiry_date, file_id, remark, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		personnelID, data["certName"], data["certNumber"], data["expiryDate"], data["fileId"], data["remark"], now, now)
	if err != nil {
		return nil, err
	}
	certID, _ := res.LastInsertId()

	// Update file owner
	if data["fileId"] != nil {
		s.DB.Exec("UPDATE uploaded_files SET owner_type='personnel_cert', owner_id=? WHERE id=?", certID, data["fileId"])
	}

	var cert models.PersonnelCertificate
	s.DB.Get(&cert, "SELECT * FROM personnel_certificates WHERE id=?", certID)
	return &cert, nil
}

func (s *PersonnelService) UpdateCert(certID int, data map[string]interface{}) error {
	now := utils.NowString()
	var cur struct {
		CertName string `db:"cert_name"`
	}
	if err := s.DB.Get(&cur, "SELECT cert_name FROM personnel_certificates WHERE id=?", certID); err != nil {
		return err
	}
	if models.IsPersonnelReservedCertName(cur.CertName) {
		data["certName"] = cur.CertName // 固定预留项：名称不可修改
	}
	if data["expiryDate"] == "" {
		data["expiryDate"] = nil
	}
	_, err := s.DB.Exec(`UPDATE personnel_certificates SET cert_name=?, cert_number=?, expiry_date=?, file_id=?, remark=?, updated_at=? WHERE id=?`,
		data["certName"], data["certNumber"], data["expiryDate"], data["fileId"], data["remark"], now, certID)
	return err
}

func (s *PersonnelService) DeleteCert(certID int) error {
	var cur struct {
		CertName string `db:"cert_name"`
		FileID   *int   `db:"file_id"`
	}
	if err := s.DB.Get(&cur, "SELECT cert_name, file_id FROM personnel_certificates WHERE id=?", certID); err != nil {
		return err
	}
	if models.IsPersonnelReservedCertName(cur.CertName) {
		return fmt.Errorf("「%s」为固定预留证件，不能删除", cur.CertName)
	}
	if cur.FileID != nil {
		var path string
		s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *cur.FileID)
		if path != "" {
			os.Remove(path)
		}
		s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *cur.FileID)
	}
	_, err := s.DB.Exec("DELETE FROM personnel_certificates WHERE id=?", certID)
	return err
}

// ClearReservedCert 清空固定预留证件行（安检证/环保证）的内容：
// 删除已上传的附件（物理文件 + uploaded_files 记录），并把编号/签发日期/到期日期/备注
// 全部复位为空，让该行回到"待上传"初始状态；证书行本身保留（名称即锚点，不可删）。
// 非预留名称调用本方法视为误用，直接报错（应走 DeleteCert 整行删除）。
func (s *PersonnelService) ClearReservedCert(certID int) error {
	var cur struct {
		CertName string `db:"cert_name"`
		FileID   *int   `db:"file_id"`
	}
	if err := s.DB.Get(&cur, "SELECT cert_name, file_id FROM personnel_certificates WHERE id=?", certID); err != nil {
		return err
	}
	if !models.IsPersonnelReservedCertName(cur.CertName) {
		return fmt.Errorf("「%s」不是固定预留证件，清空仅适用于固定预留项", cur.CertName)
	}
	if cur.FileID != nil {
		var path string
		if err := s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *cur.FileID); err == nil && path != "" {
			os.Remove(path)
		}
		if _, err := s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *cur.FileID); err != nil {
			return err
		}
	}
	_, err := s.DB.Exec(`UPDATE personnel_certificates
		SET cert_number=NULL, issue_date=NULL, expiry_date=NULL, file_id=NULL, remark=NULL, updated_at=?
		WHERE id=?`, utils.NowString(), certID)
	return err
}

// ---- 证书批量导出（打包下载） ----

// PersonnelScopeItem 是批量导出范围内的一名人员，用于生成"缺证人员名单"。
type PersonnelScopeItem struct {
	ID         int     `db:"id"`
	Name       string  `db:"name"`
	Department *string `db:"department"`
	Status     string  `db:"status"`
}

// PersonnelCertExportItem 是批量导出命中的一份证书附件文件。
type PersonnelCertExportItem struct {
	PersonnelID  int     `db:"personnel_id"`
	Name         string  `db:"name"`
	Department   *string `db:"department"`
	Status       string  `db:"status"`
	FileID       int     `db:"file_id"`
	OriginalName string  `db:"original_name"`
	StoragePath  string  `db:"storage_path"`
}

// buildPersonnelWhere 按检测站/在职状态拼 personnel 过滤条件。
func buildPersonnelWhere(department, status string) (string, []interface{}) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if department != "" {
		where += " AND p.department = ?"
		args = append(args, department)
	}
	if status != "" {
		where += " AND p.status = ?"
		args = append(args, status)
	}
	return where, args
}

// PersonnelByScope 返回按检测站/在职状态过滤的全部人员（含已离职，供缺证统计）。
func (s *PersonnelService) PersonnelByScope(department, status string) ([]PersonnelScopeItem, error) {
	where, args := buildPersonnelWhere(department, status)
	rows := []PersonnelScopeItem{}
	err := s.DB.Select(&rows, `SELECT p.id, p.name, p.department, p.status FROM personnel p `+where+
		` ORDER BY p.department IS NULL, p.department, p.name, p.id`, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CertFilesByScope 返回范围内所有人员持有的指定名称证书的附件文件行。
// cert_name 白名单由调用方（handler）把关；此处仍参数化查询，杜绝注入。
func (s *PersonnelService) CertFilesByScope(department, status, certName string) ([]PersonnelCertExportItem, error) {
	where := "pc.cert_name = ?"
	args := []interface{}{certName}
	if department != "" {
		where += " AND p.department = ?"
		args = append(args, department)
	}
	if status != "" {
		where += " AND p.status = ?"
		args = append(args, status)
	}
	rows := []PersonnelCertExportItem{}
	err := s.DB.Select(&rows, `SELECT p.id AS personnel_id, p.name, p.department, p.status,
			uf.id AS file_id, uf.original_name, uf.storage_path
		FROM personnel_certificates pc
		JOIN personnel p ON p.id = pc.personnel_id
		JOIN uploaded_files uf ON uf.id = pc.file_id
		WHERE `+where+` AND pc.file_id IS NOT NULL
		ORDER BY p.department IS NULL, p.department, p.name, p.id, pc.id`, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
