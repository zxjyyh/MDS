package handler

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zxjyyh/mds/internal/ligong"
	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

// LigongOptions 返回「理工人员报名材料生成」界面所需的选项与材料清单。
// 选项只在 rules.go 里定义一份，前端不再各自硬编码，避免和模板说明对不上。
func (h *Handler) LigongOptions(c *gin.Context) {
	utils.Success(c, ligong.BuildOptions(h.Config.LigongDefaultReceiver))
}

// ligongPersonReq 一名报名人员的报名专用信息。
// 姓名 / 性别 / 身份证号 / 联系电话一律由后端从人员档案读取，这里的数据不参与身份字段。
type ligongPersonReq struct {
	PersonnelID int    `json:"personnelId"`
	Education   string `json:"education"`  // 学历（留空则取档案里的）
	Title       string `json:"title"`      // 职称
	Position    string `json:"position"`   // 报名岗位
	LineType    string `json:"lineType"`   // 检测线类别（环保）
	DriveType   string `json:"driveType"`  // 准驾车型（环保，引车员必填）
	Remark      string `json:"remark"`     // 备注
	CertExpiry  string `json:"certExpiry"` // 证书有效日期（继续教育填进备注）
	// ApplyType 本人报考类别（new / continue / add）。留空时取批次的 ApplyType。
	// 同一个证书类型下一批里可以混报，所以类别是逐人的。
	ApplyType string `json:"applyType"`
}

// resolveApplyType 取某人有效的报考类别：本人填的优先，否则用批次默认值。
// 返回的 key 一定合法（非法时返回空串，由调用方报错）。
func (r *ligongEnrollReq) resolveApplyType(pr ligongPersonReq) string {
	key := strings.TrimSpace(pr.ApplyType)
	if key == "" {
		key = strings.TrimSpace(r.ApplyType)
	}
	if !ligong.ValidApplyType(key) {
		return ""
	}
	return key
}

type ligongEnrollReq struct {
	CertType      string `json:"certType"`  // security | env
	ApplyType     string `json:"applyType"` // new | continue | add
	Period        string `json:"period"`    // 期数，允许「第8期」这类写法，自动清洗
	EnrollDate    string `json:"enrollDate"`
	CompanyName   string `json:"companyName"`
	Receiver      string `json:"receiver"`
	ReceiverPhone string `json:"receiverPhone"`
	MailAddress   string `json:"mailAddress"`
	// RegenerateContract：档案里没有「仍在有效期内的劳动合同上传件」时，
	// 是否由系统重新生成（由界面二次确认的结果决定）。
	// 为 false 表示本次不出劳动合同，由用户自行打印提供。
	RegenerateContract bool `json:"regenerateContract"`
	// 劳动合同：签订日期与甲方地址（一律复用系统现有「劳动合同生成」的逻辑）
	ContractSignDate       string            `json:"contractSignDate"`
	ContractStationAddress string            `json:"contractStationAddress"`
	Persons                []ligongPersonReq `json:"persons"`
}

// ligongPrecheckPerson 预检时只需要知道某人报的是什么类别（增项不涉及劳动合同）。
type ligongPrecheckPerson struct {
	PersonnelID int    `json:"personnelId"`
	ApplyType   string `json:"applyType"`
}

// ligongPrecheckReq 生成前的预检入参：只用来判断「劳动合同要不要重新生成」。
type ligongPrecheckReq struct {
	ApplyType  string                 `json:"applyType"`  // 批次默认类别（逐人留空时用）
	EnrollDate string                 `json:"enrollDate"` // 过期判定的基准日；留空取今天
	Persons    []ligongPrecheckPerson `json:"persons"`
	// PersonnelIDs 兼容旧调用：未传 persons 时按 ApplyType 统一处理
	PersonnelIDs []int `json:"personnelIds"`
}

// ligongContractRow 某人的劳动合同状态。
type ligongContractRow struct {
	PersonnelID int     `json:"personnelId"`
	Name        string  `json:"name"`
	State       string  `json:"state"` // uploaded | expired | missing
	StateText   string  `json:"stateText"`
	ExpiryDate  *string `json:"expiryDate"`
}

