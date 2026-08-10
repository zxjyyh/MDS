package handler

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/zxjyyh/mds/internal/service"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.Fail(c, -1, "请选择文件")
		return
	}

	f, err := h.FileSvc.Upload(file, utils.GetAuthUserID(c))
	if err != nil {
		utils.Fail(c, -1, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"id":           f.ID,
		"originalName": f.OriginalName,
		"mimeType":     f.MimeType,
		"size":         f.Size,
		"url":          fmt.Sprintf("/api/files/%d/preview", f.ID),
	})
}

func (h *Handler) PreviewFile(c *gin.Context) { h.serveFile(c, false) }

func (h *Handler) DownloadFile(c *gin.Context) { h.serveFile(c, true) }

// serveFile 是预览与下载的统一出口。
//
// 安全要点：
//  1. Content-Type 从**落盘扩展名**推导。stored_name 由服务端生成
//     （uuid + 已通过白名单与内容嗅探校验的扩展名），不可被用户污染。
//     数据库 mime_type 列可能残留老版本写入的用户可控值，这里一律不用。
//  2. 只有图片和 PDF 允许 inline，其余强制 attachment。
//  3. 附加严格的 CSP + sandbox，即使某天白名单被绕过，内容也无法执行脚本
//     或访问同源资源。
//  4. 始终发送 X-Content-Type-Options: nosniff，阻止浏览器自行猜测类型。
func (h *Handler) serveFile(c *gin.Context, forceAttachment bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.Fail(c, -1, "参数错误")
		return
	}

	allowed, err := h.FileSvc.CanAccess(id, utils.GetAuthUserID(c))
	if err != nil || !allowed {
		utils.Fail(c, -1, "文件不存在或无权访问")
		return
	}

	path, ext, name, err := h.FileSvc.GetFilePath(id)
	if err != nil {
		utils.Fail(c, -1, "文件不存在")
		return
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		utils.Fail(c, -1, "文件不存在")
		return
	}

	inline := !forceAttachment && service.IsInlineRenderable(ext)

	c.Header("Content-Type", service.CanonicalMime(ext))
	c.Header("Content-Disposition", contentDisposition(name, inline))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy",
		"default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	// 文件链接里带着访问令牌，禁止通过 Referer 外泄
	c.Header("Referrer-Policy", "no-referrer")
	c.File(path)
}

// contentDisposition 按 RFC 5987 输出文件名，兼顾中文与老浏览器。
func contentDisposition(name string, inline bool) string {
	disp := "attachment"
	if inline {
		disp = "inline"
	}
	safe := service.SanitizeFilename(name)
	// ASCII 回退名：非 ASCII 字符替换为下划线
	var ascii strings.Builder
	for _, r := range safe {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			ascii.WriteByte('_')
		} else {
			ascii.WriteRune(r)
		}
	}
	return fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`,
		disp, ascii.String(), url.PathEscape(safe))
}

func (h *Handler) DeleteFile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.FileSvc.Delete(id); err != nil {
		utils.Fail(c, -1, "删除失败")
		return
	}
	utils.SuccessMsg(c, nil, "删除成功")
}
