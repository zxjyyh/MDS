package models

// OperationLog 操作日志：记录「谁、什么时候、做了什么」，面向审计。
type OperationLog struct {
	ID int `db:"id" json:"id"`
	// 用户被删除时 user_svc 会把 user_id 置 NULL 以保留审计记录，故用指针
	UserID   *int    `db:"user_id" json:"userId"`
	Username *string `db:"username" json:"username"` // 操作者用户名快照
	Role     *string `db:"role" json:"role"`         // 操作者角色快照（admin / user）

	Action string  `db:"action" json:"action"` // 中文动作名，如「修改人员」
	Method *string `db:"method" json:"method"`
	Path   *string `db:"path" json:"path"`

	TargetType *string `db:"target_type" json:"targetType"`
	TargetID   *int    `db:"target_id" json:"targetId"`
	Detail     *string `db:"detail" json:"detail"`

	CreatedAt string `db:"created_at" json:"createdAt"`
}
