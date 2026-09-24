package service

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/zxjyyh/mds/internal/database"
	"github.com/zxjyyh/mds/internal/scope"
)

// openCertClearTestDB 建干净库并完成迁移，供清空预留证件回归测试使用。
func openCertClearTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewDB 失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// seedCertClearFixture 构造：
//
//	人员1 安检证(预留,已上传f101,已填编号/到期)  环保证(预留,未上传)
//	人员2 身份证(自由添加,已上传f102)
func seedCertClearFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`INSERT INTO users (id, username, password_hash, real_name, email) VALUES (1,'u','x','测试','u@x.test')`)
	db.MustExec(`INSERT INTO personnel (id, name) VALUES (1,'张一'), (2,'李二')`)
	db.MustExec(`INSERT INTO uploaded_files (id, original_name, stored_name, mime_type, storage_path, uploaded_by) VALUES
		(101,'安检.jpg','a.jpg','image/jpeg','/tmp/clear-a.jpg',1),
		(102,'身份证.jpg','b.jpg','image/jpeg','/tmp/clear-b.jpg',1)`)
	db.MustExec(`INSERT INTO personnel_certificates (id, personnel_id, cert_name, cert_number, expiry_date, file_id) VALUES
		(1,1,'安检证','AJ-001','2027-12-31',101),
		(2,1,'环保证',NULL,NULL,NULL),
		(3,2,'身份证','SFZ-001',NULL,102)`)
}

func TestClearReservedCert(t *testing.T) {
	db := openCertClearTestDB(t)
	seedCertClearFixture(t, db)
	svc := &PersonnelService{DB: db}

	// 1) 清空已上传的预留行：字段复位 + 文件记录删除 + 行本身保留
	if err := svc.ClearReservedCert(scope.All(), 1); err != nil {
		t.Fatalf("ClearReservedCert(1) 失败: %v", err)
	}
	var cert struct {
		CertName   string  `db:"cert_name"`
		CertNumber *string `db:"cert_number"`
		ExpiryDate *string `db:"expiry_date"`
		FileID     *int    `db:"file_id"`
	}
	if err := db.Get(&cert, `SELECT cert_name, cert_number, expiry_date, file_id FROM personnel_certificates WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if cert.CertName != "安检证" {
		t.Fatalf("清空后预留行名称不应变化，实际 %q", cert.CertName)
	}
	if cert.CertNumber != nil || cert.ExpiryDate != nil || cert.FileID != nil {
		t.Fatalf("清空后字段应全为空，实际 cert_number=%v expiry_date=%v file_id=%v",
			cert.CertNumber, cert.ExpiryDate, cert.FileID)
	}
	var fileCnt int
	if err := db.Get(&fileCnt, `SELECT COUNT(*) FROM uploaded_files WHERE id=101`); err != nil {
		t.Fatal(err)
	}
	if fileCnt != 0 {
		t.Fatalf("清空应删除 uploaded_files 记录，101 仍存在")
	}

	// 2) 清空无内容的预留行：幂等、不报错
	if err := svc.ClearReservedCert(scope.All(), 2); err != nil {
		t.Fatalf("清空空预留行失败: %v", err)
	}

	// 3) 自由添加的证书走 ClearReservedCert 应被拒绝（应走整行删除）
	if err := svc.ClearReservedCert(scope.All(), 3); err == nil {
		t.Fatal("自由添加的证件不应允许 ClearReservedCert，应报错")
	}

	// 4) 行仍在：总共 3 行证书
	var total int
	if err := db.Get(&total, `SELECT COUNT(*) FROM personnel_certificates`); err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("清空不应删除证书行，期望 3 行，实际 %d", total)
	}
}
