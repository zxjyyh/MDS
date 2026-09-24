package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zxjyyh/mds/internal/utils"
	"strconv"
)

func (h *Handler) ListDocuments(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	systemType := c.Query("systemType")

	var categoryID *int
	if cid := c.Query("categoryId"); cid != "" {
		id, _ := strconv.Atoi(cid)
		categoryID = &id
	}

	list, total, err := h.DocumentSvc.List(currentScope(c), systemType, categoryID, c.Query("keyword"), c.Query("expiryStatus"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

func (h *Handler) GetDocument(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	doc, err := h.DocumentSvc.GetByID(currentScope(c), id)
	if err != nil {
		utils.Fail(c, -1, "文档不存在")
		return
	}
	utils.Success(c, doc)
}

func (h *Handler) CreateDocument(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.DocumentSvc.Create(currentScope(c), body, utils.GetAuthUserID(c)); err != nil {
		utils.Fail(c, -1, "创建失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "创建成功")
}

func (h *Handler) UpdateDocument(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.DocumentSvc.Update(currentScope(c), id, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteDocument(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DocumentSvc.Delete(currentScope(c), id); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}
