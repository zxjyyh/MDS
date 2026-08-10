import request from './request';

export interface LoginParams {
  username: string;
  password: string;
}

export interface UserInfo {
  id: number;
  username: string;
  realName: string;
  email: string;
  role: 'admin' | 'user';
  status?: string;
  lastLoginAt?: string;
}

export const authApi = {
  login: (data: LoginParams) =>
    request.post('/auth/login', data),
  getProfile: () =>
    request.get('/auth/profile'),
  changePassword: (data: { oldPassword: string; newPassword: string }) =>
    request.put('/auth/password', data),
  refresh: (refreshToken: string) =>
    request.post('/auth/refresh', { refreshToken }),
};
