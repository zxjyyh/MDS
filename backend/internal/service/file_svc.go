package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// allowedUploadTypes 是允许上传的扩展名 → 规范 MIME 的映射。
//
// 白名单而非黑名单：任何不在表中的扩展名一律拒绝。
// 特别注意这些**必须**被挡在外面：
//
//	.svg   —— 直接在浏览器地址栏打开会执行内嵌 <script>，是经典的存储型 XSS 载体
//	.html/.htm/.xhtml/.xml —— 同上
//	.js/.mjs —— 可被当作脚本引入
var allowedUploadTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".txt":  "text/plain; charset=utf-8",
}

// sniffAllowed 是「扩展名 → 允许的内容嗅探结果」白名单。
//
// http.DetectContentType 只认得少数几种魔数：
//   - OOXML（docx/xlsx）本质是 zip → application/zip
//   - OLE2（老版 doc/xls）→ application/octet-stream
//   - 纯文本 → text/plain; charset=utf-8
//
// 这一步的目的是拦住「把 evil.html 改名成 photo.jpg 上传」这种攻击：
// 扩展名说是图片，实际内容嗅探出 text/html，直接拒绝。
var sniffAllowed = map[string][]string{
	".jpg":  {"image/jpeg"},
	".jpeg": {"image/jpeg"},
	".png":  {"image/png"},
	".gif":  {"image/gif"},
	".webp": {"image/webp"},
	".bmp":  {"image/bmp"},
	".pdf":  {"application/pdf"},
	".doc":  {"application/octet-stream", "application/x-ole-storage"},
	".docx": {"application/zip"},
	".xls":  {"application/octet-stream", "application/x-ole-storage"},
	".xlsx": {"application/zip"},
	".txt":  {"text/plain; charset=utf-8", "text/plain; charset=utf-16be", "text/plain; charset=utf-16le"},
}

// inlineRenderable 是允许以 inline 方式直接在浏览器中渲染的类型。
// 其余一律强制 attachment（下载），避免任何内容被当作页面解析。
var inlineRenderable = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".webp": true, ".bmp": true,
	".pdf": true,
}

// CanonicalMime 返回扩展名对应的规范 MIME。
// 永远不要使用客户端上传时提供的 Content-Type —— 那是用户完全可控的字段。
func CanonicalMime(ext string) string {
	if m, ok := allowedUploadTypes[strings.ToLower(ext)]; ok {
		return m
	}
	return "application/octet-stream"
}

// IsInlineRenderable 判断该扩展名是否可以 inline 展示。
func IsInlineRenderable(ext string) bool {
	return inlineRenderable[strings.ToLower(ext)]
}

// SanitizeFilename 清洗用户提供的原始文件名。
// 该值会出现在 Content-Disposition 响应头里，必须剥离：
//   - 路径分隔符（防目录穿越）
//   - 控制字符、CR/LF（防响应头注入）
//   - 引号（防 Disposition 参数截断）
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 0x20 || r == 0x7f: // 控制字符，含 \r \n
			continue
		case r == '"' || r == '\'':
			continue
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	out = strings.TrimLeft(out, ".") // 防止生成 ".." / 隐藏文件
	if out == "" {
		out = "file"
	}
	if len([]rune(out)) > 180 {
		out = string([]rune(out)[:180])
	}
	return out
}

type FileService struct {
	DB        *sqlx.DB
	UploadDir string
	MaxSize   int64
}

