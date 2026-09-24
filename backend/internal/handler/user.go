package handler

import (
	"fmt"
	"strconv"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"

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

	// 附上每个账号的可见公司（批量取，避免 N+1）
	ids := make([]int, 0, len(list))
	for _, u := range list {
		ids = append(ids, u.ID)
	}
	cm, err := h.UserCompanySvc.NamesOfUsers(ids)
	if err != nil {
		utils.Fail(c, -1, "查询公司权限失败")
		return
	}
	for i := range list {
		if c := cm[list[i].ID]; len(c) > 0 {
			list[i].Companies = c
		} else {
			list[i].Companies = []string{}
		}
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
	companies, err := h.UserCompanySvc.CompaniesOf(id)
	if err != nil {
		utils.Fail(c, -1, "查询公司权限失败")
		return
	}
	user.Companies = companies
	utils.Success(c, user)
}

// SetUserCompanies 覆盖式设置某账号的可见公司（仅管理员，路由层已限）。
//
// 传空数组表示「清空权限」= 该账号恢复为不受限（见 internal/scope 的渐进启用规则）。
func (h *Handler) SetUserCompanies(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if _, err := h.UserSvc.GetByID(id); err != nil {
		utils.Fail(c, -1, "用户不存在")
		return
	}
	var body struct {
		Companies []string `json:"companies"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.UserCompanySvc.SetCompanies(id, body.Companies); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "公司权限已更新")
}

func (h *Handler) CreateUser(c *gin.Context) {
	var body struct {
		Username  string   `json:"username"`
		Password  string   `json:"password"`
		RealName  string   `json:"realName"`
		Email     string   `json:"email"`
		Role      string   `json:"role"`
		Companies []string `json:"companies"` // 公司权限，可空（空 = 不受限）
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
	created, err := h.UserSvc.Create(u)
	if err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	// 公司权限与创建放在同一次请求，避免「用户建好了但权限没配上」的中间态
	if len(body.Companies) > 0 {
		if err := h.UserCompanySvc.SetCompanies(created.ID, body.Companies); err != nil {
			utils.Fail(c, -1, fmt.Sprintf("用户已创建，但公司权限设置失败：%s", err.Error()))
			return
		}
		created.Companies = body.Companies
	}
	utils.SuccessMsg(c, created, "创建成功")
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Username  string   `json:"username"`
		RealName  string   `json:"realName"`
		Email     string   `json:"email"`
		Role      string   `json:"role"`
		Companies []string `json:"companies"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.UserSvc.Update(id, body.Username, body.RealName, body.Email, body.Role); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	// companies 为 nil（请求里没带这个字段）表示本次不改动；
	// 显式传空数组 [] 才是「清空权限 = 恢复不受限」。
	// 注意：把账号改成管理员时不清空已配公司 —— 管理员本来就不受限，
	// 保留配置是为了将来改回普通用户时不用重新配。
	if body.Companies != nil {
		if err := h.UserCompanySvc.SetCompanies(id, body.Companies); err != nil {
			utils.Fail(c, -1, fmt.Sprintf("资料已更新，但公司权限设置失败：%s", err.Error()))
			return
		}
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
