import request from './request';

export const userApi = {
  list: (params: any) => request.get('/users', { params }),
  detail: (id: number) => request.get(`/users/${id}`),
  create: (data: any) => request.post('/users', data),
  update: (id: number, data: any) => request.put(`/users/${id}`, data),
  remove: (id: number) => request.delete(`/users/${id}`),
  resetPassword: (id: number, newPassword: string) =>
    request.put(`/users/${id}/password`, { newPassword }),
  setStatus: (id: number, status: string) =>
    request.patch(`/users/${id}/status`, { status }),
  // 覆盖式设置账号的可见公司（数据隔离）。传空数组 = 清空 = 恢复为不受限。
  setCompanies: (id: number, companies: string[]) =>
    request.put(`/users/${id}/companies`, { companies }),
};
