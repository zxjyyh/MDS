package models

type DocCategory struct {
	ID         int           `db:"id" json:"id"`
	ParentID   *int          `db:"parent_id" json:"parentId"`
	Name       string        `db:"name" json:"name"`
	SystemType string        `db:"system_type" json:"systemType"`
	SortOrder  int           `db:"sort_order" json:"sortOrder"`
	CreatedAt  string        `db:"created_at" json:"createdAt"`
	Children   []DocCategory `db:"-" json:"children,omitempty"`
}
