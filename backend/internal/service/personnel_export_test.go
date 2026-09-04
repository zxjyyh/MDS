package service

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/zxjyyh/mds/internal/database"
)

// openExportTestDB 建一个干净库并完成迁移，供导出查询回归测试使用。
func openExportTestDB(t *testing.T) *sqlx.DB {
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

// seedExportFixture 构造导出场景：
//
//	1  红星站 在职   安检证(f1) 环保证(f2) 身份证(f3)
//	2  红星站 离职   安检证(f4)
//	3  白水站 在职   安检证(f5)           （无环保证 → 缺证）
//	4  红星站 在职   （无任何附件 → 缺证）
func seedExportFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`INSERT INTO users (id, username, password_hash, real_name, email) VALUES (1,'u','x','测试','u@x.test')`)
	db.MustExec(`INSERT INTO personnel (id, name, department, status) VALUES
		(1,'张一','红星站','active'),
		(2,'李二','红星站','resigned'),
		(3,'王三','白水站','active'),
		(4,'赵四','红星站','active')`)
	db.MustExec(`INSERT INTO uploaded_files (id, original_name, stored_name, mime_type, storage_path, uploaded_by) VALUES
		(101,'张一-安检.jpg','a.jpg','image/jpeg','/tmp/a.jpg',1),
		(102,'张一-环保.jpg','b.jpg','image/jpeg','/tmp/b.jpg',1),
		(103,'张一-身份证.jpg','c.jpg','image/jpeg','/tmp/c.jpg',1),
		(104,'李二-安检.jpg','d.jpg','image/jpeg','/tmp/d.jpg',1),
		(105,'王三-安检.jpg','e.jpg','image/jpeg','/tmp/e.jpg',1)`)
	db.MustExec(`INSERT INTO personnel_certificates (personnel_id, cert_name, file_id) VALUES
		(1,'安检证',101),
		(1,'环保证',102),
		(1,'身份证',103),
		(2,'安检证',104),
		(3,'安检证',105)`)
}

func TestPersonnelByScope(t *testing.T) {
	db := openExportTestDB(t)
	seedExportFixture(t, db)
	svc := &PersonnelService{DB: db}

	got, err := svc.PersonnelByScope("红星站", "active")
	if err != nil {
		t.Fatalf("PersonnelByScope 失败: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("红星站+在职期望 2 人(张一/赵四)，实际 %d", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 4 {
		t.Fatalf("顺序异常: %v / %v", got[0].ID, got[1].ID)
	}

	// 空条件 = 全部人员
	all, err := svc.PersonnelByScope("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("不筛选应返回 4 人，实际 %d", len(all))
	}
}

func TestCertFilesByScope(t *testing.T) {
	db := openExportTestDB(t)
	seedExportFixture(t, db)
	svc := &PersonnelService{DB: db}

	// 按范围 + 证书名
	got, err := svc.CertFilesByScope("红星站", "active", "安检证")
	if err != nil {
		t.Fatalf("CertFilesByScope 失败: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("红星站+在职的安检证文件期望 1 份(张一)，实际 %d", len(got))
	}
	if got[0].Name != "张一" || got[0].OriginalName != "张一-安检.jpg" {
		t.Fatalf("字段带回异常: %+v", got[0])
	}

	// 仅证书名（不限范围）：3 个有安检证文件的人都在，赵四缺证不出现在文件列表
	byCert, err := svc.CertFilesByScope("", "", "安检证")
	if err != nil {
		t.Fatal(err)
	}
	if len(byCert) != 3 {
		t.Fatalf("全员安检证文件期望 3 份，实际 %d", len(byCert))
	}

	// 身份证只命中张一
	idFiles, err := svc.CertFilesByScope("", "", "身份证")
	if err != nil {
		t.Fatal(err)
	}
	if len(idFiles) != 1 || idFiles[0].Name != "张一" {
		t.Fatalf("身份证文件期望 1 份(张一)，实际 %+v", idFiles)
	}

	// 范围外证书名不返回任何行（安全边界）
	none, err := svc.CertFilesByScope("白水站", "active", "环保证")
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("白水站无人有环保证附件，期望 0 份，实际 %d", len(none))
	}
}
