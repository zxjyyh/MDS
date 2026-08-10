package utils

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt 的硬性上限：超过 72 字节的部分会被**静默截断**，
// 也就是说 "同前 72 字节 + 任意后缀" 的密码都能登录成功。必须显式拒绝。
const (
	MinPasswordLen = 6
	MaxPasswordLen = 72
)

// ValidatePassword 校验密码长度是否落在可安全处理的区间内。
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLen {
		return errors.New("密码长度不能少于 6 位")
	}
	if len(password) > MaxPasswordLen {
		return errors.New("密码长度不能超过 72 字节")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
