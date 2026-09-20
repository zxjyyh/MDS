// Package ligong 生成「理工项目人员报名」所需的一整套材料（ZIP 打包下载）。
//
// 与旧 skill 脚本（quanzhou-ligong-enrollment）的差别，都是刻意为之：
//
//   - 纯 Go、零新增依赖：docx / xlsx 本质都是 zip，直接改写内部 XML 后重新打包，
//     复用 internal/docgen 的同一套路。**不引入 excelize，也不依赖 Node / Python** ——
//     skill 那套要 Node+adm-zip 与 Python+openpyxl，而 Docker 最终镜像里两个运行时都没有。
//   - 所有替换值强制 XML 转义（skill 的 10 个脚本里只有 2 个做了转义，
//     值里出现 & < " 就会把 docx 写坏）。
//   - 每份产出都校验「无残留 {{占位符}}」，宁可直接报错也不交付半成品
//     （skill 里 10 个脚本只有 2 个做了这个校验）。
//   - 汇总表页脚的联系人 / 电话 / 邮寄地址不再写死在模板里，由生成界面传入。
//   - 数据行按实际人数裁剪，不做「追加行」（skill 在数据行之后追加会覆盖页脚）。
package ligong

import (
	"strings"
	"time"
)

// ---- 证书类型 ----

const (
	CertSecurity = "security" // 安检（安检机构检验人员）
	CertEnv      = "env"      // 环保（环保检测机构人员）
)

// ---- 报考类别 ----
//
// 模板里的「报考类别」说明已按 2026-09-19 的确认改为只保留这三项。
const (
	ApplyNew   = "new"      // 新人员
	ApplyConti = "continue" // 继续教育
	ApplyAdd   = "add"      // 增项
)

// ApplyLabel 报考类别的中文名（写进汇总表「报考类别」列）。
var applyLabels = map[string]string{
	ApplyNew:   "新人员",
	ApplyConti: "继续教育",
	ApplyAdd:   "增项",
}

// ApplyLabel 返回报考类别的中文名，未知类别返回原值。
func ApplyLabel(key string) string {
	if s, ok := applyLabels[key]; ok {
		return s
	}
	return key
}

// ValidApplyType 判断报考类别是否受支持。
func ValidApplyType(key string) bool {
	_, ok := applyLabels[key]
	return ok
}

// ---- 汇总表里的字段键 ----
//
// 「哪一列填什么」全部走这张配置表，新增/调整模板时只改这里，
// 不用去翻散落各处的硬编码下标（这正是 skill 那版最容易出错的地方）。
const (
	FieldIndex     = "index"      // 序号
	FieldName      = "name"       // 姓名
	FieldGender    = "gender"     // 性别
	FieldEducation = "education"  // 学历
	FieldTitle     = "title"      // 职称
	FieldIDCard    = "id_card"    // 身份证号
	FieldPhone     = "phone"      // 联系电话
	FieldLineType  = "line_type"  // 检测线类别（仅环保表）
	FieldPosition  = "position"   // 检验岗位
	FieldApply     = "apply"      // 报考类别
	FieldDriveType = "drive_type" // 准驾车型（仅环保表）
	FieldRemark    = "remark"     // 备注
	FieldXuexin    = "xuexin"     // 学信网用户名（仅安检表，留空）
)

// Column 汇总表的一列：列字母 + 填入哪个字段。
type Column struct {
	Letter string
	Field  string
}

// CertSpec 一种证书类型的模板与填写规则。
type CertSpec struct {
	Key   string // 证书类型 key
	Label string // 中文名（用于文件名与界面）
	Excel string // 汇总表模板文件名
	Sheet string // 汇总表模板里要改写的工作表部件
	Docx  string // 安全责任书模板文件名（两种类型共用一份）
	// DataStartRow / DataEndRow 是模板里预留给数据行的区间（含端点）。
	// 实际人数 N 行用满 [DataStartRow, DataStartRow+N-1]，其余数据行整行裁掉，
	// 并把页脚与说明整体上移（不这么做的话，表格里会拖一长串空行）。
	DataStartRow int
	DataEndRow   int
	Columns      []Column
	// SafetyExam 安全责任书里「考试类型」的固定取值。
	SafetyExam string
	// SafetyExamDirector 当本批有人是「主任检验员」时改用这个取值（仅安检有意义）。
	SafetyExamDirector string
	// CompactExpiryDate 备注栏的「证书有效日期」是否写成 2026-11-09 这种紧凑格式。
	//
	// 环保表的「备注」列只有 9.125 个字符宽，写成「2026年11月09日」会把单元格撑破、
	// 打印时超出页面（用户 2026-09-20 反馈），故环保用 ISO 短横线；
	// 安检表同一列宽 15.775，沿用中文「年月日」写法不变。
	CompactExpiryDate bool
}

