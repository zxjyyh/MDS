# 按公司（检测站）做数据隔离 —— 一期实施方案

> 状态：**待实施**（拟在 GitHub 分支上开发，验证通过后并入 `main`）
> 目标读者：本项目的维护者

## 1. 目标

管理员在「系统管理 → 用户管理」中为账号配置**可见公司**（可多选）。配置后，该账号只能访问所配公司的
业务数据（人员、设备、资料库等）；**管理员账号始终不受限**。

## 2. 当前实现的前提（必须先理解，否则会做错）

| 现状 | 说明 |
|---|---|
| `personnel.department` / `equipment.department` **存的是公司名**（字符串，非外键） | 过滤只能按名称匹配 |
| 现有查询**已支持** `department` 过滤参数，但它是**用户可选的筛选项**，不是安全边界 | 请求里不传就等于不限 |
| `middleware.Auth` 只往 context 放 `userID / username / role` | 没有公司维度 |
| **`FileService.CanAccess` 明确假设「任何登录用户可查看全部业务数据」** | 见 `file_svc.go` 该函数上方注释。是本次改造必须推翻的核心假设 |
| 已有 2 个真实公司（宏兴 21 人 / 顺安 10 人），1 个非管理员账号 | 隔离需求真实存在 |

## 3. 核心设计

### 3.1 数据模型（新增一张表，不动任何已有表）

```sql
CREATE TABLE IF NOT EXISTS "user_companies" (
  "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
  "user_id"      INTEGER NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
  "company_name" TEXT NOT NULL,
  "created_at"   TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE("user_id", "company_name")
);
CREATE INDEX IF NOT EXISTS idx_user_companies_user ON user_companies(user_id);
```

- 存**公司名**而不是公司 ID：与 `personnel.department` 的匹配方式一致，一期不做关联重构。
- 三处必改（项目既有约定）：`migrate.go` 的 DDL 列表 + `timestampColumns` 映射 + 在 `handler` 层注册 service。

### 3.2 权限判定规则

```
受限（Unrestricted = false） ← 账号是普通用户 且 已配置至少一个公司
不受限（Unrestricted = true）← 管理员 或 未配置任何公司（渐进启用策略，见 §7）
```

- 受限时：所有列表查询追加 `AND department IN (...)`；所有按 ID 的读写做归属校验，越权返回 403。
- 不受限时：行为与现在**完全一致**（不加任何条件）。

### 3.3 强制手段：用「参数」而不是「记得调用」

把 scope 作为**必填参数**传进 service 方法（`scope *auth.Scope`）。这样新增接口时**漏传会编译不过**，
比「靠代码评审记得检查」可靠得多。

## 4. 改动清单

### 4.1 后端 —— 基础设施

| 文件 | 改动 |
|---|---|
| `internal/database/migrate.go` | 新增 `user_companies` DDL（§3.1）+ `timestampColumns` 加 `user_companies` |
| `internal/models/user_company.go`（新） | `UserCompany` 模型 |
| `internal/service/user_company_svc.go`（新） | `CompaniesOf(userID) []string`、`SetCompanies(userID, names)`、`NamesOfUsers(userIDs)` |
| `internal/auth/scope.go`（新，或放 middleware） | `Scope{Unrestricted bool; Companies []string}`；`Scope.Allows(dept string) bool`；`Scope.InClause() (string, []any)` |
| `internal/middleware/auth.go` | `acceptUser` 里加载用户公司名单，`c.Set("dataScope", scope)`；管理员直接 `Unrestricted=true` |
| `internal/handler/auth.go` | `Handler` 增 `UserCompanySvc`；新增 `currentScope(c) *auth.Scope` 辅助方法 |
| `internal/handler/user.go` | 列表/详情返回 `companies`；新增 `PUT /api/users/:id/companies`（仅管理员） |
| `internal/router/router.go` | 注册新路由 |

### 4.2 后端 —— 人员（核心）

`internal/service/personnel_svc.go`，下列方法**加 `scope` 参数**：

