import request from './request';

export const categoryApi = {
  getTree: (systemType: string) =>
    request.get('/categories', { params: { systemType } }),
  create: (data: any) => request.post('/categories', data),
  update: (id: number, data: any) => request.put(`/categories/${id}`, data),
  remove: (id: number) => request.delete(`/categories/${id}`),
};

export const documentApi = {
  list: (params: any) => request.get('/documents', { params }),
  detail: (id: number) => request.get(`/documents/${id}`),
  create: (data: any) => request.post('/documents', data),
  update: (id: number, data: any) => request.put(`/documents/${id}`, data),
  remove: (id: number) => request.delete(`/documents/${id}`),
};
