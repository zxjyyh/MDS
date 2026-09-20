package handler

import (
	"net/url"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/zxjyyh/mds/internal/utils"
)

// 备份文件名含中文，这里统一用 query 传（?name=...），
// 不走路径参数，避免不同 HTTP 栈对百分号编码处理不一致。
func (h *Handler) GetBackupSettings(c *gin.Context) {
	st, err := h.BackupSvc.LoadSettings()
	if err != nil {
		utils.Fail(c, -1, "读取备份设置失败")
		return
	}
	utils.Success(c, st)
}

func (h *Handler) UpdateBackupSettings(c *gin.Context) {
	var body struct {
		Enabled       bool `json:"enabled"`
		IntervalHours int  `json:"intervalHours"`
		Retention     int  `json:"retention"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Fail(c, -1, "参数错误")
		return
	}
	if err := h.BackupSvc.SaveSettings(body.Enabled, body.IntervalHours, body.Retention); err != nil {
		utils.Fail(c, -1, "保存失败："+err.Error())
		return
	}
	st, _ := h.BackupSvc.LoadSettings()
	utils.SuccessMsg(c, st, "保存成功")
}

func (h *Handler) ListBackups(c *gin.Context) {
	list, err := h.BackupSvc.List()
	if err != nil {
		utils.Fail(c, -1, "读取备份列表失败")
		return
	}
	utils.Success(c, gin.H{"list": list, "dir": h.BackupSvc.Dir()})
}

// RunBackupNow 立即执行一次备份（不等定时器）。
func (h *Handler) RunBackupNow(c *gin.Context) {
	name, err := h.BackupSvc.RunAndRecord()
	if err != nil {
		utils.Fail(c, -1, "备份失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, gin.H{"name": name}, "备份完成")
}

func (h *Handler) DownloadBackup(c *gin.Context) {
	p, err := h.BackupSvc.ResolvePath(c.Query("name"))
	if err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}
	name := filepath.Base(p)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name))
	c.File(p)
}

func (h *Handler) DeleteBackup(c *gin.Context) {
	if err := h.BackupSvc.Remove(c.Query("name")); err != nil {
		utils.Fail(c, -1, "删除失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}
