#!/bin/sh
# 容器入口：按宿主机的 UID/GID 降权运行。
#
# 为什么需要：绿联 / 群晖 / 飞牛等 NAS 上，docker 卷是宿主机目录的 bind mount。
# 若容器内以 root 写文件，宿主机上这些文件的属主就是 root，
# 你在 NAS 的文件管理器里既删不掉也改不了，备份脚本也会因权限失败。
#
# 用法（compose）：
#   environment:
#     - PUID=1000   # 宿主机执行 `id -u` 的结果
#     - PGID=1000   # 宿主机执行 `id -g` 的结果
# 设为 0 则保持 root 运行（不推荐）。

set -eu

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"
APP_USER=app
APP_HOME=/app

log() { printf '[entrypoint] %s\n' "$*"; }

# 时区：TZ 环境变量对 Go 的 time.Local 生效需要 /etc/localtime 存在
if [ -n "${TZ:-}" ] && [ -f "/usr/share/zoneinfo/${TZ}" ]; then
    ln -snf "/usr/share/zoneinfo/${TZ}" /etc/localtime
    echo "${TZ}" > /etc/timezone
fi

# 确保数据目录存在（首次启动时 bind mount 的目录可能是空的）
mkdir -p "${APP_HOME}/data" "${APP_HOME}/uploads" "${APP_HOME}/logs"

if [ "${PUID}" = "0" ] || [ "${PGID}" = "0" ]; then
    log "PUID/PGID 为 0，以 root 身份运行"
    exec "$@"
fi

# 按需创建/调整用户组，幂等
if ! getent group "${PGID}" >/dev/null 2>&1; then
    addgroup -g "${PGID}" "${APP_USER}" 2>/dev/null || true
fi
GROUP_NAME="$(getent group "${PGID}" | cut -d: -f1)"
GROUP_NAME="${GROUP_NAME:-${APP_USER}}"

if ! getent passwd "${PUID}" >/dev/null 2>&1; then
    adduser -D -H -u "${PUID}" -G "${GROUP_NAME}" "${APP_USER}" 2>/dev/null || true
fi
USER_NAME="$(getent passwd "${PUID}" | cut -d: -f1)"
USER_NAME="${USER_NAME:-${APP_USER}}"

# 只 chown 数据目录，不动只读的程序与前端资源（几万个小文件 chown 会显著拖慢启动）
chown -R "${PUID}:${PGID}" \
    "${APP_HOME}/data" "${APP_HOME}/uploads" "${APP_HOME}/logs" 2>/dev/null || \
    log "警告: 无法修改数据目录属主，若后续出现写入失败请检查宿主机目录权限"

log "以 ${USER_NAME}(${PUID}:${PGID}) 启动服务"
exec su-exec "${PUID}:${PGID}" "$@"
