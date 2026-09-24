package scope

import "testing"

func TestNewRules(t *testing.T) {
	cases := []struct {
		name             string
		role             string
		companies        []string
		wantUnrestricted bool
	}{
		{"管理员不受限", "admin", nil, true},
		{"管理员即使配了公司也不受限", "admin", []string{"A公司"}, true},
		{"普通用户未配公司 → 不受限（渐进启用）", "user", nil, true},
		{"普通用户只配了空串 → 不受限", "user", []string{"", "   "}, true},
		{"普通用户配了公司 → 受限", "user", []string{"A公司"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := New(tc.role, tc.companies)
			if sc.Unrestricted != tc.wantUnrestricted {
				t.Fatalf("Unrestricted 期望 %v，实际 %v", tc.wantUnrestricted, sc.Unrestricted)
			}
		})
	}
}

func TestAllows(t *testing.T) {
	sc := New("user", []string{"A公司", "B公司"})
	cases := []struct {
		company string
		want    bool
	}{
		{"A公司", true},
		{"B公司", true},
		{"C公司", false},
		// 空公司名对受限账号一律不可见 —— 归属不明的数据不能从后门漏出去
		{"", false},
		{"   ", false},
	}
	for _, c := range cases {
		if got := sc.Allows(c.company); got != c.want {
			t.Errorf("受限时 Allows(%q) 期望 %v，实际 %v", c.company, c.want, got)
		}
	}

	// 不受限：一律允许（含空 —— 管理员要能看到归属为空的记录）
	for _, c := range []string{"A公司", "", "任意公司"} {
		if !All().Allows(c) {
			t.Errorf("不受限时 Allows(%q) 应为 true", c)
		}
	}
}

func TestFilterAndDefensive(t *testing.T) {
	sc := New("user", []string{" A公司 ", "A公司", "B公司", ""})
	f := sc.Filter()
	if len(f) != 2 {
		t.Fatalf("去空去重后期望 2 个公司，实际 %v", f)
	}
	if f[0] != "A公司" || f[1] != "B公司" {
		t.Errorf("应保留 trim 后的原值，实际 %v", f)
	}

	if All().Filter() != nil {
		t.Error("不受限时 Filter 应返回 nil（语义：不加任何条件）")
	}

	// nil 接收者按「不受限」处理（防御：避免调用方漏判空指针炸掉）
	var n *Scope
	if !n.Allows("任意") {
		t.Error("nil Scope 应视为不受限")
	}
	if n.Filter() != nil {
		t.Error("nil Scope 的 Filter 应为 nil")
	}
}

// TestSQLFilter SQL 过滤片段的生成 —— 这是所有列表查询的隔离入口，必须精确。
func TestSQLFilter(t *testing.T) {
	// 不受限：不加任何条件
	if frag, args := All().SQLFilter("p.department"); frag != "" || args != nil {
		t.Errorf("不受限应返回空片段与 nil 参数，实际 frag=%q args=%v", frag, args)
	}

	// 受限：占位符个数必须与参数个数一致（对不上会导致参数错位、查询结果出错）
	sc := New("user", []string{"甲", "乙", "丙"})
	frag, args := sc.SQLFilter("p.department")
	want := " AND p.department IN (?,?,?)"
	if frag != want {
		t.Errorf("片段期望 %q，实际 %q", want, frag)
	}
	if len(args) != 3 {
		t.Fatalf("参数个数应与占位符一致，期望 3，实际 %d: %v", len(args), args)
	}
	for i, w := range []string{"甲", "乙", "丙"} {
		if args[i] != w {
			t.Errorf("第 %d 个参数期望 %q，实际 %v", i, w, args[i])
		}
	}

	// 单个公司
	frag1, args1 := New("user", []string{"只有一家"}).SQLFilter("e.department")
	if frag1 != " AND e.department IN (?)" || len(args1) != 1 {
		t.Errorf("单公司片段不对: %q %v", frag1, args1)
	}
}
