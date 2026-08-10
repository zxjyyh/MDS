package service

import (
	"github.com/jmoiron/sqlx"
	gomail "gopkg.in/gomail.v2"
)

type EmailService struct{ DB *sqlx.DB }

func (s *EmailService) SendReminder(to []string, subject, body string) error {
	cfg := GetConfig()
	if cfg.SMTP.Host == "" {
		return nil // SMTP not configured, skip silently
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.SMTP.From)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User, cfg.SMTP.Pass)
	return d.DialAndSend(m)
}
