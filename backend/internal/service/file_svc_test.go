package service

import (
	"net/http"
	"strings"
	"testing"
)

// TestDangerousExtensionsRejected 守住存储型 XSS 的第一道闸门。
// 这些扩展名一旦被放进白名单，攻击者上传的文件在浏览器里会被当作页面执行。
func TestDangerousExtensionsRejected(t *testing.T) {
	dangerous := []string{
		".svg", ".html", ".htm", ".xhtml", ".xml", ".js", ".mjs",
		".sh", ".bat", ".exe", ".php", ".jsp", ".swf", ".htaccess",
	}
	for _, ext := range dangerous {
		if _, ok := allowedUploadTypes[ext]; ok {
			t.Errorf("危险扩展名 %s 出现在上传白名单里", ext)
		}
		if IsInlineRenderable(ext) {
			t.Errorf("危险扩展名 %s 被允许 inline 渲染", ext)
		}
	}
}

func TestCanonicalMime(t *testing.T) {
	cases := map[string]string{
		".jpg":     "image/jpeg",
		".JPG":     "image/jpeg", // 大小写不敏感
		".pdf":     "application/pdf",
		".docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":     "text/plain; charset=utf-8",
		".svg":     "application/octet-stream", // 不在白名单 → 兜底为下载
		".unknown": "application/octet-stream",
		"":         "application/octet-stream",
	}
	for ext, want := range cases {
		if got := CanonicalMime(ext); got != want {
			t.Errorf("CanonicalMime(%q) = %q, 期望 %q", ext, got, want)
		}
	}
}

// TestInlineRenderableIsSubsetOfAllowed 保证「可 inline 渲染」永远是「允许上传」的子集，
// 防止有人往 inlineRenderable 里加了没经过上传校验的类型。
func TestInlineRenderableIsSubsetOfAllowed(t *testing.T) {
	for ext, ok := range inlineRenderable {
		if !ok {
			continue
		}
		if _, allowed := allowedUploadTypes[ext]; !allowed {
			t.Errorf("%s 允许 inline 渲染，却不在上传白名单里", ext)
		}
	}
	// Office 文档不应 inline（浏览器不渲染，但避免任何解析歧义）
	for _, ext := range []string{".doc", ".docx", ".xls", ".xlsx", ".txt"} {
		if IsInlineRenderable(ext) {
			t.Errorf("%s 不应被 inline 渲染", ext)
		}
	}
}

// TestSanitizeFilename 覆盖响应头注入与目录穿越。
// 该值会直接出现在 Content-Disposition 里。
func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"report.pdf", "report.pdf"},
		{"../../etc/passwd", "passwd"},
		{`..\..\windows\system32\cmd.exe`, "cmd.exe"},
		{"/absolute/path/a.png", "a.png"},
		{"a\r\nX-Injected: 1.png", "aX-Injected: 1.png"}, // CRLF 被剥离
		{"quo\"te'.txt", "quote.txt"},
		{"..", "file"},
		{"...", "file"},
		{"", "file"},
		{"   ", "file"},
		{".hidden", "hidden"},
		{"中文 文件名.docx", "中文 文件名.docx"},
	}
	for _, c := range cases {
		got := SanitizeFilename(c.in)
		if got != c.want {
			t.Errorf("SanitizeFilename(%q) = %q, 期望 %q", c.in, got, c.want)
		}
		if strings.ContainsAny(got, "\r\n/\\\"'") {
			t.Errorf("SanitizeFilename(%q) 结果仍含危险字符: %q", c.in, got)
		}
	}
}

func TestSanitizeFilenameTruncates(t *testing.T) {
	long := strings.Repeat("测", 300) + ".pdf"
	got := SanitizeFilename(long)
	if n := len([]rune(got)); n != 180 {
		t.Fatalf("超长文件名未截断到 180 个字符，实际 %d", n)
	}
}

// TestSniffMatches 验证内容嗅探能拦住「把 HTML 改名成 .jpg 上传」。
func TestSniffMatches(t *testing.T) {
	// 用真实字节让 http.DetectContentType 给出结论，避免手写魔数出错
	pngHead := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 32))
	htmlHead := []byte("<!DOCTYPE html><html><body><script>alert(1)</script>")
	svgHead := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)

	if d := http.DetectContentType(pngHead); !sniffMatches(".png", d) {
		t.Errorf("合法 PNG 被拒绝（detected=%s）", d)
	}
	if d := http.DetectContentType(htmlHead); sniffMatches(".jpg", d) {
		t.Errorf("HTML 内容伪装成 .jpg 竟然通过（detected=%s）", d)
	}
	if d := http.DetectContentType(svgHead); sniffMatches(".png", d) {
		t.Errorf("SVG 内容伪装成 .png 竟然通过（detected=%s）", d)
	}
	// 扩展名本身不在白名单 → 无论内容是什么都拒绝
	if sniffMatches(".svg", "image/svg+xml") {
		t.Error(".svg 不在 sniffAllowed 中，不应通过")
	}
	// docx 本质是 zip
	if !sniffMatches(".docx", "application/zip") {
		t.Error("docx(zip) 应被接受")
	}
	// 空文本文件的 charset 变体
	if !sniffMatches(".txt", "text/plain; charset=utf-16le") {
		t.Error("txt 的 utf-16 变体应被接受")
	}
}

// TestSniffAllowedCoversWhitelist 保证两张表不会走偏：
// 每个允许上传的扩展名都必须配套一条嗅探规则，否则上传会 100% 失败。
func TestSniffAllowedCoversWhitelist(t *testing.T) {
	for ext := range allowedUploadTypes {
		if _, ok := sniffAllowed[ext]; !ok {
			t.Errorf("%s 在上传白名单里，却没有对应的 sniffAllowed 规则", ext)
		}
	}
	for ext := range sniffAllowed {
		if _, ok := allowedUploadTypes[ext]; !ok {
			t.Errorf("%s 有嗅探规则，却不在上传白名单里（多余配置）", ext)
		}
	}
}
