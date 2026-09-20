import request from './request';

export const personnelApi = {
  list: (params: any) => request.get('/personnel', { params }),
  options: () => request.get('/personnel/options'),
  // 证件查漏：按检测站 / 在职状态 + 证书类型查询人员的证件状态（正常/缺证/过期）
  certGap: (params: { department?: string; status?: string; certs?: string }) =>
    request.get('/personnel/cert-gap', { params }),
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
  // 生成劳动合同 docx（返回完整响应，data 为 Blob）
  generateContract: (data: {
    personnelId: number;
    signDate: string;
    stationName?: string;
    stationAddress?: string;
    years?: number;
  }) => request.post('/personnel/docs/contract', data, { responseType: 'blob', timeout: 60000 }),
  // 理工报名材料生成：界面选项 / 材料清单（证书类型、报考类别、岗位、检测线类别…）
  ligongOptions: () => request.get('/personnel/docs/ligong-options'),
  // 生成前预检：判断本批人员的劳动合同能否复用档案已上传件（决定要不要弹二次确认）。
  // 报考类别逐人传 —— 增项的人不涉及劳动合同，后端会跳过。
  ligongPrecheck: (data: {
    enrollDate?: string;
    persons: Array<{ personnelId: number; applyType: string }>;
  }) => request.post('/personnel/docs/ligong-precheck', data),
  // 生成整套理工报名材料 ZIP（返回完整响应，data 为 Blob）
  generateLigongEnroll: (data: any) =>
    request.post('/personnel/docs/ligong-enroll', data, { responseType: 'blob', timeout: 180000 }),
};
