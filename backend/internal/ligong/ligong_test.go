package ligong

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// outDir 测试产物落盘目录，供外部脚本（Python）用独立解析器复核成品。
// 放系统临时目录，避免污染项目根目录。
var outDir = filepath.Join(os.TempDir(), "mds-ligong-test-out")

func sampleIn(cert string) *BuildInput {
	date := time.Date(2026, 9, 19, 0, 0, 0, 0, time.Local)
	persons := []Person{
		{
			ID: 1, Name: "张小明", Gender: "女", IDCard: "110105199003074821", Phone: "13800138001",
			Education: "大专", Title: "工程师", Position: "机动车检验员",
			// 档案里已有仍在有效期内的劳动合同上传件 → 直接复用，不重新生成
			UploadedContract: []Attachment{{Name: "张小明-劳动合同扫描件.pdf", Data: []byte("CONTRACT-SCAN-1")}},
			Attachments: map[string][]Attachment{
				"身份证": {{Name: "身份证正面.jpg", Data: []byte("ID-CARD-1")}},
				"毕业证": {{Name: "毕业证书.jpg", Data: []byte("DIPLOMA-1")}},
			},
		},
		{
			ID: 2, Name: "李建国", Gender: "男", IDCard: "110105198806152233", Phone: "13800138000",
			Education: "高中", Position: "机动车主任检验员",
			// 档案里没有可用劳动合同 + 用户在二次确认里选了「是」 → 由系统生成
			RegenerateContract: true,
			Attachments: map[string][]Attachment{
				"身份证": {{Name: "身份证.jpg", Data: []byte("ID-CARD-2")}},
			},
		},
	}
	if cert == CertEnv {
		persons[0].LineType = "汽车检测线"
		persons[1].LineType = "摩托车检测线"
		persons[1].Position = "引车员、登录员"
		persons[1].DriveType = "C1"
		persons[1].Attachments["驾驶证"] = []Attachment{{Name: "驾驶证.jpg", Data: []byte("DRIVER-2")}}
	} else {
		persons[1].Position = "机动车主任检验员"
	}
	return &BuildInput{
		Spec:          mustSpec(cert),
		ApplyType:     ApplyNew,
		Period:        "8",
		EnrollDate:    date,
		CompanyName:   "示例市示例机动车检测有限公司", // 下面再追加 & 专测转义
		Receiver:      "李建国",
		ReceiverPhone: "13800138000",
		MailAddress:   "福建省示例市示例区示例路 1 号",
		Persons:       persons,
		Contract: func(p Person) ([]byte, string, error) {
			return []byte("FAKE-CONTRACT-" + p.Name), p.Name + "-聘用劳动合同-20260919.docx", nil
		},
	}
}

func mustSpec(cert string) *CertSpec {
	s, ok := Spec(cert)
	if !ok {
		panic("bad cert")
	}
	return s
}

