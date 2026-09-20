package ligong

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zxjyyh/mds/internal/docgen"
)

// 汇总表（xlsx）改写规则。
//
// 为什么走「直接改 sheet XML」而不是引入 excelize：
//   - 项目已有的 docx 生成就是这个套路（internal/docgen），保持一致；
//   - 只为写几十个单元格引入一个重量级库不划算，且模板里的列宽/合并/样式
//     原样保留，不需要在 Go 里重建整张表。
//
// 写值时统一用 inlineStr（`t="inlineStr"` + `<is><t>`），不碰 sharedStrings.xml，
// 避免维护共享字符串索引表。
var (
	// 模板里带 {{占位符}} 的单元格（改造模板时已写成 inlineStr，见 backend/templates/ligong）
	inlineStrCellRe = regexp.MustCompile(`(?s)<c r="[A-Z]+\d+"[^>]*\bt="inlineStr"[^>]*>.*?</c>`)
	innerTextRe     = regexp.MustCompile(`(?s)(<t[^>]*>)(.*?)(</t>)`)
	tokenRe         = regexp.MustCompile(`\{\{[^{}]{1,40}\}\}`)

	rowRe    = regexp.MustCompile(`(?s)<row r="(\d+)"[^>]*(?:/>|>.*?</row>)`)
	rowAttr  = regexp.MustCompile(`(?s)<row r="\d+"([^>]*?)\s*/?>`)
	cellHead = regexp.MustCompile(`<c r="([A-Z]+)\d+"([^>]*?)(?:/>|>)`)

	// 同时匹配行号（`r="24"`）与单元格引用（`r="A24"`）
	refRe   = regexp.MustCompile(`r="([A-Z]*)(\d+)"`)
	single  = regexp.MustCompile(`^([A-Z]+)(\d+)$`)
	mergeRe = regexp.MustCompile(`<mergeCell ref="([^"]+)"/>`)
	dimRe   = regexp.MustCompile(`<dimension ref="[^"]*"/>`)
)

// RenderEnrollSheet 在汇总表模板上写入本批人员，产出成品 xlsx。
//
// values：表头/页脚的占位符取值，键是不带花括号的占位符名（如 "单位名称"）。
// rows：逐人数据，键为 Field* 常量。
func RenderEnrollSheet(template []byte, spec *CertSpec, values map[string]string, rows []map[string]string) ([]byte, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("至少需要一名报名人员")
	}
	if len(rows) > spec.MaxRows() {
		return nil, fmt.Errorf("%s报名汇总表最多容纳 %d 人，当前 %d 人", spec.Label, spec.MaxRows(), len(rows))
	}

	return docgen.RewriteXMLPart(template, spec.Sheet, func(xml string) (string, error) {
		out := substitutePlaceholders(xml, values)
		var err error
		out, err = writeDataRows(out, spec, rows)
		if err != nil {
			return "", err
		}
		// 兜底：占位符没被换掉说明模板与字段表对不上了，宁可直接失败
		if left := tokenRe.FindAllString(out, -1); len(left) > 0 {
			return "", fmt.Errorf("%s模板存在未替换的占位符：%s", spec.Label, strings.Join(left, "、"))
		}
		return out, nil
	})
}

// substitutePlaceholders 把 inlineStr 单元格里的 {{xx}} 换成实际值（逐 token 转义）。
func substitutePlaceholders(xml string, values map[string]string) string {
	return inlineStrCellRe.ReplaceAllStringFunc(xml, func(cell string) string {
		if !strings.Contains(cell, "{{") {
			return cell
		}
		return innerTextRe.ReplaceAllStringFunc(cell, func(seg string) string {
			m := innerTextRe.FindStringSubmatch(seg)
			if m == nil || !strings.Contains(m[2], "{{") {
				return seg
			}
			text := tokenRe.ReplaceAllStringFunc(m[2], func(tok string) string {
				name := strings.TrimSuffix(strings.TrimPrefix(tok, "{{"), "}}")
				v, ok := values[name]
				if !ok {
					return tok // 保留原样，交由调用方做残留校验并报错
				}
				return docgen.EscapeXML(v)
			})
			return m[1] + text + m[3]
		})
	})
}

