package handler

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/zxjyyh/mds/internal/docgen"
	"github.com/zxjyyh/mds/internal/scope"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

// contractGenRequest 劳动合同生成请求参数。
// 前端填写（签订日期/期限/甲方地址/甲方名称可改），乙方资料由后端按 personnelId 从库里读取，不信任前端回传。
type contractGenRequest struct {
	PersonnelID    int    `json:"personnelId"`    // 人员 id
	SignDate       string `json:"signDate"`       // 签订日期 YYYY-MM-DD，默认昨天
	StationName    string `json:"stationName"`    // 甲方（用人单位/所属检测站）名称，默认取人员 department
	StationAddress string `json:"stationAddress"` // 甲方地址（检测站地址，手填必填）
	Years          int    `json:"years"`          // 合同年限，默认 3
}

// contractGenderCN 把库里的 gender(male/female/unknown) 转成合同用中文；
// unknown 时尝试按身份证第 17 位推断（奇男偶女）。
func contractGenderCN(gender, idCard string) string {
	switch gender {
	case "male":
		return "男"
	case "female":
		return "女"
	}
	if len(idCard) >= 18 {
		d := idCard[16]
		if d >= '0' && d <= '9' {
			if (d-'0')%2 == 1 {
				return "男"
			}
			return "女"
		}
	}
	return ""
}

// dateCN 把日期转成合同里的中文样式：2025年07月10日
func dateCN(t time.Time) string {
	return t.Format("2006年01月02日")
}

func strPtrVal(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// buildContractDocx 生成劳动合同 docx 字节与建议文件名（校验 + 取值 + 渲染）。
//
// 抽出来是为了让「理工报名材料」批量生成能复用**完全同一套**逻辑：
// 两条路径产出的劳动合同必须一字不差，不能各写一份模板替换代码。
func (h *Handler) buildContractDocx(sc *scope.Scope, req contractGenRequest) ([]byte, string, error) {
	if req.PersonnelID <= 0 {
		return nil, "", fmt.Errorf("请先选择人员")
	}

	p, err := h.PersonnelSvc.GetByID(sc, req.PersonnelID)
	if err != nil || p == nil {
		return nil, "", fmt.Errorf("人员不存在")
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, "", fmt.Errorf("人员姓名为空，请先完善人员档案")
	}

	// 签订日期
	signDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.SignDate), time.Local)
	if err != nil {
		return nil, "", fmt.Errorf("签订日期格式应为 YYYY-MM-DD")
	}
	years := req.Years
	if years <= 0 {
		years = 3
	}
	endDate := signDate.AddDate(years, 0, 0).AddDate(0, 0, -1) // 3 年 -1 天，与范本一致

	// 甲方（所属检测站）
	stationName := strings.TrimSpace(req.StationName)
	if stationName == "" {
		stationName = strPtrVal(p.Department)
	}
	if stationName == "" {
		return nil, "", fmt.Errorf("甲方名称缺失：该人员未填写所属检测站，请手动填写")
	}
	// 甲方地址：前端会按「甲方名称」从公司信息管理自动带出；这里再兜一次底 ——
	// 地址为空时直接按甲方（公司）名称回查 companies 表，保证两份数据始终一致。
	stationAddr := strings.TrimSpace(req.StationAddress)
	if stationAddr == "" {
		stationAddr = h.CompanySvc.AddressByName(stationName)
	}
	if stationAddr == "" {
		return nil, "", fmt.Errorf("甲方地址缺失：请手动填写，或先在「公司信息管理」为该甲方补全公司地址")
	}

	// 乙方（人员）资料
	idCard := strPtrVal(p.IDCard)
	if idCard == "" {
		return nil, "", fmt.Errorf("该人员未填写身份证号，请先完善人员档案")
	}
	addr := strPtrVal(p.Address)
	if addr == "" {
		return nil, "", fmt.Errorf("该人员未填写住址，请先完善人员档案")
	}
	gender := contractGenderCN(p.Gender, idCard)
	if gender == "" {
		return nil, "", fmt.Errorf("该人员性别未知且无法从身份证推断，请先在人员档案中设置性别")
	}

	repl := map[string]string{
		"{{甲方名称}}":   stationName,
		"{{甲方地址}}":   stationAddr,
		"{{乙方姓名}}":   name,
		"{{乙方性别}}":   gender,
		"{{乙方身份证号}}": idCard,
		"{{乙方住址}}":   addr,
		"{{签订日期}}":   dateCN(signDate),
		"{{合同起始日期}}": dateCN(signDate),
		"{{合同结束日期}}": dateCN(endDate),
	}

	// 模板目录：backend/templates（存在则免编译生效），否则回退 go:embed 内嵌副本
	data, err := docgen.RenderContract(repl, "templates")
	if err != nil {
		return nil, "", fmt.Errorf("合同模板渲染失败：%w", err)
	}
	return data, fmt.Sprintf("%s-聘用劳动合同-%s.docx", name, signDate.Format("20060102")), nil
}

// GenerateContract 按人员档案 + 手填关键信息生成《聘用劳动合同》docx。
func (h *Handler) GenerateContract(c *gin.Context) {
	var req contractGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailStatus(c, 400, -1, "参数错误")
		return
	}

	data, fileName, err := h.buildContractDocx(currentScope(c), req)
	if err != nil {
		utils.FailStatus(c, 400, -1, err.Error())
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(fileName))
	_, _ = c.Writer.Write(data)
}
