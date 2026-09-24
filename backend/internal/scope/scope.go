// Package scope 定义「当前账号能看到哪些公司的数据」这一横切关注点的模型与判定规则。
//
// 设计要点：
//
//  1. 判定规则只有一条：**管理员 或 未配置公司 → 不受限**，否则只能看所配公司。
//     「未配置即不受限」是刻意的，用于**渐进启用**：功能上线时没有任何账号配过公司，
//     因此所有人的行为与改造前完全一致；之后逐个账号配置、逐个生效，随时可清空配置解禁。
//
//  2. 用「参数强制」而不是「记得调用」：需要隔离的 service 方法把 *Scope 作为**必填参数**，
//     这样将来新增接口时漏传会**编译不过**，比靠代码评审记得检查可靠得多。
//
//  3. 受限时**空公司名一律不可见**（见 Allows），避免归属不明的数据从后门漏出去。
//
// 详细方案见 docs/company-data-scope.md。
package scope

import (
	"errors"
	"strings"
)

// ErrForbidden 资源不在当前账号的可见范围内。
// service 层返回它，handler 层据此回 403（而不是 404 —— 明确告知"无权限"比假装不存在更利于排查）。
var ErrForbidden = errors.New("无权访问该数据")

// Scope 当前账号的数据可见范围。
type Scope struct {
	// Unrestricted 为 true 表示不受任何公司限制（管理员，或未配置公司的账号）。
	Unrestricted bool
	// Companies 受限时可访问的公司名（非空）。Unrestricted 为 true 时无意义。
	Companies []string
}

// All 返回不受限的 scope。
func All() *Scope { return &Scope{Unrestricted: true} }

// New 按角色与已配置的公司名单构造 scope。
//
// role 为 "admin"、或公司名单去空去重后为空时，返回不受限。
func New(role string, companies []string) *Scope {
	if strings.TrimSpace(role) == "admin" {
		return All()
	}
	list := make([]string, 0, len(companies))
	seen := map[string]bool{}
	for _, c := range companies {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		list = append(list, c)
	}
	if len(list) == 0 {
		return All()
	}
	return &Scope{Companies: list}
}

// Allows 判断某个公司名是否在可见范围内。
//
// 受限时**空公司名返回 false** —— 归属不明的数据不能漏给受限账号。
func (s *Scope) Allows(company string) bool {
	if s == nil || s.Unrestricted {
		return true
	}
	c := strings.TrimSpace(company)
	if c == "" {
		return false
	}
	for _, x := range s.Companies {
		if x == c {
			return true
		}
	}
	return false
}

// Filter 返回受限时的公司白名单；不受限时返回 nil（语义：不加任何条件）。
func (s *Scope) Filter() []string {
	if s == nil || s.Unrestricted {
		return nil
	}
	return s.Companies
}

// Condition 生成 "<column> IN (?,?,…)"（**不带**前导 AND）；不受限时返回 ("1=1", nil)。
//
// 与 SQLFilter 的区别：SQLFilter 用于「已有 WHERE 首条件、往后追加」的场景（返回 " AND …"）；
// Condition 用于「需要单独一段条件」的场景 —— 例如仪表盘把结果包在外层子查询里再过滤：
//
//	SELECT COUNT(*) FROM ( … ) AS sub WHERE <Condition("sub.department")>
//
// 返回 "1=1" 而不是空串，是为了让调用方可以无脑拼 `WHERE <cond>` 而不必判空。
func (s *Scope) Condition(column string) (string, []any) {
	f := s.Filter()
	if len(f) == 0 {
		return "1=1", nil
	}
	var b strings.Builder
	b.WriteString(column)
	b.WriteString(" IN (")
	args := make([]any, 0, len(f))
	for i, c := range f {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
		args = append(args, c)
	}
	b.WriteString(")")
	return b.String(), args
}

// SQLFilter 生成 " AND <column> IN (?,?,…)" 片段与对应参数；不受限时返回 ("", nil)。
//
// 设计取舍：把占位符在 Go 里逐个拼出来，而不是用 sqlx.In 展开切片 ——
// 这样调用方只需 `where += frag; args = append(args, fargs...)`，
// 不必把已有查询改写成 sqlx.In 的两段式（本项目多处查询是动态拼 where 的，改写面太大）。
//
// 安全性：column 由调用方以**代码字面量**传入（如 "p.department"），不是用户输入；
// 所有公司名都走绑定参数，不存在注入。
func (s *Scope) SQLFilter(column string) (string, []any) {
	cond, args := s.Condition(column)
	if cond == "1=1" {
		return "", nil
	}
	return " AND " + cond, args
}
