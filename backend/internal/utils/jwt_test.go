package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "unit-test-secret-do-not-use-in-prod"

func TestAccessTokenRoundTrip(t *testing.T) {
	tok, err := GenerateAccessToken(7, "alice", "admin", 3, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseTokenOfType(tok, testSecret, TokenTypeAccess)
	if err != nil {
		t.Fatalf("解析主令牌失败: %v", err)
	}
	if claims.UserID != 7 || claims.Username != "alice" || claims.Role != "admin" {
		t.Fatalf("载荷不匹配: %+v", claims)
	}
	if claims.Epoch != 3 {
		t.Fatalf("epoch 未透传，期望 3，实际 %d", claims.Epoch)
	}
}

// TestTokenTypeIsolation 是文件令牌方案的核心不变量：
// 主令牌不得当作文件令牌走 query，文件令牌也不得当作主令牌走 Header。
// 一旦某一方向被打通，短时效文件令牌就等价于完整会话凭证。
func TestTokenTypeIsolation(t *testing.T) {
	access, err := GenerateAccessToken(1, "bob", "user", 0, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	file, err := GenerateFileToken(1, "bob", "user", 0, testSecret)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ParseTokenOfType(access, testSecret, TokenTypeFile); !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("主令牌被当作文件令牌接受了，err=%v", err)
	}
	if _, err := ParseTokenOfType(file, testSecret, TokenTypeAccess); !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("文件令牌被当作主令牌接受了，err=%v", err)
	}

	if _, err := ParseTokenOfType(file, testSecret, TokenTypeFile); err != nil {
		t.Fatalf("文件令牌自身校验失败: %v", err)
	}
}

func TestFileTokenIsShortLived(t *testing.T) {
	if FileTokenTTL > 1*time.Hour {
		t.Fatalf("文件令牌有效期过长: %v", FileTokenTTL)
	}
	tok, err := GenerateFileToken(1, "bob", "user", 0, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseTokenOfType(tok, testSecret, TokenTypeFile)
	if err != nil {
		t.Fatal(err)
	}
	left := time.Until(claims.ExpiresAt.Time)
	if left > FileTokenTTL+time.Minute {
		t.Fatalf("文件令牌过期时间异常，剩余 %v", left)
	}
}

func TestWrongSecretRejected(t *testing.T) {
	tok, err := GenerateAccessToken(1, "bob", "user", 0, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(tok, "another-secret"); err == nil {
		t.Fatal("换密钥后仍然验签通过")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	tok, err := newToken(1, "bob", "user", TokenTypeAccess, 0, -time.Minute, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(tok, testSecret); err == nil {
		t.Fatal("过期令牌仍被接受")
	}
}

// TestAlgNoneRejected 防经典的 alg=none 绕过。
func TestAlgNoneRejected(t *testing.T) {
	claims := Claims{
		UserID:   1,
		Username: "attacker",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(tok, testSecret); err == nil {
		t.Fatal("alg=none 令牌被接受了")
	}
}

// TestRefreshTokenSecretSeparation 确保 access / refresh 密钥不通用，
// 否则一个泄露就等于两个都泄露。
func TestRefreshTokenSecretSeparation(t *testing.T) {
	const refreshSecret = "unit-test-refresh-secret"
	rt, err := GenerateRefreshToken(1, "bob", "user", 0, refreshSecret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(rt, testSecret); err == nil {
		t.Fatal("refresh 令牌用 access 密钥验签成功了")
	}
	if _, err := ParseRefreshToken(rt, refreshSecret); err != nil {
		t.Fatalf("refresh 令牌自身校验失败: %v", err)
	}
	if RefreshTokenTTL <= AccessTokenTTL {
		t.Fatal("refresh 有效期应长于 access")
	}
}
