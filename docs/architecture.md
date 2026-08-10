# 技术架构

## 一、前端技术栈

| 技术 | 版本 | 用途 |
|---|---|---|
| Vue 3 | 3.5 | 响应式 UI 框架，Composition API |
| TypeScript | 5.7 | 类型安全，编译期错误检查 |
| Vite | 6.x | 构建工具，毫秒级 HMR |
| Element Plus | 2.9 | 组件库：表格 / 表单 / 对话框 / 上传 |
| Vue Router | 4.5 | SPA 路由，嵌套布局 |
| Pinia | 2.3 | 全局状态管理（auth store） |
| Axios | 1.7 | HTTP 封装，自动携带 JWT |
| dayjs | 1.11 | 日期格式化 |
| vue-pdf-embed | 2.1 | PDF 在线预览 |
| docx-preview | 0.3 | Word 文档在线预览 |
| Sass | 1.83 | CSS 预处理，scoped 样式隔离 |
| unplugin-auto-import | 19.x | Element Plus 按需自动导入 |

构建配置：

```
开发服务器端口  : 8080
API 代理        : /api → localhost:3000
路径别名        : @ → src/
生产产物        : frontend/dist → 复制到 backend/web/dist（go embed）
```

## 二、后端技术栈

| 技术 | 版本 | 用途 |
|---|---|---|
| Go | 1.22+ | 编译型语言，单一二进制部署 |
| Gin | 1.10 | HTTP 框架 |
| modernc.org/sqlite | 1.30 | **纯 Go 零 CGO** SQLite 驱动 |
| sqlx | 1.4 | 扩展 `database/sql`，结构体自动映射 |
| golang-jwt/jwt/v5 | 5.2 | JWT 鉴权，access 2h + refresh 7d |
| golang.org/x/crypto | 0.24 | bcrypt 密码哈希 |
| Viper | 1.19 | `.env` + 环境变量配置 |
| zap + lumberjack | 1.27 / 2.2 | 结构化日志 + 文件轮转 |
| robfig/cron/v3 | 3.0 | 定时任务（每日 08:30 到期检查） |
| gomail.v2 | 2.0 | SMTP 邮件推送 |
| google/uuid | 1.6 | 文件唯一标识 |

### 分层结构

```
cmd/server/main.go     入口：日志、配置、DB、路由、优雅关闭
├── internal/router    路由注册、中间件装配、静态资源与 SPA fallback
├── internal/handler   HTTP 处理：参数绑定、鉴权取值、响应封装
├── internal/service   业务逻辑与 SQL 执行
├── internal/models    数据模型（db / json tag）
├── internal/database  连接、pragma 校验、自动建表迁移、种子数据
├── internal/middleware 日志 / CORS / 安全头 / JWT / 限流
├── internal/config    配置结构体与加载
├── internal/scheduler 定时任务
└── internal/utils     JWT、响应、时间、到期计算
```

### 中间件管线

```
请求 → Recovery → Logger → CORS → SecurityHeaders → RateLimit(仅登录) → JWT → Handler
```

### 数据库设计

| 模块 | 核心表 | 说明 |
|---|---|---|
| 人员管理 | `personnel` | 基本档案 + 内置证照日期字段 |
| 人员证书 | `personnel_certificates` | 独立证书文件 + 有效期（`ON DELETE CASCADE`） |
| 设备管理 | `equipment` | 设备台账 + 检定有效期 |
| 设备证书 | `equipment_certificates` | 设备检定证书（`ON DELETE CASCADE`） |
| 文件系统 | `uploaded_files` | 统一的文件元数据与存储路径 |
| 体系文档 | `documents` + `categories` | 资质 / 安全两类，树形分类 |
| 用户系统 | `users` | 账号、角色、`token_epoch`（令牌吊销） |
| 应用设置 | `app_settings` | k-v 表，存放 SMTP 等运行时可改配置 |
| 提醒规则 | `reminder_rules` | 到期提醒阈值配置 |
| 提醒日志 | `reminder_logs` | 提醒发送历史 |
| 仪表盘 | — | 无独立表，实时聚合 |

## 三、关键实现说明

### SQLite 连接参数（易错点）

