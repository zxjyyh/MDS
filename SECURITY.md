# 安全政策

## 支持版本

| 版本 | 是否安全维护 |
|---|---|
| 1.0.x | ✅ |
| < 1.0 | ❌（开源前内部版本，不再维护） |

## 报告漏洞

如果你发现了安全漏洞，**请勿开公开 Issue**，以免被恶意利用。

请通过以下方式私报：

- GitHub Security Advisory：在本仓库点击 **Security → Report a vulnerability**
- 或邮件联系维护者（见仓库主页 Profile）

请尽量提供：

1. 漏洞类型与影响范围
2. 复现步骤（最小可用 PoC）
3. 建议的修复方向（如有）

我们会在 **72 小时内**确认收到，并在确认后协商披露时间。修复合并后会同步到 CHANGELOG 并向报告者致谢（除非你要求匿名）。

## 安全设计要点

本项目面向私有化/局域网部署，已在架构层面内置以下防护，欢迎审阅与加固：

- **传输**：建议部署在反向代理（Nginx/Caddy）之后启用 HTTPS；同源单容器部署时 CORS 默认完全关闭。
- **鉴权**：
  - 双令牌 JWT（access 2h / refresh 7d），`typ` 字段隔离访问令牌与文件令牌。
  - `users.token_epoch` 参与签名，改密/登出/禁用即时吊销全部令牌。
  - 登录接口 IP 限流（15 分钟 10 次）。
- **文件上传**：
  - 扩展名白名单 + `http.DetectContentType` 内容嗅探双重校验。
  - 落盘文件名 `uuid + 已校验扩展名`，原始文件名仅作展示且经 `SanitizeFilename` 清洗。
  - 出站 `Content-Type` 从存储扩展名推导，历史脏数据同样无害。
  - 仅图片/PDF 允许 inline，其余强制 attachment 并附加 CSP。
- **数据存储**：密码 bcrypt 哈希（6–72 字节校验）；JWT 密钥持久化在 `data/.secrets.env`（0600），不入库不写 `.env`。
- **运维**：
  - 请求日志不记录 query string（文件令牌在 query 中）。
  - `SetTrustedProxies(nil)` 防止伪造 `X-Forwarded-For` 绕过限流。
  - JWT 解析强制校验签名算法，拒绝 `alg=none`。

## 安全配置建议

- 公网暴露前务必启用 HTTPS 并配置 `CORS_ORIGINS` 精确白名单。
- `TZ` 显式设置，避免定时提醒时区错位。
- 首次登录后**立即修改** admin 默认密码。
- 定期备份 `data/` 与 `uploads/`。
