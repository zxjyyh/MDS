package database

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// Seed 初始化默认管理员账号。
// 密码来源优先级：
//  1. 环境变量 ADMIN_INIT_PASSWORD
//  2. 随机生成（仅在首次初始化时打印一次，务必记录）
func Seed(db *sqlx.DB) error {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM users")
	if err != nil {
		return err
	}
	if count > 0 {
		log.Println("Database already has users, skipping seed")
		return nil
	}

	password := strings.TrimSpace(os.Getenv("ADMIN_INIT_PASSWORD"))
	generated := false
	if password == "" {
		password = randomPassword(12)
		generated = true
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO users (username, password_hash, real_name, email, role, status) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"admin", string(hash), "管理员", "admin@example.com", "admin", "active")
	if err != nil {
		return err
	}

	if generated {
		log.Println("========================================================")
		log.Println(" 已创建默认管理员账号")
		log.Printf("   用户名: admin")
		log.Printf("   密码  : %s", password)
		log.Println(" 此密码仅显示这一次，请立即登录并修改！")
		log.Println(" 提示: 可通过环境变量 ADMIN_INIT_PASSWORD 指定初始密码")
		log.Println("========================================================")
	} else {
		log.Println("已创建默认管理员账号 admin（密码来自 ADMIN_INIT_PASSWORD），请尽快登录修改。")
	}
	return nil
}

// randomPassword 生成 URL 安全的随机密码
func randomPassword(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 极端情况下的回退值，仍要求用户立即修改
		return "ChangeMe!" + base64.RawURLEncoding.EncodeToString([]byte("fallback"))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