`modernc.org/sqlite` 与 `mattn/go-sqlite3` 的 DSN 语法**不兼容**，且对无法识别的参数**静默忽略、不报错**。本项目使用：

```
file:data.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)
             &_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)&_txlock=immediate
```

启动时会调用 `database.VerifyPragmas()` 回读实际生效值并打印日志，若 `journal_mode != wal` 会输出 WARN —— 这是唯一能发现 DSN 写错的手段。

连接池限制为 4 条，并且业务代码中**禁止嵌套游标**（外层 `rows` 未关闭时又发起新查询），否则在慢盘上极易耗尽连接池死锁。

### 鉴权与令牌

- **访问令牌**（`typ` 为空）：只允许通过 `Authorization: Bearer` 头传递。
- **文件令牌**（`typ=file`）：由 `GET /api/auth/file-token` 签发，有效期 30 分钟，只允许通过 query string 传递，用于 `<img src>` 这类无法携带自定义头的场景。
- 两者**双向隔离**：query 中出现访问令牌、或 Header 中出现文件令牌，一律 401。
- `users.token_epoch` 参与签名校验，改密 / 重置 / 禁用 / 登出时自增，实现即时吊销。

### 文件上传安全

1. 扩展名白名单（小写归一化）。
2. 读取前 512 字节做 `http.DetectContentType` 内容嗅探，与扩展名比对，不一致直接拒绝——可拦住「HTML 改名成 `.jpg`」这类存储型 XSS。
3. 落盘文件名为 `uuid + 已校验的扩展名`，用户提供的原始文件名只作为展示字段并经过 `SanitizeFilename` 清洗。
4. 出站时 `Content-Type` 从**存储扩展名**推导，绝不使用数据库里记录的用户上传值，因此历史脏数据同样无害。
5. 仅图片与 PDF 允许 `inline`，其余强制 `attachment`，并附加 `Content-Security-Policy: ... sandbox`。

### 时区

SQLite 的 `DEFAULT (datetime('now'))` 返回 **UTC**，而 Go 侧写入的是带偏移的本地时间。两者格式可区分（前者 `2026-08-10 04:00:00`，后者 `2026-08-10T12:00:00+08:00`），因此启动时运行幂等的 `NormalizeLegacyTimestamps()` 把历史 UTC 值就地转换为本地时间格式。DDL 中的默认值**故意保留为 UTC**，以保证「不可识别的脏数据」永远不会产生。

## 四、API 规范

```
请求格式  : JSON (Content-Type: application/json)
鉴权方式  : Bearer Token (Authorization 头)
统一响应  : { "code": 0, "message": "ok", "data": {...} }
分页响应  : { "code": 0, "message": "ok",
              "data": { "list": [...], "total": N, "page": P, "pageSize": S } }
错误响应  : { "code": -1, "message": "错误描述" }
```

## 五、Docker 构建

```
Stage 1  node:22-alpine     → npm ci && npm run build（含 vue-tsc 类型检查）
Stage 2  golang:1.24-alpine → go build -trimpath（复制 Stage 1 的 dist 后 embed）
Stage 3  alpine:3.20        → 二进制 + su-exec + tzdata + entrypoint
```

运行时以 root 启动 `docker-entrypoint.sh`，按 `PUID`/`PGID` 创建用户、`chown` 数据目录，再 `exec su-exec` 降权运行主进程（linuxserver.io 惯例）。`PUID=0` 时保留 root 直跑。

## 六、安全措施汇总

- 密码 bcrypt 哈希存储，长度校验 6–72 字节
- 双令牌 JWT（access 2h / refresh 7d）+ `token_epoch` 吊销
- 文件访问令牌与访问令牌双向隔离
- 登录接口 IP 限流（15 分钟 10 次）
- CORS **默认不放行任何跨域来源**，需显式配置白名单
- 上传扩展名 + 内容嗅探双重白名单
- 安全响应头：`X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy: no-referrer`、下载接口独立 CSP sandbox
- `SetTrustedProxies(nil)`，避免 `X-Forwarded-For` 伪造绕过限流
- 请求日志**不记录 query string**（文件令牌在 query 中）
