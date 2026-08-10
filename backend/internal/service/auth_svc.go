package service

import (
	"errors"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/jmoiron/sqlx"
)

type AuthService struct{ DB *sqlx.DB }

func (s *AuthService) Login(username, password string) (string, string, *models.User, error) {
	var user models.User
	err := s.DB.Get(&user, `SELECT * FROM users WHERE username=? AND status='active'`, username)
	if err != nil {
		return "", "", nil, errors.New("用户名或密码错误")
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return "", "", nil, errors.New("用户名或密码错误")
	}

	_, _ = s.DB.Exec(`UPDATE users SET last_login_at=? WHERE id=?`, utils.NowString(), user.ID)

	cfg := GetConfig()
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.Role, user.TokenEpoch, cfg.JWTSecret)
	if err != nil {
		return "", "", nil, err
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Username, user.Role, user.TokenEpoch, cfg.JWTRefresh)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, &user, nil
}

func (s *AuthService) Refresh(tokenStr string) (string, string, error) {
	cfg := GetConfig()
	claims, err := utils.ParseRefreshToken(tokenStr, cfg.JWTRefresh)
	if err != nil {
		return "", "", errors.New("refresh token 无效")
	}
	if claims.Typ != utils.TokenTypeAccess {
		return "", "", errors.New("refresh token 无效")
	}

	var user models.User
	err = s.DB.Get(&user, `SELECT * FROM users WHERE id=? AND status='active'`, claims.UserID)
	if err != nil {
		return "", "", errors.New("用户不存在")
	}
	// refresh 同样受 token_epoch 约束，否则改密后仍能用旧 refresh token 续命
	if claims.Epoch != user.TokenEpoch {
		return "", "", utils.ErrTokenRevoked
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, user.Role, user.TokenEpoch, cfg.JWTSecret)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Username, user.Role, user.TokenEpoch, cfg.JWTRefresh)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// IssueFileToken 签发只能用于文件预览 / 下载的短时效令牌（30 分钟）。
func (s *AuthService) IssueFileToken(userID int) (string, int, error) {
	var user models.User
	if err := s.DB.Get(&user, `SELECT * FROM users WHERE id=? AND status='active'`, userID); err != nil {
		return "", 0, errors.New("用户不存在")
	}
	cfg := GetConfig()
	token, err := utils.GenerateFileToken(user.ID, user.Username, user.Role, user.TokenEpoch, cfg.JWTSecret)
	if err != nil {
		return "", 0, err
	}
	return token, int(utils.FileTokenTTL.Seconds()), nil
}

func (s *AuthService) GetProfile(userID int) (*models.User, error) {
	var user models.User
	err := s.DB.Get(&user, `SELECT id, username, real_name, email, role, status, last_login_at, created_at, updated_at FROM users WHERE id=?`, userID)
	return &user, err
}

// ChangePassword 修改密码，并吊销该用户此前签发的全部令牌。
func (s *AuthService) ChangePassword(userID int, oldPwd, newPwd string) error {
	var hash string
	err := s.DB.Get(&hash, `SELECT password_hash FROM users WHERE id=?`, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	if !utils.CheckPassword(oldPwd, hash) {
		return errors.New("原密码错误")
	}
	if oldPwd == newPwd {
		return errors.New("新密码不能与原密码相同")
	}

	newHash, err := utils.HashPassword(newPwd)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(
		`UPDATE users SET password_hash=?, token_epoch=token_epoch+1, updated_at=? WHERE id=?`,
		newHash, utils.NowString(), userID)
	return err
}

// Logout 自增 token_epoch，使该用户当前持有的 access / refresh / file 令牌全部立即失效。
func (s *AuthService) Logout(userID int) error {
	_, err := s.DB.Exec(`UPDATE users SET token_epoch=token_epoch+1 WHERE id=?`, userID)
	return err
}
