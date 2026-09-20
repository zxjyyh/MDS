import request from './request';

export const operationLogApi = {
  list: (params: any) => request.get('/operation-logs', { params }),
  clear: () => request.delete('/operation-logs'),
};
