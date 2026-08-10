package utils

import (
	"strings"
	"testing"
)

// TestPasswordLengthBoundary 覆盖 bcrypt 的 72 字节硬上限。
//
// bcrypt 会**静默截断**超过 72 字节的输入，若不显式拒绝，
// 「前 72 字节相同 + 任意后缀」的密码都能登录成功。
func TestPasswordLengthBoundary(t *testing.T) {
	cases := []struct {
		name    string
		pwd     string
		wantErr bool
	}{
		{"太短", "12345", true},
		{"最短合法", "123456", false},
		{"常规", "Sup3r$ecret!", false},
		{"72 字节边界", strings.Repeat("a", 72), false},
		{"73 字节超限", strings.Repeat("a", 73), true},
		{"中文 24 字（72 字节）", strings.Repeat("密", 24), false},
		{"中文 25 字（75 字节）", strings.Repeat("密", 25), true},
		{"空", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidatePassword(c.pwd)
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidatePassword(%d 字节) err=%v, wantErr=%v", len(c.pwd), err, c.wantErr)
			}
			if _, err := HashPassword(c.pwd); (err != nil) != c.wantErr {
				t.Fatalf("HashPassword 与 ValidatePassword 行为不一致: err=%v", err)
			}
		})
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	const pwd = "Sup3r$ecret!"
	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatal(err)
	}
	if hash == pwd || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("哈希结果异常: %q", hash)
	}
	if !CheckPassword(pwd, hash) {
		t.Fatal("正确密码校验失败")
	}
	if CheckPassword(pwd+"x", hash) {
		t.Fatal("错误密码竟然通过")
	}
	if CheckPassword("", hash) {
		t.Fatal("空密码竟然通过")
	}

	// 同一密码两次加盐后哈希必须不同
	hash2, err := HashPassword(pwd)
	if err != nil {
		t.Fatal(err)
	}
	if hash == hash2 {
		t.Fatal("两次哈希结果相同，疑似未加盐")
	}
}

// TestTruncationAttack 直接复现被截断时的攻击场景。
func TestTruncationAttack(t *testing.T) {
	base := strings.Repeat("a", 72)
	hash, err := HashPassword(base)
	if err != nil {
		t.Fatal(err)
	}
	// 攻击者拿 72 字节前缀 + 任意后缀尝试登录
	if _, err := HashPassword(base + "EVIL"); err == nil {
		t.Fatal("73+ 字节密码本应被拒绝")
	}
	if CheckPassword(base+"EVIL", hash) {
		// bcrypt 底层确实会截断，因此必须靠 ValidatePassword 在入口挡住
		t.Log("警告：bcrypt 截断行为已确认，入口校验是唯一防线")
	}
}
