package service

import (
	"fmt"
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
	"os"

	"github.com/jmoiron/sqlx"
)

type EquipmentService struct{ DB *sqlx.DB }

func (s *EquipmentService) List(keyword, status string, page, pageSize int) ([]models.Equipment, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if keyword != "" {
		where += " AND (e.name LIKE ? OR e.serial_number LIKE ? OR e.model LIKE ? OR e.department LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw, kw, kw)
	}
	if status != "" {
		where += " AND e.status=?"
		args = append(args, status)
	}

	var total int
	if err := s.DB.Get(&total, "SELECT COUNT(*) FROM equipment e "+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQ := fmt.Sprintf("SELECT * FROM equipment e %s ORDER BY e.id DESC LIMIT ? OFFSET ?", where)
	allArgs := append(args, pageSize, offset)

	rows, err := s.DB.Queryx(selectQ, allArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []models.Equipment
	for rows.Next() {
		var e models.Equipment
		if err := rows.StructScan(&e); err != nil {
			return nil, 0, fmt.Errorf("failed to scan equipment: %v", err)
		}
		list = append(list, e)
	}

	return list, total, nil
}

func (s *EquipmentService) GetByID(id int) (*models.Equipment, error) {
	var e models.Equipment
	if err := s.DB.Get(&e, "SELECT * FROM equipment WHERE id=?", id); err != nil {
		return nil, err
	}

	var certs []models.EquipmentCertificate
	s.DB.Select(&certs, "SELECT * FROM equipment_certificates WHERE equipment_id=? ORDER BY id", id)
	for i := range certs {
		if certs[i].FileID != nil {
			var f models.UploadedFile
			if err := s.DB.Get(&f, "SELECT * FROM uploaded_files WHERE id=?", *certs[i].FileID); err == nil {
				certs[i].File = &f
			}
		}
	}
	e.Certificates = certs
	return &e, nil
}

func (s *EquipmentService) Create(data map[string]interface{}) (*models.Equipment, error) {
	now := utils.NowString()
	res, err := s.DB.Exec(`INSERT INTO equipment (name, serial_number, department, line, model, manufacturer, status, remark, inspection_cert_number, inspection_date, inspection_expiry_date, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		data["name"], data["serialNumber"], data["department"], data["line"],
		data["model"], data["manufacturer"], data["status"], data["remark"],
		data["inspectionCertNumber"], data["inspectionDate"], data["inspectionExpiryDate"], now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	var e models.Equipment
	s.DB.Get(&e, "SELECT * FROM equipment WHERE id=?", id)
	return &e, nil
}

func (s *EquipmentService) Update(id int, data map[string]interface{}) error {
	now := utils.NowString()
	_, err := s.DB.Exec(`UPDATE equipment SET name=?, serial_number=?, department=?, line=?, model=?, manufacturer=?, status=?, remark=?, inspection_cert_number=?, inspection_date=?, inspection_expiry_date=?, updated_at=? WHERE id=?`,
		data["name"], data["serialNumber"], data["department"], data["line"],
		data["model"], data["manufacturer"], data["status"], data["remark"],
		data["inspectionCertNumber"], data["inspectionDate"], data["inspectionExpiryDate"], now, id)
	return err
}

func (s *EquipmentService) Delete(id int) error {
	var certFileIDs []*int
	s.DB.Select(&certFileIDs, "SELECT file_id FROM equipment_certificates WHERE equipment_id=? AND file_id IS NOT NULL", id)
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

	_, err := s.DB.Exec("DELETE FROM equipment WHERE id=?", id)
	return err
}

// ---- Certificate operations ----

func (s *EquipmentService) AddCert(equipmentID int, data map[string]interface{}) (*models.EquipmentCertificate, error) {
	now := utils.NowString()
	if data["expiryDate"] == "" {
		data["expiryDate"] = nil
	}
	res, err := s.DB.Exec(`INSERT INTO equipment_certificates (equipment_id, cert_type, cert_number, expiry_date, file_id, remark, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		equipmentID, data["certType"], data["certNumber"], data["expiryDate"], data["fileId"], data["remark"], now, now)
	if err != nil {
		return nil, err
	}
	certID, _ := res.LastInsertId()

	if data["fileId"] != nil {
		s.DB.Exec("UPDATE uploaded_files SET owner_type='equipment_cert', owner_id=? WHERE id=?", certID, data["fileId"])
	}

	var cert models.EquipmentCertificate
	s.DB.Get(&cert, "SELECT * FROM equipment_certificates WHERE id=?", certID)
	return &cert, nil
}

func (s *EquipmentService) UpdateCert(certID int, data map[string]interface{}) error {
	now := utils.NowString()
	if data["expiryDate"] == "" {
		data["expiryDate"] = nil
	}
	_, err := s.DB.Exec(`UPDATE equipment_certificates SET cert_type=?, cert_number=?, expiry_date=?, file_id=?, remark=?, updated_at=? WHERE id=?`,
		data["certType"], data["certNumber"], data["expiryDate"], data["fileId"], data["remark"], now, certID)
	return err
}

func (s *EquipmentService) DeleteCert(certID int) error {
	var fileID *int
	s.DB.Get(&fileID, "SELECT file_id FROM equipment_certificates WHERE id=?", certID)
	if fileID != nil {
		var path string
		s.DB.Get(&path, "SELECT storage_path FROM uploaded_files WHERE id=?", *fileID)
		if path != "" {
			os.Remove(path)
		}
		s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", *fileID)
	}
	_, err := s.DB.Exec("DELETE FROM equipment_certificates WHERE id=?", certID)
	return err
}
