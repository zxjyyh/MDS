package service

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/scope"
)

// seedScopeFixture 构造两家公司的数据：
//
//	甲检测站：人员1(张甲)、设备1、人员证书(f201)、个人头像(f101)
//	乙检测站：人员2(张乙)、设备2、人员证书(f202)、设备证书(f203)
//	游离文件 f999（不属于任何业务实体）
//
// 用最小字段即可（各表其余列可空或有默认值）。
func seedScopeFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`INSERT INTO users (id, username, password_hash, real_name, email) VALUES
		(1,'admin','x','管理员','a@x.test'),
		(2,'zj','x','庄甲','zj@x.test')`)
	// 文件必须先插：personnel.photo_file_id / *_certificates.file_id 都有外键指向 uploaded_files
	db.MustExec(`INSERT INTO uploaded_files (id, original_name, stored_name, mime_type, storage_path, uploaded_by) VALUES
		(101,'甲头像.jpg','a1.jpg','image/jpeg','/tmp/s-101.jpg',1),
		(201,'甲证件.jpg','a2.jpg','image/jpeg','/tmp/s-201.jpg',1),
		(202,'乙证件.jpg','b2.jpg','image/jpeg','/tmp/s-202.jpg',1),
		(203,'乙设备证.jpg','b3.jpg','image/jpeg','/tmp/s-203.jpg',1),
		(999,'游离.jpg','c1.jpg','image/jpeg','/tmp/s-999.jpg',1)`)
	db.MustExec(`INSERT INTO personnel (id, name, department, photo_file_id) VALUES
		(1,'张甲','甲检测站',101),
		(2,'张乙','乙检测站',NULL)`)
	db.MustExec(`INSERT INTO equipment (id, name, serial_number, department) VALUES
		(1,'设备甲','SN-A','甲检测站'),
		(2,'设备乙','SN-B','乙检测站')`)
	db.MustExec(`INSERT INTO personnel_certificates (id, personnel_id, cert_name, file_id) VALUES
		(201,1,'身份证',201),
		(202,2,'身份证',202)`)
	db.MustExec(`INSERT INTO equipment_certificates (id, equipment_id, cert_type, expiry_date, file_id) VALUES
		(203,2,'检定','2030-01-01',203)`)
}

func TestPersonnelScopeIsolation(t *testing.T) {
	db := openCertClearTestDB(t)
	seedScopeFixture(t, db)
	svc := &PersonnelService{DB: db}

	// 受限账号：只能看「甲检测站」
	jia := scope.New("user", []string{"甲检测站"})
	all := scope.All()

	// ---- 列表 ----
	list, total, err := svc.List(jia, "", "", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Name != "张甲" {
		t.Fatalf("受限列表应只含甲检测站 1 人，实际 total=%d list=%v", total, names(list))
	}
	// 不受限：两人都能看到
	if _, total, _ = svc.List(all, "", "", "", 1, 20); total != 2 {
		t.Fatalf("不受限应看到 2 人，实际 %d", total)
	}
	// 受限账号**传别家公司做筛选也无效**（请求参数不能扩大范围）
	if _, total, _ = svc.List(jia, "", "乙检测站", "", 1, 20); total != 0 {
		t.Fatalf("受限账号按乙检测站筛选应得 0 条，实际 %d", total)
	}

	// ---- 下拉选项 ----
	opts, err := svc.Options(jia)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 || opts[0] != "甲检测站" {
		t.Fatalf("受限下拉应只有甲检测站，实际 %v", opts)
	}

	// ---- 详情越权 ----
	if _, err := svc.GetByID(jia, 2); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("读取乙检测站人员应被拒，实际 err=%v", err)
	}
	if _, err := svc.GetByID(jia, 1); err != nil {
		t.Fatalf("读取本公司人员应成功，实际 err=%v", err)
	}

	// ---- 改/删越权 ----
	if err := svc.Update(jia, 2, map[string]interface{}{"name": "改名", "department": "乙检测站"}); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("修改乙检测站人员应被拒，实际 err=%v", err)
	}
	if err := svc.Delete(jia, 2); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("删除乙检测站人员应被拒，实际 err=%v", err)
	}

	// ---- 新建：不能建到别家公司，也不能建"无公司" ----
	if _, err := svc.Create(jia, map[string]interface{}{"name": "新人", "gender": "男", "status": "active", "department": "乙检测站"}, 1); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("建到乙检测站应被拒，实际 err=%v", err)
	}
	if _, err := svc.Create(jia, map[string]interface{}{"name": "新人", "gender": "男", "status": "active"}, 1); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("建无公司人员应被拒（会导致创建者自己都看不到），实际 err=%v", err)
	}
	if _, err := svc.Create(jia, map[string]interface{}{"name": "新人", "gender": "男", "status": "active", "department": "甲检测站"}, 1); err != nil {
		t.Fatalf("建到本公司应成功，实际 err=%v", err)
	}

	// ---- 把数据改到自己看不见的公司 = 变相弄丢数据 → 拒绝 ----
	if err := svc.Update(jia, 1, map[string]interface{}{"name": "张甲", "department": "乙检测站"}); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("把人员改到无权公司应被拒，实际 err=%v", err)
	}

	// ---- 证件越权（按证件 ID 反查归属）----
	if err := svc.DeleteCert(jia, 202); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("删除乙检测站人员的证件应被拒，实际 err=%v", err)
	}
	if err := svc.UpdateCert(jia, 202, map[string]interface{}{"certName": "身份证"}); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("修改乙检测站人员的证件应被拒，实际 err=%v", err)
	}
	if err := svc.ClearReservedCert(jia, 202); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("清空乙检测站人员的预留证件应被拒，实际 err=%v", err)
	}

	// ---- 范围导出/查漏也应被收窄 ----
	// 注意：上面「建到本公司应成功」已新增 1 人，所以这里是 2 人；关键是**不能含乙检测站的人**
	people, err := svc.PersonnelByScope(jia, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 2 {
		t.Fatalf("PersonnelByScope 应含甲检测站 2 人，实际 %v", people)
	}
	for _, p := range people {
		if p.Department != nil && *p.Department == "乙检测站" {
			t.Fatalf("PersonnelByScope 不应含乙检测站的人，实际 %v", people)
		}
	}
	if files, err := svc.CertFilesByScope(jia, "", "", "身份证"); err != nil || len(files) != 1 {
		t.Fatalf("CertFilesByScope 应只含本公司 1 个文件，实际 %d err=%v", len(files), err)
	}
}

