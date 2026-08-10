# 内业管理系统

面向中小型机构的**内业资料综合管理平台**：集中管理人员档案、设备台账、各类证照证书及其有效期，并在证件临近到期时自动预警与邮件提醒。

前后端一体化打包为**单个容器 / 单个可执行文件**，不依赖外部数据库和中间件，适合局域网、办公 PC 或家用 NAS 私有化部署。

- 技术栈：Vue 3 + Vite + Element Plus / Go + Gin + SQLite（`modernc.org/sqlite`，纯 Go 实现，零 CGO）
- 部署：Docker Compose（推荐）/ 绿联 UGOS Pro、飞牛 fnOS、群晖等 NAS / Windows 独立运行

## 功能一览

| 模块 | 功能 |
|---|---|
| 人员管理 | 人员档案建档、多类证照有效期管理、证件影印件上传与在线预览 |
| 设备管理 | 设备台账管理、检定/校准有效期跟踪、检定证书归档 |
| 文件系统 | 统一的文件上传 / 下载 / 预览（图片、PDF、DOCX） |
| 体系文档 | 资质体系与安全生产文档的树形分类管理 |
| 仪表盘 | 全模块证件到期自动统计（已过期 / 30 / 60 / 90 天四级预警） |
| 到期提醒 | 每日定时扫描 + SMTP 邮件通知 + 手动触发 |
| 用户管理 | 多用户、角色权限（管理员 / 普通用户） |
| 系统设置 | SMTP 邮件配置、提醒规则自定义 |

更详细的定位、技术特点与数据库设计见 [docs/overview.md](docs/overview.md) 与 [docs/architecture.md](docs/architecture.md)。

## 项目结构

```
├── frontend/                    Vue 3 前端源码
├── backend/                     Go 后端源码
│   ├── cmd/server/              入口（日志、配置、DB、路由、优雅关闭）
│   ├── internal/                核心代码（config / database / handler / middleware / models / router / scheduler / service / utils）
│   ├── web/dist/                前端构建产物（本地运行时使用，Docker 由镜像内自带）
│   └── Dockerfile               多阶段镜像构建
├── docs/                        架构、概述、使用指南
├── .github/workflows/           CI 与镜像自动构建
├── docker-compose.yml           本地构建部署
├── docker-compose.nas.yml       NAS 部署（拉取云端镜像，推荐）
└── .env.example                 环境变量模板
```

## 快速开始（Docker Compose）

```bash
cp .env.example .env      # 按需修改端口、时区、PUID/PGID
mkdir -p data uploads logs
docker compose up -d
docker compose logs -f    # 首次启动请在这里查看 admin 的随机初始密码
# 访问 http://<主机IP>:13000
```

NAS（绿联 / 飞牛 / 群晖）请改用 `docker-compose.nas.yml`，详见 [docs/overview.md](docs/overview.md) 与 `docker-compose.nas.yml` 文件顶部注释。

## 开发环境

需要 Node.js 20+ 与 Go 1.24+。

```bash
# 终端 1 —— 前端（:8080，带 HMR，API 自动代理到 3000）
cd frontend && npm install && npm run dev

# 终端 2 —— 后端（:3000）
cd backend && go run ./cmd/server
```

首次启动自动创建 `admin` 账号，**初始密码随机生成并打印在后端控制台**（也可用 `ADMIN_INIT_PASSWORD` 预置）。登录后请立即在「用户管理」中修改密码。

## 环境变量

