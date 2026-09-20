import request from './request';

export const backupApi = {
  settings: () => request.get('/backups/settings'),
  saveSettings: (data: { enabled: boolean; intervalHours: number; retention: number }) =>
    request.put('/backups/settings', data),
  list: () => request.get('/backups'),
  run: () => request.post('/backups/run'),
  // 下载走 blob：文件名含中文，由后端在 Content-Disposition 里给出；备份可能较大，超时放宽
  download: (name: string) =>
    request.get('/backups/download', { params: { name }, responseType: 'blob', timeout: 300000 }),
  remove: (name: string) => request.delete('/backups', { params: { name } }),
};
