package handler

import (
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListUsers(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	keyword := c.Query("keyword")
	role := c.Query("role")

	list, total, err := h.UserSvc.List(keyword, role, page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

func (h *Handler) GetUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	user, err := h.UserSvc.GetByID(id)
	if err != nil {
		utils.Fail(c, -1, "用户不存在")
		return
	}
	utils.Success(c, user)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		RealName string `json:"realName"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Username == "" || body.Password == "" {
		utils.Fail(c, -1, "参数不完整")
		return
	}
	if err := utils.ValidatePassword(body.Password); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}

	u := models.User{
		Username:     body.Username,
		PasswordHash: body.Password,
		RealName:     body.RealName,
		Email:        body.Email,
		Role:         body.Role,
		Status:       "active",
	}
	if u.Role == "" {
		u.Role = "user"
	}
	if err := h.UserSvc.Create(u); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "创建成功")
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		RealName string `json:"realName"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.UserSvc.Update(id, body.RealName, body.Email, body.Role); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// 需要操作人 ID：被删用户上传过的文件要移交给他，否则会撞上外键约束
	if err := h.UserSvc.Delete(id, utils.GetAuthUserID(c)); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}

func (h *Handler) ResetPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := utils.ValidatePassword(body.NewPassword); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	if err := h.UserSvc.ResetPassword(id, body.NewPassword); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "密码重置成功")
}

func (h *Handler) SetStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || (body.Status != "active" && body.Status != "disabled") {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.UserSvc.SetStatus(id, body.Status); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "状态更新成功")
}