func (s *FileService) Upload(file *multipart.FileHeader, uploadedBy int) (*models.UploadedFile, error) {
	if file.Size > s.MaxSize {
		return nil, fmt.Errorf("文件大小不能超过 %dMB", s.MaxSize/(1024*1024))
	}
	if file.Size == 0 {
		return nil, errors.New("文件内容为空")
	}

	originalName := SanitizeFilename(file.Filename)
	ext := strings.ToLower(filepath.Ext(originalName))
	if _, ok := allowedUploadTypes[ext]; !ok {
		return nil, fmt.Errorf("不支持的文件类型 %q，仅允许：图片 / PDF / Word / Excel / 文本", ext)
	}

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 读前 512 字节做内容嗅探，再 Seek 回开头写盘
	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	detected := http.DetectContentType(head[:n])
	if !sniffMatches(ext, detected) {
		return nil, fmt.Errorf("文件内容与扩展名不符（检测为 %s），已拒绝上传", detected)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	now := time.Now()
	subDir := filepath.Join(s.UploadDir, now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		return nil, err
	}

	// 落盘文件名完全由服务端生成：uuid + 已校验的小写扩展名。
	// 因此后续从 storedName 推导出的 MIME 是可信的，不受用户输入影响。
	storedName := uuid.New().String() + ext
	storagePath := filepath.Join(subDir, storedName)

	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, err
	}
	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		os.Remove(storagePath)
		return nil, copyErr
	}
	if closeErr != nil {
		os.Remove(storagePath)
		return nil, closeErr
	}

	f := &models.UploadedFile{
		OriginalName: originalName,
		StoredName:   storedName,
		MimeType:     CanonicalMime(ext), // 服务端推导，绝不使用 file.Header.Get("Content-Type")
		Size:         written,
		StoragePath:  storagePath,
		UploadedByID: uploadedBy,
		CreatedAt:    utils.NowString(),
	}

	res, err := s.DB.Exec(`INSERT INTO uploaded_files (original_name, stored_name, mime_type, size, storage_path, uploaded_by, created_at)
		VALUES (?,?,?,?,?,?,?)`,
		f.OriginalName, f.StoredName, f.MimeType, f.Size, f.StoragePath, uploadedBy, f.CreatedAt)
	if err != nil {
		os.Remove(storagePath)
		return nil, err
	}
	id, _ := res.LastInsertId()
	f.ID = int(id)
	return f, nil
}

func sniffMatches(ext, detected string) bool {
	allowed, ok := sniffAllowed[ext]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if strings.EqualFold(a, detected) {
			return true
		}
	}
	// 空文本文件等场景下 DetectContentType 会给出 text/plain 的其他 charset 变体
	if ext == ".txt" && strings.HasPrefix(detected, "text/plain") {
		return true
	}
	return false
}

func (s *FileService) GetByID(id int) (*models.UploadedFile, error) {
	var f models.UploadedFile
	err := s.DB.Get(&f, "SELECT * FROM uploaded_files WHERE id=?", id)
	return &f, err
}

// GetFilePath 返回落盘路径、存储扩展名与展示用文件名。
// 注意返回的是**存储扩展名**而不是数据库里的 mime_type：
// 老版本会把用户上传时声明的 Content-Type 直接存进 mime_type 列，
// 那些历史脏数据不可信，一律以服务端生成的 stored_name 为准。
//
// 落盘路径经过 resolveStoragePath 兜底：旧数据（如 Windows 上跑出来的）
// 可能把 Windows 绝对路径或不同子目录结构存进了 storage_path 列，
// 在 Linux 容器里直接打开会失败导致图片裂开，这里自动重定位。
func (s *FileService) GetFilePath(id int) (path, ext, name string, err error) {
	var f models.UploadedFile
	err = s.DB.Get(&f, "SELECT storage_path, stored_name, original_name FROM uploaded_files WHERE id=?", id)
	if err != nil {
		return "", "", "", err
	}
	return s.resolveStoragePath(f.StoragePath, f.StoredName), strings.ToLower(filepath.Ext(f.StoredName)), f.OriginalName, nil
}

