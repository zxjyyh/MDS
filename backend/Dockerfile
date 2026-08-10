# syntax=docker/dockerfile:1
#
# 单容器镜像：前端静态资源直接打进后端二进制所在的镜像里，由 Gin 托管。
# 构建上下文是仓库根目录（见 docker-compose.yml 的 build.context: .）。
#
# 国内网络加速（可选）：
#   docker compose build \
#     --build-arg GOPROXY=https://goproxy.cn,direct \
#     --build-arg NPM_REGISTRY=https://registry.npmmirror.com

# ---------- Stage 1: 构建前端 ----------
FROM node:22-alpine AS frontend-build
ARG NPM_REGISTRY=https://registry.npmjs.org
WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm config set registry "${NPM_REGISTRY}" \
    && npm ci --no-audit --no-fund

COPY frontend/ ./
RUN npm run build

# ---------- Stage 2: 构建 Go 后端 ----------
FROM golang:1.24-alpine AS backend-build
# TARGETARCH / TARGETOS 由 BuildKit 自动注入，支持 amd64 / arm64 多架构
ARG TARGETARCH
ARG TARGETOS
ARG GOPROXY=https://proxy.golang.org,direct
ARG APP_VERSION=dev
ENV GOPROXY=${GOPROXY} \
    CGO_ENABLED=0

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY backend/ ./
# -trimpath 去掉二进制里的绝对构建路径，既减小体积也避免泄露构建机目录结构
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X main.appVersion=${APP_VERSION}" -o /server ./cmd/server

# ---------- Stage 3: 运行时 ----------
FROM alpine:3.20

# su-exec: 以指定 UID/GID 降权执行（比 gosu 小得多）
# wget:    HEALTHCHECK 用
# tzdata:  Go 的 time.Local 需要时区数据库
RUN apk add --no-cache ca-certificates tzdata wget su-exec

ENV TZ=Asia/Shanghai \
    PORT=3000 \
    NODE_ENV=production \
    DATABASE_URL=./data/data.db \
    UPLOAD_DIR=./uploads \
    FRONTEND_DIR=./web/dist \
    LOG_DIR=./logs

WORKDIR /app

COPY --from=backend-build /server ./server
COPY --from=frontend-build /app/frontend/dist ./web/dist
COPY backend/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh /app/server \
    && mkdir -p /app/data /app/uploads /app/logs

# 刻意不复制 .env：配置一律由 compose 的 env_file / environment 注入。
# 早期版本在镜像里放 .env，容器重建时会重新生成随机 JWT 密钥，导致全员掉线。

# 刻意不声明 VOLUME：一旦声明，即使用户忘了配 bind mount，Docker 也会静默创建
# 匿名卷，数据看起来"没丢"但散落在 /var/lib/docker/volumes 下，NAS 用户几乎不可能找回。
# 不声明的话，忘配挂载就是容器内临时目录，重建即空——问题会立刻暴露，而不是半年后。

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3000/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/server"]
