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
};
