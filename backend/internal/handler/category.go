package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zxjyyh/mds/internal/utils"
	"strconv"
)

func (h *Handler) GetCategoryTree(c *gin.Context) {
	systemType := c.Query("systemType")
	tree, err := h.CategorySvc.GetTree(systemType)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Success(c, tree)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var body struct {
		Name       string `json:"name"`
		ParentID   *int   `json:"parentId"`
		SystemType string `json:"systemType"`
		SortOrder  int    `json:"sortOrder"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" || body.SystemType == "" {
		utils.Fail(c, -1, "参数不完整")
		return
	}
	cat, err := h.CategorySvc.Create(body.Name, body.ParentID, body.SystemType, body.SortOrder)
	if err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, cat, "创建成功")
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Name      string `json:"name"`
		ParentID  *int   `json:"parentId"`
		SortOrder int    `json:"sortOrder"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.CategorySvc.Update(id, body.Name, body.ParentID, body.SortOrder); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.CategorySvc.Delete(id); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}
