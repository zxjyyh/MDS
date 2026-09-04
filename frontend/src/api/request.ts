import axios from 'axios';
import { ElMessage } from 'element-plus';
import router from '../router';

const request = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    // 文件流请求（blob）：不按 JSON 契约解包，返回完整响应，便于读 headers 与触发浏览器下载
    if (response.config.responseType === 'blob') {
      return response;
    }
    const res = response.data;
    if (res.code === 0) {
      return res;
    }
    ElMessage.error(res.message || '请求失败');
    return Promise.reject(new Error(res.message || 'Error'));
  },
  async (error) => {
    if (error.response) {
      const { status, data } = error.response;
      if (status === 401) {
        localStorage.removeItem('token');
        localStorage.removeItem('refreshToken');
        ElMessage.error('登录已过期，请重新登录');
        router.push('/login');
      } else {
        // 后端下载接口在出错时也可能返回 application/json 的 Blob（如"没有可打包的文件"）
        let msg = data?.message || `请求错误 (${status})`;
        if (data instanceof Blob && data.type.includes('application/json')) {
          try {
            const parsed = JSON.parse(await data.text());
            if (parsed?.message) msg = parsed.message;
          } catch { /* 保留默认错误文案 */ }
        }
        ElMessage.error(msg);
      }
    } else {
      ElMessage.error('网络错误，请检查连接');
    }
    return Promise.reject(error);
  }
);

export default request;
