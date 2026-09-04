import request from './request';

export const personnelApi = {
  list: (params: any) => request.get('/personnel', { params }),
  options: () => request.get('/personnel/options'),
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
  // 清空固定预留证件行（安检证/环保证）：保留行，清除已填信息与已上传附件
  clearCert: (certId: number) =>
    request.post(`/personnel/certificates/${certId}/clear`),
  // 按范围批量导出证书 ZIP（返回完整响应，data 为 Blob）
  exportCerts: (params: { department?: string; status?: string; certName: string }) =>
    request.get('/personnel/certs/export', { params, responseType: 'blob', timeout: 120000 }),
};
