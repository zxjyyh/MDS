# 使用指南

## 一、启动方式

### 方式 A：Docker Compose（推荐）

```bash
cp .env.example .env      # 按需修改端口、时区、PUID/PGID
docker compose up -d
docker compose logs -f    # 首次启动请在这里查看 admin 的随机初始密码
# 访问 http://<主机IP>:13000
```

NAS 部署（绿联 / 飞牛）请改用 `docker-compose.nas.yml`，详见 [deployment-nas.md](./deployment-nas.md)。

### 方式 B：Windows 本地运行

```bat
scripts\start.bat
```

该脚本会依次执行「构建前端 → 复制产物 → 启动后端」，需要本机已安装 Go 1.22+ 和 Node.js 20+。

若只想跑已编译好的二进制：

```bash
cd backend
go build -o server.exe ./cmd/server
./server.exe
```

### 方式 C：源码开发模式

```bash
# 终端 1 —— 后端 :3000
cd backend && go run ./cmd/server

# 终端 2 —— 前端 :8080（带 HMR，API 自动代理到 3000）
cd frontend && npm install && npm run dev
```

Windows 下可直接双击 `scripts\start-dev.bat` 一次性拉起两个窗口。

## 二、首次初始化

- `data/data.db` 不存在时，首次启动自动建表并创建管理员账号。
- 已有数据库文件则直接复用，启动时自动执行缺失列补充（幂等 `ALTER TABLE`）。
- **管理员初始密码随机生成，只在启动日志中打印一次**，请立即记录。也可用环境变量 `ADMIN_INIT_PASSWORD` 预先指定。
- 要完全重置，删除 `data/data.db` 后重启（业务数据会全部丢失）。

## 三、功能操作

### 人员管理

1. 点击 **新增人员**，填写基本信息与各类证照有效期。
2. 进入详情页可上传证件影印件（支持图片 / PDF）。
3. 有效期填写后，仪表盘会自动纳入到期监控。

### 设备管理

1. **新增设备** → 填写设备编号、名称、检定有效期。
2. 详情页上传设备检定证书。

### 体系文档

按「资质体系」「安全生产」两类分别维护，支持多级树形分类，文档可挂附件。

### 仪表盘

- 自动统计各模块证件到期情况，分「已过期 / 30 天 / 60 天 / 90 天」四档。
- 点击数字卡片进入对应明细列表。
- 明细列表所有列均支持点击排序。

### 到期提醒

- 「系统管理 → 提醒设置」中配置提醒阈值与 SMTP 参数。
- 定时任务每日 08:30 自动扫描。
- 页面上可点击 **手动触发到期检查** 立即测试一次。

## 四、文件预览支持

| 类型 | 预览方式 |
|---|---|
| 图片（jpg/png/gif/webp/bmp） | 内嵌显示，支持缩放 |
| PDF | `vue-pdf-embed` 内嵌渲染 |
| DOCX | `docx-preview` 前端转换渲染 |
| 其他（xls/xlsx/doc/txt） | 不预览，直接下载 |

出于安全考虑，`.svg` / `.html` / `.js` 等可执行脚本的格式**禁止上传**。

## 五、常见问题

**Q：数据库文件在哪？**
A：`data/data.db`，标准 SQLite 单文件，可用任意 SQLite 客户端打开查看。

**Q：如何备份？**
A：停止服务后复制 `data/` 和 `uploads/` 两个目录即可。热备份建议使用 `sqlite3 data/data.db ".backup backup.db"`。

**Q：端口冲突怎么办？**
A：Docker 部署修改 `.env` 中的 `HOST_PORT`；本地运行修改 `PORT`。

**Q：忘记管理员密码怎么办？**
A：用其他管理员账号在「用户管理」中重置。若已无可用管理员，可用 SQLite 客户端直接更新 `users` 表的 `password_hash`（bcrypt 哈希值），并将同一行的 `token_epoch` 加 1 以吊销旧令牌。

**Q：改了密码之后其他设备还处于登录状态吗？**
A：不会。系统使用 `token_epoch` 机制，改密 / 重置密码 / 禁用账号 / 主动登出都会立即吊销该用户此前签发的所有令牌。

**Q：容器重建后所有人被登出？**
A：正常情况下不会。JWT 密钥持久化在 `data/.secrets.env` 中，只要 `data/` 目录被正确挂载就不会变。如果确实发生了，请检查 volume 挂载是否生效。
