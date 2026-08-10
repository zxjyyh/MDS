package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port        int
	Env         string
	DBPath      string
	JWTSecret   string
	JWTRefresh  string
	UploadDir   string
	UploadMax   int64
	FrontendDir string
	SMTP        SMTPConfig
	AppURL      string
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

// secretsFileName 存放自动生成的 JWT 密钥。
// 它被刻意放在数据库同级目录（也就是 docker-compose 里挂出来的 ./data 卷）而不是工作目录：
// 容器的工作目录属于镜像层，`docker compose up -d --force-recreate` 之后会被重置，
// 旧版本把密钥写进工作目录的 .env，导致每次重建容器所有人被强制登出。
const secretsFileName = ".secrets.env"

func Load() *Config {
	viper.SetDefault("PORT", 3000)
	viper.SetDefault("NODE_ENV", "production")
	viper.SetDefault("DATABASE_URL", "./data/data.db")
	viper.SetDefault("UPLOAD_DIR", "./uploads")
	viper.SetDefault("UPLOAD_MAX_SIZE", 52428800)
	viper.SetDefault("FRONTEND_DIR", "./web/dist")
	viper.SetDefault("APP_URL", "http://localhost:3000")

	// .env 只读：存在就读，不存在就算了，任何情况下都不会由程序写回。
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	dbPath := viper.GetString("DATABASE_URL")

	jwtSecret := strings.TrimSpace(viper.GetString("JWT_SECRET"))
	jwtRefresh := strings.TrimSpace(viper.GetString("JWT_REFRESH_SECRET"))

	if jwtSecret == "" || jwtRefresh == "" {
		s, r := resolveSecrets(dbPath, jwtSecret, jwtRefresh)
		jwtSecret, jwtRefresh = s, r
	}

	return &Config{
		Port:        viper.GetInt("PORT"),
		Env:         viper.GetString("NODE_ENV"),
		DBPath:      dbPath,
		JWTSecret:   jwtSecret,
		JWTRefresh:  jwtRefresh,
		UploadDir:   viper.GetString("UPLOAD_DIR"),
		UploadMax:   viper.GetInt64("UPLOAD_MAX_SIZE"),
		FrontendDir: viper.GetString("FRONTEND_DIR"),
		AppURL:      viper.GetString("APP_URL"),
		SMTP: SMTPConfig{
			Host: viper.GetString("SMTP_HOST"),
			Port: viper.GetInt("SMTP_PORT"),
			User: viper.GetString("SMTP_USER"),
			Pass: viper.GetString("SMTP_PASS"),
			From: viper.GetString("SMTP_FROM"),
		},
	}
}

// resolveSecrets 从数据目录下的 .secrets.env 读取密钥；缺失则生成并落盘（0600）。
// 环境变量里已显式提供的值优先，不会被文件覆盖。
func resolveSecrets(dbPath, envSecret, envRefresh string) (string, string) {
	dir := filepath.Dir(dbPath)
	path := filepath.Join(dir, secretsFileName)

	stored := readEnvFile(path)

	secret := envSecret
	if secret == "" {
		secret = stored["JWT_SECRET"]
	}
	refresh := envRefresh
	if refresh == "" {
		refresh = stored["JWT_REFRESH_SECRET"]
	}

	changed := false
	if secret == "" {
		secret = randomHex(32)
		changed = true
	}
	if refresh == "" {
		refresh = randomHex(32)
		changed = true
	}

	if changed {
		if err := writeSecrets(dir, path, secret, refresh); err != nil {
			fmt.Printf("[warn] 无法持久化 JWT 密钥到 %s: %v\n", path, err)
			fmt.Println("[warn] 本次生成的密钥仅在进程存活期间有效，重启后所有用户需要重新登录。")
			fmt.Println("[warn] 建议在 .env / compose environment 中显式设置 JWT_SECRET 与 JWT_REFRESH_SECRET。")
		}
	}

	return secret, refresh
}

func writeSecrets(dir, path, secret, refresh string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	content := fmt.Sprintf(
		"# 本文件由程序自动生成，请勿提交到版本库。\n"+
			"# 删除后所有已签发的令牌立即失效（等同于强制全员重新登录）。\n"+
			"JWT_SECRET=%s\nJWT_REFRESH_SECRET=%s\n", secret, refresh)
	return os.WriteFile(path, []byte(content), 0o600)
}

// readEnvFile 解析极简的 KEY=VALUE 文件，忽略空行与 # 注释。
func readEnvFile(path string) map[string]string {
	out := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return out
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败属于系统级异常，继续跑等于用可预测密钥签发令牌
		panic("无法生成安全随机数: " + err.Error())
	}
	return hex.EncodeToString(b)
}