type ligongPrecheckResp struct {
	Contracts []ligongContractRow `json:"contracts"`
	// NeedDecision 需要用户决定「是否重新生成劳动合同」的人；为空表示无需弹窗
	NeedDecision []ligongContractRow `json:"needDecision"`
}

// LigongPrecheck 生成前的预检：判断本批人员的劳动合同能否直接复用档案里的上传件。
//
// 为什么要单独一个接口：界面需要在**下载前**知道要不要弹「是否重新生成劳动合同」的确认框，
// 而生成接口一经调用就直接吐 ZIP，没法回头再问。
func (h *Handler) LigongPrecheck(c *gin.Context) {
	var req ligongPrecheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailStatus(c, 400, -1, "参数错误")
		return
	}
	resp := ligongPrecheckResp{Contracts: []ligongContractRow{}, NeedDecision: []ligongContractRow{}}
	base := h.ligongBaseDate(req.EnrollDate)

	// 逐人判定报考类别：「增项」只出汇总表与责任书，不涉及劳动合同，直接跳过。
	// 兼容旧调用（只传 personnelIds）时全员按批次 ApplyType 处理。
	entries := req.Persons
	if len(entries) == 0 {
		for _, id := range req.PersonnelIDs {
			entries = append(entries, ligongPrecheckPerson{PersonnelID: id, ApplyType: req.ApplyType})
		}
	}

	for _, e := range entries {
		applyKey := strings.TrimSpace(e.ApplyType)
		if applyKey == "" {
			applyKey = strings.TrimSpace(req.ApplyType)
		}
		if len(ligong.MaterialsFor(applyKey)) == 0 {
			continue // 增项：不需要劳动合同
		}
		p, err := h.PersonnelSvc.GetByID(e.PersonnelID)
		if err != nil || p == nil {
			continue
		}
		row := h.contractRow(p, base)
		resp.Contracts = append(resp.Contracts, row)
		if row.State != ligong.ContractUploaded {
			resp.NeedDecision = append(resp.NeedDecision, row)
		}
	}
	utils.Success(c, resp)
}

// ligongBaseDate 过期判定的基准日：优先用报名日期（「报名当天合同是否有效」才是业务口径），
// 解析不了或没传则退回今天。
func (h *Handler) ligongBaseDate(enrollDate string) time.Time {
	if s := strings.TrimSpace(enrollDate); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}

// contractRow 组装某人的劳动合同状态（不读附件字节，预检不需要）。
func (h *Handler) contractRow(p *models.Personnel, base time.Time) ligongContractRow {
	name := strings.TrimSpace(p.Name)
	hasFile := false
	expiry := ""
	for _, cert := range p.Certificates {
		if cert.CertName != ligong.CertNameContract {
			continue
		}
		if cert.FileID != nil {
			hasFile = true
		}
		if cert.ExpiryDate != nil && strings.TrimSpace(*cert.ExpiryDate) != "" {
			expiry = strings.TrimSpace(*cert.ExpiryDate)
		}
	}
	state := ligong.ContractState(hasFile, expiry, base)
	row := ligongContractRow{
		PersonnelID: p.ID,
		Name:        name,
		State:       state,
		StateText:   ligong.ContractStateLabel(state, expiry),
	}
	if expiry != "" {
		e := expiry
		row.ExpiryDate = &e
	}
	return row
}

var (
	periodStripRe = regexp.MustCompile(`[第期\s]`)
	digitRe       = regexp.MustCompile(`^\d+$`)
)