// resolveStoragePath 在磁盘上定位实际文件，兼容历史数据中可能写入的
// Windows 绝对路径（如 E:\...\uploads\...）或不同子目录结构。
// 兜底顺序：
//  1. storage_path 本身存在 → 直接用（新上传文件走这条快速路径）
//  2. 去掉盘符 / 绝对前缀 / uploads 前缀后拼回 UploadDir 再试
//  3. 在 UploadDir 内递归按 stored_name（基准文件名，全局唯一）查找
func (s *FileService) resolveStoragePath(storagePath, storedName string) string {
	if storagePath != "" {
		if _, err := os.Stat(storagePath); err == nil {
			return storagePath
		}
	}
	// 去掉 Windows 盘符（C:）与各类前缀，拼回 UploadDir 再试一次
	rel := storagePath
	if len(rel) >= 2 && rel[1] == ':' {
		rel = rel[2:]
	}
	rel = strings.TrimPrefix(rel, string(os.PathSeparator))
	rel = strings.TrimPrefix(rel, "uploads"+string(os.PathSeparator))
	rel = strings.TrimPrefix(rel, "uploads")
	rel = strings.TrimPrefix(rel, string(os.PathSeparator))
	if rel != "" {
		if cand := filepath.Join(s.UploadDir, rel); fileExists(cand) {
			return cand
		}
	}
	// 最后手段：在 UploadDir 内递归按基准文件名查找
	if storedName != "" {
		if found, ok := s.findByName(storedName); ok {
			return found
		}
	}
	return storagePath
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// findByName 在 UploadDir 内递归查找名为 name 的文件（stored_name 全局唯一）。
func (s *FileService) findByName(name string) (string, bool) {
	found := ""
	_ = filepath.WalkDir(s.UploadDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == name {
			found = p
			return filepath.SkipAll
		}
		return nil
	})
	return found, found != ""
}

// NormalizeFilePaths 启动时一次性把 uploaded_files 表里可能错误的 storage_path
// （例如历史 Windows 绝对路径、或被改过的子目录结构）改写为当前能定位到的真实路径，
// 避免后续每次读图都走递归查找。幂等：文件确实不存在时不会误改。
func NormalizeFilePaths(db *sqlx.DB, uploadDir string) (int, error) {
	svc := &FileService{DB: db, UploadDir: uploadDir}
	type frow struct {
		ID          int    `db:"id"`
		StoragePath string `db:"storage_path"`
		StoredName  string `db:"stored_name"`
	}
	var rows []frow
	if err := db.Select(&rows, "SELECT id, storage_path, stored_name FROM uploaded_files"); err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		resolved := svc.resolveStoragePath(r.StoragePath, r.StoredName)
		if resolved != r.StoragePath {
			if _, err := db.Exec("UPDATE uploaded_files SET storage_path=? WHERE id=?", resolved, r.ID); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// CanAccess 判断某个登录用户是否有权读取该文件。
//
// 规则：文件被任一业务实体引用（人员头像 / 人员证书 / 设备证书 / 体系文档附件），
// 或者是该用户自己上传的。
//
// 为什么不做成 owner-only：本应用的读模型就是「任何登录用户都能查看全部业务数据」，
// 人员列表页要显示所有人的头像。若限制成只能看自己上传的，普通用户会看到一片裂图。
// 这里真正要拦的是「猜 ID 遍历那些尚未被任何实体引用的游离文件」。
func (s *FileService) CanAccess(fileID, userID int) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM uploaded_files      WHERE id=? AND uploaded_by=?
			UNION ALL
			SELECT 1 FROM personnel           WHERE photo_file_id=?
			UNION ALL
			SELECT 1 FROM personnel_certificates WHERE file_id=?
			UNION ALL
			SELECT 1 FROM equipment_certificates WHERE file_id=?
			UNION ALL
			SELECT 1 FROM documents           WHERE file_id=?
		)`
	var ok bool
	err := s.DB.Get(&ok, q, fileID, userID, fileID, fileID, fileID, fileID)
	return ok, err
}

func (s *FileService) Delete(id int) error {
	var f models.UploadedFile
	if err := s.DB.Get(&f, "SELECT * FROM uploaded_files WHERE id=?", id); err != nil {
		return err
	}
	_ = os.Remove(s.resolveStoragePath(f.StoragePath, f.StoredName))
	_, err := s.DB.Exec("DELETE FROM uploaded_files WHERE id=?", id)
	return err
}
