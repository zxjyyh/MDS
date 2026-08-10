package handler

import (
	"strconv"

	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListRules(c *gin.Context) {
	rules, err := h.ReminderSvc.ListRules()
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Success(c, rules)
}

func (h *Handler) AddRule(c *gin.Context) {
	var body struct {
		Module     string `json:"module"`
		DaysBefore int    `json:"daysBefore"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.ReminderSvc.AddRule(body.Module, body.DaysBefore, body.Enabled); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "添加成功")
}

func (h *Handler) UpdateRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Module     *string `json:"module"`
		DaysBefore *int    `json:"daysBefore"`
		Enabled    *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	m := ""
	if body.Module != nil {
		m = *body.Module
	}
	d := 0
	if body.DaysBefore != nil {
		d = *body.DaysBefore
	}
	e := true
	if body.Enabled != nil {
		e = *body.Enabled
	}
	if err := h.ReminderSvc.UpdateRule(id, m, d, e); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.ReminderSvc.DeleteRule(id); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}

func (h *Handler) GetLogs(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	list, total, err := h.ReminderSvc.GetLogs(c.Query("module"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

func (h *Handler) TestRun(c *gin.Context) {
	go h.ReminderSvc.RunExpiryCheck()
	utils.SuccessMsg(c, nil, "已触发检查")
}