// writeDataRows 用本批人员重建数据行，并把多余的数据行整行裁掉、页脚与说明整体上移。
//
// 沿用模板首行数据行的行高与各列样式，保证成品与手工填写的表观感一致。
func writeDataRows(xml string, spec *CertSpec, rows []map[string]string) (string, error) {
	baseRaw := findRow(xml, spec.DataStartRow)
	if baseRaw == "" {
		return "", fmt.Errorf("%s模板缺少第 %d 行数据行", spec.Label, spec.DataStartRow)
	}
	rowAttrs := ""
	if m := rowAttr.FindStringSubmatch(baseRaw); m != nil {
		rowAttrs = m[1]
	}

	// 列字母 -> 样式索引（取模板首行数据行上的 s 属性）
	styleOf := map[string]string{}
	for _, m := range cellHead.FindAllStringSubmatch(baseRaw, -1) {
		styleOf[m[1]] = styleAttrOf(m[2])
	}

	var newRows strings.Builder
	for i, r := range rows {
		n := spec.DataStartRow + i
		fmt.Fprintf(&newRows, `<row r="%d"%s>`, n, rowAttrs)
		for _, col := range spec.Columns {
			ref := col.Letter + strconv.Itoa(n)
			style := styleOf[col.Letter]
			if col.Field == FieldIndex {
				fmt.Fprintf(&newRows, `<c r="%s"%s><v>%d</v></c>`, ref, style, i+1)
				continue
			}
			v := strings.TrimSpace(r[col.Field])
			if v == "" {
				fmt.Fprintf(&newRows, `<c r="%s"%s/>`, ref, style)
				continue
			}
			fmt.Fprintf(&newRows, `<c r="%s"%s t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
				ref, style, docgen.EscapeXML(v))
		}
		newRows.WriteString(`</row>`)
	}

	// 组装：数据行区间被替换，其后的行整体上移
	delta := len(rows) - (spec.DataEndRow - spec.DataStartRow + 1)
	minRow := spec.DataEndRow + 1

	locs := rowRe.FindAllStringSubmatchIndex(xml, -1)
	if len(locs) == 0 {
		return "", fmt.Errorf("%s模板没有任何数据行", spec.Label)
	}
	var sb strings.Builder
	sb.WriteString(xml[:locs[0][0]])
	inserted := false
	lastRow := 0
	for _, loc := range locs {
		raw := xml[loc[0]:loc[1]]
		r, _ := strconv.Atoi(xml[loc[2]:loc[3]])
		switch {
		case r < spec.DataStartRow:
			sb.WriteString(raw)
			lastRow = r
		case r <= spec.DataEndRow:
			// 数据区整体丢弃；本批人员插在数据区第一行的位置上
			if !inserted {
				sb.WriteString(newRows.String())
				inserted = true
			}
			lastRow = spec.DataStartRow + len(rows) - 1
		default:
			sb.WriteString(shiftRefs(raw, minRow, delta))
			lastRow = r + delta
		}
	}
	if !inserted {
		return "", fmt.Errorf("%s模板的数据行区间 %d-%d 不存在", spec.Label, spec.DataStartRow, spec.DataEndRow)
	}
	sb.WriteString(xml[locs[len(locs)-1][1]:])
	out := sb.String()

	// 合并区与工作表尺寸随行数一起上移
	if delta != 0 {
		out = mergeRe.ReplaceAllStringFunc(out, func(s string) string {
			m := mergeRe.FindStringSubmatch(s)
			parts := strings.Split(m[1], ":")
			for i, p := range parts {
				parts[i] = shiftSingle(p, minRow, delta)
			}
			return `<mergeCell ref="` + strings.Join(parts, ":") + `"/>`
		})
	}
	// dimension 必须指向最后一行（说明行也在数据区之后，同样要上移），
	// 写成 DataEndRow+delta 会偏小 —— Excel 靠它决定打开时的滚动范围。
	lastCol := spec.Columns[len(spec.Columns)-1].Letter
	out = dimRe.ReplaceAllString(out, `<dimension ref="A1:`+lastCol+strconv.Itoa(lastRow)+`"/>`)
	return out, nil
}

// findRow 取出指定行号的原始 XML 片段（未找到返回空串）。
// 用 submatch 下标定位，避免用正则去「猜」行号的位置。
func findRow(xml string, row int) string {
	for _, loc := range rowRe.FindAllStringSubmatchIndex(xml, -1) {
		if r, err := strconv.Atoi(xml[loc[2]:loc[3]]); err == nil && r == row {
			return xml[loc[0]:loc[1]]
		}
	}
	return ""
}

// styleAttrOf 从单元格属性串里取出 s="N"，返回可直接拼接的 ` s="N"`（无样式时为空串）。
func styleAttrOf(attrs string) string {
	m := regexp.MustCompile(`\bs="(\d+)"`).FindStringSubmatch(attrs)
	if m == nil {
		return ""
	}
	return ` s="` + m[1] + `"`
}

// shiftRefs 把行片段里所有 ≥ minRow 的行号加上 delta（含行号本身与单元格引用）。
func shiftRefs(raw string, minRow, delta int) string {
	if delta == 0 {
		return raw
	}
	return refRe.ReplaceAllStringFunc(raw, func(s string) string {
		m := refRe.FindStringSubmatch(s)
		n, err := strconv.Atoi(m[2])
		if err != nil || n < minRow {
			return s
		}
		return `r="` + m[1] + strconv.Itoa(n+delta) + `"`
	})
}

// shiftSingle 处理单个引用，如 "A24"、"24"。
func shiftSingle(p string, minRow, delta int) string {
	m := single.FindStringSubmatch(p)
	if m == nil {
		return p
	}
	n, err := strconv.Atoi(m[2])
	if err != nil || n < minRow {
		return p
	}
	return m[1] + strconv.Itoa(n+delta)
}
