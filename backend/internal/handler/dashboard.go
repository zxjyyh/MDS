package handler

import (
	"strconv"

	"github.com/zxjyyh/mds/internal/scope"
	"github.com/zxjyyh/mds/internal/service"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type statCard struct {
	Expired    int `json:"expired"`
	Expiring30 int `json:"expiring30"`
	Expiring60 int `json:"expiring60"`
	Expiring90 int `json:"expiring90"`
	Total      int `json:"total"`
}

// 以下统计助手都接受账号的数据可见范围 sc —— 仪表盘的计数与卡片必须按公司隔离，
// 否则受限账号能从数字上看出别家公司有多少过期证件（信息泄露）。
// 说明：documents（资料库）不做隔离，service 内部按表判断（见 scopeColumnFor）。

func queryStats(db *sqlx.DB, sc *scope.Scope, table, col string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpired(db, table, col, sc)
	s.Expiring30, _ = service.CountExpiring(db, table, col, 30, sc)
	s.Expiring60, _ = service.CountExpiringRange(db, table, col, 31, 60, sc)
	s.Expiring90, _ = service.CountExpiringRange(db, table, col, 61, 90, sc)
	s.Total, _ = service.CountTotal(db, table, sc)
	return s
}

func queryStatsFiltered(db *sqlx.DB, sc *scope.Scope, table, col, filter string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpiredFiltered(db, table, col, filter, sc)
	s.Expiring30, _ = service.CountExpiringFiltered(db, table, col, filter, 30, sc)
	s.Expiring60, _ = service.CountExpiringRangeFiltered(db, table, col, filter, 31, 60, sc)
	s.Expiring90, _ = service.CountExpiringRangeFiltered(db, table, col, filter, 61, 90, sc)
	s.Total, _ = service.CountTotalFiltered(db, table, filter, sc)
	return s
}

func queryStatsPersonnel(db *sqlx.DB, sc *scope.Scope, col string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpired(db, "personnel", col, sc)
	s.Expiring30, _ = service.CountExpiring(db, "personnel", col, 30, sc)
	s.Expiring60, _ = service.CountExpiringRange(db, "personnel", col, 31, 60, sc)
	s.Expiring90, _ = service.CountExpiringRange(db, "personnel", col, 61, 90, sc)
	s.Total, _ = service.CountTotalFiltered(db, "personnel", col+" IS NOT NULL AND "+col+" != ''", sc)
	return s
}

func queryStatsPersonnelCombined(db *sqlx.DB, sc *scope.Scope, fields []string) statCard {
	s := statCard{}
	for _, f := range fields {
		exp, _ := service.CountExpired(db, "personnel", f, sc)
		s30, _ := service.CountExpiring(db, "personnel", f, 30, sc)
		s60, _ := service.CountExpiringRange(db, "personnel", f, 31, 60, sc)
		s90, _ := service.CountExpiringRange(db, "personnel", f, 61, 90, sc)
		tot, _ := service.CountTotalFiltered(db, "personnel", f+" IS NOT NULL AND "+f+" != ''", sc)
		s.Expired += exp
		s.Expiring30 += s30
		s.Expiring60 += s60
		s.Expiring90 += s90
		s.Total += tot
	}
	return s
}

func queryStatsEquipment(db *sqlx.DB, sc *scope.Scope, col string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpired(db, "equipment", col, sc)
	s.Expiring30, _ = service.CountExpiring(db, "equipment", col, 30, sc)
	s.Expiring60, _ = service.CountExpiringRange(db, "equipment", col, 31, 60, sc)
	s.Expiring90, _ = service.CountExpiringRange(db, "equipment", col, 61, 90, sc)
	s.Total, _ = service.CountTotalFiltered(db, "equipment", col+" IS NOT NULL AND "+col+" != ''", sc)
	return s
}

func (h *Handler) DashboardStats(c *gin.Context) {
	ignoredCount, _ := service.CountIgnored(h.DB)
	data := gin.H{
		"personnelIdcard":  queryStatsPersonnelCombined(h.DB, currentScope(c), []string{"id_card_expiry", "driver_license_expiry"}),
		"personnelLicense": queryStatsPersonnelCombined(h.DB, currentScope(c), []string{"security_cert_date", "env_cert_date", "inspection_date", "internal_auditor_date"}),
		"equipmentCert":    queryStatsEquipment(h.DB, currentScope(c), "inspection_expiry_date"),
		"qualificationDoc": queryStatsFiltered(h.DB, currentScope(c), "documents", "expiry_date", "system_type='qualification'"),
		"safetyDoc":        queryStatsFiltered(h.DB, currentScope(c), "documents", "expiry_date", "system_type='safety'"),
		"ignored":          ignoredCount,
	}
	utils.Success(c, data)
}

func (h *Handler) DashboardExpiringList(c *gin.Context) {
	module := c.Query("module")
	days := queryInt(c, "days", 0)
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)

	list, total, err := service.GetExpiringList(h.DB, currentScope(c), module, days, page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

// ---- Ignore / Unignore ----

func (h *Handler) IgnoreWarning(c *gin.Context) {
	var body struct {
		Module      string `json:"module"`
		TargetName  string `json:"targetName"`
		RelatedName string `json:"relatedName"`
		ExpiryDate  string `json:"expiryDate"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	w, err := service.AddIgnoredWarning(h.DB, body.Module, body.TargetName, body.RelatedName, body.ExpiryDate, utils.GetAuthUserID(c))
	if err != nil {
		utils.Fail(c, -1, "忽略失败")
		return
	}
	utils.SuccessMsg(c, w, "已忽略")
}

func (h *Handler) UnignoreWarning(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := service.RemoveIgnoredWarning(h.DB, id); err != nil {
		utils.Fail(c, -1, "取消失败")
		return
	}
	utils.SuccessMsg(c, nil, "已取消忽略")
}

func (h *Handler) ListIgnoredWarnings(c *gin.Context) {
	list, err := service.ListIgnoredWarnings(h.DB)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Success(c, list)
}
