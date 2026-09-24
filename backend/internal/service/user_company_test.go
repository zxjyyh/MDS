package service

import (
	"testing"
)

// TestUserCompanies 账号可见公司（数据隔离）的读写与校验。
func TestUserCompanies(t *testing.T) {
	db := openCertClearTestDB(t)
	db.MustExec(`INSERT INTO users (id, username, password_hash, real_name, email) VALUES
		(1,'admin','x','管理员','a@x.test'),
		(2,'u1','x','甲','u1@x.test'),
		(3,'u2','x','乙','u2@x.test')`)
	// 公司档案里登记了 1 家；人员数据里出现了 1 家**未登记**的公司
	// （模拟"某公司只在人员数据里、公司档案中没有"的历史情况）
	db.MustExec(`INSERT INTO companies (name) VALUES ('已登记公司')`)
	db.MustExec(`INSERT INTO personnel (name, department) VALUES ('张三','仅数据里出现的公司')`)

	companySvc := &CompanyService{DB: db}
	svc := &UserCompanyService{DB: db, CompanySvc: companySvc}

	// ---- 合法名集合 = 公司档案 ∪ 实际出现过的检测站 ----
	valid, err := companySvc.ValidNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"已登记公司", "仅数据里出现的公司"} {
		if !valid[n] {
			t.Errorf("ValidNames 应包含 %q（并集），实际 %v", n, valid)
		}
	}
	if valid["不存在的公司"] {
		t.Error("ValidNames 不应包含从未出现过的名称")
	}

	names, err := companySvc.NameOptions()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("NameOptions 期望 2 个，实际 %v", names)
	}
	if names[0] != "已登记公司" {
		t.Errorf("公司档案里的应排在前面，实际 %v", names)
	}

	// ---- 写入 + 读回（含未登记但有数据的公司名）----
	if err := svc.SetCompanies(2, []string{"已登记公司", "仅数据里出现的公司"}); err != nil {
		t.Fatalf("SetCompanies 失败: %v", err)
	}
	got, err := svc.CompaniesOf(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("期望 2 个公司，实际 %v", got)
	}

	// ---- 覆盖式：再设一次只留一个 ----
	if err := svc.SetCompanies(2, []string{"已登记公司"}); err != nil {
		t.Fatal(err)
	}
	if got, _ = svc.CompaniesOf(2); len(got) != 1 || got[0] != "已登记公司" {
		t.Fatalf("覆盖式设置失败，实际 %v", got)
	}

	// ---- 非法名称：应报错且**不写入任何配置** ----
	if err := svc.SetCompanies(3, []string{"不存在的公司"}); err == nil {
		t.Error("非法公司名应被拒绝（否则会导致受限账号静默看不到数据）")
	}
	if got, _ = svc.CompaniesOf(3); len(got) != 0 {
		t.Errorf("被拒绝后不应留下任何配置，实际 %v", got)
	}

	// ---- 批量取（用户列表用）----
	cm, err := svc.NamesOfUsers([]int{2, 3, 999})
	if err != nil {
		t.Fatal(err)
	}
	if len(cm[2]) != 1 {
		t.Errorf("账号 2 期望 1 个公司，实际 %v", cm[2])
	}
	if len(cm[3]) != 0 {
		t.Errorf("账号 3 期望 0 个公司，实际 %v", cm[3])
	}
	if cm[999] != nil {
		t.Errorf("不存在的账号不应有记录，实际 %v", cm[999])
	}

	// ---- 去重 ----
	if err := svc.SetCompanies(2, []string{"已登记公司", "已登记公司", "  "}); err != nil {
		t.Fatal(err)
	}
	if got, _ = svc.CompaniesOf(2); len(got) != 1 {
		t.Errorf("重复项与空串应被去掉，实际 %v", got)
	}

	// ---- 清空 = 恢复不受限 ----
	if err := svc.SetCompanies(2, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ = svc.CompaniesOf(2); len(got) != 0 {
		t.Errorf("清空失败，实际 %v", got)
	}

	// ---- 未注入 CompanySvc 时必须明确报错，而不是静默跳过校验 ----
	bare := &UserCompanyService{DB: db}
	if err := bare.SetCompanies(2, []string{"已登记公司"}); err == nil {
		t.Error("未注入 CompanySvc 时应返回错误（静默跳过校验会写入笔误名称）")
	}
}
