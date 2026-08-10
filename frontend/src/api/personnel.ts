import request from './request';

export const personnelApi = {
  list: (params: any) => request.get('/personnel', { params }),
  detail: (id: number) => request.get(`/personnel/${id}`),
  create: (data: any) => request.post('/personnel', data),
  update: (id: number, data: any) => request.put(`/personnel/${id}`, data),
  remove: (id: number) => request.delete(`/personnel/${id}`),
  addCert: (personnelId: number, data: any) =>
    request.post(`/personnel/${personnelId}/certificates`, data),
  updateCert: (certId: number, data: any) =>
    request.put(`/personnel/certificates/${certId}`, data),
  removeCert: (certId: number) =>
    request.delete(`/personnel/certificates/${certId}`),
};
