package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count  int
	window time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateLimitEntry
	duration time.Duration
	maxReq   int
}

// allow 判断该 key 是否放行。注意锁只在 map 操作期间持有，
// 绝不跨越 c.Next()，否则整个登录流程（bcrypt + 数据库）会被全局串行化。
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.entries[key]
	if !ok || time.Since(entry.window) > rl.duration {
		rl.entries[key] = &rateLimitEntry{count: 1, window: time.Now()}
		return true
	}

	entry.count++
	return entry.count <= rl.maxReq
}

// cleanup 清理过期条目。没有它，map 会随着扫描器/爬虫的源 IP 无限增长，
// 长期运行的 NAS 容器最终会被 OOM kill。
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, v := range rl.entries {
		if time.Since(v.window) > rl.duration {
			delete(rl.entries, k)
		}
	}
}

// startJanitor 启动后台清理协程（进程级，只启动一次）。
func (rl *rateLimiter) startJanitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
}

var loginLimiter = &rateLimiter{
	entries:  make(map[string]*rateLimitEntry),
	duration: 15 * time.Minute,
	maxReq:   10,
}

var janitorOnce sync.Once

func LoginRateLimit() gin.HandlerFunc {
	janitorOnce.Do(func() {
		loginLimiter.startJanitor(5 * time.Minute)
	})

	return func(c *gin.Context) {
		if !loginLimiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, utils.ApiResponse{
				Code: -1, Message: "登录尝试过于频繁，请15分钟后重试", Data: nil,
			})
			return
		}
		c.Next()
	}
}
