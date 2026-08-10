package middleware

import (
	"net/http"
	"strings"

	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Auth 常规鉴权：令牌只能来自 Authorization 头，且必须是主访问令牌。
func Auth(db *sqlx.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c)
		if raw == "" {
			abortUnauthorized(c, "未登录或 token 已过期")
			return
		}
		claims, err := utils.ParseTokenOfType(raw, jwtSecret, utils.TokenTypeAccess)
		if err != nil {
			abortUnauthorized(c, authErrMessage(err))
			return
		}
		if !acceptUser(c, db, claims) {
			return
		}
		c.Next()
	}
}

// AuthWithQuery 用于文件预览 / 下载：
//   - query 中的 token 必须是文件访问令牌（typ=file）
//   - Header 中的 token 必须是主访问令牌
//
// 这样即便文件链接被日志、Referer 或浏览器历史记录泄露，
// 拿到的也只是一个 30 分钟内失效、且无法调用任何业务接口的令牌。
func AuthWithQuery(db *sqlx.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			claims *utils.Claims
			err    error
		)

		if q := c.Query("token"); q != "" {
			claims, err = utils.ParseTokenOfType(q, jwtSecret, utils.TokenTypeFile)
			if err == utils.ErrWrongTokenType {
				abortUnauthorized(c, "该链接需要文件访问令牌")
				return
			}
		} else if h := extractToken(c); h != "" {
			claims, err = utils.ParseTokenOfType(h, jwtSecret, utils.TokenTypeAccess)
		} else {
			abortUnauthorized(c, "未登录")
			return
		}

		if err != nil {
			abortUnauthorized(c, authErrMessage(err))
			return
		}
		if !acceptUser(c, db, claims) {
			return
		}
		c.Next()
	}
}

// acceptUser 校验用户仍然存在、处于启用状态，且令牌未被 token_epoch 吊销。
func acceptUser(c *gin.Context, db *sqlx.DB, claims *utils.Claims) bool {
	var row struct {
		Status     string `db:"status"`
		TokenEpoch int    `db:"token_epoch"`
	}
	if err := db.Get(&row, "SELECT status, token_epoch FROM users WHERE id=?", claims.UserID); err != nil {
		abortUnauthorized(c, "用户不存在")
		return false
	}
	if row.Status != "active" {
		abortUnauthorized(c, "账号已被禁用")
		return false
	}
	if claims.Epoch != row.TokenEpoch {
		abortUnauthorized(c, utils.ErrTokenRevoked.Error())
		return false
	}

	c.Set("userId", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("userRole", claims.Role)
	return true
}

func authErrMessage(err error) string {
	switch err {
	case utils.ErrWrongTokenType:
		return "令牌类型不正确"
	case nil:
		return ""
	default:
		return "token 无效或已过期"
	}
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ApiResponse{
		Code: -1, Message: msg, Data: nil,
	})
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}