| 方法 | 改动 |
|---|---|
| `List` | 追加 `AND p.department IN (...)`；`scope` 受限时忽略请求里传的 `department`（**不能让用户自己扩大范围**） |
| `Options` | `SELECT DISTINCT department` 也要按 scope 过滤（否则下拉里会露出别家公司名） |
| `GetByID` | 归属校验；查不到/越权统一返回 403 或 404 |
| `Create` / `Update` | 校验目标 `department` 在 scope 内（防止把人员建到/改到别家公司） |
| `Delete` | 归属校验 |
| `AddCert` / `UpdateCert` / `DeleteCert` / `ClearReservedCert` | 由 `certID → personnel_id → department` 反查后校验 |
| `PersonnelByScope` / `CertFilesByScope` / `CertGap` | 三个方法的 `department` 参数由 scope 收窄（`buildPersonnelWhere` 统一改） |

`internal/handler/personnel.go` / `document.go`(合同) / `ligong.go`：
所有调用点传 `currentScope(c)`；`ligong` 的生成与预检循环里逐个人员做归属校验。

### 4.3 后端 —— 文件（**最关键**）

| 位置 | 改动 |
|---|---|
| `internal/service/file_svc.go` `CanAccess` | 现在是「被任一实体引用即可访问」。改为：**引用它的实体必须落在 scope 内**（人员头像 / 人员证书 → JOIN personnel 判 department；设备证书 → JOIN equipment）。同步改掉该函数上方的注释（原假设已失效） |
| `internal/handler/file.go` | `Preview` / `Download` / `Delete` 传入 scope |

> 不补这条的后果：知道 file id 就能下载别家公司人员的身份证扫描件。

### 4.4 后端 —— 设备（一期做，但见 §8 数据前置）

`equipment_svc.go` 的 `List / Options / GetByID / Create / Update / Delete / AddCert / UpdateCert / DeleteCert`
同人员口径。⚠️ 上隔离前必须先给 `department` 为空的设备补公司数据，否则设备会对所有人不可见。

### 4.5 后端 —— 公司下拉

`GET /api/companies/options`（`company_svc.go` `Options`）按 scope 过滤，只返回用户可见的公司。

### 4.6 前端

| 文件 | 改动 |
|---|---|
| `views/system/UserListView.vue` | 新增/编辑用户弹窗加「公司权限」多选（数据源 `companies`）；列表列显示已配公司 |
| `api/user.ts` | 加 `getUserCompanies` / `setUserCompanies` |
| `views/personnel/PersonnelListView.vue`、`views/equipment/EquipmentListView.vue` | 检测站下拉只用 scope 内的公司 |
| `components/materials/*`（劳动合同、证件打包、理工报名） | 公司下拉同过滤；理工报名的人员候选里不能出现 scope 外的人 |
| `stores/auth.ts` | 登录/刷新后缓存自身 scope，供上述过滤使用 |

## 5. 测试计划

**为什么测试是必须的**：这类功能的失败模式是「漏了某个入口」——只要有一个没堵，隔离就等于没做。

1. **单元测试**（`internal/service/*_test.go`）：
   - 受限 scope 下 `List` 只返回本公司的数据；`Unrestricted` 时返回全部
   - `GetByID` / `Update` / `Delete` 对 scope 外 ID 返回越权错误
   - `CanAccess`：本公司的文件 ✅、别家公司的文件 ❌、游离文件 ❌、自己上传的 ✅
   - 空 scope（未配置公司）= 不受限
2. **接口级测试**：以「王婉婷 + 只配宏兴」的身份打关键接口，断言看不到顺安的数据；
   用管理员身份断言仍能看到全部（回归保护）。
3. **人工核对清单**：按 §4 的接口清单逐个用 `curl` 或界面点一遍（受限账号 + 管理员各一次）。

## 6. 交付与验证流程