// TestBuildEnroll 跑完整生成流程，并把成品落盘供独立解析器复核。
func TestBuildEnroll(t *testing.T) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, cert := range []string{CertSecurity, CertEnv} {
		in := sampleIn(cert)
		// 专门用带 & 的公司名验证 XML 转义（值里有 & 会把 docx/xlsx 写坏）
		in.CompanyName = "示例A&B检测站"

		res, err := Build(in)
		if err != nil {
			t.Fatalf("[%s] 生成失败: %v", cert, err)
		}
		if len(res.Missing) == 0 {
			t.Errorf("[%s] 期望有缺件（张小明缺驾驶证不影响；李建国缺毕业证应记录）", cert)
		}
		t.Logf("[%s] 缺件: %v", cert, res.Missing)

		zr, err := zip.NewReader(bytes.NewReader(res.Data), int64(len(res.Data)))
		if err != nil {
			t.Fatalf("[%s] 产物不是合法 zip: %v", cert, err)
		}
		t.Logf("[%s] ZIP: %s", cert, res.FileName)
		for _, f := range zr.File {
			rc, _ := f.Open()
			var sb bytes.Buffer
			_, _ = sb.ReadFrom(rc)
			rc.Close()
			data := sb.Bytes()
			t.Logf("    %-70s %6d B", f.Name, len(data))

			// 落盘：docx/xlsx 供 Python 独立解析
			if strings.HasSuffix(f.Name, ".xlsx") || strings.HasSuffix(f.Name, ".docx") {
				flat := strings.NewReplacer("/", "_", "\\", "_").Replace(f.Name)
				if err := os.WriteFile(filepath.Join(outDir, cert+"_"+flat), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

// TestSafetyLetterPerExam 一批里同时有主任检验员与普通检验员时，安全责任书应各出一份。
func TestSafetyLetterPerExam(t *testing.T) {
	in := sampleIn(CertSecurity)
	in.Persons[0].Position = "机动车检验员"   // 普通
	in.Persons[1].Position = "机动车主任检验员" // 主任
	res, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(res.Data), int64(len(res.Data)))
	var letters []string
	for _, f := range zr.File {
		if strings.Contains(f.Name, "安全责任书") {
			letters = append(letters, filepath.Base(f.Name))
		}
	}
	if len(letters) != 2 {
		t.Fatalf("期望 2 份安全责任书，实际 %d: %v", len(letters), letters)
	}
	joined := strings.Join(letters, "|")
	if !strings.Contains(joined, "普通检验员") || !strings.Contains(joined, "主任检验员") {
		t.Errorf("两份责任书未按培训班级区分: %v", letters)
	}
	t.Logf("两份责任书: %v", letters)

	// 只有一种岗位时只出一份
	in.Persons[1].Position = "机动车检验员"
	res2, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	zr2, _ := zip.NewReader(bytes.NewReader(res2.Data), int64(len(res2.Data)))
	n := 0
	for _, f := range zr2.File {
		if strings.Contains(f.Name, "安全责任书") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("期望 1 份安全责任书，实际 %d", n)
	}
}

// TestContractReuse 劳动合同：有可用上传件就直接打包那份，没有再按用户选择决定是否生成。
func TestContractReuse(t *testing.T) {
	gen := func(p Person) ([]byte, string, error) {
		return []byte("GENERATED-" + p.Name), p.Name + "-聘用劳动合同-20260919.docx", nil
	}
	base := sampleIn(CertSecurity)

	// 场景 A：档案有可用件 → 复用，且不生成
	in := sampleIn(CertSecurity)
	in.Persons = base.Persons[:1]           // 张小明带 UploadedContract
	in.Persons[0].RegenerateContract = true // 即使请求了"重新生成"，有可用上传件也必须复用
	in.Contract = gen
	res, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	names := zipNames(t, res.Data)
	if !containsSub(names, "张小明-劳动合同扫描件.pdf") {
		t.Errorf("应打包档案里的劳动合同上传件，实际: %v", names)
	}
	if containsSub(names, "聘用劳动合同") {
		t.Errorf("档案有可用合同时不应重新生成，实际: %v", names)
	}
	if len(res.ContractFromArchive) != 1 || len(res.ContractGenerated) != 0 {
		t.Errorf("复用统计不对: archive=%v generated=%v", res.ContractFromArchive, res.ContractGenerated)
	}

	// 场景 B：无上传件 + 用户选「是」 → 生成
	in2 := sampleIn(CertSecurity)
	in2.Persons = base.Persons[1:]
	in2.Persons[0].RegenerateContract = true
	in2.Contract = gen
	res2, err := Build(in2)
	if err != nil {
		t.Fatal(err)
	}
	names2 := zipNames(t, res2.Data)
	if !containsSub(names2, "聘用劳动合同") {
		t.Errorf("用户选「是」时应生成劳动合同，实际: %v", names2)
	}
	if len(res2.ContractGenerated) != 1 {
		t.Errorf("生成统计不对: %v", res2.ContractGenerated)
	}

	// 场景 C：无上传件 + 用户选「否」 → 不生成，且不计入缺件
	in3 := sampleIn(CertSecurity)
	in3.Persons = base.Persons[1:]
	in3.Persons[0].RegenerateContract = false
	in3.Contract = gen
	res3, err := Build(in3)
	if err != nil {
		t.Fatal(err)
	}
	names3 := zipNames(t, res3.Data)
	if containsSub(names3, "劳动合同") {
		t.Errorf("用户选「否」时不应出劳动合同，实际: %v", names3)
	}
	if strings.Contains(strings.Join(res3.Missing, "|"), "劳动合同") {
		t.Errorf("用户主动选择不生成，不应计入缺件: %v", res3.Missing)
	}
	if len(res3.ContractSelfProvided) != 1 {
		t.Errorf("应记录为「自行提供」: %v", res3.ContractSelfProvided)
	}
	t.Logf("自行提供: %v", res3.ContractSelfProvided)
}

// TestContractState 劳动合同可用性判定。
func TestContractState(t *testing.T) {
	today := time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)
	cases := []struct {
		hasFile bool
		expiry  string
		want    string
	}{
		{false, "", ContractMissing},
		{false, "2030-01-01", ContractMissing},
		{true, "", ContractUploaded},           // 没填到期日 → 视为可用
		{true, "2030-01-01", ContractUploaded}, // 未到期
		{true, "2026-09-20", ContractUploaded}, // 当天仍有效
		{true, "2026-09-19", ContractExpired},  // 已过期
		{true, "不是日期", ContractUploaded},       // 解析不了 → 视为可用，由人工核对
	}
	for _, c := range cases {
		if got := ContractState(c.hasFile, c.expiry, today); got != c.want {
			t.Errorf("ContractState(file=%v, expiry=%q) = %s, 期望 %s", c.hasFile, c.expiry, got, c.want)
		}
	}
}

func zipNames(t *testing.T, data []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	return out
}

func containsSub(names []string, sub string) bool {
	for _, n := range names {
		if strings.Contains(n, sub) {
			return true
		}
	}
	return false
}

// TestZipLayout 汇总表与安全责任书直接放压缩包根目录；证件类材料仍按类型分目录。
func TestZipLayout(t *testing.T) {
	in := sampleIn(CertSecurity)
	in.CompanyName = "示例A&B检测站"
	res, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	names := zipNames(t, res.Data)
	for _, n := range names {
		if strings.HasSuffix(n, "/") {
			continue
		}
		if strings.Contains(n, "/") && (strings.Contains(n, "报名汇总表") || strings.Contains(n, "安全责任书")) {
			t.Errorf("汇总表/责任书应直接放根目录，实际: %s", n)
		}
	}
	for _, dir := range []string{"身份证/", "毕业证/", "劳动合同/"} {
		if !containsSub(names, dir) {
			t.Errorf("缺少证件目录 %s，实际: %v", dir, names)
		}
	}
	t.Logf("ZIP 结构: %v", names)
}

// TestExpiryDateFormat 备注栏的「证书有效日期」写法：
// 环保表列窄用 2026-11-09，安检表沿用中文年月日。
func TestExpiryDateFormat(t *testing.T) {
	for _, tc := range []struct{ cert, want, notWant string }{
		{CertEnv, "2029-03-31", "2029年03月31日"},
		{CertSecurity, "2029年03月31日", "2029-03-31"},
	} {
		in := sampleIn(tc.cert)
		in.ApplyType = ApplyConti
		in.Persons = in.Persons[:1]
		in.Persons[0].CertExpiry = "2029-03-31"
		res, err := Build(in)
		if err != nil {
			t.Fatal(err)
		}
		// 先在 ZIP 里取出 xlsx，再从 xlsx 里取出 sheet1.xml —— 两层，别读错层级
		zr, err := zip.NewReader(bytes.NewReader(res.Data), int64(len(res.Data)))
		if err != nil {
			t.Fatal(err)
		}
		var sheet string
		for _, f := range zr.File {
			if !strings.HasSuffix(f.Name, ".xlsx") {
				continue
			}
			xlsx, err := readZipEntry(res.Data, f.Name)
			if err != nil {
				t.Fatal(err)
			}
			if sheet, err = readZipEntry([]byte(xlsx), "xl/worksheets/sheet1.xml"); err != nil {
				t.Fatal(err)
			}
			break
		}
		if sheet == "" {
			t.Fatal("没取到汇总表的 sheet1.xml")
		}
		if !strings.Contains(sheet, tc.want) {
			t.Errorf("[%s] 备注栏应为 %q，实际汇总表里没有该值", tc.cert, tc.want)
		}
		if strings.Contains(sheet, tc.notWant) {
			t.Errorf("[%s] 备注栏不应出现 %q", tc.cert, tc.notWant)
		}
	}
}

// readSheet 从成品 ZIP 里取出报名汇总表的 sheet1.xml。
// 注意是两层：先在 ZIP 里取 xlsx，再从 xlsx 里取 sheet1.xml。
func readSheet(t *testing.T, zipData []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, ".xlsx") {
			continue
		}
		xlsx, err := readZipEntry(zipData, f.Name)
		if err != nil {
			t.Fatal(err)
		}
		sheet, err := readZipEntry([]byte(xlsx), "xl/worksheets/sheet1.xml")
		if err != nil {
			t.Fatal(err)
		}
		return sheet
	}
	t.Fatal("成品里没有 xlsx")
	return ""
}

