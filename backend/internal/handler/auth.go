package handler

import (
	"net/http"
	"strconv"

	"github.com/zxjyyh/mds/internal/config"
	"github.com/zxjyyh/mds/internal/service"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Handler struct {
	DB     *sqlx.DB
	Config *config.Config
	Logger *zap.Logger

	AuthSvc         *service.AuthService
	UserSvc         *service.UserService
	PersonnelSvc    *service.PersonnelService
	EquipmentSvc    *service.EquipmentService
	CompanySvc      *service.CompanyService
	CategorySvc     *service.CategoryService
	OperationLogSvc *service.OperationLogService
	BackupSvc       *service.BackupService
	DocumentSvc     *service.DocumentService
	FileSvc         *service.FileService
	ReminderSvc     *service.ReminderService
}

func NewHandler(db *sqlx.DB, cfg *config.Config, log *zap.Logger) *Handler {
	return &Handler{
		DB:     db,
		Config: cfg,
		Logger: log,

		AuthSvc:         &service.AuthService{DB: db},
		UserSvc:         &service.UserService{DB: db},
		PersonnelSvc:    &service.PersonnelService{DB: db},
		EquipmentSvc:    &service.EquipmentService{DB: db},
		CompanySvc:      &service.CompanyService{DB: db},
		CategorySvc:     &service.CategoryService{DB: db},
		OperationLogSvc: &service.OperationLogService{DB: db},
		BackupSvc:       &service.BackupService{DB: db, Cfg: cfg},
		DocumentSvc:     &service.DocumentService{DB: db},
		FileSvc:         &service.FileService{DB: db, UploadDir: cfg.UploadDir, MaxSize: cfg.UploadMax},
		ReminderSvc:     &service.ReminderService{DB: db},
	}
}

// ---- Auth Handlers ----

func (h *Handler) Login(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}

	access, refresh, user, err := h.AuthSvc.Login(body.Username, body.Password)
	if err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}

	// 登录成功记一条操作日志：登录请求不经过 Auth 中间件，
	// 全局 OperationLog 中间件此时拿不到用户身份，只能在这里显式记录。
	// 登录失败不记，避免把密码爆破尝试混进操作日志。
	h.OperationLogSvc.Record(service.OperationEntry{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Method:   http.MethodPost,
		Path:     "/api/auth/login",
	})

	utils.SuccessMsg(c, gin.H{
		"accessToken":  access,
		"refreshToken": refresh,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"realName": user.RealName,
			"email":    user.Email,
			"role":     user.Role,
		},
	}, "登录成功")
}

func (h *Handler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}

	access, refresh, err := h.AuthSvc.Refresh(body.RefreshToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ApiResponse{Code: -1, Message: err.Error(), Data: nil})
		return
	}
	utils.Success(c, gin.H{"accessToken": access, "refreshToken": refresh})
}

// IssueFileToken 为当前登录用户签发文件访问令牌。
// 浏览器发起的 <img src> / 下载请求无法携带 Authorization 头，只能走 query，
// 因此这里单独发一个 30 分钟有效、且不能调用业务接口的令牌。
func (h *Handler) IssueFileToken(c *gin.Context) {
	userID := utils.GetAuthUserID(c)
	token, expiresIn, err := h.AuthSvc.IssueFileToken(userID)
	if err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.Success(c, gin.H{"token": token, "expiresIn": expiresIn})
}

// Logout 吊销当前用户已签发的全部令牌。
func (h *Handler) Logout(c *gin.Context) {
	userID := utils.GetAuthUserID(c)
	if err := h.AuthSvc.Logout(userID); err != nil {
		utils.Fail(c, -1, "登出失败")
		return
	}
	utils.SuccessMsg(c, nil, "已登出")
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := utils.GetAuthUserID(c)
	user, err := h.AuthSvc.GetProfile(userID)
	if err != nil {
		utils.Fail(c, -1, "用户不存在")
		return
	}
	utils.Success(c, user)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var body struct {
		OldPassword string `json:"oldPassword"`
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

	userID := utils.GetAuthUserID(c)
	if err := h.AuthSvc.ChangePassword(userID, body.OldPassword, body.NewPassword); err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "密码修改成功")
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok", "timestamp": "ok"})
}

// ---- Helpers ----

func strToInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func queryInt(c *gin.Context, key string, def int) int {
	return strToInt(c.Query(key), def)
}
