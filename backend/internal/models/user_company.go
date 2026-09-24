package models

// UserCompany 账号可见公司（数据隔离）。
//
// 存公司名而不是 company_id：personnel.department 与 equipment.department 存的就是公司名，
// 过滤时按名称精确匹配，保持一致（见 docs/company-data-scope.md §3.1）。
type UserCompany struct {
	ID          int    `db:"id" json:"id"`
	UserID      int    `db:"user_id" json:"userId"`
	CompanyName string `db:"company_name" json:"companyName"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
}
