// Package docgen 实现各类业务文档的生成：基于「带 {{占位符}} 的 docx 模板」，
// 在 document.xml 字符串层做占位符替换后重新打包，产出成品 docx。
// 模板优先读取运行时 externalDir（如 ./templates，便于免编译改模板），
// 目录不存在或文件缺失时回退到 go:embed 内嵌副本（保证单文件 exe / Docker 可用）。
package docgen

import (
	"archive/zip"
	"bytes"
	"embed"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed templates/*.docx
var templatesFS embed.FS

// ContractTemplateName 劳动合同模板文件名。
// 修改模板的约定：更新 backend/templates/ 下的同名文件后，
// 同步复制到本包 templates/ 目录（外部目录存在时运行时优先采用外部版本，
// 因此日常 dev 只需改 backend/templates；内嵌副本仅作发布兜底）。
const ContractTemplateName = "聘用劳动合同模板.docx"

// escapeXML 转义 XML 文本节点中的特殊字符，防止值中的 & < > 破坏文档结构。
func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// EscapeXML 导出 escapeXML 供其它文档生成器复用。
// 任何把用户数据写进 XML 的地方都必须走这里 —— 漏一处就会生成打不开的文档。
func EscapeXML(s string) string { return escapeXML(s) }

// RewriteXMLPart 读出 zip（docx/xlsx 本质都是 zip）中指定的 XML 部件，
// 交给 fn 处理后再与原样的其余部件一起重新打包。
// 其它部件原样保留，因此模板的样式、图片、字体设置都不会丢。
func RewriteXMLPart(zipData []byte, part string, fn func(string) (string, error)) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		// 复制 FileHeader 但清零 CRC/大小，写入内容变化后由 Writer 重算（沿用旧值会 checksum error）
		h := f.FileHeader
		h.CRC32 = 0
		h.CompressedSize = 0
		h.CompressedSize64 = 0
		h.UncompressedSize = 0
		h.UncompressedSize64 = 0
		w, err := zw.CreateHeader(&h)
		if err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		if f.Name == part {
			var sb strings.Builder
			if _, err := io.Copy(&sb, rc); err != nil {
				rc.Close()
				return nil, err
			}
			out, err := fn(sb.String())
			if err != nil {
				rc.Close()
				return nil, err
			}
			if _, err := w.Write([]byte(out)); err != nil {
				rc.Close()
				return nil, err
			}
		} else {
			if _, err := io.Copy(w, rc); err != nil {
				rc.Close()
				return nil, err
			}
		}
		rc.Close()
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// readXMLPart 只读地取出 zip 中某个文本部件的内容。
func readXMLPart(zipData []byte, part string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if f.Name != part {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		var sb strings.Builder
		if _, err := io.Copy(&sb, rc); err != nil {
			return "", err
		}
		return sb.String(), nil
	}
	return "", os.ErrNotExist
}

// render 读取模板字节流，将 document.xml 中所有 token（如 {{乙方姓名}}）替换为对应值，
// 其它部件原样保留，返回重新打包后的 docx 字节。
func render(templateData []byte, repl map[string]string) ([]byte, error) {
	return RewriteXMLPart(templateData, "word/document.xml", func(xml string) (string, error) {
		for k, v := range repl {
			xml = strings.ReplaceAll(xml, k, escapeXML(v))
		}
		return xml, nil
	})
}

// RenderDocx 用模板字节流渲染 docx（导出给其它模板目录的调用方复用）。
func RenderDocx(templateData []byte, repl map[string]string) ([]byte, error) {
	return render(templateData, repl)
}

// LoadDocxTemplate 按文件名加载 docx 模板：externalDir 下同名文件优先，否则回退内嵌副本。
// 注意内嵌副本只包含本包的模板目录，ligong 等其它模板请自行加载后调 RenderDocx。
func LoadDocxTemplate(templateName, externalDir string) ([]byte, error) {
	return loadTemplate(templateName, externalDir)
}

// placeholderRe 匹配残留的 {{占位符}}。
var placeholderRe = regexp.MustCompile(`\{\{[^{}]{1,40}\}\}`)

// UnreplacedTokens 扫描 docx 的 document.xml 中残留的 {{占位符}}（去重，最多 20 个）。
//
// 为什么必须做这一步：占位符替换是**字符串替换**，模板里少写一个、或字段表里名字对不上，
// 替换会静默跳过，成品文档里就留着一个 {{xxx}} 直接发给客户。宁可生成失败也不能这样交付。
func UnreplacedTokens(docxData []byte) ([]string, error) {
	xml, err := readXMLPart(docxData, "word/document.xml")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, m := range placeholderRe.FindAllString(xml, -1) {
		if seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
		if len(out) >= 20 {
			break
		}
	}
	return out, nil
}

// loadTemplate 读取模板：externalDir 下同名文件优先，否则用内嵌副本。
func loadTemplate(templateName, externalDir string) ([]byte, error) {
	if externalDir != "" {
		if b, err := os.ReadFile(filepath.Join(externalDir, templateName)); err == nil {
			return b, nil
		}
	}
	return templatesFS.ReadFile("templates/" + templateName)
}

// RenderContract 生成聘用劳动合同 docx。
// repl 为占位符替换表（不含转义，内部处理）；externalDir 传运行时模板目录（可传 ""）。
func RenderContract(repl map[string]string, externalDir string) ([]byte, error) {
	raw, err := loadTemplate(ContractTemplateName, externalDir)
	if err != nil {
		return nil, err
	}
	return render(raw, repl)
}
