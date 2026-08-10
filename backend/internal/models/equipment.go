package models

type Equipment struct {
	ID           int     `db:"id" json:"id"`
	Name         string  `db:"name" json:"name"`
	SerialNumber string  `db:"serial_number" json:"serialNumber"`
	Department   *string `db:"department" json:"department"`
	Line         *string `db:"line" json:"line"`
	Model        *string `db:"model" json:"model"`
	Manufacturer *string `db:"manufacturer" json:"manufacturer"`
	Status       string  `db:"status" json:"status"`
	Remark       *string `db:"remark" json:"remark"`
	CreatedAt    string  `db:"created_at" json:"createdAt"`
	UpdatedAt    string  `db:"updated_at" json:"updatedAt"`

	// 独立证书字段（与上传证书表无关）
	InspectionCertNumber *string `db:"inspection_cert_number" json:"inspectionCertNumber"`
	InspectionDate       *string `db:"inspection_date" json:"inspectionDate"`
	InspectionExpiryDate *string `db:"inspection_expiry_date" json:"inspectionExpiryDate"`

	// associations
	Certificates []EquipmentCertificate `db:"-" json:"certificates,omitempty"`
	Count        *Count                 `db:"-" json:"_count,omitempty"`
}

type EquipmentCertificate struct {
	ID          int     `db:"id" json:"id"`
	EquipmentID int     `db:"equipment_id" json:"equipmentId"`
	CertType    string  `db:"cert_type" json:"certType"`
	CertNumber  *string `db:"cert_number" json:"certNumber"`
	IssueDate   *string `db:"issue_date" json:"issueDate"`
	ExpiryDate  *string `db:"expiry_date" json:"expiryDate"`
	FileID      *int    `db:"file_id" json:"fileId"`
	Remark      *string `db:"remark" json:"remark"`
	CreatedAt   string  `db:"created_at" json:"createdAt"`
	UpdatedAt   string  `db:"updated_at" json:"updatedAt"`

	// joined
	Equipment *Equipment    `db:"-" json:"equipment,omitempty"`
	File      *UploadedFile `db:"-" json:"file,omitempty"`
}