// TestMixedApplyTypes 同一证书类型下混报不同报考类别（用户 2026-09-20 需求）：
// 汇总表逐行写各自类别、材料按各自类别收集、增项人员不附证件与劳动合同。
func TestMixedApplyTypes(t *testing.T) {
	in := sampleIn(CertSecurity)
	// 批次级类别留空：界面已去掉「批量设置」入口，只有逐人设了类别，也必须能正常生成
	in.ApplyType = ""
	in.Persons[0].ApplyType = ApplyNew
	in.Persons[0].CertExpiry = "" // 新人员不写证书有效期
	in.Persons[1].ApplyType = ApplyConti
	in.Persons[1].CertExpiry = "2029-03-31"
	in.Persons = append(in.Persons, Person{
		ID: 3, Name: "增项人", Gender: "男", IDCard: "110105199001018899", Phone: "13800000000",
		Education: "高中", Position: "机动车检验员", ApplyType: ApplyAdd,
	})

	res, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	names := zipNames(t, res.Data)

	// 文件名应体现多类别
	if !strings.Contains(res.FileName, "新人员+继续教育+增项") {
		t.Errorf("混报时文件名应体现全部类别，实际: %s", res.FileName)
	}
	// 增项的人不应有任何证件/劳动合同
	for _, n := range names {
		if strings.Contains(n, "增项人") {
			t.Errorf("增项人员不应附证件或劳动合同，实际出现: %s", n)
		}
	}
	// 非增项的两人证件应照常收集
	if !containsSub(names, "身份证/") || !containsSub(names, "毕业证/") {
		t.Errorf("非增项人员应附证件，实际: %v", names)
	}

	sheet := readSheet(t, res.Data)
	for _, want := range []string{"新人员", "继续教育", "增项"} {
		if !strings.Contains(sheet, want) {
			t.Errorf("汇总表应含报考类别 %q", want)
		}
	}
	// 证书有效期只有「继续教育」那一行才写
	if n := strings.Count(sheet, "2029年03月31日"); n != 1 {
		t.Errorf("证书有效期应只出现在继续教育那一行，实际出现 %d 次", n)
	}

	readme := ""
	if zr, err := zip.NewReader(bytes.NewReader(res.Data), int64(len(res.Data))); err == nil {
		for _, f := range zr.File {
			if f.Name == "00_报名说明.txt" {
				readme, _ = readZipEntry(res.Data, f.Name)
			}
		}
	}
	if !strings.Contains(readme, "本批同时含多种报考类别") {
		t.Errorf("说明文件应提示混报，实际:\n%s", readme)
	}
	t.Logf("混报 ZIP: %v", names)
}

