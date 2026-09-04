package handler

import (
	"strconv"

	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListPersonnel(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	list, total, err := h.PersonnelSvc.List(c.Query("keyword"), c.Query("department"), c.Query("status"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

// PersonnelOptions 返回用于筛选下拉框的检测站去重列表。
func (h *Handler) PersonnelOptions(c *gin.Context) {
	departments, err := h.PersonnelSvc.Options()
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Success(c, gin.H{"departments": departments})
}

func (h *Handler) GetPersonnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.PersonnelSvc.GetByID(id)
	if err != nil {
		utils.Fail(c, -1, "人员不存在")
		return
	}
	utils.Success(c, p)
}

func (h *Handler) CreatePersonnel(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	p, err := h.PersonnelSvc.Create(body, utils.GetAuthUserID(c))
	if err != nil {
		utils.Fail(c, -1, "创建失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, p, "创建成功")
}

func (h *Handler) UpdatePersonnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.PersonnelSvc.Update(id, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeletePersonnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.PersonnelSvc.Delete(id); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}

func (h *Handler) AddPersonnelCert(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	cert, err := h.PersonnelSvc.AddCert(id, body)
	if err != nil {
		utils.Fail(c, -1, "添加失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, cert, "添加成功")
}

func (h *Handler) UpdatePersonnelCert(c *gin.Context) {
	certID, _ := strconv.Atoi(c.Param("certId"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.PersonnelSvc.UpdateCert(certID, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeletePersonnelCert(c *gin.Context) {
	certID, _ := strconv.Atoi(c.Param("certId"))
	if err := h.PersonnelSvc.DeleteCert(certID); err != nil {
		utils.Fail(c, -1, "删除失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}
