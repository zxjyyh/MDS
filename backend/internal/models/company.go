package models

// Company 公司信息（甲方主体资料）：用于劳动合同、报名材料等对外文书自动填充。
type Company struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"` // 公司名

	CreditCode     *string `db:"credit_code" json:"creditCode"`          // 营业执照统一社会信用代码
	QualCertNo     *string `db:"qual_cert_no" json:"qualCertNo"`         // 资质证书编号
	QualCertExpiry *string `db:"qual_cert_expiry" json:"qualCertExpiry"` // 资质证书到期时间（YYYY-MM-DD）
	Address        *string `db:"address" json:"address"`                 // 公司地址（劳动合同等的甲方地址来源）

	Phone *string `db:"phone" json:"phone"` // 公司电话

	BankAccount *string `db:"bank_account" json:"bankAccount"` // 对公银行账号
	BankBranch  *string `db:"bank_branch" json:"bankBranch"`   // 开户行地址

	CreatedAt string `db:"created_at" json:"createdAt"`
	UpdatedAt string `db:"updated_at" json:"updatedAt"`
}
