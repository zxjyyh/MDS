import { ref } from 'vue';
import request from './request';

/**
 * 文件访问令牌（file token）
 *
 * 背景：浏览器原生的 <img src>、<a download>、window.open 无法携带 Authorization 头，
 * 所以文件预览/下载只能把凭证放在 query 里。直接把主 access token 放进 URL 是危险的
 * ——它会出现在 Referer、浏览器历史、反代访问日志里，一旦泄露等于账号被完全接管。
 *
 * 因此后端单独提供 POST /auth/file-token，签发一个：
 *   - typ=file，只能访问 /files/:id/preview | download，调不了任何业务接口
 *   - 30 分钟过期
 *   - 与主令牌共享 token_epoch，改密码/登出即刻失效
 * 的短时效令牌。
 *
 * 这里用 Vue 的 ref 缓存它：previewUrl() 必须保持同步签名（模板里直接 :src="..." 调用），
 * 而令牌只能异步拿到。把令牌存进 ref 后，任何在模板/computed 中调用 previewUrl() 的地方
 * 都会自动建立响应式依赖 —— 令牌到手时组件自动重渲染，URL 自动补上 token，
 * 无需改动任何调用方组件。
 */

const TOKEN_KEY = 'mds.fileToken';
const TOKEN_EXP_KEY = 'mds.fileTokenExpireAt';
const TOKEN_OWNER_KEY = 'mds.fileTokenOwner';

/** 提前 60 秒视为过期，避免边界上刚好用一个即将失效的令牌 */
const RENEW_SKEW_MS = 60 * 1000;

const fileToken = ref('');
let expireAt = 0;
let inflight: Promise<string> | null = null;

/** 用主令牌的尾部 16 字符作为归属标记：换了账号 / 重新登录后旧文件令牌立即作废 */
function accessTokenFingerprint(): string {
  const t = localStorage.getItem('token') || '';
  return t ? t.slice(-16) : '';
}

function readCache() {
  try {
    const owner = sessionStorage.getItem(TOKEN_OWNER_KEY) || '';
    if (owner !== accessTokenFingerprint()) {
      clearFileToken();
      return;
    }
    const t = sessionStorage.getItem(TOKEN_KEY) || '';
    const e = Number(sessionStorage.getItem(TOKEN_EXP_KEY) || 0);
    if (t && e > Date.now() + RENEW_SKEW_MS) {
      fileToken.value = t;
      expireAt = e;
    }
  } catch {
    /* sessionStorage 不可用（隐私模式）时退化为纯内存缓存 */
  }
}

function writeCache(token: string, expiresInSec: number) {
  fileToken.value = token;
  expireAt = Date.now() + expiresInSec * 1000;
  try {
    sessionStorage.setItem(TOKEN_KEY, token);
    sessionStorage.setItem(TOKEN_EXP_KEY, String(expireAt));
    sessionStorage.setItem(TOKEN_OWNER_KEY, accessTokenFingerprint());
  } catch {
    /* ignore */
  }
}

/** 清除缓存的文件令牌（登出、切换账号时调用） */
export function clearFileToken() {
  fileToken.value = '';
  expireAt = 0;
  inflight = null;
  try {
    sessionStorage.removeItem(TOKEN_KEY);
    sessionStorage.removeItem(TOKEN_EXP_KEY);
    sessionStorage.removeItem(TOKEN_OWNER_KEY);
  } catch {
    /* ignore */
  }
}

/** 主动获取（或续期）文件令牌。多处并发调用只会发出一个请求。 */
export function refreshFileToken(): Promise<string> {
  if (!localStorage.getItem('token')) {
    clearFileToken();
    return Promise.resolve('');
  }
  if (inflight) return inflight;

  inflight = request
    .post('/auth/file-token')
    .then((res: any) => {
      const token: string = res?.data?.token || '';
      const expiresIn: number = Number(res?.data?.expiresIn) || 1800;
      if (token) writeCache(token, expiresIn);
      return token;
    })
    .catch(() => '')
    .finally(() => {
      inflight = null;
    });

  return inflight;
}

/**
 * 同步取用令牌。若缓存缺失或即将过期，顺手触发一次后台续期；
 * 续期成功后 ref 变化会驱动调用方重渲染。
 */
function currentToken(): string {
  // 登出或换账号后主令牌变了，旧文件令牌必须立即作废（无需其它模块配合）
  let owner = '';
  try {
    owner = sessionStorage.getItem(TOKEN_OWNER_KEY) || '';
  } catch {
    owner = accessTokenFingerprint();
  }
  if (owner !== accessTokenFingerprint()) {
    clearFileToken();
  }
  if (!fileToken.value || Date.now() > expireAt - RENEW_SKEW_MS) {
    void refreshFileToken();
  }
  return fileToken.value;
}

function withToken(path: string): string {
  const t = currentToken();
  return t ? `${path}?token=${encodeURIComponent(t)}` : path;
}

// 模块加载即预热一次，尽量让首屏图片一次就带上令牌
readCache();
if (!fileToken.value) void refreshFileToken();

export const fileApi = {
  upload: (file: File, onProgress?: (percent: number) => void) => {
    const formData = new FormData();
    formData.append('file', file);
    return request.post('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          onProgress(Math.round((e.loaded * 100) / e.total));
        }
      },
    });
  },
  /** 带文件令牌的预览 URL，用于 img / el-image / pdf 等无法自定义 header 的场景 */
  previewUrl: (id: number) => withToken(`/api/files/${id}/preview`),
  /** 不带令牌的裸路径，仅用于自行 fetch 并手动加 Authorization 头的场景 */
  previewUrlRaw: (id: number) => `/api/files/${id}/preview`,
  /** 带文件令牌的下载 URL —— window.open 无法加 header，不带令牌必然 401 */
  downloadUrl: (id: number) => withToken(`/api/files/${id}/download`),
  remove: (id: number) => request.delete(`/files/${id}`),
  refreshFileToken,
  clearFileToken,
};
