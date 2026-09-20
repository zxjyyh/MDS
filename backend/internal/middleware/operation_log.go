package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zxjyyh/mds/internal/service"
	"github.com/zxjyyh/mds/internal/utils"
)

// OperationLog 自动记录有审计价值的请求。
//
// 记录范围（刻意排除噪音）：
//   - 所有写操作（POST / PUT / PATCH / DELETE）
//   - 文件的 上传 / 预览 / 下载 / 导出（虽是读操作，但涉及数据外带）
//
// 不记录：列表与详情等纯浏览 GET、以及 /auth/profile、/auth/refresh、/auth/file-token
// —— 这些每次页面加载都会打，记进去只会把日志淹掉。
//
// 注册位置：必须放在全局 Use 里（早于路由上的 Auth 中间件），
// 因为 c.Next() 会让 Auth 先执行，返回后 userId / username / userRole 才在 context 里。
func OperationLog(svc *service.OperationLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		method, path := c.Request.Method, c.Request.URL.Path

		c.Next()

		if !shouldLogOperation(method, path) {
			return
		}
		// 只记成功的请求：401/403/404 不构成「某人的一次操作」
		if c.Writer.Status() >= 300 {
			return
		}
		uid := utils.GetAuthUserID(c)
		if uid == 0 {
			return // 未通过认证（如登录失败）不记
		}
		svc.Record(service.OperationEntry{
			UserID:   uid,
			Username: c.GetString("username"),
			Role:     c.GetString("userRole"),
			Method:   method,
			Path:     path,
		})
	}
}

func shouldLogOperation(method, path string) bool {
	if !strings.HasPrefix(path, "/api/") {
		return false
	}
	switch path {
	case "/api/auth/profile", "/api/auth/refresh", "/api/auth/file-token":
		return false
	}
	if method != http.MethodGet {
		return true
	}
	// GET 里只记「数据外带」类操作。
	// 特别注意不要把 /preview 记进来：列表里的头像、附件缩略图都是 <img src=preview>，
	// 光打开一次人员管理就会产生几十条「预览文件」，直接把日志淹掉（实测过一次，已剔除）。
	// 真正的预览是用户主动点击触发的，审计价值低于下载/导出，牺牲它可以换来干净的日志。
	return strings.Contains(path, "/download") ||
		strings.Contains(path, "/export")
}
