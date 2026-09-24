import request from './request';

export const companyApi = {
  list: (params: any) => request.get('/companies', { params }),
  // 公司下拉选项（id/name/address）：劳动合同选择甲方后据此自动带出公司地址
  options: () => request.get('/companies/options'),
  // 合法公司名列表（公司档案 ∪ 人员/设备里出现过的检测站，去重）。
  // 用于「公司权限」多选，以及人员/设备表单的「所属检测站」下拉。
  names: () => request.get('/companies/names'),
  detail: (id: number) => request.get(`/companies/${id}`),
  create: (data: any) => request.post('/companies', data),
  update: (id: number, data: any) => request.put(`/companies/${id}`, data),
  remove: (id: number) => request.delete(`/companies/${id}`),
};
