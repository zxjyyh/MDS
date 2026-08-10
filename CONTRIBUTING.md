# 贡献指南

感谢你考虑为「内业管理系统」做贡献！本项目是面向私有化部署的开源项目，欢迎 Issue、文档修正与代码 PR。

## 行为准则

- 对所有人保持尊重与友善。
- 就事论事，不人身攻击。

## 如何开始

1. Fork 本仓库并克隆到本地。
2. 创建特性分支：`git checkout -b feat/your-feature`。
3. 本地验证：
   ```bash
   cd backend && go build ./... && go test ./internal/...
   cd frontend && npm ci && npm run build
   ```
4. 提交信息清晰描述「为什么改」，而不仅仅是「改了什么」。
5. 推送分支并发起 Pull Request，关联相关 Issue。

## 代码规范

- **后端（Go）**：遵循 `gofmt` 与 `go vet`；提交前请运行
  ```bash
  gofmt -l .
  go vet ./...
  ```
  新功能建议配套单元测试，重点覆盖安全相关逻辑（鉴权、上传校验、SQL 标识符校验）。
- **前端（Vue/TS）**：遵循 ESLint 配置；类型检查必须通过 `vue-tsc`（`npm run build` 已包含）。

## 安全相关

- 不要在代码中硬编码任何密钥、口令或内网地址。
- 涉及鉴权、文件上传、SQL 拼接的改动，请在 PR 描述中说明安全影响。
- 发现安全漏洞**请勿开公开 Issue**，按 [SECURITY.md](SECURITY.md) 私报告知。

## 提交信息约定（推荐）

```
feat:     新功能
fix:      缺陷修复
docs:     文档
refactor: 重构（无功能变化）
test:     测试
chore:    构建/工具/杂项
```

## 版本与发版

- 镜像 tag 跟随 Git tag（`vX.Y.Z`）自动构建并推送 GHCR。
- 变更记录统一维护在 [CHANGELOG.md](CHANGELOG.md)。

## 环境要求

- Go 1.24+
- Node.js 20+
- Docker 20.10+（部署 / 镜像构建）
