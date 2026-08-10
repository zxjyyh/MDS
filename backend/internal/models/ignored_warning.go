package models

// IgnoredWarning 已忽略预警项
type IgnoredWarning struct {
	ID          int    `db:"id" json:"id"`
	Module      string `db:"module" json:"module"`
	TargetName  string `db:"target_name" json:"targetName"`
	RelatedName string `db:"related_name" json:"relatedName"`
	ExpiryDate  string `db:"expiry_date" json:"expiryDate"`
	IgnoredBy   int    `db:"ignored_by" json:"ignoredBy"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
}
