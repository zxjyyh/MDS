package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/zxjyyh/mds/internal/utils"
)

func (h *Handler) ListOperationLogs(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)
	list, total, err := h.OperationLogSvc.List(c.Query("keyword"), c.Query("role"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

// ClearOperationLogs 清空全部操作日志（仅管理员，日志量过大时手动清理用）。
func (h *Handler) ClearOperationLogs(c *gin.Context) {
	n, err := h.OperationLogSvc.Clear()
	if err != nil {
		utils.Fail(c, -1, "清空失败")
		return
	}
	utils.SuccessMsg(c, gin.H{"deleted": n}, "已清空")
}