1. 建分支：`feature/company-data-scope`
2. 按 §4 顺序提交（建议分 5~6 个 commit：表与基础设施 → 人员 → 文件 → 设备 → 公司下拉 → 前端）
3. 本地 `go test ./...` + `vue-tsc -b` + `vite build` 全绿
4. **在数据库副本上验证**（沿用本项目已确立的做法，见 `2026-09-21.md`）：
   复制 `data.db` → 临时目录 → 用 `DATABASE_URL` 指向副本启动 → 配一个受限账号 → 核对可见范围
5. 你验收 → 并入 `main` → 推送 → CI + 镜像自动构建

### 分支为什么不会"污染"本地项目

Git 分支**不是复制一份文件夹**，而是在**同一个工作目录**里切换内容快照。要点：

| 情况 | 结果 |
|---|---|
| 在分支上改代码 | 工作目录里就是分支版本；`main` 的历史不受影响 |
| 切回 `main` | 工作目录**立刻恢复**成 main 的样子 |
| 分支上改坏了 | 切回 main + 重新 `go build` / `npm run build` 即完全恢复 |
| **未提交就切分支** | Git 会把未提交的改动带过去 → 所以**每完成一块就 commit** |

**唯一不在 Git 管理内、跨分支共享的东西**（要单独留意）：

| 项 | 说明 |
|---|---|
| `backend/data/data.db`（已 gitignore） | 切分支**不会**改它。分支新增的表在切回 main 后依然存在，但旧代码不认识它 → **向后兼容，不影响使用** |
| `backend/uploads/`（已 gitignore） | 同上，不受影响 |
| `server.exe` / `dist/`（构建产物） | 切分支后需要重新构建 |

**开工前的唯一必要动作**：把 `backend/data/` 与 `backend/uploads/` 复制一份冷备份
（这是唯一不可逆的部分；代码永远能靠 Git 回退）。

## 7. 灰度策略（把风险降到最低）

**规则：账号未配置任何公司 = 该账号不受限。**

注意这里说的是**账号**，不是数据：

| 对象 | 含义 |
|---|---|
| **账号**未配置公司 | 这个账号不受限 → 能看到**全部**数据（等于现在的行为） |
| **账号**已配置公司 | 立刻受限 → 只看得到所配公司的数据 |
| **数据**（人员/设备）的 `department` 为空 | 受限账号**看不到**这条数据（因为过滤条件是 `department IN (所配公司)`）；不受限账号照常看得到 |

**为什么这样设计**：上线那一刻没有任何账号配了公司 → 所有人的行为与现在**完全一致**，零变化。
之后你逐个账号配置，配一个生效一个；想"解禁"就把配置清空。管理员始终不受限。

**推论（重要）**：受限账号**新建**人员/设备时，必须选定 scope 内的公司，
因此不会产生"无公司"的数据；无公司的数据只可能由不受限账号（管理员）产生，也只对不受限账号可见。

## 8. 已知限制与前置条件

| 项 | 说明 |
|---|---|
| **设备数据前置（一期必修）** | `equipment.department` 目前为空（唯一 1 台）。直接上隔离会导致它对所有人不可见 → **开工前先给这台设备补上公司** |
| **公司改名** | 按**名称**关联：在「公司信息管理」里改名后，`personnel.department` 与 `user_companies.company_name` 都会失配。**这是既有问题**（现有筛选已如此），本次不解决；彻底解决需要改成按 ID 关联（更大的重构） |
| **资料库** | `documents` 只有"公司档案"类有 `company_name`，法规标准/IP 信息是全局共享。一期**暂不对资料库做隔离**，留到二期 |
| 仪表盘 / 提醒 | 一期不动（统计与邮件范围后续再定） |

## 9. 已确认的决策（2026-09-21）

| # | 决策 |
|---|---|
| 1 | 账号未配置公司 = **不受限**（§7 灰度策略）✅ |
| 2 | 一期**包含设备**（需先补那台设备的公司，见 §8）✅ |
| 3 | 一个账号可绑定**多个**公司，由管理员在权限里添加 ✅ |
| 4 | 资料库一期**不做**，二期再定 ✅ |
