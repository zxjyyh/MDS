package service

import (
	"errors"
	"strings"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/jmoiron/sqlx"
)

type UserService struct{ DB *sqlx.DB }

func (s *UserService) List(keyword, role string, page, pageSize int) ([]models.User, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if keyword != "" {
		where += " AND (username LIKE ? OR real_name LIKE ? OR email LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw, kw)
	}
	if role != "" {
		where += " AND role=?"
		args = append(args, role)
	}

	var total int
	if err := s.DB.Get(&total, "SELECT COUNT(*) FROM users "+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQ := `SELECT id, username, real_name, email, role, status, last_login_at, created_at, updated_at FROM users ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)
	var list []models.User
	if err := s.DB.Select(&list, selectQ, args...); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *UserService) GetByID(id int) (*models.User, error) {
	var u models.User
	err := s.DB.Get(&u, `SELECT id, username, real_name, email, role, status, last_login_at, created_at, updated_at FROM users WHERE id=?`, id)
	return &u, err
}

func (s *UserService) Create(data models.User) (*models.User, error) {
	var count int
	s.DB.Get(&count, "SELECT COUNT(*) FROM users WHERE username=?", data.Username)
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}
	s.DB.Get(&count, "SELECT COUNT(*) FROM users WHERE email=?", data.Email)
	if count > 0 {
		return nil, errors.New("邮箱已存在")
	}

	hash, _ := utils.HashPassword(data.PasswordHash)
	now := utils.NowString()
	res, err := s.DB.Exec(`INSERT INTO users (username, password_hash, real_name, email, role, status, created_at, updated_at) 
		VALUES (?,?,?,?,?,?,?,?)`,
		data.Username, hash, data.RealName, data.Email, data.Role, data.Status, now, now)
	if err != nil {
		return nil, err
	}
	// 返回含新 id 的用户，供调用方继续配置「公司权限」等关联数据
	if id, idErr := res.LastInsertId(); idErr == nil {
		data.ID = int(id)
	}
	data.CreatedAt, data.UpdatedAt = now, now
	data.PasswordHash = "" // 不外传
	return &data, nil
}

// Update 修改用户资料。username 是登录名，允许修改（含 admin 账号），
// 但必须全局唯一 —— 这里先显式查一次，否则会直接撞 users.username 的 UNIQUE 约束，
// 前端只能看到一句 SQLite 的英文报错。
func (s *UserService) Update(id int, username, realName, email, role string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("用户名不能为空")
	}
	var count int
	if err := s.DB.Get(&count, "SELECT COUNT(*) FROM users WHERE username=? AND id<>?", username, id); err != nil {
		return err
	}
	if count > 0 {
		return errors.New("用户名已存在")
	}
	// email 同样有 UNIQUE 约束，一并给出中文提示（顺手补上，原逻辑缺这一层）
	if err := s.DB.Get(&count, "SELECT COUNT(*) FROM users WHERE email=? AND id<>?", email, id); err != nil {
		return err
	}
	if count > 0 {
		return errors.New("邮箱已存在")
	}

	_, err := s.DB.Exec(`UPDATE users SET username=?, real_name=?, email=?, role=?, updated_at=? WHERE id=?`,
		username, realName, email, role, utils.NowString(), id)
	return err
}

// Delete 删除用户。
//
// 数据库已开启 foreign_keys=ON，而以下 4 张表都引用 users(id) 且未声明 ON DELETE
// 行为（等价于 RESTRICT）。直接 DELETE 会报 FOREIGN KEY constraint failed，
// 所以必须在同一个事务里先处理引用：
//
//	uploaded_files.uploaded_by   NOT NULL → 移交给操作人
//	documents.created_by         可空     → 置 NULL
//	operation_logs.user_id       可空     → 置 NULL（保留审计记录）
//	ignored_warnings.ignored_by  NOT NULL → 属个人偏好，直接删除
func (s *UserService) Delete(id, operatorID int) error {
	var target struct {
		Username string `db:"username"`
		Role     string `db:"role"`
		Status   string `db:"status"`
	}
	if err := s.DB.Get(&target, "SELECT username, role, status FROM users WHERE id=?", id); err != nil {
		return errors.New("用户不存在")
	}
	if target.Username == "admin" {
		return errors.New("不能删除超级管理员")
	}
	if id == operatorID {
		return errors.New("不能删除当前登录的账号")
	}
	// 用户名改成别的之后，上面按 "admin" 的判断就失效了，所以再按角色兜一道：
	// 系统必须始终保留至少一个启用状态的管理员，否则会把自己锁在门外
	if target.Role == "admin" && target.Status == "active" {
		var admins int
		if err := s.DB.Get(&admins, "SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'"); err != nil {
			return err
		}
		if admins <= 1 {
			return errors.New("系统必须保留至少一个管理员账号")
		}
	}

	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 上传记录不能丢（uploads 目录里还有实体文件），移交给执行删除的管理员
	if _, err := tx.Exec("UPDATE uploaded_files SET uploaded_by=? WHERE uploaded_by=?", operatorID, id); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE documents SET created_by=NULL WHERE created_by=?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE operation_logs SET user_id=NULL WHERE user_id=?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM ignored_warnings WHERE ignored_by=?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM users WHERE id=?", id); err != nil {
		return err
	}
	return tx.Commit()
}

// ResetPassword 重置密码，并递增 token_epoch 让该用户此前签发的所有令牌立即失效。
func (s *UserService) ResetPassword(id int, newPwd string) error {
	if err := utils.ValidatePassword(newPwd); err != nil {
		return err
	}
	hash, err := utils.HashPassword(newPwd)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(
		"UPDATE users SET password_hash=?, token_epoch=token_epoch+1, updated_at=? WHERE id=?",
		hash, utils.NowString(), id)
	return err
}

// SetStatus 启用/禁用账号。禁用时同样吊销已签发的令牌，否则对方还能继续用旧 token。
func (s *UserService) SetStatus(id int, status string) error {
	if status == "disabled" {
		_, err := s.DB.Exec(
			"UPDATE users SET status=?, token_epoch=token_epoch+1, updated_at=? WHERE id=?",
			status, utils.NowString(), id)
		return err
	}
	_, err := s.DB.Exec("UPDATE users SET status=?, updated_at=? WHERE id=?", status, utils.NowString(), id)
	return err
}

// BumpTokenEpoch 主动吊销某个用户当前所有令牌（登出、改密时调用）。
func (s *UserService) BumpTokenEpoch(id int) error {
	_, err := s.DB.Exec("UPDATE users SET token_epoch=token_epoch+1 WHERE id=?", id)
	return err
}
