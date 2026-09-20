package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zxjyyh/mds/internal/utils"
)

func (h *Handler) ListCompanies(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	list, total, err := h.CompanySvc.List(c.Query("keyword"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

func (h *Handler) GetCompany(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	company, err := h.CompanySvc.GetByID(id)
	if err != nil {
		utils.Fail(c, -1, "公司信息不存在")
		return
	}
	utils.Success(c, company)
}

func (h *Handler) CreateCompany(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	company, err := h.CompanySvc.Create(body)
	if err != nil {
		utils.Fail(c, -1, "创建失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, company, "创建成功")
}

func (h *Handler) UpdateCompany(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.CompanySvc.Update(id, body); err != nil {
		utils.Fail(c, -1, "更新失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteCompany(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.CompanySvc.Delete(id); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}

// CompanyOptions 公司下拉选项（id / name / address），
// 供劳动合同选择甲方后自动带出公司地址，使两处共用同一份数据。
func (h *Handler) CompanyOptions(c *gin.Context) {
	list, err := h.CompanySvc.Options()
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Success(c, list)
}