// GenerateLigongEnroll 生成一整套理工报名材料并打包为 ZIP 下载。
//
// 产物：报名汇总表 + 安全责任书（均按证书类型出对应版本）+ 证件原件（按报考类别）
// + 劳动合同（复用现有劳动合同生成逻辑）+ 报名说明与缺件清单。
func (h *Handler) GenerateLigongEnroll(c *gin.Context) {
	var req ligongEnrollReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailStatus(c, 400, -1, "参数错误")
		return
	}

	spec, ok := ligong.Spec(strings.TrimSpace(req.CertType))
	if !ok {
		utils.FailStatus(c, 400, -1, "不支持的证书类型")
		return
	}
	// 报考类别是**逐人**的（同证书类型下允许混报），这里不校验批次级字段 ——
	// 它在下面装配人员时按人校验（resolveApplyType 会给出「第 N 名人员未选择有效的报考类别」）。
	period := periodStripRe.ReplaceAllString(strings.TrimSpace(req.Period), "")
	if !digitRe.MatchString(period) {
		utils.FailStatus(c, 400, -1, "请填写培训期数（只需数字，如 8）")
		return
	}
	enrollDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.EnrollDate), time.Local)
	if err != nil {
		utils.FailStatus(c, 400, -1, "报名日期格式应为 YYYY-MM-DD")
		return
	}
	company := strings.TrimSpace(req.CompanyName)
	if company == "" {
		utils.FailStatus(c, 400, -1, "请填写单位名称（报名公司）")
		return
	}
	mailAddr := strings.TrimSpace(req.MailAddress)
	if mailAddr == "" {
		utils.FailStatus(c, 400, -1, "请填写邮寄地址，或先在「公司信息管理」为该单位补全公司地址")
		return
	}
	if len(req.Persons) == 0 {
		utils.FailStatus(c, 400, -1, "请至少选择一名报名人员")
		return
	}
	if len(req.Persons) > spec.MaxRows() {
		utils.FailStatus(c, 400, -1,
			fmt.Sprintf("%s报名汇总表最多容纳 %d 人，当前已选 %d 人", spec.Label, spec.MaxRows(), len(req.Persons)))
		return
	}

	// ---- 组装人员数据：身份字段全部以档案为准 ----
	// 劳动合同的过期判定基准日与预检保持一致（报名日期优先）
	baseDate := h.ligongBaseDate(req.EnrollDate)
	persons := make([]ligong.Person, 0, len(req.Persons))
	seen := map[int]bool{}
	for i, pr := range req.Persons {
		if pr.PersonnelID <= 0 {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("第 %d 名人员无效", i+1))
			return
		}
		if seen[pr.PersonnelID] {
			utils.FailStatus(c, 400, -1, "同一人员不能重复报名")
			return
		}
		seen[pr.PersonnelID] = true

		// 报考类别逐人判定（同一证书类型下允许混报）
		personApply := req.resolveApplyType(pr)
		if personApply == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("第 %d 名人员未选择有效的报考类别", i+1))
			return
		}

		p, err := h.PersonnelSvc.GetByID(pr.PersonnelID)
		if err != nil || p == nil {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("第 %d 名人员不存在，请刷新后重试", i+1))
			return
		}
		name := strings.TrimSpace(p.Name)
		if name == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("第 %d 名人员姓名为空，请先完善人员档案", i+1))
			return
		}
		idCard := strPtrVal(p.IDCard)
		if idCard == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("「%s」未填写身份证号，请先完善人员档案", name))
			return
		}
		phone := strPtrVal(p.Phone)
		if phone == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("「%s」未填写联系电话，请先完善人员档案", name))
			return
		}
		gender := contractGenderCN(p.Gender, idCard)
		if gender == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("「%s」性别未知且无法从身份证推断，请先在人员档案中设置性别", name))
			return
		}

		position := strings.TrimSpace(pr.Position)
		if position == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf("请为「%s」选择报名岗位", name))
			return
		}
		driveType := strings.TrimSpace(pr.DriveType)
		if spec.Key == ligong.CertEnv {
			if strings.TrimSpace(pr.LineType) == "" {
				utils.FailStatus(c, 400, -1, fmt.Sprintf("请为「%s」选择检测线类别", name))
				return
			}
			if ligong.IsDriver(position) && driveType == "" {
				// 说明清楚是「档案里没有」还是纯粹没填，避免用户以为是系统漏带
				hint := ""
				if strPtrVal(p.DriverLicense) == "" {
					hint = "（人员档案里未录入驾驶证准驾车型，请手工填写或先补录人员档案）"
				}
				utils.FailStatus(c, 400, -1, fmt.Sprintf("「%s」报名岗位含引车员，必须填写准驾车型%s", name, hint))
				return
			}
		}
		// 非引车员不写准驾车型（模板说明：准驾车型为引车员必填）
		if spec.Key == ligong.CertEnv && !ligong.IsDriver(position) {
			driveType = ""
		}

		// 学历：界面上填的优先，其次取人员档案。
		// 两边都没有时**直接报错**，不再兜底成某个默认学历 ——
		// 报名表上的学历必须真实，臆造一个值会把错误信息带进官方材料
		// （用户 2026-09-21 要求：取不到就让人手工填）
		education := strings.TrimSpace(pr.Education)
		if education == "" {
			education = strPtrVal(p.Education)
		}
		if education == "" {
			utils.FailStatus(c, 400, -1, fmt.Sprintf(
				"「%s」在人员档案里没有学历，请手工选择学历", name))
			return
		}

		// 证书有效期：只在「继续教育」时有意义（模板填表说明要求继续教育人员在备注栏填证书有效日期）。
		// 注意按**本人**的报考类别判断 —— 混报时同批里新人员不需要填。
		// 界面已按证书类型自动带出（环保取环保证到期时间、安检取安检证到期时间），
		// 这里再兜一次底 —— 前端选项尚未加载完等情况下也不能漏填。
		certExpiry := ""
		if personApply == ligong.ApplyConti {
			certExpiry = strings.TrimSpace(pr.CertExpiry)
			if certExpiry == "" {
				certExpiry = ligong.ArchiveCertExpiry(spec.Key, strPtrVal(p.SecurityCertDate), strPtrVal(p.EnvCertDate))
			}
			if certExpiry == "" {
				utils.FailStatus(c, 400, -1, fmt.Sprintf(
					"「%s」在人员档案里没有%s到期时间，请手工填写「证书有效期」", name, spec.Label))
				return
			}
		}

		person := ligong.Person{
			ID:        p.ID,
			Name:      name,
			Gender:    gender,
			IDCard:    idCard,
			Phone:     phone,
			Education: education,
			// 职称：人员档案里没有这个字段，界面没填就留空（不臆造）
			Title:      strings.TrimSpace(pr.Title),
			Position:   position,
			LineType:   strings.TrimSpace(pr.LineType),
			DriveType:  driveType,
			Remark:     strings.TrimSpace(pr.Remark),
			CertExpiry: certExpiry,
			ApplyType:  personApply,
		}
		// 证件附件只对「需要证件」的类别去读（增项不附证件，省掉无谓的磁盘读）
		if len(ligong.MaterialsFor(personApply)) > 0 {
			person.Attachments, person.UploadedContract = h.splitCertAttachments(p.Certificates, baseDate)
			// 档案里没有可用劳动合同 → 是否重新生成由界面二次确认的结果决定
			person.RegenerateContract = len(person.UploadedContract) == 0 && req.RegenerateContract
		}
		persons = append(persons, person)
	}

	// ---- 劳动合同渲染器：直接复用现有「劳动合同生成」的同一套逻辑 ----
	//
	// 只在「确实有人需要生成」时才构建：如果本批人员的劳动合同都能复用档案里的上传件，
	// 就完全不该要求用户填劳动合同签订日期/甲方地址 —— 那是无意义的必填项。
	needGenerate := false
	for _, p := range persons {
		if p.RegenerateContract {
			needGenerate = true
			break
		}
	}
	var renderer ligong.ContractRenderer
	if needGenerate {
		signDate := strings.TrimSpace(req.ContractSignDate)
		if signDate == "" {
			signDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		}
		if _, err := time.ParseInLocation("2006-01-02", signDate, time.Local); err != nil {
			utils.FailStatus(c, 400, -1, "劳动合同签订日期格式应为 YYYY-MM-DD")
			return
		}
		contractAddr := strings.TrimSpace(req.ContractStationAddress)
		if contractAddr == "" {
			contractAddr = h.CompanySvc.AddressByName(company)
		}
		if contractAddr == "" {
			utils.FailStatus(c, 400, -1, "需要重新生成劳动合同，请填写劳动合同甲方地址（或先在「公司信息管理」为该公司补全地址）")
			return
		}
		renderer = func(p ligong.Person) ([]byte, string, error) {
			return h.buildContractDocx(contractGenRequest{
				PersonnelID:    p.ID,
				SignDate:       signDate,
				StationName:    company,
				StationAddress: contractAddr,
				Years:          3,
			})
		}
	}

	res, err := ligong.Build(&ligong.BuildInput{
		Spec:          spec,
		ApplyType:     strings.TrimSpace(req.ApplyType),
		Period:        period,
		EnrollDate:    enrollDate,
		CompanyName:   company,
		Receiver:      strings.TrimSpace(req.Receiver),
		ReceiverPhone: strings.TrimSpace(req.ReceiverPhone),
		MailAddress:   mailAddr,
		Persons:       persons,
		Contract:      renderer,
	})
	if err != nil {
		utils.Fail(c, -1, "生成失败："+err.Error())
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(res.FileName))
	// 缺件明细与劳动合同的处理结果走自定义响应头回传（URL 编码保证纯 ASCII），供界面提示；
	// 明细同时写在压缩包内的「00_报名说明.txt」「01_缺件清单.txt」里。
	c.Header("X-Ligong-Missing-Count", strconv.Itoa(len(res.Missing)))
	c.Header("X-Ligong-Missing", url.PathEscape(strings.Join(res.Missing, ";")))
	c.Header("X-Ligong-Contract-Archive", url.PathEscape(strings.Join(res.ContractFromArchive, ";")))
	c.Header("X-Ligong-Contract-Generated", url.PathEscape(strings.Join(res.ContractGenerated, ";")))
	c.Header("X-Ligong-Contract-Self", url.PathEscape(strings.Join(res.ContractSelfProvided, ";")))
	_, _ = c.Writer.Write(res.Data)
}