// TestApplyAdd 增项：只出汇总表 + 安全责任书，不带任何证件。
func TestApplyAdd(t *testing.T) {
	in := sampleIn(CertSecurity)
	in.ApplyType = ApplyAdd
	in.CompanyName = "示例A&B检测站"
	res, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(res.Data), int64(len(res.Data)))
	names := []string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	joined := strings.Join(names, "|")
	for _, bad := range []string{"身份证", "毕业证", "劳动合同", "驾驶证", "缺件清单"} {
		if strings.Contains(joined, bad) {
			t.Errorf("增项不应包含 %s，实际: %v", bad, names)
		}
	}
	if !strings.Contains(joined, "报名汇总表") || !strings.Contains(joined, "安全责任书") {
		t.Errorf("增项应包含汇总表与责任书，实际: %v", names)
	}
}

// TestRowTrim 人数不足模板行数时，多余数据行应被裁掉、页脚上移。
func TestRowTrim(t *testing.T) {
	for _, tc := range []struct {
		cert          string
		wantFooterRow int
	}{
		{CertSecurity, 8}, // 数据行 5-23，2 人 → 5,6；页脚由 24 → 8
		{CertEnv, 6},      // 数据行 4-14，2 人 → 4,5；页脚由 15 → 6
	} {
		in := sampleIn(tc.cert)
		in.Persons = in.Persons[:2]
		res, err := Build(in)
		if err != nil {
			t.Fatal(err)
		}
		zr, _ := zip.NewReader(bytes.NewReader(res.Data), int64(len(res.Data)))
		for _, f := range zr.File {
			if !strings.HasSuffix(f.Name, ".xlsx") {
				continue
			}
			rc, _ := f.Open()
			var sb bytes.Buffer
			_, _ = sb.ReadFrom(rc)
			rc.Close()
			sheet, err := readZipEntry(sb.Bytes(), "xl/worksheets/sheet1.xml")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(sheet, "{{") {
				t.Errorf("[%s] 汇总表仍有未替换占位符", tc.cert)
			}
			want := `r="` + itoa(tc.wantFooterRow) + `"`
			if !strings.Contains(sheet, want) {
				t.Errorf("[%s] 期望页脚行移到第 %d 行，未找到 %s", tc.cert, tc.wantFooterRow, want)
			}
			// 被裁掉的行不应再出现
			for _, gone := range []int{20, 21, 22, 23} {
				if strings.Contains(sheet, `<row r="`+itoa(gone)+`"`) {
					t.Errorf("[%s] 第 %d 行应已被裁掉", tc.cert, gone)
				}
			}
			break
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func readZipEntry(data []byte, name string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		var sb bytes.Buffer
		_, err = sb.ReadFrom(rc)
		return sb.String(), err
	}
	return "", os.ErrNotExist
}