// securitySpec 安检：A序号 B姓名 C性别 D学历 E职称 F身份证号 G联系电话
// H检验岗位 I报考类别 J学信网用户名(留空) K备注
var securitySpec = &CertSpec{
	Key:          CertSecurity,
	Label:        "安检",
	Excel:        "安检报名汇总表.xlsx",
	Sheet:        "xl/worksheets/sheet1.xml",
	Docx:         "安全责任书.docx",
	DataStartRow: 5,
	DataEndRow:   23,
	Columns: []Column{
		{"A", FieldIndex},
		{"B", FieldName},
		{"C", FieldGender},
		{"D", FieldEducation},
		{"E", FieldTitle},
		{"F", FieldIDCard},
		{"G", FieldPhone},
		{"H", FieldPosition},
		{"I", FieldApply},
		{"J", FieldXuexin},
		{"K", FieldRemark},
	},
	SafetyExam:         "安检机构普通检验员",
	SafetyExamDirector: "安检机构主任检验员",
}

// envSpec 环保：A序号 B姓名 C性别 D学历 E职称 F身份证号 G联系电话
// H检测线类别 I检验岗位 J报考类别 K准驾车型 L备注
var envSpec = &CertSpec{
	Key:          CertEnv,
	Label:        "环保",
	Excel:        "环保报名汇总表.xlsx",
	Sheet:        "xl/worksheets/sheet1.xml",
	Docx:         "安全责任书.docx",
	DataStartRow: 4,
	DataEndRow:   14,
	Columns: []Column{
		{"A", FieldIndex},
		{"B", FieldName},
		{"C", FieldGender},
		{"D", FieldEducation},
		{"E", FieldTitle},
		{"F", FieldIDCard},
		{"G", FieldPhone},
		{"H", FieldLineType},
		{"I", FieldPosition},
		{"J", FieldApply},
		{"K", FieldDriveType},
		{"L", FieldRemark},
	},
	SafetyExam: "环保尾气排放检验员",
	// 备注列窄，证书有效期用 2026-11-09 写法，避免打印超出页面
	CompactExpiryDate: true,
}

var certSpecs = map[string]*CertSpec{
	CertSecurity: securitySpec,
	CertEnv:      envSpec,
}

// Spec 取证书类型的模板规则，第二个返回值表示是否支持。
func Spec(key string) (*CertSpec, bool) {
	s, ok := certSpecs[key]
	return s, ok
}

// MaxRows 模板能容纳的最大人数。
func (s *CertSpec) MaxRows() int {
	return s.DataEndRow - s.DataStartRow + 1
}

// ---- 材料清单 ----

// Material 一份需要收集的报名材料。
type Material struct {
	Key      string `json:"key"`      // ZIP 内的目录名
	Name     string `json:"name"`     // 展示名
	CertName string `json:"certName"` // 对应 personnel_certificates.cert_name；空表示系统生成
	// CanGenerate 表示「档案里没有可用的上传件时，可由系统生成」——目前只有劳动合同。
	// 生成时会先看档案里有没有仍在有效期内的上传件：有就直接打包上传的那份，不重复生成。
	CanGenerate bool `json:"canGenerate"`
	OnlyDriver  bool `json:"onlyDriver"` // 只有引车员才需要（驾驶证）
}

// CertNameContract personnel_certificates 里「劳动合同」的证件名。
// 报名时先按这个名字去查档案里有没有可复用的上传件。
const CertNameContract = "劳动合同"

