import request from './request';

export const companyApi = {
  list: (params: any) => request.get('/companies', { params }),
  // 公司下拉选项（id/name/address）：劳动合同选择甲方后据此自动带出公司地址
  options: () => request.get('/companies/options'),
  detail: (id: number) => request.get(`/companies/${id}`),
  create: (data: any) => request.post('/companies', data),
  update: (id: number, data: any) => request.put(`/companies/${id}`, data),
  remove: (id: number) => request.delete(`/companies/${id}`),
};
