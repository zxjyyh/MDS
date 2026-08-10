import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';
import AutoImport from 'unplugin-auto-import/vite';
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers';
import path from 'path';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');

  // 开发服务器允许的域名，通过 .env.local 的 VITE_ALLOWED_HOSTS 配置
  // 例如：VITE_ALLOWED_HOSTS=hx.example.com,.example.com
  const allowedHosts = (env.VITE_ALLOWED_HOSTS || '')
    .split(',')
    .map((h) => h.trim())
    .filter(Boolean);

  // 后端地址，默认本机 3000
  const apiTarget = env.VITE_API_TARGET || 'http://localhost:3000';

  return {
    plugins: [
      vue(),
      AutoImport({
        resolvers: [ElementPlusResolver()],
      }),
    ],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
    },
    // 预构建 element-plus，避免运行时发现新组件依赖导致页面重载
    optimizeDeps: {
      include: ['element-plus', 'vue', 'vue-router', 'pinia', 'axios', 'dayjs'],
    },
    server: {
      host: '0.0.0.0',
      port: 8080,
      ...(allowedHosts.length > 0 ? { allowedHosts } : {}),
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
        },
      },
    },
  };
});
