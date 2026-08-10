package models

// User 用户表
type User struct {
	ID           int     `db:"id" json:"id"`
	Username     string  `db:"username" json:"username"`
	PasswordHash string  `db:"password_hash" json:"-"`
	RealName     string  `db:"real_name" json:"realName"`
	Email        string  `db:"email" json:"email"`
	Role         string  `db:"role" json:"role"`
	Status       string  `db:"status" json:"status"`
	LastLoginAt  *string `db:"last_login_at" json:"lastLoginAt"`
	// TokenEpoch 是令牌世代号，改密 / 禁用 / 登出时自增以吊销旧令牌。
	// 注意：即使前端用不到也**必须**保留这个字段——auth_svc 里用的是
	// `SELECT * FROM users`，sqlx 找不到对应目标字段会直接返回错误。
	TokenEpoch int    `db:"token_epoch" json:"-"`
	CreatedAt  string `db:"created_at" json:"createdAt"`
	UpdatedAt  string `db:"updated_at" json:"updatedAt"`
}