// splitCertAttachments 归集某人的已上传证件附件，并把「劳动合同」单独摘出来。
//
// 劳动合同不按普通证件处理：它在有效期内时**直接复用上传件**、过期或缺失时才可能由系统生成，
// 所以这里要额外判一次可用性（见 ligong.ContractState）。
// 文件路径统一走 FileSvc 解析（历史数据里可能有 Windows 绝对路径等脏值），读不到的跳过。
func (h *Handler) splitCertAttachments(certs []models.PersonnelCertificate, base time.Time) (map[string][]ligong.Attachment, []ligong.Attachment) {
	byName := map[string][]ligong.Attachment{}
	hasContract := false
	contractExpiry := ""
	var contractFiles []ligong.Attachment

	for _, cert := range certs {
		if cert.FileID == nil {
			continue
		}
		path, _, originalName, err := h.FileSvc.GetFilePath(*cert.FileID)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(originalName)
		if name == "" {
			name = "附件"
		}
		att := ligong.Attachment{Name: name, Data: data}

		if cert.CertName != ligong.CertNameContract {
			byName[cert.CertName] = append(byName[cert.CertName], att)
			continue
		}
		hasContract = true
		if cert.ExpiryDate != nil && strings.TrimSpace(*cert.ExpiryDate) != "" {
			contractExpiry = strings.TrimSpace(*cert.ExpiryDate)
		}
		contractFiles = append(contractFiles, att)
	}

	// 只有判定为「可直接使用」时才回传上传件；过期件不打包（会被重新生成的那份取代）
	if ligong.ContractState(hasContract, contractExpiry, base) == ligong.ContractUploaded {
		return byName, contractFiles
	}
	return byName, nil
}
