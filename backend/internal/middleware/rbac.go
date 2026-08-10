package middleware

import (
	"net/http"

	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

func RequireAdmin(c *gin.Context) {
	role := c.GetString("userRole")
	if role != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, utils.ApiResponse{
			Code: -1, Message: "需要管理员权限", Data: nil,
		})
		return
	}
	c.Next()
}
