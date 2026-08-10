package handler

import (
	"strings"

	"github.com/zxjyyh/mds/internal/service"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type smtpSettings struct {
	Host string  `json:"host"`
	Port int     `json:"port"`
	User string  `json:"user"`
	Pass *string `json:"pass,omitempty"`
	From string  `json:"from"`
}

// maskPassword 只回显首尾各 4 位，用于让管理员确认"确实存了个东西"，
// 但不足以还原原始口令。
func maskPassword(pass string) string {
	if pass == "" {
		return ""
	}
	if len(pass) <= 8 {
		return strings.Repeat("*", len(pass))
	}
	return pass[:4] + strings.Repeat("*", len(pass)-8) + pass[len(pass)-4:]
}

// GetSMTPConfig 返回当前 SMTP 配置（密码打码）
func (h *Handler) GetSMTPConfig(c *gin.Context) {
	cfg := h.Config
	mp := maskPassword(cfg.SMTP.Pass)
	utils.Success(c, smtpSettings{
		Host: cfg.SMTP.Host,
		Port: cfg.SMTP.Port,
		User: cfg.SMTP.User,
		Pass: &mp,
		From: cfg.SMTP.From,
	})
}

// UpdateSMTPConfig 保存 SMTP 配置到 app_settings 表。
//
// 密码字段语义：
//   - 未提交（字段缺失）或提交空串  -> 保留原密码
//   - 提交的正好是 GET 返回的掩码   -> 保留原密码（防止前端把掩码原样回传写坏配置）
//   - 其它                          -> 覆盖
func (h *Handler) UpdateSMTPConfig(c *gin.Context) {
	var body smtpSettings
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}

	current := h.Config.SMTP.Pass
	pass := current
	if body.Pass != nil {
		v := strings.TrimSpace(*body.Pass)
		if v != "" && v != maskPassword(current) {
			pass = v
		}
	}

	if err := service.SaveSMTP(h.DB, h.Config,
		strings.TrimSpace(body.Host),
		body.Port,
		strings.TrimSpace(body.User),
		pass,
		strings.TrimSpace(body.From),
	); err != nil {
		h.Logger.Error("保存 SMTP 配置失败", zap.Error(err))
		utils.Fail(c, -1, "保存配置失败")
		return
	}

	utils.SuccessMsg(c, nil, "SMTP 配置已保存")
}