// MaterialsFor 返回该报考类别需要收集的材料。
//
// 按 2026-09-19 确认的口径：
//   - 新人员 / 继续教育：报名汇总表 + 安全责任书 + 身份证 + 毕业证 + 劳动合同（+ 引车员驾驶证）
//     其中「报名汇总表 / 安全责任书」由本模块直接生成，故不在此列表内。
//   - 增项：只有报名汇总表 + 安全责任书，不附任何证件与劳动合同。
func MaterialsFor(apply string) []Material {
	switch apply {
	case ApplyNew, ApplyConti:
		return []Material{
			{Key: "身份证", Name: "身份证", CertName: "身份证"},
			{Key: "毕业证", Name: "毕业证书", CertName: "毕业证"},
			{Key: "劳动合同", Name: "劳动合同", CertName: CertNameContract, CanGenerate: true},
			{Key: "驾驶证", Name: "驾驶证", CertName: "驾驶证", OnlyDriver: true},
		}
	}
	// 增项：只要 汇总表 + 安全责任书。
	// 返回空切片而不是 nil —— 后者序列化成 JSON 的 null，前端拿到 null 还得额外判空。
	return []Material{}
}

// ---- 劳动合同可用性判定 ----

// 劳动合同的三种状态。
const (
	ContractUploaded = "uploaded" // 已上传且可用 → 直接打包，不再生成
	ContractExpired  = "expired"  // 已上传但已过期 → 需要问用户是否重新生成
	ContractMissing  = "missing"  // 未上传 → 需要问用户是否重新生成
)

// ContractState 判断某人的劳动合同能否直接使用。
//
// 规则（用户 2026-09-20 确认）：
//   - 没有上传附件 → missing
//   - 有附件、且填了到期日、且到期日早于今天 → expired
//   - 其余（含「有附件但没填到期日」）→ uploaded
//
// 关于「有附件但没填到期日」按可用处理：本系统里劳动合同的到期日就是
// personnel_certificates.expiry_date，没填只说明录入不全，不能据此判定它已失效；
// 若判成"需要重新生成"，反而会违背「有上传件就不要重复生成」的初衷。
func ContractState(hasFile bool, expiryDate string, today time.Time) string {
	if !hasFile {
		return ContractMissing
	}
	d := strings.TrimSpace(expiryDate)
	if d == "" {
		return ContractUploaded
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02", "2006.01.02"} {
		t, err := time.ParseInLocation(layout, d, time.Local)
		if err != nil {
			continue
		}
		if t.Before(today) {
			return ContractExpired
		}
		return ContractUploaded
	}
	// 到期日格式无法解析时按可用处理，并在界面上把原值显示出来供人工核对
	return ContractUploaded
}

// ContractStateLabel 状态的中文说明，用于生成界面的二次确认文案。
func ContractStateLabel(state string, expiryDate string) string {
	switch state {
	case ContractMissing:
		return "未上传劳动合同"
	case ContractExpired:
		if strings.TrimSpace(expiryDate) != "" {
			return "劳动合同已于 " + strings.TrimSpace(expiryDate) + " 到期"
		}
		return "劳动合同已到期"
	}
	return "劳动合同有效"
}

// IsDriver 判断检验岗位是否属于「引车员」。
// 模板说明里叫「受检车辆驾驶员」，skill 那版叫「引车员」，两种写法都要认。
// 只有引车员才需要驾驶证，也才需要在环保表里填「准驾车型」。
func IsDriver(position string) bool {
	p := strings.TrimSpace(position)
	return strings.Contains(p, "引车员") || strings.Contains(p, "驾驶员")
}

// CertExpiryField 返回证书类型在人员档案里对应的「证书到期时间」字段名。
func CertExpiryField(certType string) string {
	return certExpiryFields[certType]
}

// ArchiveCertExpiry 从人员档案里挑出与证书类型匹配的证书到期时间。
// 环保报名取「环保证到期时间」，安检报名取「安检证到期时间」。
//
// 注意是 **personnel 基本信息表** 的列（env_cert_date / security_cert_date），
// 不是 personnel_certificates 里上传附件的到期日 —— 两者分开维护，用户明确要求取前者。
func ArchiveCertExpiry(certType, securityDate, envDate string) string {
	if certType == CertEnv {
		return strings.TrimSpace(envDate)
	}
	return strings.TrimSpace(securityDate)
}

