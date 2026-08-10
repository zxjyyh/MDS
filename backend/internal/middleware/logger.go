package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 访问日志中间件。
// 注意：只记录 URL.Path，绝不记录 RawQuery —— 文件短时效令牌通过 query 传递，
// 一旦写进日志文件就等同于把令牌落盘。
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// CORS 跨域中间件。
//
// 通过环境变量 CORS_ORIGINS 配置，语义如下：
//
//	留空（默认）  完全不下发任何 CORS 响应头。前后端同源部署（单容器）时这是正确且最安全的选择。
//	白名单        CORS_ORIGINS=http://192.168.1.10:13000,https://mds.example.com
//	              精确匹配 Origin，命中才下发，并允许携带凭证。
//	通配 "*"      CORS_ORIGINS=* 反射任意来源，但强制不下发 Allow-Credentials，
//	              浏览器因此不会带上 Cookie/Authorization，避免 CSRF 式凭证盗用。
//
// 旧版本在留空时反射任意 Origin 且带 Allow-Credentials，等于对全网开放带凭证的跨域请求，
// 已在此版本移除。
func CORS() gin.HandlerFunc {
	raw := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	wildcard := raw == "*"
	allowed := parseOrigins(raw)
	enabled := wildcard || len(allowed) > 0

	return func(c *gin.Context) {
		if !enabled {
			// 同源部署：不参与任何跨域协商。
			// 预检请求直接拒绝，避免给出「看起来能跨域」的错觉。
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		c.Header("Vary", "Origin")

		if origin != "" {
			switch {
			case wildcard:
				// 反射来源但不允许携带凭证
				c.Header("Access-Control-Allow-Origin", origin)
			case allowed[origin]:
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")
		c.Header("Access-Control-Max-Age", "600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func parseOrigins(raw string) map[string]bool {
	m := make(map[string]bool)
	if raw == "*" {
		return m
	}
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" || s == "*" {
			continue
		}
		// 统一去掉结尾斜杠，避免 http://a.com/ 与 http://a.com 不匹配
		m[strings.TrimRight(s, "/")] = true
	}
	return m
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Next()
	}
}
