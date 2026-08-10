package models

type Document struct {
	ID          int     `db:"id" json:"id"`
	CategoryID  int     `db:"category_id" json:"categoryId"`
	SystemType  string  `db:"system_type" json:"systemType"`
	Title       string  `db:"title" json:"title"`
	DocNumber   *string `db:"doc_number" json:"docNumber"`
	Version     *string `db:"version" json:"version"`
	IssueDate   *string `db:"issue_date" json:"issueDate"`
	ExpiryDate  *string `db:"expiry_date" json:"expiryDate"`
	Description *string `db:"description" json:"description"`
	FileID      *int    `db:"file_id" json:"fileId"`
	CreatedByID *int    `db:"created_by" json:"createdById"`
	CreatedAt   string  `db:"created_at" json:"createdAt"`
	UpdatedAt   string  `db:"updated_at" json:"updatedAt"`

	// joined
	Category  *DocCategory  `db:"-" json:"category,omitempty"`
	File      *UploadedFile `db:"-" json:"file,omitempty"`
	CreatedBy *User         `db:"-" json:"createdBy,omitempty"`
}
