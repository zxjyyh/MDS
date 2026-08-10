package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zxjyyh/mds/internal/utils"
	"strconv"
)

func (h *Handler) ListEquipment(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	list, total, err := h.EquipmentSvc.List(c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

func (h *Handler) GetEquipment(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	e, err := h.EquipmentSvc.GetByID(id)
	if err != nil {
		utils.Fail(c, -1, "设备不存在")
		return
	}
	utils.Success(c, e)
}

func (h *Handler) CreateEquipment(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	e, err := h.EquipmentSvc.Create(body)
	if err != nil {
		utils.Fail(c, -1, "创建失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, e, "创建成功")
}

func (h *Handler) UpdateEquipment(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.EquipmentSvc.Update(id, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteEquipment(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.EquipmentSvc.Delete(id); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}

func (h *Handler) AddEquipmentCert(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	cert, err := h.EquipmentSvc.AddCert(id, body)
	if err != nil {
		utils.Fail(c, -1, "添加失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, cert, "添加成功")
}

func (h *Handler) UpdateEquipmentCert(c *gin.Context) {
	certID, _ := strconv.Atoi(c.Param("certId"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.EquipmentSvc.UpdateCert(certID, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteEquipmentCert(c *gin.Context) {
	certID, _ := strconv.Atoi(c.Param("certId"))
	if err := h.EquipmentSvc.DeleteCert(certID); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}
