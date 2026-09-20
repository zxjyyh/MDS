package service

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/zxjyyh/mds/internal/models"
	"github.com/zxjyyh/mds/internal/utils"
)

type OperationLogService struct{ DB *sqlx.DB }

// OperationEntry 一条待记录的操作。调用方只需给出「谁 + 什么请求」，
// 具体的中文动作名/资源类型/资源 id 由 describeOperation 从路径推导。
type OperationEntry struct {
	UserID   int
	Username string
	Role     string
	Method   string
	Path     string
}

// 模块名：把 /api 下的第一段翻译成中文，用于日志可读性。
var operationModules = map[string]string{
	"auth": "登录", "personnel": "人员管理", "equipment": "设备管理",
	"companies": "公司信息", "documents": "资料库", "categories": "资料分类",
	"users": "用户管理", "files": "文件", "dashboard": "仪表盘",
	"reminders": "提醒", "settings": "系统设置", "operation-logs": "操作日志",
	"backups": "数据备份",
}

// 整体具名动作（按完整路径匹配）
var namedActions = map[string]string{
	"/api/auth/login":    "登录",
	"/api/auth/logout":   "退出登录",
	"/api/auth/password": "修改密码",
}

// 具名动作（按路径末段匹配）
var namedActionsByTail = map[string]string{
	"upload":   "上传文件",
	"preview":  "预览文件",
	"download": "下载文件",
	"export":   "导出",
	"contract": "生成劳动合同",
	"ignore":   "忽略预警",
	"clear":    "清空",
	"test":     "发送测试提醒",
	"run":      "执行备份",
}

func methodVerb(method string) string {
	switch method {
	case http.MethodPost:
		return "新增"
	case http.MethodPut, http.MethodPatch:
		return "修改"
	case http.MethodDelete:
		return "删除"
	case http.MethodGet:
		return "查看"
	}
	return method
}

// describeOperation 由「HTTP 方法 + 请求路径」推导出中文动作名、资源类型与资源 id。
//
// 为什么用推导而不是在每个 handler 里手写一行日志：手写容易漏（尤其以后新增接口时），
// 而路径本身已经承载了足够语义 —— 如 PUT /api/personnel/12 → 「人员管理 · 修改」，目标 id=12。
func describeOperation(method, path string) (string, string, *int) {
	segments := strings.Split(strings.Trim(strings.TrimPrefix(path, "/api/"), "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return methodVerb(method), "", nil
	}
	resource := segments[0]
	module := operationModules[resource]
	if module == "" {
		module = resource
	}
	last := segments[len(segments)-1]

	var targetID *int
	if n, err := strconv.Atoi(last); err == nil {
		targetID = &n
	}

	if named, ok := namedActions[path]; ok {
		return named, resource, targetID
	}
	// 证书相关的三条路径要区分新增/修改/删除
	if last == "certificates" {
		return module + " · 新增证书", resource, targetID
	}
	if len(segments) >= 2 && segments[len(segments)-2] == "certificates" {
		return module + " · 证书" + methodVerb(method), resource, targetID
	}
	if named, ok := namedActionsByTail[last]; ok {
		return named, resource, targetID
	}
	return module + " · " + methodVerb(method), resource, targetID
}

// Record 写入一条操作日志。失败只告警、不影响主流程（审计不能反过来阻断业务）。
func (s *OperationLogService) Record(e OperationEntry) {
	action, targetType, targetID := describeOperation(e.Method, e.Path)

	// database/sql 不接受 *int，需要解引用
	var tid interface{}
	if targetID != nil {
		tid = *targetID
	}

	_, err := s.DB.Exec(`INSERT INTO operation_logs
		(user_id, username, role, action, method, path, target_type, target_id, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		e.UserID, nilIfBlank(e.Username), nilIfBlank(e.Role), action,
		e.Method, e.Path, nilIfBlank(targetType), tid, utils.NowString())
	if err != nil {
		log.Printf("WARN: 写入操作日志失败: %v", err)
	}
}

// List 按关键字（用户名/动作/路径）与角色筛选，分页返回。role 取 admin / user，空表示不限。
func (s *OperationLogService) List(keyword, role string, page, pageSize int) ([]models.OperationLog, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if kw := strings.TrimSpace(keyword); kw != "" {
		where += " AND (l.username LIKE ? OR l.action LIKE ? OR l.path LIKE ?)"
		like := "%" + kw + "%"
		args = append(args, like, like, like)
	}
	if role != "" {
		where += " AND l.role=?"
		args = append(args, role)
	}

	var total int
	if err := s.DB.Get(&total, "SELECT COUNT(*) FROM operation_logs l "+where, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectQ := fmt.Sprintf("SELECT * FROM operation_logs l %s ORDER BY l.id DESC LIMIT ? OFFSET ?", where)
	rows, err := s.DB.Queryx(selectQ, append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := []models.OperationLog{}
	for rows.Next() {
		var r models.OperationLog
		if err := rows.StructScan(&r); err != nil {
			return nil, 0, fmt.Errorf("failed to scan operation log: %w", err)
		}
		list = append(list, r)
	}
	return list, total, nil
}

// Clear 清空操作日志（仅管理员，用于日志量过大时手动清理）。
func (s *OperationLogService) Clear() (int64, error) {
	res, err := s.DB.Exec("DELETE FROM operation_logs")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
