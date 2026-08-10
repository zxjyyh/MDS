import request from './request';

export const dashboardApi = {
  stats: () => request.get('/dashboard/stats'),
  expiring: (params: { module: string; days: number; page?: number; pageSize?: number }) =>
    request.get('/dashboard/expiring', { params }),
  ignore: (data: { module: string; targetName: string; relatedName: string; expiryDate: string }) =>
    request.post('/dashboard/ignore', data),
  unignore: (id: number) => request.delete(`/dashboard/ignored/${id}`),
  listIgnored: () => request.get('/dashboard/ignored'),
};

export const reminderApi = {
  logs: (params: { page: number; pageSize: number; module?: string }) =>
    request.get('/reminders/logs', { params }),
  testRun: () => request.post('/reminders/test'),
  // 规则管理
  listRules: () => request.get('/reminders/rules'),
  addRule: (data: { module: string; daysBefore: number; enabled?: boolean }) =>
    request.post('/reminders/rules', data),
  updateRule: (id: number, data: { module?: string; daysBefore?: number; enabled?: boolean }) =>
    request.put(`/reminders/rules/${id}`, data),
  deleteRule: (id: number) => request.delete(`/reminders/rules/${id}`),
};

export const settingsApi = {
  getSMTP: () => request.get('/settings/smtp'),
  updateSMTP: (data: { host: string; port: number; user: string; pass: string; from: string }) =>
    request.put('/settings/smtp', data),
};