// ---- 界面下拉选项（供前端渲染，保证只有一处定义） ----

// 检验岗位选项。安检与环保口径不同，来自各自模板的填表说明。
var positionOptions = map[string][]string{
	CertSecurity: {"机动车主任检验员", "机动车检验员"},
	// 环保的口径按用户 2026-09-20 确认收敛为三项（原来的 5 项里有重复表达）
	CertEnv: {"引车员", "登录员、OBD查验员", "引车员、登录员、OBD查验员"},
}

// 证书有效期在人员档案里的取值字段（前端据此自动带出「证书有效期」）。
//
// 注意：取的是 **personnel 基本信息表**的「安检证/环保证 到期时间」列，
// 不是 personnel_certificates 里上传附件的到期日 —— 两者是分开维护的。
var certExpiryFields = map[string]string{
	CertSecurity: "securityCertDate",
	CertEnv:      "envCertDate",
}

var lineTypeOptions = []string{"汽车检测线", "摩托车检测线"}

// DefaultLineType 环保「检测线类别」的默认值。
//
// 绝大多数环保检测线都是汽车线，摩托车线是少数 —— 给个默认值省掉每人必选一次
// （用户 2026-09-20 要求）。需要摩托车线时在界面上改选即可。
const DefaultLineType = "汽车检测线"

var educationOptions = []string{"初中", "高中", "中专", "大专", "本科", "硕士", "博士"}

// Options 界面所需的全部下拉选项。
type Options struct {
	CertTypes  []Option              `json:"certTypes"`
	ApplyTypes []Option              `json:"applyTypes"`
	Positions  map[string][]string   `json:"positions"`
	LineTypes  []string              `json:"lineTypes"`
	Educations []string              `json:"educations"`
	Materials  map[string][]Material `json:"materials"`
	// CertExpiryFields：证书类型 -> 人员档案里「证书到期时间」的字段名，
	// 前端用它把「证书有效期」自动带出来（环保取环保证到期时间）
	CertExpiryFields map[string]string `json:"certExpiryFields"`
	// DefaultEducation 学历查不到时填报的默认值（用户 2026-09-20 确认：高中）
	DefaultEducation string `json:"defaultEducation"`
	// DefaultLineType 环保「检测线类别」的默认值（汽车检测线）
	DefaultLineType string `json:"defaultLineType"`
	// DefaultReceiver 收件人的默认值。由部署配置 LIGONG_DEFAULT_RECEIVER 决定，
	// 留空则界面不预填收件人（开源默认，避免把真实姓名写进代码/仓库）
	DefaultReceiver string `json:"defaultReceiver"`
}

// Option 通用「值 + 中文名」选项。
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// BuildOptions 组装界面下拉数据。defaultReceiver 来自部署配置（可空）。
func BuildOptions(defaultReceiver string) *Options {
	return &Options{
		CertTypes: []Option{
			{Value: CertSecurity, Label: "安检"},
			{Value: CertEnv, Label: "环保"},
		},
		ApplyTypes: []Option{
			{Value: ApplyNew, Label: "新人员"},
			{Value: ApplyConti, Label: "继续教育"},
			{Value: ApplyAdd, Label: "增项"},
		},
		Positions:  positionOptions,
		LineTypes:  lineTypeOptions,
		Educations: educationOptions,
		Materials: map[string][]Material{
			ApplyNew:   MaterialsFor(ApplyNew),
			ApplyConti: MaterialsFor(ApplyConti),
			ApplyAdd:   MaterialsFor(ApplyAdd),
		},
		CertExpiryFields: certExpiryFields,
		DefaultEducation: DefaultEducation,
		DefaultLineType:  DefaultLineType,
		DefaultReceiver:  strings.TrimSpace(defaultReceiver),
	}
}

// DefaultEducation 学历在人员档案里查不到时填报的值（用户 2026-09-20 确认）。
const DefaultEducation = "高中"
