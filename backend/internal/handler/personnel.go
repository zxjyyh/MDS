package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListPersonnel(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	list, total, err := h.PersonnelSvc.List(currentScope(c), c.Query("keyword"), c.Query("department"), c.Query("status"), page, pageSize)
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Paginated(c, list, total, page, pageSize)
}

// PersonnelOptions 返回用于筛选下拉框的检测站去重列表。
func (h *Handler) PersonnelOptions(c *gin.Context) {
	departments, err := h.PersonnelSvc.Options(currentScope(c))
	if err != nil {
		utils.Fail(c, -1, "查询失败")
		return
	}
	utils.Success(c, gin.H{"departments": departments})
}

func (h *Handler) GetPersonnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.PersonnelSvc.GetByID(currentScope(c), id)
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
	p, err := h.PersonnelSvc.Create(currentScope(c), body, utils.GetAuthUserID(c))
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
	if err := h.PersonnelSvc.Update(currentScope(c), id, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeletePersonnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.PersonnelSvc.Delete(currentScope(c), id); err != nil {
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
	cert, err := h.PersonnelSvc.AddCert(currentScope(c), id, body)
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
	if err := h.PersonnelSvc.UpdateCert(currentScope(c), certID, body); err != nil {
		utils.Fail(c, -1, "更新失败")
		return
	}
	utils.SuccessMsg(c, nil, "更新成功")
}

func (h *Handler) DeletePersonnelCert(c *gin.Context) {
	certID, _ := strconv.Atoi(c.Param("certId"))
	if err := h.PersonnelSvc.DeleteCert(currentScope(c), certID); err != nil {
		utils.Fail(c, -1, "删除失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}

// ClearPersonnelCert 清空固定预留证件行（安检证/环保证）的已填信息与已上传附件，
// 证书行本身保留，回到"待上传"状态。
func (h *Handler) ClearPersonnelCert(c *gin.Context) {
	certID, _ := strconv.Atoi(c.Param("certId"))
	if err := h.PersonnelSvc.ClearReservedCert(currentScope(c), certID); err != nil {
		utils.Fail(c, -1, "清空失败："+err.Error())
		return
	}
	utils.SuccessMsg(c, nil, "清空成功")
}

// ExportPersonnelCerts 按「所属检测站 × 在职状态」范围，把指定名称的证书原件
// （安检证/环保证/身份证）打包为 ZIP 下载。
// 压缩包内按 证书类型/检测站/姓名 分目录归档；范围内未上传该证书的人员
// 会写入「01_缺证人员名单.txt」，便于管理人员督办补办。
func (h *Handler) ExportPersonnelCerts(c *gin.Context) {
	department := strings.TrimSpace(c.Query("department"))
	status := strings.TrimSpace(c.Query("status"))
	certName := strings.TrimSpace(c.Query("certName"))
	switch certName {
	case "安检证", "环保证", "身份证":
	default:
		utils.Fail(c, -1, "不支持的证书类型")
		return
	}

	people, err := h.PersonnelSvc.PersonnelByScope(currentScope(c), department, status)
	if err != nil {
		utils.Fail(c, -1, "查询失败："+err.Error())
		return
	}
	files, err := h.PersonnelSvc.CertFilesByScope(currentScope(c), department, status, certName)
	if err != nil {
		utils.Fail(c, -1, "查询失败："+err.Error())
		return
	}
	if len(people) == 0 {
		utils.Fail(c, -1, "所选范围内没有符合条件的人员")
		return
	}
	if len(files) == 0 {
		utils.Fail(c, -1, fmt.Sprintf("所选范围 %d 人均未上传「%s」，没有可打包的文件", len(people), certName))
		return
	}

	// 计算缺证名单（范围内未持有该证书附件的人员）
	have := make(map[int]bool, len(files))
	for _, f := range files {
		have[f.PersonnelID] = true
	}
	missingNames := make([]string, 0, len(people))
	for _, p := range people {
		if have[p.ID] {
			continue
		}
		missingNames = append(missingNames, fmt.Sprintf("%s\t%s\t%s",
			p.Name, exportDeptLabel(p.Department), exportStatusLabel(p.Status)))
	}

	now := time.Now()
	zipName := fmt.Sprintf("%s批量下载_%s.zip", certName, now.Format("2006-01-02_15-04-05"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(zipName))
	c.Header("X-Export-Total", strconv.Itoa(len(files)))
	c.Header("X-Export-Skipped", strconv.Itoa(len(missingNames)))

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	writeTxt := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, content)
		return err
	}

	scopeDept := department
	if scopeDept == "" {
		scopeDept = "全部检测站"
	}
	scopeStatus := exportStatusLabel(status)
	if scopeStatus == "" {
		scopeStatus = "全部状态"
	}
	readme := fmt.Sprintf("证书批量下载说明\n下载时间：%s\n证书类型：%s\n下载范围：%s · %s\n人员总数：%d 人\n打包文件：%d 份\n缺证人数：%d 人\n",
		now.Format("2006-01-02 15:04:05"), certName, scopeDept, scopeStatus, len(people), len(files), len(missingNames))
	if len(missingNames) > 0 {
		readme += "\n说明：缺证人员详见压缩包内「01_缺证人员名单.txt」。\n"
	}
	_ = writeTxt("00_下载说明.txt", readme)

	if len(missingNames) > 0 {
		_ = writeTxt("01_缺证人员名单.txt", "姓名\t所属检测站\t在职状态\n"+strings.Join(missingNames, "\n")+"\n")
	}

	// 归档：证书类型/检测站/姓名/原始文件名（同名文件自动加序号）
	seen := make(map[string]int)
	failures := make([]string, 0)
	for _, f := range files {
		orig := strings.TrimSpace(f.OriginalName)
		if orig == "" {
			orig = "附件"
		}
		base := fmt.Sprintf("%s/%s/%s", certName, sanitizeExportName(exportDeptLabel(f.Department)), sanitizeExportName(f.Name))
		name := base + "/" + orig
		seen[name]++
		if seen[name] > 1 {
			ext := ""
			stem := orig
			if i := strings.LastIndex(orig, "."); i > 0 {
				ext = orig[i:]
				stem = orig[:i]
			}
			name = base + "/" + fmt.Sprintf("%s(%d)%s", stem, seen[name], ext)
		}

		data, err := os.ReadFile(f.StoragePath)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s（%s）：读取服务器文件失败：%v", f.Name, f.StoragePath, err))
			continue
		}
		w, err := zw.Create(name)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s：写入压缩包失败：%v", name, err))
			continue
		}
		if _, err := w.Write(data); err != nil {
			failures = append(failures, fmt.Sprintf("%s：写入压缩包失败：%v", name, err))
		}
	}
	if len(failures) > 0 {
		_ = writeTxt("99_处理失败清单.txt", strings.Join(failures, "\n")+"\n")
	}
}

// exportStatusLabel 把人员状态码转成中文标签。
func exportStatusLabel(status string) string {
	switch status {
	case "active":
		return "在职"
	case "resigned":
		return "离职"
	}
	return ""
}

// exportDeptLabel 处理部门为空的情况。
func exportDeptLabel(dept *string) string {
	if dept == nil || strings.TrimSpace(*dept) == "" {
		return "未分组"
	}
	return strings.TrimSpace(*dept)
}

// sanitizeExportName 去掉路径分隔符等会破坏压缩包目录结构的字符。
func sanitizeExportName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	name = strings.Trim(name, ". ")
	if name == "" {
		return "未命名"
	}
	return name
}

// PersonnelCertGap 证件查漏：按检测站 / 在职状态 + 证书类型，列出人员的证件状态。
// department / status / certs 都留空时表示「全部范围 + 全部证书类型」。
func (h *Handler) PersonnelCertGap(c *gin.Context) {
	certs := []string{}
	if raw := strings.TrimSpace(c.Query("certs")); raw != "" {
		certs = strings.Split(raw, ",")
	}
	names, list, summary, err := h.PersonnelSvc.CertGap(currentScope(c), c.Query("department"), c.Query("status"), certs)
	if err != nil {
		utils.Fail(c, -1, "查询失败："+err.Error())
		return
	}
	utils.Success(c, gin.H{
		"certNames": names,
		"list":      list,
		"summary":   summary,
		"total":     len(list),
	})
}
