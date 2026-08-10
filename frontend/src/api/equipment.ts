import request from './request';

export const equipmentApi = {
  list: (params: any) => request.get('/equipment', { params }),
  detail: (id: number) => request.get(`/equipment/${id}`),
  create: (data: any) => request.post('/equipment', data),
  update: (id: number, data: any) => request.put(`/equipment/${id}`, data),
  remove: (id: number) => request.delete(`/equipment/${id}`),
  addCert: (equipmentId: number, data: any) =>
    request.post(`/equipment/${equipmentId}/certificates`, data),
  updateCert: (certId: number, data: any) =>
    request.put(`/equipment/certificates/${certId}`, data),
  removeCert: (certId: number) =>
    request.delete(`/equipment/certificates/${certId}`),
};
