package handler

import (
	"github.com/zxjyyh/mds/internal/service"
	"github.com/zxjyyh/mds/internal/utils"
	"strconv"

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

func queryStats(db *sqlx.DB, table, col string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpired(db, table, col)
	s.Expiring30, _ = service.CountExpiring(db, table, col, 30)
	s.Expiring60, _ = service.CountExpiringRange(db, table, col, 31, 60)
	s.Expiring90, _ = service.CountExpiringRange(db, table, col, 61, 90)
	s.Total, _ = service.CountTotal(db, table)
	return s
}

func queryStatsFiltered(db *sqlx.DB, table, col, filter string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpiredFiltered(db, table, col, filter)
	s.Expiring30, _ = service.CountExpiringFiltered(db, table, col, filter, 30)
	s.Expiring60, _ = service.CountExpiringRangeFiltered(db, table, col, filter, 31, 60)
	s.Expiring90, _ = service.CountExpiringRangeFiltered(db, table, col, filter, 61, 90)
	s.Total, _ = service.CountTotalFiltered(db, table, filter)
	return s
}

func queryStatsPersonnel(db *sqlx.DB, col string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpired(db, "personnel", col)
	s.Expiring30, _ = service.CountExpiring(db, "personnel", col, 30)
	s.Expiring60, _ = service.CountExpiringRange(db, "personnel", col, 31, 60)
	s.Expiring90, _ = service.CountExpiringRange(db, "personnel", col, 61, 90)
	s.Total, _ = service.CountTotalFiltered(db, "personnel", col+" IS NOT NULL AND "+col+" != ''")
	return s
}

func queryStatsPersonnelCombined(db *sqlx.DB, fields []string) statCard {
	s := statCard{}
	for _, f := range fields {
		exp, _ := service.CountExpired(db, "personnel", f)
		s30, _ := service.CountExpiring(db, "personnel", f, 30)
		s60, _ := service.CountExpiringRange(db, "personnel", f, 31, 60)
		s90, _ := service.CountExpiringRange(db, "personnel", f, 61, 90)
		tot, _ := service.CountTotalFiltered(db, "personnel", f+" IS NOT NULL AND "+f+" != ''")
		s.Expired += exp
		s.Expiring30 += s30
		s.Expiring60 += s60
		s.Expiring90 += s90
		s.Total += tot
	}
	return s
}

func queryStatsEquipment(db *sqlx.DB, col string) statCard {
	s := statCard{}
	s.Expired, _ = service.CountExpired(db, "equipment", col)
	s.Expiring30, _ = service.CountExpiring(db, "equipment", col, 30)
	s.Expiring60, _ = service.CountExpiringRange(db, "equipment", col, 31, 60)
	s.Expiring90, _ = service.CountExpiringRange(db, "equipment", col, 61, 90)
	s.Total, _ = service.CountTotalFiltered(db, "equipment", col+" IS NOT NULL AND "+col+" != ''")
	return s
}

func (h *Handler) DashboardStats(c *gin.Context) {
	ignoredCount, _ := service.CountIgnored(h.DB)
	data := gin.H{
		"personnelIdcard":  queryStatsPersonnelCombined(h.DB, []string{"id_card_expiry", "driver_license_expiry"}),
		"personnelLicense": queryStatsPersonnelCombined(h.DB, []string{"security_cert_date", "env_cert_date", "inspection_date", "internal_auditor_date"}),
		"equipmentCert":    queryStatsEquipment(h.DB, "inspection_expiry_date"),
		"qualificationDoc": queryStatsFiltered(h.DB, "documents", "expiry_date", "system_type='qualification'"),
		"safetyDoc":        queryStatsFiltered(h.DB, "documents", "expiry_date", "system_type='safety'"),
		"ignored":          ignoredCount,
	}
	utils.Success(c, data)
}

func (h *Handler) DashboardExpiringList(c *gin.Context) {
	module := c.Query("module")
	days := queryInt(c, "days", 0)
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)

	list, total, err := service.GetExpiringList(h.DB, module, days, page, pageSize)
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
