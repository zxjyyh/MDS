package ligong

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zxjyyh/mds/internal/docgen"
)

//go:embed templates/*
var templatesFS embed.FS

// TemplateDir 运行时可覆盖的模板目录（相对后端工作目录）。
//
// 存在同名文件时优先用它，便于改模板不用重新编译；目录不存在时回退 go:embed 内嵌副本。
// （Docker 镜像只复制编译产物，不含 templates/ 目录，所以内嵌副本是发布形态的兜底，
// 与 internal/docgen 对劳动合同模板的处理完全一致。）
const TemplateDir = "templates/ligong"

// loadTemplate 读模板：外部目录优先，内嵌副本兜底。
func loadTemplate(name string) ([]byte, error) {
	if b, err := os.ReadFile(filepath.Join(TemplateDir, name)); err == nil {
		return b, nil
	}
	return templatesFS.ReadFile("templates/" + name)
}

// LoadEnrollTemplate 读取报名汇总表模板字节。
func LoadEnrollTemplate(spec *CertSpec) ([]byte, error) {
	b, err := loadTemplate(spec.Excel)
	if err != nil {
		return nil, fmt.Errorf("汇总表模板 %s 缺失：%w", spec.Excel, err)
	}
	return b, nil
}

// RenderSafetyLetter 生成安全责任书 docx。
//
// name：报名人（多人报名时固定为「本次报名全体人员」，与 skill 版口径一致）
// period：期数（纯数字）；exam：考试类型文本；company：单位名称；date：中文日期
func RenderSafetyLetter(spec *CertSpec, name, period, exam, company, date string) ([]byte, error) {
	tpl, err := loadTemplate(spec.Docx)
	if err != nil {
		return nil, fmt.Errorf("安全责任书模板缺失：%w", err)
	}
	out, err := docgen.RenderDocx(tpl, map[string]string{
		"{{报名人}}":  name,
		"{{期数}}":   period,
		"{{考试类型}}": exam,
		"{{单位名称}}": company,
		"{{日期}}":   date,
	})
	if err != nil {
		return nil, err
	}
	left, err := docgen.UnreplacedTokens(out)
	if err != nil {
		return nil, err
	}
	if len(left) > 0 {
		return nil, fmt.Errorf("安全责任书存在未替换的占位符：%s", strings.Join(left, "、"))
	}
	return out, nil
}
