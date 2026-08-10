package models

type ReminderRule struct {
	ID         int    `db:"id" json:"id"`
	Module     string `db:"module" json:"module"`
	DaysBefore int    `db:"days_before" json:"daysBefore"`
	Enabled    bool   `db:"enabled" json:"enabled"`
	CreatedAt  string `db:"created_at" json:"createdAt"`
	UpdatedAt  string `db:"updated_at" json:"updatedAt"`
}

type ReminderLog struct {
	ID         int     `db:"id" json:"id"`
	Module     string  `db:"module" json:"module"`
	TargetName *string `db:"target_name" json:"targetName"`
	ExpiryDate *string `db:"expiry_date" json:"expiryDate"`
	DaysBefore *int    `db:"days_before" json:"daysBefore"`
	Status     string  `db:"status" json:"status"`
	Recipients *string `db:"recipients" json:"recipients"`
	ErrorMsg   *string `db:"error_msg" json:"errorMsg"`
	SentAt     string  `db:"sent_at" json:"sentAt"`
	CreatedAt  string  `db:"created_at" json:"createdAt"`
}
