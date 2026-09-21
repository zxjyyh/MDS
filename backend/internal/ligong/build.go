package ligong

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Attachment 一份来自人员档案的附件（证件扫描件）。
type Attachment struct {
	Name string // 展示用文件名（原始文件名）
	Data []byte
}

// Person 一名报名人员的完整数据。
//
// 身份字段（姓名 / 性别 / 身份证号 / 联系电话）一律由 handler 从人员档案读取，
// **不接受前端回传** —— 报名汇总表要盖章寄给培训机构，身份信息不能由请求体决定。
type Person struct {
	ID         int
	Name       string
	Gender     string // 男 / 女
	IDCard     string
	Phone      string
	Education  string // 学历
	Title      string // 职称
	Position   string // 检验岗位
	LineType   string // 检测线类别（仅环保表）
	DriveType  string // 准驾车型（仅环保表；引车员必填）
	Remark     string // 备注
	CertExpiry string // 证书有效日期（继续教育填进备注）

	// ApplyType 本人自己的报考类别（new / continue / add）。
	// 同一个证书类型下一批里可以混报不同类别，所以它挂在人身上而不是整批上。
	// 留空时退回 BuildInput.ApplyType（前端总会带上，这里只是兜底）。
	ApplyType string

	// Attachments：cert_name -> 附件列表（身份证 / 毕业证 / 驾驶证 扫描件）
	Attachments map[string][]Attachment

	// UploadedContract 档案里仍在有效期内的劳动合同上传件（非空时直接打包，不再生成）。
	// 过期判断见 ContractState；判定为可用的上传件才会落到这里。
	UploadedContract []Attachment
	// RegenerateContract 档案里没有可用劳动合同、且用户在二次确认里选了「是」时置 true，
	// 表示这份劳动合同由系统生成；为 false 则本次不出，由用户自行打印提供。
	RegenerateContract bool
}

// ContractRenderer 生成某人的劳动合同 docx，返回（字节, 文件名）。
// 由 handler 注入，直接复用系统现有「劳动合同生成」的同一套逻辑，
// 保证这里产出的劳动合同与在「劳动合同生成」里单独下载的完全一致。
type ContractRenderer func(p Person) ([]byte, string, error)

// BuildInput 生成入参（依赖项已由 handler 备齐，本包不碰数据库与磁盘）。
type BuildInput struct {
	Spec          *CertSpec
	ApplyType     string
	Period        string // 已清洗为纯数字
	EnrollDate    time.Time
	CompanyName   string
	Receiver      string
	ReceiverPhone string
	MailAddress   string
	// SafetyExam 安全责任书里的考试类型；为空时按证书类型取默认值
	SafetyExam string
	Persons    []Person
	// Contract 为 nil 时跳过劳动合同生成（「增项」不需要劳动合同）
	Contract ContractRenderer
}

// BuildResult 生成结果。
type BuildResult struct {
	Data     []byte
	FileName string
	// Missing 缺件明细（"姓名\t材料名"），供界面提示与压缩包内的缺件清单
	Missing []string
	// ContractFromArchive 复用了档案里已上传劳动合同的人员（未重新生成）
	ContractFromArchive []string
	// ContractGenerated 本次由系统生成劳动合同的人员
	ContractGenerated []string
	// ContractSelfProvided 本次未出劳动合同、需自行打印提供的人员
	ContractSelfProvided []string
}

// applyTypeOf 取某人自己的报考类别；为空时退回整批默认值（兜底，前端总会带上）。
func (in *BuildInput) applyTypeOf(p Person) string {
	if v := strings.TrimSpace(p.ApplyType); v != "" {
		return v
	}
	return strings.TrimSpace(in.ApplyType)
}