func TestEquipmentScopeIsolation(t *testing.T) {
	db := openCertClearTestDB(t)
	seedScopeFixture(t, db)
	svc := &EquipmentService{DB: db}

	jia := scope.New("user", []string{"甲检测站"})
	all := scope.All()

	list, total, err := svc.List(jia, "", "", "", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Name != "设备甲" {
		t.Fatalf("受限列表应只含甲检测站 1 台，实际 total=%d", total)
	}
	if _, total, _ = svc.List(all, "", "", "", "", 1, 20); total != 2 {
		t.Fatalf("不受限应看到 2 台，实际 %d", total)
	}

	depts, lines, err := svc.Options(jia)
	if err != nil {
		t.Fatal(err)
	}
	if len(depts) != 1 || depts[0] != "甲检测站" {
		t.Fatalf("受限下拉应只有甲检测站，实际 %v", depts)
	}
	if _, ok := lines["乙检测站"]; ok {
		t.Fatalf("受限时不应出现乙检测站的检测线，实际 %v", lines)
	}

	if _, err := svc.GetByID(jia, 2); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("读取乙检测站设备应被拒，实际 err=%v", err)
	}
	if err := svc.Delete(jia, 2); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("删除乙检测站设备应被拒，实际 err=%v", err)
	}
	if _, err := svc.Create(jia, map[string]interface{}{"name": "新设备", "serialNumber": "SN-N", "department": "乙检测站"}); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("建到乙检测站应被拒，实际 err=%v", err)
	}
	if err := svc.DeleteCert(jia, 203); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("删除乙检测站设备证件应被拒，实际 err=%v", err)
	}
}

// TestFileScopeIsolation 文件访问的归属校验 —— 不补这条就能"知道 file id 就下载别家公司的身份证"。
func TestFileScopeIsolation(t *testing.T) {
	db := openCertClearTestDB(t)
	seedScopeFixture(t, db)
	svc := &FileService{DB: db}

	jia := scope.New("user", []string{"甲检测站"})
	all := scope.All()

	cases := []struct {
		fileID int
		desc   string
		want   bool
	}{
		{101, "本公司人员头像", true},
		{201, "本公司人员证件", true},
		{202, "别家公司人员证件", false},
		{203, "别家公司设备证件", false},
		{999, "游离文件（没人引用）", false},
	}
	for _, c := range cases {
		got, err := svc.CanAccess(jia, c.fileID, 2)
		if err != nil {
			t.Fatalf("%s: %v", c.desc, err)
		}
		if got != c.want {
			t.Errorf("受限账号访问「%s」(id=%d) 期望 %v，实际 %v", c.desc, c.fileID, c.want, got)
		}
	}

	// 不受限：业务实体引用的文件都可访问（游离文件仍拦着，防猜 id 遍历）
	for _, c := range cases {
		got, err := svc.CanAccess(all, c.fileID, 2)
		if err != nil {
			t.Fatal(err)
		}
		want := c.fileID != 999
		if got != want {
			t.Errorf("不受限访问「%s」期望 %v，实际 %v", c.desc, want, got)
		}
	}

	// 自己上传的游离文件仍可访问（上传后尚未挂到业务实体时的正常场景）
	if ok, err := svc.CanAccess(jia, 999, 2); err != nil || ok {
		t.Fatalf("他人上传的游离文件应被拒，实际 ok=%v err=%v", ok, err)
	}
	db.MustExec(`UPDATE uploaded_files SET uploaded_by=2 WHERE id=999`)
	if ok, err := svc.CanAccess(scope.New("user", []string{"甲检测站"}), 999, 2); err != nil || !ok {
		t.Fatalf("自己上传的游离文件应可访问，实际 ok=%v err=%v", ok, err)
	}
}

func names(list []models.Personnel) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, p.Name)
	}
	return out
}
