package models

// PersonnelReservedCertNames 每位人员固定预留的两个证件名称。
// 预留行只有固定名称，编号/到期日期/附件等字段留空，由操作者上传证书后手动填写；
// 名称即锚点：不可删除、不可改名，且同人不可重复添加同名证件。
var PersonnelReservedCertNames = []string{"安检证", "环保证"}

// IsPersonnelReservedCertName 判断证件名是否为固定预留项。
func IsPersonnelReservedCertName(name string) bool {
	for _, n := range PersonnelReservedCertNames {
		if name == n {
			return true
		}
	}
	return false
}

// Count 是前端期望的嵌套 _count 对象
type Count struct {
	Certificates int `json:"certificates"`
}

// Personnel 人员表
type Personnel struct {
	ID                    int     `db:"id" json:"id"`
	Name                  string  `db:"name" json:"name"`
	Gender                string  `db:"gender" json:"gender"`
	IDCard                *string `db:"id_card" json:"idCard"`
	Phone                 *string `db:"phone" json:"phone"`
	Position              *string `db:"position" json:"position"`
	Department            *string `db:"department" json:"department"`
	Address               *string `db:"address" json:"address"`
	EntryDate             *string `db:"entry_date" json:"entryDate"`
	Education             *string `db:"education" json:"education"`
	School                *string `db:"school" json:"school"`
	Major                 *string `db:"major" json:"major"`
	DriverLicense         *string `db:"driver_license" json:"driverLicense"`
	DriverLicenseExpiry   *string `db:"driver_license_expiry" json:"driverLicenseExpiry"`
	IDCardExpiry          *string `db:"id_card_expiry" json:"idCardExpiry"`
	GraduationDate        *string `db:"graduation_date" json:"graduationDate"`
	SecurityCertDate      *string `db:"security_cert_date" json:"securityCertDate"`
	SecurityCertNumber    *string `db:"security_cert_number" json:"securityCertNumber"`
	SecurityPositions     *string `db:"security_positions" json:"securityPositions"`
	EnvCertDate           *string `db:"env_cert_date" json:"envCertDate"`
	EnvCertNumber         *string `db:"env_cert_number" json:"envCertNumber"`
	EnvPositions          *string `db:"env_positions" json:"envPositions"`
	InspectionDate        *string `db:"inspection_date" json:"inspectionDate"`
	InspectionCertNumber  *string `db:"inspection_cert_number" json:"inspectionCertNumber"`
	InternalAuditorNumber *string `db:"internal_auditor_number" json:"internalAuditorNumber"`
	InternalAuditorDate   *string `db:"internal_auditor_date" json:"internalAuditorDate"`
	Status                string  `db:"status" json:"status"`
	PhotoFileID           *int    `db:"photo_file_id" json:"photoFileId"`
	Remark                *string `db:"remark" json:"remark"`
	CreatedAt             string  `db:"created_at" json:"createdAt"`
	UpdatedAt             string  `db:"updated_at" json:"updatedAt"`
	CertCount             int     `db:"cert_count" json:"-"`

	// joined
	PhotoFile    *UploadedFile          `db:"-" json:"photoFile,omitempty"`
	Certificates []PersonnelCertificate `db:"-" json:"certificates,omitempty"`
	Count        *Count                 `db:"-" json:"_count,omitempty"`
}

type PersonnelCertificate struct {
	ID          int     `db:"id" json:"id"`
	PersonnelID int     `db:"personnel_id" json:"personnelId"`
	CertName    string  `db:"cert_name" json:"certName"`
	CertNumber  *string `db:"cert_number" json:"certNumber"`
	IssueDate   *string `db:"issue_date" json:"issueDate"`
	ExpiryDate  *string `db:"expiry_date" json:"expiryDate"`
	FileID      *int    `db:"file_id" json:"fileId"`
	Remark      *string `db:"remark" json:"remark"`
	CreatedAt   string  `db:"created_at" json:"createdAt"`
	UpdatedAt   string  `db:"updated_at" json:"updatedAt"`

	// joined
	Personnel *Personnel    `db:"-" json:"personnel,omitempty"`
	File      *UploadedFile `db:"-" json:"file,omitempty"`
}
