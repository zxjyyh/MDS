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
		where += " AND (p.name LIKE ? OR p.phone LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
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

	var p models.Personnel
	s.DB.Get(&p, "SELECT * FROM personnel WHERE id=?", id)
	return &p, nil
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
	if data["expiryDate"] == "" {
		data["expiryDate"] = nil
	}
	_, err := s.DB.Exec(`UPDATE personnel_certificates SET cert_name=?, cert_number=?, expiry_date=?, file_id=?, remark=?, updated_at=? WHERE id=?`,
		data["certName"], data["certNumber"], data["expiryDate"], data["fileId"], data["remark"], now, certID)
	return err
}

func (s *PersonnelService) DeleteCert(certID int) error {
	var fileID *int
	s.DB.Get(&fileID, "SELECT file_id FROM personnel_certificates WHERE id=?", certID)
	if fileID != nil {
		var path string
		s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *fileID)
		if path != "" {
			os.Remove(path)
		}
		s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *fileID)
	}
	_, err := s.DB.Exec("DELETE FROM personnel_certificates WHERE id=?", certID)
	return err
}