// Build 生成一整套报名材料的 ZIP。
func Build(in *BuildInput) (*BuildResult, error) {
	spec := in.Spec
	if spec == nil {
		return nil, fmt.Errorf("不支持的证书类型")
	}
	if len(in.Persons) == 0 {
		return nil, fmt.Errorf("请至少选择一名报名人员")
	}
	if len(in.Persons) > spec.MaxRows() {
		return nil, fmt.Errorf("%s报名汇总表最多容纳 %d 人，当前已选 %d 人", spec.Label, spec.MaxRows(), len(in.Persons))
	}

	// 报考类别按人独立（用户 2026-09-20 要求）：同一个证书类型下，
	// 一批里可以同时有新人员 / 继续教育 / 增项，汇总表逐行写各自的类别，
	// 材料也按各自类别收集（增项的人不附证件、不出劳动合同）。
	applySet := map[string]bool{}
	for _, p := range in.Persons {
		applySet[in.applyTypeOf(p)] = true
	}
	applyLabels := make([]string, 0, len(applySet))
	for _, k := range []string{ApplyNew, ApplyConti, ApplyAdd} {
		if applySet[k] {
			applyLabels = append(applyLabels, ApplyLabel(k))
		}
	}
	// 文件名里的类别：单一类别就写它，多类别用 + 连接
	applyCN := strings.Join(applyLabels, "+")
	if applyCN == "" {
		applyCN = ApplyLabel(in.ApplyType)
	}

	dateCN := in.EnrollDate.Format("2006年01月02日")
	dateFile := in.EnrollDate.Format("20060102")

	// ---- 1. 报名汇总表 ----
	tpl, err := LoadEnrollTemplate(spec)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(in.Persons))
	for _, p := range in.Persons {
		personApply := in.applyTypeOf(p)
		row := map[string]string{
			FieldName:      p.Name,
			FieldGender:    p.Gender,
			FieldEducation: p.Education,
			FieldTitle:     p.Title,
			FieldIDCard:    p.IDCard,
			FieldPhone:     p.Phone,
			FieldPosition:  p.Position,
			FieldApply:     ApplyLabel(personApply),
			FieldLineType:  p.LineType,
			FieldRemark:    p.Remark,
		}
		// 继续教育：备注栏写「证书有效日期」（模板填表说明第 3 条的要求）
		if personApply == ApplyConti {
			if v := spec.RemarkValue(p.CertExpiry); v != "" {
				row[FieldRemark] = v
			}
		}
		// 准驾车型只对引车员有意义，其余留空
		if spec.Key == CertEnv && IsDriver(p.Position) {
			row[FieldDriveType] = p.DriveType
		}
		rows = append(rows, row)
	}
	sheet, err := RenderEnrollSheet(tpl, spec, map[string]string{
		"单位名称": in.CompanyName,
		"报名日期": dateCN,
		"收件人":  in.Receiver,
		"收件电话": in.ReceiverPhone,
		"邮寄地址": in.MailAddress,
	}, rows)
	if err != nil {
		return nil, err
	}

	// ---- 2. 安全责任书 ----
	// 多人报名时报名人固定写「本次报名全体人员」（同 skill 口径），单人时写姓名。
	letterName := in.Persons[0].Name
	if len(in.Persons) > 1 {
		letterName = "本次报名全体人员"
	}
	exams := safetyExams(spec, in.SafetyExam, in.Persons)
	letters := make([]renderedLetter, 0, len(exams))
	for _, exam := range exams {
		data, err := RenderSafetyLetter(spec, letterName, in.Period, exam, in.CompanyName, dateCN)
		if err != nil {
			return nil, err
		}
		letters = append(letters, renderedLetter{exam: exam, data: data})
	}

	// ---- 3. 打包 ----
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	missing := []string{}

	// 本批是否存在需要收集证件的类别（增项不需要）→ 决定要不要出「缺件清单」。
	// 混报时只要有人是 新人员/继续教育，就要出。
	anyMaterial := false
	for _, p := range in.Persons {
		if len(MaterialsFor(in.applyTypeOf(p))) > 0 {
			anyMaterial = true
			break
		}
	}

	writeBytes := func(name string, data []byte) error {
		w, err := zw.Create(sanitizeZipPath(name))
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	// 报名汇总表与安全责任书**直接放压缩包根目录**：
	// 一个文件夹里只装一个文件没必要，解压后还要多点一层（用户 2026-09-20 反馈）。
	// 证件类材料仍然按类型分目录 —— 那些一个目录里有几十份扫描件，分类是有用的。
	sheetName := fmt.Sprintf("%s_%s报名汇总表_第%s期_%s.xlsx",
		sanitizeName(in.CompanyName), spec.Label, in.Period, dateFile)
	if err := writeBytes(sheetName, sheet); err != nil {
		return nil, err
	}

	letterFileName := func(l renderedLetter) string {
		suffix := ""
		if len(letters) > 1 {
			suffix = "（" + examShort(spec, l.exam) + "）"
		}
		return fmt.Sprintf("%s_%s安全责任书%s_第%s期_%s.docx",
			sanitizeName(letterName), spec.Label, suffix, in.Period, dateFile)
	}
	for _, l := range letters {
		if err := writeBytes(letterFileName(l), l.data); err != nil {
			return nil, err
		}
	}

	// 3.1 逐人收集材料（按报考类别）
	contractFromArchive := []string{} // 直接复用档案里已上传劳动合同的人
	contractGenerated := []string{}   // 本次由系统生成劳动合同的人
	contractSelf := []string{}        // 本次不出劳动合同、由用户自行打印提供的人
	for _, p := range in.Persons {
		for _, m := range MaterialsFor(in.applyTypeOf(p)) {
			collect, optional := spec.ResolveMaterial(m, p)
			if !collect {
				continue
			}
			if m.CanGenerate {
				// 劳动合同：档案里有仍在有效期内的上传件就直接用它，绝不重复生成
				// （用户 2026-09-20 确认的口径）
				if len(p.UploadedContract) > 0 {
					for _, f := range p.UploadedContract {
						name := fmt.Sprintf("%s/%s", m.Key, attachName(p.Name, f.Name))
						if err := writeBytes(name, f.Data); err != nil {
							return nil, err
						}
					}
					contractFromArchive = append(contractFromArchive, p.Name)
					continue
				}
				if p.RegenerateContract && in.Contract != nil {
					data, fname, err := in.Contract(p)
					if err != nil {
						return nil, fmt.Errorf("生成 %s 的劳动合同失败：%w", p.Name, err)
					}
					if err := writeBytes(fmt.Sprintf("%s/%s", m.Key, sanitizeName(fname)), data); err != nil {
						return nil, err
					}
					contractGenerated = append(contractGenerated, p.Name)
					continue
				}
				// 用户在二次确认里选了「否」：本次不生成，也不计入缺件（由他自己打印提供）
				contractSelf = append(contractSelf, p.Name)
				continue
			}

			files := p.Attachments[m.CertName]
			if len(files) == 0 {
				// 非必填材料（安检的驾驶证）「有则附、无则不附」，不进缺件清单
				if !optional {
					missing = append(missing, fmt.Sprintf("%s\t%s", p.Name, m.Name))
				}
				continue
			}
			for _, f := range files {
				name := fmt.Sprintf("%s/%s", m.Key, attachName(p.Name, f.Name))
				if err := writeBytes(name, f.Data); err != nil {
					return nil, err
				}
			}
		}
	}

	// 3.2 说明与缺件清单
	readme := buildReadme(in, spec, applyCN, dateCN, rows, readmeExtra{
		letterCount:       len(letters),
		applyLabels:       applyLabels,
		anyMaterial:       anyMaterial,
		contractFromFiles: contractFromArchive,
		contractGenerated: contractGenerated,
		contractSelf:      contractSelf,
	})
	if err := writeBytes("00_报名说明.txt", []byte(readme)); err != nil {
		return nil, err
	}
	if anyMaterial {
		content := "姓名\t缺失材料\n"
		if len(missing) == 0 {
			content = "本批所有人员的证件材料齐全，无需补办。\n"
		} else {
			content += strings.Join(missing, "\n") + "\n"
		}
		if err := writeBytes("01_缺件清单.txt", []byte(content)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	zipName := fmt.Sprintf("%s_%s%s报名材料_第%s期_%s.zip",
		sanitizeName(in.CompanyName), spec.Label, applyCN, in.Period, dateFile)
	return &BuildResult{
		Data:                 buf.Bytes(),
		FileName:             zipName,
		Missing:              missing,
		ContractFromArchive:  contractFromArchive,
		ContractGenerated:    contractGenerated,
		ContractSelfProvided: contractSelf,
	}, nil
}

// readmeExtra 说明文件里需要的几个「本次实际怎么处理」的小结。
type readmeExtra struct {
	letterCount       int
	applyLabels       []string // 本批实际涉及的报考类别（中文名），可能不止一种
	anyMaterial       bool     // 是否有需要收集证件的类别（增项不需要）
	contractFromFiles []string
	contractGenerated []string
	contractSelf      []string
}

// buildReadme 生成压缩包内的说明文件。
func buildReadme(in *BuildInput, spec *CertSpec, applyCN, dateCN string, rows []map[string]string, ex readmeExtra) string {
	var b strings.Builder
	fmt.Fprintf(&b, "理工项目报名材料\n")
	fmt.Fprintf(&b, "生成时间：%s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "证书类型：%s\n", spec.Label)
	fmt.Fprintf(&b, "报考类别：%s\n", applyCN)
	fmt.Fprintf(&b, "培训期数：第%s期\n", in.Period)
	fmt.Fprintf(&b, "报名日期：%s\n", dateCN)
	fmt.Fprintf(&b, "单位名称：%s\n", in.CompanyName)
	fmt.Fprintf(&b, "收件人：%s    联系电话：%s\n", in.Receiver, in.ReceiverPhone)
	fmt.Fprintf(&b, "邮寄地址：%s\n", in.MailAddress)
	fmt.Fprintf(&b, "报名人数：%d 人\n\n", len(in.Persons))
	b.WriteString("人员名单：\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %s\t%s\t%s\t%s\n", r[FieldName], r[FieldGender], r[FieldPhone], r[FieldApply])
	}

	// 材料构成：取本批实际涉及类别的**并集**（混报时不是单一类别的清单）
	included := map[string]Material{}
	for _, p := range in.Persons {
		for _, m := range MaterialsFor(in.applyTypeOf(p)) {
			included[m.Key] = m
		}
	}
	b.WriteString("\n材料构成：\n")
	b.WriteString("  " + padLabel("根目录") + "报名汇总表、安全责任书（加盖公章后寄出，直接就是这两个文件）\n")
	// 用「新人员」的材料清单当规范顺序，避免 map 遍历顺序不稳定
	for _, m := range MaterialsFor(ApplyNew) {
		if _, ok := included[m.Key]; !ok {
			continue
		}
		switch {
		case m.CanGenerate:
			b.WriteString("  " + padLabel("劳动合同/") + "优先复用人员档案里已上传且在有效期内的劳动合同；档案里没有可用件时才由系统生成\n")
		case m.OnlyDriver:
			// 驾驶证：环保仅引车员必附；安检则是「档案里有就一并打包」的非必填材料
			if spec.Key == CertEnv {
				b.WriteString("  " + padLabel(m.Key+"/") + "仅引车员需要\n")
			} else {
				b.WriteString("  " + padLabel(m.Key+"/") + "档案里已上传则一并附上（非必填，缺失不计缺件）\n")
			}
		default:
			b.WriteString("  " + padLabel(m.Key+"/") + "取自人员档案已上传的证件原件\n")
		}
	}

	if len(ex.applyLabels) > 1 {
		fmt.Fprintf(&b, "\n说明：本批同时含多种报考类别（%s），材料按每人各自类别收集。\n"+
			"      「增项」人员只出报名汇总表与安全责任书，不附证件与劳动合同。\n",
			strings.Join(ex.applyLabels, "、"))
	}
	if ex.letterCount > 1 {
		fmt.Fprintf(&b, "\n说明：本批同时含主任检验员与普通检验员，安全责任书已按培训班级各出一份（共 %d 份）。\n", ex.letterCount)
	}
	if len(ex.contractFromFiles) > 0 {
		fmt.Fprintf(&b, "\n劳动合同复用档案上传件（未重新生成）：%s\n", strings.Join(ex.contractFromFiles, "、"))
	}
	if len(ex.contractGenerated) > 0 {
		fmt.Fprintf(&b, "劳动合同本次由系统生成：%s\n", strings.Join(ex.contractGenerated, "、"))
	}
	if len(ex.contractSelf) > 0 {
		fmt.Fprintf(&b, "劳动合同本次未生成，请自行打印提供：%s\n", strings.Join(ex.contractSelf, "、"))
	}
	if ex.anyMaterial {
		b.WriteString("\n证件缺失情况详见「01_缺件清单.txt」。\n")
	}
	return b.String()
}

// renderedLetter 一份已生成的安全责任书。
type renderedLetter struct {
	exam string
	data []byte
}

// safetyExams 决定本批要出几份安全责任书、分别用哪个考试类型。
//
// 责任书正文是「现公司 X 参加贵处举办的第 B 期 C 培训」，C 是**培训班级类型**而非人名：
// 一批里若同时有主任检验员与普通检验员，一份责任书覆盖不了，必须各出一份。
// 环保只有一种考试类型；调用方显式指定 SafetyExam 时以指定值为准（只出一份）。
func safetyExams(spec *CertSpec, override string, persons []Person) []string {
	if o := strings.TrimSpace(override); o != "" {
		return []string{o}
	}
	if spec.SafetyExamDirector == "" {
		return []string{spec.SafetyExam}
	}
	normal, director := false, false
	for _, p := range persons {
		if strings.Contains(p.Position, "主任") {
			director = true
		} else {
			normal = true
		}
	}
	out := make([]string, 0, 2)
	if normal {
		out = append(out, spec.SafetyExam)
	}
	if director {
		out = append(out, spec.SafetyExamDirector)
	}
	return out
}

// examShort 考试类型的短名，用于「一批出两份责任书」时的文件名区分。
func examShort(spec *CertSpec, exam string) string {
	switch exam {
	case spec.SafetyExamDirector:
		return "主任检验员"
	case spec.SafetyExam:
		return "普通检验员"
	}
	return exam
}

// displayWidth 估算等宽字体下的显示宽度：CJK 记 2 列，其余记 1 列。
// 说明文件是给人看的 txt，用 rune 数对齐会因为中文占两列而错位。
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if r > 0x2E7F {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// padLabel 把目录名补到统一宽度，便于阅读。
func padLabel(s string) string {
	const width = 14
	if d := width - displayWidth(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s + " "
}

// parseDate 尝试几种常见的日期写法，解析不出来返回 false。
func parseDate(s string) (time.Time, bool) {
	v := strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "2006/01/02", "2006.01.02"} {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// formatDateCN 把日期转成 YYYY年MM月DD日；无法解析时原样返回。
func formatDateCN(s string) string {
	if t, ok := parseDate(s); ok {
		return t.Format("2006年01月02日")
	}
	return s
}

// formatDateISO 把日期统一成 YYYY-MM-DD（短横线），用于列宽吃紧的表格。
func formatDateISO(s string) string {
	if t, ok := parseDate(s); ok {
		return t.Format("2006-01-02")
	}
	return strings.TrimSpace(s)
}

// RemarkValue 备注栏要填的内容。
//
// 目前只有一种来源：继续教育的「证书有效日期」。模板填表说明要求
// 「继续教育人员要在备注栏填写证书有效日期」，所以它不是自由文本。
// 日期写法按证书类型区分 —— 环保表备注列窄，用 YYYY-MM-DD 免得撑破单元格。
func (s *CertSpec) RemarkValue(certExpiry string) string {
	d := strings.TrimSpace(certExpiry)
	if d == "" {
		return ""
	}
	if s.CompactExpiryDate {
		return formatDateISO(d)
	}
	return formatDateCN(d)
}

// attachName 给附件名加姓名前缀便于区分；原始文件名里**已经含**这个人名时不再重复加。
// 实际操作者的命名习惯就有「张三_身份证.jpg」「李四 劳动合同28-09-02.pdf」两种，
// 一律加前缀会变成「张三_张三_身份证.jpg」。
func attachName(personName, fileName string) string {
	pn := sanitizeName(personName)
	fn := sanitizeName(fileName)
	if strings.Contains(fn, pn) {
		return fn
	}
	return pn + "_" + fn
}

var invalidNameRe = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

// sanitizeName 清洗文件名片段：剥掉路径分隔符与控制字符，并压掉首尾空白。
func sanitizeName(s string) string {
	s = invalidNameRe.ReplaceAllString(strings.TrimSpace(s), "_")
	s = strings.Trim(s, ".")
	if s == "" {
		return "未命名"
	}
	return s
}

// sanitizeZipPath 逐段清洗 ZIP 内路径，防止出现 ../ 之类的穿越段。
func sanitizeZipPath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if i == len(parts)-1 {
			parts[i] = sanitizeFileName(s)
			continue
		}
		parts[i] = sanitizeName(s)
	}
	return strings.Join(parts, "/")
}

// sanitizeFileName 与 sanitizeName 同义，但保留扩展名前的点。
func sanitizeFileName(s string) string {
	s = invalidNameRe.ReplaceAllString(strings.TrimSpace(s), "_")
	if s == "" {
		return "未命名"
	}
	return s
}
