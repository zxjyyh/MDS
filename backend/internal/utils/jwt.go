package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 令牌类型。
//
// 设计目的：`<img src>`、`window.open` 这类浏览器发起的请求无法携带自定义
// Authorization 头，只能把令牌放进 query string。但 query string 会出现在
// 服务端访问日志、浏览器历史、以及跨站跳转的 Referer 里。
// 因此文件访问单独签发一种短时效、只能走 query 的令牌，并与主访问令牌
// **双向隔离**：主令牌不允许出现在 query，文件令牌不允许出现在 Header。
const (
	TokenTypeAccess = ""     // 主访问令牌，仅限 Authorization 头
	TokenTypeFile   = "file" // 文件访问令牌，仅限 query string
)

const (
	AccessTokenTTL  = 2 * time.Hour
	RefreshTokenTTL = 7 * 24 * time.Hour
	FileTokenTTL    = 30 * time.Minute
)

var (
	ErrWrongTokenType = errors.New("令牌类型不匹配")
	ErrTokenRevoked   = errors.New("令牌已失效，请重新登录")
)

type Claims struct {
	UserID   int    `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	// Typ 区分主访问令牌与文件访问令牌，见上方常量。
	Typ string `json:"typ,omitempty"`
	// Epoch 对应 users.token_epoch。用户改密 / 被禁用 / 主动登出时该值会自增，
	// 使此前签发的全部令牌立即失效（无需维护黑名单）。
	Epoch int `json:"epoch"`
	jwt.RegisteredClaims
}

func newToken(userID int, username, role, typ string, epoch int, ttl time.Duration, secret string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Typ:      typ,
		Epoch:    epoch,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func GenerateAccessToken(userID int, username, role string, epoch int, secret string) (string, error) {
	return newToken(userID, username, role, TokenTypeAccess, epoch, AccessTokenTTL, secret)
}

func GenerateRefreshToken(userID int, username, role string, epoch int, refreshSecret string) (string, error) {
	return newToken(userID, username, role, TokenTypeAccess, epoch, RefreshTokenTTL, refreshSecret)
}

// GenerateFileToken 签发只能通过 query string 使用的短时效文件访问令牌。
func GenerateFileToken(userID int, username, role string, epoch int, secret string) (string, error) {
	return newToken(userID, username, role, TokenTypeFile, epoch, FileTokenTTL, secret)
}

func ParseToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// 只接受 HMAC 家族，拒绝 alg=none 以及 RS256 公钥混淆攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ParseTokenOfType 在验签之外额外断言令牌类型，用于 Header / query 双向隔离。
func ParseTokenOfType(tokenString, secret, want string) (*Claims, error) {
	claims, err := ParseToken(tokenString, secret)
	if err != nil {
		return nil, err
	}
	if claims.Typ != want {
		return nil, ErrWrongTokenType
	}
	return claims, nil
}

func ParseRefreshToken(tokenString, refreshSecret string) (*Claims, error) {
	return ParseToken(tokenString, refreshSecret)
}
