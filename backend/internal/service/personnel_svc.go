package service

import (
	"fmt"
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
	"os"
	"strings"
	"time"

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

// ---- 证件查漏 ----

// CertGapCertNames 支持查漏的证件类型，顺序即前端复选框与表格列顺序。
//
// 判定口径（2026-09-17 按用户要求调整）：**只看证件上传台账**
// personnel_certificates，按「证书名称」匹配；不再读 personnel 的基本信息日期列。
// 该证件名称下没有"有效行"（编号 / 到期日 / 附件任一非空）即视为「未上传」。
//
// 为什么要求"有内容"而不是"有行"：安检证 / 环保证 由系统给每人预置了空行，
// 只按"有行"判断会让这两类永远显示已上传（未上传恒为 0），查漏就失去意义。
// 实测当前数据里「无附件但有编号/到期日」的行数为 0，故本判据与"有没有传附件"等价。
var CertGapCertNames = []string{
	"安检证", "环保证", "查验证", "内审员证", "身份证", "驾驶证", "毕业证", "劳动合同",
}

func IsCertGapCertName(name string) bool {
	for _, n := range CertGapCertNames {
		if n == name {
			return true
		}
	}
	return false
}

// CertGapItem 某人在某一种证件上的状态。
// State 取值：normal 正常 / missing 未上传（界面文案） / expired 过期。
type CertGapItem struct {
	CertName   string  `json:"certName"`
	State      string  `json:"state"`
	ExpiryDate *string `json:"expiryDate"`
}

// CertGapRow 一名人员及其所选证件的状态。
type CertGapRow struct {
	ID         int           `json:"id"`
	Name       string        `json:"name"`
	Department *string       `json:"department"`
	Status     string        `json:"status"`
	Items      []CertGapItem `json:"items"`
}

// certGapPerson 查漏只需要人员的基本身份字段，证件状态全部来自证件上传台账。
type certGapPerson struct {
	ID         int     `db:"id"`
	Name       string  `db:"name"`
	Department *string `db:"department"`
	Status     string  `db:"status"`
}

// CertGap 按检测站/在职状态列出人员的指定证件状态。
// department、status 为空表示不限范围；certNames 为空表示全部支持的证书类型。
// 返回：归一化后的证书名列表（顺序即列顺序）、逐人结果、三态计数汇总。
func (s *PersonnelService) CertGap(department, status string, certNames []string) ([]string, []CertGapRow, map[string]int, error) {
	names := normalizeCertGapNames(certNames)
	where, args := buildPersonnelWhere(department, status)

	people := []certGapPerson{}
	err := s.DB.Select(&people, `SELECT p.id, p.name, p.department, p.status
		FROM personnel p `+where+
		` ORDER BY p.department IS NULL, p.department, p.name, p.id`, args...)
	if err != nil {
		return nil, nil, nil, err
	}

	ledger, err := s.certGapLedger()
	if err != nil {
		return nil, nil, nil, err
	}

	summary := map[string]int{"normal": 0, "missing": 0, "expired": 0}
	list := make([]CertGapRow, 0, len(people))
	for _, p := range people {
		row := CertGapRow{ID: p.ID, Name: p.Name, Department: p.Department, Status: p.Status}
		for _, name := range names {
			item := CertGapItem{CertName: name}
			li := ledger[fmt.Sprintf("%d|%s", p.ID, name)]
			switch {
			case li.expiry == "" && li.number == "" && !li.hasFile:
				item.State = "missing" // 证件上传里没有这个证书名称的有效行 → 未上传
			case li.expiry == "":
				item.State = "normal" // 已上传但没填到期日 = 长期有效 / 无需有效期（如毕业证）
			default:
				d := li.expiry
				item.ExpiryDate = &d
				if certGapExpired(li.expiry) {
					item.State = "expired"
				} else {
					item.State = "normal"
				}
			}
			summary[item.State]++
			row.Items = append(row.Items, item)
		}
		list = append(list, row)
	}
	return names, list, summary, nil
}

// certGapLedgerInfo 某人在某一证件上的「台账登记情况」（来自证件上传）。
type certGapLedgerInfo struct {
	expiry  string
	number  string
	hasFile bool
}

// certGapLedger 取出所有支持类型的证件上传台账，按「人员|证件」归并。
// 同一证件有多行时取更完整的一条：到期日/编号先遇到有值的就用，附件任一存在即为有。
func (s *PersonnelService) certGapLedger() (map[string]certGapLedgerInfo, error) {
	type row struct {
		PersonnelID int     `db:"personnel_id"`
		CertName    string  `db:"cert_name"`
		CertNumber  *string `db:"cert_number"`
		ExpiryDate  *string `db:"expiry_date"`
		FileID      *int    `db:"file_id"`
	}
	inList := strings.TrimSuffix(strings.Repeat("?,", len(CertGapCertNames)), ",")
	inArgs := make([]interface{}, 0, len(CertGapCertNames))
	for _, n := range CertGapCertNames {
		inArgs = append(inArgs, n)
	}
	rows := []row{}
	if err := s.DB.Select(&rows, `SELECT personnel_id, cert_name, cert_number, expiry_date, file_id
		FROM personnel_certificates WHERE cert_name IN (`+inList+`)`, inArgs...); err != nil {
		return nil, err
	}
	out := make(map[string]certGapLedgerInfo, len(rows))
	for _, r := range rows {
		key := fmt.Sprintf("%d|%s", r.PersonnelID, r.CertName)
		cur := out[key]
		if v := certGapTrim(r.ExpiryDate); v != "" && cur.expiry == "" {
			cur.expiry = v
		}
		if v := certGapTrim(r.CertNumber); v != "" && cur.number == "" {
			cur.number = v
		}
		if r.FileID != nil {
			cur.hasFile = true
		}
		out[key] = cur
	}
	return out, nil
}

// certGapTrim 取指针字符串并去空格：NULL 与空串在这里统一成 ""。
func certGapTrim(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// normalizeCertGapNames 去重、过滤非法名，并按 CertGapCertNames 的固定顺序输出，
// 这样列顺序稳定（不随前端复选框点击顺序变化）。全为空则视为"全部类型"。
func normalizeCertGapNames(certNames []string) []string {
	picked := map[string]bool{}
	for _, n := range certNames {
		if IsCertGapCertName(strings.TrimSpace(n)) {
			picked[strings.TrimSpace(n)] = true
		}
	}
	out := []string{}
	for _, n := range CertGapCertNames {
		if picked[n] {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		out = append(out, CertGapCertNames...)
	}
	return out
}

// certGapExpired 复用 utils.ExpiryStatus 的日界判断，保证与前端 ExpiryBadge 一致。
func certGapExpired(dateStr string) bool {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateStr), time.Local)
	if err != nil {
		return false // 脏数据解析不了时按未过期处理，避免误报
	}
	return utils.ExpiryStatus(&t).Status == "expired"
}
