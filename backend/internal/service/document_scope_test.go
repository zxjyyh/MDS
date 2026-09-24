package service

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/zxjyyh/mds/internal/scope"
)

// seedDocScopeFixture 构造资料库数据：
//
//	分类 1 = company_archive（公司档案）  分类 2 = legal_standard（法规标准）
//	文档 1 = 甲检测站的公司档案（有公司名）  文档 2 = 乙检测站的公司档案
//	文档 3 = 法规标准（全局共享，无公司名）
func seedDocScopeFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`INSERT INTO users (id, username, password_hash, real_name, email) VALUES
		(1,'admin','x','管理员','a@x.test')`)
	// 注意：AutoMigrate 会预置「公司档案 / 法规标准」的默认分类（id 1..9），
	// 所以测试数据用高位 id，避免主键冲突。
	db.MustExec(`INSERT INTO doc_categories (id, system_type, name) VALUES
		(901,'company_archive','测试-营业执照'),
		(902,'legal_standard','测试-公安专网标准')`)
	db.MustExec(`INSERT INTO uploaded_files (id, original_name, stored_name, mime_type, storage_path, uploaded_by) VALUES
		(801,'甲营业执照.jpg','d1.jpg','image/jpeg','/tmp/doc-801.jpg',1),
		(802,'乙营业执照.jpg','d2.jpg','image/jpeg','/tmp/doc-802.jpg',1),
		(803,'公安专网标准.pdf','d3.pdf','application/pdf','/tmp/doc-803.pdf',1)`)
	db.MustExec(`INSERT INTO documents (id, category_id, system_type, title, company_name, file_id) VALUES
		(901,901,'company_archive','甲公司营业执照','甲检测站',801),
		(902,901,'company_archive','乙公司营业执照','乙检测站',802),
		(903,902,'legal_standard','公安专网标准汇编',NULL,803)`)
}

func TestDocumentScopeIsolation(t *testing.T) {
	db := openCertClearTestDB(t)
	seedDocScopeFixture(t, db)
	svc := &DocumentService{DB: db}

	jia := scope.New("user", []string{"甲检测站"})
	all := scope.All()

	// ---- 列表：公司档案按公司过滤 ----
	list, total, err := svc.List(jia, "company_archive", nil, "", "", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Title != "甲公司营业执照" {
		t.Fatalf("受限列表应只含甲检测站的公司档案，实际 total=%d", total)
	}
	if _, total, _ = svc.List(all, "company_archive", nil, "", "", 1, 50); total != 2 {
		t.Fatalf("不受限应看到 2 份公司档案，实际 %d", total)
	}

	// ---- 列表：法规标准全局共享（即使 company_name 为空也要能看到）----
	list2, total2, err := svc.List(jia, "legal_standard", nil, "", "", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total2 != 1 || len(list2) != 1 {
		t.Fatalf("法规标准应全局可见（不受公司限制），实际 total=%d", total2)
	}

	// ---- 列表：不指定类型时，应为「本公司的 + 法规标准」----
	_, totalAll, err := svc.List(jia, "", nil, "", "", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if totalAll != 2 {
		t.Fatalf("不指定类型时应为 1 条本公司 + 1 条法规标准 = 2，实际 %d", totalAll)
	}

	// ---- 详情越权 ----
	if _, err := svc.GetByID(jia, 902); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("读取乙检测站的公司档案应被拒，实际 err=%v", err)
	}
	if _, err := svc.GetByID(jia, 901); err != nil {
		t.Fatalf("读取本公司档案应成功，实际 err=%v", err)
	}
	if _, err := svc.GetByID(jia, 903); err != nil {
		t.Fatalf("读取法规标准应成功（全局共享），实际 err=%v", err)
	}

	// ---- 改/删越权 ----
	if err := svc.Delete(jia, 902); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("删除乙检测站档案应被拒，实际 err=%v", err)
	}
	if err := svc.Delete(jia, 903); err != nil {
		t.Fatalf("删除法规标准不应被公司隔离挡住（是否允许是另一回事），实际 err=%v", err)
	}

	// ---- 新建：非全局必须归入可见公司 ----
	if err := svc.Create(jia, map[string]interface{}{
		"categoryId": 901, "title": "新档案", "companyName": "乙检测站",
	}, 1); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("建到乙检测站应被拒，实际 err=%v", err)
	}
	if err := svc.Create(jia, map[string]interface{}{
		"categoryId": 901, "title": "新档案",
	}, 1); !errors.Is(err, scope.ErrForbidden) {
		t.Fatalf("公司为空应被拒（否则新文档对自己也不可见），实际 err=%v", err)
	}
	if err := svc.Create(jia, map[string]interface{}{
		"categoryId": 901, "title": "新档案", "companyName": "甲检测站",
	}, 1); err != nil {
		t.Fatalf("建到本公司应成功，实际 err=%v", err)
	}
	// 法规标准（全局）不要求公司
	if err := svc.Create(jia, map[string]interface{}{
		"categoryId": 902, "title": "新标准",
	}, 1); err != nil {
		t.Fatalf("法规标准不要求公司，应成功，实际 err=%v", err)
	}
}

// TestDocFileScopeIsolation 资料库**附件**的跨公司下载也必须被挡住。
//
// 背景：文档列表被隔离后，如果文件访问这条路径不隔离，受限账号依然能靠 file id
// 直接下载别家公司公司档案的扫描件（营业执照 / 法人身份证），列表隔离就形同虚设。
func TestDocFileScopeIsolation(t *testing.T) {
	db := openCertClearTestDB(t)
	seedDocScopeFixture(t, db)
	fsvc := &FileService{DB: db}

	jia := scope.New("user", []string{"甲检测站"})

	cases := []struct {
		fileID int
		desc   string
		want   bool
	}{
		{801, "本公司公司档案的附件", true},
		{802, "别家公司公司档案的附件", false},
		{803, "法规标准的附件（全局共享，应可访问）", true},
	}
	for _, c := range cases {
		// 用 userID=2（非上传者，上传者是 1）以避开「自己上传的」那条放行规则
		got, err := fsvc.CanAccess(jia, c.fileID, 2)
		if err != nil {
			t.Fatalf("%s: %v", c.desc, err)
		}
		if got != c.want {
			t.Errorf("受限账号访问「%s」(file_id=%d) 期望 %v，实际 %v", c.desc, c.fileID, c.want, got)
		}
	}
}

// TestDocIsGlobal 全局文档类型的判定。
func TestDocIsGlobal(t *testing.T) {
	if !DocIsGlobal(DocSystemLegalStandard) {
		t.Error("法规标准应为全局共享")
	}
	for _, st := range []string{DocSystemCompanyArchive, DocSystemQualification, DocSystemSafety, DocSystemIP} {
		if DocIsGlobal(st) {
			t.Errorf("%s 不应为全局（需按公司隔离）", st)
		}
	}
}
