# 变更记录

本项目采用 [Keep a Changelog](https://keepachangelog.com/zh-CN/) 风格，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.0] - 2026-08

首个开源版本。基于原内部「宏兴内业管理系统」清理敏感信息、修复安全漏洞、适配 NAS 部署后开源。

### 新增

- 统一项目显示名为「内业管理系统」，移除原公司专属命名。
- 支持绿联 UGOS Pro / 飞牛 fnOS / 群晖等 NAS 的 Docker Compose 部署（`docker-compose.nas.yml`）。
- JWT 密钥持久化到 `data/.secrets.env`，容器重建不再全员掉线。
- 短信/邮件提醒的 SMTP 配置迁入数据库，支持在页面内修改。
- 文件访问令牌（`typ=file`）与访问令牌双向隔离，用于 `<img>`/下载等无法携带 Header 的场景。
- 登录接口 IP 限流（15 分钟 10 次）并带后台 janitor 防 map 无限增长 OOM。
- 多架构镜像（linux/amd64 + linux/arm64）自动构建并推送到 GHCR。
- 运行时 `PUID`/`PGID` + su-exec 降权，解决 NAS bind mount 文件属主问题。
- 日志轮转（lumberjack）+ SIGTERM 优雅关闭。
- 后端单元测试（pragma 生效、外键约束、时间戳自愈、文件安全、JWT 隔离、bcrypt 边界）。
- GitHub Actions CI（gofmt/vet/build/test + 前端构建 + 镜像构建）。

### 安全

- **存储型 XSS**：上传改为「扩展名白名单 + 内容嗅探 + 出站 MIME 从存储扩展名推导」，拦截「HTML 改名成 .jpg」。
- **SQL 注入**：仪表盘动态表名/列名改为标识符白名单 + 正则校验。
- **信息泄露**：`uploaded_files.storage_path` 不再经 API 返回；JWT 密钥不再写在 `.env`；请求日志不记 query string（防文件令牌落盘）。
- **CORS 默认安全**：留空则完全不下发 CORS 头；`*` 不带凭证；白名单精确匹配。
- **令牌吊销**：`users.token_epoch` 参与签名，改密/登出/禁用即时失效。
- **bcrypt 截断绕过**：入口校验密码 6–72 字节。
- **可信代理**：`SetTrustedProxies(nil)`，防 `X-Forwarded-For` 伪造绕过限流。
- **alg=none 绕过**：JWT 解析强制校验签名算法。

### 修复

- 修复文件下载 401（下载路由改用 `AuthWithQuery`，支持 query 中的文件令牌）。
- 修复遗留 UTC 时间戳与本地时间混用导致的排序/显示错乱（启动幂等自愈）。
- 删除 VOLUME 声明，避免匿名卷静默吞数据；删除构建期复制 `.env` 导致每次重建随机密钥。

### 变更

- Go 1.22 → 1.24，Node 20 → 22（镜像基础镜像同步升级）。
- 配置改为只读（不再回写 `.env`）。
- 内部文档收敛到 `docs/`。