复制 `.env.example` 为 `.env` 后修改。**`.env` 不会被提交到仓库。**

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `HOST_PORT` | 宿主机映射端口（容器内固定 3000） | 13000 |
| `CONTAINER_NAME` | 容器名（同机多套时区分） | mds |
| `TZ` | 时区（到期提醒按本地时间触发，**建议显式设置**） | Asia/Shanghai |
| `PUID` / `PGID` | 以宿主机用户身份写文件（NAS 强烈建议设置） | 1000 / 1000 |
| `JWT_SECRET` / `JWT_REFRESH_SECRET` | JWT 签名密钥，**留空则自动生成并持久化到 `data/.secrets.env`** | 空 |
| `ADMIN_INIT_PASSWORD` | 初始管理员密码（仅首次初始化生效，留空随机） | 空 |
| `UPLOAD_MAX_SIZE` | 上传大小限制（字节） | 52428800（50MB） |
| `CORS_ORIGINS` | 跨域白名单，逗号分隔；**留空 = 完全不下发 CORS 头（最安全）** | 空 |
| `TRUSTED_PROXIES` | 反向代理 IP/网段（直连场景留空） | 空 |
| `SMTP_*` | 邮件提醒配置（也可在页面配置，页面值优先） | 空 |
| `APP_URL` | 提醒邮件中的访问链接 | http://localhost:13000 |
| `REMINDER_CRON` | 到期检查 cron（默认每天 08:30） | 30 8 * * * |
| `LOG_*` | 日志轮转（大小 / 备份数 / 保留天数） | 10MB / 5 / 30天 |
| `IMAGE_NAME` | 镜像地址（仅 nas.yml 用） | ghcr.io/zxjyyh/mds:latest |

生成 JWT 密钥：

```bash
# Linux / macOS / NAS
openssl rand -hex 32
# Windows PowerShell
-join ((1..64) | ForEach-Object { '{0:x}' -f (Get-Random -Max 16) })
```

> `JWT_SECRET` 一旦部署后不要随意更换，否则所有已登录用户会被强制登出（密钥持久化在 `data/.secrets.env`，只要 `data/` 正确挂载就不会变）。

## Windows 独立部署

```bat
scripts\start.bat
```

该脚本依次执行「构建前端 → 复制产物到 backend/web/dist → 启动后端」，需要本机已安装 Go 1.24+ 与 Node.js 20+。

或手动：

```bash
cd frontend && npm install && npm run build && xcopy dist ..\backend\web\dist /E /I
cd ..\backend && go build -o server.exe ./cmd/server && server.exe
```

访问 http://localhost:3000 。注册为 Windows 服务（断开远程桌面不停止）可用 [NSSM](https://nssm.cc/)。

## 数据与备份

| 目录 | 内容 | 是否必须持久化 |
|---|---|---|
| `data/` | SQLite 数据库 + 自动生成的 `data/.secrets.env`（JWT 密钥） | 是 |
| `uploads/` | 上传的证件、证书原件 | 是 |
| `logs/` | 运行日志 | 否 |

备份只需复制 `data/` 与 `uploads/` 两个目录。迁移到新机器时，把目录复制过去再启动容器即可。

## 安全说明

- 密码 bcrypt 哈希存储（长度校验 6–72 字节，防 bcrypt 72 字节截断绕过）
- 双令牌 JWT（access 2h / refresh 7d）+ `token_epoch` 即时吊销（改密/登出/禁用立即失效）
- 文件访问令牌与访问令牌**双向隔离**（防短时效文件令牌等价于完整会话）
- 登录接口 IP 限流（15 分钟 10 次）
- CORS **默认不放行任何跨域来源**，需显式配置白名单
- 上传扩展名 + 内容嗅探双重白名单（拦住「HTML 改名成 .jpg」类存储型 XSS）
- 安全响应头：`X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、`Content-Security-Policy`
- 请求日志**不记录 query string**（文件令牌在 query 中，避免落盘）
- 完整的安全与实现细节见 [docs/architecture.md](docs/architecture.md) 与 [SECURITY.md](SECURITY.md)

## 构建与测试

```bash
cd backend && go build ./...          # 编译后端
cd backend && go test ./internal/...  # 运行单元测试
cd frontend && npm ci && npm run build # 构建前端（含 vue-tsc 类型检查）
```

GitHub Actions 会在每次 push 时运行 CI（CI 见 `.github/workflows/ci.yml`），并自动构建多架构镜像推送到 GHCR（见 `.github/workflows/docker.yml`）。

## 许可证

[MIT](LICENSE)

---

贡献指南见 [CONTRIBUTING.md](CONTRIBUTING.md)，安全漏洞请按 [SECURITY.md](SECURITY.md) 私报告知，变更记录见 [CHANGELOG.md](CHANGELOG.md)。
