package docgen

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestRenderContract(t *testing.T) {
	repl := map[string]string{
		"{{甲方名称}}":   "示例市示例机动车检测有限公司",
		"{{甲方地址}}":   "福建省示例市示例区示例路 1 号",
		"{{乙方姓名}}":   "张三",
		"{{乙方性别}}":   "男",
		"{{乙方身份证号}}": "110105199001010011",
		"{{乙方住址}}":   "示例市示例镇解放路 1 号",
		"{{签订日期}}":   "2026年09月06日",
		"{{合同起始日期}}": "2026年09月06日",
		"{{合同结束日期}}": "2029年09月05日",
	}
	data, err := RenderContract(repl, "")
	if err != nil {
		t.Fatalf("RenderContract: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty output")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("output not a valid zip: %v", err)
	}
	var xml string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := new(bytes.Buffer)
			buf.ReadFrom(rc)
			rc.Close()
			xml = buf.String()
		}
	}
	if xml == "" {
		t.Fatal("document.xml missing")
	}
	for _, want := range []string{"张三", "示例市示例机动车检测有限公司", "2026年09月06日", "2029年09月05日", "110105199001010011"} {
		if !strings.Contains(xml, want) {
			t.Errorf("document.xml missing %q", want)
		}
	}
	if strings.Contains(xml, "{{") || strings.Contains(xml, "}}") {
		t.Errorf("leftover template token: %s", xml)
	}
	// 户口所在地应已删除
	if strings.Contains(xml, "户口所在地") {
		t.Error("template should not contain 户口所在地")
	}
}
